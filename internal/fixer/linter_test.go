package fixer

import (
	"testing"

	testpkg "github.com/mateus/flow-cli/internal/testing"
)

func TestNewLinter(t *testing.T) {
	cfg := LinterConfig{
		WorkDir:     "/test/project",
		ProjectType: testpkg.ProjectGo,
	}

	linter := NewLinter(cfg)

	if linter == nil {
		t.Fatal("expected linter to be created")
	}
	if linter.workDir != "/test/project" {
		t.Errorf("expected workDir '/test/project', got '%s'", linter.workDir)
	}
	if linter.projectType != testpkg.ProjectGo {
		t.Errorf("expected projectType Go, got %s", linter.projectType)
	}
}

func TestParseGoVetOutput(t *testing.T) {
	linter := &Linter{
		workDir:     "/test",
		projectType: testpkg.ProjectGo,
	}

	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{
			name:     "empty output",
			output:   "",
			expected: 0,
		},
		{
			name:     "single issue",
			output:   "main.go:10:5: undefined: foo",
			expected: 1,
		},
		{
			name: "multiple issues",
			output: `main.go:10:5: undefined: foo
handler.go:25:10: unused variable bar
utils.go:42:1: missing return`,
			expected: 3,
		},
		{
			name:     "non-matching lines",
			output:   "some random text\nwithout proper format",
			expected: 0,
		},
		{
			name: "mixed output",
			output: `# mypackage
main.go:10:5: undefined: foo
vet: some warning`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := linter.parseGoVetOutput(tt.output)
			if len(issues) != tt.expected {
				t.Errorf("expected %d issues, got %d", tt.expected, len(issues))
			}
		})
	}
}

func TestParseGoVetOutputDetails(t *testing.T) {
	linter := &Linter{}

	output := "main.go:10:5: undefined: foo"
	issues := linter.parseGoVetOutput(output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.File != "main.go" {
		t.Errorf("expected file 'main.go', got '%s'", issue.File)
	}
	if issue.Line != 10 {
		t.Errorf("expected line 10, got %d", issue.Line)
	}
	if issue.Column != 5 {
		t.Errorf("expected column 5, got %d", issue.Column)
	}
	if issue.Message != "undefined: foo" {
		t.Errorf("expected message 'undefined: foo', got '%s'", issue.Message)
	}
	if issue.Rule != "go-vet" {
		t.Errorf("expected rule 'go-vet', got '%s'", issue.Rule)
	}
	if issue.Severity != "error" {
		t.Errorf("expected severity 'error', got '%s'", issue.Severity)
	}
}

func TestLintIssueStruct(t *testing.T) {
	issue := LintIssue{
		File:     "test.go",
		Line:     42,
		Column:   10,
		Severity: "warning",
		Message:  "test message",
		Rule:     "test-rule",
		Source:   "var x = 1",
	}

	if issue.File != "test.go" {
		t.Errorf("expected File 'test.go', got '%s'", issue.File)
	}
	if issue.Line != 42 {
		t.Errorf("expected Line 42, got %d", issue.Line)
	}
	if issue.Column != 10 {
		t.Errorf("expected Column 10, got %d", issue.Column)
	}
	if issue.Severity != "warning" {
		t.Errorf("expected Severity 'warning', got '%s'", issue.Severity)
	}
}
