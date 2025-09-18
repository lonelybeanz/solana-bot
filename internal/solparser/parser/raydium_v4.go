package parser

import (
	"errors"

	"solana-bot/internal/solparser/parser/coder"
	"solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

func (s *SolParser) ParseRaydiumAmmSwapEvent(ammSwapIx *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	ammDecoder := coder.NewRaydiumAmmInstructionCoder()
	_, instructionType, err1 := ammDecoder.Decode(ammSwapIx.Data)
	if err1 != nil {
		return nil, err1
	}
	if instructionType != coder.SwapBaseInInstruction && instructionType != coder.SwapBaseOutInstruction {

		return nil, errors.New("instructionType not supported")
	}
	acc2 := accounts.ParseAccountsIntoStruct[accounts.RaydiumSwapBaseAccounts](ammSwapIx.Accounts)
	swapEvent := &types.SwapTransactionEvent{
		PoolAddress: acc2.Amm.String(),
	}
	return swapEvent, nil
}
