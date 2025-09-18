package stream

import (
	"google.golang.org/grpc"
)

func NewTritonStream() *GrpcStream {
	var (
		GRPCUrl = []string{
			"api.rpcpool.com:443",
		}
		xtoken = "your x-token"
	)

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
		Xtoken: xtoken,
	}
}
