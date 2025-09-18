package global

import (
	"context"
	"errors"
	"fmt"
	"log"
	"solana-bot/internal/stream"
	"sync"
	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

var (
	NonceAccount = []string{
		"DouUTgawHYhmxbU8UghapgBtvBWZmjg9NzBMxazAZKPh",
		"HJuxZbBwuoDsvHTZ1c3Xqws56bLyGEdfqBtnXP9fVxgB",
		"EDKVigjAcfMVxMXRCdpfzUQKgXYxb3MYv7g84MnXBjrU",
	}
	nonceMutex        = &sync.RWMutex{}
	currentNonceIndex = 0
	NonceAccountHash  = make(map[string]solana.Hash)
)

func NonceSubscribeWithRelay(conn *grpc.ClientConn) {

	go func() {

		for {
			for _, n := range NonceAccount {
				go syncNonceAccount(n)
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
	subscription.Accounts["account_sub"].Account = NonceAccount
	go stream.Grpc_subscribe(conn, &subscription, context.Background(), subscribe)

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

		nonceMutex.Lock()
		NonceAccountHash[string(accountSub.Account.GetPubkey())] = hash
		nonceMutex.Unlock()
	}
}

func syncNonceAccount(nonceAddress string) {
	nonceAccount := solana.MustPublicKeyFromBase58(nonceAddress)
	//首次赋值
	account, err := GetRPCForRequest().GetAccountInfo(context.Background(), nonceAccount)
	if err != nil {
		logx.Must(err)
	}
	if account == nil {
		logx.Must(errors.New("nonce account not found"))
	}

	dec := bin.NewBinDecoder(account.Value.Data.GetBinary()) //.Decode(&nonceAccountData)
	acc := new(system.NonceAccount)

	err = acc.UnmarshalWithDecoder(dec)
	if err != nil {
		logx.Must(fmt.Errorf("solana.NewBinDecoder() => %v", err))
	}
	hash := solana.Hash(acc.Nonce)
	nonceMutex.Lock()
	NonceAccountHash[nonceAddress] = hash
	nonceMutex.Unlock()
}

func GetNonceAccountAndHash() (solana.PublicKey, solana.Hash) {
	nonceMutex.RLock()
	defer nonceMutex.RUnlock()

	// 轮流获取nonce account
	index := currentNonceIndex
	currentNonceIndex = (currentNonceIndex + 1) % len(NonceAccount)
	logx.Infof("[%s]nonce account hash: %s", NonceAccount[index], NonceAccountHash[NonceAccount[index]])
	nonceAddress := NonceAccount[index]
	nonceAccount := solana.MustPublicKeyFromBase58(nonceAddress)

	return nonceAccount, NonceAccountHash[NonceAccount[index]]

}
