package cryptoexchange

import (
	"testing"
	"time"

	"github.com/ccxt/ccxt/go/v4"
)

func TestBuildExchangeSymbolKlines(t *testing.T) {
	closedBefore := time.Date(2026, 4, 12, 12, 10, 0, 0, time.UTC)
	ohlcvs := []ccxt.OHLCV{
		{
			Timestamp: time.Date(2026, 4, 12, 12, 8, 0, 0, time.UTC).UnixMilli(),
			Open:      100,
			High:      110,
			Low:       90,
			Close:     105,
			Volume:    10.8,
		},
		{
			Timestamp: time.Date(2026, 4, 12, 12, 10, 0, 0, time.UTC).UnixMilli(),
			Open:      105,
			High:      115,
			Low:       95,
			Close:     110,
			Volume:    8.2,
		},
	}

	got := buildExchangeSymbolKlines("exchange-guid", "symbol-guid", ohlcvs, closedBefore)
	if len(got) != 1 {
		t.Fatalf("len(buildExchangeSymbolKlines) = %d, want 1", len(got))
	}
	if got[0].OpenPrice != "100" {
		t.Fatalf("OpenPrice = %q, want %q", got[0].OpenPrice, "100")
	}
	if got[0].ClosePrice != "105" {
		t.Fatalf("ClosePrice = %q, want %q", got[0].ClosePrice, "105")
	}
	if got[0].HighPrice != "110" {
		t.Fatalf("HighPrice = %q, want %q", got[0].HighPrice, "110")
	}
	if got[0].LowPrice != "90" {
		t.Fatalf("LowPrice = %q, want %q", got[0].LowPrice, "90")
	}
	if got[0].Volume != "10" {
		t.Fatalf("Volume = %q, want %q", got[0].Volume, "10")
	}
	if !got[0].CreatedAt.Equal(time.Date(2026, 4, 12, 12, 8, 0, 0, time.UTC)) {
		t.Fatalf("CreatedAt = %v", got[0].CreatedAt)
	}
}
