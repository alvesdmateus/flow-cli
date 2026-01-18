package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ProjectConfig holds project-specific configuration
// loaded from .vibe/config.yaml in the project root
type ProjectConfig struct {
	// Inherits from global config but can override
	LLM      *LLMConfig      `yaml:"llm,omitempty"`
	Security *SecurityConfig `yaml:"security,omitempty"`
	Commands *CommandsConfig `yaml:"commands,omitempty"`
	Search   *SearchConfig   `yaml:"search,omitempty"`

	// Project-specific settings
	Project ProjectSettings `yaml:"project,omitempty"`

	// Custom prompts and templates
	Prompts PromptsConfig `yaml:"prompts,omitempty"`

	// Tool configurations
	Tools ToolsConfig `yaml:"tools,omitempty"`
}

// ProjectSettings holds project metadata
type ProjectSettings struct {
	Name        string   `yaml:"name,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Language    string   `yaml:"language,omitempty"`
	Framework   string   `yaml:"framework,omitempty"`
	TestCommand string   `yaml:"test_command,omitempty"`
	BuildCommand string  `yaml:"build_command,omitempty"`
	LintCommand string   `yaml:"lint_command,omitempty"`
	IgnoreFiles []string `yaml:"ignore_files,omitempty"`
}

// PromptsConfig holds custom prompt templates
type PromptsConfig struct {
	SystemPrompt    string            `yaml:"system_prompt,omitempty"`
	CodeReview      string            `yaml:"code_review,omitempty"`
	TestGeneration  string            `yaml:"test_generation,omitempty"`
	Documentation   string            `yaml:"documentation,omitempty"`
	Refactoring     string            `yaml:"refactoring,omitempty"`
	CustomPrompts   map[string]string `yaml:"custom,omitempty"`
}

// ToolsConfig holds tool-specific settings
type ToolsConfig struct {
	Git        GitToolConfig      `yaml:"git,omitempty"`
	Analysis   AnalysisToolConfig `yaml:"analysis,omitempty"`
	FileSystem FileSystemConfig   `yaml:"filesystem,omitempty"`
}

// GitToolConfig holds git tool settings
type GitToolConfig struct {
	AutoStage     bool   `yaml:"auto_stage,omitempty"`
	SignCommits   bool   `yaml:"sign_commits,omitempty"`
	DefaultBranch string `yaml:"default_branch,omitempty"`
}

// AnalysisToolConfig holds code analysis settings
type AnalysisToolConfig struct {
	IncludeTests    bool     `yaml:"include_tests,omitempty"`
	MaxFileSize     int      `yaml:"max_file_size,omitempty"`
	IgnorePatterns  []string `yaml:"ignore_patterns,omitempty"`
}

// FileSystemConfig holds filesystem tool settings
type FileSystemConfig struct {
	MaxReadSize   int      `yaml:"max_read_size,omitempty"`
	AllowedWrite  []string `yaml:"allowed_write,omitempty"`
	DeniedWrite   []string `yaml:"denied_write,omitempty"`
}

// ProjectConfigPath returns the path to the project config file
func ProjectConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".vibe", "config.yaml")
}

// ProjectConfigExists checks if a project config exists
func ProjectConfigExists(projectDir string) bool {
	_, err := os.Stat(ProjectConfigPath(projectDir))
	return err == nil
}

// LoadProjectConfig loads project-specific configuration
func LoadProjectConfig(projectDir string) (*ProjectConfig, error) {
	configPath := ProjectConfigPath(projectDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectConfig{}, nil
		}
		return nil, err
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveProjectConfig saves project configuration to .vibe/config.yaml
func SaveProjectConfig(projectDir string, cfg *ProjectConfig) error {
	vibeDir := filepath.Join(projectDir, ".vibe")
	if err := os.MkdirAll(vibeDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ProjectConfigPath(projectDir), data, 0644)
}

// MergeProjectConfig merges project config into the global viper config
func MergeProjectConfig(projectDir string) error {
	cfg, err := LoadProjectConfig(projectDir)
	if err != nil {
		return err
	}

	// Override global settings with project-specific ones
	if cfg.LLM != nil {
		if cfg.LLM.Provider != "" {
			viper.Set("llm.provider", cfg.LLM.Provider)
		}
		if cfg.LLM.Endpoint != "" {
			viper.Set("llm.endpoint", cfg.LLM.Endpoint)
		}
		if cfg.LLM.Model != "" {
			viper.Set("llm.model", cfg.LLM.Model)
		}
		if cfg.LLM.Temperature > 0 {
			viper.Set("llm.temperature", cfg.LLM.Temperature)
		}
	}

	if cfg.Security != nil {
		if len(cfg.Security.TrustedPaths) > 0 {
			existing := viper.GetStringSlice("security.trusted_paths")
			viper.Set("security.trusted_paths", append(existing, cfg.Security.TrustedPaths...))
		}
		if len(cfg.Security.DeniedPaths) > 0 {
			existing := viper.GetStringSlice("security.denied_paths")
			viper.Set("security.denied_paths", append(existing, cfg.Security.DeniedPaths...))
		}
	}

	if cfg.Commands != nil {
		if len(cfg.Commands.Allowed) > 0 {
			existing := viper.GetStringSlice("commands.allowed")
			viper.Set("commands.allowed", append(existing, cfg.Commands.Allowed...))
		}
		if len(cfg.Commands.Blocked) > 0 {
			existing := viper.GetStringSlice("commands.blocked")
			viper.Set("commands.blocked", append(existing, cfg.Commands.Blocked...))
		}
	}

	// Store project config for tools to access
	viper.Set("project.name", cfg.Project.Name)
	viper.Set("project.description", cfg.Project.Description)
	viper.Set("project.language", cfg.Project.Language)
	viper.Set("project.framework", cfg.Project.Framework)
	viper.Set("project.test_command", cfg.Project.TestCommand)
	viper.Set("project.build_command", cfg.Project.BuildCommand)
	viper.Set("project.lint_command", cfg.Project.LintCommand)
	viper.Set("project.ignore_files", cfg.Project.IgnoreFiles)

	// Store prompts
	if cfg.Prompts.SystemPrompt != "" {
		viper.Set("prompts.system", cfg.Prompts.SystemPrompt)
	}
	if cfg.Prompts.CodeReview != "" {
		viper.Set("prompts.code_review", cfg.Prompts.CodeReview)
	}
	if cfg.Prompts.TestGeneration != "" {
		viper.Set("prompts.test_generation", cfg.Prompts.TestGeneration)
	}
	if cfg.Prompts.Documentation != "" {
		viper.Set("prompts.documentation", cfg.Prompts.Documentation)
	}
	if cfg.Prompts.Refactoring != "" {
		viper.Set("prompts.refactoring", cfg.Prompts.Refactoring)
	}
	for key, value := range cfg.Prompts.CustomPrompts {
		viper.Set("prompts.custom."+key, value)
	}

	return nil
}

// InitProjectConfig creates a default project config file
func InitProjectConfig(projectDir string) error {
	cfg := &ProjectConfig{
		Project: ProjectSettings{
			Name:        filepath.Base(projectDir),
			Language:    "auto",
			TestCommand: "go test ./...",
			BuildCommand: "go build ./...",
			IgnoreFiles: []string{
				"*.exe", "*.dll", "*.so", "*.dylib",
				"node_modules/", "vendor/", ".git/",
				"*.log", "*.tmp", "*.cache",
			},
		},
		Prompts: PromptsConfig{
			SystemPrompt: "",
			CustomPrompts: map[string]string{},
		},
		Tools: ToolsConfig{
			Git: GitToolConfig{
				AutoStage:     false,
				SignCommits:   false,
				DefaultBranch: "main",
			},
			Analysis: AnalysisToolConfig{
				IncludeTests:   true,
				MaxFileSize:    1024 * 1024, // 1MB
				IgnorePatterns: []string{"*_test.go", "test_*.py"},
			},
			FileSystem: FileSystemConfig{
				MaxReadSize: 10 * 1024 * 1024, // 10MB
				AllowedWrite: []string{"."},
				DeniedWrite:  []string{".git/", ".vibe/"},
			},
		},
	}

	return SaveProjectConfig(projectDir, cfg)
}

// GetProjectSetting returns a project setting with fallback to default
func GetProjectSetting(key string) string {
	return viper.GetString("project." + key)
}

// GetProjectIgnoreFiles returns the list of files to ignore
func GetProjectIgnoreFiles() []string {
	return viper.GetStringSlice("project.ignore_files")
}

// GetCustomPrompt returns a custom prompt by name
func GetCustomPrompt(name string) string {
	return viper.GetString("prompts.custom." + name)
}

// GetTestCommand returns the project's test command
func GetTestCommand() string {
	cmd := viper.GetString("project.test_command")
	if cmd == "" {
		// Try to auto-detect based on project files
		if _, err := os.Stat("go.mod"); err == nil {
			return "go test ./..."
		}
		if _, err := os.Stat("package.json"); err == nil {
			return "npm test"
		}
		if _, err := os.Stat("Cargo.toml"); err == nil {
			return "cargo test"
		}
		if _, err := os.Stat("requirements.txt"); err == nil {
			return "pytest"
		}
	}
	return cmd
}

// GetBuildCommand returns the project's build command
func GetBuildCommand() string {
	cmd := viper.GetString("project.build_command")
	if cmd == "" {
		if _, err := os.Stat("go.mod"); err == nil {
			return "go build ./..."
		}
		if _, err := os.Stat("package.json"); err == nil {
			return "npm run build"
		}
		if _, err := os.Stat("Cargo.toml"); err == nil {
			return "cargo build"
		}
	}
	return cmd
}
