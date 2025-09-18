package solparser

import (
	"context"
	"fmt"
	"solana-bot/internal/solparser/consts"
	"solana-bot/internal/solparser/parser"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestMain(t *testing.T) {
	// Initialize RPC client
	client := rpc.New("https://api.mainnet-beta.solana.com")
	uint64One := uint64(1)

	// Create parser instance
	p := parser.NewSolParser(client)

	// Transaction signature to parse
	sig := solana.MustSignatureFromBase58("gkkuKB6uMXgePdbWkwxpMA6c3as5PzPErwBpNweYjWZsa521TvepmV73foYWbDnVd8jJYpMqPEUseyFEZvBHQYC")

	// Get parsed transaction
	opts := &rpc.GetParsedTransactionOpts{
		MaxSupportedTransactionVersion: &uint64One,
		Commitment:                     rpc.CommitmentConfirmed,
	}

	parsedTx, err := client.GetParsedTransaction(context.Background(), sig, opts)
	if err != nil {
		panic(err)
	}

	// Parse swap events
	events, err := p.ParseSwapEvent(parsedTx)
	if err != nil {
		panic(err)
	}

	// Process swap events
	for _, event := range events {
		fmt.Printf("Swap Event:\n")
		fmt.Printf("  Pool: %s\n", event.PoolAddress)
		fmt.Printf("  Market: %s\n", consts.ProgramToString(event.MarketProgramId))
		fmt.Printf("  Input Token: %s Amount: %s\n", event.InToken.Code, event.InToken.Amount)
		fmt.Printf("  Output Token: %s Amount: %s\n", event.OutToken.Code, event.OutToken.Amount)
	}
}
