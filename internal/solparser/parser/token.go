package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"solana-bot/internal/solparser/parser/coder"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"strings"
)

func (s *SolParser) parseInstruction(ix *rpc.ParsedInstruction) ([]byte, error) {
	if ix == nil {
		return nil, errors.New("parsed instruction is nil")
	}

	if ix.Parsed == nil && len(ix.Data) == 0 {
		return nil, errors.New("instruction has no parseable data")
	}

	if ix.Data == nil {
		return ix.Parsed.MarshalJSON()
	}
	return ix.Data, nil
}

func (s *SolParser) ParseTokenTransfer(ix *rpc.ParsedInstruction) (*coder.TokenTransfer, error) {
	byteMsg, err := s.parseInstruction(ix)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction: %w", err)
	}

	msgStr := string(byteMsg)
	if strings.Contains(msgStr, "transferChecked") {
		transfer1 := &coder.TokenTransferChecked{}
		if err := json.Unmarshal(byteMsg, transfer1); err != nil {
			return nil, fmt.Errorf("unmarshaling checked transfer: %w", err)
		}
		return &coder.TokenTransfer{
			Info: struct {
				Amount      string `json:"amount"`
				Authority   string `json:"authority"`
				Destination string `json:"destination"`
				Source      string `json:"source"`
			}{
				Amount:      transfer1.Info.TokenAmount.Amount,
				Authority:   transfer1.Info.Authority,
				Destination: transfer1.Info.Destination,
				Source:      transfer1.Info.Source,
			},
		}, nil
	}

	if strings.Contains(msgStr, "transfer") {
		transfer := &coder.TokenTransfer{}
		if err := json.Unmarshal(byteMsg, transfer); err != nil {
			return nil, fmt.Errorf("unmarshaling transfer: %w", err)
		}
		return transfer, nil
	}

	return nil, errors.New(fmt.Sprintf("not a valid transfer instruction %s", ix.ProgramId.String()))
}

func (s *SolParser) ParseSystemTransfer(ix *rpc.ParsedInstruction) (*coder.SystemTransfer, error) {
	if ix.ProgramId != solana.SystemProgramID {
		return nil, errors.New("not a system transfer")
	}

	byteMsg, err := s.parseInstruction(ix)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction: %w", err)
	}

	transfer := &coder.SystemTransfer{}
	if err := json.Unmarshal(byteMsg, transfer); err != nil {
		return nil, fmt.Errorf("unmarshaling system transfer: %w", err)
	}
	return transfer, nil
}

func (s *SolParser) ParseTransfer(ix *rpc.ParsedInstruction) (*coder.TokenTransfer, error) {
	switch ix.ProgramId {
	case solana.TokenProgramID, solana.Token2022ProgramID:
		return s.ParseTokenTransfer(ix)
	case solana.SystemProgramID:
		// 将 Solana System Program 的 transfer struct 转换为 Token Transfer
		solTransfer, err := s.ParseSystemTransfer(ix)
		if err != nil {
			return nil, err
		}
		tokenTransfer := &coder.TokenTransfer{
			Info: struct {
				Amount      string `json:"amount"`
				Authority   string `json:"authority"`
				Destination string `json:"destination"`
				Source      string `json:"source"`
			}{
				Amount:      fmt.Sprintf("%d", solTransfer.Info.Lamports),
				Authority:   solTransfer.Info.Source,
				Destination: solTransfer.Info.Destination,
				Source:      solTransfer.Info.Source,
			},
			InstructionType: "transfer",
		}
		return tokenTransfer, nil
	default:
		return nil, errors.New(fmt.Sprintf("not a valid transfer instruction %s", ix.ProgramId.String()))
	}
}
