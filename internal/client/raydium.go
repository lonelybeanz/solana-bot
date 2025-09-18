package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	// InputMint   = "7YpUFdHWmjbLF559hppQcixmx5napb3gWpg9LCgQ9oyd"
	// OutputMint  = "So11111111111111111111111111111111111111112"
	// Amount      = 100000
	// Slippage    = 1
	TxVersion = "V0"
	IsV0Tx    = true
	BaseHost  = "https://api.raydium.io"
	SwapHost  = "https://transaction-v1.raydium.io"
)

type PriorityFeeResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Data    struct {
		Default struct {
			VH int64 `json:"vh"`
			H  int64 `json:"h"`
			M  int64 `json:"m"`
		} `json:"default"`
	} `json:"data"`
}

type SwapQuoteResponse struct {
	ID      string                 `json:"id"`
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
}

type SwapTransactionResponse struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Success bool   `json:"success"`
	Data    []struct {
		Transaction string `json:"transaction"`
	} `json:"data"`
}

func RaySwapByApi(client *rpc.Client, wallet *solana.Wallet, InputMint, OutputMint, inputAccount, outputAccount string, Amount, Slippage uint64) error {
	// log.Println("Getting priority fee...")
	// // Get priority fee
	// priorityFeeResp, err := http.Get(fmt.Sprintf("%s/priority-fee", BaseHost))
	// if err != nil {
	// 	return fmt.Errorf("failed to get priority fee: %v", err)
	// }
	// defer priorityFeeResp.Body.Close()

	// respBody, err := io.ReadAll(priorityFeeResp.Body)
	// if err != nil {
	// 	return fmt.Errorf("failed to read priority fee response: %v", err)
	// }
	// log.Printf("Priority fee response: %s", string(respBody))

	// var feeData PriorityFeeResponse
	// if err := json.Unmarshal(respBody, &feeData); err != nil {
	// 	return fmt.Errorf("failed to decode priority fee response: %v", err)
	// }

	log.Println("Getting swap quote...")
	// Get swap quote
	swapURL := fmt.Sprintf("%s/compute/swap-base-in?inputMint=%s&outputMint=%s&amount=%d&slippageBps=%d&txVersion=%s",
		SwapHost, InputMint, OutputMint, Amount, Slippage*100, TxVersion)
	log.Printf("Swap URL: %s", swapURL)

	swapResp, err := http.Get(swapURL)
	if err != nil {
		return fmt.Errorf("failed to get swap quote: %v", err)
	}
	defer swapResp.Body.Close()

	swapRespBody, err := io.ReadAll(swapResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read swap quote response: %v", err)
	}
	log.Printf("Swap quote response: %s", string(swapRespBody))

	var swapQuote SwapQuoteResponse
	if err := json.Unmarshal(swapRespBody, &swapQuote); err != nil {
		return fmt.Errorf("failed to decode swap quote: %v", err)
	}

	if !swapQuote.Success {
		return fmt.Errorf("failed to get quote: %s", swapRespBody)
	}

	log.Println("Getting swap transactions...")
	// Get swap transactions
	swapTxBody := map[string]interface{}{
		"computeUnitPriceMicroLamports": "15000", //fmt.Sprintf("%d", feeData.Data.Default.H),
		"swapResponse":                  json.RawMessage(swapRespBody),
		"txVersion":                     TxVersion,
		"wallet":                        wallet.PublicKey().String(),
		"wrapSol":                       inputAccount == solana.WrappedSol.String(),
		"unwrapSol":                     false,
		"inputAccount":                  inputAccount,  //"2qCcBUbmChGyvAjT9QRamZRgpWp943YXYXTix7wdJ3xB",
		"outputAccount":                 outputAccount, //"AXLBAuVFPmGg9wnjNR1TWTGwoXuvYhdATseA7ibcSgH7",
	}

	swapTxJSON, err := json.Marshal(swapTxBody)
	if err != nil {
		return fmt.Errorf("failed to marshal swap tx body: %v", err)
	}
	log.Printf("Swap transaction request: %s", string(swapTxJSON))

	swapTxResp, err := http.Post(
		fmt.Sprintf("%s/transaction/swap-base-in", SwapHost),
		"application/json",
		strings.NewReader(string(swapTxJSON)),
	)
	if err != nil {
		return fmt.Errorf("failed to get swap transactions: %v", err)
	}
	defer swapTxResp.Body.Close()

	swapTxRespBody, err := io.ReadAll(swapTxResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read swap transaction response: %v", err)
	}
	log.Printf("Swap transaction response: %s", string(swapTxRespBody))

	var swapTxData SwapTransactionResponse
	if err := json.Unmarshal(swapTxRespBody, &swapTxData); err != nil {
		return fmt.Errorf("failed to decode swap transactions: %v", err)
	}

	if !swapTxData.Success {
		return fmt.Errorf("swap transaction request failed: %s", string(swapTxRespBody))
	}

	// Process and send transactions
	for idx, txData := range swapTxData.Data {
		log.Printf("Processing transaction %d...", idx+1)
		txBytes, err := base64.StdEncoding.DecodeString(txData.Transaction)
		if err != nil {
			return fmt.Errorf("failed to decode transaction %d: %v", idx+1, err)
		}

		if IsV0Tx {
			// Deserialize the versioned transaction
			tx, err := solana.TransactionFromBytes(txBytes)
			if err != nil {
				return fmt.Errorf("failed to deserialize transaction %d: %v", idx+1, err)
			}

			log.Printf("Signing transaction %d...", idx+1)
			// Sign the transaction
			_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
				if key.Equals(wallet.PublicKey()) {
					return &wallet.PrivateKey
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to sign transaction %d: %v", idx+1, err)
			}

			log.Printf("Sending transaction %d...", idx+1)
			// Send the transaction
			sig, err := client.SendTransactionWithOpts(context.Background(), tx,
				rpc.TransactionOpts{
					SkipPreflight: true,
				},
			)
			if err != nil {
				return fmt.Errorf("failed to send transaction %d: %v", idx+1, err)
			}

			fmt.Printf("Transaction %d sent, signature: %s\n", idx+1, sig)
			fmt.Printf("🔍http://solscan.io/tx/%s\n", sig)
		}
	}

	return nil
}
