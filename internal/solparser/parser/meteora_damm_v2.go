package parser

import (
	"reflect"
	"solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

func (s *SolParser) ProcessMeteoraDammV2SwapEvent(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	var acc accounts.MeteoraDAMMV2SwapAccounts
	var totalNumberOfCorrectAccount = reflect.TypeOf(acc).NumField()
	swapEvent := &types.SwapTransactionEvent{}
	if len(ix.Accounts) >= totalNumberOfCorrectAccount {
		acc = accounts.ParseAccountsIntoStruct[accounts.MeteoraDAMMV2SwapAccounts](ix.Accounts)
		swapEvent.PoolAddress = acc.Pool.String()
	} else {
		return nil, nil
	}
	return swapEvent, nil
}
