package shot

import (
	"context"
	"log"
	"math/big"
	"solana-bot/internal/global"

	"github.com/gagliardetto/solana-go"
	associated_token_account "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	token_program "github.com/gagliardetto/solana-go/programs/token"
)

func QuoteBuyByCurve(
	solInLamports *big.Int,
	virtualSolReserves *big.Int,
	virtualTokenReserves *big.Int,
) *big.Int {
	if solInLamports.Sign() == 0 || virtualSolReserves.Sign() == 0 || virtualTokenReserves.Sign() == 0 {
		return big.NewInt(0)
	}

	// product = x * y
	product := new(big.Int).Mul(virtualSolReserves, virtualTokenReserves)

	// x' = x + Δx
	newVirtualSolReserves := new(big.Int).Add(virtualSolReserves, solInLamports)

	// y' = product / x'
	newVirtualTokenReserves := new(big.Int).Div(product, newVirtualSolReserves)
	newVirtualTokenReserves.Add(newVirtualTokenReserves, big.NewInt(1)) // +1 for rounding safety

	// tokenOut = y - y'
	tokenOut := new(big.Int).Sub(virtualTokenReserves, newVirtualTokenReserves)

	// 限制 tokenOut 不超过 y
	if tokenOut.Cmp(virtualTokenReserves) > 0 {
		tokenOut.Set(virtualTokenReserves)
	}
	return tokenOut
}

func QuoteSellByCurve(
	amountIn *big.Int,
	virtualSolReserves *big.Int,
	virtualTokenReserves *big.Int,
) *big.Int {
	// 新的虚拟 token 储备 = 旧的 + 用户卖出的
	newReserves := new(big.Int).Add(virtualTokenReserves, amountIn)

	// 按照常数乘积公式计算能拿多少 SOL
	temp := new(big.Int).Mul(amountIn, virtualSolReserves)
	amountOut := new(big.Int).Div(temp, newReserves)

	// 计算手续费
	fee := pumpGetFee(amountOut, FeeBasisPoints)

	// 减掉手续费
	amountOutAfterFee := new(big.Int).Sub(amountOut, fee)

	// 防止超发真实 SOL 储备（重要！）
	if amountOutAfterFee.Uint64() > virtualSolReserves.Uint64() {
		amountOutAfterFee = virtualSolReserves
	}

	return amountOutAfterFee
}

// slippage is a value between 0 - 100
func applySlippage(amount *big.Int, slippage float32) *big.Int {

	slippageBP := (int64(100*slippage) + 25) * SlippageAdjustment
	maxSlippage := new(big.Int).Mul(global.Big10000, big.NewInt(SlippageAdjustment))

	if slippageBP > maxSlippage.Int64() {
		slippageBP = global.Big10000.Int64()
	}

	slippageBPBN := big.NewInt(slippageBP)

	// we adjust slippage so that it caps out at 50%
	slippageNumeratorMul := new(big.Int).Sub(maxSlippage, slippageBPBN)
	slippageNumerator := new(big.Int).Mul(amount, slippageNumeratorMul)
	amountWithSlippage := new(big.Int).Div(slippageNumerator, maxSlippage)
	return amountWithSlippage
}

func EnsurePDAAccount(
	instrs *[]solana.Instruction,
	funder solana.PublicKey, // 付费者
	seeds [][]byte, // 种子
	programID solana.PublicKey, // 程序ID
	space uint64, // 账户空间大小
	owner solana.PublicKey, // 账户owner
	lamports uint64, // 要分配的 lamports
) {
	// 1. 通过 FindProgramAddress 得到 PDA
	pda, bump, _ := solana.FindProgramAddress(seeds, programID)

	log.Printf("📌 PDA: %s (bump=%d)", pda, bump)

	// 2. 检查账户是否存在
	accountInfo, err := global.GetRPCForRequest().GetAccountInfo(context.Background(), pda)
	if err == nil && accountInfo.Value != nil {
		log.Println("✅ 账户已存在")
		return
	}

	// 3. 如果不存在，则构建 SystemProgram.CreateAccount
	ix := system.NewCreateAccountInstruction(
		lamports, // 分配 lamports
		space,    // 空间大小
		owner,    // owner 程序
		funder,   // from
		pda,      // new account
	).Build()

	*instrs = append(*instrs, ix)
}

