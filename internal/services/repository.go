// Package services provides repository interfaces and SQLite implementations
// for data access. This layer bridges the raw SQLite store with the HTTP API
// and dashboard, providing a clean abstraction over persistence operations.
package services

import "errors"

// ListOptions controls pagination and sorting for list queries.
type ListOptions struct {
	Limit     int    // Max results per page (default 50, max 1000).
	Offset    int    // Number of results to skip.
	SortBy    string // Column name (validated per-repository).
	SortOrder string // "asc" or "desc" (default "desc").
}

// ListResult wraps a paginated result set with a total count.
type ListResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// Sentinel errors returned by repositories.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// normalizeListOptions applies defaults and caps to list options.
func normalizeListOptions(opts ListOptions) ListOptions {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 1000 {
		opts.Limit = 1000
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.SortOrder != "asc" {
		opts.SortOrder = "desc"
	}
	return opts
}
