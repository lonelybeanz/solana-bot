package dex

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestParser(t *testing.T) {
	// Set up RPC client for Solana mainnet
	rpcClient := rpc.New(rpc.MainNetBeta.RPC)

	txSig := solana.MustSignatureFromBase58("3foFs6SxaS9DwbokMnZmC4QvYBKYLEjY5xd2kGhh8pzoDw68jUF5Uu1UgMDruAQSiUW3CKq7hCQKAW3HP5UqYopm")

	// Specify the maximum transaction version supported
	var maxTxVersion uint64 = 0

	// Fetch the transaction data using the RPC client
	tx, err := rpcClient.GetTransaction(
		context.TODO(),
		txSig,
		&rpc.GetTransactionOpts{
			Commitment:                     rpc.CommitmentConfirmed,
			MaxSupportedTransactionVersion: &maxTxVersion,
		},
	)
	if err != nil {
		log.Fatalf("Error fetching transaction: %s", err)
	}

	// Initialize the transaction parser
	parser, err := solanaswapgo.NewTransactionParser(tx)
	if err != nil {
		log.Fatalf("Error initializing transaction parser: %s", err)
	}

	// Parse the transaction to extract basic data
	transactionData, err := parser.ParseTransactionForSwap()
	if err != nil {
		log.Fatalf("Error parsing transaction: %s", err)
	}

	// Print the parsed transaction data
	// marshalledData, _ := json.MarshalIndent(transactionData, "", "  ")
	// // fmt.Println(string(marshalledData))

	// Process and extract swap-specific data from the parsed transaction
	swapData, err := parser.ProcessSwapData(transactionData)
	if err != nil {
		log.Fatalf("Error processing swap data: %s", err)
	}

	// Print the parsed swap data
	marshalledSwapData, _ := json.MarshalIndent(swapData, "", "  ")
	fmt.Println(string(marshalledSwapData))
}

func TestCreate(t *testing.T) {
	// Set up RPC client for Solana mainnet
	rpcClient := rpc.New(rpc.MainNetBeta.RPC)

	txSig := solana.MustSignatureFromBase58("2U4hBbTBpXEiXmwXmqpx4U1PFvRkbUQQfhpD9uSRhdDU7xgfdK72sm5Un46EZ3P3uPNiFHsJ2qVuNqKBPwrpq418")

	// Specify the maximum transaction version supported
	var maxTxVersion uint64 = 0

	// Fetch the transaction data using the RPC client
	tx, err := rpcClient.GetTransaction(
		context.TODO(),
		txSig,
		&rpc.GetTransactionOpts{
			Commitment:                     rpc.CommitmentConfirmed,
			MaxSupportedTransactionVersion: &maxTxVersion,
		},
	)
	if err != nil {
		log.Fatalf("Error fetching transaction: %s", err)
	}

	// Initialize the transaction parser
	parser, err := solanaswapgo.NewTransactionParser(tx)
	if err != nil {
		log.Fatalf("Error initializing transaction parser: %s", err)
	}

	// Parse the transaction to extract basic data
	mintData, err := parser.ParseTransactionForMint()
	if err != nil {
		log.Fatalf("Error parsing transaction: %s", err)
	}

	// Print the parsed swap data
	marshalledSwapData, _ := json.MarshalIndent(mintData, "", "  ")
	fmt.Println(string(marshalledSwapData))

}
