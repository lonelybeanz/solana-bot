package global

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	atomic_ "solana-bot/internal/global/utils/atomic"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/valyala/fasthttp"
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
}

func UpdateBlock(rpcClient *rpc.Client, slot uint64) {
	updateMutex.Lock()
	defer updateMutex.Unlock()
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

		UpdateBlockData(slot, recent.Value.Blockhash)

		// BlockChan <- []byte(fmt.Sprintf("%d", slot))
		// fmt.Printf("Updated block hash,slot:%d \n", slot)
	}
}

func UpdateBlockData(slot uint64, prevBlockHash solana.Hash) {
	blockMutex.Lock()
	block.BlockNum = slot
	block.Slot = slot
	block.PrevBlockHash = prevBlockHash
	currentBlock.Store(slot)
	blockMutex.Unlock()
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

func UpdateGasData(low, medium, high, veryHigh uint64) {
	gasAPIMap.mu.Lock()
	gasAPIMap.Low = low
	gasAPIMap.Medium = medium
	gasAPIMap.High = high
	gasAPIMap.VeryHigh = veryHigh
	gasAPIMap.mu.Unlock()
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
