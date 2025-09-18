package client

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	solana "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func TestGetTx(t *testing.T) {
	wsClient, err := ws.Connect(context.Background(), WSUrl)
	if err != nil {
		t.Fatal(err)
	}
	rpcClient := rpc.New(HTTPUrl)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	out, err := WaitForTransaction(ctx, rpcClient, wsClient, "35EepQVTMKSZ3ZndSU7FLNqqJ6L7eSqJVdqqZYWWoipFwsTWdnzmnyjhR2ZSQhTGFcfrBJeRi4mKHXWQ6tze7wZG")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}

func TestBurn(t *testing.T) {
	rpcClient := rpc.New(HTTPUrl)
	wallet := solana.MustPrivateKeyFromBase58("your private key")

	tokenAddress := "4PpMir7mjTkf3ixqEnUxpibSzCMmSnxfZLCKWmnXyJ5X"
	mint := solana.MustPublicKeyFromBase58(tokenAddress)

	// 计算Associated Token Account的地址
	tokenAccount, _, err := solana.FindAssociatedTokenAddress(wallet.PublicKey(), mint)
	if err != nil {
		t.Fatal(err)
	}

	balance, err := rpcClient.GetTokenAccountBalance(context.Background(), tokenAccount, "confirmed")
	if err != nil {
		t.Fatal(err)
	}
	wDecimals, _ := new(big.Int).SetString(balance.Value.Amount, 10)

	tokens := []struct {
		Mint         solana.PublicKey
		TokenAccount solana.PublicKey
		Amount       uint64
	}{
		{
			mint,
			tokenAccount,
			wDecimals.Uint64(),
		},
	}
	txHash, err := BatchBurnAndClose(rpcClient, wallet, tokens)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Transaction Hash:", txHash)
}

func TestTracker(t *testing.T) {
	startTime := time.Now()
	privateKey := "" // replace with your base58 private key

	// Create a keypair from the secret key
	keypair, err := solana.PrivateKeyFromBase58(privateKey)
	if err != nil {
		log.Fatalf("Error creating keypair: %v", err)
	}

	rpcUrl := "https://api.mainnet-beta.solana.com"

	// Initialize a new Solana tracker with the keypair and RPC endpoint
	tracker := NewSolanaTracker(keypair, rpcUrl)

	priorityFee := 0.00005 // priorityFee requires a pointer, thus we store it in a variable

	// Get the swap instructions for the specified tokens, amounts, and other parameters
	swapResponse, err := tracker.GetSwapInstructions(
		"So11111111111111111111111111111111111111112",
		"2UqK55pmAWxdzTnfW1dGyi11msbgziCZbNpPrre6pump",
		0.0001,                       // Amount to swap
		30,                           // Slippage
		keypair.PublicKey().String(), // Payer public key
		&priorityFee,                 // Priority fee (Recommended while network is congested)
		false,
	)
	if err != nil {
		// Log and exit if there's an error getting the swap instructions
		log.Fatalf("Error getting swap instructions: %v", err)
	}

	maxRetries := uint(5) // maxRetries requires a pointer, thus we store it in a variable

	// Define the options for the swap transaction
	options := SwapOptions{
		SendOptions: rpc.TransactionOpts{
			SkipPreflight: true,
			MaxRetries:    &maxRetries,
		},
		ConfirmationRetries:        50,
		ConfirmationRetryTimeout:   1000 * time.Millisecond,
		LastValidBlockHeightBuffer: 200,
		Commitment:                 rpc.CommitmentProcessed,
		ResendInterval:             1500 * time.Millisecond,
		ConfirmationCheckInterval:  100 * time.Millisecond,
		SkipConfirmationCheck:      false,
	}

	// Perform the swap transaction with the specified options
	sendTime := time.Now()
	txid, err := tracker.PerformSwap(swapResponse, options)
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime).Seconds()
	if err != nil {
		fmt.Printf("Swap failed: %v\n", err)
		fmt.Printf("Time elapsed before failure: %.2f seconds\n", elapsedTime)
		// Add retries or additional error handling as needed
	} else {
		fmt.Printf("Transaction ID: %s\n", txid)
		fmt.Printf("Transaction URL: https://solscan.io/tx/%s\n", txid)
		fmt.Printf("Swap completed in %.2f seconds\n", elapsedTime)
		fmt.Printf("Transaction finished in %.2f seconds\n", endTime.Sub(sendTime).Seconds())
	}
}

func TestMemo(t *testing.T) {
	rpcClient := rpc.New(HTTPUrl)
	wallet := solana.MustPrivateKeyFromBase58("your private key")

	recevice := solana.MustPublicKeyFromBase58("nDn9URUkHhCH4eoLT2bLU3w6tXckfXx8ziVTmTceauQ")
	SendMemoWithTransfer(rpcClient, wallet, recevice, 1, "I can help you sell SCM ")
	// SendMemoWithTransfer(rpcClient, wallet, recevice, 1, "Transfer some tokens to me. I can attract more people.")
}

