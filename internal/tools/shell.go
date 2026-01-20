package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mateus/flow-cli/internal/sandbox"
)

const (
	// DefaultTimeout is the default command timeout
	DefaultTimeout = 60 * time.Second

	// MaxOutputSize is the maximum output size in bytes
	MaxOutputSize = 100 * 1024 // 100KB
)

// RunCommandTool executes shell commands
type RunCommandTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewRunCommandTool creates a new run command tool
func NewRunCommandTool(permissions *sandbox.Manager, workDir string) *RunCommandTool {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	return &RunCommandTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *RunCommandTool) Name() string {
	return "run_command"
}

func (t *RunCommandTool) Description() string {
	return "Execute a shell command and return its output"
}

func (t *RunCommandTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "command",
			Type:        TypeString,
			Description: "The command to execute",
			Required:    true,
		},
		{
			Name:        "working_dir",
			Type:        TypeString,
			Description: "The working directory for the command (defaults to project directory)",
			Required:    false,
		},
		{
			Name:        "timeout",
			Type:        TypeNumber,
			Description: "Timeout in seconds (default: 60)",
			Required:    false,
			Default:     60,
		},
	}
}

func (t *RunCommandTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpExecuteCmd
}

func (t *RunCommandTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	command, err := RequiredStringArg(args, "command")
	if err != nil {
		return NewErrorResult(err), nil
	}

	workDir := GetStringArg(args, "working_dir", t.workDir)
	timeoutSec := GetIntArg(args, "timeout", 60)

	// Check permission
	op := sandbox.NewOperation(sandbox.OpExecuteCmd, command, fmt.Sprintf("Execute: %s", command)).
		WithDetail("working_dir", workDir)

	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Create timeout context
	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build the command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", "-c", command)
	}

	cmd.Dir = workDir

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	startTime := time.Now()
	runErr := cmd.Run()
	duration := time.Since(startTime)

	// Combine output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Truncate if too long
	if len(output) > MaxOutputSize {
		output = output[:MaxOutputSize] + fmt.Sprintf("\n... (output truncated, showing first %d bytes)", MaxOutputSize)
	}

	// Build result
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			return NewErrorResult(fmt.Errorf("command timed out after %s", timeout)), nil
		} else {
			return NewErrorResult(fmt.Errorf("command failed: %w", runErr)), nil
		}
	}

	data := map[string]any{
		"exit_code": exitCode,
		"duration":  duration.String(),
		"command":   command,
	}

	if exitCode != 0 {
		return &Result{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("command exited with code %d", exitCode),
			Data:    data,
		}, nil
	}

	return NewSuccessResultWithData(output, data), nil
}

// GetShell returns the appropriate shell for the current OS
func GetShell() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C"}
	}
	return "sh", []string{"-c"}
}

// IsCommandSafe performs basic safety checks on a command
func IsCommandSafe(command string) (bool, string) {
	command = strings.ToLower(strings.TrimSpace(command))

	// Check for dangerous patterns
	dangerous := []struct {
		pattern string
		reason  string
	}{
		{"rm -rf /", "attempts to delete root filesystem"},
		{"rm -rf ~", "attempts to delete home directory"},
		{":(){", "fork bomb detected"},
		{"> /dev/", "attempts to write to device files"},
		{"mkfs", "attempts to format filesystem"},
		{"dd if=", "low-level disk operation"},
		{"|sh", "piped shell execution"},
		{"|bash", "piped shell execution"},
		{"curl|", "piped download execution"},
		{"wget|", "piped download execution"},
	}

	for _, d := range dangerous {
		if strings.Contains(command, d.pattern) {
			return false, d.reason
		}
	}

	return true, ""
}
