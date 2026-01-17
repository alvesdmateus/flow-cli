# vibe-cli Development Roadmap

> Last updated: 2026-01-17
> Status: Phase 1 - Foundation

## Overview

This roadmap outlines the development plan to make vibe-cli a competitive AI coding assistant comparable to Gemini CLI, Claude Code, Cursor, and similar tools.

## Current Status

### Implemented Features
- [x] CLI commands: run, chat, arch, config, session, completion, version
- [x] LLM providers: Ollama, OpenAI-compatible (LocalAI, LM Studio, vLLM)
- [x] Tools: filesystem (read/write/list/delete), shell, process management, web search
- [x] Sandbox: 5-tier permission system, path validation, command blocking
- [x] UI: Charmbracelet-based terminal UI, diff display, plan formatting
- [x] Sessions: persistence, resume, list, delete
- [x] CI/CD: GitHub Actions workflows, automated releases

---

## Phase 1: Foundation (Sprints 1-2)
**Goal:** Core reliability and essential missing features

### Sprint 1: Core Completeness ✅
- [x] Implement `fetch_url` tool (HTML parsing, markdown extraction)
- [x] Add `edit_file` tool (patch-based edits instead of full rewrites)
- [x] Add `grep_search` tool (regex search across codebase)
- [x] Token counting and cost estimation display
- [x] Add `insert_lines` and `delete_lines` tools

### Sprint 1.5: Context & Token Management ✅
- [x] Context window management with smart truncation
- [x] Conversation summarization (compress old messages)
- [x] Sliding window context (keep recent + important messages)
- [x] Token budget allocation (reserve tokens for response)
- [x] Context priority scoring (rank messages by relevance)
- [x] Automatic context pruning when approaching limits
- [x] Message importance tagging (system, tool results, user)
- [x] Context size display in UI (tokens used / max)
- [x] Configurable context strategies per model
- [ ] Memory/facts extraction from conversations (future)

### Sprint 2: Git Integration ✅
- [x] `git_status`, `git_diff`, `git_commit` tools
- [x] `git_log` with commit analysis
- [x] `git_add`, `git_branch`, `git_checkout` tools
- [ ] PR/MR description generation (future)
- [ ] Conflict detection and resolution suggestions (future)

---

## Phase 2: Code Intelligence (Sprints 3-4)
**Goal:** Semantic code understanding

### Sprint 3: Code Analysis ✅
- [x] AST parsing for Go (using go/ast), regex parsing for Python, JS, TS, Rust
- [x] Symbol extraction (functions, classes, types, methods, interfaces, etc.)
- [x] `find_definition`, `find_references` tools
- [x] `code_outline` tool for file structure
- [x] `list_symbols` tool for codebase exploration
- [ ] Import/dependency analysis (future enhancement)

### Sprint 4: Codebase Indexing ✅
- [x] Local embedding generation (with Ollama)
- [x] Vector store for semantic search (SQLite with pure-Go driver)
- [x] `semantic_search` tool for natural language code queries
- [x] Automatic re-indexing on file changes
- [x] `.vibeignore` for excluding files from indexing

---

## Phase 3: Developer Experience (Sprints 5-6)
**Goal:** Productivity features

### Sprint 5: Enhanced Workflows ✅
- [x] `vibe init` - Project scaffolding with templates
- [x] `vibe review` - Code review mode (diff analysis)
- [x] `vibe test` - Test generation and execution
- [x] `vibe fix` - Auto-fix linter errors
- [x] `vibe explain` - Explain code/errors in detail

### Sprint 6: Terminal UX ✅
- [x] Multi-file edit preview with unified diff
- [x] Undo/redo for file changes
- [x] Keyboard shortcuts (Ctrl+C cancel, Ctrl+R retry)
- [x] Progress indicators for long operations
- [x] Command history with fuzzy search
- [x] Markdown rendering in terminal

---

## Phase 4: Security & Reliability (Sprints 7-8)
**Goal:** Production-grade safety

### Sprint 7: Security Hardening
- [ ] Command injection prevention (shell escaping audit)
- [ ] Path traversal protection (symlink resolution)
- [ ] Secrets detection (block commits with API keys)
- [ ] Audit logging (all file/command operations)
- [ ] Rate limiting for LLM calls
- [ ] Checksum verification for file writes

### Sprint 8: Reliability
- [ ] Automatic retries with exponential backoff
- [ ] Graceful degradation when LLM unavailable
- [ ] Transaction-like file operations (atomic writes)
- [ ] Backup before destructive operations
- [ ] Health checks for LLM providers
- [ ] Crash recovery (resume from last state)

---

## Phase 5: Advanced AI (Sprints 9-10)
**Goal:** Intelligent assistance

