package lsp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mateus/flow-cli/internal/llm"
)

// FlowHandler implements the LSP Handler interface for flow.
type FlowHandler struct {
	documents     map[URI]*Document
	mu            sync.RWMutex
	workspacePath string
	server        *Server

	// LLM integration
	llmClient llm.Client
	model     string
}

// Document represents an open document.
type Document struct {
	URI        URI
	LanguageID string
	Version    int
	Content    string
	Lines      []string
}

// HandlerConfig holds configuration for the FlowHandler.
type HandlerConfig struct {
	LLMClient llm.Client
	Model     string
}

// NewFlowHandler creates a new flow handler.
func NewFlowHandler() *FlowHandler {
	return &FlowHandler{
		documents: make(map[URI]*Document),
	}
}

// NewFlowHandlerWithConfig creates a new flow handler with LLM support.
func NewFlowHandlerWithConfig(cfg HandlerConfig) *FlowHandler {
	return &FlowHandler{
		documents: make(map[URI]*Document),
		llmClient: cfg.LLMClient,
		model:     cfg.Model,
	}
}

// SetLLMClient sets the LLM client for AI-powered features.
func (h *FlowHandler) SetLLMClient(client llm.Client, model string) {
	h.llmClient = client
	h.model = model
}

// SetServer sets the server reference for sending notifications.
func (h *FlowHandler) SetServer(server *Server) {
	h.server = server
}

// Initialize handles the initialize request.
func (h *FlowHandler) Initialize(params InitializeParams) (*InitializeResult, error) {
	if len(params.WorkspaceFolders) > 0 {
		h.workspacePath = uriToPath(params.WorkspaceFolders[0].URI)
	} else if params.RootURI != "" {
		h.workspacePath = uriToPath(params.RootURI)
	} else if params.RootPath != "" {
		h.workspacePath = params.RootPath
	}

	return nil, nil // Let server use default capabilities
}

// Initialized handles the initialized notification.
func (h *FlowHandler) Initialized(params InitializedParams) error {
	return nil
}

// Shutdown handles the shutdown request.
func (h *FlowHandler) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.documents = make(map[URI]*Document)
	return nil
}

// TextDocumentDidOpen handles textDocument/didOpen.
func (h *FlowHandler) TextDocumentDidOpen(params DidOpenTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	doc := &Document{
		URI:        params.TextDocument.URI,
		LanguageID: params.TextDocument.LanguageID,
		Version:    params.TextDocument.Version,
		Content:    params.TextDocument.Text,
		Lines:      strings.Split(params.TextDocument.Text, "\n"),
	}
	h.documents[params.TextDocument.URI] = doc

	return nil
}

// TextDocumentDidChange handles textDocument/didChange.
func (h *FlowHandler) TextDocumentDidChange(params DidChangeTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	doc, ok := h.documents[params.TextDocument.URI]
	if !ok {
		return nil
	}

	doc.Version = params.TextDocument.Version

	for _, change := range params.ContentChanges {
		if change.Range == nil {
			// Full document update
			doc.Content = change.Text
			doc.Lines = strings.Split(change.Text, "\n")
		} else {
			// Incremental update
			doc.Content = applyChange(doc.Content, *change.Range, change.Text)
			doc.Lines = strings.Split(doc.Content, "\n")
		}
	}

	return nil
}

// TextDocumentDidClose handles textDocument/didClose.
func (h *FlowHandler) TextDocumentDidClose(params DidCloseTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.documents, params.TextDocument.URI)
	return nil
}

// TextDocumentDidSave handles textDocument/didSave.
func (h *FlowHandler) TextDocumentDidSave(params DidSaveTextDocumentParams) error {
	// Could trigger analysis or other operations on save
	return nil
}

// TextDocumentCompletion handles textDocument/completion.
func (h *FlowHandler) TextDocumentCompletion(params CompletionParams) (*CompletionList, error) {
	h.mu.RLock()
	doc, ok := h.documents[params.TextDocument.URI]
	h.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	// Get context around cursor
	line := ""
	if params.Position.Line < len(doc.Lines) {
		line = doc.Lines[params.Position.Line]
	}

	// Get prefix before cursor
	prefix := ""
	if params.Position.Character <= len(line) {
		prefix = line[:params.Position.Character]
	}

	items := []CompletionItem{}

	// Check for flow command trigger
	if strings.Contains(prefix, "@flow") || strings.HasSuffix(prefix, "@") {
		items = append(items, h.getFlowCompletions()...)
	}

	// Add language-specific completions
	items = append(items, h.getLanguageCompletions(doc.LanguageID, prefix)...)

	return &CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func (h *FlowHandler) getFlowCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:      "@flow explain",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Explain this code",
			InsertText: "@flow explain",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Ask flow to explain the selected code",
			},
		},
		{
			Label:      "@flow fix",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Fix this issue",
			InsertText: "@flow fix",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Ask flow to fix the current issue",
			},
		},
		{
			Label:      "@flow test",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Generate tests",
			InsertText: "@flow test",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Generate tests for this code",
			},
		},
		{
			Label:      "@flow refactor",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Refactor code",
			InsertText: "@flow refactor",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Suggest refactoring for this code",
			},
		},
		{
			Label:      "@flow doc",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Generate documentation",
			InsertText: "@flow doc",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Generate documentation for this code",
			},
		},
	}
}

