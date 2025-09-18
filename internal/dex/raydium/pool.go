package raydium

import (
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

// Perform a swap on a DBC pool
func Swap(
	buy bool,
	globalConfig solana.PublicKey,
	platformConfig solana.PublicKey,
	poolState solana.PublicKey,
	userBaseTokenAccount solana.PublicKey,
	userQuoteTokenAccount solana.PublicKey,
	baseVault solana.PublicKey,
	quoteVault solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	payer solana.PublicKey,
	amountIn uint64,
	minOut uint64,
) solana.Instruction {
	swapDisc := []byte{250, 234, 13, 123, 213, 156, 19, 236}
	if !buy {
		swapDisc = []byte{0x95, 0x27, 0xde, 0x9b, 0xd3, 0x7c, 0x98, 0x1a}
	}

	buf := make([]byte, 8+8+8+8)
	copy(buf, swapDisc)
	binary.LittleEndian.PutUint64(buf[8:], amountIn)
	binary.LittleEndian.PutUint64(buf[16:], minOut*88/100)
	binary.LittleEndian.PutUint64(buf[24:], uint64(0))

	poolAuthority := solana.MustPublicKeyFromBase58(RaydiumLaunchpadAuthority)
	tokenBaseProgram := solana.MustPublicKeyFromBase58(TokenProgram)
	tokenQuoteProgram := solana.MustPublicKeyFromBase58(TokenProgram)
	eventAuthority := solana.MustPublicKeyFromBase58(EventAuthority)

	acctMetaSwap := solana.AccountMetaSlice{
		// 1. payer
		{PublicKey: payer, IsSigner: true, IsWritable: true},
		// 2. Authority
		{PublicKey: poolAuthority, IsSigner: false, IsWritable: false},
		// 3. Global Config
		{PublicKey: globalConfig, IsSigner: false, IsWritable: true},
		// 4. Platform Config
		{PublicKey: platformConfig, IsSigner: false, IsWritable: false},
		// 5. Pool State
		{PublicKey: poolState, IsSigner: false, IsWritable: true},
		// 6. User Base Token
		{PublicKey: userBaseTokenAccount, IsSigner: false, IsWritable: true},
		// 7. User Quote Token
		{PublicKey: userQuoteTokenAccount, IsSigner: false, IsWritable: true},
		// 8. base_vault
		{PublicKey: baseVault, IsSigner: false, IsWritable: true},
		// 9. quote_vault
		{PublicKey: quoteVault, IsSigner: false, IsWritable: true},
		// 10. base_mint
		{PublicKey: baseMint, IsSigner: false, IsWritable: false},
		// 11. quote_mint
		{PublicKey: quoteMint, IsSigner: false, IsWritable: false},
		// 12. token_base_program
		{PublicKey: tokenBaseProgram, IsSigner: false, IsWritable: false},
		// 13. token_quote_program
		{PublicKey: tokenQuoteProgram, IsSigner: false, IsWritable: false},
		// 14. event_authority
		{PublicKey: eventAuthority, IsSigner: false, IsWritable: false},
		// 15. program
		{PublicKey: solana.MustPublicKeyFromBase58(RaydiumLaunchpadProgramID), IsSigner: false, IsWritable: false},
	}

	return solana.NewInstruction(
		solana.MustPublicKeyFromBase58(RaydiumLaunchpadProgramID),
		acctMetaSwap,
		buf,
	)
}
