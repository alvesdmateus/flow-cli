package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mateus/vibe-cli/internal/llm"
)

// AnalyzerConfig contains configuration for the failure analyzer.
type AnalyzerConfig struct {
	LLMClient   llm.Client
	Model       string
	ProjectType ProjectType
	WorkDir     string
}

// Analyzer analyzes test failures and suggests fixes.
type Analyzer struct {
	client      llm.Client
	model       string
	projectType ProjectType
	workDir     string
}

// NewAnalyzer creates a new failure analyzer.
func NewAnalyzer(cfg AnalyzerConfig) *Analyzer {
	return &Analyzer{
		client:      cfg.LLMClient,
		model:       cfg.Model,
		projectType: cfg.ProjectType,
		workDir:     cfg.WorkDir,
	}
}

// AnalyzeFailures analyzes test failures and returns suggestions.
func (a *Analyzer) AnalyzeFailures(ctx context.Context, failures []TestFailure) (*AnalysisResult, error) {
	if len(failures) == 0 {
		return &AnalysisResult{}, nil
	}

	prompt := a.buildPrompt(failures)

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	opts := llm.ChatOptions{
		Model:       a.model,
		Temperature: 0.3,
	}

	response, err := a.client.ChatSync(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse the response
	result, err := a.parseResponse(response, failures)
	if err != nil {
		// If parsing fails, return a basic result
		return &AnalysisResult{
			Suggestions: []FixSuggestion{
				{
					TestName: "Analysis",
					Issue:    "Could not parse structured response",
					Fix:      response,
				},
			},
		}, nil
	}

	return result, nil
}

func (a *Analyzer) buildPrompt(failures []TestFailure) string {
	var sb strings.Builder

	sb.WriteString("Analyze the following test failures and suggest fixes.\n\n")

	for i, f := range failures {
		sb.WriteString(fmt.Sprintf("## Failure %d: %s\n", i+1, f.Name))

		if f.File != "" {
			sb.WriteString(fmt.Sprintf("File: %s", f.File))
			if f.Line > 0 {
				sb.WriteString(fmt.Sprintf(":%d", f.Line))
			}
			sb.WriteString("\n")

			// Try to read the test file for context
			testFilePath := f.File
			if !filepath.IsAbs(testFilePath) {
				testFilePath = filepath.Join(a.workDir, testFilePath)
			}
			if content, err := os.ReadFile(testFilePath); err == nil {
				sb.WriteString("\nTest file content:\n```\n")
				sb.WriteString(string(content))
				sb.WriteString("\n```\n")
			}
		}

		if f.Message != "" {
			sb.WriteString(fmt.Sprintf("\nError message: %s\n", f.Message))
		}

		if f.Output != "" {
			sb.WriteString("\nTest output:\n```\n")
			sb.WriteString(f.Output)
			sb.WriteString("\n```\n")
		}

		sb.WriteString("\n")
	}

	sb.WriteString(`Respond with a JSON array in the following format:
[
  {
    "test_name": "name of the failing test",
    "issue": "brief description of what's wrong",
    "fix": "how to fix it",
    "code": "corrected code if applicable (optional)"
  }
]

Provide practical, actionable suggestions. Focus on:
1. Why the test is failing
2. Whether the test or the implementation needs fixing
3. Specific code changes needed
`)

	return sb.String()
}

func (a *Analyzer) parseResponse(response string, failures []TestFailure) (*AnalysisResult, error) {
	// Try to extract JSON from the response
	response = strings.TrimSpace(response)

	// Remove markdown code blocks if present
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Find JSON array boundaries
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	var suggestions []struct {
		TestName string `json:"test_name"`
		Issue    string `json:"issue"`
		Fix      string `json:"fix"`
		Code     string `json:"code,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := &AnalysisResult{
		Suggestions: make([]FixSuggestion, len(suggestions)),
	}

	for i, s := range suggestions {
		result.Suggestions[i] = FixSuggestion{
			TestName: s.TestName,
			Issue:    s.Issue,
			Fix:      s.Fix,
			Code:     s.Code,
		}
	}

	return result, nil
}
