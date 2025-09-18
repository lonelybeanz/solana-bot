package shot

import (
	"math/big"
	"solana-bot/internal/global"

	"solana-bot/pkg/token2022"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
)

type MDbcAdapter struct {
}

func NewMDbcAdapter() *MDbcAdapter {
	return &MDbcAdapter{}
}

func (a *MDbcAdapter) Name() string {
	return "Meteora Dynamic Bonding Curve"
}

func (a *MDbcAdapter) BuildInstructions(txInfo *TxContext, accounts ...solana.PublicKey) []solana.Instruction {
	isBuy := true
	srcMint := txInfo.SrcMint
	dstMint := txInfo.DstMint
	if dstMint.Equals(solana.WrappedSol) {
		isBuy = false
	}

	maxAmountIn := txInfo.MaxAmountIn
	priorityFee := txInfo.PriorityFee
	// slippage := txInfo.Slippage
	sqrtPrice := txInfo.SqrtPrice

	instrs := []solana.Instruction{}
	signerAndOwner := txInfo.SignerAndOwner

	if isBuy {
		nonceAccount := accounts[0]
		instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())
	}
	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(80000).Build())

	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	config := accounts[1]
	pool := accounts[2]
	baseVault := accounts[3]
	quoteVault := accounts[4]
	tokenBaseProgram := accounts[5]

	userInputTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		srcMint,
	)
	userOutputTokenAccount, _, _ := solana.FindAssociatedTokenAddress(
		signerAndOwner.PublicKey(),
		dstMint,
	)

	if isBuy {
		amountInAfterOurFee := new(big.Int).Sub(big.NewInt(int64(maxAmountIn)), big.NewInt(int64(priorityFee)))

		global.CreateSOLAccountOrWrap(&instrs, signerAndOwner.PublicKey(), amountInAfterOurFee)

		if tokenBaseProgram.Equals(solana.Token2022ProgramID) {
			instrs = append(instrs, token2022.NewCreate2022Instruction(
				signerAndOwner.PublicKey(),
				signerAndOwner.PublicKey(),
				dstMint,
			).Build())
		} else {
			instrs = append(instrs, associatedtokenaccount.NewCreateInstruction(signerAndOwner.PublicKey(), signerAndOwner.PublicKey(), dstMint).Build())

		}

		minOut := EstimateBackrunBuyOutAmount(
			sqrtPrice,
			amountInAfterOurFee,
			6,
			9,
			100,
		)

		instrs = append(instrs, DbcSwap(
			config,
			pool,
			userInputTokenAccount,
			userOutputTokenAccount,
			baseVault,
			quoteVault,
			dstMint,
			srcMint,
			signerAndOwner.PublicKey(),
			tokenBaseProgram,
			userInputTokenAccount, // using input token account as referral
			amountInAfterOurFee.Uint64(),
			minOut, // minOut
		))
	} else {

	}

	return instrs
}
