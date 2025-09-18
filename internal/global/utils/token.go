package utils

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/gagliardetto/solana-go"
)

var (
	Some = []byte{1, 0, 0, 0}
	None = []byte{0, 0, 0, 0}
)

var (
	ErrInvalidAccountOwner    = errors.New("invalid account owner")
	ErrInvalidAccountDataSize = errors.New("invalid account data size")
)

const MaxSigners = 11

const MultisigAccountSize = 355

type MultisigAccount struct {
	M             uint8 // Number of signers required
	N             uint8 // Number of valid signers
	IsInitialized bool
	Signers       []solana.PublicKey
}

func MultisigAccountFromData(data []byte) (MultisigAccount, error) {
	if len(data) != MultisigAccountSize {
		return MultisigAccount{}, ErrInvalidAccountDataSize
	}

	m := uint8(data[0])

	n := uint8(data[1])

	isInitialized := data[2] == 1

	signers := make([]solana.PublicKey, 0, MaxSigners)
	current := 3
	for current < MultisigAccountSize {
		pubkey := solana.PublicKeyFromBytes(data[current : current+32])
		if pubkey == (solana.PublicKey{}) {
			break
		}
		signers = append(signers, pubkey)
		current += 32
	}

	return MultisigAccount{
		M:             m,
		N:             n,
		IsInitialized: isInitialized,
		Signers:       signers,
	}, nil
}

const MintAccountSize = 82

type MintAccount struct {
	MintAuthority   *solana.PublicKey
	Supply          uint64
	Decimals        uint8
	IsInitialized   bool
	FreezeAuthority *solana.PublicKey
}

func MintAccountFromData(data []byte) (MintAccount, error) {
	if len(data) != MintAccountSize {
		return MintAccount{}, ErrInvalidAccountDataSize
	}

	var mint *solana.PublicKey
	if bytes.Equal(data[:4], Some) {
		key := solana.PublicKeyFromBytes(data[4:36])
		mint = &key
	}

	supply := binary.LittleEndian.Uint64(data[36:44])

	decimals := uint8(data[44])

	isInitialized := data[45] == 1

	var freezeAuthority *solana.PublicKey
	if bytes.Equal(data[46:50], Some) {
		key := solana.PublicKeyFromBytes(data[50:82])
		freezeAuthority = &key
	}

	return MintAccount{
		MintAuthority:   mint,
		Supply:          supply,
		Decimals:        decimals,
		IsInitialized:   isInitialized,
		FreezeAuthority: freezeAuthority,
	}, nil
}

const TokenAccountSize = 165

type TokenAccountState uint8

const (
	TokenAccountStateUninitialized TokenAccountState = iota
	TokenAccountStateInitialized
	TokenAccountFrozen
)

// TokenAccount is token program account
type TokenAccount struct {
	Mint     solana.PublicKey
	Owner    solana.PublicKey
	Amount   uint64
	Delegate *solana.PublicKey
	State    TokenAccountState
	// if is wrapped SOL, IsNative is the rent-exempt value
	IsNative        *uint64
	DelegatedAmount uint64
	CloseAuthority  *solana.PublicKey
}

func TokenAccountFromData(data []byte) (TokenAccount, error) {
	if len(data) != TokenAccountSize {
		return TokenAccount{}, ErrInvalidAccountDataSize
	}

	mint := solana.PublicKeyFromBytes(data[:32])

	owner := solana.PublicKeyFromBytes(data[32:64])

	amount := binary.LittleEndian.Uint64(data[64:72])

	var delegate *solana.PublicKey
	if bytes.Equal(data[72:76], Some) {
		key := solana.PublicKeyFromBytes(data[76:108])
		delegate = &key
	}

	state := TokenAccountState(data[108])

	var isNative *uint64
	if bytes.Equal(data[109:113], Some) {
		num := binary.LittleEndian.Uint64(data[113:121])
		isNative = &num
	}

	delegatedAmount := binary.LittleEndian.Uint64(data[121:129])

	var closeAuthority *solana.PublicKey
	if bytes.Equal(data[129:133], Some) {
		key := solana.PublicKeyFromBytes(data[133:165])
		closeAuthority = &key
	}

	return TokenAccount{
		Mint:            mint,
		Owner:           owner,
		Amount:          amount,
		Delegate:        delegate,
		State:           state,
		IsNative:        isNative,
		DelegatedAmount: delegatedAmount,
		CloseAuthority:  closeAuthority,
	}, nil
}

func DeserializeTokenAccount(data []byte, accountOwner solana.PublicKey) (TokenAccount, error) {
	if accountOwner != solana.TokenProgramID {
		return TokenAccount{}, ErrInvalidAccountOwner
	}
	return TokenAccountFromData(data)
}