func createTokenAccountIfNotExists(instrs *[]solana.Instruction, owner solana.PublicKey, mint solana.PublicKey) {
	bal, _ := global.GetTokenBalance(owner, mint)
	if bal == nil || bal.Uint64() == 0 {
		*instrs = append(*instrs, associated_token_account.NewCreateInstruction(owner, owner, mint).Build())
	}
}

func closeATA(instrs *[]solana.Instruction, owner solana.PublicKey, mint solana.PublicKey) {
	ata, _, _ := solana.FindAssociatedTokenAddress(owner, mint)
	closeInst := token_program.NewCloseAccountInstruction(
		ata,
		owner,
		owner,
		[]solana.PublicKey{},
	).Build()
	*instrs = append(*instrs, closeInst)
}

// EstimateSwapOut 计算从 AMM 池中 tokenIn -> tokenOut 的 swap 输出量
// 适用于 Raydium LaunchLab、Uniswap 等 x*y=k 恒定乘积模型
//
// 参数：
//   - amountIn: 用户输入的 tokenIn 数量（一般是 lamports）
//   - reserveIn: 池子的 tokenIn 储备
//   - reserveOut: 池子的 tokenOut 储备
//   - feeBps: 手续费（basis points，1% = 100 bps）
//
// 返回：
//   - 实际可获得的 tokenOut 数量（不包含 slippage 容错）
func EstimateSwapOut(amountIn, reserveIn, reserveOut *big.Int, feeBps uint64) *big.Int {
	if amountIn.Sign() <= 0 || reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return big.NewInt(0)
	}

	// 扣除手续费：amountInWithFee = amountIn * (10000 - feeBps)
	feeNumerator := big.NewInt(int64(10000 - feeBps))
	amountInWithFee := new(big.Int).Mul(amountIn, feeNumerator)

	// 分母 = reserveIn * 10000 + amountIn * (10000 - feeBps)
	denominator := new(big.Int).Add(
		new(big.Int).Mul(reserveIn, big.NewInt(10000)),
		amountInWithFee,
	)

	// 输出 = amountInWithFee * reserveOut / denominator
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
	out := new(big.Int).Div(numerator, denominator)

	// 限制输出不能超过池中储备（极端情况下可能误差过大）
	if out.Cmp(reserveOut) > 0 {
		return reserveOut
	}

	return out
}

// GetAmountOut implements constant product swap output
func GetAmountOut(amountIn, inputReserve, outputReserve *big.Int) *big.Int {
	if amountIn.Sign() == 0 {
		return big.NewInt(0)
	}
	// amountOut = amountIn * outputReserve / (inputReserve + amountIn)
	numerator := new(big.Int).Mul(amountIn, outputReserve)
	denominator := new(big.Int).Add(inputReserve, amountIn)

	// Prevent division by zero
	if denominator.Sign() == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).Div(numerator, denominator)
}

// PredictNextOutputByRealState 转换自 buyExactIn 的预测逻辑
func PredictNextOutputByRealState(
	virtualBase, virtualQuote *big.Int,
	realBaseAfter, realQuoteAfter *big.Int,
	amountIn *big.Int, // 输入是 quote
) *big.Int {
	// inputReserve = virtualQuote + realQuoteAfter
	inputReserve := new(big.Int).Add(virtualQuote, realQuoteAfter)

	// outputReserve = virtualBase - realBaseAfter
	outputReserve := new(big.Int).Sub(virtualBase, realBaseAfter)

	// 使用标准恒定乘积公式
	return GetAmountOut(amountIn, inputReserve, outputReserve)
}

// PredictSellExactInByRealState 实现 sellExactIn 预测逻辑
func PredictSellExactInByRealState(
	virtualBase, virtualQuote *big.Int,
	realBaseAfter, realQuoteAfter *big.Int,
	amountIn *big.Int, // 输入是 base
) *big.Int {
	// inputReserve = virtualBase + realBaseAfter
	inputReserve := new(big.Int).Add(virtualBase, realBaseAfter)

	// outputReserve = virtualQuote - realQuoteAfter
	outputReserve := new(big.Int).Sub(virtualQuote, realQuoteAfter)

	// 使用标准恒定乘积公式
	return GetAmountOut(amountIn, inputReserve, outputReserve)
}
