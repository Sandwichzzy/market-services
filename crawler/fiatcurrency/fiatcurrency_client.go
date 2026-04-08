package fiatcurrency

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type URLBuilder func(baseURL, apiKey, baseCurrency string) string

type ResponseParser func(body []byte, targetCurrencies []string) (map[string]float64, error)

type ExchangeRateWorker struct {
	db              *database.DB
	config          *config.Config
	providers       map[string]ExchangeRateProvider
	providerConfigs map[string]string // platform name -> API key
	strategyConfigs map[string]struct {
		urlBuilder     URLBuilder
		responseParser ResponseParser
		defaultBaseURL string
	}
}

type ExchangeRateProvider interface {
	FetchRates(ctx context.Context, baseCurrency string, targetCurrencies []string) (map[string]float64, error)
}

type GenericProvider struct {
	Name           string
	APIKey         string
	BaseURL        string
	URLBuilder     URLBuilder
	ResponseParser ResponseParser
}

func NewGenericProvider(name, apiKey, baseURL string, urlBuilder URLBuilder, responseParser ResponseParser) *GenericProvider {
	return &GenericProvider{
		Name:           name,
		APIKey:         apiKey,
		BaseURL:        baseURL,
		URLBuilder:     urlBuilder,
		ResponseParser: responseParser,
	}
}

func (p *GenericProvider) FetchRates(ctx context.Context, baseCurrency string, targetCurrencies []string) (map[string]float64, error) {
	url := p.URLBuilder(p.BaseURL, p.APIKey, baseCurrency)

	log.Debug("Fetching exchange rates", "provider", p.Name, "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch rates error: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response error: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	rates, err := p.ResponseParser(body, targetCurrencies)
	if err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("parse response error: %w (response preview: %s)", err, preview)
	}
	return rates, nil
}

func NewExchangeRateWorker(
	db *database.DB,
	config *config.Config,
	providerConfigs map[string]string,
	strategyConfigs map[string]struct {
		urlBuilder     URLBuilder
		responseParser ResponseParser
		defaultBaseURL string
	},
) (*ExchangeRateWorker, error) {

	worker := &ExchangeRateWorker{
		db:              db,
		config:          config,
		providerConfigs: providerConfigs,
		strategyConfigs: strategyConfigs,
		providers:       make(map[string]ExchangeRateProvider),
	}

	if err := worker.initializeProviders(); err != nil {
		return nil, err
	}

	return worker, nil
}

func (w *ExchangeRateWorker) initializeProviders() error {
	platforms := w.config.ExchangeRatePlatforms
	if len(platforms) == 0 {
		return fmt.Errorf("no exchange rate platforms configured")
	}

	for _, platform := range platforms {
		strategyConfig, ok := w.strategyConfigs[platform.Name]
		if !ok {
			return fmt.Errorf("missing strategy config for provider %s", platform.Name)
		}

		apiKey := w.providerConfigs[platform.Name]
		if apiKey == "" && platform.Name != "FawazExchange" {
			return fmt.Errorf("missing API key for provider %s", platform.Name)
		}

		baseURL := platform.BaseURL
		if baseURL == "" {
			baseURL = strategyConfig.defaultBaseURL
		}
		if baseURL == "" {
			return fmt.Errorf("missing base URL for provider %s", platform.Name)
		}

		w.providers[platform.Name] = NewGenericProvider(
			platform.Name,
			apiKey,
			baseURL,
			strategyConfig.urlBuilder,
			strategyConfig.responseParser,
		)
	}
	if len(w.providers) == 0 {
		return fmt.Errorf("no exchange rate providers initialized")
	}

	log.Info("Initialized exchange rate providers", "count", len(w.providers))
	return nil
}

type rateResult struct {
	platformGUID string
	platformName string
	rates        map[string]float64
	err          error
}

func (w *ExchangeRateWorker) FetchAndStoreRates() {
	batchTimestamp := time.Now()
	currencies, err := w.db.Currency.QueryActiveCurrency()
	if err != nil {
		log.Error("Failed to get currencies", "error", err)
		return
	}

	platforms := w.config.ExchangeRatePlatforms
	if len(platforms) == 0 {
		log.Warn("No enabled platforms found")
		return
	}

	currencyCodes := make([]string, 0, len(currencies))
	currencyMap := make(map[string]*database.Currency)
	seenCurrencyCodes := make(map[string]struct{}, len(currencies))
	for _, currency := range currencies {
		if _, exists := seenCurrencyCodes[currency.CurrencyCode]; exists {
			continue
		}
		seenCurrencyCodes[currency.CurrencyCode] = struct{}{}
		currencyCodes = append(currencyCodes, currency.CurrencyCode)
		currencyMap[currency.CurrencyCode] = currency
	}
	if len(currencyCodes) == 0 {
		log.Info("No enabled currencies found, bootstrapping from provider response")
	}

	var wg sync.WaitGroup
	rateChan := make(chan *rateResult, len(platforms))

	for _, platform := range platforms {
		provider, ok := w.providers[platform.Name]
		if !ok {
			log.Warn("Provider not initialized", "platform", platform.Name)
			continue
		}

		wg.Add(1)
		go func(p *config.ExchangeRatePlatformConfig, prov ExchangeRateProvider) {
			defer wg.Done()
			w.fetchRatesFromProvider(p, prov, currencyCodes, rateChan)
		}(&platform, provider)
	}
	go func() {
		wg.Wait()
		close(rateChan)
	}()

	// Process results
	successCount := 0
	for result := range rateChan {
		if result.err != nil {
			log.Error("Failed to fetch rates from provider",
				"platform", result.platformName,
				"error", result.err)
			continue
		}

		// Store rates in database with batch timestamp
		if err := w.storeRates(result, currencyMap, batchTimestamp); err != nil {
			log.Error("Failed to store rates",
				"platform", result.platformName,
				"error", err)
			continue
		}

		successCount++
		log.Info("Successfully fetched and stored rates",
			"platform", result.platformName,
			"currencies", len(result.rates))
	}

	log.Info("Exchange rate fetch cycle completed",
		"success", successCount,
		"total", len(platforms),
		"batch_timestamp", batchTimestamp)
}

