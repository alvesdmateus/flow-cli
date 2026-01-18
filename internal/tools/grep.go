package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mateus/flow-cli/internal/sandbox"
)

// GrepSearchTool searches for patterns in files across the codebase
type GrepSearchTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGrepSearchTool creates a new grep search tool
func NewGrepSearchTool(permissions *sandbox.Manager, workDir string) *GrepSearchTool {
	return &GrepSearchTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GrepSearchTool) Name() string {
	return "grep_search"
}

func (t *GrepSearchTool) Description() string {
	return "Search for a regex pattern across files in the codebase. Returns matching lines with file paths and line numbers."
}

func (t *GrepSearchTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "pattern",
			Type:        TypeString,
			Description: "The regex pattern to search for",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Directory or file to search in (default: current directory)",
			Required:    false,
			Default:     ".",
		},
		{
			Name:        "include",
			Type:        TypeString,
			Description: "File pattern to include (e.g., '*.go', '*.js'). Comma-separated for multiple patterns.",
			Required:    false,
		},
		{
			Name:        "exclude",
			Type:        TypeString,
			Description: "File pattern to exclude (e.g., '*_test.go', 'vendor/*'). Comma-separated for multiple patterns.",
			Required:    false,
		},
		{
			Name:        "case_insensitive",
			Type:        TypeBoolean,
			Description: "Perform case-insensitive search (default: false)",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "max_results",
			Type:        TypeNumber,
			Description: "Maximum number of matching lines to return (default: 100)",
			Required:    false,
			Default:     100,
		},
		{
			Name:        "context_lines",
			Type:        TypeNumber,
			Description: "Number of context lines to show before and after each match (default: 0)",
			Required:    false,
			Default:     0,
		},
	}
}

