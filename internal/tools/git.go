package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mateus/vibe-cli/internal/sandbox"
)

// GitRunner interface for executing git commands (allows mocking in tests)
type GitRunner interface {
	Run(ctx context.Context, workDir string, args ...string) (string, error)
}

// DefaultGitRunner executes real git commands
type DefaultGitRunner struct{}

// Run executes a git command and returns the output
func (r *DefaultGitRunner) Run(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return "", err
	}

	return stdout.String(), nil
}

// gitRunner is the default runner, can be replaced for testing
var gitRunner GitRunner = &DefaultGitRunner{}

// SetGitRunner sets the git runner (for testing)
func SetGitRunner(runner GitRunner) {
	gitRunner = runner
}

// ResetGitRunner resets to the default git runner
func ResetGitRunner() {
	gitRunner = &DefaultGitRunner{}
}

// --- GitStatusTool ---

// GitStatusTool shows the working tree status
type GitStatusTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGitStatusTool creates a new git status tool
func NewGitStatusTool(permissions *sandbox.Manager, workDir string) *GitStatusTool {
	return &GitStatusTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GitStatusTool) Name() string {
	return "git_status"
}

func (t *GitStatusTool) Description() string {
	return "Show the working tree status (staged, unstaged, untracked files)"
}

func (t *GitStatusTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "short",
			Type:        TypeBoolean,
			Description: "Show short format output",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "path",
			Type:        TypeString,
			Description: "Limit status to specific path",
			Required:    false,
		},
	}
}

func (t *GitStatusTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile // Read-only operation
}

func (t *GitStatusTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	short := GetBoolArg(args, "short", false)
	path := GetStringArg(args, "path", "")

	gitArgs := []string{"status"}
	if short {
		gitArgs = append(gitArgs, "--short")
	}
	if path != "" {
		gitArgs = append(gitArgs, "--", path)
	}

	output, err := gitRunner.Run(ctx, t.workDir, gitArgs...)
	if err != nil {
		return NewErrorResult(fmt.Errorf("git status failed: %w", err)), nil
	}

	// Parse status for structured data
	data := parseGitStatus(output, short)

	return NewSuccessResultWithData(strings.TrimSpace(output), data), nil
}

// parseGitStatus extracts structured info from git status output
func parseGitStatus(output string, short bool) map[string]any {
	lines := strings.Split(output, "\n")
	data := map[string]any{
		"clean": strings.Contains(output, "nothing to commit"),
	}

	if short {
		staged := []string{}
		unstaged := []string{}
		untracked := []string{}

		for _, line := range lines {
			if len(line) < 3 {
				continue
			}
			status := line[:2]
			file := strings.TrimSpace(line[2:])

			if status[0] != ' ' && status[0] != '?' {
				staged = append(staged, file)
			}
			if status[1] != ' ' && status[1] != '?' {
				unstaged = append(unstaged, file)
			}
			if status == "??" {
				untracked = append(untracked, file)
			}
		}

		data["staged"] = staged
		data["unstaged"] = unstaged
		data["untracked"] = untracked
	}

	return data
}

// --- GitDiffTool ---

// GitDiffTool shows changes between commits, working tree, etc.
type GitDiffTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGitDiffTool creates a new git diff tool
func NewGitDiffTool(permissions *sandbox.Manager, workDir string) *GitDiffTool {
	return &GitDiffTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GitDiffTool) Name() string {
	return "git_diff"
}

func (t *GitDiffTool) Description() string {
	return "Show changes between commits, commit and working tree, etc."
}

func (t *GitDiffTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "staged",
			Type:        TypeBoolean,
			Description: "Show staged changes (--cached)",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "file",
			Type:        TypeString,
			Description: "Show diff for specific file",
			Required:    false,
		},
		{
			Name:        "commit",
			Type:        TypeString,
			Description: "Compare with specific commit (e.g., HEAD~1, abc123)",
			Required:    false,
		},
		{
			Name:        "stat",
			Type:        TypeBoolean,
			Description: "Show diffstat instead of full diff",
			Required:    false,
			Default:     false,
		},
	}
}

func (t *GitDiffTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile // Read-only operation
}

