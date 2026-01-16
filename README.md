# vibe-cli

A self-hosted AI coding assistant CLI that uses local LLMs to help you code, plan, and build software projects.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Self-Hosted LLMs** - Works with Ollama, LocalAI, LM Studio, vLLM, and any OpenAI-compatible API
- **Interactive Chat** - Conversational coding assistance with streaming responses
- **Architecture Mode** - Plan before you build with structured requirements gathering
- **Tool Execution** - Read/write files, run commands, manage processes
- **Web Search** - Search the web via SearXNG for up-to-date information
- **Permission-First** - Always asks before making changes (unless you opt out)
- **Session Persistence** - Save and resume chat sessions
- **Cross-Platform** - Works on Windows, macOS, and Linux

## Quick Start

### Prerequisites

- [Go 1.21+](https://golang.org/dl/) (for building from source)
- [Ollama](https://ollama.ai/) or another supported LLM backend

### Installation

**From Source:**
```bash
git clone https://github.com/mateus/vibe-cli.git
cd vibe-cli
make build
make install
```

**Or build directly:**
```bash
go install github.com/mateus/vibe-cli@latest
```

### First Run

1. Start your LLM backend (e.g., Ollama):
   ```bash
   ollama serve
   ollama pull llama3:8b
   ```

2. Initialize configuration:
   ```bash
   vibe config init
   ```

3. Start chatting:
   ```bash
   vibe chat
   ```

## Usage

### Interactive Chat

Start an interactive session with your AI assistant:

```bash
vibe chat
```

Inside chat, you can:
- Ask questions about code
- Request file modifications
- Run commands
- Search the web

**Chat Commands:**
- `/help` - Show help
- `/clear` - Clear conversation
- `/status` - Show session info
- `/save` - Save session
- `/quit` - Exit (auto-saves)

### Single Prompt

Run a single prompt and get a response:

```bash
vibe run "Explain what a goroutine is"
vibe run --model mistral:7b "Write a hello world in Rust"
```

### Architecture Mode

Enter planning mode for larger features:

```bash
vibe arch "Add user authentication to my app"
```

This will:
1. Ask clarifying questions about requirements
2. Analyze your codebase structure
3. Create a detailed implementation plan
4. Wait for your approval before making changes

### Session Management

```bash
vibe session list              # List saved sessions
vibe session resume            # Resume a session (interactive)
vibe session resume abc123     # Resume by ID
vibe session delete abc123     # Delete a session
```

## Configuration

### Config File

Create a config file at `~/.vibe/config.yaml`:

```yaml
llm:
  provider: ollama
  endpoint: http://localhost:11434
  model: llama3:8b
  temperature: 0.7

search:
  enabled: true
  provider: searxng
  endpoint: http://localhost:8080

security:
  auto_approve: false
  project_dir: .
  denied_paths:
    - ~/.ssh
    - ~/.aws
```

### Interactive Setup

Configure your LLM provider interactively:

```bash
vibe config provider
```

### View Configuration

```bash
vibe config show
```

## Supported LLM Providers

| Provider | Endpoint | API Key Required |
|----------|----------|------------------|
| Ollama | `http://localhost:11434` | No |
| LocalAI | `http://localhost:8080` | No |
| LM Studio | `http://localhost:1234` | No |
| vLLM | `http://localhost:8000` | No |
| text-generation-webui | `http://localhost:5000` | No |
| OpenAI | `https://api.openai.com` | Yes |

The provider is auto-detected based on the endpoint port, or you can specify it explicitly:

```bash
vibe config set llm.provider lmstudio
vibe config set llm.endpoint http://localhost:1234
```

## Global Flags

```
--config string    Config file path
--model, -m        Model to use
--auto-approve     Skip confirmation prompts
--verbose, -v      Enable verbose output
--debug            Enable debug output
```

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
vibe completion bash > /etc/bash_completion.d/vibe

# Zsh
vibe completion zsh > "${fpath[1]}/_vibe"

# Fish
vibe completion fish > ~/.config/fish/completions/vibe.fish

# PowerShell
vibe completion powershell > vibe.ps1
```

## Security

vibe-cli operates with a permission-first model:

- **File Access** - Only files within the project directory can be accessed
- **Command Execution** - Commands require approval (configurable allowlist)
- **Sensitive Paths** - `.ssh`, `.aws`, `.gnupg` are blocked by default
- **Explicit Consent** - Every action requires approval unless auto-approve is enabled

To enable auto-approve (use with caution):
```bash
vibe chat --auto-approve
# or in config
vibe config set security.auto_approve true
```

## Project Structure

```
vibe-cli/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command, global flags
│   ├── chat.go            # Interactive chat
│   ├── run.go             # Single prompt
│   ├── arch.go            # Architecture mode
│   ├── config.go          # Configuration management
│   ├── session.go         # Session management
│   └── ...
├── internal/
│   ├── agent/             # Agent orchestration
│   ├── llm/               # LLM client abstraction
│   ├── tools/             # Executable tools
│   ├── sandbox/           # Permission system
│   ├── search/            # Web search
│   ├── context/           # Session management
│   └── ui/                # Terminal UI
├── config.example.yaml    # Example configuration
├── Makefile               # Build automation
└── install.sh             # Installation script
```

## Development

### Building

```bash
make build          # Build binary
make install        # Install to GOPATH/bin
make test           # Run tests
make lint           # Run linter
make release        # Build for all platforms
```

### Version Information

```bash
vibe version
```

Output includes version, commit hash, build date, and Go version.

## Troubleshooting

### Cannot connect to LLM service

1. Ensure your LLM backend is running:
   ```bash
   # For Ollama
   ollama serve
   ```

2. Check the endpoint configuration:
   ```bash
   vibe config show
   ```

3. Test connectivity:
   ```bash
   curl http://localhost:11434/api/tags  # Ollama
   ```

### No models available

Pull a model first:
```bash
ollama pull llama3:8b
```

### Permission denied errors

Check your security configuration:
```bash
vibe config show
```

Ensure the project directory is correctly set and paths are not in the denied list.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Charmbracelet](https://github.com/charmbracelet) - Terminal UI components
- [Ollama](https://ollama.ai/) - Local LLM runtime
