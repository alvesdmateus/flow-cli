package llm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

// OllamaManager handles Ollama server lifecycle and model management
type OllamaManager struct {
	client   *OllamaClient
	process  *os.Process
	endpoint string
}

// NewOllamaManager creates a new Ollama manager
func NewOllamaManager(endpoint string) (*OllamaManager, error) {
	client, err := NewOllamaClient(endpoint)
	if err != nil {
		return nil, err
	}

	return &OllamaManager{
		client:   client,
		endpoint: endpoint,
	}, nil
}

// EnsureRunning ensures Ollama server is running, starting it if necessary
func (m *OllamaManager) EnsureRunning(ctx context.Context) error {
	// First check if Ollama is already running
	if err := m.client.Ping(ctx); err == nil {
		return nil // Already running
	}

	// Check if Ollama is installed
	ollamaPath, err := m.findOllamaBinary()
	if err != nil {
		return fmt.Errorf("ollama not found: %w. Please install Ollama from https://ollama.ai", err)
	}

	// Start Ollama server
	cmd := exec.Command(ollamaPath, "serve")
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Ollama: %w", err)
	}

	m.process = cmd.Process

	// Wait for Ollama to be ready
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := m.client.Ping(ctx); err == nil {
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	return fmt.Errorf("timeout waiting for Ollama to start")
}

// findOllamaBinary locates the Ollama binary
func (m *OllamaManager) findOllamaBinary() (string, error) {
	// Check if ollama is in PATH
	if path, err := exec.LookPath("ollama"); err == nil {
		return path, nil
	}

	// Check common installation locations
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/usr/local/bin/ollama",
			"/opt/homebrew/bin/ollama",
			os.ExpandEnv("$HOME/.ollama/bin/ollama"),
		}
	case "linux":
		paths = []string{
			"/usr/local/bin/ollama",
			"/usr/bin/ollama",
			os.ExpandEnv("$HOME/.ollama/bin/ollama"),
		}
	case "windows":
		paths = []string{
			os.ExpandEnv("$LOCALAPPDATA\\Ollama\\ollama.exe"),
			os.ExpandEnv("$PROGRAMFILES\\Ollama\\ollama.exe"),
			"C:\\Program Files\\Ollama\\ollama.exe",
		}
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("ollama binary not found in PATH or common locations")
}

// EnsureModel ensures a model is available, pulling it if necessary
func (m *OllamaManager) EnsureModel(ctx context.Context, model string, progressFn func(status string, completed, total int64)) error {
	// Check if model exists
	models, err := m.client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Normalize model name (remove :latest suffix for comparison)
	normalizedModel := normalizeModelName(model)
	for _, m := range models {
		if normalizeModelName(m.Name) == normalizedModel {
			return nil // Model already exists
		}
	}

	// Model not found, pull it
	return m.PullModel(ctx, model, progressFn)
}

// PullModel downloads a model from the Ollama registry
func (m *OllamaManager) PullModel(ctx context.Context, model string, progressFn func(status string, completed, total int64)) error {
	req := &api.PullRequest{
		Model:  model,
		Stream: boolPtr(true),
	}

	return m.client.client.Pull(ctx, req, func(resp api.ProgressResponse) error {
		if progressFn != nil {
			progressFn(resp.Status, resp.Completed, resp.Total)
		}
		return nil
	})
}

// HasModel checks if a model is available locally
func (m *OllamaManager) HasModel(ctx context.Context, model string) (bool, error) {
	models, err := m.client.ListModels(ctx)
	if err != nil {
		return false, err
	}

	normalizedModel := normalizeModelName(model)
	for _, m := range models {
		if normalizeModelName(m.Name) == normalizedModel {
			return true, nil
		}
	}
	return false, nil
}

// DeleteModel removes a model from local storage
func (m *OllamaManager) DeleteModel(ctx context.Context, model string) error {
	req := &api.DeleteRequest{
		Model: model,
	}
	return m.client.client.Delete(ctx, req)
}

// GetModelInfo returns information about a model
func (m *OllamaManager) GetModelInfo(ctx context.Context, model string) (*api.ShowResponse, error) {
	req := &api.ShowRequest{
		Model: model,
	}
	return m.client.client.Show(ctx, req)
}

// Stop stops the Ollama server if it was started by this manager
func (m *OllamaManager) Stop() error {
	if m.process != nil {
		return m.process.Kill()
	}
	return nil
}

// Client returns the underlying Ollama client
func (m *OllamaManager) Client() *OllamaClient {
	return m.client
}

// IsRunning checks if Ollama server is running
func (m *OllamaManager) IsRunning(ctx context.Context) bool {
	return m.client.Ping(ctx) == nil
}

// normalizeModelName removes the :latest tag if present
func normalizeModelName(name string) string {
	name = strings.TrimSuffix(name, ":latest")
	if !strings.Contains(name, ":") {
		return name
	}
	return name
}

// RecommendedModels returns a list of recommended models for different use cases
var RecommendedModels = map[string]string{
	"chat":      "llama3.2",           // General chat
	"code":      "codellama",          // Code generation
	"small":     "llama3.2:1b",        // Fast, small
	"embedding": "nomic-embed-text",   // Embeddings
	"instruct":  "llama3.2:3b-instruct-q4_K_M", // Instruction following
}

// GetRecommendedModel returns the recommended model for a use case
func GetRecommendedModel(useCase string) string {
	if model, ok := RecommendedModels[useCase]; ok {
		return model
	}
	return RecommendedModels["chat"]
}
