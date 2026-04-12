package worker

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	symbolKlineSyncInterval = time.Minute
	exchangeKlineLookback   = 3 * time.Minute
)

type SymbolKlineHandle struct {
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

type symbolKlineAggregate struct {
	symbolGuid string
	createdAt  time.Time
	openSum    decimal.Decimal
	closeSum   decimal.Decimal
	highPrice  decimal.Decimal
	lowPrice   decimal.Decimal
	volumeSum  decimal.Decimal
	count      int64
}

func NewSymbolKlineHandle(db *database.DB, shutDown context.CancelCauseFunc) (*SymbolKlineHandle, error) {
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &SymbolKlineHandle{
		db:             db,
		resourceCtx:    resourceCtx,
		resourceCancel: resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutDown(fmt.Errorf("symbol kline handle critical error:%v", err))
			},
		},
	}, nil
}

func (s *SymbolKlineHandle) Close() error {
	s.resourceCancel()
	return s.tasks.Wait()
}

func (s *SymbolKlineHandle) Start() error {
	s.tasks.Go(func() error {
		if err := s.syncSymbolKlineData(); err != nil {
			log.Error("syncSymbolKlineData initial error", "error", err)
			return err
		}

		tickerOperator := time.NewTicker(symbolKlineSyncInterval)
		defer tickerOperator.Stop()
		for {
			select {
			case <-tickerOperator.C:
				if err := s.syncSymbolKlineData(); err != nil {
					log.Error("syncSymbolKlineData error", "error", err)
					return err
				}
			case <-s.resourceCtx.Done():
				log.Info("symbol kline handle shutting down")
				return nil
			}
		}
	})
	return nil
}

func (s *SymbolKlineHandle) Stop() error {
	s.resourceCancel()
	return s.tasks.Wait()
}

// syncSymbolKlineData 从近几分钟的交易所 K 线中拉取已闭合数据，并聚合写入 symbol_kline。
// 这里按分钟对齐时间窗口，避免把当前正在形成的 K 线提前聚合进全市场结果。
func (s *SymbolKlineHandle) syncSymbolKlineData() error {
	// closedBefore 表示“当前分钟开始时间”，只有早于它的 K 线才算已闭合。
	closedBefore := time.Now().UTC().Truncate(time.Minute)
	// 只回看最近几分钟，兼顾去重更新和避免每轮全表扫描。
	since := closedBefore.Add(-exchangeKlineLookback)

	exchangeKlines, err := s.db.ExchangeSymbolKline.QueryExchangeSymbolKlinesSince(since)
	if err != nil {
		return err
	}

	// 先按 symbol + 分钟聚合，再幂等写入全市场 K 线表。
	aggregated, err := buildSymbolKlinesFromExchangeKlines(exchangeKlines, closedBefore)
	if err != nil {
		return err
	}
	return s.db.SymbolKline.UpsertSymbolKlines(aggregated)
}

// buildSymbolKlinesFromExchangeKlines 将交易所级别 K 线按 symbol 和分钟窗口聚合成全市场 K 线。
// 当前聚合规则为：open/close 取简单平均，high 取最大值，low 取最小值，volume 求和。
func buildSymbolKlinesFromExchangeKlines(exchangeKlines []*database.ExchangeSymbolKline, closedBefore time.Time) ([]database.SymbolKline, error) {
	if len(exchangeKlines) == 0 {
		return nil, nil
	}

	// aggregates 保存每个 symbol 在每个分钟窗口上的临时聚合状态。
	aggregates := make(map[string]*symbolKlineAggregate)

	for _, item := range exchangeKlines {
		if item == nil {
			continue
		}
		// 统一把时间截断到分钟，确保不同交易所同一根 1m K 线能归并到同一桶里。
		openedAt := item.CreatedAt.UTC().Truncate(time.Minute)
		// 当前分钟仍在形成中的 K 线不参与聚合。
		if !openedAt.Before(closedBefore) {
			continue
		}

		// 数据库存的是字符串数值，这里统一转成 decimal 做高精度聚合计算。
		openPrice, err := decimal.NewFromString(item.OpenPrice)
		if err != nil {
			return nil, fmt.Errorf("parse open_price: %w", err)
		}
		closePrice, err := decimal.NewFromString(item.ClosePrice)
		if err != nil {
			return nil, fmt.Errorf("parse close_price: %w", err)
		}
		highPrice, err := decimal.NewFromString(item.HighPrice)
		if err != nil {
			return nil, fmt.Errorf("parse high_price: %w", err)
		}
		lowPrice, err := decimal.NewFromString(item.LowPrice)
		if err != nil {
			return nil, fmt.Errorf("parse low_price: %w", err)
		}
		volume, err := decimal.NewFromString(item.Volume)
		if err != nil {
			return nil, fmt.Errorf("parse volume: %w", err)
		}

		// key 由交易对和分钟窗口组成，对应一根全市场 1m K 线。
		key := item.SymbolGuid + "|" + openedAt.Format(time.RFC3339)
		aggregate, ok := aggregates[key]
		if !ok {
			// 首次遇到该分钟窗口时初始化聚合器。
			aggregates[key] = &symbolKlineAggregate{
				symbolGuid: item.SymbolGuid,
				createdAt:  openedAt,
				openSum:    openPrice,
				closeSum:   closePrice,
				highPrice:  highPrice,
				lowPrice:   lowPrice,
				volumeSum:  volume,
				count:      1,
			}
			continue
		}

		// open/close 用于后续求简单平均；high/low 保留极值；volume 累加。
		aggregate.openSum = aggregate.openSum.Add(openPrice)
		aggregate.closeSum = aggregate.closeSum.Add(closePrice)
		if highPrice.GreaterThan(aggregate.highPrice) {
			aggregate.highPrice = highPrice
		}
		if lowPrice.LessThan(aggregate.lowPrice) {
			aggregate.lowPrice = lowPrice
		}
		aggregate.volumeSum = aggregate.volumeSum.Add(volume)
		aggregate.count++
	}

	if len(aggregates) == 0 {
		return nil, nil
	}

	// 对聚合 key 排序，保证输出顺序稳定，便于测试和排查。
	keys := make([]string, 0, len(aggregates))
	for key := range aggregates {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	now := time.Now().UTC()
	result := make([]database.SymbolKline, 0, len(keys))
	for _, key := range keys {
		aggregate := aggregates[key]
		// open/close 用参与聚合的交易所数量做简单平均。
		countDecimal := decimal.NewFromInt(aggregate.count)
		result = append(result, database.SymbolKline{
			Guid:       uuid.New().String(),
			SymbolGuid: aggregate.symbolGuid,
			OpenPrice:  aggregate.openSum.Div(countDecimal).String(),
			ClosePrice: aggregate.closeSum.Div(countDecimal).String(),
			HighPrice:  aggregate.highPrice.String(),
			LowPrice:   aggregate.lowPrice.String(),
			Volume:     aggregate.volumeSum.Floor().StringFixed(0),
			MarketCap:  "0",
			IsActive:   true,
			CreatedAt:  aggregate.createdAt,
			UpdatedAt:  now,
		})
	}

	return result, nil
}
