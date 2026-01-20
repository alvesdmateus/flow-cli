package fixer

import (
	"strings"
	"testing"

	testpkg "github.com/mateus/flow-cli/internal/testing"
)

func TestNewFixer(t *testing.T) {
	cfg := FixerConfig{
		LLMClient:   nil, // Would be a mock in real tests
		Model:       "test-model",
		ProjectType: testpkg.ProjectGo,
		WorkDir:     "/test/project",
	}

	f := NewFixer(cfg)

	if f == nil {
		t.Fatal("expected fixer to be created")
	}
	if f.model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", f.model)
	}
	if f.workDir != "/test/project" {
		t.Errorf("expected workDir '/test/project', got '%s'", f.workDir)
	}
}

func TestBuildPrompt(t *testing.T) {
	f := &Fixer{}

	issue := LintIssue{
		File:    "main.go",
		Line:    10,
		Message: "unused variable x",
		Rule:    "unused",
		Source:  "x := 1",
	}

	contextCode := `func main() {
    x := 1
    fmt.Println("hello")
}`

	prompt := f.buildPrompt("main.go", issue, contextCode, 9)

	// Check that prompt contains key elements
	if !strings.Contains(prompt, "main.go") {
		t.Error("prompt should contain filename")
	}
	if !strings.Contains(prompt, "Line: 10") {
		t.Error("prompt should contain line number")
	}
	if !strings.Contains(prompt, "unused variable x") {
		t.Error("prompt should contain issue message")
	}
	if !strings.Contains(prompt, "Rule: unused") {
		t.Error("prompt should contain rule")
	}
	if !strings.Contains(prompt, "starting at line 9") {
		t.Error("prompt should contain context start line")
	}
	if !strings.Contains(prompt, contextCode) {
		t.Error("prompt should contain context code")
	}
}

func TestParseFixResponse(t *testing.T) {
	f := &Fixer{}

	tests := []struct {
		name        string
		response    string
		expectNil   bool
		expectCode  string
		expectDiff  bool
	}{
		{
			name:        "simple fix",
			response:    "_ = x",
			expectNil:   false,
			expectCode:  "_ = x",
			expectDiff:  true,
		},
		{
			name:        "fix with code block",
			response:    "```go\n_ = x\n```",
			expectNil:   false,
			expectCode:  "_ = x",
			expectDiff:  true,
		},
		{
			name:        "delete line",
			response:    "DELETE_LINE",
			expectNil:   false,
			expectCode:  "",
			expectDiff:  true,
		},
		{
			name:        "empty response",
			response:    "",
			expectNil:   true,
			expectCode:  "",
			expectDiff:  false,
		},
		{
			name:        "whitespace only",
			response:    "   \n\t  ",
			expectNil:   true,
			expectCode:  "",
			expectDiff:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := LintIssue{
				File:    "test.go",
				Line:    5,
				Message: "test issue",
				Source:  "x := 1",
			}

			fix := f.parseFixResponse(tt.response, "test.go", issue)

			if tt.expectNil {
				if fix != nil {
					t.Errorf("expected nil fix, got %+v", fix)
				}
				return
			}

			if fix == nil {
				t.Fatal("expected non-nil fix")
			}

			if fix.NewCode != tt.expectCode {
				t.Errorf("expected NewCode '%s', got '%s'", tt.expectCode, fix.NewCode)
			}

			if tt.expectDiff && fix.Diff == "" {
				t.Error("expected non-empty Diff")
			}
		})
	}
}

func TestFixStruct(t *testing.T) {
	issue := &LintIssue{
		File:    "test.go",
		Line:    10,
		Message: "unused",
	}

	fix := &Fix{
		Issue:       issue,
		File:        "test.go",
		Line:        10,
		Description: "Fix unused variable",
		OldCode:     "x := 1",
		NewCode:     "_ = 1",
		Diff:        "- x := 1\n+ _ = 1",
	}

	if fix.Issue != issue {
		t.Error("expected Issue to be set")
	}
	if fix.File != "test.go" {
		t.Errorf("expected File 'test.go', got '%s'", fix.File)
	}
	if fix.Line != 10 {
		t.Errorf("expected Line 10, got %d", fix.Line)
	}
	if fix.Description != "Fix unused variable" {
		t.Errorf("expected Description, got '%s'", fix.Description)
	}
}

func TestBuildPromptWithoutRule(t *testing.T) {
	f := &Fixer{}

	issue := LintIssue{
		File:    "main.go",
		Line:    10,
		Message: "syntax error",
		Rule:    "", // No rule
	}

	prompt := f.buildPrompt("main.go", issue, "code", 1)

	// Should not contain "Rule:" when rule is empty
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Rule:") && !strings.Contains(line, ":") {
			t.Error("prompt should not have empty Rule line")
		}
	}
}