func (t *GitDiffTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	staged := GetBoolArg(args, "staged", false)
	file := GetStringArg(args, "file", "")
	commit := GetStringArg(args, "commit", "")
	stat := GetBoolArg(args, "stat", false)

	gitArgs := []string{"diff"}

	if staged {
		gitArgs = append(gitArgs, "--cached")
	}
	if stat {
		gitArgs = append(gitArgs, "--stat")
	}
	if commit != "" {
		gitArgs = append(gitArgs, commit)
	}
	if file != "" {
		gitArgs = append(gitArgs, "--", file)
	}

	output, err := gitRunner.Run(ctx, t.workDir, gitArgs...)
	if err != nil {
		return NewErrorResult(fmt.Errorf("git diff failed: %w", err)), nil
	}

	if output == "" {
		return NewSuccessResult("No changes"), nil
	}

	// Count files changed and lines for data
	data := parseDiffStats(output)

	return NewSuccessResultWithData(strings.TrimSpace(output), data), nil
}

// parseDiffStats extracts statistics from diff output
func parseDiffStats(output string) map[string]any {
	lines := strings.Split(output, "\n")
	filesChanged := 0
	additions := 0
	deletions := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			filesChanged++
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}

	return map[string]any{
		"files_changed": filesChanged,
		"additions":     additions,
		"deletions":     deletions,
	}
}

// --- GitLogTool ---

// GitLogTool shows commit logs
type GitLogTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGitLogTool creates a new git log tool
func NewGitLogTool(permissions *sandbox.Manager, workDir string) *GitLogTool {
	return &GitLogTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GitLogTool) Name() string {
	return "git_log"
}

func (t *GitLogTool) Description() string {
	return "Show commit logs with optional filtering"
}

func (t *GitLogTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "limit",
			Type:        TypeNumber,
			Description: "Maximum number of commits to show (default: 10)",
			Required:    false,
			Default:     10,
		},
		{
			Name:        "oneline",
			Type:        TypeBoolean,
			Description: "Show each commit on a single line",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "file",
			Type:        TypeString,
			Description: "Show commits that modified this file",
			Required:    false,
		},
		{
			Name:        "author",
			Type:        TypeString,
			Description: "Filter commits by author",
			Required:    false,
		},
		{
			Name:        "since",
			Type:        TypeString,
			Description: "Show commits since date (e.g., '2 weeks ago', '2024-01-01')",
			Required:    false,
		},
	}
}

func (t *GitLogTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile // Read-only operation
}

func (t *GitLogTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	limit := GetIntArg(args, "limit", 10)
	oneline := GetBoolArg(args, "oneline", false)
	file := GetStringArg(args, "file", "")
	author := GetStringArg(args, "author", "")
	since := GetStringArg(args, "since", "")

	gitArgs := []string{"log", fmt.Sprintf("-n%d", limit)}

	if oneline {
		gitArgs = append(gitArgs, "--oneline")
	} else {
		gitArgs = append(gitArgs, "--pretty=format:%h %s (%an, %ar)")
	}

	if author != "" {
		gitArgs = append(gitArgs, fmt.Sprintf("--author=%s", author))
	}
	if since != "" {
		gitArgs = append(gitArgs, fmt.Sprintf("--since=%s", since))
	}
	if file != "" {
		gitArgs = append(gitArgs, "--", file)
	}

	output, err := gitRunner.Run(ctx, t.workDir, gitArgs...)
	if err != nil {
		return NewErrorResult(fmt.Errorf("git log failed: %w", err)), nil
	}

	if output == "" {
		return NewSuccessResult("No commits found"), nil
	}

	// Count commits
	commits := strings.Split(strings.TrimSpace(output), "\n")
	data := map[string]any{
		"commit_count": len(commits),
	}

	return NewSuccessResultWithData(strings.TrimSpace(output), data), nil
}

// --- GitCommitTool ---

// GitCommitTool creates a new commit
type GitCommitTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGitCommitTool creates a new git commit tool
func NewGitCommitTool(permissions *sandbox.Manager, workDir string) *GitCommitTool {
	return &GitCommitTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GitCommitTool) Name() string {
	return "git_commit"
}

func (t *GitCommitTool) Description() string {
	return "Create a new commit with staged changes"
}

func (t *GitCommitTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "message",
			Type:        TypeString,
			Description: "Commit message",
			Required:    true,
		},
		{
			Name:        "all",
			Type:        TypeBoolean,
			Description: "Automatically stage all modified files (-a)",
			Required:    false,
			Default:     false,
		},
	}
}

