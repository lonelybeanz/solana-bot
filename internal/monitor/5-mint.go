package monitor

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"solana-bot/internal/global"
	"solana-bot/internal/stream"
	"sync"
	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/gagliardetto/solana-go"
	"github.com/zeromicro/go-zero/core/logx"
)

func (p *PumpFunMonitor) workerForMint() {
	p.ListenPumpAccountTransation()
}

func (p *PumpFunMonitor) ListenPumpAccountTransation() {

	logx.Infof("[%s]:监听账户交易", "all")
	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	commitment := pb.CommitmentLevel_PROCESSED
	subscription.Commitment = &commitment
	subscription.Transactions = make(map[string]*pb.SubscribeRequestFilterTransactions)
	failed := false
	vote := false
	subscription.Transactions["transactions_sub"] = &pb.SubscribeRequestFilterTransactions{
		Failed: &failed,
		Vote:   &vote,
	}

	subscription.Transactions["transactions_sub"].AccountInclude = []string{PumpTokenMintAuthority, RaydiumLaunchpadAuthority}

	var once sync.Once
	for _, s := range p.streams {
		s.Subscribe(p.ctx, &subscription, &once, subscribe)
	}

	p.runWithCtx(context.Background(), subscribe, func(msg interface{}) {

		hour := time.Now().Hour()
		params := GetStrategyParamsByHour(hour)
		if !params.MintStart {
			return
		}

		var got *pb.SubscribeUpdate
		v := msg.(*stream.StreamMessage)
		got = v.Data.(*pb.SubscribeUpdate)
		tx := got.GetTransaction()
		if tx == nil {
			return
		}
		// token, err := p.ParseCreateTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
		// if err != nil || token == nil {
		// 	return
		// }

		if global.Sol_Balance.Load() != 0 && global.Sol_Balance.Load() < 5e8 {
			return
		}

		swapInfo, err := ParseSwapTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
		if err != nil || swapInfo == nil {
			return
		}
		var isCreate bool
		if swapInfo.SwapType == "RaydiumLaunchpad" {
			for _, inner := range tx.Transaction.Transaction.Message.Instructions {
				if len(inner.Accounts) == 18 {
					first8 := inner.Data[:8]
					if bytes.Equal(first8, []byte{0xaf, 0xaf, 0x6d, 0x1f, 0x0d, 0x98, 0x9b, 0xed}) {
						isCreate = true
						break
					}
				}
			}
			if !isCreate {
				return
			}
		}

		devBuyAmount := swapInfo.TokenInAmount
		if swapInfo.TokenInMint.String() != global.Solana || //不是sol买入
			swapInfo.TokenOutMint.String() == global.Solana || // 排除卖出操作
			devBuyAmount < 1e9 || //排除小额交易
			devBuyAmount > 3e9 { // 排除大额交易
			return
		}

		// if strings.Contains(swapInfo.SwapType, "PumpFun") {
		// 	return
		// }

		logx.Infof("[%s]:收到new交易{%s}", swapInfo.TokenOutMint, solana.SignatureFromBytes(tx.Transaction.Signature).String())

		if _, b := BuyCache.Get(swapInfo.TokenOutMint.String()); b {
			return
		}
		BuyCache.Set(swapInfo.TokenOutMint.String(), true)

		ata, _, _ := solana.FindAssociatedTokenAddress(p.wallet.PublicKey(), swapInfo.TokenOutMint)
		ts := NewTokenSwap(true, swapInfo.Signatures[0].String(), swapInfo.TokenOutMint.String(), []string{swapInfo.Signers[0].String()}, ata.String(), swapInfo.PoolData)
		ts.Tracked.BuyAmount = big.NewInt(int64(devBuyAmount))
		ts.Tracked.RemainingAmount.Store(big.NewInt(int64(devBuyAmount)))

		// if canBuy.Load() {
		go p.NewTokenBackRun(ts, tx.Slot)
		// }

	})

}

func (p *PumpFunMonitor) NewTokenBackRun(ts *TokenSwap, slot uint64) {
	hour := time.Now().Hour()
	params := GetStrategyParamsByHour(hour)

	devBuyAmount := ts.Tracked.BuyAmount.Uint64()
	tokenAddress := ts.Token.TokenAddress

	logx.Infof("[%s]:BackRun Mint 交易 ", tokenAddress)

	// 动态持有时间逻辑：买入金额越少，持有时间越长；≥2 SOL 固定 100ms
	solAmount := float64(devBuyAmount) / 1e9
	holdDuration := calculateHoldDuration(solAmount, NegativeCurve)

	delayMillisecond := time.Duration(params.DelayMillisecond)
	fmt.Println("延迟时间：", delayMillisecond.Seconds())
	//等待延迟
	time.Sleep(time.Millisecond * time.Duration(params.DelayMillisecond))

	err := p.BuyBefore(ts)
	if err != nil {
		logx.Errorf("[%s]:买入校验失败: %v", tokenAddress, err)
		p.BuyError(ts)
		return
	}

	// err = mcapCheck(token)
	// if err != nil {
	// 	logx.Errorf("[%s]:mcapCheck err:%v", tokenAddress, err)
	// 	return
	// }

	// err = holderCheck(tokenAddress, bondingCurveData.BondingCurvePk.String())
	// if err != nil {
	// 	logx.Errorf("[%s]:holderCheck err:%v", tokenAddress, err)
	// 	return
	// }

	// 计算买入数量
	// buyAmount := tools.CalculateBuyAmount(token.mcap)
	// if buyAmount == nil {
	// 	logx.Errorf("[%s]:计算买入数量失败", tokenAddress)
	// 	return
	// }
	buyAmount := calculateDynamicBuyAmount(devBuyAmount)

	slippage := float32(params.BuySlippage) //float32(0.5)

	// if big.NewFloat(solAmount).Cmp(buyAmount) > 0 {
	// 	slippage = 50
	// 	holdDuration = 3 * time.Second
	// }

	if devBuyAmount > 10e9 {
		slippage = 20
		holdDuration = 1 * time.Minute
	}

	if buyAmount.Cmp(big.NewFloat(solAmount)) >= 0 {
		holdDuration = 400 * time.Millisecond
	}

	// if strings.Contains(tokenAddress, "pump") {
	// 	holdDuration = 1200 * time.Millisecond
	// }

	// 买入
	resp, err := p.buyToken(ts, buyAmount, slippage)
	if err != nil {
		logx.Errorf("[%s] 买入失败,err:%v", ts.Token.TokenAddress, err)
		p.BuyError(ts)
		return
	}

	p.BuyDone(ts, resp)

	// if resp.Slot == slot && solAmount < 1 {
	// 	// holdDuration = holdDuration * 2
	// } else if resp.Slot-slot > 3 {
	// 	holdDuration = 1 * time.Millisecond
	// }

	if resp.Slot-slot > 1 {
		holdDuration = 1 * time.Millisecond
	}

	p.StartHoldTimer(ts, holdDuration)
	// 设置限价单
	// buyPrice, amount := tools.GetBuyPriceAndAmount(resp, tokenAddress)
	// token.buyPrice = buyPrice
	// token.buyAmount = amount
	// token.remainingAmount.Add(amount)

	// if buyPrice != 0 && amount != nil {

	go p.SetLimitSell(ts)

	//提前卖出
	go p.ListenSell(ts)
	// //开发者卖出
	// go p.ListenWatchSell(ts)

	// }
}
