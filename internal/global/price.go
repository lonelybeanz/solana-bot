package global

import (
	"encoding/json"
	"fmt"
	"math/big"

	"strconv"
	"sync"

	"github.com/valyala/fasthttp"
)

const (
	tolerance       = 0
	stableTolerance = 0
)

var (
	priceBlock    uint64
	currentPrice  float64
	currentPriceF *big.Float

	priceMutex       = &sync.RWMutex{}
	tokenpriceMutex  = &sync.RWMutex{}
	tokenprice       = make(map[string]float64)
	tokenpriceupdate = make(map[string]uint64)
)

type PriceQuoteTokenData struct {
	ID            string  `json:"id"`
	MintSymbol    string  `json:"mintSymbol"`
	VsToken       string  `json:"vsToken"`
	VsTokenSymbol string  `json:"vsTokenSymbol"`
	Price         float64 `json:"price"`
}

type JupiterPrice struct {
	Data map[string]struct {
		ID    string `json:"id"`
		Price string `json:"price"`
	} `json:"data"`
}

type PriceQuote struct {
	Data      map[string]PriceQuoteTokenData `json:"data"`
	TimeTaken float64                        `json:"timeTaken"`
}

type CryptoPrices struct {
	Dai      CryptoPrice `json:"dai"`
	Ethereum CryptoPrice `json:"ethereum"`
	Tether   CryptoPrice `json:"tether"`
	UsdCoin  CryptoPrice `json:"usd-coin"`
	Solana   CryptoPrice `json:"solana"`
}

type CryptoPrice struct {
	Usd float64 `json:"usd"`
}

func getCurrentBlockFromAPI() uint64 {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("")

	err := fasthttp.Do(req, resp)
	if err != nil {
		//Log.Printf("Error fetching block: %v", err)
		return 0
	}

	var block uint64
	fmt.Sscanf(string(resp.Body()), "%d", &block)
	return block
}

func getPriceFromAPI() float64 {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("")

	err := fasthttp.Do(req, resp)
	if err != nil {
		fmt.Println("Error fetching price:", err)
		return 0
	}

	var price float64
	fmt.Sscanf(string(resp.Body()), "%f", &price)
	return price
}

func GetCurrentPrice() float64 {
	priceMutex.RLock()
	defer priceMutex.RUnlock()
	return currentPrice
}

func GetTokenPriceByString(token string) float64 {
	if token == "" {
		fmt.Println("Invalid Active Token to Fetch Price.")
		return 0
	}
	if price, exists := tokenprice[token]; exists && tokenpriceupdate[token] >= GetSlot()-3 {
		return price
	}
	return UpdateTokenPrice(token)
}

func UpdateTokenPriceBySub(token string, price float64) {
	fmt.Printf("Updated Token Price for %s to %0.10f\n", token, price)
	tokenpriceMutex.Lock()
	tokenprice[token] = price
	tokenpriceupdate[token] = GetSlot()
	tokenpriceMutex.Unlock()
}

func UpdateTokenPrices() {
	for token := range tokenprice {
		go UpdateTokenPrice(token)
	}
}

func UpdateTokenPrice(token string) float64 {
	url := fmt.Sprintf("https://api.jup.ag/price/v2?ids=%s,%s&vsToken=%s", token, Solana, Solana)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)

	err := fasthttp.Do(req, resp)
	if err != nil {
		fmt.Println("Token price fetch failed for ", err)
		return -1
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return -1
	}

	body := resp.Body()

	quote := JupiterPrice{}
	err = json.Unmarshal(body, &quote)
	if err != nil {
		fmt.Println("Token price unmarshal failed for ", err)
		return -1
	}

	price, _ := strconv.ParseFloat(quote.Data[token].Price, 64)
	tokenpriceMutex.Lock()
	tokenprice[token] = price
	tokenpriceupdate[token] = GetSlot()
	tokenpriceMutex.Unlock()

	return price
}

func GetCurrentPriceF() *big.Float {
	priceMutex.RLock()
	defer priceMutex.RUnlock()
	return currentPriceF
}
