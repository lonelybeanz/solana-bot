package raydium

import (
	"math/big"
	"solana-bot/internal/global"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
)

func GetBuyTx(
	signerAndOwner *solana.PrivateKey,
	globalConfig, platformConfig, poolState, baseVault, quoteVault, baseMint, quoteMint solana.PublicKey,
	base_balance_tokens *big.Float,
	quote_balance_sol *big.Float,
	maxAmountIn *big.Float,
	slippage float64,
	priorityFee uint64,
	fee uint64,
	jitoTip uint64,
) (*solana.Transaction, error) {

	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{*signerAndOwner}

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()

	instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())

	amountInAfterOurFee := new(big.Float).Sub(maxAmountIn, big.NewFloat(float64(fee/1e9)))

	sol_amount, _ := amountInAfterOurFee.Float64()
	base_amount_out, max_quote_amount_in := convert_sol_to_base_tokens(sol_amount, base_balance_tokens, quote_balance_sol, 6, slippage/100)

	if jitoTip > 0 {
		instrs = append(instrs, system.NewTransferInstruction(jitoTip, signerAndOwner.PublicKey(), global.PickRandomTip()).Build())

	}

	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(120_000).Build())

	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	if fee > 0 {
		instrs = append(instrs, system.NewTransferInstruction(fee, signerAndOwner.PublicKey(), global.FeeAccountBuys).Build())
	}

	userQuoteTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		quoteMint,
	)
	userBaseTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		baseMint,
	)

	// 3. 检查并创建 WSOL ATA（如果不存在）
	global.CreateSOLAccountOrWrap(&instrs, signerAndOwner.PublicKey(), big.NewInt(int64(max_quote_amount_in)))

	instrs = append(instrs, associatedtokenaccount.NewCreateInstruction(signerAndOwner.PublicKey(), signerAndOwner.PublicKey(), baseMint).Build())

	instrs = append(instrs, Swap(
		true,
		globalConfig,
		platformConfig,
		poolState,
		userBaseTokenAccount,
		userQuoteTokenAccount,
		baseVault,
		quoteVault,
		baseMint,
		quoteMint,
		signerAndOwner.PublicKey(),
		max_quote_amount_in,
		base_amount_out, // minOut
	))
	// spew.Dump(instrs)
	tx, err := BuildTransaction(nonceHash, signers, *signerAndOwner, instrs...)
	return tx, err
}

func GetSellTx(
	signerAndOwner *solana.PrivateKey,
	globalConfig, platformConfig, poolState, baseVault, quoteVault, baseMint, quoteMint solana.PublicKey,
	base_balance_tokens *big.Float,
	quote_balance_sol *big.Float,
	maxAmountIn *big.Int,
	slippage float64,
	priorityFee uint64,
	fee uint64,
	jitoTip uint64,
	shouldCloseTokenAccount bool,
) (*solana.Transaction, error) {

	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{*signerAndOwner}

	// nonceAccount, nonceHash := global.GetNonceAccountAndHash()
	// instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())

	base_amount_in, min_quote_amount_out := convert_base_tokens_to_sol(maxAmountIn, base_balance_tokens, quote_balance_sol, 6, slippage/100)

	if jitoTip > 0 {
		instrs = append(instrs, system.NewTransferInstruction(jitoTip, signerAndOwner.PublicKey(), global.PickRandomTip()).Build())

	}

	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(120_000).Build())

	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	if fee > 0 {
		instrs = append(instrs, system.NewTransferInstruction(fee, signerAndOwner.PublicKey(), global.FeeAccountBuys).Build())
	}

	userQuoteTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		quoteMint,
	)
	userBaseTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		baseMint,
	)

	instrs = append(instrs, Swap(
		false,
		globalConfig,
		platformConfig,
		poolState,
		userBaseTokenAccount,
		userQuoteTokenAccount,
		baseVault,
		quoteVault,
		baseMint,
		quoteMint,
		signerAndOwner.PublicKey(),
		base_amount_in,
		min_quote_amount_out, // minOut
	))

	if shouldCloseTokenAccount {
		closeInst := token.NewCloseAccountInstruction(
			userBaseTokenAccount,
			signerAndOwner.PublicKey(),
			signerAndOwner.PublicKey(),
			[]solana.PublicKey{},
		).Build()
		instrs = append(instrs, closeInst)
	}

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
