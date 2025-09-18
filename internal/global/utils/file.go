package utils

import (
	"encoding/csv"
	"os"
	"solana-bot/internal/global/utils/pubsub"
)

func WriteProfit(filename string, inputChan pubsub.Subscriber) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	stat, _ := file.Stat()
	if stat.Size() == 0 {
		headers := []string{"time", "token", "buy", "profit"}
		writer.Write(headers)
	}

	batch := make([][]string, 0, 1)
	for input := range inputChan {
		info := input.(string)
		batch = append(batch, []string{info})
		if len(batch) >= cap(batch) {
			writer.WriteAll(batch)
			batch = batch[:0]
		}

	}
	if len(batch) > 0 {
		writer.WriteAll(batch)
	}
}
