package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mateus/vibe-cli/internal/sandbox"
	"github.com/mateus/vibe-cli/internal/search"
)

// WebSearchTool searches the web using SearXNG
type WebSearchTool struct {
	permissions  *sandbox.Manager
	searchClient search.Client
}

// NewWebSearchTool creates a new web search tool
func NewWebSearchTool(permissions *sandbox.Manager, searchClient search.Client) *WebSearchTool {
	return &WebSearchTool{
		permissions:  permissions,
		searchClient: searchClient,
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for information using SearXNG"
}

func (t *WebSearchTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "query",
			Type:        TypeString,
			Description: "The search query",
			Required:    true,
		},
		{
			Name:        "limit",
			Type:        TypeNumber,
			Description: "Maximum number of results to return (default: 10)",
			Required:    false,
			Default:     10,
		},
		{
			Name:        "category",
			Type:        TypeString,
			Description: "Search category: general, news, images, videos, science, it (default: general)",
			Required:    false,
			Default:     "general",
		},
		{
			Name:        "time_range",
			Type:        TypeString,
			Description: "Time range filter: day, week, month, year (optional)",
			Required:    false,
		},
	}
}

func (t *WebSearchTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWebSearch
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	if t.searchClient == nil {
		return NewErrorResult(fmt.Errorf("search client not configured")), nil
	}

	query, err := RequiredStringArg(args, "query")
	if err != nil {
		return NewErrorResult(err), nil
	}

	limit := GetIntArg(args, "limit", 10)
	category := GetStringArg(args, "category", "general")
	timeRange := GetStringArg(args, "time_range", "")

	// Check permission
	op := sandbox.NewOperation(sandbox.OpWebSearch, query, fmt.Sprintf("Web search: %s", query)).
		WithDetail("category", category)
	if timeRange != "" {
		op.WithDetail("time_range", timeRange)
	}

	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Perform search
	opts := search.SearchOptions{
		Categories: []string{category},
		Limit:      limit,
		TimeRange:  timeRange,
	}

	results, err := t.searchClient.Search(ctx, query, opts)
	if err != nil {
		return NewErrorResult(fmt.Errorf("search failed: %w", err)), nil
	}

	if len(results) == 0 {
		return NewSuccessResult("No results found for query: " + query), nil
	}

	// Format results
	output := search.FormatResultsForLLM(results)

	// Build data for structured access
	resultData := make([]map[string]string, len(results))
	for i, r := range results {
		resultData[i] = map[string]string{
			"title":   r.Title,
			"url":     r.URL,
			"content": r.Content,
			"engine":  r.Engine,
		}
	}

	return NewSuccessResultWithData(output, map[string]any{
		"query":   query,
		"count":   len(results),
		"results": resultData,
	}), nil
}

// FetchURLTool fetches content from a URL
type FetchURLTool struct {
	permissions *sandbox.Manager
}

// NewFetchURLTool creates a new fetch URL tool
func NewFetchURLTool(permissions *sandbox.Manager) *FetchURLTool {
	return &FetchURLTool{permissions: permissions}
}

func (t *FetchURLTool) Name() string {
	return "fetch_url"
}

func (t *FetchURLTool) Description() string {
	return "Fetch content from a URL (text-based content only)"
}

func (t *FetchURLTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "url",
			Type:        TypeString,
			Description: "The URL to fetch",
			Required:    true,
		},
	}
}

func (t *FetchURLTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpNetworkAccess
}

func (t *FetchURLTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	url, err := RequiredStringArg(args, "url")
	if err != nil {
		return NewErrorResult(err), nil
	}

	// Validate URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return NewErrorResult(fmt.Errorf("invalid URL: must start with http:// or https://")), nil
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpNetworkAccess, url, fmt.Sprintf("Fetch URL: %s", url))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Note: Actual implementation would use net/http to fetch content
	// For now, return a placeholder that indicates this needs implementation
	return NewErrorResult(fmt.Errorf("fetch_url is not yet implemented - use web_search instead")), nil
}
