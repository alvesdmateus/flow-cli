package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateus/vibe-cli/internal/config"
	"github.com/mateus/vibe-cli/internal/llm"
	"github.com/mateus/vibe-cli/internal/review"
	"github.com/mateus/vibe-cli/internal/tools"
	"github.com/mateus/vibe-cli/internal/ui"
)

var reviewCmd = &cobra.Command{
	Use:   "review [file...]",
	Short: "Review code changes using AI",
	Long: `Review code changes using AI-powered analysis.

This command analyzes git diffs and provides feedback on:
- Code quality and best practices
- Potential bugs and issues
- Security concerns
- Performance considerations
- Suggested improvements

By default, it reviews all uncommitted changes. You can specify files
or use flags to review staged changes or compare with specific commits.

Examples:
  vibe review                    # Review all uncommitted changes
  vibe review --staged           # Review only staged changes
  vibe review --commit HEAD~1    # Compare with previous commit
  vibe review src/main.go        # Review specific file
  vibe review --pr 123           # Review a pull request (requires gh)
  vibe review --focus security   # Focus on security issues`,
	RunE: runReview,
}

var (
	reviewStaged  bool
	reviewCommit  string
	reviewPR      string
	reviewFocus   string
	reviewVerbose bool
)

func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().BoolVarP(&reviewStaged, "staged", "s", false, "Review only staged changes")
	reviewCmd.Flags().StringVarP(&reviewCommit, "commit", "c", "", "Compare with specific commit")
	reviewCmd.Flags().StringVar(&reviewPR, "pr", "", "Review a pull request (number or URL)")
	reviewCmd.Flags().StringVarP(&reviewFocus, "focus", "f", "", "Focus area: security, performance, bugs, style, all (default: all)")
	reviewCmd.Flags().BoolVar(&reviewVerbose, "detailed", false, "Include detailed explanations")
}

func runReview(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Check if we're in a git repo
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if !tools.IsGitRepository(workDir) {
		return fmt.Errorf("not a git repository. Run this command from a git repository")
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
		return fmt.Errorf("cannot connect to LLM service at %s: %w", config.GetLLMEndpoint(), err)
	}

	// Determine model
	model := modelFlag
	if model == "" {
		model = config.GetLLMModel()
	}

	// If still no model, prompt user to select one
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

	// Get the diff to review
	ui.PrintInfo("Gathering changes to review...")

	var diffContent string
	var diffSource string

	if reviewPR != "" {
		// Review a PR (requires gh CLI)
		diffContent, err = getPRDiff(ctx, workDir, reviewPR)
		if err != nil {
			return fmt.Errorf("failed to get PR diff: %w", err)
		}
		diffSource = fmt.Sprintf("Pull Request %s", reviewPR)
	} else {
		// Get git diff
		diffContent, err = getGitDiff(ctx, workDir, args, reviewStaged, reviewCommit)
		if err != nil {
			return fmt.Errorf("failed to get diff: %w", err)
		}

		if reviewStaged {
			diffSource = "staged changes"
		} else if reviewCommit != "" {
			diffSource = fmt.Sprintf("changes since %s", reviewCommit)
		} else if len(args) > 0 {
			diffSource = fmt.Sprintf("changes in %s", strings.Join(args, ", "))
		} else {
			diffSource = "uncommitted changes"
		}
	}

	if diffContent == "" {
		ui.PrintInfo("No changes to review.")
		return nil
	}

	// Parse the diff
	diffInfo := review.ParseDiff(diffContent)

	// Print summary
	fmt.Println()
	ui.PrintTitle(fmt.Sprintf("Reviewing %s", diffSource))
	fmt.Printf("Files changed: %d | Additions: %d | Deletions: %d\n",
		diffInfo.FilesChanged, diffInfo.Additions, diffInfo.Deletions)
	fmt.Println()

	// Create reviewer
	reviewer := review.NewReviewer(review.ReviewerConfig{
		LLMClient: llmClient,
		Model:     model,
		Focus:     reviewFocus,
		Verbose:   reviewVerbose,
	})

	// Perform review
	ui.PrintInfo("Analyzing code...")
	fmt.Println()

	result, err := reviewer.Review(ctx, diffContent, diffInfo)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Display results
	displayReviewResult(result)

	return nil
}

