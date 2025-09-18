package shot

import (
	"encoding/binary"
	"math/big"
	"solana-bot/internal/dex/meteora/common"
	"solana-bot/internal/dex/meteora/helpers"

	"github.com/gagliardetto/solana-go"
)

// Perform a swap on a DBC pool
func DbcSwap(
	config solana.PublicKey,
	pool solana.PublicKey,
	userInputTokenAccount solana.PublicKey,
	userOutputTokenAccount solana.PublicKey,
	baseVault solana.PublicKey,
	quoteVault solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	payer solana.PublicKey,
	tokenBaseProgram solana.PublicKey,
	referralTokenAccount solana.PublicKey,
	amountIn uint64,
	minOut uint64,
) solana.Instruction {
	swapDisc := []byte{248, 198, 158, 145, 225, 117, 135, 200}
	buf := make([]byte, 8+8+8)
	copy(buf, swapDisc)
	binary.LittleEndian.PutUint64(buf[8:], amountIn)
	binary.LittleEndian.PutUint64(buf[16:], minOut)

	poolAuthority := solana.MustPublicKeyFromBase58(common.PoolAuthority)
	tokenQuoteProgram := solana.MustPublicKeyFromBase58(common.TokenProgram)
	eventAuthority := helpers.DeriveEventAuthorityPDA()

	acctMetaSwap := solana.AccountMetaSlice{
		// 1. pool_authority
		{PublicKey: poolAuthority, IsSigner: false, IsWritable: false},
		// 2. config
		{PublicKey: config, IsSigner: false, IsWritable: false},
		// 3. pool
		{PublicKey: pool, IsSigner: false, IsWritable: true},
		// 4. input_token_account (user's token account for input token)
		{PublicKey: userInputTokenAccount, IsSigner: false, IsWritable: true},
		// 5. output_token_account (user's token account for output token)
		{PublicKey: userOutputTokenAccount, IsSigner: false, IsWritable: true},
		// 6. base_vault
		{PublicKey: baseVault, IsSigner: false, IsWritable: true},
		// 7. quote_vault
		{PublicKey: quoteVault, IsSigner: false, IsWritable: true},
		// 8. base_mint
		{PublicKey: baseMint, IsSigner: false, IsWritable: false},
		// 9. quote_mint
		{PublicKey: quoteMint, IsSigner: false, IsWritable: false},
		// 10. payer
		{PublicKey: payer, IsSigner: true, IsWritable: true},
		// 11. token_base_program
		{PublicKey: tokenBaseProgram, IsSigner: false, IsWritable: false},
		// 12. token_quote_program
		{PublicKey: tokenQuoteProgram, IsSigner: false, IsWritable: false},
		// 13. referral_token_account (optional; use a valid token account)
		{PublicKey: referralTokenAccount, IsSigner: false, IsWritable: true},
		// 14. event_authority
		{PublicKey: eventAuthority, IsSigner: false, IsWritable: false},
		// 15. program
		{PublicKey: solana.MustPublicKeyFromBase58(common.DbcProgramID), IsSigner: false, IsWritable: false},
	}

	return solana.NewInstruction(
		solana.MustPublicKeyFromBase58(common.DbcProgramID),
		acctMetaSwap,
		buf,
	)
}

func EstimateBackrunBuyOutAmount(
	sqrtPrice *big.Int, // next sqrt price from previous tx
	amountInQuote *big.Int, // 你要投入的 quote（如 SOL）数量
	decimalsBase int,
	decimalsQuote int,
	feeBps int, // e.g. 20 for 0.2%
) uint64 {
	// 解析 sqrtPrice
	sqrtP := new(big.Float).SetInt(sqrtPrice)

	// price = sqrtPrice^2 / Q96^2（我们简化使用精度为 1e18）
	precision := big.NewFloat(1e18)
	price := new(big.Float).Quo(
		new(big.Float).Mul(sqrtP, sqrtP),
		precision,
	)

	// 转入的 quote，乘精度
	amountIn := big.NewFloat(float64(amountInQuote.Uint64()))

	// 扣手续费
	feeRatio := new(big.Float).Quo(big.NewFloat(float64(feeBps)), big.NewFloat(10000))
	fee := new(big.Float).Mul(amountIn, feeRatio)
	amountInNet := new(big.Float).Sub(amountIn, fee)

	// 得到的 base token = amountInNet / price
	baseOut := new(big.Float).Quo(amountInNet, price)

	// 除以精度，转换成人类可读格式
	baseOutFloat := new(big.Float).Quo(baseOut,
		new(big.Float).SetFloat64(float64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalsBase)), nil).Int64())),
	)

	out, _ := baseOutFloat.Uint64()

	return out
}
