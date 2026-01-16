package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mateus/vibe-cli/internal/indexing"
	"github.com/mateus/vibe-cli/internal/sandbox"
)

// SemanticSearchTool performs semantic search across the codebase
type SemanticSearchTool struct {
	indexer *indexing.Indexer
}

// NewSemanticSearchTool creates a new semantic search tool
func NewSemanticSearchTool(indexer *indexing.Indexer) *SemanticSearchTool {
	return &SemanticSearchTool{
		indexer: indexer,
	}
}

func (t *SemanticSearchTool) Name() string {
	return "semantic_search"
}

func (t *SemanticSearchTool) Description() string {
	return "Search the codebase using natural language queries. Uses embeddings to find semantically similar code, even if exact keywords don't match. Useful for finding code by describing what it does."
}

func (t *SemanticSearchTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "query",
			Type:        TypeString,
			Description: "Natural language description of what you're looking for (e.g., 'function that handles user authentication', 'code that parses JSON files')",
			Required:    true,
		},
		{
			Name:        "limit",
			Type:        TypeNumber,
			Description: "Maximum number of results to return (default: 10)",
			Required:    false,
		},
		{
			Name:        "kind",
			Type:        TypeString,
			Description: "Filter by symbol kind: function, method, class, struct, interface, type, const, var",
			Required:    false,
		},
		{
			Name:        "language",
			Type:        TypeString,
			Description: "Filter by programming language: go, python, javascript, typescript, rust, etc.",
			Required:    false,
		},
	}
}

func (t *SemanticSearchTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *SemanticSearchTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	if t.indexer == nil {
		return NewErrorResult(fmt.Errorf("semantic search not available: index not initialized")), nil
	}

	query, err := RequiredStringArg(args, "query")
	if err != nil {
		return NewErrorResult(err), nil
	}

	limit := GetIntArg(args, "limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// Build filter
	var filter *indexing.SearchFilter
	kind := GetStringArg(args, "kind", "")
	language := GetStringArg(args, "language", "")

	if kind != "" || language != "" {
		filter = &indexing.SearchFilter{
			Kind:     kind,
			Language: language,
		}
	}

	// Perform search
	results, err := t.indexer.Search(ctx, query, limit, filter)
	if err != nil {
		return NewErrorResult(fmt.Errorf("search failed: %w", err)), nil
	}

	if len(results) == 0 {
		return NewSuccessResult(fmt.Sprintf("No results found for query: %s", query)), nil
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d result(s) for: %s\n\n", len(results), query))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, r.Document.Path))
		if r.Document.Symbol != "" {
			sb.WriteString(fmt.Sprintf(" - %s", r.Document.Symbol))
		}
		sb.WriteString(fmt.Sprintf(" (score: %.2f)\n", r.Score))

		if r.Document.Kind != "" {
			sb.WriteString(fmt.Sprintf("   Kind: %s\n", r.Document.Kind))
		}
		if r.Document.Language != "" {
			sb.WriteString(fmt.Sprintf("   Language: %s\n", r.Document.Language))
		}

		// Show content preview
		content := r.Document.Content
		if len(content) > 200 {
			content = content[:197] + "..."
		}
		content = strings.ReplaceAll(content, "\n", "\n   ")
		sb.WriteString(fmt.Sprintf("   Content: %s\n\n", content))
	}

	// Convert results to JSON format
	jsonResults := make([]map[string]any, len(results))
	for i, r := range results {
		jsonResults[i] = map[string]any{
			"path":     r.Document.Path,
			"symbol":   r.Document.Symbol,
			"kind":     r.Document.Kind,
			"language": r.Document.Language,
			"content":  r.Document.Content,
			"score":    r.Score,
		}
	}

	return NewSuccessResultWithData(sb.String(), jsonResults), nil
}

// IndexStatusTool shows the status of the codebase index
type IndexStatusTool struct {
	indexer *indexing.Indexer
}

// NewIndexStatusTool creates a new index status tool
func NewIndexStatusTool(indexer *indexing.Indexer) *IndexStatusTool {
	return &IndexStatusTool{
		indexer: indexer,
	}
}

func (t *IndexStatusTool) Name() string {
	return "index_status"
}

func (t *IndexStatusTool) Description() string {
	return "Shows the status of the semantic search index, including number of indexed files and documents."
}

func (t *IndexStatusTool) Parameters() []Parameter {
	return []Parameter{}
}

func (t *IndexStatusTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *IndexStatusTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	if t.indexer == nil {
		return NewSuccessResult("Index not initialized. Run `vibe index` to create the index."), nil
	}

	stats := t.indexer.GetStats()

	var sb strings.Builder
	sb.WriteString("Codebase Index Status\n")
	sb.WriteString("=====================\n\n")
	sb.WriteString(fmt.Sprintf("Total Documents: %d\n", stats.TotalDocuments))
	sb.WriteString(fmt.Sprintf("Unique Files: %d\n", stats.UniqueFiles))

	if len(stats.Languages) > 0 {
		sb.WriteString("\nBy Language:\n")
		for lang, count := range stats.Languages {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", lang, count))
		}
	}

	if len(stats.Kinds) > 0 {
		sb.WriteString("\nBy Symbol Kind:\n")
		for kind, count := range stats.Kinds {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", kind, count))
		}
	}

	return NewSuccessResultWithData(sb.String(), stats), nil
}

// ReindexFileTool re-indexes a specific file
type ReindexFileTool struct {
	indexer *indexing.Indexer
}

// NewReindexFileTool creates a new reindex file tool
func NewReindexFileTool(indexer *indexing.Indexer) *ReindexFileTool {
	return &ReindexFileTool{
		indexer: indexer,
	}
}

func (t *ReindexFileTool) Name() string {
	return "reindex_file"
}

func (t *ReindexFileTool) Description() string {
	return "Re-indexes a specific file after it has been modified. Updates the semantic search index."
}

func (t *ReindexFileTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Path to the file to re-index",
			Required:    true,
		},
	}
}

func (t *ReindexFileTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *ReindexFileTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	if t.indexer == nil {
		return NewErrorResult(fmt.Errorf("index not initialized")), nil
	}

	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	reindexed, err := t.indexer.RefreshFile(ctx, path)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to reindex file: %w", err)), nil
	}

	if reindexed {
		return NewSuccessResult(fmt.Sprintf("Successfully re-indexed: %s", path)), nil
	}
	return NewSuccessResult(fmt.Sprintf("File unchanged, no re-indexing needed: %s", path)), nil
}
