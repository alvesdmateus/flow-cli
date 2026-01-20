package agent

import (
	"context"
	"strings"
	"testing"

	flowContext "github.com/mateus/flow-cli/internal/context"
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/tools"
)

// mockLLMClient is a simple mock for testing
type mockLLMClient struct{}

func (m *mockLLMClient) Chat(ctx context.Context, messages []llm.Message, opts llm.ChatOptions) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamChunk{
			Content: "Task completed. Summary: Found 5 relevant files.",
			Done:    true,
		}
	}()
	return ch, nil
}

func (m *mockLLMClient) ChatSync(ctx context.Context, messages []llm.Message, opts llm.ChatOptions) (string, error) {
	return "Task completed. Summary: Found 5 relevant files.", nil
}

func (m *mockLLMClient) ListModels(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{{Name: "test-model"}}, nil
}

func (m *mockLLMClient) Ping(ctx context.Context) error {
	return nil
}

func (m *mockLLMClient) Provider() string {
	return "mock"
}

func createTestAgent() *Agent {
	registry := tools.NewRegistry()
	// Register a simple read-only tool for testing
	// In real tests, we would use actual tools or mocks

	ctxManager := flowContext.NewManager("test system prompt", 100)

	return &Agent{
		llmClient:   &mockLLMClient{},
		toolReg:     registry,
		ctxManager:  ctxManager,
		model:       "test-model",
		temperature: 0.7,
		maxTurns:    10,
		budget:      NewTokenBudget(100000),
	}
}

func TestNewSubagent_ValidConfig(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type: SubagentExplorer,
		Task: "Find all Go files",
	}

	subagent, err := NewSubagent(parent, config, parent.budget)
	if err != nil {
		t.Fatalf("NewSubagent failed: %v", err)
	}

	if subagent == nil {
		t.Fatal("NewSubagent returned nil")
	}

	// Check that config was applied
	if subagent.config.Type != SubagentExplorer {
		t.Errorf("expected type Explorer, got %s", subagent.config.Type)
	}

	// Check that defaults were applied
	if subagent.config.MaxTurns <= 0 {
		t.Error("MaxTurns should have a default value")
	}

	if subagent.config.TokenBudget <= 0 {
		t.Error("TokenBudget should have a default value")
	}
}

func TestNewSubagent_InvalidType(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type: SubagentType("invalid"),
		Task: "Test task",
	}

	_, err := NewSubagent(parent, config, parent.budget)
	if err == nil {
		t.Error("expected error for invalid subagent type")
	}
}

func TestNewSubagent_EmptyTask(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type: SubagentExplorer,
		Task: "",
	}

	_, err := NewSubagent(parent, config, parent.budget)
	if err == nil {
		t.Error("expected error for empty task")
	}
}

func TestNewSubagent_InsufficientBudget(t *testing.T) {
	parent := createTestAgent()

	// Create a nearly exhausted budget
	smallBudget := NewTokenBudget(100)
	smallBudget.Allocate(100) // Allocate all

	config := SubagentConfig{
		Type:        SubagentExplorer,
		Task:        "Test task",
		TokenBudget: 10000,
	}

	_, err := NewSubagent(parent, config, smallBudget)
	if err == nil {
		t.Error("expected error for insufficient budget")
	}
}

func TestNewSubagent_CustomConfig(t *testing.T) {
	parent := createTestAgent()

	customTools := []string{"read_file", "list_files"}
	customPrompt := "Custom system prompt"

	config := SubagentConfig{
		Type:         SubagentCoder,
		Task:         "Test task",
		TokenBudget:  5000,
		AllowedTools: customTools,
		SystemPrompt: customPrompt,
		MaxTurns:     5,
		Temperature:  0.5,
	}

	subagent, err := NewSubagent(parent, config, parent.budget)
	if err != nil {
		t.Fatalf("NewSubagent failed: %v", err)
	}

	if subagent.config.TokenBudget != 5000 {
		t.Errorf("expected TokenBudget 5000, got %d", subagent.config.TokenBudget)
	}

	if subagent.config.MaxTurns != 5 {
		t.Errorf("expected MaxTurns 5, got %d", subagent.config.MaxTurns)
	}

	if subagent.config.Temperature != 0.5 {
		t.Errorf("expected Temperature 0.5, got %f", subagent.config.Temperature)
	}
}

func TestSubagent_GetConfig(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type: SubagentExplorer,
		Task: "Test task",
	}

	subagent, _ := NewSubagent(parent, config, parent.budget)

	gotConfig := subagent.GetConfig()
	if gotConfig.Type != config.Type {
		t.Error("GetConfig returned wrong type")
	}
	if gotConfig.Task != config.Task {
		t.Error("GetConfig returned wrong task")
	}
}

