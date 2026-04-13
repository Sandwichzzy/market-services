package model

type PaginationRequest struct {
	Page     int64 `json:"page"`
	PageSize int64 `json:"page_size"`
}

type PaginationResponse struct {
	Page     int64 `json:"page"`
	PageSize int64 `json:"page_size"`
	Total    int64 `json:"total"`
}
