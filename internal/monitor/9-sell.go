package monitor

import (
	"errors"
	"fmt"
	"math/big"
	"solana-bot/internal/client"
	"solana-bot/internal/dex/meteora"
	dex "solana-bot/internal/dex/okx"
	"solana-bot/internal/dex/pump"
	"solana-bot/internal/dex/raydium"
	"solana-bot/internal/global"
	"solana-bot/internal/shot"
	"solana-bot/internal/stream"
	"strconv"
	"sync"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"

	"strings"
	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	token_program "github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	maxRetries = 10 // 最大重试次数
)

func (p *PumpFunMonitor) SellDone(ts *TokenSwap, resp *rpc.GetTransactionResult) {
	logx.Infof("[%s]: 🈹 卖出成功!", ts.Token.TokenAddress)

	go func() {
		time.Sleep(8 * time.Second)
		remaining := ts.GetRemainingAmount()
		if remaining.Cmp(big.NewInt(0)) <= 0 {
			logx.Infof("[%s]: Ⓜ️ sell done ", ts.Token.TokenAddress)
			profit := new(big.Float).Add(ts.MySwap.BuyBalanceChange.Load(), big.NewFloat(0))
			for _, s := range ts.MySwap.SnapshotSellBalanceChange() {
				profit = profit.Add(profit, s)
			}
			buy := new(big.Float).Quo(ts.MySwap.BuyBalanceChange.Load(), big.NewFloat(1e9))
			log := fmt.Sprintf("%s,%s,%s,%s", time.Now().Format(time.DateTime), ts.Token.TokenAddress, buy.String(), new(big.Float).Quo(profit, big.NewFloat(1e9)).String())
			p.pubsub.Publish(log)

			ts.Cancel()
			buyCount.Decrement()
		}
	}()

	if _, b := BuyCache.Get(ts.Token.TokenAddress); b {
		BuyCache.Delete(ts.Token.TokenAddress)
	}

	if resp == nil {
		ts.Cancel()
		buyCount.Decrement()
		return
	}

	sellPrice, _ := global.GetBuyPriceAndAmount(resp, ts.Token.TokenAddress)
	ts.MySwap.SellPrice.Store(big.NewFloat(sellPrice))

	costSol, _ := global.GetBalacneChange(resp)
	ts.MySwap.AppendSellBalanceChange(big.NewFloat(costSol))

}

func (p *PumpFunMonitor) ListenWatchSell(ts *TokenSwap) {
	tokenAddress := ts.Token.TokenAddress

	logx.Infof("[%s]:监听钱包卖出", tokenAddress)

	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	subscription.Transactions = make(map[string]*pb.SubscribeRequestFilterTransactions)
	failed := false
	vote := false
	subscription.Transactions["transactions_sub"] = &pb.SubscribeRequestFilterTransactions{
		Failed: &failed,
		Vote:   &vote,
	}

	subscription.Transactions["transactions_sub"].AccountInclude = ts.Tracked.TrackedAddress
	// subscription.Transactions["transactions_sub"].AccountExclude = transactionsAccountsExclude
	var once sync.Once
	for _, s := range p.streams {
		s.Subscribe(p.ctx, &subscription, &once, subscribe)
	}

	p.runWithCtx(ts.Ctx, subscribe, func(msg interface{}) {
		var got *pb.SubscribeUpdate
		v := msg.(*stream.StreamMessage)
		got = v.Data.(*pb.SubscribeUpdate)
		tx := got.GetTransaction()
		if tx == nil {
			return
		}

		swapInfo, err := ParseSwapTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
		if err != nil || swapInfo == nil {
			return
		}
		if swapInfo.TokenInMint.String() == tokenAddress {
			logx.Infof("[%s]:发现监听钱包卖出,数量:%d", tokenAddress, swapInfo.TokenInAmount)

			ts.Tracked.RemainingAmount.Sub(big.NewInt(int64(swapInfo.TokenInAmount)))

			err := p.ExecuteSell(ts, ts.GetRemainingAmount())
			if err != nil {
				logx.Errorf("[%s]:跟随卖出失败: %v", tokenAddress, err)
			} else {
				logx.Infof("[%s]:跟随卖出成功", tokenAddress)
			}
			return

		}
	})

}

