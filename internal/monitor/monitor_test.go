package monitor

import (
	"context"
	"fmt"
	"log"
	"solana-bot/internal/client"
	"solana-bot/internal/global"
	"time"

	solanaswapgo "github.com/lonelybeanz/solanaswap-go/solanaswap-go"

	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestPump(t *testing.T) {
	p, err := NewPumpFunMonitor()
	if err != nil {
		t.Fatal(err)
	}
	p.Start()
	// p.listenSmartWallet()

}

func TestBurnToken(t *testing.T) {
	p, err := NewPumpFunMonitor()
	if err != nil {
		t.Fatal(err)
	}
	p.BurnToken("BM5oY7JPjpnr7ennNWVaQU8YhdCaHPAQK52r3xSRbonk")
}

func TestParse(t *testing.T) {
	price := float64(0.000000034681380)
	fmt.Printf("%.15f", price*1.5)
}

func TestSell(t *testing.T) {
	_, err := NewPumpFunMonitor()
	if err != nil {
		t.Fatal(err)
	}
	rpcClient, wsClient := global.GetWSRPCForRequest()
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	resp, err := client.WaitForTransaction(ctx, rpcClient, wsClient, "bouUCybKxRVr4xeyfDLSYBHhn77KBYpvNev4FCT8HdZtM1nSZ55aGuaSS5UzdU4NAPUa4BSpYgLCSS79JCvxvj9")
	if err != nil || resp == nil || resp.Meta.Err != nil {

		t.Fatal(resp.Meta.Err)
	} else {
		//设定限价单

		buyPrice, amount := global.GetBalacneChange(resp) //global.GetBuyPriceAndAmount(resp, "4z8cxysomktixueMXL8cc9iN8x6s67sUHToLrQeWbonk")
		// buyPrice = buyPrice * utils.GetTokenPriceByString(global.Solana)
		t.Logf("%.15f", buyPrice)
		t.Logf("%v", amount)

		// r.SetLimitOrder("Agwr8ZGQveACuiPFgPEisUda3cXwNfCuHgcZbeZApump", buyPrice, amount)

	}

}

func TestGetPric(t *testing.T) {
	rpcClient := rpc.New(client.HTTPUrl)
	out, err := client.GetTransactionByHash(rpcClient, "5xqzpFoUnXsG2TzhYQusNXZZr8TopF9sWL8CQHqKhbXTS4z2GAnCPXZzbmhZv494SncGKLG29gtndrYHrdDmdgPs")
	if err != nil {
		t.Fatal(err)
	}
	f, a := global.GetBuyPriceAndAmount(out, "2yUbnKS9cfouvVySbmMpe4kSvyMLx3Da3MkYRTJXpump")
	t.Logf("%.10f", f)
	t.Logf("%.10f", a)
}

func TestDevSell(t *testing.T) {
	t.Log(calculateDynamicBuyAmount(4e9))

	t.Log(calculateHoldDuration(5, PositiveCurve))

	t.Log(calculateHoldDuration(5, NegativeCurve))

}

func TestSmartBuy(t *testing.T) {
	_, err := NewPumpFunMonitor()
	if err != nil {
		t.Fatal(err)
	}

	rpcClient := rpc.New(rpc.MainNetBeta.RPC)

	txSig := solana.MustSignatureFromBase58("63pVvtCrvE44FPLaRQApJQ4ZZGEAme58dwctTRLhBcpVA6EgcKp5F5MwNxkZcM2EnC1ywh3G3xyV1K5NAqXYs5x7")

	// Specify the maximum transaction version supported
	var maxTxVersion uint64 = 0

	// Fetch the transaction data using the RPC client
	tx, err := rpcClient.GetTransaction(
		context.TODO(),
		txSig,
		&rpc.GetTransactionOpts{
			Commitment:                     rpc.CommitmentConfirmed,
			MaxSupportedTransactionVersion: &maxTxVersion,
		},
	)
	if err != nil {
		log.Fatalf("Error fetching transaction: %s", err)
	}

	// Initialize the transaction parser
	parser, err := solanaswapgo.NewTransactionParser(tx)
	if err != nil {
		log.Fatalf("Error initializing transaction parser: %s", err)
	}

	// Parse the transaction to extract basic data
	transactionData, err := parser.ParseTransactionForSwap()
	if err != nil {
		log.Fatalf("Error parsing transaction: %s", err)
	}

	// Print the parsed transaction data
	// marshalledData, _ := json.MarshalIndent(transactionData, "", "  ")
	// // fmt.Println(string(marshalledData))

	// Process and extract swap-specific data from the parsed transaction
	swapData, err := parser.ProcessSwapData(transactionData)
	if err != nil {
		log.Fatalf("Error processing swap data: %s", err)
	}

	t.Log(swapData)
}
