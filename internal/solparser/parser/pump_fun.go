package parser

import (
	"errors"
	"fmt"
	"solana-bot/internal/solparser/consts"
	"solana-bot/internal/solparser/parser/coder"
	"solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

var (
	PumpFunEventAuthority = "Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1"
)

func (s *SolParser) ParsePumpFunCpiLog(ix *rpc.ParsedInstruction, swapEvent *types.SwapTransactionEvent) (*types.SwapTransactionEvent, error) {
	if ix.ProgramId.String() != consts.PUMP_FUN_PROGRAM_ID {
		return nil, errors.New("not a pump.fun swap instruction")
	}

	if len(ix.Accounts) != 1 || ix.Accounts[0].String() != PumpFunEventAuthority {
		return nil, errors.New("bad accounts")
	}

	data, err := coder.DecodePumpFunCpiLog(ix.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding pump.fun cpi log: %w", err)
	}
	if data.IsBuy {
		swapEvent.OutToken.Code = data.Mint.String()
		swapEvent.OutToken.Amount = fmt.Sprintf("%d", data.TokenAmount)
		swapEvent.InToken.Code = consts.SOL_TOKEN_PROGRAM_ID
		swapEvent.InToken.Amount = fmt.Sprintf("%d", data.SolAmount)
	} else {
		swapEvent.OutToken.Code = consts.SOL_TOKEN_PROGRAM_ID
		swapEvent.OutToken.Amount = fmt.Sprintf("%d", data.SolAmount)
		swapEvent.InToken.Code = data.Mint.String()
		swapEvent.InToken.Amount = fmt.Sprintf("%d", data.TokenAmount)
	}
	return swapEvent, nil
}

func (s *SolParser) ParsePumpFunSwapEvent(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	if ix.ProgramId.String() != consts.PUMP_FUN_PROGRAM_ID {
		return nil, errors.New("not a pump.fun swap instruction")
	}

	if len(ix.Accounts) == 1 && ix.Accounts[0].String() == PumpFunEventAuthority {
		return nil, nil
	}

	if len(ix.Accounts) != 12 {
		return nil, errors.New("invalid number of accounts")
	}

	acc := accounts.ParseAccountsIntoStruct[accounts.PumpFunSwapAccounts](ix.Accounts)

	swapEvent := &types.SwapTransactionEvent{
		PoolAddress: acc.Global.String(),
	}
	return swapEvent, nil
}
