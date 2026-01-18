package indexing

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mateus/flow-cli/internal/analysis"
)

// Indexer manages codebase indexing with embeddings
type Indexer struct {
	embedClient *EmbeddingClient
	store       *VectorStore
	analyzer    *analysis.Analyzer
	workDir     string
	ignoreRules []string
	mu          sync.RWMutex

	// File hash cache for change detection
	fileHashes map[string]string
}

// IndexerConfig holds configuration for the indexer
type IndexerConfig struct {
	Endpoint       string // Ollama endpoint
	EmbeddingModel string // Embedding model name
	DBPath         string // SQLite database path
	WorkDir        string // Working directory to index
}

// NewIndexer creates a new codebase indexer
func NewIndexer(cfg IndexerConfig) (*Indexer, error) {
	if cfg.WorkDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		cfg.WorkDir = wd
	}

	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(cfg.WorkDir, ".flow", "index.db")
	}

	embedClient, err := NewEmbeddingClient(cfg.Endpoint, cfg.EmbeddingModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding client: %w", err)
	}

	store, err := NewVectorStore(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	indexer := &Indexer{
		embedClient: embedClient,
		store:       store,
		analyzer:    analysis.NewAnalyzer(),
		workDir:     cfg.WorkDir,
		fileHashes:  make(map[string]string),
	}

	// Load .flowignore rules
	indexer.loadIgnoreRules()

	return indexer, nil
}

// loadIgnoreRules loads ignore patterns from .flowignore
func (idx *Indexer) loadIgnoreRules() {
	// Default ignore patterns
	idx.ignoreRules = []string{
		".git",
		".flow",
		"node_modules",
		"vendor",
		"__pycache__",
		".cache",
		"target",
		"dist",
		"build",
		".next",
		"*.min.js",
		"*.min.css",
		"*.map",
		"*.lock",
		"package-lock.json",
		"yarn.lock",
		"go.sum",
	}

	// Load custom rules from .flowignore
	ignorePath := filepath.Join(idx.workDir, ".flowignore")
	file, err := os.Open(ignorePath)
	if err != nil {
		return // No .flowignore file
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			idx.ignoreRules = append(idx.ignoreRules, line)
		}
	}
}

// shouldIgnore checks if a path should be ignored
func (idx *Indexer) shouldIgnore(path string) bool {
	relPath, err := filepath.Rel(idx.workDir, path)
	if err != nil {
		return true
	}

	for _, rule := range idx.ignoreRules {
		// Check if it's a glob pattern
		if strings.Contains(rule, "*") {
			matched, _ := filepath.Match(rule, filepath.Base(path))
			if matched {
				return true
			}
			continue
		}

		// Check if path contains the ignore rule
		if strings.Contains(relPath, rule) {
			return true
		}

		// Check if base name matches
		if filepath.Base(path) == rule {
			return true
		}
	}

	return false
}

// supportedExtensions returns extensions that can be indexed
func supportedExtensions() map[string]string {
	return map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".tsx":  "typescript",
		".jsx":  "javascript",
		".rs":   "rust",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".h":    "c",
		".hpp":  "cpp",
		".rb":   "ruby",
		".php":  "php",
		".md":   "markdown",
		".txt":  "text",
	}
}

// isSupported checks if a file extension is supported
func isSupported(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	lang, ok := supportedExtensions()[ext]
	return lang, ok
}

