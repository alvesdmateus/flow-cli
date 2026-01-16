package tools

import (
	"github.com/mateus/vibe-cli/internal/sandbox"
	"github.com/mateus/vibe-cli/internal/search"
	"github.com/mateus/vibe-cli/internal/ui"
)

// SetupOptions contains options for setting up the tool registry
type SetupOptions struct {
	Permissions  *sandbox.Manager
	SearchClient search.Client
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
	}
}