func (p *PumpFunMonitor) ListenSell(ts *TokenSwap) {
	tokenAddress := ts.Token.TokenAddress
	for {
		select {
		case <-ts.Ctx.Done():
			logx.Infof("[%s]:停止提前卖出监听", tokenAddress)
			return
		case msg := <-ts.Cmd:
			if msg == "break-even" {
				logx.Infof("[%s] 🌫 收到回本信号，快速卖出", tokenAddress)
				amountBought := ts.GetRemainingAmount()
				buyPrice := ts.MySwap.BuyPrice.Load()
				currentPrice := ts.Token.TokenPrice.Load()
				//卖出价值回本的金额
				amountToSell := calculateBreakEvenAmount(
					buyPrice,
					currentPrice,
					amountBought,
				)
				_ = p.ExecuteSell(ts, amountToSell)
				continue
			}

			if msg == "sell-some" {
				logx.Infof("[%s] 🌫 收到部分信号，快速卖出", tokenAddress)
				amountBought := ts.GetRemainingAmount()
				_ = p.ExecuteSell(ts, new(big.Int).Quo(amountBought, big.NewInt(3)))
				continue
			}

			amount := ts.GetRemainingAmount()
			logx.Infof("[%s]⛔ 收到停止信号，快速卖出", tokenAddress)
			_ = p.ExecuteSell(ts, amount)
			return
		default:
			balance, _ := pump.GetTokenBalance(p.httpClient, p.wallet.PublicKey(), solana.MustPublicKeyFromBase58(tokenAddress))
			if balance != nil && balance.Cmp(big.NewInt(0)) == 0 {
				logx.Infof("[%s]:提前卖出成功", tokenAddress)
				p.SellDone(ts, nil)
				return
			}
			time.Sleep(time.Second)
		}
	}
}

func (p *PumpFunMonitor) StartHoldTimer(t *TokenSwap, initialDuration time.Duration) {
	tokenAddress := t.Token.TokenAddress
	logx.Infof("[%s]:开始持有时间计时器，总时间 %v", tokenAddress, initialDuration)
	if initialDuration <= 0 {
		return
	}

	holdInfoMutex.Lock()
	if existing, exists := holdInfoMap[tokenAddress]; exists {
		close(existing.StopChan)
		delete(holdInfoMap, tokenAddress)
	}

	info := &TokenHoldInfo{
		TokenAddress: tokenAddress,
		ExtendChan:   make(chan time.Duration, 10),
		StopChan:     make(chan struct{}),
	}
	info.Remaining.Store(initialDuration)
	holdInfoMap[tokenAddress] = info
	holdInfoMutex.Unlock()

	go func() {
		defer func() {
			holdInfoMutex.Lock()
			delete(holdInfoMap, tokenAddress)
			holdInfoMutex.Unlock()
		}()

		sellStep := 0
		totalSteps := 3

		// 初始分配每段时间
		stepDurations := make([]time.Duration, totalSteps)
		for i := range stepDurations {
			stepDurations[i] = initialDuration / time.Duration(totalSteps)
		}

		timer := time.NewTimer(stepDurations[0])
		defer timer.Stop()

		for {
			select {
			case <-info.StopChan:
				logx.Infof("[%s]:提前终止持有", tokenAddress)
				return

			case extend := <-info.ExtendChan:
				// 延长当前 step 和总剩余时间
				info.Remaining.Store(info.Remaining.Load().(time.Duration) + extend)
				stepDurations[sellStep] += extend
				logx.Infof("[%s]:延长当前阶段持有时间 %v，新剩余 %v", tokenAddress, extend, stepDurations[sellStep])

				// 重设当前计时器
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(stepDurations[sellStep])

			case <-timer.C:
				var amount *big.Int
				switch sellStep {
				case 0:
					amount = new(big.Int).Div(t.GetRemainingAmount(), big.NewInt(3))
				case 1:
					amount = new(big.Int).Div(t.GetRemainingAmount(), big.NewInt(2))
				case 2:
					amount = t.GetRemainingAmount()
				}

				err := p.ExecuteSell(t, amount)
				logx.Infof("[%s]:第 %d 次卖出 %v, err: %v", tokenAddress, sellStep+1, amount, err)

				sellStep++
				if sellStep >= totalSteps {
					logx.Infof("[%s]:完成所有卖出阶段", tokenAddress)
					return
				}

				// 自动缩短下一阶段时间：剩余时间 / 剩余阶段（可选逻辑）
				stepDurations[sellStep] = stepDurations[sellStep] / 2
				timer.Reset(stepDurations[sellStep])
			}
		}
	}()
}