func (w *ExchangeRateWorker) fetchRatesFromProvider(
	platform *config.ExchangeRatePlatformConfig,
	provider ExchangeRateProvider,
	currencyCodes []string,
	resultChan chan<- *rateResult,
) {

	rates, err := provider.FetchRates(context.Background(), w.config.BaseCurrency, currencyCodes)

	resultChan <- &rateResult{
		platformGUID: uuid.New().String(),
		platformName: platform.Name,
		rates:        rates,
		err:          err,
	}
}

func (w *ExchangeRateWorker) storeRates(result *rateResult, currencyMap map[string]*database.Currency, batchTimestamp time.Time) error {
	for currencyCode, baseRate := range result.rates {
		now := time.Now()
		baseRateDecimal := decimal.NewFromFloat(baseRate)
		existingCurrency, ok := currencyMap[currencyCode]
		currencyName := currencyCode
		guid := uuid.New().String()
		buySpread := 0.05
		sellSpread := 0.05
		isActive := true
		createdAt := now

		if ok {
			guid = existingCurrency.Guid
			if existingCurrency.CurrencyName != "" {
				currencyName = existingCurrency.CurrencyName
			}
			buySpread = existingCurrency.BuySpread
			sellSpread = existingCurrency.SellSpread
			isActive = existingCurrency.IsActive
			createdAt = existingCurrency.CreatedAt
		}

		currentRate := &database.Currency{
			Guid:         guid,
			CurrencyCode: currencyCode,
			CurrencyName: currencyName,
			Rate:         baseRateDecimal.InexactFloat64(),
			BuySpread:    buySpread,
			SellSpread:   sellSpread,
			IsActive:     isActive,
			CreatedAt:    createdAt,
			UpdatedAt:    batchTimestamp,
		}

		if err := w.db.Currency.StoreCurrency(currentRate); err != nil {
			log.Error("Failed to upsert current rate",
				"currency", currencyCode,
				"platform", result.platformName,
				"error", err)
			continue
		}
	}
	return nil
}

func exchangeRateAPIURLBuilder(baseURL, apiKey, baseCurrency string) string {
	return fmt.Sprintf("%s/%s/latest/%s", baseURL, apiKey, baseCurrency)
}

func exchangeRateAPIResponseParser(body []byte, targetCurrencies []string) (map[string]float64, error) {
	var result struct {
		Result          string             `json:"result"`
		ConversionRates map[string]float64 `json:"conversion_rates"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response error: %w", err)
	}

	if result.Result != "success" {
		return nil, fmt.Errorf("API returned error status: %s", result.Result)
	}
	if len(result.ConversionRates) == 0 {
		return nil, fmt.Errorf("conversion_rates is empty")
	}
	if len(targetCurrencies) == 0 {
		return result.ConversionRates, nil
	}

	rates := make(map[string]float64)
	for _, currency := range targetCurrencies {
		if rate, ok := result.ConversionRates[currency]; ok {
			rates[currency] = rate
		}
	}

	return rates, nil
}

func BuildStrategyConfigs(platformConfigs []config.ExchangeRatePlatformConfig) map[string]struct {
	urlBuilder     URLBuilder
	responseParser ResponseParser
	defaultBaseURL string
} {
	strategyMap := map[string]struct {
		urlBuilder     URLBuilder
		responseParser ResponseParser
		defaultBaseURL string
	}{
		"ExchangeRate-API": {
			urlBuilder:     exchangeRateAPIURLBuilder,
			responseParser: exchangeRateAPIResponseParser,
			defaultBaseURL: "https://v6.exchangerate-api.com/v6",
		},
		//"Fixer.io": {
		//	urlBuilder:     fiexerIOURLBuilder,
		//	responseParser: fixerIOResponseParser,
		//},
	}

	// 从配置文件构建策略配置
	result := make(map[string]struct {
		urlBuilder     URLBuilder
		responseParser ResponseParser
		defaultBaseURL string
	})

	for _, platformConfig := range platformConfigs {
		if strategy, exists := strategyMap[platformConfig.Name]; exists {
			defaultBaseURL := strategy.defaultBaseURL
			if platformConfig.BaseURL != "" {
				defaultBaseURL = platformConfig.BaseURL
			}
			result[platformConfig.Name] = struct {
				urlBuilder     URLBuilder
				responseParser ResponseParser
				defaultBaseURL string
			}{
				urlBuilder:     strategy.urlBuilder,
				responseParser: strategy.responseParser,
				defaultBaseURL: defaultBaseURL,
			}
		}
	}

	return result
}
