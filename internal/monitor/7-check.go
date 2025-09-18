package monitor

import (
	"errors"
	"math/big"
	"time"

	"solana-bot/internal/client"
	"solana-bot/internal/dex/pump"
	"solana-bot/internal/global"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"

	"github.com/gagliardetto/solana-go"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	TotalSupply = uint64(1e9)
)

// 持仓监测
func holderCheck(ts *TokenSwap) error {
	start := time.Now()
	defer func() { // 添加括号以修复 defer 报错
		end := time.Now()
		logx.Infof("[holderCheck]cost time: %v", end.Sub(start))
	}()
	tokenAddress := ts.Token.TokenAddress

	var pool *solana.PublicKey
	switch ts.SwapType.Load() {
	case PumpFunType:
		data := ts.Token.PoolData.Data.(*solanaswapgo.PumpFunPool)
		pool = &data.AssociatedBondingCurve
	case PumpAmmType:
		data := ts.Token.PoolData.Data.(*solanaswapgo.PumpAmmPool)
		pool = &data.PoolQuoteTokenAccount
	}
	if pool == nil {
		return nil
	}
	logx.Infof("[%s] 持仓监测 pool:%s", tokenAddress, pool.String())
	out, err := global.GetTopHolders(global.GetRPCForRequest(), tokenAddress)
	if err != nil {
		return err
	}
	// if len(out) >= 10 {
	// 	return errors.New("持币人数大于10")
	// }
	var robotCount uint64
	//计算top10 持币地址的余额
	var count uint64
	limit := 0
	for k, v := range out {
		if pool != nil && k == pool.String() {
			continue
		}
		robotCfg, _ := GetRobotConfig()
		if global.Contains(robotCfg.Robot, k) {
			robotCount++
		}
		if v > TotalSupply*1e6/10 {
			return errors.New("单地址持币超过10%")
		}
		count += v
		limit++
		if limit > 5 {
			break
		}
	}
	//占比超过40% 直接返回
	if count > TotalSupply*1e6/2 {
		return errors.New("持币地址超过50%")
	}

	if robotCount == 0 {
		// return errors.New("没有机器人地址")
	}

	return nil
}

func mcapCheck(ts *TokenSwap) error {
	start := time.Now()
	defer func() {
		end := time.Now()
		logx.Infof("[mcapCheck]cost time: %v", end.Sub(start))
	}()
	mintAddress := ts.Token.TokenAddress
	logx.Infof("[%s]:市值监测 ", mintAddress)
	var mcap *big.Float

	switch ts.SwapType.Load() {
	case PumpFunType:
		bondingCurveData := ts.GetBondingCurveData()
		if bondingCurveData == nil || bondingCurveData.BondingCurve == nil {
			return errors.New("bonding curve data is nil")
		}
		solTokenPrice, _, _ := pump.GetPriceAndLiquidityAndDexFromPump(bondingCurveData)

		mcap = new(big.Float).Mul(new(big.Float).SetUint64(TotalSupply), solTokenPrice)
	case PumpAmmType:
		tokenBalance := new(big.Float).Quo(new(big.Float).SetUint64(ts.Token.PoolTokenBalance.Load()), big.NewFloat(1e6))
		solBalance := new(big.Float).Quo(new(big.Float).SetUint64(ts.Token.PoolSolBalance.Load()), big.NewFloat(1e9))
		price := new(big.Float).Quo(solBalance, tokenBalance)
		mcap = new(big.Float).Mul(new(big.Float).SetUint64(TotalSupply), price)
	}

	// if mcap.Cmp(big.NewFloat(2e3)) < 0 {
	// 	return errors.New("市值小于3k")
	// }

	if mcap != nil && mcap.Cmp(big.NewFloat(6e3)) > 0 { //6000sol
		return errors.New("市值超过1M")
	}

	return nil
}

func RugCheck(ts *TokenSwap) error {
	start := time.Now()
	defer func() {
		end := time.Now()
		logx.Infof("[rugCheck]cost time: %v", end.Sub(start))
	}()
	logx.Infof("[%s]:rugCheck start", ts.Token.TokenAddress)
	res, err := client.GetRugCheck(ts.Token.TokenAddress)
	if err != nil {
		return nil
	}
	if res.CreatorBalance > 0 {
		return errors.New("dev not run")
	}
	return nil
}
