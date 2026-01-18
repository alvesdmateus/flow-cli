package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatClient implements the Client interface for OpenAI-compatible APIs
// Works with: LocalAI, LM Studio, vLLM, text-generation-webui, etc.
type OpenAICompatClient struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

// OpenAI API request/response types

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      chatMessage `json:"message"`
		Delta        chatMessage `json:"delta"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type modelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

type streamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// NewOpenAICompatClient creates a new OpenAI-compatible client
func NewOpenAICompatClient(endpoint, apiKey string) (*OpenAICompatClient, error) {
	if endpoint == "" {
		endpoint = "http://localhost:8080" // LocalAI default
	}

	// Ensure endpoint doesn't have trailing slash
	endpoint = strings.TrimSuffix(endpoint, "/")

	// Ensure endpoint has /v1 suffix for API calls
	if !strings.HasSuffix(endpoint, "/v1") {
		endpoint = endpoint + "/v1"
	}

	return &OpenAICompatClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

// Provider returns the provider name
func (c *OpenAICompatClient) Provider() string {
	return "openai-compatible"
}

// Ping checks if the API is available
func (c *OpenAICompatClient) Ping(ctx context.Context) error {
	_, err := c.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionError, err)
	}
	return nil
}

// ListModels returns available models
func (c *OpenAICompatClient) ListModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var modelsResp modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]Model, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		models[i] = Model{
			Name:       m.ID,
			ModifiedAt: time.Unix(m.Created, 0).Format(time.RFC3339),
		}
	}

	return models, nil
}

// Chat sends messages and returns a channel of streamed responses
func (c *OpenAICompatClient) Chat(ctx context.Context, messages []Message, opts ChatOptions) (<-chan StreamChunk, error) {
	// Convert messages
	chatMessages := make([]chatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = chatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	reqBody := chatRequest{
		Model:       opts.Model,
		Messages:    chatMessages,
		Temperature: opts.Temperature,
		Stream:      true,
	}

	if opts.MaxTokens > 0 {
		reqBody.MaxTokens = opts.MaxTokens
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req) //nolint:bodyclose // Body is closed in goroutine below
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	chunks := make(chan StreamChunk, 100)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{Error: err}
				}
				return
			}

			line = strings.TrimSpace(line)

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// Parse SSE data
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				chunks <- StreamChunk{Done: true}
				return
			}

			// Parse chunk
			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue // Skip malformed chunks
			}

			// Extract content
			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				if content != "" {
					chunks <- StreamChunk{Content: content}
				}

				// Check if done
				if chunk.Choices[0].FinishReason != nil {
					chunks <- StreamChunk{Done: true}
					return
				}
			}
		}
	}()

	return chunks, nil
}

// ChatSync sends messages and waits for complete response
func (c *OpenAICompatClient) ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (string, error) {
	// Convert messages
	chatMessages := make([]chatMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = chatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	reqBody := chatRequest{
		Model:       opts.Model,
		Messages:    chatMessages,
		Temperature: opts.Temperature,
		Stream:      false,
	}

	if opts.MaxTokens > 0 {
		reqBody.MaxTokens = opts.MaxTokens
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// setHeaders sets common headers for API requests
func (c *OpenAICompatClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// Presets for common providers

// LocalAIEndpoint is the default LocalAI endpoint
const LocalAIEndpoint = "http://localhost:8080"

// LMStudioEndpoint is the default LM Studio endpoint
const LMStudioEndpoint = "http://localhost:1234"

// VLLMEndpoint is the default vLLM endpoint
const VLLMEndpoint = "http://localhost:8000"

// TextGenWebUIEndpoint is the default text-generation-webui endpoint
const TextGenWebUIEndpoint = "http://localhost:5000"

// ProviderPreset contains preset configuration for a provider
type ProviderPreset struct {
	Name        string
	Endpoint    string
	Description string
	RequiresKey bool
}

// GetProviderPresets returns presets for known providers
func GetProviderPresets() []ProviderPreset {
	return []ProviderPreset{
		{
			Name:        "ollama",
			Endpoint:    "http://localhost:11434",
			Description: "Ollama - Run LLMs locally",
			RequiresKey: false,
		},
		{
			Name:        "localai",
			Endpoint:    LocalAIEndpoint,
			Description: "LocalAI - OpenAI-compatible local AI",
			RequiresKey: false,
		},
		{
			Name:        "lmstudio",
			Endpoint:    LMStudioEndpoint,
			Description: "LM Studio - User-friendly local LLM",
			RequiresKey: false,
		},
		{
			Name:        "vllm",
			Endpoint:    VLLMEndpoint,
			Description: "vLLM - Fast LLM inference server",
			RequiresKey: false,
		},
		{
			Name:        "textgen",
			Endpoint:    TextGenWebUIEndpoint,
			Description: "text-generation-webui - Gradio web UI",
			RequiresKey: false,
		},
		{
			Name:        "openai",
			Endpoint:    "https://api.openai.com",
			Description: "OpenAI API (cloud)",
			RequiresKey: true,
		},
	}
}
