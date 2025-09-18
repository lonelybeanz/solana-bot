// Copyright 2025 github.com/dwnfan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package token2022

import (
	"errors"
	"fmt"

	bin "github.com/gagliardetto/binary"
	solana "github.com/gagliardetto/solana-go"
	format "github.com/gagliardetto/solana-go/text/format"
	treeout "github.com/gagliardetto/treeout"
)

// ProgramName is the name of the Associated Token Account program for Token 2022
const ProgramName = "Associated Token Account Program"

// ProgramID is the ID of the Associated Token Account program
var ProgramID = solana.SPLAssociatedTokenAccountProgramID

type Create2022 struct {
	Payer  solana.PublicKey `bin:"-" borsh_skip:"true"`
	Wallet solana.PublicKey `bin:"-" borsh_skip:"true"`
	Mint   solana.PublicKey `bin:"-" borsh_skip:"true"`

	// [0] = [WRITE, SIGNER] Payer
	// ··········· Funding account
	//
	// [1] = [WRITE] AssociatedTokenAccount
	// ··········· Associated token account address to be created
	//
	// [2] = [] Wallet
	// ··········· Wallet address for the new associated token account
	//
	// [3] = [] TokenMint
	// ··········· The token mint for the new associated token account
	//
	// [4] = [] SystemProgram
	// ··········· System program ID
	//
	// [5] = [] TokenProgram
	// ··········· Token 2022 program ID
	//
	// [6] = [] SysVarRent
	// ··········· SysVarRentPubkey
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

// NewCreate2022InstructionBuilder creates a new `Create2022` instruction builder.
func NewCreate2022InstructionBuilder() *Create2022 {
	nd := &Create2022{}
	return nd
}

func (inst *Create2022) SetPayer(payer solana.PublicKey) *Create2022 {
	inst.Payer = payer
	return inst
}

func (inst *Create2022) SetWallet(wallet solana.PublicKey) *Create2022 {
	inst.Wallet = wallet
	return inst
}

func (inst *Create2022) SetMint(mint solana.PublicKey) *Create2022 {
	inst.Mint = mint
	return inst
}

func (inst Create2022) Build() *Instruction {

	associatedTokenAddress, _, _ := FindAssociatedTokenAddress2022(
		inst.Wallet,
		inst.Mint,
	)

	keys := []*solana.AccountMeta{
		{
			PublicKey:  inst.Payer,
			IsSigner:   true,
			IsWritable: true,
		},
		{
			PublicKey:  associatedTokenAddress,
			IsSigner:   false,
			IsWritable: true,
		},
		{
			PublicKey:  inst.Wallet,
			IsSigner:   false,
			IsWritable: false,
		},
		{
			PublicKey:  inst.Mint,
			IsSigner:   false,
			IsWritable: false,
		},
		{
			PublicKey:  solana.SystemProgramID,
			IsSigner:   false,
			IsWritable: false,
		},
		{
			PublicKey:  solana.Token2022ProgramID,
			IsSigner:   false,
			IsWritable: false,
		},
		{
			PublicKey:  solana.SysVarRentPubkey,
			IsSigner:   false,
			IsWritable: false,
		},
	}

	inst.AccountMetaSlice = keys

	return &Instruction{BaseVariant: bin.BaseVariant{
		Impl:   inst,
		TypeID: bin.NoTypeIDDefaultID,
	}}
}

// ValidateAndBuild validates the instruction accounts.
// If there is a validation error, return the error.
// Otherwise, build and return the instruction.
func (inst Create2022) ValidateAndBuild() (*Instruction, error) {
	if err := inst.Validate(); err != nil {
		return nil, err
	}
	return inst.Build(), nil
}

func (inst *Create2022) Validate() error {
	if inst.Payer.IsZero() {
		return errors.New("Payer not set")
	}
	if inst.Wallet.IsZero() {
		return errors.New("Wallet not set")
	}
	if inst.Mint.IsZero() {
		return errors.New("Mint not set")
	}
	_, _, err := FindAssociatedTokenAddress2022(
		inst.Wallet,
		inst.Mint,
	)
	if err != nil {
		return fmt.Errorf("error while FindAssociatedTokenAddress2022: %w", err)
	}
	return nil
}

func (inst *Create2022) EncodeToTree(parent treeout.Branches) {
	parent.Child(format.Program(ProgramName, ProgramID)).
		//
		ParentFunc(func(programBranch treeout.Branches) {
			programBranch.Child(format.Instruction("Create2022")).
				//
				ParentFunc(func(instructionBranch treeout.Branches) {

					// Parameters of the instruction:
					instructionBranch.Child("Params[len=0]").ParentFunc(func(paramsBranch treeout.Branches) {})

					// Accounts of the instruction:
					instructionBranch.Child("Accounts[len=7").ParentFunc(func(accountsBranch treeout.Branches) {
						accountsBranch.Child(format.Meta("                 payer", inst.AccountMetaSlice.Get(0)))
						accountsBranch.Child(format.Meta("associatedTokenAddress", inst.AccountMetaSlice.Get(1)))
						accountsBranch.Child(format.Meta("                wallet", inst.AccountMetaSlice.Get(2)))
						accountsBranch.Child(format.Meta("             tokenMint", inst.AccountMetaSlice.Get(3)))
						accountsBranch.Child(format.Meta("         systemProgram", inst.AccountMetaSlice.Get(4)))
						accountsBranch.Child(format.Meta("     token2022Program", inst.AccountMetaSlice.Get(5)))
						accountsBranch.Child(format.Meta("            sysVarRent", inst.AccountMetaSlice.Get(6)))
					})
				})
		})
}

func (inst Create2022) MarshalWithEncoder(encoder *bin.Encoder) error {
	return encoder.WriteBytes([]byte{}, false)
}

func (inst *Create2022) UnmarshalWithDecoder(decoder *bin.Decoder) error {
	return nil
}

// GetAccounts implements the AccountMetaGettable interface
func (inst Create2022) GetAccounts() []*solana.AccountMeta {
	return inst.AccountMetaSlice
}

// NewCreate2022Instruction creates a new instruction for creating an associated token account for Token 2022
func NewCreate2022Instruction(
	payer solana.PublicKey,
	walletAddress solana.PublicKey,
	splTokenMintAddress solana.PublicKey,
) *Create2022 {
	return NewCreate2022InstructionBuilder().
		SetPayer(payer).
		SetWallet(walletAddress).
		SetMint(splTokenMintAddress)
}
