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
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/testing"
	"github.com/mateus/flow-cli/internal/ui"
)

var testCmd = &cobra.Command{
	Use:   "test [file...]",
	Short: "Run tests or generate tests using AI",
	Long: `Run tests or generate tests for your code using AI.

This command can:
- Run existing tests in your project
- Generate new tests for specific files
- Analyze test failures and suggest fixes
- Show test coverage information

Examples:
  vibe test                      # Run all tests in the project
  vibe test --generate main.go   # Generate tests for main.go
  vibe test --coverage           # Run tests with coverage report
  vibe test ./pkg/...            # Run tests for specific packages
  vibe test --fix                # Run tests and suggest fixes for failures`,
	RunE: runTest,
}

var (
	testGenerate string
	testCoverage bool
	testFix      bool
	testWatch    bool
	testTimeout  string
)

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVarP(&testGenerate, "generate", "g", "", "Generate tests for specified file")
	testCmd.Flags().BoolVarP(&testCoverage, "coverage", "c", false, "Run tests with coverage")
	testCmd.Flags().BoolVar(&testFix, "fix", false, "Analyze failures and suggest fixes")
	testCmd.Flags().BoolVarP(&testWatch, "watch", "w", false, "Watch for changes and re-run tests")
	testCmd.Flags().StringVar(&testTimeout, "timeout", "5m", "Test timeout duration")
}

func runTest(cmd *cobra.Command, args []string) error {
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

	// Handle test generation
	if testGenerate != "" {
		return runTestGeneration(ctx, workDir, projectType)
	}

	// Handle test execution
	return runTestExecution(ctx, workDir, projectType, args)
}

func runTestGeneration(ctx context.Context, workDir string, projectType testing.ProjectType) error {
	// Resolve the file path
	filePath := testGenerate
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(workDir, filePath)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", testGenerate)
	}

	// Read the source file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create LLM client
	llmClient, err := llm.NewClient(
		config.GetLLMProvider(),
		config.GetLLMEndpoint(),
	)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Check connection
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

	// Create test generator
	generator := testing.NewGenerator(testing.GeneratorConfig{
		LLMClient:   llmClient,
		Model:       model,
		ProjectType: projectType,
	})

	ui.PrintInfo(fmt.Sprintf("Generating tests for %s...", testGenerate))
	fmt.Println()

	// Generate tests
	result, err := generator.GenerateTests(ctx, filePath, string(content))
	if err != nil {
		return fmt.Errorf("test generation failed: %w", err)
	}

	// Display generated tests
	ui.PrintTitle("Generated Tests")
	fmt.Println()
	fmt.Println(result.TestCode)
	fmt.Println()

	// Determine test file path
	testFilePath := testing.GetTestFilePath(filePath, projectType)
	relTestPath, _ := filepath.Rel(workDir, testFilePath)

	// Check if test file already exists
	if _, err := os.Stat(testFilePath); err == nil {
		ui.PrintWarning(fmt.Sprintf("Test file already exists: %s", relTestPath))
		overwrite, err := ui.Confirm("Overwrite existing test file?")
		if err != nil {
			return err
		}
		if !overwrite {
			ui.PrintInfo("Test generation cancelled. Tests printed above for reference.")
			return nil
		}
	}

	// Ask to save
	save, err := ui.Confirm(fmt.Sprintf("Save tests to %s?", relTestPath))
	if err != nil {
		return err
	}

	if save {
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(testFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(testFilePath, []byte(result.TestCode), 0644); err != nil {
			return fmt.Errorf("failed to write test file: %w", err)
		}
		ui.PrintSuccess(fmt.Sprintf("Tests saved to %s", relTestPath))

		// Offer to run the tests
		runNow, err := ui.Confirm("Run the generated tests now?")
		if err != nil {
			return err
		}
		if runNow {
			return runTestExecution(ctx, workDir, projectType, []string{testFilePath})
		}
	}

	return nil
}

