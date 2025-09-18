package shot

import (
	"math/big"

	"github.com/gagliardetto/solana-go"
)

type ShotAdapter interface {
	Name() string
	BuildInstructions(txInfo *TxContext, accounts ...solana.PublicKey) ([]solana.Instruction, error)
}

type TxContext struct {
	SignerAndOwner       solana.PrivateKey
	SrcMint              solana.PublicKey
	DstMint              solana.PublicKey
	VirtualSolReserves   *big.Int
	VirtualTokenReserves *big.Int
	SqrtPrice            *big.Int
	MaxAmountIn          uint64
	Slippage             float32
	PriorityFee          uint64
	Fee                  uint64
}
