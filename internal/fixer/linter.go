package fixer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/mateus/vibe-cli/internal/testing"
)

// Linter runs linters for different project types.
type Linter struct {
	workDir     string
	projectType testing.ProjectType
}

// NewLinter creates a new linter.
func NewLinter(cfg LinterConfig) *Linter {
	return &Linter{
		workDir:     cfg.WorkDir,
		projectType: cfg.ProjectType,
	}
}

// Run executes the linter and returns issues.
func (l *Linter) Run(ctx context.Context, files []string) ([]LintIssue, error) {
	switch l.projectType {
	case testing.ProjectGo:
		return l.runGoLinter(ctx, files)
	case testing.ProjectPython:
		return l.runPythonLinter(ctx, files)
	case testing.ProjectNode:
		return l.runNodeLinter(ctx, files)
	case testing.ProjectRust:
		return l.runRustLinter(ctx, files)
	default:
		return nil, fmt.Errorf("unsupported project type: %s", l.projectType)
	}
}

func (l *Linter) runGoLinter(ctx context.Context, files []string) ([]LintIssue, error) {
	// Try golangci-lint first, fall back to go vet
	issues, err := l.runGolangciLint(ctx, files)
	if err != nil {
		// Fall back to go vet
		return l.runGoVet(ctx, files)
	}
	return issues, nil
}

func (l *Linter) runGolangciLint(ctx context.Context, files []string) ([]LintIssue, error) {
	args := []string{"run", "--out-format=json"}
	if len(files) > 0 {
		args = append(args, files...)
	} else {
		args = append(args, "./...")
	}

	cmd := exec.CommandContext(ctx, "golangci-lint", args...)
	cmd.Dir = l.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // golangci-lint exits with non-zero on issues

	// Parse JSON output
	var result struct {
		Issues []struct {
			FromLinter string `json:"FromLinter"`
			Text       string `json:"Text"`
			Pos        struct {
				Filename string `json:"Filename"`
				Line     int    `json:"Line"`
				Column   int    `json:"Column"`
			} `json:"Pos"`
			SourceLines []string `json:"SourceLines"`
		} `json:"Issues"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse golangci-lint output: %w", err)
	}

	issues := make([]LintIssue, len(result.Issues))
	for i, issue := range result.Issues {
		source := ""
		if len(issue.SourceLines) > 0 {
			source = issue.SourceLines[0]
		}
		issues[i] = LintIssue{
			File:     issue.Pos.Filename,
			Line:     issue.Pos.Line,
			Column:   issue.Pos.Column,
			Severity: "error",
			Message:  issue.Text,
			Rule:     issue.FromLinter,
			Source:   source,
		}
	}

	return issues, nil
}

func (l *Linter) runGoVet(ctx context.Context, files []string) ([]LintIssue, error) {
	args := []string{"vet"}
	if len(files) > 0 {
		args = append(args, files...)
	} else {
		args = append(args, "./...")
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = l.workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run()

	return l.parseGoVetOutput(stderr.String()), nil
}

func (l *Linter) parseGoVetOutput(output string) []LintIssue {
	// Format: file.go:line:column: message
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s*(.+)$`)
	lines := strings.Split(output, "\n")
	var issues []LintIssue

	for _, line := range lines {
		if matches := re.FindStringSubmatch(line); matches != nil {
			lineNum, _ := strconv.Atoi(matches[2])
			colNum, _ := strconv.Atoi(matches[3])
			issues = append(issues, LintIssue{
				File:     matches[1],
				Line:     lineNum,
				Column:   colNum,
				Severity: "error",
				Message:  matches[4],
				Rule:     "go-vet",
			})
		}
	}

	return issues
}

func (l *Linter) runPythonLinter(ctx context.Context, files []string) ([]LintIssue, error) {
	// Try ruff first, fall back to pylint
	issues, err := l.runRuff(ctx, files)
	if err != nil {
		return l.runPylint(ctx, files)
	}
	return issues, nil
}

