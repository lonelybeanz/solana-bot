package shot

import (
	"math/big"
	"solana-bot/internal/global"

	"github.com/gagliardetto/solana-go"
	associated_token_account "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
)

type PumpAmmAdapter struct {
}

func NewPumpAmmAdapter() *PumpAmmAdapter {
	return &PumpAmmAdapter{}
}

func (a *PumpAmmAdapter) Name() string {
	return "PumpAmm"
}

func (a *PumpAmmAdapter) BuildInstructions(txInfo *TxContext, accounts ...solana.PublicKey) []solana.Instruction {
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

	// 1. 设置 compute unit limit（ComputeBudgetProgram）
	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(120_000).Build())

	// 2. 设置 compute unit price（ComputeBudgetProgram）
	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	pool := accounts[1]
	globalConfig := accounts[2]
	poolBaseTokenAccount := accounts[3]
	poolQuoteTokenAccount := accounts[4]
	protocolFeeRecipient := accounts[5]
	protocolFeeRecipientATA := accounts[6]
	coinCreatorVaultAta := accounts[7]
	coinCreatorVaultAuthority := accounts[8]

	if isBuy {
		base_amount_out := EstimateSwapOut(big.NewInt(int64(maxAmountIn)), quote_balance_sol, base_balance_tokens, FeeBasisPoints)
		base_amount_out = applySlippage(base_amount_out, slippage)

		global.CreateSOLAccountOrWrap(&instrs, signerAndOwner.PublicKey(), big.NewInt(int64(maxAmountIn)))

		instrs = append(instrs, associated_token_account.NewCreateInstruction(signerAndOwner.PublicKey(), signerAndOwner.PublicKey(), dstMint).Build())

		addPumpAmmBuyIx(&instrs, base_amount_out.Uint64(), maxAmountIn, signerAndOwner.PublicKey(), pool, globalConfig, dstMint, srcMint, poolBaseTokenAccount, poolQuoteTokenAccount, protocolFeeRecipient, protocolFeeRecipientATA, coinCreatorVaultAta, coinCreatorVaultAuthority)

	} else {

		quote_amount_out := EstimateSwapOut(big.NewInt(int64(maxAmountIn)), base_balance_tokens, quote_balance_sol, FeeBasisPoints)
		min_quote_amount_out := applySlippage(quote_amount_out, slippage).Uint64()

		addPumpAmmSellIx(&instrs, signerAndOwner.PublicKey(), maxAmountIn, min_quote_amount_out, pool, globalConfig, dstMint, srcMint, poolBaseTokenAccount, poolQuoteTokenAccount, protocolFeeRecipient, protocolFeeRecipientATA, coinCreatorVaultAta, coinCreatorVaultAuthority)

		closeATA(&instrs, signerAndOwner.PublicKey(), srcMint)
	}

	return instrs
}

