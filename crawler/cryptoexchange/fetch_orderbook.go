package cryptoexchange

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

type ExchangeOrderbook struct {
	db             *database.DB
	redisCli       *redis.Client
	exchangeClient *ExchangeClient
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewExchangeOrderbook(db *database.DB, redisCli *redis.Client, shutDown context.CancelCauseFunc) (*ExchangeOrderbook, error) {
	exchangeClient, err := NewExchangeClient("http://127.0.0.1:7890", "http")
	if err != nil {
		log.Error("Failed to create exchange client")
		return nil, err
	}
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &ExchangeOrderbook{
		db:             db,
		redisCli:       redisCli,
		exchangeClient: exchangeClient,
		resourceCtx:    resourceCtx,
		resourceCancel: resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutDown(fmt.Errorf("ExchangeOrderbook  critical Error:%v", err))
			},
		},
	}, nil
}

func (bc *ExchangeOrderbook) Close() error {
	bc.resourceCancel()
	return bc.tasks.Wait()
}

func (bc *ExchangeOrderbook) Start() error {
	bc.tasks.Go(func() error {
		tickerOperator := time.NewTicker(time.Second * 10)
		defer tickerOperator.Stop()
		for {
			select {
			case <-tickerOperator.C:
				log.Debug("Fetching order book start")
				err := bc.syncOrderBookData()
				if err != nil {
					log.Error("syncOrderBookData error:", err)
					return err
				}
			case <-bc.resourceCtx.Done():
				log.Info("exchange fetch orderbook shutting down")
				return errors.New("exchange stopped")
			}
		}
	})
	return nil
}

func (bc *ExchangeOrderbook) syncOrderBookData() error {
	exchangeList, err := bc.db.Exchange.QueryExchanges()
	if err != nil {
		log.Error("QueryExchanges error:", err)
		return err
	}
	for _, exchange := range exchangeList {
		log.Info("exchange", "exchange", exchange.Name)
		exchangeSymbols, err := bc.db.ExchangeSymbol.QuerySymbolsByExchangeId(exchange.Guid)
		if err != nil {
			return err
		}
		for _, exchangeSymbol := range exchangeSymbols {
			symbol, err := bc.db.Symbol.QuerySymbolByGuid(exchangeSymbol.SymbolGuid)
			if err != nil {
				log.Error("Query symbol fail", "error", err)
				return err
			}
			log.Info("symbol", "symbolName", symbol.SymbolName)

			orderBook, err := bc.exchangeClient.FetchOrderBook(exchange.Name, symbol.SymbolName)
			if err != nil {
				log.Error("Fetch order book fail", "symbol", symbol.SymbolName, "error", err)
				return err
			}
			if orderBook == nil {
				continue
			}
			askPrice := orderBook.Asks[0][0]
			bidPrice := orderBook.Bids[0][0]
			avgPrice := (askPrice + bidPrice) / 2
			key := exchange.Guid + "%" + exchange.Name + "%" + symbol.Guid + "%" + symbol.SymbolName
			log.Info("Fetch orderbook success", "key", key, "askPrice", askPrice, "bidPrice", bidPrice, "avgPrice", avgPrice)

			err = bc.redisCli.Set(bc.resourceCtx, key, avgPrice, time.Second*600)
			if err != nil {
				log.Error("Set avgPrice fail", "symbol", symbol.SymbolName, "error", err)
				return err
			}

			askPriceKey := key + "askPrice"
			err = bc.redisCli.Set(bc.resourceCtx, askPriceKey, askPrice, time.Second*600)
			if err != nil {
				log.Error("Set askPrice fail", "symbol", symbol.SymbolName, "error", err)
				return err
			}

			bidPriceKey := key + "bidPrice"
			err = bc.redisCli.Set(bc.resourceCtx, bidPriceKey, bidPrice, time.Second*600)
			if err != nil {
				log.Error("Set askPrice fail", "symbol", symbol.SymbolName, "error", err)
				return err
			}
		}
	}
	return nil
}

func (bc *ExchangeOrderbook) Stop() error {
	bc.resourceCancel()
	return bc.tasks.Wait()
}
