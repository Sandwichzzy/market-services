package binance

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func TestFetchOrderBook(T *testing.T) {
	client, err := NewBinanceBinanceClient()
	if err != nil {
		log.Error("NewBinanceBinance err", err)
		return
	}
	book, err := client.FetchOrderBook("BTC-USDT")
	if err != nil {
		return
	}
	fmt.Println(book)
}
