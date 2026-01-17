package fixer

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mateus/vibe-cli/internal/llm"
	"github.com/mateus/vibe-cli/internal/testing"
)

// Fixer generates and applies fixes for lint issues.
type Fixer struct {
	client      llm.Client
	model       string
	projectType testing.ProjectType
	workDir     string
}

// NewFixer creates a new fixer.
func NewFixer(cfg FixerConfig) *Fixer {
	return &Fixer{
		client:      cfg.LLMClient,
		model:       cfg.Model,
		projectType: cfg.ProjectType,
		workDir:     cfg.WorkDir,
	}
}

// GenerateFixes generates fixes for the given lint issues.
func (f *Fixer) GenerateFixes(ctx context.Context, issues []LintIssue) ([]*Fix, error) {
	// Group issues by file for efficiency
	issuesByFile := make(map[string][]LintIssue)
	for _, issue := range issues {
		issuesByFile[issue.File] = append(issuesByFile[issue.File], issue)
	}

	var fixes []*Fix

	for file, fileIssues := range issuesByFile {
		fileFixes, err := f.generateFixesForFile(ctx, file, fileIssues)
		if err != nil {
			continue // Skip files that fail
		}
		fixes = append(fixes, fileFixes...)
	}

	return fixes, nil
}

func (f *Fixer) generateFixesForFile(ctx context.Context, file string, issues []LintIssue) ([]*Fix, error) {
	// Read the file content
	filePath := file
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(f.workDir, file)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	var fixes []*Fix

	for _, issue := range issues {
		fix, err := f.generateFixForIssue(ctx, file, lines, issue)
		if err != nil {
			continue
		}
		if fix != nil {
			fixes = append(fixes, fix)
		}
	}

	return fixes, nil
}

func (f *Fixer) generateFixForIssue(ctx context.Context, file string, lines []string, issue LintIssue) (*Fix, error) {
	// Get context around the issue
	startLine := issue.Line - 3
	if startLine < 0 {
		startLine = 0
	}
	endLine := issue.Line + 3
	if endLine > len(lines) {
		endLine = len(lines)
	}

	contextLines := lines[startLine:endLine]
	contextCode := strings.Join(contextLines, "\n")

	// Build prompt
	prompt := f.buildPrompt(file, issue, contextCode, startLine+1)

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	opts := llm.ChatOptions{
		Model:       f.model,
		Temperature: 0.2,
	}

	response, err := f.client.ChatSync(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse the response
	fix := f.parseFixResponse(response, file, issue)
	return fix, nil
}

func (f *Fixer) buildPrompt(file string, issue LintIssue, contextCode string, startLine int) string {
	var sb strings.Builder

	sb.WriteString("Fix the following linter issue.\n\n")

	sb.WriteString(fmt.Sprintf("File: %s\n", file))
	sb.WriteString(fmt.Sprintf("Line: %d\n", issue.Line))
	sb.WriteString(fmt.Sprintf("Issue: %s\n", issue.Message))
	if issue.Rule != "" {
		sb.WriteString(fmt.Sprintf("Rule: %s\n", issue.Rule))
	}
	sb.WriteString("\n")

	sb.WriteString("Code context (starting at line ")
	sb.WriteString(fmt.Sprintf("%d):\n", startLine))
	sb.WriteString("```\n")
	sb.WriteString(contextCode)
	sb.WriteString("\n```\n\n")

	sb.WriteString(`Respond with ONLY the fixed line(s) of code.
- Output the corrected code that fixes the issue
- Only include the line(s) that need to change
- Do not include explanations, just the code
- If the fix requires removing the line, output "DELETE_LINE"
- Keep the same indentation
`)

	return sb.String()
}

func (f *Fixer) parseFixResponse(response string, file string, issue LintIssue) *Fix {
	response = strings.TrimSpace(response)

	// Remove code blocks if present
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			response = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	if response == "" {
		return nil
	}

	fix := &Fix{
		Issue:       &issue,
		File:        file,
		Line:        issue.Line,
		Description: fmt.Sprintf("Fix: %s", issue.Message),
		OldCode:     issue.Source,
		NewCode:     response,
	}

	// Generate diff
	if response == "DELETE_LINE" {
		fix.Diff = fmt.Sprintf("- %s", issue.Source)
		fix.NewCode = ""
	} else {
		fix.Diff = fmt.Sprintf("- %s\n+ %s", issue.Source, response)
	}

	return fix
}

// ApplyFix applies a fix to the file.
func (f *Fixer) ApplyFix(fix *Fix) error {
	filePath := fix.File
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(f.workDir, fix.File)
	}

	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Apply the fix
	lineIdx := fix.Line - 1
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line %d out of range", fix.Line)
	}

	if fix.NewCode == "" {
		// Delete the line
		lines = append(lines[:lineIdx], lines[lineIdx+1:]...)
	} else {
		// Replace the line
		lines[lineIdx] = fix.NewCode
	}

	// Write the file back
	output := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
