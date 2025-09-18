package shot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"solana-bot/internal/global"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

var (
	PUMPDex                      = "pump fun (bonding curve)"
	PUMPManager                  = solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P")
	EventAuthority               = solana.MustPublicKeyFromBase58("Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1")
	PUMPQuoteSellAmountIn        = big.NewInt(1000000)
	PUMPBuyMethod                = []byte{0x66, 0x06, 0x3d, 0x12, 0x01, 0xda, 0xeb, 0xea}
	PUMPSellMethod               = []byte{0x33, 0xe6, 0x85, 0xa4, 0x01, 0x7f, 0x83, 0xad}
	SlippageAdjustment    int64  = 2
	DefaultSlippage              = float32(3.0)
	FeeBasisPoints        uint64 = 100
)

type PUMPBondingCurveData struct {
	BondingCurve             *BondingCurveLayout
	BondingCurvePk           solana.PublicKey
	AssociatedBondingCurvePk solana.PublicKey
	GlobalSettings           *GlobalSettingsLayout
	GlobalSettingsPk         solana.PublicKey
	MintAuthority            solana.PublicKey
}

type BondingCurveLayout struct {
	Blob1                uint64
	VirtualTokenReserves uint64
	VirtualSOLReserves   uint64
	RealTokenReserves    uint64
	RealSOLReserves      uint64
	TokenTotalSupply     uint64
	Complete             bool
}

type GlobalSettingsLayout struct {
	Blob1                       [8]byte
	Initialized                 bool
	Authority                   solana.PublicKey
	FeeRecipient                solana.PublicKey
	InitialVirtualTokenReserves uint64
	InitialVirtualSOLReserves   uint64
	InitialRealTokenReserves    uint64
	TokenTotalSupply            uint64
	FeeBasisPoints              uint64
}

type PumpBuyInstruction struct {
	bin.BaseVariant
	MethodId                []byte
	AmountOut               uint64
	MaxAmountIn             uint64
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

func (inst *PumpBuyInstruction) ProgramID() solana.PublicKey {
	return PUMPManager
}

func (inst *PumpBuyInstruction) Accounts() (out []*solana.AccountMeta) {
	return inst.Impl.(solana.AccountsGettable).GetAccounts()
}

func (inst *PumpBuyInstruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := bin.NewBorshEncoder(buf).Encode(inst); err != nil {
		return nil, fmt.Errorf("unable to encode instruction: %w", err)
	}
	return buf.Bytes(), nil
}

func (inst *PumpBuyInstruction) MarshalWithEncoder(encoder *bin.Encoder) (err error) {
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

type PumpSellInstruction struct {
	bin.BaseVariant
	MethodId                []byte
	AmountIn                uint64
	AmountOutMin            uint64
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

func (inst *PumpSellInstruction) ProgramID() solana.PublicKey {
	return PUMPManager
}

func (inst *PumpSellInstruction) Accounts() (out []*solana.AccountMeta) {
	return inst.Impl.(solana.AccountsGettable).GetAccounts()
}

func (inst *PumpSellInstruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := bin.NewBorshEncoder(buf).Encode(inst); err != nil {
		return nil, fmt.Errorf("unable to encode instruction: %w", err)
	}
	return buf.Bytes(), nil
}

func (inst *PumpSellInstruction) MarshalWithEncoder(encoder *bin.Encoder) (err error) {
	// Swap instruction is number 9
	err = encoder.WriteBytes(inst.MethodId, false)
	if err != nil {
		return err
	}
	err = encoder.WriteUint64(inst.AmountIn, binary.LittleEndian)
	if err != nil {
		return err
	}
	err = encoder.WriteUint64(inst.AmountOutMin, binary.LittleEndian)
	if err != nil {
		return err
	}
	return nil
}

func pumpGetFee(amount *big.Int, feeBP uint64) *big.Int {
	temp := new(big.Int).Mul(amount, big.NewInt(int64(feeBP)))
	feeAmount := new(big.Int).Div(temp, global.Big10000)
	return feeAmount
}