### Sprint 9: Planning & Reasoning
- [ ] Multi-step task decomposition
- [ ] Dependency graph for task ordering
- [ ] Self-validation (verify changes work)
- [ ] Automatic rollback on failure
- [ ] Learning from corrections (session-based)

### Sprint 10: Code Generation
- [ ] Test generation from implementation
- [ ] Documentation generation (JSDoc, GoDoc, etc.)
- [ ] Boilerplate generation (CRUD, API endpoints)
- [ ] Refactoring suggestions (extract function, rename)
- [ ] Bug fix suggestions from error messages

---

## Phase 6: Integrations (Sprints 11-12)
**Goal:** Ecosystem connectivity

### Sprint 11: External Tools
- [ ] Docker tools (`docker_build`, `docker_run`, `docker_logs`)
- [ ] Database tools (`query_db` for SQLite/Postgres)
- [ ] HTTP client tool (`http_request`)
- [ ] Package manager tools (npm, pip, cargo info)
- [ ] CI/CD status checking (GitHub Actions)

### Sprint 12: IDE & Editor
- [ ] LSP server mode (integrate with any editor)
- [ ] VS Code extension
- [ ] Neovim plugin
- [ ] JetBrains plugin (basic)
- [ ] `.vibe/config.yaml` project-level settings

---

## Priority Matrix

| Feature | Impact | Effort | Priority | Status |
|---------|--------|--------|----------|--------|
| fetch_url completion | High | Low | **P0** | ✅ Done |
| Token counting | Medium | Low | **P0** | ✅ Done |
| edit_file (patches) | High | Medium | **P0** | ✅ Done |
| grep_search | High | Low | **P0** | ✅ Done |
| Context management | High | High | **P0** | ✅ Done |
| Git integration | High | Medium | **P1** | ✅ Done |
| Context summarization | High | Medium | **P1** | ✅ Done |
| Code Analysis (AST/regex) | High | High | **P1** | ✅ Done |
| Secrets detection | High | Medium | **P1** | Pending |
| Semantic search | High | High | P2 | ✅ Done |
| LSP server mode | High | High | P2 | Pending |
| Test generation | Medium | Medium | P2 | Pending |

---

## Security Checklist (Ongoing)

- [ ] OWASP command injection audit
- [ ] Fuzz testing for file path handling
- [ ] Permission escalation prevention
- [ ] Secure credential storage (keyring integration)
- [ ] TLS verification for all HTTP calls
- [ ] Input sanitization for all tools
- [ ] Memory safety (no unbounded buffers)
- [ ] Dependency vulnerability scanning (in CI)

---

## Reliability Checklist (Ongoing)

- [ ] Unit test coverage > 70%
- [ ] Integration tests for all tools
- [ ] E2E tests for CLI commands
- [ ] Error handling audit
- [ ] Timeout handling for all external calls
- [ ] Resource cleanup (file handles, connections)

---

## Metrics to Track

| Metric | Description | Target |
|--------|-------------|--------|
| Tool Success Rate | % of tool executions that succeed | > 95% |
| Response Latency P50 | Median response time | < 2s |
| Response Latency P95 | 95th percentile response time | < 10s |
| Blocked Operations | Dangerous operations prevented | Track |
| User Corrections | How often users undo AI changes | < 20% |

---

## Check-in Schedule

Use this section to track periodic progress reviews.

### Check-in Template
```
## Check-in: YYYY-MM-DD

### Completed Since Last Check-in
-

### In Progress
-

### Blockers
-

### Next Sprint Focus
-

### Notes
-
```

---

### Check-in: 2026-01-16 (Initial)

#### Completed
- Project structure and core implementation
- CI/CD pipelines (GitHub Actions)
- Basic tool set (filesystem, shell, process, web search)
- Sandbox permission system
- Session persistence

#### In Progress
- Planning roadmap and priorities

#### Next Sprint Focus
- Sprint 1: Core completeness (fetch_url, edit_file, grep_search)
- Sprint 7 security items in parallel

#### Notes
- Starting with Phase 1 + Security hardening in parallel
- Focus on feature parity with essential tools first

---

### Check-in: 2026-01-16 (Sprint 1 Complete)

#### Completed
- **Sprint 1: Core Completeness** ✅
  - `fetch_url` tool with HTML→markdown conversion
  - `grep_search` tool with regex and context lines
  - `edit_file` tool for surgical text replacement
  - `insert_lines` and `delete_lines` tools
  - Token counting module with usage tracking
- Updated CI workflow to support `develop` branch
- Established branching workflow: feature → develop → main

#### In Progress
- Sprint 1 PR to develop branch

#### Next Sprint Focus
- **Sprint 1.5: Context & Token Management** (high priority)
  - Context window management
  - Conversation summarization
  - Token budget allocation
- Sprint 2: Git Integration (parallel)

#### Notes
- Context management is critical for long conversations
- Need to handle model-specific context limits (4K, 8K, 32K, 128K)

