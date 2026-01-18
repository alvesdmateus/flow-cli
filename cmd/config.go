package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage flow configuration",
	Long: `View and modify flow configuration settings.

Use subcommands to configure different aspects:
  flow config show          - Display current configuration
  flow config set           - Set a configuration value
  flow config provider      - Configure LLM provider interactively
  flow config init          - Initialize configuration file`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Examples:
  flow config set llm.provider ollama
  flow config set llm.endpoint http://localhost:11434
  flow config set llm.model llama3:8b
  flow config set llm.api_key sk-xxx
  flow config set security.auto_approve true`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Configure LLM provider interactively",
	Long: `Interactively configure the LLM provider.

This will guide you through selecting a provider and configuring
the endpoint, model, and API key if needed.`,
	RunE: runConfigProvider,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long: `Create a new configuration file with default values.

The configuration file will be created at:
  - $HOME/.flow/config.yaml (default)
  - Or in the current directory with --local flag`,
	RunE: runConfigInit,
}

var localConfig bool

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configProviderCmd)
	configCmd.AddCommand(configInitCmd)

	configInitCmd.Flags().BoolVar(&localConfig, "local", false, "Create config in current directory")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	fmt.Println("Current Configuration:")
	fmt.Println("======================")
	fmt.Println()

	// LLM settings
	fmt.Println("LLM Settings:")
	fmt.Printf("  Provider:    %s\n", viper.GetString("llm.provider"))
	fmt.Printf("  Endpoint:    %s\n", viper.GetString("llm.endpoint"))
	fmt.Printf("  Model:       %s\n", viper.GetString("llm.model"))
	fmt.Printf("  Temperature: %.1f\n", viper.GetFloat64("llm.temperature"))
	if viper.GetString("llm.api_key") != "" {
		fmt.Printf("  API Key:     %s\n", maskAPIKey(viper.GetString("llm.api_key")))
	}
	fmt.Println()

	// Search settings
	fmt.Println("Search Settings:")
	fmt.Printf("  Enabled:  %t\n", viper.GetBool("search.enabled"))
	fmt.Printf("  Provider: %s\n", viper.GetString("search.provider"))
	fmt.Printf("  Endpoint: %s\n", viper.GetString("search.endpoint"))
	fmt.Printf("  Language: %s\n", viper.GetString("search.language"))
	fmt.Printf("  Limit:    %d\n", viper.GetInt("search.limit"))
	fmt.Println()

	// Security settings
	fmt.Println("Security Settings:")
	fmt.Printf("  Project Dir:   %s\n", viper.GetString("security.project_dir"))
	fmt.Printf("  Auto Approve:  %t\n", viper.GetBool("security.auto_approve"))
	fmt.Printf("  Trusted Paths: %v\n", viper.GetStringSlice("security.trusted_paths"))
	fmt.Printf("  Denied Paths:  %v\n", viper.GetStringSlice("security.denied_paths"))
	fmt.Println()

	// Commands settings
	fmt.Println("Command Settings:")
	fmt.Printf("  Allowed: %v\n", viper.GetStringSlice("commands.allowed"))
	fmt.Printf("  Blocked: %v\n", viper.GetStringSlice("commands.blocked"))

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Validate key exists
	validKeys := []string{
		"llm.provider", "llm.endpoint", "llm.model", "llm.api_key", "llm.temperature",
		"search.enabled", "search.provider", "search.endpoint", "search.language", "search.limit",
		"security.project_dir", "security.auto_approve",
		"commands.allowed", "commands.blocked",
	}

	isValid := false
	for _, k := range validKeys {
		if k == key {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("unknown configuration key: %s\nValid keys: %s", key, strings.Join(validKeys, ", "))
	}

	// Set the value
	viper.Set(key, value)

	// Write to config file
	if err := writeConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Set %s = %s", key, value))
	return nil
}

func runConfigProvider(cmd *cobra.Command, args []string) error {
	presets := llm.GetProviderPresets()

	// Build options list
	options := make([]string, len(presets)+1)
	for i, p := range presets {
		options[i] = fmt.Sprintf("%s - %s", p.Name, p.Description)
	}
	options[len(presets)] = "custom - Custom OpenAI-compatible endpoint"

	// Select provider
	fmt.Println("Select your LLM provider:")
	fmt.Println()

	choice, err := ui.AskQuestion("Provider:", options)
	if err != nil {
		return err
	}

	// Parse selection
	var selectedPreset *llm.ProviderPreset
	for _, p := range presets {
		if strings.HasPrefix(choice, p.Name+" -") {
			selectedPreset = &p
			break
		}
	}

	var provider, endpoint, apiKey string

	if selectedPreset != nil {
		provider = selectedPreset.Name
		endpoint = selectedPreset.Endpoint

		// Ask for custom endpoint if user wants to change default
		fmt.Printf("\nDefault endpoint: %s\n", endpoint)
		useDefault, err := ui.Confirm("Use default endpoint?")
		if err != nil {
			return err
		}

		if !useDefault {
			endpoint, err = ui.PromptInput("Enter endpoint URL:")
			if err != nil {
				return err
			}
		}

		// Ask for API key if required
		if selectedPreset.RequiresKey {
			apiKey, err = ui.PromptInput("Enter API key:")
			if err != nil {
				return err
			}
		}
	} else {
		// Custom provider
		provider = "openai-compatible"

		endpoint, err = ui.PromptInput("Enter endpoint URL (e.g., http://localhost:8080):")
		if err != nil {
			return err
		}

		needsKey, err := ui.Confirm("Does this endpoint require an API key?")
		if err != nil {
			return err
		}

		if needsKey {
			apiKey, err = ui.PromptInput("Enter API key:")
			if err != nil {
				return err
			}
		}
	}

	// Set values
	viper.Set("llm.provider", provider)
	viper.Set("llm.endpoint", endpoint)
	if apiKey != "" {
		viper.Set("llm.api_key", apiKey)
	}

	// Write config
	if err := writeConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	ui.PrintSuccess("Provider configured successfully!")
	fmt.Println()
	fmt.Printf("  Provider: %s\n", provider)
	fmt.Printf("  Endpoint: %s\n", endpoint)
	if apiKey != "" {
		fmt.Printf("  API Key:  %s\n", maskAPIKey(apiKey))
	}

	// Offer to test connection
	fmt.Println()
	testConn, err := ui.Confirm("Would you like to test the connection?")
	if err != nil {
		return err
	}

	if testConn {
		return testProviderConnection(provider, endpoint, apiKey)
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	var configPath string

	if localConfig {
		configPath = "config.yaml"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		flowDir := filepath.Join(home, ".flow")
		if err := os.MkdirAll(flowDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		configPath = filepath.Join(flowDir, "config.yaml")
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		overwrite, err := ui.Confirm(fmt.Sprintf("Config file already exists at %s. Overwrite?", configPath))
		if err != nil {
			return err
		}
		if !overwrite {
			ui.PrintInfo("Configuration not modified.")
			return nil
		}
	}

	// Write default config
	configContent := `# Vibe CLI Configuration
