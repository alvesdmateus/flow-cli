package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestSetDefaults(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	SetDefaults()

	// Test LLM defaults
	if viper.GetString("llm.provider") != "ollama" {
		t.Errorf("llm.provider = %q, want 'ollama'", viper.GetString("llm.provider"))
	}

	if viper.GetString("llm.endpoint") != "http://localhost:11434" {
		t.Errorf("llm.endpoint = %q, want 'http://localhost:11434'", viper.GetString("llm.endpoint"))
	}

	if viper.GetFloat64("llm.temperature") != 0.7 {
		t.Errorf("llm.temperature = %v, want 0.7", viper.GetFloat64("llm.temperature"))
	}

	// Test Search defaults
	if !viper.GetBool("search.enabled") {
		t.Error("search.enabled should be true by default")
	}

	if viper.GetString("search.provider") != "searxng" {
		t.Errorf("search.provider = %q, want 'searxng'", viper.GetString("search.provider"))
	}

	if viper.GetInt("search.limit") != 10 {
		t.Errorf("search.limit = %d, want 10", viper.GetInt("search.limit"))
	}

	// Test Security defaults
	if viper.GetBool("security.auto_approve") {
		t.Error("security.auto_approve should be false by default")
	}

	deniedPaths := viper.GetStringSlice("security.denied_paths")
	if len(deniedPaths) == 0 {
		t.Error("security.denied_paths should not be empty by default")
	}

	// Test Commands defaults
	allowedCommands := viper.GetStringSlice("commands.allowed")
	if len(allowedCommands) == 0 {
		t.Error("commands.allowed should not be empty by default")
	}

	blockedCommands := viper.GetStringSlice("commands.blocked")
	if len(blockedCommands) == 0 {
		t.Error("commands.blocked should not be empty by default")
	}
}

func TestLoad(t *testing.T) {
	// Reset and set defaults
	viper.Reset()
	SetDefaults()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Verify loaded values match defaults
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("cfg.LLM.Provider = %q, want 'ollama'", cfg.LLM.Provider)
	}

	if cfg.LLM.Endpoint != "http://localhost:11434" {
		t.Errorf("cfg.LLM.Endpoint = %q, want 'http://localhost:11434'", cfg.LLM.Endpoint)
	}

	if cfg.Search.Enabled != true {
		t.Error("cfg.Search.Enabled should be true")
	}

	if cfg.Security.AutoApprove != false {
		t.Error("cfg.Security.AutoApprove should be false")
	}
}

func TestGetLLMEndpoint(t *testing.T) {
	viper.Reset()
	SetDefaults()

	endpoint := GetLLMEndpoint()
	if endpoint != "http://localhost:11434" {
		t.Errorf("GetLLMEndpoint() = %q, want 'http://localhost:11434'", endpoint)
	}

	// Test custom value
	viper.Set("llm.endpoint", "http://custom:8000")
	endpoint = GetLLMEndpoint()
	if endpoint != "http://custom:8000" {
		t.Errorf("GetLLMEndpoint() = %q, want 'http://custom:8000'", endpoint)
	}
}

func TestGetLLMModel(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Default model is llama3.2 (for auto-pull support)
	model := GetLLMModel()
	if model != "llama3.2" {
		t.Errorf("GetLLMModel() = %q, want 'llama3.2' (default)", model)
	}

	// Test custom value
	viper.Set("llm.model", "llama2:7b")
	model = GetLLMModel()
	if model != "llama2:7b" {
		t.Errorf("GetLLMModel() = %q, want 'llama2:7b'", model)
	}
}

func TestGetLLMProvider(t *testing.T) {
	viper.Reset()
	SetDefaults()

	provider := GetLLMProvider()
	if provider != "ollama" {
		t.Errorf("GetLLMProvider() = %q, want 'ollama'", provider)
	}

	// Test custom value
	viper.Set("llm.provider", "openai-compatible")
	provider = GetLLMProvider()
	if provider != "openai-compatible" {
		t.Errorf("GetLLMProvider() = %q, want 'openai-compatible'", provider)
	}
}

func TestGetLLMAPIKey(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Default API key is empty
	apiKey := GetLLMAPIKey()
	if apiKey != "" {
		t.Errorf("GetLLMAPIKey() = %q, want empty string (default)", apiKey)
	}

	// Test custom value
	viper.Set("llm.api_key", "sk-test-key")
	apiKey = GetLLMAPIKey()
	if apiKey != "sk-test-key" {
		t.Errorf("GetLLMAPIKey() = %q, want 'sk-test-key'", apiKey)
	}
}

func TestIsLLMAutoStart(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Default should be true
	if !IsLLMAutoStart() {
		t.Error("IsLLMAutoStart() should be true by default")
	}

	// Disable auto-start
	viper.Set("llm.auto_start", false)
	if IsLLMAutoStart() {
		t.Error("IsLLMAutoStart() should be false after disabling")
	}
}

func TestIsLLMAutoPull(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Default should be true
	if !IsLLMAutoPull() {
		t.Error("IsLLMAutoPull() should be true by default")
	}

	// Disable auto-pull
	viper.Set("llm.auto_pull", false)
	if IsLLMAutoPull() {
		t.Error("IsLLMAutoPull() should be false after disabling")
	}
}

