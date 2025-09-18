package main

import (
	"github.com/gagliardetto/solana-go"
	"github.com/gin-gonic/gin"
)

var (
	endpointRPC = "http://185.209.179.15:8899"
	wSolMint    = solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")
	usdcMint    = solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
	buyAmount   = uint64(5e8)
)

func main() {

	arbBot := NewArbBot()
	arbBot.Start()

	r := gin.Default()
	// r.GET("/map", printMap)
	r.Run(":9000")
}
