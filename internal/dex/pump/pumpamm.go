package pump

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"solana-bot/internal/global"
	"solana-bot/internal/global/utils"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	associated_token_account "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

var (
	PUMPSWAP_PROGRAM_ID = solana.MustPublicKeyFromBase58("pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA")
	TOKEN_PROGRAM_PUB   = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	ASSOCIATED_TOKEN    = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	SYSTEM_PROGRAM_ID   = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	EVENT_AUTHORITY     = solana.MustPublicKeyFromBase58("GS4CU59F31iL7aR2Q8zVS8DRrcRnXX1yjQ66TqNVQnaR")
)

const (
	LamportsPerSol = 1000000000 // 1 SOL = 10^9 lamports
)

type PumpAmmBuyInstruction struct {
	bin.BaseVariant
	MethodId                []byte
	AmountOut               uint64
	MaxAmountIn             uint64
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

func (inst *PumpAmmBuyInstruction) ProgramID() solana.PublicKey {
	return PUMPSWAP_PROGRAM_ID
}

func (inst *PumpAmmBuyInstruction) Accounts() (out []*solana.AccountMeta) {
	return inst.Impl.(solana.AccountsGettable).GetAccounts()
}

func (inst *PumpAmmBuyInstruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := bin.NewBorshEncoder(buf).Encode(inst); err != nil {
		return nil, fmt.Errorf("unable to encode instruction: %w", err)
	}
	return buf.Bytes(), nil
}

func (inst *PumpAmmBuyInstruction) MarshalWithEncoder(encoder *bin.Encoder) (err error) {
	// Swap instruction is number 9
	err = encoder.WriteBytes(inst.MethodId, false)
	if err != nil {
		return err
	}
	err = encoder.WriteUint64(inst.AmountOut, binary.LittleEndian)
	if err != nil {
		return err
	}
	err = encoder.WriteUint64(inst.MaxAmountIn, binary.LittleEndian)
	if err != nil {
		return err
	}
	return nil
}

func Async_get_pool_reserves(rpcClient rpc.Client, pool_base_token_account, pool_quote_token_account solana.PublicKey) (base_balance, quote_balance *uint64) {
	opts := &rpc.GetMultipleAccountsOpts{
		Commitment: rpc.CommitmentConfirmed,
	}
	accounts := []solana.PublicKey{
		pool_base_token_account,
		pool_quote_token_account,
	}
	accountInfos, err := rpcClient.GetMultipleAccountsWithOpts(context.Background(), accounts, opts)
	if err != nil {
		fmt.Errorf("getMultipleAccountsWithOpts err:%v ", err)
		return nil, nil
	}

	if accountInfos == nil {
		return nil, nil
	}

	if accountInfos.Value[0] == nil || (accountInfos.Value[0] != nil && len(accountInfos.Value[0].Data.GetBinary()) == 0) {
		return nil, nil
	}

	if accountInfos.Value[1] == nil || (accountInfos.Value[1] != nil && len(accountInfos.Value[1].Data.GetBinary()) == 0) {
		return nil, nil
	}

	baseToken, err := utils.TokenAccountFromData(accountInfos.Value[0].Data.GetBinary())
	if err != nil {
		return nil, nil
	}

	// baseBalanceValue := float64(baseToken.Amount) / 1e6
	// base_balance = new(float64)
	// *base_balance = baseBalanceValue

	quoteToken, err := utils.TokenAccountFromData(accountInfos.Value[1].Data.GetBinary())
	if err != nil {

		return nil, nil
	}

	// quoteBalanceValue := float64(quoteToken.Amount) / 1e9
	// quote_balance = new(float64)
	// *quote_balance = quoteBalanceValue

	return &baseToken.Amount, &quoteToken.Amount
}

// 计算base token数量以及最大可接受quote（lamports）数量
func convert_sol_to_base_tokens(
	solAmount float64,
	baseBalanceTokens *big.Float,
	quoteBalanceSol *big.Float,
	decimalsBase int,
	slippagePct float64,
) (baseAmountOut uint64, maxQuoteInLamports uint64) {
	// 计算池子的price = quote / base
	price := GetPrice(baseBalanceTokens, quoteBalanceSol)

	// 想买多少raw base token数量
	rawTokens := new(big.Float).Quo(big.NewFloat(solAmount), price)

	// 乘上decimals
	baseAmountOut, _ = new(big.Float).Mul(rawTokens, big.NewFloat(math.Pow10(decimalsBase))).Uint64()

	// 考虑滑点
	maxSol := solAmount * (1 + slippagePct)

	maxQuoteInLamports = uint64(maxSol * LamportsPerSol)

	return
}

func convert_base_tokens_to_sol(
	tokenAmountUser *big.Int,
	baseBalanceTokens *big.Float,
	quoteBalanceSol *big.Float,
	decimalsBase int,
	slippagePct float64,
) (base_amount_in uint64, min_quote_amount_out uint64) {
	if slippagePct == 0 {
		slippagePct = 0.01 // default slippage 1%
	}

	price := GetPrice(baseBalanceTokens, quoteBalanceSol)
	base_amount_in = tokenAmountUser.Uint64()

	pf, _ := price.Float64()
	raw_sol := float64(tokenAmountUser.Uint64()) / math.Pow10(decimalsBase) * pf
	min_sol_out := raw_sol * (1 - slippagePct)
	min_quote_amount_out = uint64(min_sol_out * LamportsPerSol)

	return
}

// 计算价格
func GetPrice(baseBalanceTokens *big.Float, quoteBalanceSol *big.Float) *big.Float {
	return new(big.Float).Quo(quoteBalanceSol, baseBalanceTokens)
}

func GetPumpAMMBuyTx(
	signerAndOwner *solana.PrivateKey,
	pool solana.PublicKey,
	globalConfig solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	poolBaseTokenAccount solana.PublicKey,
	poolQuoteTokenAccount solana.PublicKey,
	protocolFeeRecipient solana.PublicKey,
	protocolFeeRecipientATA solana.PublicKey,
	coinCreatorVaultAta solana.PublicKey,
	coinCreatorVaultAuthority solana.PublicKey,
	base_balance_tokens *big.Float,
	quote_balance_sol *big.Float,
	maxAmountIn *big.Int,
	slippage float64,
	priorityFee uint64,
	fee uint64,
	jitoTip uint64,
) (*solana.Transaction, error) {
	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{*signerAndOwner}

	nonceAccount, nonceHash := global.GetNonceAccountAndHash()

	instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())

	amountInAfterOurFee := new(big.Int).Sub(maxAmountIn, big.NewInt(int64(fee)))

	sol_amount := float64(amountInAfterOurFee.Uint64()) / 1e9
	base_amount_out, max_quote_amount_in := convert_sol_to_base_tokens(sol_amount, base_balance_tokens, quote_balance_sol, 6, slippage/100)

	// 1. 设置 compute unit limit（ComputeBudgetProgram）
	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(120_000).Build())

	// 2. 设置 compute unit price（ComputeBudgetProgram）
	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	if fee > 0 {
		instrs = append(instrs, system.NewTransferInstruction(fee, signerAndOwner.PublicKey(), global.FeeAccountBuys).Build())
	}

	if jitoTip > 0 {
		instrs = append(instrs, system.NewTransferInstruction(jitoTip, signerAndOwner.PublicKey(), global.PickRandomTip()).Build())

	}

	// 3. 检查并创建 WSOL ATA（如果不存在）
	global.CreateSOLAccountOrWrap(&instrs, signerAndOwner.PublicKey(), big.NewInt(int64(max_quote_amount_in)))

	instrs = append(instrs, associated_token_account.NewCreateInstruction(signerAndOwner.PublicKey(), signerAndOwner.PublicKey(), baseMint).Build())

	addPumpAmmBuyIx(&instrs, signerAndOwner.PublicKey(), base_amount_out, max_quote_amount_in, base_balance_tokens, quote_balance_sol, pool, globalConfig, baseMint, quoteMint, poolBaseTokenAccount, poolQuoteTokenAccount, protocolFeeRecipient, protocolFeeRecipientATA, coinCreatorVaultAta, coinCreatorVaultAuthority)

	tx, err := BuildTransaction(nonceHash, signers, *signerAndOwner, instrs...)
	return tx, err
}

