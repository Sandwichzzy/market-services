package cryptoexchange

import (
	"context"
	"fmt"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/ccxt/ccxt/go/v4"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

//当前抓取k线逻辑等价于：
//- 每分钟执行一次
//- 对每个交易所、每个交易对
//- 拉最近 2 根 1m K 线
//- 只把已经闭合的那部分写进 exchange_symbol_kline
//  如果只拉 1 根，很多时候拉到的正好是“当前正在形成中的分钟 K 线”，代码又会把它过滤掉，这样这一轮就可能什么都拿不到。
//  设成 2 的目的就是：
//  - 一根可能是未闭合的当前 K 线
//  - 另一根通常是已经闭合、可以安全入库的上一根

const (
	defaultKlineTimeframe  = "1m"     //K 线周期是 1 分钟线
	defaultKlineFetchLimit = int64(2) // 表示每次向交易所拉 最近 2 根 K 线
	klineSyncInterval      = time.Minute
)

type ExchangeKlineCrawler struct {
	db             *database.DB
	exchangeClient *ExchangeClient
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

// NewExchangeKlineCrawler 创建交易所 K 线抓取器。
// 抓取器复用统一的 ExchangeClient，并在内部维护独立的生命周期上下文。
func NewExchangeKlineCrawler(db *database.DB, shutDown context.CancelCauseFunc) (*ExchangeKlineCrawler, error) {
	exchangeClient, err := NewExchangeClient("http://127.0.0.1:7890", "http")
	if err != nil {
		log.Error("Failed to create exchange client for kline crawler")
		return nil, err
	}
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &ExchangeKlineCrawler{
		db:             db,
		exchangeClient: exchangeClient,
		resourceCtx:    resourceCtx,
		resourceCancel: resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutDown(fmt.Errorf("ExchangeKlineCrawler critical Error:%v", err))
			},
		},
	}, nil
}

// Close 停止抓取器并等待后台任务退出。
func (ec *ExchangeKlineCrawler) Close() error {
	ec.resourceCancel()
	return ec.tasks.Wait()
}

// Start 启动交易所 K 线同步循环。
// 启动时会先执行一次同步，之后按固定周期拉取最近已闭合的 1m K 线。
func (ec *ExchangeKlineCrawler) Start() error {
	ec.tasks.Go(func() error {
		if err := ec.syncKlineData(); err != nil {
			log.Error("syncKlineData initial error", "error", err)
			return err
		}

		tickerOperator := time.NewTicker(klineSyncInterval)
		defer tickerOperator.Stop()

		for {
			select {
			case <-tickerOperator.C:
				log.Debug("Fetching exchange klines start")
				if err := ec.syncKlineData(); err != nil {
					log.Error("syncKlineData error", "error", err)
					return err
				}
			case <-ec.resourceCtx.Done():
				log.Info("exchange kline crawler shutting down")
				return nil
			}
		}
	})
	return nil
}

// syncKlineData 遍历启用交易所和交易对，抓取最近一小段 OHLCV 并幂等写入数据库。
// 当前仅保留已闭合的分钟 K 线，避免未收盘数据被提前落库。
func (ec *ExchangeKlineCrawler) syncKlineData() error {
	exchangeList, err := ec.db.Exchange.QueryExchanges()
	if err != nil {
		log.Error("QueryExchanges for kline error", "error", err)
		return err
	}

	closedBefore := time.Now().UTC().Truncate(time.Minute)
	var rows []database.ExchangeSymbolKline

	for _, exchange := range exchangeList {
		exchangeSymbols, err := ec.db.ExchangeSymbol.QuerySymbolsByExchangeId(exchange.Guid)
		if err != nil {
			return err
		}

		for _, exchangeSymbol := range exchangeSymbols {
			symbol, err := ec.db.Symbol.QuerySymbolByGuid(exchangeSymbol.SymbolGuid)
			if err != nil {
				log.Error("Query symbol for kline fail", "error", err)
				return err
			}

			ohlcvs, err := ec.exchangeClient.FetchOHLCV(exchange.Name, symbol.SymbolName, defaultKlineTimeframe, 0, defaultKlineFetchLimit)
			if err != nil {
				log.Warn("Fetch OHLCV fail, skipping", "exchange", exchange.Name, "symbol", symbol.SymbolName, "error", err)
				continue
			}
			if len(ohlcvs) == 0 {
				continue
			}

			rows = append(rows, buildExchangeSymbolKlines(exchange.Guid, symbol.Guid, ohlcvs, closedBefore)...)
		}
	}

	if err := ec.db.ExchangeSymbolKline.UpsertExchangeSymbolKlines(rows); err != nil {
		return err
	}
	return nil
}

// Stop 主动停止抓取器并等待后台任务完成清理。
func (ec *ExchangeKlineCrawler) Stop() error {
	ec.resourceCancel()
	return ec.tasks.Wait()
}

// buildExchangeSymbolKlines 将 CCXT 返回的 OHLCV 列表转换为交易所 K 线落库模型。
// 只有开盘时间早于 closedBefore 的已闭合 K 线会被保留。
func buildExchangeSymbolKlines(exchangeGuid, symbolGuid string, ohlcvs []ccxt.OHLCV, closedBefore time.Time) []database.ExchangeSymbolKline {
	result := make([]database.ExchangeSymbolKline, 0, len(ohlcvs))
	updatedAt := time.Now().UTC()

	for _, ohlcv := range ohlcvs {
		openedAt := time.UnixMilli(ohlcv.Timestamp).UTC()
		if !openedAt.Before(closedBefore) {
			continue
		}

		result = append(result, database.ExchangeSymbolKline{
			Guid:         uuid.New().String(),
			ExchangeGuid: exchangeGuid,
			SymbolGuid:   symbolGuid,
			OpenPrice:    floatToNumericString(ohlcv.Open),
			ClosePrice:   floatToNumericString(ohlcv.Close),
			HighPrice:    floatToNumericString(ohlcv.High),
			LowPrice:     floatToNumericString(ohlcv.Low),
			Volume:       floatToUintString(ohlcv.Volume),
			MarketCap:    "0",
			IsActive:     true,
			CreatedAt:    openedAt,
			UpdatedAt:    updatedAt,
		})
	}

	return result
}

// floatToNumericString 将浮点价格转换为数据库数值字段使用的字符串格式。
func floatToNumericString(value float64) string {
	if value <= 0 {
		return "0"
	}
	return decimal.NewFromFloat(value).String()
}

// floatToUintString 将浮点成交量向下取整为无符号整数字符串。
func floatToUintString(value float64) string {
	if value <= 0 {
		return "0"
	}
	return decimal.NewFromFloat(value).Floor().StringFixed(0)
}
