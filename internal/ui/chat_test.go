package ui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewChatModel(t *testing.T) {
	m := NewChatModel()

	if m == nil {
		t.Fatal("NewChatModel() returned nil")
	}

	if m.state != StateInput {
		t.Errorf("initial state = %v, want StateInput", m.state)
	}

	if m.messages == nil {
		t.Error("messages should be initialized")
	}

	if len(m.messages) != 0 {
		t.Errorf("initial messages length = %d, want 0", len(m.messages))
	}
}

func TestChatModel_SetCallbacks(t *testing.T) {
	m := NewChatModel()

	submitCalled := false
	quitCalled := false

	m.SetCallbacks(
		func(input string) { submitCalled = true },
		func() { quitCalled = true },
	)

	if m.onSubmit == nil {
		t.Error("onSubmit callback should be set")
	}

	if m.onQuit == nil {
		t.Error("onQuit callback should be set")
	}

	// Verify callbacks work
	m.onSubmit("test")
	if !submitCalled {
		t.Error("onSubmit callback was not called")
	}

	m.onQuit()
	if !quitCalled {
		t.Error("onQuit callback was not called")
	}
}

func TestChatModel_AddMessage(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	m.AddMessage("user", "Hello")

	if len(m.messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(m.messages))
	}

	msg := m.messages[0]
	if msg.Role != "user" {
		t.Errorf("message role = %q, want %q", msg.Role, "user")
	}

	if msg.Content != "Hello" {
		t.Errorf("message content = %q, want %q", msg.Content, "Hello")
	}

	if msg.IsError {
		t.Error("message should not be marked as error")
	}
}

func TestChatModel_AddMessage_Error(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	m.AddMessage("error", "Something went wrong")

	if len(m.messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(m.messages))
	}

	msg := m.messages[0]
	if !msg.IsError {
		t.Error("error message should be marked as error")
	}
}

func TestChatModel_AddMessage_MultipleRoles(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	roles := []string{"user", "assistant", "system", "tool", "error"}

	for _, role := range roles {
		m.AddMessage(role, "content for "+role)
	}

	if len(m.messages) != len(roles) {
		t.Errorf("messages length = %d, want %d", len(m.messages), len(roles))
	}

	for i, role := range roles {
		if m.messages[i].Role != role {
			t.Errorf("message[%d].Role = %q, want %q", i, m.messages[i].Role, role)
		}
	}
}

func TestChatModel_StartStreaming(t *testing.T) {
	m := NewChatModel()

	m.StartStreaming()

	if m.state != StateStreaming {
		t.Errorf("state = %v, want StateStreaming", m.state)
	}
}

func TestChatModel_AppendStream(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	m.StartStreaming()
	m.AppendStream("Hello ")
	m.AppendStream("World")

	content := m.streamBuf.String()
	if content != "Hello World" {
		t.Errorf("streamBuf = %q, want %q", content, "Hello World")
	}
}

func TestChatModel_EndStreaming(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	m.StartStreaming()
	m.AppendStream("Complete response")
	m.EndStreaming()

	if m.state != StateInput {
		t.Errorf("state = %v, want StateInput", m.state)
	}

	// Should have converted streaming message to assistant message
	if len(m.messages) == 0 {
		t.Fatal("messages should contain the finalized message")
	}

	lastMsg := m.messages[len(m.messages)-1]
	if lastMsg.Role != "assistant" {
		t.Errorf("last message role = %q, want %q", lastMsg.Role, "assistant")
	}
}

func TestChatModel_SetError(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	testErr := errors.New("test error")
	m.SetError(testErr)

	if m.state != StateInput {
		t.Errorf("state = %v, want StateInput", m.state)
	}

	if len(m.messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(m.messages))
	}

	msg := m.messages[0]
	if msg.Role != "error" {
		t.Errorf("message role = %q, want %q", msg.Role, "error")
	}
}

func TestChatModel_Reset(t *testing.T) {
	m := NewChatModel()
	m.state = StateWaiting

	m.Reset()

	if m.state != StateInput {
		t.Errorf("state = %v, want StateInput", m.state)
	}
}