func (p *PumpFunMonitor) ExecuteSell(ts *TokenSwap, amount *big.Int) error {
	if amount == nil || amount.Cmp(big.NewInt(0)) <= 0 {
		return nil
	}
	tokenAddress := ts.Token.TokenAddress
	logx.Infof("[%s]:开始执行卖出, 数量: %d", tokenAddress, amount.Int64())

	retryCount := 0
	slippage := float32(10)
	var err error
	for retryCount < maxRetries {

		select {
		case <-ts.Ctx.Done():
			logx.Info("收到停止信号，停止卖出操作")
			return nil
		default:
			resp, err := p.SellToken(ts, amount, slippage)
			if err == nil && resp != nil && resp.Meta != nil && resp.Meta.Err == nil {
				p.SellDone(ts, resp)
				return nil
			}

			logx.Errorf("[%s] 卖出失败,err:%v", tokenAddress, err)

			if strings.Contains(err.Error(), ":6023") {
				amount = ts.GetRemainingAmount()
				resp, err := p.SellToken(ts, amount, slippage)
				if err == nil && resp != nil && resp.Meta != nil && resp.Meta.Err == nil {
					p.SellDone(ts, resp)
					return nil
				}
			}

			if strings.Contains(err.Error(), ":38") || strings.Contains(err.Error(), ":3012") {
				p.SellDone(ts, resp)
				return nil
			}

			if strings.Contains(err.Error(), ":6001") ||
				strings.Contains(err.Error(), ":6002") ||
				strings.Contains(err.Error(), ":6003") ||
				strings.Contains(err.Error(), ":6004") {
				// 增加50%的滑点
				slippage += slippage * 0.5
				continue
			}

			if strings.Contains(err.Error(), ": 429") {
				time.Sleep(2 * time.Second)
				continue
			}

			// {
			// 	ts.SwapType.Store(JupiterType)
			// 	resp, err = p.sellWithJupiter(ts, amount, slippage)
			// 	if err == nil {
			// 		p.SellDone(ts, resp)
			// 		return nil
			// 	}
			// }

			logx.Infof("[%s]:retry sell %d  %f %s %d", tokenAddress, amount, slippage, err.Error(), retryCount)
			amount = ts.GetRemainingAmount()
			retryCount++
			if retryCount < maxRetries {
				time.Sleep(500 * time.Millisecond)
			}

		}
	}

	logx.Errorf("[%s]:卖出失败，重试次数达到上限,err: %v", tokenAddress, err)

	resp, err := p.sellWithJupiter(ts, amount, float32(200))
	if err == nil {
		p.SellDone(ts, resp)
		return nil
	}
	return errors.New("卖出失败")
}

func (p *PumpFunMonitor) SellToken(ts *TokenSwap, amountIn *big.Int, slippage float32) (*rpc.GetTransactionResult, error) {
	// 修复：避免复制包含锁的对象
	remaining := ts.GetRemainingAmount()
	newAmount := new(big.Int).Sub(remaining, amountIn)
	shouldCloseTokenAccount := newAmount.Cmp(big.NewInt(0)) <= 0
	logx.Infof("[%s]:卖出 %v, 剩余: %v, SwapType: %d", ts.Token.TokenAddress, amountIn, newAmount, ts.SwapType.Load())
	if remaining.Cmp(big.NewInt(0)) <= 0 {
		logx.Infof("[%s]:卖出完成", ts.Token.TokenAddress)
		p.SellDone(ts, nil)
	}
	go func() {
		go p.sellWithOkx(ts, amountIn, slippage)
		go p.sellWithJupiter(ts, amountIn, slippage)
	}()

	switch ts.SwapType.Load() {
	case PumpFunType:
		return p.sellWithFun(ts, amountIn, slippage, shouldCloseTokenAccount)
	case PumpAmmType:
		return p.sellWithAmm(ts, amountIn, float64(slippage), shouldCloseTokenAccount)
	case MeteoraDbcType:
		return p.sellWithMeteoraDbc(ts, amountIn, float64(slippage))
	case RaydiumLaunchpadType:
		return p.sellWithRaydiumLaunchpad(ts, amountIn, float64(slippage), shouldCloseTokenAccount)
	default:
		return p.sellWithJupiter(ts, amountIn, slippage)
	}

}

