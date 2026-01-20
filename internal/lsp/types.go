package lsp

// URI represents a document URI.
type URI string

// Position represents a position in a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a document.
type Location struct {
	URI   URI   `json:"uri"`
	Range Range `json:"range"`
}

// TextDocumentIdentifier identifies a text document.
type TextDocumentIdentifier struct {
	URI URI `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a versioned text document.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentItem represents a text document.
type TextDocumentItem struct {
	URI        URI    `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentPositionParams contains a text document and position.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// TextEdit represents a text edit.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// WorkspaceEdit represents changes to multiple documents.
type WorkspaceEdit struct {
	Changes         map[URI][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []TextDocumentEdit `json:"documentChanges,omitempty"`
}

// TextDocumentEdit represents edits to a single document.
type TextDocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                      `json:"edits"`
}

// WorkspaceFolder represents a workspace folder.
type WorkspaceFolder struct {
	URI  URI    `json:"uri"`
	Name string `json:"name"`
}

// Initialize types

// InitializeParams are the parameters for the initialize request.
type InitializeParams struct {
	ProcessID             int               `json:"processId,omitempty"`
	RootPath              string            `json:"rootPath,omitempty"`
	RootURI               URI               `json:"rootUri,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	Trace                 string            `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder `json:"workspaceFolders,omitempty"`
}

// ClientCapabilities describes client capabilities.
type ClientCapabilities struct {
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *WindowClientCapabilities       `json:"window,omitempty"`
}

// WorkspaceClientCapabilities describes workspace capabilities.
type WorkspaceClientCapabilities struct {
	ApplyEdit              bool `json:"applyEdit,omitempty"`
	WorkspaceEdit          *WorkspaceEditClientCapabilities `json:"workspaceEdit,omitempty"`
	DidChangeConfiguration *DidChangeConfigurationClientCapabilities `json:"didChangeConfiguration,omitempty"`
	WorkspaceFolders       bool `json:"workspaceFolders,omitempty"`
}

// WorkspaceEditClientCapabilities describes workspace edit capabilities.
type WorkspaceEditClientCapabilities struct {
	DocumentChanges bool `json:"documentChanges,omitempty"`
}

// DidChangeConfigurationClientCapabilities describes configuration change capabilities.
type DidChangeConfigurationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// TextDocumentClientCapabilities describes text document capabilities.
type TextDocumentClientCapabilities struct {
	Synchronization *TextDocumentSyncClientCapabilities `json:"synchronization,omitempty"`
	Completion      *CompletionClientCapabilities       `json:"completion,omitempty"`
	Hover           *HoverClientCapabilities            `json:"hover,omitempty"`
}

// TextDocumentSyncClientCapabilities describes sync capabilities.
type TextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	WillSave            bool `json:"willSave,omitempty"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil,omitempty"`
	DidSave             bool `json:"didSave,omitempty"`
}

// CompletionClientCapabilities describes completion capabilities.
type CompletionClientCapabilities struct {
	DynamicRegistration bool                        `json:"dynamicRegistration,omitempty"`
	CompletionItem      *CompletionItemCapabilities `json:"completionItem,omitempty"`
}

// CompletionItemCapabilities describes completion item capabilities.
type CompletionItemCapabilities struct {
	SnippetSupport          bool     `json:"snippetSupport,omitempty"`
	CommitCharactersSupport bool     `json:"commitCharactersSupport,omitempty"`
	DocumentationFormat     []string `json:"documentationFormat,omitempty"`
}