func (t *GitCommitTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWriteFile // Write operation (modifies repository)
}

func (t *GitCommitTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	message, err := RequiredStringArg(args, "message")
	if err != nil {
		return NewErrorResult(err), nil
	}

	all := GetBoolArg(args, "all", false)

	// Check permission for git commit
	op := sandbox.NewOperation(sandbox.OpWriteFile, t.workDir, fmt.Sprintf("Git commit: %s", truncateMessage(message, 50)))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	gitArgs := []string{"commit"}
	if all {
		gitArgs = append(gitArgs, "-a")
	}
	gitArgs = append(gitArgs, "-m", message)

	output, err := gitRunner.Run(ctx, t.workDir, gitArgs...)
	if err != nil {
		return NewErrorResult(fmt.Errorf("git commit failed: %w", err)), nil
	}

	// Extract commit hash from output
	commitHash := extractCommitHash(output)
	data := map[string]any{
		"commit": commitHash,
	}

	return NewSuccessResultWithData(strings.TrimSpace(output), data), nil
}

// truncateMessage truncates a message to a maximum length
func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-3] + "..."
}

// extractCommitHash extracts the commit hash from git commit output
func extractCommitHash(output string) string {
	// Output format: [branch hash] message
	// e.g., [main abc1234] Fix bug
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "[") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Remove the trailing ]
				hash := strings.TrimSuffix(parts[1], "]")
				return hash
			}
		}
	}
	return ""
}

// --- GitAddTool ---

// GitAddTool stages files for commit
type GitAddTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGitAddTool creates a new git add tool
func NewGitAddTool(permissions *sandbox.Manager, workDir string) *GitAddTool {
	return &GitAddTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GitAddTool) Name() string {
	return "git_add"
}

func (t *GitAddTool) Description() string {
	return "Stage files for commit"
}

func (t *GitAddTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "files",
			Type:        TypeString,
			Description: "Files to stage (space-separated, or '.' for all)",
			Required:    true,
		},
	}
}

func (t *GitAddTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWriteFile // Modifies staging area
}

func (t *GitAddTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	files, err := RequiredStringArg(args, "files")
	if err != nil {
		return NewErrorResult(err), nil
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpWriteFile, t.workDir, fmt.Sprintf("Git add: %s", files))
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Split files and add them
	fileList := strings.Fields(files)
	gitArgs := append([]string{"add"}, fileList...)

	output, err := gitRunner.Run(ctx, t.workDir, gitArgs...)
	if err != nil {
		return NewErrorResult(fmt.Errorf("git add failed: %w", err)), nil
	}

	// Get status of staged files
	statusOutput, _ := gitRunner.Run(ctx, t.workDir, "status", "--short")

	data := map[string]any{
		"files": fileList,
	}

	if output == "" {
		return NewSuccessResultWithData(fmt.Sprintf("Staged: %s\n\n%s", files, strings.TrimSpace(statusOutput)), data), nil
	}

	return NewSuccessResultWithData(strings.TrimSpace(output), data), nil
}

// --- GitBranchTool ---

// GitBranchTool manages branches
type GitBranchTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGitBranchTool creates a new git branch tool
func NewGitBranchTool(permissions *sandbox.Manager, workDir string) *GitBranchTool {
	return &GitBranchTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GitBranchTool) Name() string {
	return "git_branch"
}

func (t *GitBranchTool) Description() string {
	return "List, create, or delete branches"
}

func (t *GitBranchTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "name",
			Type:        TypeString,
			Description: "Branch name (for create/delete)",
			Required:    false,
		},
		{
			Name:        "create",
			Type:        TypeBoolean,
			Description: "Create a new branch",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "delete",
			Type:        TypeBoolean,
			Description: "Delete the branch",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "all",
			Type:        TypeBoolean,
			Description: "List all branches including remotes",
			Required:    false,
			Default:     false,
		},
	}
}

func (t *GitBranchTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpReadFile // Default to read, will escalate for create/delete
}

