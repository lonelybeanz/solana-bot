package parser

import (
	"reflect"
	"solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

func (s *SolParser) ProcessMeteoraSwapEvent(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	var acc accounts.MeteoraDLMMSwapAccounts
	var totalNumberOfCorrectAccount = reflect.TypeOf(acc).NumField()
	swapEvent := &types.SwapTransactionEvent{}
	if len(ix.Accounts) >= totalNumberOfCorrectAccount {
		acc = accounts.ParseAccountsIntoStruct[accounts.MeteoraDLMMSwapAccounts](ix.Accounts)
		swapEvent.PoolAddress = acc.LbPair.String()
	} else {
		return nil, nil
	}
	return swapEvent, nil
}
