package tools

import (
	"context"
	"errors"
	"testing"
)

// mockSubagentSpawner implements SubagentSpawner for testing
type mockSubagentSpawner struct {
	spawnFunc func(ctx context.Context, config SubagentConfig) (*SubagentResult, error)
}

func (m *mockSubagentSpawner) Spawn(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
	if m.spawnFunc != nil {
		return m.spawnFunc(ctx, config)
	}
	return &SubagentResult{
		Success:    true,
		Summary:    "Task completed successfully",
		TokensUsed: 1000,
	}, nil
}

func TestNewSpawnSubagentTool(t *testing.T) {
	spawner := &mockSubagentSpawner{}
	tool := NewSpawnSubagentTool(spawner)

	if tool == nil {
		t.Fatal("NewSpawnSubagentTool returned nil")
	}

	if tool.spawner != spawner {
		t.Error("tool should reference the spawner")
	}
}

func TestSpawnSubagentTool_Name(t *testing.T) {
	tool := NewSpawnSubagentTool(&mockSubagentSpawner{})

	if tool.Name() != "spawn_subagent" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "spawn_subagent")
	}
}

func TestSpawnSubagentTool_Description(t *testing.T) {
	tool := NewSpawnSubagentTool(&mockSubagentSpawner{})

	desc := tool.Description()

	if desc == "" {
		t.Error("Description() should not be empty")
	}

	// Should mention subagent types
	types := []string{"explorer", "coder", "reviewer", "planner", "research"}
	for _, st := range types {
		if !containsString(desc, st) {
			t.Errorf("Description() should mention %q subagent type", st)
		}
	}
}

func TestSpawnSubagentTool_Parameters(t *testing.T) {
	tool := NewSpawnSubagentTool(&mockSubagentSpawner{})

	params := tool.Parameters()

	if len(params) < 2 {
		t.Error("should have at least 2 parameters (type and task)")
	}

	// Check for required parameters
	hasType := false
	hasTask := false
	hasTokenBudget := false

	for _, p := range params {
		switch p.Name {
		case "type":
			hasType = true
			if !p.Required {
				t.Error("type parameter should be required")
			}
		case "task":
			hasTask = true
			if !p.Required {
				t.Error("task parameter should be required")
			}
		case "token_budget":
			hasTokenBudget = true
			if p.Required {
				t.Error("token_budget parameter should not be required")
			}
		}
	}

	if !hasType {
		t.Error("should have 'type' parameter")
	}
	if !hasTask {
		t.Error("should have 'task' parameter")
	}
	if !hasTokenBudget {
		t.Error("should have 'token_budget' parameter")
	}
}

func TestSpawnSubagentTool_Execute_Success(t *testing.T) {
	spawner := &mockSubagentSpawner{
		spawnFunc: func(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
			return &SubagentResult{
				Success:       true,
				Summary:       "Found 5 files",
				FilesModified: []string{"file1.go", "file2.go"},
				TokensUsed:    1500,
			}, nil
		},
	}
	tool := NewSpawnSubagentTool(spawner)

	args := map[string]any{
		"type": "explorer",
		"task": "Find all Go files",
	}

	result, err := tool.Execute(context.Background(), args)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("result.Success should be true")
	}

	if result.Output == "" {
		t.Error("result.Output should not be empty")
	}

	if !containsString(result.Output, "Found 5 files") {
		t.Error("output should contain summary")
	}

	if !containsString(result.Output, "file1.go") {
		t.Error("output should contain modified files")
	}
}

func TestSpawnSubagentTool_Execute_Failure(t *testing.T) {
	spawner := &mockSubagentSpawner{
		spawnFunc: func(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
			return &SubagentResult{
				Success:    false,
				Summary:    "Task failed",
				TokensUsed: 500,
				Error:      errors.New("test error"),
			}, nil
		},
	}
	tool := NewSpawnSubagentTool(spawner)

	args := map[string]any{
		"type": "coder",
		"task": "Write code",
	}

	result, err := tool.Execute(context.Background(), args)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("result.Success should be false for failed subagent")
	}

	if !containsString(result.Output, "Failed") {
		t.Error("output should indicate failure")
	}
}

func TestSpawnSubagentTool_Execute_SpawnError(t *testing.T) {
	spawner := &mockSubagentSpawner{
		spawnFunc: func(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
			return nil, errors.New("spawn error")
		},
	}
	tool := NewSpawnSubagentTool(spawner)

	args := map[string]any{
		"type": "explorer",
		"task": "Find files",
	}

	result, err := tool.Execute(context.Background(), args)

	if err != nil {
		t.Fatalf("Execute() should not return error, got %v", err)
	}

	if result.Success {
		t.Error("result.Success should be false for spawn error")
	}

	// Error message format: "subagent failed: spawn error"
	if !containsString(result.Error, "subagent failed") {
		t.Errorf("Error should indicate subagent failed, got: %q", result.Error)
	}
}

func TestSpawnSubagentTool_Execute_InvalidType(t *testing.T) {
	tool := NewSpawnSubagentTool(&mockSubagentSpawner{})

	args := map[string]any{
		"type": "invalid_type",
		"task": "Do something",
	}

	result, err := tool.Execute(context.Background(), args)

	if err != nil {
		t.Fatalf("Execute() should not return error, got %v", err)
	}

	if result.Success {
		t.Error("result.Success should be false for invalid type")
	}

	// Error message contains "invalid subagent type"
	if !containsString(result.Error, "invalid subagent type") {
		t.Errorf("Error should indicate invalid type, got: %q", result.Error)
	}
}

