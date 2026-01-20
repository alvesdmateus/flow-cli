package templates

var pythonTemplate = Template{
	Name:        "python",
	Language:    "Python",
	Description: "Python project with virtual environment support",
	Features: []string{
		"pyproject.toml for modern Python packaging",
		"requirements.txt for dependencies",
		"src layout with package structure",
		".gitignore for Python projects",
		".flow configuration",
	},
	Files: []TemplateFile{
		{
			Path: "pyproject.toml",
			Content: `[build-system]
requires = ["setuptools>=61.0", "wheel"]
build-backend = "setuptools.build_meta"

[project]
name = "{{.Name}}"
version = "0.1.0"
description = "{{.Description}}"
readme = "README.md"
requires-python = ">=3.9"
license = {text = "MIT"}
authors = [
    {name = "{{.Author}}"}
]
classifiers = [
    "Development Status :: 3 - Alpha",
    "Intended Audience :: Developers",
    "License :: OSI Approved :: MIT License",
    "Programming Language :: Python :: 3",
    "Programming Language :: Python :: 3.9",
    "Programming Language :: Python :: 3.10",
    "Programming Language :: Python :: 3.11",
    "Programming Language :: Python :: 3.12",
]
dependencies = []

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "pytest-cov>=4.0",
    "black>=23.0",
    "ruff>=0.1.0",
    "mypy>=1.0",
]

[project.scripts]
{{.Name}} = "{{.Name}}.main:main"

[tool.setuptools.packages.find]
where = ["src"]

[tool.black]
line-length = 88
target-version = ["py39"]

[tool.ruff]
line-length = 88
select = ["E", "F", "I", "N", "W", "UP"]

[tool.mypy]
python_version = "3.9"
strict = true

[tool.pytest.ini_options]
testpaths = ["tests"]
addopts = "-v --cov=src/{{.Name}} --cov-report=term-missing"
`,
		},
		{
			Path: "requirements.txt",
			Content: `# Production dependencies
# Add your dependencies here

# Development dependencies (install with: pip install -r requirements-dev.txt)
`,
		},
		{
			Path: "requirements-dev.txt",
			Content: `# Development dependencies
-r requirements.txt
pytest>=7.0
pytest-cov>=4.0
black>=23.0
ruff>=0.1.0
mypy>=1.0
`,
		},
		{
			Path: "src/{{.Name}}/__init__.py",
			Content: `"""{{.Description}}"""

__version__ = "0.1.0"
`,
		},
		{
			Path: "src/{{.Name}}/main.py",
			Content: `"""Main entry point for {{.Name}}."""


def main() -> None:
    """Main function."""
    print("Hello from {{.Name}}!")


if __name__ == "__main__":
    main()
`,
		},
		{
			Path: "tests/__init__.py",
			Content: `"""Tests for {{.Name}}."""
`,
		},
		{
			Path: "tests/test_main.py",
			Content: `"""Tests for main module."""

from {{.Name}}.main import main


def test_main(capsys):
    """Test main function."""
    main()
    captured = capsys.readouterr()
    assert "Hello from {{.Name}}" in captured.out
`,
		},
		{
			Path: ".gitignore",
			Content: `# Byte-compiled / optimized / DLL files
__pycache__/
*.py[cod]
*$py.class

# C extensions
*.so

# Distribution / packaging
.Python
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
*.egg-info/
.installed.cfg
*.egg

# Virtual environments
.venv/
venv/
ENV/
env/

# Testing
.tox/
.nox/
.coverage
.coverage.*
htmlcov/
.pytest_cache/
.mypy_cache/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Environment files
.env
.env.local
*.env

# Jupyter Notebook
.ipynb_checkpoints
`,
		},
		{
			Path: "README.md",
			Content: `# {{.Name}}

{{.Description}}

## Getting Started

### Prerequisites

- Python 3.9 or later

### Installation

` + "```bash" + `
# Create virtual environment
python -m venv .venv

# Activate virtual environment
source .venv/bin/activate  # Linux/macOS
# .venv\Scripts\activate   # Windows

# Install dependencies
pip install -e ".[dev]"
` + "```" + `

### Running

` + "```bash" + `
python -m {{.Name}}.main
# or
{{.Name}}
` + "```" + `

### Testing

` + "```bash" + `
pytest
` + "```" + `

### Code Quality

` + "```bash" + `
# Format code
black src/ tests/

# Lint code
ruff check src/ tests/

# Type checking
mypy src/
` + "```" + `

## Project Structure

` + "```" + `
{{.Name}}/
├── src/
│   └── {{.Name}}/
│       ├── __init__.py
│       └── main.py
├── tests/
│   ├── __init__.py
│   └── test_main.py
├── pyproject.toml
├── requirements.txt
├── .flow/
│   └── config.yaml
└── .flowignore
` + "```" + `

## License

MIT
`,
		},
		{
			Path: ".flow/config.yaml",
			Content: `# Flow CLI Project Configuration
# This file contains project-specific settings for flow-cli

# Project metadata
project:
  name: "{{.Name}}"
  description: "{{.Description}}"
  language: python

# Security settings
security:
  trusted_paths: []
  denied_paths:
    - .venv/
    - venv/
    - __pycache__/
    - .pytest_cache/
    - .mypy_cache/
    - dist/
    - build/
    - "*.egg-info/"

# Commands specific to this project
commands:
  allowed:
    - python
    - pip install
    - pip freeze
    - pytest
    - black
    - ruff
    - mypy
`,
		},
		{
			Path: ".flowignore",
			Content: `# Files and directories to exclude from flow indexing

# Virtual environments
.venv/
venv/
ENV/
env/

# Build artifacts
build/
dist/
*.egg-info/
*.egg

# Cache directories
__pycache__/
.pytest_cache/
.mypy_cache/
.ruff_cache/
.tox/
.nox/

# Coverage
.coverage
htmlcov/

# IDE and editor files
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Jupyter
.ipynb_checkpoints/
`,
		},
	},
}
