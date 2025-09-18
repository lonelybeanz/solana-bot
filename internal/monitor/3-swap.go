package monitor

import (
	"context"
	"fmt"
	"math/big"
	"solana-bot/internal/config"
	"solana-bot/internal/dex/pump"
	"solana-bot/internal/global"
	"solana-bot/internal/stream"
	"sync"
	"sync/atomic"
	"time"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"

	"github.com/gagliardetto/solana-go/rpc"

	atomic_ "solana-bot/internal/global/utils/atomic"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"slices"

	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	PumpFunType          = int32(1)
	PumpAmmType          = int32(2)
	JupiterType          = int32(3)
	MeteoraDbcType       = int32(4)
	RaydiumLaunchpadType = int32(5)
	PumpSwapType         = map[string]int32{
		"PumpFun":          PumpFunType,
		"PumpAmm":          PumpAmmType,
		"Jupiter":          JupiterType,
		"MeteoraDbc":       MeteoraDbcType,
		"RaydiumLaunchpad": RaydiumLaunchpadType,
	}
)

var (
	holdInfoMap   = make(map[string]*TokenHoldInfo)
	holdInfoMutex sync.RWMutex
)

// 代币本身的链上和市场状态
type TokenInfo struct {
	TokenAddress     string
	PoolData         *solanaswapgo.PoolData
	BondingCurveData atomic.Value
	PoolTokenBalance atomic.Uint64
	PoolSolBalance   atomic.Uint64
	TokenPrice       atomic_.BigFloat // 当前估算价格
}

// 作为狙击者的状态和行为
type MySwapState struct {
	SellMu            sync.Mutex
	AtaAddress        atomic_.String
	BuyAmount         atomic_.BigInt
	BuyTime           atomic_.BigInt
	BuyPrice          atomic_.BigFloat
	SellPrice         atomic_.BigFloat
	RemainingAmount   atomic_.BigInt
	BuyBalanceChange  atomic_.BigFloat
	SellBalanceChange []*big.Float
}

func (m *MySwapState) AppendSellBalanceChange(f *big.Float) {
	m.SellMu.Lock()
	defer m.SellMu.Unlock()
	m.SellBalanceChange = append(m.SellBalanceChange, f)
}

func (m *MySwapState) SnapshotSellBalanceChange() []*big.Float {
	m.SellMu.Lock()
	defer m.SellMu.Unlock()
	return append([]*big.Float(nil), m.SellBalanceChange...)
}

// 监控的钱包行为数据
type TrackedWalletInfo struct {
	TrackedAddress  []string
	InToken         string
	BuyAmount       *big.Int
	RemainingAmount atomic_.BigInt
}

type TokenHoldInfo struct {
	TokenAddress string
	Remaining    atomic.Value // time.Duration
	ExtendChan   chan time.Duration
	StopChan     chan struct{}
	Done         atomic.Bool
}

type TokenSwap struct {
	streams         []*stream.GrpcStream
	Ctx             context.Context
	Cancel          context.CancelFunc
	Cmd             chan string
	IsMint          bool
	SwapType        atomic.Int32
	BundleTx        string
	readyToSell     atomic.Bool
	Token           *TokenInfo
	MySwap          *MySwapState
	Tracked         *TrackedWalletInfo
	FollowChan      chan *solanaswapgo.SwapInfo
	SellSignalCount atomic.Int32 // 新增字段，用于记录卖出信号次数
}

func NewTokenJupiterSwap(tokenAddress string) *TokenSwap {

	ctx, cancel := context.WithCancel(context.Background())
	ts := &TokenSwap{
		Ctx:    ctx,
		Cancel: cancel,
		Cmd:    make(chan string),
		Token: &TokenInfo{
			TokenAddress: tokenAddress,
		},
		MySwap:     &MySwapState{},
		Tracked:    &TrackedWalletInfo{},
		FollowChan: make(chan *solanaswapgo.SwapInfo, 100),
	}
	ts.SwapType.Store(JupiterType)
	return ts
}

