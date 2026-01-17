# vibe-cli Development Roadmap

> Last updated: 2026-01-16
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
- [x] Vector store for semantic search (SQLite)
- [x] `semantic_search` tool for natural language code queries
- [x] `index_status` tool for checking index statistics
- [x] `reindex_file` tool for manual re-indexing
- [x] Automatic re-indexing on file changes (file watcher)
- [x] `.flowignore` for excluding files from indexing
- [x] CLI commands: `flow index build`, `flow index status`, `flow index clear`, `flow index search`
- [x] Configuration: `indexing.enabled`, `indexing.embedding_model`, `indexing.auto_index`, `indexing.watch_changes`

---

## Phase 3: Developer Experience (Sprints 5-6)
**Goal:** Productivity features

### Sprint 5: Enhanced Workflows ✅
- [x] `flow init` - Project scaffolding with templates (Go, Python, Node.js, Rust)
- [x] `flow review` - Code review mode (diff analysis, PR review via gh CLI)
- [x] `flow test` - Test generation and execution with coverage
- [x] `flow fix` - Auto-fix linter errors (golangci-lint, ruff, eslint, clippy)
- [x] `flow explain` - Explain code/errors in detail

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

### Sprint 10: Code Generation ✅
- [x] Test generation from implementation
- [x] Documentation generation (JSDoc, GoDoc, etc.)
- [x] Boilerplate generation (CRUD, API endpoints)
- [x] Refactoring suggestions (extract function, rename)
- [x] Bug fix suggestions from error messages

---

## Phase 6: Integrations (Sprints 11-12)
**Goal:** Ecosystem connectivity

### Sprint 11: External Tools ✅
- [x] Docker tools (`docker_build`, `docker_run`, `docker_logs`)
- [x] Database tools (`query_db` for SQLite/Postgres)
- [x] HTTP client tool (`http_request`)
- [x] Package manager tools (npm, pip, cargo info)
- [x] CI/CD status checking (GitHub Actions)

### Sprint 12: IDE & Editor ✅
- [x] LSP server mode (integrate with any editor)
- [x] VS Code extension
- [x] Neovim plugin
- [x] JetBrains plugin (basic)
- [x] `.vibe/config.yaml` project-level settings

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
| Test generation | Medium | Medium | P2 | ✅ Done |
| LSP server mode | High | High | P2 | ✅ Done |
| Test generation | Medium | Medium | P2 | Pending |
| External tools | High | High | P2 | ✅ Done |
| IDE plugins | High | High | P2 | ✅ Done |

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

### Check-in: 2026-01-17 (Sprint 10 Complete)

#### Completed
- **Sprint 10: Code Generation** ✅
  - `internal/codegen/testgen.go` - Test generation from implementation
    - Multi-language support (Go, Python, JavaScript, TypeScript)
    - Table-driven test generation for Go
    - Language-specific analyzers for function extraction
  - `internal/codegen/docgen.go` - Documentation generation
    - GoDoc, JSDoc, PyDoc, RustDoc styles
    - Function and class documentation
    - Auto-insertion of docs for undocumented functions
  - `internal/codegen/boilerplate.go` - Boilerplate generation
    - CRUD operations for entities
    - Model, Repository, Service, Handler patterns
    - Multi-language support (Go, Python, TypeScript)
  - `internal/codegen/refactor.go` - Refactoring suggestions
    - Long function detection
    - Duplicate code detection
    - Magic number detection
    - Deep nesting detection
    - Large class detection
    - Extract function, rename, extract variable operations
  - `internal/codegen/bugfix.go` - Bug fix suggestions
    - Error message pattern matching
    - Language-specific error patterns (Go, Python, JS, TS)
    - Generic error patterns (timeout, permission denied, etc.)
    - Compiler output parsing
  - `internal/codegen/codegen_test.go` - 44 tests

#### In Progress
- None

#### Next Sprint Focus
- **Sprint 11: External Tools** (Phase 6)
  - Docker tools
  - Database tools
  - HTTP client tool

#### Notes
- Phase 5 (Advanced AI) complete
- Code generation package provides comprehensive tooling
- Ready for Phase 6: Integrations
- **Sprint 4: Codebase Indexing** (pending)
- **Security hardening** (ongoing)

#### Notes
- Phase 6 (Integrations) complete
- All IDE/editor plugins provide basic functionality
- LSP server enables integration with any LSP-compatible editor
- Project config allows per-project customization

---

### Check-in: 2026-01-20 (Sprint 4 Complete)

#### Completed
- **Sprint 4: Codebase Indexing** ✅
  - `internal/indexing/embeddings.go` - Ollama embedding client
  - `internal/indexing/store.go` - SQLite vector store with cosine similarity
  - `internal/indexing/indexer.go` - Orchestrates indexing with file watching
  - `internal/tools/semantic.go` - Semantic search tools
  - `cmd/index.go` - CLI commands for index management
  - `internal/config/config.go` - IndexingConfig struct and helpers
  - Wired up indexer in `cmd/chat.go` and `cmd/arch.go`
  - Configuration options: enabled, embedding_model, auto_index, watch_changes

#### In Progress
- None

#### Next Sprint Focus
- **Sprint 5: Enhanced Workflows** (Phase 3)
  - `flow init` - Project scaffolding
  - `flow review` - Code review mode
  - `flow test` - Test generation

#### Notes
- Phase 2 (Code Intelligence) complete with semantic search
- Embedding model defaults to `nomic-embed-text`
- Feature is opt-in via `indexing.enabled` config
- Automatic background indexing and file watching available

---

### Check-in: 2026-01-20 (Sprint 5 Complete)

#### Completed
- **Sprint 5: Enhanced Workflows** ✅
  - `cmd/init.go` - Project scaffolding with templates (Go, Python, Node.js, Rust, Empty)
  - `cmd/review.go` - AI-powered code review (git diff, staged changes, PR review)
  - `cmd/test.go` - Test generation and execution with coverage
  - `cmd/fix.go` - Auto-fix linter errors (golangci-lint, ruff, eslint, clippy)
  - `cmd/explain.go` - Explain code, functions, or errors
  - `internal/templates/` - Project template scaffolding system
  - `internal/review/` - Code review diff parsing and analysis
  - `internal/testing/` - Test runner, generator, and analyzer
  - `internal/fixer/` - Linter integration and fix generation
  - `internal/explainer/` - Code and error explanation

#### In Progress
- None

#### Next Sprint Focus
- **Sprint 6: Terminal UX** (Phase 3)
  - Multi-file edit preview
  - Undo/redo for file changes
  - Keyboard shortcuts

#### Notes
- Phase 3 (Developer Experience) in progress
- All Sprint 5 commands use LLM for intelligent assistance
- Templates support multiple languages with best practices
- Code review supports PR analysis via GitHub CLI

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
