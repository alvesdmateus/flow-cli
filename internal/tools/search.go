package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/mateus/flow-cli/internal/sandbox"
	"github.com/mateus/flow-cli/internal/search"
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
	httpClient  *http.Client
}

// NewFetchURLTool creates a new fetch URL tool
func NewFetchURLTool(permissions *sandbox.Manager) *FetchURLTool {
	return &FetchURLTool{
		permissions: permissions,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *FetchURLTool) Name() string {
	return "fetch_url"
}

func (t *FetchURLTool) Description() string {
	return "Fetch and extract readable content from a URL. Converts HTML to clean text/markdown format."
}

func (t *FetchURLTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "url",
			Type:        TypeString,
			Description: "The URL to fetch",
			Required:    true,
		},
		{
			Name:        "max_length",
			Type:        TypeNumber,
			Description: "Maximum content length to return in characters (default: 50000)",
			Required:    false,
			Default:     50000,
		},
		{
			Name:        "timeout",
			Type:        TypeNumber,
			Description: "Request timeout in seconds (default: 30)",
			Required:    false,
			Default:     30,
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

	maxLength := GetIntArg(args, "max_length", 50000)
	timeout := GetIntArg(args, "timeout", 30)

	// Validate URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return NewErrorResult(fmt.Errorf("invalid URL: must start with http:// or https://")), nil
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpNetworkAccess, url, fmt.Sprintf("Fetch URL: %s", url))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Create request with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to create request: %w", err)), nil
	}

	// Set user agent to avoid blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; flow-cli/1.0; +https://github.com/mateus/flow-cli)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to fetch URL: %w", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewErrorResult(fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)), nil
	}

	// Limit body size to prevent memory issues (10MB max)
	const maxBodySize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxBodySize)

	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to read response: %w", err)), nil
	}

	contentType := resp.Header.Get("Content-Type")
	var content string
	var title string

	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		// Parse HTML and extract text
		title, content = extractHTMLContent(string(body))
	} else if strings.Contains(contentType, "text/plain") {
		content = string(body)
	} else if strings.Contains(contentType, "application/json") {
		content = string(body)
	} else {
		return NewErrorResult(fmt.Errorf("unsupported content type: %s", contentType)), nil
	}

	// Truncate if needed
	if len(content) > maxLength {
		content = content[:maxLength] + "\n\n[Content truncated at " + fmt.Sprint(maxLength) + " characters]"
	}

	// Build output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Fetched: %s\n\n", url))
	if title != "" {
		output.WriteString(fmt.Sprintf("**Title:** %s\n\n", title))
	}
	output.WriteString(fmt.Sprintf("**Content-Type:** %s\n", contentType))
	output.WriteString(fmt.Sprintf("**Length:** %d characters\n\n", len(content)))
	output.WriteString("---\n\n")
	output.WriteString(content)

	return NewSuccessResultWithData(output.String(), map[string]any{
		"url":          url,
		"title":        title,
		"content_type": contentType,
		"length":       len(content),
	}), nil
}

// extractHTMLContent parses HTML and returns title and readable text content
func extractHTMLContent(htmlContent string) (title string, content string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", htmlContent
	}

	var titleBuilder strings.Builder
	var contentBuilder strings.Builder
	var inTitle, inScript, inStyle, inHead bool

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		switch n.Type {
		case html.ElementNode:
			tag := strings.ToLower(n.Data)

			// Track special elements
			switch tag {
			case "title":
				inTitle = true
			case "script", "noscript":
				inScript = true
			case "style":
				inStyle = true
			case "head":
				inHead = true
			}

			// Skip hidden elements
			for _, attr := range n.Attr {
				if attr.Key == "hidden" || (attr.Key == "style" && strings.Contains(attr.Val, "display:none")) {
					return
				}
			}

			// Process children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extract(c)
			}

			// Reset flags and add formatting after element
			switch tag {
			case "title":
				inTitle = false
			case "script", "noscript":
				inScript = false
			case "style":
				inStyle = false
			case "head":
				inHead = false
			case "p", "div", "section", "article", "br":
				contentBuilder.WriteString("\n\n")
			case "h1":
				contentBuilder.WriteString("\n\n# ")
			case "h2":
				contentBuilder.WriteString("\n\n## ")
			case "h3":
				contentBuilder.WriteString("\n\n### ")
			case "h4", "h5", "h6":
				contentBuilder.WriteString("\n\n#### ")
			case "li":
				contentBuilder.WriteString("\n- ")
			case "tr":
				contentBuilder.WriteString("\n")
			case "td", "th":
				contentBuilder.WriteString(" | ")
			case "a":
				// Extract href for links
				for _, attr := range n.Attr {
					if attr.Key == "href" && !strings.HasPrefix(attr.Val, "#") && !strings.HasPrefix(attr.Val, "javascript:") {
						contentBuilder.WriteString(fmt.Sprintf(" [%s]", attr.Val))
					}
				}
			}

		case html.TextNode:
			text := strings.TrimSpace(n.Data)
			if text == "" {
				return
			}

			if inTitle {
				titleBuilder.WriteString(text)
			} else if !inScript && !inStyle && !inHead {
				contentBuilder.WriteString(text)
				contentBuilder.WriteString(" ")
			}
		default:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extract(c)
			}
		}
	}

	extract(doc)

	// Clean up content
	content = contentBuilder.String()
	content = cleanupText(content)
	title = strings.TrimSpace(titleBuilder.String())

	return title, content
}

// cleanupText normalizes whitespace and removes excessive blank lines
func cleanupText(text string) string {
	// Replace multiple spaces with single space
	spaceRe := regexp.MustCompile(`[ \t]+`)
	text = spaceRe.ReplaceAllString(text, " ")

	// Replace more than 2 newlines with 2 newlines
	newlineRe := regexp.MustCompile(`\n{3,}`)
	text = newlineRe.ReplaceAllString(text, "\n\n")

	// Trim each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	return strings.TrimSpace(text)
}
