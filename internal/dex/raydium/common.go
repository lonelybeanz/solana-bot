package raydium

import (
	"math"
	"math/big"
)

const (
	// Program IDs
	RaydiumLaunchpadProgramID = "LanMV9sAd7wArD4vJFi2qDdfnVhFxYSUg6eADduJ3uj"

	// Other Program IDs
	TokenProgram     = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	Token2022Program = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"

	// Authority addresses
	RaydiumLaunchpadAuthority = "WLHv2UAZm6z4KyaaELi5pjdbJh6RESMva1Rnn8pJVVh"
	EventAuthority            = "2DPAtwB8L12vrMRExbLuyGnC7n2J5LNoZQSejeQGpwkr"

	// Token addresses
	NativeMint     = "So11111111111111111111111111111111111111112"
	LamportsPerSol = 1000000000 // 1 SOL = 10^9 lamports
)

func convert_sol_to_base_tokens(
	solAmount float64, // 想投入的 SOL 数量（单位：SOL）
	baseBalanceTokens *big.Float, // Base token 的池子余额（已除以 decimals）
	quoteBalanceSol *big.Float, // Quote token 的池子余额（单位：SOL）
	decimalsBase int, // base token 的精度
	slippagePct float64, // 滑点百分比（如 0.01 表示 1%）
) (minBaseOut uint64, quoteInLamports uint64) {
	if solAmount == 0 || baseBalanceTokens.Cmp(big.NewFloat(0)) == 0 {
		return 0, 0
	}

	// 当前价格: quote / base = 每个 base token 的价格（单位 SOL）
	price := new(big.Float).Quo(quoteBalanceSol, baseBalanceTokens)

	priceInt, _ := price.Float64()

	price = new(big.Float).SetFloat64(priceInt)

	// 无滑点情况下可以获得多少 base token（不含 decimals）
	baseAmount := new(big.Float).Quo(big.NewFloat(solAmount), price)

	baseAmountInt, _ := baseAmount.Float64()
	baseAmount = new(big.Float).SetFloat64(baseAmountInt)

	// 应用滑点：减少预期输出
	slippageFactor := big.NewFloat(1 - slippagePct)
	baseWithSlippage := new(big.Float).Mul(baseAmount, slippageFactor)

	// 乘上 decimals，转成 uint64 raw token
	baseRaw := new(big.Float).Mul(baseWithSlippage, big.NewFloat(math.Pow10(decimalsBase)))
	minBaseOut, _ = baseRaw.Uint64()

	// SOL 输入转 lamports
	quoteInLamports = uint64(solAmount * LamportsPerSol)

	return
}
func convert_base_tokens_to_sol(
	baseAmount *big.Int, // 想卖出的 base token 数量（单位是实际数量，如1.23）
	baseBalanceTokens *big.Float, // 池子的 base token 余额
	quoteBalanceSol *big.Float, // 池子的 quote token（SOL）余额
	decimalsBase int, // base token 的 decimals
	slippagePct float64, // 可接受的滑点百分比
) (baseInRaw uint64, minQuoteOutLamports uint64) {
	// 当前价格：1 base = quote / base
	price := new(big.Float).Quo(quoteBalanceSol, baseBalanceTokens)

	quote := new(big.Float).Quo(big.NewFloat(float64(baseAmount.Uint64())), big.NewFloat(math.Pow10(decimalsBase)))

	// 期望卖出 base 得到多少 SOL
	quoteOut := new(big.Float).Mul(quote, price)

	// 滑点影响：最低接受 quoteOut * (1 - slippage)
	quoteWithSlippage := new(big.Float).Mul(quoteOut, big.NewFloat(1-slippagePct))

	// quote 最小 SOL（单位：lamports）
	lamports := new(big.Float).Mul(quoteWithSlippage, big.NewFloat(LamportsPerSol))
	minQuoteOutLamports, _ = lamports.Uint64()

	baseInRaw = baseAmount.Uint64()

	return
}

// 计算价格
func GetPrice(baseBalanceTokens *big.Float, quoteBalanceSol *big.Float) *big.Float {
	return new(big.Float).Quo(quoteBalanceSol, baseBalanceTokens)
}
