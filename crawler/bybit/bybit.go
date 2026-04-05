package bybit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/redis"
	"github.com/ethereum/go-ethereum/log"
)

type BybitCrawler struct {
	db             *database.DB
	redisCli       *redis.Client
	bybitClient    *bybitClient
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewBybitCrawler(db *database.DB, redisCli *redis.Client, shutDown context.CancelCauseFunc) (*BybitCrawler, error) {
	bybitClient, err := NewBybitClient()
	if err != nil {
		log.Error("Failed to create Bybit client")
		return nil, err
	}
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	defer resourceCancel()
	return &BybitCrawler{
		db:             db,
		redisCli:       redisCli,
		bybitClient:    bybitClient,
		resourceCtx:    resourceCtx,
		resourceCancel: resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutDown(fmt.Errorf("BybitCrawler  critical Error:%v", err))
			},
		},
	}, nil
}

func (bc *BybitCrawler) Close() error {
	bc.resourceCancel()
	return bc.tasks.Wait()
}

func (bc *BybitCrawler) Start() error {
	bc.tasks.Go(func() error {
		tickerOperator := time.NewTicker(time.Second * 5)
		defer tickerOperator.Stop()
		for {
			select {
			case <-tickerOperator.C:
				err := bc.syncOrderBookData()
				if err != nil {
					log.Error("syncOrderBookData error:", err)
					return err
				}
				err = bc.syncCurrencyRateData()
				if err != nil {
					log.Error("syncCurrencyRateData error:", err)
					return err
				}
			case <-bc.resourceCtx.Done():
				log.Info("Bybit fetch data  stopped")
				return errors.New("Bybit stopped")
			}
		}
	})
	return nil
}

func (bc *BybitCrawler) syncOrderBookData() error {
	symbolList, err := bc.db.Symbol.QuerySymbols()
	if err != nil {
		log.Error("QuerySymbols error:", err)
		return err
	}
	for _, symbol := range symbolList {
		orderbook, err := bc.bybitClient.FetchOrderBook(symbol.SymbolName)
		if err != nil {
			log.Error("FetchOrderBook error:", "symbol", symbol.SymbolName, "error", err)
			return err
		}
		err = bc.redisCli.Set(bc.resourceCtx, symbol.SymbolName, orderbook, time.Minute*10)
		if err != nil {
			log.Error("Failed to set orderbook", "symbol", symbol.SymbolName, "error", err)
			return err
		}
	}
	return nil
}

func (bc *BybitCrawler) syncCurrencyRateData() error {
	return nil
}
