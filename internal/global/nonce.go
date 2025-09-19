package global

import (
	"context"
	"errors"
	"fmt"
	"sync"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/zeromicro/go-zero/core/logx"
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

func SyncNonceAccount(nonceAddress string) {
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

func UpdataNonceAccountHash(nonceAddress string, nonceHash solana.Hash) {
	nonceMutex.Lock()
	NonceAccountHash[nonceAddress] = nonceHash
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
