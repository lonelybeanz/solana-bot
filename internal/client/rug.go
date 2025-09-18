package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

/*
		{
	    "mint": "9DVNHn7haa3ssQgBK1bgVKZkQ7myV8aDdPywXQsipump",
	    "tokenProgram": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	    "creator": "A1wRAMAQLuCfAQNUjnAa2Q4Ci5CQv2fGmhtSooGvBAaW",
	    "creatorBalance": 66439628457730,
	    "token": {
	        "mintAuthority": null,
	        "supply": 1000000000000000,
	        "decimals": 6,
	        "isInitialized": true,
	        "freezeAuthority": null
	    },
	    "token_extensions": null,
	    "tokenMeta": {
	        "name": "MEMELON",
	        "symbol": "MEMELON",
	        "uri": "https://ipfs.io/ipfs/QmQ4wUWytoxQAJiWbWSwpT1pynFy14jgUrC3yXYSm5Y65y",
	        "mutable": false,
	        "updateAuthority": "TSLvdd1pWpHVjahSpsvCXUbgwsL3JAcvokwaKt1eokM"
	    },
	    "topHolders": null,
	    "freezeAuthority": null,
	    "mintAuthority": null,
	    "risks": [
	        {
	            "name": "Creator history of rugged tokens",
	            "value": "",
	            "description": "Creator has a history of rugging tokens.",
	            "score": 16800,
	            "level": "danger"
	        }
	    ],
	    "score": 16801,
	    "score_normalised": 57,
	    "fileMeta": {
	        "description": "",
	        "name": "MEMELON",
	        "symbol": "MEMELON",
	        "image": "https://ipfs.io/ipfs/QmT3Ex8DNRs3XS3xGXxZCHPB4GqQ7jLMkeT7dmr2Jnuwm9"
	    },
	    "lockerOwners": {},
	    "lockers": {},
	    "markets": [
	        {
	            "pubkey": "4Qwh8JREqkeuVErzhASUYVAJ1MDVex5bZc4xyUtzc7Mq",
	            "marketType": "pump_fun",
	            "mintA": "9DVNHn7haa3ssQgBK1bgVKZkQ7myV8aDdPywXQsipump",
	            "mintB": "So11111111111111111111111111111111111111112",
	            "mintLP": "11111111111111111111111111111111",
	            "liquidityA": "9CRwKhhM6torvPcdhYRCJ1i758CZpLjtgNhWqJbi5m1h",
	            "liquidityB": "4Qwh8JREqkeuVErzhASUYVAJ1MDVex5bZc4xyUtzc7Mq",
	            "mintAAccount": {
	                "mintAuthority": null,
	                "supply": 1000000000000000,
	                "decimals": 6,
	                "isInitialized": true,
	                "freezeAuthority": null
	            },
	            "mintBAccount": {
	                "mintAuthority": null,
	                "supply": 0,
	                "decimals": 9,
	                "isInitialized": false,
	                "freezeAuthority": null
	            },
	            "mintLPAccount": {
	                "mintAuthority": null,
	                "supply": 0,
	                "decimals": 0,
	                "isInitialized": false,
	                "freezeAuthority": null
	            },
	            "liquidityAAccount": {
	                "mint": "9DVNHn7haa3ssQgBK1bgVKZkQ7myV8aDdPywXQsipump",
	                "owner": "4Qwh8JREqkeuVErzhASUYVAJ1MDVex5bZc4xyUtzc7Mq",
	                "amount": 695031067816913,
	                "delegate": null,
	                "state": 0,
	                "delegatedAmount": 0,
	                "closeAuthority": null
	            },
	            "liquidityBAccount": {
	                "mint": "So11111111111111111111111111111111111111112",
	                "owner": "4Qwh8JREqkeuVErzhASUYVAJ1MDVex5bZc4xyUtzc7Mq",
	                "amount": 3017718959,
	                "delegate": null,
	                "state": 0,
	                "delegatedAmount": 0,
	                "closeAuthority": null
	            },
	            "lp": {
	                "baseMint": "9DVNHn7haa3ssQgBK1bgVKZkQ7myV8aDdPywXQsipump",
	                "quoteMint": "So11111111111111111111111111111111111111112",
	                "lpMint": "11111111111111111111111111111111",
	                "quotePrice": 148.10680444651302,
	                "basePrice": 0.00000484881442986372,
	                "base": 695031067.816913,
	                "quote": 3.017718959,
	                "reserveSupply": 3017718959,
	                "currentSupply": 0,
	                "quoteUSD": 446.9447117351479,
	                "baseUSD": 3370.0766708342376,
	                "pctReserve": 0,
	                "pctSupply": 0,
	                "holders": null,
	                "totalTokensUnlocked": 0,
	                "tokenSupply": 0,
	                "lpLocked": 3017718959,
	                "lpUnlocked": 0,
	                "lpLockedPct": 100,
	                "lpLockedUSD": 3817.0213825693854,
	                "lpMaxSupply": 0,
	                "lpCurrentSupply": 0,
	                "lpTotalSupply": 3017718959
	            }
	        }
	    ],
	    "totalMarketLiquidity": 3817.0213825693854,
	    "totalLPProviders": 0,
	    "totalHolders": 2,
	    "price": 0.00000484881442986372,
	    "rugged": false,
	    "tokenType": "",
	    "transferFee": {
	        "pct": 0,
	        "maxAmount": 0,
	        "authority": "11111111111111111111111111111111"
	    },
	    "knownAccounts": {
	        "4Qwh8JREqkeuVErzhASUYVAJ1MDVex5bZc4xyUtzc7Mq": {
	            "name": "Pump Fun",
	            "type": "AMM"
	        },
	        "9CRwKhhM6torvPcdhYRCJ1i758CZpLjtgNhWqJbi5m1h": {
	            "name": "Pump Fun",
	            "type": "AMM"
	        },
	        "A1wRAMAQLuCfAQNUjnAa2Q4Ci5CQv2fGmhtSooGvBAaW": {
	            "name": "Creator",
	            "type": "CREATOR"
	        }
	    },
	    "events": [],
	    "verification": null,
	    "graphInsidersDetected": 0,
	    "insiderNetworks": null,
	    "detectedAt": "2025-04-28T22:52:49.138449156Z",
	    "creatorTokens": [
	        {
	            "mint": "4UxdHnvHsCLywjknprfji1VqYw748gyoDVJMvV7Vpump",
	            "marketCap": 3796.6389476805393,
	            "createdAt": "2025-03-14T18:55:58.098286145Z"
	        },
	        {
	            "mint": "26dyCsUVcz7HysQsZq9SYFhQ5UNDfAQQeMzc4a4upump",
	            "marketCap": 3745.5426805528978,
	            "createdAt": "2025-03-14T18:53:49.566823877Z"
	        },
	        {
	            "mint": "4CEdnFGcZuYJDsZP3SYLb4dXZjrHr21DcSXYPJ1tpump",
	            "marketCap": 3580.335491333126,
	            "createdAt": "2025-03-14T07:59:25.583826389Z"
	        },
	        {
	            "mint": "2F6boWvbdxMNu3S8TNBUK6rVHvYGBpfkUDM11E14pump",
	            "marketCap": 3501.4105398862926,
	            "createdAt": "2025-03-14T07:55:32.14103927Z"
	        },
	        {
	            "mint": "Usvm2NeqoEBPV69XwFhLmjCh298AXc9iGv3EtV2pump",
	            "marketCap": 3536.94030951497,
	            "createdAt": "2025-03-14T03:57:08.530523139Z"
	        },
	        {
	            "mint": "AMkjVQJpq7aXKqywbQ3sM5fGhN77zcEJm1o6oL4qpump",
	            "marketCap": 3539.1655730993025,
	            "createdAt": "2025-03-14T03:53:56.523092951Z"
	        },
	        {
	            "mint": "9aSH2hGvEKRErJczQW4VigLH74oanL9aUKfr19fZpump",
	            "marketCap": 3495.6952215369706,
	            "createdAt": "2025-03-14T03:22:40.594025005Z"
	        }
	    ]
	}
*/
type RugCheckResponse struct {
	CreatorBalance uint64 `json:"creatorBalance"`
	Token          struct {
		MintAuthority   string  `json:"mintAuthority"`
		Supply          int64   `json:"supply"`
		Decimals        int     `json:"decimals"`
		IsInitialized   bool    `json:"isInitialized"`
		FreezeAuthority *string `json:"freezeAuthority"`
	} `json:"token"`
	Risks []struct {
		Name        string `json:"name"`
		Value       string `json:"value"`
		Description string `json:"description"`
		Score       int    `json:"score"`
		Level       string `json:"level"`
	} `json:"risks"`
	Score                int     `json:"score"`
	TotalMarketLiquidity float64 `json:"totalMarketLiquidity"`
	Price                float64 `json:"price"`
}

func GetRugCheck(token string) (*RugCheckResponse, error) {
	url := fmt.Sprintf("https://api.rugcheck.xyz/v1/tokens/%s/report", token)
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RugCheck API 返回错误: %s", resp.Status)
	}
	// 解析响应体
	var response RugCheckResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	// infoJSON, _ := json.MarshalIndent(response, "", "  ")
	// fmt.Println("信息:", string(infoJSON))

	return &response, nil
}
