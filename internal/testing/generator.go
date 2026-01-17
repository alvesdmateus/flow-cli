package testing

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mateus/vibe-cli/internal/llm"
)

// GeneratorConfig contains configuration for the test generator.
type GeneratorConfig struct {
	LLMClient   llm.Client
	Model       string
	ProjectType ProjectType
}

// Generator generates tests using AI.
type Generator struct {
	client      llm.Client
	model       string
	projectType ProjectType
}

// NewGenerator creates a new test generator.
func NewGenerator(cfg GeneratorConfig) *Generator {
	return &Generator{
		client:      cfg.LLMClient,
		model:       cfg.Model,
		projectType: cfg.ProjectType,
	}
}

// GenerateTests generates tests for the given source code.
func (g *Generator) GenerateTests(ctx context.Context, filePath, sourceCode string) (*GenerateResult, error) {
	prompt := g.buildPrompt(filePath, sourceCode)

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	opts := llm.ChatOptions{
		Model:       g.model,
		Temperature: 0.3,
	}

	response, err := g.client.ChatSync(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Extract test code from response
	testCode := extractCodeBlock(response, g.projectType)
	if testCode == "" {
		testCode = response
	}

	return &GenerateResult{
		TestCode: testCode,
		TestFile: GetTestFilePath(filePath, g.projectType),
	}, nil
}

func (g *Generator) buildPrompt(filePath, sourceCode string) string {
	var sb strings.Builder

	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)

	sb.WriteString(fmt.Sprintf("Generate comprehensive unit tests for the following %s code.\n\n", g.projectType))

	// Language-specific instructions
	switch g.projectType {
	case ProjectGo:
		sb.WriteString(`Requirements:
- Use the standard "testing" package
- Follow Go testing conventions (Test* function names)
- Use table-driven tests where appropriate
- Include edge cases and error conditions
- Add clear test names that describe what is being tested
- Use t.Run for subtests when testing multiple cases
- Include t.Parallel() for independent tests

`)
	case ProjectPython:
		sb.WriteString(`Requirements:
- Use pytest for testing
- Follow Python testing conventions (test_* function names)
- Use fixtures where appropriate
- Include edge cases and error conditions
- Add clear test names that describe what is being tested
- Use parametrize for testing multiple inputs
- Include type hints

`)
	case ProjectNode:
		sb.WriteString(`Requirements:
- Use Jest for testing
- Follow JavaScript/TypeScript testing conventions
- Use describe blocks to group related tests
- Include edge cases and error conditions
- Add clear test descriptions
- Use beforeEach/afterEach for setup/teardown
- Include type annotations if TypeScript

`)
	case ProjectRust:
		sb.WriteString(`Requirements:
- Use the standard #[test] attribute
- Follow Rust testing conventions
- Use #[should_panic] for panic tests
- Include edge cases and error conditions
- Add clear test names with underscores
- Use assert!, assert_eq!, assert_ne! appropriately

`)
	}

	sb.WriteString(fmt.Sprintf("Source file: %s\n\n", fileName))
	sb.WriteString("```" + strings.TrimPrefix(ext, ".") + "\n")
	sb.WriteString(sourceCode)
	sb.WriteString("\n```\n\n")

	sb.WriteString("Generate complete, runnable test code. Output ONLY the test code in a code block, no explanations.")

	return sb.String()
}

// extractCodeBlock extracts code from a markdown code block in the response.
func extractCodeBlock(response string, projectType ProjectType) string {
	// Try to find a code block
	response = strings.TrimSpace(response)

	// Get the expected language for code blocks
	lang := ""
	switch projectType {
	case ProjectGo:
		lang = "go"
	case ProjectPython:
		lang = "python"
	case ProjectNode:
		lang = "typescript"
	case ProjectRust:
		lang = "rust"
	}

	// Try language-specific block first
	if lang != "" {
		start := strings.Index(response, "```"+lang)
		if start >= 0 {
			start += len("```" + lang)
			end := strings.Index(response[start:], "```")
			if end >= 0 {
				return strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// Try generic code block
	start := strings.Index(response, "```")
	if start >= 0 {
		// Find the end of the first line (language specifier)
		lineEnd := strings.Index(response[start+3:], "\n")
		if lineEnd >= 0 {
			codeStart := start + 3 + lineEnd + 1
			end := strings.Index(response[codeStart:], "```")
			if end >= 0 {
				return strings.TrimSpace(response[codeStart : codeStart+end])
			}
		}
	}

	return ""
}