# See https://github.com/mateus/flow-cli for documentation

# LLM Provider Settings
llm:
  # Provider: ollama, openai-compatible, localai, lmstudio, vllm, textgen, openai
  provider: ollama

  # LLM API endpoint
  endpoint: http://localhost:11434

  # Model to use (leave empty to select interactively)
  model: ""

  # API key (required for OpenAI and some providers)
  api_key: ""

  # Temperature for generation (0.0 - 1.0)
  temperature: 0.7

# Web Search Settings (SearXNG)
search:
  enabled: true
  provider: searxng
  endpoint: http://localhost:8080
  language: en
  limit: 10

# Security Settings
security:
  # Base directory for file operations (. = current directory)
  project_dir: "."

  # Additional paths that can be read
  trusted_paths: []

  # Paths that are never accessible
  denied_paths:
    - ~/.ssh
    - ~/.gnupg
    - ~/.aws
    - ~/.azure
    - ~/.config/gcloud

  # Skip confirmation prompts (use with caution)
  auto_approve: false

# Command Execution Settings
commands:
  # Commands that are always allowed
  allowed:
    - go build
    - go test
    - go run
    - go mod
    - npm install
    - npm test
    - npm run
    - npm start
    - yarn install
    - yarn test
    - yarn run
    - yarn start
    - cargo build
    - cargo test
    - cargo run
    - git status
    - git diff
    - git log
    - git branch
    - ls
    - dir
    - cat
    - head
    - tail
    - grep

  # Commands that are never allowed
  blocked:
    - rm -rf /
    - rm -rf ~
    - sudo
    - chmod 777
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Configuration file created at %s", configPath))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit the config file to set your LLM provider and endpoint")
	fmt.Println("  2. Or run 'flow config provider' for interactive setup")
	fmt.Println("  3. Run 'flow run \"hello\"' to test the connection")

	return nil
}

func testProviderConnection(provider, endpoint, apiKey string) error {
	ui.PrintInfo("Testing connection...")

	client, err := llm.NewClientWithConfig(llm.ClientConfig{
		Provider: provider,
		Endpoint: endpoint,
		APIKey:   apiKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		ui.PrintError(fmt.Sprintf("Connection failed: %v", err))
		return nil
	}

	ui.PrintSuccess("Connection successful!")

	// List available models
	models, err := client.ListModels(ctx)
	if err != nil {
		ui.PrintInfo(fmt.Sprintf("Could not list models: %v", err))
		return nil
	}

	if len(models) > 0 {
		fmt.Println()
		fmt.Println("Available models:")
		for i, m := range models {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(models)-10)
				break
			}
			fmt.Printf("  - %s\n", m.Name)
		}
	}

	return nil
}

func writeConfig() error {
	// Determine config file path
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// Create default config location
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		flowDir := filepath.Join(home, ".flow")
		if err := os.MkdirAll(flowDir, 0755); err != nil {
			return err
		}

		configFile = filepath.Join(flowDir, "config.yaml")
	}

	return viper.WriteConfigAs(configFile)
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
