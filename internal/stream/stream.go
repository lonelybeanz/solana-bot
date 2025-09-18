package stream

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io"
	"sync"
	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// StreamMessage 包装来自不同流的消息，添加来源信息
type StreamMessage struct {
	Source string      // 来源标识，通常是连接地址
	Data   interface{} // 原始数据
}

type GrpcStream struct {
	Conns  []*grpc.ClientConn
	Xtoken string
}

func (b *GrpcStream) Subscribe(ctx context.Context, subscription *pb.SubscribeRequest, once *sync.Once, recv chan interface{}) {
	for _, conn := range b.Conns {
		go Grpc_subscribe_once(ctx, conn, b.Xtoken, subscription, once, recv)
	}
}

var kacp = keepalive.ClientParameters{
	Time:                10 * time.Second, // 客户端多久主动 ping 一次
	Timeout:             time.Second,      // ping 超时判定
	PermitWithoutStream: true,             // 即使没有流也允许 ping
}

func Grpc_connect(address string, plaintext bool) *grpc.ClientConn {
	var opts []grpc.DialOption
	if plaintext {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		pool, _ := x509.SystemCertPool()
		creds := credentials.NewClientTLSFromCert(pool, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	opts = append(opts, grpc.WithKeepaliveParams(kacp))

	logx.Infof("Starting grpc client, connecting to %s", address)
	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		logx.Errorf("[%s] fail to dial: %v", address, err)
		return nil
	}

	return conn
}

func Grpc_subscribe_once(ctx context.Context, conn *grpc.ClientConn, xtoken string, subscription *pb.SubscribeRequest, once *sync.Once, recv chan interface{}) {
	var err error
	client := pb.NewGeyserClient(conn)

	subscriptionJson, err := json.Marshal(&subscription)
	if err != nil {
		logx.Errorf("[%s]: Failed to marshal subscription request: %v", conn.CanonicalTarget(), subscriptionJson)
		return
	}

	logx.Infof("[%s]: Subscription request: %s", conn.CanonicalTarget(), string(subscriptionJson))

	// Set up the subscription request
	relayCtx := context.Background()
	md := metadata.New(map[string]string{"x-token": xtoken})
	relayCtx = metadata.NewOutgoingContext(relayCtx, md)
	stream, err := client.Subscribe(relayCtx)
	if err != nil {
		logx.Errorf("[%s]: Subscribe error:%v", conn.CanonicalTarget(), err)
		return
	}
	err = stream.Send(subscription)
	if err != nil {
		logx.Errorf("[%s]: Subscribe Send error:%v", conn.CanonicalTarget(), err)
		return
	}
	logx.Infof("[%s]: start recv ...", conn.CanonicalTarget())
	for {
		select {
		case <-ctx.Done():
			logx.Infof("[%s]: Context done, stopping subscription", conn.CanonicalTarget())
			once.Do(func() {
				close(recv)
			})
			return
		default:
			resp, err := stream.Recv()

			//打印
			// spew.Dump(resp)

			if err == io.EOF {
				logx.Infof("[%s]: Stream closed", conn.CanonicalTarget())
				return
			}
			if err != nil {
				logx.Errorf("[%s]: Error occurred in receiving update: %v", conn.CanonicalTarget(), err)
				continue
			}

			// 包装消息，添加来源信息
			wrappedMsg := &StreamMessage{
				Source: conn.CanonicalTarget(),
				Data:   resp,
			}
			// recv <- wrappedMsg
			if !safeSend(recv, wrappedMsg) {
				logx.Errorf("[%s]: Failed to send message to channel", conn.CanonicalTarget())
			}
		}
	}
}

func safeSend(ch chan interface{}, v *StreamMessage) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	ch <- v
	return true
}
