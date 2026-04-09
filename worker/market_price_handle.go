package worker

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/redis"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type MarketPriceHandle struct {
	db             *database.DB
	redisCli       *redis.Client
	cmcClient      *coinMarketCapClient // 用于补全市值和 24h 交易量
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

// marketPriceRecord 表示一条待入库的市场价快照。
// Redis 负责提供价格字段，CMC 负责补充基础币级别的 volume / market cap。
type marketPriceRecord struct {
	symbol          *database.Symbol
	avgPrice        string
	askPrice        string
	bidPrice        string
	baseAssetSymbol string
}

// NewMarketPriceHandle 创建 worker 的市场价处理器。
// 这里会校验 CMC key 是否存在，避免 worker 启动后继续写入占位值。
func NewMarketPriceHandle(db *database.DB, redisCli *redis.Client, cfg *config.Config, shutDown context.CancelCauseFunc) (*MarketPriceHandle, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.APIKeyConfig.CoinMarketCap == "" {
		return nil, fmt.Errorf("coin market cap API key is required")
	}

	cmcClient, err := newCoinMarketCapClient(cfg.APIKeyConfig.CoinMarketCap, nil)
	if err != nil {
		return nil, err
	}

	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &MarketPriceHandle{
		db:             db,
		redisCli:       redisCli,
		cmcClient:      cmcClient,
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
				return nil
			}
		}
	})
	return nil
}

// onPriceData 是 worker 的一轮处理逻辑：
// 1. 从 asset 表构建基础币 GUID -> symbol 映射
// 2. 从 Redis 读取交易对最新价格
// 3. 收集并去重基础币 symbol，批量请求 CMC
// 4. 将 Redis 价格与 CMC 市值/成交量合并后写入 symbol_market
func (mph *MarketPriceHandle) onPriceData() error {
	assetList, err := mph.db.Asset.QueryAssets()
	if err != nil {
		log.Error("Query assets fail", "error", err)
		return err
	}
	// base_asset_guid 存在 symbol 表里，CMC 查询需要的是资产符号，如 BTC、ETH。
	assetSymbolByGUID := make(map[string]string, len(assetList))
	for _, asset := range assetList {
		assetSymbolByGUID[asset.Guid] = asset.AssetSymbol
	}

	exchangeList, err := mph.db.Exchange.QueryExchanges()
	if err != nil {
		log.Error("Query exchanges fail", "error", err)
		return err
	}
	var records []marketPriceRecord
	cmcSymbolSet := make(map[string]struct{})

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

			// 通过 base_asset_guid 找到基础币符号，后续用于批量查询 CMC。
			baseAssetSymbol := assetSymbolByGUID[symbol.BaseAssetGuid]
			if baseAssetSymbol == "" {
				log.Warn("Base asset symbol not found, volume and market cap will default to zero",
					"symbol_guid", symbol.Guid,
					"base_asset_guid", symbol.BaseAssetGuid)
			} else {
				cmcSymbolSet[baseAssetSymbol] = struct{}{}
			}

			records = append(records, marketPriceRecord{
				symbol:          symbol,
				avgPrice:        avgPrice,
				askPrice:        askPrice,
				bidPrice:        bidPrice,
				baseAssetSymbol: baseAssetSymbol,
			})
		}
	}

	cmcQuotes := make(map[string]cmcQuote)
	cmcSymbols := make([]string, 0, len(cmcSymbolSet))
	for symbol := range cmcSymbolSet {
		cmcSymbols = append(cmcSymbols, symbol)
	}
	sort.Strings(cmcSymbols)

	// 同一轮内把所有基础币一次性发给 CMC，减少请求数和延迟。
	if len(cmcSymbols) > 0 {
		cmcQuotes, err = mph.cmcClient.FetchQuotes(mph.resourceCtx, cmcSymbols, "USD")
		if err != nil {
			// CMC 失败不阻断主流程，当前轮 volume / market cap 回退为 0。
			log.Error("Fetch CMC quotes fail, volume and market cap will default to zero", "error", err)
			cmcQuotes = map[string]cmcQuote{}
		}
	}

	for _, record := range records {
		guid, _ := uuid.NewUUID()
		radio := strconv.FormatFloat(mph.calcRate(record.avgPrice), 'f', 2, 64)
		volume := "0"
		marketCap := "0"

		if record.baseAssetSymbol != "" {
			quote, ok := cmcQuotes[record.baseAssetSymbol]
			if !ok {
				// 单个币种查不到时只影响当前记录，不中断整轮处理。
				log.Warn("CMC quote not found, volume and market cap will default to zero", "symbol", record.baseAssetSymbol)
			} else {
				volume = quote.Volume24h
				marketCap = quote.MarketCap
			}
		}

		// 最终写入的是“价格 + CMC 扩展字段”的合并快照。
		dataSymbolMk := &database.SymbolMarket{
			Guid:       guid.String(),
			SymbolGuid: record.symbol.Guid,
			Price:      record.avgPrice,
			AskPrice:   record.askPrice,
			BidPrice:   record.bidPrice,
			Volume:     volume,
			MarketCap:  marketCap,
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
	return nil
}

// calcRate 计算当前价格相对今日首条记录的涨跌幅。
// 若当日无历史数据或价格异常，则保守返回 0。
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