// HoverClientCapabilities describes hover capabilities.
type HoverClientCapabilities struct {
	DynamicRegistration bool     `json:"dynamicRegistration,omitempty"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

// WindowClientCapabilities describes window capabilities.
type WindowClientCapabilities struct {
	WorkDoneProgress bool `json:"workDoneProgress,omitempty"`
}

// InitializeResult is the result of the initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerInfo contains server information.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerCapabilities describes server capabilities.
type ServerCapabilities struct {
	TextDocumentSync           *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	CompletionProvider         *CompletionOptions       `json:"completionProvider,omitempty"`
	HoverProvider              bool                     `json:"hoverProvider,omitempty"`
	SignatureHelpProvider      *SignatureHelpOptions    `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider         bool                     `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider     bool                     `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider     bool                     `json:"implementationProvider,omitempty"`
	ReferencesProvider         bool                     `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider  bool                     `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider     bool                     `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider    bool                     `json:"workspaceSymbolProvider,omitempty"`
	CodeActionProvider         bool                     `json:"codeActionProvider,omitempty"`
	DocumentFormattingProvider bool                     `json:"documentFormattingProvider,omitempty"`
	RenameProvider             bool                     `json:"renameProvider,omitempty"`
	ExecuteCommandProvider     *ExecuteCommandOptions   `json:"executeCommandProvider,omitempty"`
	Workspace                  *ServerWorkspaceCapabilities `json:"workspace,omitempty"`
}

// TextDocumentSyncOptions describes text document sync options.
type TextDocumentSyncOptions struct {
	OpenClose bool                 `json:"openClose,omitempty"`
	Change    TextDocumentSyncKind `json:"change,omitempty"`
	Save      *SaveOptions         `json:"save,omitempty"`
}

// TextDocumentSyncKind represents sync kind.
type TextDocumentSyncKind int

const (
	TextDocumentSyncKindNone        TextDocumentSyncKind = 0
	TextDocumentSyncKindFull        TextDocumentSyncKind = 1
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2
)

// SaveOptions describes save options.
type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

// CompletionOptions describes completion options.
type CompletionOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	ResolveProvider     bool     `json:"resolveProvider,omitempty"`
	WorkDoneProgress    bool     `json:"workDoneProgress,omitempty"`
}

// SignatureHelpOptions describes signature help options.
type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

// ExecuteCommandOptions describes execute command options.
type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

// ServerWorkspaceCapabilities describes workspace capabilities.
type ServerWorkspaceCapabilities struct {
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
}

// WorkspaceFoldersServerCapabilities describes workspace folder capabilities.
type WorkspaceFoldersServerCapabilities struct {
	Supported           bool `json:"supported,omitempty"`
	ChangeNotifications bool `json:"changeNotifications,omitempty"`
}

// InitializedParams are the parameters for the initialized notification.
type InitializedParams struct{}

// Document sync types

// DidOpenTextDocumentParams are the parameters for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams are the parameters for textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent describes a content change.
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// DidCloseTextDocumentParams are the parameters for textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams are the parameters for textDocument/didSave.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// Completion types

// CompletionParams are the parameters for textDocument/completion.
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext describes completion context.
type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

// CompletionTriggerKind represents how completion was triggered.
type CompletionTriggerKind int

const (
	CompletionTriggerKindInvoked                         CompletionTriggerKind = 1
	CompletionTriggerKindTriggerCharacter                CompletionTriggerKind = 2
	CompletionTriggerKindTriggerForIncompleteCompletions CompletionTriggerKind = 3
)

// CompletionList represents a list of completions.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// CompletionItem represents a completion item.
type CompletionItem struct {
	Label            string             `json:"label"`
	Kind             CompletionItemKind `json:"kind,omitempty"`
	Detail           string             `json:"detail,omitempty"`
	Documentation    interface{}        `json:"documentation,omitempty"`
	Deprecated       bool               `json:"deprecated,omitempty"`
	Preselect        bool               `json:"preselect,omitempty"`
	SortText         string             `json:"sortText,omitempty"`
	FilterText       string             `json:"filterText,omitempty"`
	InsertText       string             `json:"insertText,omitempty"`
	InsertTextFormat InsertTextFormat   `json:"insertTextFormat,omitempty"`
	TextEdit         *TextEdit          `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit      `json:"additionalTextEdits,omitempty"`
	CommitCharacters []string           `json:"commitCharacters,omitempty"`
	Command          *Command           `json:"command,omitempty"`
	Data             interface{}        `json:"data,omitempty"`
}

// CompletionItemKind represents the kind of a completion item.
type CompletionItemKind int

