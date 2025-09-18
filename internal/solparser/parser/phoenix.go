package parser

import (
	"reflect"
	"solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

func (s *SolParser) ParsePhoenixSwapEvent(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	var acc accounts.PhoenixSwapAccounts

	if len(ix.Accounts) != reflect.TypeOf(acc).NumField() {
		return nil, nil
	}
	acc = accounts.ParseAccountsIntoStruct[accounts.PhoenixSwapAccounts](ix.Accounts)
	swapEvent := &types.SwapTransactionEvent{}
	swapEvent.PoolAddress = acc.Market.String()

	return swapEvent, nil
}
