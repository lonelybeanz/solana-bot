package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"strconv"

	"github.com/BlockRazorinc/solana-trader-client-go/pb/serverpb"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

type ArbBot struct {
	rpcClient  *rpc.Client
	wallet     *solana.Wallet
	sendClient serverpb.ServerClient
}

func NewArbBot() *ArbBot {
	wallet, err := solana.WalletFromPrivateKeyBase58("your private key")
	if err != nil {
		logx.Must(err)
		return nil
	}
	return &ArbBot{
		rpcClient:  rpc.New(endpointRPC),
		wallet:     wallet,
		sendClient: GetServerClient(),
	}
}

func (a *ArbBot) Start() {
	go a.run()
}

func (a *ArbBot) run() {
	for {
		// 2. quote0
		quote0, err := Quote(wSolMint.String(), usdcMint.String(), buyAmount, 0)
		if err != nil {
			logx.Errorf("quote0 error: %v", err)
			return
		}

		// 3. quote1
		quote1, err := Quote(usdcMint.String(), wSolMint.String(), uint64(MustDecodeUint64(quote0.OutAmount)), 0)
		if err != nil {
			logx.Errorf("quote1 error: %v", err)
			return
		}

		// 4. 计算利润
		diffLamports := MustDecodeUint64(quote1.OutAmount) - MustDecodeUint64(quote0.InAmount)

		// threhold
		thre := int64(2e5 + BribeAmount)
		if diffLamports > thre {
			fmt.Println("利润:", diffLamports)
			quote0.OutputMint = quote1.OutputMint
			quote0.OutAmount = strconv.FormatUint(buyAmount+BribeAmount, 10)
			quote0.OtherAmountThreshold = strconv.FormatUint(buyAmount+BribeAmount, 10)
			quote0.PriceImpactPct = "0"
			quote0.RoutePlan = append(quote0.RoutePlan, quote1.RoutePlan...)
			a.SwapSend(quote0)
		}
	}
}

func MustDecodeUint64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

func (a *ArbBot) SwapSend(quote *QuoteResponse) {
	r, err := SwapInstructions(quote, a.wallet.PublicKey().String())
	if err != nil {
		logx.Errorf("Error creating swap transaction: %s", err)
		return
	}
	// Create an array of instructions
	var instructions []solana.Instruction

	// Add compute budget instruction if present
	for _, instruction := range r.ComputeBudgetInstructions {
		instructions = append(instructions, createTransactionInstruction(instruction))
	}

	// Add setup instructions
	var i, j int
	for _, instruction := range r.SetupInstructions {
		if instruction.ProgramId == solana.SystemProgramID.String() {
			i++
			continue
		}
		if instruction.ProgramId == solana.SPLAssociatedTokenAccountProgramID.String() {
			j++
			continue
		}
		if instruction.ProgramId == solana.TokenProgramID.String() {
			j++
			continue
		}
		instructions = append(instructions, createTransactionInstruction(instruction))
	}

	// Add main swap instruction
	instructions = append(instructions, createTransactionInstruction(*r.SwapInstruction))

	// Add cleanup instruction if present
	// instructions = append(instructions, createTransactionInstruction(*r.CleanupInstruction))

	instructions = append(instructions, system.NewTransferInstruction(BribeAmount, a.wallet.PublicKey(), BribeAccount).Build())

	// 获取最近 blockhash
	recentBlockhashResp, err := a.rpcClient.GetLatestBlockhash(context.TODO(), rpc.CommitmentConfirmed)
	if err != nil {
		logx.Errorf("GetLatestBlockhash: %v", err)
		return
	}
	blockhash := recentBlockhashResp.Value.Blockhash

	// Create the transaction with all instructions
	tx, err := solana.NewTransaction(
		instructions,
		blockhash,
		solana.TransactionPayer(a.wallet.PublicKey()),
	)
	if err != nil {
		logx.Errorf("Error creating transaction: %v", err)
		return
	}

	// 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			return &a.wallet.PrivateKey
		},
	)
	if err != nil {
		logx.Errorf("Error signing transaction: %v", err)
		return
	}

	baseTx, err := tx.ToBase64()
	if err != nil {
		logx.Errorf("Error base64 transaction: %v", err)
		return
	}
	logx.Infof("tx base64: %s", baseTx)

	signaturs, err := SendTransactionWithRelay(a.sendClient, tx, true)
	if err != nil {
		logx.Errorf("Error sending transaction: %v", err)
		return
	}
	logx.Infof("tx hash: %s", signaturs)

}

func createTransactionInstruction(instructionData Instruction) solana.Instruction {
	programID := solana.MustPublicKeyFromBase58(instructionData.ProgramId)
	accounts := solana.AccountMetaSlice{}
	for _, acc := range instructionData.Accounts {
		pubkey := solana.MustPublicKeyFromBase58(acc.Pubkey)
		accounts = append(accounts, &solana.AccountMeta{
			PublicKey:  pubkey,
			IsSigner:   acc.IsSigner,
			IsWritable: acc.IsWritable,
		})
	}
	data, _ := base64.StdEncoding.DecodeString(instructionData.Data)

	return solana.NewInstruction(programID, accounts, data)
}