const (
	CompletionItemKindText          CompletionItemKind = 1
	CompletionItemKindMethod        CompletionItemKind = 2
	CompletionItemKindFunction      CompletionItemKind = 3
	CompletionItemKindConstructor   CompletionItemKind = 4
	CompletionItemKindField         CompletionItemKind = 5
	CompletionItemKindVariable      CompletionItemKind = 6
	CompletionItemKindClass         CompletionItemKind = 7
	CompletionItemKindInterface     CompletionItemKind = 8
	CompletionItemKindModule        CompletionItemKind = 9
	CompletionItemKindProperty      CompletionItemKind = 10
	CompletionItemKindUnit          CompletionItemKind = 11
	CompletionItemKindValue         CompletionItemKind = 12
	CompletionItemKindEnum          CompletionItemKind = 13
	CompletionItemKindKeyword       CompletionItemKind = 14
	CompletionItemKindSnippet       CompletionItemKind = 15
	CompletionItemKindColor         CompletionItemKind = 16
	CompletionItemKindFile          CompletionItemKind = 17
	CompletionItemKindReference     CompletionItemKind = 18
	CompletionItemKindFolder        CompletionItemKind = 19
	CompletionItemKindEnumMember    CompletionItemKind = 20
	CompletionItemKindConstant      CompletionItemKind = 21
	CompletionItemKindStruct        CompletionItemKind = 22
	CompletionItemKindEvent         CompletionItemKind = 23
	CompletionItemKindOperator      CompletionItemKind = 24
	CompletionItemKindTypeParameter CompletionItemKind = 25
)

// InsertTextFormat represents the format of the insert text.
type InsertTextFormat int

const (
	InsertTextFormatPlainText InsertTextFormat = 1
	InsertTextFormatSnippet   InsertTextFormat = 2
)

// Hover types

// HoverParams are the parameters for textDocument/hover.
type HoverParams struct {
	TextDocumentPositionParams
}

// Hover represents hover information.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent represents markup content.
type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

// MarkupKind represents the kind of markup.
type MarkupKind string

const (
	MarkupKindPlainText MarkupKind = "plaintext"
	MarkupKindMarkdown  MarkupKind = "markdown"
)

// Definition types

// DefinitionParams are the parameters for textDocument/definition.
type DefinitionParams struct {
	TextDocumentPositionParams
}

// Reference types

// ReferenceParams are the parameters for textDocument/references.
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext describes reference context.
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// Code action types

// CodeActionParams are the parameters for textDocument/codeAction.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext describes code action context.
type CodeActionContext struct {
	Diagnostics []Diagnostic     `json:"diagnostics"`
	Only        []CodeActionKind `json:"only,omitempty"`
}

// CodeActionKind represents the kind of a code action.
type CodeActionKind string

const (
	CodeActionKindQuickFix              CodeActionKind = "quickfix"
	CodeActionKindRefactor              CodeActionKind = "refactor"
	CodeActionKindRefactorExtract       CodeActionKind = "refactor.extract"
	CodeActionKindRefactorInline        CodeActionKind = "refactor.inline"
	CodeActionKindRefactorRewrite       CodeActionKind = "refactor.rewrite"
	CodeActionKindSource                CodeActionKind = "source"
	CodeActionKindSourceOrganizeImports CodeActionKind = "source.organizeImports"
)

// CodeAction represents a code action.
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        CodeActionKind `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
	Data        interface{}    `json:"data,omitempty"`
}

// Command represents a command.
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// Formatting types

// DocumentFormattingParams are the parameters for textDocument/formatting.
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// FormattingOptions describes formatting options.
type FormattingOptions struct {
	TabSize                int  `json:"tabSize"`
	InsertSpaces           bool `json:"insertSpaces"`
	TrimTrailingWhitespace bool `json:"trimTrailingWhitespace,omitempty"`
	InsertFinalNewline     bool `json:"insertFinalNewline,omitempty"`
	TrimFinalNewlines      bool `json:"trimFinalNewlines,omitempty"`
}

// Execute command types

// ExecuteCommandParams are the parameters for workspace/executeCommand.
type ExecuteCommandParams struct {
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// Diagnostic types

// Diagnostic represents a diagnostic.
type Diagnostic struct {
	Range              Range              `json:"range"`
	Severity           DiagnosticSeverity `json:"severity,omitempty"`
	Code               interface{}        `json:"code,omitempty"`
	Source             string             `json:"source,omitempty"`
	Message            string             `json:"message"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

// DiagnosticSeverity represents diagnostic severity.
type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

// DiagnosticRelatedInformation represents related information.
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// PublishDiagnosticsParams are the parameters for textDocument/publishDiagnostics.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     int          `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Message types

// MessageType represents message types.
type MessageType int

const (
	MessageTypeError   MessageType = 1
	MessageTypeWarning MessageType = 2
	MessageTypeInfo    MessageType = 3
	MessageTypeLog     MessageType = 4
)

// ShowMessageParams are the parameters for window/showMessage.
type ShowMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

// LogMessageParams are the parameters for window/logMessage.
type LogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}
