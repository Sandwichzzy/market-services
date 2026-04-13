package model

type ListMarketSymbolsRequest struct {
	Pagination     PaginationRequest `json:"pagination"`
	OnlyActive     bool              `json:"only_active"`
	BaseAssetGuid  string            `json:"base_asset_guid"`
	QuoteAssetGuid string            `json:"quote_asset_guid"`
	MarketType     string            `json:"market_type"`
}

type MarketSymbolItem struct {
	Guid           string `json:"guid"`
	SymbolName     string `json:"symbol_name"`
	BaseAssetGuid  string `json:"base_asset_guid"`
	QuoteAssetGuid string `json:"quote_asset_guid"`
	MarketType     string `json:"market_type"`
	IsActive       bool   `json:"is_active"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type ListMarketSymbolsResponse struct {
	Code       uint64             `json:"code"`
	Message    string             `json:"message"`
	Pagination PaginationResponse `json:"pagination"`
	Result     []MarketSymbolItem `json:"result"`
}

type ListSymbolMarketsRequest struct {
	Pagination PaginationRequest `json:"pagination"`
	OnlyActive bool              `json:"only_active"`
	SymbolGuid string            `json:"symbol_guid"`
}

type SymbolMarketItem struct {
	Guid       string `json:"guid"`
	SymbolGuid string `json:"symbol_guid"`
	Price      string `json:"price"`
	AskPrice   string `json:"ask_price"`
	BidPrice   string `json:"bid_price"`
	Volume     string `json:"volume"`
	MarketCap  string `json:"market_cap"`
	Radio      string `json:"radio"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type ListSymbolMarketsResponse struct {
	Code       uint64             `json:"code"`
	Message    string             `json:"message"`
	Pagination PaginationResponse `json:"pagination"`
	Result     []SymbolMarketItem `json:"result"`
}

type GetSymbolMarketRequest struct {
	SymbolGuid string `json:"symbol_guid"`
}

type GetSymbolMarketResponse struct {
	Code    uint64            `json:"code"`
	Message string            `json:"message"`
	Result  *SymbolMarketItem `json:"result,omitempty"`
}
