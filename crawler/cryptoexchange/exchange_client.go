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

// FetchOHLCV 按交易所名称和交易对抓取统一格式的 OHLCV 数据。
// timeframe、since、limit 会透传给 CCXT，用于控制 K 线周期和拉取范围。
func (ec *ExchangeClient) FetchOHLCV(exchangeName, symbol, timeframe string, since, limit int64) ([]ccxt.OHLCV, error) {
	options := make([]ccxt.FetchOHLCVOptions, 0, 3)
	if timeframe != "" {
		options = append(options, ccxt.WithFetchOHLCVTimeframe(timeframe))
	}
	if since > 0 {
		options = append(options, ccxt.WithFetchOHLCVSince(since))
	}
	if limit > 0 {
		options = append(options, ccxt.WithFetchOHLCVLimit(limit))
	}

	switch exchangeName {
	case "Binance":
		if ec.BinanceClient == nil {
			log.Warn("Binance client not initialized, skipping OHLCV")
			return nil, nil
		}
		return ec.BinanceClient.FetchOHLCV(symbol, options...)
	case "OKX":
		if ec.OxkClient == nil {
			log.Warn("OKX client not initialized, skipping OHLCV")
			return nil, nil
		}
		return ec.OxkClient.FetchOHLCV(symbol, options...)
	case "Bybit":
		if ec.BybitClient == nil {
			log.Warn("Bybit client not initialized, skipping OHLCV")
			return nil, nil
		}
		return ec.BybitClient.FetchOHLCV(symbol, options...)
	default:
		log.Warn("Unsupported exchange for OHLCV", "exchangeName", exchangeName)
		return nil, nil
	}
}
