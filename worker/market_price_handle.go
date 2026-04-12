package worker

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/redis"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MarketPriceHandle struct {
	db             *database.DB
	redisCli       *redis.Client
	cmcClient      *coinMarketCapClient // 用于补全市值和 24h 交易量
	baseCurrency   string
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

// marketPriceRecord 表示一条待入库的市场价快照。
// Redis 负责提供价格字段，CMC 负责补充基础币级别的 volume / market cap。
type marketPriceRecord struct {
	symbol           *database.Symbol
	price            string
	askPrice         string
	bidPrice         string
	baseAssetSymbol  string
	quoteAssetSymbol string
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
		baseCurrency:   cfg.BaseCurrency, //USDT
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
// 5. 事务处理SymbolMarketCurrencies 和 symbol_market 入库
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
	currencies, err := mph.db.Currency.QueryActiveCurrency()
	if err != nil {
		log.Error("Query currencies fail", "error", err)
		return err
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
				symbol:           symbol,
				price:            avgPrice,
				askPrice:         askPrice,
				bidPrice:         bidPrice,
				baseAssetSymbol:  baseAssetSymbol,
				quoteAssetSymbol: assetSymbolByGUID[symbol.QuoteAssetGuid], //symbol计价资产（USDT）在asset表
			})
		}
	}

	aggregatedRecords, err := aggregateMarketPriceRecords(records)
	if err != nil {
		return err
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

	for _, record := range aggregatedRecords {
		guid, _ := uuid.NewUUID()
		radio := strconv.FormatFloat(mph.calcRate(record.symbol.Guid, record.price), 'f', 2, 64)
		volume := "0"
		marketCap := "0"
		snapshotTime := time.Now()

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
			Price:      record.price,
			AskPrice:   record.askPrice,
			BidPrice:   record.bidPrice,
			Volume:     volume,
			MarketCap:  marketCap,
			Radio:      radio,
			IsActive:   true,
			CreatedAt:  snapshotTime,
			UpdatedAt:  snapshotTime,
		}

		err = mph.db.Transaction(func(tx *database.DB) error {
			if err := tx.SymbolMarket.StoreSymbolMarket(dataSymbolMk); err != nil {
				return err
			}
			if len(currencies) == 0 {
				return nil
			}
			return storeSymbolMarketCurrencies(tx, record, currencies, mph.baseCurrency, snapshotTime)
		})
		if err != nil {
			log.Error("Store symbol market snapshots fail", "symbol", record.symbol.SymbolName, "error", err)
			return err
		}
	}
	return nil
}

// aggregateMarketPriceRecords 按 symbol 聚合多个交易所的价格快照。
// price 取简单均价，ask_price 取全市场最小 ask，bid_price 取全市场最大 bid。
func aggregateMarketPriceRecords(records []marketPriceRecord) ([]marketPriceRecord, error) {
	// marketPriceAccumulator 只作为当前函数的临时聚合状态使用。
	type marketPriceAccumulator struct {
		symbol           *database.Symbol
		baseAssetSymbol  string
		quoteAssetSymbol string
		priceSum         decimal.Decimal
		minAskPrice      decimal.Decimal
		maxBidPrice      decimal.Decimal
		count            int64
	}

	accumulators := make(map[string]*marketPriceAccumulator)
	for _, record := range records {
		// Redis 中价格以字符串保存，这里先转成 decimal，避免浮点精度误差。
		avgPrice, err := decimal.NewFromString(record.price)
		if err != nil {
			return nil, fmt.Errorf("parse avg price for %s: %w", record.symbol.SymbolName, err)
		}
		askPrice, err := decimal.NewFromString(record.askPrice)
		if err != nil {
			return nil, fmt.Errorf("parse ask price for %s: %w", record.symbol.SymbolName, err)
		}
		bidPrice, err := decimal.NewFromString(record.bidPrice)
		if err != nil {
			return nil, fmt.Errorf("parse bid price for %s: %w", record.symbol.SymbolName, err)
		}

		accumulator, exists := accumulators[record.symbol.Guid]
		if !exists {
			// 每个 symbol 首次出现时初始化聚合状态。
			accumulators[record.symbol.Guid] = &marketPriceAccumulator{
				symbol:           record.symbol,
				baseAssetSymbol:  record.baseAssetSymbol,
				quoteAssetSymbol: record.quoteAssetSymbol,
				priceSum:         avgPrice,
				minAskPrice:      askPrice,
				maxBidPrice:      bidPrice,
				count:            1,
			}
			continue
		}

		// price 用于后续求简单均价；ask/bid 分别保留全市场最优报价。
		accumulator.priceSum = accumulator.priceSum.Add(avgPrice)
		if askPrice.LessThan(accumulator.minAskPrice) {
			accumulator.minAskPrice = askPrice
		}
		if bidPrice.GreaterThan(accumulator.maxBidPrice) {
			accumulator.maxBidPrice = bidPrice
		}
		accumulator.count++
	}

	// 对 key 排序，保证聚合输出顺序稳定，便于测试和排查日志。
	keys := make([]string, 0, len(accumulators))
	for key := range accumulators {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]marketPriceRecord, 0, len(keys))
	for _, key := range keys {
		accumulator := accumulators[key]
		// price 是所有交易所该 symbol 中间价的简单平均值。
		avgPrice := accumulator.priceSum.Div(decimal.NewFromInt(accumulator.count))
		result = append(result, marketPriceRecord{
			symbol:           accumulator.symbol,
			price:            avgPrice.String(),
			askPrice:         accumulator.minAskPrice.String(),
			bidPrice:         accumulator.maxBidPrice.String(),
			baseAssetSymbol:  accumulator.baseAssetSymbol,
			quoteAssetSymbol: accumulator.quoteAssetSymbol,
		})
	}

	return result, nil
}

