package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	vibeContext "github.com/mateus/vibe-cli/internal/context"
	"github.com/mateus/vibe-cli/internal/llm"
	"github.com/mateus/vibe-cli/internal/tools"
)

// ResponseHandler handles streaming responses from the agent
type ResponseHandler interface {
	// OnStreamStart is called when streaming begins
	OnStreamStart()

	// OnStreamChunk is called for each chunk of streamed text
	OnStreamChunk(chunk string)

	// OnStreamEnd is called when streaming ends
	OnStreamEnd()

	// OnToolStart is called when a tool execution begins
	OnToolStart(toolName, description string)

	// OnToolEnd is called when a tool execution ends
	OnToolEnd(toolName string, success bool, result string)

	// OnThinking is called when the agent is processing
	OnThinking(message string)

	// OnError is called when an error occurs
	OnError(err error)
}

// DefaultHandler provides a simple console output handler
type DefaultHandler struct {
	Writer io.Writer
}

func (h *DefaultHandler) OnStreamStart()              {}
func (h *DefaultHandler) OnStreamChunk(chunk string)  { fmt.Fprint(h.Writer, chunk) }
func (h *DefaultHandler) OnStreamEnd()                { fmt.Fprintln(h.Writer) }
func (h *DefaultHandler) OnToolStart(name, desc string) {
	fmt.Fprintf(h.Writer, "\n[Executing: %s] %s\n", name, desc)
}
func (h *DefaultHandler) OnToolEnd(name string, success bool, result string) {
	status := "✓"
	if !success {
		status = "✗"
	}
	fmt.Fprintf(h.Writer, "[%s %s]\n", status, name)
}
func (h *DefaultHandler) OnThinking(msg string) { fmt.Fprintf(h.Writer, "⏳ %s\n", msg) }
func (h *DefaultHandler) OnError(err error)     { fmt.Fprintf(h.Writer, "Error: %v\n", err) }

// Agent orchestrates the conversation between user, LLM, and tools
type Agent struct {
	llmClient   llm.Client
	toolReg     *tools.Registry
	ctxManager  *vibeContext.Manager
	model       string
	temperature float64
	maxTurns    int // Maximum tool execution turns per request
}

// Config holds agent configuration
type Config struct {
	LLMClient   llm.Client
	ToolReg     *tools.Registry
	Model       string
	Temperature float64
	MaxTurns    int
	SystemPrompt string
}

// New creates a new agent
func New(cfg Config) *Agent {
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = 10
	}
	if cfg.Temperature <= 0 {
		cfg.Temperature = 0.7
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = DefaultSystemPrompt()
	}

	ctxManager := vibeContext.NewManager(cfg.SystemPrompt, 100)
	ctxManager.SetModel(cfg.Model)

	return &Agent{
		llmClient:   cfg.LLMClient,
		toolReg:     cfg.ToolReg,
		ctxManager:  ctxManager,
		model:       cfg.Model,
		temperature: cfg.Temperature,
		maxTurns:    cfg.MaxTurns,
	}
}

// ProcessMessage handles a user message and generates a response
func (a *Agent) ProcessMessage(ctx context.Context, userMessage string, handler ResponseHandler) error {
	if handler == nil {
		handler = &DefaultHandler{Writer: io.Discard}
	}

	// Add user message to context
	a.ctxManager.AddUserMessage(userMessage)

	// Agent loop - may involve multiple tool calls
	for turn := 0; turn < a.maxTurns; turn++ {
		// Get response from LLM
		messages := a.ctxManager.GetMessages()

		opts := llm.ChatOptions{
			Model:       a.model,
			Temperature: a.temperature,
			Stream:      true,
		}

		handler.OnStreamStart()

		chunks, err := a.llmClient.Chat(ctx, messages, opts)
		if err != nil {
			handler.OnError(err)
			return fmt.Errorf("LLM chat failed: %w", err)
		}

		// Collect response
		var response strings.Builder
		for chunk := range chunks {
			if chunk.Error != nil {
				handler.OnError(chunk.Error)
				return chunk.Error
			}
			handler.OnStreamChunk(chunk.Content)
			response.WriteString(chunk.Content)
		}

		handler.OnStreamEnd()

		responseText := response.String()

		// Check if response contains tool calls
		toolCalls, hasTools := a.parseToolCalls(responseText)

		if !hasTools {
			// No tool calls, we're done
			a.ctxManager.AddAssistantMessage(responseText, nil)
			return nil
		}

		// Execute tool calls
		var executedCalls []vibeContext.ToolCall
		var toolResults strings.Builder
		toolResults.WriteString("\n\nTool Results:\n")

		for _, tc := range toolCalls {
			tool, exists := a.toolReg.Get(tc.Name)
			if !exists {
				handler.OnToolEnd(tc.Name, false, "Tool not found")
				toolResults.WriteString(fmt.Sprintf("- %s: Error - tool not found\n", tc.Name))
				continue
			}

			handler.OnToolStart(tc.Name, tool.Description())

			startTime := time.Now()
			result, err := tool.Execute(ctx, tc.Arguments)
			duration := time.Since(startTime)

			execCall := vibeContext.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
				Duration:  duration,
			}

			if err != nil || (result != nil && !result.Success) {
				errMsg := ""
				if err != nil {
					errMsg = err.Error()
				} else if result != nil {
					errMsg = result.Error
				}
				handler.OnToolEnd(tc.Name, false, errMsg)
				toolResults.WriteString(fmt.Sprintf("- %s: Error - %s\n", tc.Name, errMsg))
				execCall.Success = false
				execCall.Result = errMsg
			} else {
				handler.OnToolEnd(tc.Name, true, result.Output)
				toolResults.WriteString(fmt.Sprintf("- %s: %s\n", tc.Name, truncateResult(result.Output, 500)))
				execCall.Success = true
				execCall.Result = result.Output
			}

			executedCalls = append(executedCalls, execCall)
		}

		// Add assistant message with tool calls
		a.ctxManager.AddAssistantMessage(responseText, executedCalls)

		// Add tool results as user message for next turn
		a.ctxManager.AddUserMessage(toolResults.String())
	}

	return fmt.Errorf("max tool execution turns (%d) exceeded", a.maxTurns)
}

