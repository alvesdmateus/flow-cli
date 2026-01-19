package context

import (
	"fmt"
	"strings"

	"github.com/mateus/flow-cli/internal/llm"
)

// MessagePriority defines the importance level of a message
type MessagePriority int

const (
	// PriorityLow - can be removed first (old tool results, verbose output)
	PriorityLow MessagePriority = iota
	// PriorityMedium - standard messages
	PriorityMedium
	// PriorityHigh - should be kept (recent messages, important context)
	PriorityHigh
	// PriorityCritical - must not be removed (system prompt, current task)
	PriorityCritical
)

// ModelContextLimits defines context window sizes for different models
type ModelContextLimits struct {
	MaxTokens        int     // Maximum context window size
	ReservedTokens   int     // Tokens reserved for response
	WarningThreshold float64 // Percentage at which to warn (0.0-1.0)
}

// DefaultModelLimits provides context limits for common models
var DefaultModelLimits = map[string]ModelContextLimits{
	// Ollama models
	"llama2":         {MaxTokens: 4096, ReservedTokens: 512, WarningThreshold: 0.8},
	"llama3":         {MaxTokens: 8192, ReservedTokens: 1024, WarningThreshold: 0.8},
	"llama3.1":       {MaxTokens: 131072, ReservedTokens: 4096, WarningThreshold: 0.9},
	"llama3.2":       {MaxTokens: 131072, ReservedTokens: 4096, WarningThreshold: 0.9},
	"codellama":      {MaxTokens: 16384, ReservedTokens: 2048, WarningThreshold: 0.8},
	"mistral":        {MaxTokens: 8192, ReservedTokens: 1024, WarningThreshold: 0.8},
	"mixtral":        {MaxTokens: 32768, ReservedTokens: 2048, WarningThreshold: 0.8},
	"deepseek-coder": {MaxTokens: 16384, ReservedTokens: 2048, WarningThreshold: 0.8},
	"qwen2.5-coder":  {MaxTokens: 32768, ReservedTokens: 2048, WarningThreshold: 0.8},
	"gemma2":         {MaxTokens: 8192, ReservedTokens: 1024, WarningThreshold: 0.8},
	"phi3":           {MaxTokens: 4096, ReservedTokens: 512, WarningThreshold: 0.8},

	// OpenAI models
	"gpt-3.5-turbo": {MaxTokens: 16384, ReservedTokens: 2048, WarningThreshold: 0.8},
	"gpt-4":         {MaxTokens: 8192, ReservedTokens: 1024, WarningThreshold: 0.8},
	"gpt-4-turbo":   {MaxTokens: 128000, ReservedTokens: 4096, WarningThreshold: 0.9},
	"gpt-4o":        {MaxTokens: 128000, ReservedTokens: 4096, WarningThreshold: 0.9},

	// Default for unknown models
	"default": {MaxTokens: 4096, ReservedTokens: 512, WarningThreshold: 0.8},
}

// GetModelLimits returns context limits for a model
func GetModelLimits(model string) ModelContextLimits {
	// Normalize model name (remove version suffixes, etc.)
	normalizedModel := normalizeModelName(model)

	if limits, ok := DefaultModelLimits[normalizedModel]; ok {
		return limits
	}

	// Try partial match for model families
	for name, limits := range DefaultModelLimits {
		if strings.Contains(strings.ToLower(model), name) {
			return limits
		}
	}

	return DefaultModelLimits["default"]
}

// normalizeModelName extracts the base model name
func normalizeModelName(model string) string {
	model = strings.ToLower(model)
	// Remove common suffixes like :latest, :7b, :13b, etc.
	if idx := strings.Index(model, ":"); idx > 0 {
		model = model[:idx]
	}
	return model
}

// PruningStrategy defines how to prune messages when context is full
type PruningStrategy int

const (
	// StrategyRemoveOldest removes oldest messages first
	StrategyRemoveOldest PruningStrategy = iota
	// StrategyRemoveLowPriority removes low priority messages first
	StrategyRemoveLowPriority
	// StrategySummarize summarizes old messages into a single message
	StrategySummarize
	// StrategyHybrid combines priority-based removal with summarization
	StrategyHybrid
)

// WindowConfig configures the context window manager
type WindowConfig struct {
	Model            string
	MaxTokens        int // Override default max tokens
	ReservedTokens   int // Tokens reserved for response
	Strategy         PruningStrategy
	KeepSystemPrompt bool // Always keep system prompt
	KeepRecentCount  int  // Always keep N most recent messages
}

// WindowManager manages context window limits and pruning
type WindowManager struct {
	config WindowConfig
	limits ModelContextLimits
}

