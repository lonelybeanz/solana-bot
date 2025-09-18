package raydium

import (
	"math"
	"math/big"
	"testing"
)

func TestC(t *testing.T) {
	t.Log(LaunchpadSwapBaseOut(3000000000, 1073025605596382, 30000852951, 37_500_000, 6, 1/100))
	t.Log(ConvertSolToBaseTokensLaunchpad(3, big.NewFloat(float64(1073025605596382/1e6)), big.NewFloat(float64(30000852951/1e9)), 6, 1/100))
	t.Log(convert_sol_to_base_tokens(3, big.NewFloat(float64(1073025605596382/1e6)), big.NewFloat(float64(30000852951/1e9)), 6, 1/100))

	// t.Log(convert_base_tokens_to_sol(big.NewInt(2747165339225), big.NewFloat(float64(931453287908447/1e6)), big.NewFloat(float64(4099900000/1e9)), 6, 10/100))
}

func LaunchpadSwapBaseOut(
	amountInLamports uint64, // 3000000000
	virtualBase, virtualQuote uint64, // 都是 raw
	feeLamports uint64, // 37500000
	decimalsBase int,
	slippagePct float64,
) (minBaseOut uint64) {
	// 精度换算
	amountIn := new(big.Float).SetUint64(amountInLamports)
	fee := new(big.Float).SetUint64(feeLamports)
	effectiveIn := new(big.Float).Sub(amountIn, fee)

	// 虚拟池子换算成 float
	virtualBaseF := new(big.Float).SetUint64(virtualBase)
	virtualQuoteF := new(big.Float).SetUint64(virtualQuote)

	// baseOut = virtualBase * effectiveIn / (virtualQuote + effectiveIn)
	denom := new(big.Float).Add(virtualQuoteF, effectiveIn)
	numer := new(big.Float).Mul(virtualBaseF, effectiveIn)
	baseOut := new(big.Float).Quo(numer, denom)

	// 滑点
	if slippagePct > 0 {
		baseOut.Mul(baseOut, big.NewFloat(1-slippagePct))
	}

	baseOutUint, _ := baseOut.Uint64()
	return baseOutUint
}

func ConvertSolToBaseTokensLaunchpad(
	solAmount float64, // 想投入的 SOL 数量（单位 SOL）
	virtualBase *big.Float, // 虚拟 base token（已除以 decimals）
	virtualQuote *big.Float, // 虚拟 quote token（单位 SOL）
	decimalsBase int, // base token 的 decimals
	slippagePct float64, // 滑点容忍百分比
) (minBaseOut uint64, quoteInLamports uint64) {
	if solAmount == 0 || virtualQuote.Cmp(big.NewFloat(0)) == 0 {
		return 0, 0
	}

	// 真实输入 SOL 金额
	quoteIn := big.NewFloat(solAmount)

	// baseOut = virtualBase * quoteIn / (virtualQuote + quoteIn)
	denominator := new(big.Float).Add(virtualQuote, quoteIn)
	numerator := new(big.Float).Mul(virtualBase, quoteIn)
	baseOut := new(big.Float).Quo(numerator, denominator)

	// 应用滑点：输出下限
	slippageFactor := big.NewFloat(1 - slippagePct)
	baseOutWithSlippage := new(big.Float).Mul(baseOut, slippageFactor)

	// 转换成原始单位（uint64）
	baseRaw := new(big.Float).Mul(baseOutWithSlippage, big.NewFloat(math.Pow10(decimalsBase)))
	minBaseOut, _ = baseRaw.Uint64()

	// 输入的 quote 金额转为 lamports
	quoteInLamports = uint64(solAmount * LamportsPerSol)

	return
}
