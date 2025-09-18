package accounts

import (
	"reflect"
	"strconv"

	"github.com/gagliardetto/solana-go"
)

// RaydiumSwapBaseAccounts are the accounts for a swap base in or swap base out
type RaydiumSwapBaseAccounts struct {
	Amm                         solana.PublicKey `idx:"1"`
	AmmAuthority                solana.PublicKey `idx:"2"`
	AmmOpenOrders               solana.PublicKey `idx:"3"`
	AmmTargetOrders             solana.PublicKey `idx:"4"`
	PoolCoinTokenAccount        solana.PublicKey `idx:"5"`
	PoolPcTokenAccount          solana.PublicKey `idx:"6"`
	SerumMarket                 solana.PublicKey `idx:"8"`
	SerumBids                   solana.PublicKey `idx:"9"`
	SerumAsks                   solana.PublicKey `idx:"10"`
	SerumEventQueue             solana.PublicKey `idx:"11"`
	SerumCoinVaultAccount       solana.PublicKey `idx:"12"`
	SerumPcVaultAccount         solana.PublicKey `idx:"13"`
	SerumVaultSigner            solana.PublicKey `idx:"14"`
	UserSourceTokenAccount      solana.PublicKey `idx:"15"`
	UserDestinationTokenAccount solana.PublicKey `idx:"16"`
	UserSourceOwner             solana.PublicKey `idx:"17"`
}

type RaydiumCpmmSwapAccounts struct {
	Payer              solana.PublicKey `idx:"0"`  // Payer
	Authority          solana.PublicKey `idx:"1"`  // Authority
	AmmConfig          solana.PublicKey `idx:"2"`  // Amm Config
	PoolState          solana.PublicKey `idx:"3"`  // Pool State
	InputTokenAccount  solana.PublicKey `idx:"4"`  // Input Token Account
	OutputTokenAccount solana.PublicKey `idx:"5"`  // Output Token Account
	InputVault         solana.PublicKey `idx:"6"`  // Input Vault
	OutputVault        solana.PublicKey `idx:"7"`  // Output Vault
	InputTokenProgram  solana.PublicKey `idx:"8"`  // Input Token Program
	OutputTokenProgram solana.PublicKey `idx:"9"`  // Output Token Program
	InputTokenMint     solana.PublicKey `idx:"10"` // Input Token Mint
	OutputTokenMint    solana.PublicKey `idx:"11"` // Output Token Mint
	ObservationState   solana.PublicKey `idx:"12"` // Observation State
}

type PumpFunSwapAccounts struct {
	Global            solana.PublicKey `idx:"0"`  // 4wTV1YmiEkRvAtNtsSGPtUrqRYQMe5SKy2uB4Jjaxnjf， 内盘池地址
	FeeRecipient      solana.PublicKey `idx:"1"`  // Fee Recipient
	Mint              solana.PublicKey `idx:"2"`  // Target Token Mint Address
	BondingCurve      solana.PublicKey `idx:"3"`  // Pump.fun  Bonding Curve
	AssociatedBonding solana.PublicKey `idx:"4"`  // Pump.fun  Vault
	AssociatedUser    solana.PublicKey `idx:"5"`  //
	User              solana.PublicKey `idx:"6"`  //
	SystemProgram     solana.PublicKey `idx:"7"`  // System Program
	TokenProgram      solana.PublicKey `idx:"8"`  // Token Program
	Rent              solana.PublicKey `idx:"9"`  // Rent Program
	EventAuthority    solana.PublicKey `idx:"10"` //
	Program           solana.PublicKey `idx:"11"` // Pump.fun Program
}

// https://github.com/orca-so/whirlpools/blob/main/programs/whirlpool/src/instructions/swap.rs
type OrcaSwapWhirlPoolAccounts struct {
	TokenProgram   solana.PublicKey `idx:"0"`  // Token Program
	TokenAuthority solana.PublicKey `idx:"1"`  // Token Authority (Writable, Signer, Fee Payer)
	Whirlpool      solana.PublicKey `idx:"2"`  // Orca Market (Writable)
	TokenOwnerA    solana.PublicKey `idx:"3"`  // Token Owner Account A (Writable)
	TokenVaultA    solana.PublicKey `idx:"4"`  // Token Vault A (Writable)
	TokenOwnerB    solana.PublicKey `idx:"5"`  // Token Owner Account B (Writable)
	TokenVaultB    solana.PublicKey `idx:"6"`  // Token Vault B (Writable)
	TickArray0     solana.PublicKey `idx:"7"`  // Tick Array 0 (Writable)
	TickArray1     solana.PublicKey `idx:"8"`  // Tick Array 1 (Writable)
	TickArray2     solana.PublicKey `idx:"9"`  // Tick Array 2 (Writable)
	Oracle         solana.PublicKey `idx:"10"` // Oracle
}

