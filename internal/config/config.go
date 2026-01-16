package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	LLM      LLMConfig      `mapstructure:"llm"`
	Search   SearchConfig   `mapstructure:"search"`
	Security SecurityConfig `mapstructure:"security"`
	Commands CommandsConfig `mapstructure:"commands"`
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	Endpoint    string  `mapstructure:"endpoint"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
}

// SearchConfig holds web search configuration
type SearchConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"`
	Endpoint string `mapstructure:"endpoint"`
	Language string `mapstructure:"language"`
	Limit    int    `mapstructure:"limit"`
}

// SecurityConfig holds security and permission settings
type SecurityConfig struct {
	ProjectDir   string   `mapstructure:"project_dir"`
	TrustedPaths []string `mapstructure:"trusted_paths"`
	DeniedPaths  []string `mapstructure:"denied_paths"`
	AutoApprove  bool     `mapstructure:"auto_approve"`
}

// CommandsConfig holds command allowlist/blocklist
type CommandsConfig struct {
	Allowed []string `mapstructure:"allowed"`
	Blocked []string `mapstructure:"blocked"`
}

// SetDefaults sets default configuration values
func SetDefaults() {
	home, _ := os.UserHomeDir()

	// LLM defaults
	viper.SetDefault("llm.provider", "ollama")
	viper.SetDefault("llm.endpoint", "http://localhost:11434")
	viper.SetDefault("llm.model", "")
	viper.SetDefault("llm.temperature", 0.7)

	// Search defaults
	viper.SetDefault("search.enabled", true)
	viper.SetDefault("search.provider", "searxng")
	viper.SetDefault("search.endpoint", "http://localhost:8080")
	viper.SetDefault("search.language", "en")
	viper.SetDefault("search.limit", 10)

	// Security defaults
	viper.SetDefault("security.project_dir", ".")
	viper.SetDefault("security.trusted_paths", []string{})
	viper.SetDefault("security.denied_paths", []string{
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".gnupg"),
		filepath.Join(home, ".aws"),
		filepath.Join(home, ".azure"),
		filepath.Join(home, ".config", "gcloud"),
	})
	viper.SetDefault("security.auto_approve", false)

	// Commands defaults
	viper.SetDefault("commands.allowed", []string{
		"go build", "go test", "go run", "go mod",
		"npm install", "npm test", "npm run", "npm start",
		"yarn install", "yarn test", "yarn run", "yarn start",
		"cargo build", "cargo test", "cargo run",
		"git status", "git diff", "git log", "git branch",
		"ls", "dir", "cat", "head", "tail", "grep",
	})
	viper.SetDefault("commands.blocked", []string{
		"rm -rf /",
		"rm -rf ~",
		"sudo",
		"chmod 777",
		":(){ :|:& };:",
	})
}

// Load returns the current configuration
func Load() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetLLMEndpoint returns the configured LLM endpoint
func GetLLMEndpoint() string {
	return viper.GetString("llm.endpoint")
}

// GetLLMModel returns the configured model
func GetLLMModel() string {
	return viper.GetString("llm.model")
}

// GetLLMProvider returns the configured provider
func GetLLMProvider() string {
	return viper.GetString("llm.provider")
}

// IsAutoApprove returns whether auto-approve is enabled
func IsAutoApprove() bool {
	return viper.GetBool("security.auto_approve")
}

// IsSearchEnabled returns whether web search is enabled
func IsSearchEnabled() bool {
	return viper.GetBool("search.enabled")
}

// GetSearchProvider returns the configured search provider
func GetSearchProvider() string {
	return viper.GetString("search.provider")
}

// GetSearchEndpoint returns the configured search endpoint
func GetSearchEndpoint() string {
	return viper.GetString("search.endpoint")
}

// GetSearchLanguage returns the configured search language
func GetSearchLanguage() string {
	return viper.GetString("search.language")
}

// GetSearchLimit returns the configured search result limit
func GetSearchLimit() int {
	return viper.GetInt("search.limit")
}
