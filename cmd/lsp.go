package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateus/flow-cli/internal/config"
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/logging"
	"github.com/mateus/flow-cli/internal/lsp"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Language Server Protocol server",
	Long: `Start the flow LSP server for editor integration.

The LSP server communicates via stdin/stdout using the Language Server Protocol,
enabling integration with editors like VS Code, Neovim, and JetBrains IDEs.

This command is typically invoked by editor extensions, not directly by users.

Examples:
  flow lsp
  flow lsp --debug`,
	RunE: runLSP,
}

func init() {
	rootCmd.AddCommand(lspCmd)
}

func runLSP(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logging.Debug("Starting LSP server")

	// Create handler
	handler := lsp.NewFlowHandler()

	// Try to set up LLM client for AI-powered features
	llmClient, err := llm.NewClient(
		config.GetLLMProvider(),
		config.GetLLMEndpoint(),
	)
	if err == nil {
		// Determine model
		model := modelFlag
		if model == "" {
			model = config.GetLLMModel()
		}
		if model == "" {
			model = "llama3" // Default fallback
		}

		handler.SetLLMClient(llmClient, model)
		logging.Debug("LLM client configured for LSP: provider=%s model=%s", config.GetLLMProvider(), model)
	} else {
		logging.Debug("LLM client not available: %v (AI features disabled)", err)
	}

	// Create server
	server := lsp.NewServer(handler)

	// Link handler to server for notifications
	handler.SetServer(server)

	logging.Debug("LSP server initialized, entering main loop")

	// Run the server (blocks until context cancelled or EOF)
	err = server.Run(ctx)

	logging.Debug("LSP server shutdown")

	return err
}