func (t *GitBranchTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	name := GetStringArg(args, "name", "")
	create := GetBoolArg(args, "create", false)
	del := GetBoolArg(args, "delete", false)
	all := GetBoolArg(args, "all", false)

	var gitArgs []string

	if create && name != "" {
		// Check permission for creating branch
		op := sandbox.NewOperation(sandbox.OpWriteFile, t.workDir, fmt.Sprintf("Create branch: %s", name))
		if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
			return NewErrorResult(err), nil
		}
		gitArgs = []string{"branch", name}
	} else if del && name != "" {
		// Check permission for deleting branch
		op := sandbox.NewOperation(sandbox.OpWriteFile, t.workDir, fmt.Sprintf("Delete branch: %s", name))
		if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
			return NewErrorResult(err), nil
		}
		gitArgs = []string{"branch", "-d", name}
	} else {
		// List branches
		gitArgs = []string{"branch"}
		if all {
			gitArgs = append(gitArgs, "-a")
		}
	}

	output, err := gitRunner.Run(ctx, t.workDir, gitArgs...)
	if err != nil {
		return NewErrorResult(fmt.Errorf("git branch failed: %w", err)), nil
	}

	// Parse branches for data
	data := parseBranches(output)

	return NewSuccessResultWithData(strings.TrimSpace(output), data), nil
}

// parseBranches extracts branch info from output
func parseBranches(output string) map[string]any {
	lines := strings.Split(output, "\n")
	branches := []string{}
	currentBranch := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "* ") {
			currentBranch = strings.TrimPrefix(line, "* ")
			branches = append(branches, currentBranch)
		} else {
			branches = append(branches, line)
		}
	}

	return map[string]any{
		"current":  currentBranch,
		"branches": branches,
	}
}

// --- GitCheckoutTool ---

// GitCheckoutTool switches branches or restores files
type GitCheckoutTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewGitCheckoutTool creates a new git checkout tool
func NewGitCheckoutTool(permissions *sandbox.Manager, workDir string) *GitCheckoutTool {
	return &GitCheckoutTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *GitCheckoutTool) Name() string {
	return "git_checkout"
}

func (t *GitCheckoutTool) Description() string {
	return "Switch branches or restore working tree files"
}

func (t *GitCheckoutTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "branch",
			Type:        TypeString,
			Description: "Branch name to switch to",
			Required:    false,
		},
		{
			Name:        "create",
			Type:        TypeBoolean,
			Description: "Create and switch to new branch (-b)",
			Required:    false,
			Default:     false,
		},
		{
			Name:        "file",
			Type:        TypeString,
			Description: "File to restore from HEAD",
			Required:    false,
		},
	}
}

func (t *GitCheckoutTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpWriteFile // Can modify working tree
}

func (t *GitCheckoutTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	branch := GetStringArg(args, "branch", "")
	create := GetBoolArg(args, "create", false)
	file := GetStringArg(args, "file", "")

	if branch == "" && file == "" {
		return NewErrorResult(fmt.Errorf("must specify either branch or file")), nil
	}

	var gitArgs []string
	var description string

	if file != "" {
		// Restore file
		description = fmt.Sprintf("Restore file: %s", file)
		gitArgs = []string{"checkout", "--", file}
	} else if create {
		// Create and switch to new branch
		description = fmt.Sprintf("Create and switch to branch: %s", branch)
		gitArgs = []string{"checkout", "-b", branch}
	} else {
		// Switch branch
		description = fmt.Sprintf("Switch to branch: %s", branch)
		gitArgs = []string{"checkout", branch}
	}

	// Check permission
	op := sandbox.NewOperation(sandbox.OpWriteFile, t.workDir, description)
	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	output, err := gitRunner.Run(ctx, t.workDir, gitArgs...)
	if err != nil {
		return NewErrorResult(fmt.Errorf("git checkout failed: %w", err)), nil
	}

	if output == "" {
		if file != "" {
			output = fmt.Sprintf("Restored: %s", file)
		} else {
			output = fmt.Sprintf("Switched to branch: %s", branch)
		}
	}

	return NewSuccessResult(strings.TrimSpace(output)), nil
}

// --- Helper: Check if in git repo ---

// IsGitRepository checks if the given directory is inside a git repository
func IsGitRepository(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// GetGitRoot returns the root directory of the git repository
func GetGitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetRepoInfo returns information about the git repository
func GetRepoInfo(dir string) map[string]any {
	info := map[string]any{
		"is_repo": false,
	}

	if !IsGitRepository(dir) {
		return info
	}

	info["is_repo"] = true

	if root, err := GetGitRoot(dir); err == nil {
		info["root"] = root
		info["name"] = filepath.Base(root)
	}

	if branch, err := GetCurrentBranch(dir); err == nil {
		info["branch"] = branch
	}

	return info
}
