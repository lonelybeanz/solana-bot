package main

import (
	"context"
	"fmt"

	"github.com/BlockRazorinc/solana-trader-client-go/pb/serverpb"
	"github.com/gagliardetto/solana-go"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	BribeAmount  = uint64(1e6)
	BribeAccount = solana.MustPublicKeyFromBase58("Gywj98ophM7GmkDdaWs4isqZnDdFCW7B46TXmKfvyqSm")
)

type Authentication struct {
	apiKey string
}

func (a *Authentication) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"apiKey": a.apiKey}, nil
}

func (a *Authentication) RequireTransportSecurity() bool {
	return false
}

func GetServerClient() serverpb.ServerClient {
	blzRelayEndpoint := "newyork.solana-grpc.blockrazor.xyz:80"
	// blzRelayEndpoint := "tokyo.solana-grpc.blockrazor.xyz:80"
	authKey := "your authKey"
	// setup grpc connect
	conn, err := grpc.NewClient(blzRelayEndpoint,
		// grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})), // Enable tls configuration when connecting to a genernal endpoint
		grpc.WithTransportCredentials(insecure.NewCredentials()), // regional endpoints
		grpc.WithPerRPCCredentials(&Authentication{authKey}),
	)
	if err != nil {
		panic(fmt.Sprintf("connect error: %v", err))
	}

	// use the Gateway client connection interface
	client := serverpb.NewServerClient(conn)

	// grpc request warmup
	client.GetHealth(context.Background(), &serverpb.HealthRequest{})

	return client

}

func SendTransactionWithRelay(client serverpb.ServerClient, buyTx *solana.Transaction, skipPreflight bool) (string, error) {
	txBase64, _ := buyTx.ToBase64()
	txHash := buyTx.Signatures[0].String()
	// log.Printf("【SendTransactionWithRelay】txHash: %s\n", txHash)
	// log.Printf("【SendTransactionWithRelay】txBase64: %s\n", txBase64)
	// log.Printf("【SendTransactionWithRelay】slot: %d\n", slot)

	sendRes, err := client.SendTransaction(context.TODO(), &serverpb.SendRequest{
		Transaction:      txBase64,
		Mode:             "fast",
		RevertProtection: true,
	})

	if err != nil {
		logx.Errorf("[%s]:Error sending  transaction: %v", txHash, err)
		return "", err
	}
	return sendRes.Signature, nil
}
