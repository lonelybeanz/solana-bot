package global

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"net/http"
// 	"time"

// 	"github.com/gagliardetto/solana-go"
// 	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
// 	"github.com/gagliardetto/solana-go/rpc"
// )

// func CloseATAs() {
// 	wallet, _ := solana.PrivateKeyFromBase58("your private key here")
// 	swap := token.NewRaydiumSwap(global.GetRPCForRequest(), wallet)

// 	tokens := GetTokenAccountsByOwner(wallet.PublicKey().String())
// 	count := 0
// 	for _, token_addr := range tokens {
// 		out, err_t := global.RPCServers[0].GetTokenLargestAccounts(context.TODO(), solana.MustPublicKeyFromBase58(token_addr), rpc.CommitmentFinalized)
// 		if err_t != nil {
// 			fmt.Println("GetLargestIssue", err_t.Error())
// 		}
// 		tx, _ := swap.CloseAccount(context.TODO(), out.Value[0].Address.String(), wallet, token_addr, wallet.PublicKey())
// 		var (
// 			signature *solana.Signature
// 			rec       *rpc.GetTransactionResult
// 			err       error
// 		)
// 		fmt.Println("RemoveSent:", tx.Signatures[0].String())
// 		rec, signature, err = utils.SendTransactionWaitConfirmed(tx)

// 		if signature == nil || err != nil || rec == nil {
// 			fmt.Println("Failed to sent transaction")
// 		} else {
// 			fmt.Println("Closed:")
// 			count++
// 		}
// 		time.Sleep(300 * time.Millisecond)
// 	}
// 	fmt.Println("Success:", count)
// }

// type RpcRequest struct {
// 	Jsonrpc string        `json:"jsonrpc"`
// 	ID      int           `json:"id"`
// 	Method  string        `json:"method"`
// 	Params  []interface{} `json:"params"`
// }

// type Parameter struct {
// 	Pubkey string `json:"pubkey"`
// 	Config Config `json:"config"`
// }

// type Config struct {
// 	Encoding string `json:"encoding"`
// }

// func GetTokenAccountsByOwner(walletAddress string) []string {
// 	url := "https://api.mainnet-beta.solana.com" // Change this URL to your Solana cluster endpoint

// 	requestBody, err := json.Marshal(RpcRequest{
// 		Jsonrpc: "2.0",
// 		ID:      1,
// 		Method:  "getTokenAccountsByOwner",
// 		Params: []interface{}{
// 			walletAddress,
// 			map[string]interface{}{
// 				"programId": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
// 			},
// 			map[string]interface{}{
// 				"encoding": "jsonParsed",
// 			},
// 		},
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer resp.Body.Close()

// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	type Result struct {
// 		Result struct {
// 			Value []struct {
// 				Pubkey  string `json:"pubkey"`
// 				Account struct {
// 					Data struct {
// 						Parsed struct {
// 							Info struct {
// 								Mint string `json:"mint"`
// 							} `json:"info"`
// 						} `json:"parsed"`
// 					} `json:"data"`
// 				} `json:"account"`
// 			} `json:"value"`
// 		} `json:"result"`
// 	}

// 	var result Result

// 	// Unmarshal the JSON data into the struct
// 	err = json.Unmarshal(body, &result)
// 	if err != nil {
// 		fmt.Println("Error unmarshalling JSON:", err)
// 		return []string{}
// 	}

// 	var pubkeys []string
// 	var tokens []string
// 	for _, item := range result.Result.Value {
// 		pubkeys = append(pubkeys, item.Pubkey)
// 		if item.Account.Data.Parsed.Info.Mint != global.Solana {
// 			tokens = append(tokens, item.Account.Data.Parsed.Info.Mint)
// 		}

// 	}
// 	fmt.Printf("%+v\n", tokens)
// 	return tokens
// }

// func (s *RaydiumSwap) CloseAccount(
// 	ctx context.Context,
// 	vault string,
// 	wallet solana.PrivateKey,
// 	token_addr string,
// 	account solana.PublicKey,
// ) (*solana.Transaction, error) {
// 	instrs := []solana.Instruction{}
// 	signers := []solana.PrivateKey{s.account}
// 	fromAccountDt, _, _ := solana.FindAssociatedTokenAddress(s.account.PublicKey(), solana.MustPublicKeyFromBase58(token_addr))

