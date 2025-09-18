package shot

import (
	"math/big"
	"solana-bot/internal/global"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	associated_token_account "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"

	"github.com/gagliardetto/solana-go/programs/system"
)

type PumpFunAdapter struct {
}

func NewPumpFunAdapter() *PumpFunAdapter {
	return &PumpFunAdapter{}
}

func (a *PumpFunAdapter) Name() string {
	return "pump.fun"
}

func (a *PumpFunAdapter) BuildInstructions(txInfo *TxContext, accounts ...solana.PublicKey) []solana.Instruction {
	isBuy := true
	srcMint := txInfo.SrcMint
	dstMint := txInfo.DstMint
	if dstMint.Equals(solana.WrappedSol) {
		isBuy = false
	}

	maxAmountIn := txInfo.MaxAmountIn
	fee := txInfo.Fee
	priorityFee := txInfo.PriorityFee
	slippage := txInfo.Slippage

	instrs := []solana.Instruction{}
	signerAndOwner := txInfo.SignerAndOwner

	if isBuy {
		nonceAccount := accounts[0]
		instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())
	}

	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(90000).Build())

	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	creatorVault := accounts[1]
	globalSettingsPk := accounts[2]
	bondingCurvePk := accounts[3]
	associatedBondingCurvePk := accounts[4]

	if isBuy {

		instrs = append(instrs, associated_token_account.NewCreateInstruction(signerAndOwner.PublicKey(), signerAndOwner.PublicKey(), dstMint).Build())

		amountInAfterOurFee := new(big.Int).Sub(big.NewInt(int64(maxAmountIn)), big.NewInt(int64(fee)))

		addPumpBuyIx(&instrs, amountInAfterOurFee, txInfo.VirtualSolReserves, txInfo.VirtualTokenReserves, slippage, signerAndOwner.PublicKey(), dstMint, creatorVault, globalSettingsPk, bondingCurvePk, associatedBondingCurvePk)
	} else {
		addPumpSellIx(&instrs, big.NewInt(int64(maxAmountIn)), txInfo.VirtualSolReserves, txInfo.VirtualTokenReserves, slippage, signerAndOwner.PublicKey(), srcMint, creatorVault, globalSettingsPk, bondingCurvePk, associatedBondingCurvePk)
		closeATA(&instrs, signerAndOwner.PublicKey(), srcMint)
	}

	return instrs
}

func addPumpBuyIx(
	instrs *[]solana.Instruction,
	amountInAfterOurFee *big.Int,
	virtualSolReserves *big.Int,
	virtualTokenReserves *big.Int,
	slippage float32,
	owner solana.PublicKey,
	mint solana.PublicKey,
	creatorVault solana.PublicKey,
	globalSettingsPk solana.PublicKey,
	bondingCurvePk solana.PublicKey,
	associatedBondingCurvePk solana.PublicKey,
) {

	pumpFee := pumpGetFee(amountInAfterOurFee, FeeBasisPoints)

	// this is used for quoting
	amountInAfterPumpFee := new(big.Int).Sub(amountInAfterOurFee, pumpFee)
	amountOut := QuoteBuyByCurve(amountInAfterPumpFee, virtualSolReserves, virtualTokenReserves)
	amountOutWithSlippage := applySlippage(amountOut, slippage)

	instruction := &PumpBuyInstruction{
		MethodId:         PUMPBuyMethod,
		MaxAmountIn:      amountInAfterOurFee.Uint64(),
		AmountOut:        amountOutWithSlippage.Uint64(),
		AccountMetaSlice: make(solana.AccountMetaSlice, 16),
	}

	instruction.BaseVariant = bin.BaseVariant{
		Impl: instruction,
	}

	ataUser, _, _ := solana.FindAssociatedTokenAddress(owner, mint)

	globalVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("global_volume_accumulator"),
	}, PUMPManager)

	userVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("user_volume_accumulator"),
		owner.Bytes(),
	}, PUMPManager)

	instruction.AccountMetaSlice[0] = solana.Meta(globalSettingsPk)
	instruction.AccountMetaSlice[1] = solana.Meta(global.PickRandomPumpProtocolFee()).WRITE()
	instruction.AccountMetaSlice[2] = solana.Meta(mint)
	instruction.AccountMetaSlice[3] = solana.Meta(bondingCurvePk).WRITE()
	instruction.AccountMetaSlice[4] = solana.Meta(associatedBondingCurvePk).WRITE()
	instruction.AccountMetaSlice[5] = solana.Meta(ataUser).WRITE()
	instruction.AccountMetaSlice[6] = solana.Meta(owner).WRITE().SIGNER()
	instruction.AccountMetaSlice[7] = solana.Meta(solana.SystemProgramID)
	instruction.AccountMetaSlice[8] = solana.Meta(solana.TokenProgramID)
	instruction.AccountMetaSlice[9] = solana.Meta(creatorVault).WRITE()
	instruction.AccountMetaSlice[10] = solana.Meta(EventAuthority)
	instruction.AccountMetaSlice[11] = solana.Meta(PUMPManager)
	instruction.AccountMetaSlice[12] = solana.Meta(globalVolumeAccumulation).WRITE()
	instruction.AccountMetaSlice[13] = solana.Meta(userVolumeAccumulation).WRITE()
	instruction.AccountMetaSlice[14] = solana.Meta(solana.MustPublicKeyFromBase58("8Wf5TiAheLUqBrKXeYg2JtAFFMWtKdG2BSFgqUcPVwTt")) //Fee Config:
	instruction.AccountMetaSlice[15] = solana.Meta(solana.MustPublicKeyFromBase58("pfeeUxB6jkeY1Hxd7CsFCAjcbHA9rWtchMGdZ6VojVZ"))  //Fee Program:

	*instrs = append(*instrs, instruction)
}

