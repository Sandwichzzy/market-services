package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/shopspring/decimal"
)

// CoinMarketCap 最新报价接口，按 symbol 批量查询币种行情。
const coinMarketCapQuotesURL = "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest"

// cmcQuote 保存 worker 落库需要的两个核心字段。
type cmcQuote struct {
	MarketCap string
	Volume24h string
}

// coinMarketCapClient 封装对 CMC quotes/latest 接口的访问。
// 目前只负责按 symbol 拉取 USD 计价的市值和 24h 交易量。
type coinMarketCapClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// newCoinMarketCapClient 创建一个可复用的 CMC 客户端。
// 如果未传入 httpClient，则使用带超时的默认客户端。
func newCoinMarketCapClient(apiKey string, httpClient *http.Client) (*coinMarketCapClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("coin market cap API key is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &coinMarketCapClient{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    coinMarketCapQuotesURL,
	}, nil
}

// FetchQuotes 按基础币 symbol 批量查询 CMC 数据。
// 返回值以 symbol 为 key，value 为已经转换成数据库可存储格式的市值和成交量。
func (c *coinMarketCapClient) FetchQuotes(ctx context.Context, symbols []string, convert string) (map[string]cmcQuote, error) {
	uniqueSymbols := uniqueSymbols(symbols)
	if len(uniqueSymbols) == 0 {
		return map[string]cmcQuote{}, nil
	}
	if convert == "" {
		convert = "USD"
	}

	requestURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	query := requestURL.Query()
	query.Set("symbol", strings.Join(uniqueSymbols, ","))
	query.Set("convert", convert)
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create CMC request: %w", err)
	}
	req.Header.Set("X-CMC_PRO_API_KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request CMC quotes: %w", err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read CMC response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CMC quotes HTTP %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse CMC response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("CMC response data is empty")
	}

	quotes := make(map[string]cmcQuote, len(parsed.Data))
	for symbol, rawEntries := range parsed.Data {
		quotePayload, err := parseCMCQuotePayload(rawEntries, convert)
		if err != nil {
			return nil, fmt.Errorf("parse CMC quote for %s: %w", symbol, err)
		}
		if quotePayload == nil {
			continue
		}
		quote, ok := quotePayload.Quote[convert]
		if !ok {
			continue
		}
		quotes[symbol] = cmcQuote{
			MarketCap: decimalToUintString(quote.MarketCap),
			Volume24h: decimalToUintString(quote.Volume24h),
		}
		log.Info("Fetched CMC quote",
			"symbol", symbol,
			"convert", convert,
			"market_cap", quotes[symbol].MarketCap,
			"volume_24h", quotes[symbol].Volume24h)
	}

	return quotes, nil
}

type cmcQuotePayload struct {
	Quote map[string]struct {
		MarketCap float64 `json:"market_cap"`
		Volume24h float64 `json:"volume_24h"`
	} `json:"quote"`
}

// parseCMCQuotePayload 兼容解析 CMC `data.<SYMBOL>` 的两种可能结构：
// 1. 对象：`"BTC": { "quote": { "USD": ... } }`
// 2. 数组：`"BTC": [{ "quote": { "USD": ... } }]`
//
// 之所以这样做，是因为不同接口说明、示例或返回形态可能不完全一致。
// 当前 worker 只需要拿到一个可用的 quote，因此：
// - 如果是对象，直接返回该对象
// - 如果是数组，取第一项作为当前币种报价
// - 如果数组为空，返回 nil，表示当前 symbol 没有可用数据
func parseCMCQuotePayload(rawEntries json.RawMessage, convert string) (*cmcQuotePayload, error) {
	// 第一优先级：按“对象”结构解析。
	// 这是当前 CMC quotes/latest 实际更常见的返回形式。
	var objectPayload cmcQuotePayload
	if err := json.Unmarshal(rawEntries, &objectPayload); err == nil {
		return &objectPayload, nil
	}

	// 如果对象解析失败，再退回按“数组”结构解析，
	// 兼容某些示例或历史返回形态。
	var arrayPayload []cmcQuotePayload
	if err := json.Unmarshal(rawEntries, &arrayPayload); err != nil {
		return nil, err
	}
	// 数组解析成功但没有元素，说明该 symbol 没有可用报价。
	if len(arrayPayload) == 0 {
		return nil, nil
	}
	// 当前逻辑只需要一个报价快照，因此取第一项即可。
	return &arrayPayload[0], nil
}

// uniqueSymbols 对 symbol 列表做去重并过滤空值，
// 避免同一轮 worker 重复请求相同基础币。
func uniqueSymbols(symbols []string) []string {
	seen := make(map[string]struct{}, len(symbols))
	result := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		if symbol == "" {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		result = append(result, symbol)
	}
	return result
}

// decimalToUintString 将 CMC 的浮点结果转换为向下取整的整数字符串。
// 这是为了兼容库表中的 UINT256 / 大整数语义，避免写入小数字段格式。
func decimalToUintString(value float64) string {
	if value <= 0 {
		return "0"
	}
	return decimal.NewFromFloat(value).Floor().StringFixed(0)
}
