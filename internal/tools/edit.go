package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mateus/vibe-cli/internal/sandbox"
)

// EditFileTool performs surgical edits on files
type EditFileTool struct {
	permissions *sandbox.Manager
	showDiff    func(path, oldContent, newContent string) error
}

// NewEditFileTool creates a new edit file tool
func NewEditFileTool(permissions *sandbox.Manager, showDiff func(path, oldContent, newContent string) error) *EditFileTool {
	return &EditFileTool{
		permissions: permissions,
		showDiff:    showDiff,
	}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Make surgical edits to a file by replacing specific text. More precise than rewriting the entire file."
}

func (t *EditFileTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Path to the file to edit",
			Required:    true,
		},
		{
			Name:        "old_text",
			Type:        TypeString,
			Description: "The exact text to find and replace. Must match exactly including whitespace and newlines.",
			Required:    true,
		},
		{
			Name:        "new_text",
			Type:        TypeString,
			Description: "The text to replace old_text with. Can be empty to delete the text.",
			Required:    true,
		},
		{
			Name:        "occurrence",
			Type:        TypeNumber,
			Description: "Which occurrence to replace: 0 for all, 1 for first, 2 for second, etc. (default: 1)",
			Required:    false,
			Default:     1,
		},
		{
			Name:        "create_if_missing",
			Type:        TypeBoolean,
			Description: "Create the file if it doesn't exist (default: false)",
			Required:    false,
			Default:     false,
		},
	}
}

func (t *EditFileTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWriteFile
}

func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	oldText, err := RequiredStringArg(args, "old_text")
	if err != nil {
		return NewErrorResult(err), nil
	}

	newText := GetStringArg(args, "new_text", "")
	occurrence := GetIntArg(args, "occurrence", 1)
	createIfMissing := GetBoolArg(args, "create_if_missing", false)

	// Check permission
	op := sandbox.NewOperation(sandbox.OpWriteFile, path, fmt.Sprintf("Edit file: %s", path)).
		WithDetail("operation", "replace")
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Read existing content
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && createIfMissing {
			// Create new file with just the new text
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return NewErrorResult(fmt.Errorf("failed to create directory: %w", err)), nil
			}
			if err := os.WriteFile(path, []byte(newText), 0644); err != nil {
				return NewErrorResult(fmt.Errorf("failed to create file: %w", err)), nil
			}
			return NewSuccessResult(fmt.Sprintf("Created new file: %s", path)), nil
		}
		return NewErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
	}

	oldContent := string(content)

	// Check if old_text exists
	count := strings.Count(oldContent, oldText)
	if count == 0 {
		return NewErrorResult(fmt.Errorf("text not found in file: %q", truncateForError(oldText, 100))), nil
	}

	// Perform replacement
	var newContent string
	if occurrence == 0 {
		// Replace all occurrences
		newContent = strings.ReplaceAll(oldContent, oldText, newText)
	} else if occurrence > 0 {
		// Replace specific occurrence
		if occurrence > count {
			return NewErrorResult(fmt.Errorf("occurrence %d not found (only %d occurrences exist)", occurrence, count)), nil
		}
		newContent = replaceNth(oldContent, oldText, newText, occurrence)
	} else {
		return NewErrorResult(fmt.Errorf("occurrence must be >= 0")), nil
	}

	// Check if any change was made
	if oldContent == newContent {
		return NewSuccessResult("No changes needed - file already matches desired content"), nil
	}

	// Show diff
	if t.showDiff != nil {
		if err := t.showDiff(path, oldContent, newContent); err != nil {
			return NewErrorResult(fmt.Errorf("diff display failed: %w", err)), nil
		}
	}

	// Write the file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return NewErrorResult(fmt.Errorf("failed to write file: %w", err)), nil
	}

	// Calculate statistics
	oldLines := strings.Count(oldContent, "\n")
	newLines := strings.Count(newContent, "\n")
	linesDiff := newLines - oldLines

	var changeDesc string
	if occurrence == 0 {
		changeDesc = fmt.Sprintf("replaced %d occurrences", count)
	} else {
		changeDesc = fmt.Sprintf("replaced occurrence %d of %d", occurrence, count)
	}

	result := fmt.Sprintf("Edited %s: %s", path, changeDesc)
	if linesDiff > 0 {
		result += fmt.Sprintf(" (+%d lines)", linesDiff)
	} else if linesDiff < 0 {
		result += fmt.Sprintf(" (%d lines)", linesDiff)
	}

	return NewSuccessResultWithData(result, map[string]any{
		"path":               path,
		"occurrences_found":  count,
		"occurrences_replaced": func() int {
			if occurrence == 0 {
				return count
			}
			return 1
		}(),
		"lines_before": oldLines + 1,
		"lines_after":  newLines + 1,
	}), nil
}

// replaceNth replaces the nth occurrence of old with new in s
func replaceNth(s, old, new string, n int) string {
	if n <= 0 {
		return s
	}

	idx := 0
	for i := 0; i < n; i++ {
		pos := strings.Index(s[idx:], old)
		if pos == -1 {
			return s
		}
		if i == n-1 {
			return s[:idx+pos] + new + s[idx+pos+len(old):]
		}
		idx += pos + len(old)
	}
	return s
}