func (p *PumpFunMonitor) sellWithFun(ts *TokenSwap, amountIn *big.Int, slippage float32, shouldCloseTokenAccount bool) (*rpc.GetTransactionResult, error) {
	mintAddress := ts.Token.TokenAddress
	tokenMint := solana.MustPublicKeyFromBase58(mintAddress)

	priorityFee := global.GetMedium()

	bondingCurveData := ts.GetBondingCurveData()
	if bondingCurveData == nil {
		return nil, errors.New("bondingCurveData is nil")
	}
	poolData := ts.Token.PoolData.Data.(*solanaswapgo.PumpFunPool)

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	punpFunAdapter := shot.NewPumpFunAdapter()
	sellIns := punpFunAdapter.BuildInstructions(
		&shot.TxContext{
			SignerAndOwner:       p.wallet.PrivateKey,
			SrcMint:              tokenMint,
			DstMint:              solana.WrappedSol,
			VirtualSolReserves:   big.NewInt(int64(bondingCurveData.BondingCurve.VirtualSOLReserves)),
			VirtualTokenReserves: big.NewInt(int64(bondingCurveData.BondingCurve.VirtualTokenReserves)),
			MaxAmountIn:          amountIn.Uint64(),
			Slippage:             slippage,
			PriorityFee:          priorityFee,
			Fee:                  0,
		},
		nonceAccount,
		poolData.CreatorVault,
		poolData.Global,
		poolData.BondingCurve,
		poolData.AssociatedBondingCurve,
	)

	txBuilder := global.NewTxBuilder(p.wallet.PublicKey(), nonceHash)
	txBuilder.AddInstruction(sellIns...)

	return p.SendAndWait2(mintAddress, uint64(1e5), txBuilder)
}

func (p *PumpFunMonitor) sellWithJupiter(ts *TokenSwap, maxAmountIn *big.Int, slippage float32) (*rpc.GetTransactionResult, error) {
	var (
		BribeAmount  = uint64(1e6)
		BribeAccount = solana.MustPublicKeyFromBase58("AP6qExwrbRgBAVaehg4b5xHENX815sMabtBzUzVB4v8S")
	)
	mintAddress := ts.Token.TokenAddress
	slippageBps := uint64(slippage * 100)
	q, err := client.Quote(mintAddress, global.Solana, maxAmountIn.Uint64(), slippageBps)
	if err != nil {
		logx.Errorf("Error creating quote transaction: %s", err)
		return nil, err
	}

	r, err := client.SwapInstructions(q, p.wallet.PublicKey().String())
	if err != nil {
		logx.Errorf("Error creating swap transaction: %s", err)
		return nil, err
	}
	// Create an array of instructions
	var instructions []solana.Instruction

	// Add compute budget instruction if present
	for _, instruction := range r.ComputeBudgetInstructions {
		instructions = append(instructions, createTransactionInstruction(instruction))
	}

	// Add setup instructions
	for _, instruction := range r.SetupInstructions {
		instructions = append(instructions, createTransactionInstruction(instruction))
	}

	// Add main swap instruction
	instructions = append(instructions, createTransactionInstruction(*r.SwapInstruction))

	// Add cleanup instruction if present
	// instructions = append(instructions, createTransactionInstruction(*r.CleanupInstruction))

	instructions = append(instructions, system.NewTransferInstruction(BribeAmount, p.wallet.PublicKey(), BribeAccount).Build())

	ata, _, _ := solana.FindAssociatedTokenAddress(p.wallet.PublicKey(), solana.MustPublicKeyFromBase58(mintAddress))
	closeInst := token_program.NewCloseAccountInstruction(
		ata,
		p.wallet.PublicKey(),
		p.wallet.PublicKey(),
		[]solana.PublicKey{},
	).Build()
	instructions = append(instructions, closeInst)

	// Create the transaction with all instructions
	blockHash := global.GetBlockHash()
	tx, err := solana.NewTransaction(
		instructions,
		blockHash,
		solana.TransactionPayer(p.wallet.PublicKey()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			return &p.wallet.PrivateKey
		},
	)
	if err != nil {
		logx.Errorf("Error signing transaction: %v", err)
		return nil, err
	}

	txHash := tx.Signatures[0].String()

	logx.Infof("[%s]:开始卖出, 数量: %d, 滑点:%.10f,txHash:%s", ts.Token.TokenAddress, maxAmountIn.Int64(), slippage, txHash)

	return p.SendAndWait(tx, true)
}

