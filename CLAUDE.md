# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

flow-cli is a coding assistant CLI that uses self-hosted LLMs to suggest, architect, plan, and create artifacts for projects.

## Technology Stack

- **Language:** Go
- **CLI Framework:** Cobra + Viper
- **Terminal UI:** Charmbracelet (huh, bubbletea, lipgloss)
- **LLM Integration:** Ollama SDK + OpenAI-compatible API

## Build & Development

```bash
# Using Make (recommended)
make build              # Build with version info
make install            # Install to GOPATH/bin
make test               # Run all tests
make lint               # Run linter
make release            # Build for all platforms

# Using Go directly
go build -o vibe .      # Build binary
go test ./...           # Run all tests
go mod tidy             # Clean dependencies
```

## Project Structure

```
cmd/                       # Cobra commands
  root.go                  # Root command, global flags
  run.go                   # Single prompt execution
  chat.go                  # Interactive chat session
  arch.go                  # Architecture planning mode
  config.go                # Configuration management command
  session.go               # Session management commands
  index.go                 # Semantic search index management
  init.go                  # Project initialization with templates
  review.go                # Code review command
  test.go                  # Test running and generation
  fix.go                   # Auto-fix linter errors
  explain.go               # Code/error explanation
  completion.go            # Shell completion generation
  version.go               # Version information
  lsp.go                   # Language Server Protocol server
internal/
  agent/                   # Agent orchestration
    agent.go               # Main agent loop, tool execution
    planner.go             # Architecture mode planner
  app/                     # Application container
    app.go                 # Dependency injection, initialization
  config/                  # Configuration management (Viper)
  context/                 # Conversation management
    manager.go             # Message history, session persistence
  indexing/                # Semantic search indexing
    embeddings.go          # Ollama embedding client
    store.go               # SQLite vector store
    indexer.go             # File indexing orchestrator
  llm/                     # LLM client abstraction
    client.go              # Interface definitions, auto-detection
    ollama.go              # Ollama implementation
    openai_compat.go       # OpenAI-compatible (LocalAI, LM Studio, vLLM)
  logging/                 # Logging infrastructure
    logger.go              # Structured logging with levels
  lsp/                     # Language Server Protocol
    server.go              # LSP server, JSON-RPC handling
    handler.go             # Request handlers, document management
    types.go               # LSP protocol type definitions
  sandbox/                 # Permission & security system
    policy.go              # Security policy definitions
    validator.go           # Path and command validation
    permissions.go         # Permission manager with approval flow
  secrets/                 # Secrets detection
    detector.go            # Pattern-based secret detection
  search/                  # Web search abstraction
    client.go              # Search interface
    searxng.go             # SearXNG implementation
  tools/                   # Executable tools for agent
    registry.go            # Tool interface and registry
    filesystem.go          # read_file, write_file, list_files, create_directory
    shell.go               # run_command
    search.go              # web_search, fetch_url
    semantic.go            # semantic_search, index_status, reindex_file
    process.go             # check_port, kill_process, start_process
  ui/                      # Terminal UI components (huh/bubbletea)
    prompt.go              # Multiple choice, confirmations, approval UI
    chat.go                # Interactive chat interface
    diff.go                # File diff display
    plan.go                # Plan formatting and display
  templates/               # Project scaffolding templates
    templates.go           # Template registry
    scaffolder.go          # File generation from templates
  review/                  # Code review system
    reviewer.go            # AI-powered code review
    parser.go              # Diff parsing
  testing/                 # Test management
    runner.go              # Test execution
    generator.go           # AI test generation
    analyzer.go            # Failure analysis
  fixer/                   # Linter integration
    linter.go              # Multi-language linter support
    fixer.go               # AI-powered fix generation
  explainer/               # Code explanation
    explainer.go           # AI code/error explanation
```

## CLI Commands

```bash
flow run "prompt"          # Single prompt execution
flow chat                  # Interactive chat session
flow arch                  # Architecture planning mode
flow lsp                   # Start LSP server for editor integration
flow config show           # Show current configuration
flow config set key value  # Set configuration value
flow config provider       # Interactive provider setup
flow config init           # Create default config file
flow session list          # List saved sessions
flow session resume [id]   # Resume a saved session
flow session delete <id>   # Delete a session
flow index build           # Build/rebuild the semantic search index
flow index status          # Show index statistics
flow index clear           # Clear the index database
flow index search "query"  # Search the codebase using natural language
flow init                  # Initialize new project with templates
flow init --template go    # Use specific template (go, python, node, rust)
flow review                # Review uncommitted changes
flow review --staged       # Review only staged changes
flow review --pr 123       # Review a pull request (requires gh CLI)
flow test                  # Run project tests
flow test --generate file  # Generate tests for a file
flow test --coverage       # Run tests with coverage report
flow fix                   # Auto-fix linter errors
flow fix --dry-run         # Show fixes without applying
flow explain file.go       # Explain code in a file
flow explain --error "msg" # Explain an error message
flow secrets scan          # Scan for hardcoded secrets
flow secrets check         # Check staged files for secrets (pre-commit)
flow completion bash       # Generate shell completion
flow version               # Show version info
```

## Global Flags

```bash
--config string    # Config file path
--model, -m        # Model to use
--auto-approve     # Skip confirmation prompts
--verbose, -v      # Enable verbose output
--debug            # Enable debug output
```

## Chat Commands

Inside `flow chat`:
- `/help` - Show help
- `/clear` - Clear conversation
- `/status` - Show conversation status
- `/save` - Save current session
- `/sessions` - List saved sessions
- `/quit` - Exit (auto-saves)

## Configuration

Configuration loads from (in order of precedence):
1. Command-line flags
2. Environment variables (FLOW_ prefix)
3. `~/.flow/config.yaml` or `./.flow.yaml`
4. Built-in defaults

