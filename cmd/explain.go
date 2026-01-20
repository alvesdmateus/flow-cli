package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateus/flow-cli/internal/config"
	"github.com/mateus/flow-cli/internal/explainer"
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/ui"
)

var explainCmd = &cobra.Command{
	Use:   "explain [file|code|error]",
	Short: "Explain code or errors using AI",
	Long: `Explain code, functions, or error messages using AI.

This command helps you understand:
- What a piece of code does
- Why an error occurred and how to fix it
- How a function or module works
- Complex algorithms or patterns

Examples:
  flow explain main.go                    # Explain entire file
  flow explain main.go:42                 # Explain specific line
  flow explain main.go:10-50              # Explain line range
  flow explain "what does this regex do?" # Explain concept
  flow explain --error "panic: nil pointer dereference"
  flow explain --function main.go:MyFunc  # Explain specific function
  cat error.log | flow explain --stdin    # Explain from stdin`,
	RunE: runExplain,
}

var (
	explainError    string
	explainFunction string
	explainStdin    bool
	explainDetailed bool
)

func init() {
	rootCmd.AddCommand(explainCmd)

	explainCmd.Flags().StringVarP(&explainError, "error", "e", "", "Explain an error message")
	explainCmd.Flags().StringVarP(&explainFunction, "function", "f", "", "Explain a specific function (file:FuncName)")
	explainCmd.Flags().BoolVar(&explainStdin, "stdin", false, "Read input from stdin")
	explainCmd.Flags().BoolVarP(&explainDetailed, "detailed", "d", false, "Provide detailed explanation")
}

func runExplain(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create LLM client
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

	// Create explainer
	exp := explainer.NewExplainer(explainer.ExplainerConfig{
		LLMClient: llmClient,
		Model:     model,
		Detailed:  explainDetailed,
	})

	// Determine what to explain
	var explanation string

	if explainStdin {
		// Read from stdin
		content, err := readStdin()
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		ui.PrintInfo("Explaining input...")
		fmt.Println()
		explanation, err = exp.ExplainCode(ctx, content, "")
		if err != nil {
			return fmt.Errorf("explanation failed: %w", err)
		}
	} else if explainError != "" {
		// Explain error
		ui.PrintInfo("Analyzing error...")
		fmt.Println()
		explanation, err = exp.ExplainError(ctx, explainError)
		if err != nil {
			return fmt.Errorf("explanation failed: %w", err)
		}
	} else if explainFunction != "" {
		// Explain function
		explanation, err = handleFunctionExplain(ctx, exp, explainFunction)
		if err != nil {
			return err
		}
	} else if len(args) > 0 {
		// Determine if it's a file or text
		input := strings.Join(args, " ")
		explanation, err = handleInputExplain(ctx, exp, input)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("please provide something to explain. See 'flow explain --help'")
	}

	// Display explanation
	ui.PrintTitle("Explanation")
	fmt.Println()
	fmt.Println(explanation)

	return nil
}

func handleFunctionExplain(ctx context.Context, exp *explainer.Explainer, funcSpec string) (string, error) {
	// Parse file:FuncName format
	parts := strings.SplitN(funcSpec, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid function specification. Use format: file.go:FunctionName")
	}

	file := parts[0]
	funcName := parts[1]

	// Read the file
	content, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("Explaining function %s in %s...", funcName, file))
	fmt.Println()

	return exp.ExplainFunction(ctx, string(content), funcName, filepath.Ext(file))
}

func handleInputExplain(ctx context.Context, exp *explainer.Explainer, input string) (string, error) {
	// Check if input looks like a file path
	if fileExists(input) {
		return handleFileExplain(ctx, exp, input)
	}

	// Check for file:line or file:line-line format
	if strings.Contains(input, ":") {
		parts := strings.SplitN(input, ":", 2)
		if fileExists(parts[0]) {
			return handleFileExplain(ctx, exp, input)
		}
	}

	// Treat as a question or code snippet
	ui.PrintInfo("Processing question...")
	fmt.Println()
	return exp.ExplainQuestion(ctx, input)
}

func handleFileExplain(ctx context.Context, exp *explainer.Explainer, fileSpec string) (string, error) {
	// Parse file:line or file:start-end format
	file := fileSpec
	startLine := 0
	endLine := 0

	if idx := strings.Index(fileSpec, ":"); idx > 0 {
		file = fileSpec[:idx]
		lineSpec := fileSpec[idx+1:]

		if strings.Contains(lineSpec, "-") {
			// Range: file:10-50
			fmt.Sscanf(lineSpec, "%d-%d", &startLine, &endLine)
		} else {
			// Single line: file:42
			fmt.Sscanf(lineSpec, "%d", &startLine)
			endLine = startLine
		}
	}

	// Read the file
	content, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Extract the relevant portion
	var codeToExplain string
	if startLine > 0 {
		// Adjust for 0-based index
		startLine--
		if endLine == 0 {
			endLine = startLine + 1
		}
		if startLine >= len(lines) {
			return "", fmt.Errorf("line %d is beyond file length", startLine+1)
		}
		if endLine > len(lines) {
			endLine = len(lines)
		}
		codeToExplain = strings.Join(lines[startLine:endLine], "\n")
		ui.PrintInfo(fmt.Sprintf("Explaining %s lines %d-%d...", file, startLine+1, endLine))
	} else {
		codeToExplain = string(content)
		ui.PrintInfo(fmt.Sprintf("Explaining %s...", file))
	}
	fmt.Println()

	return exp.ExplainCode(ctx, codeToExplain, filepath.Ext(file))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readStdin() (string, error) {
	var content strings.Builder
	buf := make([]byte, 1024)

	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			content.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return content.String(), nil
}