func (h *FlowHandler) getLanguageCompletions(languageID, prefix string) []CompletionItem {
	// Basic completions based on language
	// In a real implementation, this would integrate with flow's LLM
	return []CompletionItem{}
}

// TextDocumentHover handles textDocument/hover.
func (h *FlowHandler) TextDocumentHover(params HoverParams) (*Hover, error) {
	h.mu.RLock()
	doc, ok := h.documents[params.TextDocument.URI]
	h.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	// Get word at position
	word := getWordAtPosition(doc, params.Position)
	if word == "" {
		return nil, nil
	}

	// Get surrounding context for better explanations
	codeContext := h.getSurroundingContext(doc, params.Position, 3)

	// If LLM is available and word looks significant, provide AI explanation
	if h.llmClient != nil && len(word) > 2 {
		// Use a short timeout for hover to keep UI responsive
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful coding assistant. Provide a brief, one-paragraph explanation of the code element. Be concise."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Briefly explain what '%s' does in this context:\n\n```%s\n%s\n```", word, doc.LanguageID, codeContext)},
		}

		response, err := h.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
			Model:       h.model,
			Temperature: 0.2,
			MaxTokens:   150,
		})
		if err == nil && response != "" {
			return &Hover{
				Contents: MarkupContent{
					Kind:  MarkupKindMarkdown,
					Value: fmt.Sprintf("**%s**\n\n%s", word, response),
				},
			}, nil
		}
	}

	// Fallback to basic info
	return &Hover{
		Contents: MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: fmt.Sprintf("**%s**\n\nUse `@flow explain` for AI-powered explanation.", word),
		},
	}, nil
}

// getSurroundingContext gets lines around a position for context.
func (h *FlowHandler) getSurroundingContext(doc *Document, pos Position, linesBefore int) string {
	startLine := pos.Line - linesBefore
	if startLine < 0 {
		startLine = 0
	}
	endLine := pos.Line + linesBefore
	if endLine >= len(doc.Lines) {
		endLine = len(doc.Lines) - 1
	}

	var result strings.Builder
	for i := startLine; i <= endLine; i++ {
		result.WriteString(doc.Lines[i])
		if i < endLine {
			result.WriteString("\n")
		}
	}
	return result.String()
}

// TextDocumentDefinition handles textDocument/definition.
func (h *FlowHandler) TextDocumentDefinition(params DefinitionParams) ([]Location, error) {
	// In a real implementation, this would search the codebase
	// using flow's analysis tools
	return nil, nil
}

// TextDocumentReferences handles textDocument/references.
func (h *FlowHandler) TextDocumentReferences(params ReferenceParams) ([]Location, error) {
	// In a real implementation, this would search the codebase
	return nil, nil
}

// TextDocumentCodeAction handles textDocument/codeAction.
func (h *FlowHandler) TextDocumentCodeAction(params CodeActionParams) ([]CodeAction, error) {
	actions := []CodeAction{}

	// Add flow-powered code actions
	actions = append(actions, CodeAction{
		Title: "Explain with Flow",
		Kind:  CodeActionKindQuickFix,
		Command: &Command{
			Title:   "Explain with Flow",
			Command: "flow.explainCode",
			Arguments: []interface{}{
				string(params.TextDocument.URI),
				params.Range,
			},
		},
	})

	actions = append(actions, CodeAction{
		Title: "Generate Tests with Flow",
		Kind:  CodeActionKindRefactor,
		Command: &Command{
			Title:   "Generate Tests",
			Command: "flow.generateTests",
			Arguments: []interface{}{
				string(params.TextDocument.URI),
				params.Range,
			},
		},
	})

	actions = append(actions, CodeAction{
		Title: "Refactor with Flow",
		Kind:  CodeActionKindRefactor,
		Command: &Command{
			Title:   "Refactor Code",
			Command: "flow.refactor",
			Arguments: []interface{}{
				string(params.TextDocument.URI),
				params.Range,
			},
		},
	})

	// Add fix actions for diagnostics
	for _, diag := range params.Context.Diagnostics {
		actions = append(actions, CodeAction{
			Title: fmt.Sprintf("Fix: %s", diag.Message),
			Kind:  CodeActionKindQuickFix,
			Diagnostics: []Diagnostic{diag},
			Command: &Command{
				Title:   "Fix with Flow",
				Command: "flow.fixError",
				Arguments: []interface{}{
					string(params.TextDocument.URI),
					diag,
				},
			},
		})
	}

	return actions, nil
}

