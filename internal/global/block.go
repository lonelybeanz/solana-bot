package global

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"solana-bot/internal/config"
	"solana-bot/internal/pb/feepb"
	"solana-bot/internal/stream"

	"solana-bot/internal/global/utils"
	atomic_ "solana-bot/internal/global/utils/atomic"
	"strings"
	"sync"
	"time"

	pb "github.com/lonelybeanz/solanaswap-go/yellowstone-grpc"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

var (
	BlockChan      = make(chan []byte, 4096)
	blockMutex     = &sync.RWMutex{}
	updateMutex    = &sync.Mutex{}
	currentBlock   = &atomic_.Uint64{}
	block          = &CurrentBlock{}
	gasAPIMap      = gasPrio{}
	gasAPI         = "https://mainnet.helius-rpc.com/?api-key=61c2b0a0-3d9b-499b-9f56-ad064bbc7311"
	SolATA         atomic_.Bool
	SolATA_Balance atomic_.BigInt
	Sol_Balance    atomic_.Uint64
)

type gasPrio struct {
	VeryHigh uint64
	High     uint64
	Medium   uint64
	Low      uint64
	mu       sync.RWMutex
}

type CurrentBlock struct {
	PrevBlockHash solana.Hash
	BlockNum      uint64
	LastTime      int64
	Time          int64
	Slot          uint64
}

type Clock struct {
	Slot                uint64
	EpochStartTimestamp int64
	Epoch               uint64
	LeaderScheduleEpoch uint64
	UnixTimestamp       int64
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
	go stream.Grpc_subscribe(conn, &subscription, context.Background(), subscribe)

	for msg := range subscribe {
		got := msg.(*pb.SubscribeUpdate)
		//打印
		// spew.Dump(got)

		updateBlock := got.GetBlockMeta()

		blockMutex.Lock()
		block.BlockNum = updateBlock.Slot
		block.Slot = updateBlock.Slot
		block.LastTime = updateBlock.BlockTime.Timestamp
		block.Time = updateBlock.GetBlockTime().Timestamp
		block.PrevBlockHash = solana.MustHashFromBase58(updateBlock.GetBlockhash())
		currentBlock.Store(updateBlock.Slot)
		blockMutex.Unlock()

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
	go stream.Grpc_subscribe(conn, &subscription, context.Background(), subscribe)

	for msg := range subscribe {
		got := msg.(*pb.SubscribeUpdate)
		// spew.Dump(got)
		updateBlock := got.GetBlockMeta()
		if updateBlock == nil {
			continue
		}

		blockMutex.Lock()
		block.BlockNum = updateBlock.Slot
		block.Slot = updateBlock.Slot
		block.LastTime = updateBlock.BlockTime.Timestamp
		block.Time = updateBlock.GetBlockTime().Timestamp
		block.PrevBlockHash = solana.MustHashFromBase58(updateBlock.GetBlockhash())
		currentBlock.Store(updateBlock.Slot)
		blockMutex.Unlock()

	}
}

func WSOLSubscribeWithRelay(conn *grpc.ClientConn) {

	bal, _ := GetTokenBalance(
		solana.MustPublicKeyFromBase58(config.C.Bot.Player),
		solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"),
	)
	if bal == nil || bal.Uint64() == 0 {
		SolATA_Balance.Store(big.NewInt(0))
		SolATA.Store(false)
	} else {
		SolATA_Balance.Store(big.NewInt(int64(bal.Uint64())))
		SolATA.Store(true)
	}

	balF := GetBalanceByPublic(config.C.Bot.Player)
	logx.Infof("获取 SOL 余额成功: %v \n", balF)
	if bal != nil {
		balU, _ := balF.Uint64()
		Sol_Balance.Store(balU)
	}

	subscribe := make(chan interface{})
	var subscription pb.SubscribeRequest
	commitment := pb.CommitmentLevel_PROCESSED
	subscription.Commitment = &commitment
	subscription.Accounts = make(map[string]*pb.SubscribeRequestFilterAccounts)
	subscription.Accounts["account_sub"] = &pb.SubscribeRequestFilterAccounts{}
	subscription.Accounts["account_sub"].Account = []string{"7GYJkSBgbrSysKMjVP8AyfD52Pe2oVNtqs66g6ynWaWb"}
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
		wsolToken, err := utils.TokenAccountFromData(accountData)
		if err != nil {
			log.Printf("解析 TokenAccount 数据失败: %v \n", err)
			continue
		}

		amount := int64(wsolToken.Amount)
		if amount > 0 {
			SolATA_Balance.Store(big.NewInt(amount)) // 可优化复用
			SolATA.Store(true)
		} else {
			SolATA_Balance.Store(big.NewInt(0))
			SolATA.Store(false)
		}

		go func() {
			bal := GetBalanceByPublic(config.C.Bot.Player)
			logx.Infof("获取 SOL 余额成功: %v \n", bal)
			if bal != nil {
				balU, _ := bal.Uint64()
				Sol_Balance.Store(balU)
			}
		}()

	}
}

