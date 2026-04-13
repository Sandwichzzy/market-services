package model

type ListKlinesRequest struct {
	SymbolGuid     string            `json:"symbol_guid"`
	Timeframe      string            `json:"timeframe"`
	StartTimestamp int64             `json:"start_timestamp"`
	EndTimestamp   int64             `json:"end_timestamp"`
	Pagination     PaginationRequest `json:"pagination"`
	OnlyActive     bool              `json:"only_active"`
}

type SymbolKlineItem struct {
	Guid       string `json:"guid"`
	SymbolGuid string `json:"symbol_guid"`
	Timeframe  string `json:"timeframe"`
	OpenPrice  string `json:"open_price"`
	ClosePrice string `json:"close_price"`
	HighPrice  string `json:"high_price"`
	LowPrice   string `json:"low_price"`
	Volume     string `json:"volume"`
	MarketCap  string `json:"market_cap"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type ListKlinesResponse struct {
	Code       uint64             `json:"code"`
	Message    string             `json:"message"`
	Pagination PaginationResponse `json:"pagination"`
	Result     []SymbolKlineItem  `json:"result"`
}

type ListExchangeKlinesRequest struct {
	SymbolGuid     string            `json:"symbol_guid"`
	ExchangeGuid   string            `json:"exchange_guid"`
	Timeframe      string            `json:"timeframe"`
	StartTimestamp int64             `json:"start_timestamp"`
	EndTimestamp   int64             `json:"end_timestamp"`
	Pagination     PaginationRequest `json:"pagination"`
	OnlyActive     bool              `json:"only_active"`
}

type ExchangeSymbolKlineItem struct {
	Guid         string `json:"guid"`
	ExchangeGuid string `json:"exchange_guid"`
	SymbolGuid   string `json:"symbol_guid"`
	Timeframe    string `json:"timeframe"`
	OpenPrice    string `json:"open_price"`
	ClosePrice   string `json:"close_price"`
	HighPrice    string `json:"high_price"`
	LowPrice     string `json:"low_price"`
	Volume       string `json:"volume"`
	MarketCap    string `json:"market_cap"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type ListExchangeKlinesResponse struct {
	Code       uint64                    `json:"code"`
	Message    string                    `json:"message"`
	Pagination PaginationResponse        `json:"pagination"`
	Result     []ExchangeSymbolKlineItem `json:"result"`
}
