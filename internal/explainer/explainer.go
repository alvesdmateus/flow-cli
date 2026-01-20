// Package explainer provides AI-powered code and error explanations.
package explainer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mateus/flow-cli/internal/llm"
)

// ExplainerConfig contains configuration for the explainer.
type ExplainerConfig struct {
	LLMClient llm.Client
	Model     string
	Detailed  bool
}

// Explainer provides AI-powered explanations.
type Explainer struct {
	client   llm.Client
	model    string
	detailed bool
}

// NewExplainer creates a new explainer.
func NewExplainer(cfg ExplainerConfig) *Explainer {
	return &Explainer{
		client:   cfg.LLMClient,
		model:    cfg.Model,
		detailed: cfg.Detailed,
	}
}

// ExplainCode explains a piece of code.
func (e *Explainer) ExplainCode(ctx context.Context, code string, ext string) (string, error) {
	language := languageFromExt(ext)

	var sb strings.Builder
	sb.WriteString("Explain the following code")
	if language != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", language))
	}
	sb.WriteString(":\n\n```")
	if language != "" {
		sb.WriteString(language)
	}
	sb.WriteString("\n")
	sb.WriteString(code)
	sb.WriteString("\n```\n\n")

	if e.detailed {
		sb.WriteString(`Provide a detailed explanation including:
1. What the code does (high-level overview)
2. How it works (step by step)
3. Key concepts and patterns used
4. Any potential issues or improvements
5. Example usage if applicable`)
	} else {
		sb.WriteString(`Provide a clear and concise explanation of:
1. What the code does
2. How it works
3. Any notable patterns or techniques used`)
	}

	return e.query(ctx, sb.String())
}

// ExplainError explains an error message.
func (e *Explainer) ExplainError(ctx context.Context, errorMsg string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Explain the following error message and how to fix it:\n\n")
	sb.WriteString("```\n")
	sb.WriteString(errorMsg)
	sb.WriteString("\n```\n\n")

	if e.detailed {
		sb.WriteString(`Provide a detailed analysis including:
1. What the error means
2. Common causes of this error
3. Step-by-step debugging approach
4. Specific fixes with code examples
5. How to prevent this error in the future`)
	} else {
		sb.WriteString(`Provide a clear explanation of:
1. What this error means
2. The most likely cause
3. How to fix it`)
	}

	return e.query(ctx, sb.String())
}

// ExplainFunction explains a specific function in the code.
func (e *Explainer) ExplainFunction(ctx context.Context, code string, funcName string, ext string) (string, error) {
	language := languageFromExt(ext)

	// Try to extract the function from the code
	funcCode := extractFunction(code, funcName, language)
	if funcCode == "" {
		// If we can't extract it, use the whole code
		funcCode = code
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Explain the function `%s`", funcName))
	if language != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", language))
	}
	sb.WriteString(":\n\n```")
	if language != "" {
		sb.WriteString(language)
	}
	sb.WriteString("\n")
	sb.WriteString(funcCode)
	sb.WriteString("\n```\n\n")

	if e.detailed {
		sb.WriteString(`Provide a detailed explanation including:
1. Function purpose and responsibility
2. Parameters and their types
3. Return value(s) and what they represent
4. Step-by-step logic breakdown
5. Error handling approach
6. Any side effects
7. Example usage`)
	} else {
		sb.WriteString(`Explain:
1. What this function does
2. Its parameters and return value
3. How it works`)
	}

	return e.query(ctx, sb.String())
}

// ExplainQuestion answers a general programming question.
func (e *Explainer) ExplainQuestion(ctx context.Context, question string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Answer the following programming question:\n\n")
	sb.WriteString(question)
	sb.WriteString("\n\n")

	if e.detailed {
		sb.WriteString(`Provide a comprehensive answer including:
1. Direct answer to the question
2. Detailed explanation with examples
3. Common use cases
4. Best practices
5. Related concepts to explore`)
	} else {
		sb.WriteString("Provide a clear and concise answer with examples where helpful.")
	}

	return e.query(ctx, sb.String())
}

func (e *Explainer) query(ctx context.Context, prompt string) (string, error) {
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "You are an expert programmer who explains code clearly and accurately. Provide helpful explanations that are easy to understand. Use markdown formatting for better readability.",
		},
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	opts := llm.ChatOptions{
		Model:       e.model,
		Temperature: 0.3,
	}

	response, err := e.client.ChatSync(ctx, messages, opts)
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}

	return response, nil
}

