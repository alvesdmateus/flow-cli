package indexing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChunkContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		chunkSize int
		wantMin   int // Minimum expected chunks
	}{
		{"empty", "", 100, 0},
		{"single line", "hello world", 100, 1},
		{"multiple lines under limit", "line1\nline2\nline3", 100, 1},
		// chunkContent splits by lines, so a single long line = 1 chunk
		{"long single line", strings.Repeat("word ", 100), 50, 1},
		// Multiple lines should chunk properly
		{"force chunking", strings.Repeat("line\n", 100), 20, 5}, // 100 lines of 5 chars each = 500 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkContent(tt.content, tt.chunkSize)

			// For empty content, allow 0 or 1 empty chunks
			if tt.content == "" {
				if len(chunks) > 1 {
					t.Errorf("chunkContent() got %d chunks for empty content, want 0 or 1", len(chunks))
				}
				return
			}

			if len(chunks) < tt.wantMin {
				t.Errorf("chunkContent() got %d chunks, want at least %d", len(chunks), tt.wantMin)
			}

			// Verify content is preserved (combined chunks should contain all original content)
			combined := strings.Join(chunks, "")
			for _, word := range strings.Fields(tt.content) {
				if !strings.Contains(combined, word) {
					t.Errorf("Content not preserved: missing %q", word)
				}
			}
		})
	}
}

func TestHashContent(t *testing.T) {
	content1 := []byte("hello world")
	content2 := []byte("hello world")
	content3 := []byte("different content")

	hash1 := hashContent(content1)
	hash2 := hashContent(content2)
	hash3 := hashContent(content3)

	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}

	if len(hash1) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("Expected 64 character hash, got %d", len(hash1))
	}
}

func TestSupportedExtensions(t *testing.T) {
	exts := supportedExtensions()

	// Check some expected extensions
	if _, ok := exts[".go"]; !ok {
		t.Error("Expected .go to be supported")
	}
	if _, ok := exts[".py"]; !ok {
		t.Error("Expected .py to be supported")
	}
	if _, ok := exts[".js"]; !ok {
		t.Error("Expected .js to be supported")
	}
	if _, ok := exts[".ts"]; !ok {
		t.Error("Expected .ts to be supported")
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		path     string
		wantLang string
		wantSupp bool
	}{
		{"main.go", "go", true},
		{"script.py", "python", true},
		{"app.js", "javascript", true},
		{"component.tsx", "typescript", true},
		{"readme.md", "markdown", true},
		{"data.json", "", false},
		{"image.png", "", false},
		{"no_extension", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			lang, supported := isSupported(tt.path)
			if supported != tt.wantSupp {
				t.Errorf("isSupported(%q) supported = %v, want %v", tt.path, supported, tt.wantSupp)
			}
			if lang != tt.wantLang {
				t.Errorf("isSupported(%q) language = %q, want %q", tt.path, lang, tt.wantLang)
			}
		})
	}
}

func TestIndexer_ShouldIgnore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal indexer for testing ignore rules
	idx := &Indexer{
		workDir: tmpDir,
		ignoreRules: []string{
			".git",
			"node_modules",
			"*.min.js",
			"vendor",
		},
	}

	tests := []struct {
		path       string
		wantIgnore bool
	}{
		{filepath.Join(tmpDir, ".git", "config"), true},
		{filepath.Join(tmpDir, "node_modules", "package", "index.js"), true},
		{filepath.Join(tmpDir, "vendor", "lib.go"), true},
		{filepath.Join(tmpDir, "bundle.min.js"), true},
		{filepath.Join(tmpDir, "main.go"), false},
		{filepath.Join(tmpDir, "src", "app.js"), false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := idx.shouldIgnore(tt.path)
			if got != tt.wantIgnore {
				t.Errorf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.wantIgnore)
			}
		})
	}
}

func TestIndexer_LoadIgnoreRules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .flowignore file
	flowignore := filepath.Join(tmpDir, ".flowignore")
	content := `# Comment line
custom_dir
*.log
temp_*
`
	if err := os.WriteFile(flowignore, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write .flowignore: %v", err)
	}

	idx := &Indexer{
		workDir: tmpDir,
	}
	idx.loadIgnoreRules()

	// Check that custom rules were loaded
	customRulesFound := 0
	for _, rule := range idx.ignoreRules {
		if rule == "custom_dir" || rule == "*.log" || rule == "temp_*" {
			customRulesFound++
		}
	}

	if customRulesFound != 3 {
		t.Errorf("Expected 3 custom rules to be loaded, found %d", customRulesFound)
	}

	// Check that default rules are still present
	hasGit := false
	for _, rule := range idx.ignoreRules {
		if rule == ".git" {
			hasGit = true
			break
		}
	}
	if !hasGit {
		t.Error("Default .git rule should be present")
	}
}

func TestIndexer_LoadIgnoreRules_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	idx := &Indexer{
		workDir: tmpDir,
	}
	idx.loadIgnoreRules()

	// Should have default rules even without .flowignore
	if len(idx.ignoreRules) == 0 {
		t.Error("Should have default ignore rules even without .flowignore")
	}

	hasNodeModules := false
	for _, rule := range idx.ignoreRules {
		if rule == "node_modules" {
			hasNodeModules = true
			break
		}
	}
	if !hasNodeModules {
		t.Error("Default node_modules rule should be present")
	}
}

func TestSearchFilter_PathPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	docs := []*Document{
		{ID: "doc-1", Path: "src/auth/login.go", Embedding: []float64{0.9, 0.1}},
		{ID: "doc-2", Path: "src/auth/logout.go", Embedding: []float64{0.8, 0.2}},
		{ID: "doc-3", Path: "src/user/profile.go", Embedding: []float64{0.7, 0.3}},
		{ID: "doc-4", Path: "pkg/utils.go", Embedding: []float64{0.6, 0.4}},
	}

	for _, doc := range docs {
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	queryEmbedding := []float64{0.85, 0.15}

	// Filter by path prefix
	results := store.Search(queryEmbedding, 10, &SearchFilter{PathPrefix: []string{"src/auth/"}})
	if len(results) != 2 {
		t.Errorf("Expected 2 results for src/auth/ prefix, got %d", len(results))
	}

	results = store.Search(queryEmbedding, 10, &SearchFilter{PathPrefix: []string{"src/"}})
	if len(results) != 3 {
		t.Errorf("Expected 3 results for src/ prefix, got %d", len(results))
	}

	results = store.Search(queryEmbedding, 10, &SearchFilter{PathPrefix: []string{"nonexistent/"}})
	if len(results) != 0 {
		t.Errorf("Expected 0 results for nonexistent prefix, got %d", len(results))
	}
}

func TestSearchResult_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add 20 documents
	for i := 0; i < 20; i++ {
		err := store.Add(&Document{
			ID:        string(rune('a' + i)),
			Path:      "file.go",
			Content:   "content",
			Embedding: []float64{float64(i) / 20, 1 - float64(i)/20},
		})
		if err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	queryEmbedding := []float64{0.5, 0.5}

	// Test limit
	results := store.Search(queryEmbedding, 5, nil)
	if len(results) != 5 {
		t.Errorf("Expected 5 results with limit=5, got %d", len(results))
	}

	results = store.Search(queryEmbedding, 100, nil)
	if len(results) != 20 {
		t.Errorf("Expected 20 results (all), got %d", len(results))
	}

	results = store.Search(queryEmbedding, 0, nil) // 0 should use default
	if len(results) != 10 {
		t.Errorf("Expected 10 results (default), got %d", len(results))
	}
}
