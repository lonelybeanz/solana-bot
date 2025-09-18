package parser

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	ClmmSwapTxHash         = "28qhhZd9rVwjc6hB8mv5fVmuuJ1xjFAha5AbD3v3Hev5GmqiHquAHNR1E3cVTAJfizJ4AEhCJ71S8Gc3fazP6X8v"
	OutSellPumpFunTxHash   = "5sroYxPV2KXNHHogPuP2kUuFHjonpcZNfYw83gtineFBt9ZVzwYViwseiev5dxgZXcg9mddBavmmGkYfugw7DCHN"
	OutBuyPumpFunTxHash    = "2QAvDuz2eMxbWyFQgG42dKYCTt3c7zhj9JiVVSL4uSb8QN4DYSFmjjhPBZ94epLjU6AdQdkg5NczTn4jeWEjDybP"
	InnerSellPumpFunTxHash = ""
	InnerBuyPumpFunTxHash  = "5zrZnZa1bNawuJofcdPUu7ZnHF13xTuyeixoVS8Ev8MmfVZtZ5kNmxaSaiB9URxp57WAwzSV9zuma9KD5eHcxyvU"
	ClmmAndAmmTxHash       = "5x3wqVsfh9VapEDeT5Zbh5o7ZC35s9swVmkVYK34bRatQSazDD4REiLZTZ92Ge5ShqUaxJyHrUFuiwxDzbRcsWug"
	DammV2TxHash           = "ak5UqXUkpxyMKa4HUQmSrWUhmkezt3ESoPUyH547kceUceTborJGiqSQ6tBGQMg8QmWUvPRS6tyXgcAxcpwh5n9"
)

var (
	testParser *SolParser
	testRpc    *rpc.Client
)

func Before(t *testing.T) {
	//testRpc = rpc.New("https://api.mainnet-beta.solana.com")
	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 15 * time.Second,
	}
	cluster := rpc.MainNetBeta
	testRpc = rpc.NewWithCustomRPCClient(jsonrpc.NewClientWithOpts(
		cluster.RPC,
		&jsonrpc.RPCClientOpts{
			HTTPClient:    httpClient,
			CustomHeaders: map[string]string{},
		},
	))
	testParser = &SolParser{cli: testRpc}
}

func TestSolParser_ParseSwapEvent(t *testing.T) {
	Before(t)
	intOne := uint64(1)
	intPtr := &intOne
	ctx := context.Background()
	opts := &rpc.GetParsedTransactionOpts{MaxSupportedTransactionVersion: intPtr,
		Commitment: rpc.CommitmentConfirmed}
	sig := solana.MustSignatureFromBase58(DammV2TxHash)
	p, err := testRpc.GetParsedTransaction(ctx, sig, opts)
	if err != nil {
		t.Error(err)
	}
	z, err := testParser.ParseSwapEvent(p)
	if err != nil {
		t.Error(err)
	}
	for i, d := range z {
		t.Logf("Swap Event %d %d:\n", i, d.EventIndex)
		t.Logf("  Pool: %s\n", d.PoolAddress)
		t.Logf("  Market: %s\n", d.MarketProgramId)
		t.Logf("  Input Token: %s Amount: %s\n", d.InToken.Code, d.InToken.Amount)
		t.Logf("  Output Token: %s Amount: %s\n", d.OutToken.Code, d.OutToken.Amount)
	}

}
