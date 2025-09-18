package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"solana-bot/internal/client"
	"solana-bot/internal/global"
	atomic_ "solana-bot/internal/global/utils/atomic"
	"solana-bot/internal/shot"
	"time"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	buyCount = &atomic_.Uint64{}
	buyLimit = uint64(100)
)

const buyCooldown = 2 * time.Second

func (p *PumpFunMonitor) BuyBefore(ts *TokenSwap) error {
	if buyCount.Load() > buyLimit {
		return fmt.Errorf("当前买入数量超过限制，跳过")
	}

	cfg, err := GetMintConfig()
	if err == nil && cfg != nil && len(cfg.Blacklist) > 0 {
		if global.Contains(cfg.Blacklist, ts.Tracked.TrackedAddress[0]) {
			return fmt.Errorf("当前账户为黑名单账户，跳过")
		}
	}

	// if ts.IsMint {
	// 	if _, ok := RobotBuyCache.Get(ts.Token.TokenAddress); ok {
	// 		return fmt.Errorf("已有狙击手，跳过")
	// 	}
	// }

	// go func() {
	// tx, err := client.GetTransactionByHash(p.httpClient, ts.BundleTx)
	// if err == nil {
	// 	if tx != nil {
	// 		return fmt.Errorf("已落后，跳过")
	// 	}
	// }
	// }()

	// 检查是否在冷却期内
	now := time.Now()
	if lastTime, ok := p.lastBuyTime.Load(ts.Token.TokenAddress); ok {
		if now.Sub(lastTime) < buyCooldown {
			logx.Infof("[%s]:距离上次买入不足 %v，跳过", ts.Token.TokenAddress, buyCooldown)
			return fmt.Errorf("[%s]:距离上次买入不足 %v，跳过", ts.Token.TokenAddress, buyCooldown)
		}
	}

	// 更新最后买入时间（预占位）
	p.lastBuyTime.Store(ts.Token.TokenAddress, now)

	// err := RugCheck(token)
	// if err != nil {
	// 	return err
	// }

	// err := mcapCheck(token)
	// if err != nil {
	// 	return err
	// }

	// if err := holderCheck(token); err != nil {
	// 	return err
	// }

	buyCount.Increment()

	return nil
}

func (p *PumpFunMonitor) BuyDone(ts *TokenSwap, resp *rpc.GetTransactionResult) {
	logx.Infof("✅[%s]BuyDone", ts.Token.TokenAddress)
	BuyCache.Set(ts.Token.TokenAddress, true)

	costSol, _ := global.GetBalacneChange(resp)
	buyPrice, amount := global.GetBuyPriceAndAmount(resp, ts.Token.TokenAddress)
	ts.MySwap.BuyPrice.Store(big.NewFloat(buyPrice))
	ts.MySwap.BuyAmount.Store(amount)
	ts.MySwap.RemainingAmount.Store(amount)
	ts.MySwap.BuyBalanceChange.Store(big.NewFloat(costSol))
	ts.Token.TokenPrice.Store(big.NewFloat(buyPrice))

	// 真正买入完成后更新最后买入时间
	p.lastBuyTime.Store(ts.Token.TokenAddress, time.Now())

	data, _ := json.MarshalIndent(ts, "", "  ")
	fmt.Println(string(data))

}
func (p *PumpFunMonitor) BuyError(ts *TokenSwap) {
	ts.Cancel()
	buyCount.Decrement()

	p.lastBuyTime.Delete(ts.Token.TokenAddress)
}

func (p *PumpFunMonitor) buyToken(ts *TokenSwap, maxAmountIn *big.Float, slippage float32) (*rpc.GetTransactionResult, error) {
	maxAmountIn = new(big.Float).Mul(maxAmountIn, big.NewFloat(p.GetMultiplier()))
	if maxAmountIn.Cmp(big.NewFloat(0.05)) <= 0 {
		maxAmountIn = big.NewFloat(0.05)
	}

	in, _ := maxAmountIn.Float64()

	Profit.lastInvestmentAmount.Store(in)

	select {
	case <-ts.Ctx.Done():
		return nil, fmt.Errorf("[%s]:交易取消", ts.Token.TokenAddress)
	case <-ts.Cmd:
		return nil, fmt.Errorf("[%s]:交易取消", ts.Token.TokenAddress)
	default:
		switch ts.SwapType.Load() {
		case PumpFunType:
			return p.buyWithFun(ts, maxAmountIn, slippage)
		case PumpAmmType:
			return p.buyWithAmm(ts, maxAmountIn, slippage)
		case MeteoraDbcType:
			return p.buyWithMeteoraDbc(ts, maxAmountIn, slippage)
		case RaydiumLaunchpadType:
			return p.buyWithRaydiumLaunchpad(ts, maxAmountIn, slippage)
		default:
			return p.buyWithJupiter(ts, maxAmountIn, slippage)
		}
	}

}