func runTestExecution(ctx context.Context, workDir string, projectType testing.ProjectType, args []string) error {
	// Create test runner
	runner := testing.NewRunner(testing.RunnerConfig{
		WorkDir:     workDir,
		ProjectType: projectType,
		Coverage:    testCoverage,
		Timeout:     testTimeout,
	})

	ui.PrintInfo("Running tests...")
	fmt.Println()

	// Run tests
	result, err := runner.Run(ctx, args)
	if err != nil {
		return fmt.Errorf("failed to run tests: %w", err)
	}

	// Display results
	displayTestResults(result)

	// If tests failed and --fix flag is set, analyze failures
	if !result.Passed && testFix {
		return analyzeFailures(ctx, workDir, projectType, result)
	}

	if !result.Passed {
		return fmt.Errorf("tests failed")
	}

	return nil
}

func displayTestResults(result *testing.RunResult) {
	// Summary line
	if result.Passed {
		ui.PrintSuccess(fmt.Sprintf("PASS - %d tests passed", result.TestsPassed))
	} else {
		ui.PrintError(fmt.Sprintf("FAIL - %d passed, %d failed", result.TestsPassed, result.TestsFailed))
	}

	// Duration
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Println()

	// Coverage if available
	if result.Coverage > 0 {
		coverageColor := "green"
		if result.Coverage < 50 {
			coverageColor = "red"
		} else if result.Coverage < 80 {
			coverageColor = "yellow"
		}
		_ = coverageColor // Would use for coloring in real implementation
		fmt.Printf("Coverage: %.1f%%\n", result.Coverage)
		fmt.Println()
	}

	// Failed tests details
	if len(result.Failures) > 0 {
		ui.PrintTitle("Failed Tests")
		fmt.Println()
		for _, f := range result.Failures {
			fmt.Printf("--- %s ---\n", f.Name)
			if f.File != "" {
				fmt.Printf("File: %s", f.File)
				if f.Line > 0 {
					fmt.Printf(":%d", f.Line)
				}
				fmt.Println()
			}
			if f.Message != "" {
				fmt.Printf("Error: %s\n", f.Message)
			}
			if f.Output != "" {
				fmt.Println("Output:")
				fmt.Println(f.Output)
			}
			fmt.Println()
		}
	}

	// Raw output if verbose
	if verbose && result.Output != "" {
		ui.PrintTitle("Full Output")
		fmt.Println()
		fmt.Println(result.Output)
	}
}

func analyzeFailures(ctx context.Context, workDir string, projectType testing.ProjectType, result *testing.RunResult) error {
	if len(result.Failures) == 0 {
		return nil
	}

	ui.PrintInfo("Analyzing test failures...")
	fmt.Println()

	// Create LLM client
	llmClient, err := llm.NewClient(
		config.GetLLMProvider(),
		config.GetLLMEndpoint(),
	)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	if err := llmClient.Ping(ctx); err != nil {
		return fmt.Errorf("cannot connect to LLM service: %w", err)
	}

	model := modelFlag
	if model == "" {
		model = config.GetLLMModel()
	}

	analyzer := testing.NewAnalyzer(testing.AnalyzerConfig{
		LLMClient:   llmClient,
		Model:       model,
		ProjectType: projectType,
		WorkDir:     workDir,
	})

	analysis, err := analyzer.AnalyzeFailures(ctx, result.Failures)
	if err != nil {
		return fmt.Errorf("failure analysis failed: %w", err)
	}

	// Display analysis
	ui.PrintTitle("Failure Analysis")
	fmt.Println()

	for _, suggestion := range analysis.Suggestions {
		fmt.Printf("Test: %s\n", suggestion.TestName)
		fmt.Printf("Issue: %s\n", suggestion.Issue)
		fmt.Printf("Suggestion: %s\n", suggestion.Fix)
		if suggestion.Code != "" {
			fmt.Println("Suggested code:")
			fmt.Println("```")
			fmt.Println(suggestion.Code)
			fmt.Println("```")
		}
		fmt.Println()
	}

	return nil
}