func (l *Linter) runRuff(ctx context.Context, files []string) ([]LintIssue, error) {
	args := []string{"check", "--output-format=json"}
	if len(files) > 0 {
		args = append(args, files...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.CommandContext(ctx, "ruff", args...)
	cmd.Dir = l.workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	_ = cmd.Run()

	// Parse JSON output
	var results []struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Filename string `json:"filename"`
		Location struct {
			Row    int `json:"row"`
			Column int `json:"column"`
		} `json:"location"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("failed to parse ruff output: %w", err)
	}

	issues := make([]LintIssue, len(results))
	for i, r := range results {
		issues[i] = LintIssue{
			File:     r.Filename,
			Line:     r.Location.Row,
			Column:   r.Location.Column,
			Severity: "error",
			Message:  r.Message,
			Rule:     r.Code,
		}
	}

	return issues, nil
}

func (l *Linter) runPylint(ctx context.Context, files []string) ([]LintIssue, error) {
	args := []string{"--output-format=json"}
	if len(files) > 0 {
		args = append(args, files...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.CommandContext(ctx, "pylint", args...)
	cmd.Dir = l.workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	_ = cmd.Run()

	// Parse JSON output
	var results []struct {
		Type      string `json:"type"`
		Symbol    string `json:"symbol"`
		Message   string `json:"message"`
		Path      string `json:"path"`
		Line      int    `json:"line"`
		Column    int    `json:"column"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("failed to parse pylint output: %w", err)
	}

	issues := make([]LintIssue, len(results))
	for i, r := range results {
		issues[i] = LintIssue{
			File:     r.Path,
			Line:     r.Line,
			Column:   r.Column,
			Severity: r.Type,
			Message:  r.Message,
			Rule:     r.Symbol,
		}
	}

	return issues, nil
}

func (l *Linter) runNodeLinter(ctx context.Context, files []string) ([]LintIssue, error) {
	args := []string{"--format=json"}
	if len(files) > 0 {
		args = append(args, files...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.CommandContext(ctx, "eslint", args...)
	cmd.Dir = l.workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	_ = cmd.Run()

	// Parse JSON output
	var results []struct {
		FilePath string `json:"filePath"`
		Messages []struct {
			RuleId   string `json:"ruleId"`
			Severity int    `json:"severity"`
			Message  string `json:"message"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("failed to parse eslint output: %w", err)
	}

	var issues []LintIssue
	for _, file := range results {
		for _, msg := range file.Messages {
			severity := "warning"
			if msg.Severity == 2 {
				severity = "error"
			}
			issues = append(issues, LintIssue{
				File:     file.FilePath,
				Line:     msg.Line,
				Column:   msg.Column,
				Severity: severity,
				Message:  msg.Message,
				Rule:     msg.RuleId,
			})
		}
	}

	return issues, nil
}

func (l *Linter) runRustLinter(ctx context.Context, files []string) ([]LintIssue, error) {
	args := []string{"clippy", "--message-format=json"}

	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Dir = l.workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	_ = cmd.Run()

	// Parse JSON lines output
	var issues []LintIssue
	lines := strings.Split(stdout.String(), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var msg struct {
			Reason  string `json:"reason"`
			Message struct {
				Code    *struct{ Code string } `json:"code"`
				Level   string                 `json:"level"`
				Message string                 `json:"message"`
				Spans   []struct {
					FileName    string `json:"file_name"`
					LineStart   int    `json:"line_start"`
					ColumnStart int    `json:"column_start"`
				} `json:"spans"`
			} `json:"message"`
		}

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Reason != "compiler-message" {
			continue
		}

		if msg.Message.Level != "warning" && msg.Message.Level != "error" {
			continue
		}

		issue := LintIssue{
			Severity: msg.Message.Level,
			Message:  msg.Message.Message,
		}

		if msg.Message.Code != nil {
			issue.Rule = msg.Message.Code.Code
		}

		if len(msg.Message.Spans) > 0 {
			issue.File = msg.Message.Spans[0].FileName
			issue.Line = msg.Message.Spans[0].LineStart
			issue.Column = msg.Message.Spans[0].ColumnStart
		}

		issues = append(issues, issue)
	}

	return issues, nil
}
