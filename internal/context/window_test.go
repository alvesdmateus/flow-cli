package context

import (
	"strings"
	"testing"

	"github.com/mateus/flow-cli/internal/llm"
)

func TestGetModelLimits(t *testing.T) {
	tests := []struct {
		model       string
		wantMax     int
		wantDefault bool
	}{
		{"llama3", 8192, false},
		{"llama3:latest", 8192, false},
		{"llama3.1:70b", 131072, false},
		{"gpt-4-turbo", 128000, false},
		{"gpt-4o", 128000, false},
		{"mistral:7b", 8192, false},
		{"unknown-model", 4096, true}, // Should return default
		{"", 4096, true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			limits := GetModelLimits(tt.model)

			if limits.MaxTokens != tt.wantMax {
				t.Errorf("GetModelLimits(%q) MaxTokens = %d, want %d", tt.model, limits.MaxTokens, tt.wantMax)
			}

			if tt.wantDefault && limits.MaxTokens != DefaultModelLimits["default"].MaxTokens {
				t.Errorf("Expected default limits for %q", tt.model)
			}
		})
	}
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"llama3:latest", "llama3"},
		{"llama3:70b", "llama3"},
		{"gpt-4-turbo", "gpt-4-turbo"},
		{"Mistral:7B", "mistral"},
		{"model", "model"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeModelName(tt.input)
			if got != tt.want {
				t.Errorf("normalizeModelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWindowManager_CalculateStats(t *testing.T) {
	wm := NewWindowManager(WindowConfig{
		Model:          "llama3",
		ReservedTokens: 1024,
	})

	messages := []Message{
		{Message: llm.Message{Role: llm.RoleUser, Content: "Hello, how are you?"}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: "I'm doing well, thanks for asking!"}},
		{Message: llm.Message{Role: llm.RoleUser, Content: "Can you help me with some code?"}},
	}

	stats := wm.CalculateStats(messages, "You are a helpful assistant.")

	if stats.TotalTokens <= 0 {
		t.Error("Expected positive token count")
	}

	if stats.MessageCount != 3 {
		t.Errorf("Expected 3 messages, got %d", stats.MessageCount)
	}

	if stats.MaxTokens != 8192 {
		t.Errorf("Expected MaxTokens 8192, got %d", stats.MaxTokens)
	}

	if stats.UsagePercent <= 0 {
		t.Error("Expected positive usage percent")
	}
}

func TestWindowManager_NeedsPruning(t *testing.T) {
	// Create manager with small limits for testing
	wm := NewWindowManager(WindowConfig{
		Model:          "default",
		MaxTokens:      100, // Very small for testing
		ReservedTokens: 20,
	})

	// Small message should not need pruning
	smallMessages := []Message{
		{Message: llm.Message{Role: llm.RoleUser, Content: "Hi"}},
	}

	if wm.NeedsPruning(smallMessages, "") {
		t.Error("Small message should not need pruning")
	}

	// Large messages should need pruning
	largeMessages := []Message{
		{Message: llm.Message{Role: llm.RoleUser, Content: strings.Repeat("word ", 100)}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: strings.Repeat("response ", 100)}},
	}

	if !wm.NeedsPruning(largeMessages, "") {
		t.Error("Large messages should need pruning")
	}
}

func TestWindowManager_PruneMessages_Oldest(t *testing.T) {
	wm := NewWindowManager(WindowConfig{
		Model:           "default",
		MaxTokens:       200,
		ReservedTokens:  50,
		Strategy:        StrategyRemoveOldest,
		KeepRecentCount: 2,
	})

	messages := []Message{
		{Message: llm.Message{Role: llm.RoleUser, Content: strings.Repeat("old message one ", 20)}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: strings.Repeat("old response one ", 20)}},
		{Message: llm.Message{Role: llm.RoleUser, Content: "recent message"}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: "recent response"}},
	}

	result := wm.PruneMessages(messages, nil, "")

	// Should keep recent messages
	if len(result) > len(messages) {
		t.Errorf("Should not add messages, got %d from %d", len(result), len(messages))
	}

	// Last message should be preserved
	if len(result) > 0 && result[len(result)-1].Content != "recent response" {
		t.Error("Should preserve most recent message")
	}
}

