package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	flowContext "github.com/mateus/flow-cli/internal/context"
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/tools"
)

// Subagent represents an isolated agent spawned for a specific task
type Subagent struct {
	config     SubagentConfig
	agent      *Agent         // Underlying agent with isolated context
	parent     *Agent         // Reference to parent agent
	budget     *TokenBudget   // Allocated budget for this subagent
	allocated  int            // Amount allocated from parent budget
	tokensUsed int            // Actual tokens consumed
	handler    SubagentHandler
}

// SubagentHandler handles subagent events (optional)
type SubagentHandler interface {
	OnSubagentStart(subagentType SubagentType, task string)
	OnSubagentToolCall(toolName string, success bool)
	OnSubagentEnd(result *SubagentResult)
}

// SilentSubagentHandler is a no-op handler for subagents
type SilentSubagentHandler struct{}

func (h *SilentSubagentHandler) OnSubagentStart(SubagentType, string) {}
func (h *SilentSubagentHandler) OnSubagentToolCall(string, bool)      {}
func (h *SilentSubagentHandler) OnSubagentEnd(*SubagentResult)        {}

// SubagentResponseHandler captures subagent output without displaying it
type SubagentResponseHandler struct {
	output     strings.Builder
	toolCalls  []string
	subagent   *Subagent
}

func (h *SubagentResponseHandler) OnStreamStart()             {}
func (h *SubagentResponseHandler) OnStreamChunk(chunk string) { h.output.WriteString(chunk) }
func (h *SubagentResponseHandler) OnStreamEnd()               {}
func (h *SubagentResponseHandler) OnToolStart(name, desc string) {
	h.toolCalls = append(h.toolCalls, name)
}
func (h *SubagentResponseHandler) OnToolEnd(name string, success bool, result string) {
	if h.subagent.handler != nil {
		h.subagent.handler.OnSubagentToolCall(name, success)
	}
}
func (h *SubagentResponseHandler) OnThinking(msg string) {}
func (h *SubagentResponseHandler) OnError(err error)     {}

// NewSubagent creates a new subagent with isolated context
func NewSubagent(parent *Agent, config SubagentConfig, parentBudget *TokenBudget) (*Subagent, error) {
	// Validate config
	if config.Task == "" {
		return nil, fmt.Errorf("subagent task cannot be empty")
	}

	if !IsValidSubagentType(string(config.Type)) {
		return nil, fmt.Errorf("invalid subagent type: %s", config.Type)
	}

	// Apply defaults based on type
	if config.TokenBudget <= 0 {
		config.TokenBudget = DefaultTokenBudgetForType(config.Type)
	}
	if config.MaxTurns <= 0 {
		config.MaxTurns = DefaultMaxTurnsForType(config.Type)
	}
	if len(config.AllowedTools) == 0 {
		config.AllowedTools = DefaultToolsForType(config.Type)
	}
	if config.SystemPrompt == "" {
		config.SystemPrompt = BuildTaskPrompt(config.Type, config.Task)
	}
	if config.Temperature <= 0 {
		config.Temperature = 0.7
	}

	// Allocate budget from parent
	var allocated int
	var subBudget *TokenBudget

	if parentBudget != nil {
		var ok bool
		allocated, ok = parentBudget.Allocate(config.TokenBudget)
		if !ok {
			return nil, fmt.Errorf("insufficient token budget: requested %d, available %d",
				config.TokenBudget, parentBudget.Available())
		}
		subBudget = NewTokenBudget(allocated)
	} else {
		// No parent budget, create standalone budget
		subBudget = NewTokenBudget(config.TokenBudget)
		allocated = config.TokenBudget
	}

	// Create filtered tool registry
	filteredTools := parent.toolReg.FilterByNames(config.AllowedTools)

	// Create isolated context manager
	ctxManager := parent.ctxManager.CloneWithSystemPrompt(config.SystemPrompt)

	// Create underlying agent
	agent := &Agent{
		llmClient:   parent.llmClient,
		toolReg:     filteredTools,
		ctxManager:  ctxManager,
		model:       parent.model,
		temperature: config.Temperature,
		maxTurns:    config.MaxTurns,
	}

	return &Subagent{
		config:    config,
		agent:     agent,
		parent:    parent,
		budget:    subBudget,
		allocated: allocated,
		handler:   &SilentSubagentHandler{},
	}, nil
}

