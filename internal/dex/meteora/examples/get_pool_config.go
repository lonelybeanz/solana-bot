package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"solana-bot/internal/dex/meteora/common"
	"solana-bot/internal/dex/meteora/helpers"
)

func GetPoolConfig() {
	rpcClient := rpc.New("https://mainnet.helius-rpc.com/?api-key=YOUR_API_KEY")

	configAddressStr := "J83XMX95H4dzsxtVdregSVymGmYQucDzfRRxuuqAtaqc"

	fmt.Println("Getting pool config...")
	configAddress := solana.MustPublicKeyFromBase58(configAddressStr)
	ctx := context.Background()
	poolConfig, err := helpers.GetPoolConfig(ctx, configAddress, rpcClient)
	if err != nil {
		log.Fatalf("Failed to get pool config: %v", err)
	}
	printPoolConfig(poolConfig)

}

func printPoolConfig(poolConfig *common.PoolConfig) {
	fmt.Println("Pool Configuration Details:")
	fmt.Println("===========================")
	fmt.Printf("Quote Mint: %s\n", poolConfig.QuoteMint.String())
	fmt.Printf("Fee Claimer: %s\n", poolConfig.FeeClaimer.String())
	fmt.Printf("Leftover Receiver: %s\n", poolConfig.LeftoverReceiver.String())

	// Fee details
	fmt.Printf("Pool Fees - Swap Fee Numerator: %d\n", poolConfig.PoolFees.SwapFeeNumerator)
	fmt.Printf("Pool Fees - Swap Fee Denominator: %d\n", poolConfig.PoolFees.SwapFeeDenominator)

	// Other config details
	fmt.Printf("Token Decimal: %d\n", poolConfig.TokenDecimal)
	fmt.Printf("Token Type: %d\n", poolConfig.TokenType)
	fmt.Printf("Version: %d\n", poolConfig.Version)
	fmt.Printf("Migration Option: %d\n", poolConfig.MigrationOption)
	fmt.Printf("Collect Fee Mode: %d\n", poolConfig.CollectFeeMode)
	fmt.Printf("Creator Trading Fee Percentage: %d\n", poolConfig.CreatorTradingFeePercentage)

	// Token supply details
	fmt.Printf("Pre-Migration Token Supply: %d\n", poolConfig.PreMigrationTokenSupply)
	fmt.Printf("Post-Migration Token Supply: %d\n", poolConfig.PostMigrationTokenSupply)
	fmt.Printf("Fixed Token Supply Flag: %d\n", poolConfig.FixedTokenSupplyFlag)

	// LP Details
	fmt.Printf("Partner LP Percentage: %d\n", poolConfig.PartnerLpPercentage)
	fmt.Printf("Partner Locked LP Percentage: %d\n", poolConfig.PartnerLockedLpPercentage)
	fmt.Printf("Creator LP Percentage: %d\n", poolConfig.CreatorLpPercentage)
	fmt.Printf("Creator Locked LP Percentage: %d\n", poolConfig.CreatorLockedLpPercentage)

	// Price and thresholds
	fmt.Printf("Sqrt Start Price: %v\n", poolConfig.SqrtStartPrice)
	fmt.Printf("Migration Sqrt Price: %v\n", poolConfig.MigrationSqrtPrice)
	fmt.Printf("Swap Base Amount: %d\n", poolConfig.SwapBaseAmount)
	fmt.Printf("Migration Quote Threshold: %d\n", poolConfig.MigrationQuoteThreshold)
	fmt.Printf("Migration Base Threshold: %d\n", poolConfig.MigrationBaseThreshold)
}

func main() {
	GetPoolConfig()
}
