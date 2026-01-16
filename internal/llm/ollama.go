package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

// OllamaClient implements the Client interface for Ollama
type OllamaClient struct {
	client   *api.Client
	endpoint string
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(endpoint string) (*OllamaClient, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Minute, // Long timeout for generation
	}

	client := api.NewClient(parsedURL, httpClient)

	return &OllamaClient{
		client:   client,
		endpoint: endpoint,
	}, nil
}

// Provider returns the provider name
func (o *OllamaClient) Provider() string {
	return "ollama"
}

// Ping checks if Ollama is available
func (o *OllamaClient) Ping(ctx context.Context) error {
	_, err := o.client.List(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionError, err)
	}
	return nil
}

// ListModels returns available models from Ollama
func (o *OllamaClient) ListModels(ctx context.Context) ([]Model, error) {
	resp, err := o.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	models := make([]Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, Model{
			Name:       m.Name,
			Size:       m.Size,
			ModifiedAt: m.ModifiedAt.Format(time.RFC3339),
		})
	}

	return models, nil
}

// Chat sends messages and returns a channel of streamed responses
func (o *OllamaClient) Chat(ctx context.Context, messages []Message, opts ChatOptions) (<-chan StreamChunk, error) {
	ollamaMessages := make([]api.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = api.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	req := &api.ChatRequest{
		Model:    opts.Model,
		Messages: ollamaMessages,
		Stream:   boolPtr(true),
		Options: map[string]interface{}{
			"temperature": opts.Temperature,
		},
	}

	chunks := make(chan StreamChunk, 100)

	go func() {
		defer close(chunks)

		err := o.client.Chat(ctx, req, func(resp api.ChatResponse) error {
			chunks <- StreamChunk{
				Content: resp.Message.Content,
				Done:    resp.Done,
			}
			return nil
		})

		if err != nil {
			chunks <- StreamChunk{
				Error: fmt.Errorf("chat error: %w", err),
			}
		}
	}()

	return chunks, nil
}

// ChatSync sends messages and waits for complete response
func (o *OllamaClient) ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (string, error) {
	chunks, err := o.Chat(ctx, messages, opts)
	if err != nil {
		return "", err
	}

	var response strings.Builder
	for chunk := range chunks {
		if chunk.Error != nil {
			return "", chunk.Error
		}
		response.WriteString(chunk.Content)
	}

	return response.String(), nil
}

func boolPtr(b bool) *bool {
	return &b
}
