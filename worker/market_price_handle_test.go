package worker

import (
	"math"
	"testing"
	"time"

	"github.com/Sandwichzzy/market-services/database"
)

func TestAggregateMarketPriceRecords(t *testing.T) {
	symbolBTC := &database.Symbol{Guid: "btc-guid", SymbolName: "BTC/USDT"}
	symbolETH := &database.Symbol{Guid: "eth-guid", SymbolName: "ETH/USDT"}

	got, err := aggregateMarketPriceRecords([]marketPriceRecord{
		{
			symbol:           symbolBTC,
			price:            "100",
			askPrice:         "101",
			bidPrice:         "99",
			baseAssetSymbol:  "BTC",
			quoteAssetSymbol: "USDT",
		},
		{
			symbol:           symbolBTC,
			price:            "110",
			askPrice:         "100.5",
			bidPrice:         "100",
			baseAssetSymbol:  "BTC",
			quoteAssetSymbol: "USDT",
		},
		{
			symbol:           symbolETH,
			price:            "200",
			askPrice:         "201",
			bidPrice:         "199",
			baseAssetSymbol:  "ETH",
			quoteAssetSymbol: "USDT",
		},
	})
	if err != nil {
		t.Fatalf("aggregateMarketPriceRecords: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	if got[0].symbol.Guid != "btc-guid" {
		t.Fatalf("first symbol guid = %q, want %q", got[0].symbol.Guid, "btc-guid")
	}
	if got[0].price != "105" {
		t.Fatalf("btc price = %q, want %q", got[0].price, "105")
	}
	if got[0].askPrice != "100.5" {
		t.Fatalf("btc askPrice = %q, want %q", got[0].askPrice, "100.5")
	}
	if got[0].bidPrice != "100" {
		t.Fatalf("btc bidPrice = %q, want %q", got[0].bidPrice, "100")
	}

	if got[1].symbol.Guid != "eth-guid" {
		t.Fatalf("second symbol guid = %q, want %q", got[1].symbol.Guid, "eth-guid")
	}
	if got[1].price != "200" {
		t.Fatalf("eth price = %q, want %q", got[1].price, "200")
	}
}

func TestBuildSymbolMarketCurrency(t *testing.T) {
	snapshotTime := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	currency := &database.Currency{
		Guid:         "currency-guid",
		CurrencyCode: "CNY",
		Rate:         7.2,
		IsActive:     true,
	}

	got, err := buildSymbolMarketCurrency("symbol-guid", currency, "100.5", "101", "100", snapshotTime)
	if err != nil {
		t.Fatalf("buildSymbolMarketCurrency: %v", err)
	}
	if got.SymbolGuid != "symbol-guid" {
		t.Fatalf("SymbolGuid = %q, want %q", got.SymbolGuid, "symbol-guid")
	}
	if got.CurrencyGuid != "currency-guid" {
		t.Fatalf("CurrencyGuid = %q, want %q", got.CurrencyGuid, "currency-guid")
	}
	if got.Price != "723.6" {
		t.Fatalf("Price = %q, want %q", got.Price, "723.6")
	}
	if got.AskPrice != "727.2" {
		t.Fatalf("AskPrice = %q, want %q", got.AskPrice, "727.2")
	}
	if got.BidPrice != "720" {
		t.Fatalf("BidPrice = %q, want %q", got.BidPrice, "720")
	}
}

func TestSupportsFiatConversion(t *testing.T) {
	tests := []struct {
		name             string
		quoteAssetSymbol string
		baseCurrency     string
		want             bool
	}{
		{name: "same currency", quoteAssetSymbol: "USD", baseCurrency: "USD", want: true},
		{name: "usdt treated as usd", quoteAssetSymbol: "USDT", baseCurrency: "USD", want: true},
		{name: "unsupported quote asset", quoteAssetSymbol: "ETH", baseCurrency: "USD", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := supportsFiatConversion(tt.quoteAssetSymbol, tt.baseCurrency); got != tt.want {
				t.Fatalf("supportsFiatConversion(%q, %q) = %v, want %v", tt.quoteAssetSymbol, tt.baseCurrency, got, tt.want)
			}
		})
	}
}

func TestCalculateRate(t *testing.T) {
	tests := []struct {
		name         string
		currentPrice float64
		firstPrice   float64
		want         float64
	}{
		{name: "normal increase", currentPrice: 71550.75, firstPrice: 70000, want: 0.02215357142857143},
		{name: "normal decrease", currentPrice: 63000, firstPrice: 70000, want: -0.1},
		{name: "zero first price", currentPrice: 1, firstPrice: 0, want: 0},
		{name: "clamp upper bound", currentPrice: 1000, firstPrice: 10, want: 10},
		{name: "clamp lower bound", currentPrice: -100, firstPrice: 10, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateRate(tt.currentPrice, tt.firstPrice)
			if math.Abs(got-tt.want) > 1e-12 {
				t.Fatalf("calculateRate(%v, %v) = %v, want %v", tt.currentPrice, tt.firstPrice, got, tt.want)
			}
		})
	}
}

func TestClampRate(t *testing.T) {
	tests := []struct {
		name string
		rate float64
		want float64
	}{
		{name: "keep value", rate: 0.25, want: 0.25},
		{name: "nan to zero", rate: math.NaN(), want: 0},
		{name: "positive inf to zero", rate: math.Inf(1), want: 0},
		{name: "negative inf to zero", rate: math.Inf(-1), want: 0},
		{name: "below min", rate: -2, want: -1},
		{name: "above max", rate: 20, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampRate(tt.rate)
			if got != tt.want {
				t.Fatalf("clampRate(%v) = %v, want %v", tt.rate, got, tt.want)
			}
		})
	}
}