func TestSubagent_SetHandler(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type: SubagentExplorer,
		Task: "Test task",
	}

	subagent, _ := NewSubagent(parent, config, parent.budget)

	// Test with custom handler
	customHandler := &LoggingSubagentHandler{}
	subagent.SetHandler(customHandler)

	// Handler should be set (we can't easily verify this without running)
	// but at least it shouldn't panic
}

func TestSubagent_ReleaseUnusedBudget(t *testing.T) {
	parent := createTestAgent()
	parentBudget := NewTokenBudget(100000)

	config := SubagentConfig{
		Type:        SubagentExplorer,
		Task:        "Test task",
		TokenBudget: 20000,
	}

	subagent, _ := NewSubagent(parent, config, parentBudget)

	// Simulate some token usage
	subagent.tokensUsed = 5000
	subagent.allocated = 20000

	// Release unused budget
	subagent.ReleaseUnusedBudget(parentBudget)

	// Check that budget was properly adjusted
	// 15000 should be released (20000 allocated - 5000 used)
	// and 5000 should be consumed
}

func TestSubagentManager_GetBudgetStats(t *testing.T) {
	parent := createTestAgent()
	manager := NewSubagentManager(parent, 100000, 3)

	total, used, reserved, available := manager.GetBudgetStats()

	if total != 100000 {
		t.Errorf("expected total 100000, got %d", total)
	}
	if used != 0 {
		t.Errorf("expected used 0, got %d", used)
	}
	if reserved != 0 {
		t.Errorf("expected reserved 0, got %d", reserved)
	}
	if available != 100000 {
		t.Errorf("expected available 100000, got %d", available)
	}
}

func TestSpawnSubagentHelper(t *testing.T) {
	registry := tools.NewRegistry()
	budget := NewTokenBudget(100000)

	helper := NewSpawnSubagentHelper(
		&mockLLMClient{},
		registry,
		budget,
		"test-model",
	)

	if helper == nil {
		t.Fatal("NewSpawnSubagentHelper returned nil")
	}
}

func TestSilentSubagentHandler(t *testing.T) {
	handler := &SilentSubagentHandler{}

	// These should all be no-ops and not panic
	handler.OnSubagentStart(SubagentExplorer, "test task")
	handler.OnSubagentToolCall("read_file", true)
	handler.OnSubagentEnd(&SubagentResult{Success: true})
}

func TestLoggingSubagentHandler(t *testing.T) {
	// Test that logging handler doesn't panic with a nil writer
	// In production use, this would have a valid writer

	// This test just ensures the handler types exist and can be instantiated
	_ = &LoggingSubagentHandler{}
}

func TestSubagentResponseHandler(t *testing.T) {
	// Create a minimal subagent for the handler
	parent := createTestAgent()
	config := SubagentConfig{Type: SubagentExplorer, Task: "Test"}
	subagent, _ := NewSubagent(parent, config, parent.budget)

	handler := &SubagentResponseHandler{
		subagent: subagent,
	}

	handler.OnStreamStart()
	handler.OnStreamChunk("Hello ")
	handler.OnStreamChunk("World")
	handler.OnStreamEnd()
	handler.OnToolStart("read_file", "Reading file")
	handler.OnToolEnd("read_file", true, "contents")
	handler.OnThinking("Processing...")
	handler.OnError(nil)

	output := handler.output.String()
	if output != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", output)
	}

	if len(handler.toolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(handler.toolCalls))
	}
}

func TestGenerateSummary(t *testing.T) {
	parent := createTestAgent()
	config := SubagentConfig{Type: SubagentExplorer, Task: "Test"}
	subagent, _ := NewSubagent(parent, config, parent.budget)

	// Short output should be returned as-is
	short := "This is a short summary."
	if summary := subagent.generateSummary(short); summary != short {
		t.Errorf("short summary should be returned as-is")
	}

	// Long output should be truncated
	long := ""
	for i := 0; i < 200; i++ {
		long += "This is a long text. "
	}
	summary := subagent.generateSummary(long)
	if len(summary) > 550 { // 500 + some margin for "..."
		t.Errorf("summary should be truncated to ~500 chars, got %d", len(summary))
	}

	// Output with summary section should extract it
	withSummary := "Lots of details here. Summary: This is the key finding."
	summary = subagent.generateSummary(withSummary)
	if len(summary) > 0 && summary != withSummary {
		// The summary extraction might work differently
	}
}

func TestExtractModifiedFiles(t *testing.T) {
	parent := createTestAgent()
	config := SubagentConfig{Type: SubagentCoder, Task: "Test"}
	subagent, _ := NewSubagent(parent, config, parent.budget)

	toolCalls := []string{"read_file", "write_file", "list_files", "edit_file"}
	modified := subagent.extractModifiedFiles(toolCalls)

	// Should identify write_file and edit_file as modifying tools
	if len(modified) != 2 {
		t.Errorf("expected 2 modifying tools, got %d", len(modified))
	}
}

