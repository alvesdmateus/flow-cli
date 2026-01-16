package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// MockGitRunner is a mock implementation of GitRunner for testing
type MockGitRunner struct {
	outputs map[string]string
	errors  map[string]error
	calls   [][]string
}

// NewMockGitRunner creates a new mock git runner
func NewMockGitRunner() *MockGitRunner {
	return &MockGitRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
		calls:   [][]string{},
	}
}

// SetOutput sets the output for a specific command
func (m *MockGitRunner) SetOutput(cmd string, output string) {
	m.outputs[cmd] = output
}

// SetError sets an error for a specific command
func (m *MockGitRunner) SetError(cmd string, err error) {
	m.errors[cmd] = err
}

// Run implements GitRunner interface
func (m *MockGitRunner) Run(ctx context.Context, workDir string, args ...string) (string, error) {
	m.calls = append(m.calls, args)
	cmd := strings.Join(args, " ")

	if err, ok := m.errors[cmd]; ok {
		return "", err
	}

	if output, ok := m.outputs[cmd]; ok {
		return output, nil
	}

	// Check for partial matches (for commands with dynamic parts)
	for key, output := range m.outputs {
		if strings.HasPrefix(cmd, key) {
			return output, nil
		}
	}

	return "", nil
}

// GetCalls returns all recorded calls
func (m *MockGitRunner) GetCalls() [][]string {
	return m.calls
}

// LastCall returns the last call made
func (m *MockGitRunner) LastCall() []string {
	if len(m.calls) == 0 {
		return nil
	}
	return m.calls[len(m.calls)-1]
}

func TestGitStatusTool_Name(t *testing.T) {
	tool := NewGitStatusTool(nil, "/tmp")
	if tool.Name() != "git_status" {
		t.Errorf("expected 'git_status', got '%s'", tool.Name())
	}
}

func TestGitStatusTool_Execute(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		mockOutput string
		mockError  error
		wantClean  bool
		wantErr    bool
	}{
		{
			name: "clean repository",
			args: map[string]any{},
			mockOutput: `On branch main
nothing to commit, working tree clean`,
			wantClean: true,
			wantErr:   false,
		},
		{
			name: "with changes - short format",
			args: map[string]any{"short": true},
			mockOutput: ` M file1.go
?? file2.go
A  file3.go`,
			wantClean: false,
			wantErr:   false,
		},
		{
			name:      "git error",
			args:      map[string]any{},
			mockError: errors.New("not a git repository"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockGitRunner()
			if tt.mockError != nil {
				mock.SetError("status", tt.mockError)
			} else {
				short := GetBoolArg(tt.args, "short", false)
				if short {
					mock.SetOutput("status --short", tt.mockOutput)
				} else {
					mock.SetOutput("status", tt.mockOutput)
				}
			}

			// Set the mock runner
			oldRunner := gitRunner
			SetGitRunner(mock)
			defer func() { gitRunner = oldRunner }()

			tool := NewGitStatusTool(nil, "/tmp")
			result, err := tool.Execute(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result.Success {
					t.Error("expected failure result")
				}
				return
			}

			if !result.Success {
				t.Errorf("expected success, got error: %s", result.Error)
				return
			}

			data, ok := result.Data.(map[string]any)
			if !ok {
				t.Fatal("expected data to be map[string]any")
			}

			clean, _ := data["clean"].(bool)
			if clean != tt.wantClean {
				t.Errorf("expected clean=%v, got %v", tt.wantClean, clean)
			}
		})
	}
}

func TestGitStatusTool_ParseShortFormat(t *testing.T) {
	output := ` M modified.go
A  staged.go
?? untracked.go
MM both.go`

	data := parseGitStatus(output, true)

	staged, _ := data["staged"].([]string)
	unstaged, _ := data["unstaged"].([]string)
	untracked, _ := data["untracked"].([]string)

	if len(staged) != 2 { // "staged.go" and "both.go"
		t.Errorf("expected 2 staged files, got %d: %v", len(staged), staged)
	}

	if len(unstaged) != 2 { // "modified.go" and "both.go"
		t.Errorf("expected 2 unstaged files, got %d: %v", len(unstaged), unstaged)
	}

	if len(untracked) != 1 { // "untracked.go"
		t.Errorf("expected 1 untracked file, got %d: %v", len(untracked), untracked)
	}
}

