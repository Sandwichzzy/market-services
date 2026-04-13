package service

import (
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/services/http/model"
)

type RestService interface {
	GetSupportAsset(*model.SupportAssetRequest) (*model.SupportAssetResponse, error)
	ListMarketSymbols(*model.ListMarketSymbolsRequest) (*model.ListMarketSymbolsResponse, error)
	ListSymbolMarkets(*model.ListSymbolMarketsRequest) (*model.ListSymbolMarketsResponse, error)
	GetSymbolMarket(*model.GetSymbolMarketRequest) (*model.GetSymbolMarketResponse, error)
	ListCurrencies(*model.ListCurrenciesRequest) (*model.ListCurrenciesResponse, error)
	GetSymbolMarketCurrency(*model.GetSymbolMarketCurrencyRequest) (*model.GetSymbolMarketCurrencyResponse, error)
	ListKlines(*model.ListKlinesRequest) (*model.ListKlinesResponse, error)
	ListExchangeKlines(*model.ListExchangeKlinesRequest) (*model.ListExchangeKlinesResponse, error)
}

type HandleSvc struct {
	assetView                database.AssetView
	currencyView             database.CurrencyView
	symbolView               database.SymbolView
	symbolMarketView         database.SymbolMarketView
	symbolMarketCurrencyView database.SymbolMarketCurrencyView
	symbolKlineView          database.SymbolKlineView
	exchangeSymbolKlineView  database.ExchangeSymbolKlineView
}

func NewHandleSvc(
	assetView database.AssetView,
	currencyView database.CurrencyView,
	symbolView database.SymbolView,
	symbolMarketView database.SymbolMarketView,
	symbolMarketCurrencyView database.SymbolMarketCurrencyView,
	symbolKlineView database.SymbolKlineView,
	exchangeSymbolKlineView database.ExchangeSymbolKlineView,
) RestService {
	return &HandleSvc{
		assetView:                assetView,
		currencyView:             currencyView,
		symbolView:               symbolView,
		symbolMarketView:         symbolMarketView,
		symbolMarketCurrencyView: symbolMarketCurrencyView,
		symbolKlineView:          symbolKlineView,
		exchangeSymbolKlineView:  exchangeSymbolKlineView,
	}
}
