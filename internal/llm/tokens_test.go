package llm

import (
	"testing"
	"time"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minTokens int
		maxTokens int
	}{
		{
			name:      "empty string",
			input:     "",
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "single word",
			input:     "hello",
			minTokens: 1,
			maxTokens: 2,
		},
		{
			name:      "short sentence",
			input:     "Hello, world!",
			minTokens: 2,
			maxTokens: 5,
		},
		{
			name:      "code snippet",
			input:     "func main() { fmt.Println(\"hello\") }",
			minTokens: 5,
			maxTokens: 15,
		},
		{
			name:      "whitespace only",
			input:     "   \t\n   ",
			minTokens: 0,
			maxTokens: 1,
		},
		{
			name:      "long text",
			input:     "This is a longer piece of text that should result in more tokens being estimated by the function.",
			minTokens: 15,
			maxTokens: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result < tt.minTokens || result > tt.maxTokens {
				t.Errorf("EstimateTokens(%q) = %d, want between %d and %d",
					tt.input, result, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestEstimateTokens_NonEmpty(t *testing.T) {
	// Non-empty text should always return at least 1 token
	inputs := []string{"a", "ab", "abc", "hello"}
	for _, input := range inputs {
		result := EstimateTokens(input)
		if result < 1 {
			t.Errorf("EstimateTokens(%q) = %d, want >= 1", input, result)
		}
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	tests := []struct {
		name      string
		messages  []Message
		minTokens int
		maxTokens int
	}{
		{
			name:      "empty messages",
			messages:  []Message{},
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name: "single message",
			messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			minTokens: 4, // overhead + content
			maxTokens: 10,
		},
		{
			name: "multiple messages",
			messages: []Message{
				{Role: RoleSystem, Content: "You are a helpful assistant."},
				{Role: RoleUser, Content: "Hello"},
				{Role: RoleAssistant, Content: "Hi there!"},
			},
			minTokens: 12, // 3 * 4 overhead + content
			maxTokens: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateMessagesTokens(tt.messages)
			if result < tt.minTokens || result > tt.maxTokens {
				t.Errorf("EstimateMessagesTokens() = %d, want between %d and %d",
					result, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestNewUsageStats(t *testing.T) {
	stats := NewUsageStats()

	if stats == nil {
		t.Fatal("NewUsageStats() returned nil")
	}

	if stats.TotalTokens != 0 {
		t.Errorf("expected TotalTokens 0, got %d", stats.TotalTokens)
	}

	if stats.RequestCount != 0 {
		t.Errorf("expected RequestCount 0, got %d", stats.RequestCount)
	}

	if stats.maxHistory != 100 {
		t.Errorf("expected maxHistory 100, got %d", stats.maxHistory)
	}
}

func TestUsageStats_RecordRequest(t *testing.T) {
	stats := NewUsageStats()

	// Record a successful request
	stats.RecordRequest("test-model", 100, 50, time.Second, true, nil)

	if stats.RequestCount != 1 {
		t.Errorf("expected RequestCount 1, got %d", stats.RequestCount)
	}

	if stats.SuccessCount != 1 {
		t.Errorf("expected SuccessCount 1, got %d", stats.SuccessCount)
	}

	if stats.TotalPromptTokens != 100 {
		t.Errorf("expected TotalPromptTokens 100, got %d", stats.TotalPromptTokens)
	}

	if stats.TotalCompletionTokens != 50 {
		t.Errorf("expected TotalCompletionTokens 50, got %d", stats.TotalCompletionTokens)
	}

	if stats.TotalTokens != 150 {
		t.Errorf("expected TotalTokens 150, got %d", stats.TotalTokens)
	}

	// Record a failed request
	stats.RecordRequest("test-model", 50, 0, time.Second, false, nil)

	if stats.RequestCount != 2 {
		t.Errorf("expected RequestCount 2, got %d", stats.RequestCount)
	}

	if stats.ErrorCount != 1 {
		t.Errorf("expected ErrorCount 1, got %d", stats.ErrorCount)
	}
}

func TestUsageStats_RecordRequestWithPricing(t *testing.T) {
	stats := NewUsageStats()
	pricing := &Pricing{
		PromptPricePerMillion:     10.0,
		CompletionPricePerMillion: 30.0,
	}

	// Record request with 1M prompt tokens and 1M completion tokens
	stats.RecordRequest("test-model", 1000000, 1000000, time.Second, true, pricing)

	// Expected cost: $10 + $30 = $40
	expectedCost := 40.0
	if stats.EstimatedCost != expectedCost {
		t.Errorf("expected EstimatedCost %.2f, got %.2f", expectedCost, stats.EstimatedCost)
	}
}

func TestUsageStats_HistoryLimit(t *testing.T) {
	stats := NewUsageStats()

	// Record more than maxHistory requests
	for i := 0; i < 150; i++ {
		stats.RecordRequest("test-model", 10, 5, time.Millisecond, true, nil)
	}

	if len(stats.History) > 100 {
		t.Errorf("expected History length <= 100, got %d", len(stats.History))
	}
}

func TestUsageStats_Reset(t *testing.T) {
	stats := NewUsageStats()

	// Add some data
	stats.RecordRequest("test-model", 100, 50, time.Second, true, nil)

	// Reset
	stats.Reset()

	if stats.TotalTokens != 0 {
		t.Errorf("expected TotalTokens 0 after reset, got %d", stats.TotalTokens)
	}

	if stats.RequestCount != 0 {
		t.Errorf("expected RequestCount 0 after reset, got %d", stats.RequestCount)
	}

	if len(stats.History) != 0 {
		t.Errorf("expected empty History after reset, got %d items", len(stats.History))
	}
}

func TestUsageStats_GetStats(t *testing.T) {
	stats := NewUsageStats()
	stats.RecordRequest("test-model", 100, 50, time.Second, true, nil)

	// Get a copy of stats
	copy := stats.GetStats()

	// Modify original
	stats.RecordRequest("test-model", 100, 50, time.Second, true, nil)

	// Copy should be unchanged
	if copy.TotalTokens != 150 {
		t.Errorf("expected copy TotalTokens 150, got %d", copy.TotalTokens)
	}
}

func TestUsageStats_Summary(t *testing.T) {
	stats := NewUsageStats()
	stats.RecordRequest("test-model", 100, 50, time.Second, true, nil)

	summary := stats.Summary()

	if summary == "" {
		t.Error("expected non-empty summary")
	}

	// Check that summary contains expected elements
	if !containsSubstring(summary, "Token Usage Summary") {
		t.Error("summary should contain 'Token Usage Summary'")
	}
}

func TestUsageStats_ShortSummary(t *testing.T) {
	stats := NewUsageStats()
	stats.RecordRequest("test-model", 100, 50, time.Second, true, nil)

	short := stats.ShortSummary()

	if short == "" {
		t.Error("expected non-empty short summary")
	}

	if !containsSubstring(short, "Tokens:") {
		t.Error("short summary should contain 'Tokens:'")
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{10000, "10.0K"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		result := formatTokenCount(tt.count)
		if result != tt.expected {
			t.Errorf("formatTokenCount(%d) = %s, want %s", tt.count, result, tt.expected)
		}
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
