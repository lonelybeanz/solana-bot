package parser

import (
	"context"
	"errors"
	"fmt"
	"time"

	"solana-bot/internal/solparser/consts"
	"solana-bot/internal/solparser/parser/coder"
	"solana-bot/internal/solparser/types"

	"github.com/avast/retry-go"
	"github.com/decert-me/solana-go-sdk/common"
	"github.com/decert-me/solana-go-sdk/program/token"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	minInnerInstructionsForSwap = 2
)

type SwapInstructions struct {
	swapIx      *rpc.ParsedInstruction
	transferIx1 *rpc.ParsedInstruction
	transferIx2 *rpc.ParsedInstruction
	transferIx3 *rpc.ParsedInstruction //for Lifinity V2
}

type InstructionContext struct {
	instruction *rpc.ParsedInstruction
	index       int
	innerInst   *rpc.ParsedInnerInstruction // nil for outer instructions
	innerIdx    int

	eventIndexIdentifier int
}

type SolParser struct {
	cli *rpc.Client
}

func NewSolParser(cli *rpc.Client) *SolParser {
	return &SolParser{cli: cli}
}

func (s *SolParser) GetParseFuncByProgramId(programId string) (func(*rpc.ParsedInstruction) (*types.SwapTransactionEvent, error), bool) {
	parseFuncs := map[string]func(*rpc.ParsedInstruction) (*types.SwapTransactionEvent, error){
		consts.RAYDIUM_V4_PROGRAM_ID:         s.ParseRaydiumAmmSwapEvent,
		consts.RAYDIUM_CLMM_PROGRAM_ID:       s.ParseClmmSwapEvent,
		consts.RAYDIUM_CPMM_PROGRAM_ID:       s.ParseRaydiumCpmmSwap,
		consts.ORCA_WHIRL_POOL_PROGRAM_ID:    s.ParseOrcaSwapEvent,
		consts.ORCA_TOKEN_SWAP_PROGRAM_ID:    s.ParseOrcaSwapEvent,
		consts.ORCA_TOKEN_SWAP_V2_PROGRAM_ID: s.ParseOrcaSwapEvent,
		consts.PUMP_FUN_PROGRAM_ID:           s.ParsePumpFunSwapEvent,
		consts.METEORA_DLMM_PROGRAM_ID:       s.ProcessMeteoraSwapEvent,
		consts.METEORA_DAMM_V2_PROGRAM_ID:    s.ProcessMeteoraDammV2SwapEvent,
		consts.PHNX_SWAP_PROGRAM_ID:          s.ParsePhoenixSwapEvent,
		//consts.LIFINITY_SWAP_V2_PROGRAM_ID: s.ParseLifinitySwapEvent,
	}
	parseFunc, exists := parseFuncs[programId]
	return parseFunc, exists
}

func (s *SolParser) ParseSwapEvent(parsedTransaction *rpc.GetParsedTransactionResult) ([]*types.SwapTransactionEvent, error) {
	if err := validateTransaction(parsedTransaction); err != nil {
		return nil, fmt.Errorf("invalid transaction: %w", err)
	}

	events := []*types.SwapTransactionEvent{}

	outerEvents := s.processInstructions(parsedTransaction, s.getOuterSwapInstructions)
	events = append(events, outerEvents...)

	innerEvents := s.processInstructions(parsedTransaction, s.getInnerSwapInstructions)
	events = append(events, innerEvents...)
	return events, nil
}

