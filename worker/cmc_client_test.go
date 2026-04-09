package worker

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Sandwichzzy/market-services/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCoinMarketCapClientFetchQuotes(t *testing.T) {
	var receivedAPIKey string
	var receivedQuery string

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		receivedAPIKey = r.Header.Get("X-CMC_PRO_API_KEY")
		receivedQuery = r.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"BTC": {
						"quote": {
							"USD": {
								"market_cap": 123456.99,
								"volume_24h": 4567.89
							}
						}
					}
				}
			}`)),
		}, nil
	})}

	client, err := newCoinMarketCapClient("cmc-key", httpClient)
	if err != nil {
		t.Fatalf("newCoinMarketCapClient: %v", err)
	}
	client.baseURL = "https://example.com/cmc"

	quotes, err := client.FetchQuotes(context.Background(), []string{"BTC", "BTC"}, "USD")
	if err != nil {
		t.Fatalf("FetchQuotes: %v", err)
	}
	if receivedAPIKey != "cmc-key" {
		t.Fatalf("X-CMC_PRO_API_KEY = %q, want %q", receivedAPIKey, "cmc-key")
	}
	if receivedQuery != "convert=USD&symbol=BTC" && receivedQuery != "symbol=BTC&convert=USD" {
		t.Fatalf("unexpected query string: %q", receivedQuery)
	}
	if quotes["BTC"].MarketCap != "123456" {
		t.Fatalf("MarketCap = %q, want %q", quotes["BTC"].MarketCap, "123456")
	}
	if quotes["BTC"].Volume24h != "4567" {
		t.Fatalf("Volume24h = %q, want %q", quotes["BTC"].Volume24h, "4567")
	}
}

func TestCoinMarketCapClientFetchQuotesErrors(t *testing.T) {
	t.Run("fails on empty data", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
			}, nil
		})}

		client, err := newCoinMarketCapClient("cmc-key", httpClient)
		if err != nil {
			t.Fatalf("newCoinMarketCapClient: %v", err)
		}
		client.baseURL = "https://example.com/cmc"

		if _, err := client.FetchQuotes(context.Background(), []string{"BTC"}, "USD"); err == nil {
			t.Fatal("expected empty data error")
		}
	})

	t.Run("supports array payload shape", func(t *testing.T) {
		httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(`{
					"data": {
						"BTC": [{
							"quote": {
								"USD": {
									"market_cap": 123456.99,
									"volume_24h": 4567.89
								}
							}
						}]
					}
				}`)),
			}, nil
		})}

		client, err := newCoinMarketCapClient("cmc-key", httpClient)
		if err != nil {
			t.Fatalf("newCoinMarketCapClient: %v", err)
		}
		client.baseURL = "https://example.com/cmc"

		quotes, err := client.FetchQuotes(context.Background(), []string{"BTC"}, "USD")
		if err != nil {
			t.Fatalf("FetchQuotes: %v", err)
		}
		if quotes["BTC"].MarketCap != "123456" {
			t.Fatalf("MarketCap = %q, want %q", quotes["BTC"].MarketCap, "123456")
		}
	})

	t.Run("fails on missing API key", func(t *testing.T) {
		if _, err := newCoinMarketCapClient("", nil); err == nil {
			t.Fatal("expected missing API key error")
		}
	})
}

func TestNewMarketPriceHandleRequiresCMCKey(t *testing.T) {
	cfg := &config.Config{}

	if _, err := NewMarketPriceHandle(nil, nil, cfg, func(error) {}); err == nil {
		t.Fatal("expected missing CMC key error")
	}
}

func TestDecimalToUintString(t *testing.T) {
	if got := decimalToUintString(123.99); got != "123" {
		t.Fatalf("decimalToUintString(123.99) = %q, want %q", got, "123")
	}
	if got := decimalToUintString(0); got != "0" {
		t.Fatalf("decimalToUintString(0) = %q, want %q", got, "0")
	}
}
