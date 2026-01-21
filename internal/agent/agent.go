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

func (h *DefaultHandler) OnStreamStart()             {}
func (h *DefaultHandler) OnStreamChunk(chunk string) { fmt.Fprint(h.Writer, chunk) }
func (h *DefaultHandler) OnStreamEnd()               { fmt.Fprintln(h.Writer) }
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
	ctxManager  *flowContext.Manager
	model       string
	temperature float64
	maxTurns    int          // Maximum tool execution turns per request
	budget      *TokenBudget // Token budget for this agent and its subagents
}

// Config holds agent configuration
type Config struct {
	LLMClient    llm.Client
	ToolReg      *tools.Registry
	Model        string
	Temperature  float64
	MaxTurns     int
	SystemPrompt string
	TokenBudget  int // Total token budget for agent and subagents (0 = 100000)
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
	if cfg.TokenBudget <= 0 {
		cfg.TokenBudget = 100000 // Default 100k tokens
	}

	ctxManager := flowContext.NewManager(cfg.SystemPrompt, 100)
	ctxManager.SetModel(cfg.Model)

	return &Agent{
		llmClient:   cfg.LLMClient,
		toolReg:     cfg.ToolReg,
		ctxManager:  ctxManager,
		model:       cfg.Model,
		temperature: cfg.Temperature,
		maxTurns:    cfg.MaxTurns,
		budget:      NewTokenBudget(cfg.TokenBudget),
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
		var executedCalls []flowContext.ToolCall
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

			execCall := flowContext.ToolCall{
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

	// Pattern 1: XML-style tool calls (preferred format)
	for _, toolName := range a.toolReg.List() {
		startTag := fmt.Sprintf("<tool_call name=\"%s\">", toolName)
		endTag := "</tool_call>"

		// Find all occurrences
		searchStr := response
		for {
			idx := strings.Index(searchStr, startTag)
			if idx < 0 {
				break
			}
			start := idx + len(startTag)
			end := strings.Index(searchStr[start:], endTag)
			if end < 0 {
				break
			}
			argsJSON := searchStr[start : start+end]
			args, err := tools.ParseArgs(argsJSON)
			if err == nil {
				calls = append(calls, ToolCallRequest{
					ID:        fmt.Sprintf("call_%d", len(calls)),
					Name:      toolName,
					Arguments: args,
				})
			}
			searchStr = searchStr[start+end+len(endTag):]
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

	// Pattern 3: Alternative XML format with single quotes
	// <tool_call name='tool_name'>{"arg": "value"}</tool_call>
	for _, toolName := range a.toolReg.List() {
		startTag := fmt.Sprintf("<tool_call name='%s'>", toolName)
		endTag := "</tool_call>"

		searchStr := response
		for {
			idx := strings.Index(searchStr, startTag)
			if idx < 0 {
				break
			}
			start := idx + len(startTag)
			end := strings.Index(searchStr[start:], endTag)
			if end < 0 {
				break
			}
			argsJSON := searchStr[start : start+end]
			args, err := tools.ParseArgs(argsJSON)
			if err == nil {
				// Avoid duplicates
				duplicate := false
				for _, c := range calls {
					if c.Name == toolName && fmt.Sprintf("%v", c.Arguments) == fmt.Sprintf("%v", args) {
						duplicate = true
						break
					}
				}
				if !duplicate {
					calls = append(calls, ToolCallRequest{
						ID:        fmt.Sprintf("call_%d", len(calls)),
						Name:      toolName,
						Arguments: args,
					})
				}
			}
			searchStr = searchStr[start+end+len(endTag):]
		}
	}

	return calls, len(calls) > 0
}

// GetContextManager returns the context manager
func (a *Agent) GetContextManager() *flowContext.Manager {
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

// GetBudget returns the token budget for this agent
func (a *Agent) GetBudget() *TokenBudget {
	return a.budget
}

// SetBudget sets the token budget for this agent
func (a *Agent) SetBudget(budget *TokenBudget) {
	a.budget = budget
}

// SpawnSubagent creates and runs a subagent with the given configuration
func (a *Agent) SpawnSubagent(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
	subagent, err := NewSubagent(a, config, a.budget)
	if err != nil {
		return nil, fmt.Errorf("failed to create subagent: %w", err)
	}

	result, err := subagent.Run(ctx)

	// Release unused budget
	subagent.ReleaseUnusedBudget(a.budget)

	return result, err
}

// GetSubagentManager creates a new subagent manager for this agent
func (a *Agent) GetSubagentManager(maxParallel int) *SubagentManager {
	return &SubagentManager{
		parent:      a,
		budget:      a.budget,
		subagents:   make([]*Subagent, 0),
		maxParallel: maxParallel,
	}
}

// GetToolRegistry returns the tool registry
func (a *Agent) GetToolRegistry() *tools.Registry {
	return a.toolReg
}

// GetLLMClient returns the LLM client
func (a *Agent) GetLLMClient() llm.Client {
	return a.llmClient
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
	return `You are flow-cli, an AI coding assistant running in a terminal.

CRITICAL: You MUST use tools to perform actions. DO NOT just show code - you must EXECUTE tools.

## Tool Usage Format
To use a tool, you MUST format it exactly like this:
<tool_call name="tool_name">{"arg1": "value1", "arg2": "value2"}</tool_call>

## Available Tools
- read_file: Read file contents. Args: {"path": "file/path"}
- write_file: Write content to a file. Args: {"path": "file/path", "content": "file content"}
- list_files: List directory contents. Args: {"path": "directory/path"}
- create_directory: Create a directory. Args: {"path": "directory/path"}
- run_command: Execute a shell command. Args: {"command": "command to run"}
- web_search: Search the web. Args: {"query": "search query"}
- check_port: Check if a port is in use. Args: {"port": 8080}
- kill_process: Kill a process. Args: {"pid": 1234}
- start_process: Start a background process. Args: {"command": "command", "args": ["arg1"]}

## IMPORTANT Rules
1. When the user asks you to CREATE or WRITE a file, you MUST use the write_file tool
2. When the user asks to READ a file, you MUST use the read_file tool
3. When the user asks to RUN a command, you MUST use the run_command tool
4. NEVER just show code without using write_file to actually create the file
5. First explain what you will do, then IMMEDIATELY use the tool

## Example: Creating a file
User: "Create a hello.go file"
Correct response:
I'll create a hello.go file with a basic Go program.

<tool_call name="write_file">{"path": "hello.go", "content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"}</tool_call>

## Example: Reading a file
User: "Show me the contents of main.go"
Correct response:
I'll read the main.go file for you.

<tool_call name="read_file">{"path": "main.go"}</tool_call>

## Example: Running a command
User: "Run the tests"
Correct response:
I'll run the tests.

<tool_call name="run_command">{"command": "go test ./..."}</tool_call>

Remember: Tool calls are REQUIRED for any file operations or commands. Just showing code without a tool call does NOTHING.`
}
