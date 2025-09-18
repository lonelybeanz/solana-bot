package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type AiStory struct {
	En string `json:"en"`
	Cn string `json:"cn"`
}

type Token struct {
	TokenName     string  `json:"tokenName"`
	TokenSymbol   string  `json:"tokenSymbol"`
	TokenAddress  string  `json:"tokenAddress"`
	AiStory       string  `json:"aiStory"`
	PushTimes     int     `json:"pushTimes"`
	FirstPriceAt  string  `json:"firstPriceAt"`
	FirstPriceUsd float64 `json:"firstPriceUsd"`
	TokenPriceUsd float64 `json:"tokenPriceUsd"`
}

//	curl 'https://hidex.ai/api/frontend/app/signal/public' \
//	  -H 'accept-language: zh-CN,zh;q=0.9' \
//	  -H 'content-type: application/json' \
//	  -H 'origin: https://hidex.ai' \
//	  --data-raw '{"pageNum":1,"pageSize":3,"chainId":102,"params":{"sortType":"desc","sortField":"updated_at","pinIds":[],"launchpadNames":""},"tokenAddress":""}'
func GetTokenFromHidex() {
	client := &http.Client{}
	body := `{"pageNum":1,"pageSize":3,"chainId":102,"params":{"sortType":"desc","sortField":"updated_at","pinIds":[],"launchpadNames":""},"tokenAddress":""}}`

	req, err := http.NewRequest("POST", "https://hidex.ai/api/frontend/app/signal/public", strings.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://hidex.ai")
	req.Header.Set("referer", "https://hidex.ai/")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var response struct {
		Data struct {
			Rows []Token `json:"rows"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return
	}
	for _, token := range response.Data.Rows {
		println(token.TokenName, token.TokenSymbol, token.TokenAddress)
	}

}

func main() {
	GetTokenFromHidex()
}