func TestChatModel_Init(t *testing.T) {
	m := NewChatModel()

	cmd := m.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestChatModel_View(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	view := m.View()

	if view == "" {
		t.Error("View() returned empty string")
	}

	// View should contain the header
	if len(view) < 10 {
		t.Error("View() output is too short")
	}
}

func TestChatModel_View_AllStates(t *testing.T) {
	states := []ChatState{StateInput, StateWaiting, StateStreaming, StateToolExec}

	for _, state := range states {
		t.Run(stateName(state), func(t *testing.T) {
			m := NewChatModel()
			m.width = 80
			m.height = 24
			m.state = state
			m.currentTool = "test_tool"

			view := m.View()

			if view == "" {
				t.Error("View() returned empty string")
			}
		})
	}
}

func stateName(s ChatState) string {
	switch s {
	case StateInput:
		return "StateInput"
	case StateWaiting:
		return "StateWaiting"
	case StateStreaming:
		return "StateStreaming"
	case StateToolExec:
		return "StateToolExec"
	default:
		return "Unknown"
	}
}

func TestChatModel_Update_WindowResize(t *testing.T) {
	m := NewChatModel()

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updated, _ := m.Update(msg)

	model := updated.(*ChatModel)
	if model.width != 100 {
		t.Errorf("width = %d, want 100", model.width)
	}

	if model.height != 30 {
		t.Errorf("height = %d, want 30", model.height)
	}
}

func TestChatModel_Update_StreamChunkMsg(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24
	m.state = StateStreaming

	msg := StreamChunkMsg{Content: "chunk"}
	updated, _ := m.Update(msg)

	model := updated.(*ChatModel)
	if model.streamBuf.String() != "chunk" {
		t.Errorf("streamBuf = %q, want %q", model.streamBuf.String(), "chunk")
	}
}

func TestChatModel_Update_StreamEndMsg(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24
	m.state = StateStreaming
	m.streamBuf.WriteString("response")

	msg := StreamEndMsg{}
	updated, _ := m.Update(msg)

	model := updated.(*ChatModel)
	if model.state != StateInput {
		t.Errorf("state = %v, want StateInput", model.state)
	}
}

func TestChatModel_Update_ToolStartMsg(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	msg := ToolStartMsg{Name: "read_file", Description: "Reading file"}
	updated, _ := m.Update(msg)

	model := updated.(*ChatModel)
	if model.state != StateToolExec {
		t.Errorf("state = %v, want StateToolExec", model.state)
	}

	if model.currentTool != "read_file" {
		t.Errorf("currentTool = %q, want %q", model.currentTool, "read_file")
	}
}

func TestChatModel_Update_ToolEndMsg(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24
	m.state = StateToolExec

	msg := ToolEndMsg{Name: "read_file", Success: true, Result: "content"}
	updated, _ := m.Update(msg)

	model := updated.(*ChatModel)
	if model.state != StateStreaming {
		t.Errorf("state = %v, want StateStreaming", model.state)
	}
}

func TestChatModel_Update_ErrorMsg(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24

	msg := ErrorMsg{Error: errors.New("test error")}
	updated, _ := m.Update(msg)

	model := updated.(*ChatModel)
	if model.state != StateInput {
		t.Errorf("state = %v, want StateInput", model.state)
	}

	if len(model.messages) == 0 {
		t.Fatal("messages should contain the error")
	}

	lastMsg := model.messages[len(model.messages)-1]
	if lastMsg.Role != "error" {
		t.Errorf("last message role = %q, want %q", lastMsg.Role, "error")
	}
}

func TestSendStreamChunk(t *testing.T) {
	cmd := SendStreamChunk("test content")

	if cmd == nil {
		t.Fatal("SendStreamChunk() returned nil")
	}

	msg := cmd()
	chunk, ok := msg.(StreamChunkMsg)
	if !ok {
		t.Fatalf("message type = %T, want StreamChunkMsg", msg)
	}

	if chunk.Content != "test content" {
		t.Errorf("Content = %q, want %q", chunk.Content, "test content")
	}
}

func TestSendStreamEnd(t *testing.T) {
	cmd := SendStreamEnd()

	if cmd == nil {
		t.Fatal("SendStreamEnd() returned nil")
	}

	msg := cmd()
	_, ok := msg.(StreamEndMsg)
	if !ok {
		t.Fatalf("message type = %T, want StreamEndMsg", msg)
	}
}

func TestSendToolStart(t *testing.T) {
	cmd := SendToolStart("read_file", "Reading file")

	if cmd == nil {
		t.Fatal("SendToolStart() returned nil")
	}

	msg := cmd()
	toolMsg, ok := msg.(ToolStartMsg)
	if !ok {
		t.Fatalf("message type = %T, want ToolStartMsg", msg)
	}

	if toolMsg.Name != "read_file" {
		t.Errorf("Name = %q, want %q", toolMsg.Name, "read_file")
	}

	if toolMsg.Description != "Reading file" {
		t.Errorf("Description = %q, want %q", toolMsg.Description, "Reading file")
	}
}

func TestSendToolEnd(t *testing.T) {
	cmd := SendToolEnd("read_file", true, "content")

	if cmd == nil {
		t.Fatal("SendToolEnd() returned nil")
	}

	msg := cmd()
	toolMsg, ok := msg.(ToolEndMsg)
	if !ok {
		t.Fatalf("message type = %T, want ToolEndMsg", msg)
	}

	if toolMsg.Name != "read_file" {
		t.Errorf("Name = %q, want %q", toolMsg.Name, "read_file")
	}

	if !toolMsg.Success {
		t.Error("Success should be true")
	}

	if toolMsg.Result != "content" {
		t.Errorf("Result = %q, want %q", toolMsg.Result, "content")
	}
}

func TestSendError(t *testing.T) {
	testErr := errors.New("test error")
	cmd := SendError(testErr)

	if cmd == nil {
		t.Fatal("SendError() returned nil")
	}

	msg := cmd()
	errMsg, ok := msg.(ErrorMsg)
	if !ok {
		t.Fatalf("message type = %T, want ErrorMsg", msg)
	}

	if errMsg.Error.Error() != "test error" {
		t.Errorf("Error = %q, want %q", errMsg.Error.Error(), "test error")
	}
}

func TestChatMessage_Fields(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "Hello",
		IsError: false,
	}

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}

	if msg.Content != "Hello" {
		t.Errorf("Content = %q, want %q", msg.Content, "Hello")
	}

	if msg.IsError {
		t.Error("IsError should be false")
	}
}

