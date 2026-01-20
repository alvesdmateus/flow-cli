package tools

import (
	"fmt"

	"github.com/mateus/flow-cli/internal/budget"
	"github.com/mateus/flow-cli/internal/indexing"
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/logging"
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

	// Subagent options
	LLMClient        llm.Client
	SubagentModel    string
	TokenBudget      *budget.TokenBudget
	SubagentSpawner  SubagentSpawner // Optional spawner implementation
}

// SetupRegistry creates and populates a tool registry with all available tools
func SetupRegistry(opts SetupOptions) (*Registry, error) {
	registry := NewRegistry()

	// Helper to register tools and track errors
	var registrationErrors []error
	register := func(tool Tool) {
		if err := registry.Register(tool); err != nil {
			logging.Debug("Failed to register tool %s: %v", tool.Name(), err)
			registrationErrors = append(registrationErrors, err)
		}
	}

	// Filesystem tools (core - must succeed)
	register(NewReadFileTool(opts.Permissions))
	register(NewWriteFileTool(opts.Permissions, ui.ShowFileDiff))
	register(NewListFilesTool(opts.Permissions))
	register(NewCreateDirectoryTool(opts.Permissions))
	register(NewDeleteFileTool(opts.Permissions))

	// Edit tools
	register(NewEditFileTool(opts.Permissions, ui.ShowFileDiff))
	register(NewInsertLinesTool(opts.Permissions, ui.ShowFileDiff))
	register(NewDeleteLinesTool(opts.Permissions, ui.ShowFileDiff))

	// Search tools
	register(NewGrepSearchTool(opts.Permissions, opts.WorkDir))
	if opts.SearchClient != nil {
		register(NewWebSearchTool(opts.Permissions, opts.SearchClient))
	}
	register(NewFetchURLTool(opts.Permissions))

	// Shell tools
	register(NewRunCommandTool(opts.Permissions, opts.WorkDir))

	// Process tools
	register(NewCheckPortTool(opts.Permissions))
	register(NewKillProcessTool(opts.Permissions))
	register(NewStartProcessTool(opts.Permissions, opts.WorkDir))
	register(NewListProcessesTool(opts.Permissions))

	// Git tools
	register(NewGitStatusTool(opts.Permissions, opts.WorkDir))
	register(NewGitDiffTool(opts.Permissions, opts.WorkDir))
	register(NewGitLogTool(opts.Permissions, opts.WorkDir))
	register(NewGitCommitTool(opts.Permissions, opts.WorkDir))
	register(NewGitAddTool(opts.Permissions, opts.WorkDir))
	register(NewGitBranchTool(opts.Permissions, opts.WorkDir))
	register(NewGitCheckoutTool(opts.Permissions, opts.WorkDir))

	// Code analysis tools
	register(NewCodeOutlineTool(opts.WorkDir))
	register(NewFindDefinitionTool(opts.WorkDir))
	register(NewFindReferencesTool(opts.WorkDir))
	register(NewListSymbolsTool(opts.WorkDir))

	// Semantic search tools (requires indexer)
	if opts.Indexer != nil {
		register(NewSemanticSearchTool(opts.Indexer))
		register(NewIndexStatusTool(opts.Indexer))
		register(NewReindexFileTool(opts.Indexer))
	}

	// Subagent tool (requires subagent spawner)
	if opts.SubagentSpawner != nil {
		register(NewSpawnSubagentTool(opts.SubagentSpawner))
	}

	// Log summary if there were any registration errors
	if len(registrationErrors) > 0 {
		logging.Warn("Tool registry: %d tools failed to register", len(registrationErrors))
		// Don't fail completely - some tools may still work
		return registry, fmt.Errorf("%d tools failed to register", len(registrationErrors))
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
		"spawn_subagent",
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
		"spawn_subagent":   "Spawn a specialized subagent with isolated context for specific tasks",
	}
}
