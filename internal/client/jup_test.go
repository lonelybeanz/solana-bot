package client

import (
	"context"
	"log"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func TestSwap(t *testing.T) {
	r, err := SwapByJupiter("So11111111111111111111111111111111111111112", "DZ63v1ATfZczB3JCSBc6HQXpZ8hduQGXiSmhvqM3pump", 100, 1, "7vNSg7vPnQDgg7APyAnvYCMW9xH3fNtDwxwhHQTsTmtU")
	if err != nil {
		t.Error(err)
	}
	t.Log(r)
}

func TestSend(t *testing.T) {
	r, err := SwapByJupiter("So11111111111111111111111111111111111111112", "DZ63v1ATfZczB3JCSBc6HQXpZ8hduQGXiSmhvqM3pump", 100, 1, "7vNSg7vPnQDgg7APyAnvYCMW9xH3fNtDwxwhHQTsTmtU")
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	// 连接到 WebSocket 节点
	wsClient, err := ws.Connect(ctx, WSUrl)
	if err != nil {
		log.Fatalf("WebSocket 连接失败: %v", err)
	}

	rpcClient := rpc.New(HTTPUrl)
	pk, err := solana.PrivateKeyFromBase58("")
	if err != nil {
		log.Fatal(err)
	}
	txHash, err := SendTransactionWithJupiter(rpcClient, wsClient, r.SwapTransaction, &pk)
	if err != nil {
		t.Error(err)
	}
	t.Log(txHash)
}

func TestRug(t *testing.T) {
	token := "E1CBmxJErm4UtHF7WZ83dkZ5ThmE6xeJ4yZkmym2pump"
	res, err := GetRugCheck(token)
	if err != nil {
		t.Error(err)
	}
	t.Log(res.CreatorBalance > 0)
	t.Log(uint64(0.00008 * 1e9))
}
