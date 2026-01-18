package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mateus/flow-cli/internal/analysis"
	"github.com/mateus/flow-cli/internal/sandbox"
)

// CodeOutlineTool returns the structure of a source file
type CodeOutlineTool struct {
	analyzer   *analysis.Analyzer
	workingDir string
}

// NewCodeOutlineTool creates a new code outline tool
func NewCodeOutlineTool(workingDir string) *CodeOutlineTool {
	return &CodeOutlineTool{
		analyzer:   analysis.NewAnalyzer(),
		workingDir: workingDir,
	}
}

func (t *CodeOutlineTool) Name() string {
	return "code_outline"
}

func (t *CodeOutlineTool) Description() string {
	return "Parses a source file and returns its structure including functions, classes, methods, imports, and other symbols. Supports Go, Python, JavaScript, TypeScript, and Rust."
}

func (t *CodeOutlineTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Path to the source file to analyze",
			Required:    true,
		},
	}
}

func (t *CodeOutlineTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *CodeOutlineTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	// Resolve path
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(t.workingDir, path)
	}

	// Parse the file
	outline, err := t.analyzer.ParseFile(fullPath)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to parse file: %w", err)), nil
	}

	// Format output
	output := formatOutline(outline)

	// Return both formatted output and structured data
	return NewSuccessResultWithData(output, outline), nil
}

// formatOutline formats a file outline for display
func formatOutline(outline *analysis.FileOutline) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("File: %s\n", outline.File))
	sb.WriteString(fmt.Sprintf("Language: %s\n", outline.Language))

	if outline.Package != "" {
		sb.WriteString(fmt.Sprintf("Package: %s\n", outline.Package))
	}

	if len(outline.Imports) > 0 {
		sb.WriteString(fmt.Sprintf("\nImports (%d):\n", len(outline.Imports)))
		for _, imp := range outline.Imports {
			sb.WriteString(fmt.Sprintf("  - %s\n", imp))
		}
	}

	if len(outline.Symbols) > 0 {
		sb.WriteString(fmt.Sprintf("\nSymbols (%d):\n", len(outline.Symbols)))

		// Group by kind
		groups := make(map[analysis.SymbolKind][]analysis.Symbol)
		for _, sym := range outline.Symbols {
			groups[sym.Kind] = append(groups[sym.Kind], sym)
		}

		// Print in order
		kindOrder := []analysis.SymbolKind{
			analysis.SymbolInterface,
			analysis.SymbolStruct,
			analysis.SymbolClass,
			analysis.SymbolType,
			analysis.SymbolFunction,
			analysis.SymbolMethod,
			analysis.SymbolConst,
			analysis.SymbolVar,
		}

		for _, kind := range kindOrder {
			symbols, ok := groups[kind]
			if !ok || len(symbols) == 0 {
				continue
			}

			sb.WriteString(fmt.Sprintf("\n  %ss:\n", capitalize(string(kind))))
			for _, sym := range symbols {
				exportedMark := ""
				if sym.Exported {
					exportedMark = " [exported]"
				}

				parentInfo := ""
				if sym.Parent != "" {
					parentInfo = fmt.Sprintf(" (on %s)", sym.Parent)
				}

				if sym.Signature != "" {
					sb.WriteString(fmt.Sprintf("    - %s%s%s (line %d)\n      %s\n",
						sym.Name, parentInfo, exportedMark, sym.Line, sym.Signature))
				} else {
					sb.WriteString(fmt.Sprintf("    - %s%s%s (line %d)\n",
						sym.Name, parentInfo, exportedMark, sym.Line))
				}

				if sym.Doc != "" {
					sb.WriteString(fmt.Sprintf("      Doc: %s\n", truncateDoc(sym.Doc, 80)))
				}
			}
		}
	}

	return sb.String()
}

// capitalize returns the string with first letter capitalized
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func truncateDoc(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FindDefinitionTool finds where a symbol is defined
type FindDefinitionTool struct {
	analyzer   *analysis.Analyzer
	workingDir string
}

// NewFindDefinitionTool creates a new find definition tool
func NewFindDefinitionTool(workingDir string) *FindDefinitionTool {
	return &FindDefinitionTool{
		analyzer:   analysis.NewAnalyzer(),
		workingDir: workingDir,
	}
}

func (t *FindDefinitionTool) Name() string {
	return "find_definition"
}

func (t *FindDefinitionTool) Description() string {
	return "Searches for the definition of a symbol (function, class, type, etc.) in the codebase. Returns file location and signature."
}

func (t *FindDefinitionTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "symbol",
			Type:        TypeString,
			Description: "Name of the symbol to find (function, class, type, etc.)",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Directory or file to search in (defaults to working directory)",
			Required:    false,
		},
		{
			Name:        "kind",
			Type:        TypeString,
			Description: "Filter by symbol kind: function, method, type, struct, interface, class, const, var",
			Required:    false,
		},
	}
}