func TestIsAutoApprove(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Default should be false
	if IsAutoApprove() {
		t.Error("IsAutoApprove() should be false by default")
	}

	// Enable auto-approve
	viper.Set("security.auto_approve", true)
	if !IsAutoApprove() {
		t.Error("IsAutoApprove() should be true after setting")
	}
}

func TestIsSearchEnabled(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Default should be true
	if !IsSearchEnabled() {
		t.Error("IsSearchEnabled() should be true by default")
	}

	// Disable search
	viper.Set("search.enabled", false)
	if IsSearchEnabled() {
		t.Error("IsSearchEnabled() should be false after disabling")
	}
}

func TestGetSearchProvider(t *testing.T) {
	viper.Reset()
	SetDefaults()

	provider := GetSearchProvider()
	if provider != "searxng" {
		t.Errorf("GetSearchProvider() = %q, want 'searxng'", provider)
	}

	// Test custom value
	viper.Set("search.provider", "custom")
	provider = GetSearchProvider()
	if provider != "custom" {
		t.Errorf("GetSearchProvider() = %q, want 'custom'", provider)
	}
}

func TestGetSearchEndpoint(t *testing.T) {
	viper.Reset()
	SetDefaults()

	endpoint := GetSearchEndpoint()
	if endpoint != "http://localhost:8080" {
		t.Errorf("GetSearchEndpoint() = %q, want 'http://localhost:8080'", endpoint)
	}

	// Test custom value
	viper.Set("search.endpoint", "http://searx.example.com")
	endpoint = GetSearchEndpoint()
	if endpoint != "http://searx.example.com" {
		t.Errorf("GetSearchEndpoint() = %q, want 'http://searx.example.com'", endpoint)
	}
}

func TestGetSearchLanguage(t *testing.T) {
	viper.Reset()
	SetDefaults()

	lang := GetSearchLanguage()
	if lang != "en" {
		t.Errorf("GetSearchLanguage() = %q, want 'en'", lang)
	}

	// Test custom value
	viper.Set("search.language", "es")
	lang = GetSearchLanguage()
	if lang != "es" {
		t.Errorf("GetSearchLanguage() = %q, want 'es'", lang)
	}
}

func TestGetSearchLimit(t *testing.T) {
	viper.Reset()
	SetDefaults()

	limit := GetSearchLimit()
	if limit != 10 {
		t.Errorf("GetSearchLimit() = %d, want 10", limit)
	}

	// Test custom value
	viper.Set("search.limit", 25)
	limit = GetSearchLimit()
	if limit != 25 {
		t.Errorf("GetSearchLimit() = %d, want 25", limit)
	}
}

func TestConfigStructs(t *testing.T) {
	// Test that Config structs have correct types
	cfg := &Config{
		LLM: LLMConfig{
			Provider:    "ollama",
			Endpoint:    "http://localhost:11434",
			Model:       "llama2",
			Temperature: 0.7,
		},
		Search: SearchConfig{
			Enabled:  true,
			Provider: "searxng",
			Endpoint: "http://localhost:8080",
			Language: "en",
			Limit:    10,
		},
		Security: SecurityConfig{
			ProjectDir:   "/project",
			TrustedPaths: []string{"/tmp"},
			DeniedPaths:  []string{"/etc"},
			AutoApprove:  false,
		},
		Commands: CommandsConfig{
			Allowed: []string{"ls", "cat"},
			Blocked: []string{"rm -rf"},
		},
	}

	// Verify values
	if cfg.LLM.Provider != "ollama" {
		t.Error("LLMConfig.Provider not set correctly")
	}

	if !cfg.Search.Enabled {
		t.Error("SearchConfig.Enabled not set correctly")
	}

	if cfg.Security.AutoApprove {
		t.Error("SecurityConfig.AutoApprove not set correctly")
	}

	if len(cfg.Commands.Allowed) != 2 {
		t.Errorf("CommandsConfig.Allowed length = %d, want 2", len(cfg.Commands.Allowed))
	}
}

func TestDefaultDeniedPaths_ContainsSensitive(t *testing.T) {
	viper.Reset()
	SetDefaults()

	deniedPaths := viper.GetStringSlice("security.denied_paths")

	// Check that sensitive directories are in denied paths
	sensitiveDirs := []string{".ssh", ".gnupg", ".aws", ".azure"}

	for _, sensitive := range sensitiveDirs {
		found := false
		for _, denied := range deniedPaths {
			if containsString(denied, sensitive) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("denied_paths should contain path with %q", sensitive)
		}
	}
}

func TestDefaultAllowedCommands_ContainsSafe(t *testing.T) {
	viper.Reset()
	SetDefaults()

	allowedCommands := viper.GetStringSlice("commands.allowed")

	// Check that common safe commands are allowed
	safeCommands := []string{"go build", "go test", "npm install", "git status", "ls"}

	for _, safe := range safeCommands {
		found := false
		for _, allowed := range allowedCommands {
			if allowed == safe {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("commands.allowed should contain %q", safe)
		}
	}
}

func TestDefaultBlockedCommands_ContainsDangerous(t *testing.T) {
	viper.Reset()
	SetDefaults()

	blockedCommands := viper.GetStringSlice("commands.blocked")

	// Check that dangerous commands are blocked
	dangerousCommands := []string{"rm -rf /", "sudo"}

	for _, dangerous := range dangerousCommands {
		found := false
		for _, blocked := range blockedCommands {
			if blocked == dangerous {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("commands.blocked should contain %q", dangerous)
		}
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
