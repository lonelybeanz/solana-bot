package pump

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"solana-bot/internal/global"

	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestGetAddres(t *testing.T) {
	global.ConnectToEndpoints()
	o, err := GetPumpBondingCurveDataIfPoolExists(global.GetRPCForRequest(), "DCrwhb7f6LL4dtvsTty1PwT2wL5AoxHzuwYMUqM5pump")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(o)
	p, _, _ := GetPriceAndLiquidityAndDexFromPump(o)
	t.Log(p)

	mcap := new(big.Float).Mul(new(big.Float).SetUint64(1e9), p)

	mcap = new(big.Float).Quo(mcap, big.NewFloat(1e4))

	t.Log(mcap)
}

func TestGas(t *testing.T) {
	global.ConnectToEndpoints()
	go global.BlockSubscribe()

	o := global.GetBalanceByPublic("7vNSg7vPnQDgg7APyAnvYCMW9xH3fNtDwxwhHQTsTmtU")
	fmt.Println(o)
	time.Sleep(1 * time.Hour)
}

func TestAccount(t *testing.T) {
	global.ConnectToEndpoints()
	o, _ := global.GetProgramAccounts("9eQycfJwEv2RYiCxXTTXxPBPugaHmPrU1eL9Uhgcpump")
	fmt.Println(o)
}

func TestTop(t *testing.T) {
	global.ConnectToEndpoints()
	o, _ := global.GetTopHolders(global.GetRPCForRequest(), "DCrwhb7f6LL4dtvsTty1PwT2wL5AoxHzuwYMUqM5pump")
	fmt.Println(o)
}

func TestPrice(t *testing.T) {
	global.ConnectToEndpoints()
	go global.BlockSubscribe()
	for {
		p := global.GetTokenPriceByString("8ZjdK7YKqkBv9wPc2i4MitcvdVKNr6YBjRjENjUSpump")
		mcap := new(big.Float).Mul(new(big.Float).SetUint64(1e9), big.NewFloat(p))
		fmt.Printf("市值: %.2f\n", mcap)
		f, _ := mcap.Float64()
		t.Logf("市值:%f", f)
		time.Sleep(1 * time.Second)
	}

}

func TestSubUpdatePrice(t *testing.T) {
	global.ConnectToEndpoints()
	go global.BlockSubscribe()
	time.Sleep(5 * time.Second)
}

func TestParse(t *testing.T) {
	decodedData, err := base64.StdEncoding.DecodeString("vdt/007mYe5SG4uB1u3SvpWW6pGhgPXsacdUMxAGmrn2IP07Yt1u/wCzP3EAAAAAolup/h86AAABStz3tma2JvBp30mCmjFgUtzF9L4iyK1Tm9W5VBdj+42uEu5nAAAAAABfY20HAAAAXrQuScOVAwAAsz9xAAAAAF4cHP0xlwIA")
	if err != nil {
		log.Fatal("解码失败:", err)
	}
	fmt.Println("解码后的数据:", decodedData)
}

func TestBanlance(t *testing.T) {
	global.ConnectToEndpoints()
	wallet := solana.MustPublicKeyFromBase58("8LVspLb436sBbhyPUFM3oMv6efFWHmfHbjpxNCzHzsgo")
	token := solana.MustPublicKeyFromBase58("DGbwpEn7QvYFWpVGqtXeSbvWs2tXBoutvH5SKtoKpump")
	w, f := GetTokenBalance(global.GetRPCForRequest(), wallet, token)
	fmt.Println("钱包余额:", w)
	fmt.Println("手续费:", f)
}

func TestPoll(t *testing.T) {
	global.ConnectToEndpoints()
	pool_base_token_account := solana.MustPublicKeyFromBase58("2VA14keuc1Z97LUo8uSeMiDR7Nzixy48pq3BSeCZ8SbR")
	pool_quote_token_account := solana.MustPublicKeyFromBase58("8CRGR3ynXTw3Dd3MxaM1xyxC653PYQTyVD22vGtX9UEA")
	balance_base, balance_quote := Async_get_pool_reserves(*global.GetRPCForRequest(), pool_base_token_account, pool_quote_token_account)
	t.Log(balance_base)
	t.Log(balance_quote)

	out, err := global.GetRPCForRequest().GetTokenAccountBalance(context.Background(), pool_base_token_account, rpc.CommitmentConfirmed)
	if err != nil {
		t.Error(err)
	}
	t.Log(out)
}

func TestLi(t *testing.T) {

	amountInAfterOurFee := big.NewInt(1e8)
	slippage := float32(0.5)

	pumpFee := pumpGetFee(amountInAfterOurFee, FeeBasisPoints)

	// this is used for quoting
	amountInAfterPumpFee := new(big.Int).Sub(amountInAfterOurFee, pumpFee)
	amountOut := pumpQuoteBuy(amountInAfterPumpFee, &PUMPBondingCurveData{
		BondingCurve: &BondingCurveLayout{
			RealSOLReserves:      990099009,
			RealTokenReserves:    758818849870455,
			VirtualTokenReserves: 1038718849870455,
			VirtualSOLReserves:   30990099009,
		},
	})
	amountOutWithSlippage := applySlippage(amountOut, slippage)
	t.Log(amountInAfterPumpFee)
	t.Log(amountOutWithSlippage)
}
