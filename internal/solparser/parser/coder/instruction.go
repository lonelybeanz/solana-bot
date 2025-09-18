package coder

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	InitializeInstruction          = 0
	Initialize2Instruction         = 1
	MonitorStepInstruction         = 2
	DepositInstruction             = 3
	WithdrawInstruction            = 4
	MigrateToOpenBookInstruction   = 5
	SetParamsInstruction           = 6
	WithdrawPnlInstruction         = 7
	WithdrawSrmInstruction         = 8
	SwapBaseInInstruction          = 9
	PreInitializeInstruction       = 10
	SwapBaseOutInstruction         = 11
	SimulateInfoInstruction        = 12
	AdminCancelOrdersInstruction   = 13
	CreateConfigAccountInstruction = 14
	UpdateConfigAccountInstruction = 15
)

func DecodeData[T any](data []byte) (T, error) {
	buf := bytes.NewReader(data)
	var state T

	err := binary.Read(buf, binary.LittleEndian, &state)
	return state, err
}

func DecodePumpFunCpiLog(data []byte) (PumpFunAnchorSelfCPILogData, error) {
	return DecodeData[PumpFunAnchorSelfCPILogData](data)
}

// https://github.com/raydium-io/raydium-amm/blob/master/program/src/instruction.rs

// RaydiumAmmInstructionCoder implements the Coder interface.
type RaydiumAmmInstructionCoder struct{}

func NewRaydiumAmmInstructionCoder() *RaydiumAmmInstructionCoder {
	return &RaydiumAmmInstructionCoder{}
}

// Decode decodes the given byte array into an instruction.
func (coder *RaydiumAmmInstructionCoder) Decode(data []byte) (interface{}, int, error) {
	return decodeData(data)
}

func decodeData(data []byte) (interface{}, int, error) {
	buf := bytes.NewReader(data)
	var instructionID byte

	var result interface{}
	var err error
	if err = binary.Read(buf, binary.LittleEndian, &instructionID); err != nil {
		return nil, 0, err
	}
	switch instructionID {
	// case InitializeInstruction:
	// 	result, err = decodeInitialize(buf)
	case Initialize2Instruction:
		result, err = decodeInitialize2(buf)
	// case MonitorStepInstruction:
	// 	result, err = decodeMonitorStep(buf)
	case DepositInstruction:
		result, err = decodeDeposit(buf)
	case WithdrawInstruction:
		result, err = decodeWithdraw(buf)
	case MigrateToOpenBookInstruction:
		result = nil // No data to decode
	case SetParamsInstruction:
		result, err = decodeSetParams(buf)
	case WithdrawPnlInstruction:
		result = nil // No data to decode
	// case WithdrawSrmInstruction:
	// 	result, err = decodeWithdrawSrm(buf)
	case SwapBaseInInstruction:
		result, err = decodeSwapBaseIn(buf)
	case PreInitializeInstruction:
		result, err = decodePreInitialize(buf)
	case SwapBaseOutInstruction:
		result, err = decodeSwapBaseOut(buf)
	case SimulateInfoInstruction:
		result, err = decodeSimulateInfo(buf)
	case AdminCancelOrdersInstruction:
		result, err = decodeAdminCancelOrders(buf)
	case CreateConfigAccountInstruction:
		result = nil // No data to decode
	case UpdateConfigAccountInstruction:
		result, err = decodeUpdateConfigAccount(buf)
	default:
		return nil, 0, fmt.Errorf("invalid instruction ID %d", instructionID)
	}

	return result, int(instructionID), err
}

func decodeInitialize2(buf *bytes.Reader) (Initialize2, error) {
	var instruction Initialize2
	binary.Read(buf, binary.LittleEndian, &instruction.Nonce)
	binary.Read(buf, binary.LittleEndian, &instruction.OpenTime)
	binary.Read(buf, binary.LittleEndian, &instruction.InitPcAmount)
	binary.Read(buf, binary.LittleEndian, &instruction.InitCoinAmount)

	return instruction, nil
}

func decodeDeposit(buf *bytes.Reader) (Deposit, error) {
	var instruction Deposit

	binary.Read(buf, binary.LittleEndian, &instruction.MaxCoinAmount)
	binary.Read(buf, binary.LittleEndian, &instruction.MaxPcAmount)
	binary.Read(buf, binary.LittleEndian, &instruction.BaseSide)

	if buf.Len() >= 8 {
		var otherAmount uint64
		binary.Read(buf, binary.LittleEndian, &otherAmount)
		instruction.OtherAmountMin = &otherAmount
	}

	return instruction, nil
}