func (p *PumpFunMonitor) buyWithFun(ts *TokenSwap, maxAmountIn *big.Float, slippage float32) (*rpc.GetTransactionResult, error) {
	mintAddress := ts.Token.TokenAddress
	tokenMint := solana.MustPublicKeyFromBase58(mintAddress)

	priorityFee := global.GetMedium()
	// fee := uint64(1e6) // 基础费用自动扣除
	tip := global.GetHigh()
	if tip > uint64(1e6) {
		tip = uint64(1e6)
	}
	if tip < uint64(1e6) {
		tip = uint64(1e6)
	}

	amountIn, _ := new(big.Float).Mul(maxAmountIn, global.Float1Lamp).Uint64()

	bondingCurveData := ts.GetBondingCurveData()
	if bondingCurveData == nil {
		return nil, errors.New("bondingCurveData is nil")
	}
	poolData := ts.Token.PoolData.Data.(*solanaswapgo.PumpFunPool)

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	punpFunAdapter := shot.NewPumpFunAdapter()
	buyIns := punpFunAdapter.BuildInstructions(
		&shot.TxContext{
			SignerAndOwner:       p.wallet.PrivateKey,
			SrcMint:              solana.WrappedSol,
			DstMint:              tokenMint,
			VirtualSolReserves:   big.NewInt(int64(poolData.VirtualSolReserves)),
			VirtualTokenReserves: big.NewInt(int64(poolData.VirtualTokenReserves)),
			MaxAmountIn:          amountIn,
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
	txBuilder.AddInstruction(buyIns...)

	return p.SendAndWait2(mintAddress, tip, txBuilder)
}

func (p *PumpFunMonitor) buyWithAmm(ts *TokenSwap, maxAmountIn *big.Float, slippage float32) (*rpc.GetTransactionResult, error) {

	priorityFee := global.GetMedium()
	// fee := uint64(1e6) // 基础费用自动扣除
	tip := global.GetHigh()
	if tip > uint64(1e6) {
		tip = uint64(1e6)
	}
	if tip < uint64(1e6) {
		tip = uint64(1e6)
	}

	amountIn, _ := new(big.Float).Mul(maxAmountIn, global.Float1Lamp).Uint64()

	poolData := ts.Token.PoolData.Data.(*solanaswapgo.PumpAmmPool)

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	adapter := shot.NewPumpAmmAdapter()
	buyIns := adapter.BuildInstructions(
		&shot.TxContext{
			SignerAndOwner:       p.wallet.PrivateKey,
			SrcMint:              poolData.QuoteMint,
			DstMint:              poolData.BaseMint,
			VirtualSolReserves:   big.NewInt(int64(poolData.PoolBaseTokenReserves)),
			VirtualTokenReserves: big.NewInt(int64(poolData.PoolQuoteTokenReserves)),
			MaxAmountIn:          amountIn,
			Slippage:             slippage,
			PriorityFee:          priorityFee,
			Fee:                  0,
		},
		nonceAccount,
		poolData.Pool,
		poolData.GlobalConfig,
		poolData.PoolBaseTokenAccount,
		poolData.PoolQuoteTokenAccount,
		poolData.ProtocolFeeRecipient,
		poolData.ProtocolFeeRecipientTokenAccount,
		poolData.CoinCreatorVaultAta,
		poolData.CoinCreatorVaultAuthority,
	)

	txBuilder := global.NewTxBuilder(p.wallet.PublicKey(), nonceHash)
	txBuilder.AddInstruction(buyIns...)

	return p.SendAndWait2(ts.Token.TokenAddress, tip, txBuilder)
}

func (p *PumpFunMonitor) buyWithJupiter(ts *TokenSwap, maxAmountIn *big.Float, slippage float32) (*rpc.GetTransactionResult, error) {
	var (
		BribeAmount  = uint64(1e6)
		BribeAccount = solana.MustPublicKeyFromBase58("AP6qExwrbRgBAVaehg4b5xHENX815sMabtBzUzVB4v8S")
	)

	mintAddress := ts.Token.TokenAddress
	amountIn, _ := new(big.Float).Mul(maxAmountIn, global.Float1Lamp).Uint64()

	slippageBps := uint64(slippage * 100)
	q, err := client.Quote(global.Solana, mintAddress, amountIn, slippageBps)
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

	blockHash := global.GetBlockHash()
	// Create the transaction with all instructions
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

	logx.Infof("[%s]:开始买入, 数量: %.10f, 滑点:%.10f,txHash:%s", ts.Token.TokenAddress, maxAmountIn, slippage, txHash)

	return p.SendAndWait(tx, true)

}

func (p *PumpFunMonitor) buyWithMeteoraDbc(ts *TokenSwap, maxAmountIn *big.Float, slippage float32) (*rpc.GetTransactionResult, error) {
	priorityFee := global.GetMedium()
	// fee := uint64(1e6) // 基础费用自动扣除
	tip := global.GetHigh()
	if tip > uint64(1e6) {
		tip = uint64(1e6)
	}
	if tip < uint64(1e6) {
		tip = uint64(1e6)
	} // 0.001 SOL（如果使用 Jito）

	amountIn, _ := new(big.Float).Mul(maxAmountIn, global.Float1Lamp).Uint64()

	poolData := ts.Token.PoolData.Data.(*solanaswapgo.MeteoraDbcPool)

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	adapter := shot.NewMDbcAdapter()

	buyIns := adapter.BuildInstructions(
		&shot.TxContext{
			SignerAndOwner: p.wallet.PrivateKey,
			SrcMint:        poolData.QuoteMint,
			DstMint:        poolData.BaseMint,
			SqrtPrice:      big.NewInt(int64(poolData.NextSqrtPrice)),
			MaxAmountIn:    amountIn,
			Slippage:       slippage,
			PriorityFee:    priorityFee,
			Fee:            0,
		},
		nonceAccount,
		poolData.Config,
		poolData.Pool,
		poolData.BaseVault,
		poolData.QuoteVault,
		poolData.TokenBaseProgram,
		poolData.TokenQuoteProgram,
	)

	txBuilder := global.NewTxBuilder(p.wallet.PublicKey(), nonceHash)
	txBuilder.AddInstruction(buyIns...)

	return p.SendAndWait2(ts.Token.TokenAddress, tip, txBuilder)

}

func (p *PumpFunMonitor) buyWithRaydiumLaunchpad(ts *TokenSwap, maxAmountIn *big.Float, slippage float32) (*rpc.GetTransactionResult, error) {

	priorityFee := global.GetMedium()
	// fee := uint64(1e6) // 基础费用自动扣除
	tip := global.GetHigh()
	if tip > uint64(1e6) {
		tip = uint64(1e6)
	}
	if tip < uint64(1e6) {
		tip = uint64(1e6)
	}

	amountIn, _ := new(big.Float).Mul(maxAmountIn, global.Float1Lamp).Uint64()

	poolData := ts.Token.PoolData.Data.(*solanaswapgo.RaydiumLaunchpadPool)

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	adapter := shot.NewBonkAdapter()
	buyIns := adapter.BuildInstructions(
		&shot.TxContext{
			SignerAndOwner:       p.wallet.PrivateKey,
			SrcMint:              poolData.QuoteMint,
			DstMint:              poolData.BaseMint,
			VirtualSolReserves:   big.NewInt(int64(ts.Token.PoolSolBalance.Load())),
			VirtualTokenReserves: big.NewInt(int64(ts.Token.PoolTokenBalance.Load())),
			MaxAmountIn:          amountIn,
			Slippage:             slippage,
			PriorityFee:          priorityFee,
			Fee:                  0,
		},
		nonceAccount,
		poolData.GlobalConfig,
		poolData.PlatformConfig,
		poolData.PoolState,
		poolData.BaseVault,
		poolData.QuoteVault,
	)

	txBuilder := global.NewTxBuilder(p.wallet.PublicKey(), nonceHash)
	txBuilder.AddInstruction(buyIns...)

	return p.SendAndWait2(ts.Token.TokenAddress, tip, txBuilder)
}
