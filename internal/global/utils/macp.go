package utils

import (
	"math"
	"math/big"
)

var (
	peakCap      = big.NewFloat(3000)  // 峰值市值
	maxBuy       = big.NewFloat(0.1)   // 最大买入 0.1
	minBuy       = big.NewFloat(0.005) // 最小买入 0.005
	decayDivider = big.NewFloat(1000)  // 控制衰减速度，可调
)

// 转 float64（用于 math.Exp）
func toF64(f *big.Float) float64 {
	val, _ := f.Float64()
	return val
}

// 核心函数：根据市值返回买入金额（big.Float）
func CalculateBuyAmount(marketCap *big.Float) *big.Float {
	if marketCap == nil {
		return nil
	}
	// 如果小于等于峰值市值，直接返回最大买入金额
	if marketCap.Cmp(peakCap) <= 0 {
		return maxBuy
	}

	// 衰减计算： ratio = e^(-(marketCap - peakCap)/decayDivider)
	diff := new(big.Float).Sub(marketCap, peakCap)
	div := new(big.Float).Quo(diff, decayDivider)
	divFloat := toF64(div)
	decayRatio := math.Exp(-divFloat)

	// buyAmount = maxBuy * decayRatio
	decayF := big.NewFloat(decayRatio)
	buyAmount := new(big.Float).Mul(maxBuy, decayF)

	// 最小买入限制
	if buyAmount.Cmp(minBuy) < 0 {
		return minBuy
	}

	return buyAmount
}
