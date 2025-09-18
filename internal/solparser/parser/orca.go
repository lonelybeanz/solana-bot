package parser

import (
	"fmt"
	"reflect"

	"solana-bot/internal/solparser/consts"
	"solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

func (s *SolParser) ParseOrcaSwapEvent(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	switch ix.ProgramId.String() {
	case consts.ORCA_WHIRL_POOL_PROGRAM_ID:
		return s.ParseOrcaWhirlPoolSwapEvent(ix)
	case consts.ORCA_TOKEN_SWAP_PROGRAM_ID:
		return s.ParseOrcaSwapV1Event(ix)
	case consts.ORCA_TOKEN_SWAP_V2_PROGRAM_ID:
		return s.ParseOrcaSwapV2Event(ix)
	default:
		return nil, fmt.Errorf("unknown orca swap program: %s", ix.ProgramId.String())
	}
}

func (s *SolParser) ParseOrcaWhirlPoolSwapEvent(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	var acc accounts.OrcaSwapV2WhirlPoolAccounts
	var acc2 accounts.OrcaSwapWhirlPoolAccounts
	var totalNumberOfCorrectAccount = reflect.TypeOf(acc).NumField()
	var totalNumberOfCorrectAccount2 = reflect.TypeOf(acc2).NumField()
	swapEvent := &types.SwapTransactionEvent{}
	switch len(ix.Accounts) {
	case totalNumberOfCorrectAccount:
		acc = accounts.ParseAccountsIntoStruct[accounts.OrcaSwapV2WhirlPoolAccounts](ix.Accounts)
		swapEvent.PoolAddress = acc.Whirlpool.String()
	case totalNumberOfCorrectAccount2:
		acc2 = accounts.ParseAccountsIntoStruct[accounts.OrcaSwapWhirlPoolAccounts](ix.Accounts)

		swapEvent.PoolAddress = acc2.Whirlpool.String()
	default:
		return nil, fmt.Errorf("invalid number of accounts")
	}

	return swapEvent, nil
}

func (s *SolParser) ParseOrcaSwapV2Event(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	var acc accounts.OrcaSwapV2Accounts
	var totalNumberOfCorrectAccount = reflect.TypeOf(acc).NumField()
	swapEvent := &types.SwapTransactionEvent{}
	if len(ix.Accounts) != totalNumberOfCorrectAccount {
		return nil, fmt.Errorf("invalid number of accounts")
	}
	acc = accounts.ParseAccountsIntoStruct[accounts.OrcaSwapV2Accounts](ix.Accounts)

	swapEvent.PoolAddress = acc.TokenSwap.String()
	return swapEvent, nil
}

func (s *SolParser) ParseOrcaSwapV1Event(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	var acc accounts.OrcaSwapAccounts
	var totalNumberOfCorrectAccount = reflect.TypeOf(acc).NumField()
	swapEvent := &types.SwapTransactionEvent{}
	if len(ix.Accounts) != totalNumberOfCorrectAccount {
		return nil, fmt.Errorf("invalid number of accounts")
	}
	acc = accounts.ParseAccountsIntoStruct[accounts.OrcaSwapAccounts](ix.Accounts)
	swapEvent.PoolAddress = acc.TokenSwap.String()
	return swapEvent, nil
}
