package okx

import (
	"fmt"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/ethereum/go-ethereum/log"
)

type okxClient struct {
	okxCli *ccxt.Okx
}

func NewOkxClient() (*okxClient, error) {
	okxCli := ccxt.NewOkx(map[string]interface{}{
		"enableRateLimit": true,
	})
	return &okxClient{
		okxCli: okxCli,
	}, nil
}

func (bc *okxClient) FetchOrderBook(symbol string) (*ccxt.OrderBook, error) {
	markets, err := bc.okxCli.LoadMarkets()
	if err != nil {
		log.Error("okxCli load markets error", err)
		return nil, err
	}
	log.Info(fmt.Sprintf("okxCli load markets %v", markets))
	book, err := bc.okxCli.FetchOrderBook(symbol)
	if err != nil {
		log.Error("fetch okx order book err:", "error", err)
		return nil, err
	}
	fmt.Println(book.Bids[0][0], book.Asks[0][0])
	return &book, nil
}
