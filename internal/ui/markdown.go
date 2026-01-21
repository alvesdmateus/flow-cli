package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Markdown styles
	mdH1Style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("75")).
			MarginTop(1).
			MarginBottom(1)

	mdH2Style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("75")).
			MarginTop(1)

	mdH3Style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("250"))

	mdBoldStyle = lipgloss.NewStyle().
			Bold(true)

	mdItalicStyle = lipgloss.NewStyle().
			Italic(true)

	mdCodeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("229")).
			Padding(0, 1)

	mdCodeBlockStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Foreground(lipgloss.Color("252")).
				Padding(1).
				MarginTop(1).
				MarginBottom(1)

	mdLinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("45")).
			Underline(true)

	mdBlockquoteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("240")).
				PaddingLeft(1)

	mdListStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	mdHorizontalRule = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(strings.Repeat("─", 50))
)

// RenderMarkdown renders markdown content to styled terminal output
func RenderMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	var inCodeBlock bool
	var codeBlockContent []string
	var codeBlockLang string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End code block
				rendered := renderCodeBlock(strings.Join(codeBlockContent, "\n"), codeBlockLang)
				result = append(result, rendered)
				codeBlockContent = nil
				codeBlockLang = ""
				inCodeBlock = false
			} else {
				// Start code block
				inCodeBlock = true
				codeBlockLang = strings.TrimPrefix(line, "```")
			}
			continue
		}

		if inCodeBlock {
			codeBlockContent = append(codeBlockContent, line)
			continue
		}

		// Handle other markdown elements
		rendered := renderLine(line)
		result = append(result, rendered)
	}

	// Handle unclosed code block
	if inCodeBlock && len(codeBlockContent) > 0 {
		rendered := renderCodeBlock(strings.Join(codeBlockContent, "\n"), codeBlockLang)
		result = append(result, rendered)
	}

	return strings.Join(result, "\n")
}

func renderLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Headers
	if strings.HasPrefix(trimmed, "### ") {
		return mdH3Style.Render(strings.TrimPrefix(trimmed, "### "))
	}
	if strings.HasPrefix(trimmed, "## ") {
		return mdH2Style.Render(strings.TrimPrefix(trimmed, "## "))
	}
	if strings.HasPrefix(trimmed, "# ") {
		return mdH1Style.Render(strings.TrimPrefix(trimmed, "# "))
	}

	// Horizontal rule
	if trimmed == "---" || trimmed == "***" || trimmed == "___" {
		return mdHorizontalRule
	}

	// Blockquote
	if strings.HasPrefix(trimmed, "> ") {
		return mdBlockquoteStyle.Render(strings.TrimPrefix(trimmed, "> "))
	}

	// Unordered list
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Render("•")
		content := renderInline(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* "))
		return mdListStyle.Render(bullet + " " + content)
	}

	// Ordered list
	orderedListRegex := regexp.MustCompile(`^(\d+)\.\s+(.*)$`)
	if matches := orderedListRegex.FindStringSubmatch(trimmed); matches != nil {
		num := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Render(matches[1] + ".")
		content := renderInline(matches[2])
		return mdListStyle.Render(num + " " + content)
	}

	// Task list
	if strings.HasPrefix(trimmed, "- [ ] ") {
		checkbox := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("☐")
		content := renderInline(strings.TrimPrefix(trimmed, "- [ ] "))
		return mdListStyle.Render(checkbox + " " + content)
	}
	if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") {
		checkbox := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("☑")
		content := renderInline(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [x] "), "- [X] "))
		return mdListStyle.Render(checkbox + " " + content)
	}

	// Regular text with inline formatting
	return renderInline(line)
}

func renderInline(text string) string {
	result := text

	// Inline code (do this first to prevent other formatting inside code)
	codeRegex := regexp.MustCompile("`([^`]+)`")
	result = codeRegex.ReplaceAllStringFunc(result, func(match string) string {
		code := strings.Trim(match, "`")
		return mdCodeStyle.Render(code)
	})

	// Bold with ** or __
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	result = boldRegex.ReplaceAllStringFunc(result, func(match string) string {
		content := strings.Trim(strings.Trim(match, "*"), "_")
		return mdBoldStyle.Render(content)
	})

	// Italic with * or _
	italicRegex := regexp.MustCompile(`\*([^*]+)\*|_([^_]+)_`)
	result = italicRegex.ReplaceAllStringFunc(result, func(match string) string {
		content := strings.Trim(strings.Trim(match, "*"), "_")
		return mdItalicStyle.Render(content)
	})

	// Links [text](url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	result = linkRegex.ReplaceAllStringFunc(result, func(match string) string {
		matches := linkRegex.FindStringSubmatch(match)
		if len(matches) >= 3 {
			text := matches[1]
			url := matches[2]
			return mdLinkStyle.Render(text) + dimStyle.Render(" ("+url+")")
		}
		return match
	})

	// Strikethrough ~~text~~
	strikeRegex := regexp.MustCompile(`~~([^~]+)~~`)
	result = strikeRegex.ReplaceAllStringFunc(result, func(match string) string {
		content := strings.Trim(match, "~")
		return lipgloss.NewStyle().Strikethrough(true).Render(content)
	})

	return result
}

func renderCodeBlock(code, lang string) string {
	// Use Chroma-based syntax highlighting
	return RenderCodeBlock(code, lang)
}

// PrintMarkdown renders and prints markdown content
func PrintMarkdown(content string) {
	rendered := RenderMarkdown(content)
	println(rendered)
}

// Table renders a markdown-style table
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a new table
func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		widths:  widths,
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	// Pad or truncate to match header count
	row := make([]string, len(t.headers))
	for i := 0; i < len(t.headers); i++ {
		if i < len(cells) {
			row[i] = cells[i]
		} else {
			row[i] = ""
		}
		// Update max width
		if len(row[i]) > t.widths[i] {
			t.widths[i] = len(row[i])
		}
	}
	t.rows = append(t.rows, row)
}

// Render renders the table to a string
func (t *Table) Render() string {
	var sb strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Top border
	sb.WriteString(borderStyle.Render("┌"))
	for i, w := range t.widths {
		sb.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
		if i < len(t.widths)-1 {
			sb.WriteString(borderStyle.Render("┬"))
		}
	}
	sb.WriteString(borderStyle.Render("┐"))
	sb.WriteString("\n")

	// Header row
	sb.WriteString(borderStyle.Render("│"))
	for i, h := range t.headers {
		cell := headerStyle.Render(padRight(h, t.widths[i]))
		sb.WriteString(" " + cell + " ")
		sb.WriteString(borderStyle.Render("│"))
	}
	sb.WriteString("\n")

	// Header separator
	sb.WriteString(borderStyle.Render("├"))
	for i, w := range t.widths {
		sb.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
		if i < len(t.widths)-1 {
			sb.WriteString(borderStyle.Render("┼"))
		}
	}
	sb.WriteString(borderStyle.Render("┤"))
	sb.WriteString("\n")

	// Data rows
	for _, row := range t.rows {
		sb.WriteString(borderStyle.Render("│"))
		for i, cell := range row {
			sb.WriteString(" " + padRight(cell, t.widths[i]) + " ")
			sb.WriteString(borderStyle.Render("│"))
		}
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString(borderStyle.Render("└"))
	for i, w := range t.widths {
		sb.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
		if i < len(t.widths)-1 {
			sb.WriteString(borderStyle.Render("┴"))
		}
	}
	sb.WriteString(borderStyle.Render("┘"))

	return sb.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
