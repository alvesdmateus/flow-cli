package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mateus/flow-cli/internal/sandbox"
)

// ReadFileTool reads the contents of a file
type ReadFileTool struct {
	permissions *sandbox.Manager
}

// NewReadFileTool creates a new read file tool
func NewReadFileTool(permissions *sandbox.Manager) *ReadFileTool {
	return &ReadFileTool{permissions: permissions}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file at the specified path"
}

func (t *ReadFileTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "The path to the file to read",
			Required:    true,
		},
		{
			Name:        "max_lines",
			Type:        TypeNumber,
			Description: "Maximum number of lines to read (0 = all)",
			Required:    false,
			Default:     0,
		},
	}
}

func (t *ReadFileTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	maxLines := GetIntArg(args, "max_lines", 0)

	// Check permission
	op := sandbox.NewOperation(sandbox.OpReadFile, path, fmt.Sprintf("Read file: %s", path))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
	}

	result := string(content)

	// Limit lines if requested
	if maxLines > 0 {
		lines := strings.Split(result, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			result = strings.Join(lines, "\n") + fmt.Sprintf("\n... (truncated, showing %d of %d lines)", maxLines, len(strings.Split(string(content), "\n")))
		}
	}

	return NewSuccessResultWithData(result, map[string]any{
		"path":  path,
		"size":  len(content),
		"lines": len(strings.Split(string(content), "\n")),
	}), nil
}

// WriteFileTool writes content to a file
type WriteFileTool struct {
	permissions *sandbox.Manager
	showDiff    func(path, oldContent, newContent string) error
}

// NewWriteFileTool creates a new write file tool
func NewWriteFileTool(permissions *sandbox.Manager, showDiff func(path, oldContent, newContent string) error) *WriteFileTool {
	return &WriteFileTool{
		permissions: permissions,
		showDiff:    showDiff,
	}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file, creating it if it doesn't exist"
}

func (t *WriteFileTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "The path to the file to write",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        TypeString,
			Description: "The content to write to the file",
			Required:    true,
		},
		{
			Name:        "append",
			Type:        TypeBoolean,
			Description: "If true, append to the file instead of overwriting",
			Required:    false,
			Default:     false,
		},
	}
}

func (t *WriteFileTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWriteFile
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	content, err := RequiredStringArg(args, "content")
	if err != nil {
		return NewErrorResult(err), nil
	}

	appendMode := GetBoolArg(args, "append", false)

	// Get existing content for diff
	var oldContent string
	if existingContent, err := os.ReadFile(path); err == nil {
		oldContent = string(existingContent)
	}

	// Show diff if available
	if t.showDiff != nil && !appendMode {
		if err := t.showDiff(path, oldContent, content); err != nil {
			return NewErrorResult(fmt.Errorf("diff display failed: %w", err)), nil
		}
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpWriteFile, path, fmt.Sprintf("Write file: %s", path)).
		WithDetail("size", fmt.Sprintf("%d bytes", len(content))).
		WithDetail("mode", map[bool]string{true: "append", false: "overwrite"}[appendMode])

	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewErrorResult(fmt.Errorf("failed to create directory: %w", err)), nil
	}

	// Write the file
	var writeErr error
	if appendMode {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			writeErr = err
		} else {
			_, writeErr = f.WriteString(content)
			f.Close()
		}
	} else {
		writeErr = os.WriteFile(path, []byte(content), 0644)
	}

	if writeErr != nil {
		return NewErrorResult(fmt.Errorf("failed to write file: %w", writeErr)), nil
	}

	return NewSuccessResult(fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)), nil
}

// ListFilesTool lists files in a directory
type ListFilesTool struct {
	permissions *sandbox.Manager
}

// NewListFilesTool creates a new list files tool
func NewListFilesTool(permissions *sandbox.Manager) *ListFilesTool {
	return &ListFilesTool{permissions: permissions}
}

func (t *ListFilesTool) Name() string {
	return "list_files"
}

func (t *ListFilesTool) Description() string {
	return "List files and directories at the specified path"
}