/*
#1 Pool
#2 User
#3 Global Config
#4 Base Mint
#5 Quote Mint
#6 User Base ATA
#7 User Quote ATA
#8 Pool Base ATA
#9 Pool Quote ATA
#10 Protocol Fee Recipient
#11 Protocol Fee Recipient Token Account
#12 Base Token Program
#13 Quote Token Program
#14 System Program
#15 Associated Token Program
#16 Event Authority
#17 PumpSwap Program
*/
func addPumpAmmBuyIx(
	instrs *[]solana.Instruction,
	base_amount_out uint64,
	max_quote_amount_in uint64,
	owner solana.PublicKey,
	pool solana.PublicKey,
	globalConfig solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	poolBaseTokenAccount solana.PublicKey,
	poolQuoteTokenAccount solana.PublicKey,
	protocolFeeRecipient solana.PublicKey,
	protocolFeeRecipientATA solana.PublicKey,
	coinCreatorVaultAta solana.PublicKey,
	coinCreatorVaultAuthority solana.PublicKey,
) {

	userBaseAta, _, _ := solana.FindAssociatedTokenAddress(owner, baseMint)
	userQuoteAta, _, _ := solana.FindAssociatedTokenAddress(owner, quoteMint)

	globalVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("global_volume_accumulator"),
	}, PUMPSWAP_PROGRAM_ID)

	userVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("user_volume_accumulator"),
		owner.Bytes(),
	}, PUMPSWAP_PROGRAM_ID)

	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(pool, true, false),                    // 写权限
		solana.NewAccountMeta(owner, true, true),                    // 写权限 + 签名者
		solana.NewAccountMeta(globalConfig, false, false),           // 只读
		solana.NewAccountMeta(baseMint, false, false),               // 只读
		solana.NewAccountMeta(quoteMint, false, false),              // 只读
		solana.NewAccountMeta(userBaseAta, true, false),             // 写权限
		solana.NewAccountMeta(userQuoteAta, true, false),            // 写权限
		solana.NewAccountMeta(poolBaseTokenAccount, true, false),    // 写权限
		solana.NewAccountMeta(poolQuoteTokenAccount, true, false),   // 写权限
		solana.NewAccountMeta(protocolFeeRecipient, false, false),   // 只读
		solana.NewAccountMeta(protocolFeeRecipientATA, true, false), // 写权限
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读（重复传两次）
		solana.NewAccountMeta(SYSTEM_PROGRAM_ID, false, false),      // 只读
		solana.NewAccountMeta(ASSOCIATED_TOKEN, false, false),       // 只读
		solana.NewAccountMeta(EVENT_AUTHORITY, false, false),        // 只读
		solana.NewAccountMeta(PUMPSWAP_PROGRAM_ID, false, false),    // 只读
		solana.NewAccountMeta(coinCreatorVaultAta, true, false),
		solana.NewAccountMeta(coinCreatorVaultAuthority, false, false),
		solana.NewAccountMeta(globalVolumeAccumulation, true, false),
		solana.NewAccountMeta(userVolumeAccumulation, true, false),
	}
	data := BuildBuyInstructionData(base_amount_out, max_quote_amount_in)
	*instrs = append(*instrs, solana.NewInstruction(PUMPSWAP_PROGRAM_ID, accounts, data))

}

func addPumpAmmSellIx(
	instrs *[]solana.Instruction,
	owner solana.PublicKey,
	base_amount_in uint64,
	min_quote_amount_out uint64,
	pool solana.PublicKey,
	globalConfig solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	poolBaseTokenAccount solana.PublicKey,
	poolQuoteTokenAccount solana.PublicKey,
	protocolFeeRecipient solana.PublicKey,
	protocolFeeRecipientATA solana.PublicKey,
	coinCreatorVaultAta solana.PublicKey,
	coinCreatorVaultAuthority solana.PublicKey,
) {
	userBaseAta, _, _ := solana.FindAssociatedTokenAddress(owner, baseMint)
	userQuoteAta, _, _ := solana.FindAssociatedTokenAddress(owner, quoteMint)

	globalVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("global_volume_accumulator"),
	}, PUMPSWAP_PROGRAM_ID)

	userVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("user_volume_accumulator"),
		owner.Bytes(),
	}, PUMPSWAP_PROGRAM_ID)

	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(pool, true, false),                    // 写权限
		solana.NewAccountMeta(owner, true, true),                    // 写权限 + 签名者
		solana.NewAccountMeta(globalConfig, false, false),           // 只读
		solana.NewAccountMeta(baseMint, false, false),               // 只读
		solana.NewAccountMeta(quoteMint, false, false),              // 只读
		solana.NewAccountMeta(userBaseAta, true, false),             // 写权限
		solana.NewAccountMeta(userQuoteAta, true, false),            // 写权限
		solana.NewAccountMeta(poolBaseTokenAccount, true, false),    // 写权限
		solana.NewAccountMeta(poolQuoteTokenAccount, true, false),   // 写权限
		solana.NewAccountMeta(protocolFeeRecipient, false, false),   // 只读
		solana.NewAccountMeta(protocolFeeRecipientATA, true, false), // 写权限
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读（重复传两次）
		solana.NewAccountMeta(SYSTEM_PROGRAM_ID, false, false),      // 只读
		solana.NewAccountMeta(ASSOCIATED_TOKEN, false, false),       // 只读
		solana.NewAccountMeta(EVENT_AUTHORITY, false, false),        // 只读
		solana.NewAccountMeta(PUMPSWAP_PROGRAM_ID, false, false),    // 只读
		solana.NewAccountMeta(coinCreatorVaultAta, true, false),
		solana.NewAccountMeta(coinCreatorVaultAuthority, false, false),
		solana.NewAccountMeta(globalVolumeAccumulation, true, false),
		solana.NewAccountMeta(userVolumeAccumulation, true, false),
	}
	data := BuildSellInstructionData(base_amount_in, min_quote_amount_out)
	*instrs = append(*instrs, solana.NewInstruction(PUMPSWAP_PROGRAM_ID, accounts, data))
}
