package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ChatState represents the current state of the chat
type ChatState int

const (
	StateInput ChatState = iota
	StateWaiting
	StateStreaming
	StateToolExec
)

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string
	Content string
	IsError bool
}

// ChatModel is the bubbletea model for the chat interface
type ChatModel struct {
	// UI Components
	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	// State
	state       ChatState
	messages    []ChatMessage
	streamBuf   strings.Builder
	currentTool string

	// Dimensions
	width  int
	height int

	// Callbacks
	onSubmit func(input string)
	onQuit   func()

	// Styling
	userStyle      lipgloss.Style
	assistantStyle lipgloss.Style
	systemStyle    lipgloss.Style
	errorStyle     lipgloss.Style
	toolStyle      lipgloss.Style
	headerStyle    lipgloss.Style
	helpStyle      lipgloss.Style
}

// NewChatModel creates a new chat model
func NewChatModel() *ChatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Shift+Enter for newline)"
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetKeys("shift+enter")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(80, 20)

	return &ChatModel{
		textarea: ta,
		viewport: vp,
		spinner:  sp,
		state:    StateInput,
		messages: make([]ChatMessage, 0),

		userStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		assistantStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")),
		systemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		toolStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("236")).
			Padding(0, 1),
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// SetCallbacks sets the callback functions
func (m *ChatModel) SetCallbacks(onSubmit func(string), onQuit func()) {
	m.onSubmit = onSubmit
	m.onQuit = onQuit
}

// Init implements tea.Model
func (m *ChatModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// Update implements tea.Model
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.onQuit != nil {
				m.onQuit()
			}
			return m, tea.Quit

		case "ctrl+l":
			// Clear screen
			m.messages = nil
			m.updateViewport()
			return m, nil

		case "esc":
			if m.state == StateInput {
				m.textarea.Reset()
			}
			return m, nil

		case "enter":
			if m.state == StateInput && m.textarea.Value() != "" {
				input := strings.TrimSpace(m.textarea.Value())
				if input != "" {
					m.AddMessage("user", input)
					m.textarea.Reset()
					m.state = StateWaiting

					if m.onSubmit != nil {
						m.onSubmit(input)
					}
				}
			}
			return m, nil

		case "ctrl+h":
			// Show help
			m.AddMessage("system", m.getHelpText())
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case spinner.TickMsg:
		if m.state == StateWaiting || m.state == StateStreaming || m.state == StateToolExec {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case StreamChunkMsg:
		m.streamBuf.WriteString(msg.Content)
		m.updateStreamingMessage()
		return m, nil

	case StreamEndMsg:
		if m.streamBuf.Len() > 0 {
			m.finalizeStreamingMessage()
		}
		m.state = StateInput
		return m, nil

	case ToolStartMsg:
		m.state = StateToolExec
		m.currentTool = msg.Name
		m.AddMessage("tool", fmt.Sprintf("🔧 Executing: %s", msg.Name))
		return m, nil

	case ToolEndMsg:
		icon := "✓"
		if !msg.Success {
			icon = "✗"
		}
		m.AddMessage("tool", fmt.Sprintf("%s %s completed", icon, msg.Name))
		m.state = StateStreaming
		return m, nil

	case ErrorMsg:
		m.AddMessage("error", msg.Error.Error())
		m.state = StateInput
		return m, nil
	}

	// Update textarea when in input state
	if m.state == StateInput {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *ChatModel) View() string {
	var b strings.Builder

	// Header
	header := m.headerStyle.Render(" vibe-cli ")
	statusInfo := m.getStatusInfo()
	headerLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		header,
		"  ",
		m.helpStyle.Render(statusInfo),
	)
	b.WriteString(headerLine)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Chat viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Input area or status
	switch m.state {
	case StateInput:
		b.WriteString(m.textarea.View())
	case StateWaiting:
		b.WriteString(fmt.Sprintf("%s Thinking...", m.spinner.View()))
	case StateStreaming:
		b.WriteString(fmt.Sprintf("%s Responding...", m.spinner.View()))
	case StateToolExec:
		b.WriteString(fmt.Sprintf("%s Executing %s...", m.spinner.View(), m.currentTool))
	}

	b.WriteString("\n")

	// Help line
	help := m.helpStyle.Render("Enter: send • Shift+Enter: newline • Ctrl+L: clear • Ctrl+H: help • Ctrl+C: quit")
	b.WriteString(help)

	return b.String()
}

// AddMessage adds a message to the chat
func (m *ChatModel) AddMessage(role, content string) {
	m.messages = append(m.messages, ChatMessage{
		Role:    role,
		Content: content,
		IsError: role == "error",
	})
	m.updateViewport()
}

// StartStreaming begins streaming mode
func (m *ChatModel) StartStreaming() {
	m.state = StateStreaming
	m.streamBuf.Reset()
}

// AppendStream adds content to the current streaming message
func (m *ChatModel) AppendStream(content string) {
	m.streamBuf.WriteString(content)
	m.updateStreamingMessage()
}