/*
#1 Pool
#2 User
#3 Global Config
#4 Base Mint
#5 Quote Mint
#6 User Base ATA
#7 User Quote ATA
#8 Pool Base ATA
#9 Pool Quote ATA
#10 Protocol Fee Recipient
#11 Protocol Fee Recipient Token Account
#12 Base Token Program
#13 Quote Token Program
#14 System Program
#15 Associated Token Program
#16 Event Authority
#17 PumpSwap Program
*/
func addPumpAmmBuyIx(
	instrs *[]solana.Instruction,
	owner solana.PublicKey,
	base_amount_out uint64,
	max_quote_amount_in uint64,
	base_balance_tokens *big.Float,
	quote_balance_sol *big.Float,
	pool solana.PublicKey,
	globalConfig solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	poolBaseTokenAccount solana.PublicKey,
	poolQuoteTokenAccount solana.PublicKey,
	protocolFeeRecipient solana.PublicKey,
	protocolFeeRecipientATA solana.PublicKey,
	coinCreatorVaultAta solana.PublicKey,
	coinCreatorVaultAuthority solana.PublicKey,
) {

	userBaseAta, _, _ := solana.FindAssociatedTokenAddress(owner, baseMint)
	userQuoteAta, _, _ := solana.FindAssociatedTokenAddress(owner, quoteMint)

	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(pool, true, false),                    // 写权限
		solana.NewAccountMeta(owner, true, true),                    // 写权限 + 签名者
		solana.NewAccountMeta(globalConfig, false, false),           // 只读
		solana.NewAccountMeta(baseMint, false, false),               // 只读
		solana.NewAccountMeta(quoteMint, false, false),              // 只读
		solana.NewAccountMeta(userBaseAta, true, false),             // 写权限
		solana.NewAccountMeta(userQuoteAta, true, false),            // 写权限
		solana.NewAccountMeta(poolBaseTokenAccount, true, false),    // 写权限
		solana.NewAccountMeta(poolQuoteTokenAccount, true, false),   // 写权限
		solana.NewAccountMeta(protocolFeeRecipient, false, false),   // 只读
		solana.NewAccountMeta(protocolFeeRecipientATA, true, false), // 写权限
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读（重复传两次）
		solana.NewAccountMeta(SYSTEM_PROGRAM_ID, false, false),      // 只读
		solana.NewAccountMeta(ASSOCIATED_TOKEN, false, false),       // 只读
		solana.NewAccountMeta(EVENT_AUTHORITY, false, false),        // 只读
		solana.NewAccountMeta(PUMPSWAP_PROGRAM_ID, false, false),    // 只读
		solana.NewAccountMeta(coinCreatorVaultAta, true, false),
		solana.NewAccountMeta(coinCreatorVaultAuthority, false, false),
	}
	data := BuildBuyInstructionData(base_amount_out, max_quote_amount_in)
	*instrs = append(*instrs, solana.NewInstruction(PUMPSWAP_PROGRAM_ID, accounts, data))

}

