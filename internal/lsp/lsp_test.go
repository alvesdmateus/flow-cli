package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)

	if server == nil {
		t.Fatal("Expected server to be created")
	}
	if server.handler == nil {
		t.Error("Expected handler to be set")
	}
}

func TestServerSetIO(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)

	input := &bytes.Buffer{}
	output := &bytes.Buffer{}

	server.SetIO(input, output)

	if server.input != input {
		t.Error("Expected input to be set")
	}
	if server.output != output {
		t.Error("Expected output to be set")
	}
}

func TestServerReadWriteMessage(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)

	output := &bytes.Buffer{}
	server.SetIO(nil, output)

	id := json.RawMessage(`1`)
	msg := &Message{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "test",
	}

	err := server.writeMessage(msg)
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	written := output.String()
	if !strings.Contains(written, "Content-Length:") {
		t.Error("Expected Content-Length header")
	}
	if !strings.Contains(written, `"jsonrpc":"2.0"`) {
		t.Error("Expected jsonrpc field")
	}
}

func TestServerHandleInitialize(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)

	params := InitializeParams{
		RootURI: "file:///test/project",
	}
	paramsJSON, _ := json.Marshal(params)

	id := json.RawMessage(`1`)
	msg := &Message{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "initialize",
		Params:  paramsJSON,
	}

	response := server.handleMessage(msg)

	if response == nil {
		t.Fatal("Expected response")
	}
	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	var result InitializeResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if result.ServerInfo == nil || result.ServerInfo.Name != "flow-lsp" {
		t.Error("Expected server info with name flow-lsp")
	}
}

func TestServerNotInitialized(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)

	id := json.RawMessage(`1`)
	msg := &Message{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "textDocument/completion",
		Params:  []byte(`{}`),
	}

	response := server.handleMessage(msg)

	if response == nil {
		t.Fatal("Expected error response")
	}
	if response.Error == nil {
		t.Error("Expected error for uninitialized server")
	}
	if response.Error.Code != ServerNotInitialized {
		t.Errorf("Expected ServerNotInitialized error code, got %d", response.Error.Code)
	}
}

func TestServerMethodNotFound(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)
	server.initialized = true

	id := json.RawMessage(`1`)
	msg := &Message{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "unknownMethod",
	}

	response := server.handleMessage(msg)

	if response == nil {
		t.Fatal("Expected error response")
	}
	if response.Error == nil {
		t.Error("Expected error for unknown method")
	}
	if response.Error.Code != MethodNotFound {
		t.Errorf("Expected MethodNotFound error code, got %d", response.Error.Code)
	}
}

func TestFlowHandlerDocumentLifecycle(t *testing.T) {
	handler := NewFlowHandler()

	uri := URI("file:///test/file.go")

	// Open document
	err := handler.TextDocumentDidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "package main\n\nfunc main() {}\n",
		},
	})
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	handler.mu.RLock()
	doc, ok := handler.documents[uri]
	handler.mu.RUnlock()

	if !ok {
		t.Fatal("Document should be in documents map")
	}
	if doc.LanguageID != "go" {
		t.Errorf("Expected language ID 'go', got '%s'", doc.LanguageID)
	}
	// Note: "package main\n\nfunc main() {}\n" splits to 4 lines (trailing newline creates empty line)
	if len(doc.Lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(doc.Lines))
	}

	// Change document
	err = handler.TextDocumentDidChange(DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to change document: %v", err)
	}

	handler.mu.RLock()
	doc = handler.documents[uri]
	handler.mu.RUnlock()

	if doc.Version != 2 {
		t.Errorf("Expected version 2, got %d", doc.Version)
	}
	// "...\n}\n" splits to 6 lines (trailing newline creates empty line)
	if len(doc.Lines) != 6 {
		t.Errorf("Expected 6 lines, got %d", len(doc.Lines))
	}

	// Close document
	err = handler.TextDocumentDidClose(DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("Failed to close document: %v", err)
	}

	handler.mu.RLock()
	_, ok = handler.documents[uri]
	handler.mu.RUnlock()

	if ok {
		t.Error("Document should be removed after close")
	}
}