// languageFromExt returns the language name from a file extension.
func languageFromExt(ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	switch ext {
	case "go":
		return "go"
	case "py":
		return "python"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "jsx":
		return "jsx"
	case "tsx":
		return "tsx"
	case "rs":
		return "rust"
	case "rb":
		return "ruby"
	case "java":
		return "java"
	case "c":
		return "c"
	case "cpp", "cc", "cxx":
		return "cpp"
	case "h", "hpp":
		return "cpp"
	case "cs":
		return "csharp"
	case "php":
		return "php"
	case "swift":
		return "swift"
	case "kt":
		return "kotlin"
	case "scala":
		return "scala"
	case "sh", "bash":
		return "bash"
	case "sql":
		return "sql"
	case "html":
		return "html"
	case "css":
		return "css"
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "xml":
		return "xml"
	case "md":
		return "markdown"
	default:
		return ""
	}
}

// extractFunction attempts to extract a function definition from code.
func extractFunction(code string, funcName string, language string) string {
	lines := strings.Split(code, "\n")

	// Build patterns based on language
	var patterns []*regexp.Regexp

	switch language {
	case "go":
		// func FuncName(...) or func (r Receiver) FuncName(...)
		patterns = append(patterns,
			regexp.MustCompile(fmt.Sprintf(`^func\s+(?:\([^)]+\)\s+)?%s\s*\(`, regexp.QuoteMeta(funcName))),
		)
	case "python":
		// def func_name(...):
		patterns = append(patterns,
			regexp.MustCompile(fmt.Sprintf(`^(\s*)def\s+%s\s*\(`, regexp.QuoteMeta(funcName))),
		)
	case "javascript", "typescript", "jsx", "tsx":
		// function funcName(...) or const funcName = (...) => or funcName(...) {
		patterns = append(patterns,
			regexp.MustCompile(fmt.Sprintf(`function\s+%s\s*\(`, regexp.QuoteMeta(funcName))),
			regexp.MustCompile(fmt.Sprintf(`(?:const|let|var)\s+%s\s*=`, regexp.QuoteMeta(funcName))),
			regexp.MustCompile(fmt.Sprintf(`%s\s*\([^)]*\)\s*{`, regexp.QuoteMeta(funcName))),
		)
	case "rust":
		// fn func_name(...) or pub fn func_name(...)
		patterns = append(patterns,
			regexp.MustCompile(fmt.Sprintf(`(?:pub\s+)?fn\s+%s\s*[<(]`, regexp.QuoteMeta(funcName))),
		)
	case "java", "kotlin", "csharp":
		// various method patterns
		patterns = append(patterns,
			regexp.MustCompile(fmt.Sprintf(`(?:public|private|protected|static|final|\s)+\s+\w+\s+%s\s*\(`, regexp.QuoteMeta(funcName))),
		)
	default:
		// Generic pattern: look for funcName followed by parentheses
		patterns = append(patterns,
			regexp.MustCompile(fmt.Sprintf(`%s\s*\(`, regexp.QuoteMeta(funcName))),
		)
	}

	// Find the start of the function
	startIdx := -1
	for i, line := range lines {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				startIdx = i
				break
			}
		}
		if startIdx >= 0 {
			break
		}
	}

	if startIdx < 0 {
		return ""
	}

	// Find the end of the function by tracking braces/indentation
	return extractFunctionBody(lines, startIdx, language)
}

// extractFunctionBody extracts the function body starting from startIdx.
func extractFunctionBody(lines []string, startIdx int, language string) string {
	if language == "python" {
		return extractPythonFunction(lines, startIdx)
	}

	// For brace-based languages
	braceCount := 0
	started := false
	endIdx := startIdx

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			if ch == '{' {
				braceCount++
				started = true
			} else if ch == '}' {
				braceCount--
			}
		}
		endIdx = i
		if started && braceCount == 0 {
			break
		}
	}

	return strings.Join(lines[startIdx:endIdx+1], "\n")
}

// extractPythonFunction extracts a Python function using indentation.
func extractPythonFunction(lines []string, startIdx int) string {
	if startIdx >= len(lines) {
		return ""
	}

	// Get the indentation of the def line
	defLine := lines[startIdx]
	defIndent := len(defLine) - len(strings.TrimLeft(defLine, " \t"))

	endIdx := startIdx
	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			endIdx = i
			continue
		}
		// Check indentation
		lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
		if lineIndent <= defIndent && strings.TrimSpace(line) != "" {
			// Found a line with same or less indentation
			break
		}
		endIdx = i
	}

	return strings.Join(lines[startIdx:endIdx+1], "\n")
}
