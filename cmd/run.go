package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateus/vibe-cli/internal/config"
	"github.com/mateus/vibe-cli/internal/llm"
	"github.com/mateus/vibe-cli/internal/ui"
)

var runCmd = &cobra.Command{
	Use:   "run [prompt]",
	Short: "Run a single prompt and get a response",
	Long: `Run a single prompt against the configured LLM and display the response.
If no model is specified and none is configured, you will be prompted to select one.

Examples:
  vibe run "Explain what a goroutine is"
  vibe run --model llama3:8b "Write a hello world in Go"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCommand,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runCommand(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	prompt := strings.Join(args, " ")

	// Create LLM client
	client, err := llm.NewClient(
		config.GetLLMProvider(),
		config.GetLLMEndpoint(),
	)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Check connection
	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("cannot connect to LLM service at %s: %w", config.GetLLMEndpoint(), err)
	}

	// Determine which model to use
	model := modelFlag
	if model == "" {
		model = config.GetLLMModel()
	}

	// If still no model, prompt user to select one
	if model == "" {
		models, err := client.ListModels(ctx)
		if err != nil {
			return fmt.Errorf("failed to list models: %w", err)
		}

		if len(models) == 0 {
			return fmt.Errorf("no models available. Please pull a model first with: ollama pull <model>")
		}

		modelNames := make([]string, len(models))
		for i, m := range models {
			modelNames[i] = m.Name
		}

		selected, err := ui.SelectModel(modelNames)
		if err != nil {
			return fmt.Errorf("model selection cancelled: %w", err)
		}
		model = selected
	}

	if verbose {
		ui.PrintInfo(fmt.Sprintf("Using model: %s", model))
	}

	// Prepare messages
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "You are a helpful coding assistant. Provide clear, concise answers.",
		},
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	// Send request with streaming
	opts := llm.ChatOptions{
		Model:       model,
		Temperature: 0.7,
		Stream:      true,
	}

	chunks, err := client.Chat(ctx, messages, opts)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}

	// Print streamed response
	fmt.Println()
	for chunk := range chunks {
		if chunk.Error != nil {
			return chunk.Error
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()

	return nil
}
