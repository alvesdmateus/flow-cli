package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Diff styles
	addedLineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Background(lipgloss.Color("22"))

	removedLineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Background(lipgloss.Color("52"))

	contextLineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250"))

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(4).
			Align(lipgloss.Right)

	newFileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	modifiedFileStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)
)

// DiffLine represents a line in a diff
type DiffLine struct {
	Type    DiffLineType
	Content string
	OldNum  int
	NewNum  int
}

// DiffLineType represents the type of diff line
type DiffLineType int

const (
	DiffContext DiffLineType = iota
	DiffAdded
	DiffRemoved
)

// ShowDiff displays a diff between old and new content
func ShowDiff(path, oldContent, newContent string) error {
	// Detect language for syntax highlighting
	currentDiffLanguage = detectLanguageFromPath(path)

	if oldContent == "" {
		// New file
		fmt.Println()
		fmt.Println(newFileStyle.Render("+ New file: " + path))
		fmt.Println()

		lines := strings.Split(newContent, "\n")
		maxLines := 30
		if len(lines) > maxLines {
			for i, line := range lines[:maxLines] {
				fmt.Printf("%s %s\n",
					lineNumberStyle.Render(fmt.Sprintf("%d", i+1)),
					addedLineStyle.Render("+ "+line))
			}
			fmt.Printf("%s ... (%d more lines)\n",
				lineNumberStyle.Render(""),
				len(lines)-maxLines)
		} else {
			for i, line := range lines {
				fmt.Printf("%s %s\n",
					lineNumberStyle.Render(fmt.Sprintf("%d", i+1)),
					addedLineStyle.Render("+ "+line))
			}
		}
		fmt.Println()
		return nil
	}

	// Modified file
	fmt.Println()
	fmt.Println(modifiedFileStyle.Render("~ Modified: " + path))
	fmt.Println()

	// Generate unified diff
	diffLines := generateDiff(oldContent, newContent)

	// Display diff with context
	displayDiff(diffLines)

	fmt.Println()
	return nil
}

// generateDiff creates a simple line-based diff
func generateDiff(oldContent, newContent string) []DiffLine {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple LCS-based diff
	diff := make([]DiffLine, 0)

	// Use a simple approach for demonstration
	// In production, use a proper diff algorithm
	oldIdx, newIdx := 0, 0

	for oldIdx < len(oldLines) || newIdx < len(newLines) {
		if oldIdx >= len(oldLines) {
			// Rest are additions
			diff = append(diff, DiffLine{
				Type:    DiffAdded,
				Content: newLines[newIdx],
				NewNum:  newIdx + 1,
			})
			newIdx++
		} else if newIdx >= len(newLines) {
			// Rest are deletions
			diff = append(diff, DiffLine{
				Type:    DiffRemoved,
				Content: oldLines[oldIdx],
				OldNum:  oldIdx + 1,
			})
			oldIdx++
		} else if oldLines[oldIdx] == newLines[newIdx] {
			// Same line
			diff = append(diff, DiffLine{
				Type:    DiffContext,
				Content: oldLines[oldIdx],
				OldNum:  oldIdx + 1,
				NewNum:  newIdx + 1,
			})
			oldIdx++
			newIdx++
		} else {
			// Different - try to find matching line ahead
			foundOld := findLineAhead(newLines[newIdx], oldLines, oldIdx, 5)
			foundNew := findLineAhead(oldLines[oldIdx], newLines, newIdx, 5)

			if foundOld >= 0 && (foundNew < 0 || foundOld-oldIdx <= foundNew-newIdx) {
				// Remove lines until we find match
				for oldIdx < foundOld {
					diff = append(diff, DiffLine{
						Type:    DiffRemoved,
						Content: oldLines[oldIdx],
						OldNum:  oldIdx + 1,
					})
					oldIdx++
				}
			} else if foundNew >= 0 {
				// Add lines until we find match
				for newIdx < foundNew {
					diff = append(diff, DiffLine{
						Type:    DiffAdded,
						Content: newLines[newIdx],
						NewNum:  newIdx + 1,
					})
					newIdx++
				}
			} else {
				// No match found, treat as modification
				diff = append(diff, DiffLine{
					Type:    DiffRemoved,
					Content: oldLines[oldIdx],
					OldNum:  oldIdx + 1,
				})
				diff = append(diff, DiffLine{
					Type:    DiffAdded,
					Content: newLines[newIdx],
					NewNum:  newIdx + 1,
				})
				oldIdx++
				newIdx++
			}
		}
	}

	return diff
}

