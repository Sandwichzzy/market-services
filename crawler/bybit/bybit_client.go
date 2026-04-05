package bybit

import (
	"fmt"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/ethereum/go-ethereum/log"
)

type bybitClient struct {
	bybitCli *ccxt.Bybit
}

func NewBybitClient() (*bybitClient, error) {
	bybitCli := ccxt.NewBybit(map[string]interface{}{
		"enableRateLimit": true,
	})
	return &bybitClient{
		bybitCli: bybitCli,
	}, nil
}

func (bc *bybitClient) FetchOrderBook(symbol string) (*ccxt.OrderBook, error) {
	markets, err := bc.bybitCli.LoadMarkets()
	if err != nil {
		log.Error("bybitCli load markets error", err)
		return nil, err
	}
	log.Info(fmt.Sprintf("bybitCli load markets %v", markets))
	book, err := bc.bybitCli.FetchOrderBook(symbol)
	if err != nil {
		log.Error("fetch bybit order book err:", "error", err)
		return nil, err
	}
	fmt.Println(book.Bids[0][0], book.Asks[0][0])
	return &book, nil
}
