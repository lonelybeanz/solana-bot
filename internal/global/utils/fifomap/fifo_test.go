package fifomap

import (
	"fmt"
	"testing"
)

func TestFifo(t *testing.T) {
	cache := NewFIFOMap(3) // 最多保存3个元素

	cache.Set("a", 1) // 插入 a:1
	cache.Set("b", 2) // 插入 b:2
	cache.Set("c", 3) // 插入 c:3
	cache.Set("d", 4) // 插入 d:4 → 自动淘汰 a:1

	fmt.Println(cache.Get("a")) // <nil> false (已被淘汰)
	fmt.Println(cache.Get("b")) // 2 true
	fmt.Println(cache.Get("d")) // 4 true
}