func NewTokenSwap(isMint bool, bundleTx, tokenAddress string, trackedAddress []string, ata string, poolData *solanaswapgo.PoolData) *TokenSwap {

	streams := []*stream.GrpcStream{
		stream.NewBlzStream(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	ts := &TokenSwap{
		IsMint:   isMint,
		BundleTx: bundleTx,
		streams:  streams,
		Ctx:      ctx,
		Cancel:   cancel,
		Cmd:      make(chan string),
		Token: &TokenInfo{
			TokenAddress: tokenAddress,
			PoolData:     poolData,
		},
		MySwap: &MySwapState{
			AtaAddress: *atomic_.NewString(ata),
		},
		Tracked: &TrackedWalletInfo{
			TrackedAddress: trackedAddress,
		},
		FollowChan: make(chan *solanaswapgo.SwapInfo, 100),
	}

	if poolData == nil {
		return ts
	}

	ts.SwapType.Store(PumpSwapType[poolData.PoolType])

	switch ts.SwapType.Load() {
	case PumpFunType:
		fumPool := poolData.Data.(*solanaswapgo.PumpFunPool)
		bondingCurveData := &pump.PUMPBondingCurveData{
			BondingCurve: &pump.BondingCurveLayout{
				VirtualTokenReserves: fumPool.VirtualTokenReserves,
				VirtualSOLReserves:   fumPool.VirtualSolReserves,
				RealTokenReserves:    fumPool.RealTokenReserves,
				RealSOLReserves:      fumPool.RealSOLReserves,
			},
			BondingCurvePk:           fumPool.BondingCurve,
			AssociatedBondingCurvePk: fumPool.AssociatedBondingCurve,
			GlobalSettingsPk:         fumPool.Global,
			MintAuthority:            fumPool.EventAuthority,
		}
		ts.Token.BondingCurveData.Store(bondingCurveData)
		ts.Token.PoolTokenBalance.Store(fumPool.VirtualTokenReserves)
		ts.Token.PoolSolBalance.Store(fumPool.VirtualSolReserves)

	case PumpAmmType:
		ammPool := poolData.Data.(*solanaswapgo.PumpAmmPool)
		ts.Token.PoolTokenBalance.Store(ammPool.PoolBaseTokenReserves)
		ts.Token.PoolSolBalance.Store(ammPool.PoolQuoteTokenReserves)

	case MeteoraDbcType:
		pool := poolData.Data.(*solanaswapgo.MeteoraDbcPool)
		sqrtP := new(big.Float).SetFloat64(float64(pool.NextSqrtPrice))
		ts.Token.TokenPrice.Store(sqrtP)

	case RaydiumLaunchpadType:
		pool := poolData.Data.(*solanaswapgo.RaydiumLaunchpadPool)
		ts.Token.PoolTokenBalance.Store(pool.RealBaseBefore)
		ts.Token.PoolSolBalance.Store(pool.RealQuoteBefore)

	}

	// spew.Dump(ts)
	go ts.SubTokenSwap()

	return ts
}

func (t *TokenSwap) SubTokenSwap() {
	logx.Infof("[%s]:监听Token交易", t.Token.TokenAddress)
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

	subscription.Transactions["transactions_sub"].AccountInclude = []string{t.Token.TokenAddress}

	var once sync.Once
	for _, s := range t.streams {
		s.Subscribe(t.Ctx, &subscription, &once, subscribe)
	}

	deduper := NewTxDeduper(1_000_000, 0.001) // 预计 100 万交易，误判率 0.1%

	for {
		select {
		case <-t.Ctx.Done():
			logx.Infof("[%s] 停止监听Token交易", t.Token.TokenAddress)
			return
		case msg := <-subscribe:
			if msg == nil {
				continue
			}
			// 检查消息是否包含来源信息
			var got *pb.SubscribeUpdate
			v := msg.(*stream.StreamMessage)
			got = v.Data.(*pb.SubscribeUpdate)

			// spew.Dump(got)
			tx := got.GetTransaction()
			if tx == nil || tx.Transaction.Transaction == nil || tx.Transaction.Meta == nil {
				continue
			}

			swapInfo, err := ParseSwapTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
			if err != nil || swapInfo == nil {
				continue
			}

			if deduper.SeenOrAdd(swapInfo.Signatures[0].String()) {
				// logx.Infof("Skip duplicate:%s", swapInfo.Signatures[0].String())
				continue
			}
			// 处理交易
			// logx.Infof("[%s]:Token交易:%v", t.Token.TokenAddress, swapInfo)

			t.OnTrade(swapInfo)

		}

	}
}

func (t *TokenSwap) OnTrade(swapInfo *solanaswapgo.SwapInfo) {
	go t.UpdatePoolData(swapInfo.PoolData)

	select {
	case <-t.Ctx.Done():
		logx.Infof("[%s]:TokenSwap ctx 已取消，停止处理交易", t.Token.TokenAddress)
		return
	default:
		t.doTrade(swapInfo)
	}
}

// 处理监听到的交易
func (t *TokenSwap) doTrade(swapInfo *solanaswapgo.SwapInfo) {

	buyPrice := t.MySwap.BuyPrice.Load()
	sellPrice := t.MySwap.SellPrice.Load()
	currentPrice := t.Token.TokenPrice.Load()
	buyAmount := t.MySwap.BuyAmount.Load()

	// logx.Infof("[%s]-「%s」 监听到买入/卖出 %d token,价格 %v", t.Token.TokenAddress, swapInfo.Signatures[0].String(), swapInfo.TokenInAmount, currentPrice)

	buyer := swapInfo.Signers[0].String()

	if buyer == config.C.Bot.Player {
		if swapInfo.TokenInMint.String() == global.Solana {
			t.readyToSell.Store(true)
			fmt.Printf("[准备卖出] 我在 %s 买入 %d token\n", time.Now().Format(time.RFC822), swapInfo.TokenOutAmount)
			t.MySwap.BuyAmount.Store(big.NewInt(int64(swapInfo.TokenOutAmount)))
			t.MySwap.BuyTime.Store(big.NewInt(time.Now().Unix()))
		}
		return
	}

	//买入
	if swapInfo.TokenInMint.String() == global.Solana {
		t.FollowChan <- swapInfo
	}

	if !t.readyToSell.Load() {
		return
	}

	// 在止损部分增加计数器逻辑
	if sellPrice != nil && currentPrice.Cmp(new(big.Float).Mul(sellPrice, big.NewFloat(0.8))) < 0 {
		// 当前价格 < 卖出价格的 80%，触发卖出
		logx.Infof("[%s] ⛔ 部分止损 ", t.Token.TokenAddress)
		t.Cmd <- "sell"
		t.readyToSell.Store(false)
		return

	}

	//监听钱包卖出
	if slices.Contains(t.Tracked.TrackedAddress, buyer) {
		if swapInfo.TokenOutMint.String() == global.Solana {

			if buyPrice != nil && currentPrice != nil && currentPrice.Cmp(new(big.Float).Mul(buyPrice, big.NewFloat(1.5))) > 0 {
				// 回本
				t.Cmd <- "break-even"
				return
			}

			logx.Infof("[%s] ‼️ 监听到钱包卖出，立即跟随 ", t.Token.TokenAddress)
			logx.Infof("[%s] ‼️ 跟随钱包持有 %d，监听钱包卖出部分 %d ", t.Token.TokenAddress, t.Tracked.RemainingAmount.Load(), swapInfo.TokenInAmount)
			if t.Tracked.RemainingAmount.Load().Cmp(big.NewInt(int64(swapInfo.TokenInAmount))) <= 0 {
				logx.Infof("[%s] ‼️ 跟随钱包卖出，监听钱包卖出全部 ", t.Token.TokenAddress)

				t.Cmd <- "sell"
				t.readyToSell.Store(false)
				return
			}
			logx.Infof("[%s] ‼️ 跟随钱包卖出，监听钱包卖出部分 ", t.Token.TokenAddress)
			t.SellSignalCount.Add(1)
			if t.SellSignalCount.Load() >= 2 {
				t.Cmd <- "sell"
				t.readyToSell.Store(false)
				return
			}
		}

	}

	if swapInfo.TokenOutMint.String() == global.Solana { //卖出逻辑
		if buyPrice != nil && currentPrice != nil && currentPrice.Cmp(new(big.Float).Mul(buyPrice, big.NewFloat(1.5))) > 0 {
			// 回本
			t.Cmd <- "break-even"
			return
		}

		//卖出数量比我多
		// if big.NewInt(int64(swapInfo.TokenOutAmount)).Cmp(buyAmount) > 0 {
		// 	fmt.Printf("[卖出触发] %s 在 %s 卖出 %d > 我的 %s，立即卖出\n",
		// 		buyer, time.Now().Format(time.RFC822), swapInfo.TokenOutAmount, buyAmount.String())
		// 	// 卖出
		// 	t.Cmd <- "sell"
		// 	// 重置状态
		// 	t.readyToSell.Store(false)
		// 	return
		// }

	}

	//持续买入 增加时间
	if big.NewInt(int64(swapInfo.TokenOutAmount)).Cmp(buyAmount) > 0 {
		// 获取当前交易的 TokenInAmount 作为 big.Int
		amount := big.NewInt(int64(swapInfo.TokenInAmount))

		// 计算比例因子：ratio = amount / t.MySwap.BuyAmount
		ratioF, _ := new(big.Float).Quo(new(big.Float).SetInt(amount), new(big.Float).SetInt(buyAmount)).Float64()

		// 按照比例计算要增加的时间
		duration := time.Duration(float64(100) * (1 + ratioF) * float64(time.Millisecond))

		ExtendHoldDuration(t.Token.TokenAddress, duration)

	}

	//持续卖出，减少持仓时间
	if swapInfo.TokenOutMint.String() == global.Solana {
		// 获取当前交易的 TokenInAmount 作为 big.Int
		amount := big.NewInt(int64(swapInfo.TokenInAmount))

		// 计算比例因子：ratio = amount / t.MySwap.BuyAmount
		ratioF, _ := new(big.Float).Quo(new(big.Float).SetInt(amount), new(big.Float).SetInt(buyAmount)).Float64()

		// 按照比例计算要减少的时间，基准为 -800ms
		duration := time.Duration(float64(-200) * ratioF * float64(time.Millisecond))

		ExtendHoldDuration(t.Token.TokenAddress, duration)

	}

	//新币策略
	if t.IsMint {

		// 启动 1 秒内监控跟单逻辑
		go func(buySig string, buyTime time.Time) {
			monitorTime := 1200 * time.Millisecond
			timer := time.NewTimer(monitorTime)
			defer timer.Stop()
			for {
				select {
				case <-timer.C:
					logx.Infof("[%s] ⛔ 买入后 %d 毫秒无人跟单，考虑止损", t.Token.TokenAddress, monitorTime.Milliseconds())
					t.Cmd <- "stop"
					t.readyToSell.Store(false)
					return
				case follow := <-t.FollowChan:
					if follow.Signers[0].String() != buyer && follow.Timestamp.After(buyTime) && follow.Timestamp.Before(buyTime.Add(1*time.Second)) {
						logx.Infof("[%s] ✅ 有人跟单买入，继续持有", t.Token.TokenAddress)
						return
					}
				}
			}
		}(swapInfo.Signatures[0].String(), time.Now())

		robotCfg, _ := GetRobotConfig()
		if global.Contains(robotCfg.Robot, buyer) { //已被狙击
			t.Cmd <- "stop"
			return
		}

		if swapInfo.TokenOutMint.String() != global.Solana { //第一次有人买就卖
			if buyPrice != nil && currentPrice != nil && currentPrice.Cmp(new(big.Float).Mul(buyPrice, big.NewFloat(1.5))) > 0 {
				// 回本
				t.Cmd <- "break-even"
				return
			}
			// 卖出
			t.Cmd <- "sell"
			// 重置状态
			t.readyToSell.Store(false)
			return
		}

	}

}

func ExtendHoldDuration(tokenAddress string, additionalTime time.Duration) {
	holdInfoMutex.RLock()
	info, exists := holdInfoMap[tokenAddress]
	holdInfoMutex.RUnlock()

	if exists && !info.Done.Load() {
		info.ExtendChan <- additionalTime
	}
}

func (t *TokenSwap) UpdatePoolData(poolData *solanaswapgo.PoolData) {
	if poolData == nil {
		return
	}
	switch pool := poolData.Data.(type) {
	case *solanaswapgo.PumpFunPool:
		t.UpdateBondingCurve(pool.VirtualSolReserves, pool.VirtualTokenReserves, pool.RealSOLReserves, pool.RealTokenReserves)
		t.SwapType.Store(PumpFunType)
	case *solanaswapgo.PumpAmmPool:
		t.UpdateAmmPool(pool.PoolBaseTokenReserves, pool.PoolQuoteTokenReserves)
		t.SwapType.Store(PumpAmmType)
	case *solanaswapgo.MeteoraDbcPool:
		t.UpdatePricePool(pool.NextSqrtPrice)
		t.SwapType.Store(MeteoraDbcType)
	case *solanaswapgo.RaydiumLaunchpadPool:
		t.UpdateAmmPool(pool.RealBaseBefore, pool.RealQuoteBefore)
		t.SwapType.Store(RaydiumLaunchpadType)
	}

}

func (t *TokenSwap) UpdatePricePool(nextSqrtPrice uint64) {
	sqrtP := new(big.Float).SetFloat64(float64(nextSqrtPrice))
	t.Token.TokenPrice.Store(sqrtP)
}

func (t *TokenSwap) UpdateAmmPool(baseBalance, quoteBalance uint64) {

	t.Token.PoolTokenBalance.Store(quoteBalance)

	t.Token.PoolSolBalance.Store(baseBalance)

	// 原始数据
	solReserves := new(big.Float).SetUint64(t.Token.PoolSolBalance.Load())
	lamportsPerSol := new(big.Float).SetUint64(solana.LAMPORTS_PER_SOL)
	tokenReserves := new(big.Float).SetUint64(t.Token.PoolTokenBalance.Load())

	// 计算：VirtualSOLReserves / LAMPORTS_PER_SOL / VirtualTokenReserves * 1e6
	var solTokenPrice big.Float
	solTokenPrice.Quo(solReserves, lamportsPerSol)       // SOL数量
	solTokenPrice.Quo(&solTokenPrice, tokenReserves)     // 单位token价格
	solTokenPrice.Mul(&solTokenPrice, big.NewFloat(1e6)) // 转换为每百万单位token的价格

	t.Token.TokenPrice.Store(&solTokenPrice)
}

func (t *TokenSwap) UpdateBondingCurve(virtualSolReserves, virtualTokenReserves, realSolReserves, realTokenReserves uint64) {

	data, ok := t.Token.BondingCurveData.Load().(*pump.PUMPBondingCurveData)
	if ok {
		data.BondingCurve = &pump.BondingCurveLayout{
			VirtualTokenReserves: virtualTokenReserves,
			VirtualSOLReserves:   virtualSolReserves,
			RealTokenReserves:    realTokenReserves,
			RealSOLReserves:      realSolReserves,
		}

		t.Token.BondingCurveData.Store(data)
	}

	t.Token.PoolTokenBalance.Store(realTokenReserves)
	t.Token.PoolSolBalance.Store(realSolReserves)

	lamportsPerSol := new(big.Float).SetUint64(solana.LAMPORTS_PER_SOL)

	// 计算：VirtualSOLReserves / LAMPORTS_PER_SOL / VirtualTokenReserves * 1e6
	var solTokenPrice big.Float
	solTokenPrice.Quo(new(big.Float).SetUint64(realSolReserves), lamportsPerSol)   // SOL数量
	solTokenPrice.Quo(&solTokenPrice, new(big.Float).SetUint64(realTokenReserves)) // 单位token价格
	solTokenPrice.Mul(&solTokenPrice, big.NewFloat(1e6))                           // 转换为每百万单位token的价格

	t.Token.TokenPrice.Store(&solTokenPrice)

}

func (t *TokenSwap) GetBondingCurveData() *pump.PUMPBondingCurveData {
	data, _ := t.Token.BondingCurveData.Load().(*pump.PUMPBondingCurveData)
	return data

}

func (t *TokenSwap) UpdateRemainingAmount(newAmount *big.Int) {
	t.MySwap.RemainingAmount.Store(newAmount)
}

func (t *TokenSwap) GetRemainingAmount() *big.Int {

	go func() {
		ata := solana.MustPublicKeyFromBase58(t.MySwap.AtaAddress.Load())
		balance, err := global.GetRPCForRequest().GetTokenAccountBalance(context.Background(), ata, rpc.CommitmentProcessed)
		if err != nil {
			if strings.Contains(err.Error(), "-32602") {
				t.UpdateRemainingAmount(big.NewInt(0))
				return
			}
			logx.Error("Failed to query token balance", err)
		}
		if err == nil && balance != nil && balance.Value != nil {
			wDecimals, _ := new(big.Int).SetString(balance.Value.Amount, 10)
			t.UpdateRemainingAmount(wDecimals)
		}
	}()

	return t.MySwap.RemainingAmount.Load()
}
