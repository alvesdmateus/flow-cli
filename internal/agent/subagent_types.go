package agent

// SubagentType defines the type of subagent for specialized tasks
type SubagentType string

const (
	// SubagentExplorer is for codebase exploration, read-only operations
	SubagentExplorer SubagentType = "explorer"

	// SubagentCoder is for code generation and editing tasks
	SubagentCoder SubagentType = "coder"

	// SubagentReviewer is for code review and analysis
	SubagentReviewer SubagentType = "reviewer"

	// SubagentPlanner is for architecture planning and design
	SubagentPlanner SubagentType = "planner"

	// SubagentResearch is for web search and documentation lookup
	SubagentResearch SubagentType = "research"
)

// AllSubagentTypes returns all available subagent types
func AllSubagentTypes() []SubagentType {
	return []SubagentType{
		SubagentExplorer,
		SubagentCoder,
		SubagentReviewer,
		SubagentPlanner,
		SubagentResearch,
	}
}

// IsValidSubagentType checks if a string is a valid subagent type
func IsValidSubagentType(t string) bool {
	switch SubagentType(t) {
	case SubagentExplorer, SubagentCoder, SubagentReviewer, SubagentPlanner, SubagentResearch:
		return true
	default:
		return false
	}
}

// DefaultToolsForType returns the default allowed tools for a subagent type
func DefaultToolsForType(t SubagentType) []string {
	switch t {
	case SubagentExplorer:
		return []string{
			"read_file",
			"list_files",
			"grep_search",
			"code_outline",
			"find_definition",
			"find_references",
			"list_symbols",
		}
	case SubagentCoder:
		return []string{
			"read_file",
			"write_file",
			"edit_file",
			"insert_lines",
			"delete_lines",
			"list_files",
			"create_directory",
			"delete_file",
			"run_command",
			"grep_search",
		}
	case SubagentReviewer:
		return []string{
			"read_file",
			"list_files",
			"grep_search",
			"code_outline",
			"find_definition",
			"find_references",
			"list_symbols",
			"git_diff",
			"git_log",
		}
	case SubagentPlanner:
		return []string{
			"read_file",
			"list_files",
			"grep_search",
			"code_outline",
			"find_definition",
			"find_references",
			"list_symbols",
			"web_search",
			"fetch_url",
		}
	case SubagentResearch:
		return []string{
			"web_search",
			"fetch_url",
			"read_file",
			"list_files",
		}
	default:
		return []string{"read_file", "list_files"}
	}
}

// DefaultMaxTurnsForType returns the default max turns for a subagent type
func DefaultMaxTurnsForType(t SubagentType) int {
	switch t {
	case SubagentExplorer:
		return 15
	case SubagentCoder:
		return 20
	case SubagentReviewer:
		return 10
	case SubagentPlanner:
		return 15
	case SubagentResearch:
		return 10
	default:
		return 10
	}
}

// DefaultTokenBudgetForType returns the default token budget for a subagent type
func DefaultTokenBudgetForType(t SubagentType) int {
	switch t {
	case SubagentExplorer:
		return 15000
	case SubagentCoder:
		return 30000
	case SubagentReviewer:
		return 15000
	case SubagentPlanner:
		return 20000
	case SubagentResearch:
		return 10000
	default:
		return 15000
	}
}

// SubagentConfig holds configuration for creating a subagent
type SubagentConfig struct {
	Type         SubagentType // Type of subagent
	Task         string       // Task description for the subagent
	TokenBudget  int          // Max tokens for this subagent (0 = use type default)
	AllowedTools []string     // Tool whitelist (nil = use type defaults)
	SystemPrompt string       // Custom system prompt (empty = use type default)
	MaxTurns     int          // Max agentic turns (0 = default based on type)
	Temperature  float64      // LLM temperature (0 = use default 0.7)
}

// SubagentResult holds the result of a subagent execution
type SubagentResult struct {
	Success       bool     // Whether the subagent completed successfully
	Summary       string   // Concise summary for parent context
	Details       string   // Full details (not added to parent context)
	FilesModified []string // List of files changed
	TokensUsed    int      // Actual tokens consumed
	Error         error    // Error if failed
}
