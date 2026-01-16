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

### Sprint 1: Core Completeness
- [ ] Implement `fetch_url` tool (HTML parsing, markdown extraction)
- [ ] Add `edit_file` tool (patch-based edits instead of full rewrites)
- [ ] Add `grep_search` tool (regex search across codebase)
- [ ] Token counting and cost estimation display
- [ ] Context window management with smart truncation

### Sprint 2: Git Integration
- [ ] `git_status`, `git_diff`, `git_commit` tools
- [ ] `git_log` with commit analysis
- [ ] PR/MR description generation
- [ ] Branch management helpers
- [ ] Conflict detection and resolution suggestions

---

## Phase 2: Code Intelligence (Sprints 3-4)
**Goal:** Semantic code understanding

### Sprint 3: Code Analysis
- [ ] Tree-sitter integration for AST parsing (Go, JS, Python, Rust)
- [ ] Symbol extraction (functions, classes, types)
- [ ] `find_definition`, `find_references` tools
- [ ] Code outline/structure tool
- [ ] Import/dependency analysis

### Sprint 4: Codebase Indexing
- [ ] Local embedding generation (with Ollama)
- [ ] Vector store for semantic search (SQLite + vector extension)
- [ ] `semantic_search` tool for natural language code queries
- [ ] Automatic re-indexing on file changes
- [ ] `.vibeignore` for excluding files from indexing

---

## Phase 3: Developer Experience (Sprints 5-6)
**Goal:** Productivity features

### Sprint 5: Enhanced Workflows
- [ ] `vibe init` - Project scaffolding with templates
- [ ] `vibe review` - Code review mode (diff analysis)
- [ ] `vibe test` - Test generation and execution
- [ ] `vibe fix` - Auto-fix linter errors
- [ ] `vibe explain` - Explain code/errors in detail

### Sprint 6: Terminal UX
- [ ] Multi-file edit preview with unified diff
- [ ] Undo/redo for file changes
- [ ] Keyboard shortcuts (Ctrl+C cancel, Ctrl+R retry)
- [ ] Progress indicators for long operations
- [ ] Command history with fuzzy search
- [ ] Markdown rendering in terminal

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

| Feature | Impact | Effort | Priority |
|---------|--------|--------|----------|
| fetch_url completion | High | Low | **P0** |
| Git integration | High | Medium | **P0** |
| Token counting | Medium | Low | **P0** |
| edit_file (patches) | High | Medium | **P1** |
| grep_search | High | Low | **P1** |
| Tree-sitter AST | High | High | **P1** |
| Secrets detection | High | Medium | **P1** |
| Semantic search | High | High | P2 |
| LSP server mode | High | High | P2 |
| Test generation | Medium | Medium | P2 |

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

## Contributing

When working on roadmap items:
1. Create a feature branch from `main`
2. Reference the sprint/item in commit messages
3. Update this roadmap when items are completed
4. Add tests for new functionality

## Version Milestones

| Version | Target | Key Features |
|---------|--------|--------------|
| v0.1.0 | Phase 1 complete | Core tools, Git integration |
| v0.2.0 | Phase 2 complete | Code intelligence, semantic search |
| v0.3.0 | Phase 3-4 complete | Enhanced UX, security hardening |
| v0.5.0 | Phase 5 complete | Advanced AI features |
| v1.0.0 | Phase 6 complete | Full IDE integration, production ready |