func TestGitDiffTool_Name(t *testing.T) {
	tool := NewGitDiffTool(nil, "/tmp")
	if tool.Name() != "git_diff" {
		t.Errorf("expected 'git_diff', got '%s'", tool.Name())
	}
}

func TestGitDiffTool_Execute(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		mockOutput string
		wantFiles  int
	}{
		{
			name:       "no changes",
			args:       map[string]any{},
			mockOutput: "",
			wantFiles:  0,
		},
		{
			name: "with changes",
			args: map[string]any{},
			mockOutput: `diff --git a/file.go b/file.go
index 1234567..abcdefg 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+
+import "fmt"
-import "os"`,
			wantFiles: 1,
		},
		{
			name: "staged changes",
			args: map[string]any{"staged": true},
			mockOutput: `diff --git a/staged.go b/staged.go
index 1234567..abcdefg 100644
--- a/staged.go
+++ b/staged.go
@@ -1 +1 @@
-old
+new`,
			wantFiles: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockGitRunner()
			staged := GetBoolArg(tt.args, "staged", false)
			if staged {
				mock.SetOutput("diff --cached", tt.mockOutput)
			} else {
				mock.SetOutput("diff", tt.mockOutput)
			}

			oldRunner := gitRunner
			SetGitRunner(mock)
			defer func() { gitRunner = oldRunner }()

			tool := NewGitDiffTool(nil, "/tmp")
			result, err := tool.Execute(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Success {
				t.Errorf("expected success, got error: %s", result.Error)
				return
			}

			if tt.mockOutput == "" {
				if result.Output != "No changes" {
					t.Errorf("expected 'No changes', got '%s'", result.Output)
				}
				return
			}

			data, ok := result.Data.(map[string]any)
			if !ok {
				t.Fatal("expected data to be map[string]any")
			}

			filesChanged, _ := data["files_changed"].(int)
			if filesChanged != tt.wantFiles {
				t.Errorf("expected %d files changed, got %d", tt.wantFiles, filesChanged)
			}
		})
	}
}

func TestParseDiffStats(t *testing.T) {
	diff := `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
+line1
+line2
-removed
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
+added`

	stats := parseDiffStats(diff)

	if stats["files_changed"] != 2 {
		t.Errorf("expected 2 files changed, got %v", stats["files_changed"])
	}
	if stats["additions"] != 3 {
		t.Errorf("expected 3 additions, got %v", stats["additions"])
	}
	if stats["deletions"] != 1 {
		t.Errorf("expected 1 deletion, got %v", stats["deletions"])
	}
}

func TestGitLogTool_Name(t *testing.T) {
	tool := NewGitLogTool(nil, "/tmp")
	if tool.Name() != "git_log" {
		t.Errorf("expected 'git_log', got '%s'", tool.Name())
	}
}

func TestGitLogTool_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		mockOutput  string
		wantCommits int
	}{
		{
			name: "default log",
			args: map[string]any{},
			mockOutput: `abc1234 Fix bug (John, 2 days ago)
def5678 Add feature (Jane, 1 week ago)
ghi9012 Initial commit (John, 2 weeks ago)`,
			wantCommits: 3,
		},
		{
			name:        "no commits",
			args:        map[string]any{},
			mockOutput:  "",
			wantCommits: 0,
		},
		{
			name: "with limit",
			args: map[string]any{"limit": 5},
			mockOutput: `abc1234 Commit 1
def5678 Commit 2`,
			wantCommits: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockGitRunner()
			mock.SetOutput("log", tt.mockOutput)

			oldRunner := gitRunner
			SetGitRunner(mock)
			defer func() { gitRunner = oldRunner }()

			tool := NewGitLogTool(nil, "/tmp")
			result, err := tool.Execute(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Success {
				t.Errorf("expected success, got error: %s", result.Error)
				return
			}

			if tt.wantCommits == 0 {
				if result.Output != "No commits found" {
					t.Errorf("expected 'No commits found', got '%s'", result.Output)
				}
				return
			}

			data, ok := result.Data.(map[string]any)
			if !ok {
				t.Fatal("expected data to be map[string]any")
			}

			commitCount, _ := data["commit_count"].(int)
			if commitCount != tt.wantCommits {
				t.Errorf("expected %d commits, got %d", tt.wantCommits, commitCount)
			}
		})
	}
}

