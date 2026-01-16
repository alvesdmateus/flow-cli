package indexing

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

// EmbeddingClient generates embeddings using Ollama
type EmbeddingClient struct {
	client *api.Client
	model  string
}

// NewEmbeddingClient creates a new embedding client
func NewEmbeddingClient(endpoint, model string) (*EmbeddingClient, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	if model == "" {
		model = "nomic-embed-text" // Default embedding model
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}

	client := api.NewClient(parsedURL, httpClient)

	return &EmbeddingClient{
		client: client,
		model:  model,
	}, nil
}

// SetModel changes the embedding model
func (e *EmbeddingClient) SetModel(model string) {
	e.model = model
}

// GetModel returns the current embedding model
func (e *EmbeddingClient) GetModel() string {
	return e.model
}

// Embed generates an embedding for the given text
func (e *EmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error) {
	req := &api.EmbedRequest{
		Model: e.model,
		Input: text,
	}

	resp, err := e.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert float32 to float64
	embedding := make([]float64, len(resp.Embeddings[0]))
	for i, v := range resp.Embeddings[0] {
		embedding[i] = float64(v)
	}

	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *EmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))

	for i, text := range texts {
		emb, err := e.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// Ping checks if the embedding service is available
func (e *EmbeddingClient) Ping(ctx context.Context) error {
	_, err := e.client.List(ctx)
	if err != nil {
		return fmt.Errorf("embedding service unavailable: %w", err)
	}
	return nil
}

// CheckModel verifies the embedding model is available
func (e *EmbeddingClient) CheckModel(ctx context.Context) error {
	// Try a simple embedding to verify the model works
	_, err := e.Embed(ctx, "test")
	if err != nil {
		return fmt.Errorf("embedding model '%s' not available: %w", e.model, err)
	}
	return nil
}
