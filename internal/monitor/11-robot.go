package monitor

import (
	"context"
	"errors"
	"fmt"
	"solana-bot/internal/global"
	"solana-bot/internal/global/utils/fifomap"
	"solana-bot/internal/stream"

	"sync"
	"sync/atomic"
	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

var (
	botActivity   = &sync.Map{}
	RobotBuyCache *fifomap.FIFOMap

	canBuy atomic.Bool
)

type RobotMonitor struct {
	grpcClient *grpc.ClientConn
	Ctx        context.Context
	Cancel     context.CancelFunc
}

func NewRobotMonitor() (*RobotMonitor, error) {
	grpcClient := stream.Grpc_connect(stream.GRPCUrls[0], true)
	if grpcClient == nil {
		return nil, errors.New("无法连接到GRPC服务器")
	}
	ctx, cancel := context.WithCancel(context.Background())

	RobotBuyCache = fifomap.NewFIFOMap(10)

	return &RobotMonitor{
		grpcClient: grpcClient,
		Ctx:        ctx,
		Cancel:     cancel,
	}, nil

}

func (r *RobotMonitor) Start() {

	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-r.Ctx.Done():
				logx.Info("停止监听机器人")
				return
			case <-ticker.C:

				fmt.Printf("一分钟内活跃机器人数量：%d\n", GetActiveBots())

				if GetActiveBots() >= 4 { // 自定义阈值
					fmt.Println("🔥 活跃度高，准备狙击")
					canBuy.Store(true)
				} else {
					canBuy.Store(false)
				}
				botActivity = &sync.Map{}

			}
		}
	}()

	logx.Infof("[%s]:监听机器人交易", "all")
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

	robots, err := GetRobotConfig()
	if err != nil {
		logx.Error("获取机器人配置失败")
		return
	}

	subscription.Transactions["transactions_sub"].AccountInclude = robots.Robot
	go stream.Grpc_subscribe(r.grpcClient, &subscription, r.Ctx, subscribe)

	for {
		select {
		case <-r.Ctx.Done():
			logx.Info("停止监听机器人")
			return
		case msg := <-subscribe:
			got := msg.(*pb.SubscribeUpdate)
			tx := got.GetTransaction()
			if tx == nil || tx.Transaction.Transaction == nil || tx.Transaction.Meta == nil {
				continue
			}
			swapInfo, err := ParseSwapTransaction(tx.Transaction.Transaction, tx.Transaction.Meta)
			if err != nil || swapInfo == nil {
				continue
			}
			if swapInfo.TokenInMint.String() != global.Solana || //不是sol买入
				swapInfo.TokenOutMint.String() == global.Solana { // 排除卖出操作
				continue
			}
			robot := swapInfo.Signers[0].String()
			// logx.Infof("[%s]:收到机器人[%s]的交易{%s}", swapInfo.TokenOutMint, robot, solana.SignatureFromBytes(tx.Transaction.Signature).String())

			RobotBuyCache.Set(swapInfo.TokenOutMint.String(), robot)
			current, _ := botActivity.LoadOrStore(robot, 0)
			botActivity.Store(robot, current.(int)+1)

		}

	}
}

func GetActiveBots() int {
	activeBots := 0
	botActivity.Range(func(_, _ interface{}) bool {
		activeBots++
		return true
	})
	return activeBots
}