func (p *PumpFunMonitor) sellWithAmm(ts *TokenSwap, amountIn *big.Int, slippage float64, shouldCloseTokenAccount bool) (*rpc.GetTransactionResult, error) {
	mintAddress := ts.Token.TokenAddress
	priorityFee := global.GetMedium() // 0.0005 SOL
	fee := uint64(10e4)               // 基础费用自动扣除
	jitoTip := uint64(0)              // 0.001 SOL（如果使用 Jito）

	poolData := ts.Token.PoolData.Data.(*solanaswapgo.PumpAmmPool)
	sellTx, err := pump.GetPumpAMMSellTx(
		&p.wallet.PrivateKey,
		poolData.Pool,
		poolData.GlobalConfig,
		poolData.BaseMint,
		poolData.QuoteMint,
		poolData.PoolBaseTokenAccount,
		poolData.PoolQuoteTokenAccount,
		poolData.ProtocolFeeRecipient,
		poolData.ProtocolFeeRecipientTokenAccount,
		poolData.CoinCreatorVaultAta,
		poolData.CoinCreatorVaultAuthority,
		new(big.Float).Quo(big.NewFloat(float64(ts.Token.PoolTokenBalance.Load())), big.NewFloat(1e6)),
		new(big.Float).Quo(big.NewFloat(float64(ts.Token.PoolSolBalance.Load())), big.NewFloat(1e9)),
		amountIn,
		slippage,
		priorityFee,
		fee,
		jitoTip,
		shouldCloseTokenAccount,
	)
	if err != nil {
		logx.Errorf("[%s]:获取卖出交易失败: %v", mintAddress, err)
		return nil, err
	}

	txHash := sellTx.Signatures[0].String()

	logx.Infof("[%s]:开始卖出, 数量: %d, 滑点:%.10f,txHash:%s", ts.Token.TokenAddress, amountIn.Int64(), slippage, txHash)

	return p.SendAndWait(sellTx, true)

}

func (p *PumpFunMonitor) sellWithMeteoraDbc(ts *TokenSwap, maxAmountIn *big.Int, slippage float64) (*rpc.GetTransactionResult, error) {
	priorityFee := global.GetMedium() // 0.0005 SOL
	fee := uint64(10e4)               // 基础费用自动扣除
	jitoTip := uint64(0)              // 0.001 SOL（如果使用 Jito）

	poolData := ts.Token.PoolData.Data.(*solanaswapgo.MeteoraDbcPool)

	np, _ := ts.Token.TokenPrice.Load().Int64()

	buyTx, err := meteora.GetSellTx(
		&p.wallet.PrivateKey,
		poolData.Config,
		poolData.Pool,
		poolData.BaseVault,
		poolData.QuoteVault,
		poolData.BaseMint,
		poolData.QuoteMint,
		big.NewInt(np),
		maxAmountIn,
		slippage,
		priorityFee,
		fee,
		jitoTip,
	)

	if err != nil {
		logx.Errorf("[%s]:获取买入交易失败: %v", ts.Token.TokenAddress, err)
		return nil, err
	}

	txHash := buyTx.Signatures[0].String()

	logx.Infof("[%s]:开始买入, 数量: %.10f, 滑点:%.10f,txHash:%s", ts.Token.TokenAddress, maxAmountIn, slippage, txHash)

	return p.SendAndWait(buyTx, true)

}

