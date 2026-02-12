package utils

import (
	"net/http"
	"strconv"
)

const (
	// DefaultLimit is the default number of items per page
	DefaultLimit = 20
	// MaxLimit is the maximum number of items per page
	MaxLimit = 100
)

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Limit  int
	Offset int
}

// PaginationMeta represents pagination metadata in responses
type PaginationMeta struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// ParsePagination extracts pagination parameters from request
func ParsePagination(r *http.Request) PaginationParams {
	limit := parseIntParam(r, "limit", DefaultLimit)
	offset := parseIntParam(r, "offset", 0)

	// Enforce max limit
	if limit > MaxLimit {
		limit = MaxLimit
	}
	
	// Ensure non-negative values
	if limit < 1 {
		limit = DefaultLimit
	}
	if offset < 0 {
		offset = 0
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
}

// CreatePaginationMeta creates pagination metadata
func CreatePaginationMeta(total, limit, offset int) PaginationMeta {
	hasMore := (offset + limit) < total
	
	return PaginationMeta{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}
}

// parseIntParam parses an integer query parameter with a default value
func parseIntParam(r *http.Request, param string, defaultValue int) int {
	valueStr := r.URL.Query().Get(param)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// ParseFloatParam parses a float query parameter
func ParseFloatParam(r *http.Request, param string) *float64 {
	valueStr := r.URL.Query().Get(param)
	if valueStr == "" {
		return nil
	}
	
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil
	}
	
	return &value
}