// SetHandler sets the event handler for the subagent
func (s *Subagent) SetHandler(handler SubagentHandler) {
	s.handler = handler
}

// Run executes the subagent's task and returns the result
func (s *Subagent) Run(ctx context.Context) (*SubagentResult, error) {
	if s.handler != nil {
		s.handler.OnSubagentStart(s.config.Type, s.config.Task)
	}

	// Track start tokens
	startTokens := s.agent.ctxManager.GetTokenCount()

	// Create response handler to capture output
	responseHandler := &SubagentResponseHandler{
		subagent: s,
	}

	// Run the agent with the task
	err := s.agent.ProcessMessage(ctx, s.config.Task, responseHandler)

	// Calculate tokens used
	endTokens := s.agent.ctxManager.GetTokenCount()
	s.tokensUsed = endTokens - startTokens

	// Build result
	result := &SubagentResult{
		TokensUsed: s.tokensUsed,
	}

	if err != nil {
		result.Success = false
		result.Error = err
		result.Summary = fmt.Sprintf("Subagent failed: %v", err)
		result.Details = responseHandler.output.String()
	} else {
		result.Success = true
		result.Details = responseHandler.output.String()
		result.Summary = s.generateSummary(responseHandler.output.String())
		result.FilesModified = s.extractModifiedFiles(responseHandler.toolCalls)
	}

	if s.handler != nil {
		s.handler.OnSubagentEnd(result)
	}

	return result, nil
}

// generateSummary creates a concise summary from the subagent's output
func (s *Subagent) generateSummary(fullOutput string) string {
	// If the output is short enough, use it directly
	if len(fullOutput) <= 500 {
		return fullOutput
	}

	// Try to extract a summary from the output
	// Look for summary indicators
	summaryIndicators := []string{
		"Summary:",
		"In summary,",
		"To summarize:",
		"Key findings:",
		"Result:",
	}

	for _, indicator := range summaryIndicators {
		if idx := strings.Index(fullOutput, indicator); idx != -1 {
			// Extract from indicator to end or next section
			summary := fullOutput[idx:]
			if len(summary) > 500 {
				summary = summary[:500] + "..."
			}
			return summary
		}
	}

	// Otherwise truncate intelligently
	// Try to find a good break point
	truncated := fullOutput
	if len(truncated) > 500 {
		truncated = truncated[:500]
		// Find last sentence end
		if lastPeriod := strings.LastIndex(truncated, ". "); lastPeriod > 300 {
			truncated = truncated[:lastPeriod+1]
		} else {
			truncated += "..."
		}
	}

	return truncated
}

// extractModifiedFiles identifies files that were modified by the subagent
func (s *Subagent) extractModifiedFiles(toolCalls []string) []string {
	// This is a simple heuristic - tools that might modify files
	modifyingTools := map[string]bool{
		"write_file":   true,
		"edit_file":    true,
		"insert_lines": true,
		"delete_lines": true,
		"delete_file":  true,
	}

	var modified []string
	for _, tc := range toolCalls {
		if modifyingTools[tc] {
			// Note: We don't have the actual file paths here
			// In a more complete implementation, we'd track this in tool execution
			modified = append(modified, tc)
		}
	}

	return modified
}

// GetTokensUsed returns the number of tokens consumed
func (s *Subagent) GetTokensUsed() int {
	return s.tokensUsed
}

// ReleaseUnusedBudget returns unused tokens to the parent budget
func (s *Subagent) ReleaseUnusedBudget(parentBudget *TokenBudget) {
	if parentBudget == nil {
		return
	}

	unused := s.allocated - s.tokensUsed
	if unused > 0 {
		parentBudget.Release(unused)
	}

	// Mark tokens as consumed in parent budget
	if s.tokensUsed > 0 {
		parentBudget.Consume(s.tokensUsed)
	}
}

// GetConfig returns the subagent configuration
func (s *Subagent) GetConfig() SubagentConfig {
	return s.config
}

