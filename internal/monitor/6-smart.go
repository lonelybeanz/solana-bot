package monitor

import (
	"context"
	"math/big"
	"solana-bot/internal/global"
	"solana-bot/internal/stream"
	"sync"
	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"slices"

	"github.com/gagliardetto/solana-go"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	QuoteMint = []string{
		"So11111111111111111111111111111111111111112",
		"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
	}

	USDC = solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
)

func (p *PumpFunMonitor) workerForSmart() {
	p.ListenSmartAccountTransation()
}

func (p *PumpFunMonitor) ListenSmartAccountTransation() {

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

	smartKey := make([]string, 0)
	for k := range GetSmartAddresses() {
		smartKey = append(smartKey, k)
	}

	subscription.Transactions["transactions_sub"].AccountInclude = smartKey
	// subscription.Transactions["transactions_sub"].AccountExclude = transactionsAccountsExclude
	// go client.Grpc_subscribe(p.grpcClient, &subscription, p.ctx, subscribe)

	var once sync.Once
	for _, s := range p.streams {
		s.Subscribe(p.ctx, &subscription, &once, subscribe)
	}

	p.runWithCtx(context.Background(), subscribe, func(msg interface{}) {

		hour := time.Now().Hour()
		params := GetStrategyParamsByHour(hour)
		if !params.SmartStart {
			return
		}
		// 检查消息是否包含来源信息
		var got *pb.SubscribeUpdate
		v := msg.(*stream.StreamMessage)
		got = v.Data.(*pb.SubscribeUpdate)

		tx := got.GetTransaction()
		if tx == nil {
			return
		}

		if global.Sol_Balance.Load() != 0 && global.Sol_Balance.Load() < 5e8 {
			return
		}

		// logx.Infof("收到交易{%s}", solana.SignatureFromBytes(tx.Transaction.Signature).String())

		swapInfo, err := ParseSwapTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
		if err != nil || swapInfo == nil {
			return
		}

		smart := GetSmartAddresses()[swapInfo.Signers[0].String()]
		logx.Infof("[%s]:收到[%s]的交易{%s}", swapInfo.TokenOutMint, smart, solana.SignatureFromBytes(tx.Transaction.Signature).String())
		if smart == "" {
			return
		}

		smartBuyAmount := swapInfo.TokenInAmount

		//不是买入
		if !slices.Contains(QuoteMint, swapInfo.TokenInMint.String()) {
			return
		}

		// 排除小额交易
		if swapInfo.TokenInMint.Equals(solana.WrappedSol) && smartBuyAmount < 1e9 {
			return
		}
		if swapInfo.TokenInMint.Equals(USDC) {
			if smartBuyAmount < 100e6 {
				return
			}
		}

		if swapInfo.PoolData == nil {
			return
		}

		if _, b := BuyCache.Get(swapInfo.TokenOutMint.String()); b {
			logx.Infof("[%s]:BackRun交易已存在", swapInfo.TokenOutMint.String())
			return
		}

		ata, _, _ := solana.FindAssociatedTokenAddress(p.wallet.PublicKey(), swapInfo.TokenOutMint)
		ts := NewTokenSwap(false, swapInfo.Signatures[0].String(), swapInfo.TokenOutMint.String(), smartKey, ata.String(), swapInfo.PoolData)
		ts.Tracked.InToken = swapInfo.TokenInMint.String()
		ts.Tracked.BuyAmount = big.NewInt(int64(smartBuyAmount))
		ts.Tracked.RemainingAmount.Store(big.NewInt(int64(swapInfo.TokenOutAmount)))

		go p.SmartBackRun(ts, tx.Slot, swapInfo.Signers[0].String())

	})

}

func (p *PumpFunMonitor) SmartBackRun(ts *TokenSwap, slot uint64, smart string) {

	devBuyAmount := ts.Tracked.BuyAmount.Uint64()
	smartInToken := ts.Tracked.InToken
	tokenAddress := ts.Token.TokenAddress
	logx.Infof("[%s]:BackRun Smart 交易 ", tokenAddress)

	// err := mcapCheck(ts)
	// if err != nil {
	// 	logx.Errorf("[%s]:mcapCheck err:%v", tokenAddress, err)
	// 	return
	// }

	// err = holderCheck(tokenAddress, bondingCurveData.BondingCurvePk.String())
	// if err != nil {
	// 	logx.Errorf("[%s]:holderCheck err:%v", tokenAddress, err)
	// 	return
	// }

	solAmount := float64(devBuyAmount) / 1e9
	holdDuration := calculateHoldDuration(solAmount, NegativeCurve)

	err := p.BuyBefore(ts)
	if err != nil {
		logx.Errorf("[%s]:买入校验失败: %v", tokenAddress, err)
		p.BuyError(ts)
		return
	}

	slippage := 0.00002

	buyAmount := calculateDynamicBuyAmount(devBuyAmount)

	if smartInToken == USDC.String() {
		buyAmount = big.NewFloat(float64(50 * 1e6))
	}

	resp, err := p.buyToken(ts, buyAmount, float32(slippage*100))
	if err != nil {
		logx.Errorf("[%s] 买入失败,err:%v", ts.Token.TokenAddress, err)
		p.BuyError(ts)
		return
	}

	p.BuyDone(ts, resp)

	// if smart == "DfMxre4cKmvogbLrPigxmibVTTQDuzjdXojWzjCXXhzj" {
	// 	holdDuration = 400 * time.Millisecond
	// }

	// if resp.Slot-slot > 2 {
	// 	holdDuration = 1 * time.Millisecond
	// }

	logx.Infof("[%s]:将持有 %v 秒 后自动卖出", tokenAddress, holdDuration.Seconds())

	p.normalBackRun(ts, holdDuration)

}

func (p *PumpFunMonitor) normalBackRun(token *TokenSwap, holdDuration time.Duration) {

	go p.ListenSell(token)

	// 开发者卖出
	// go p.ListenWatchSell(token)

	p.StartHoldTimer(token, holdDuration)

}
