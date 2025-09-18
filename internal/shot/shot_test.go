package shot

import (
	"fmt"
	"math/big"
	"testing"
)

func TestEN(t *testing.T) {

	out := EstimateSwapOut(big.NewInt(800000000), big.NewInt(77170179973), big.NewInt(772648883227511), 125)
	t.Log(out.String())

	t.Log(out.Uint64() * 66 / 100)

	virtualBase := big.NewInt(1073025605596382)  // 1e9
	virtualQuote := big.NewInt(30000852951)      // 5e8
	realBaseAfter := big.NewInt(772629507767841) // 6e8
	realQuoteAfter := big.NewInt(77163267473)    // 4e8
	amountIn := big.NewInt(7000000)              // 输入 quote = 1e8

	output := PredictNextOutputByRealState(virtualBase, virtualQuote, realBaseAfter, realQuoteAfter, amountIn)

	fmt.Printf("Predicted output base: %s\n", output.String())

}


// 模拟 token0 -> token1 swap
func SimulateSwapToken0ToToken1(amountIn, liquidity, sqrtPrice float64) (amountOut float64, sqrtPriceAfter float64) {
    if amountIn <= 0 || liquidity <= 0 || sqrtPrice <= 0 {
        return 0, sqrtPrice
    }

    // 计算新的 sqrtPrice（交易后价格）
    sqrtPriceAfter = liquidity / (liquidity/sqrtPrice + amountIn)

    // 计算输出的 token1 数量
    amountOut = liquidity * (sqrtPrice - sqrtPriceAfter)

    return amountOut, sqrtPriceAfter
}