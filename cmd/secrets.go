package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mateus/flow-cli/internal/secrets"
	"github.com/mateus/flow-cli/internal/ui"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Scan for secrets and sensitive data",
	Long: `Scan files and directories for hardcoded secrets and sensitive data.

This command detects:
- API keys (AWS, Google, Stripe, GitHub, GitLab, etc.)
- Private keys (RSA, OpenSSH, EC, PGP)
- Database connection strings with passwords
- Tokens (JWT, OAuth, Slack, npm)
- Hardcoded passwords and credentials

Examples:
  flow secrets scan                    # Scan current directory
  flow secrets scan ./src              # Scan specific directory
  flow secrets scan --file config.go   # Scan specific file
  flow secrets scan --severity high    # Only show high+ severity
  flow secrets check                   # Check staged files (for pre-commit)`,
}

var secretsScanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan files or directories for secrets",
	Long: `Scan files or directories for hardcoded secrets and sensitive data.

Supported file types: .go, .py, .js, .ts, .yaml, .json, .env, .sh, etc.
Ignores common directories: .git, node_modules, vendor, .venv, etc.`,
	RunE: runSecretsScan,
}

var secretsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check staged git files for secrets (pre-commit hook)",
	Long: `Check staged git files for secrets before committing.

This command is designed to be used as a pre-commit hook. It will:
1. Get the list of staged files from git
2. Scan each file for secrets
3. Exit with code 1 if any secrets are found

To use as a git pre-commit hook, add to .git/hooks/pre-commit:
  #!/bin/sh
  flow secrets check`,
	RunE: runSecretsCheck,
}

var (
	secretsFile     string
	secretsSeverity string
	secretsJSON     bool
)

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(secretsScanCmd)
	secretsCmd.AddCommand(secretsCheckCmd)

	secretsScanCmd.Flags().StringVarP(&secretsFile, "file", "f", "", "Scan a specific file")
	secretsScanCmd.Flags().StringVarP(&secretsSeverity, "severity", "s", "", "Minimum severity: low, medium, high, critical")
	secretsScanCmd.Flags().BoolVar(&secretsJSON, "json", false, "Output in JSON format")

	secretsCheckCmd.Flags().StringVarP(&secretsSeverity, "severity", "s", "high", "Minimum severity to fail: low, medium, high, critical")
}

