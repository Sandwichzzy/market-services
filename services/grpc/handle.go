package grpc

import (
	"context"
	"strconv"
	"time"

	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/services/grpc/proto"
	"github.com/ethereum/go-ethereum/log"
)

const (
	returnCodeSuccess         = 200
	returnCodeInvalidArgument = 400
	returnCodeNotFound        = 404
	returnCodeInternalError   = 4000
)

func (ms *MarketRpcService) GetSupportAsset(ctx context.Context, req *proto.SupportAssetRequest) (*proto.SupportAssetResponse, error) {
	assetList, err := ms.db.Asset.QueryAssets()
	if err != nil {
		log.Error("get support asset fail", "error", err)
		return &proto.SupportAssetResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "get support asset fail",
		}, nil
	}

	var returnAssetList []*proto.AssetItem
	for _, asset := range assetList {
		assetItem := &proto.AssetItem{
			Guid:        asset.Guid,
			AssetName:   asset.AssetName,
			AssetLogo:   asset.AssetLogo,
			AssetSymbol: asset.AssetSymbol,
		}
		returnAssetList = append(returnAssetList, assetItem)
	}

	return &proto.SupportAssetResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "get support asset success",
		Asset:      returnAssetList,
	}, nil
}

func (ms *MarketRpcService) ListMarketSymbols(ctx context.Context, req *proto.ListMarketSymbolsRequest) (*proto.ListMarketSymbolsResponse, error) {
	page, pageSize := normalizePagination(req.GetPagination())
	symbols, total, err := ms.db.Symbol.QuerySymbolListByFilter(
		page,
		pageSize,
		req.GetOnlyActive(),
		req.GetBaseAssetGuid(),
		req.GetQuoteAssetGuid(),
		req.GetMarketType(),
	)
	if err != nil {
		log.Error("list market symbols fail", "error", err)
		return &proto.ListMarketSymbolsResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "list market symbols fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]*proto.MarketSymbolItem, 0, len(symbols))
	for _, item := range symbols {
		result = append(result, &proto.MarketSymbolItem{
			Guid:           item.Guid,
			SymbolName:     item.SymbolName,
			BaseAssetGuid:  item.BaseAssetGuid,
			QuoteAssetGuid: item.QuoteAssetGuid,
			MarketType:     item.MarketType,
			IsActive:       item.IsActive,
			CreatedAt:      item.CreatedAt.UnixMilli(),
			UpdatedAt:      item.UpdatedAt.UnixMilli(),
		})
	}

	return &proto.ListMarketSymbolsResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "list market symbols success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (ms *MarketRpcService) ListSymbolMarkets(ctx context.Context, req *proto.ListSymbolMarketsRequest) (*proto.ListSymbolMarketsResponse, error) {
	page, pageSize := normalizePagination(req.GetPagination())
	items, total, err := ms.db.SymbolMarket.QuerySymbolMarketListByFilter(
		page,
		pageSize,
		req.GetSymbolGuid(),
		req.GetOnlyActive(),
	)
	if err != nil {
		log.Error("list symbol markets fail", "error", err)
		return &proto.ListSymbolMarketsResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "list symbol markets fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]*proto.SymbolMarketItem, 0, len(items))
	for _, item := range items {
		result = append(result, buildSymbolMarketItem(item))
	}

	return &proto.ListSymbolMarketsResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "list symbol markets success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (ms *MarketRpcService) GetSymbolMarket(ctx context.Context, req *proto.GetSymbolMarketRequest) (*proto.GetSymbolMarketResponse, error) {
	if req.GetSymbolGuid() == "" {
		return &proto.GetSymbolMarketResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "symbol_guid is required",
		}, nil
	}

	item, err := ms.db.SymbolMarket.QueryLatestSymbolMarketBySymbol(req.GetSymbolGuid(), true)
	if err != nil {
		log.Error("get symbol market fail", "symbol_guid", req.GetSymbolGuid(), "error", err)
		return &proto.GetSymbolMarketResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "get symbol market fail",
		}, nil
	}
	if item == nil {
		return &proto.GetSymbolMarketResponse{
			ReturnCode: returnCodeNotFound,
			Message:    "symbol market not found",
		}, nil
	}

	return &proto.GetSymbolMarketResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "get symbol market success",
		Result:     buildSymbolMarketItem(item),
	}, nil
}

