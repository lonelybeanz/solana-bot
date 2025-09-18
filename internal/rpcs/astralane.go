package rpcs

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"solana-bot/internal/global"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/zeromicro/go-zero/core/logx"
)

type AstralaneChannel struct {
	Name string
	Urls []string
	Tips []string
}

func NewAstralaneChannel() *AstralaneChannel {
	apiKey := "workbOV3jJl2xQa0YtLdZUWH2sUcNmOwUf7Tk88EfPb630nshyAvhDvepWJOJ4i0"
	return &AstralaneChannel{
		Name: "astralane",
		Urls: []string{
			// fmt.Sprintf("http://fr.gateway.astralane.io/iris?api-key=%s", apiKey),
			// fmt.Sprintf("http://lax.gateway.astralane.io/iris?api-key=%s", apiKey),
			// fmt.Sprintf("http://jp.gateway.astralane.io/iris?api-key=%s", apiKey),
			fmt.Sprintf("http://ny.gateway.astralane.io/iris?api-key=%s", apiKey),
			// fmt.Sprintf("http://ams.gateway.astralane.io/iris?api-key=%s", apiKey),
			// fmt.Sprintf("http://lim.gateway.astralane.io/iris?api-key=%s", apiKey),
		},
		Tips: []string{
			"astrazznxsGUhWShqgNtAdfrzP2G83DzcWVJDxwV9bF",
			"astra4uejePWneqNaJKuFFA8oonqCE1sqF6b45kDMZm",
			"astra9xWY93QyfG6yM8zwsKsRodscjQ2uU2HKNL5prk",
			"astraRVUuTHjpwEVvNBeQEgwYx9w9CFyfxjYoobCZhL",
			"astraEJ2fEj8Xmy6KLG7B3VfbKfsHXhHrNdCQx7iGJK",
			"astraubkDw81n4LuutzSQ8uzHCv4BhPVhfvTcYv8SKC",
			"astraZW5GLFefxNPAatceHhYjfA1ciq9gvfEg2S47xk",
			"astrawVNP4xDBKT7rAdxrLYiTSTdqtUr63fSMduivXK",
		},
	}
}

func (f *AstralaneChannel) GetTipInstruction(owner solana.PublicKey, tip uint64) solana.Instruction {
	randomIndex := rand.Intn(len(f.Tips))
	tipPublicKey := solana.MustPublicKeyFromBase58(f.Tips[randomIndex])
	return system.NewTransferInstruction(tip, owner, tipPublicKey).Build()
}

func (f *AstralaneChannel) SendTransaction(wallet solana.PrivateKey, tip uint64, txBuilder global.TxBuilder) (string, error) {

	txBuilder.AddInstruction(f.GetTipInstruction(wallet.PublicKey(), tip))

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

	// Serialize and base64 encode the signed transaction
	serializedTx, err := tx.MarshalBinary()
	if err != nil {
		logx.Errorf("Failed to serialize transaction: %v", err)
		return "", err
	}
	base64EncodedTx := base64.StdEncoding.EncodeToString(serializedTx)

	payload := []interface{}{
		base64EncodedTx,
		map[string]interface{}{
			"encoding":      "base64",
			"skipPreflight": true,
		},
		map[string]interface{}{
			"mevProtect": "false",
		},
	}

	sendTransactionJson := SendTransactionJson{
		Id:      1,
		Jsonrpc: "2.0",
		Methon:  "sendTransaction",
		Params:  payload,
	}

	jsonBody, err := json.Marshal(sendTransactionJson)
	if err != nil {
		logx.Errorf("JSON marshal error: %v", err)
		return "", err
	}

	resultCh := make(chan error, len(f.Urls)) // 缓冲通道，避免 goroutine 阻塞
	successCh := make(chan struct{})          // 成功信号

	for _, url := range f.Urls {
		go func(url string) {
			err := postToURL(url, jsonBody)
			if err != nil {
				logx.Errorf("❌ %s failed: %v\n", url, err)
				resultCh <- err
				return
			}
			successCh <- struct{}{}
		}(url)
	}

	logx.Infof("✅ [%s] tx sent: %s", f.Name, txSignature)

	// 等待第一个成功，或全部失败
	var failCount int
	for {
		select {
		case <-successCh:
			logx.Infof("🎉 [%s] One succeeded, exiting.", f.Name)
			return txSignature, nil
		case <-resultCh:
			failCount++
			if failCount == len(f.Urls) {
				return "", errors.New("❌ all requests failed")
			}
		case <-time.After(10 * time.Second):
			return "", errors.New("⏰ Timeout waiting for responses")
		}
	}

}
