package model

type ListCurrenciesRequest struct {
	Pagination   PaginationRequest `json:"pagination"`
	OnlyActive   bool              `json:"only_active"`
	CurrencyCode string            `json:"currency_code"`
}

type CurrencyItem struct {
	Guid         string `json:"guid"`
	CurrencyName string `json:"currency_name"`
	CurrencyCode string `json:"currency_code"`
	Rate         string `json:"rate"`
	BuySpread    string `json:"buy_spread"`
	SellSpread   string `json:"sell_spread"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type ListCurrenciesResponse struct {
	Code       uint64             `json:"code"`
	Message    string             `json:"message"`
	Pagination PaginationResponse `json:"pagination"`
	Result     []CurrencyItem     `json:"result"`
}

type GetSymbolMarketCurrencyRequest struct {
	SymbolGuid   string `json:"symbol_guid"`
	CurrencyGuid string `json:"currency_guid"`
}

type SymbolMarketCurrencyItem struct {
	Guid         string `json:"guid"`
	SymbolGuid   string `json:"symbol_guid"`
	CurrencyGuid string `json:"currency_guid"`
	Price        string `json:"price"`
	AskPrice     string `json:"ask_price"`
	BidPrice     string `json:"bid_price"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type GetSymbolMarketCurrencyResponse struct {
	Code    uint64                    `json:"code"`
	Message string                    `json:"message"`
	Result  *SymbolMarketCurrencyItem `json:"result,omitempty"`
}