func (ms *MarketRpcService) ListCurrencies(ctx context.Context, req *proto.ListCurrenciesRequest) (*proto.ListCurrenciesResponse, error) {
	page, pageSize := normalizePagination(req.GetPagination())
	items, total, err := ms.db.Currency.QueryCurrencyListByFilter(
		page,
		pageSize,
		req.GetOnlyActive(),
		req.GetCurrencyCode(),
	)
	if err != nil {
		log.Error("list currencies fail", "error", err)
		return &proto.ListCurrenciesResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "list currencies fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]*proto.CurrencyItem, 0, len(items))
	for _, item := range items {
		result = append(result, &proto.CurrencyItem{
			Guid:         item.Guid,
			CurrencyName: item.CurrencyName,
			CurrencyCode: item.CurrencyCode,
			Rate:         formatFloat(item.Rate),
			BuySpread:    formatFloat(item.BuySpread),
			SellSpread:   formatFloat(item.SellSpread),
			IsActive:     item.IsActive,
			CreatedAt:    item.CreatedAt.UnixMilli(),
			UpdatedAt:    item.UpdatedAt.UnixMilli(),
		})
	}

	return &proto.ListCurrenciesResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "list currencies success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (ms *MarketRpcService) GetSymbolMarketCurrency(ctx context.Context, req *proto.GetSymbolMarketCurrencyRequest) (*proto.GetSymbolMarketCurrencyResponse, error) {
	if req.GetSymbolGuid() == "" || req.GetCurrencyGuid() == "" {
		return &proto.GetSymbolMarketCurrencyResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "symbol_guid and currency_guid are required",
		}, nil
	}

	item, err := ms.db.SymbolMarketCurrency.QuerySymbolMarketCurrency(req.GetSymbolGuid(), req.GetCurrencyGuid(), true)
	if err != nil {
		log.Error("get symbol market currency fail", "symbol_guid", req.GetSymbolGuid(), "currency_guid", req.GetCurrencyGuid(), "error", err)
		return &proto.GetSymbolMarketCurrencyResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "get symbol market currency fail",
		}, nil
	}
	if item == nil {
		return &proto.GetSymbolMarketCurrencyResponse{
			ReturnCode: returnCodeNotFound,
			Message:    "symbol market currency not found",
		}, nil
	}

	return &proto.GetSymbolMarketCurrencyResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "get symbol market currency success",
		Result:     buildSymbolMarketCurrencyItem(item),
	}, nil
}

