package types

type TokenAmt struct {
	Code   string `json:"code"`
	Amount string `json:"amount"`
}

type SwapTransactionEvent struct {
	EventIndex      int      `json:"eventIndex"`
	Sender          string   `json:"sender"`
	Receiver        string   `json:"receiver"`
	InToken         TokenAmt `json:"inToken"`
	OutToken        TokenAmt `json:"outToken"`
	PoolAddress     string   `json:"poolAddress"`
	MarketProgramId string   `json:"market"`
}
