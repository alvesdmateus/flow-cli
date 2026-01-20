package review

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Regex patterns for diff parsing
	diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)
	oldFileRegex    = regexp.MustCompile(`^--- (?:a/)?(.+)$`)
	newFileRegex    = regexp.MustCompile(`^\+\+\+ (?:b/)?(.+)$`)
	binaryRegex     = regexp.MustCompile(`^Binary files .+ differ$`)
	renameFromRegex = regexp.MustCompile(`^rename from (.+)$`)
	renameToRegex   = regexp.MustCompile(`^rename to (.+)$`)
)

// ParseDiff parses a unified diff and returns structured information.
func ParseDiff(diff string) *DiffInfo {
	info := &DiffInfo{
		Files: []FileDiff{},
	}

	if diff == "" {
		return info
	}

	lines := strings.Split(diff, "\n")
	var currentFile *FileDiff
	var currentHunk *Hunk
	oldLine, newLine := 0, 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for diff header
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous file if exists
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				info.Files = append(info.Files, *currentFile)
			}

			currentFile = &FileDiff{
				Path:   matches[2],
				Status: "modified",
				Hunks:  []Hunk{},
			}
			currentHunk = nil
			continue
		}

		if currentFile == nil {
			continue
		}

		// Check for old file path
		if matches := oldFileRegex.FindStringSubmatch(line); matches != nil {
			if matches[1] == "/dev/null" {
				currentFile.Status = "added"
			}
			continue
		}

		// Check for new file path
		if matches := newFileRegex.FindStringSubmatch(line); matches != nil {
			if matches[1] == "/dev/null" {
				currentFile.Status = "deleted"
			}
			currentFile.Language = detectLanguage(currentFile.Path)
			continue
		}

		// Check for binary file
		if binaryRegex.MatchString(line) {
			currentFile.IsBinary = true
			continue
		}

		// Check for rename
		if matches := renameFromRegex.FindStringSubmatch(line); matches != nil {
			currentFile.OldPath = matches[1]
			currentFile.Status = "renamed"
			continue
		}

		if matches := renameToRegex.FindStringSubmatch(line); matches != nil {
			currentFile.Path = matches[1]
			continue
		}

		// Check for hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous hunk
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			oldStart, _ := strconv.Atoi(matches[1])
			oldCount := 1
			if matches[2] != "" {
				oldCount, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newCount := 1
			if matches[4] != "" {
				newCount, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				Header:   strings.TrimSpace(matches[5]),
				Lines:    []DiffLine{},
			}
			oldLine = oldStart
			newLine = newStart
			continue
		}

		// Parse diff lines
		if currentHunk != nil && len(line) > 0 {
			diffLine := DiffLine{Content: line[1:]} // Remove the prefix character

			switch line[0] {
			case '+':
				diffLine.Type = LineAddition
				diffLine.NewLine = newLine
				newLine++
				currentFile.Additions++
				info.Additions++
			case '-':
				diffLine.Type = LineDeletion
				diffLine.OldLine = oldLine
				oldLine++
				currentFile.Deletions++
				info.Deletions++
			case ' ':
				diffLine.Type = LineContext
				diffLine.OldLine = oldLine
				diffLine.NewLine = newLine
				oldLine++
				newLine++
			default:
				// Skip other lines (like "\ No newline at end of file")
				continue
			}

			currentHunk.Lines = append(currentHunk.Lines, diffLine)
		}
	}

	// Save last file and hunk
	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		info.Files = append(info.Files, *currentFile)
	}

	info.FilesChanged = len(info.Files)
	return info
}

// detectLanguage determines the programming language from file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescript-react"
	case ".jsx":
		return "javascript-react"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".sh", ".bash":
		return "shell"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".sql":
		return "sql"
	case ".md", ".markdown":
		return "markdown"
	case ".dockerfile":
		return "dockerfile"
	default:
		// Check for special filenames
		base := strings.ToLower(filepath.Base(path))
		switch base {
		case "dockerfile":
			return "dockerfile"
		case "makefile":
			return "makefile"
		case ".gitignore", ".dockerignore":
			return "ignore"
		}
		return ""
	}
}

// GetDiffSummary returns a concise summary of the diff for LLM context.
func GetDiffSummary(info *DiffInfo) string {
	var sb strings.Builder

	sb.WriteString("Diff Summary:\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString("Files changed: " + strconv.Itoa(info.FilesChanged) + "\n")
	sb.WriteString("Additions: " + strconv.Itoa(info.Additions) + "\n")
	sb.WriteString("Deletions: " + strconv.Itoa(info.Deletions) + "\n\n")

	sb.WriteString("Files:\n")
	for _, f := range info.Files {
		status := f.Status
		if f.IsBinary {
			status += " (binary)"
		}
		sb.WriteString("  " + f.Path + " (" + status + ")")
		if f.Language != "" {
			sb.WriteString(" [" + f.Language + "]")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
