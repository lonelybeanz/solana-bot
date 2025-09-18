package global

import (
	"math"
	"math/big"
	"strconv"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

func GetBalacneChange(tx *rpc.GetTransactionResult) (float64, float64) {
	if tx == nil {
		logx.Error("tx is nil")
		return 0, 0
	}
	// 获取交易的 Meta 数据
	meta := tx.Meta
	if meta == nil {
		logx.Error("未获取到 Meta 数据")
		return 0, 0
	}

	// 1️⃣ 获取交易前的 SOL 余额
	solBefore := float64(meta.PreBalances[0])

	// 2️⃣ 获取交易后的 SOL 余额
	solAfter := float64(meta.PostBalances[0])

	// 3️⃣ 计算交易费用
	fee := float64(meta.Fee)

	// 4️⃣ 计算实际花费的 SOL
	actualSolSpent := solAfter - solBefore

	var wsolBefore, wsolAfter float64
	for _, bal := range meta.PostTokenBalances {
		if bal.Mint.String() == Solana {
			wsolAfter, _ = strconv.ParseFloat(bal.UiTokenAmount.Amount, 10)
			break
		}
	}
	for _, bal := range meta.PreTokenBalances {
		if bal.Mint.String() == Solana {
			wsolBefore, _ = strconv.ParseFloat(bal.UiTokenAmount.Amount, 10)
			break
		}
	}

	wsolChange := wsolAfter - wsolBefore
	actualSolSpent = actualSolSpent + wsolChange

	return actualSolSpent, fee

}

func GetBuyPriceAndAmountWithY(meta *pb.TransactionStatusMeta, token string) (float64, *big.Int) {

	if meta == nil {
		logx.Error("未获取到 Meta 数据")
		return 0, nil
	}

	// 1️⃣ 获取交易前的 SOL 余额
	solBefore := float64(meta.PreBalances[0])

	// 2️⃣ 获取交易后的 SOL 余额
	solAfter := float64(meta.PostBalances[0])

	// 3️⃣ 计算交易费用
	// fee := float64(meta.Fee) / 1e9

	// 4️⃣ 计算实际花费的 SOL
	actualSolSpent := solBefore - solAfter

	// 5️⃣ 获取购买的 Token 数量
	var boughtAmount *big.Int
	for _, bal := range meta.PostTokenBalances {
		if bal.Mint == token {
			amount, _ := big.NewInt(0).SetString(bal.UiTokenAmount.Amount, 10)
			boughtAmount = amount
			break
		}
	}

	var wsolBefore, wsolAfter float64
	for _, bal := range meta.PostTokenBalances {
		if bal.Mint == Solana {
			wsolAfter, _ = strconv.ParseFloat(bal.UiTokenAmount.Amount, 10)
			break
		}
	}

	for _, bal := range meta.PreTokenBalances {
		if bal.Mint == Solana {
			wsolBefore, _ = strconv.ParseFloat(bal.UiTokenAmount.Amount, 10)
			break
		}
	}

	wsolChange := wsolBefore - wsolAfter
	actualSolSpent = actualSolSpent + wsolChange
	var costPrice float64
	// 6️⃣ 计算买入成本价
	if boughtAmount != nil && boughtAmount.Cmp(big.NewInt(0)) > 0 {
		costPrice = actualSolSpent / 1e9 / (float64(boughtAmount.Int64()) / 1e6)
	} else {
		return 0, nil
	}
	costPrice = math.Abs(costPrice)
	boughtAmount = boughtAmount.Abs(boughtAmount)
	return costPrice, boughtAmount
}

func GetBuyPriceAndAmount(tx *rpc.GetTransactionResult, token string) (float64, *big.Int) {
	if tx == nil {
		logx.Error("tx is nil")
		return 0, nil
	}
	// 获取交易的 Meta 数据
	meta := tx.Meta
	if meta == nil {
		logx.Error("未获取到 Meta 数据")
		return 0, nil
	}

	// 1️⃣ 获取交易前的 SOL 余额
	solBefore := float64(meta.PreBalances[0])

	// 2️⃣ 获取交易后的 SOL 余额
	solAfter := float64(meta.PostBalances[0])

	// 3️⃣ 计算交易费用
	// fee := float64(meta.Fee) / 1e9

	// 4️⃣ 计算实际花费的 SOL
	actualSolSpent := solBefore - solAfter

	// 5️⃣ 获取购买的 Token 数量
	var boughtAmount *big.Int
	for _, bal := range meta.PostTokenBalances {
		if bal.Mint.String() == token {
			amount, _ := big.NewInt(0).SetString(bal.UiTokenAmount.Amount, 10)
			boughtAmount = amount
			break
		}
	}

	var wsolBefore, wsolAfter float64
	for _, bal := range meta.PostTokenBalances {
		if bal.Mint.String() == Solana {
			wsolAfter, _ = strconv.ParseFloat(bal.UiTokenAmount.Amount, 10)
			break
		}
	}

	for _, bal := range meta.PreTokenBalances {
		if bal.Mint.String() == Solana {
			wsolBefore, _ = strconv.ParseFloat(bal.UiTokenAmount.Amount, 10)
			break
		}
	}

	wsolChange := wsolBefore - wsolAfter
	actualSolSpent = actualSolSpent + wsolChange
	var costPrice float64
	// 6️⃣ 计算买入成本价
	if boughtAmount != nil && boughtAmount.Cmp(big.NewInt(0)) > 0 {
		costPrice = actualSolSpent / 1e9 / (float64(boughtAmount.Int64()) / 1e6)
	} else {
		return 0, nil
	}
	costPrice = math.Abs(costPrice)
	boughtAmount = boughtAmount.Abs(boughtAmount)
	return costPrice, boughtAmount
}

func Contains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