func (s *SolParser) processInstructions(
	tx *rpc.GetParsedTransactionResult,
	getInstructions func(*rpc.GetParsedTransactionResult) []InstructionContext,
) []*types.SwapTransactionEvent {
	var events []*types.SwapTransactionEvent
	for _, ctx := range getInstructions(tx) {
		swapInsts, err := s.extractSwapInstructions(tx, ctx)

		if err != nil {
			fmt.Printf("Unsupported swap instruction: [%d:%d] %s, %v", ctx.innerIdx+1, ctx.index+1, tx.Transaction.Signatures[0], err)
			continue
		}

		if swapInsts == nil {
			continue
		}
		var event *types.SwapTransactionEvent

		//log.Info("swapInsts: %+v, ctx %+v %+v", swapInsts.swapIx.ProgramId, swapInsts.transferIx1.ProgramId, swapInsts.transferIx2.ProgramId)
		if swapInsts.swapIx.ProgramId.String() == consts.METEORA_DAMM_V2_PROGRAM_ID {
			event, err = s.ParseInstructionIntoSwapEvent(
				tx,
				ctx.eventIndexIdentifier,
				swapInsts.swapIx,
				swapInsts.transferIx1,
				swapInsts.transferIx2,
			)
			//if swapInsts.transferIx3 != nil {
			//} else {
			//	fmt.Printf("lifinity v2 swap event has no transferIx3")
			//	continue
			//}
		} else if swapInsts.swapIx.ProgramId.String() == consts.PUMP_FUN_PROGRAM_ID {
			if swapInsts.transferIx2.ProgramId.String() == consts.PUMP_FUN_PROGRAM_ID {
				// tricky: 卖
				event, err = s.ParseInstructionIntoSwapEvent(
					tx,
					ctx.eventIndexIdentifier,
					swapInsts.swapIx,
					swapInsts.transferIx1,
					swapInsts.transferIx1, // on purpose
				)
				decodedPumpFunLog, err := coder.DecodePumpFunCpiLog(swapInsts.transferIx2.Data)
				if err != nil {
					fmt.Printf("error decoding pump.fun cpi log: %v", err)
					continue
				}
				event.OutToken = types.TokenAmt{
					Code:   consts.SOL_TOKEN_PROGRAM_ID,
					Amount: fmt.Sprintf("%d", decodedPumpFunLog.SolAmount),
				}
				// TODO: validation?
			} else {
				event, err = s.ParseInstructionIntoSwapEvent(
					tx,
					ctx.eventIndexIdentifier,
					swapInsts.swapIx,
					swapInsts.transferIx2,
					swapInsts.transferIx1,
				)
			}

		} else {
			event, err = s.ParseInstructionIntoSwapEvent(
				tx,
				ctx.eventIndexIdentifier,
				swapInsts.swapIx,
				swapInsts.transferIx1,
				swapInsts.transferIx2,
			)
		}
		if err != nil {
			//fmt.Errorf("error parsing %d:%d swap event %s: swapIx %+v, transferIx1 %+v, transferIx2 %+v, err %+v", ctx.innerIdx+1, ctx.index+1, tx.Transaction.Signatures[0], swapInsts.swapIx, swapInsts.transferIx1, swapInsts.transferIx2, err)
			fmt.Printf("error parsing %d:%d swap event %s: swapIx %+v, transferIx1 %+v, transferIx2 %+v, err %+v", ctx.innerIdx+1, ctx.index+1, tx.Transaction.Signatures[0], swapInsts.swapIx, swapInsts.transferIx1, swapInsts.transferIx2, err)
			continue
		}
		if event != nil {
			events = append(events, event)
		}
	}
	return events
}

// Get outer instructions context
func (s *SolParser) getOuterSwapInstructions(tx *rpc.GetParsedTransactionResult) []InstructionContext {
	var contexts []InstructionContext
	for idx, inst := range tx.Transaction.Message.Instructions {
		if isSwapInstruction(inst.ProgramId.String()) {
			contexts = append(contexts, InstructionContext{
				instruction:          inst,
				index:                idx,
				eventIndexIdentifier: idx + 1,
			})
		}
	}
	return contexts
}

