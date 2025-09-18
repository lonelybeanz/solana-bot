package monitor

import (
	"context"
	"fmt"
	"os"
	"time"

	"solana-bot/internal/client"
	"solana-bot/internal/dex/pump"
	"solana-bot/internal/global"
	atomic_ "solana-bot/internal/global/utils/atomic"
	"solana-bot/internal/global/utils/fifomap"
	"solana-bot/internal/global/utils/pubsub"

	"solana-bot/internal/rpcs"
	"solana-bot/internal/stream"

	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	PumpTokenMintAuthority    = "TSLvdd1pWpHVjahSpsvCXUbgwsL3JAcvokwaKt1eokM"
	RaydiumLaunchpadAuthority = "WLHv2UAZm6z4KyaaELi5pjdbJh6RESMva1Rnn8pJVVh"
	MonitorType               = map[string]bool{
		"mint":  false,
		"smart": true,
		"scm":   false,
	}

	BuyCache *fifomap.FIFOMap
)

func SetMonitorType(mt map[string]bool) {
	MonitorType = mt
}

// PumpFunMonitor 监控Pump.fun上的交易
type PumpFunMonitor struct {
	Wg            *sync.WaitGroup
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	paused        atomic_.Bool // 新增字段，用于控制暂停状态
	rpcs          []rpcs.RpcChannel
	streams       []*stream.GrpcStream
	httpClient    *rpc.Client
	wallet        *solana.Wallet
	pubsub        *pubsub.PubSub
	buyMultiplier *atomic_.Float64 // 买入系数
	lastBuyTime   *atomic_.Map     // string -> time.Time
}

var PumpMonitor *PumpFunMonitor

// NewPumpFunMonitor 创建监控实例
func NewPumpFunMonitor() (*PumpFunMonitor, error) {

	wallet, err := solana.WalletFromPrivateKeyBase58(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		logx.Must(err)
		return nil, err
	}
	grpcClient := stream.Grpc_connect(stream.GRPCUrl[0], true)
	if grpcClient == nil {
		return nil, err
	}

	global.ConnectToEndpoints()

	//开启区块订阅
	go global.BlockSubscribeWithRelay(grpcClient)
	go global.NonceSubscribeWithRelay(grpcClient)
	go global.WSOLSubscribeWithRelay(grpcClient)
	go global.UpdateGasWithRelay()

	ctx, cancel := context.WithCancel(context.Background())

	BuyCache = fifomap.NewFIFOMap(5)

	streams := []*stream.GrpcStream{
		stream.NewBlzStream(),
	}

	rpcs := []rpcs.RpcChannel{
		rpcs.NewJitoChannel(),
		rpcs.NewBlzChannel(),
		rpcs.NewAstralaneChannel(),
		// rpcs.NewTempgoralChannel(),
	}

	return &PumpFunMonitor{
			Wg:            &sync.WaitGroup{},
			mu:            sync.Mutex{},
			ctx:           ctx,
			cancel:        cancel,
			rpcs:          rpcs,
			streams:       streams,
			httpClient:    rpc.New(rpc.MainNetBeta.RPC),
			wallet:        wallet,
			lastBuyTime:   atomic_.NewMap(), // 初始化 Map
			pubsub:        pubsub.NewPubSub(),
			buyMultiplier: atomic_.NewFloat64(1.0),
		},
		nil
}

func (p *PumpFunMonitor) Go(fn func()) {
	p.mu.Lock()
	p.Wg.Add(1)
	p.mu.Unlock()

	go func() {
		defer p.Wg.Done()
		fn()
	}()
}

func (p *PumpFunMonitor) Start() {
	for k, v := range MonitorType {
		if k == "mint" && v {
			p.Go(func() {
				p.workerForMint()
			})
		}
		if k == "smart" && v {
			p.Go(func() {
				p.workerForSmart()
			})
		}
		if k == "scm" && v {
			p.Go(func() {
				p.workerForScm()
			})
		}
	}

	go p.Profit()
}

func (p *PumpFunMonitor) Stop() {
	//先停止监听
	p.cancel()

	p.Wg.Wait()

}

func (p *PumpFunMonitor) runWithCtx(ctx context.Context, ch <-chan interface{}, handler func(interface{})) {
	p.Go(func() {
		for {
			select {
			case <-ctx.Done():
				logx.Info("ctx stopped")
				return
			case <-p.ctx.Done():
				logx.Info("PumpFunMonitor stopped:", context.Cause(p.ctx))
				return
			case msg, ok := <-ch:
				if !ok {
					logx.Info("PumpFunMonitor channel closed")
					return
				}
				if p.paused.Load() {
					time.Sleep(time.Second) // 暂停时休眠一段时间
					continue
				}
				handler(msg)
			}
		}

	})
}

