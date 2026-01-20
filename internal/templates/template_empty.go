package templates

var emptyTemplate = Template{
	Name:        "empty",
	Language:    "Any",
	Description: "Vibe configuration only (for existing projects)",
	Features: []string{
		".vibe/config.yaml for project settings",
		".vibeignore for index exclusions",
		"No project files created",
	},
	Files: []TemplateFile{
		{
			Path: ".vibe/config.yaml",
			Content: `# Vibe CLI Project Configuration
# This file contains project-specific settings for flow-cli

# Project metadata
project:
  name: "{{.Name}}"
  description: "{{.Description}}"
  # language: auto-detected

# LLM settings (inherits from global config if not set)
# Uncomment to override global settings
# llm:
#   model: ""
#   temperature: 0.7

# Security settings
security:
  # Additional trusted paths for this project
  trusted_paths: []

  # Project-specific denied paths
  denied_paths: []

# Commands specific to this project
commands:
  # Commands that are always allowed in this project
  allowed: []

  # Commands that are blocked in this project
  blocked: []
`,
		},
		{
			Path: ".vibeignore",
			Content: `# Files and directories to exclude from vibe indexing
# Add patterns below, one per line

# Dependencies
node_modules/
vendor/
.venv/
venv/

# Build artifacts
dist/
build/
target/
bin/
out/

# IDE and editor files
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Large or binary files
*.zip
*.tar.gz
*.exe
*.dll
*.so
*.dylib

# Logs
*.log
logs/

# Environment files with secrets
.env
.env.local
.env.*.local
`,
		},
	},
}