// findLineAhead looks for a matching line within lookahead distance
func findLineAhead(target string, lines []string, start, lookahead int) int {
	end := start + lookahead
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i < end; i++ {
		if lines[i] == target {
			return i
		}
	}
	return -1
}

// displayDiff shows the diff with context collapsing
func displayDiff(lines []DiffLine) {
	const contextLines = 3
	const maxTotalLines = 50

	// Find regions of changes
	type region struct {
		start, end int
	}

	var regions []region
	inChange := false
	changeStart := 0

	for i, line := range lines {
		if line.Type != DiffContext {
			if !inChange {
				inChange = true
				changeStart = i
			}
		} else {
			if inChange {
				regions = append(regions, region{changeStart, i})
				inChange = false
			}
		}
	}
	if inChange {
		regions = append(regions, region{changeStart, len(lines)})
	}

	if len(regions) == 0 {
		fmt.Println(dimStyle.Render("  (no changes)"))
		return
	}

	// Display each region with context
	displayed := 0
	for _, r := range regions {
		// Add context before
		contextStart := r.start - contextLines
		if contextStart < 0 {
			contextStart = 0
		}

		// Show separator if not at start
		if contextStart > 0 && displayed > 0 {
			fmt.Println(dimStyle.Render("  ..."))
		}

		// Display context before
		for i := contextStart; i < r.start && displayed < maxTotalLines; i++ {
			printDiffLine(lines[i])
			displayed++
		}

		// Display changes
		for i := r.start; i < r.end && displayed < maxTotalLines; i++ {
			printDiffLine(lines[i])
			displayed++
		}

		// Display context after
		contextEnd := r.end + contextLines
		if contextEnd > len(lines) {
			contextEnd = len(lines)
		}

		for i := r.end; i < contextEnd && displayed < maxTotalLines; i++ {
			printDiffLine(lines[i])
			displayed++
		}

		if displayed >= maxTotalLines {
			fmt.Println(dimStyle.Render(fmt.Sprintf("  ... (%d more lines)", len(lines)-displayed)))
			break
		}
	}
}

// currentDiffLanguage stores the detected language for current diff
var currentDiffLanguage string

// printDiffLine prints a single diff line with optional syntax highlighting
func printDiffLine(line DiffLine) {
	var prefix string
	var style lipgloss.Style
	var lineNum string

	switch line.Type {
	case DiffAdded:
		prefix = "+"
		style = addedLineStyle
		lineNum = fmt.Sprintf("%d", line.NewNum)
	case DiffRemoved:
		prefix = "-"
		style = removedLineStyle
		lineNum = fmt.Sprintf("%d", line.OldNum)
	default:
		prefix = " "
		style = contextLineStyle
		lineNum = fmt.Sprintf("%d", line.NewNum)
	}

	// Apply syntax highlighting to content if we have a detected language
	content := line.Content
	if currentDiffLanguage != "" && line.Type == DiffContext {
		// For context lines, apply subtle syntax highlighting
		content = HighlightCode(content, currentDiffLanguage)
	}

	fmt.Printf("%s %s\n",
		lineNumberStyle.Render(lineNum),
		style.Render(prefix+" "+content))
}

// detectLanguageFromPath returns the language based on file extension
func detectLanguageFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	// Map extensions to languages
	extMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".rs":    "rust",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".sh":    "bash",
		".bash":  "bash",
		".zsh":   "bash",
		".sql":   "sql",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".toml":  "toml",
		".xml":   "xml",
		".md":    "markdown",
	}

	if lang, ok := extMap[ext]; ok {
		return lang
	}

	// Check for special filenames
	base := strings.ToLower(filepath.Base(path))
	specialFiles := map[string]string{
		"dockerfile": "docker",
		"makefile":   "make",
		"cmakelists.txt": "cmake",
	}

	if lang, ok := specialFiles[base]; ok {
		return lang
	}

	return ""
}

// ShowFileDiff is a convenience function for showing file diffs
func ShowFileDiff(path, oldContent, newContent string) error {
	return ShowDiff(path, oldContent, newContent)
}
