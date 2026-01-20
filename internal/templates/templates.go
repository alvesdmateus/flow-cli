// Package templates provides project scaffolding templates for vibe init.
package templates

// Template represents a project template with its files and configuration.
type Template struct {
	Name        string        // Template identifier (e.g., "go", "python")
	Language    string        // Primary language
	Description string        // Human-readable description
	Features    []string      // List of features/tools included
	Files       []TemplateFile // Files to create
}

// TemplateFile represents a single file in a template.
type TemplateFile struct {
	Path     string // Relative path from project root
	Content  string // File content (can contain {{.Name}}, {{.Description}}, etc.)
	Mode     uint32 // File permissions (0644 default)
	Optional bool   // Skip if file already exists
}

// ProjectInfo contains information about the project being created.
type ProjectInfo struct {
	Name        string // Project name
	Description string // Project description
	Author      string // Author name
	Path        string // Absolute path to project
}

// GetAllTemplates returns all available templates.
func GetAllTemplates() []Template {
	return []Template{
		goTemplate,
		pythonTemplate,
		nodeTemplate,
		rustTemplate,
		emptyTemplate,
	}
}

// GetTemplate returns a template by name, or nil if not found.
func GetTemplate(name string) *Template {
	for _, t := range GetAllTemplates() {
		if t.Name == name {
			return &t
		}
	}
	return nil
}
