package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mateus/flow-cli/internal/llm"
)

// Message extends llm.Message with metadata
type Message struct {
	llm.Message
	Timestamp time.Time         `json:"timestamp"`
	ToolCalls []ToolCall        `json:"tool_calls,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Result    string         `json:"result,omitempty"`
	Success   bool           `json:"success"`
	Duration  time.Duration  `json:"duration,omitempty"`
}

// Conversation represents a chat session
type Conversation struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Messages   []Message `json:"messages"`
	Model      string    `json:"model"`
	ProjectDir string    `json:"project_dir"`
}

// Manager handles conversation history and context
type Manager struct {
	mu            sync.RWMutex
	conversation  *Conversation
	maxMessages   int
	systemPrompt  string
	windowManager *WindowManager
	priorities    []MessagePriority
}

// NewManager creates a new context manager
func NewManager(systemPrompt string, maxMessages int) *Manager {
	if maxMessages <= 0 {
		maxMessages = 100
	}

	return &Manager{
		conversation: &Conversation{
			ID:        generateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Messages:  make([]Message, 0),
		},
		maxMessages:  maxMessages,
		systemPrompt: systemPrompt,
	}
}

// generateID creates a simple unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// SetModel sets the model for this conversation
func (m *Manager) SetModel(model string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conversation.Model = model
}

// SetProjectDir sets the project directory for this conversation
func (m *Manager) SetProjectDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conversation.ProjectDir = dir
}

// AddUserMessage adds a user message to the conversation
func (m *Manager) AddUserMessage(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := Message{
		Message: llm.Message{
			Role:    llm.RoleUser,
			Content: content,
		},
		Timestamp: time.Now(),
	}

	m.conversation.Messages = append(m.conversation.Messages, msg)
	m.conversation.UpdatedAt = time.Now()

	// Set title from first message if not set
	if m.conversation.Title == "" {
		m.conversation.Title = truncateTitle(content, 50)
	}

	m.trimIfNeeded()
}

// AddAssistantMessage adds an assistant message to the conversation
func (m *Manager) AddAssistantMessage(content string, toolCalls []ToolCall) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := Message{
		Message: llm.Message{
			Role:    llm.RoleAssistant,
			Content: content,
		},
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
	}

	m.conversation.Messages = append(m.conversation.Messages, msg)
	m.conversation.UpdatedAt = time.Now()

	m.trimIfNeeded()
}

// AddSystemMessage adds a system message
func (m *Manager) AddSystemMessage(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := Message{
		Message: llm.Message{
			Role:    llm.RoleSystem,
			Content: content,
		},
		Timestamp: time.Now(),
	}

	m.conversation.Messages = append(m.conversation.Messages, msg)
	m.conversation.UpdatedAt = time.Now()
}

// GetMessages returns all messages for LLM context
func (m *Manager) GetMessages() []llm.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := make([]llm.Message, 0, len(m.conversation.Messages)+1)

	// Add system prompt first
	if m.systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: m.systemPrompt,
		})
	}

	// Add conversation messages
	for _, msg := range m.conversation.Messages {
		messages = append(messages, msg.Message)
	}

	return messages
}

// GetConversation returns the current conversation
func (m *Manager) GetConversation() *Conversation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversation
}

// MessageCount returns the number of messages
func (m *Manager) MessageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conversation.Messages)
}

// Clear resets the conversation
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.conversation = &Conversation{
		ID:         generateID(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Messages:   make([]Message, 0),
		Model:      m.conversation.Model,
		ProjectDir: m.conversation.ProjectDir,
	}
}

// trimIfNeeded removes old messages if we exceed max
func (m *Manager) trimIfNeeded() {
	if len(m.conversation.Messages) > m.maxMessages {
		// Keep the most recent messages
		excess := len(m.conversation.Messages) - m.maxMessages
		m.conversation.Messages = m.conversation.Messages[excess:]
	}
}

// Save persists the conversation to disk
func (m *Manager) Save(dir string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filename := filepath.Join(dir, fmt.Sprintf("conversation_%s.json", m.conversation.ID))

	data, err := json.MarshalIndent(m.conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load loads a conversation from disk
func (m *Manager) Load(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	m.conversation = &conv
	return nil
}

// GetLastUserMessage returns the last user message
func (m *Manager) GetLastUserMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := len(m.conversation.Messages) - 1; i >= 0; i-- {
		if m.conversation.Messages[i].Role == llm.RoleUser {
			return m.conversation.Messages[i].Content
		}
	}
	return ""
}

// GetLastAssistantMessage returns the last assistant message
func (m *Manager) GetLastAssistantMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := len(m.conversation.Messages) - 1; i >= 0; i-- {
		if m.conversation.Messages[i].Role == llm.RoleAssistant {
			return m.conversation.Messages[i].Content
		}
	}
	return ""
}

// truncateTitle truncates a string for use as a title
func truncateTitle(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")

	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}

// EstimateTokens provides a rough token count estimate
func (m *Manager) EstimateTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, msg := range m.conversation.Messages {
		// Rough estimate: ~4 chars per token
		total += len(msg.Content) / 4
	}

	if m.systemPrompt != "" {
		total += len(m.systemPrompt) / 4
	}

	return total
}

// SetWindowManager configures context window management
func (m *Manager) SetWindowManager(model string, strategy PruningStrategy) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.windowManager = NewWindowManager(WindowConfig{
		Model:            model,
		Strategy:         strategy,
		KeepSystemPrompt: true,
		KeepRecentCount:  4,
	})
}

// GetContextStats returns current context window statistics
func (m *Manager) GetContextStats() ContextStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.windowManager == nil {
		// Return basic stats without window manager
		tokens := 0
		for _, msg := range m.conversation.Messages {
			tokens += llm.EstimateTokens(msg.Content) + 4
		}
		if m.systemPrompt != "" {
			tokens += llm.EstimateTokens(m.systemPrompt) + 4
		}
		return ContextStats{
			TotalTokens:  tokens,
			MaxTokens:    4096, // Default
			MessageCount: len(m.conversation.Messages),
		}
	}

	return m.windowManager.CalculateStats(m.conversation.Messages, m.systemPrompt)
}

// SetMessagePriority sets the priority of a specific message
func (m *Manager) SetMessagePriority(index int, priority MessagePriority) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure priorities slice is large enough
	for len(m.priorities) <= index {
		m.priorities = append(m.priorities, PriorityMedium)
	}
	m.priorities[index] = priority
}

// SetLastMessagePriority sets the priority of the most recent message
func (m *Manager) SetLastMessagePriority(priority MessagePriority) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.conversation.Messages) == 0 {
		return
	}

	index := len(m.conversation.Messages) - 1
	for len(m.priorities) <= index {
		m.priorities = append(m.priorities, PriorityMedium)
	}
	m.priorities[index] = priority
}

// PruneIfNeeded prunes messages if context window is near capacity
func (m *Manager) PruneIfNeeded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.windowManager == nil {
		return false
	}

	if !m.windowManager.NeedsPruning(m.conversation.Messages, m.systemPrompt) {
		return false
	}

	// Ensure priorities match messages
	for len(m.priorities) < len(m.conversation.Messages) {
		m.priorities = append(m.priorities, PriorityMedium)
	}

	// Prune messages
	prunedMessages := m.windowManager.PruneMessages(m.conversation.Messages, m.priorities, m.systemPrompt)

	// Update conversation
	m.conversation.Messages = prunedMessages
	m.conversation.UpdatedAt = time.Now()

	// Trim priorities to match
	if len(m.priorities) > len(prunedMessages) {
		m.priorities = m.priorities[len(m.priorities)-len(prunedMessages):]
	}

	return true
}

// IsNearContextLimit checks if we're approaching context limits
func (m *Manager) IsNearContextLimit() bool {
	stats := m.GetContextStats()
	return stats.IsNearLimit
}

// GetContextStatsFormatted returns formatted context stats for display
func (m *Manager) GetContextStatsFormatted() string {
	stats := m.GetContextStats()
	return FormatStats(stats)
}

// GetContextStatsCompact returns compact context stats for status bar
func (m *Manager) GetContextStatsCompact() string {
	stats := m.GetContextStats()
	return FormatStatsCompact(stats)
}

// GetID returns the conversation ID
func (m *Manager) GetID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversation.ID
}

// GetTitle returns the conversation title
func (m *Manager) GetTitle() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversation.Title
}

// SessionInfo contains summary information about a saved session
type SessionInfo struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	Model        string    `json:"model"`
	ProjectDir   string    `json:"project_dir"`
	FilePath     string    `json:"file_path"`
}

// GetSessionsDir returns the default sessions directory
func GetSessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".flow", "sessions"), nil
}

// ListSessions returns all saved sessions
func ListSessions(dir string) ([]SessionInfo, error) {
	if dir == "" {
		var err error
		dir, err = GetSessionsDir()
		if err != nil {
			return nil, err
		}
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var conv Conversation
		if err := json.Unmarshal(data, &conv); err != nil {
			continue
		}

		sessions = append(sessions, SessionInfo{
			ID:           conv.ID,
			Title:        conv.Title,
			CreatedAt:    conv.CreatedAt,
			UpdatedAt:    conv.UpdatedAt,
			MessageCount: len(conv.Messages),
			Model:        conv.Model,
			ProjectDir:   conv.ProjectDir,
			FilePath:     filePath,
		})
	}

	// Sort by updated time (most recent first)
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].UpdatedAt.After(sessions[i].UpdatedAt) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	return sessions, nil
}

// DeleteSession removes a saved session
func DeleteSession(filePath string) error {
	return os.Remove(filePath)
}

// SaveToDefaultDir saves the conversation to the default sessions directory
func (m *Manager) SaveToDefaultDir() error {
	dir, err := GetSessionsDir()
	if err != nil {
		return err
	}
	return m.Save(dir)
}
