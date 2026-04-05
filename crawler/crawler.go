package crawler

import (
	"context"
	"sync/atomic"

	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/crawler/binance"
	"github.com/Sandwichzzy/market-services/crawler/bybit"
	"github.com/Sandwichzzy/market-services/crawler/fiatcurrency"
	"github.com/Sandwichzzy/market-services/crawler/okx"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/redis"
	"github.com/ethereum/go-ethereum/log"
)

type Crawler struct {
	BinanceCrawler *binance.BinanceCrawler
	OkxCrawler     *okx.OkxCrawler
	BybitCrawler   *bybit.BybitCrawler

	FiatCurrencyCrawler *fiatcurrency.FiatCurrencyCrawler
	stopped             atomic.Bool
}

func NewCrawler(db *database.DB, redisCli *redis.Client, config *config.Config, shutdown context.CancelCauseFunc) (*Crawler, error) {
	binanceCrawler, err := binance.NewBinanceCrawler(db, redisCli, shutdown)
	if err != nil {
		log.Error("Crawler NewBinanceCrawler err: ", err)
		return nil, err
	}
	okxCrawler, err := okx.NewOkxCrawler(db, redisCli, shutdown)
	if err != nil {
		log.Error("Crawler okxCrawler error", err)
		return nil, err
	}

	bybitCrawler, err := bybit.NewBybitCrawler(db, redisCli, shutdown)
	if err != nil {
		log.Error("Crawler bybitCrawler error", err)
		return nil, err
	}

	fiatcurrencyCrawler, err := fiatcurrency.NewFiatCurrencyCrawler(db, config, shutdown)
	if err != nil {
		log.Error("Crawler FiatCurrencyCrawler error", err)
		return nil, err
	}

	return &Crawler{
		BinanceCrawler:      binanceCrawler,
		OkxCrawler:          okxCrawler,
		BybitCrawler:        bybitCrawler,
		FiatCurrencyCrawler: fiatcurrencyCrawler,
	}, nil
}

func (cl *Crawler) Start(ctx context.Context) error {
	err := cl.BinanceCrawler.Start()
	if err != nil {
		log.Error("Crawler BinanceCrawler error", err)
		return err
	}
	err = cl.OkxCrawler.Start()
	if err != nil {
		log.Error("Crawler OkxCrawler error", err)
		return err
	}
	err = cl.BybitCrawler.Start()
	if err != nil {
		log.Error("Crawler BybitCrawler error", err)
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
	if err := cl.BinanceCrawler.Close(); err != nil {
		log.Error("Crawler BinanceCrawler error", err)
		return err
	}

	if err := cl.OkxCrawler.Close(); err != nil {
		log.Error("Crawler OkxCrawler error", err)
		return err
	}

	if err := cl.BybitCrawler.Close(); err != nil {
		log.Error("Crawler BybitCrawler error", err)
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
