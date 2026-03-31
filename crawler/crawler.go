package crawler

import (
	"context"
	"sync/atomic"

	"github.com/Sandwichzzy/market-services/crawler/binance"
	"github.com/Sandwichzzy/market-services/crawler/bybit"
	"github.com/Sandwichzzy/market-services/crawler/okx"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/ethereum/go-ethereum/log"
)

type Crawler struct {
	BinanceCrawler *binance.BinanceCrawler
	OkxCrawler     *okx.OkxCrawler
	BybitCrawler   *bybit.BybitCrawler

	stopped atomic.Bool
}

func NewCrawler(db *database.DB, shutdown context.CancelCauseFunc) (*Crawler, error) {
	binanceCrawler, err := binance.NewBinanceCrawler(db, shutdown)
	if err != nil {
		log.Error("Crawler NewBinanceCrawler err: ", err)
		return nil, err
	}
	okxCrawler, err := okx.NewOkxCrawler(db, shutdown)
	if err != nil {
		log.Error("Crawler okxCrawler error", err)
		return nil, err
	}

	bybitCrawler, err := bybit.NewBybitCrawler(db, shutdown)
	if err != nil {
		log.Error("Crawler bybitCrawler error", err)
		return nil, err
	}
	return &Crawler{
		BinanceCrawler: binanceCrawler,
		OkxCrawler:     okxCrawler,
		BybitCrawler:   bybitCrawler,
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
	return nil
}

func (cl *Crawler) Stopped() bool {
	return cl.stopped.Load()
}
