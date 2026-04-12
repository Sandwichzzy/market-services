package worker

import (
	"testing"
	"time"

	"github.com/Sandwichzzy/market-services/database"
)

func TestBuildSymbolKlinesFromExchangeKlines(t *testing.T) {
	closedBefore := time.Date(2026, 4, 12, 12, 10, 0, 0, time.UTC)
	rows := []*database.ExchangeSymbolKline{
		{
			SymbolGuid: "symbol-guid",
			OpenPrice:  "100",
			ClosePrice: "110",
			HighPrice:  "120",
			LowPrice:   "95",
			Volume:     "10",
			CreatedAt:  time.Date(2026, 4, 12, 12, 8, 0, 0, time.UTC),
		},
		{
			SymbolGuid: "symbol-guid",
			OpenPrice:  "102",
			ClosePrice: "112",
			HighPrice:  "125",
			LowPrice:   "94",
			Volume:     "7",
			CreatedAt:  time.Date(2026, 4, 12, 12, 8, 0, 0, time.UTC),
		},
		{
			SymbolGuid: "symbol-guid",
			OpenPrice:  "200",
			ClosePrice: "205",
			HighPrice:  "210",
			LowPrice:   "190",
			Volume:     "5",
			CreatedAt:  time.Date(2026, 4, 12, 12, 10, 0, 0, time.UTC),
		},
	}

	got, err := buildSymbolKlinesFromExchangeKlines(rows, closedBefore)
	if err != nil {
		t.Fatalf("buildSymbolKlinesFromExchangeKlines: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].OpenPrice != "101" {
		t.Fatalf("OpenPrice = %q, want %q", got[0].OpenPrice, "101")
	}
	if got[0].ClosePrice != "111" {
		t.Fatalf("ClosePrice = %q, want %q", got[0].ClosePrice, "111")
	}
	if got[0].HighPrice != "125" {
		t.Fatalf("HighPrice = %q, want %q", got[0].HighPrice, "125")
	}
	if got[0].LowPrice != "94" {
		t.Fatalf("LowPrice = %q, want %q", got[0].LowPrice, "94")
	}
	if got[0].Volume != "17" {
		t.Fatalf("Volume = %q, want %q", got[0].Volume, "17")
	}
}
