package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Scaffolder handles the creation of project files from templates.
type Scaffolder struct {
	targetDir string
	info      *ProjectInfo
}

// NewScaffolder creates a new scaffolder for the given directory and project info.
func NewScaffolder(targetDir string, info *ProjectInfo) *Scaffolder {
	return &Scaffolder{
		targetDir: targetDir,
		info:      info,
	}
}

// Scaffold creates all files from the given template.
// Returns a list of created file paths.
func (s *Scaffolder) Scaffold(tmpl *Template) ([]string, error) {
	var createdFiles []string

	for _, file := range tmpl.Files {
		path, err := s.createFile(file)
		if err != nil {
			return createdFiles, fmt.Errorf("failed to create %s: %w", file.Path, err)
		}
		if path != "" {
			createdFiles = append(createdFiles, path)
		}
	}

	return createdFiles, nil
}

// CreateFlowConfig creates only the flow configuration files.
// Used for initializing flow in existing projects.
func (s *Scaffolder) CreateFlowConfig() ([]string, error) {
	emptyTmpl := GetTemplate("empty")
	if emptyTmpl == nil {
		return nil, fmt.Errorf("empty template not found")
	}
	return s.Scaffold(emptyTmpl)
}

// createFile creates a single file from a template file definition.
func (s *Scaffolder) createFile(file TemplateFile) (string, error) {
	// Process the path template (for dynamic paths like src/{{.Name}}/...)
	pathTmpl, err := template.New("path").Parse(file.Path)
	if err != nil {
		return "", fmt.Errorf("invalid path template: %w", err)
	}

	var pathBuf bytes.Buffer
	if err := pathTmpl.Execute(&pathBuf, s.info); err != nil {
		return "", fmt.Errorf("failed to process path: %w", err)
	}

	// Sanitize the path (replace invalid characters)
	processedPath := sanitizePath(pathBuf.String())
	fullPath := filepath.Join(s.targetDir, processedPath)

	// Check if file already exists
	if file.Optional {
		if _, err := os.Stat(fullPath); err == nil {
			return "", nil // Skip existing optional files
		}
	}

	// Create parent directory
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	// Process the content template
	contentTmpl, err := template.New("content").Parse(file.Content)
	if err != nil {
		return "", fmt.Errorf("invalid content template: %w", err)
	}

	var contentBuf bytes.Buffer
	if err := contentTmpl.Execute(&contentBuf, s.info); err != nil {
		return "", fmt.Errorf("failed to process content: %w", err)
	}

	// Determine file mode
	mode := os.FileMode(0644)
	if file.Mode != 0 {
		mode = os.FileMode(file.Mode)
	}

	// Write the file
	if err := os.WriteFile(fullPath, contentBuf.Bytes(), mode); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fullPath, nil
}

// sanitizePath cleans up a path, replacing characters that might be
// problematic in file names (like hyphens in Go package names).
func sanitizePath(path string) string {
	// For Go packages, replace hyphens with underscores in directory names
	// This is needed for paths like src/{{.Name}}/main.py where Name might be "my-project"
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Don't modify file names, only directory names that are also package names
		if i < len(parts)-1 && strings.HasSuffix(parts[len(parts)-1], ".py") {
			// For Python packages, replace hyphens with underscores
			parts[i] = strings.ReplaceAll(part, "-", "_")
		}
	}
	return strings.Join(parts, string(filepath.Separator))
}
