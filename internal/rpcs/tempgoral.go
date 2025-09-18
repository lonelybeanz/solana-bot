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

type TempgoralChannel struct {
	Name string
	Urls []string
	Tips []string
}

func NewTempgoralChannel() *TempgoralChannel {
	apiKey := ""
	return &TempgoralChannel{
		Name: "astralane",
		Urls: []string{
			fmt.Sprintf("http://pit1.nozomi.temporal.xyz/?c=%s", apiKey),
			fmt.Sprintf("http://tyo1.nozomi.temporal.xyz/?c=%s", apiKey),
			fmt.Sprintf("http://fra2.nozomi.temporal.xyz/?c=%s", apiKey),
			fmt.Sprintf("http://ewr1.nozomi.temporal.xyz/?c=%s", apiKey),
			fmt.Sprintf("http://sgp1.nozomi.temporal.xyz/?c=%s", apiKey),
			fmt.Sprintf("http://ams1.nozomi.temporal.xyz/?c=%s", apiKey),
		},
		Tips: []string{
			"TEMPaMeCRFAS9EKF53Jd6KpHxgL47uWLcpFArU1Fanq",
			"noz3jAjPiHuBPqiSPkkugaJDkJscPuRhYnSpbi8UvC4",
			"noz3str9KXfpKknefHji8L1mPgimezaiUyCHYMDv1GE",
			"noz6uoYCDijhu1V7cutCpwxNiSovEwLdRHPwmgCGDNo",
			"noz9EPNcT7WH6Sou3sr3GGjHQYVkN3DNirpbvDkv9YJ",
			"nozc5yT15LazbLTFVZzoNZCwjh3yUtW86LoUyqsBu4L",
			"nozFrhfnNGoyqwVuwPAW4aaGqempx4PU6g6D9CJMv7Z",
			"nozievPk7HyK1Rqy1MPJwVQ7qQg2QoJGyP71oeDwbsu",
			"noznbgwYnBLDHu8wcQVCEw6kDrXkPdKkydGJGNXGvL7",
			"nozNVWs5N8mgzuD3qigrCG2UoKxZttxzZ85pvAQVrbP",
			"nozpEGbwx4BcGp6pvEdAh1JoC2CQGZdU6HbNP1v2p6P",
			"nozrhjhkCr3zXT3BiT4WCodYCUFeQvcdUkM7MqhKqge",
			"nozrwQtWhEdrA6W8dkbt9gnUaMs52PdAv5byipnadq3",
			"nozUacTVWub3cL4mJmGCYjKZTnE9RbdY5AP46iQgbPJ",
			"nozWCyTPppJjRuw2fpzDhhWbW355fzosWSzrrMYB1Qk",
			"nozWNju6dY353eMkMqURqwQEoM3SFgEKC6psLCSfUne",
			"nozxNBgWohjR75vdspfxR5H9ceC7XXH99xpxhVGt3Bb",
		},
	}
}

func (f *TempgoralChannel) GetTipInstruction(owner solana.PublicKey, tip uint64) solana.Instruction {
	randomIndex := rand.Intn(len(f.Tips))
	tipPublicKey := solana.MustPublicKeyFromBase58(f.Tips[randomIndex])
	return system.NewTransferInstruction(tip, owner, tipPublicKey).Build()
}

func (f *TempgoralChannel) SendTransaction(wallet solana.PrivateKey, tip uint64, txBuilder global.TxBuilder) (string, error) {

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
			"encoding": "base64",
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