func (p *PumpFunMonitor) sellWithRaydiumLaunchpad(ts *TokenSwap, maxAmountIn *big.Int, slippage float64, shouldCloseTokenAccount bool) (*rpc.GetTransactionResult, error) {
	mintAddress := ts.Token.TokenAddress

	priorityFee := global.GetMedium() // 0.0005 SOL
	fee := uint64(10e4)               // 基础费用自动扣除
	jitoTip := uint64(0)              // 0.001 SOL（如果使用 Jito）

	poolData := ts.Token.PoolData.Data.(*solanaswapgo.RaydiumLaunchpadPool)
	buyTx, err := raydium.GetSellTx(
		&p.wallet.PrivateKey,
		poolData.GlobalConfig,
		poolData.PlatformConfig,
		poolData.PoolState,
		poolData.BaseVault,
		poolData.QuoteVault,
		poolData.BaseMint,
		poolData.QuoteMint,
		new(big.Float).Quo(big.NewFloat(float64(ts.Token.PoolTokenBalance.Load())), big.NewFloat(1e6)),
		new(big.Float).Quo(big.NewFloat(float64(ts.Token.PoolSolBalance.Load())), big.NewFloat(1e9)),
		maxAmountIn,
		slippage,
		priorityFee,
		fee,
		jitoTip,
		shouldCloseTokenAccount,
	)
	if err != nil {
		logx.Errorf("[%s]:获取卖出交易失败: %v", mintAddress, err)
		return nil, err
	}

	txHash := buyTx.Signatures[0].String()

	logx.Infof("[%s]:开始卖出, 数量: %d, 滑点:%.10f,txHash:%s", ts.Token.TokenAddress, maxAmountIn.Uint64(), slippage, txHash)

	return p.SendAndWait(buyTx, true)
}

func (p *PumpFunMonitor) sellWithOkx(ts *TokenSwap, maxAmountIn *big.Int, slippage float32) (*rpc.GetTransactionResult, error) {
	var (
		BribeAmount  = uint64(1e6)
		BribeAccount = solana.MustPublicKeyFromBase58("AP6qExwrbRgBAVaehg4b5xHENX815sMabtBzUzVB4v8S")
	)

	mintAddress := ts.Token.TokenAddress
	slippageStr := strconv.FormatFloat(float64(slippage/100), 'f', 10, 64)

	dexApi := dex.NewDexAPI()
	r, err := dex.GetSolSwapInstruction(dexApi, p.wallet.PublicKey().String(), mintAddress, global.Solana, maxAmountIn.String(), slippageStr)

	// r, err := client.SwapInstructions(q, p.wallet.PublicKey().String())
	if err != nil {
		logx.Errorf("Error creating swap transaction: %s", err)
		return nil, err
	}
	// Create an array of instructions
	var instructions []solana.Instruction

	// Add compute budget instruction if present
	for _, instruction := range r.InstructionLists {
		instructions = append(instructions, createTransactionInstructionWithOkx(instruction))
	}

	instructions = append(instructions, system.NewTransferInstruction(BribeAmount, p.wallet.PublicKey(), BribeAccount).Build())

	ata, _, _ := solana.FindAssociatedTokenAddress(p.wallet.PublicKey(), solana.MustPublicKeyFromBase58(mintAddress))
	closeInst := token_program.NewCloseAccountInstruction(
		ata,
		p.wallet.PublicKey(),
		p.wallet.PublicKey(),
		[]solana.PublicKey{},
	).Build()
	instructions = append(instructions, closeInst)

	// Create the transaction with all instructions
	blockHash := global.GetBlockHash()
	tx, err := solana.NewTransaction(
		instructions,
		blockHash,
		solana.TransactionPayer(p.wallet.PublicKey()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			return &p.wallet.PrivateKey
		},
	)
	if err != nil {
		logx.Errorf("Error signing transaction: %v", err)
		return nil, err
	}

	txHash := tx.Signatures[0].String()

	logx.Infof("[%s]:开始卖出, 数量: %d, 滑点:%.10f,txHash:%s", ts.Token.TokenAddress, maxAmountIn.Int64(), slippage, txHash)

	return p.SendAndWait(tx, true)
}
