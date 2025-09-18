package client

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/memo"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

const (
	HTTPUrl = "https://mainnet.helius-rpc.com/?api-key=cae6aa3f-a766-4ad7-a0fd-d9ae935d414e"
	WSUrl   = "wss://mainnet.helius-rpc.com/?api-key=cae6aa3f-a766-4ad7-a0fd-d9ae935d414e" // WebSocket 节点                           // Pump 合约的 Program ID

	nonceAccountSize = uint64(80)
)

func SimulateAndSendTransaction(rpcClient *rpc.Client, transaction *solana.Transaction) (string, error) {
	sopts := &rpc.SimulateTransactionOpts{
		ReplaceRecentBlockhash: true,
		Commitment:             rpc.CommitmentProcessed,
	}
	// 模拟交易
	out, err := rpcClient.SimulateTransactionWithOpts(context.Background(), transaction, sopts)
	if err != nil {
		return "", err
	}
	if out.Value.Err != nil {
		var logs string
		for _, log := range out.Value.Logs {
			logs += log + ""
		}
		return "", errors.New("simulation failed")
	}

	opts := rpc.TransactionOpts{
		SkipPreflight:       false, // 先开启预检查
		PreflightCommitment: rpc.CommitmentConfirmed,
	}

	// 发送交易
	signature, err := rpcClient.SendTransactionWithOpts(context.Background(), transaction, opts)
	if err != nil {
		log.Fatalf("SendTransaction failed: %v", err)
		return "", err
	}

	return signature.String(), nil
}

func SimulateTransaction(rpcClient *rpc.Client, transaction *solana.Transaction) error {
	out, err := rpcClient.SimulateTransaction(context.Background(), transaction)
	if err != nil {
		log.Fatalf("SimulateTransaction failed: %v", err)
	} else {
		fmt.Println("SimulateTransaction success:", out)
	}
	return nil

}

func SendTransactionWithJupiter(rpcClient *rpc.Client, wsClient *ws.Client, txStr string, senderPrivateKey *solana.PrivateKey) (string, error) {
	tx, err := solana.TransactionFromBase64(txStr)
	if err != nil {
		log.Fatalf("Invalid transaction: %v", err)
		return "", err
	}

	// 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			return senderPrivateKey
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("发送买入交易...")
	signature, err := rpcClient.SendTransactionWithOpts(context.Background(), tx, rpc.TransactionOpts{
		SkipPreflight:       true,
		PreflightCommitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		log.Fatal(err)
	}

	isSuccess := make(chan *bool)
	go SubForTransaction(wsClient, signature.String(), isSuccess)
	// 等待通道信号
	result := <-isSuccess
	if result != nil && *result {
		return signature.String(), nil
	} else {
		return "", errors.New("交易失败")
	}

}

func WaitForTransaction(ctx context.Context, client *rpc.Client, wsClient *ws.Client, txSignature string) (*rpc.GetTransactionResult, error) {
	isSuccess := make(chan *bool)

	if wsClient != nil {
		go SubForTransaction(wsClient, txSignature, isSuccess)
	}

	timeout := time.After(8 * time.Second) // 设定超时
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, errors.New("交易超时未确认:" + txSignature)

		case successPtr := <-isSuccess:
			if successPtr == nil {
				continue
			}
			if *successPtr {
				result, err := GetTransactionByHash(client, txSignature)
				if err != nil {
					continue
				} else {
					return result, nil
				}
			} else {
				return nil, errors.New("订阅通知交易失败:" + txSignature)
			}

		case <-ticker.C:
			result, err := GetTransactionByHash(client, txSignature)
			if err == nil {
				return result, nil
			}
		}
	}
}

func GetTransactionByHash(client *rpc.Client, txSignature string) (*rpc.GetTransactionResult, error) {
	// start := time.Now()
	// defer func() {
	// 	end := time.Now()
	// 	log.Printf("[GetTransactionByHash]cost time: %v", end.Sub(start))
	// }()

	maxSupportedTransactionVersion := uint64(0)
	signature, err := solana.SignatureFromBase58(txSignature)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction signature: %v", err)
	}

	// 交易确认后查询交易详情
	txResp, err := client.GetTransaction(
		context.Background(),
		signature,
		&rpc.GetTransactionOpts{
			Commitment:                     rpc.CommitmentConfirmed,
			Encoding:                       solana.EncodingBase58,
			MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction details: %v;txSignature:%s", err, txSignature)
	}

	return txResp, nil
}