// truncateForError truncates a string for error messages
func truncateForError(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// InsertLinesTool inserts lines at a specific position
type InsertLinesTool struct {
	permissions *sandbox.Manager
	showDiff    func(path, oldContent, newContent string) error
}

// NewInsertLinesTool creates a new insert lines tool
func NewInsertLinesTool(permissions *sandbox.Manager, showDiff func(path, oldContent, newContent string) error) *InsertLinesTool {
	return &InsertLinesTool{
		permissions: permissions,
		showDiff:    showDiff,
	}
}

func (t *InsertLinesTool) Name() string {
	return "insert_lines"
}

func (t *InsertLinesTool) Description() string {
	return "Insert lines at a specific line number in a file. Useful for adding imports, functions, or blocks of code."
}

func (t *InsertLinesTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Path to the file to edit",
			Required:    true,
		},
		{
			Name:        "line",
			Type:        TypeNumber,
			Description: "Line number to insert at (1-based). Content will be inserted BEFORE this line. Use 0 or negative to append at end.",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        TypeString,
			Description: "The content to insert",
			Required:    true,
		},
	}
}

func (t *InsertLinesTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWriteFile
}

func (t *InsertLinesTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	lineNum := GetIntArg(args, "line", 0)

	insertContent, err := RequiredStringArg(args, "content")
	if err != nil {
		return NewErrorResult(err), nil
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpWriteFile, path, fmt.Sprintf("Insert lines in: %s at line %d", path, lineNum)).
		WithDetail("operation", "insert")
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Read existing content
	file, err := os.Open(path)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to open file: %w", err)), nil
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return NewErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
	}

	oldContent := strings.Join(lines, "\n")
	if len(lines) > 0 {
		oldContent += "\n"
	}

	// Insert content
	insertLines := strings.Split(insertContent, "\n")

	var newLines []string
	if lineNum <= 0 || lineNum > len(lines) {
		// Append at end
		newLines = append(lines, insertLines...)
	} else {
		// Insert before specified line (1-based to 0-based)
		idx := lineNum - 1
		newLines = make([]string, 0, len(lines)+len(insertLines))
		newLines = append(newLines, lines[:idx]...)
		newLines = append(newLines, insertLines...)
		newLines = append(newLines, lines[idx:]...)
	}

	newContent := strings.Join(newLines, "\n")
	if len(newLines) > 0 {
		newContent += "\n"
	}

	// Show diff
	if t.showDiff != nil {
		if err := t.showDiff(path, oldContent, newContent); err != nil {
			return NewErrorResult(fmt.Errorf("diff display failed: %w", err)), nil
		}
	}

	// Write the file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return NewErrorResult(fmt.Errorf("failed to write file: %w", err)), nil
	}

	return NewSuccessResultWithData(
		fmt.Sprintf("Inserted %d lines at line %d in %s", len(insertLines), lineNum, path),
		map[string]any{
			"path":           path,
			"lines_inserted": len(insertLines),
			"at_line":        lineNum,
		},
	), nil
}

// DeleteLinesTool deletes lines from a file
type DeleteLinesTool struct {
	permissions *sandbox.Manager
	showDiff    func(path, oldContent, newContent string) error
}

// NewDeleteLinesTool creates a new delete lines tool
func NewDeleteLinesTool(permissions *sandbox.Manager, showDiff func(path, oldContent, newContent string) error) *DeleteLinesTool {
	return &DeleteLinesTool{
		permissions: permissions,
		showDiff:    showDiff,
	}
}

func (t *DeleteLinesTool) Name() string {
	return "delete_lines"
}

func (t *DeleteLinesTool) Description() string {
	return "Delete a range of lines from a file."
}

func (t *DeleteLinesTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Path to the file to edit",
			Required:    true,
		},
		{
			Name:        "start_line",
			Type:        TypeNumber,
			Description: "First line to delete (1-based, inclusive)",
			Required:    true,
		},
		{
			Name:        "end_line",
			Type:        TypeNumber,
			Description: "Last line to delete (1-based, inclusive). Same as start_line to delete a single line.",
			Required:    true,
		},
	}
}

func (t *DeleteLinesTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWriteFile
}

func (t *DeleteLinesTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	startLine := GetIntArg(args, "start_line", 0)
	endLine := GetIntArg(args, "end_line", 0)

	if startLine <= 0 || endLine <= 0 {
		return NewErrorResult(fmt.Errorf("line numbers must be positive")), nil
	}
	if startLine > endLine {
		return NewErrorResult(fmt.Errorf("start_line must be <= end_line")), nil
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpWriteFile, path, fmt.Sprintf("Delete lines %d-%d from: %s", startLine, endLine, path)).
		WithDetail("operation", "delete")
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Read existing content
	file, err := os.Open(path)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to open file: %w", err)), nil
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return NewErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
	}

	// Validate line numbers
	if startLine > len(lines) {
		return NewErrorResult(fmt.Errorf("start_line %d exceeds file length (%d lines)", startLine, len(lines))), nil
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	oldContent := strings.Join(lines, "\n")
	if len(lines) > 0 {
		oldContent += "\n"
	}

	// Delete lines (1-based to 0-based)
	newLines := make([]string, 0, len(lines)-(endLine-startLine+1))
	newLines = append(newLines, lines[:startLine-1]...)
	newLines = append(newLines, lines[endLine:]...)

	newContent := strings.Join(newLines, "\n")
	if len(newLines) > 0 {
		newContent += "\n"
	}

	// Show diff
	if t.showDiff != nil {
		if err := t.showDiff(path, oldContent, newContent); err != nil {
			return NewErrorResult(fmt.Errorf("diff display failed: %w", err)), nil
		}
	}

	// Write the file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return NewErrorResult(fmt.Errorf("failed to write file: %w", err)), nil
	}

	linesDeleted := endLine - startLine + 1
	return NewSuccessResultWithData(
		fmt.Sprintf("Deleted %d lines (%d-%d) from %s", linesDeleted, startLine, endLine, path),
		map[string]any{
			"path":          path,
			"lines_deleted": linesDeleted,
			"start_line":    startLine,
			"end_line":      endLine,
		},
	), nil
}
