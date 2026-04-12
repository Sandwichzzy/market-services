package worker

import (
	"testing"
	"time"

	"github.com/Sandwichzzy/market-services/database"
)

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
