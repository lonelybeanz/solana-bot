package meteora

import (
	"math/big"
	"solana-bot/internal/dex/meteora/instructions"
	"solana-bot/internal/global"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
)

func GetBuyTx(
	signerAndOwner *solana.PrivateKey,
	config, pool, baseVault, quoteVault, baseMint, quoteMint solana.PublicKey,
	sqrtPrice *big.Int,
	maxAmountIn *big.Int,
	slippage float64,
	priorityFee uint64,
	fee uint64,
	jitoTip uint64,
) (*solana.Transaction, error) {

	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{*signerAndOwner}
	amountInAfterOurFee := new(big.Int).Sub(maxAmountIn, big.NewInt(int64(fee)))

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	// nonce advance的操作一定要在第一個instruction
	instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())

	if jitoTip > 0 {
		instrs = append(instrs, system.NewTransferInstruction(jitoTip, signerAndOwner.PublicKey(), global.PickRandomTip()).Build())

	}

	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(80000).Build())

	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	if fee > 0 {
		instrs = append(instrs, system.NewTransferInstruction(fee, signerAndOwner.PublicKey(), global.FeeAccountBuys).Build())
	}

	userInputTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		quoteMint,
	)
	userOutputTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		baseMint,
	)

	global.CreateSOLAccountOrWrap(&instrs, signerAndOwner.PublicKey(), amountInAfterOurFee)

	instrs = append(instrs, associatedtokenaccount.NewCreateInstruction(signerAndOwner.PublicKey(), signerAndOwner.PublicKey(), baseMint).Build())

	minOut := EstimateBackrunBuyOutAmount(
		sqrtPrice,
		amountInAfterOurFee,
		6,
		9,
		100,
	)

	instrs = append(instrs, instructions.Swap(
		config,
		pool,
		userInputTokenAccount,
		userOutputTokenAccount,
		baseVault,
		quoteVault,
		baseMint,
		quoteMint,
		signerAndOwner.PublicKey(),
		userInputTokenAccount, // using input token account as referral
		amountInAfterOurFee.Uint64(),
		minOut, // minOut
	))
	// spew.Dump(instrs)
	tx, err := BuildTransaction(nonceHash, signers, *signerAndOwner, instrs...)
	return tx, err
}

func GetSellTx(
	signerAndOwner *solana.PrivateKey,
	config, pool, baseVault, quoteVault, baseMint, quoteMint solana.PublicKey,
	sqrtPrice *big.Int,
	maxAmountIn *big.Int,
	slippage float64,
	priorityFee uint64,
	fee uint64,
	jitoTip uint64,
) (*solana.Transaction, error) {

	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{*signerAndOwner}
	amountInAfterOurFee := new(big.Int).Sub(maxAmountIn, big.NewInt(int64(fee)))

	// nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	// // nonce advance的操作一定要在第一個instruction
	// instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())

	if jitoTip > 0 {
		instrs = append(instrs, system.NewTransferInstruction(jitoTip, signerAndOwner.PublicKey(), global.PickRandomTip()).Build())

	}

	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(80000).Build())

	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	if fee > 0 {
		instrs = append(instrs, system.NewTransferInstruction(fee, signerAndOwner.PublicKey(), global.FeeAccountBuys).Build())
	}

	userInputTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		quoteMint,
	)
	userOutputTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		baseMint,
	)

	minOut := EstimateBackrunBuyOutAmount(
		sqrtPrice,
		amountInAfterOurFee,
		6,
		9,
		100,
	)

	instrs = append(instrs, instructions.Swap(
		config,
		pool,
		userInputTokenAccount,
		userOutputTokenAccount,
		baseVault,
		quoteVault,
		baseMint,
		quoteMint,
		signerAndOwner.PublicKey(),
		userInputTokenAccount, // using input token account as referral
		amountInAfterOurFee.Uint64(),
		minOut, // minOut
	))
	// spew.Dump(instrs)
	tx, err := BuildTransaction(global.GetBlockHash(), signers, *signerAndOwner, instrs...)
	return tx, err
}

func EstimateBackrunBuyOutAmount(
	sqrtPrice *big.Int, // next sqrt price from previous tx
	amountInQuote *big.Int, // 你要投入的 quote（如 SOL）数量
	decimalsBase int,
	decimalsQuote int,
	feeBps int, // e.g. 20 for 0.2%
) uint64 {
	// 解析 sqrtPrice
	sqrtP := new(big.Float).SetInt(sqrtPrice)

	// price = sqrtPrice^2 / Q96^2（我们简化使用精度为 1e18）
	precision := big.NewFloat(1e18)
	price := new(big.Float).Quo(
		new(big.Float).Mul(sqrtP, sqrtP),
		precision,
	)

	// 转入的 quote，乘精度
	amountIn := big.NewFloat(float64(amountInQuote.Uint64()))

	// 扣手续费
	feeRatio := new(big.Float).Quo(big.NewFloat(float64(feeBps)), big.NewFloat(10000))
	fee := new(big.Float).Mul(amountIn, feeRatio)
	amountInNet := new(big.Float).Sub(amountIn, fee)

	// 得到的 base token = amountInNet / price
	baseOut := new(big.Float).Quo(amountInNet, price)

	// 除以精度，转换成人类可读格式
	baseOutFloat := new(big.Float).Quo(baseOut,
		new(big.Float).SetFloat64(float64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalsBase)), nil).Int64())),
	)

	out, _ := baseOutFloat.Uint64()

	return out
}

func BuildTransaction(nonceHash solana.Hash, signers []solana.PrivateKey, signer solana.PrivateKey, instrs ...solana.Instruction) (*solana.Transaction, error) {
	tx, err := solana.NewTransaction(
		instrs,
		nonceHash,
		solana.TransactionPayer(signers[0].PublicKey()),
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			return &signer
		},
	)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
