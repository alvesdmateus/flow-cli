package ui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// OutputType represents the type of output being rendered
type OutputType int

const (
	OutputNormal OutputType = iota
	OutputCode
	OutputError
	OutputToolResult
	OutputDiff
	OutputSuccess
	OutputWarning
)

var (
	// Use a terminal-friendly style
	chromaStyle = styles.Get("monokai")

	// Formatter for terminal output (256 colors)
	chromaFormatter = formatters.Get("terminal256")

	// Styles for different output types
	outputStyles = map[OutputType]lipgloss.Style{
		OutputNormal:     lipgloss.NewStyle(),
		OutputCode:       mdCodeBlockStyle,
		OutputError:      errorStyle,
		OutputToolResult: boxStyle,
		OutputSuccess:    successStyle,
		OutputWarning:    warningStyle,
	}

	// Tool output box style
	toolOutputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	// Tool name style
	toolNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	// Tool success indicator
	toolSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true)

	// Tool error indicator
	toolErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// HighlightCode returns syntax-highlighted code using Chroma
func HighlightCode(code, language string) string {
	if code == "" {
		return ""
	}

	// Get the lexer for the language
	lexer := getLexer(language)
	if lexer == nil {
		// Fall back to plain text if no lexer found
		return code
	}

	// Coalesce runs
	lexer = chroma.Coalesce(lexer)

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	// Format to string
	var buf bytes.Buffer
	err = chromaFormatter.Format(&buf, chromaStyle, iterator)
	if err != nil {
		return code
	}

	return buf.String()
}

// getLexer returns the appropriate lexer for a language
func getLexer(language string) chroma.Lexer {
	if language == "" {
		return nil
	}

	// Normalize language name
	language = strings.ToLower(strings.TrimSpace(language))

	// Map common aliases
	aliases := map[string]string{
		"js":         "javascript",
		"ts":         "typescript",
		"py":         "python",
		"rb":         "ruby",
		"sh":         "bash",
		"shell":      "bash",
		"yml":        "yaml",
		"dockerfile": "docker",
		"makefile":   "make",
		"md":         "markdown",
		"c++":        "cpp",
		"c#":         "csharp",
	}

	if alias, ok := aliases[language]; ok {
		language = alias
	}

	// Try to get lexer by name
	lexer := lexers.Get(language)
	if lexer != nil {
		return lexer
	}

	// Try to match by filename extension
	lexer = lexers.Match("file." + language)
	if lexer != nil {
		return lexer
	}

	return nil
}

// RenderOutput renders content with appropriate styling for the output type
func RenderOutput(content string, outputType OutputType) string {
	style, ok := outputStyles[outputType]
	if !ok {
		style = outputStyles[OutputNormal]
	}
	return style.Render(content)
}

// RenderToolOutput renders tool output with a styled box
func RenderToolOutput(toolName string, output string, isError bool) string {
	var header string
	if isError {
		header = toolErrorStyle.Render("[x " + toolName + "]")
	} else {
		header = toolSuccessStyle.Render("[v " + toolName + "]")
	}

	// Truncate long outputs
	maxLines := 20
	lines := strings.Split(output, "\n")
	if len(lines) > maxLines {
		truncated := strings.Join(lines[:maxLines], "\n")
		truncated += "\n" + dimStyle.Render("... (output truncated)")
		output = truncated
	}

	return header + "\n" + toolOutputStyle.Render(output)
}

// RenderCodeBlock renders a code block with syntax highlighting and a language label
func RenderCodeBlock(code, language string) string {
	var header string
	if language != "" {
		header = dimStyle.Render("  " + language) + "\n"
	}

	highlighted := HighlightCode(code, language)
	return header + mdCodeBlockStyle.Render(highlighted)
}

// DetectLanguage attempts to detect the language from file extension or content
func DetectLanguage(filename string, content string) string {
	// Try by filename first
	lexer := lexers.Match(filename)
	if lexer != nil {
		return lexer.Config().Name
	}

	// Try to analyze content
	lexer = lexers.Analyse(content)
	if lexer != nil {
		return lexer.Config().Name
	}

	return ""
}

// GetSupportedLanguages returns a list of commonly supported languages
func GetSupportedLanguages() []string {
	return []string{
		"go", "python", "javascript", "typescript", "rust", "java",
		"c", "cpp", "csharp", "ruby", "php", "swift", "kotlin",
		"scala", "haskell", "elixir", "erlang", "clojure",
		"bash", "powershell", "sql", "graphql",
		"html", "css", "scss", "json", "yaml", "toml", "xml",
		"markdown", "dockerfile", "makefile", "terraform",
	}
}
