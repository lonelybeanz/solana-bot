package helpers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"solana-bot/internal/dex/meteora/common"
)

// DeriveDbcPoolPDA derives the dbc pool address
func DeriveDbcPoolPDA(quoteMint, baseMint, config solana.PublicKey) solana.PublicKey {
	// pda order: the larger public key bytes goes first
	var mintA, mintB solana.PublicKey
	if bytes.Compare(quoteMint.Bytes(), baseMint.Bytes()) > 0 {
		mintA = quoteMint
		mintB = baseMint
	} else {
		mintA = baseMint
		mintB = quoteMint
	}
	seeds := [][]byte{
		[]byte("pool"),
		config.Bytes(),
		mintA.Bytes(),
		mintB.Bytes(),
	}
	pda, _, err := solana.FindProgramAddress(seeds, solana.MustPublicKeyFromBase58(common.DbcProgramID))
	if err != nil {
		log.Fatalf("find pool PDA: %v", err)
	}
	return pda
}

// DeriveTokenVaultPDA derives the dbc token vault address
func DeriveTokenVaultPDA(pool, mint solana.PublicKey) solana.PublicKey {
	seed := [][]byte{
		[]byte("token_vault"),
		mint.Bytes(),
		pool.Bytes(),
	}
	pda, _, err := solana.FindProgramAddress(seed, solana.MustPublicKeyFromBase58(common.DbcProgramID))
	if err != nil {
		log.Fatalf("find vault PDA: %v", err)
	}
	return pda
}

// DeriveEventAuthorityPDA derives the program event authority address
func DeriveEventAuthorityPDA() solana.PublicKey {
	seed := [][]byte{
		[]byte("__event_authority"),
	}
	pda, _, err := solana.FindProgramAddress(seed, solana.MustPublicKeyFromBase58(common.DbcProgramID))
	if err != nil {
		log.Fatalf("find event authority PDA: %v", err)
	}
	return pda
}

// DeriveMintMetadataPDA derives the mint metadata address
func DeriveMintMetadataPDA(mint solana.PublicKey) solana.PublicKey {
	seeds := [][]byte{
		[]byte("metadata"),
		solana.MustPublicKeyFromBase58(common.MetadataProgram).Bytes(),
		mint.Bytes(),
	}
	pda, _, err := solana.FindProgramAddress(seeds, solana.MustPublicKeyFromBase58(common.MetadataProgram))
	if err != nil {
		log.Fatalf("find mint metadata PDA: %v", err)
	}
	return pda
}

// GetPoolConfig fetches and deserializes pool configuration data from the Solana blockchain
func GetPoolConfig(ctx context.Context, configAddress solana.PublicKey, rpcClient *rpc.Client) (*common.PoolConfig, error) {
	account, err := rpcClient.GetAccountInfo(ctx, configAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool config account: %w", err)
	}

	if account == nil || account.Value == nil {
		return nil, fmt.Errorf("pool config account not found")
	}

	data := account.Value.Data.GetBinary()

	if len(data) < 8 {
		return nil, fmt.Errorf("data too short")
	}

	expectedDiscriminator := []byte{26, 108, 14, 123, 116, 230, 129, 43}
	if !bytes.Equal(data[:8], expectedDiscriminator) {
		return nil, fmt.Errorf("invalid discriminator, not a pool config account")
	}

	return DeserializePoolConfig(data)
}

