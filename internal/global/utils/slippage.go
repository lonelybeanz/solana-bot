package utils

import "math"

// 参数权重（可调整）
const (
	MaxSlippage      = 0.2   // 最大允许滑点
	MinSlippage      = 0.001 // 最小滑点
	BaseSlippage     = 0.005 // 默认滑点
	HighBotThreshold = 4     // 高密度机器人阈值
	LowLiquiditySOL  = 10    // 低流动性池标准（sol）
	NightHoursStart  = 0
	NightHoursEnd    = 6
)

// CalcDynamicSlippage 根据市场情况返回推荐滑点百分比（例如 0.12 = 12%）
func CalcDynamicSlippage(
	botCount int,
	poolLiquiditySOL float64,
	tokenCreatorHistoryRugRate float64,
	hourOfDay int,
	marketTrendScore float64, // 0-1之间，例如 Pump.fun 热度评分
) float64 {
	slippage := BaseSlippage

	// 如果机器人多，说明竞争激烈
	if botCount > HighBotThreshold {
		slippage += 0.05
	}

	// 如果池子很小，滑点容易变大
	if poolLiquiditySOL < LowLiquiditySOL {
		slippage += 0.05
	}

	// 如果历史 rug 率低，可以稍微激进
	if tokenCreatorHistoryRugRate < 0.2 {
		slippage += 0.02
	} else if tokenCreatorHistoryRugRate > 0.6 {
		slippage -= 0.02
	}

	// 如果是夜间，项目少、量小，滑点设低保守一点
	if hourOfDay >= NightHoursStart && hourOfDay <= NightHoursEnd {
		slippage -= 0.01
	}

	// 市场很热的时候，滑点适当提高
	if marketTrendScore > 0.8 {
		slippage += 0.05
	} else if marketTrendScore < 0.2 {
		slippage -= 0.02
	}

	// 限制范围
	slippage = math.Max(MinSlippage, math.Min(MaxSlippage, slippage))

	return slippage
}
