package rpcs

import (
	"context"
	"log"
	"math/rand"
	"solana-bot/internal/global"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

var (
	slotName   = "0slot"
	slotUrl    = "https://de.0slot.trade?api-key="
	slotTipKey = []string{
		"4HiwLEP2Bzqj3hM2ENxJuzhcPCdsafwiet3oGkMkuQY4",
		"6fQaVhYZA4w3MBSXjJ81Vf6W1EDYeUPXpgVQ6UQyU1Av",
		"7toBU3inhmrARGngC7z6SjyP85HgGMmCTEwGNRAcYnEK",
		"8mR3wB1nh4D6J9RUCugxUpc6ya8w38LPxZ3ZjcBhgzws",
		"6SiVU5WEwqfFapRuYCndomztEwDjvS5xgtEof3PLEGm9",
		"TpdxgNJBWZRL8UXF5mrEsyWxDWx9HQexA9P1eTWQ42p",
		"D8f3WkQu6dCF33cZxuAsrKHrGsqGP2yvAHf8mX6RXnwf",
		"GQPFicsy3P3NXxB5piJohoxACqTvWE9fKpLgdsMduoHE",
		"Ey2JEr8hDkgN8qKJGrLf2yFjRhW7rab99HVxwi5rcvJE",
		"4iUgjMT8q2hNZnLuhpqZ1QtiV8deFPy2ajvvjEpKKgsS",
		"3Rz8uD83QsU8wKvZbgWAPvCNDU6Fy8TSZTMcPm3RB6zt",
	}
)

type SlotChannel struct {
	rpcClient *rpc.Client
}

func NewSlotChannel() *SlotChannel {
	rpcClient := rpc.New(slotUrl)
	return &SlotChannel{
		rpcClient: rpcClient,
	}
}

func (c *SlotChannel) GetTipInstruction(owner solana.PublicKey, tip uint64) solana.Instruction {
	randomIndex := rand.Intn(len(slotTipKey))
	tipPublicKey := solana.MustPublicKeyFromBase58(slotTipKey[randomIndex])
	return system.NewTransferInstruction(tip, owner, tipPublicKey).Build()
}

func (c *SlotChannel) SendTransaction(wallet solana.PrivateKey, tip uint64, txBuilder global.TxBuilder) (string, error) {

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

	sig, err := c.rpcClient.SendTransactionWithOpts(
		context.TODO(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       true,
			PreflightCommitment: rpc.CommitmentConfirmed,
		},
	)
	if err != nil {
		return "", err
	}

	log.Printf("✅ [%s] tx sent: %s", slotName, sig)

	return sig.String(), nil
}