// DeserializePoolConfig deserializes the binary data into a PoolConfig structure
func DeserializePoolConfig(data []byte) (*common.PoolConfig, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short to deserialize")
	}

	// Skip the 8-byte discriminator
	data = data[8:]

	config := &common.PoolConfig{}
	reader := bytes.NewReader(data)

	// Read QuoteMint
	if err := binary.Read(reader, binary.LittleEndian, &config.QuoteMint); err != nil {
		return nil, fmt.Errorf("failed to read QuoteMint: %w", err)
	}

	// Read FeeClaimer
	if err := binary.Read(reader, binary.LittleEndian, &config.FeeClaimer); err != nil {
		return nil, fmt.Errorf("failed to read FeeClaimer: %w", err)
	}

	// Read LeftoverReceiver
	if err := binary.Read(reader, binary.LittleEndian, &config.LeftoverReceiver); err != nil {
		return nil, fmt.Errorf("failed to read LeftoverReceiver: %w", err)
	}

	// Read PoolFees
	if err := binary.Read(reader, binary.LittleEndian, &config.PoolFees); err != nil {
		return nil, fmt.Errorf("failed to read PoolFees: %w", err)
	}

	// Read the uint8 fields
	if err := binary.Read(reader, binary.LittleEndian, &config.CollectFeeMode); err != nil {
		return nil, fmt.Errorf("failed to read CollectFeeMode: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.MigrationOption); err != nil {
		return nil, fmt.Errorf("failed to read MigrationOption: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.ActivationType); err != nil {
		return nil, fmt.Errorf("failed to read ActivationType: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.TokenDecimal); err != nil {
		return nil, fmt.Errorf("failed to read TokenDecimal: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.Version); err != nil {
		return nil, fmt.Errorf("failed to read Version: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.TokenType); err != nil {
		return nil, fmt.Errorf("failed to read TokenType: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.QuoteTokenFlag); err != nil {
		return nil, fmt.Errorf("failed to read QuoteTokenFlag: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.PartnerLockedLpPercentage); err != nil {
		return nil, fmt.Errorf("failed to read PartnerLockedLpPercentage: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.PartnerLpPercentage); err != nil {
		return nil, fmt.Errorf("failed to read PartnerLpPercentage: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.CreatorLockedLpPercentage); err != nil {
		return nil, fmt.Errorf("failed to read CreatorLockedLpPercentage: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.CreatorLpPercentage); err != nil {
		return nil, fmt.Errorf("failed to read CreatorLpPercentage: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.MigrationFeeOption); err != nil {
		return nil, fmt.Errorf("failed to read MigrationFeeOption: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.FixedTokenSupplyFlag); err != nil {
		return nil, fmt.Errorf("failed to read FixedTokenSupplyFlag: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.CreatorTradingFeePercentage); err != nil {
		return nil, fmt.Errorf("failed to read CreatorTradingFeePercentage: %w", err)
	}

	// Read padding fields
	if err := binary.Read(reader, binary.LittleEndian, &config.Padding0); err != nil {
		return nil, fmt.Errorf("failed to read Padding0: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.Padding1); err != nil {
		return nil, fmt.Errorf("failed to read Padding1: %w", err)
	}

	// Read uint64 fields
	if err := binary.Read(reader, binary.LittleEndian, &config.SwapBaseAmount); err != nil {
		return nil, fmt.Errorf("failed to read SwapBaseAmount: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.MigrationQuoteThreshold); err != nil {
		return nil, fmt.Errorf("failed to read MigrationQuoteThreshold: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.MigrationBaseThreshold); err != nil {
		return nil, fmt.Errorf("failed to read MigrationBaseThreshold: %w", err)
	}

	// Read MigrationSqrtPrice (uint128)
	if err := binary.Read(reader, binary.LittleEndian, &config.MigrationSqrtPrice); err != nil {
		return nil, fmt.Errorf("failed to read MigrationSqrtPrice: %w", err)
	}

	// Read LockedVestingConfig
	if err := binary.Read(reader, binary.LittleEndian, &config.LockedVestingConfig); err != nil {
		return nil, fmt.Errorf("failed to read LockedVestingConfig: %w", err)
	}

	// Read PreMigrationTokenSupply and PostMigrationTokenSupply
	if err := binary.Read(reader, binary.LittleEndian, &config.PreMigrationTokenSupply); err != nil {
		return nil, fmt.Errorf("failed to read PreMigrationTokenSupply: %w", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &config.PostMigrationTokenSupply); err != nil {
		return nil, fmt.Errorf("failed to read PostMigrationTokenSupply: %w", err)
	}

	// Read Padding2
	if err := binary.Read(reader, binary.LittleEndian, &config.Padding2); err != nil {
		return nil, fmt.Errorf("failed to read Padding2: %w", err)
	}

	// Read SqrtStartPrice
	if err := binary.Read(reader, binary.LittleEndian, &config.SqrtStartPrice); err != nil {
		return nil, fmt.Errorf("failed to read SqrtStartPrice: %w", err)
	}

	// Read Curve
	if err := binary.Read(reader, binary.LittleEndian, &config.Curve); err != nil {
		return nil, fmt.Errorf("failed to read Curve: %w", err)
	}

	return config, nil
}