// TextDocumentFormatting handles textDocument/formatting.
func (h *FlowHandler) TextDocumentFormatting(params DocumentFormattingParams) ([]TextEdit, error) {
	// In a real implementation, this would use language-specific formatters
	return nil, nil
}

// ExecuteCommand handles workspace/executeCommand.
func (h *FlowHandler) ExecuteCommand(params ExecuteCommandParams) (interface{}, error) {
	switch params.Command {
	case "flow.runPrompt":
		return h.executeRunPrompt(params.Arguments)
	case "flow.explainCode":
		return h.executeExplainCode(params.Arguments)
	case "flow.generateTests":
		return h.executeGenerateTests(params.Arguments)
	case "flow.fixError":
		return h.executeFixError(params.Arguments)
	case "flow.refactor":
		return h.executeRefactor(params.Arguments)
	default:
		return nil, fmt.Errorf("unknown command: %s", params.Command)
	}
}

func (h *FlowHandler) executeRunPrompt(args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("missing prompt argument")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid prompt argument")
	}

	// Use LLM if available
	if h.llmClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful coding assistant integrated into an IDE."},
			{Role: llm.RoleUser, Content: prompt},
		}

		response, err := h.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
			Model:       h.model,
			Temperature: 0.7,
		})
		if err != nil {
			return map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}, nil
		}

		return map[string]interface{}{
			"status":   "ok",
			"response": response,
		}, nil
	}

	return map[string]string{
		"status":  "error",
		"message": "LLM client not configured",
	}, nil
}

func (h *FlowHandler) executeExplainCode(args []interface{}) (interface{}, error) {
	// Extract URI and range from arguments
	var uri string
	var code string

	if len(args) >= 1 {
		uri, _ = args[0].(string)
	}

	// Get code from document if we have a URI
	if uri != "" {
		h.mu.RLock()
		doc, ok := h.documents[URI(uri)]
		h.mu.RUnlock()
		if ok {
			// If range provided, extract that portion; otherwise use full content
			if len(args) >= 2 {
				if rng, ok := args[1].(map[string]interface{}); ok {
					code = h.extractCodeFromRange(doc, rng)
				}
			}
			if code == "" {
				code = doc.Content
			}
		}
	}

	if h.llmClient != nil && code != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful coding assistant. Explain the following code clearly and concisely."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Please explain this code:\n\n```\n%s\n```", code)},
		}

		response, err := h.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
			Model:       h.model,
			Temperature: 0.3,
		})
		if err != nil {
			return map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}, nil
		}

		return map[string]interface{}{
			"status":      "ok",
			"explanation": response,
		}, nil
	}

	return map[string]string{
		"status":  "error",
		"message": "LLM client not configured or no code provided",
	}, nil
}

func (h *FlowHandler) executeGenerateTests(args []interface{}) (interface{}, error) {
	code := h.extractCodeFromArgs(args)

	if h.llmClient != nil && code != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful coding assistant. Generate comprehensive unit tests for the provided code. Include edge cases and use appropriate testing patterns for the language."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Generate tests for this code:\n\n```\n%s\n```", code)},
		}

		response, err := h.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
			Model:       h.model,
			Temperature: 0.3,
		})
		if err != nil {
			return map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}, nil
		}

		return map[string]interface{}{
			"status": "ok",
			"tests":  response,
		}, nil
	}

	return map[string]string{
		"status":  "error",
		"message": "LLM client not configured or no code provided",
	}, nil
}

func (h *FlowHandler) executeFixError(args []interface{}) (interface{}, error) {
	code := h.extractCodeFromArgs(args)

	// Extract diagnostic info if available
	var diagnostic string
	if len(args) >= 2 {
		if diag, ok := args[1].(map[string]interface{}); ok {
			if msg, ok := diag["message"].(string); ok {
				diagnostic = msg
			}
		}
	}

	if h.llmClient != nil && code != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		prompt := fmt.Sprintf("Fix the following code:\n\n```\n%s\n```", code)
		if diagnostic != "" {
			prompt = fmt.Sprintf("Fix the following error in the code:\n\nError: %s\n\nCode:\n```\n%s\n```", diagnostic, code)
		}

		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful coding assistant. Analyze the code and provide a fix. Show the corrected code and explain what was wrong."},
			{Role: llm.RoleUser, Content: prompt},
		}

		response, err := h.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
			Model:       h.model,
			Temperature: 0.2,
		})
		if err != nil {
			return map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}, nil
		}

		return map[string]interface{}{
			"status": "ok",
			"fix":    response,
		}, nil
	}

	return map[string]string{
		"status":  "error",
		"message": "LLM client not configured or no code provided",
	}, nil
}

