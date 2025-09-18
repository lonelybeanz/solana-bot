package coder

import "github.com/gagliardetto/solana-go"

// TODO: A better struct, maybe from anchor generated code
type TokenTransferChecked struct {
	Info struct {
		Authority   string `json:"authority"`
		Destination string `json:"destination"`
		Mint        string `json:"mint"`
		Source      string `json:"source"`
		TokenAmount struct {
			Amount         string  `json:"amount"`
			Decimals       int     `json:"decimals"`
			UiAmount       float64 `json:"uiAmount"`
			UiAmountString string  `json:"uiAmountString"`
		} `json:"tokenAmount"`
	} `json:"info"`
	InstructionType string `json:"type"`
}

type SystemTransfer struct {
	// programId:11111111111111111111111111111111
	Info struct {
		Destination string `json:"destination"`
		Lamports    int    `json:"lamports"`
		Source      string `json:"source"`
	} `json:"info"`
	Type string `json:"type"`
}
type TokenTransfer struct {
	Info struct {
		Amount      string `json:"amount"`
		Authority   string `json:"authority"`
		Destination string `json:"destination"`
		Source      string `json:"source"`
	} `json:"info"`
	InstructionType string `json:"type"`
}

type PumpFunAnchorSelfCPILogData struct {
	Unused1              [16]byte         `compare:"-"`
	Mint                 solana.PublicKey `compare:"true"`
	SolAmount            uint64           `compare:"true"`
	TokenAmount          uint64           `compare:"true"`
	IsBuy                bool             `compare:"true"`
	User                 solana.PublicKey `compare:"true"`
	Timestamp            int64            `compare:"true"`
	VirtualSolReserves   uint64           `compare:"true"`
	VirtualTokenReserves uint64           `compare:"true"`
}

type Initialize2 struct {
	Nonce          byte
	OpenTime       uint64
	InitPcAmount   uint64
	InitCoinAmount uint64
}

type Withdraw struct {
	Amount uint64
}

type SwapBaseIn struct {
	AmountIn         uint64
	MinimumAmountOut uint64
}

type SwapBaseOut struct {
	MaxAmountIn uint64
	AmountOut   uint64
}

type Compute struct {
	Instruction uint8
	Value       uint32
}

type Transfer struct {
	Instruction uint32
	Amount      int64
}

type Initialize struct {
	Nonce    byte
	OpenTime uint64
}

type MonitorStep struct {
	PlanOrderLimit   uint16
	PlaceOrderLimit  uint16
	CancelOrderLimit uint16
}

type Deposit struct {
	MaxCoinAmount  uint64
	MaxPcAmount    uint64
	BaseSide       uint64
	OtherAmountMin *uint64
}

type SetParams struct {
	Param             uint8
	Value             *uint64
	NewPubkey         []byte
	Fees              *Fees
	LastOrderDistance *LastOrderDistance
}

type Fees struct {
	// Add fee structure fields
}

type LastOrderDistance struct {
	LastOrderNumerator   uint64
	LastOrderDenominator uint64
}

type PreInitialize struct {
	Nonce byte
}

type SimulateInfo struct {
	Param            uint8
	SwapBaseInValue  *SwapBaseIn
	SwapBaseOutValue *SwapBaseOut
}

type AdminCancelOrders struct {
	Limit uint16
}

type ConfigArgs struct {
	Param         uint8
	Owner         []byte
	CreatePoolFee *uint64
}
