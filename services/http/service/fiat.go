package service

import (
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/services/http/model"
	"github.com/ethereum/go-ethereum/log"
)

func (h HandleSvc) ListCurrencies(request *model.ListCurrenciesRequest) (*model.ListCurrenciesResponse, error) {
	page, pageSize := normalizePagination(request.Pagination)
	items, total, err := h.currencyView.QueryCurrencyListByFilter(
		page,
		pageSize,
		request.OnlyActive,
		request.CurrencyCode,
	)
	if err != nil {
		log.Error("list currencies fail", "error", err)
		return &model.ListCurrenciesResponse{
			Code:       returnCodeInternalError,
			Message:    "list currencies fail",
			Pagination: buildPaginationResponse(page, pageSize, 0),
		}, nil
	}

	result := make([]model.CurrencyItem, 0, len(items))
	for _, item := range items {
		result = append(result, model.CurrencyItem{
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

	return &model.ListCurrenciesResponse{
		Code:       returnCodeSuccess,
		Message:    "list currencies success",
		Pagination: buildPaginationResponse(page, pageSize, total),
		Result:     result,
	}, nil
}

func (h HandleSvc) GetSymbolMarketCurrency(request *model.GetSymbolMarketCurrencyRequest) (*model.GetSymbolMarketCurrencyResponse, error) {
	if request.SymbolGuid == "" || request.CurrencyGuid == "" {
		return &model.GetSymbolMarketCurrencyResponse{
			Code:    returnCodeInvalidArgument,
			Message: "symbol_guid and currency_guid are required",
		}, nil
	}

	item, err := h.symbolMarketCurrencyView.QuerySymbolMarketCurrency(request.SymbolGuid, request.CurrencyGuid, true)
	if err != nil {
		log.Error("get symbol market currency fail", "symbol_guid", request.SymbolGuid, "currency_guid", request.CurrencyGuid, "error", err)
		return &model.GetSymbolMarketCurrencyResponse{
			Code:    returnCodeInternalError,
			Message: "get symbol market currency fail",
		}, nil
	}
	if item == nil {
		return &model.GetSymbolMarketCurrencyResponse{
			Code:    returnCodeNotFound,
			Message: "symbol market currency not found",
		}, nil
	}

	result := buildSymbolMarketCurrencyItem(item)
	return &model.GetSymbolMarketCurrencyResponse{
		Code:    returnCodeSuccess,
		Message: "get symbol market currency success",
		Result:  &result,
	}, nil
}

func buildSymbolMarketCurrencyItem(item *database.SymbolMarketCurrency) model.SymbolMarketCurrencyItem {
	return model.SymbolMarketCurrencyItem{
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
