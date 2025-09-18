package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestP(t *testing.T) {
	str := "2025-05-18 09:10:56,BYGVg25kjwxoEFamL6DDHToFFYSDxEscdRzN3QR4bonk,0.004877124"
	parts := strings.Split(str, ",")
	if len(parts) >= 3 {
		profitStr := parts[2]
		profit, err := strconv.ParseFloat(profitStr, 64)
		if err != nil {
			// 处理解析错误
			fmt.Fprintf(os.Stderr, "无法解析 profit: %v\n", err)

		}
		t.Log(profit)
	}
}
