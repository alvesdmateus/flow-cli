package tools

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mateus/vibe-cli/internal/sandbox"
)

// CheckPortTool checks if a port is in use
type CheckPortTool struct {
	permissions *sandbox.Manager
}

// NewCheckPortTool creates a new check port tool
func NewCheckPortTool(permissions *sandbox.Manager) *CheckPortTool {
	return &CheckPortTool{permissions: permissions}
}

func (t *CheckPortTool) Name() string {
	return "check_port"
}

func (t *CheckPortTool) Description() string {
	return "Check if a TCP port is in use on localhost"
}

func (t *CheckPortTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "port",
			Type:        TypeNumber,
			Description: "The port number to check",
			Required:    true,
		},
		{
			Name:        "host",
			Type:        TypeString,
			Description: "The host to check (default: localhost)",
			Required:    false,
			Default:     "localhost",
		},
	}
}

func (t *CheckPortTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpNetworkAccess
}

func (t *CheckPortTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	port := GetIntArg(args, "port", 0)
	if port <= 0 || port > 65535 {
		return NewErrorResult(fmt.Errorf("invalid port number: %d", port)), nil
	}

	host := GetStringArg(args, "host", "localhost")

	// This is a low-risk operation, no permission needed for localhost
	address := net.JoinHostPort(host, strconv.Itoa(port))

	// Try to connect to the port
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		// Port is not in use (connection refused)
		return NewSuccessResultWithData(
			fmt.Sprintf("Port %d is available (not in use)", port),
			map[string]any{
				"port":      port,
				"host":      host,
				"available": true,
			},
		), nil
	}

	conn.Close()

	// Port is in use
	return NewSuccessResultWithData(
		fmt.Sprintf("Port %d is in use", port),
		map[string]any{
			"port":      port,
			"host":      host,
			"available": false,
		},
	), nil
}

// KillProcessTool kills a process by PID or port
type KillProcessTool struct {
	permissions *sandbox.Manager
}

// NewKillProcessTool creates a new kill process tool
func NewKillProcessTool(permissions *sandbox.Manager) *KillProcessTool {
	return &KillProcessTool{permissions: permissions}
}

func (t *KillProcessTool) Name() string {
	return "kill_process"
}

func (t *KillProcessTool) Description() string {
	return "Kill a process by PID or by port number"
}

func (t *KillProcessTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "pid",
			Type:        TypeNumber,
			Description: "The process ID to kill",
			Required:    false,
		},
		{
			Name:        "port",
			Type:        TypeNumber,
			Description: "Kill the process using this port",
			Required:    false,
		},
		{
			Name:        "force",
			Type:        TypeBoolean,
			Description: "Force kill (SIGKILL instead of SIGTERM)",
			Required:    false,
			Default:     false,
		},
	}
}

func (t *KillProcessTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpProcessKill
}

func (t *KillProcessTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	pid := GetIntArg(args, "pid", 0)
	port := GetIntArg(args, "port", 0)
	force := GetBoolArg(args, "force", false)

	if pid <= 0 && port <= 0 {
		return NewErrorResult(fmt.Errorf("must specify either pid or port")), nil
	}

	var targetPID int
	var description string

	if pid > 0 {
		targetPID = pid
		description = fmt.Sprintf("Kill process with PID %d", pid)
	} else {
		// Find PID by port
		foundPID, err := findPIDByPort(port)
		if err != nil {
			return NewErrorResult(fmt.Errorf("failed to find process on port %d: %w", port, err)), nil
		}
		targetPID = foundPID
		description = fmt.Sprintf("Kill process on port %d (PID %d)", port, targetPID)
	}

	// Check permission (high risk)
	op := sandbox.NewOperation(sandbox.OpProcessKill, strconv.Itoa(targetPID), description).
		WithDetail("force", strconv.FormatBool(force))

	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Kill the process
	process, err := os.FindProcess(targetPID)
	if err != nil {
		return NewErrorResult(fmt.Errorf("process not found: %w", err)), nil
	}

	var killErr error
	if runtime.GOOS == "windows" || force {
		killErr = process.Kill()
	} else {
		killErr = process.Signal(os.Interrupt)
	}

	if killErr != nil {
		return NewErrorResult(fmt.Errorf("failed to kill process: %w", killErr)), nil
	}

	return NewSuccessResult(fmt.Sprintf("Successfully killed process %d", targetPID)), nil
}

