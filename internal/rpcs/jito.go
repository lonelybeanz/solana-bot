package rpcs

import (
	"context"
	"math/rand"
	"solana-bot/internal/global"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"

	jito_go "github.com/weeaa/jito-go"
	"github.com/weeaa/jito-go/clients/searcher_client"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	jitoName = "Jito"
	rpcAddr  = "https://mainnet.block-engine.jito.wtf"
)

type JitoChannel struct {
	clients []*searcher_client.Client
}

func NewJitoChannel() *JitoChannel {

	clients := []*searcher_client.Client{}

	for k, v := range jito_go.JitoEndpoints {
		if k != "NY" {
			continue
		}
		ctx := context.Background()
		client, err := searcher_client.NewNoAuth(
			ctx,
			v.BlockEngineURL,
			rpc.New(rpcAddr),
			rpc.New(rpc.MainNetBeta_RPC),
			"",
			nil,
		)
		if err != nil {
			logx.Errorf("创建 Jito 客户端失败: %v", err)
			continue
		}
		clients = append(clients, client)
	}

	return &JitoChannel{
		clients: clients,
	}
}

func (c *JitoChannel) GetTipInstruction(owner solana.PublicKey, tip uint64) solana.Instruction {
	randomIndex := rand.Intn(len(jito_go.MainnetTipAccounts))
	tipPublicKey := jito_go.MainnetTipAccounts[randomIndex]

	return system.NewTransferInstruction(tip, owner, tipPublicKey).Build()
}

func (c *JitoChannel) SendTransaction(wallet solana.PrivateKey, tip uint64, txBuilder global.TxBuilder) (string, error) {
	// max per bundle is 5 transactions
	txns := make([]*solana.Transaction, 0, 5)

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

	// spew.Dump(tx)

	txns = append(txns, tx)

	for _, c := range c.clients {
		go func(c *searcher_client.Client) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			_, err := c.BroadcastBundleWithConfirmation(ctx, txns)
			if err != nil {
				logx.Errorf("❌[%s] send err:%v", jitoName, err)
			}
		}(c)
	}

	logx.Infof("✅[%s] tx sent: %s", jitoName, txSignature)

	return txSignature, nil
}
