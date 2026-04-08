package cryptoexchange

import (
	"github.com/ccxt/ccxt/go/v4"
	"github.com/ethereum/go-ethereum/log"
)

type ExchangeClient struct {
	BybitClient   *ccxt.Bybit
	OxkClient     *ccxt.Okx
	BinanceClient *ccxt.Binance
}

func NewExchangeClient(proxy string, proxyType string) (*ExchangeClient, error) {
	cfg := map[string]interface{}{
		"enableRateLimit": true,
		"timeout":         60000,
		"options": map[string]interface{}{
			"defaultType": "spot",
		},
	}
	switch proxyType {
	case "http":
		cfg["httpProxy"] = proxy
	case "socks5":
		cfg["socksProxy"] = proxy
	}

	var bybitCli *ccxt.Bybit
	var okxCli *ccxt.Okx
	var binanceCli *ccxt.Binance
	successCount := 0

	// Try to initialize Bybit
	bybitCli = ccxt.NewBybit(cfg)
	_, err := bybitCli.LoadMarkets()
	if err != nil {
		log.Warn("bybit load markets error, skipping", "proxyType", proxyType, "proxy", proxy, "error", err)
		bybitCli = nil
	} else {
		log.Info("bybit create success", "proxyType", proxyType, "proxy", proxy)
		successCount++
	}

	// Try to initialize OKX
	okxCli = ccxt.NewOkx(cfg)
	_, err = okxCli.LoadMarkets()
	if err != nil {
		log.Warn("okx load markets error, skipping", "proxyType", proxyType, "proxy", proxy, "error", err)
		okxCli = nil
	} else {
		log.Info("oxk create success", "proxyType", proxyType, "proxy", proxy)
		successCount++
	}

	// Try to initialize Binance
	binanceCli = ccxt.NewBinance(cfg)
	_, err = binanceCli.LoadMarkets()
	if err != nil {
		log.Warn("binance load markets error, skipping", "proxyType", proxyType, "proxy", proxy, "error", err)
		binanceCli = nil
	} else {
		log.Info("binance create success", "proxyType", proxyType, "proxy", proxy)
		successCount++
	}

	if successCount == 0 {
		return nil, err
	}

	log.Info("Exchange client initialized", "successCount", successCount, "total", 3)
	return &ExchangeClient{
		BybitClient:   bybitCli,
		OxkClient:     okxCli,
		BinanceClient: binanceCli,
	}, nil
}

func (ec *ExchangeClient) FetchOrderBook(exchangeName, symbol string) (*ccxt.OrderBook, error) {
	var orderBook ccxt.OrderBook
	var err error
	switch exchangeName {
	case "Binance":
		if ec.BinanceClient == nil {
			log.Warn("Binance client not initialized, skipping")
			return nil, nil
		}
		orderBook, err = ec.BinanceClient.FetchOrderBook(symbol)
		if err != nil {
			log.Error("binance fetch order book error", "exchangeName", exchangeName, "symbol", symbol, "error", err)
			return nil, err
		}
	case "OKX":
		if ec.OxkClient == nil {
			log.Warn("OKX client not initialized, skipping")
			return nil, nil
		}
		orderBook, err = ec.OxkClient.FetchOrderBook(symbol)
		if err != nil {
			log.Error("okx fetch order book error", "exchangeName", exchangeName, "symbol", symbol, "error", err)
			return nil, err
		}
	case "Bybit":
		if ec.BybitClient == nil {
			log.Warn("Bybit client not initialized, skipping")
			return nil, nil
		}
		orderBook, err = ec.BybitClient.FetchOrderBook(symbol)
		if err != nil {
			log.Error("bybit fetch order book error", "exchangeName", exchangeName, "symbol", symbol, "error", err)
			return nil, err
		}
	}
	return &orderBook, nil
}
