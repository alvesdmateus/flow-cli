package tools

import (
	"github.com/mateus/flow-cli/internal/indexing"
	"github.com/mateus/flow-cli/internal/sandbox"
	"github.com/mateus/flow-cli/internal/search"
	"github.com/mateus/flow-cli/internal/ui"
)

// SetupOptions contains options for setting up the tool registry
type SetupOptions struct {
	Permissions  *sandbox.Manager
	SearchClient search.Client
	Indexer      *indexing.Indexer
	WorkDir      string
}

// SetupRegistry creates and populates a tool registry with all available tools
func SetupRegistry(opts SetupOptions) (*Registry, error) {
	registry := NewRegistry()

	// Filesystem tools
	_ = registry.Register(NewReadFileTool(opts.Permissions))
	_ = registry.Register(NewWriteFileTool(opts.Permissions, ui.ShowFileDiff))
	_ = registry.Register(NewListFilesTool(opts.Permissions))
	_ = registry.Register(NewCreateDirectoryTool(opts.Permissions))
	_ = registry.Register(NewDeleteFileTool(opts.Permissions))

	// Edit tools
	_ = registry.Register(NewEditFileTool(opts.Permissions, ui.ShowFileDiff))
	_ = registry.Register(NewInsertLinesTool(opts.Permissions, ui.ShowFileDiff))
	_ = registry.Register(NewDeleteLinesTool(opts.Permissions, ui.ShowFileDiff))

	// Search tools
	_ = registry.Register(NewGrepSearchTool(opts.Permissions, opts.WorkDir))
	if opts.SearchClient != nil {
		_ = registry.Register(NewWebSearchTool(opts.Permissions, opts.SearchClient))
	}
	_ = registry.Register(NewFetchURLTool(opts.Permissions))

	// Shell tools
	_ = registry.Register(NewRunCommandTool(opts.Permissions, opts.WorkDir))

	// Process tools
	_ = registry.Register(NewCheckPortTool(opts.Permissions))
	_ = registry.Register(NewKillProcessTool(opts.Permissions))
	_ = registry.Register(NewStartProcessTool(opts.Permissions, opts.WorkDir))
	_ = registry.Register(NewListProcessesTool(opts.Permissions))

	// Git tools
	_ = registry.Register(NewGitStatusTool(opts.Permissions, opts.WorkDir))
	_ = registry.Register(NewGitDiffTool(opts.Permissions, opts.WorkDir))
	_ = registry.Register(NewGitLogTool(opts.Permissions, opts.WorkDir))
	_ = registry.Register(NewGitCommitTool(opts.Permissions, opts.WorkDir))
	_ = registry.Register(NewGitAddTool(opts.Permissions, opts.WorkDir))
	_ = registry.Register(NewGitBranchTool(opts.Permissions, opts.WorkDir))
	_ = registry.Register(NewGitCheckoutTool(opts.Permissions, opts.WorkDir))

	// Code analysis tools
	_ = registry.Register(NewCodeOutlineTool(opts.WorkDir))
	_ = registry.Register(NewFindDefinitionTool(opts.WorkDir))
	_ = registry.Register(NewFindReferencesTool(opts.WorkDir))
	_ = registry.Register(NewListSymbolsTool(opts.WorkDir))

	// Semantic search tools (requires indexer)
	if opts.Indexer != nil {
		_ = registry.Register(NewSemanticSearchTool(opts.Indexer))
		_ = registry.Register(NewIndexStatusTool(opts.Indexer))
		_ = registry.Register(NewReindexFileTool(opts.Indexer))
	}

	return registry, nil
}

// AllToolNames returns the names of all available tools
func AllToolNames() []string {
	return []string{
		"read_file",
		"write_file",
		"list_files",
		"create_directory",
		"delete_file",
		"edit_file",
		"insert_lines",
		"delete_lines",
		"grep_search",
		"run_command",
		"web_search",
		"fetch_url",
		"check_port",
		"kill_process",
		"start_process",
		"list_processes",
		"git_status",
		"git_diff",
		"git_log",
		"git_commit",
		"git_add",
		"git_branch",
		"git_checkout",
		"code_outline",
		"find_definition",
		"find_references",
		"list_symbols",
		"semantic_search",
		"index_status",
		"reindex_file",
	}
}

// ToolDescriptions returns descriptions of all available tools
func ToolDescriptions() map[string]string {
	return map[string]string{
		"read_file":        "Read the contents of a file",
		"write_file":       "Write content to a file",
		"list_files":       "List files in a directory",
		"create_directory": "Create a new directory",
		"delete_file":      "Delete a file or empty directory",
		"edit_file":        "Make surgical text replacements in a file",
		"insert_lines":     "Insert lines at a specific position in a file",
		"delete_lines":     "Delete a range of lines from a file",
		"grep_search":      "Search for regex patterns across the codebase",
		"run_command":      "Execute a shell command",
		"web_search":       "Search the web using SearXNG",
		"fetch_url":        "Fetch and extract content from a URL",
		"check_port":       "Check if a port is in use",
		"kill_process":     "Kill a process by PID or port",
		"start_process":    "Start a background process",
		"list_processes":   "List running processes",
		"git_status":       "Show git working tree status",
		"git_diff":         "Show changes between commits or working tree",
		"git_log":          "Show commit history",
		"git_commit":       "Create a new commit",
		"git_add":          "Stage files for commit",
		"git_branch":       "List, create, or delete branches",
		"git_checkout":     "Switch branches or restore files",
		"code_outline":     "Parse source file and show structure (functions, classes, types)",
		"find_definition":  "Find where a symbol is defined in the codebase",
		"find_references":  "Find all references to a symbol in the codebase",
		"list_symbols":     "List all symbols (functions, classes, types) in a directory",
		"semantic_search":  "Search the codebase using natural language queries",
		"index_status":     "Show the status of the semantic search index",
		"reindex_file":     "Re-index a specific file after modification",
	}
}