func addPumpSellIx(
	instrs *[]solana.Instruction,
	amountIn *big.Int,
	virtualSolReserves *big.Int,
	virtualTokenReserves *big.Int,
	slippage float32,
	owner solana.PublicKey,
	mint solana.PublicKey,
	creatorVault solana.PublicKey,
	globalSettingsPk solana.PublicKey,
	bondingCurvePk solana.PublicKey,
	associatedBondingCurvePk solana.PublicKey,
) {
	amountOut := QuoteSellByCurve(amountIn, virtualSolReserves, virtualTokenReserves)
	amountOutWithSlippage := applySlippage(amountOut, slippage)

	instruction := &PumpSellInstruction{
		MethodId:         PUMPSellMethod,
		AmountIn:         amountIn.Uint64(),
		AmountOutMin:     amountOutWithSlippage.Uint64(),
		AccountMetaSlice: make(solana.AccountMetaSlice, 14),
	}

	instruction.BaseVariant = bin.BaseVariant{
		Impl: instruction,
	}

	ataUser, _, _ := solana.FindAssociatedTokenAddress(owner, mint)

	// globalVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
	// 	[]byte("global_volume_accumulator"),
	// }, PUMPManager)

	// userVolumeAccumulation, _, _ := solana.FindProgramAddress([][]byte{
	// 	[]byte("user_volume_accumulator"),
	// 	owner.Bytes(),
	// }, PUMPManager)

	instruction.AccountMetaSlice[0] = solana.Meta(globalSettingsPk)
	instruction.AccountMetaSlice[1] = solana.Meta(global.PickRandomPumpProtocolFee()).WRITE()
	instruction.AccountMetaSlice[2] = solana.Meta(mint)
	instruction.AccountMetaSlice[3] = solana.Meta(bondingCurvePk).WRITE()
	instruction.AccountMetaSlice[4] = solana.Meta(associatedBondingCurvePk).WRITE()
	instruction.AccountMetaSlice[5] = solana.Meta(ataUser).WRITE()
	instruction.AccountMetaSlice[6] = solana.Meta(owner).WRITE().SIGNER()
	instruction.AccountMetaSlice[7] = solana.Meta(solana.SystemProgramID)
	instruction.AccountMetaSlice[8] = solana.Meta(creatorVault).WRITE()
	instruction.AccountMetaSlice[9] = solana.Meta(solana.TokenProgramID)
	instruction.AccountMetaSlice[10] = solana.Meta(EventAuthority)
	instruction.AccountMetaSlice[11] = solana.Meta(PUMPManager)
	// instruction.AccountMetaSlice[12] = solana.Meta(globalVolumeAccumulation).WRITE()
	// instruction.AccountMetaSlice[13] = solana.Meta(userVolumeAccumulation).WRITE()
	instruction.AccountMetaSlice[12] = solana.Meta(solana.MustPublicKeyFromBase58("8Wf5TiAheLUqBrKXeYg2JtAFFMWtKdG2BSFgqUcPVwTt")) //Fee Config:
	instruction.AccountMetaSlice[13] = solana.Meta(solana.MustPublicKeyFromBase58("pfeeUxB6jkeY1Hxd7CsFCAjcbHA9rWtchMGdZ6VojVZ"))  //Fee Program:

	*instrs = append(*instrs, instruction)
}