func TestFlowHandlerCompletion(t *testing.T) {
	handler := NewFlowHandler()

	uri := URI("file:///test/file.go")
	handler.TextDocumentDidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "// @flow\n",
		},
	})

	result, err := handler.TextDocumentCompletion(CompletionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     Position{Line: 0, Character: 8},
		},
	})

	if err != nil {
		t.Fatalf("Failed to get completions: %v", err)
	}
	if result == nil {
		t.Fatal("Expected completion result")
	}

	// Should have flow completions
	found := false
	for _, item := range result.Items {
		if strings.Contains(item.Label, "@flow") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected @flow completion items")
	}
}

func TestFlowHandlerHover(t *testing.T) {
	handler := NewFlowHandler()

	uri := URI("file:///test/file.go")
	handler.TextDocumentDidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "func testFunction() {}\n",
		},
	})

	result, err := handler.TextDocumentHover(HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     Position{Line: 0, Character: 7},
		},
	})

	if err != nil {
		t.Fatalf("Failed to get hover: %v", err)
	}
	if result == nil {
		t.Fatal("Expected hover result")
	}
	if !strings.Contains(result.Contents.Value, "testFunction") {
		t.Error("Expected hover to contain word at position")
	}
}

func TestFlowHandlerCodeAction(t *testing.T) {
	handler := NewFlowHandler()

	uri := URI("file:///test/file.go")

	result, err := handler.TextDocumentCodeAction(CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 10},
		},
		Context: CodeActionContext{
			Diagnostics: []Diagnostic{},
		},
	})

	if err != nil {
		t.Fatalf("Failed to get code actions: %v", err)
	}

	// Should have flow code actions
	if len(result) < 3 {
		t.Errorf("Expected at least 3 code actions, got %d", len(result))
	}

	titles := make(map[string]bool)
	for _, action := range result {
		titles[action.Title] = true
	}

	expectedActions := []string{"Explain with Vibe", "Generate Tests with Vibe", "Refactor with Vibe"}
	for _, expected := range expectedActions {
		if !titles[expected] {
			t.Errorf("Expected code action '%s'", expected)
		}
	}
}

func TestFlowHandlerExecuteCommand(t *testing.T) {
	handler := NewFlowHandler()

	tests := []struct {
		command   string
		args      []interface{}
		expectErr bool
	}{
		{"flow.runPrompt", []interface{}{"test prompt"}, false},
		{"flow.explainCode", []interface{}{}, false},
		{"flow.generateTests", []interface{}{}, false},
		{"flow.fixError", []interface{}{}, false},
		{"flow.refactor", []interface{}{}, false},
		{"unknown.command", []interface{}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			_, err := handler.ExecuteCommand(ExecuteCommandParams{
				Command:   tt.command,
				Arguments: tt.args,
			})

			if tt.expectErr && err == nil {
				t.Error("Expected error")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestURIToPath(t *testing.T) {
	tests := []struct {
		uri      URI
		expected string
	}{
		{"file:///home/user/project/file.go", "/home/user/project/file.go"},
		{"file:///C:/Users/test/file.go", "C:/Users/test/file.go"},
		{"file://localhost/path/file.go", "localhost/path/file.go"},
	}

	for _, tt := range tests {
		t.Run(string(tt.uri), func(t *testing.T) {
			result := uriToPath(tt.uri)
			// Normalize for comparison
			result = strings.ReplaceAll(result, "\\", "/")
			expected := strings.ReplaceAll(tt.expected, "\\", "/")
			if result != expected {
				t.Errorf("uriToPath(%s) = %s, want %s", tt.uri, result, expected)
			}
		})
	}
}

func TestApplyChange(t *testing.T) {
	content := "line1\nline2\nline3\n"

	// Replace middle of line2
	result := applyChange(content, Range{
		Start: Position{Line: 1, Character: 0},
		End:   Position{Line: 1, Character: 5},
	}, "modified")

	expected := "line1\nmodified\nline3\n"
	if result != expected {
		t.Errorf("applyChange result = %q, want %q", result, expected)
	}
}

func TestGetWordAtPosition(t *testing.T) {
	doc := &Document{
		Lines: []string{"func testFunction() {}", "    return value"},
	}

	tests := []struct {
		pos      Position
		expected string
	}{
		{Position{Line: 0, Character: 7}, "testFunction"},
		{Position{Line: 1, Character: 12}, "value"},
		{Position{Line: 0, Character: 0}, "func"},
		{Position{Line: 0, Character: 4}, "func"},  // end of word still returns word
		{Position{Line: 0, Character: 5}, "testFunction"}, // start of next word
		{Position{Line: 5, Character: 0}, ""}, // out of range
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("line%d_char%d", tt.pos.Line, tt.pos.Character), func(t *testing.T) {
			result := getWordAtPosition(doc, tt.pos)
			if result != tt.expected {
				t.Errorf("getWordAtPosition at (%d,%d) = %q, want %q",
					tt.pos.Line, tt.pos.Character, result, tt.expected)
			}
		})
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		char     byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'_', true},
		{' ', false},
		{'.', false},
		{'(', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isWordChar(tt.char)
			if result != tt.expected {
				t.Errorf("isWordChar(%c) = %v, want %v", tt.char, result, tt.expected)
			}
		})
	}
}

func TestServerRun(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)

	// Create a simple initialize request
	initParams := InitializeParams{RootURI: "file:///test"}
	paramsJSON, _ := json.Marshal(initParams)
	initMsg := Message{
		JSONRPC: "2.0",
		ID:      func() *json.RawMessage { r := json.RawMessage(`1`); return &r }(),
		Method:  "initialize",
		Params:  paramsJSON,
	}
	initMsgJSON, _ := json.Marshal(initMsg)
	initReq := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initMsgJSON), initMsgJSON)

	input := strings.NewReader(initReq)
	output := &bytes.Buffer{}
	server.SetIO(input, output)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run server (will exit when input is exhausted)
	err := server.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		// EOF is expected when input is exhausted
	}

	// Check response was written
	response := output.String()
	if !strings.Contains(response, "Content-Length:") {
		t.Error("Expected response to be written")
	}
	if !strings.Contains(response, "flow-lsp") {
		t.Error("Expected response to contain server name")
	}
}

