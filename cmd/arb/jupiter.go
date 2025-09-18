package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	// quoteUrl = "https://lite-api.jup.ag/swap/v1/quote"
	// swapUrl  = "https://lite-api.jup.ag/swap/v1"

	quoteUrl = "http://127.0.0.1:8081/quote"
	swapUrl  = "http://127.0.0.1:8081"
)

// 交易路径
type RoutePlan struct {
	SwapInfo SwapInfo `json:"swapInfo"`
	Percent  int      `json:"percent"`
	Bps      int      `json:"bps"`
}

// 交换信息
type SwapInfo struct {
	AmmKey     string `json:"ammKey"`
	Label      string `json:"label"`
	InputMint  string `json:"inputMint"`
	OutputMint string `json:"outputMint"`
	InAmount   string `json:"inAmount"`
	OutAmount  string `json:"outAmount"`
	FeeAmount  string `json:"feeAmount"`
	FeeMint    string `json:"feeMint"`
}

type QuoteResponse struct {
	Error                string      `json:"error"`
	InputMint            string      `json:"inputMint"`
	OutputMint           string      `json:"outputMint"`
	InAmount             string      `json:"inAmount"`
	OutAmount            string      `json:"outAmount"`
	OtherAmountThreshold string      `json:"otherAmountThreshold"`
	SwapMode             string      `json:"swapMode"`
	SlippageBps          int         `json:"slippageBps"`
	PriceImpactPct       string      `json:"priceImpactPct"`
	RoutePlan            []RoutePlan `json:"routePlan"`
}

type SwapRequest struct {
	QuoteResponse                 *QuoteResponse             `json:"quoteResponse"`
	UserPublicKey                 string                     `json:"userPublicKey"`
	WrapAndUnwrapSol              bool                       `json:"wrapAndUnwrapSol,omitempty"`
	FeeAccount                    *string                    `json:"feeAccount,omitempty"` // 可能为空
	TrackingAccount               *string                    `json:"trackingAccount,omitempty"`
	AsLegacyTransaction           bool                       `json:"asLegacyTransaction,omitempty"`
	DestinationTokenAccount       *string                    `json:"destinationTokenAccount,omitempty"`
	DynamicComputeUnitLimit       bool                       `json:"dynamicComputeUnitLimit,omitempty"`
	SkipUserAccountsRpcCalls      bool                       `json:"skipUserAccountsRpcCalls,omitempty"`
	DynamicSlippage               bool                       `json:"dynamicSlippage,omitempty"`
	ComputeUnitPriceMicroLamports int64                      `json:"computeUnitPriceMicroLamports,omitempty"`
	PrioritizationFeeLamports     *PrioritizationFeeLamports `json:"prioritizationFeeLamports,omitempty"`
	UseSharedAccounts             bool                       `json:"useSharedAccounts"`
}

type AccountMeta struct {
	Pubkey     string `json:"pubkey"`
	IsSigner   bool   `json:"isSigner"`
	IsWritable bool   `json:"isWritable"`
}

type Instruction struct {
	ProgramId string        `json:"programId"`
	Accounts  []AccountMeta `json:"accounts"`
	Data      string        `json:"data"`
}

type SwapInstructionsResponse struct {
	TokenLedgerInstruction *Instruction `json:"tokenLedgerInstruction,omitempty"`
	// The necessary instructions to setup the compute budget.
	ComputeBudgetInstructions []Instruction `json:"computeBudgetInstructions"`
	// Setup missing ATA for the users.
	SetupInstructions  []Instruction `json:"setupInstructions"`
	SwapInstruction    *Instruction  `json:"swapInstruction"`
	CleanupInstruction *Instruction  `json:"cleanupInstruction,omitempty"`
	// The lookup table addresses that you can use if you are using versioned transaction.
	AddressLookupTableAddresses []string `json:"addressLookupTableAddresses"`
}