func (t *FindDefinitionTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *FindDefinitionTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	symbolName, err := RequiredStringArg(args, "symbol")
	if err != nil {
		return NewErrorResult(err), nil
	}

	searchPath := GetStringArg(args, "path", t.workingDir)
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(t.workingDir, searchPath)
	}

	kindFilter := GetStringArg(args, "kind", "")

	// Collect files to search
	files, err := collectSourceFiles(searchPath)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to collect files: %w", err)), nil
	}

	// Parse all files and collect outlines
	var outlines []*analysis.FileOutline
	for _, file := range files {
		outline, err := t.analyzer.ParseFile(file)
		if err != nil {
			continue // Skip files that can't be parsed
		}
		outlines = append(outlines, outline)
	}

	// Find matching symbols
	results := analysis.FindSymbol(outlines, symbolName)

	// Filter by kind if specified
	if kindFilter != "" {
		kind := analysis.SymbolKind(kindFilter)
		var filtered []analysis.SymbolResult
		for i := range results {
			if results[i].Symbol.Kind == kind {
				filtered = append(filtered, results[i])
			}
		}
		results = filtered
	}

	if len(results) == 0 {
		return NewSuccessResult(fmt.Sprintf("No definition found for '%s'", symbolName)), nil
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d definition(s) for '%s':\n\n", len(results), symbolName))

	for i, r := range results {
		relPath, _ := filepath.Rel(t.workingDir, r.File)
		if relPath == "" {
			relPath = r.File
		}

		sb.WriteString(fmt.Sprintf("%d. %s:%d\n", i+1, relPath, r.Symbol.Line))
		sb.WriteString(fmt.Sprintf("   Kind: %s\n", r.Symbol.Kind))

		if r.Symbol.Exported {
			sb.WriteString("   Exported: yes\n")
		}

		if r.Symbol.Parent != "" {
			sb.WriteString(fmt.Sprintf("   Parent: %s\n", r.Symbol.Parent))
		}

		if r.Symbol.Signature != "" {
			sb.WriteString(fmt.Sprintf("   Signature: %s\n", r.Symbol.Signature))
		}

		if r.Symbol.Doc != "" {
			sb.WriteString(fmt.Sprintf("   Doc: %s\n", truncateDoc(r.Symbol.Doc, 100)))
		}

		sb.WriteString("\n")
	}

	// Convert results to JSON-serializable format
	jsonResults := make([]map[string]any, len(results))
	for i, r := range results {
		jsonResults[i] = map[string]any{
			"file":      r.File,
			"line":      r.Symbol.Line,
			"name":      r.Symbol.Name,
			"kind":      string(r.Symbol.Kind),
			"exported":  r.Symbol.Exported,
			"signature": r.Symbol.Signature,
			"doc":       r.Symbol.Doc,
			"parent":    r.Symbol.Parent,
		}
	}

	return NewSuccessResultWithData(sb.String(), jsonResults), nil
}

// FindReferencesTool finds all references to a symbol
type FindReferencesTool struct {
	workingDir string
}

// NewFindReferencesTool creates a new find references tool
func NewFindReferencesTool(workingDir string) *FindReferencesTool {
	return &FindReferencesTool{
		workingDir: workingDir,
	}
}

func (t *FindReferencesTool) Name() string {
	return "find_references"
}

func (t *FindReferencesTool) Description() string {
	return "Finds all occurrences of a symbol in the codebase using pattern matching. Note: This is text-based matching, not semantic analysis."
}

func (t *FindReferencesTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "symbol",
			Type:        TypeString,
			Description: "Name of the symbol to find references for",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Directory to search in (defaults to working directory)",
			Required:    false,
		},
		{
			Name:        "extensions",
			Type:        TypeString,
			Description: "Comma-separated list of file extensions to search (e.g., '.go,.py'). Defaults to all supported extensions.",
			Required:    false,
		},
	}
}

func (t *FindReferencesTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *FindReferencesTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	symbolName, err := RequiredStringArg(args, "symbol")
	if err != nil {
		return NewErrorResult(err), nil
	}

	searchPath := GetStringArg(args, "path", t.workingDir)
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(t.workingDir, searchPath)
	}

	extensionsStr := GetStringArg(args, "extensions", "")
	var extensions []string
	if extensionsStr != "" {
		extensions = strings.Split(extensionsStr, ",")
		for i, ext := range extensions {
			extensions[i] = strings.TrimSpace(ext)
		}
	}

	// Collect files to search
	files, err := collectSourceFilesWithExtensions(searchPath, extensions)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to collect files: %w", err)), nil
	}

	type reference struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}

	var references []reference

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.Contains(line, symbolName) {
				relPath, _ := filepath.Rel(t.workingDir, file)
				if relPath == "" {
					relPath = file
				}
				references = append(references, reference{
					File:    relPath,
					Line:    i + 1,
					Content: strings.TrimSpace(line),
				})
			}
		}
	}

	if len(references) == 0 {
		return NewSuccessResult(fmt.Sprintf("No references found for '%s'", symbolName)), nil
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d reference(s) for '%s':\n\n", len(references), symbolName))

	// Group by file
	fileRefs := make(map[string][]reference)
	for _, ref := range references {
		fileRefs[ref.File] = append(fileRefs[ref.File], ref)
	}

	for file, refs := range fileRefs {
		sb.WriteString(fmt.Sprintf("%s (%d references):\n", file, len(refs)))
		for _, ref := range refs {
			content := ref.Content
			if len(content) > 100 {
				content = content[:97] + "..."
			}
			sb.WriteString(fmt.Sprintf("  Line %d: %s\n", ref.Line, content))
		}
		sb.WriteString("\n")
	}

	return NewSuccessResultWithData(sb.String(), references), nil
}