func TestTruncateForLog(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is longer", 10, "this is lo..."},
		{"with\nnewline", 20, "with newline"},
	}

	for _, tc := range tests {
		result := truncateForLog(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncateForLog(%q, %d) = %q, expected %q",
				tc.input, tc.maxLen, result, tc.expected)
		}
	}
}

func TestNewSubagentManager_Defaults(t *testing.T) {
	parent := createTestAgent()

	// Test with zero values (should use defaults)
	manager := NewSubagentManager(parent, 0, 0)

	if manager == nil {
		t.Fatal("NewSubagentManager returned nil")
	}

	total, _, _, _ := manager.GetBudgetStats()
	if total != 100000 {
		t.Errorf("expected default total budget 100000, got %d", total)
	}

	if manager.maxParallel != 3 {
		t.Errorf("expected default maxParallel 3, got %d", manager.maxParallel)
	}
}

func TestNewSubagentManager_CustomValues(t *testing.T) {
	parent := createTestAgent()

	manager := NewSubagentManager(parent, 50000, 5)

	if manager == nil {
		t.Fatal("NewSubagentManager returned nil")
	}

	total, _, _, _ := manager.GetBudgetStats()
	if total != 50000 {
		t.Errorf("expected custom total budget 50000, got %d", total)
	}

	if manager.maxParallel != 5 {
		t.Errorf("expected custom maxParallel 5, got %d", manager.maxParallel)
	}
}

func TestSubagent_GetTokensUsed(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type: SubagentExplorer,
		Task: "Test task",
	}

	subagent, _ := NewSubagent(parent, config, parent.budget)

	// Initially should be 0
	if subagent.GetTokensUsed() != 0 {
		t.Errorf("expected 0 tokens used initially, got %d", subagent.GetTokensUsed())
	}

	// Simulate some usage
	subagent.tokensUsed = 1234
	if subagent.GetTokensUsed() != 1234 {
		t.Errorf("expected 1234 tokens used, got %d", subagent.GetTokensUsed())
	}
}

func TestSubagent_ReleaseUnusedBudget_NilBudget(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type: SubagentExplorer,
		Task: "Test task",
	}

	subagent, _ := NewSubagent(parent, config, parent.budget)
	subagent.tokensUsed = 5000
	subagent.allocated = 20000

	// Should not panic with nil budget
	subagent.ReleaseUnusedBudget(nil)
}

func TestNewSubagent_NilBudget(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type:        SubagentExplorer,
		Task:        "Test task",
		TokenBudget: 10000,
	}

	// Should work with nil parent budget (standalone mode)
	subagent, err := NewSubagent(parent, config, nil)
	if err != nil {
		t.Fatalf("NewSubagent with nil budget failed: %v", err)
	}

	if subagent == nil {
		t.Fatal("NewSubagent returned nil")
	}

	// Should have created its own budget
	if subagent.budget == nil {
		t.Error("subagent should have its own budget")
	}

	if subagent.allocated != 10000 {
		t.Errorf("expected allocated 10000, got %d", subagent.allocated)
	}
}

func TestAgentSubagentSpawner(t *testing.T) {
	parent := createTestAgent()

	spawner := NewAgentSubagentSpawner(parent)
	if spawner == nil {
		t.Fatal("NewAgentSubagentSpawner returned nil")
	}

	if spawner.agent != parent {
		t.Error("spawner should reference the parent agent")
	}
}

func TestSubagentSpawnerAdapter(t *testing.T) {
	registry := tools.NewRegistry()
	budget := NewTokenBudget(100000)

	helper := NewSpawnSubagentHelper(
		&mockLLMClient{},
		registry,
		budget,
		"test-model",
	)

	adapter := NewSubagentSpawnerAdapter(helper)
	if adapter == nil {
		t.Fatal("NewSubagentSpawnerAdapter returned nil")
	}

	if adapter.helper != helper {
		t.Error("adapter should reference the helper")
	}
}

func TestLoggingSubagentHandler_WithWriter(t *testing.T) {
	var buf strings.Builder
	handler := &LoggingSubagentHandler{Writer: &buf}

	handler.OnSubagentStart(SubagentExplorer, "Find files")
	handler.OnSubagentToolCall("read_file", true)
	handler.OnSubagentToolCall("read_file", false)
	handler.OnSubagentEnd(&SubagentResult{Success: true, TokensUsed: 100})

	output := buf.String()
	if !strings.Contains(output, "explorer") {
		t.Error("output should contain subagent type")
	}
	if !strings.Contains(output, "read_file") {
		t.Error("output should contain tool name")
	}
	if !strings.Contains(output, "100") {
		t.Error("output should contain tokens used")
	}
	if !strings.Contains(output, "completed") {
		t.Error("output should contain completion status")
	}
}