// IndexFile indexes a single file
func (idx *Indexer) IndexFile(ctx context.Context, path string) error {
	if idx.shouldIgnore(path) {
		return nil
	}

	language, supported := isSupported(path)
	if !supported {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check if file has changed
	hash := hashContent(content)
	relPath, _ := filepath.Rel(idx.workDir, path)

	idx.mu.RLock()
	oldHash, exists := idx.fileHashes[relPath]
	idx.mu.RUnlock()

	if exists && oldHash == hash {
		return nil // File hasn't changed
	}

	// Remove old documents for this file
	if err := idx.store.RemoveByPath(relPath); err != nil {
		return fmt.Errorf("failed to remove old documents: %w", err)
	}

	// Parse file for symbols if supported
	outline, _ := idx.analyzer.ParseFile(path)

	// Index symbols if available
	if outline != nil && len(outline.Symbols) > 0 {
		if err := idx.indexSymbols(ctx, relPath, language, outline); err != nil {
			return err
		}
	}

	// Also index file content in chunks for general search
	if err := idx.indexContent(ctx, relPath, language, string(content)); err != nil {
		return err
	}

	// Update hash cache
	idx.mu.Lock()
	idx.fileHashes[relPath] = hash
	idx.mu.Unlock()

	return nil
}

// indexSymbols indexes individual symbols from a file
func (idx *Indexer) indexSymbols(ctx context.Context, path, language string, outline *analysis.FileOutline) error {
	for i, sym := range outline.Symbols {
		// Create document content from symbol
		content := sym.Name
		if sym.Signature != "" {
			content = sym.Signature
		}
		if sym.Doc != "" {
			content = fmt.Sprintf("%s\n%s", content, sym.Doc)
		}

		// Generate embedding
		embedding, err := idx.embedClient.Embed(ctx, content)
		if err != nil {
			continue // Skip failed embeddings
		}

		doc := &Document{
			ID:        fmt.Sprintf("%s:symbol:%d", path, i),
			Path:      path,
			Content:   content,
			Chunk:     i,
			Symbol:    sym.Name,
			Kind:      string(sym.Kind),
			Language:  language,
			Embedding: embedding,
		}

		if err := idx.store.Add(doc); err != nil {
			continue
		}
	}

	return nil
}

// indexContent indexes file content in chunks
func (idx *Indexer) indexContent(ctx context.Context, path, language, content string) error {
	chunks := chunkContent(content, 500) // ~500 chars per chunk

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		embedding, err := idx.embedClient.Embed(ctx, chunk)
		if err != nil {
			continue // Skip failed embeddings
		}

		doc := &Document{
			ID:        fmt.Sprintf("%s:chunk:%d", path, i),
			Path:      path,
			Content:   chunk,
			Chunk:     i,
			Language:  language,
			Embedding: embedding,
		}

		if err := idx.store.Add(doc); err != nil {
			continue
		}
	}

	return nil
}

// IndexDirectory recursively indexes a directory
func (idx *Indexer) IndexDirectory(ctx context.Context, dir string, progress func(path string)) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			if idx.shouldIgnore(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if idx.shouldIgnore(path) {
			return nil
		}

		if progress != nil {
			progress(path)
		}

		return idx.IndexFile(ctx, path)
	})
}

// IndexWorkspace indexes the entire workspace
func (idx *Indexer) IndexWorkspace(ctx context.Context, progress func(path string)) error {
	return idx.IndexDirectory(ctx, idx.workDir, progress)
}

// Search performs semantic search across the index
func (idx *Indexer) Search(ctx context.Context, query string, limit int, filter *SearchFilter) ([]SearchResult, error) {
	// Generate embedding for query
	queryEmbedding, err := idx.embedClient.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	results := idx.store.Search(queryEmbedding, limit, filter)
	return results, nil
}

// GetStats returns indexing statistics
func (idx *Indexer) GetStats() StoreStats {
	return idx.store.Stats()
}

// GetFileCount returns the number of indexed files
func (idx *Indexer) GetFileCount() int {
	stats := idx.store.Stats()
	return stats.UniqueFiles
}

// GetDocumentCount returns the total number of indexed documents
func (idx *Indexer) GetDocumentCount() int {
	return idx.store.Count()
}

// Close closes the indexer and releases resources
func (idx *Indexer) Close() error {
	return idx.store.Close()
}

// NeedsReindex checks if a file needs re-indexing
func (idx *Indexer) NeedsReindex(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	hash := hashContent(content)
	relPath, _ := filepath.Rel(idx.workDir, path)

	idx.mu.RLock()
	oldHash, exists := idx.fileHashes[relPath]
	idx.mu.RUnlock()

	return !exists || oldHash != hash, nil
}

// RefreshFile re-indexes a file if it has changed
func (idx *Indexer) RefreshFile(ctx context.Context, path string) (bool, error) {
	needsReindex, err := idx.NeedsReindex(path)
	if err != nil {
		return false, err
	}

	if needsReindex {
		return true, idx.IndexFile(ctx, path)
	}

	return false, nil
}

// chunkContent splits content into chunks for embedding
func chunkContent(content string, chunkSize int) []string {
	lines := strings.Split(content, "\n")
	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0

	for _, line := range lines {
		lineLen := len(line)

		if currentSize+lineLen > chunkSize && currentSize > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentSize = 0
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
		currentSize += lineLen + 1
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// hashContent generates a SHA256 hash of content
func hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// WatchForChanges starts watching for file changes (placeholder)
func (idx *Indexer) WatchForChanges(ctx context.Context, onChange func(path string)) error {
	// This is a placeholder - full implementation would use fsnotify
	// For now, users can call RefreshFile manually or use IndexWorkspace
	return fmt.Errorf("file watching not yet implemented - use RefreshFile or IndexWorkspace")
}

// LastIndexed returns when a file was last indexed
func (idx *Indexer) LastIndexed(path string) (time.Time, bool) {
	relPath, _ := filepath.Rel(idx.workDir, path)
	docs := idx.store.GetByPath(relPath)
	if len(docs) == 0 {
		return time.Time{}, false
	}
	return docs[0].UpdatedAt, true
}