func getGitDiff(ctx context.Context, workDir string, files []string, staged bool, commit string) (string, error) {
	args := []string{"diff"}

	if staged {
		args = append(args, "--cached")
	}

	if commit != "" {
		args = append(args, commit)
	}

	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	runner := &tools.DefaultGitRunner{}
	return runner.Run(ctx, workDir, args...)
}

func getPRDiff(ctx context.Context, workDir string, pr string) (string, error) {
	// Use gh CLI to get PR diff
	cmd := exec.CommandContext(ctx, "gh", "pr", "diff", pr)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("gh command failed: %s (is gh CLI installed?)", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("gh command failed: %w (is gh CLI installed?)", err)
	}

	return stdout.String(), nil
}

func displayReviewResult(result *review.ReviewResult) {
	// Overall summary
	if result.Summary != "" {
		ui.PrintTitle("Summary")
		fmt.Println(result.Summary)
		fmt.Println()
	}

	// Issues by severity
	if len(result.Issues) > 0 {
		ui.PrintTitle("Issues Found")
		fmt.Println()

		// Group by severity
		critical := filterIssues(result.Issues, review.SeverityCritical)
		high := filterIssues(result.Issues, review.SeverityHigh)
		medium := filterIssues(result.Issues, review.SeverityMedium)
		low := filterIssues(result.Issues, review.SeverityLow)

		if len(critical) > 0 {
			printIssueGroup("CRITICAL", critical, ui.PrintError)
		}
		if len(high) > 0 {
			printIssueGroup("HIGH", high, ui.PrintWarning)
		}
		if len(medium) > 0 {
			printIssueGroup("MEDIUM", medium, ui.PrintInfo)
		}
		if len(low) > 0 {
			printIssueGroup("LOW", low, printDim)
		}
	}

	// Suggestions
	if len(result.Suggestions) > 0 {
		ui.PrintTitle("Suggestions")
		fmt.Println()
		for i, s := range result.Suggestions {
			fmt.Printf("%d. %s\n", i+1, s)
		}
		fmt.Println()
	}

	// File-specific feedback
	if len(result.FileReviews) > 0 {
		ui.PrintTitle("File-by-File Review")
		fmt.Println()
		for _, fr := range result.FileReviews {
			fmt.Printf("--- %s ---\n", fr.File)
			if fr.Summary != "" {
				fmt.Println(fr.Summary)
			}
			for _, note := range fr.Notes {
				if note.Line > 0 {
					fmt.Printf("  Line %d: %s\n", note.Line, note.Comment)
				} else {
					fmt.Printf("  %s\n", note.Comment)
				}
			}
			fmt.Println()
		}
	}

	// Final verdict
	fmt.Println(strings.Repeat("-", 50))
	switch result.Verdict {
	case review.VerdictApprove:
		ui.PrintSuccess("Verdict: APPROVED - Changes look good!")
	case review.VerdictRequestChanges:
		ui.PrintWarning("Verdict: REQUEST CHANGES - Please address the issues above")
	case review.VerdictComment:
		ui.PrintInfo("Verdict: COMMENT - Some suggestions to consider")
	}
}

func filterIssues(issues []review.Issue, severity review.Severity) []review.Issue {
	var result []review.Issue
	for _, i := range issues {
		if i.Severity == severity {
			result = append(result, i)
		}
	}
	return result
}

func printIssueGroup(label string, issues []review.Issue, printer func(string)) {
	printer(fmt.Sprintf("[%s]", label))
	for _, issue := range issues {
		location := ""
		if issue.File != "" {
			if issue.Line > 0 {
				location = fmt.Sprintf(" (%s:%d)", issue.File, issue.Line)
			} else {
				location = fmt.Sprintf(" (%s)", issue.File)
			}
		}
		fmt.Printf("  - %s%s\n", issue.Message, location)
		if issue.Suggestion != "" {
			fmt.Printf("    Suggestion: %s\n", issue.Suggestion)
		}
	}
	fmt.Println()
}

func printDim(text string) {
	fmt.Println(text)
}
