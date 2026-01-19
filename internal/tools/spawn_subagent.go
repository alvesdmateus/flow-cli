package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mateus/flow-cli/internal/sandbox"
)

// SubagentResult holds the result of a subagent execution
// This mirrors agent.SubagentResult to avoid circular imports
type SubagentResult struct {
	Success       bool     // Whether the subagent completed successfully
	Summary       string   // Concise summary for parent context
	Details       string   // Full details (not added to parent context)
	FilesModified []string // List of files changed
	TokensUsed    int      // Actual tokens consumed
	Error         error    // Error if failed
}

// SubagentConfig holds configuration for spawning a subagent
// This mirrors agent.SubagentConfig to avoid circular imports
type SubagentConfig struct {
	Type         string   // Type of subagent: explorer, coder, reviewer, planner, research
	Task         string   // Task description for the subagent
	TokenBudget  int      // Max tokens for this subagent (0 = use type default)
	AllowedTools []string // Tool whitelist (nil = use type defaults)
	SystemPrompt string   // Custom system prompt (empty = use type default)
	MaxTurns     int      // Max agentic turns (0 = default based on type)
	Temperature  float64  // LLM temperature (0 = use default 0.7)
}

// SubagentSpawner defines the interface for spawning subagents
// This allows the tools package to use subagents without importing the agent package
type SubagentSpawner interface {
	// Spawn creates and runs a subagent with the given configuration
	Spawn(ctx context.Context, config SubagentConfig) (*SubagentResult, error)
}

// SpawnSubagentTool allows the agent to spawn specialized subagents
type SpawnSubagentTool struct {
	spawner SubagentSpawner
}

// NewSpawnSubagentTool creates a new spawn_subagent tool
func NewSpawnSubagentTool(spawner SubagentSpawner) *SpawnSubagentTool {
	return &SpawnSubagentTool{
		spawner: spawner,
	}
}

// Name returns the tool name
func (t *SpawnSubagentTool) Name() string {
	return "spawn_subagent"
}

// Description returns the tool description
func (t *SpawnSubagentTool) Description() string {
	return `Spawn a specialized subagent to handle a specific task with isolated context.
Subagents have their own conversation history and don't pollute the main context.

Available subagent types:
- explorer: For codebase exploration (read-only). Use for finding files, understanding structure.
- coder: For code generation and editing. Use for implementing changes.
- reviewer: For code review and analysis. Use for finding bugs and improvements.
- planner: For architecture planning. Use for designing solutions.
- research: For web search and documentation. Use for finding external information.

The subagent will execute the task and return a concise summary of its findings/actions.`
}

// Parameters returns the tool parameters
func (t *SpawnSubagentTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "type",
			Type:        TypeString,
			Description: "Subagent type: explorer, coder, reviewer, planner, or research",
			Required:    true,
		},
		{
			Name:        "task",
			Type:        TypeString,
			Description: "Detailed task description for the subagent",
			Required:    true,
		},
		{
			Name:        "token_budget",
			Type:        TypeNumber,
			Description: "Maximum tokens for the subagent (default varies by type: 10000-30000)",
			Required:    false,
		},
	}
}

// Execute runs the subagent
func (t *SpawnSubagentTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	// Parse arguments
	subagentType, err := RequiredStringArg(args, "type")
	if err != nil {
		return NewErrorResult(err), nil
	}

	task, err := RequiredStringArg(args, "task")
	if err != nil {
		return NewErrorResult(err), nil
	}

	tokenBudget := GetIntArg(args, "token_budget", 0)

	// Validate subagent type
	validTypes := map[string]bool{
		"explorer": true,
		"coder":    true,
		"reviewer": true,
		"planner":  true,
		"research": true,
	}
	if !validTypes[subagentType] {
		return NewErrorResult(fmt.Errorf(
			"invalid subagent type '%s', must be one of: explorer, coder, reviewer, planner, research",
			subagentType,
		)), nil
	}

	// Check if spawner is available
	if t.spawner == nil {
		return NewErrorResult(fmt.Errorf("subagent spawner not configured")), nil
	}

	// Create subagent configuration
	config := SubagentConfig{
		Type:        subagentType,
		Task:        task,
		TokenBudget: tokenBudget,
	}

	// Spawn and run subagent
	result, err := t.spawner.Spawn(ctx, config)
	if err != nil {
		return NewErrorResult(fmt.Errorf("subagent failed: %w", err)), nil
	}

	// Format the result
	var output strings.Builder
	output.WriteString(fmt.Sprintf("=== Subagent Result (%s) ===\n", subagentType))
	output.WriteString(fmt.Sprintf("Status: %s\n", statusString(result.Success)))
	output.WriteString(fmt.Sprintf("Tokens used: %d\n\n", result.TokensUsed))
	output.WriteString("Summary:\n")
	output.WriteString(result.Summary)

	if len(result.FilesModified) > 0 {
		output.WriteString("\n\nFiles modified:\n")
		for _, f := range result.FilesModified {
			output.WriteString(fmt.Sprintf("- %s\n", f))
		}
	}

	if result.Error != nil {
		output.WriteString(fmt.Sprintf("\n\nError: %v", result.Error))
	}

	return &Result{
		Success: result.Success,
		Output:  output.String(),
		Data: map[string]any{
			"success":        result.Success,
			"summary":        result.Summary,
			"tokens_used":    result.TokensUsed,
			"files_modified": result.FilesModified,
		},
	}, nil
}

// RequiredPermission returns the required permission for this tool
func (t *SpawnSubagentTool) RequiredPermission() sandbox.OperationType {
	// Subagent tool itself doesn't need special permissions
	// The subagent's tools will handle their own permissions
	return sandbox.OpReadFile
}

// statusString converts a boolean to a status string
func statusString(success bool) string {
	if success {
		return "Success"
	}
	return "Failed"
}
