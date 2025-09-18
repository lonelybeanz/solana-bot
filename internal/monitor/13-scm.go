package monitor

import (
	"context"
	"math/big"
	"solana-bot/internal/client"
	"solana-bot/internal/global"
	"solana-bot/internal/stream"
	"sync"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

func (p *PumpFunMonitor) workerForScm() {
	p.ListenScmAccountTransation()
}

func (p *PumpFunMonitor) ListenScmAccountTransation() {

	logx.Infof("[%s]:监听账户交易", "all")
	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	commitment := pb.CommitmentLevel_PROCESSED
	subscription.Commitment = &commitment
	subscription.Transactions = make(map[string]*pb.SubscribeRequestFilterTransactions)
	failed := false
	vote := false
	subscription.Transactions["transactions_sub"] = &pb.SubscribeRequestFilterTransactions{
		Failed: &failed,
		Vote:   &vote,
	}

	subscription.Transactions["transactions_sub"].AccountInclude = []string{"2x4Mp5dLefbx8V684sQDxNxUwgp3bqoibAgacLqs5z19"}

	var once sync.Once
	for _, s := range p.streams {
		s.Subscribe(p.ctx, &subscription, &once, subscribe)
	}

	p.runWithCtx(context.Background(), subscribe, func(msg interface{}) {
		var got *pb.SubscribeUpdate
		v := msg.(*stream.StreamMessage)
		got = v.Data.(*pb.SubscribeUpdate)
		tx := got.GetTransaction()
		if tx == nil {
			return
		}
		// token, err := p.ParseCreateTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
		// if err != nil || token == nil {
		// 	return
		// }

		swapInfo, err := ParseSwapTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
		if err != nil || swapInfo == nil {
			return
		}

		logx.Infof("[%s]:收到scm交易{%s}", swapInfo.TokenOutMint, solana.SignatureFromBytes(tx.Transaction.Signature).String())

		// if canBuy.Load() {
		go p.ScmBackRun(swapInfo, tx.Slot)
		// }

	})

}

func (p *PumpFunMonitor) ScmBackRun(swapInfo *solanaswapgo.SwapInfo, slot uint64) {
	if swapInfo.TokenInMint.String() != global.Solana {
		return
	}
	//别人买多少，就卖多少
	logx.Infof("[%s]:开始回本", swapInfo.TokenOutMint)
	ts := NewTokenJupiterSwap(swapInfo.TokenOutMint.String())
	for i := 0; i < 10; i++ {
		_, err := p.sellWithRaydium(ts, big.NewInt(int64(swapInfo.TokenOutAmount)), float32(100))
		if err != nil {
			logx.Errorf("[%s]:回本失败: %v", swapInfo.TokenOutMint, err)
			continue
		}
	}

}

func (p *PumpFunMonitor) sellWithRaydium(ts *TokenSwap, amount *big.Int, slippage float32) (*rpc.GetTransactionResult, error) {

	wallet, _ := solana.WalletFromPrivateKeyBase58("2rGpKU7MXBWsdrugTyTcRepa25zmEkeYYj4Bjqv68rRXkzgAvh2rcVQfSrAz2G8Bdcv3Ve3nfCftLZPErCMaH65W")

	if err := client.RaySwapByApi(p.httpClient, wallet, "7YpUFdHWmjbLF559hppQcixmx5napb3gWpg9LCgQ9oyd", "So11111111111111111111111111111111111111112", "2qCcBUbmChGyvAjT9QRamZRgpWp943YXYXTix7wdJ3xB", "Dcs6Nf2MMPsMEkbBv6Vy6f76fHYerLMFhAaVXJa9gX4Q", amount.Uint64(), uint64(slippage)); err != nil {
		logx.Errorf("[%s]:Raydium swap error: %v", ts.Token.TokenAddress, err)
		return nil, err
	}
	return nil, nil
}