func GetPumpAMMSellTx(
	signerAndOwner *solana.PrivateKey,
	pool solana.PublicKey,
	globalConfig solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	poolBaseTokenAccount solana.PublicKey,
	poolQuoteTokenAccount solana.PublicKey,
	protocolFeeRecipient solana.PublicKey,
	protocolFeeRecipientATA solana.PublicKey,
	coinCreatorVaultAta solana.PublicKey,
	coinCreatorVaultAuthority solana.PublicKey,
	base_balance_tokens *big.Float,
	quote_balance_sol *big.Float,
	maxAmountIn *big.Int,
	slippage float64,
	priorityFee uint64,
	fee uint64,
	jitoTip uint64,
	shouldCloseTokenInAccount bool,
) (*solana.Transaction, error) {
	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{*signerAndOwner}

	// nonceAccount, nonceHash := global.GetNonceAccountAndHash()

	// instrs = append(instrs, system.NewAdvanceNonceAccountInstruction(nonceAccount, solana.SysVarRecentBlockHashesPubkey, signerAndOwner.PublicKey()).Build())

	base_amount_in, min_quote_amount_out := convert_base_tokens_to_sol(maxAmountIn, base_balance_tokens, quote_balance_sol, 6, slippage/100.0)

	// 1. 设置 compute unit limit（ComputeBudgetProgram）
	instrs = append(instrs, computebudget.NewSetComputeUnitLimitInstruction(120_000).Build())

	// 2. 设置 compute unit price（ComputeBudgetProgram）
	if priorityFee > 0 {
		instrs = append(instrs, computebudget.NewSetComputeUnitPriceInstruction(priorityFee).Build())
	}

	if fee > 0 {
		instrs = append(instrs, system.NewTransferInstruction(fee, signerAndOwner.PublicKey(), global.FeeAccountBuys).Build())
	}

	if jitoTip > 0 {
		instrs = append(instrs, system.NewTransferInstruction(jitoTip, signerAndOwner.PublicKey(), global.PickRandomTip()).Build())

	}

	addPumpAmmSellIx(&instrs, signerAndOwner.PublicKey(), base_amount_in, min_quote_amount_out, base_balance_tokens, quote_balance_sol, pool, globalConfig, baseMint, quoteMint, poolBaseTokenAccount, poolQuoteTokenAccount, protocolFeeRecipient, protocolFeeRecipientATA, coinCreatorVaultAta, coinCreatorVaultAuthority)

	if shouldCloseTokenInAccount {
		closeATA(&instrs, signerAndOwner.PublicKey(), baseMint)
	}
	tx, err := BuildTransaction(global.GetBlockHash(), signers, *signerAndOwner, instrs...)
	return tx, err
}

