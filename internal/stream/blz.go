package stream

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"os"
	"solana-bot/internal/pb/feepb"
	"strings"

	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// 参考github.com/BlockRazorinc/solana-trader-client-go

var (
	//添加shred加速的rpc节点
	GRPCUrl = []string{}
)

func NewBlzStream() *GrpcStream {
	GRPCUrl = strings.Split(os.Getenv("BLZ_GRPC_URLS"), ",")
	var conns []*grpc.ClientConn
	for _, url := range GRPCUrl {
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
