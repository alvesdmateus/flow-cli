package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateus/vibe-cli/internal/config"
	"github.com/mateus/vibe-cli/internal/fixer"
	"github.com/mateus/vibe-cli/internal/llm"
	"github.com/mateus/vibe-cli/internal/testing"
	"github.com/mateus/vibe-cli/internal/ui"
)

var fixCmd = &cobra.Command{
	Use:   "fix [file...]",
	Short: "Auto-fix linter errors using AI",
	Long: `Run linters and automatically fix errors using AI.

This command runs the appropriate linter for your project, analyzes
the errors, and uses AI to generate and apply fixes.

Supported linters:
- Go: golangci-lint, go vet
- Python: ruff, pylint, flake8
- Node.js: eslint
- Rust: clippy

Examples:
  vibe fix                    # Fix all linter errors in project
  vibe fix main.go            # Fix errors in specific file
  vibe fix --dry-run          # Show fixes without applying
  vibe fix --lint-only        # Only run linter, don't fix
  vibe fix --auto             # Auto-apply all fixes without prompting`,
	RunE: runFix,
}

var (
	fixDryRun   bool
	fixLintOnly bool
	fixAuto     bool
)

func init() {
	rootCmd.AddCommand(fixCmd)

	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "Show fixes without applying them")
	fixCmd.Flags().BoolVar(&fixLintOnly, "lint-only", false, "Only run linter, don't generate fixes")
	fixCmd.Flags().BoolVar(&fixAuto, "auto", false, "Auto-apply all fixes without prompting")
}

func runFix(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Detect project type
	projectType := testing.DetectProjectType(workDir)
	if projectType == testing.ProjectUnknown {
		return fmt.Errorf("could not detect project type. Supported: Go, Python, Node.js, Rust")
	}

	ui.PrintInfo(fmt.Sprintf("Detected project type: %s", projectType))

	// Create linter
	linter := fixer.NewLinter(fixer.LinterConfig{
		WorkDir:     workDir,
		ProjectType: projectType,
	})

	// Run linter
	ui.PrintInfo("Running linter...")
	fmt.Println()

	issues, err := linter.Run(ctx, args)
	if err != nil {
		return fmt.Errorf("linter failed: %w", err)
	}

	if len(issues) == 0 {
		ui.PrintSuccess("No linter issues found!")
		return nil
	}

	// Display issues
	ui.PrintTitle(fmt.Sprintf("Found %d issues", len(issues)))
	fmt.Println()

	for i, issue := range issues {
		fmt.Printf("%d. [%s] %s\n", i+1, issue.Severity, issue.Message)
		if issue.File != "" {
			fmt.Printf("   %s", issue.File)
			if issue.Line > 0 {
				fmt.Printf(":%d", issue.Line)
				if issue.Column > 0 {
					fmt.Printf(":%d", issue.Column)
				}
			}
			fmt.Println()
		}
		if issue.Rule != "" {
			fmt.Printf("   Rule: %s\n", issue.Rule)
		}
	}
	fmt.Println()

	// If lint-only, stop here
	if fixLintOnly {
		return nil
	}

	// Create LLM client for fixes
	llmClient, err := llm.NewClient(
		config.GetLLMProvider(),
		config.GetLLMEndpoint(),
	)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	ui.PrintInfo("Connecting to LLM service...")
	if err := llmClient.Ping(ctx); err != nil {
		return fmt.Errorf("cannot connect to LLM service: %w", err)
	}

	// Determine model
	model := modelFlag
	if model == "" {
		model = config.GetLLMModel()
	}
	if model == "" {
		models, err := llmClient.ListModels(ctx)
		if err != nil {
			return fmt.Errorf("failed to list models: %w", err)
		}
		if len(models) == 0 {
			return fmt.Errorf("no models available")
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

	// Generate fixes
	ui.PrintInfo("Generating fixes...")
	fmt.Println()

	fixerSvc := fixer.NewFixer(fixer.FixerConfig{
		LLMClient:   llmClient,
		Model:       model,
		ProjectType: projectType,
		WorkDir:     workDir,
	})

	fixes, err := fixerSvc.GenerateFixes(ctx, issues)
	if err != nil {
		return fmt.Errorf("failed to generate fixes: %w", err)
	}

	if len(fixes) == 0 {
		ui.PrintInfo("No automatic fixes available.")
		return nil
	}

	// Display and apply fixes
	ui.PrintTitle(fmt.Sprintf("Generated %d fixes", len(fixes)))
	fmt.Println()

	for i, fix := range fixes {
		fmt.Printf("--- Fix %d: %s ---\n", i+1, fix.Description)
		fmt.Printf("File: %s\n", fix.File)
		if fix.Line > 0 {
			fmt.Printf("Line: %d\n", fix.Line)
		}
		fmt.Println()
		fmt.Println("Change:")
		fmt.Println(fix.Diff)
		fmt.Println()

		if fixDryRun {
			continue
		}

		// Apply fix
		apply := fixAuto
		if !apply && !autoApprove {
			confirmed, err := ui.Confirm("Apply this fix?")
			if err != nil {
				return err
			}
			apply = confirmed
		}

		if apply {
			if err := fixerSvc.ApplyFix(fix); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to apply fix: %v", err))
			} else {
				ui.PrintSuccess("Fix applied!")
			}
		} else {
			ui.PrintInfo("Fix skipped.")
		}
		fmt.Println()
	}

	if fixDryRun {
		ui.PrintInfo("Dry run complete. No changes were made.")
	}

	return nil
}
