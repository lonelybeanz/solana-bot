package common

import (
	"github.com/gagliardetto/solana-go"
)

// PoolFeesConfig represents the fee configuration for a pool
type PoolFeesConfig struct {
	SwapFeeNumerator   uint64
	SwapFeeDenominator uint64
}

// LiquidityDistributionConfig represents the liquidity distribution configuration
type LiquidityDistributionConfig struct {
	TickIndex          int32
	StartX             uint64
	StartY             uint64
	LiquidityParameter uint64
}

// LockedVestingConfig represents the locked vesting configuration
type LockedVestingConfig struct {
	PeriodInSeconds        uint64
	NumberOfPeriods        uint8
	PeriodsCliff           uint8
	UnlockBeforeMigration  uint8
	UnlockingBehavior      uint8
	EnabledBeforeMigration uint8
	EnabledAfterMigration  uint8
	PaddingBuffer          [10]uint8
}

// PoolConfig represents the pool configuration data structure
type PoolConfig struct {
	QuoteMint                   solana.PublicKey
	FeeClaimer                  solana.PublicKey
	LeftoverReceiver            solana.PublicKey
	PoolFees                    PoolFeesConfig
	CollectFeeMode              uint8
	MigrationOption             uint8
	ActivationType              uint8
	TokenDecimal                uint8
	Version                     uint8
	TokenType                   uint8
	QuoteTokenFlag              uint8
	PartnerLockedLpPercentage   uint8
	PartnerLpPercentage         uint8
	CreatorLockedLpPercentage   uint8
	CreatorLpPercentage         uint8
	MigrationFeeOption          uint8
	FixedTokenSupplyFlag        uint8
	CreatorTradingFeePercentage uint8
	Padding0                    [2]uint8
	Padding1                    [8]uint8
	SwapBaseAmount              uint64
	MigrationQuoteThreshold     uint64
	MigrationBaseThreshold      uint64
	MigrationSqrtPrice          uint128
	LockedVestingConfig         LockedVestingConfig
	PreMigrationTokenSupply     uint64
	PostMigrationTokenSupply    uint64
	Padding2                    [2]uint128
	SqrtStartPrice              uint128
	Curve                       [20]LiquidityDistributionConfig
}

// uint128 is a custom type to handle 128-bit integers
type uint128 struct {
	Lo uint64
	Hi uint64
}
