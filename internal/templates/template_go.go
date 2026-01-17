package templates

var goTemplate = Template{
	Name:        "go",
	Language:    "Go",
	Description: "Go project with module support",
	Features: []string{
		"go.mod with module initialization",
		"Main package with hello world",
		"Makefile for common tasks",
		".gitignore for Go projects",
		".vibe configuration",
	},
	Files: []TemplateFile{
		{
			Path: "go.mod",
			Content: `module {{.Name}}

go 1.21
`,
		},
		{
			Path: "main.go",
			Content: `package main

import "fmt"

func main() {
	fmt.Println("Hello, {{.Name}}!")
}
`,
		},
		{
			Path: "Makefile",
			Content: `.PHONY: build run test clean

# Build the application
build:
	go build -o bin/{{.Name}} .

# Run the application
run:
	go run .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy
`,
		},
		{
			Path: ".gitignore",
			Content: `# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Coverage
coverage.out
coverage.html

# Dependency directories
vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Build output
dist/

# Environment files
.env
.env.local
`,
		},
		{
			Path: "README.md",
			Content: `# {{.Name}}

{{.Description}}

## Getting Started

### Prerequisites

- Go 1.21 or later

### Installation

` + "```bash" + `
go mod download
` + "```" + `

### Running

` + "```bash" + `
make run
` + "```" + `

### Testing

` + "```bash" + `
make test
` + "```" + `

## Project Structure

` + "```" + `
{{.Name}}/
├── main.go          # Application entry point
├── go.mod           # Go module definition
├── Makefile         # Build automation
├── .vibe/           # Vibe CLI configuration
│   └── config.yaml
└── .vibeignore      # Files excluded from vibe indexing
` + "```" + `

## License

MIT
`,
		},
		{
			Path: ".vibe/config.yaml",
			Content: `# Vibe CLI Project Configuration
# This file contains project-specific settings for vibe-cli

# Project metadata
project:
  name: "{{.Name}}"
  description: "{{.Description}}"
  language: go

# LLM settings (inherits from global config if not set)
# llm:
#   model: ""
#   temperature: 0.7

# Security settings
security:
  # Additional trusted paths for this project
  trusted_paths: []

  # Project-specific denied paths
  denied_paths:
    - bin/
    - vendor/

# Commands specific to this project
commands:
  # Commands that are always allowed in this project
  allowed:
    - go build
    - go test
    - go run
    - go mod
    - make
`,
		},
		{
			Path: ".vibeignore",
			Content: `# Files and directories to exclude from vibe indexing

# Build artifacts
bin/
dist/
vendor/

# Dependencies
go.sum

# Test artifacts
coverage.out
coverage.html
*.test

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
*.exe
*.dll
*.so
*.dylib
*.zip
*.tar.gz
`,
		},
	},
}
