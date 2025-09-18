package shot

import (
	"math/big"
	"solana-bot/internal/global"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
)

type BonkAdapter struct {
	TipAccounts []solana.PublicKey
}

func NewBonkAdapter() *BonkAdapter {
	return &BonkAdapter{}
}

func (a *BonkAdapter) Name() string {
	return "Bonk"
}

func (a *BonkAdapter) BuildInstructions(txInfo *TxContext, accounts ...solana.PublicKey) []solana.Instruction {
	isBuy := true
	srcMint := txInfo.SrcMint
	dstMint := txInfo.DstMint
	if dstMint.Equals(solana.WrappedSol) {
		isBuy = false
	}

	maxAmountIn := txInfo.MaxAmountIn
	priorityFee := txInfo.PriorityFee
	slippage := txInfo.Slippage
	base_balance_tokens := txInfo.VirtualTokenReserves
	quote_balance_sol := txInfo.VirtualSolReserves

	instrs := []solana.Instruction{}
	signerAndOwner := txInfo.SignerAndOwner

	if isBuy {
		nonceAccount := accounts[0]
		instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())
	}

	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(120_000).Build())

	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	globalConfig := accounts[1]
	platformConfig := accounts[2]
	poolState := accounts[3]
	baseVault := accounts[4]
	quoteVault := accounts[5]

	userQuoteTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		srcMint,
	)
	userBaseTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		dstMint,
	)

	if isBuy {

		virtualBase := big.NewInt(1073025605596382) // 1e9
		virtualQuote := big.NewInt(30000852951)     // 5e8
		realBaseAfter := base_balance_tokens        // 6e8
		realQuoteAfter := quote_balance_sol         // 4e8
		amountIn := big.NewInt(int64(maxAmountIn))  // 输入 quote = 1e8

		base_amount_out := PredictNextOutputByRealState(virtualBase, virtualQuote, realBaseAfter, realQuoteAfter, amountIn)

		// base_amount_out := EstimateSwapOut(big.NewInt(int64(maxAmountIn)), quote_balance_sol, base_balance_tokens, FeeBasisPoints)
		base_amount_out = applySlippage(base_amount_out, slippage)

		global.CreateSOLAccountOrWrap(&instrs, signerAndOwner.PublicKey(), big.NewInt(int64(maxAmountIn)))

		instrs = append(instrs, associatedtokenaccount.NewCreateInstruction(signerAndOwner.PublicKey(), signerAndOwner.PublicKey(), dstMint).Build())

		instrs = append(instrs, BonkSwap(
			true,
			globalConfig,
			platformConfig,
			poolState,
			userBaseTokenAccount,
			userQuoteTokenAccount,
			baseVault,
			quoteVault,
			dstMint,
			srcMint,
			signerAndOwner.PublicKey(),
			maxAmountIn,
			base_amount_out.Uint64(), // minOut
		))
	} else {

		virtualBase := big.NewInt(1073025605596382) // 1e9
		virtualQuote := big.NewInt(30000852951)     // 5e8
		realBaseAfter := base_balance_tokens        // 6e8
		realQuoteAfter := quote_balance_sol         // 4e8
		amountIn := big.NewInt(int64(maxAmountIn))  // 输入 quote = 1e8

		quote_amount_out := PredictSellExactInByRealState(virtualBase, virtualQuote, realBaseAfter, realQuoteAfter, amountIn)

		// quote_amount_out := EstimateSwapOut(big.NewInt(int64(maxAmountIn)), base_balance_tokens, quote_balance_sol, FeeBasisPoints)
		min_quote_amount_out := applySlippage(quote_amount_out, slippage).Uint64()

		instrs = append(instrs, BonkSwap(
			false,
			globalConfig,
			platformConfig,
			poolState,
			userBaseTokenAccount,
			userQuoteTokenAccount,
			baseVault,
			quoteVault,
			dstMint,
			srcMint,
			signerAndOwner.PublicKey(),
			maxAmountIn,
			min_quote_amount_out, // minOut
		))

		closeATA(&instrs, signerAndOwner.PublicKey(), srcMint)
	}

	return instrs
}
