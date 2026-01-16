package indexing

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewVectorStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	if store.Count() != 0 {
		t.Errorf("Expected empty store, got %d documents", store.Count())
	}
}

func TestVectorStore_AddAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	doc := &Document{
		ID:        "test-doc-1",
		Path:      "main.go",
		Content:   "func main() {}",
		Symbol:    "main",
		Kind:      "function",
		Language:  "go",
		Embedding: []float64{0.1, 0.2, 0.3, 0.4},
	}

	if err := store.Add(doc); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Retrieve document
	retrieved, ok := store.Get("test-doc-1")
	if !ok {
		t.Fatal("Document not found")
	}

	if retrieved.Path != "main.go" {
		t.Errorf("Expected path 'main.go', got '%s'", retrieved.Path)
	}
	if retrieved.Symbol != "main" {
		t.Errorf("Expected symbol 'main', got '%s'", retrieved.Symbol)
	}
	if len(retrieved.Embedding) != 4 {
		t.Errorf("Expected 4 embedding dimensions, got %d", len(retrieved.Embedding))
	}
}

func TestVectorStore_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	doc := &Document{
		ID:        "test-doc-1",
		Path:      "main.go",
		Content:   "func main() {}",
		Embedding: []float64{0.1, 0.2, 0.3},
	}

	if err := store.Add(doc); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Expected 1 document, got %d", store.Count())
	}

	if err := store.Remove("test-doc-1"); err != nil {
		t.Fatalf("Failed to remove document: %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("Expected 0 documents after removal, got %d", store.Count())
	}

	_, ok := store.Get("test-doc-1")
	if ok {
		t.Error("Document should not be found after removal")
	}
}

func TestVectorStore_RemoveByPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add multiple documents for the same path
	for i := 0; i < 3; i++ {
		doc := &Document{
			ID:        "doc-" + string(rune('a'+i)),
			Path:      "main.go",
			Content:   "chunk " + string(rune('a'+i)),
			Chunk:     i,
			Embedding: []float64{float64(i), 0.2, 0.3},
		}
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	// Add document for different path
	if err := store.Add(&Document{
		ID:        "other-doc",
		Path:      "other.go",
		Content:   "other content",
		Embedding: []float64{0.9, 0.8, 0.7},
	}); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	if store.Count() != 4 {
		t.Errorf("Expected 4 documents, got %d", store.Count())
	}

	if err := store.RemoveByPath("main.go"); err != nil {
		t.Fatalf("Failed to remove documents: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Expected 1 document after removal, got %d", store.Count())
	}

	_, ok := store.Get("other-doc")
	if !ok {
		t.Error("other-doc should still exist")
	}
}

func TestVectorStore_Search(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add documents with embeddings
	docs := []*Document{
		{ID: "doc-1", Path: "auth.go", Content: "authentication", Embedding: []float64{0.9, 0.1, 0.0}},
		{ID: "doc-2", Path: "user.go", Content: "user management", Embedding: []float64{0.7, 0.3, 0.1}},
		{ID: "doc-3", Path: "config.go", Content: "configuration", Embedding: []float64{0.1, 0.8, 0.2}},
	}

	for _, doc := range docs {
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	// Search with query embedding similar to auth.go
	queryEmbedding := []float64{0.85, 0.15, 0.0}
	results := store.Search(queryEmbedding, 10, nil)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// First result should be auth.go (most similar)
	if results[0].Document.Path != "auth.go" {
		t.Errorf("Expected first result to be auth.go, got %s", results[0].Document.Path)
	}

	// Scores should be descending
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Error("Results should be sorted by score descending")
		}
	}
}

func TestVectorStore_SearchWithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	docs := []*Document{
		{ID: "doc-1", Path: "auth.go", Kind: "function", Language: "go", Embedding: []float64{0.9, 0.1}},
		{ID: "doc-2", Path: "auth.py", Kind: "function", Language: "python", Embedding: []float64{0.8, 0.2}},
		{ID: "doc-3", Path: "user.go", Kind: "struct", Language: "go", Embedding: []float64{0.7, 0.3}},
	}

	for _, doc := range docs {
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	queryEmbedding := []float64{0.85, 0.15}

	// Filter by language
	results := store.Search(queryEmbedding, 10, &SearchFilter{Language: "go"})
	if len(results) != 2 {
		t.Errorf("Expected 2 Go results, got %d", len(results))
	}

	// Filter by kind
	results = store.Search(queryEmbedding, 10, &SearchFilter{Kind: "function"})
	if len(results) != 2 {
		t.Errorf("Expected 2 function results, got %d", len(results))
	}

	// Combined filter
	results = store.Search(queryEmbedding, 10, &SearchFilter{Language: "go", Kind: "function"})
	if len(results) != 1 {
		t.Errorf("Expected 1 result with combined filter, got %d", len(results))
	}
}

func TestVectorStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create and populate store
	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	doc := &Document{
		ID:        "persistent-doc",
		Path:      "main.go",
		Content:   "persistent content",
		Symbol:    "main",
		Kind:      "function",
		Language:  "go",
		Embedding: []float64{0.1, 0.2, 0.3},
	}
	if err := store.Add(doc); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}
	store.Close()

	// Re-open store and verify data persisted
	store2, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to re-open store: %v", err)
	}
	defer store2.Close()

	if store2.Count() != 1 {
		t.Errorf("Expected 1 document after reopening, got %d", store2.Count())
	}

	retrieved, ok := store2.Get("persistent-doc")
	if !ok {
		t.Fatal("Document not found after reopening")
	}

	if retrieved.Content != "persistent content" {
		t.Errorf("Expected content 'persistent content', got '%s'", retrieved.Content)
	}
}

func TestVectorStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add some documents
	for i := 0; i < 5; i++ {
		if err := store.Add(&Document{
			ID:        "doc-" + string(rune('a'+i)),
			Path:      "file.go",
			Content:   "content",
			Embedding: []float64{0.1},
		}); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	if store.Count() != 5 {
		t.Errorf("Expected 5 documents, got %d", store.Count())
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Failed to clear store: %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("Expected 0 documents after clear, got %d", store.Count())
	}
}

func TestVectorStore_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	docs := []*Document{
		{ID: "doc-1", Path: "main.go", Kind: "function", Language: "go", Embedding: []float64{0.1}},
		{ID: "doc-2", Path: "main.go", Kind: "struct", Language: "go", Embedding: []float64{0.2}},
		{ID: "doc-3", Path: "app.py", Kind: "function", Language: "python", Embedding: []float64{0.3}},
	}

	for _, doc := range docs {
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	stats := store.Stats()

	if stats.TotalDocuments != 3 {
		t.Errorf("Expected 3 total documents, got %d", stats.TotalDocuments)
	}

	if stats.UniqueFiles != 2 {
		t.Errorf("Expected 2 unique files, got %d", stats.UniqueFiles)
	}

	if stats.Languages["go"] != 2 {
		t.Errorf("Expected 2 Go documents, got %d", stats.Languages["go"])
	}

	if stats.Kinds["function"] != 2 {
		t.Errorf("Expected 2 functions, got %d", stats.Kinds["function"])
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		a, b     []float64
		expected float64
	}{
		{[]float64{1, 0, 0}, []float64{1, 0, 0}, 1.0},      // Identical vectors
		{[]float64{1, 0, 0}, []float64{0, 1, 0}, 0.0},      // Orthogonal vectors
		{[]float64{1, 0, 0}, []float64{-1, 0, 0}, -1.0},    // Opposite vectors
		{[]float64{1, 1, 0}, []float64{1, 0, 0}, 0.707107}, // 45 degree angle
		{[]float64{}, []float64{}, 0},                      // Empty vectors
		{[]float64{1, 2}, []float64{1, 2, 3}, 0},           // Different lengths
	}

	for i, tt := range tests {
		result := cosineSimilarity(tt.a, tt.b)
		if math.Abs(result-tt.expected) > 0.001 {
			t.Errorf("Test %d: cosineSimilarity(%v, %v) = %f, want %f", i, tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestVectorStore_GetByPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add multiple documents for the same path
	for i := 0; i < 3; i++ {
		if err := store.Add(&Document{
			ID:        "doc-" + string(rune('a'+i)),
			Path:      "main.go",
			Content:   "chunk " + string(rune('a'+i)),
			Chunk:     i,
			Embedding: []float64{float64(i)},
		}); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	if err := store.Add(&Document{
		ID:        "other",
		Path:      "other.go",
		Content:   "other",
		Embedding: []float64{0.9},
	}); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	docs := store.GetByPath("main.go")
	if len(docs) != 3 {
		t.Errorf("Expected 3 documents for main.go, got %d", len(docs))
	}

	docs = store.GetByPath("other.go")
	if len(docs) != 1 {
		t.Errorf("Expected 1 document for other.go, got %d", len(docs))
	}

	docs = store.GetByPath("nonexistent.go")
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents for nonexistent.go, got %d", len(docs))
	}
}

func TestDocument_UpdatedAt(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	before := time.Now()

	doc := &Document{
		ID:        "test-doc",
		Path:      "main.go",
		Content:   "content",
		Embedding: []float64{0.1},
	}
	if err := store.Add(doc); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	after := time.Now()

	retrieved, ok := store.Get("test-doc")
	if !ok {
		t.Fatal("Document not found")
	}

	if retrieved.UpdatedAt.Before(before) || retrieved.UpdatedAt.After(after) {
		t.Error("UpdatedAt should be between before and after times")
	}
}

func TestVectorStore_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	store, err := NewVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store with nested path: %v", err)
	}
	defer store.Close()

	// Verify the file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}
