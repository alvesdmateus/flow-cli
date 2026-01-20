package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mateus/flow-cli/internal/sandbox"
)

// ParameterType represents the type of a tool parameter
type ParameterType string

const (
	TypeString  ParameterType = "string"
	TypeNumber  ParameterType = "number"
	TypeBoolean ParameterType = "boolean"
	TypeArray   ParameterType = "array"
	TypeObject  ParameterType = "object"
)

// Parameter describes a tool parameter
type Parameter struct {
	Name        string        `json:"name"`
	Type        ParameterType `json:"type"`
	Description string        `json:"description"`
	Required    bool          `json:"required"`
	Default     any           `json:"default,omitempty"`
}

// Result represents the result of a tool execution
type Result struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// NewSuccessResult creates a successful result
func NewSuccessResult(output string) *Result {
	return &Result{
		Success: true,
		Output:  output,
	}
}

// NewSuccessResultWithData creates a successful result with additional data
func NewSuccessResultWithData(output string, data any) *Result {
	return &Result{
		Success: true,
		Output:  output,
		Data:    data,
	}
}

// NewErrorResult creates an error result
func NewErrorResult(err error) *Result {
	return &Result{
		Success: false,
		Error:   err.Error(),
	}
}

// Tool defines the interface for executable tools
type Tool interface {
	// Name returns the tool name (used in function calls)
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// Parameters returns the list of parameters the tool accepts
	Parameters() []Parameter

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]any) (*Result, error)

	// RequiredPermission returns the operation type for permission checking
	RequiredPermission() sandbox.OperationType
}

// Registry holds all available tools
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool already registered: %s", name)
	}

	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tool names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// All returns all registered tools
func (r *Registry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// FilterByNames creates a new registry containing only the specified tools
func (r *Registry) FilterByNames(names []string) *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := NewRegistry()
	nameSet := make(map[string]bool, len(names))
	for _, name := range names {
		nameSet[name] = true
	}

	for name, tool := range r.tools {
		if nameSet[name] {
			filtered.tools[name] = tool
		}
	}

	return filtered
}

// Clone creates a copy of the registry
func (r *Registry) Clone() *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cloned := NewRegistry()
	for name, tool := range r.tools {
		cloned.tools[name] = tool
	}

	return cloned
}

// Count returns the number of registered tools
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// ToJSONSchema converts a tool to a JSON schema for LLM function calling
func ToJSONSchema(tool Tool) map[string]any {
	properties := make(map[string]any)
	required := make([]string, 0)

	for _, param := range tool.Parameters() {
		properties[param.Name] = map[string]any{
			"type":        string(param.Type),
			"description": param.Description,
		}
		if param.Required {
			required = append(required, param.Name)
		}
	}

	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters": map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}

// RegistryToJSONSchema converts all tools in a registry to JSON schema
func RegistryToJSONSchema(registry *Registry) []map[string]any {
	tools := registry.All()
	schemas := make([]map[string]any, len(tools))
	for i, tool := range tools {
		schemas[i] = ToJSONSchema(tool)
	}
	return schemas
}

// ParseArgs parses JSON arguments into a map
func ParseArgs(argsJSON string) (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	return args, nil
}

// GetStringArg extracts a string argument with a default value
func GetStringArg(args map[string]any, name, defaultVal string) string {
	if val, ok := args[name]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

// GetIntArg extracts an integer argument with a default value
func GetIntArg(args map[string]any, name string, defaultVal int) int {
	if val, ok := args[name]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return defaultVal
}

// GetBoolArg extracts a boolean argument with a default value
func GetBoolArg(args map[string]any, name string, defaultVal bool) bool {
	if val, ok := args[name]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// RequiredStringArg extracts a required string argument
func RequiredStringArg(args map[string]any, name string) (string, error) {
	if val, ok := args[name]; ok {
		if str, ok := val.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("argument %s must be a string", name)
	}
	return "", fmt.Errorf("missing required argument: %s", name)
}