func (p *PumpFunMonitor) BurnToken(tokenAddress string) {
	mint := solana.MustPublicKeyFromBase58(tokenAddress)
	// 计算Associated Token Account的地址
	tokenAccount, _, err := solana.FindAssociatedTokenAddress(p.wallet.PublicKey(), mint)
	if err != nil {
		logx.Errorf("计算Associated Token Account地址失败: %v", err)
		return
	}

	rpcClient := global.GetRPCForRequest()
	big, _ := pump.GetTokenBalance(rpcClient, p.wallet.PublicKey(), mint)
	tokens := []struct {
		Mint         solana.PublicKey
		TokenAccount solana.PublicKey
		Amount       uint64
	}{
		{
			mint,
			tokenAccount,
			big.Uint64(),
		},
	}

	txHash, err := client.BatchBurnAndClose(rpcClient, p.wallet.PrivateKey, tokens)
	if err != nil {
		logx.Errorf("批量销毁和关闭交易失败: %v", err)
		return
	}
	logx.Infof("批量销毁和关闭交易成功, txHash: %s", txHash)

}

func (p *PumpFunMonitor) SendAndWait2(token string, tip uint64, txBuilder *global.TxBuilder) (*rpc.GetTransactionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	resultChan := make(chan *rpc.GetTransactionResult, len(p.rpcs))
	done := make(chan struct{}, 1)

	var wg sync.WaitGroup
	for _, r := range p.rpcs {
		r := r
		txBuilderCopy := *txBuilder
		wg.Add(1)
		go func() {
			defer wg.Done()

			sig, err := r.SendTransaction(p.wallet.PrivateKey, tip, txBuilderCopy)
			if err != nil {
				logx.Errorf("[%s]: {%s} SendTransaction error:%v", token, sig, err)
				select {
				case resultChan <- nil:
				case <-done:
				}
				return
			}

			resp, err := client.WaitForTransaction(ctx, p.httpClient, nil, sig)
			if err != nil {
				logx.Errorf("[%s]: {%s} SendAndWait error:%v", token, sig, err)
				select {
				case resultChan <- nil:
				case <-done:
				}
				return
			} else {
				logx.Infof("[%s]: {%s} success", token, sig)
			}

			select {
			case resultChan <- resp:
			case <-done:
			}

			select {
			case done <- struct{}{}:
			default:
			}
		}()
	}

	// 另起一个 goroutine 等待所有发送结束后关闭 resultChan
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 读取第一个非 nil 结果并返回
	for resp := range resultChan {
		if resp != nil {
			if resp.Meta != nil && resp.Meta.Err != nil {
				return nil, fmt.Errorf("SendAndWait error:%v", resp.Meta.Err)
			}
			return resp, nil
		}
	}

	return nil, fmt.Errorf("SendAndWait error: resp is nil")
}

func (p *PumpFunMonitor) SendAndWait(tx *solana.Transaction, skipPreflight bool) (*rpc.GetTransactionResult, error) {
	txHash := tx.Signatures[0].String()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	txResp := make(chan *rpc.GetTransactionResult)
	go func() {
		// rpcClient, wsClient := global.GetWSRPCForRequest()
		resp, err := client.WaitForTransaction(ctx, p.httpClient, nil, txHash)
		if err != nil {
			logx.Errorf("[%s]:SendAndWait error:%v", txHash, err)
		} else if resp.Meta.Err != nil {
			logx.Errorf("[%s]:SendAndWait: %v", txHash, resp.Meta.Err)
		} else {
			logx.Infof("[%s]: success", txHash)
		}
		txResp <- resp
	}()

	_, err := rpcs.SendTransaction(p.httpClient, tx)
	if err != nil {
		// spew.Dump(tx)
		logx.Errorf("[%s]:发送交易失败: %v", txHash, err.Error())
		return nil, err
	}

	resp := <-txResp
	if resp == nil {
		return nil, fmt.Errorf("SendAndWait error: resp is nil")
	}
	if resp.Meta != nil && resp.Meta.Err != nil {
		return nil, fmt.Errorf("SendAndWait error:%v", resp.Meta.Err)
	}

	return resp, nil
}