func (ms *MarketRpcService) ListKlines(ctx context.Context, req *proto.ListKlinesRequest) (*proto.ListKlinesResponse, error) {
	if req.GetSymbolGuid() == "" {
		return &proto.ListKlinesResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "symbol_guid is required",
			Pagination: buildPaginationResponse(1, 10, 0),
		}, nil
	}
	if !isSupportedTimeframe(req.GetTimeframe()) {
		return &proto.ListKlinesResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "unsupported timeframe",
			Pagination: buildPaginationResponse(1, 10, 0),
		}, nil
	}

	page, pageSize := normalizePagination(req.GetPagination())
	startAt, endAt, ok := parseTimeRange(req.GetStartTimestamp(), req.GetEndTimestamp())
	if !ok {
		return &proto.ListKlinesResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "invalid time range",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	items, total, err := ms.db.SymbolKline.QuerySymbolKlineListByFilter(
		page,
		pageSize,
		req.GetSymbolGuid(),
		req.GetOnlyActive(),
		startAt,
		endAt,
	)
	if err != nil {
		log.Error("list klines fail", "symbol_guid", req.GetSymbolGuid(), "error", err)
		return &proto.ListKlinesResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "list klines fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]*proto.SymbolKlineItem, 0, len(items))
	for _, item := range items {
		result = append(result, buildSymbolKlineItem(item))
	}

	return &proto.ListKlinesResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "list klines success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (ms *MarketRpcService) ListExchangeKlines(ctx context.Context, req *proto.ListExchangeKlinesRequest) (*proto.ListExchangeKlinesResponse, error) {
	if req.GetSymbolGuid() == "" {
		return &proto.ListExchangeKlinesResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "symbol_guid is required",
			Pagination: buildPaginationResponse(1, 10, 0),
		}, nil
	}
	if !isSupportedTimeframe(req.GetTimeframe()) {
		return &proto.ListExchangeKlinesResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "unsupported timeframe",
			Pagination: buildPaginationResponse(1, 10, 0),
		}, nil
	}

	page, pageSize := normalizePagination(req.GetPagination())
	startAt, endAt, ok := parseTimeRange(req.GetStartTimestamp(), req.GetEndTimestamp())
	if !ok {
		return &proto.ListExchangeKlinesResponse{
			ReturnCode: returnCodeInvalidArgument,
			Message:    "invalid time range",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	items, total, err := ms.db.ExchangeSymbolKline.QueryExchangeSymbolKlineListByFilter(
		page,
		pageSize,
		req.GetExchangeGuid(),
		req.GetSymbolGuid(),
		req.GetOnlyActive(),
		startAt,
		endAt,
	)
	if err != nil {
		log.Error("list exchange klines fail", "symbol_guid", req.GetSymbolGuid(), "exchange_guid", req.GetExchangeGuid(), "error", err)
		return &proto.ListExchangeKlinesResponse{
			ReturnCode: returnCodeInternalError,
			Message:    "list exchange klines fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]*proto.ExchangeSymbolKlineItem, 0, len(items))
	for _, item := range items {
		result = append(result, buildExchangeSymbolKlineItem(item))
	}

	return &proto.ListExchangeKlinesResponse{
		ReturnCode: returnCodeSuccess,
		Message:    "list exchange klines success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func normalizePagination(req *proto.PaginationRequest) (int64, int64) {
	page := int64(1)
	pageSize := int64(10)
	if req == nil {
		return page, pageSize
	}
	if req.GetPage() > 0 {
		page = req.GetPage()
	}
	if req.GetPageSize() > 0 {
		pageSize = req.GetPageSize()
	}
	return page, pageSize
}

func buildPaginationResponse(page, pageSize, total int64) *proto.PaginationResponse {
	return &proto.PaginationResponse{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}

func buildSymbolMarketItem(item *database.SymbolMarket) *proto.SymbolMarketItem {
	return &proto.SymbolMarketItem{
		Guid:       item.Guid,
		SymbolGuid: item.SymbolGuid,
		Price:      item.Price,
		AskPrice:   item.AskPrice,
		BidPrice:   item.BidPrice,
		Volume:     item.Volume,
		MarketCap:  item.MarketCap,
		Radio:      item.Radio,
		IsActive:   item.IsActive,
		CreatedAt:  item.CreatedAt.UnixMilli(),
		UpdatedAt:  item.UpdatedAt.UnixMilli(),
	}
}

func buildSymbolMarketCurrencyItem(item *database.SymbolMarketCurrency) *proto.SymbolMarketCurrencyItem {
	return &proto.SymbolMarketCurrencyItem{
		Guid:         item.Guid,
		SymbolGuid:   item.SymbolGuid,
		CurrencyGuid: item.CurrencyGuid,
		Price:        item.Price,
		AskPrice:     item.AskPrice,
		BidPrice:     item.BidPrice,
		IsActive:     item.IsActive,
		CreatedAt:    item.CreatedAt.UnixMilli(),
		UpdatedAt:    item.UpdatedAt.UnixMilli(),
	}
}

func buildSymbolKlineItem(item *database.SymbolKline) *proto.SymbolKlineItem {
	return &proto.SymbolKlineItem{
		Guid:       item.Guid,
		SymbolGuid: item.SymbolGuid,
		Timeframe:  proto.KlineTimeframe_KLINE_TIMEFRAME_1M,
		OpenPrice:  item.OpenPrice,
		ClosePrice: item.ClosePrice,
		HighPrice:  item.HighPrice,
		LowPrice:   item.LowPrice,
		Volume:     item.Volume,
		MarketCap:  item.MarketCap,
		IsActive:   item.IsActive,
		CreatedAt:  item.CreatedAt.UnixMilli(),
		UpdatedAt:  item.UpdatedAt.UnixMilli(),
	}
}

func buildExchangeSymbolKlineItem(item *database.ExchangeSymbolKline) *proto.ExchangeSymbolKlineItem {
	return &proto.ExchangeSymbolKlineItem{
		Guid:         item.Guid,
		ExchangeGuid: item.ExchangeGuid,
		SymbolGuid:   item.SymbolGuid,
		Timeframe:    proto.KlineTimeframe_KLINE_TIMEFRAME_1M,
		OpenPrice:    item.OpenPrice,
		ClosePrice:   item.ClosePrice,
		HighPrice:    item.HighPrice,
		LowPrice:     item.LowPrice,
		Volume:       item.Volume,
		MarketCap:    item.MarketCap,
		IsActive:     item.IsActive,
		CreatedAt:    item.CreatedAt.UnixMilli(),
		UpdatedAt:    item.UpdatedAt.UnixMilli(),
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func isSupportedTimeframe(timeframe proto.KlineTimeframe) bool {
	return timeframe == proto.KlineTimeframe_KLINE_TIMEFRAME_UNSPECIFIED ||
		timeframe == proto.KlineTimeframe_KLINE_TIMEFRAME_1M
}

func parseTimeRange(startTimestamp, endTimestamp int64) (*time.Time, *time.Time, bool) {
	var startAt *time.Time
	var endAt *time.Time
	if startTimestamp > 0 {
		start := time.UnixMilli(startTimestamp).UTC()
		startAt = &start
	}
	if endTimestamp > 0 {
		end := time.UnixMilli(endTimestamp).UTC()
		endAt = &end
	}
	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return nil, nil, false
	}
	return startAt, endAt, true
}
