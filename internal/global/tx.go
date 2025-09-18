package global

import "github.com/gagliardetto/solana-go"

type TxBuilder struct {
	payer        solana.PublicKey
	blockhash    solana.Hash
	instructions []solana.Instruction
}

func NewTxBuilder(payer solana.PublicKey, blockhash solana.Hash) *TxBuilder {
	return &TxBuilder{
		payer:        payer,
		blockhash:    blockhash,
		instructions: make([]solana.Instruction, 0),
	}
}

func (b *TxBuilder) AddInstruction(instrs ...solana.Instruction) {
	b.instructions = append(b.instructions, instrs...)
}

func (b *TxBuilder) BuildTx(signers []solana.PrivateKey) (*solana.Transaction, error) {
	return solana.NewTransaction(
		b.instructions,
		b.blockhash,
		solana.TransactionPayer(b.payer),
	)
}