// NewWindowManager creates a new context window manager
func NewWindowManager(config WindowConfig) *WindowManager {
	limits := GetModelLimits(config.Model)

	// Apply overrides
	if config.MaxTokens > 0 {
		limits.MaxTokens = config.MaxTokens
	}
	if config.ReservedTokens > 0 {
		limits.ReservedTokens = config.ReservedTokens
	}
	if config.KeepRecentCount == 0 {
		config.KeepRecentCount = 4 // Default: keep last 4 messages
	}

	return &WindowManager{
		config: config,
		limits: limits,
	}
}

// ContextStats provides statistics about current context usage
type ContextStats struct {
	TotalTokens     int
	MaxTokens       int
	AvailableTokens int
	UsagePercent    float64
	MessageCount    int
	IsNearLimit     bool
	CanAddTokens    int
}

// CalculateStats calculates context statistics for messages
func (w *WindowManager) CalculateStats(messages []Message, systemPrompt string) ContextStats {
	totalTokens := 0

	// Count system prompt tokens
	if systemPrompt != "" {
		totalTokens += llm.EstimateTokens(systemPrompt) + 4 // +4 for message overhead
	}

	// Count message tokens
	for _, msg := range messages {
		totalTokens += w.estimateMessageTokens(msg)
	}

	availableForPrompt := w.limits.MaxTokens - w.limits.ReservedTokens
	available := availableForPrompt - totalTokens
	if available < 0 {
		available = 0
	}

	usagePercent := float64(totalTokens) / float64(availableForPrompt)
	isNearLimit := usagePercent >= w.limits.WarningThreshold

	return ContextStats{
		TotalTokens:     totalTokens,
		MaxTokens:       w.limits.MaxTokens,
		AvailableTokens: available,
		UsagePercent:    usagePercent,
		MessageCount:    len(messages),
		IsNearLimit:     isNearLimit,
		CanAddTokens:    available,
	}
}

// estimateMessageTokens estimates tokens for a single message
func (w *WindowManager) estimateMessageTokens(msg Message) int {
	tokens := 4 // Message structure overhead
	tokens += llm.EstimateTokens(msg.Content)

	// Add tokens for tool calls
	for _, tc := range msg.ToolCalls {
		tokens += 10 // Tool call structure
		tokens += llm.EstimateTokens(tc.Name)
		tokens += llm.EstimateTokens(tc.Result)
	}

	return tokens
}

// NeedsPruning checks if pruning is needed
func (w *WindowManager) NeedsPruning(messages []Message, systemPrompt string) bool {
	stats := w.CalculateStats(messages, systemPrompt)
	return stats.AvailableTokens < w.limits.ReservedTokens/2
}

// PruneMessages removes messages to fit within context limits
func (w *WindowManager) PruneMessages(messages []Message, priorities []MessagePriority, systemPrompt string) []Message {
	if len(messages) == 0 {
		return messages
	}

	// Ensure priorities slice matches messages
	if len(priorities) != len(messages) {
		priorities = make([]MessagePriority, len(messages))
		for i := range priorities {
			priorities[i] = PriorityMedium
		}
	}

	switch w.config.Strategy {
	case StrategyRemoveLowPriority:
		return w.pruneLowPriority(messages, priorities, systemPrompt)
	case StrategySummarize:
		return w.pruneWithSummary(messages, systemPrompt)
	case StrategyHybrid:
		return w.pruneHybrid(messages, priorities, systemPrompt)
	default:
		return w.pruneOldest(messages, systemPrompt)
	}
}

// pruneOldest removes oldest messages first
func (w *WindowManager) pruneOldest(messages []Message, systemPrompt string) []Message {
	stats := w.CalculateStats(messages, systemPrompt)
	targetTokens := w.limits.MaxTokens - w.limits.ReservedTokens

	if stats.TotalTokens <= targetTokens {
		return messages
	}

	// Keep removing oldest messages until we fit
	result := make([]Message, len(messages))
	copy(result, messages)

	for len(result) > w.config.KeepRecentCount {
		stats = w.CalculateStats(result, systemPrompt)
		if stats.TotalTokens <= targetTokens {
			break
		}
		result = result[1:]
	}

	return result
}

// pruneLowPriority removes low priority messages first
func (w *WindowManager) pruneLowPriority(messages []Message, priorities []MessagePriority, systemPrompt string) []Message {
	stats := w.CalculateStats(messages, systemPrompt)
	targetTokens := w.limits.MaxTokens - w.limits.ReservedTokens

	if stats.TotalTokens <= targetTokens {
		return messages
	}

	// Create indexed list for sorting
	type indexedMsg struct {
		index    int
		msg      Message
		priority MessagePriority
	}

	indexed := make([]indexedMsg, len(messages))
	for i, msg := range messages {
		indexed[i] = indexedMsg{
			index:    i,
			msg:      msg,
			priority: priorities[i],
		}
	}

	// Mark messages for removal starting with lowest priority
	removed := make(map[int]bool)
	tokensToRemove := stats.TotalTokens - targetTokens

	// First pass: remove low priority
	for _, im := range indexed {
		if tokensToRemove <= 0 {
			break
		}
		// Don't remove recent messages
		if im.index >= len(messages)-w.config.KeepRecentCount {
			continue
		}
		if im.priority == PriorityLow {
			removed[im.index] = true
			tokensToRemove -= w.estimateMessageTokens(im.msg)
		}
	}

	// Second pass: remove medium priority if still needed
	if tokensToRemove > 0 {
		for _, im := range indexed {
			if tokensToRemove <= 0 {
				break
			}
			if im.index >= len(messages)-w.config.KeepRecentCount {
				continue
			}
			if im.priority == PriorityMedium && !removed[im.index] {
				removed[im.index] = true
				tokensToRemove -= w.estimateMessageTokens(im.msg)
			}
		}
	}

	// Build result
	result := make([]Message, 0, len(messages)-len(removed))
	for i, msg := range messages {
		if !removed[i] {
			result = append(result, msg)
		}
	}

	return result
}