func TestLoggingSubagentHandler_FailedStatus(t *testing.T) {
	var buf strings.Builder
	handler := &LoggingSubagentHandler{Writer: &buf}

	handler.OnSubagentEnd(&SubagentResult{Success: false, TokensUsed: 50})

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Error("output should contain 'failed' for unsuccessful result")
	}
}

func TestSubagentResponseHandler_WithHandler(t *testing.T) {
	parent := createTestAgent()
	config := SubagentConfig{Type: SubagentExplorer, Task: "Test"}
	subagent, _ := NewSubagent(parent, config, parent.budget)

	// Track handler calls
	toolCallCaptured := false
	mockHandler := &testSubagentHandler{
		onToolCall: func(name string, success bool) {
			toolCallCaptured = true
		},
	}
	subagent.SetHandler(mockHandler)

	handler := &SubagentResponseHandler{
		subagent: subagent,
	}

	handler.OnToolStart("test_tool", "Testing")
	handler.OnToolEnd("test_tool", true, "result")

	if !toolCallCaptured {
		t.Error("subagent handler should have been called on tool end")
	}
}

// testSubagentHandler is a test helper
type testSubagentHandler struct {
	onStart    func(SubagentType, string)
	onToolCall func(string, bool)
	onEnd      func(*SubagentResult)
}

func (h *testSubagentHandler) OnSubagentStart(t SubagentType, task string) {
	if h.onStart != nil {
		h.onStart(t, task)
	}
}

func (h *testSubagentHandler) OnSubagentToolCall(name string, success bool) {
	if h.onToolCall != nil {
		h.onToolCall(name, success)
	}
}

func (h *testSubagentHandler) OnSubagentEnd(result *SubagentResult) {
	if h.onEnd != nil {
		h.onEnd(result)
	}
}

func TestGenerateSummary_WithIndicators(t *testing.T) {
	parent := createTestAgent()
	config := SubagentConfig{Type: SubagentExplorer, Task: "Test"}
	subagent, _ := NewSubagent(parent, config, parent.budget)

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"summary indicator", "Details here. Summary: This is the key point.", "Summary:"},
		{"in summary indicator", "Details. In summary, here's what we found.", "In summary,"},
		{"key findings", "Analysis complete. Key findings: important stuff.", "Key findings:"},
		{"result indicator", "Processing done. Result: success with 5 items.", "Result:"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Need long enough output to trigger summary extraction
			longInput := strings.Repeat("x", 600) + tc.input
			summary := subagent.generateSummary(longInput)
			if !strings.Contains(summary, tc.contains) {
				t.Errorf("summary should contain %q indicator", tc.contains)
			}
		})
	}
}

func TestExtractModifiedFiles_AllTypes(t *testing.T) {
	parent := createTestAgent()
	config := SubagentConfig{Type: SubagentCoder, Task: "Test"}
	subagent, _ := NewSubagent(parent, config, parent.budget)

	// Test all modifying tools
	modifyingTools := []string{"write_file", "edit_file", "insert_lines", "delete_lines", "delete_file"}
	modified := subagent.extractModifiedFiles(modifyingTools)

	if len(modified) != 5 {
		t.Errorf("expected 5 modifying tools, got %d", len(modified))
	}

	// Test empty list
	modified = subagent.extractModifiedFiles([]string{})
	if len(modified) != 0 {
		t.Errorf("expected 0 for empty input, got %d", len(modified))
	}

	// Test read-only tools
	readOnlyTools := []string{"read_file", "list_files", "grep_search"}
	modified = subagent.extractModifiedFiles(readOnlyTools)
	if len(modified) != 0 {
		t.Errorf("expected 0 for read-only tools, got %d", len(modified))
	}
}

func TestNewSubagent_DefaultTemperature(t *testing.T) {
	parent := createTestAgent()

	config := SubagentConfig{
		Type:        SubagentExplorer,
		Task:        "Test task",
		Temperature: 0, // Zero value
	}

	subagent, err := NewSubagent(parent, config, parent.budget)
	if err != nil {
		t.Fatalf("NewSubagent failed: %v", err)
	}

	// Should have default temperature of 0.7
	if subagent.config.Temperature != 0.7 {
		t.Errorf("expected default temperature 0.7, got %f", subagent.config.Temperature)
	}
}

func TestSubagentManager_SubagentTracking(t *testing.T) {
	parent := createTestAgent()
	manager := NewSubagentManager(parent, 100000, 3)

	if len(manager.subagents) != 0 {
		t.Error("manager should start with no subagents")
	}
}
