package service

import (
	"strconv"
	"time"

	"github.com/Sandwichzzy/market-services/services/http/model"
)

const (
	returnCodeSuccess         = 200
	returnCodeInvalidArgument = 400
	returnCodeNotFound        = 404
	returnCodeInternalError   = 4000
	defaultPage               = int64(1)
	defaultPageSize           = int64(10)
	klineTimeframe1m          = "1m"
)

func normalizePagination(req model.PaginationRequest) (int64, int64) {
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	return page, pageSize
}

func buildPaginationResponse(page, pageSize, total int64) model.PaginationResponse {
	return model.PaginationResponse{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
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

func isSupportedTimeframe(timeframe string) bool {
	return timeframe == "" || timeframe == klineTimeframe1m
}
