package indexing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Document represents an indexed document
type Document struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Chunk     int       `json:"chunk"`  // Chunk index within file
	Symbol    string    `json:"symbol"` // Associated symbol name (if any)
	Kind      string    `json:"kind"`   // Symbol kind (function, class, etc.)
	Language  string    `json:"language"`
	Embedding []float64 `json:"embedding"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document   Document `json:"document"`
	Score      float64  `json:"score"`      // Cosine similarity score (0-1)
	Highlights []string `json:"highlights"` // Relevant snippets
}

// VectorStore manages document embeddings and similarity search
type VectorStore struct {
	mu        sync.RWMutex
	documents map[string]*Document // In-memory cache
	db        *sql.DB              // SQLite persistence
	dbPath    string
}

// NewVectorStore creates a new vector store
func NewVectorStore(dbPath string) (*VectorStore, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &VectorStore{
		documents: make(map[string]*Document),
		db:        db,
		dbPath:    dbPath,
	}

	if err := store.initDB(); err != nil {
		db.Close()
		return nil, err
	}

	if err := store.loadFromDB(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// initDB creates the database schema
func (s *VectorStore) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		path TEXT NOT NULL,
		content TEXT NOT NULL,
		chunk INTEGER DEFAULT 0,
		symbol TEXT,
		kind TEXT,
		language TEXT,
		embedding BLOB,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_documents_path ON documents(path);
	CREATE INDEX IF NOT EXISTS idx_documents_symbol ON documents(symbol);
	CREATE INDEX IF NOT EXISTS idx_documents_kind ON documents(kind);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// loadFromDB loads all documents from the database
func (s *VectorStore) loadFromDB() error {
	rows, err := s.db.Query(`
		SELECT id, path, content, chunk, symbol, kind, language, embedding, updated_at
		FROM documents
	`)
	if err != nil {
		return fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var doc Document
		var embeddingJSON []byte
		var symbol, kind, language sql.NullString
		var updatedAt string

		err := rows.Scan(
			&doc.ID, &doc.Path, &doc.Content, &doc.Chunk,
			&symbol, &kind, &language, &embeddingJSON, &updatedAt,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		if symbol.Valid {
			doc.Symbol = symbol.String
		}
		if kind.Valid {
			doc.Kind = kind.String
		}
		if language.Valid {
			doc.Language = language.String
		}

		if len(embeddingJSON) > 0 {
			if err := json.Unmarshal(embeddingJSON, &doc.Embedding); err != nil {
				continue
			}
		}

		doc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		s.documents[doc.ID] = &doc
	}

	return nil
}

// Add adds or updates a document in the store
func (s *VectorStore) Add(doc *Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc.UpdatedAt = time.Now()

	// Store in memory
	s.documents[doc.ID] = doc

	// Persist to database
	embeddingJSON, err := json.Marshal(doc.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO documents
		(id, path, content, chunk, symbol, kind, language, embedding, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		doc.ID, doc.Path, doc.Content, doc.Chunk,
		nullString(doc.Symbol), nullString(doc.Kind), nullString(doc.Language),
		embeddingJSON, doc.UpdatedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

// Remove removes a document by ID
func (s *VectorStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.documents, id)

	_, err := s.db.Exec("DELETE FROM documents WHERE id = ?", id)
	return err
}

// RemoveByPath removes all documents for a given file path
func (s *VectorStore) RemoveByPath(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from memory
	for id, doc := range s.documents {
		if doc.Path == path {
			delete(s.documents, id)
		}
	}

	// Remove from database
	_, err := s.db.Exec("DELETE FROM documents WHERE path = ?", path)
	return err
}

// Get retrieves a document by ID
func (s *VectorStore) Get(id string) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	return doc, ok
}

// Search performs semantic similarity search
func (s *VectorStore) Search(queryEmbedding []float64, limit int, filter *SearchFilter) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	var results []SearchResult

	for _, doc := range s.documents {
		// Apply filters
		if filter != nil {
			if filter.Path != "" && doc.Path != filter.Path {
				continue
			}
			if filter.Kind != "" && doc.Kind != filter.Kind {
				continue
			}
			if filter.Language != "" && doc.Language != filter.Language {
				continue
			}
			if len(filter.PathPrefix) > 0 {
				match := false
				for _, prefix := range filter.PathPrefix {
					if len(doc.Path) >= len(prefix) && doc.Path[:len(prefix)] == prefix {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
		}

		// Skip documents without embeddings
		if len(doc.Embedding) == 0 {
			continue
		}

		score := cosineSimilarity(queryEmbedding, doc.Embedding)
		results = append(results, SearchResult{
			Document: *doc,
			Score:    score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// SearchFilter defines filters for search queries
type SearchFilter struct {
	Path       string   // Exact path match
	PathPrefix []string // Path prefix matches
	Kind       string   // Symbol kind filter
	Language   string   // Language filter
}

// Count returns the number of indexed documents
func (s *VectorStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

// GetByPath returns all documents for a given file path
func (s *VectorStore) GetByPath(path string) []*Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var docs []*Document
	for _, doc := range s.documents {
		if doc.Path == path {
			docs = append(docs, doc)
		}
	}
	return docs
}

// Clear removes all documents
func (s *VectorStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents = make(map[string]*Document)
	_, err := s.db.Exec("DELETE FROM documents")
	return err
}

// Close closes the database connection
func (s *VectorStore) Close() error {
	return s.db.Close()
}

// Stats returns statistics about the store
func (s *VectorStore) Stats() StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := StoreStats{
		TotalDocuments: len(s.documents),
		Languages:      make(map[string]int),
		Kinds:          make(map[string]int),
	}

	paths := make(map[string]bool)
	for _, doc := range s.documents {
		paths[doc.Path] = true
		if doc.Language != "" {
			stats.Languages[doc.Language]++
		}
		if doc.Kind != "" {
			stats.Kinds[doc.Kind]++
		}
	}
	stats.UniqueFiles = len(paths)

	return stats
}

// StoreStats contains statistics about the vector store
type StoreStats struct {
	TotalDocuments int            `json:"total_documents"`
	UniqueFiles    int            `json:"unique_files"`
	Languages      map[string]int `json:"languages"`
	Kinds          map[string]int `json:"kinds"`
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// nullString converts empty strings to sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