func (t *ListFilesTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "The directory path to list",
			Required:    true,
		},
		{
			Name:        "recursive",
			Type:        TypeBoolean,
			Description: "If true, list files recursively",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "pattern",
			Type:        TypeString,
			Description: "Glob pattern to filter files (e.g., '*.go')",
			Required:    false,
		},
	}
}

func (t *ListFilesTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpListDir
}

func (t *ListFilesTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	recursive := GetBoolArg(args, "recursive", false)
	pattern := GetStringArg(args, "pattern", "")

	// Check permission
	op := sandbox.NewOperation(sandbox.OpListDir, path, fmt.Sprintf("List directory: %s", path))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	var files []string

	if recursive {
		err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			relPath, _ := filepath.Rel(path, filePath)
			if relPath == "." {
				return nil
			}

			// Skip hidden directories
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}

			// Apply pattern filter
			if pattern != "" {
				matched, _ := filepath.Match(pattern, info.Name())
				if !matched {
					return nil
				}
			}

			entry := relPath
			if info.IsDir() {
				entry += "/"
			}
			files = append(files, entry)
			return nil
		})
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return NewErrorResult(fmt.Errorf("failed to read directory: %w", err)), nil
		}

		for _, entry := range entries {
			name := entry.Name()

			// Apply pattern filter
			if pattern != "" {
				matched, _ := filepath.Match(pattern, name)
				if !matched {
					continue
				}
			}

			if entry.IsDir() {
				name += "/"
			}
			files = append(files, name)
		}
	}

	if err != nil {
		return NewErrorResult(fmt.Errorf("failed to list files: %w", err)), nil
	}

	output := strings.Join(files, "\n")
	if len(files) == 0 {
		output = "(empty directory)"
	}

	return NewSuccessResultWithData(output, map[string]any{
		"path":  path,
		"count": len(files),
		"files": files,
	}), nil
}

// CreateDirectoryTool creates a directory
type CreateDirectoryTool struct {
	permissions *sandbox.Manager
}

// NewCreateDirectoryTool creates a new create directory tool
func NewCreateDirectoryTool(permissions *sandbox.Manager) *CreateDirectoryTool {
	return &CreateDirectoryTool{permissions: permissions}
}

func (t *CreateDirectoryTool) Name() string {
	return "create_directory"
}

func (t *CreateDirectoryTool) Description() string {
	return "Create a directory at the specified path"
}

func (t *CreateDirectoryTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "The path of the directory to create",
			Required:    true,
		},
	}
}

func (t *CreateDirectoryTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpCreateDir
}

func (t *CreateDirectoryTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpCreateDir, path, fmt.Sprintf("Create directory: %s", path))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Create the directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return NewErrorResult(fmt.Errorf("failed to create directory: %w", err)), nil
	}

	return NewSuccessResult(fmt.Sprintf("Successfully created directory: %s", path)), nil
}

// DeleteFileTool deletes a file or directory
type DeleteFileTool struct {
	permissions *sandbox.Manager
}

// NewDeleteFileTool creates a new delete file tool
func NewDeleteFileTool(permissions *sandbox.Manager) *DeleteFileTool {
	return &DeleteFileTool{permissions: permissions}
}

func (t *DeleteFileTool) Name() string {
	return "delete_file"
}

func (t *DeleteFileTool) Description() string {
	return "Delete a file or empty directory at the specified path"
}

func (t *DeleteFileTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "path",
			Type:        TypeString,
			Description: "The path of the file or directory to delete",
			Required:    true,
		},
	}
}

func (t *DeleteFileTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpDeleteFile
}

func (t *DeleteFileTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	path, err := RequiredStringArg(args, "path")
	if err != nil {
		return NewErrorResult(err), nil
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return NewErrorResult(fmt.Errorf("path does not exist: %w", err)), nil
	}

	// Check permission (high risk operation)
	op := sandbox.NewOperation(sandbox.OpDeleteFile, path, fmt.Sprintf("Delete: %s", path)).
		WithDetail("type", map[bool]string{true: "directory", false: "file"}[info.IsDir()])

	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Delete the file/directory
	if err := os.Remove(path); err != nil {
		return NewErrorResult(fmt.Errorf("failed to delete: %w", err)), nil
	}

	return NewSuccessResult(fmt.Sprintf("Successfully deleted: %s", path)), nil
}
