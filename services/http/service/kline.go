package service

import (
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/services/http/model"
	"github.com/ethereum/go-ethereum/log"
)

func (h HandleSvc) ListKlines(request *model.ListKlinesRequest) (*model.ListKlinesResponse, error) {
	if request.SymbolGuid == "" {
		return &model.ListKlinesResponse{
			Code:       returnCodeInvalidArgument,
			Message:    "symbol_guid is required",
			Pagination: buildPaginationResponse(defaultPage, defaultPageSize, 0),
		}, nil
	}
	if !isSupportedTimeframe(request.Timeframe) {
		return &model.ListKlinesResponse{
			Code:       returnCodeInvalidArgument,
			Message:    "unsupported timeframe",
			Pagination: buildPaginationResponse(defaultPage, defaultPageSize, 0),
		}, nil
	}

	page, pageSize := normalizePagination(request.Pagination)
	startAt, endAt, ok := parseTimeRange(request.StartTimestamp, request.EndTimestamp)
	if !ok {
		return &model.ListKlinesResponse{
			Code:       returnCodeInvalidArgument,
			Message:    "invalid time range",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	items, total, err := h.symbolKlineView.QuerySymbolKlineListByFilter(
		page,
		pageSize,
		request.SymbolGuid,
		request.OnlyActive,
		startAt,
		endAt,
	)
	if err != nil {
		log.Error("list klines fail", "symbol_guid", request.SymbolGuid, "error", err)
		return &model.ListKlinesResponse{
			Code:       returnCodeInternalError,
			Message:    "list klines fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]model.SymbolKlineItem, 0, len(items))
	for _, item := range items {
		result = append(result, buildSymbolKlineItem(item))
	}

	return &model.ListKlinesResponse{
		Code:       returnCodeSuccess,
		Message:    "list klines success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (h HandleSvc) ListExchangeKlines(request *model.ListExchangeKlinesRequest) (*model.ListExchangeKlinesResponse, error) {
	if request.SymbolGuid == "" {
		return &model.ListExchangeKlinesResponse{
			Code:       returnCodeInvalidArgument,
			Message:    "symbol_guid is required",
			Pagination: buildPaginationResponse(defaultPage, defaultPageSize, 0),
		}, nil
	}
	if !isSupportedTimeframe(request.Timeframe) {
		return &model.ListExchangeKlinesResponse{
			Code:       returnCodeInvalidArgument,
			Message:    "unsupported timeframe",
			Pagination: buildPaginationResponse(defaultPage, defaultPageSize, 0),
		}, nil
	}

	page, pageSize := normalizePagination(request.Pagination)
	startAt, endAt, ok := parseTimeRange(request.StartTimestamp, request.EndTimestamp)
	if !ok {
		return &model.ListExchangeKlinesResponse{
			Code:       returnCodeInvalidArgument,
			Message:    "invalid time range",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	items, total, err := h.exchangeSymbolKlineView.QueryExchangeSymbolKlineListByFilter(
		page,
		pageSize,
		request.ExchangeGuid,
		request.SymbolGuid,
		request.OnlyActive,
		startAt,
		endAt,
	)
	if err != nil {
		log.Error("list exchange klines fail", "symbol_guid", request.SymbolGuid, "exchange_guid", request.ExchangeGuid, "error", err)
		return &model.ListExchangeKlinesResponse{
			Code:       returnCodeInternalError,
			Message:    "list exchange klines fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]model.ExchangeSymbolKlineItem, 0, len(items))
	for _, item := range items {
		result = append(result, buildExchangeSymbolKlineItem(item))
	}

	return &model.ListExchangeKlinesResponse{
		Code:       returnCodeSuccess,
		Message:    "list exchange klines success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func buildSymbolKlineItem(item *database.SymbolKline) model.SymbolKlineItem {
	return model.SymbolKlineItem{
		Guid:       item.Guid,
		SymbolGuid: item.SymbolGuid,
		Timeframe:  klineTimeframe1m,
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

func buildExchangeSymbolKlineItem(item *database.ExchangeSymbolKline) model.ExchangeSymbolKlineItem {
	return model.ExchangeSymbolKlineItem{
		Guid:         item.Guid,
		ExchangeGuid: item.ExchangeGuid,
		SymbolGuid:   item.SymbolGuid,
		Timeframe:    klineTimeframe1m,
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