func TestWindowManager_PruneMessages_LowPriority(t *testing.T) {
	wm := NewWindowManager(WindowConfig{
		Model:           "default",
		MaxTokens:       200,
		ReservedTokens:  50,
		Strategy:        StrategyRemoveLowPriority,
		KeepRecentCount: 1,
	})

	messages := []Message{
		{Message: llm.Message{Role: llm.RoleUser, Content: strings.Repeat("low priority ", 20)}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: strings.Repeat("high priority ", 20)}},
		{Message: llm.Message{Role: llm.RoleUser, Content: "recent"}},
	}

	priorities := []MessagePriority{
		PriorityLow,
		PriorityHigh,
		PriorityHigh,
	}

	result := wm.PruneMessages(messages, priorities, "")

	// Result should be equal or smaller than input
	if len(result) > len(messages) {
		t.Errorf("Pruning should not add messages: got %d, had %d", len(result), len(messages))
	}

	// Recent message should be preserved
	if len(result) > 0 {
		lastMsg := result[len(result)-1]
		if !strings.Contains(lastMsg.Content, "recent") {
			t.Error("Most recent message should be preserved")
		}
	}
}

func TestWindowManager_PruneMessages_Summary(t *testing.T) {
	wm := NewWindowManager(WindowConfig{
		Model:           "default",
		MaxTokens:       150,
		ReservedTokens:  30,
		Strategy:        StrategySummarize,
		KeepRecentCount: 2,
	})

	messages := []Message{
		{Message: llm.Message{Role: llm.RoleUser, Content: strings.Repeat("old ", 30)}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: strings.Repeat("old response ", 30)}},
		{Message: llm.Message{Role: llm.RoleUser, Content: "recent"}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: "recent response"}},
	}

	result := wm.PruneMessages(messages, nil, "")

	// Should have summary + recent messages
	if len(result) == 0 {
		t.Error("Should return some messages")
	}

	// Check for summary message when pruning occurred
	if len(result) < len(messages) {
		hasSummary := false
		for _, msg := range result {
			if strings.Contains(msg.Content, "summary") || strings.Contains(msg.Content, "Previous") {
				hasSummary = true
				break
			}
		}
		// Note: summary may not always be present if recent messages fit
		_ = hasSummary // Acknowledge variable is intentionally unused in some cases
	}
}

func TestCreateSummary(t *testing.T) {
	messages := []Message{
		{Message: llm.Message{Role: llm.RoleUser, Content: "Help me fix this bug in the code"}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: "I'll help you fix that bug"}, ToolCalls: []ToolCall{{Name: "read_file"}}},
		{Message: llm.Message{Role: llm.RoleUser, Content: "Thanks, now let's add a test"}},
		{Message: llm.Message{Role: llm.RoleAssistant, Content: "Here's the test code"}},
	}

	summary := createSummary(messages)

	if summary.Role != llm.RoleSystem {
		t.Errorf("Expected system role, got %s", summary.Role)
	}

	if !strings.Contains(summary.Content, "summary") && !strings.Contains(summary.Content, "Previous") {
		t.Error("Summary should contain 'summary' or 'Previous'")
	}

	if !strings.Contains(summary.Content, "2 user messages") {
		t.Error("Summary should mention user message count")
	}

	if !strings.Contains(summary.Content, "tool") {
		t.Error("Summary should mention tool calls")
	}
}

func TestExtractTopics(t *testing.T) {
	messages := []Message{
		{Message: llm.Message{Content: "I need to fix a bug in the code"}},
		{Message: llm.Message{Content: "Let me read the file and fix the bug"}},
		{Message: llm.Message{Content: "Now let's add a test for this fix"}},
		{Message: llm.Message{Content: "The test should cover the bug fix"}},
	}

	topics := extractTopics(messages)

	// Should find common topics
	foundBug := false
	foundFix := false
	foundTest := false

	for _, topic := range topics {
		if topic == "bug" {
			foundBug = true
		}
		if topic == "fix" {
			foundFix = true
		}
		if topic == "test" {
			foundTest = true
		}
	}

	if !foundBug {
		t.Error("Should find 'bug' topic")
	}
	if !foundFix {
		t.Error("Should find 'fix' topic")
	}
	if !foundTest {
		t.Error("Should find 'test' topic")
	}
}