---

### Check-in: 2026-01-16 (Sprint 1.5 & 2 Complete)

#### Completed
- **Sprint 1.5: Context & Token Management** ✅
  - WindowManager with model-specific context limits
  - 4 pruning strategies (oldest, low-priority, summarize, hybrid)
  - Message priority scoring (Low, Medium, High, Critical)
  - Automatic context pruning when approaching limits
  - Context stats display (tokens used / max)
  - 16 test cases for context management

- **Sprint 2: Git Integration** ✅
  - `git_status` with short format parsing
  - `git_diff` with staged/unstaged support
  - `git_log` with filtering (author, date, file)
  - `git_commit` with auto-stage option
  - `git_add`, `git_branch`, `git_checkout` tools
  - MockGitRunner for comprehensive testing
  - 21 test cases for git tools

#### In Progress
- Roadmap update and planning

#### Next Sprint Focus
- **Sprint 3: Code Analysis**
  - Tree-sitter integration for AST parsing
  - Symbol extraction (functions, classes, types)
  - `find_definition`, `find_references` tools

#### Notes
- Phase 1 complete with all core features
- Moving to Phase 2: Code Intelligence
- Consider secrets detection in parallel (security)

---

### Check-in: 2026-01-16 (Sprint 3 Complete)

#### Completed
- **Sprint 3: Code Analysis** ✅
  - `internal/analysis` package with multi-language support
  - Go parser using `go/ast` for accurate AST parsing
  - Regex-based parsers for Python, JavaScript, TypeScript, Rust
  - Symbol extraction: functions, methods, classes, types, interfaces, structs, consts, vars
  - `code_outline` tool for file structure analysis
  - `find_definition` tool to locate symbol definitions
  - `find_references` tool for text-based reference search
  - `list_symbols` tool for codebase exploration
  - 10 tests for analysis package, 11 tests for tools
  - Integrated with tool registry

#### In Progress
- None

#### Next Sprint Focus
- **Sprint 4: Codebase Indexing**
  - Local embedding generation
  - Semantic search implementation

#### Notes
- Chose go/ast over tree-sitter for Go (native, zero dependencies)
- Regex parsers provide good coverage for common patterns
- Analysis tools ready for LLM use

---

### Check-in: 2026-01-16 (Sprint 4 Complete)

#### Completed
- **Sprint 4: Codebase Indexing** ✅
  - `internal/indexing` package with embedding and vector store support
  - `EmbeddingClient` using Ollama for local embedding generation
  - `VectorStore` with SQLite persistence using pure-Go driver (modernc.org/sqlite)
  - Cosine similarity search for semantic matching
  - `semantic_search` tool for natural language code queries
  - `index_status` tool to view index statistics
  - `reindex_file` tool for updating specific files
  - `.vibeignore` support for excluding files from indexing
  - File hash-based change detection for incremental re-indexing
  - 23 tests for indexing package, tools integrated with registry

#### In Progress
- None

#### Next Sprint Focus
- **Sprint 5: Enhanced Workflows**
  - `vibe init` - Project scaffolding
  - `vibe review` - Code review mode
  - `vibe test` - Test generation

#### Notes
- Used pure-Go SQLite driver to avoid CGO dependency
- Embeddings converted from float32 (Ollama) to float64 for search
- Phase 2: Code Intelligence now complete

---

### Check-in: 2026-01-17 (Sprint 5 - vibe init & review)

#### Completed
- **Sprint 5: `vibe init`** ✅
  - `cmd/init.go` - New CLI command with flags: `--template`, `--list-templates`, `--existing`, `--yes`
  - `internal/templates` package with scaffolding support
  - 5 project templates: Go, Python, Node.js/TypeScript, Rust, Empty
  - Interactive project initialization with name, description, author prompts
  - `.vibe/config.yaml` and `.vibeignore` generation for all templates
  - Template variable substitution ({{.Name}}, {{.Description}}, {{.Author}})
  - 12 tests for templates package

- **Sprint 5: `vibe review`** ✅
  - `cmd/review.go` - AI-powered code review command
  - `internal/review` package with diff parsing and LLM integration
  - Flags: `--staged`, `--commit`, `--pr`, `--focus`, `--detailed`
  - Focus areas: security, performance, bugs, style, all
  - Unified diff parser with hunk-level analysis
  - Language detection for 25+ languages
  - Structured JSON output from LLM with severity levels
  - PR review support via gh CLI
  - 12 tests for diff parser

- **Sprint 5: `vibe test`** ✅
  - `cmd/test.go` - Test running and generation command
  - `internal/testing` package with runner, generator, analyzer
  - Flags: `--generate`, `--coverage`, `--fix`, `--watch`, `--timeout`
  - Multi-language support: Go, Python, Node.js, Rust
  - Project type auto-detection
  - Test output parsing for each language
  - AI-powered test generation from source code
  - Failure analysis with fix suggestions
  - 17 tests for testing package

