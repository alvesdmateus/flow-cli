package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mateus/flow-cli/internal/llm"
)

// ReviewerConfig contains configuration for the reviewer.
type ReviewerConfig struct {
	LLMClient llm.Client
	Model     string
	Focus     string // security, performance, bugs, style, all
	Verbose   bool
}

// Reviewer performs AI-powered code reviews.
type Reviewer struct {
	client  llm.Client
	model   string
	focus   string
	verbose bool
}

// NewReviewer creates a new code reviewer.
func NewReviewer(cfg ReviewerConfig) *Reviewer {
	focus := cfg.Focus
	if focus == "" {
		focus = "all"
	}

	return &Reviewer{
		client:  cfg.LLMClient,
		model:   cfg.Model,
		focus:   focus,
		verbose: cfg.Verbose,
	}
}

// Review performs a code review on the given diff.
func (r *Reviewer) Review(ctx context.Context, diff string, info *DiffInfo) (*ReviewResult, error) {
	// Build the review prompt
	prompt := r.buildPrompt(diff, info)

	// Call the LLM
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	opts := llm.ChatOptions{
		Model:       r.model,
		Temperature: 0.3, // Lower temperature for more consistent reviews
	}

	response, err := r.client.ChatSync(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse the response
	result, err := r.parseResponse(response)
	if err != nil {
		// If parsing fails, return a basic result with the raw response
		return &ReviewResult{
			Summary: response,
			Verdict: VerdictComment,
		}, nil
	}

	return result, nil
}

func (r *Reviewer) buildPrompt(diff string, info *DiffInfo) string {
	var sb strings.Builder

	sb.WriteString("You are an expert code reviewer. Analyze the following code changes and provide a thorough review.\n\n")

	// Add focus area instructions
	switch r.focus {
	case "security":
		sb.WriteString("FOCUS: Security issues - look for vulnerabilities, injection risks, authentication/authorization problems, sensitive data exposure, etc.\n\n")
	case "performance":
		sb.WriteString("FOCUS: Performance issues - look for inefficient algorithms, memory leaks, unnecessary operations, N+1 queries, etc.\n\n")
	case "bugs":
		sb.WriteString("FOCUS: Potential bugs - look for logic errors, edge cases, null pointer issues, race conditions, etc.\n\n")
	case "style":
		sb.WriteString("FOCUS: Code style and best practices - look for naming conventions, code organization, documentation, readability, etc.\n\n")
	default:
		sb.WriteString("Review for ALL aspects: security, performance, bugs, code style, and best practices.\n\n")
	}

	// Add context about the changes
	sb.WriteString("DIFF SUMMARY:\n")
	sb.WriteString(GetDiffSummary(info))
	sb.WriteString("\n")

	// Add the actual diff
	sb.WriteString("CODE CHANGES:\n")
	sb.WriteString("```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```\n\n")

	// Add response format instructions
	sb.WriteString(`RESPONSE FORMAT:
Respond with a JSON object in the following format (and ONLY the JSON, no other text):

{
  "summary": "Brief overall summary of the changes and their quality",
  "verdict": "approve" | "request_changes" | "comment",
  "issues": [
    {
      "severity": "critical" | "high" | "medium" | "low",
      "category": "security" | "performance" | "bug" | "style" | "other",
      "message": "Description of the issue",
      "file": "path/to/file.go",
      "line": 42,
      "suggestion": "How to fix this issue"
    }
  ],
  "suggestions": [
    "General improvement suggestions that aren't issues"
  ],
  "file_reviews": [
    {
      "file": "path/to/file.go",
      "summary": "Brief summary of changes to this file",
      "notes": [
        {"line": 10, "comment": "Specific comment about this line"},
        {"comment": "General comment about the file"}
      ]
    }
  ]
}

Guidelines:
- Use "approve" if changes look good with no significant issues
- Use "request_changes" if there are critical or high severity issues
- Use "comment" if there are only suggestions or minor issues
- Be specific about file paths and line numbers when possible
- Focus on actionable feedback
- Don't nitpick minor style issues unless --focus=style was specified
`)

	if r.verbose {
		sb.WriteString("\nProvide detailed explanations for each issue and suggestion.\n")
	}

	return sb.String()
}

func (r *Reviewer) parseResponse(response string) (*ReviewResult, error) {
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

	// Find JSON object boundaries
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	var result ReviewResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse review response: %w", err)
	}

	// Validate verdict
	switch result.Verdict {
	case VerdictApprove, VerdictRequestChanges, VerdictComment:
		// Valid
	default:
		result.Verdict = VerdictComment
	}

	// Validate severities
	for i := range result.Issues {
		switch result.Issues[i].Severity {
		case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
			// Valid
		default:
			result.Issues[i].Severity = SeverityMedium
		}
	}

	return &result, nil
}