// findPIDByPort finds the PID of the process using a specific port
func findPIDByPort(port int) (int, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("netstat", "-ano")
	} else if runtime.GOOS == "darwin" {
		cmd = exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-t")
	} else {
		cmd = exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-t")
	}

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("command failed: %w", err)
	}

	output := strings.TrimSpace(out.String())

	if runtime.GOOS == "windows" {
		// Parse netstat output for Windows
		lines := strings.Split(output, "\n")
		portStr := fmt.Sprintf(":%d", port)
		for _, line := range lines {
			if strings.Contains(line, portStr) && strings.Contains(line, "LISTENING") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					pid, err := strconv.Atoi(fields[len(fields)-1])
					if err == nil && pid > 0 {
						return pid, nil
					}
				}
			}
		}
	} else {
		// lsof -t returns just the PID
		pid, err := strconv.Atoi(output)
		if err == nil && pid > 0 {
			return pid, nil
		}
	}

	return 0, fmt.Errorf("no process found on port %d", port)
}

// StartProcessTool starts a process in the background
type StartProcessTool struct {
	permissions *sandbox.Manager
	workDir     string
}

// NewStartProcessTool creates a new start process tool
func NewStartProcessTool(permissions *sandbox.Manager, workDir string) *StartProcessTool {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	return &StartProcessTool{
		permissions: permissions,
		workDir:     workDir,
	}
}

func (t *StartProcessTool) Name() string {
	return "start_process"
}

func (t *StartProcessTool) Description() string {
	return "Start a process in the background"
}

func (t *StartProcessTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "command",
			Type:        TypeString,
			Description: "The command to run",
			Required:    true,
		},
		{
			Name:        "working_dir",
			Type:        TypeString,
			Description: "The working directory (default: project directory)",
			Required:    false,
		},
	}
}

func (t *StartProcessTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpProcessStart
}

func (t *StartProcessTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	command, err := RequiredStringArg(args, "command")
	if err != nil {
		return NewErrorResult(err), nil
	}

	workDir := GetStringArg(args, "working_dir", t.workDir)

	// Check permission
	op := sandbox.NewOperation(sandbox.OpProcessStart, command, fmt.Sprintf("Start background process: %s", command)).
		WithDetail("working_dir", workDir)

	if err := t.permissions.CheckAndApprove(ctx, op); err != nil {
		return NewErrorResult(err), nil
	}

	// Build the command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "start", "/B", command)
	} else {
		cmd = exec.Command("sh", "-c", command+" &")
	}

	cmd.Dir = workDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return NewErrorResult(fmt.Errorf("failed to start process: %w", err)), nil
	}

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	// Detach the process
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}

	return NewSuccessResultWithData(
		fmt.Sprintf("Started background process: %s (PID: %d)", command, pid),
		map[string]any{
			"pid":     pid,
			"command": command,
		},
	), nil
}

// ListProcessesTool lists running processes
type ListProcessesTool struct {
	permissions *sandbox.Manager
}

// NewListProcessesTool creates a new list processes tool
func NewListProcessesTool(permissions *sandbox.Manager) *ListProcessesTool {
	return &ListProcessesTool{permissions: permissions}
}

func (t *ListProcessesTool) Name() string {
	return "list_processes"
}

func (t *ListProcessesTool) Description() string {
	return "List running processes, optionally filtered by name or port"
}

func (t *ListProcessesTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "filter",
			Type:        TypeString,
			Description: "Filter processes by name (optional)",
			Required:    false,
		},
		{
			Name:        "port",
			Type:        TypeNumber,
			Description: "Filter by port number (optional)",
			Required:    false,
		},
	}
}

func (t *ListProcessesTool) RequiredPermission() sandbox.OperationType {
	return sandbox.OpListDir // Low risk, just listing
}

func (t *ListProcessesTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	filter := GetStringArg(args, "filter", "")
	port := GetIntArg(args, "port", 0)

	var cmd *exec.Cmd
	var output string

	if port > 0 {
		// List processes on specific port
		if runtime.GOOS == "windows" {
			cmd = exec.Command("netstat", "-ano", "-p", "tcp")
		} else {
			cmd = exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
		}
	} else if filter != "" {
		// Filter by process name
		if runtime.GOOS == "windows" {
			cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq *%s*", filter))
		} else {
			cmd = exec.Command("pgrep", "-l", filter)
		}
	} else {
		// List all processes (limited)
		if runtime.GOOS == "windows" {
			cmd = exec.Command("tasklist")
		} else {
			cmd = exec.Command("ps", "aux")
		}
	}

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// pgrep returns exit 1 if no processes found
		if filter != "" && runtime.GOOS != "windows" {
			return NewSuccessResult("No matching processes found"), nil
		}
		return NewErrorResult(fmt.Errorf("failed to list processes: %w", err)), nil
	}

	output = out.String()

	// Limit output size
	lines := strings.Split(output, "\n")
	if len(lines) > 50 {
		lines = lines[:50]
		output = strings.Join(lines, "\n") + "\n... (output truncated)"
	}

	return NewSuccessResult(output), nil
}
