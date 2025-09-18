package shot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
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
	baseAmount *big.Int,
	baseBalanceTokens *big.Float,
	quoteBalanceSol *big.Float,
	decimalsBase int,
	slippagePct float64,
) (base_amount_in uint64, min_quote_amount_out uint64) {
	if slippagePct == 0 {
		slippagePct = 0.01 // default slippage 1%
	}

	price := GetPrice(baseBalanceTokens, quoteBalanceSol)
	base_amount_in = baseAmount.Uint64()

	pf, _ := price.Float64()
	raw_sol := float64(baseAmount.Uint64()) / math.Pow10(decimalsBase) * pf
	min_sol_out := raw_sol * (1 - slippagePct)
	min_quote_amount_out = uint64(min_sol_out * LamportsPerSol)

	return
}

// 计算价格
func GetPrice(baseBalanceTokens *big.Float, quoteBalanceSol *big.Float) *big.Float {
	return new(big.Float).Quo(quoteBalanceSol, baseBalanceTokens)
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
