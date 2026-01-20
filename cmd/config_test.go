package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "long key",
			input: "sk-1234567890abcdef",
			want:  "sk-1...cdef",
		},
		{
			name:  "short key",
			input: "short",
			want:  "****",
		},
		{
			name:  "exactly 8 chars",
			input: "12345678",
			want:  "****",
		},
		{
			name:  "9 chars",
			input: "123456789",
			want:  "1234...6789",
		},
		{
			name:  "empty",
			input: "",
			want:  "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskAPIKey(tt.input)
			if got != tt.want {
				t.Errorf("maskAPIKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfigCmd_Structure(t *testing.T) {
	if configCmd == nil {
		t.Fatal("configCmd is nil")
	}

	if configCmd.Use != "config" {
		t.Errorf("configCmd.Use = %q, want %q", configCmd.Use, "config")
	}

	// Check subcommands exist
	subcommands := configCmd.Commands()
	expectedSubs := []string{"show", "set", "provider", "init"}

	for _, expected := range expectedSubs {
		found := false
		for _, sub := range subcommands {
			if sub.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("configCmd missing subcommand %q", expected)
		}
	}
}

func TestConfigShowCmd_Structure(t *testing.T) {
	if configShowCmd == nil {
		t.Fatal("configShowCmd is nil")
	}

	if configShowCmd.Use != "show" {
		t.Errorf("configShowCmd.Use = %q, want %q", configShowCmd.Use, "show")
	}

	if configShowCmd.RunE == nil {
		t.Error("configShowCmd.RunE should not be nil")
	}
}

func TestConfigSetCmd_Structure(t *testing.T) {
	if configSetCmd == nil {
		t.Fatal("configSetCmd is nil")
	}

	if configSetCmd.Use != "set <key> <value>" {
		t.Errorf("configSetCmd.Use = %q, want %q", configSetCmd.Use, "set <key> <value>")
	}

	// Should require exactly 2 arguments
	if configSetCmd.Args == nil {
		t.Error("configSetCmd.Args should not be nil")
	}
}

func TestConfigProviderCmd_Structure(t *testing.T) {
	if configProviderCmd == nil {
		t.Fatal("configProviderCmd is nil")
	}

	if configProviderCmd.Use != "provider" {
		t.Errorf("configProviderCmd.Use = %q, want %q", configProviderCmd.Use, "provider")
	}
}

func TestConfigInitCmd_Structure(t *testing.T) {
	if configInitCmd == nil {
		t.Fatal("configInitCmd is nil")
	}

	if configInitCmd.Use != "init" {
		t.Errorf("configInitCmd.Use = %q, want %q", configInitCmd.Use, "init")
	}

	// Check --local flag exists
	localFlag := configInitCmd.Flags().Lookup("local")
	if localFlag == nil {
		t.Error("configInitCmd should have --local flag")
	}
}

func TestRunConfigShow(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Set some test values
	viper.Set("llm.provider", "test-provider")
	viper.Set("llm.endpoint", "http://test:8080")
	viper.Set("llm.model", "test-model")
	viper.Set("llm.temperature", 0.5)
	viper.Set("search.enabled", true)
	viper.Set("search.provider", "searxng")
	viper.Set("search.endpoint", "http://search:8080")
	viper.Set("search.language", "en")
	viper.Set("search.limit", 10)
	viper.Set("security.project_dir", "/test")
	viper.Set("security.auto_approve", false)
	viper.Set("security.trusted_paths", []string{"/tmp"})
	viper.Set("security.denied_paths", []string{"/etc"})
	viper.Set("commands.allowed", []string{"ls"})
	viper.Set("commands.blocked", []string{"rm"})

	// Should complete without error
	err := runConfigShow(configShowCmd, nil)
	if err != nil {
		t.Errorf("runConfigShow() error = %v", err)
	}
}

func TestRunConfigShow_WithAPIKey(t *testing.T) {
	viper.Reset()

	viper.Set("llm.provider", "openai")
	viper.Set("llm.endpoint", "https://api.openai.com")
	viper.Set("llm.api_key", "sk-test1234567890")
	viper.Set("llm.model", "gpt-4")
	viper.Set("llm.temperature", 0.7)
	viper.Set("search.enabled", false)
	viper.Set("search.provider", "")
	viper.Set("search.endpoint", "")
	viper.Set("search.language", "en")
	viper.Set("search.limit", 10)
	viper.Set("security.project_dir", ".")
	viper.Set("security.auto_approve", false)
	viper.Set("security.trusted_paths", []string{})
	viper.Set("security.denied_paths", []string{})
	viper.Set("commands.allowed", []string{})
	viper.Set("commands.blocked", []string{})

	// Should complete without error (API key should be masked in output)
	err := runConfigShow(configShowCmd, nil)
	if err != nil {
		t.Errorf("runConfigShow() with API key error = %v", err)
	}
}

func TestRunConfigSet_InvalidKey(t *testing.T) {
	err := runConfigSet(configSetCmd, []string{"invalid.key", "value"})
	if err == nil {
		t.Error("runConfigSet() should error on invalid key")
	}
}

func TestRunConfigSet_ValidKeys(t *testing.T) {
	validKeys := []string{
		"llm.provider",
		"llm.endpoint",
		"llm.model",
		"llm.api_key",
		"llm.temperature",
		"search.enabled",
		"search.provider",
		"search.endpoint",
		"search.language",
		"search.limit",
		"security.project_dir",
		"security.auto_approve",
		"commands.allowed",
		"commands.blocked",
	}

	for _, key := range validKeys {
		// Just verify the key is recognized (will fail on write, but that's ok)
		viper.Reset()
		// The function will error on writeConfig, but we just want to verify key validation
		err := runConfigSet(configSetCmd, []string{key, "test-value"})
		// If error contains "unknown configuration key", the key validation failed
		if err != nil && err.Error() != "" {
			if contains(err.Error(), "unknown configuration key") {
				t.Errorf("Key %q should be valid but was rejected", key)
			}
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
