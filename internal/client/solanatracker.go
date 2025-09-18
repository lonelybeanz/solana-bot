package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

/*
SolanaTracker struct is used to interact with the Solana swap API.

Fields:
  - BaseURL: The base URL for the Solana swap API.
  - RPC: The URL for the Solana RPC node.
  - Keypair: The private key used for signing transactions.
*/
type SolanaTracker struct {
	BaseURL string            // The base URL for the Solana Swap API
	RPC     string            // The URL for the Solana RPC node.
	Keypair solana.PrivateKey // The private key used for signing transactions.
}

/*
NewSolanaTracker creates a new instance of SolanaTracker with the given keypair and RPC URL.

The base URL for the Solana Swap API is hardcoded as "https://swap-v2.solanatracker.io".

Parameters:
  - keypair: The private key used for signing transactions.
  - rpcURL: The URL for the Solana RPC node.

Returns:
  - *SolanaTracker: A pointer to a new SolanaTracker struct with the provided keypair, rpcURL, and base URL.
*/
func NewSolanaTracker(keypair solana.PrivateKey, rpcURL string) *SolanaTracker {
	return &SolanaTracker{
		BaseURL: "https://swap-v2.solanatracker.io",
		RPC:     rpcURL,
		Keypair: keypair,
	}
}

/*
TrackerSwapResponse is the response from SolanaTracker.

Contains the instructions to perform the swap.

Fields:
  - Txn: The base64-encoded transaction that needs to be sent.
  - ForceLegacy: Whether to force legacy mode for the transaction.
*/
type TrackerSwapResponse struct {
	Txn         string `json:"txn"`         // The base64-encoded transaction that needs to be sent.
	ForceLegacy bool   `json:"forceLegacy"` // Whether to force legacy mode for the transaction.
}

/*
GetSwapInstructions fetches the swap instructions from SolanaTracker.

Parameters:

  - fromToken: The address of the token to swap from.
  - toToken: The address of the token to swap to.
  - fromAmount: The amount of token to swap from.
  - slippage: Maximum tolerated price slippage before cancelling.
  - payer: The address of the payer for the swap.
  - priorityFee: The priority fee for the swap transaction.
  - forceLegacy: Wether to force legacy mode.

Returns:

  - TrackerSwapResponse: The instructions to perform the swap.
  - (Optional) Error if one occurs.
*/
func (st *SolanaTracker) GetSwapInstructions(fromToken string, toToken string, fromAmount float64, slippage float64, payer string, priorityFee *float64, forceLegacy bool) (*TrackerSwapResponse, error) {
	// Prepare the query parameters for the API request
	params := url.Values{}
	params.Add("from", fromToken)
	params.Add("to", toToken)
	params.Add("fromAmount", fmt.Sprintf("%f", fromAmount))
	params.Add("slippage", fmt.Sprintf("%f", slippage))
	params.Add("payer", payer)
	params.Add("forceLegacy", strconv.FormatBool(forceLegacy))
	if priorityFee != nil {
		params.Add("priorityFee", fmt.Sprintf("%f", *priorityFee))
	}

	// Construct the API request URL with the query parameters
	url := fmt.Sprintf("%s/swap?%s", st.BaseURL, params.Encode())

	// Send the API request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching swap instructions: %w", err)
	}
	defer resp.Body.Close()

	// Decode the JSON response into a TrackerSwapResponse struct
	var swapResp TrackerSwapResponse
	if err := json.NewDecoder(resp.Body).Decode(&swapResp); err != nil {
		return nil, fmt.Errorf("error decoding swap response: %w", err)
	}

	// Set the ForceLegacy field of the swap response to the value of the forceLegacy parameter
	swapResp.ForceLegacy = forceLegacy

	// Return the swap response
	return &swapResp, nil
}

