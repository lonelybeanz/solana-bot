package monitor

import (
	"math/big"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

func (p *PumpFunMonitor) SetLimitSell(ts *TokenSwap) {
	tokenAddress := ts.Token.TokenAddress
	price, _ := ts.MySwap.BuyPrice.Load().Float64()
	amount := ts.MySwap.BuyAmount.Load()
	logx.Infof("[%s] 买入价格: %.15f,买入数量:%d", tokenAddress, price, amount.Int64())

	//止损单，低于10%止损
	// go p.DoLimitSell(ts, price*0.7, amount, false)
	//止盈单
	sellAmount := new(big.Int)
	sellAmountFloat := new(big.Float).Mul(new(big.Float).SetInt(amount), big.NewFloat(0.8))
	sellAmount, _ = sellAmountFloat.Int(sellAmount)
	go p.DoLimitSell(ts, price*1.5, sellAmount, true)

	go p.DoLimitSell(ts, price*4, new(big.Int).Mul(sellAmount, big.NewInt(5)), true)

	go p.DoLimitSell(ts, price*8, new(big.Int).Mul(sellAmount, big.NewInt(10)), true)

}

/*
 */
func (p *PumpFunMonitor) DoLimitSell(ts *TokenSwap, limitPrice float64, amount *big.Int, profitOrLoss bool) {
	tokenAddress := ts.Token.TokenAddress
	logx.Infof("[%s] 开始监听止盈/止损 , 待监听价格: %.15f ", tokenAddress, limitPrice)
	for {
		select {
		case <-ts.Ctx.Done():
			logx.Info("收到停止信号，停止限价单监听")
			return
		default:
			// 获取当前价格
			nowPrice, _ := ts.Token.TokenPrice.Load().Float64()
			if nowPrice <= 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			logx.Debugf("[%s] 当前价格: %.15f, 待监听价格: %.15f", tokenAddress, nowPrice, limitPrice)

			// 触发止损
			if !profitOrLoss && nowPrice < limitPrice {
				logx.Infof("[%s] 当前价格: %.15f, 触发止损数量: %d", tokenAddress, nowPrice, amount.Int64())
				err := p.ExecuteSell(ts, amount)
				if err != nil {
					continue
				}
				break
			}

			// 触发止盈
			if profitOrLoss && nowPrice > limitPrice {
				logx.Infof("[%s] 当前价格: %.15f, 触发止盈数量: %d", tokenAddress, nowPrice, amount.Int64())
				err := p.ExecuteSell(ts, amount)
				if err != nil {
					continue
				}
				break
			}

			time.Sleep(400 * time.Millisecond)
		}
	}

}
