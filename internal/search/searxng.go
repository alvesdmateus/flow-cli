package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearXNGClient implements the Client interface for SearXNG
type SearXNGClient struct {
	endpoint   string
	httpClient *http.Client
}

// searxngResponse represents the JSON response from SearXNG
type searxngResponse struct {
	Query           string          `json:"query"`
	NumberOfResults int             `json:"number_of_results"`
	Results         []searxngResult `json:"results"`
}

type searxngResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// NewSearXNGClient creates a new SearXNG client
func NewSearXNGClient(endpoint string) (*SearXNGClient, error) {
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}

	// Ensure endpoint doesn't have trailing slash
	endpoint = strings.TrimSuffix(endpoint, "/")

	return &SearXNGClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Provider returns the provider name
func (s *SearXNGClient) Provider() string {
	return "searxng"
}

// Ping checks if SearXNG is available
func (s *SearXNGClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+"/config", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrConnectionError, resp.StatusCode)
	}

	return nil
}

// Search performs a web search using SearXNG
func (s *SearXNGClient) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Build search URL with parameters
	searchURL, err := s.buildSearchURL(query, opts)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrInvalidResponse, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp searxngResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(searchResp.Results) == 0 {
		return nil, ErrNoResults
	}

	// Convert to our Result type
	results := make([]Result, 0, len(searchResp.Results))
	for i, r := range searchResp.Results {
		if opts.Limit > 0 && i >= opts.Limit {
			break
		}
		results = append(results, Result{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
			Engine:  r.Engine,
		})
	}

	return results, nil
}

// buildSearchURL constructs the SearXNG search URL with parameters
func (s *SearXNGClient) buildSearchURL(query string, opts SearchOptions) (string, error) {
	baseURL := s.endpoint + "/search"

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")

	// Categories
	if len(opts.Categories) > 0 {
		params.Set("categories", strings.Join(opts.Categories, ","))
	}

	// Language
	if opts.Language != "" {
		params.Set("language", opts.Language)
	}

	// Time range
	if opts.TimeRange != "" {
		params.Set("time_range", opts.TimeRange)
	}

	return baseURL + "?" + params.Encode(), nil
}

// FormatResultsForLLM formats search results as a string suitable for LLM context
func FormatResultsForLLM(results []Result) string {
	if len(results) == 0 {
		return "No search results found."
	}

	var sb strings.Builder
	sb.WriteString("Web Search Results:\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		if r.Content != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Content))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