func BlockSubscribe() {
	rpcClient, wsClient := GetWSRPCForRequest()
	sub, err := wsClient.SlotSubscribe()
	if err != nil {
		log.Println("Failed to subscribe to new blocks", err)
	} else {
		fmt.Println("Subbed to new blocks")
	}
	for {
		rec, err := sub.Recv(context.Background())
		if err != nil {
			fmt.Println("Error block sub", err)
			break
		}
		go UpdateBlock(rpcClient, rec.Slot)
		// go UpdateTokenPrices()
		go updateGas()
	}

	time.Sleep(1 * time.Second)
	fmt.Println("Attempting to reconnect block sub")
	go BlockSubscribe()
}

func UpdateBlock(rpcClient *rpc.Client, slot uint64) {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	clock := Clock{}
	ctx, exp := context.WithTimeout(context.Background(), 1*time.Second)
	defer exp()
	if currentBlock.Load() < slot {
		var recent *rpc.GetLatestBlockhashResult
		var err error
		maxRetries := 3
		retryCount := 0

		for retryCount < maxRetries {
			recent, err = rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
			if err == nil {
				break
			}

			if strings.Contains(err.Error(), "-32429") {
				rpcClient = GetRPCForRequest()
			}

			retryCount++
			if retryCount < maxRetries {
				time.Sleep(100 * time.Millisecond) // 等待100毫秒后重试
			}
		}

		if err != nil {
			log.Println("Failed grabbing new slot block hash after retries:", err)
			return
		}

		blockMutex.Lock()
		block.BlockNum = slot
		block.Slot = slot
		block.LastTime = clock.UnixTimestamp
		block.Time = clock.UnixTimestamp
		block.PrevBlockHash = recent.Value.Blockhash
		currentBlock.Store(slot)
		blockMutex.Unlock()

		// BlockChan <- []byte(fmt.Sprintf("%d", slot))
		// fmt.Printf("Updated block hash,slot:%d \n", slot)
	}
}

func GetBlockHash() solana.Hash {
	blockMutex.RLock()
	defer blockMutex.RUnlock()
	return block.PrevBlockHash
}

func GetSlot() uint64 {
	blockMutex.RLock()
	defer blockMutex.RUnlock()
	return block.Slot
}

func UpdateGasWithRelay() {
	subscribe := make(chan interface{})
	go stream.Fee_subscribe(context.Background(), subscribe)

	for msg := range subscribe {
		got := msg.(*feepb.TransactionFeeStreamResponse)
		// spew.Dump(got)
		fee := got.GetPriorityFee()
		tip := got.GetTip()
		tipUint := uint64(tip.Value * 1e99)
		if fee == nil {
			continue
		}
		gasAPIMap.mu.Lock()
		gasAPIMap.Medium = uint64(fee.Value)
		gasAPIMap.High = tipUint
		gasAPIMap.mu.Unlock()

	}
}

func updateGas() {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(gasAPI)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")

	reqBody := []byte(`{
		"jsonrpc": "2.0",
		"id": "1",
		"method": "getPriorityFeeEstimate",
		"params": [{
			"accountKeys": ["JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4"],
			"options": {
				"includeAllPriorityFeeLevels": true
			}
		}]
	}`)

	req.SetBody(reqBody)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.Do(req, resp); err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	respBody := resp.Body()
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Printf("Error decoding JSON: %s\n", err)
		return
	}

	// Assuming `result` is of type `map[string]interface{}`
	if result != nil {
		resultMap, ok := result["result"].(map[string]interface{})
		if ok {
			priorityFeeLevels, ok := resultMap["priorityFeeLevels"].(map[string]interface{})
			if ok {
				for level, fee := range priorityFeeLevels {
					switch level {
					case "low":
						gasAPIMap.mu.Lock()
						gasAPIMap.Low = uint64(fee.(float64))
						gasAPIMap.mu.Unlock()
					case "medium":
						gasAPIMap.mu.Lock()
						gasAPIMap.Medium = uint64(fee.(float64))
						gasAPIMap.mu.Unlock()
					case "high":
						gasAPIMap.mu.Lock()
						gasAPIMap.High = uint64(fee.(float64))
						gasAPIMap.mu.Unlock()
					case "veryHigh":
						gasAPIMap.mu.Lock()
						gasAPIMap.VeryHigh = uint64(fee.(float64))
						gasAPIMap.mu.Unlock()
					}
				}
			}
		}
	}
}

func GetLow() uint64 {
	gasAPIMap.mu.RLock()
	defer gasAPIMap.mu.RUnlock()
	return gasAPIMap.Low
}

func GetMedium() uint64 {
	gasAPIMap.mu.RLock()
	defer gasAPIMap.mu.RUnlock()
	return gasAPIMap.Medium
}

func GetHigh() uint64 {
	gasAPIMap.mu.RLock()
	defer gasAPIMap.mu.RUnlock()
	return gasAPIMap.High
}

func GetVeryHigh() uint64 {
	gasAPIMap.mu.RLock()
	defer gasAPIMap.mu.RUnlock()
	return gasAPIMap.VeryHigh
}