#### In Progress
- Sprint 5: Remaining enhanced workflows

#### Next Sprint Focus
- `vibe fix` - Auto-fix linter errors
- `vibe explain` - Explain code/errors in detail

#### Notes
- Templates include language-specific tooling configs (Makefile, pyproject.toml, tsconfig.json, Cargo.toml)
- Project-level vibe config allows per-project LLM and security settings
- Review command outputs issues by severity (critical, high, medium, low)
- Test command supports go test -json, pytest, Jest, cargo test output formats

---

### Check-in: 2026-01-17 (Sprint 5 Complete)

#### Completed
- **Sprint 5: Enhanced Workflows** ✅
  - `vibe fix` - Auto-fix linter errors with AI
    - `cmd/fix.go` - CLI command with flags: `--dry-run`, `--lint-only`, `--auto`
    - `internal/fixer` package with linter integration and fix generation
    - Multi-language linter support: golangci-lint, go vet, ruff, pylint, eslint, cargo clippy
    - JSON output parsing for each linter
    - LLM-powered fix generation
    - Interactive fix approval with diff preview
    - 10 tests for fixer package

  - `vibe explain` - Explain code/errors in detail
    - `cmd/explain.go` - CLI command with flags: `--error`, `--function`, `--stdin`, `--detailed`
    - `internal/explainer` package with code explanation capabilities
    - File explanation with line range support (file:42 or file:10-50)
    - Function extraction for targeted explanations
    - Error message analysis with fix suggestions
    - Stdin support for piped input
    - Language detection for 25+ file extensions
    - 10 tests for explainer package

#### In Progress
- None

#### Next Sprint Focus
- **Sprint 6: Terminal UX**
  - Multi-file edit preview with unified diff
  - Undo/redo for file changes
  - Keyboard shortcuts
  - Progress indicators

#### Notes
- Sprint 5 completes Phase 3: Developer Experience (Enhanced Workflows portion)
- All 5 enhanced workflow commands now implemented: init, review, test, fix, explain
- Linter integration supports the most popular linters for each language
- Explanation system uses function extraction with language-aware parsing

---

### Check-in: 2026-01-17 (Sprint 6 Complete)

#### Completed
- **Sprint 6: Terminal UX** ✅
  - Multi-file edit preview with unified diff
    - `EditPreview` manager for collecting and previewing file changes
    - Unified diff generation with hunk support
    - Summary view with added/removed line counts
    - Interactive approval flow with per-file review option

  - Undo/redo for file changes
    - `ChangeHistory` manager with full undo/redo stack
    - Records create, modify, and delete operations
    - Automatic file restoration on undo
    - Interactive undo/redo with confirmation

  - Progress indicators for long operations
    - `Spinner` with multiple animation styles
    - `ProgressBar` with percentage and ETA
    - `MultiProgress` for parallel operations
    - `WithProgress` helper for easy integration

  - Command history with fuzzy search
    - `CommandHistory` with persistence to file
    - Fuzzy matching with scoring (exact, contains, character)
    - Frequency-based ranking
    - `HistoryNavigator` for up/down navigation
    - `FuzzyFinder` for interactive search

  - Markdown rendering in terminal
    - Headers (H1-H3), lists, blockquotes
    - Code blocks with basic syntax highlighting
    - Inline formatting (bold, italic, code, links)
    - Table rendering with borders
    - Support for Go, Python, JavaScript, TypeScript, Rust syntax

  - 40+ tests for UI components

#### In Progress
- None

#### Next Sprint Focus
- **Sprint 7: Security Hardening**
  - Command injection prevention
  - Path traversal protection
  - Secrets detection

#### Notes
- Phase 3: Developer Experience is now complete
- All UI components are reusable and well-tested
- Progress indicators support multiple styles for different use cases
- Markdown rendering provides good terminal experience for AI responses

---

## Contributing

### Branching Workflow
```
feature/* → develop → main
```

When working on roadmap items:
1. Create a feature branch from `develop` (e.g., `feature/sprint-2-git`)
2. Reference the sprint/item in commit messages
3. Create PR to merge into `develop`
4. After testing in `develop`, create PR to merge into `main`
5. Update this roadmap when items are completed
6. Add tests for new functionality

## Version Milestones

| Version | Target | Key Features |
|---------|--------|--------------|
| v0.1.0 | Phase 1 complete | Core tools, Git integration |
| v0.2.0 | Phase 2 complete | Code intelligence, semantic search |
| v0.3.0 | Phase 3-4 complete | Enhanced UX, security hardening |
| v0.5.0 | Phase 5 complete | Advanced AI features |
| v1.0.0 | Phase 6 complete | Full IDE integration, production ready |
