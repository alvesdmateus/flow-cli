package search

import (
	"context"
	"errors"
)

// Result represents a single search result
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// SearchOptions holds options for search requests
type SearchOptions struct {
	// Categories to search in (general, images, news, etc.)
	Categories []string

	// Language for results (e.g., "en", "pt")
	Language string

	// Number of results to return
	Limit int

	// Time range filter (day, week, month, year)
	TimeRange string
}

// Client defines the interface for web search providers
type Client interface {
	// Search performs a web search and returns results
	Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error)

	// Ping checks if the search service is available
	Ping(ctx context.Context) error

	// Provider returns the name of this provider
	Provider() string
}

// Common errors
var (
	ErrNoResults       = errors.New("no search results found")
	ErrConnectionError = errors.New("failed to connect to search service")
	ErrInvalidResponse = errors.New("invalid response from search service")
	ErrRateLimited     = errors.New("rate limited by search service")
)

// DefaultOptions returns sensible default search options
func DefaultOptions() SearchOptions {
	return SearchOptions{
		Categories: []string{"general"},
		Language:   "en",
		Limit:      10,
		TimeRange:  "",
	}
}

// NewClient creates a new search client based on the provider
func NewClient(provider, endpoint string) (Client, error) {
	switch provider {
	case "searxng":
		return NewSearXNGClient(endpoint)
	default:
		return NewSearXNGClient(endpoint)
	}
}