func (t *GrepSearchTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *GrepSearchTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	pattern, err := RequiredStringArg(args, "pattern")
	if err != nil {
		return NewErrorResult(err), nil
	}

	searchPath := GetStringArg(args, "path", ".")
	includePattern := GetStringArg(args, "include", "")
	excludePattern := GetStringArg(args, "exclude", "")
	caseInsensitive := GetBoolArg(args, "case_insensitive", false)
	maxResults := GetIntArg(args, "max_results", 100)
	contextLines := GetIntArg(args, "context_lines", 0)

	// Resolve search path
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(t.workDir, searchPath)
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpReadFile, searchPath, fmt.Sprintf("Grep search: %s in %s", pattern, searchPath))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Compile regex
	if caseInsensitive {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return NewErrorResult(fmt.Errorf("invalid regex pattern: %w", err)), nil
	}

	// Parse include/exclude patterns
	includePatterns := parsePatterns(includePattern)
	excludePatterns := parsePatterns(excludePattern)

	// Default excludes for common non-searchable directories
	defaultExcludes := []string{
		".git/*", ".svn/*", ".hg/*",
		"node_modules/*", "vendor/*", "__pycache__/*",
		"*.exe", "*.dll", "*.so", "*.dylib",
		"*.jpg", "*.jpeg", "*.png", "*.gif", "*.ico",
		"*.pdf", "*.zip", "*.tar", "*.gz",
	}
	excludePatterns = append(excludePatterns, defaultExcludes...)

	// Perform search
	var matches []GrepMatch
	matchCount := 0

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Skip excluded directories
			relPath, _ := filepath.Rel(searchPath, path)
			for _, exc := range excludePatterns {
				if matched, _ := filepath.Match(exc, relPath); matched {
					return filepath.SkipDir
				}
				if matched, _ := filepath.Match(exc, info.Name()); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if we've hit max results
		if matchCount >= maxResults {
			return filepath.SkipAll
		}

		// Get relative path for matching
		relPath, _ := filepath.Rel(searchPath, path)

		// Check exclude patterns
		for _, exc := range excludePatterns {
			if matched, _ := filepath.Match(exc, relPath); matched {
				return nil
			}
			if matched, _ := filepath.Match(exc, info.Name()); matched {
				return nil
			}
		}

		// Check include patterns (if specified)
		if len(includePatterns) > 0 {
			included := false
			for _, inc := range includePatterns {
				if matched, _ := filepath.Match(inc, relPath); matched {
					included = true
					break
				}
				if matched, _ := filepath.Match(inc, info.Name()); matched {
					included = true
					break
				}
			}
			if !included {
				return nil
			}
		}

		// Skip binary/large files
		if info.Size() > 1024*1024 { // Skip files > 1MB
			return nil
		}

		// Search in file
		fileMatches, err := searchInFile(path, re, contextLines, maxResults-matchCount)
		if err != nil {
			return nil // Skip files we can't read
		}

		for _, m := range fileMatches {
			m.File = relPath
			matches = append(matches, m)
			matchCount++
			if matchCount >= maxResults {
				break
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return NewErrorResult(fmt.Errorf("search failed: %w", err)), nil
	}

	if len(matches) == 0 {
		return NewSuccessResult(fmt.Sprintf("No matches found for pattern: %s", pattern)), nil
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Search Results for: `%s`\n\n", pattern))
	output.WriteString(fmt.Sprintf("Found %d matches", len(matches)))
	if matchCount >= maxResults {
		output.WriteString(fmt.Sprintf(" (limited to %d)", maxResults))
	}
	output.WriteString("\n\n")

	currentFile := ""
	for _, m := range matches {
		if m.File != currentFile {
			if currentFile != "" {
				output.WriteString("\n")
			}
			output.WriteString(fmt.Sprintf("## %s\n", m.File))
			currentFile = m.File
		}

		// Show context before
		for _, ctx := range m.ContextBefore {
			output.WriteString(fmt.Sprintf("  %4d │ %s\n", ctx.Line, ctx.Content))
		}

		// Show match with highlighting
		output.WriteString(fmt.Sprintf("▶ %4d │ %s\n", m.Line, m.Content))

		// Show context after
		for _, ctx := range m.ContextAfter {
			output.WriteString(fmt.Sprintf("  %4d │ %s\n", ctx.Line, ctx.Content))
		}
	}

	// Build structured data
	resultData := make([]map[string]any, len(matches))
	for i, m := range matches {
		resultData[i] = map[string]any{
			"file":    m.File,
			"line":    m.Line,
			"content": m.Content,
		}
	}

	return NewSuccessResultWithData(output.String(), map[string]any{
		"pattern":     pattern,
		"match_count": len(matches),
		"matches":     resultData,
	}), nil
}

// GrepMatch represents a single match
type GrepMatch struct {
	File          string
	Line          int
	Content       string
	ContextBefore []LineContext
	ContextAfter  []LineContext
}

// LineContext represents a context line
type LineContext struct {
	Line    int
	Content string
}

// parsePatterns splits a comma-separated pattern string into a slice
func parsePatterns(pattern string) []string {
	if pattern == "" {
		return nil
	}
	parts := strings.Split(pattern, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// searchInFile searches for a pattern in a file and returns matches
func searchInFile(path string, re *regexp.Regexp, contextLines, maxMatches int) ([]GrepMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read all lines for context support
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var matches []GrepMatch
	for i, line := range lines {
		if len(matches) >= maxMatches {
			break
		}

		if re.MatchString(line) {
			match := GrepMatch{
				Line:    i + 1,
				Content: line,
			}

			// Add context before
			if contextLines > 0 {
				start := i - contextLines
				if start < 0 {
					start = 0
				}
				for j := start; j < i; j++ {
					match.ContextBefore = append(match.ContextBefore, LineContext{
						Line:    j + 1,
						Content: lines[j],
					})
				}
			}

			// Add context after
			if contextLines > 0 {
				end := i + contextLines + 1
				if end > len(lines) {
					end = len(lines)
				}
				for j := i + 1; j < end; j++ {
					match.ContextAfter = append(match.ContextAfter, LineContext{
						Line:    j + 1,
						Content: lines[j],
					})
				}
			}

			matches = append(matches, match)
		}
	}

	return matches, nil
}
