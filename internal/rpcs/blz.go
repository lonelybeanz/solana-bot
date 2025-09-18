package rpcs

import (
	"context"
	"errors"
	"math/rand"
	"os"

	"solana-bot/internal/global"
	"time"

	"github.com/BlockRazorinc/solana-trader-client-go/pb/serverpb"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// 参考github.com/BlockRazorinc/solana-trader-client-go

type BlzChannel struct {
	Name    string
	Clients []serverpb.ServerClient
	Tips    []string
}

func NewBlzChannel() *BlzChannel {
	clients := []serverpb.ServerClient{
		// GetGrpcClient("frankfurt.solana-grpc.blockrazor.xyz:80"),
		GetGrpcClient("newyork.solana-grpc.blockrazor.xyz:80"),
		// GetGrpcClient("amsterdam.solana-grpc.blockrazor.xyz:80"),
		// GetGrpcClient("tokyo.solana-grpc.blockrazor.xyz:80"),
	}

	return &BlzChannel{
		Name:    "Blockrazor",
		Clients: clients,
		Tips: []string{
			"Gywj98ophM7GmkDdaWs4isqZnDdFCW7B46TXmKfvyqSm",
			"FjmZZrFvhnqqb9ThCuMVnENaM3JGVuGWNyCAxRJcFpg9",
			"6No2i3aawzHsjtThw81iq1EXPJN6rh8eSJCLaYZfKDTG",
			"A9cWowVAiHe9pJfKAj3TJiN9VpbzMUq6E4kEvf5mUT22",
			"68Pwb4jS7eZATjDfhmTXgRJjCiZmw1L7Huy4HNpnxJ3o",
			"4ABhJh5rZPjv63RBJBuyWzBK3g9gWMUQdTZP2kiW31V9",
			"B2M4NG5eyZp5SBQrSdtemzk5TqVuaWGQnowGaCBt8GyM",
			"5jA59cXMKQqZAVdtopv8q3yyw9SYfiE3vUCbt7p8MfVf",
			"5YktoWygr1Bp9wiS1xtMtUki1PeYuuzuCF98tqwYxf61",
			"295Avbam4qGShBYK7E9H5Ldew4B3WyJGmgmXfiWdeeyV",
			"EDi4rSy2LZgKJX74mbLTFk4mxoTgT6F7HxxzG2HBAFyK",
			"BnGKHAC386n4Qmv9xtpBVbRaUTKixjBe3oagkPFKtoy6",
			"Dd7K2Fp7AtoN8xCghKDRmyqr5U169t48Tw5fEd3wT9mq",
			"AP6qExwrbRgBAVaehg4b5xHENX815sMabtBzUzVB4v8S",
		},
	}
}

func GetGrpcClient(blzRelayEndpoint string) serverpb.ServerClient {
	// setup grpc connect
	conn, err := grpc.NewClient(blzRelayEndpoint,
		// grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})), // Enable tls configuration when connecting to a genernal endpoint
		grpc.WithTransportCredentials(insecure.NewCredentials()), // regional endpoints
		grpc.WithPerRPCCredentials(&Authentication{os.Getenv("BLZ_AUTH_KEY")}),
	)
	if err != nil {
		logx.Must(err)
	}

	// use the Gateway client connection interface
	client := serverpb.NewServerClient(conn)

	// grpc request warmup
	client.GetHealth(context.Background(), &serverpb.HealthRequest{})
	return client
}

func (c *BlzChannel) GetTipInstruction(owner solana.PublicKey, tip uint64) solana.Instruction {
	randomIndex := rand.Intn(len(c.Tips))
	tipPublicKey := solana.MustPublicKeyFromBase58(c.Tips[randomIndex])
	return system.NewTransferInstruction(tip, owner, tipPublicKey).Build()
}

func (c *BlzChannel) SendTransaction(wallet solana.PrivateKey, tip uint64, txBuilder global.TxBuilder) (string, error) {
	txBuilder.AddInstruction(c.GetTipInstruction(wallet.PublicKey(), tip))
	tx, err := txBuilder.BuildTx([]solana.PrivateKey{wallet})
	if err != nil {
		return "", err
	}

	// 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			return &wallet
		},
	)
	if err != nil {
		return "", err
	}

	txSignature := tx.Signatures[0].String()

	resultCh := make(chan error, len(c.Clients)) // 缓冲通道，避免 goroutine 阻塞
	successCh := make(chan struct{})             // 成功信号

	for _, client := range c.Clients {
		go func(client serverpb.ServerClient) {
			_, err := SendTransactionWithRelay(client, tx, true)
			if err != nil {
				logx.Errorf("❌ %v failed: %v\n", client, err)
				resultCh <- err
				return
			}
			successCh <- struct{}{}
		}(client)
	}

	logx.Infof("✅ [%s] tx sent: %s", c.Name, txSignature)

	// 等待第一个成功，或全部失败
	var failCount int
	for {
		select {
		case <-successCh:
			logx.Infof("🎉 [%s] One succeeded, exiting.", c.Name)
			return txSignature, nil
		case <-resultCh:
			failCount++
			if failCount == len(c.Clients) {
				return "", errors.New("❌ all requests failed")
			}
		case <-time.After(10 * time.Second):
			return "", errors.New("⏰ Timeout waiting for responses")
		}
	}

}

type Authentication struct {
	auth string
}

func (a *Authentication) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"apiKey": a.auth}, nil
}

func (a *Authentication) RequireTransportSecurity() bool {
	return false
}

func SendTransactionWithRelay(client serverpb.ServerClient, buyTx *solana.Transaction, skipPreflight bool) (string, error) {
	txBase64, _ := buyTx.ToBase64()
	// txHash := buyTx.Signatures[0].String()
	// log.Printf("【SendTransactionWithRelay】txHash: %s\n", txHash)
	logx.Infof("☯️【SendTransactionWithRelay】txBase64: %s\n", txBase64)
	// log.Printf("【SendTransactionWithRelay】slot: %d\n", slot)

	sendRes, err := client.SendTransaction(context.TODO(), &serverpb.SendRequest{
		Transaction:      txBase64,
		Mode:             "fast",
		RevertProtection: true,
	})

	if err != nil {
		// log.Printf("[%s]:Error sending  transaction: %v", txHash, err)
		return "", err
	}
	return sendRes.Signature, nil
}
