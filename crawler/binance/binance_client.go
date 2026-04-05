package binance

import (
	"fmt"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/ethereum/go-ethereum/log"
)

type BinanceClient struct {
	BinanceCli *ccxt.Binance
}

func NewBinanceBinanceClient() (*BinanceClient, error) {
	binanceCli := ccxt.NewBinance(map[string]interface{}{
		"enableRateLimit": true,
	})
	return &BinanceClient{
		BinanceCli: binanceCli,
	}, nil
}

func (bc *BinanceClient) FetchOrderBook(symbol string) (*ccxt.OrderBook, error) {
	markets, err := bc.BinanceCli.LoadMarkets()
	if err != nil {
		log.Error("BinanceCli load markets error", err)
		return nil, err
	}
	log.Info(fmt.Sprintf("BinanceCli load markets %v", markets))
	book, err := bc.BinanceCli.FetchOrderBook(symbol)
	if err != nil {
		log.Error("fetch binance order book err:", "error", err)
		return nil, err
	}
	fmt.Println(book.Bids[0][0], book.Asks[0][0])
	return &book, nil
}
