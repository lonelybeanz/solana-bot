package parser

import (
	"fmt"
	"reflect"
	"solana-bot/internal/solparser/types"
	"solana-bot/internal/solparser/types/accounts"

	"github.com/gagliardetto/solana-go/rpc"
)

func (s *SolParser) ParseLifinitySwapEvent(ix *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {
	var acc accounts.LifinitySwapV2Accounts
	if len(ix.Accounts) != reflect.TypeOf(acc).NumField() {
		return nil, fmt.Errorf("invalid number of accounts")
	}
	acc = accounts.ParseAccountsIntoStruct[accounts.LifinitySwapV2Accounts](ix.Accounts)
	swapEvent := &types.SwapTransactionEvent{}
	swapEvent.PoolAddress = acc.Amm.String()
	return swapEvent, nil
}
