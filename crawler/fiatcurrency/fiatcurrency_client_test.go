package fiatcurrency

import (
	"testing"

	"github.com/Sandwichzzy/market-services/config"
)

func TestExchangeRateAPIResponseParser(t *testing.T) {
	successBody := []byte(`{"result":"success","conversion_rates":{"USD":1,"CNY":7.2,"EUR":0.92}}`)

	t.Run("returns all rates when target list empty", func(t *testing.T) {
		rates, err := exchangeRateAPIResponseParser(successBody, nil)
		if err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if len(rates) != 3 {
			t.Fatalf("len(rates) = %d, want 3", len(rates))
		}
		if rates["CNY"] != 7.2 {
			t.Fatalf("CNY rate = %v, want %v", rates["CNY"], 7.2)
		}
	})

	t.Run("filters requested currencies", func(t *testing.T) {
		rates, err := exchangeRateAPIResponseParser(successBody, []string{"CNY"})
		if err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if len(rates) != 1 {
			t.Fatalf("len(rates) = %d, want 1", len(rates))
		}
		if rates["CNY"] != 7.2 {
			t.Fatalf("CNY rate = %v, want %v", rates["CNY"], 7.2)
		}
	})

	t.Run("fails when api reports error", func(t *testing.T) {
		body := []byte(`{"result":"error"}`)
		if _, err := exchangeRateAPIResponseParser(body, nil); err == nil {
			t.Fatal("expected error status to fail")
		}
	})

	t.Run("fails when rates missing", func(t *testing.T) {
		body := []byte(`{"result":"success","conversion_rates":{}}`)
		if _, err := exchangeRateAPIResponseParser(body, nil); err == nil {
			t.Fatal("expected empty conversion rates to fail")
		}
	})
}

func TestNewExchangeRateWorkerRequiresAPIKey(t *testing.T) {
	cfg := &config.Config{
		BaseCurrency: "USD",
		ExchangeRatePlatforms: []config.ExchangeRatePlatformConfig{
			{Name: "ExchangeRate-API"},
		},
	}

	_, err := NewExchangeRateWorker(nil, cfg, map[string]string{}, BuildStrategyConfigs(cfg.ExchangeRatePlatforms))
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestBuildStrategyConfigsUsesDefaultBaseURL(t *testing.T) {
	platforms := []config.ExchangeRatePlatformConfig{
		{Name: "ExchangeRate-API"},
	}

	strategies := BuildStrategyConfigs(platforms)
	strategy, ok := strategies["ExchangeRate-API"]
	if !ok {
		t.Fatal("expected ExchangeRate-API strategy")
	}
	if strategy.defaultBaseURL == "" {
		t.Fatal("expected default base URL to be set")
	}
}