// SpawnSubagentHelper is a helper for agents to spawn subagents
type SpawnSubagentHelper struct {
	llmClient llm.Client
	toolReg   *tools.Registry
	budget    *TokenBudget
	model     string
}

// NewSpawnSubagentHelper creates a new helper for spawning subagents
func NewSpawnSubagentHelper(
	llmClient llm.Client,
	toolReg *tools.Registry,
	budget *TokenBudget,
	model string,
) *SpawnSubagentHelper {
	return &SpawnSubagentHelper{
		llmClient: llmClient,
		toolReg:   toolReg,
		budget:    budget,
		model:     model,
	}
}

// Spawn creates and runs a subagent with the given configuration
func (h *SpawnSubagentHelper) Spawn(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
	// Create a minimal parent agent for the subagent
	parent := &Agent{
		llmClient:   h.llmClient,
		toolReg:     h.toolReg,
		ctxManager:  flowContext.NewManager("", 100),
		model:       h.model,
		temperature: 0.7,
		maxTurns:    10,
	}

	subagent, err := NewSubagent(parent, config, h.budget)
	if err != nil {
		return nil, fmt.Errorf("failed to create subagent: %w", err)
	}

	result, err := subagent.Run(ctx)

	// Release unused budget
	subagent.ReleaseUnusedBudget(h.budget)

	return result, err
}

// SpawnFromToolConfig creates and runs a subagent from a tools.SubagentConfig
// This implements the tools.SubagentSpawner interface
func (h *SpawnSubagentHelper) SpawnFromToolConfig(ctx context.Context, config tools.SubagentConfig) (*tools.SubagentResult, error) {
	// Convert tools.SubagentConfig to agent.SubagentConfig
	agentConfig := SubagentConfig{
		Type:         SubagentType(config.Type),
		Task:         config.Task,
		TokenBudget:  config.TokenBudget,
		AllowedTools: config.AllowedTools,
		SystemPrompt: config.SystemPrompt,
		MaxTurns:     config.MaxTurns,
		Temperature:  config.Temperature,
	}

	result, err := h.Spawn(ctx, agentConfig)
	if err != nil {
		return nil, err
	}

	// Convert agent.SubagentResult to tools.SubagentResult
	return &tools.SubagentResult{
		Success:       result.Success,
		Summary:       result.Summary,
		Details:       result.Details,
		FilesModified: result.FilesModified,
		TokensUsed:    result.TokensUsed,
		Error:         result.Error,
	}, nil
}

// SubagentSpawnerAdapter adapts SpawnSubagentHelper to implement tools.SubagentSpawner
type SubagentSpawnerAdapter struct {
	helper *SpawnSubagentHelper
}

// NewSubagentSpawnerAdapter creates a new adapter
func NewSubagentSpawnerAdapter(helper *SpawnSubagentHelper) *SubagentSpawnerAdapter {
	return &SubagentSpawnerAdapter{helper: helper}
}

// Spawn implements tools.SubagentSpawner
func (a *SubagentSpawnerAdapter) Spawn(ctx context.Context, config tools.SubagentConfig) (*tools.SubagentResult, error) {
	return a.helper.SpawnFromToolConfig(ctx, config)
}

// AgentSubagentSpawner wraps an Agent to implement tools.SubagentSpawner
type AgentSubagentSpawner struct {
	agent *Agent
}

// NewAgentSubagentSpawner creates a new spawner from an agent
func NewAgentSubagentSpawner(agent *Agent) *AgentSubagentSpawner {
	return &AgentSubagentSpawner{agent: agent}
}

// Spawn implements tools.SubagentSpawner
func (s *AgentSubagentSpawner) Spawn(ctx context.Context, config tools.SubagentConfig) (*tools.SubagentResult, error) {
	// Convert tools.SubagentConfig to agent.SubagentConfig
	agentConfig := SubagentConfig{
		Type:         SubagentType(config.Type),
		Task:         config.Task,
		TokenBudget:  config.TokenBudget,
		AllowedTools: config.AllowedTools,
		SystemPrompt: config.SystemPrompt,
		MaxTurns:     config.MaxTurns,
		Temperature:  config.Temperature,
	}

	result, err := s.agent.SpawnSubagent(ctx, agentConfig)
	if err != nil {
		return nil, err
	}

	// Convert agent.SubagentResult to tools.SubagentResult
	return &tools.SubagentResult{
		Success:       result.Success,
		Summary:       result.Summary,
		Details:       result.Details,
		FilesModified: result.FilesModified,
		TokensUsed:    result.TokensUsed,
		Error:         result.Error,
	}, nil
}