// pruneWithSummary creates a summary of old messages
func (w *WindowManager) pruneWithSummary(messages []Message, systemPrompt string) []Message {
	stats := w.CalculateStats(messages, systemPrompt)
	targetTokens := w.limits.MaxTokens - w.limits.ReservedTokens

	if stats.TotalTokens <= targetTokens {
		return messages
	}

	// Keep recent messages, summarize the rest
	if len(messages) <= w.config.KeepRecentCount {
		return messages
	}

	oldMessages := messages[:len(messages)-w.config.KeepRecentCount]
	recentMessages := messages[len(messages)-w.config.KeepRecentCount:]

	// Create a summary message
	summary := createSummary(oldMessages)

	result := make([]Message, 0, len(recentMessages)+1)
	result = append(result, summary)
	result = append(result, recentMessages...)

	return result
}

// pruneHybrid combines priority-based removal with summarization
func (w *WindowManager) pruneHybrid(messages []Message, priorities []MessagePriority, systemPrompt string) []Message {
	// First, try priority-based removal
	result := w.pruneLowPriority(messages, priorities, systemPrompt)

	// If still too large, use summarization
	stats := w.CalculateStats(result, systemPrompt)
	targetTokens := w.limits.MaxTokens - w.limits.ReservedTokens

	if stats.TotalTokens > targetTokens {
		result = w.pruneWithSummary(result, systemPrompt)
	}

	return result
}

// createSummary creates a summary message from old messages
func createSummary(messages []Message) Message {
	var sb strings.Builder
	sb.WriteString("[Previous conversation summary]\n")

	userMsgs := 0
	assistantMsgs := 0
	toolCalls := 0

	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleUser:
			userMsgs++
		case llm.RoleAssistant:
			assistantMsgs++
			toolCalls += len(msg.ToolCalls)
		}
	}

	sb.WriteString(fmt.Sprintf("- %d user messages, %d assistant responses\n", userMsgs, assistantMsgs))
	if toolCalls > 0 {
		sb.WriteString(fmt.Sprintf("- %d tool calls executed\n", toolCalls))
	}

	// Extract key topics (simple heuristic)
	topics := extractTopics(messages)
	if len(topics) > 0 {
		sb.WriteString("- Topics discussed: ")
		sb.WriteString(strings.Join(topics, ", "))
		sb.WriteString("\n")
	}

	return Message{
		Message: llm.Message{
			Role:    llm.RoleSystem,
			Content: sb.String(),
		},
	}
}

// extractTopics extracts key topics from messages (simple heuristic)
func extractTopics(messages []Message) []string {
	topics := make(map[string]int)

	keywords := []string{
		"file", "function", "error", "test", "bug", "feature",
		"code", "refactor", "fix", "add", "remove", "update",
		"database", "api", "ui", "config", "deploy", "build",
	}

	for _, msg := range messages {
		content := strings.ToLower(msg.Content)
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				topics[kw]++
			}
		}
	}

	// Return top topics
	result := make([]string, 0)
	for topic, count := range topics {
		if count >= 2 {
			result = append(result, topic)
		}
	}

	if len(result) > 5 {
		result = result[:5]
	}

	return result
}

// FormatStats formats context stats for display
func FormatStats(stats ContextStats) string {
	var sb strings.Builder

	usedK := float64(stats.TotalTokens) / 1000
	maxK := float64(stats.MaxTokens) / 1000
	percent := stats.UsagePercent * 100

	sb.WriteString(fmt.Sprintf("Context: %.1fK/%.1fK tokens (%.0f%%)", usedK, maxK, percent))

	if stats.IsNearLimit {
		sb.WriteString(" ⚠️ Near limit")
	}

	return sb.String()
}

// FormatStatsCompact formats context stats for status bar
func FormatStatsCompact(stats ContextStats) string {
	usedK := float64(stats.TotalTokens) / 1000
	maxK := float64(stats.MaxTokens) / 1000
	return fmt.Sprintf("%.1fK/%.1fK", usedK, maxK)
}
