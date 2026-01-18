package lsp

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// VibeHandler implements the LSP Handler interface for vibe.
type VibeHandler struct {
	documents     map[URI]*Document
	mu            sync.RWMutex
	workspacePath string
	server        *Server
}

// Document represents an open document.
type Document struct {
	URI        URI
	LanguageID string
	Version    int
	Content    string
	Lines      []string
}

// NewVibeHandler creates a new vibe handler.
func NewVibeHandler() *VibeHandler {
	return &VibeHandler{
		documents: make(map[URI]*Document),
	}
}

// SetServer sets the server reference for sending notifications.
func (h *VibeHandler) SetServer(server *Server) {
	h.server = server
}

// Initialize handles the initialize request.
func (h *VibeHandler) Initialize(params InitializeParams) (*InitializeResult, error) {
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
func (h *VibeHandler) Initialized(params InitializedParams) error {
	return nil
}

// Shutdown handles the shutdown request.
func (h *VibeHandler) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.documents = make(map[URI]*Document)
	return nil
}

// TextDocumentDidOpen handles textDocument/didOpen.
func (h *VibeHandler) TextDocumentDidOpen(params DidOpenTextDocumentParams) error {
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
func (h *VibeHandler) TextDocumentDidChange(params DidChangeTextDocumentParams) error {
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
func (h *VibeHandler) TextDocumentDidClose(params DidCloseTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.documents, params.TextDocument.URI)
	return nil
}

// TextDocumentDidSave handles textDocument/didSave.
func (h *VibeHandler) TextDocumentDidSave(params DidSaveTextDocumentParams) error {
	// Could trigger analysis or other operations on save
	return nil
}

// TextDocumentCompletion handles textDocument/completion.
func (h *VibeHandler) TextDocumentCompletion(params CompletionParams) (*CompletionList, error) {
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

	// Check for vibe command trigger
	if strings.Contains(prefix, "@vibe") || strings.HasSuffix(prefix, "@") {
		items = append(items, h.getVibeCompletions()...)
	}

	// Add language-specific completions
	items = append(items, h.getLanguageCompletions(doc.LanguageID, prefix)...)

	return &CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func (h *VibeHandler) getVibeCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:      "@vibe explain",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Explain this code",
			InsertText: "@vibe explain",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Ask vibe to explain the selected code",
			},
		},
		{
			Label:      "@vibe fix",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Fix this issue",
			InsertText: "@vibe fix",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Ask vibe to fix the current issue",
			},
		},
		{
			Label:      "@vibe test",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Generate tests",
			InsertText: "@vibe test",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Generate tests for this code",
			},
		},
		{
			Label:      "@vibe refactor",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Refactor code",
			InsertText: "@vibe refactor",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Suggest refactoring for this code",
			},
		},
		{
			Label:      "@vibe doc",
			Kind:       CompletionItemKindSnippet,
			Detail:     "Generate documentation",
			InsertText: "@vibe doc",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Generate documentation for this code",
			},
		},
	}
}

func (h *VibeHandler) getLanguageCompletions(languageID, prefix string) []CompletionItem {
	// Basic completions based on language
	// In a real implementation, this would integrate with vibe's LLM
	return []CompletionItem{}
}

// TextDocumentHover handles textDocument/hover.
func (h *VibeHandler) TextDocumentHover(params HoverParams) (*Hover, error) {
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

	// For now, return basic info
	// In a real implementation, this would use vibe's LLM for explanations
	return &Hover{
		Contents: MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: fmt.Sprintf("**%s**\n\nUse `@vibe explain` for AI-powered explanation.", word),
		},
	}, nil
}

// TextDocumentDefinition handles textDocument/definition.
func (h *VibeHandler) TextDocumentDefinition(params DefinitionParams) ([]Location, error) {
	// In a real implementation, this would search the codebase
	// using vibe's analysis tools
	return nil, nil
}

// TextDocumentReferences handles textDocument/references.
func (h *VibeHandler) TextDocumentReferences(params ReferenceParams) ([]Location, error) {
	// In a real implementation, this would search the codebase
	return nil, nil
}

// TextDocumentCodeAction handles textDocument/codeAction.
func (h *VibeHandler) TextDocumentCodeAction(params CodeActionParams) ([]CodeAction, error) {
	actions := []CodeAction{}

	// Add vibe-powered code actions
	actions = append(actions, CodeAction{
		Title: "Explain with Vibe",
		Kind:  CodeActionKindQuickFix,
		Command: &Command{
			Title:   "Explain with Vibe",
			Command: "vibe.explainCode",
			Arguments: []interface{}{
				string(params.TextDocument.URI),
				params.Range,
			},
		},
	})

	actions = append(actions, CodeAction{
		Title: "Generate Tests with Vibe",
		Kind:  CodeActionKindRefactor,
		Command: &Command{
			Title:   "Generate Tests",
			Command: "vibe.generateTests",
			Arguments: []interface{}{
				string(params.TextDocument.URI),
				params.Range,
			},
		},
	})

	actions = append(actions, CodeAction{
		Title: "Refactor with Vibe",
		Kind:  CodeActionKindRefactor,
		Command: &Command{
			Title:   "Refactor Code",
			Command: "vibe.refactor",
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
				Title:   "Fix with Vibe",
				Command: "vibe.fixError",
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
func (h *VibeHandler) TextDocumentFormatting(params DocumentFormattingParams) ([]TextEdit, error) {
	// In a real implementation, this would use language-specific formatters
	return nil, nil
}

// ExecuteCommand handles workspace/executeCommand.
func (h *VibeHandler) ExecuteCommand(params ExecuteCommandParams) (interface{}, error) {
	switch params.Command {
	case "vibe.runPrompt":
		return h.executeRunPrompt(params.Arguments)
	case "vibe.explainCode":
		return h.executeExplainCode(params.Arguments)
	case "vibe.generateTests":
		return h.executeGenerateTests(params.Arguments)
	case "vibe.fixError":
		return h.executeFixError(params.Arguments)
	case "vibe.refactor":
		return h.executeRefactor(params.Arguments)
	default:
		return nil, fmt.Errorf("unknown command: %s", params.Command)
	}
}

func (h *VibeHandler) executeRunPrompt(args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("missing prompt argument")
	}

	prompt, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid prompt argument")
	}

	// In a real implementation, this would call vibe's LLM
	return map[string]string{
		"status": "ok",
		"prompt": prompt,
	}, nil
}

func (h *VibeHandler) executeExplainCode(args []interface{}) (interface{}, error) {
	// In a real implementation, this would use vibe's LLM to explain code
	return map[string]string{
		"status":  "ok",
		"message": "Code explanation would appear here",
	}, nil
}

func (h *VibeHandler) executeGenerateTests(args []interface{}) (interface{}, error) {
	// In a real implementation, this would use vibe's test generation
	return map[string]string{
		"status":  "ok",
		"message": "Generated tests would appear here",
	}, nil
}

func (h *VibeHandler) executeFixError(args []interface{}) (interface{}, error) {
	// In a real implementation, this would use vibe's bug fix suggestions
	return map[string]string{
		"status":  "ok",
		"message": "Fix suggestion would appear here",
	}, nil
}

func (h *VibeHandler) executeRefactor(args []interface{}) (interface{}, error) {
	// In a real implementation, this would use vibe's refactoring tools
	return map[string]string{
		"status":  "ok",
		"message": "Refactoring suggestion would appear here",
	}, nil
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
