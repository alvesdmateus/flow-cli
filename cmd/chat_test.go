package cmd

import (
	"testing"
)

func TestChatCmd_Structure(t *testing.T) {
	if chatCmd == nil {
		t.Fatal("chatCmd is nil")
	}

	if chatCmd.Use != "chat" {
		t.Errorf("chatCmd.Use = %q, want %q", chatCmd.Use, "chat")
	}

	if chatCmd.Short == "" {
		t.Error("chatCmd.Short should not be empty")
	}

	if chatCmd.Long == "" {
		t.Error("chatCmd.Long should not be empty")
	}

	if chatCmd.RunE == nil {
		t.Error("chatCmd.RunE should not be nil")
	}
}

func TestChatCmd_ParentIsRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "chat" {
			found = true
			break
		}
	}
	if !found {
		t.Error("chatCmd should be a subcommand of rootCmd")
	}
}

func TestConsoleHandler_Interface(t *testing.T) {
	handler := &ConsoleHandler{}

	// Test that handler implements expected methods
	// These should not panic

	handler.OnStreamStart()
	handler.OnStreamChunk("test chunk")
	handler.OnStreamEnd()
	handler.OnToolStart("test_tool", "test description")
	handler.OnToolEnd("test_tool", true, "success")
	handler.OnToolEnd("test_tool", false, "error message")
	handler.OnThinking("thinking message")
	handler.OnError(nil)
}

func TestConsoleHandler_OnStreamChunk(t *testing.T) {
	handler := &ConsoleHandler{}

	// Start stream to initialize spinner
	handler.OnStreamStart()

	// First chunk should stop spinner
	handler.OnStreamChunk("first")

	// Subsequent chunks should just print
	handler.OnStreamChunk(" second")
	handler.OnStreamChunk(" third")

	// End stream
	handler.OnStreamEnd()
}

func TestConsoleHandler_OnToolStart_StopsSpinner(t *testing.T) {
	handler := &ConsoleHandler{}

	// Start stream (creates spinner)
	handler.OnStreamStart()

	// Tool start should stop the spinner
	handler.OnToolStart("test_tool", "Running test")

	// Should complete without panic
	handler.OnToolEnd("test_tool", true, "")
}

func TestConsoleHandler_OnError_WithSpinner(t *testing.T) {
	handler := &ConsoleHandler{}

	// Start stream (creates spinner)
	handler.OnStreamStart()

	// Error should stop spinner with error message
	handler.OnError(nil)

	// Should complete without panic
}

func TestConsoleHandler_OnThinking_WithSpinner(t *testing.T) {
	handler := &ConsoleHandler{}

	// Start stream (creates spinner)
	handler.OnStreamStart()

	// Thinking should update spinner message
	handler.OnThinking("Processing...")

	// Stop the stream
	handler.OnStreamEnd()
}

func TestConsoleHandler_OnThinking_WithoutSpinner(t *testing.T) {
	handler := &ConsoleHandler{}

	// Thinking without spinner should just print
	handler.OnThinking("Processing...")

	// Should complete without panic
}

func TestPrintWelcome(t *testing.T) {
	// Should complete without panic
	printWelcome("test-model", "/test/dir")
}

func TestPrintHelp(t *testing.T) {
	// Should complete without panic
	printHelp()
}

func TestHandleCommand_Help(t *testing.T) {
	// Create a mock agent (nil is ok for this test)
	// handleCommand should return true for /help
	result := handleCommand("/help", nil)
	if !result {
		t.Error("handleCommand(/help) should return true")
	}

	result = handleCommand("/h", nil)
	if !result {
		t.Error("handleCommand(/h) should return true")
	}
}

func TestHandleCommand_Unknown(t *testing.T) {
	// Unknown commands should return true (handled by printing error)
	result := handleCommand("/unknown", nil)
	if !result {
		t.Error("handleCommand(/unknown) should return true")
	}
}

func TestHandleCommand_Quit(t *testing.T) {
	// Quit commands should return false to signal exit
	result := handleCommand("/quit", nil)
	if result {
		t.Error("handleCommand(/quit) should return false")
	}

	result = handleCommand("/exit", nil)
	if result {
		t.Error("handleCommand(/exit) should return false")
	}

	result = handleCommand("/q", nil)
	if result {
		t.Error("handleCommand(/q) should return false")
	}
}

func TestHandleCommand_Empty(t *testing.T) {
	// Empty command should return true
	result := handleCommand("", nil)
	if !result {
		t.Error("handleCommand('') should return true")
	}
}
