package llm

import (
	"context"
	"errors"
)

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a chat message
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Model represents an available LLM model
type Model struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

// ChatOptions holds options for chat requests
type ChatOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
	Stream      bool
}

// StreamChunk represents a chunk of streamed response
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// Client defines the interface for LLM providers
type Client interface {
	// Chat sends messages and returns a channel of streamed responses
	Chat(ctx context.Context, messages []Message, opts ChatOptions) (<-chan StreamChunk, error)

	// ChatSync sends messages and waits for complete response
	ChatSync(ctx context.Context, messages []Message, opts ChatOptions) (string, error)

	// ListModels returns available models
	ListModels(ctx context.Context) ([]Model, error)

	// Ping checks if the LLM service is available
	Ping(ctx context.Context) error

	// Provider returns the name of this provider
	Provider() string
}

// Common errors
var (
	ErrNoModels        = errors.New("no models available")
	ErrModelNotFound   = errors.New("model not found")
	ErrConnectionError = errors.New("failed to connect to LLM service")
	ErrStreamClosed    = errors.New("response stream closed unexpectedly")
)

// ClientConfig holds configuration for creating a client
type ClientConfig struct {
	Provider string
	Endpoint string
	APIKey   string
}

// NewClient creates a new LLM client based on the provider
func NewClient(provider, endpoint string) (Client, error) {
	return NewClientWithConfig(ClientConfig{
		Provider: provider,
		Endpoint: endpoint,
	})
}

// NewClientWithConfig creates a new LLM client with full configuration
func NewClientWithConfig(cfg ClientConfig) (Client, error) {
	provider := normalizeProvider(cfg.Provider)

	switch provider {
	case "ollama":
		return NewOllamaClient(cfg.Endpoint)

	case "openai-compatible", "openai", "localai", "lmstudio", "vllm", "textgen":
		return NewOpenAICompatClient(cfg.Endpoint, cfg.APIKey)

	default:
		// Try auto-detection
		return autoDetectClient(cfg.Endpoint, cfg.APIKey)
	}
}

// normalizeProvider normalizes provider names
func normalizeProvider(provider string) string {
	switch provider {
	case "openai-compat", "openai-compatible", "openai_compatible":
		return "openai-compatible"
	case "local-ai", "local_ai":
		return "localai"
	case "lm-studio", "lm_studio":
		return "lmstudio"
	case "text-gen", "textgen-webui", "text-generation-webui":
		return "textgen"
	default:
		return provider
	}
}

// autoDetectClient attempts to detect the provider from the endpoint
func autoDetectClient(endpoint, apiKey string) (Client, error) {
	if endpoint == "" {
		// Default to Ollama
		return NewOllamaClient("")
	}

	// Check for known endpoints
	switch {
	case containsPort(endpoint, "11434"):
		// Ollama default port
		return NewOllamaClient(endpoint)

	case containsPort(endpoint, "8080"):
		// LocalAI default port
		return NewOpenAICompatClient(endpoint, apiKey)

	case containsPort(endpoint, "1234"):
		// LM Studio default port
		return NewOpenAICompatClient(endpoint, apiKey)

	case containsPort(endpoint, "8000"):
		// vLLM default port
		return NewOpenAICompatClient(endpoint, apiKey)

	case containsPort(endpoint, "5000"):
		// text-generation-webui default port
		return NewOpenAICompatClient(endpoint, apiKey)

	case containsHost(endpoint, "api.openai.com"):
		// OpenAI cloud API
		return NewOpenAICompatClient(endpoint, apiKey)

	default:
		// Try OpenAI-compatible as fallback
		return NewOpenAICompatClient(endpoint, apiKey)
	}
}

// containsPort checks if endpoint contains a specific port
func containsPort(endpoint, port string) bool {
	return len(endpoint) > 0 && (
	// Check for :port at end or :port/
	len(endpoint) > len(port)+1 &&
		(endpoint[len(endpoint)-len(port)-1:] == ":"+port ||
			(len(endpoint) > len(port)+2 && endpoint[len(endpoint)-len(port)-2:len(endpoint)-1] == ":"+port)))
}

// containsHost checks if endpoint contains a specific host
func containsHost(endpoint, host string) bool {
	return len(endpoint) > 0 && len(host) > 0 &&
		(endpoint == host ||
			len(endpoint) > len(host) &&
				(endpoint[:len(host)] == host ||
					endpoint[len(endpoint)-len(host):] == host))
}

// AvailableProviders returns a list of supported providers
func AvailableProviders() []string {
	return []string{
		"ollama",
		"openai-compatible",
		"localai",
		"lmstudio",
		"vllm",
		"textgen",
		"openai",
	}
}
