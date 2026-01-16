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
	registry.Register(NewReadFileTool(opts.Permissions))
	registry.Register(NewWriteFileTool(opts.Permissions, ui.ShowFileDiff))
	registry.Register(NewListFilesTool(opts.Permissions))
	registry.Register(NewCreateDirectoryTool(opts.Permissions))
	registry.Register(NewDeleteFileTool(opts.Permissions))

	// Shell tools
	registry.Register(NewRunCommandTool(opts.Permissions, opts.WorkDir))

	// Search tools
	if opts.SearchClient != nil {
		registry.Register(NewWebSearchTool(opts.Permissions, opts.SearchClient))
	}
	registry.Register(NewFetchURLTool(opts.Permissions))

	// Process tools
	registry.Register(NewCheckPortTool(opts.Permissions))
	registry.Register(NewKillProcessTool(opts.Permissions))
	registry.Register(NewStartProcessTool(opts.Permissions, opts.WorkDir))
	registry.Register(NewListProcessesTool(opts.Permissions))

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
		"run_command":      "Execute a shell command",
		"web_search":       "Search the web using SearXNG",
		"fetch_url":        "Fetch content from a URL",
		"check_port":       "Check if a port is in use",
		"kill_process":     "Kill a process by PID or port",
		"start_process":    "Start a background process",
		"list_processes":   "List running processes",
	}
}
