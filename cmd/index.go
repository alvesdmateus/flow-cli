package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateus/flow-cli/internal/config"
	"github.com/mateus/flow-cli/internal/indexing"
	"github.com/mateus/flow-cli/internal/ui"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Manage the semantic search index",
	Long: `Build and manage the semantic search index for your codebase.

The index enables natural language code search using embeddings.
Requires an embedding model (default: nomic-embed-text) running in Ollama.`,
}

var indexBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build or rebuild the codebase index",
	Long: `Build the semantic search index for the current workspace.

This will:
1. Scan all supported source files in the current directory
2. Generate embeddings using the configured embedding model
3. Store embeddings in a local SQLite database

Supported file types: .go, .py, .js, .ts, .tsx, .jsx, .rs, .java, .c, .cpp, .h, .hpp, .rb, .php, .md, .txt`,
	RunE: runIndexBuild,
}

var indexStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show index statistics",
	RunE:  runIndexStatus,
}

var indexClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the index database",
	RunE:  runIndexClear,
}

var indexSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the codebase using natural language",
	Long: `Search the indexed codebase using natural language queries.

Examples:
  flow index search "function that handles authentication"
  flow index search "error handling code"
  flow index search "database connection logic"`,
	Args: cobra.ExactArgs(1),
	RunE: runIndexSearch,
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.AddCommand(indexBuildCmd)
	indexCmd.AddCommand(indexStatusCmd)
	indexCmd.AddCommand(indexClearCmd)
	indexCmd.AddCommand(indexSearchCmd)
}

func runIndexBuild(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get configuration
	endpoint := config.GetLLMEndpoint()
	embeddingModel := config.GetEmbeddingModel()

	ui.PrintInfo(fmt.Sprintf("Building index for: %s", workDir))
	ui.PrintInfo(fmt.Sprintf("Embedding model: %s", embeddingModel))
	fmt.Println()

	// Create embedding client to check connectivity
	embedClient, err := indexing.NewEmbeddingClient(endpoint, embeddingModel)
	if err != nil {
		return fmt.Errorf("failed to create embedding client: %w", err)
	}

	// Check if Ollama is running
	spinner := ui.SpinnerConnecting(endpoint)
	spinner.Start()

	if err := embedClient.Ping(ctx); err != nil {
		spinner.StopWithError("Connection failed")
		return fmt.Errorf("cannot connect to Ollama at %s: %w\nPlease ensure Ollama is running", endpoint, err)
	}
	spinner.StopWithSuccess("Connected to Ollama")

	// Check if embedding model is available
	spinner = ui.SpinnerProcessing("Checking embedding model")
	spinner.Start()

	if err := embedClient.CheckModel(ctx); err != nil {
		spinner.StopWithError("Model not available")
		fmt.Println()
		ui.PrintWarning(fmt.Sprintf("Embedding model '%s' is not installed.", embeddingModel))
		fmt.Println()
		fmt.Printf("To install it, run:\n  ollama pull %s\n\n", embeddingModel)
		return fmt.Errorf("embedding model not available")
	}
	spinner.StopWithSuccess("Embedding model ready")

	// Create indexer
	dbPath := filepath.Join(workDir, ".flow", "index.db")
	indexer, err := indexing.NewIndexer(indexing.IndexerConfig{
		Endpoint:       endpoint,
		EmbeddingModel: embeddingModel,
		DBPath:         dbPath,
		WorkDir:        workDir,
	})
	if err != nil {
		return fmt.Errorf("failed to create indexer: %w", err)
	}
	defer indexer.Close()

	// Index workspace with progress
	fmt.Println()
	spinner = ui.SpinnerProcessing("Indexing files")
	spinner.Start()

	fileCount := 0
	err = indexer.IndexWorkspace(ctx, func(path string) {
		fileCount++
		relPath, _ := filepath.Rel(workDir, path)
		spinner.UpdateMessage(fmt.Sprintf("Indexing: %s (%d files)", truncateLeft(relPath, 40), fileCount))
	})

	if err != nil {
		spinner.StopWithError("Indexing failed")
		return fmt.Errorf("indexing failed: %w", err)
	}

	spinner.StopWithSuccess("Indexing complete")

	// Show statistics
	stats := indexer.GetStats()
	fmt.Println()
	ui.PrintSuccess("Index built successfully!")
	fmt.Println()
	fmt.Printf("  Files indexed:    %d\n", stats.UniqueFiles)
	fmt.Printf("  Total documents:  %d\n", stats.TotalDocuments)
	fmt.Printf("  Database path:    %s\n", dbPath)
	fmt.Println()

	if len(stats.Languages) > 0 {
		fmt.Println("  By language:")
		for lang, count := range stats.Languages {
			fmt.Printf("    %s: %d\n", lang, count)
		}
		fmt.Println()
	}

	fmt.Println("Enable indexing in config to use semantic search tools:")
	fmt.Println("  flow config set indexing.enabled true")

	return nil
}

