package monitor

import (
	"fmt"
	"math"
	"solana-bot/internal/global/utils"
	atomic_ "solana-bot/internal/global/utils/atomic"
	"strconv"
	"strings"
	"time"
)

var Profit *ProfitStrategy

func (p *PumpFunMonitor) Profit() {

	sub1 := p.pubsub.Subscribe()

	go utils.WriteProfit("profit.csv", sub1)

	sub2 := p.pubsub.Subscribe()
	strategy := NewProfitStrategy(p.GetMultiplier, p.SetMultiplier)

	Profit = strategy

	go func() {
		for input := range sub2 {
			strategy.OnProfitInfo(input.(string))
		}
	}()

}

type ProfitStrategy struct {
	profits              []float64
	lastMultiplier       float64
	lastUpdate           time.Time
	lastBigWin           time.Time
	lastInvestmentAmount atomic_.Float64 // 上一次实际投入金额
	maxMultiplier        float64         // 🧱 最大倍数限制
	GetMultiplierFn      func() float64
	SetMultiplierFn      func(float64)
}

func NewProfitStrategy(getFn func() float64, setFn func(float64)) *ProfitStrategy {
	return &ProfitStrategy{
		GetMultiplierFn: getFn,
		SetMultiplierFn: setFn,
		profits:         []float64{},
		lastUpdate:      time.Now(),
		maxMultiplier:   5.0,
	}
}

// 接收格式："time,token,buyAmount,profit"
func (ps *ProfitStrategy) OnProfitInfo(info string) {
	info = strings.ReplaceAll(info, "\"", "")
	parts := strings.Split(info, ",")
	if len(parts) < 4 {
		fmt.Println("[策略解析错误] 格式错误，跳过:", info)
		return
	}
	token := parts[1]
	buyAmountStr := parts[2]
	profitStr := parts[3]

	buyAmount, err1 := strconv.ParseFloat(buyAmountStr, 64)
	buyAmount = math.Abs(buyAmount)
	profit, err2 := strconv.ParseFloat(profitStr, 64)

	if err1 != nil || err2 != nil || buyAmount == 0 {
		fmt.Printf("[策略解析错误] buyAmount=%.4f profit=%s\n", buyAmount, profitStr)
		return
	}

	// 计算收益率
	profitRate := profit / buyAmount * 100 // 转为百分比
	fmt.Printf("[分析] %s收益率: %.4f%%\n", token, profitRate)

	ps.profits = append(ps.profits, profitRate)

	if len(ps.profits) > 5 {
		ps.profits = ps.profits[len(ps.profits)-5:]
	}
	if len(ps.profits) < 5 {
		return
	}

	ps.evaluateStrategy(profitRate)
}

func (ps *ProfitStrategy) evaluateStrategy(currentProfit float64) {
	timePassed := time.Since(ps.lastUpdate).Minutes()
	if timePassed > 1 {
		decay := math.Min(timePassed/5, 5) // 每5分钟最多恢复5倍
		ps.lastMultiplier = math.Min(1.0, ps.lastMultiplier*decay)
		ps.lastUpdate = time.Now()
	}

	multiplier := ps.GetMultiplierFn()
	maxMultiplier := ps.maxMultiplier
	lastInvestAmount := ps.lastInvestmentAmount.Load()

	// ✅ 回本优先：回到盈利或持平，恢复初始倍数
	if currentProfit >= 0 && multiplier > 1.0 {
		fmt.Println("[策略建议] 已回本，重置倍数为 1.0x")
		ps.SetMultiplierFn(1.0)
		ps.lastMultiplier = 1.0
		return
	}

	// ✅ 避免无限加码：只在亏损大于 5% 时尝试加码
	if currentProfit < -5 {
		if time.Since(ps.lastBigWin) < 3*time.Minute {
			fmt.Println("[策略保护] 刚刚大赚，暂不加码")
			return
		}
		if lastInvestAmount <= 0 {
			fmt.Println("[策略警告] 上次投入为 0，跳过加码")
			return
		}

		lossRatio := -currentProfit / 100.0 // 转换为 0.05 这种比例
		newMultiplier := multiplier * (1.0 + lossRatio)

		newMultiplier = math.Min(newMultiplier, maxMultiplier)
		if newMultiplier < 1.0 {
			newMultiplier = 1.0
		}

		fmt.Printf("[策略建议] 当前亏损 %.2f%%，建议增仓至 %.2fx\n", -currentProfit, newMultiplier)
		ps.lastMultiplier = multiplier
		ps.SetMultiplierFn(newMultiplier)
		return
	}

	// ✅ 小亏容忍，不动
	fmt.Printf("[策略建议] 当前收益率 %.2f%%，维持当前仓位 %.2fx\n", currentProfit, multiplier)
}

// 获取当前的买入系数
func (p *PumpFunMonitor) GetMultiplier() float64 {
	return p.buyMultiplier.Load()
}

// 设置买入系数（需控制上下限）
func (p *PumpFunMonitor) SetMultiplier(newVal float64) {
	// 限制范围在 [0.1, 1.0]
	newVal = math.Max(0.2, math.Min(4.0, newVal))
	p.buyMultiplier.Store(newVal)
}