/*
SwapOptions is a struct that contains options that can be used to customize the behavior of the transaction sending and confirmation process.

Fields:

  - SendOptions: A solana-go/rpc.TransactionOpts struct.
    It includes the preflight commitment level, skipping preflight, and maximum retries.
  - ConfirmationRetries: Number of times to retry confirming a transaction.
  - ConfirmationRetryTimeout: Maximum amount of time to wait for transaction confirmation.
  - LastValidBlockHeightBuffer: Number of blocks to buffer when checking the last valid block height.
  - Determines the required commitment level the transaction must have before it is considered successful.
  - ResendInterval: Amount of time to wait between resending a transaction.
  - ConfirmationCheckInterval: Amount of time to wait between transaction confirmation status checks.
  - SkipConfirmationCheck: Whether to skip the transaction confirmation check.
*/
type SwapOptions struct {
	SendOptions rpc.TransactionOpts

	/*
		Number of times to retry confirming a transaction.

		If the transaction confirmation fails, the client will retry confirming the transaction this many times before giving up.
	*/
	ConfirmationRetries int

	/*
		Maximum amount of time to wait for transaction confirmation.

		If transaction confirmation takes longer than this timeout, the client will give up and return an error.
	*/
	ConfirmationRetryTimeout time.Duration

	/*
		Number of blocks to buffer when checking the last valid block height.

		Used to ensure that the transaction is still valid when it is confirmed.
	*/
	LastValidBlockHeightBuffer uint64

	/*
		Determines the required commitment level the transaction must have before it is considered successful.

		Default: CommitmentFinalized

		Options:

			- CommitmentFinalized
			- CommitmentConfirmed
			- CommitmentProcessed

		Starting with solana-go@v1.5.5, commitment levels `max`, `recent`, `root`, `single` and `singleGossip` are deprecated.
	*/
	Commitment rpc.CommitmentType

	// Amount of time to wait between resending a transaction.
	ResendInterval time.Duration

	// Amount of time to wait between transaction confirmation status checks.
	ConfirmationCheckInterval time.Duration

	/*
		Whether to skip the transaction confirmation check.

		If true, the client will not wait for the transaction to be confirmed and will return the signature immediately after sending the transaction.
	*/
	SkipConfirmationCheck bool
}

/*
PerformSwap sends and confirms a transaction.

Parameters:

  - TrackerSwapResponse: The swap instructions provided by SolanaTracker.
  - options: The SwapOptions for the transaction

Returns:

  - Signature of the transaction.
  - (Optional) Error if one occurs.
*/
func (st *SolanaTracker) PerformSwap(trackerSwapResponse *TrackerSwapResponse, options SwapOptions) (string, error) {
	client := rpc.New(st.RPC)
	ctx := context.Background()

	// Decode the base64-encoded transaction from the swap response
	serializedTx, err := base64.StdEncoding.DecodeString(trackerSwapResponse.Txn)
	if err != nil {
		return "", fmt.Errorf("error decoding transaction: %w", err)
	}

	// Deserialize the transaction from its binary representation
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(serializedTx))
	if err != nil {
		return "", fmt.Errorf("error deserializing transaction: %w", err)
	}

	// Get the recent blockhash
	recentBlockhash, err := client.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("error getting recent blockhash: %w", err)
	}

	// Set the recent blockhash for the transaction
	tx.Message.RecentBlockhash = recentBlockhash.Value.Blockhash

	// Sign the transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if st.Keypair.PublicKey().Equals(key) {
			return &st.Keypair
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error signing transaction: %w", err)
	}

	// Send and confirm the transaction
	signature, err := st.sendAndConfirmTransaction(client, ctx, tx, options)
	if err != nil {
		return "", fmt.Errorf("error sending and confirming transaction: %w", err)
	}

	// If successful, return the signature
	return signature.String(), nil
}

/*
sendAndConfirmTransaction sends a transaction and confirms it.

Parameters:

- client:
  - context: An empty context.
  - transaction: The signed transaction to send.
  - options: The SwapOptions for the transaction.

Returns:

  - Signature of the transaction.
  - (Optional) Error if one occurs.
*/
func (st *SolanaTracker) sendAndConfirmTransaction(client *rpc.Client, ctx context.Context, tx *solana.Transaction, options SwapOptions) (solana.Signature, error) {
	// Send the transaction using the provided client and options
	sig, err := client.SendTransactionWithOpts(ctx, tx, options.SendOptions)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("error sending transaction: %w", err)
	}

	// If SkipConfirmationCheck is true, return the signature immediately without confirmation
	if options.SkipConfirmationCheck {
		return sig, nil
	}

	// Loop through the number of confirmation retries specified in the options
	for i := 0; i < options.ConfirmationRetries; i++ {
		// Get the signature status using the provided client and signature
		status, err := client.GetSignatureStatuses(ctx, false, sig)
		if err != nil {
			return solana.Signature{}, fmt.Errorf("error getting signature status: %w", err)
		}

		// Check if the signature status is not nil and if there was an error
		if status.Value[0] != nil {
			if status.Value[0].Err != nil {
				return solana.Signature{}, fmt.Errorf("transaction failed: %v", status.Value[0].Err)
			}
			// Check if the confirmation status is finalized
			if status.Value[0].ConfirmationStatus >= rpc.ConfirmationStatusFinalized {
				// Return the signature if the transaction is confirmed
				return sig, nil
			}
		}

		// Wait for the confirmation check interval before retrying
		time.Sleep(options.ConfirmationCheckInterval)
	}

	// Return an error if the transaction confirmation timed out
	return solana.Signature{}, fmt.Errorf("transaction confirmation timed out")
}