func TestNonceAccount(t *testing.T) {

	client := rpc.New(rpc.MainNetBeta_RPC)

	// 钱包
	// wallet := solana.NewWallet()
	wallet, err := solana.WalletFromPrivateKeyBase58("your private key")
	if err != nil {
		log.Fatal(err)
	}
	payer := wallet.PublicKey()

	// payer, err := solana.PrivateKeyFromBase58(key)
	// if err != nil {
	// 	panic(err)
	// }

	// 创建 nonce account
	nonceAccount := solana.NewWallet()

	fmt.Println("Nonce account:", nonceAccount.PublicKey())
	fmt.Println("PrivateKey:", nonceAccount.PrivateKey.String())

	// 获取租金豁免的最小 lamports（nonce account 大小）
	minBalance, err := client.GetMinimumBalanceForRentExemption(
		context.TODO(),
		nonceAccountSize,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Rent-exempt balance:", minBalance)

	// 获取最近 blockhash
	recentBlockhashResp, err := client.GetLatestBlockhash(context.TODO(), rpc.CommitmentConfirmed)
	if err != nil {
		log.Fatal(err)
	}
	blockhash := recentBlockhashResp.Value.Blockhash

	// 创建 + 初始化 nonce account
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewCreateAccountInstruction(
				minBalance,
				nonceAccountSize,
				system.ProgramID,
				payer,
				nonceAccount.PublicKey(),
			).Build(),
			system.NewInitializeNonceAccountInstruction(
				payer,
				nonceAccount.PublicKey(),
				solana.SysVarRecentBlockHashesPubkey,
				solana.SysVarRentPubkey,
			).Build(),
		},
		blockhash,
		solana.TransactionPayer(wallet.PublicKey()),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 签名
	tx.Sign(
		func(pub solana.PublicKey) *solana.PrivateKey {
			switch pub {
			case wallet.PublicKey():
				return &wallet.PrivateKey
			case nonceAccount.PublicKey():
				return &nonceAccount.PrivateKey
			default:
				return nil
			}
		},
	)

	// 发送交易
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
}

// 34t46xZJq1TjRKcfDrinmrRhpCT9JvQjLcs75EEgbQvcqLBwuRnExM67qTsGT9pCwYK77sefuWWyiepG4WdCx9ef
// 63RSXvZRPmVoZPrCXGDTwR35Ua7Xzj9AxGhbbFeuuHhfUGqPgf3DByNRr16HXCDiLiP5QwxRc77BvmJKk17xi9a4
// 4znPKt2yHyVyWCaJNTjuvmiuWkpyKY59M69V2UPP1ahPJUEVrXJq1TSSXxkBqrPzCQwvQVwTwkTw1UhZNADotS5d
func TestCloseNonceAccount(t *testing.T) {
	client := rpc.New(rpc.MainNetBeta_RPC)

	// 钱包
	// wallet := solana.NewWallet()
	wallet, err := solana.WalletFromPrivateKeyBase58("your private key")
	if err != nil {
		log.Fatal(err)
	}
	payer := wallet.PublicKey()

	// payer, err := solana.PrivateKeyFromBase58(key)
	// if err != nil {
	// 	panic(err)
	// }

	// 创建 nonce account
	nonceAccount := solana.MustPublicKeyFromBase58("4XxYgdxdSu7a3evtGcU1H7nbwm7LFq8MNKwTfdiKHEr3")

	// 获取租金豁免的最小 lamports（nonce account 大小）
	minBalance, err := client.GetMinimumBalanceForRentExemption(
		context.TODO(),
		nonceAccountSize,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Rent-exempt balance:", minBalance)

	// 获取最近 blockhash
	recentBlockhashResp, err := client.GetLatestBlockhash(context.TODO(), rpc.CommitmentConfirmed)
	if err != nil {
		log.Fatal(err)
	}
	blockhash := recentBlockhashResp.Value.Blockhash

	// 创建 + 初始化 nonce account
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewWithdrawNonceAccountInstruction(
				minBalance,
				nonceAccount,
				payer,
				solana.SysVarRecentBlockHashesPubkey,
				solana.SysVarRentPubkey,
				payer,
			).Build(),
		},
		blockhash,
		solana.TransactionPayer(wallet.PublicKey()),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 签名
	tx.Sign(
		func(pub solana.PublicKey) *solana.PrivateKey {
			switch pub {
			case wallet.PublicKey():
				return &wallet.PrivateKey
			default:
				return nil
			}
		},
	)

	// 发送交易
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
}

func TestGetNonce(t *testing.T) {
	client := rpc.New(rpc.MainNetBeta_RPC)
	nonceAccount := solana.MustPublicKeyFromBase58("3Fx3o9PnUmPaa2VPbmozrFvyLPYymoLn14pkxhkJbfGx")
	nonce, err := GetNonceAccountHash(client, nonceAccount)
	if err != nil {
		log.Fatalf("获取nonce失败: %v", err)
	}
	fmt.Println("nonce:", nonce.String())
}