func TestSpawnSubagentTool_Execute_MissingType(t *testing.T) {
	tool := NewSpawnSubagentTool(&mockSubagentSpawner{})

	args := map[string]any{
		"task": "Do something",
	}

	result, err := tool.Execute(context.Background(), args)

	if err != nil {
		t.Fatalf("Execute() should not return error, got %v", err)
	}

	if result.Success {
		t.Error("result.Success should be false for missing type")
	}
}

func TestSpawnSubagentTool_Execute_MissingTask(t *testing.T) {
	tool := NewSpawnSubagentTool(&mockSubagentSpawner{})

	args := map[string]any{
		"type": "explorer",
	}

	result, err := tool.Execute(context.Background(), args)

	if err != nil {
		t.Fatalf("Execute() should not return error, got %v", err)
	}

	if result.Success {
		t.Error("result.Success should be false for missing task")
	}
}

func TestSpawnSubagentTool_Execute_NilSpawner(t *testing.T) {
	tool := NewSpawnSubagentTool(nil)

	args := map[string]any{
		"type": "explorer",
		"task": "Find files",
	}

	result, err := tool.Execute(context.Background(), args)

	if err != nil {
		t.Fatalf("Execute() should not return error, got %v", err)
	}

	if result.Success {
		t.Error("result.Success should be false for nil spawner")
	}

	// Error message: "subagent spawner not configured"
	if !containsString(result.Error, "spawner not configured") {
		t.Errorf("Error should indicate spawner not configured, got: %q", result.Error)
	}
}

func TestSpawnSubagentTool_Execute_WithTokenBudget(t *testing.T) {
	var receivedConfig SubagentConfig
	spawner := &mockSubagentSpawner{
		spawnFunc: func(ctx context.Context, config SubagentConfig) (*SubagentResult, error) {
			receivedConfig = config
			return &SubagentResult{Success: true, Summary: "Done"}, nil
		},
	}
	tool := NewSpawnSubagentTool(spawner)

	args := map[string]any{
		"type":         "coder",
		"task":         "Write code",
		"token_budget": 25000.0, // JSON numbers are float64
	}

	_, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if receivedConfig.TokenBudget != 25000 {
		t.Errorf("TokenBudget = %d, want 25000", receivedConfig.TokenBudget)
	}
}

func TestSpawnSubagentTool_Execute_AllTypes(t *testing.T) {
	spawner := &mockSubagentSpawner{}
	tool := NewSpawnSubagentTool(spawner)

	types := []string{"explorer", "coder", "reviewer", "planner", "research"}

	for _, st := range types {
		t.Run(st, func(t *testing.T) {
			args := map[string]any{
				"type": st,
				"task": "Test task for " + st,
			}

			result, err := tool.Execute(context.Background(), args)

			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if !result.Success {
				t.Errorf("Execute() for type %s should succeed", st)
			}
		})
	}
}

func TestSpawnSubagentTool_RequiredPermission(t *testing.T) {
	tool := NewSpawnSubagentTool(&mockSubagentSpawner{})

	// Should not require special permissions (subagent tools handle their own)
	permission := tool.RequiredPermission()
	if permission == "" {
		t.Error("RequiredPermission() should return a value")
	}
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		success  bool
		expected string
	}{
		{true, "Success"},
		{false, "Failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := statusString(tt.success)
			if result != tt.expected {
				t.Errorf("statusString(%v) = %q, want %q", tt.success, result, tt.expected)
			}
		})
	}
}

func TestSubagentConfig_Fields(t *testing.T) {
	config := SubagentConfig{
		Type:         "explorer",
		Task:         "Find files",
		TokenBudget:  15000,
		AllowedTools: []string{"read_file"},
		SystemPrompt: "Custom prompt",
		MaxTurns:     10,
		Temperature:  0.5,
	}

	if config.Type != "explorer" {
		t.Errorf("Type = %q, want %q", config.Type, "explorer")
	}
	if config.Task != "Find files" {
		t.Errorf("Task = %q, want %q", config.Task, "Find files")
	}
	if config.TokenBudget != 15000 {
		t.Errorf("TokenBudget = %d, want 15000", config.TokenBudget)
	}
	if len(config.AllowedTools) != 1 {
		t.Errorf("AllowedTools length = %d, want 1", len(config.AllowedTools))
	}
	if config.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, want 10", config.MaxTurns)
	}
	if config.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want 0.5", config.Temperature)
	}
}

func TestSubagentResult_Fields(t *testing.T) {
	result := SubagentResult{
		Success:       true,
		Summary:       "Done",
		Details:       "Full details",
		FilesModified: []string{"a.go", "b.go"},
		TokensUsed:    2000,
		Error:         nil,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Summary != "Done" {
		t.Errorf("Summary = %q, want %q", result.Summary, "Done")
	}
	if result.Details != "Full details" {
		t.Errorf("Details = %q, want %q", result.Details, "Full details")
	}
	if len(result.FilesModified) != 2 {
		t.Errorf("FilesModified length = %d, want 2", len(result.FilesModified))
	}
	if result.TokensUsed != 2000 {
		t.Errorf("TokensUsed = %d, want 2000", result.TokensUsed)
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
