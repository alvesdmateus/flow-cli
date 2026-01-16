package app

import (
	"context"
	"fmt"

	"github.com/mateus/vibe-cli/internal/config"
	"github.com/mateus/vibe-cli/internal/llm"
	"github.com/mateus/vibe-cli/internal/sandbox"
	"github.com/mateus/vibe-cli/internal/search"
	"github.com/mateus/vibe-cli/internal/ui"
)

// App is the main application container that holds all dependencies
type App struct {
	Config      *config.Config
	LLMClient   llm.Client
	SearchClient search.Client
	Permissions *sandbox.Manager
}

// New creates a new App instance with all dependencies initialized
func New(ctx context.Context) (*App, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create LLM client
	llmClient, err := llm.NewClient(cfg.LLM.Provider, cfg.LLM.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Create search client (optional, may fail if not configured)
	var searchClient search.Client
	if cfg.Search.Enabled {
		searchClient, _ = search.NewClient(cfg.Search.Provider, cfg.Search.Endpoint)
	}

	// Create security policy from config
	policy := &sandbox.Policy{
		ProjectDir:      cfg.Security.ProjectDir,
		TrustedPaths:    cfg.Security.TrustedPaths,
		DeniedPaths:     cfg.Security.DeniedPaths,
		AllowedCommands: cfg.Commands.Allowed,
		BlockedCommands: cfg.Commands.Blocked,
		AutoApprove:     cfg.Security.AutoApprove,
		AllowNetwork:    cfg.Search.Enabled,
	}

	// Create permission manager with UI approval function
	permissions, err := sandbox.NewManager(policy, ui.RequestApproval)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission manager: %w", err)
	}

	return &App{
		Config:       cfg,
		LLMClient:    llmClient,
		SearchClient: searchClient,
		Permissions:  permissions,
	}, nil
}

// CheckLLMConnection verifies the LLM service is available
func (a *App) CheckLLMConnection(ctx context.Context) error {
	return a.LLMClient.Ping(ctx)
}

// CheckSearchConnection verifies the search service is available
func (a *App) CheckSearchConnection(ctx context.Context) error {
	if a.SearchClient == nil {
		return fmt.Errorf("search client not configured")
	}
	return a.SearchClient.Ping(ctx)
}

// SetAutoApprove enables or disables auto-approve mode
func (a *App) SetAutoApprove(enabled bool) {
	a.Permissions.SetAutoApprove(enabled)
}

// RequestFileRead requests approval to read a file
func (a *App) RequestFileRead(ctx context.Context, path string) error {
	op := sandbox.NewOperation(sandbox.OpReadFile, path, fmt.Sprintf("Read file: %s", path))
	return a.Permissions.CheckAndApprove(ctx, op)
}

// RequestFileWrite requests approval to write a file
func (a *App) RequestFileWrite(ctx context.Context, path, content string) error {
	op := sandbox.NewOperation(sandbox.OpWriteFile, path, fmt.Sprintf("Write file: %s", path)).
		WithDetail("size", fmt.Sprintf("%d bytes", len(content)))
	return a.Permissions.CheckAndApprove(ctx, op)
}

// RequestCommand requests approval to execute a command
func (a *App) RequestCommand(ctx context.Context, command string) error {
	op := sandbox.NewOperation(sandbox.OpExecuteCmd, command, fmt.Sprintf("Execute command: %s", command))
	return a.Permissions.CheckAndApprove(ctx, op)
}

// RequestWebSearch requests approval to perform a web search
func (a *App) RequestWebSearch(ctx context.Context, query string) error {
	op := sandbox.NewOperation(sandbox.OpWebSearch, query, fmt.Sprintf("Web search: %s", query))
	return a.Permissions.CheckAndApprove(ctx, op)
}