func TestChatState_Constants(t *testing.T) {
	// Verify state constants have distinct values
	states := map[ChatState]string{
		StateInput:     "StateInput",
		StateWaiting:   "StateWaiting",
		StateStreaming: "StateStreaming",
		StateToolExec:  "StateToolExec",
	}

	seen := make(map[ChatState]bool)
	for state := range states {
		if seen[state] {
			t.Errorf("duplicate state value: %v", state)
		}
		seen[state] = true
	}
}

func TestChatModel_resize(t *testing.T) {
	m := NewChatModel()
	m.width = 100
	m.height = 40

	m.resize()

	if m.viewport.Width != 100 {
		t.Errorf("viewport width = %d, want 100", m.viewport.Width)
	}

	// Viewport height should be calculated based on available space
	if m.viewport.Height <= 0 {
		t.Error("viewport height should be positive")
	}
}

func TestChatModel_resize_MinimumHeight(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 5 // Very small height

	m.resize()

	// Should enforce minimum viewport height
	if m.viewport.Height < 5 {
		t.Errorf("viewport height = %d, should be at least 5", m.viewport.Height)
	}
}

func TestChatModel_getStatusInfo(t *testing.T) {
	m := NewChatModel()
	m.AddMessage("user", "Hello")
	m.AddMessage("assistant", "Hi")

	status := m.getStatusInfo()

	if status == "" {
		t.Error("getStatusInfo() returned empty string")
	}
}

func TestChatModel_getHelpText(t *testing.T) {
	m := NewChatModel()

	help := m.getHelpText()

	if help == "" {
		t.Error("getHelpText() returned empty string")
	}

	// Help should contain key information
	if len(help) < 100 {
		t.Error("getHelpText() output seems too short")
	}
}

func TestChatModel_updateStreamingMessage_NewMessage(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24
	m.streamBuf.WriteString("streaming content")

	m.updateStreamingMessage()

	if len(m.messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(m.messages))
	}

	if m.messages[0].Role != "streaming" {
		t.Errorf("message role = %q, want %q", m.messages[0].Role, "streaming")
	}
}

func TestChatModel_updateStreamingMessage_UpdateExisting(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24
	m.messages = append(m.messages, ChatMessage{Role: "streaming", Content: "old"})
	m.streamBuf.WriteString("new content")

	m.updateStreamingMessage()

	if len(m.messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(m.messages))
	}

	if m.messages[0].Content != "new content" {
		t.Errorf("message content = %q, want %q", m.messages[0].Content, "new content")
	}
}

func TestChatModel_finalizeStreamingMessage(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24
	m.messages = append(m.messages, ChatMessage{Role: "streaming", Content: "temp"})
	m.streamBuf.WriteString("final content")

	m.finalizeStreamingMessage()

	if len(m.messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(m.messages))
	}

	if m.messages[0].Role != "assistant" {
		t.Errorf("message role = %q, want %q", m.messages[0].Role, "assistant")
	}

	if m.messages[0].Content != "final content" {
		t.Errorf("message content = %q, want %q", m.messages[0].Content, "final content")
	}
}

func TestChatModel_updateViewport(t *testing.T) {
	m := NewChatModel()
	m.width = 80
	m.height = 24
	m.resize()

	m.AddMessage("user", "Hello")
	m.AddMessage("assistant", "Hi there!")
	m.AddMessage("system", "System message")
	m.AddMessage("tool", "Tool output")
	m.AddMessage("error", "Error message")

	// updateViewport is called by AddMessage, so content should be set
	content := m.viewport.View()
	if content == "" {
		t.Error("viewport content should not be empty")
	}
}
