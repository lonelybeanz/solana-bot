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
	"bytes"

	bin "github.com/gagliardetto/binary"
	solana "github.com/gagliardetto/solana-go"
)

// InstructionImpl is the interface that all instructions implement.
type InstructionImpl interface {
	bin.EncoderDecoder
	Validate() error
}

// AccountMetaGettable is an interface for getting account metas
type AccountMetaGettable interface {
	GetAccounts() []*solana.AccountMeta
}

// Instruction is a base type for all instructions.
type Instruction struct {
	bin.BaseVariant
}

// ProgramID returns the program ID.
func (inst *Instruction) ProgramID() solana.PublicKey {
	return ProgramID
}

// Accounts returns the list of accounts that this instruction requires.
func (inst *Instruction) Accounts() []*solana.AccountMeta {
	return inst.Impl.(AccountMetaGettable).GetAccounts()
}

// Data serializes the instruction data.
func (inst *Instruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	encoder := bin.NewBorshEncoder(buf)
	err := encoder.Encode(inst)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalWithEncoder implements the bin.EncoderDecoder interface
func (inst *Instruction) MarshalWithEncoder(encoder *bin.Encoder) error {
	return encoder.Encode(inst.Impl)
}

// UnmarshalWithDecoder implements the bin.EncoderDecoder interface
func (inst *Instruction) UnmarshalWithDecoder(decoder *bin.Decoder) error {
	return decoder.Decode(inst.Impl)
}

// InstructionImplDef is the interface that all instruction implementations must satisfy.
var _ solana.Instruction = (*Instruction)(nil)
var _ bin.EncoderDecoder = (*Instruction)(nil)