func addPumpAmmSellIx(
	instrs *[]solana.Instruction,
	owner solana.PublicKey,
	base_amount_in uint64,
	min_quote_amount_out uint64,
	base_balance_tokens *big.Float,
	quote_balance_sol *big.Float,
	pool solana.PublicKey,
	globalConfig solana.PublicKey,
	baseMint solana.PublicKey,
	quoteMint solana.PublicKey,
	poolBaseTokenAccount solana.PublicKey,
	poolQuoteTokenAccount solana.PublicKey,
	protocolFeeRecipient solana.PublicKey,
	protocolFeeRecipientATA solana.PublicKey,
	coinCreatorVaultAta solana.PublicKey,
	coinCreatorVaultAuthority solana.PublicKey,
) {
	userBaseAta, _, _ := solana.FindAssociatedTokenAddress(owner, baseMint)
	userQuoteAta, _, _ := solana.FindAssociatedTokenAddress(owner, quoteMint)

	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(pool, true, false),                    // 写权限
		solana.NewAccountMeta(owner, true, true),                    // 写权限 + 签名者
		solana.NewAccountMeta(globalConfig, false, false),           // 只读
		solana.NewAccountMeta(baseMint, false, false),               // 只读
		solana.NewAccountMeta(quoteMint, false, false),              // 只读
		solana.NewAccountMeta(userBaseAta, true, false),             // 写权限
		solana.NewAccountMeta(userQuoteAta, true, false),            // 写权限
		solana.NewAccountMeta(poolBaseTokenAccount, true, false),    // 写权限
		solana.NewAccountMeta(poolQuoteTokenAccount, true, false),   // 写权限
		solana.NewAccountMeta(protocolFeeRecipient, false, false),   // 只读
		solana.NewAccountMeta(protocolFeeRecipientATA, true, false), // 写权限
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读
		solana.NewAccountMeta(TOKEN_PROGRAM_PUB, false, false),      // 只读（重复传两次）
		solana.NewAccountMeta(SYSTEM_PROGRAM_ID, false, false),      // 只读
		solana.NewAccountMeta(ASSOCIATED_TOKEN, false, false),       // 只读
		solana.NewAccountMeta(EVENT_AUTHORITY, false, false),        // 只读
		solana.NewAccountMeta(PUMPSWAP_PROGRAM_ID, false, false),    // 只读
		solana.NewAccountMeta(coinCreatorVaultAta, true, false),
		solana.NewAccountMeta(coinCreatorVaultAuthority, false, false),
	}
	data := BuildSellInstructionData(base_amount_in, min_quote_amount_out)
	*instrs = append(*instrs, solana.NewInstruction(PUMPSWAP_PROGRAM_ID, accounts, data))
}

// 假设：指令 discriminator 是 8 字节
var BUY_INSTR_DISCRIM = []byte{102, 6, 61, 18, 1, 218, 235, 234}

var SELL_INSTR_DISCRIM = []byte{51, 230, 133, 164, 1, 127, 131, 173}

// 构造数据
func BuildBuyInstructionData(amountOut uint64, maxAmountIn uint64) []byte {
	data := make([]byte, 8+8+8) // 8 bytes for discriminator, 8 + 8 for two uint64
	copy(data[0:8], BUY_INSTR_DISCRIM)
	binary.LittleEndian.PutUint64(data[8:16], amountOut)
	binary.LittleEndian.PutUint64(data[16:24], maxAmountIn)
	return data
}

func BuildSellInstructionData(amountOut uint64, maxAmountIn uint64) []byte {
	data := make([]byte, 8+8+8) // 8 bytes for discriminator, 8 + 8 for two uint64
	copy(data[0:8], SELL_INSTR_DISCRIM)
	binary.LittleEndian.PutUint64(data[8:16], amountOut)
	binary.LittleEndian.PutUint64(data[16:24], maxAmountIn)
	return data
}
