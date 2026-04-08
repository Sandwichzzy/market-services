package worker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/redis"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type MarketPriceHandle struct {
	db             *database.DB
	redisCli       *redis.Client
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewMarketPriceHandle(db *database.DB, redisCli *redis.Client, shutDown context.CancelCauseFunc) (*MarketPriceHandle, error) {
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &MarketPriceHandle{
		db:             db,
		redisCli:       redisCli,
		resourceCtx:    resourceCtx,
		resourceCancel: resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutDown(fmt.Errorf("market price handle critical error:%v", err))
			},
		},
	}, nil
}

func (mph *MarketPriceHandle) Close() error {
	mph.resourceCancel()
	return mph.tasks.Wait()
}

func (mph *MarketPriceHandle) Start() error {
	mph.tasks.Go(func() error {
		tickerOperator := time.NewTicker(time.Second * 10)
		defer tickerOperator.Stop()
		for {
			select {
			case <-tickerOperator.C:
				err := mph.onPriceData()
				if err != nil {
					log.Error("market price handle fail", "error", err)
					return err
				}
			case <-mph.resourceCtx.Done():
				log.Info("market price handle shutting down")
				return errors.New("market price service stopped")
			}
		}
	})
	return nil
}

func (mph *MarketPriceHandle) onPriceData() error {
	exchangeList, err := mph.db.Exchange.QueryExchanges()
	if err != nil {
		log.Error("Query exchanges fail", "error", err)
		return err
	}
	for _, exchange := range exchangeList {
		log.Info("exchange", "exchange", exchange.Name)
		exchangeSymbols, err := mph.db.ExchangeSymbol.QuerySymbolsByExchangeId(exchange.Guid)
		if err != nil {
			return err
		}

		for _, exchangeSymbol := range exchangeSymbols {
			symbol, err := mph.db.Symbol.QuerySymbolByGuid(exchangeSymbol.SymbolGuid)
			if err != nil {
				log.Error("Query symbol fail", "error", err)
				return err
			}
			log.Info("symbol", "symbolName", symbol.SymbolName)
			key := exchange.Guid + "%" + exchange.Name + "%" + symbol.Guid + "%" + symbol.SymbolName
			avgPrice, err := mph.redisCli.Get(mph.resourceCtx, key)
			if err != nil || avgPrice == "" {
				log.Warn("Price data not found in Redis, skipping", "key", key)
				continue // 跳过这个交易对
			}
			askPriceKey := key + "askPrice"
			askPrice, err := mph.redisCli.Get(mph.resourceCtx, askPriceKey)
			if err != nil || askPrice == "" {
				log.Warn("Ask price not found in Redis, skipping", "key", askPriceKey)
				continue
			}
			bidPriceKey := key + "bidPrice"
			bidPrice, err := mph.redisCli.Get(mph.resourceCtx, bidPriceKey)
			if err != nil || bidPrice == "" {
				log.Warn("Bid price not found in Redis, skipping", "key", bidPriceKey)
				continue
			}

			guid, _ := uuid.NewUUID()
			radio := strconv.FormatFloat(mph.calcRate(avgPrice), 'f', 2, 64)

			//todo:Volume交易量 和 MarketCap市值 从 CMC 获取数据完善
			dataSymbolMk := &database.SymbolMarket{
				Guid:       guid.String(),
				SymbolGuid: symbol.Guid,
				Price:      avgPrice,
				AskPrice:   askPrice,
				BidPrice:   bidPrice,
				Volume:     "10000",
				MarketCap:  "10000",
				Radio:      radio,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			err = mph.db.SymbolMarket.StoreSymbolMarket(dataSymbolMk)
			if err != nil {
				log.Error("Store symbol market fail", "error", err)
				return err
			}
		}
	}
	return nil
}

func (mph *MarketPriceHandle) calcRate(curPrice string) float64 {
	marketDataPrice, err := mph.db.SymbolMarket.QuerySymbolMarketTodayFirstData()
	if err != nil {
		log.Warn("No historical data found, using 0 as initial rate", "error", err)
		return 0
	}

	currentPrice, err := strconv.ParseFloat(curPrice, 64)
	if err != nil {
		log.Error("Failed to parse current price", "price", curPrice, "error", err)
		return 0
	}

	firstPrice, err := strconv.ParseFloat(marketDataPrice.Price, 64)
	if err != nil {
		log.Error("Failed to parse first price", "price", marketDataPrice.Price, "error", err)
		return 0
	}

	// 防止除以零
	if firstPrice == 0 {
		log.Warn("First price is zero, cannot calculate rate")
		return 0
	}

	radio := (currentPrice - firstPrice) / firstPrice
	return radio
}

func (mph *MarketPriceHandle) Stop() error {
	mph.resourceCancel()
	return mph.tasks.Wait()
}