// storeSymbolMarketCurrencies 为单条聚合行情生成所有启用法币的最新态价格并幂等入库。
// 当前仅在行情计价资产可映射到基础法币体系时执行写入，避免错误换算。
func storeSymbolMarketCurrencies(db *database.DB, record marketPriceRecord, currencies []*database.Currency, baseCurrency string, snapshotTime time.Time) error {
	if !supportsFiatConversion(record.quoteAssetSymbol, baseCurrency) {
		log.Warn("Skip symbol_market_currency snapshot for unsupported quote asset",
			"symbol", record.symbol.SymbolName,
			"quote_asset", record.quoteAssetSymbol,
			"base_currency", baseCurrency)
		return nil
	}

	items := make([]database.SymbolMarketCurrency, 0, len(currencies))
	for _, currency := range currencies {
		data, err := buildSymbolMarketCurrency(record.symbol.Guid, currency, record.price, record.askPrice, record.bidPrice, snapshotTime)
		if err != nil {
			return fmt.Errorf("build symbol_market_currency for %s/%s: %w", record.symbol.SymbolName, currency.CurrencyCode, err)
		}
		items = append(items, *data)
	}
	if err := db.SymbolMarketCurrency.UpsertSymbolMarketCurrencies(items); err != nil {
		return fmt.Errorf("upsert symbol_market_currency for %s: %w", record.symbol.SymbolName, err)
	}
	return nil
}

// buildSymbolMarketCurrency 将一条基础行情按指定法币汇率换算成法币最新态对象。
// 这里负责统一换算 price、ask_price、bid_price，并填充公共元数据字段。
func buildSymbolMarketCurrency(symbolGUID string, currency *database.Currency, price, askPrice, bidPrice string, snapshotTime time.Time) (*database.SymbolMarketCurrency, error) {
	convertedPrice, err := convertPriceByRate(price, currency.Rate)
	if err != nil {
		return nil, fmt.Errorf("convert price: %w", err)
	}
	convertedAskPrice, err := convertPriceByRate(askPrice, currency.Rate)
	if err != nil {
		return nil, fmt.Errorf("convert ask price: %w", err)
	}
	convertedBidPrice, err := convertPriceByRate(bidPrice, currency.Rate)
	if err != nil {
		return nil, fmt.Errorf("convert bid price: %w", err)
	}

	return &database.SymbolMarketCurrency{
		Guid:         uuid.New().String(),
		SymbolGuid:   symbolGUID,
		CurrencyGuid: currency.Guid,
		Price:        convertedPrice,
		AskPrice:     convertedAskPrice,
		BidPrice:     convertedBidPrice,
		IsActive:     currency.IsActive,
		CreatedAt:    snapshotTime,
		UpdatedAt:    snapshotTime,
	}, nil
}

// convertPriceByRate 使用高精度 decimal 将字符串价格按汇率相乘，避免浮点精度损失。
func convertPriceByRate(price string, rate float64) (string, error) {
	priceDecimal, err := decimal.NewFromString(price)
	if err != nil {
		return "", err
	}
	rateDecimal := decimal.NewFromFloat(rate)
	return priceDecimal.Mul(rateDecimal).String(), nil
}

// supportsFiatConversion 判断当前交易对的计价资产（比如USDT）是否支持直接换算到法币快照。
// 目前支持基础法币自身计价，以及在基础法币为 USD 时将 USDT 视作等价美元处理。
// quoteAssetSymbol：quoteAssetSymbol交易对里的“计价资产符号”，例子：BTC/USDT 里的 USDT
// baseCurrency：法币汇率系统的“基础法币”，当前默认是 USD，来自 config.Config.BaseCurrency
// BTC/BTC 或 BTC/ETH,这不是法币基础计价,直接乘法币汇率会错，所以返回 false
func supportsFiatConversion(quoteAssetSymbol, baseCurrency string) bool {
	normalizedQuoteAsset := strings.ToUpper(strings.TrimSpace(quoteAssetSymbol))
	normalizedBaseCurrency := strings.ToUpper(strings.TrimSpace(baseCurrency))
	if normalizedQuoteAsset == "" || normalizedBaseCurrency == "" {
		return false
	}
	if normalizedQuoteAsset == normalizedBaseCurrency {
		return true
	}
	return normalizedBaseCurrency == "USD" && normalizedQuoteAsset == "USDT"
}

// calcRate 计算当前价格相对今日首条记录的涨跌幅。
// 若当日无历史数据或价格异常，则保守返回 0。
func (mph *MarketPriceHandle) calcRate(symbolGuid, curPrice string) float64 {
	marketDataPrice, err := mph.db.SymbolMarket.QuerySymbolMarketTodayFirstDataBySymbol(symbolGuid)
	if err != nil {
		log.Error("Query symbol market first data fail", "symbol_guid", symbolGuid, "error", err)
		return 0
	}
	if marketDataPrice == nil {
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

	return calculateRate(currentPrice, firstPrice)
}

// calculateRate 计算涨跌幅，并将结果限制在数据库约束允许的区间内。
func calculateRate(currentPrice, firstPrice float64) float64 {
	if firstPrice == 0 {
		log.Warn("First price is zero, cannot calculate rate")
		return 0
	}

	radio := (currentPrice - firstPrice) / firstPrice
	return clampRate(radio)
}

// clampRate 确保涨跌幅满足 symbol_market.radio 的数据库约束。
func clampRate(rate float64) float64 {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0
	}
	if rate < -1 {
		return -1
	}
	if rate > 10 {
		return 10
	}
	return rate
}

func (mph *MarketPriceHandle) Stop() error {
	mph.resourceCancel()
	return mph.tasks.Wait()
}
