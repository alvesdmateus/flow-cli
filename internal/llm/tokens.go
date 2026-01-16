package llm

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// TokenUsage represents token counts for a single request
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// UsageStats tracks cumulative token usage across a session
type UsageStats struct {
	mu sync.RWMutex

	// Cumulative token counts
	TotalPromptTokens     int `json:"total_prompt_tokens"`
	TotalCompletionTokens int `json:"total_completion_tokens"`
	TotalTokens           int `json:"total_tokens"`

	// Request counts
	RequestCount   int `json:"request_count"`
	SuccessCount   int `json:"success_count"`
	ErrorCount     int `json:"error_count"`

	// Cost tracking (if pricing is configured)
	EstimatedCost float64 `json:"estimated_cost"`

	// Timing
	StartTime     time.Time `json:"start_time"`
	TotalDuration time.Duration `json:"total_duration"`

	// Per-request history (limited to last N requests)
	History []RequestStats `json:"history,omitempty"`
	maxHistory int
}

// RequestStats tracks stats for a single request
type RequestStats struct {
	Timestamp        time.Time     `json:"timestamp"`
	Model            string        `json:"model"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	Duration         time.Duration `json:"duration"`
	Success          bool          `json:"success"`
}

// Pricing represents token pricing for cost estimation
type Pricing struct {
	PromptPricePerMillion     float64 // Price per 1M prompt tokens
	CompletionPricePerMillion float64 // Price per 1M completion tokens
}

// Common pricing (approximate, may vary)
var PricingPresets = map[string]Pricing{
	"gpt-4":        {PromptPricePerMillion: 30.0, CompletionPricePerMillion: 60.0},
	"gpt-4-turbo":  {PromptPricePerMillion: 10.0, CompletionPricePerMillion: 30.0},
	"gpt-3.5":      {PromptPricePerMillion: 0.5, CompletionPricePerMillion: 1.5},
	"claude-3":     {PromptPricePerMillion: 15.0, CompletionPricePerMillion: 75.0},
	"local":        {PromptPricePerMillion: 0.0, CompletionPricePerMillion: 0.0},
	"ollama":       {PromptPricePerMillion: 0.0, CompletionPricePerMillion: 0.0},
}

// NewUsageStats creates a new usage stats tracker
func NewUsageStats() *UsageStats {
	return &UsageStats{
		StartTime:  time.Now(),
		History:    make([]RequestStats, 0),
		maxHistory: 100, // Keep last 100 requests
	}
}

// EstimateTokens estimates the number of tokens in a text
// This uses a simple heuristic: ~4 characters per token for English text
// For more accurate counting, use a proper tokenizer library
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count characters (excluding excessive whitespace)
	text = strings.TrimSpace(text)
	charCount := len(text)

	// Rough estimate: 4 characters per token on average
	// This is a reasonable approximation for English text with code
	tokens := (charCount + 3) / 4

	// Minimum 1 token for non-empty text
	if tokens == 0 && charCount > 0 {
		tokens = 1
	}

	return tokens
}

// EstimateMessagesTokens estimates tokens for a list of messages
func EstimateMessagesTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// Add overhead for message structure (~4 tokens per message)
		total += 4
		total += EstimateTokens(msg.Content)
	}
	return total
}

// RecordRequest records stats for a completed request
func (s *UsageStats) RecordRequest(model string, promptTokens, completionTokens int, duration time.Duration, success bool, pricing *Pricing) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.RequestCount++
	if success {
		s.SuccessCount++
	} else {
		s.ErrorCount++
	}

	s.TotalPromptTokens += promptTokens
	s.TotalCompletionTokens += completionTokens
	s.TotalTokens += promptTokens + completionTokens
	s.TotalDuration += duration

	// Calculate cost if pricing provided
	if pricing != nil {
		promptCost := float64(promptTokens) * pricing.PromptPricePerMillion / 1_000_000
		completionCost := float64(completionTokens) * pricing.CompletionPricePerMillion / 1_000_000
		s.EstimatedCost += promptCost + completionCost
	}

	// Add to history
	stats := RequestStats{
		Timestamp:        time.Now(),
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Duration:         duration,
		Success:          success,
	}

	s.History = append(s.History, stats)

	// Trim history if too long
	if len(s.History) > s.maxHistory {
		s.History = s.History[len(s.History)-s.maxHistory:]
	}
}

// GetStats returns a copy of current stats
func (s *UsageStats) GetStats() UsageStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return UsageStats{
		TotalPromptTokens:     s.TotalPromptTokens,
		TotalCompletionTokens: s.TotalCompletionTokens,
		TotalTokens:           s.TotalTokens,
		RequestCount:          s.RequestCount,
		SuccessCount:          s.SuccessCount,
		ErrorCount:            s.ErrorCount,
		EstimatedCost:         s.EstimatedCost,
		StartTime:             s.StartTime,
		TotalDuration:         s.TotalDuration,
	}
}

// Summary returns a formatted summary of usage stats
func (s *UsageStats) Summary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("=== Token Usage Summary ===\n")
	sb.WriteString(fmt.Sprintf("Session Duration: %s\n", time.Since(s.StartTime).Round(time.Second)))
	sb.WriteString(fmt.Sprintf("Requests: %d total (%d success, %d error)\n",
		s.RequestCount, s.SuccessCount, s.ErrorCount))
	sb.WriteString(fmt.Sprintf("Tokens Used:\n"))
	sb.WriteString(fmt.Sprintf("  Prompt:     %s\n", formatTokenCount(s.TotalPromptTokens)))
	sb.WriteString(fmt.Sprintf("  Completion: %s\n", formatTokenCount(s.TotalCompletionTokens)))
	sb.WriteString(fmt.Sprintf("  Total:      %s\n", formatTokenCount(s.TotalTokens)))

	if s.EstimatedCost > 0 {
		sb.WriteString(fmt.Sprintf("Estimated Cost: $%.4f\n", s.EstimatedCost))
	}

	if s.RequestCount > 0 {
		avgTokens := s.TotalTokens / s.RequestCount
		avgDuration := s.TotalDuration / time.Duration(s.RequestCount)
		sb.WriteString(fmt.Sprintf("Average per request:\n"))
		sb.WriteString(fmt.Sprintf("  Tokens: %d\n", avgTokens))
		sb.WriteString(fmt.Sprintf("  Duration: %s\n", avgDuration.Round(time.Millisecond)))
	}

	return sb.String()
}

// ShortSummary returns a one-line summary
func (s *UsageStats) ShortSummary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.EstimatedCost > 0 {
		return fmt.Sprintf("Tokens: %s (≈$%.4f)", formatTokenCount(s.TotalTokens), s.EstimatedCost)
	}
	return fmt.Sprintf("Tokens: %s", formatTokenCount(s.TotalTokens))
}

// formatTokenCount formats a token count with K/M suffixes
func formatTokenCount(count int) string {
	if count >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	}
	if count >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(count)/1_000)
	}
	return fmt.Sprintf("%d", count)
}

// Reset clears all stats
func (s *UsageStats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalPromptTokens = 0
	s.TotalCompletionTokens = 0
	s.TotalTokens = 0
	s.RequestCount = 0
	s.SuccessCount = 0
	s.ErrorCount = 0
	s.EstimatedCost = 0
	s.StartTime = time.Now()
	s.TotalDuration = 0
	s.History = make([]RequestStats, 0)
}