// EndStreaming finalizes the streaming message
func (m *ChatModel) EndStreaming() {
	m.finalizeStreamingMessage()
	m.state = StateInput
}

// SetError sets an error state
func (m *ChatModel) SetError(err error) {
	m.AddMessage("error", err.Error())
	m.state = StateInput
}

// Reset returns to input state
func (m *ChatModel) Reset() {
	m.state = StateInput
}

// resize adjusts component sizes
func (m *ChatModel) resize() {
	headerHeight := 2
	inputHeight := 5
	helpHeight := 1
	separators := 2

	viewportHeight := m.height - headerHeight - inputHeight - helpHeight - separators
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	m.viewport.Width = m.width
	m.viewport.Height = viewportHeight
	m.textarea.SetWidth(m.width - 2)
}

// updateViewport updates the viewport content
func (m *ChatModel) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		prefix := ""
		style := m.assistantStyle

		switch msg.Role {
		case "user":
			prefix = "You: "
			style = m.userStyle
		case "assistant":
			prefix = "Assistant: "
			style = m.assistantStyle
		case "system":
			prefix = ""
			style = m.systemStyle
		case "tool":
			prefix = ""
			style = m.toolStyle
		case "error":
			prefix = "Error: "
			style = m.errorStyle
		}

		lines := strings.Split(msg.Content, "\n")
		for i, line := range lines {
			if i == 0 && prefix != "" {
				content.WriteString(style.Render(prefix + line))
			} else {
				content.WriteString(style.Render(line))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// updateStreamingMessage updates the display during streaming
func (m *ChatModel) updateStreamingMessage() {
	// Update the last message or add a new one
	streamContent := m.streamBuf.String()

	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "streaming" {
		m.messages[len(m.messages)-1].Content = streamContent
	} else {
		m.messages = append(m.messages, ChatMessage{
			Role:    "streaming",
			Content: streamContent,
		})
	}

	m.updateViewport()
}

// finalizeStreamingMessage converts streaming message to assistant message
func (m *ChatModel) finalizeStreamingMessage() {
	content := m.streamBuf.String()
	m.streamBuf.Reset()

	// Replace streaming message with assistant message
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "streaming" {
		m.messages[len(m.messages)-1] = ChatMessage{
			Role:    "assistant",
			Content: content,
		}
	} else if content != "" {
		m.messages = append(m.messages, ChatMessage{
			Role:    "assistant",
			Content: content,
		})
	}

	m.updateViewport()
}

// getStatusInfo returns status information
func (m *ChatModel) getStatusInfo() string {
	return fmt.Sprintf("Messages: %d", len(m.messages))
}

// getHelpText returns the help text
func (m *ChatModel) getHelpText() string {
	return `
╭─────────────────────────────────────────────────────────╮
│                    vibe-cli Help                        │
├─────────────────────────────────────────────────────────┤
│ Commands:                                               │
│   /clear    - Clear conversation history                │
│   /model    - Change the current model                  │
│   /help     - Show this help message                    │
│   /quit     - Exit the chat                             │
│                                                         │
│ Keyboard Shortcuts:                                     │
│   Enter       - Send message                            │
│   Shift+Enter - New line in message                     │
│   Ctrl+L      - Clear screen                            │
│   Ctrl+H      - Show this help                          │
│   Ctrl+C      - Quit                                    │
│   Esc         - Clear current input                     │
│                                                         │
│ Tips:                                                   │
│   • Be specific about what you want to accomplish       │
│   • The AI will ask for permission before changes       │
│   • Use /clear to start a fresh conversation            │
╰─────────────────────────────────────────────────────────╯`
}

// Message types for tea.Cmd

// StreamChunkMsg represents a streaming chunk
type StreamChunkMsg struct {
	Content string
}

// StreamEndMsg signals end of streaming
type StreamEndMsg struct{}

// ToolStartMsg signals tool execution start
type ToolStartMsg struct {
	Name        string
	Description string
}

// ToolEndMsg signals tool execution end
type ToolEndMsg struct {
	Name    string
	Success bool
	Result  string
}

// ErrorMsg represents an error
type ErrorMsg struct {
	Error error
}

// SendStreamChunk creates a command to send a stream chunk
func SendStreamChunk(content string) tea.Cmd {
	return func() tea.Msg {
		return StreamChunkMsg{Content: content}
	}
}

// SendStreamEnd creates a command to signal stream end
func SendStreamEnd() tea.Cmd {
	return func() tea.Msg {
		return StreamEndMsg{}
	}
}

// SendToolStart creates a command to signal tool start
func SendToolStart(name, desc string) tea.Cmd {
	return func() tea.Msg {
		return ToolStartMsg{Name: name, Description: desc}
	}
}

// SendToolEnd creates a command to signal tool end
func SendToolEnd(name string, success bool, result string) tea.Cmd {
	return func() tea.Msg {
		return ToolEndMsg{Name: name, Success: success, Result: result}
	}
}

// SendError creates a command to send an error
func SendError(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Error: err}
	}
}