func runIndexStatus(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	dbPath := filepath.Join(workDir, ".flow", "index.db")

	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		ui.PrintInfo("No index found for this workspace.")
		fmt.Println()
		fmt.Println("Run 'flow index build' to create the index.")
		return nil
	}

	// Open the store to get stats
	store, err := indexing.NewVectorStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open index: %w", err)
	}
	defer store.Close()

	stats := store.Stats()

	ui.PrintTitle("Semantic Search Index Status")
	fmt.Println()
	fmt.Printf("  Database path:    %s\n", dbPath)
	fmt.Printf("  Files indexed:    %d\n", stats.UniqueFiles)
	fmt.Printf("  Total documents:  %d\n", stats.TotalDocuments)
	fmt.Println()

	if len(stats.Languages) > 0 {
		fmt.Println("  By language:")
		for lang, count := range stats.Languages {
			fmt.Printf("    %-12s %d\n", lang+":", count)
		}
		fmt.Println()
	}

	if len(stats.Kinds) > 0 {
		fmt.Println("  By symbol kind:")
		for kind, count := range stats.Kinds {
			fmt.Printf("    %-12s %d\n", kind+":", count)
		}
		fmt.Println()
	}

	// Show config status
	if config.IsIndexingEnabled() {
		ui.PrintSuccess("Indexing is enabled in config")
	} else {
		ui.PrintWarning("Indexing is disabled in config")
		fmt.Println("  Enable with: flow config set indexing.enabled true")
	}

	return nil
}

func runIndexClear(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	dbPath := filepath.Join(workDir, ".flow", "index.db")

	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		ui.PrintInfo("No index found to clear.")
		return nil
	}

	// Confirm deletion
	confirmed, err := ui.Confirm("Clear the entire index? This cannot be undone.")
	if err != nil {
		return err
	}

	if !confirmed {
		ui.PrintInfo("Operation cancelled.")
		return nil
	}

	// Remove the database file
	if err := os.Remove(dbPath); err != nil {
		return fmt.Errorf("failed to remove index database: %w", err)
	}

	ui.PrintSuccess("Index cleared successfully.")
	return nil
}

func runIndexSearch(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	query := args[0]

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	dbPath := filepath.Join(workDir, ".flow", "index.db")

	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		ui.PrintWarning("No index found. Run 'flow index build' first.")
		return nil
	}

	// Create indexer
	endpoint := config.GetLLMEndpoint()
	embeddingModel := config.GetEmbeddingModel()

	indexer, err := indexing.NewIndexer(indexing.IndexerConfig{
		Endpoint:       endpoint,
		EmbeddingModel: embeddingModel,
		DBPath:         dbPath,
		WorkDir:        workDir,
	})
	if err != nil {
		return fmt.Errorf("failed to open index: %w", err)
	}
	defer indexer.Close()

	// Perform search
	spinner := ui.SpinnerProcessing("Searching")
	spinner.Start()

	results, err := indexer.Search(ctx, query, 10, nil)
	if err != nil {
		spinner.StopWithError("Search failed")
		return fmt.Errorf("search failed: %w", err)
	}

	spinner.Stop()

	if len(results) == 0 {
		ui.PrintInfo(fmt.Sprintf("No results found for: %s", query))
		return nil
	}

	// Display results
	fmt.Println()
	ui.PrintTitle(fmt.Sprintf("Search results for: %s", query))
	fmt.Println()

	for i, r := range results {
		fmt.Printf("%d. %s", i+1, r.Document.Path)
		if r.Document.Symbol != "" {
			fmt.Printf(" - %s", r.Document.Symbol)
		}
		fmt.Printf(" (score: %.2f)\n", r.Score)

		if r.Document.Kind != "" {
			fmt.Printf("   Kind: %s\n", r.Document.Kind)
		}

		// Show content preview
		content := r.Document.Content
		if len(content) > 200 {
			content = content[:197] + "..."
		}
		// Indent content
		lines := splitLines(content)
		for _, line := range lines {
			fmt.Printf("   %s\n", line)
		}
		fmt.Println()
	}

	return nil
}

// truncateLeft truncates a string from the left if it exceeds maxLen
func truncateLeft(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "..." + s[len(s)-maxLen+3:]
}

// splitLines splits content into lines for display
func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, r := range s {
		if r == '\n' {
			if current != "" {
				lines = append(lines, current)
			}
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) > 5 {
		lines = lines[:5]
		lines = append(lines, "...")
	}
	return lines
}
