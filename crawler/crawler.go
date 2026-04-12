package crawler

import (
	"context"
	"sync/atomic"

	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/crawler/cryptoexchange"
	"github.com/Sandwichzzy/market-services/crawler/fiatcurrency"

	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/redis"
	"github.com/ethereum/go-ethereum/log"
)

type Crawler struct {
	ExchangeOrderbook   *cryptoexchange.ExchangeOrderbook
	ExchangeKline       *cryptoexchange.ExchangeKlineCrawler
	FiatCurrencyCrawler *fiatcurrency.FiatCurrencyCrawler
	stopped             atomic.Bool
}

func NewCrawler(db *database.DB, redisCli *redis.Client, config *config.Config, shutdown context.CancelCauseFunc) (*Crawler, error) {
	exchangeOrderbook, err := cryptoexchange.NewExchangeOrderbook(db, redisCli, shutdown)
	if err != nil {
		log.Error("Crawler NewBinanceCrawler error", err)
		return nil, err
	}
	exchangeKline, err := cryptoexchange.NewExchangeKlineCrawler(db, shutdown)
	if err != nil {
		log.Error("Crawler ExchangeKlineCrawler error", err)
		return nil, err
	}

	fiatcurrencyCrawler, err := fiatcurrency.NewFiatCurrencyCrawler(db, config, shutdown)
	if err != nil {
		log.Error("Crawler FiatCurrencyCrawler error", err)
		return nil, err
	}

	return &Crawler{
		ExchangeOrderbook:   exchangeOrderbook,
		ExchangeKline:       exchangeKline,
		FiatCurrencyCrawler: fiatcurrencyCrawler,
	}, nil
}

func (cl *Crawler) Start(ctx context.Context) error {
	err := cl.ExchangeOrderbook.Start()
	if err != nil {
		log.Error("Crawler ExchangeOrderbook Start error", err)
		return err
	}
	err = cl.ExchangeKline.Start()
	if err != nil {
		log.Error("Crawler ExchangeKline Start error", err)
		return err
	}
	err = cl.FiatCurrencyCrawler.Start()
	if err != nil {
		log.Error("Crawler FiatCurrencyCrawler error", err)
		return err
	}
	return nil
}

func (cl *Crawler) Stop(ctx context.Context) error {
	if err := cl.ExchangeOrderbook.Close(); err != nil {
		log.Error("Crawler ExchangeOrderbook Stop error", err)
		return err
	}
	if err := cl.ExchangeKline.Close(); err != nil {
		log.Error("Crawler ExchangeKline Stop error", err)
		return err
	}
	if err := cl.FiatCurrencyCrawler.Close(); err != nil {
		log.Error("Crawler FiatCurrencyCrawler error", err)
		return err
	}
	return nil
}

func (cl *Crawler) Stopped() bool {
	return cl.stopped.Load()
}
