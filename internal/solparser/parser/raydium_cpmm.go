package parser

import (
	"fmt"
	"reflect"

	types2 "solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

func (s *SolParser) ParseRaydiumCpmmSwap(tx *rpc.ParsedInstruction) (*types2.SwapTransactionEvent, error) {
	var acc accounts.RaydiumCpmmSwapAccounts
	if len(tx.Accounts) != reflect.TypeOf(acc).NumField() {
		return nil, fmt.Errorf("invalid number of accounts")
	}
	acc = accounts.ParseAccountsIntoStruct[accounts.RaydiumCpmmSwapAccounts](tx.Accounts)
	swapEvent := &types2.SwapTransactionEvent{}
	swapEvent.PoolAddress = acc.PoolState.String()
	return swapEvent, nil
}