func decodeWithdraw(buf *bytes.Reader) (Withdraw, error) {
	var instruction Withdraw
	binary.Read(buf, binary.LittleEndian, &instruction.Amount)

	return instruction, nil
}

func decodeSwapBaseIn(buf *bytes.Reader) (SwapBaseIn, error) {
	var instruction SwapBaseIn
	binary.Read(buf, binary.LittleEndian, &instruction.AmountIn)
	binary.Read(buf, binary.LittleEndian, &instruction.MinimumAmountOut)

	return instruction, nil
}

func decodeSwapBaseOut(buf *bytes.Reader) (SwapBaseOut, error) {
	var instruction SwapBaseOut
	binary.Read(buf, binary.LittleEndian, &instruction.MaxAmountIn)
	binary.Read(buf, binary.LittleEndian, &instruction.AmountOut)

	return instruction, nil
}

func decodeSetParams(buf *bytes.Reader) (SetParams, error) {
	var instruction SetParams
	binary.Read(buf, binary.LittleEndian, &instruction.Param)

	// Different decoding based on param type
	switch instruction.Param {
	case 0, 1: // AmmOwner
		if buf.Len() >= 32 {
			instruction.NewPubkey = make([]byte, 32)
			binary.Read(buf, binary.LittleEndian, &instruction.NewPubkey)
		}
	case 2: // Fees
		if buf.Len() >= 8 {
			var value uint64
			binary.Read(buf, binary.LittleEndian, &value)
			instruction.Value = &value
		}
	case 3: // LastOrderDistance
		if buf.Len() >= 16 {
			instruction.LastOrderDistance = &LastOrderDistance{}
			binary.Read(buf, binary.LittleEndian, &instruction.LastOrderDistance.LastOrderNumerator)
			binary.Read(buf, binary.LittleEndian, &instruction.LastOrderDistance.LastOrderDenominator)
		}
	default:
		if buf.Len() >= 8 {
			var value uint64
			binary.Read(buf, binary.LittleEndian, &value)
			instruction.Value = &value
		}
	}
	return instruction, nil
}

func decodeSimulateInfo(buf *bytes.Reader) (SimulateInfo, error) {
	var instruction SimulateInfo
	binary.Read(buf, binary.LittleEndian, &instruction.Param)

	switch instruction.Param {
	case 0, 1: // PoolInfo, RunCrankInfo
		// No additional data to decode
	case 2: // SwapBaseInInfo
		if buf.Len() >= 16 {
			var swapBaseIn SwapBaseIn
			binary.Read(buf, binary.LittleEndian, &swapBaseIn.AmountIn)
			binary.Read(buf, binary.LittleEndian, &swapBaseIn.MinimumAmountOut)
			instruction.SwapBaseInValue = &swapBaseIn
		}
	case 3: // SwapBaseOutInfo
		if buf.Len() >= 16 {
			var swapBaseOut SwapBaseOut
			binary.Read(buf, binary.LittleEndian, &swapBaseOut.MaxAmountIn)
			binary.Read(buf, binary.LittleEndian, &swapBaseOut.AmountOut)
			instruction.SwapBaseOutValue = &swapBaseOut
		}
	}
	return instruction, nil
}

func decodePreInitialize(buf *bytes.Reader) (PreInitialize, error) {
	var instruction PreInitialize
	binary.Read(buf, binary.LittleEndian, &instruction.Nonce)
	return instruction, nil
}

func decodeAdminCancelOrders(buf *bytes.Reader) (AdminCancelOrders, error) {
	var instruction AdminCancelOrders
	binary.Read(buf, binary.LittleEndian, &instruction.Limit)
	return instruction, nil
}

func decodeUpdateConfigAccount(buf *bytes.Reader) (ConfigArgs, error) {
	var instruction ConfigArgs
	binary.Read(buf, binary.LittleEndian, &instruction.Param)

	switch instruction.Param {
	case 0, 1: // Owner related params
		instruction.Owner = make([]byte, 32)
		binary.Read(buf, binary.LittleEndian, &instruction.Owner)
	case 2: // CreatePoolFee
		var fee uint64
		binary.Read(buf, binary.LittleEndian, &fee)
		instruction.CreatePoolFee = &fee
	}
	return instruction, nil
}