func (h *FlowHandler) executeRefactor(args []interface{}) (interface{}, error) {
	code := h.extractCodeFromArgs(args)

	if h.llmClient != nil && code != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful coding assistant. Suggest refactoring improvements for the provided code. Focus on readability, maintainability, and best practices. Show the refactored code and explain the changes."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Suggest refactoring for this code:\n\n```\n%s\n```", code)},
		}

		response, err := h.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
			Model:       h.model,
			Temperature: 0.4,
		})
		if err != nil {
			return map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}, nil
		}

		return map[string]interface{}{
			"status":      "ok",
			"refactoring": response,
		}, nil
	}

	return map[string]string{
		"status":  "error",
		"message": "LLM client not configured or no code provided",
	}, nil
}

// extractCodeFromArgs extracts code from command arguments.
func (h *FlowHandler) extractCodeFromArgs(args []interface{}) string {
	if len(args) < 1 {
		return ""
	}

	uri, ok := args[0].(string)
	if !ok {
		return ""
	}

	h.mu.RLock()
	doc, ok := h.documents[URI(uri)]
	h.mu.RUnlock()
	if !ok {
		return ""
	}

	// If range provided, extract that portion
	if len(args) >= 2 {
		if rng, ok := args[1].(map[string]interface{}); ok {
			code := h.extractCodeFromRange(doc, rng)
			if code != "" {
				return code
			}
		}
	}

	return doc.Content
}

// extractCodeFromRange extracts code from a document given a range.
func (h *FlowHandler) extractCodeFromRange(doc *Document, rng map[string]interface{}) string {
	start, ok := rng["start"].(map[string]interface{})
	if !ok {
		return ""
	}
	end, ok := rng["end"].(map[string]interface{})
	if !ok {
		return ""
	}

	startLine := int(start["line"].(float64))
	startChar := int(start["character"].(float64))
	endLine := int(end["line"].(float64))
	endChar := int(end["character"].(float64))

	if startLine >= len(doc.Lines) || endLine >= len(doc.Lines) {
		return ""
	}

	if startLine == endLine {
		line := doc.Lines[startLine]
		if startChar > len(line) {
			startChar = len(line)
		}
		if endChar > len(line) {
			endChar = len(line)
		}
		return line[startChar:endChar]
	}

	var result strings.Builder
	for i := startLine; i <= endLine; i++ {
		line := doc.Lines[i]
		if i == startLine {
			if startChar < len(line) {
				result.WriteString(line[startChar:])
			}
		} else if i == endLine {
			if endChar > len(line) {
				endChar = len(line)
			}
			result.WriteString(line[:endChar])
		} else {
			result.WriteString(line)
		}
		if i < endLine {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// Helper functions

func uriToPath(uri URI) string {
	path := string(uri)
	path = strings.TrimPrefix(path, "file://")
	path = strings.TrimPrefix(path, "file:")

	// Handle Windows paths
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	return filepath.Clean(path)
}

func applyChange(content string, rng Range, newText string) string {
	lines := strings.Split(content, "\n")

	// Get start and end positions
	startLine := rng.Start.Line
	startChar := rng.Start.Character
	endLine := rng.End.Line
	endChar := rng.End.Character

	// Ensure valid range
	if startLine >= len(lines) {
		return content + newText
	}

	// Build prefix (before the change)
	var prefix string
	for i := 0; i < startLine; i++ {
		prefix += lines[i] + "\n"
	}
	if startChar <= len(lines[startLine]) {
		prefix += lines[startLine][:startChar]
	}

	// Build suffix (after the change)
	var suffix string
	if endLine < len(lines) {
		if endChar <= len(lines[endLine]) {
			suffix = lines[endLine][endChar:]
		}
		for i := endLine + 1; i < len(lines); i++ {
			suffix += "\n" + lines[i]
		}
	}

	return prefix + newText + suffix
}

func getWordAtPosition(doc *Document, pos Position) string {
	if pos.Line >= len(doc.Lines) {
		return ""
	}

	line := doc.Lines[pos.Line]
	if pos.Character >= len(line) {
		return ""
	}

	// Find word boundaries
	start := pos.Character
	end := pos.Character

	// Go backwards to find start
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Go forwards to find end
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	if start == end {
		return ""
	}

	return line[start:end]
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}
