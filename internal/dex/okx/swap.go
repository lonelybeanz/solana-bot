// Copyright (c) 2024-NOW imzhongqi <imzhongqi@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package dex

import (
	"context"

	"github.com/imzhongqi/okxos/client"
	"github.com/imzhongqi/okxos/dex"
	"github.com/zeromicro/go-zero/core/logx"
)

func NewDexAPI() *dex.DexAPI {
	client := client.NewClient("36221d92-173f-4715-99e6-1558301b2369", "C1634386CB15F7F56D6F4B5C24482F8C", "Doujiao)89",
		client.WithProjectID("bot"),
	)
	dexApi := dex.NewDexAPI(client)
	return dexApi
}

func GetSolSwapInstruction(dexApi *dex.DexAPI, spender, from, to, amount, slippage string) (*dex.GetSolSwapInstructionResult, error) {
	req := &dex.GetSolSwapInstructionRequest{
		ChainId:           "501",
		UserWalletAddress: spender,
		FromTokenAddress:  from,
		ToTokenAddress:    to,
		Amount:            amount,
		Slippage:          slippage,
	}
	resp, err := dexApi.GetSolSwapInstruction(context.Background(), req)
	if err != nil {
		logx.Errorf("ok swap error: %v", err)
	}
	return resp, err

}

// type Tx struct {
// 	Data                 string   `json:"data"`
// 	From                 string   `json:"from"`
// 	Gas                  string   `json:"gas"`
// 	GasPrice             string   `json:"gasPrice"`
// 	MaxPriorityFeePerGas string   `json:"maxPriorityFeePerGas"`
// 	MinReceiveAmount     string   `json:"minReceiveAmount"`
// 	SignatureData        []string `json:"signatureData"`
// 	To                   string   `json:"to"`
// 	Value                string   `json:"value"`
// }

// type GetSolSwapInstructionRequest struct {
// 	ChainId                         string
// 	Amount                          string
// 	FromTokenAddress                string
// 	ToTokenAddress                  string
// 	Slippage                        string
// 	UserWalletAddress               string
// 	SwapReceiverAddress             string
// 	FeePercent                      string
// 	FromTokenReferrerWalletAddress  string
// 	ToTokenReferrerWalletAddress    string
// 	DexIds                          []string
// 	PriceImpactProtectionPercentage string
// 	ComputeUnitPrice                string
// 	ComputeUnitLimit                string
// }

// type GetSolSwapInstructionResult struct {
// 	AddressLookupTableAccount []string          `json:"addressLookupTableAccount"`
// 	InstructionLists          []InstructionInfo `json:"instructionLists"`
// }

// type InstructionInfo struct {
// 	Data      string        `json:"data"`
// 	Accounts  []AccountInfo `json:"accounts"`
// 	ProgramId string        `json:"programId"`
// }

// type AccountInfo struct {
// 	IsSigner   bool   `json:"isSigner"`
// 	IsWritable bool   `json:"isWritable"`
// 	Pubkey     string `json:"pubkey"`
// }

// func (d DexAPI) GetSolSwapInstruction(ctx context.Context, req *GetSolSwapInstructionRequest) (result *GetSolSwapInstructionResult, err error) {
// 	params := map[string]string{
// 		"chainId":           req.ChainId,
// 		"amount":            req.Amount,
// 		"fromTokenAddress":  req.FromTokenAddress,
// 		"toTokenAddress":    req.ToTokenAddress,
// 		"slippage":          req.Slippage,
// 		"userWalletAddress": req.UserWalletAddress,
// 	}
// 	if req.SwapReceiverAddress != "" {
// 		params["swapReceiverAddress"] = req.SwapReceiverAddress
// 	}
// 	if req.FeePercent != "" {
// 		params["feePercent"] = req.FeePercent
// 	}
// 	if req.FromTokenReferrerWalletAddress != "" {
// 		params["fromTokenReferrerWalletAddress"] = req.FromTokenReferrerWalletAddress
// 	}
// 	if req.ToTokenReferrerWalletAddress != "" {
// 		params["toTokenReferrerWalletAddress"] = req.ToTokenReferrerWalletAddress
// 	}
// 	if len(req.DexIds) > 0 {
// 		params["dexIds"] = strings.Join(req.DexIds, ",")
// 	}
// 	if req.PriceImpactProtectionPercentage != "" {
// 		params["priceImpactProtectionPercentage"] = req.PriceImpactProtectionPercentage
// 	}
// 	if req.ComputeUnitPrice != "" {
// 		params["computeUnitPrice"] = req.ComputeUnitPrice
// 	}
// 	if req.ComputeUnitLimit != "" {
// 		params["computeUnitLimit"] = req.ComputeUnitLimit
// 	}

// 	if err = d.tr.Get(ctx, "/api/v5/dex/aggregator/swap-instruction", params, &result); err != nil {
// 		return nil, err
// 	}

// 	return result, nil
// }
