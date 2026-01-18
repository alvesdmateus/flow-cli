package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Server implements a Language Server Protocol server.
type Server struct {
	input   io.Reader
	output  io.Writer
	handler Handler
	running bool
	mu      sync.Mutex

	// State
	initialized    bool
	workspaceFolders []WorkspaceFolder
	capabilities   ServerCapabilities
}

// Handler processes LSP requests.
type Handler interface {
	Initialize(params InitializeParams) (*InitializeResult, error)
	Initialized(params InitializedParams) error
	Shutdown() error
	TextDocumentDidOpen(params DidOpenTextDocumentParams) error
	TextDocumentDidChange(params DidChangeTextDocumentParams) error
	TextDocumentDidClose(params DidCloseTextDocumentParams) error
	TextDocumentDidSave(params DidSaveTextDocumentParams) error
	TextDocumentCompletion(params CompletionParams) (*CompletionList, error)
	TextDocumentHover(params HoverParams) (*Hover, error)
	TextDocumentDefinition(params DefinitionParams) ([]Location, error)
	TextDocumentReferences(params ReferenceParams) ([]Location, error)
	TextDocumentCodeAction(params CodeActionParams) ([]CodeAction, error)
	TextDocumentFormatting(params DocumentFormattingParams) ([]TextEdit, error)
	ExecuteCommand(params ExecuteCommandParams) (interface{}, error)
}

// Message represents a JSON-RPC message.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError represents a JSON-RPC error.
type ResponseError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error codes
const (
	ParseError           = -32700
	InvalidRequest       = -32600
	MethodNotFound       = -32601
	InvalidParams        = -32602
	InternalError        = -32603
	ServerNotInitialized = -32002
	RequestCancelled     = -32800
)

// NewServer creates a new LSP server.
func NewServer(handler Handler) *Server {
	return &Server{
		input:   os.Stdin,
		output:  os.Stdout,
		handler: handler,
		capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    TextDocumentSyncKindIncremental,
				Save: &SaveOptions{
					IncludeText: true,
				},
			},
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{".", "/", "@"},
			},
			HoverProvider:            true,
			DefinitionProvider:       true,
			ReferencesProvider:       true,
			CodeActionProvider:       true,
			DocumentFormattingProvider: true,
			ExecuteCommandProvider: &ExecuteCommandOptions{
				Commands: []string{
					"vibe.runPrompt",
					"vibe.explainCode",
					"vibe.generateTests",
					"vibe.fixError",
					"vibe.refactor",
				},
			},
		},
	}
}

// SetIO sets custom input/output streams.
func (s *Server) SetIO(input io.Reader, output io.Writer) {
	s.input = input
	s.output = output
}

// Run starts the LSP server main loop.
func (s *Server) Run(ctx context.Context) error {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	reader := bufio.NewReader(s.input)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read message
		msg, err := s.readMessage(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			continue
		}

		// Handle message
		response := s.handleMessage(msg)
		if response != nil {
			s.writeMessage(response)
		}
	}
}

func (s *Server) readMessage(reader *bufio.Reader) (*Message, error) {
	// Read headers
	var contentLength int
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(value)
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(reader, content)
	if err != nil {
		return nil, err
	}

	// Parse message
	var msg Message
	if err := json.Unmarshal(content, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (s *Server) writeMessage(msg *Message) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	_, err = s.output.Write([]byte(header))
	if err != nil {
		return err
	}

	_, err = s.output.Write(content)
	return err
}

func (s *Server) handleMessage(msg *Message) *Message {
	// Check if initialized for non-initialize methods
	if !s.initialized && msg.Method != "initialize" && msg.Method != "exit" {
		return s.errorResponse(msg.ID, ServerNotInitialized, "Server not initialized", nil)
	}

	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "initialized":
		return s.handleInitialized(msg)
	case "shutdown":
		return s.handleShutdown(msg)
	case "exit":
		os.Exit(0)
		return nil
	case "textDocument/didOpen":
		return s.handleDidOpen(msg)
	case "textDocument/didChange":
		return s.handleDidChange(msg)
	case "textDocument/didClose":
		return s.handleDidClose(msg)
	case "textDocument/didSave":
		return s.handleDidSave(msg)
	case "textDocument/completion":
		return s.handleCompletion(msg)
	case "textDocument/hover":
		return s.handleHover(msg)
	case "textDocument/definition":
		return s.handleDefinition(msg)
	case "textDocument/references":
		return s.handleReferences(msg)
	case "textDocument/codeAction":
		return s.handleCodeAction(msg)
	case "textDocument/formatting":
		return s.handleFormatting(msg)
	case "workspace/executeCommand":
		return s.handleExecuteCommand(msg)
	default:
		if msg.ID != nil {
			return s.errorResponse(msg.ID, MethodNotFound, "Method not found: "+msg.Method, nil)
		}
		return nil
	}
}

