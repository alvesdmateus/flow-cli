package llm

import (
	"context"
	"fmt"
)

// SetupOptions configures the LLM setup process
type SetupOptions struct {
	Endpoint    string
	Model       string
	AutoStart   bool
	AutoPull    bool
	ProgressFn  func(status string, completed, total int64)
	MessageFn   func(msg string)
}

// SetupResult contains the result of the setup process
type SetupResult struct {
	Client       Client
	Manager      *OllamaManager
	ModelPulled  bool
	ServerStarted bool
}

// Setup ensures the LLM backend is ready for use
// It handles auto-starting Ollama and auto-pulling models
func Setup(ctx context.Context, opts SetupOptions) (*SetupResult, error) {
	result := &SetupResult{}

	// Set defaults
	if opts.Endpoint == "" {
		opts.Endpoint = "http://localhost:11434"
	}

	// Create manager
	manager, err := NewOllamaManager(opts.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama manager: %w", err)
	}
	result.Manager = manager

	// Check if Ollama is running
	if !manager.IsRunning(ctx) {
		if opts.AutoStart {
			if opts.MessageFn != nil {
				opts.MessageFn("Starting Ollama server...")
			}
			if err := manager.EnsureRunning(ctx); err != nil {
				return nil, fmt.Errorf("failed to start Ollama: %w", err)
			}
			result.ServerStarted = true
			if opts.MessageFn != nil {
				opts.MessageFn("Ollama server started")
			}
		} else {
			return nil, fmt.Errorf("Ollama is not running. Start it with 'ollama serve' or enable auto-start")
		}
	}

	// Check if model is available
	if opts.Model != "" && opts.AutoPull {
		hasModel, err := manager.HasModel(ctx, opts.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to check model: %w", err)
		}

		if !hasModel {
			if opts.MessageFn != nil {
				opts.MessageFn(fmt.Sprintf("Pulling model '%s'...", opts.Model))
			}
			if err := manager.EnsureModel(ctx, opts.Model, opts.ProgressFn); err != nil {
				return nil, fmt.Errorf("failed to pull model '%s': %w", opts.Model, err)
			}
			result.ModelPulled = true
			if opts.MessageFn != nil {
				opts.MessageFn(fmt.Sprintf("Model '%s' ready", opts.Model))
			}
		}
	}

	result.Client = manager.Client()
	return result, nil
}

// QuickSetup is a simplified setup for common use cases
func QuickSetup(ctx context.Context, endpoint, model string) (Client, error) {
	result, err := Setup(ctx, SetupOptions{
		Endpoint:  endpoint,
		Model:     model,
		AutoStart: true,
		AutoPull:  true,
	})
	if err != nil {
		return nil, err
	}
	return result.Client, nil
}
