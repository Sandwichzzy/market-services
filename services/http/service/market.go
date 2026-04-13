package service

import (
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/services/http/model"
	"github.com/ethereum/go-ethereum/log"
)

func (h HandleSvc) ListMarketSymbols(request *model.ListMarketSymbolsRequest) (*model.ListMarketSymbolsResponse, error) {
	page, pageSize := normalizePagination(request.Pagination)
	items, total, err := h.symbolView.QuerySymbolListByFilter(
		page,
		pageSize,
		request.OnlyActive,
		request.BaseAssetGuid,
		request.QuoteAssetGuid,
		request.MarketType,
	)
	if err != nil {
		log.Error("list market symbols fail", "error", err)
		return &model.ListMarketSymbolsResponse{
			Code:       returnCodeInternalError,
			Message:    "list market symbols fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]model.MarketSymbolItem, 0, len(items))
	for _, item := range items {
		result = append(result, model.MarketSymbolItem{
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

	return &model.ListMarketSymbolsResponse{
		Code:       returnCodeSuccess,
		Message:    "list market symbols success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (h HandleSvc) ListSymbolMarkets(request *model.ListSymbolMarketsRequest) (*model.ListSymbolMarketsResponse, error) {
	page, pageSize := normalizePagination(request.Pagination)
	items, total, err := h.symbolMarketView.QuerySymbolMarketListByFilter(
		page,
		pageSize,
		request.SymbolGuid,
		request.OnlyActive,
	)
	if err != nil {
		log.Error("list symbol markets fail", "error", err)
		return &model.ListSymbolMarketsResponse{
			Code:       returnCodeInternalError,
			Message:    "list symbol markets fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]model.SymbolMarketItem, 0, len(items))
	for _, item := range items {
		result = append(result, buildSymbolMarketItem(item))
	}

	return &model.ListSymbolMarketsResponse{
		Code:       returnCodeSuccess,
		Message:    "list symbol markets success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (h HandleSvc) GetSymbolMarket(request *model.GetSymbolMarketRequest) (*model.GetSymbolMarketResponse, error) {
	if request.SymbolGuid == "" {
		return &model.GetSymbolMarketResponse{
			Code:    returnCodeInvalidArgument,
			Message: "symbol_guid is required",
		}, nil
	}

	item, err := h.symbolMarketView.QueryLatestSymbolMarketBySymbol(request.SymbolGuid, true)
	if err != nil {
		log.Error("get symbol market fail", "symbol_guid", request.SymbolGuid, "error", err)
		return &model.GetSymbolMarketResponse{
			Code:    returnCodeInternalError,
			Message: "get symbol market fail",
		}, nil
	}
	if item == nil {
		return &model.GetSymbolMarketResponse{
			Code:    returnCodeNotFound,
			Message: "symbol market not found",
		}, nil
	}

	result := buildSymbolMarketItem(item)
	return &model.GetSymbolMarketResponse{
		Code:    returnCodeSuccess,
		Message: "get symbol market success",
		Result:  &result,
	}, nil
}

func buildSymbolMarketItem(item *database.SymbolMarket) model.SymbolMarketItem {
	return model.SymbolMarketItem{
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