// 	account = fromAccountDt

// 	// Get the balance of the ATA
// 	balance, _ := utils.GetTokenBalance(wallet.PublicKey(), solana.MustPublicKeyFromBase58(token_addr))

// 	instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(100).Build())
// 	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(10000).Build())

// 	// Create a transfer instruction to send the balance to the recipient
// 	if vault != "" && balance.Uint64() > 0 {
// 		instrs = append(instrs, Transfer(TransferParam{
// 			From:    account,
// 			To:      solana.MustPublicKeyFromBase58(vault),
// 			Auth:    wallet.PublicKey(),
// 			Signers: nil,
// 			Amount:  balance.Uint64(),
// 		}))
// 	}

// 	closeInst, err := token.NewCloseAccountInstruction(
// 		account,
// 		s.account.PublicKey(),
// 		s.account.PublicKey(),
// 		[]solana.PublicKey{},
// 	).ValidateAndBuild()
// 	if err != nil {
// 		return nil, err
// 	}
// 	instrs = append(instrs, closeInst)

// 	tx, err := BuildTransaction(signers, wallet, instrs...)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return tx, nil
// }

type KeyedAccount struct {
	Pubkey  string `json:"pubkey"`
	Account struct {
		Lamports   int    `json:"lamports"`
		Data       string `json:"data"`
		Owner      string `json:"owner"`
		Executable bool   `json:"executable"`
		// RentEpoch  int64    `json:"rentEpoch"`
		// Space      int    `json:"space"`
	} `json:"account"`
}

type GetProgramAccountsResult struct {
	Result []*KeyedAccount `json:"result"`
}

func GetProgramAccounts(mintAddress string) (int, error) {

	// ✅ 构造 JSON 请求体
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getProgramAccounts",
		"params": []interface{}{
			"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA", // SPL Token Program
			map[string]interface{}{
				"dataSlice": map[string]interface{}{
					"offset": 0,
					"length": 0,
				},
				"filters": []map[string]interface{}{
					{"dataSize": 165}, // 过滤 Token 账户
					{"memcmp": map[string]interface{}{
						"offset": 0,
						"bytes":  mintAddress,
					}},
				},
			},
		},
	}

	// 将请求对象转换为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logx.Error("JSON 编码失败: ", err)
		return 0, err
	}

	// 发送 HTTP POST 请求
	resp, err := http.Post(RPCs[0], "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logx.Error("请求失败: ", err)
		return 0, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logx.Error("读取响应失败: ", err)
		return 0, err
	}
	var out GetProgramAccountsResult
	err = json.Unmarshal(body, &out)
	if err != nil {
		logx.Error("解析 JSON 失败: ", err)
		return 0, err
	}

	return len(out.Result), nil
}

func GetTopHolders(rpcClient *rpc.Client, mintAddress string) (map[string]uint64, error) {
	result := make(map[string]uint64)
	var out *rpc.GetTokenLargestAccountsResult
	var err error
	for i := 0; i < 5; i++ {
		out, err = rpcClient.GetTokenLargestAccounts(context.Background(), solana.MustPublicKeyFromBase58(mintAddress), rpc.CommitmentProcessed)
		if err != nil {
			if strings.Contains(err.Error(), "limit") {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if strings.Contains(err.Error(), "-32429") {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if strings.Contains(err.Error(), "Too many requests") {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if strings.Contains(err.Error(), "Invalid param") {
				time.Sleep(time.Second * 1)
				continue
			}
			logx.Error("GetTokenLargestAccounts", err.Error())
			return result, err
		} else {
			break
		}
	}
	if out == nil {
		logx.Error("GetTokenLargestAccounts", "out is nil")
		return result, errors.New("out is nil")
	}

	for _, item := range out.Value {
		fmt.Printf("Address: %s, Amount: %s\n", item.Address.String(), item.Amount)
		amountUint, err := strconv.ParseUint(item.Amount, 10, 64)
		if err != nil {
			logx.Error("解析金额失败: ", err)
			return result, err
		}
		result[item.Address.String()] = amountUint

	}
	return result, nil
}