type PrioritizationFeeLamports struct {
	PriorityLevelWithMaxLamports *PriorityLevelWithMaxLamports `json:"priorityLevelWithMaxLamports,omitempty"`
	JitoTipLamports              int32                         `json:"jitoTipLamports,omitempty"`
}

type PriorityLevelWithMaxLamports struct {
	MaxLamports   int    `json:"maxLamports,omitempty"`
	PriorityLevel string `json:"priorityLevel,omitempty"`
}

type SwapResponse struct {
	Error                string `json:"error"`
	SwapTransaction      string `json:"swapTransaction"`
	LastValidBlockHeight int    `json:"lastValidBlockHeight"`
}

func SwapByJupiter(fromToken, toToken string, amount, slippageBps uint64, spender string) (*SwapResponse, error) {
	quote, err := Quote(fromToken, toToken, amount, slippageBps)
	if err != nil {
		return nil, err
	}
	return Swap(quote, spender)
}

// https://api.jup.ag/swap/v1/quote?inputMint=So11111111111111111111111111111111111111112&outputMint=EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v&amount=100000000&slippageBps=50&restrictIntermediateTokens=true
func Quote(fromToken, toToken string, amount, slippageBps uint64) (*QuoteResponse, error) {

	url := quoteUrl + "?inputMint=%s&outputMint=%s&amount=%d&slippageBps=%d&maxAccounts=60"
	apiUrl := fmt.Sprintf(url, fromToken, toToken, amount, slippageBps)
	resp, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[%s] request failed with status code: %d", apiUrl, resp.StatusCode)
	}
	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	// 解析 JSON 响应
	var quoteResponse QuoteResponse
	err = json.Unmarshal(body, &quoteResponse)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	return &quoteResponse, nil

}

func Swap(quote *QuoteResponse, spender string) (*SwapResponse, error) {
	start := time.Now()

	// 构建请求体
	swapRequest := SwapRequest{
		QuoteResponse:                 quote,
		UserPublicKey:                 spender, // 替换为实际的公钥
		WrapAndUnwrapSol:              false,
		DynamicComputeUnitLimit:       true,
		DynamicSlippage:               false,
		SkipUserAccountsRpcCalls:      true,
		UseSharedAccounts:             false,
		ComputeUnitPriceMicroLamports: 1,
	}

	// 将请求体序列化为 JSON
	requestBody, err := json.Marshal(swapRequest)
	if err != nil {
		fmt.Println("Error marshaling request body:", err)
		return nil, err
	}

	// 发起 POST 请求
	resp, err := http.Post(swapUrl+"/swap", "application/json", strings.NewReader(string(requestBody)))
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	// 解析 JSON 响应
	var swapResponse SwapResponse
	err = json.Unmarshal(body, &swapResponse)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	end := time.Now()
	fmt.Printf("[swap]cost time: %v\n", end.Sub(start))

	return &swapResponse, nil
}

func SwapInstructions(quote *QuoteResponse, spender string) (*SwapInstructionsResponse, error) {
	// 构建请求体
	swapRequest := SwapRequest{
		QuoteResponse:            quote,
		UserPublicKey:            spender, // 替换为实际的公钥
		WrapAndUnwrapSol:         false,
		DynamicComputeUnitLimit:  true,
		DynamicSlippage:          false,
		SkipUserAccountsRpcCalls: true,
		UseSharedAccounts:        false,
	}

	// 将请求体序列化为 JSON
	requestBody, err := json.Marshal(swapRequest)
	if err != nil {
		fmt.Println("Error marshaling request body:", err)
		return nil, err
	}
	// fmt.Println(string(requestBody))
	// 发起 POST 请求
	resp, err := http.Post(swapUrl+"/swap-instructions", "application/json", strings.NewReader(string(requestBody)))
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	// fmt.Println(string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// 解析 JSON 响应
	var swapInstructionsResponse SwapInstructionsResponse
	err = json.Unmarshal(body, &swapInstructionsResponse)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	return &swapInstructionsResponse, nil

}
