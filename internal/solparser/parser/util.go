package parser

import (
	"errors"
	"fmt"
	"strconv"

	"solana-bot/internal/solparser/consts"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func validateTransaction(tx *rpc.GetParsedTransactionResult) error {
	if tx == nil || tx.Transaction == nil {
		return errors.New("parsedTransaction is nil")
	}
	if len(tx.Transaction.Message.AccountKeys) == 0 {
		return errors.New("no instructions found")
	}
	if len(tx.Transaction.Signatures) == 0 {
		return errors.New("no signatures found")
	}
	return nil
}

func createUniqueIndex(outerIdx, innerIdx int) (int, error) {
	prefIdx := fmt.Sprintf("%d%d", outerIdx+1, innerIdx+1)
	return strconv.Atoi(prefIdx)
}

func isSwapInstruction(programID string) bool {
	switch programID {
	case consts.RAYDIUM_V4_PROGRAM_ID,
		consts.RAYDIUM_CLMM_PROGRAM_ID,
		consts.ORCA_WHIRL_POOL_PROGRAM_ID,
		consts.ORCA_TOKEN_SWAP_PROGRAM_ID,
		consts.ORCA_TOKEN_SWAP_V2_PROGRAM_ID,
		consts.PUMP_FUN_PROGRAM_ID,
		consts.PUMP_AMM_PROGRAM_ID,
		consts.METEORA_DLMM_PROGRAM_ID,
		consts.PHNX_SWAP_PROGRAM_ID,
		consts.LIFINITY_SWAP_V2_PROGRAM_ID,
		consts.RAYDIUM_CPMM_PROGRAM_ID:
		return true
	default:
		return false
	}
}

func IsTokenProgramId(program solana.PublicKey) bool {
	return program == solana.TokenProgramID || program == solana.Token2022ProgramID
}

func IsSystemProgrmId(program solana.PublicKey) bool {
	return program == solana.SystemProgramID
}
