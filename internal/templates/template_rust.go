package templates

var rustTemplate = Template{
	Name:        "rust",
	Language:    "Rust",
	Description: "Rust project with Cargo",
	Features: []string{
		"Cargo.toml with project metadata",
		"Main binary with hello world",
		"Library module structure",
		"Unit tests included",
		".flow configuration",
	},
	Files: []TemplateFile{
		{
			Path: "Cargo.toml",
			Content: `[package]
name = "{{.Name}}"
version = "0.1.0"
edition = "2021"
description = "{{.Description}}"
authors = ["{{.Author}}"]
license = "MIT"
readme = "README.md"

[dependencies]

[dev-dependencies]

[[bin]]
name = "{{.Name}}"
path = "src/main.rs"

[lib]
name = "{{.Name}}"
path = "src/lib.rs"

[profile.release]
lto = true
codegen-units = 1
panic = "abort"
strip = true
`,
		},
		{
			Path: "src/main.rs",
			Content: `//! {{.Name}} - {{.Description}}

use {{.Name}}::greet;

fn main() {
    println!("{}", greet("{{.Name}}"));
}
`,
		},
		{
			Path: "src/lib.rs",
			Content: `//! {{.Name}}
//!
//! {{.Description}}

/// Returns a greeting message for the given name.
///
/// # Examples
///
/// ` + "```" + `
/// use {{.Name}}::greet;
/// assert_eq!(greet("World"), "Hello, World!");
/// ` + "```" + `
pub fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_greet() {
        assert_eq!(greet("World"), "Hello, World!");
    }

    #[test]
    fn test_greet_with_name() {
        assert_eq!(greet("Rust"), "Hello, Rust!");
    }
}
`,
		},
		{
			Path: ".gitignore",
			Content: `# Cargo build artifacts
/target/
debug/
release/

# Cargo lock file (uncomment for libraries)
# Cargo.lock

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Environment
.env
.env.local

# Profiling
perf.data
perf.data.old
flamegraph.svg
`,
		},
		{
			Path: "README.md",
			Content: `# {{.Name}}

{{.Description}}

## Getting Started

### Prerequisites

- Rust 1.70 or later
- Cargo (comes with Rust)

### Installation

` + "```bash" + `
# Build the project
cargo build

# Build for release
cargo build --release
` + "```" + `

### Running

` + "```bash" + `
# Run in development mode
cargo run

# Run release build
cargo run --release
` + "```" + `

### Testing

` + "```bash" + `
# Run all tests
cargo test

# Run tests with output
cargo test -- --nocapture

# Run specific test
cargo test test_greet
` + "```" + `

### Code Quality

` + "```bash" + `
# Check for errors without building
cargo check

# Format code
cargo fmt

# Lint code
cargo clippy

# Run all checks
cargo fmt -- --check && cargo clippy -- -D warnings && cargo test
` + "```" + `

## Project Structure

` + "```" + `
{{.Name}}/
├── src/
│   ├── main.rs    # Binary entry point
│   └── lib.rs     # Library with core logic
├── Cargo.toml     # Package manifest
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
  language: rust

# Security settings
security:
  trusted_paths: []
  denied_paths:
    - target/

# Commands specific to this project
commands:
  allowed:
    - cargo build
    - cargo run
    - cargo test
    - cargo check
    - cargo fmt
    - cargo clippy
    - cargo doc
    - cargo clean
`,
		},
		{
			Path: ".flowignore",
			Content: `# Files and directories to exclude from flow indexing

# Cargo build artifacts
target/

# Cargo lock (optional - uncomment for binaries)
# Cargo.lock

# IDE and editor files
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Profiling data
perf.data
perf.data.old
flamegraph.svg
`,
		},
	},
}