func TestServerNotifications(t *testing.T) {
	handler := NewFlowHandler()
	server := NewServer(handler)

	output := &bytes.Buffer{}
	server.SetIO(nil, output)

	// Test PublishDiagnostics
	err := server.PublishDiagnostics("file:///test.go", []Diagnostic{
		{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 10},
			},
			Message:  "test error",
			Severity: DiagnosticSeverityError,
		},
	})
	if err != nil {
		t.Errorf("PublishDiagnostics failed: %v", err)
	}

	if !strings.Contains(output.String(), "publishDiagnostics") {
		t.Error("Expected publishDiagnostics notification")
	}

	// Test ShowMessage
	output.Reset()
	err = server.ShowMessage(MessageTypeInfo, "Test message")
	if err != nil {
		t.Errorf("ShowMessage failed: %v", err)
	}

	if !strings.Contains(output.String(), "showMessage") {
		t.Error("Expected showMessage notification")
	}

	// Test LogMessage
	output.Reset()
	err = server.LogMessage(MessageTypeLog, "Log entry")
	if err != nil {
		t.Errorf("LogMessage failed: %v", err)
	}

	if !strings.Contains(output.String(), "logMessage") {
		t.Error("Expected logMessage notification")
	}
}

func TestFlowHandlerShutdown(t *testing.T) {
	handler := NewFlowHandler()

	// Add a document
	uri := URI("file:///test/file.go")
	handler.TextDocumentDidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "package main",
		},
	})

	// Shutdown should clear documents
	err := handler.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	handler.mu.RLock()
	docCount := len(handler.documents)
	handler.mu.RUnlock()

	if docCount != 0 {
		t.Errorf("Expected documents to be cleared, got %d", docCount)
	}
}

func TestIncrementalChange(t *testing.T) {
	handler := NewFlowHandler()

	uri := URI("file:///test/file.go")
	handler.TextDocumentDidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       "hello world",
		},
	})

	// Apply incremental change
	err := handler.TextDocumentDidChange(DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{
				Range: &Range{
					Start: Position{Line: 0, Character: 6},
					End:   Position{Line: 0, Character: 11},
				},
				Text: "vibe",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed incremental change: %v", err)
	}

	handler.mu.RLock()
	doc := handler.documents[uri]
	handler.mu.RUnlock()

	if doc.Content != "hello vibe" {
		t.Errorf("Expected 'hello vibe', got '%s'", doc.Content)
	}
}