type OrcaSwapV2WhirlPoolAccounts struct {
	TokenProgramA  solana.PublicKey `idx:"0"`  // Token Program
	TokenProgramB  solana.PublicKey `idx:"1"`  // Token Program
	MemoProgram    solana.PublicKey `idx:"2"`  // Memo Program
	TokenAuthority solana.PublicKey `idx:"3"`  // Token Authority (Writable, Signer, Fee Payer)
	Whirlpool      solana.PublicKey `idx:"4"`  // Orca Market (Writable)
	TokenMintA     solana.PublicKey `idx:"5"`  // Token Mint A
	TokenMintB     solana.PublicKey `idx:"6"`  // Token Mint B
	TokenOwnerA    solana.PublicKey `idx:"7"`  // Token Owner Account A (Writable)
	TokenVaultA    solana.PublicKey `idx:"8"`  // Token Vault A (Writable)
	TokenOwnerB    solana.PublicKey `idx:"9"`  // Token Owner Account B (Writable)
	TokenVaultB    solana.PublicKey `idx:"10"` // Token Vault B (Writable)
	TickArray0     solana.PublicKey `idx:"11"` // Tick Array 0 (Writable)
	TickArray1     solana.PublicKey `idx:"12"` // Tick Array 1 (Writable)
	TickArray2     solana.PublicKey `idx:"13"` // Tick Array 2 (Writable)
	Oracle         solana.PublicKey `idx:"14"` // Oracle
}

type OrcaSwapV2Accounts struct {
	TokenSwap             solana.PublicKey `idx:"0"` // Orca Market
	Authority             solana.PublicKey `idx:"1"` // Authority
	UserTransferAuthority solana.PublicKey `idx:"2"` // User Transfer Authority (Writable, Signer, Fee Payer)
	UserSource            solana.PublicKey `idx:"3"` // User Source Token Account (Writable)
	PoolSource            solana.PublicKey `idx:"4"` // Pool Source Token Account (Writable)
	PoolDestination       solana.PublicKey `idx:"5"` // Pool Destination Token Account (Writable)
	UserDestination       solana.PublicKey `idx:"6"` // User Destination Token Account (Writable)
	PoolMint              solana.PublicKey `idx:"7"` // Pool LP Token Mint (Writable)
	FeeAccount            solana.PublicKey `idx:"8"` // Fee Account (Writable)
	TokenProgram          solana.PublicKey `idx:"9"` // Token Program
}