func (s *SolParser) extractSwapInstructions(
	tx *rpc.GetParsedTransactionResult,
	ctx InstructionContext,
) (*SwapInstructions, error) {
	if !isSwapInstruction(ctx.instruction.ProgramId.String()) {
		return nil, nil
	}

	// Handle inner instructions
	// Swap Ix Inner Comes with Inner instructions
	// Example:
	// ix1
	// ix2
	// ix3 聚合器
	// ix3.1 swap ix
	// ix3.2 transfer ix1
	// ix3.2 transfer ix2
	// ...
	if ctx.innerInst == nil {
		if len(tx.Meta.InnerInstructions) == 0 {
			return nil, errors.New("no inner instructions found for swap")
		}
		for _, inst := range tx.Meta.InnerInstructions {
			if int(inst.Index) == ctx.index && len(inst.Instructions) >= 2 {
				// Check if first two instructions are token program transfers
				// 情况1: ix1: Token Program: transfer   (transfer1) In  Amount
				//       ix2: Token Program: transfer   (transfer2) Out Amount
				// 情况2: lifinity v2 ix1: Token Program: transfer (transfer1) In  Amount
				//                   ix2: Token Program: mintTo   (lpMint)
				//                   ix3: Token Program: transfer (transfer2) Out Amount
				// 情况3: pump.fun 内盘买  ix1: Token Program: transfer (transfer1) Out  Amount (内盘 Bonding Curve)
				//                       ix2: System Program: transfer (transfer2) In Amount (WSOL)
				//                       ix3: System Program: transfer (transfer3) In Amount (WSOL, pump.fun feeAccount)
				//                       ix4: Pump.fun: anchor Self CPI Log -  In Amount (Token), Out Amount (Token), IsBuy (Direction)
				// 情况4: pump.fun 内盘卖  ix1: Token Program: transfer (transfer1) User -> Pool (Token)
				//                       ix2: Pump.fun: anchor Self CPI Log -  In Amount (Token), Out Amount (Token), IsBuy (Direction)

				if (!IsTokenProgramId(inst.Instructions[0].ProgramId) || !IsTokenProgramId(inst.Instructions[1].ProgramId)) &&
					(!IsTokenProgramId(inst.Instructions[0].ProgramId) || !IsSystemProgrmId(inst.Instructions[1].ProgramId)) &&
					inst.Instructions[1].ProgramId.String() != consts.PUMP_FUN_PROGRAM_ID {
					continue
				}

				swapInst := &SwapInstructions{
					swapIx:      ctx.instruction,
					transferIx1: inst.Instructions[0],
					transferIx2: inst.Instructions[1],
				}

				// Check for optional third token transfer
				if len(inst.Instructions) >= 3 && IsTokenProgramId(inst.Instructions[2].ProgramId) {
					swapInst.transferIx3 = inst.Instructions[2]
				}

				return swapInst, nil
			}
		}
		return nil, errors.New("no inner instructions found for swap")
	}
	// Handle outer instructions
	// Swap Ix Outer Comes with Inner instructions
	// Example:
	// ix1
	// ix2
	// ix3 Swap
	// ix3.1 Transfer1
	// ix3.2 Transfer2
	if len(ctx.innerInst.Instructions) <= ctx.index+minInnerInstructionsForSwap {
		// for _, inst := range ctx.innerInst.Instructions {
		// 	fmt.Errorf("inner instruction: %+v", inst)
		// }
		// fmt.Errorf("%+v, %d", ctx.innerInst.Instructions, ctx.index)
		// fmt.Errorf("insufficient inner instructions for swap %+v", ctx.innerInst.Instructions)
		return nil, errors.New("insufficient inner instructions for swap")
	}

	swapInst := &SwapInstructions{
		swapIx:      ctx.instruction,
		transferIx1: ctx.innerInst.Instructions[ctx.index+1],
		transferIx2: ctx.innerInst.Instructions[ctx.index+2],
	}

	if len(ctx.innerInst.Instructions) > ctx.index+3 && IsTokenProgramId(ctx.innerInst.Instructions[ctx.index+2].ProgramId) {
		swapInst.transferIx3 = ctx.innerInst.Instructions[ctx.index+3]
	}

	return swapInst, nil
}

// Get inner instructions context
func (s *SolParser) getInnerSwapInstructions(tx *rpc.GetParsedTransactionResult) []InstructionContext {
	var contexts []InstructionContext

	for _, innerInst := range tx.Meta.InnerInstructions {
		for innerIdx, inst := range innerInst.Instructions {
			if isSwapInstruction(inst.ProgramId.String()) {
				finalIdx, err := createUniqueIndex(int(innerInst.Index), innerIdx)
				if err != nil {
					fmt.Printf("error creating unique index: %v", err)
					continue
				}
				contexts = append(contexts, InstructionContext{
					instruction:          inst,
					index:                innerIdx,
					innerInst:            &innerInst,
					innerIdx:             int(innerInst.Index),
					eventIndexIdentifier: finalIdx,
				})
			}

		}
	}
	return contexts
}