// ListSymbolsTool lists all symbols in a directory
type ListSymbolsTool struct {
	analyzer   *analysis.Analyzer
	workingDir string
}

// NewListSymbolsTool creates a new list symbols tool
func NewListSymbolsTool(workingDir string) *ListSymbolsTool {
	return &ListSymbolsTool{
		analyzer:   analysis.NewAnalyzer(),
		workingDir: workingDir,
	}
}

func (t *ListSymbolsTool) Name() string {
	return "list_symbols"
}

func (t *ListSymbolsTool) Description() string {
	return "Lists all symbols (functions, classes, types, etc.) in a directory or file. Useful for understanding codebase structure."
}

func (t *ListSymbolsTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Directory or file to analyze (defaults to working directory)",
			Required:    false,
		},
		{
			Name:        "kind",
			Type:        TypeString,
			Description: "Filter by symbol kind: function, method, type, struct, interface, class, const, var",
			Required:    false,
		},
		{
			Name:        "exported_only",
			Type:        TypeBoolean,
			Description: "Only show exported/public symbols",
			Required:    false,
		},
	}
}

func (t *ListSymbolsTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *ListSymbolsTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	searchPath := GetStringArg(args, "path", t.workingDir)
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(t.workingDir, searchPath)
	}

	kindFilter := GetStringArg(args, "kind", "")
	exportedOnly := GetBoolArg(args, "exported_only", false)

	// Collect files
	files, err := collectSourceFiles(searchPath)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to collect files: %w", err)), nil
	}

	type symbolInfo struct {
		Name      string `json:"name"`
		Kind      string `json:"kind"`
		File      string `json:"file"`
		Line      int    `json:"line"`
		Exported  bool   `json:"exported"`
		Signature string `json:"signature,omitempty"`
	}

	var symbols []symbolInfo

	for _, file := range files {
		outline, err := t.analyzer.ParseFile(file)
		if err != nil {
			continue
		}

		relPath, _ := filepath.Rel(t.workingDir, file)
		if relPath == "" {
			relPath = file
		}

		for _, sym := range outline.Symbols {
			// Apply filters
			if kindFilter != "" && string(sym.Kind) != kindFilter {
				continue
			}
			if exportedOnly && !sym.Exported {
				continue
			}

			symbols = append(symbols, symbolInfo{
				Name:      sym.Name,
				Kind:      string(sym.Kind),
				File:      relPath,
				Line:      sym.Line,
				Exported:  sym.Exported,
				Signature: sym.Signature,
			})
		}
	}

	if len(symbols) == 0 {
		return NewSuccessResult("No symbols found matching the criteria"), nil
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d symbol(s):\n\n", len(symbols)))

	// Group by file
	fileSyms := make(map[string][]symbolInfo)
	for _, sym := range symbols {
		fileSyms[sym.File] = append(fileSyms[sym.File], sym)
	}

	for file, syms := range fileSyms {
		sb.WriteString(fmt.Sprintf("%s:\n", file))
		for _, sym := range syms {
			exportMark := ""
			if sym.Exported {
				exportMark = " [exported]"
			}
			if sym.Signature != "" {
				sb.WriteString(fmt.Sprintf("  %s %s%s (line %d)\n    %s\n",
					sym.Kind, sym.Name, exportMark, sym.Line, sym.Signature))
			} else {
				sb.WriteString(fmt.Sprintf("  %s %s%s (line %d)\n",
					sym.Kind, sym.Name, exportMark, sym.Line))
			}
		}
		sb.WriteString("\n")
	}

	// Convert to JSON format
	jsonData, _ := json.Marshal(symbols)

	return NewSuccessResultWithData(sb.String(), json.RawMessage(jsonData)), nil
}

// collectSourceFiles collects all supported source files from a path
func collectSourceFiles(path string) ([]string, error) {
	return collectSourceFilesWithExtensions(path, nil)
}

// collectSourceFilesWithExtensions collects source files with specific extensions
func collectSourceFilesWithExtensions(path string, extensions []string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Default supported extensions
	supportedExt := map[string]bool{
		".go":  true,
		".py":  true,
		".js":  true,
		".ts":  true,
		".tsx": true,
		".jsx": true,
		".rs":  true,
	}

	// Use custom extensions if provided
	if len(extensions) > 0 {
		supportedExt = make(map[string]bool)
		for _, ext := range extensions {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			supportedExt[ext] = true
		}
	}

	var files []string

	if !info.IsDir() {
		// Single file
		ext := strings.ToLower(filepath.Ext(path))
		if supportedExt[ext] {
			files = append(files, path)
		}
		return files, nil
	}

	// Walk directory
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories and common non-source directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(p))
		if supportedExt[ext] {
			files = append(files, p)
		}

		return nil
	})

	return files, err
}
