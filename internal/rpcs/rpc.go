package rpcs

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"solana-bot/internal/global"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type RpcModel struct {
	Name string
	Urls []string
	Tips []string
}

type RpcChannel interface {
	SendTransaction(wallet solana.PrivateKey, tip uint64, txBuilder global.TxBuilder) (string, error)
	GetTipInstruction(owner solana.PublicKey, tip uint64) solana.Instruction
}

type SendTransactionJson struct {
	Id      int64       `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
	Methon  string      `json:"method"`
	Params  interface{} `json:"params"`
}

func postToURL(url string, body []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "586cc34cdb9747a0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func SendTransaction(client *rpc.Client, tx *solana.Transaction) (string, error) {
	sig, err := client.SendTransactionWithOpts(
		context.TODO(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       true,
			PreflightCommitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		log.Fatalf("发送失败: %v", err)
	}
	fmt.Println("NonceAccount 创建成功:", sig.String())
	return sig.String(), nil
}
