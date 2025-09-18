package client

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestRay(t *testing.T) {
	wallet, err := solana.WalletFromPrivateKeyBase58("2rGpKU7MXBWsdrugTyTcRepa25zmEkeYYj4Bjqv68rRXkzgAvh2rcVQfSrAz2G8Bdcv3Ve3nfCftLZPErCMaH65W")
	if err != nil {
		t.Error(err)
		return
	}

	// Initialize Solana client
	client := rpc.New("https://api.mainnet-beta.solana.com")
	if err := RaySwapByApi(client, wallet, "7YpUFdHWmjbLF559hppQcixmx5napb3gWpg9LCgQ9oyd", "So11111111111111111111111111111111111111112", "2qCcBUbmChGyvAjT9QRamZRgpWp943YXYXTix7wdJ3xB", "Dcs6Nf2MMPsMEkbBv6Vy6f76fHYerLMFhAaVXJa9gX4Q", 10, 100); err != nil {
		t.Fatalf("Swap failed: %v", err)
	}
}