func runSecretsScan(cmd *cobra.Command, args []string) error {
	detector := secrets.NewDetector()

	var findings []secrets.Finding
	var err error

	// Determine what to scan
	if secretsFile != "" {
		// Scan specific file
		findings, err = detector.ScanFile(secretsFile)
		if err != nil {
			return fmt.Errorf("failed to scan file: %w", err)
		}
	} else {
		// Scan directory
		scanPath := "."
		if len(args) > 0 {
			scanPath = args[0]
		}

		absPath, err := filepath.Abs(scanPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		ui.PrintInfo(fmt.Sprintf("Scanning %s for secrets...", absPath))
		fmt.Println()

		findings, err = detector.ScanDirectory(absPath)
		if err != nil {
			return fmt.Errorf("failed to scan directory: %w", err)
		}
	}

	// Filter by severity if specified
	if secretsSeverity != "" {
		severity := parseSeverity(secretsSeverity)
		findings = secrets.FilterBySeverity(findings, severity)
	}

	// Display results
	if len(findings) == 0 {
		ui.PrintSuccess("No secrets detected!")
		return nil
	}

	displayFindings(findings)

	// Return error if critical findings
	if secrets.HasCriticalFindings(findings) {
		return fmt.Errorf("critical secrets detected")
	}

	return nil
}

func runSecretsCheck(cmd *cobra.Command, args []string) error {
	// Get staged files from git
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if we're in a git repository
	gitDir := filepath.Join(workDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}

	// Get list of staged files using git diff --cached --name-only
	stagedFiles, err := getStagedFiles(workDir)
	if err != nil {
		return fmt.Errorf("failed to get staged files: %w", err)
	}

	if len(stagedFiles) == 0 {
		ui.PrintInfo("No staged files to check.")
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("Checking %d staged file(s) for secrets...", len(stagedFiles)))
	fmt.Println()

	detector := secrets.NewDetector()
	var allFindings []secrets.Finding

	for _, file := range stagedFiles {
		filePath := filepath.Join(workDir, file)

		// Skip if file doesn't exist (deleted file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		findings, err := detector.ScanFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		allFindings = append(allFindings, findings...)
	}

	// Filter by severity
	severity := parseSeverity(secretsSeverity)
	allFindings = secrets.FilterBySeverity(allFindings, severity)

	if len(allFindings) == 0 {
		ui.PrintSuccess("No secrets detected in staged files!")
		return nil
	}

	displayFindings(allFindings)

	ui.PrintError("Commit blocked: secrets detected in staged files!")
	fmt.Println()
	fmt.Println("To fix this:")
	fmt.Println("  1. Remove the secrets from the files")
	fmt.Println("  2. Use environment variables instead")
	fmt.Println("  3. Add sensitive files to .gitignore")
	fmt.Println()
	fmt.Println("To bypass this check (not recommended):")
	fmt.Println("  git commit --no-verify")

	return fmt.Errorf("secrets detected in staged files")
}

func getStagedFiles(workDir string) ([]string, error) {
	// Read git index to get staged files
	// Using a simple approach: run git diff --cached --name-only
	cmd := filepath.Join(workDir, ".git")
	if _, err := os.Stat(cmd); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a git repository")
	}

	// For simplicity, we'll read from a pipe or use exec
	// But since we want to avoid exec in tests, let's check for staged files
	// by looking at the git index

	// Alternative: look for files that would be committed
	// This is a simplified version - in production, use git command

	// Try to execute git command
	output, err := runGitCommand(workDir, "diff", "--cached", "--name-only")
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

func runGitCommand(workDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	output, err := cmd.Output()
	return string(output), err
}

func displayFindings(findings []secrets.Finding) {
	grouped := secrets.GroupByFile(findings)

	// Count by severity
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, f := range findings {
		switch f.Severity {
		case secrets.SeverityCritical:
			criticalCount++
		case secrets.SeverityHigh:
			highCount++
		case secrets.SeverityMedium:
			mediumCount++
		case secrets.SeverityLow:
			lowCount++
		}
	}

	// Print summary
	ui.PrintTitle(fmt.Sprintf("Found %d potential secret(s)", len(findings)))
	fmt.Println()

	if criticalCount > 0 {
		ui.PrintError(fmt.Sprintf("  Critical: %d", criticalCount))
	}
	if highCount > 0 {
		ui.PrintWarning(fmt.Sprintf("  High:     %d", highCount))
	}
	if mediumCount > 0 {
		ui.PrintInfo(fmt.Sprintf("  Medium:   %d", mediumCount))
	}
	if lowCount > 0 {
		fmt.Printf("  Low:      %d\n", lowCount)
	}
	fmt.Println()

	// Print by file
	for file, fileFindings := range grouped {
		fmt.Printf("--- %s ---\n", file)
		for _, f := range fileFindings {
			severityLabel := formatSeverity(f.Severity)
			fmt.Printf("  Line %d: [%s] %s\n", f.Line, severityLabel, f.Description)
			fmt.Printf("           Match: %s\n", f.Match)
			fmt.Printf("           Rule: %s\n", f.Rule)
		}
		fmt.Println()
	}
}

func formatSeverity(s secrets.Severity) string {
	switch s {
	case secrets.SeverityCritical:
		return "CRITICAL"
	case secrets.SeverityHigh:
		return "HIGH"
	case secrets.SeverityMedium:
		return "MEDIUM"
	case secrets.SeverityLow:
		return "LOW"
	default:
		return string(s)
	}
}

func parseSeverity(s string) secrets.Severity {
	switch strings.ToLower(s) {
	case "critical":
		return secrets.SeverityCritical
	case "high":
		return secrets.SeverityHigh
	case "medium":
		return secrets.SeverityMedium
	case "low":
		return secrets.SeverityLow
	default:
		return secrets.SeverityLow
	}
}