func SubForTransaction(wsClient *ws.Client, txSignature string, isSuccess chan *bool) {
	defer close(isSuccess)
	log.Printf("SubForTransaction signature: %s \n", txSignature)
	signature, err := solana.SignatureFromBase58(txSignature)
	if err != nil {
		log.Printf("[%s]:Invalid transaction signature: %v \n", txSignature, err)
		return
	}
	// 监听交易状态
	sub, err := wsClient.SignatureSubscribe(
		signature,
		rpc.CommitmentProcessed,
	)
	if err != nil {
		log.Printf("[%s]:Failed to subscribe to transaction: %v \n", txSignature, err)
		return
	}
	defer sub.Unsubscribe()
	for {
		got, err := sub.RecvWithTimeout(80 * time.Second)
		// spew.Dump(got)
		if err != nil {
			log.Printf("[%s]:Failed to receive transaction: %v \n", txSignature, err)
			isSuccess <- new(bool) // 发送 false
			return
		}
		if got.Value.Err != nil {
			log.Printf("[%s]:Transaction failed: %v \n", txSignature, got.Value.Err)
			isSuccess <- new(bool) // 发送 false
			return
		} else {
			success := true
			isSuccess <- &success // 发送 true
			return
		}
	}
}

func SubForLogs(wsClient *ws.Client, programID solana.PublicKey, msg chan interface{}) {
	sub, err := wsClient.LogsSubscribeMentions(
		programID,
		rpc.CommitmentProcessed,
	)
	if err != nil {
		log.Printf("Failed to subscribe to logs: %v \n", err)
		return
	}
	defer sub.Unsubscribe()

	fmt.Println("开始监听 Pump 合约新盘创建事件...")

	for {
		// 接收事件
		got, err := sub.Recv(context.Background())

		//打印
		// spew.Dump(resp)

		if err == io.EOF {
			log.Println("Stream closed")
			return
		}
		if err != nil {
			// log.Fatalf("Error occurred in receiving update: %v", err)
			continue
		}
		msg <- got
	}

}

func BatchBurnAndClose(client *rpc.Client, wallet solana.PrivateKey, tokens []struct {
	Mint         solana.PublicKey
	TokenAccount solana.PublicKey
	Amount       uint64
}) (string, error) {
	instructions := buildBatchBurnAndCloseInstructions(tokens, wallet.PublicKey())
	recentBlockhash, err := client.GetLatestBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("获取最新区块哈希失败: %v", err)
	}
	txHash, err := sendBurnAndCloseTx(client, recentBlockhash.Value.Blockhash, instructions, wallet)
	if err != nil {
		return "", fmt.Errorf("发送交易失败: %v", err)
	}
	// // 等待交易确认
	// _, err = WaitForTransaction(client, nil, txHash)
	// if err != nil {
	// 	return "", fmt.Errorf("交易确认失败: %v", err)
	// }
	return txHash, nil
}