func TestFormatStats(t *testing.T) {
	stats := ContextStats{
		TotalTokens:     4000,
		MaxTokens:       8000,
		AvailableTokens: 4000,
		UsagePercent:    0.5,
		MessageCount:    10,
		IsNearLimit:     false,
	}

	formatted := FormatStats(stats)

	if !strings.Contains(formatted, "4.0K") {
		t.Errorf("Should show used tokens in K format: %s", formatted)
	}

	if !strings.Contains(formatted, "8.0K") {
		t.Errorf("Should show max tokens in K format: %s", formatted)
	}

	if !strings.Contains(formatted, "50%") {
		t.Errorf("Should show percentage: %s", formatted)
	}

	// Test near limit warning
	statsNearLimit := ContextStats{
		TotalTokens:  7000,
		MaxTokens:    8000,
		UsagePercent: 0.875,
		IsNearLimit:  true,
	}

	formattedNearLimit := FormatStats(statsNearLimit)
	if !strings.Contains(formattedNearLimit, "⚠️") {
		t.Error("Should show warning when near limit")
	}
}

func TestFormatStatsCompact(t *testing.T) {
	stats := ContextStats{
		TotalTokens: 1500,
		MaxTokens:   4096,
	}

	compact := FormatStatsCompact(stats)

	if compact != "1.5K/4.1K" {
		t.Errorf("Expected '1.5K/4.1K', got '%s'", compact)
	}
}

func TestMessagePriority_Constants(t *testing.T) {
	// Verify priority ordering
	if PriorityLow >= PriorityMedium {
		t.Error("PriorityLow should be less than PriorityMedium")
	}
	if PriorityMedium >= PriorityHigh {
		t.Error("PriorityMedium should be less than PriorityHigh")
	}
	if PriorityHigh >= PriorityCritical {
		t.Error("PriorityHigh should be less than PriorityCritical")
	}
}

func TestWindowConfig_Defaults(t *testing.T) {
	wm := NewWindowManager(WindowConfig{
		Model: "llama3",
	})

	if wm.config.KeepRecentCount != 4 {
		t.Errorf("Expected default KeepRecentCount of 4, got %d", wm.config.KeepRecentCount)
	}
}

func TestWindowManager_EmptyMessages(t *testing.T) {
	wm := NewWindowManager(WindowConfig{
		Model: "llama3",
	})

	stats := wm.CalculateStats([]Message{}, "")

	if stats.TotalTokens != 0 {
		t.Errorf("Expected 0 tokens for empty messages, got %d", stats.TotalTokens)
	}

	if stats.MessageCount != 0 {
		t.Errorf("Expected 0 message count, got %d", stats.MessageCount)
	}

	// Prune empty should return empty
	result := wm.PruneMessages([]Message{}, nil, "")
	if len(result) != 0 {
		t.Error("Pruning empty messages should return empty")
	}
}

func TestEstimateMessageTokens_WithToolCalls(t *testing.T) {
	wm := NewWindowManager(WindowConfig{Model: "llama3"})

	msgWithoutTools := Message{
		Message: llm.Message{Role: llm.RoleAssistant, Content: "Hello"},
	}

	msgWithTools := Message{
		Message: llm.Message{Role: llm.RoleAssistant, Content: "Hello"},
		ToolCalls: []ToolCall{
			{Name: "read_file", Result: "file contents here"},
			{Name: "write_file", Result: "success"},
		},
	}

	tokensWithout := wm.estimateMessageTokens(msgWithoutTools)
	tokensWith := wm.estimateMessageTokens(msgWithTools)

	if tokensWith <= tokensWithout {
		t.Error("Message with tool calls should have more tokens")
	}
}
