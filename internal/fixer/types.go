// Package fixer provides linting and auto-fix functionality.
package fixer

import (
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/testing"
)

// LintIssue represents a single linter issue.
type LintIssue struct {
	File     string
	Line     int
	Column   int
	Severity string // error, warning, info
	Message  string
	Rule     string
	Source   string // the source code line
}

// Fix represents a fix for a linter issue.
type Fix struct {
	Issue       *LintIssue
	File        string
	Line        int
	Description string
	OldCode     string
	NewCode     string
	Diff        string
}

// LinterConfig contains configuration for the linter.
type LinterConfig struct {
	WorkDir     string
	ProjectType testing.ProjectType
}

// FixerConfig contains configuration for the fixer.
type FixerConfig struct {
	LLMClient   llm.Client
	Model       string
	ProjectType testing.ProjectType
	WorkDir     string
}