func sendBurnAndCloseTx(
	client *rpc.Client,
	recentBlockhash solana.Hash,
	instructions []solana.Instruction,
	signer solana.PrivateKey, // 钱包私钥
) (string, error) {
	tx, err := solana.NewTransaction(
		instructions,
		recentBlockhash,
		solana.TransactionPayer(signer.PublicKey()),
	)
	if err != nil {
		return "", fmt.Errorf("构建交易失败: %v", err)
	}

	// 签名交易
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(signer.PublicKey()) {
			return &signer
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("签名失败: %v", err)
	}

	sopts := &rpc.SimulateTransactionOpts{
		ReplaceRecentBlockhash: true,
		Commitment:             rpc.CommitmentProcessed,
	}
	//模拟交易
	out, err := client.SimulateTransactionWithOpts(context.Background(), tx, sopts)
	if err != nil {
		return "", fmt.Errorf("模拟交易失败: %v", err)
	}
	if out.Value.Err != nil {
		var logs string
		for _, log := range out.Value.Logs {
			logs += log + ""
		}
		fmt.Println(logs)
		return "", fmt.Errorf("模拟交易失败: %v", out.Value.Err)
	}

	// 发送交易
	txHash, err := client.SendTransactionWithOpts(
		context.TODO(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentConfirmed,
		},
	)
	if err != nil {
		return "", fmt.Errorf("发送失败: %v", err)
	}

	return txHash.String(), nil
}
func buildBatchBurnAndCloseInstructions(
	tokens []struct {
		Mint         solana.PublicKey
		TokenAccount solana.PublicKey
		Amount       uint64
	},
	wallet solana.PublicKey,
) []solana.Instruction {
	tokenProgramID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	instructions := []solana.Instruction{}

	for _, t := range tokens {
		if t.Amount > 0 {

			// inst := token.NewBurnInstruction(t.Amount, t.TokenAccount, t.Mint, wallet, []solana.PublicKey{}).Build()

			// Burn 指令
			burnData := make([]byte, 9)
			burnData[0] = 8 // Burn
			binary.LittleEndian.PutUint64(burnData[1:], t.Amount)

			burnAccounts := []*solana.AccountMeta{
				{PublicKey: t.TokenAccount, IsSigner: false, IsWritable: true},
				{PublicKey: t.Mint, IsSigner: false, IsWritable: true},
				{PublicKey: wallet, IsSigner: true, IsWritable: false},
			}
			instructions = append(instructions, solana.NewInstruction(tokenProgramID, burnAccounts, burnData))
		}

		// CloseAccount 指令
		closeData := []byte{9} // CloseAccount

		closeAccounts := []*solana.AccountMeta{
			{PublicKey: t.TokenAccount, IsSigner: false, IsWritable: true},
			{PublicKey: wallet, IsSigner: false, IsWritable: true}, // 收 lamports
			{PublicKey: wallet, IsSigner: true, IsWritable: false},
		}
		instructions = append(instructions, solana.NewInstruction(tokenProgramID, closeAccounts, closeData))
	}

	return instructions
}

func SendMemoWithTransfer(client *rpc.Client, sender solana.PrivateKey, receiver solana.PublicKey, lamports uint64, msg string) {
	// recentBlockhash, err := client.GetLatestBlockhash(context.TODO(), rpc.CommitmentFinalized)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// 转账指令
	transferIx := system.NewTransferInstruction(
		lamports,
		sender.PublicKey(),
		receiver,
	).Build()

	// Memo 指令
	memoIx := memo.NewMemoInstruction([]byte(msg), sender.PublicKey()).Build()

	nonceAccount := solana.MustPublicKeyFromBase58("4XxYgdxdSu7a3evtGcU1H7nbwm7LFq8MNKwTfdiKHEr3")
	nonce, err := GetNonceAccountHash(client, nonceAccount)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("nonce: ", nonce)

	// 构建交易
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, sender.PublicKey()).Build(),
			transferIx,
			memoIx,
		},
		*nonce,
		solana.TransactionPayer(sender.PublicKey()),
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(sender.PublicKey()) {
			return &sender
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	sig, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Fatalf("发送失败: %v", err)
	}
	log.Printf("✅ 成功发给 %s，签名: %s", receiver.String(), sig.String())
}

func GetNonceAccountHash(client *rpc.Client, nonceAccount solana.PublicKey) (*solana.Hash, error) {
	account, err := client.GetAccountInfo(context.Background(), nonceAccount)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("nonce account not found")
	}

	dec := bin.NewBinDecoder(account.Value.Data.GetBinary()) //.Decode(&nonceAccountData)
	acc := new(system.NonceAccount)

	err = acc.UnmarshalWithDecoder(dec)
	if err != nil {
		return nil, fmt.Errorf("solana.NewBinDecoder() => %v", err)
	}
	hash := solana.Hash(acc.Nonce)

	return &hash, nil
}