### Supported LLM Providers

- `ollama` - Ollama (default, localhost:11434)
- `openai-compatible` - Any OpenAI-compatible API
- `localai` - LocalAI (localhost:8080)
- `lmstudio` - LM Studio (localhost:1234)
- `vllm` - vLLM (localhost:8000)
- `textgen` - text-generation-webui (localhost:5000)
- `openai` - OpenAI API (requires API key)

## Core Design Principles

### Permission Model
- **Always ask permission** before executing actions, unless user explicitly grants auto-accept
- Clarify changes before making them
- Self-contained operation with explicit user consent

### Sandboxed File Access
- Can only read files within the allowed project directory
- Trusted directories must be explicitly configured
- Sensitive paths (`.ssh`, `.aws`, etc.) are blocked by default

### User Interaction
- When AI has doubts, present **multiple choice questions** for user selection
- Never assume; always clarify ambiguous requirements

### Architecture Mode
- Dedicated planning mode for designing before implementation
- Multi-phase workflow: requirements gathering, analysis, planning
- Clarifying questions flow with multiple choice options

### Self-Hosted LLM
- Supports user-selectable models
- Provider auto-detection based on endpoint
- Designed for self-hosted backends (not cloud-dependent)

### Development Commands
The CLI can execute development-related commands:
- Create folders and files
- Check ports
- Kill and start processes
- Run build/test commands

### Web Search
- LLM can search the web via SearXNG
- Results are used to inform responses with up-to-date data

### Session Persistence
- Chat sessions are auto-saved on exit
- Sessions can be listed, resumed, and deleted
- Stored in `~/.flow/sessions/`

### Semantic Search
- Natural language code search using embeddings
- Requires Ollama with an embedding model (default: `nomic-embed-text`)
- Index stored in `.flow/index.db` (SQLite)
- Automatic re-indexing on file changes (when enabled)
- Tools: `semantic_search`, `index_status`, `reindex_file`

**Configuration:**
```yaml
indexing:
  enabled: false           # Enable/disable semantic search (opt-in)
  embedding_model: "nomic-embed-text"  # Ollama embedding model
  auto_index: true         # Auto-index when index is empty
  watch_changes: true      # Watch files and re-index on changes
```

**Supported file types:** `.go`, `.py`, `.js`, `.ts`, `.tsx`, `.jsx`, `.rs`, `.java`, `.c`, `.cpp`, `.h`, `.hpp`, `.rb`, `.php`, `.md`, `.txt`

**Usage:**
```bash
# Build the index first
flow index build

# Enable in config for chat/arch tools
flow config set indexing.enabled true

# Search from CLI
flow index search "function that handles authentication"

# Or use the semantic_search tool in chat
flow chat
> Use semantic_search to find functions related to "file reading"
```

**Troubleshooting:**
- If model not found: `ollama pull nomic-embed-text`
- Large codebases may take time to index
- Use `.flowignore` to exclude files from indexing

### Secrets Detection

flow-cli includes built-in secrets detection to prevent accidental commits of sensitive data.

**Detected secret types:**
- API keys: AWS, Google, Stripe, SendGrid, generic
- Tokens: GitHub, GitLab, Slack, npm, JWT
- Private keys: RSA, OpenSSH, EC, PGP
- Connection strings: PostgreSQL, MongoDB, MySQL, Redis
- Passwords: Generic password patterns

**CLI Commands:**
```bash
flow secrets scan                    # Scan current directory
flow secrets scan ./src              # Scan specific directory
flow secrets scan --file config.go   # Scan specific file
flow secrets scan --severity high    # Only show high+ severity
flow secrets check                   # Check staged git files (pre-commit)
```

**Git Integration:**
- `git_commit` tool automatically scans staged files before committing
- Use `skip_secrets_check: true` parameter to bypass (not recommended)
- Can be used as a pre-commit hook via `flow secrets check`

**Pre-commit hook setup:**
```bash
# Add to .git/hooks/pre-commit:
#!/bin/sh
flow secrets check
```

**False positive handling:**
- Placeholders (`your_api_key_here`, `<your-key>`) are ignored
- Template variables (`${API_KEY}`, `{{.ApiKey}}`) are ignored
- Test/fake keys are detected and filtered
- Comments with example patterns are ignored

## Adding New Features

When adding new tools:
1. Create tool in `internal/tools/`
2. Implement `Tool` interface from `registry.go`
3. Register in `internal/tools/setup.go`

When adding new commands:
1. Create command file in `cmd/`
2. Add command to parent in `init()`
3. Update help text and documentation

When adding new LLM providers:
1. Implement `Client` interface from `internal/llm/client.go`
2. Add to provider switch in `NewClientWithConfig`
3. Add preset in `openai_compat.go` if applicable

## IDE & Editor Integration

flow-cli includes a Language Server Protocol (LSP) server for editor integration.

### LSP Server

Start with `flow lsp`. The server communicates via stdin/stdout.

**Supported Features:**
- Document synchronization (open, change, close, save)
- Code completion with `@flow` triggers
- Hover information (AI-powered with LLM)
- Code actions (Explain, Generate Tests, Refactor, Fix Error)
- Execute commands (flow.runPrompt, flow.explainCode, etc.)

### Editor Extensions

Extensions are provided in `editors/`:
- **VS Code** (`editors/vscode/`) - Full extension with chat panel
- **Neovim** (`editors/neovim/`) - Lua plugin with LSP integration
- **JetBrains** (`editors/jetbrains/`) - IntelliJ platform plugin

### Project Configuration

Project-specific settings in `.flow/config.yaml`:
- Custom prompts and templates
- Tool configurations
- Build/test command overrides

See `editors/config.yaml.example` for full options.
