package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	solana "github.com/gagliardetto/solana-go"
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type SwapRoute struct {
	Quote struct {
		InputMint  string `json:"inputMint"`
		InAmount   string `json:"inAmount"`
		OutputMint string `json:"outputMint"`
		OutAmount  string `json:"outAmount"`
	} `json:"quote"`
	RawTx struct {
		SwapTransaction      string `json:"swapTransaction"`
		LastValidBlockHeight int64  `json:"lastValidBlockHeight"`
	}
}

type SwapData struct {
	OrderId              string `json:"order_id"`
	BundleId             string `json:"bundle_id"`
	LastValidBlockNumber int64  `json:"last_valid_block_number"`
	TxHash               string `json:"tx_hash"`
}

func GetSwap(inAddress, outToken, amount, fromAddress, slippage string) (*Result, error) {

	urlFormat := "https://gmgn.ai/defi/router/v1/sol/tx/get_swap_route?token_in_address=%s&token_out_address=%s&in_amount=%s&from_address=%s&slippage=%s"
	apiUrl := fmt.Sprintf(urlFormat, inAddress, outToken, amount, fromAddress, slippage)
	resp, err := http.Get(apiUrl)
	if err != nil {
		log.Fatalf("Failed to get swap route: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: %s", resp.Status)
		return nil, fmt.Errorf("error: %s", resp.Status)
	}
	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
		return nil, err
	}
	if result.Code != 0 {
		log.Fatalf("Error: %s", result.Msg)
		return nil, fmt.Errorf("error: %s", result.Msg)
	}
	return &result, nil

}

func SendTransaction(signedTx, from_address string, signer solana.PrivateKey) (*Result, error) {
	url := "https://gmgn.ai/defi/router/v1/sol/tx/submit_signed_bundle_transaction"

	var tx solana.Transaction
	err := tx.UnmarshalBase64(signedTx)
	if err != nil {
		log.Fatal("解析 VersionedTransaction 失败:", err)
	}

	// 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			return &signer
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	payload := map[string]string{
		"signed_tx":    tx.MustToBase64(),
		"from_address": from_address,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
		return nil, err
	}
	// 发送 POST 请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: %s", resp.Status)
		return nil, fmt.Errorf("error: %s", resp.Status)
	}
	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
		return nil, err
	}
	if result.Code != 0 {
		log.Fatalf("Error: %s", result.Msg)
		return nil, fmt.Errorf("error: %s", result.Msg)
	}
	return &result, nil
}
