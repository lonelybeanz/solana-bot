package monitor

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"solana-bot/internal/client"
	"time"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"
	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/gagliardetto/solana-go"
	"github.com/imzhongqi/okxos/dex"
	"github.com/zeromicro/go-zero/core/logx"
)

func createTransactionInstruction(instructionData client.Instruction) solana.Instruction {
	programID := solana.MustPublicKeyFromBase58(instructionData.ProgramId)
	accounts := solana.AccountMetaSlice{}
	for _, acc := range instructionData.Accounts {
		pubkey := solana.MustPublicKeyFromBase58(acc.Pubkey)
		accounts = append(accounts, &solana.AccountMeta{
			PublicKey:  pubkey,
			IsSigner:   acc.IsSigner,
			IsWritable: acc.IsWritable,
		})
	}
	data, _ := base64.StdEncoding.DecodeString(instructionData.Data)

	return solana.NewInstruction(programID, accounts, data)
}

func createTransactionInstructionWithOkx(instructionData dex.InstructionInfo) solana.Instruction {
	programID := solana.MustPublicKeyFromBase58(instructionData.ProgramId)
	accounts := solana.AccountMetaSlice{}
	for _, acc := range instructionData.Accounts {
		pubkey := solana.MustPublicKeyFromBase58(acc.Pubkey)
		accounts = append(accounts, &solana.AccountMeta{
			PublicKey:  pubkey,
			IsSigner:   acc.IsSigner,
			IsWritable: acc.IsWritable,
		})
	}
	data, _ := base64.StdEncoding.DecodeString(instructionData.Data)

	return solana.NewInstruction(programID, accounts, data)
}

func ParseSwapTransaction(pbtx *pb.Transaction, pbtxMeta *pb.TransactionStatusMeta) (*solanaswapgo.SwapInfo, error) {
	// Initialize the transaction parser
	parser, err := solanaswapgo.NewPbTransactionParserFromTransaction(pbtx, pbtxMeta)
	if err != nil {
		return nil, err
	}

	// Parse the transaction to extract basic data
	transactionData, err := parser.ParseTransactionForSwap()
	if err != nil {
		return nil, err
	}

	// Process and extract swap-specific data from the parsed transaction
	swapData, err := parser.ProcessSwapData(transactionData)
	if err != nil {
		logx.Debugf("Error processing swap data: %v", err)
		return nil, err
	}

	// Print the parsed swap data
	// marshalledSwapData, _ := json.MarshalIndent(swapData, "", "  ")
	// fmt.Println(string(marshalledSwapData))

	return swapData, nil
}

func SellProportionallyByRecentSell(ts *TokenSwap, theirSoldAmount *big.Int) *big.Int {
	theirCurrentBalance := ts.Tracked.RemainingAmount.Load()
	myHolding := ts.GetRemainingAmount()
	totalBefore := new(big.Int).Add(theirCurrentBalance, theirSoldAmount)
	if totalBefore.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0)
	}

	// ratio = sold / total
	ratio := new(big.Rat).SetFrac(theirSoldAmount, totalBefore)
	ratioFloat, _ := ratio.Float64()
	// 如果对方卖出了 >90%，我们就全部卖掉
	if ratioFloat >= 0.9 {
		return new(big.Int).Set(myHolding)
	}

	// mySell = myHolding * ratio
	myHoldingRat := new(big.Rat).SetInt(myHolding)
	mySellRat := new(big.Rat).Mul(myHoldingRat, ratio)

	// 向下取整转为 big.Int
	mySellInt := new(big.Int)
	mySellRat.FloatString(0)
	mySellRatNum := mySellRat.Num()
	mySellInt.Div(mySellRatNum, mySellRat.Denom())

	return mySellInt
}

// 动态买入数量计算函数
func calculateDynamicBuyAmount(devBuyAmount uint64) *big.Float {
	var (
		k = 0.4
	)

	hour := time.Now().Hour()
	params := GetStrategyParamsByHour(hour)
	maxBuy := params.MaxBuyAmount
	minBuy := params.MinBuyAmount

	devBuySOL := float64(devBuyAmount) / 1e9
	buy := maxBuy - k*devBuySOL
	if buy < minBuy {
		buy = minBuy
	}
	return big.NewFloat(buy)
}

// HoldDurationCurve 定义计算持仓时间的函数类型
type HoldDurationCurve func(solAmount float64) float64

func PositiveCurve(solAmount float64) float64 {
	referenceAmount := 2.0
	ratio := solAmount / referenceAmount
	return ratio // 越大 factor 越大 → 持有时间越长
}

func NegativeCurve(solAmount float64) float64 {
	referenceAmount := 2.0
	ratio := solAmount / referenceAmount
	return 1.0 - ratio // 越小 factor 越大 → 持有时间越长
}

// calculateHoldDuration 根据买入金额和指定曲线动态计算持仓时间
func calculateHoldDuration(solAmount float64, curve HoldDurationCurve) time.Duration {
	if solAmount <= 0 {
		return 0
	}

	hour := time.Now().Hour()
	params := GetStrategyParamsByHour(hour)
	maxHold := time.Duration(params.MaxHoldMillisecond) * time.Millisecond
	minHold := time.Duration(params.MinHoldMillisecond) * time.Millisecond

	factor := curve(solAmount)
	// 将 Duration 转为纳秒整数再参与浮点运算
	diffNanos := float64(maxHold - minHold)
	holdDurationNanos := diffNanos*factor + 0.5

	holdDuration := time.Duration(holdDurationNanos) + minHold
	holdDuration = max(minHold, min(maxHold, holdDuration))

	return holdDuration
}

func calculateBreakEvenAmount(buyPrice, currentPrice *big.Float, amountBought *big.Int) *big.Int {
	fmt.Printf("buyPrice: %v, currentPrice: %v, amountBought: %v\n", buyPrice, currentPrice, amountBought)
	// 计算总成本: buyPrice * amountBought
	totalCost := new(big.Float).Mul(buyPrice, new(big.Float).SetInt(amountBought))

	// 如果当前价格低于或等于买入价，则返回全部卖出
	if currentPrice.Cmp(buyPrice) <= 0 {
		return new(big.Int).Set(amountBought)
	}

	// 计算回本所需数量: totalCost / currentPrice
	breakEvenAmountFloat := new(big.Float).Quo(totalCost, currentPrice)

	// 向上取整，确保完全覆盖成本
	breakEvenAmount := new(big.Int)
	breakEvenAmountFloat.Int(breakEvenAmount)
	if breakEvenAmountFloat.Cmp(new(big.Float).SetInt(breakEvenAmount)) != 0 {
		breakEvenAmount.Add(breakEvenAmount, big.NewInt(1))
	}

	// 确保不会超过实际持有量
	if breakEvenAmount.Cmp(amountBought) > 0 {
		return new(big.Int).Set(amountBought)
	}

	return breakEvenAmount
}

type TxDeduper struct {
	filter *bloom.BloomFilter
}

func NewTxDeduper(n uint, fpRate float64) *TxDeduper {
	return &TxDeduper{
		filter: bloom.NewWithEstimates(n, fpRate), // n=预计交易数量, fpRate=可接受误判率
	}
}

func (d *TxDeduper) SeenOrAdd(sig string) bool {
	if d.filter.Test([]byte(sig)) {
		return true // 已经见过（可能误判）
	}
	d.filter.Add([]byte(sig))
	return false // 第一次见到
}
