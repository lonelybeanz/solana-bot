package token2022

import (
	"github.com/gagliardetto/solana-go"
)

func FindAssociatedTokenAddress2022(
	wallet solana.PublicKey,
	mint solana.PublicKey,
) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress([][]byte{
		wallet[:],
		solana.Token2022ProgramID[:],
		mint[:],
	},
		solana.SPLAssociatedTokenAccountProgramID,
	)
}