func (s *SolParser) ParseInstructionIntoSwapEvent(parsedTransaction *rpc.GetParsedTransactionResult, idxOuter int, swapIx *rpc.ParsedInstruction, transferIx1 *rpc.ParsedInstruction, transferIx2 *rpc.ParsedInstruction) (*types.SwapTransactionEvent, error) {

	feePayer := parsedTransaction.Transaction.Message.AccountKeys[0]
	if swapIx == nil {
		return nil, nil
	}

	parseFunc, exists := s.GetParseFuncByProgramId(swapIx.ProgramId.String())
	if !exists {
		return nil, fmt.Errorf("unsupported swap instruction: %s", swapIx.ProgramId.String())
	}

	swapEvent, err := parseFunc(swapIx)
	if err != nil || swapEvent == nil {
		return nil, fmt.Errorf("parsing swap event: %w", err)
	}

	swapEvent.MarketProgramId = swapIx.ProgramId.String()

	// Fill token amounts
	if err := s.fillTokenAmounts(swapEvent, transferIx1, transferIx2); err != nil {
		return swapEvent, err
	}

	// Set base fields
	swapEvent.Sender = feePayer.PublicKey.String()
	swapEvent.Receiver = feePayer.PublicKey.String()
	swapEvent.EventIndex = idxOuter

	return swapEvent, nil
}

// Helper function to fill token amounts
func (s *SolParser) fillTokenAmounts(swapEvent *types.SwapTransactionEvent, transferIx1, transferIx2 *rpc.ParsedInstruction) error {
	var err error
	if swapEvent.MarketProgramId == consts.PHNX_SWAP_PROGRAM_ID {
		tmp := transferIx1
		transferIx1 = transferIx2
		transferIx2 = tmp
	}
	if swapEvent.InToken, err = s.FillTokenAmtWithTransferIx(swapEvent.InToken, transferIx1); err != nil {
		return fmt.Errorf("filling in token amount: %w", err)
	}
	if transferIx2 == nil && swapEvent.MarketProgramId == consts.PUMP_FUN_PROGRAM_ID {
		return nil
	}
	if swapEvent.OutToken, err = s.FillTokenAmtWithTransferIx(swapEvent.OutToken, transferIx2); err != nil {
		return fmt.Errorf("filling out token amount: %w", err)
	}
	return nil
}

func (s *SolParser) FillTokenAmtWithTransferIx(tkAmt types.TokenAmt, ix *rpc.ParsedInstruction) (types.TokenAmt, error) {
	transfer, err := s.ParseTransfer(ix)
	if err != nil {
		return tkAmt, err
	}
	tkAmt.Amount = transfer.Info.Amount

	var mintAddress string // token mint address
	var tokenInfo *token.TokenAccount
	if tokenInfo, err = s.RetryGetTokenAccountInfoByTokenAccount(transfer.Info.Destination); err == nil && tokenInfo != nil {
		mintAddress = tokenInfo.Mint.String()
	} else if tokenInfo, err = s.RetryGetTokenAccountInfoByTokenAccount(transfer.Info.Source); err == nil && tokenInfo != nil {
		mintAddress = tokenInfo.Mint.String()
	} else {
		return tkAmt, err
	}
	tkAmt.Code = mintAddress
	return tkAmt, nil
}

func (s *SolParser) RetryGetTokenAccountInfoByTokenAccount(tokenAccount string) (*token.TokenAccount, error) {
	var tokenInfo *token.TokenAccount
	var err error
	err = retry.Do(func() error {
		tokenInfo, err = s.GetTokenAccountInfoByTokenAccount(tokenAccount)
		if err == nil {
			return nil
		}
		return err
	}, retry.Attempts(3), retry.Delay(1*time.Second), retry.LastErrorOnly(true), retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
		return retry.BackOffDelay(n, err, config)
	}))
	return tokenInfo, err

}

func (s *SolParser) GetTokenAccountInfoByTokenAccount(tokenAccount string) (*token.TokenAccount, error) {
	ctx := context.Background()
	accountInfo, e := s.cli.GetAccountInfo(ctx, solana.MustPublicKeyFromBase58(tokenAccount))
	if e != nil {
		return nil, e
	}
	if accountInfo.Value != nil && accountInfo.Value.Owner == solana.SystemProgramID {
		uint64One := uint64(1)
		// 账户本身就是一个 lamport 账户，非Token账户
		t := &token.TokenAccount{
			Mint:            common.PublicKey(solana.SolMint),
			Owner:           common.PublicKeyFromString(tokenAccount),
			Amount:          accountInfo.Value.Lamports,
			Delegate:        nil,
			State:           1, //tokenAccount.Initialized
			IsNative:        &uint64One,
			DelegatedAmount: 0,
			CloseAuthority:  nil,
		}
		return t, nil
	}
	t, err2 := token.TokenAccountFromData(accountInfo.GetBinary())
	if err2 != nil {
		return nil, errors.New(fmt.Sprintf("error decoding token account data: %v", err2))
	}
	return &t, nil
}