func TestGitCommitTool_Name(t *testing.T) {
	tool := NewGitCommitTool(nil, "/tmp")
	if tool.Name() != "git_commit" {
		t.Errorf("expected 'git_commit', got '%s'", tool.Name())
	}
}

func TestGitCommitTool_Execute_MissingMessage(t *testing.T) {
	tool := NewGitCommitTool(nil, "/tmp")
	result, err := tool.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure for missing message")
	}

	if !strings.Contains(result.Error, "message") {
		t.Errorf("expected error about missing message, got: %s", result.Error)
	}
}

func TestGitAddTool_Name(t *testing.T) {
	tool := NewGitAddTool(nil, "/tmp")
	if tool.Name() != "git_add" {
		t.Errorf("expected 'git_add', got '%s'", tool.Name())
	}
}

func TestGitAddTool_Execute_MissingFiles(t *testing.T) {
	tool := NewGitAddTool(nil, "/tmp")
	result, err := tool.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure for missing files")
	}

	if !strings.Contains(result.Error, "files") {
		t.Errorf("expected error about missing files, got: %s", result.Error)
	}
}

func TestGitBranchTool_Name(t *testing.T) {
	tool := NewGitBranchTool(nil, "/tmp")
	if tool.Name() != "git_branch" {
		t.Errorf("expected 'git_branch', got '%s'", tool.Name())
	}
}

func TestGitBranchTool_Execute_List(t *testing.T) {
	mock := NewMockGitRunner()
	mock.SetOutput("branch", `* main
  develop
  feature/test`)

	oldRunner := gitRunner
	SetGitRunner(mock)
	defer func() { gitRunner = oldRunner }()

	tool := NewGitBranchTool(nil, "/tmp")
	result, err := tool.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
		return
	}

	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be map[string]any")
	}

	current, _ := data["current"].(string)
	if current != "main" {
		t.Errorf("expected current branch 'main', got '%s'", current)
	}

	branches, _ := data["branches"].([]string)
	if len(branches) != 3 {
		t.Errorf("expected 3 branches, got %d", len(branches))
	}
}

func TestParseBranches(t *testing.T) {
	output := `* main
  develop
  feature/test
  remotes/origin/main`

	data := parseBranches(output)

	current, _ := data["current"].(string)
	if current != "main" {
		t.Errorf("expected current 'main', got '%s'", current)
	}

	branches, _ := data["branches"].([]string)
	if len(branches) != 4 {
		t.Errorf("expected 4 branches, got %d: %v", len(branches), branches)
	}
}

func TestGitCheckoutTool_Name(t *testing.T) {
	tool := NewGitCheckoutTool(nil, "/tmp")
	if tool.Name() != "git_checkout" {
		t.Errorf("expected 'git_checkout', got '%s'", tool.Name())
	}
}

func TestGitCheckoutTool_Execute_MissingArgs(t *testing.T) {
	mock := NewMockGitRunner()
	oldRunner := gitRunner
	SetGitRunner(mock)
	defer func() { gitRunner = oldRunner }()

	tool := NewGitCheckoutTool(nil, "/tmp")
	result, err := tool.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure for missing branch/file")
	}

	if !strings.Contains(result.Error, "must specify") {
		t.Errorf("expected error about missing args, got: %s", result.Error)
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a long message", 10, "this is..."},
		{"exact", 5, "exact"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		got := truncateMessage(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateMessage(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestExtractCommitHash(t *testing.T) {
	tests := []struct {
		output string
		want   string
	}{
		{"[main abc1234] Fix bug\n 1 file changed", "abc1234"},
		{"[feature/test def5678] Add feature", "def5678"},
		{"some other output", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractCommitHash(tt.output)
		if got != tt.want {
			t.Errorf("extractCommitHash(%q) = %q, want %q", tt.output, got, tt.want)
		}
	}
}

func TestGetRepoInfo(t *testing.T) {
	// This test uses the actual git command on the current directory
	// It should work since we're in a git repository
	info := GetRepoInfo(".")

	isRepo, _ := info["is_repo"].(bool)
	if !isRepo {
		t.Skip("not running in a git repository")
	}

	if _, ok := info["branch"]; !ok {
		t.Error("expected branch in repo info")
	}

	if _, ok := info["root"]; !ok {
		t.Error("expected root in repo info")
	}
}
