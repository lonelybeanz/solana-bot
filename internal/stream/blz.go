package stream

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"os"
	"solana-bot/internal/config"
	"solana-bot/internal/global"
	"solana-bot/internal/global/utils"
	"solana-bot/internal/pb/feepb"

	"strings"

	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// 参考github.com/BlockRazorinc/solana-trader-client-go

var (
	//添加shred加速的rpc节点
	GRPCUrls = []string{}
)

func NewBlzStream() *GrpcStream {
	GRPCUrls = strings.Split(os.Getenv("BLZ_GRPC_URLS"), ",")
	var conns []*grpc.ClientConn
	for _, url := range GRPCUrls {
		conn := Grpc_connect(url, true)
		if conn == nil {
			continue
		}
		conns = append(conns, conn)
	}
	return &GrpcStream{
		Conns:  conns,
		Xtoken: os.Getenv("BLZ_XTOKEN"),
	}
}

type Authentication struct {
	auth string
}

func (a *Authentication) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"apiKey": a.auth}, nil
}

func (a *Authentication) RequireTransportSecurity() bool {
	return false
}

const (
	gRPCEndpoint = "grpc.solana-fee.blockrazor.xyz:443" // endpoint address
)

func Fee_subscribe(ctx context.Context, recv chan interface{}) {

	const retryDelay = 3 * time.Second // 重连等待时间

	for {
		select {
		case <-ctx.Done():
			log.Println("Context canceled before (re)connecting, exiting")
			return
		default:
		}

		conn, err := grpc.Dial(
			gRPCEndpoint,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
			grpc.WithPerRPCCredentials(&Authentication{
				auth: os.Getenv("BLZ_AUTH_KEY"),
			}),
		)
		if err != nil {
			log.Printf("Dial error: %v. Retrying in %v...", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		defer conn.Close()

		client := feepb.NewServerClient(conn)

		stream, err := client.GetTransactionFeeStream(context.Background(), &feepb.TransactionFee{
			Percentile: 75,
			SlotRange:  5,
		})
		if err != nil {
			log.Printf("Failed to get stream: %v. Retrying in %v...", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		log.Println("Fee stream connected successfully")

	loop:
		for {
			select {
			case <-ctx.Done():
				log.Println("Context canceled, exiting stream loop")
				return
			default:
				message, err := stream.Recv()
				if err == io.EOF {
					log.Println("Stream closed by server")
					break loop // 退出内层 for，重新连接
				}
				if err != nil {
					log.Printf("Stream receive error: %v", err)
					break loop // 出错后退出内层 for，重新连接
				}

				// log.Printf("Received priority fee: %v", message.PriorityFee)
				recv <- message
			}
		}

		log.Printf("reconnecting after %v...", retryDelay)
		time.Sleep(retryDelay)
	}
}

func Grpc_subscribe(conn *grpc.ClientConn, subscription *pb.SubscribeRequest, ctx context.Context, recv chan interface{}) {
	var err error
	client := pb.NewGeyserClient(conn)

	subscriptionJson, err := json.Marshal(&subscription)
	if err != nil {
		logx.Errorf("Failed to marshal subscription request: %v", subscriptionJson)
	}
	logx.Infof("Subscription request: %s", string(subscriptionJson))

	// Set up the subscription request
	relayCtx := context.Background()
	md := metadata.New(map[string]string{"x-token": os.Getenv("BLZ_XTOKEN")})
	relayCtx = metadata.NewOutgoingContext(relayCtx, md)
	stream, err := client.Subscribe(relayCtx)
	if err != nil {
		logx.Errorf("%v", err)
		return
	}
	err = stream.Send(subscription)
	if err != nil {
		logx.Errorf("%v", err)
		return
	}
	logx.Info("yellowStoneSubscribe start recv ...")
	for {
		select {
		case <-ctx.Done():
			logx.Info("Context done, stopping subscription")
			close(recv)
			return
		default:
			resp, err := stream.Recv()

			//打印
			// spew.Dump(resp)

			if err == io.EOF {
				logx.Info("Stream closed")
				return
			}
			if err != nil {
				// log.Fatalf("Error occurred in receiving update: %v", err)
				continue
			}
			recv <- resp
		}
	}
}

func SlotSubscribeWithRelay(conn *grpc.ClientConn) {
	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	commitment := pb.CommitmentLevel_PROCESSED
	subscription.Commitment = &commitment
	if subscription.Slots == nil {
		subscription.Slots = make(map[string]*pb.SubscribeRequestFilterSlots)
	}

	subscription.Slots["slots"] = &pb.SubscribeRequestFilterSlots{}
	go Grpc_subscribe(conn, &subscription, context.Background(), subscribe)

	for msg := range subscribe {
		got := msg.(*pb.SubscribeUpdate)
		//打印
		// spew.Dump(got)

		updateBlock := got.GetBlockMeta()

		global.UpdateBlockData(updateBlock.Slot, solana.MustHashFromBase58(updateBlock.GetBlockhash()))

	}
}

func BlockSubscribeWithRelay(conn *grpc.ClientConn) {
	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	commitment := pb.CommitmentLevel_PROCESSED
	subscription.Commitment = &commitment
	if subscription.BlocksMeta == nil {
		subscription.BlocksMeta = make(map[string]*pb.SubscribeRequestFilterBlocksMeta)
	}
	subscription.BlocksMeta["block_meta"] = &pb.SubscribeRequestFilterBlocksMeta{}
	go Grpc_subscribe(conn, &subscription, context.Background(), subscribe)

	for msg := range subscribe {
		got := msg.(*pb.SubscribeUpdate)
		// spew.Dump(got)
		updateBlock := got.GetBlockMeta()
		if updateBlock == nil {
			continue
		}

		global.UpdateBlockData(updateBlock.Slot, solana.MustHashFromBase58(updateBlock.GetBlockhash()))

	}
}

func WSOLSubscribeWithRelay(conn *grpc.ClientConn) {

	player := solana.MustPublicKeyFromBase58(config.C.Bot.Player)

	bal, _ := global.GetTokenBalance(
		player,
		solana.SolMint,
	)
	if bal == nil || bal.Uint64() == 0 {
		global.SolATA_Balance.Store(big.NewInt(0))
		global.SolATA.Store(false)
	} else {
		global.SolATA_Balance.Store(big.NewInt(int64(bal.Uint64())))
		global.SolATA.Store(true)
	}

	balF := global.GetBalanceByPublic(config.C.Bot.Player)
	logx.Infof("获取 SOL 余额成功: %v \n", balF)
	if bal != nil {
		balU, _ := balF.Uint64()
		global.Sol_Balance.Store(balU)
	}

	tokenAccount, _, _ := solana.FindAssociatedTokenAddress(player, solana.SolMint)

	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	commitment := pb.CommitmentLevel_PROCESSED
	subscription.Commitment = &commitment
	subscription.Accounts = make(map[string]*pb.SubscribeRequestFilterAccounts)
	subscription.Accounts["account_sub"] = &pb.SubscribeRequestFilterAccounts{}
	subscription.Accounts["account_sub"].Account = []string{tokenAccount.String()}
	go Grpc_subscribe(conn, &subscription, context.Background(), subscribe)

	for msg := range subscribe {
		update, ok := msg.(*pb.SubscribeUpdate)
		if !ok {
			log.Printf("收到非预期类型消息: %T \n", msg)
			continue
		}

		accountSub := update.GetAccount()
		if accountSub == nil || accountSub.Account == nil {
			continue
		}

		accountData := accountSub.GetAccount().Data
		wsolToken, err := utils.TokenAccountFromData(accountData)
		if err != nil {
			log.Printf("解析 TokenAccount 数据失败: %v \n", err)
			continue
		}

		amount := int64(wsolToken.Amount)
		if amount > 0 {
			global.SolATA_Balance.Store(big.NewInt(amount)) // 可优化复用
			global.SolATA.Store(true)
		} else {
			global.SolATA_Balance.Store(big.NewInt(0))
			global.SolATA.Store(false)
		}

		go func() {
			bal := global.GetBalanceByPublic(config.C.Bot.Player)
			logx.Infof("获取 SOL 余额成功: %v \n", bal)
			if bal != nil {
				balU, _ := bal.Uint64()
				global.Sol_Balance.Store(balU)
			}
		}()

	}
}

func UpdateGasWithRelay() {
	subscribe := make(chan interface{})
	go Fee_subscribe(context.Background(), subscribe)

	for msg := range subscribe {
		got := msg.(*feepb.TransactionFeeStreamResponse)
		// spew.Dump(got)
		fee := got.GetPriorityFee()
		tip := got.GetTip()
		tipUint := uint64(tip.Value * 1e99)
		if fee == nil {
			continue
		}

		global.UpdateGasData(uint64(fee.Value), uint64(fee.Value), tipUint, tipUint)

	}
}

func NonceSubscribeWithRelay(conn *grpc.ClientConn) {

	go func() {

		for {
			for _, n := range global.NonceAccount {
				go global.SyncNonceAccount(n)
			}
			time.Sleep(time.Second * 1)
		}

	}()

	//订阅
	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	commitment := pb.CommitmentLevel_PROCESSED
	subscription.Commitment = &commitment
	subscription.Accounts = make(map[string]*pb.SubscribeRequestFilterAccounts)
	subscription.Accounts["account_sub"] = &pb.SubscribeRequestFilterAccounts{}
	subscription.Accounts["account_sub"].Account = global.NonceAccount
	go Grpc_subscribe(conn, &subscription, context.Background(), subscribe)

	for msg := range subscribe {
		update, ok := msg.(*pb.SubscribeUpdate)
		if !ok {
			log.Printf("收到非预期类型消息: %T \n", msg)
			continue
		}

		accountSub := update.GetAccount()
		if accountSub == nil || accountSub.Account == nil {
			continue
		}

		accountData := accountSub.GetAccount().Data
		dec := bin.NewBinDecoder(accountData)
		acc := new(system.NonceAccount)

		err := acc.UnmarshalWithDecoder(dec)
		if err != nil {
			log.Printf("解析nonce account失败: %v \n", err)
			continue
		}
		hash := solana.Hash(acc.Nonce)

		global.UpdataNonceAccountHash(string(accountSub.Account.GetPubkey()), hash)

	}
}