// ToolCallRequest represents a parsed tool call from LLM response
type ToolCallRequest struct {
	ID        string
	Name      string
	Arguments map[string]any
}

// parseToolCalls extracts tool calls from the response
// This is a simple implementation - in production, use proper function calling API
func (a *Agent) parseToolCalls(response string) ([]ToolCallRequest, bool) {
	// Look for tool call patterns in the response
	// Format: <tool_call name="tool_name">{"arg": "value"}</tool_call>
	// Or: ```tool:tool_name\n{"arg": "value"}\n```

	var calls []ToolCallRequest

	// Pattern 1: XML-style tool calls
	for _, toolName := range a.toolReg.List() {
		startTag := fmt.Sprintf("<tool_call name=\"%s\">", toolName)
		endTag := "</tool_call>"

		idx := strings.Index(response, startTag)
		if idx >= 0 {
			start := idx + len(startTag)
			end := strings.Index(response[start:], endTag)
			if end >= 0 {
				argsJSON := response[start : start+end]
				args, err := tools.ParseArgs(argsJSON)
				if err == nil {
					calls = append(calls, ToolCallRequest{
						ID:        fmt.Sprintf("call_%d", len(calls)),
						Name:      toolName,
						Arguments: args,
					})
				}
			}
		}
	}

	// Pattern 2: Code block style
	// ```tool:read_file
	// {"path": "/some/file"}
	// ```
	lines := strings.Split(response, "\n")
	inToolBlock := false
	var currentTool string
	var toolContent strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "```tool:") {
			inToolBlock = true
			currentTool = strings.TrimPrefix(line, "```tool:")
			currentTool = strings.TrimSpace(currentTool)
			toolContent.Reset()
		} else if inToolBlock && strings.HasPrefix(line, "```") {
			// End of tool block
			inToolBlock = false
			if currentTool != "" {
				args, err := tools.ParseArgs(toolContent.String())
				if err == nil {
					calls = append(calls, ToolCallRequest{
						ID:        fmt.Sprintf("call_%d", len(calls)),
						Name:      currentTool,
						Arguments: args,
					})
				}
			}
		} else if inToolBlock {
			toolContent.WriteString(line)
			toolContent.WriteString("\n")
		}
	}

	return calls, len(calls) > 0
}

// GetContextManager returns the context manager
func (a *Agent) GetContextManager() *vibeContext.Manager {
	return a.ctxManager
}

// SetModel updates the model
func (a *Agent) SetModel(model string) {
	a.model = model
	a.ctxManager.SetModel(model)
}

// Clear resets the conversation
func (a *Agent) Clear() {
	a.ctxManager.Clear()
}

// truncateResult limits result length
func truncateResult(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// DefaultSystemPrompt returns the default system prompt
func DefaultSystemPrompt() string {
	return `You are vibe-cli, an AI coding assistant running in a terminal.

Your capabilities:
- Read and write files in the project directory
- Execute shell commands
- Search the web for information
- Manage processes (check ports, start/stop)

Guidelines:
1. Be concise and direct in your responses
2. When you need to perform an action, use the appropriate tool
3. Always explain what you're about to do before doing it
4. If you're unsure, ask clarifying questions
5. Format code in markdown code blocks with language specifiers

To use a tool, format your response like this:
<tool_call name="tool_name">{"arg1": "value1", "arg2": "value2"}</tool_call>

Available tools:
- read_file: Read file contents
- write_file: Write content to a file
- list_files: List directory contents
- create_directory: Create a directory
- run_command: Execute a shell command
- web_search: Search the web
- check_port: Check if a port is in use
- kill_process: Kill a process
- start_process: Start a background process

Remember: Always get user approval before making changes.`
}