// SubagentManager manages multiple subagents for a parent agent
type SubagentManager struct {
	parent      *Agent
	budget      *TokenBudget
	subagents   []*Subagent
	maxParallel int
}

// NewSubagentManager creates a new manager for subagents
func NewSubagentManager(parent *Agent, totalBudget int, maxParallel int) *SubagentManager {
	if maxParallel <= 0 {
		maxParallel = 3
	}
	if totalBudget <= 0 {
		totalBudget = 100000 // Default 100k tokens
	}

	return &SubagentManager{
		parent:      parent,
		budget:      NewTokenBudget(totalBudget),
		subagents:   make([]*Subagent, 0),
		maxParallel: maxParallel,
	}
}

// SpawnAndRun creates a subagent and runs it
func (m *SubagentManager) SpawnAndRun(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
	subagent, err := NewSubagent(m.parent, config, m.budget)
	if err != nil {
		return nil, err
	}

	m.subagents = append(m.subagents, subagent)

	result, err := subagent.Run(ctx)

	// Release unused budget
	subagent.ReleaseUnusedBudget(m.budget)

	return result, err
}

// GetBudgetStats returns current budget statistics
func (m *SubagentManager) GetBudgetStats() (total, used, reserved, available int) {
	return m.budget.Stats()
}

// QuickExplore spawns an explorer subagent for a quick exploration task
func (m *SubagentManager) QuickExplore(ctx context.Context, task string) (*SubagentResult, error) {
	return m.SpawnAndRun(ctx, SubagentConfig{
		Type: SubagentExplorer,
		Task: task,
	})
}

// QuickCode spawns a coder subagent for a coding task
func (m *SubagentManager) QuickCode(ctx context.Context, task string) (*SubagentResult, error) {
	return m.SpawnAndRun(ctx, SubagentConfig{
		Type: SubagentCoder,
		Task: task,
	})
}

// QuickReview spawns a reviewer subagent for a code review task
func (m *SubagentManager) QuickReview(ctx context.Context, task string) (*SubagentResult, error) {
	return m.SpawnAndRun(ctx, SubagentConfig{
		Type: SubagentReviewer,
		Task: task,
	})
}

// QuickResearch spawns a research subagent for information gathering
func (m *SubagentManager) QuickResearch(ctx context.Context, task string) (*SubagentResult, error) {
	return m.SpawnAndRun(ctx, SubagentConfig{
		Type: SubagentResearch,
		Task: task,
	})
}

// LoggingSubagentHandler logs subagent events to a writer
type LoggingSubagentHandler struct {
	Writer io.Writer
}

func (h *LoggingSubagentHandler) OnSubagentStart(subagentType SubagentType, task string) {
	fmt.Fprintf(h.Writer, "[Subagent:%s] Starting task: %s\n", subagentType, truncateForLog(task, 50))
}

func (h *LoggingSubagentHandler) OnSubagentToolCall(toolName string, success bool) {
	status := "ok"
	if !success {
		status = "failed"
	}
	fmt.Fprintf(h.Writer, "[Subagent] Tool %s: %s\n", toolName, status)
}

func (h *LoggingSubagentHandler) OnSubagentEnd(result *SubagentResult) {
	status := "completed"
	if !result.Success {
		status = "failed"
	}
	fmt.Fprintf(h.Writer, "[Subagent] %s (tokens used: %d)\n", status, result.TokensUsed)
}

func truncateForLog(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// RunWithTimeout runs the subagent with a timeout
func (s *Subagent) RunWithTimeout(ctx context.Context, timeout time.Duration) (*SubagentResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultCh := make(chan *SubagentResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := s.Run(ctx)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return &SubagentResult{
			Success:    false,
			Error:      ctx.Err(),
			Summary:    "Subagent timed out",
			TokensUsed: s.tokensUsed,
		}, ctx.Err()
	}
}