type OrcaSwapAccounts struct {
	TokenSwap             solana.PublicKey `idx:"0"` // Orca Market
	Authority             solana.PublicKey `idx:"1"` // Authority
	UserTransferAuthority solana.PublicKey `idx:"2"` // User Transfer Authority (Writable, Signer, Fee Payer)
	UserSource            solana.PublicKey `idx:"3"` // User Source Token Account (Writable)
	PoolSource            solana.PublicKey `idx:"4"` // Pool Source Token Account (Writable)
	PoolDestination       solana.PublicKey `idx:"5"` // Pool Destination Token Account (Writable)
	UserDestination       solana.PublicKey `idx:"6"` // User Destination Token Account (Writable)
	PoolMint              solana.PublicKey `idx:"7"` // Pool LP Token Mint (Writable)
	FeeAccount            solana.PublicKey `idx:"8"` // Fee Account (Writable)
	TokenProgram          solana.PublicKey `idx:"9"` // Token Program
}
type MeteoraDAMMV2SwapAccounts struct {
	PollAuthority        solana.PublicKey `idx:"0"`
	Pool                 solana.PublicKey `idx:"1"`
	InputTokenAccount    solana.PublicKey `idx:"2"`
	OutputTokenAccount   solana.PublicKey `idx:"3"`
	TokenAVault          solana.PublicKey `idx:"4"`
	TokenBVault          solana.PublicKey `idx:"5"`
	TokenAMint           solana.PublicKey `idx:"6"`
	TokenBMint           solana.PublicKey `idx:"7"`
	Payer                solana.PublicKey `idx:"8"`
	TokenAProgram        solana.PublicKey `idx:"9"`
	TokenBProgram        solana.PublicKey `idx:"10"`
	ReferralTokenAccount solana.PublicKey `idx:"11"`
	EventAuthority       solana.PublicKey `idx:"12"`
	Program              solana.PublicKey `idx:"13"`
}
type MeteoraDLMMSwapAccounts struct {
	LbPair                  solana.PublicKey `idx:"0"`  // Meteora Market
	BinArrayBitmapExtension solana.PublicKey `idx:"1"`  // Meteora DLMM Program
	ReserveX                solana.PublicKey `idx:"2"`  // Meteora Pool 1
	ReserveY                solana.PublicKey `idx:"3"`  // Meteora Pool 2
	UserTokenIn             solana.PublicKey `idx:"4"`  // User Token Input Account
	UserTokenOut            solana.PublicKey `idx:"5"`  // User Token Output Account
	TokenXMint              solana.PublicKey `idx:"6"`  // Token X Mint
	TokenYMint              solana.PublicKey `idx:"7"`  // Token Y Mint
	Oracle                  solana.PublicKey `idx:"8"`  // Oracle Account
	HostFeeIn               solana.PublicKey `idx:"9"`  // Host Fee Account
	User                    solana.PublicKey `idx:"10"` // User Account (Writable, Signer, Fee Payer)
	TokenXProgram           solana.PublicKey `idx:"11"` // Token Program for X
	TokenYProgram           solana.PublicKey `idx:"12"` // Token Program for Y
	EventAuthority          solana.PublicKey `idx:"13"` // Event Authority
	Program                 solana.PublicKey `idx:"14"` // Meteora DLMM Program LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo
	Account1                solana.PublicKey `idx:"15"` // Additional Account 1
	Account2                solana.PublicKey `idx:"16"` // Additional Account 2
}

type PhoenixSwapAccounts struct {
	PhoenixProgram solana.PublicKey `idx:"0"` // Phoenix Program
	LogAuthority   solana.PublicKey `idx:"1"` // Log Authority
	Market         solana.PublicKey `idx:"2"` // Phoenix Market (Writable) - PoolID
	Trader         solana.PublicKey `idx:"3"` // Trader Account (Writable, Signer, Fee Payer)
	BaseAccount    solana.PublicKey `idx:"4"` // Base Token Account (Writable)
	QuoteAccount   solana.PublicKey `idx:"5"` // Quote Token Account (Writable)
	BaseVault      solana.PublicKey `idx:"6"` // Base Vault (Writable)
	QuoteVault     solana.PublicKey `idx:"7"` // Quote Vault (Writable)
	TokenProgram   solana.PublicKey `idx:"8"` // Token Program
}

type LifinitySwapV2Accounts struct {
	Authority             solana.PublicKey `idx:"0"`  // Pool Authority
	Amm                   solana.PublicKey `idx:"1"`  // Market (Writable) - Pool ID
	UserTransferAuthority solana.PublicKey `idx:"2"`  // User Transfer Authority (Writable, Signer, Fee Payer)
	SourceInfo            solana.PublicKey `idx:"3"`  // Source Info (Writable)
	DestinationInfo       solana.PublicKey `idx:"4"`  // Destination Info (Writable)
	SwapSource            solana.PublicKey `idx:"5"`  // Pool 2 (Writable)
	SwapDestination       solana.PublicKey `idx:"6"`  // Pool 1 (Writable)
	PoolMint              solana.PublicKey `idx:"7"`  // LP Token (Writable)
	FeeAccount            solana.PublicKey `idx:"8"`  // LP Fee Account (Writable)
	TokenProgram          solana.PublicKey `idx:"9"`  // Token Program
	OracleMain            solana.PublicKey `idx:"10"` // Oracle Main Account
	OracleSub             solana.PublicKey `idx:"11"` // Oracle Sub Account
	OraclePc              solana.PublicKey `idx:"12"` // Oracle Pc Account
}

// ParseAccountsIntoStruct is a generic function that parses accounts into any struct with account tags
func ParseAccountsIntoStruct[T any](accounts []solana.PublicKey) (result T) {
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		if accountIdx, ok := field.Tag.Lookup("idx"); ok {
			idx, err := strconv.Atoi(accountIdx)
			if err != nil {
				continue
			}
			if idx < len(accounts) {
				resultValue.Field(i).Set(reflect.ValueOf(accounts[idx]))
			}
		}
	}
	return result
}