func (s *Server) handleInitialize(msg *Message) *Message {
	var params InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.Initialize(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	if result == nil {
		result = &InitializeResult{
			Capabilities: s.capabilities,
			ServerInfo: &ServerInfo{
				Name:    "vibe-lsp",
				Version: "0.1.0",
			},
		}
	}

	s.workspaceFolders = params.WorkspaceFolders
	s.initialized = true

	return s.successResponse(msg.ID, result)
}

func (s *Server) handleInitialized(msg *Message) *Message {
	var params InitializedParams
	json.Unmarshal(msg.Params, &params)
	s.handler.Initialized(params)
	return nil
}

func (s *Server) handleShutdown(msg *Message) *Message {
	s.handler.Shutdown()
	s.initialized = false
	return s.successResponse(msg.ID, nil)
}

func (s *Server) handleDidOpen(msg *Message) *Message {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return nil
	}
	s.handler.TextDocumentDidOpen(params)
	return nil
}

func (s *Server) handleDidChange(msg *Message) *Message {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return nil
	}
	s.handler.TextDocumentDidChange(params)
	return nil
}

func (s *Server) handleDidClose(msg *Message) *Message {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return nil
	}
	s.handler.TextDocumentDidClose(params)
	return nil
}

func (s *Server) handleDidSave(msg *Message) *Message {
	var params DidSaveTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return nil
	}
	s.handler.TextDocumentDidSave(params)
	return nil
}

func (s *Server) handleCompletion(msg *Message) *Message {
	var params CompletionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.TextDocumentCompletion(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(msg.ID, result)
}

func (s *Server) handleHover(msg *Message) *Message {
	var params HoverParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.TextDocumentHover(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(msg.ID, result)
}

func (s *Server) handleDefinition(msg *Message) *Message {
	var params DefinitionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.TextDocumentDefinition(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(msg.ID, result)
}

func (s *Server) handleReferences(msg *Message) *Message {
	var params ReferenceParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.TextDocumentReferences(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(msg.ID, result)
}

func (s *Server) handleCodeAction(msg *Message) *Message {
	var params CodeActionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.TextDocumentCodeAction(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(msg.ID, result)
}

func (s *Server) handleFormatting(msg *Message) *Message {
	var params DocumentFormattingParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.TextDocumentFormatting(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(msg.ID, result)
}

func (s *Server) handleExecuteCommand(msg *Message) *Message {
	var params ExecuteCommandParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.errorResponse(msg.ID, InvalidParams, err.Error(), nil)
	}

	result, err := s.handler.ExecuteCommand(params)
	if err != nil {
		return s.errorResponse(msg.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(msg.ID, result)
}

func (s *Server) successResponse(id *json.RawMessage, result interface{}) *Message {
	resultJSON, _ := json.Marshal(result)
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultJSON,
	}
}

func (s *Server) errorResponse(id *json.RawMessage, code int, message string, data interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ResponseError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// SendNotification sends a notification to the client.
func (s *Server) SendNotification(method string, params interface{}) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	msg := &Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	return s.writeMessage(msg)
}

// PublishDiagnostics sends diagnostics to the client.
func (s *Server) PublishDiagnostics(uri string, diagnostics []Diagnostic) error {
	return s.SendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// ShowMessage shows a message to the user.
func (s *Server) ShowMessage(msgType MessageType, message string) error {
	return s.SendNotification("window/showMessage", ShowMessageParams{
		Type:    msgType,
		Message: message,
	})
}

// LogMessage logs a message.
func (s *Server) LogMessage(msgType MessageType, message string) error {
	return s.SendNotification("window/logMessage", LogMessageParams{
		Type:    msgType,
		Message: message,
	})
}
