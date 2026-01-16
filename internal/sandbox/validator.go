package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validator handles validation of paths and commands against the security policy
type Validator struct {
	policy *Policy

	// Cached absolute paths for faster comparison
	projectDirAbs   string
	trustedPathsAbs []string
	deniedPathsAbs  []string
}

// NewValidator creates a new validator with the given policy
func NewValidator(policy *Policy) (*Validator, error) {
	v := &Validator{
		policy: policy,
	}

	// Resolve project directory to absolute path
	absProject, err := filepath.Abs(policy.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project directory: %w", err)
	}
	v.projectDirAbs = absProject

	// Resolve trusted paths
	v.trustedPathsAbs = make([]string, 0, len(policy.TrustedPaths))
	for _, p := range policy.TrustedPaths {
		abs, err := expandAndAbs(p)
		if err != nil {
			continue // Skip invalid paths
		}
		v.trustedPathsAbs = append(v.trustedPathsAbs, abs)
	}

	// Resolve denied paths
	v.deniedPathsAbs = make([]string, 0, len(policy.DeniedPaths))
	for _, p := range policy.DeniedPaths {
		abs, err := expandAndAbs(p)
		if err != nil {
			continue // Skip invalid paths
		}
		v.deniedPathsAbs = append(v.deniedPathsAbs, abs)
	}

	return v, nil
}

// expandAndAbs expands ~ to home directory and returns absolute path
func expandAndAbs(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return filepath.Abs(path)
}

// ValidateReadPath checks if a path can be read
func (v *Validator) ValidateReadPath(path string) (PermissionLevel, error) {
	absPath, err := expandAndAbs(path)
	if err != nil {
		return PermissionDenied, fmt.Errorf("invalid path: %w", err)
	}

	// Check denied paths first (highest priority)
	if v.isDeniedPath(absPath) {
		return PermissionDenied, fmt.Errorf("path is in denied list: %s", path)
	}

	// Check if within project directory (no permission needed)
	if v.isWithinProject(absPath) {
		return PermissionNone, nil
	}

	// Check if in trusted paths (low permission)
	if v.isTrustedPath(absPath) {
		return PermissionLow, nil
	}

	// Outside project and trusted paths - needs approval
	return PermissionMedium, nil
}

// ValidateWritePath checks if a path can be written to
func (v *Validator) ValidateWritePath(path string) (PermissionLevel, error) {
	absPath, err := expandAndAbs(path)
	if err != nil {
		return PermissionDenied, fmt.Errorf("invalid path: %w", err)
	}

	// Check denied paths first
	if v.isDeniedPath(absPath) {
		return PermissionDenied, fmt.Errorf("path is in denied list: %s", path)
	}

	// Writing within project directory - medium permission (still confirm)
	if v.isWithinProject(absPath) {
		return PermissionMedium, nil
	}

	// Writing outside project - high permission
	return PermissionHigh, nil
}

// ValidateDeletePath checks if a path can be deleted
func (v *Validator) ValidateDeletePath(path string) (PermissionLevel, error) {
	absPath, err := expandAndAbs(path)
	if err != nil {
		return PermissionDenied, fmt.Errorf("invalid path: %w", err)
	}

	// Check denied paths first
	if v.isDeniedPath(absPath) {
		return PermissionDenied, fmt.Errorf("path is in denied list: %s", path)
	}

	// Deleting within project directory - high permission (destructive)
	if v.isWithinProject(absPath) {
		return PermissionHigh, nil
	}

	// Deleting outside project - denied
	return PermissionDenied, fmt.Errorf("cannot delete files outside project directory: %s", path)
}

// ValidateCommand checks if a command can be executed
func (v *Validator) ValidateCommand(command string) (PermissionLevel, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return PermissionDenied, fmt.Errorf("empty command")
	}

	// Check blocked commands first (highest priority)
	for _, blocked := range v.policy.BlockedCommands {
		if strings.Contains(command, blocked) {
			return PermissionDenied, fmt.Errorf("command contains blocked pattern: %s", blocked)
		}
	}

	// Check allowed commands (prefix match)
	for _, allowed := range v.policy.AllowedCommands {
		if strings.HasPrefix(command, allowed) {
			return PermissionLow, nil
		}
	}

	// Unknown command - needs approval
	return PermissionMedium, nil
}

// ValidateOperation validates an operation and sets its permission level
func (v *Validator) ValidateOperation(op *Operation) error {
	var level PermissionLevel
	var err error

	switch op.Type {
	case OpReadFile, OpListDir:
		level, err = v.ValidateReadPath(op.Target)
	case OpWriteFile, OpCreateDir:
		level, err = v.ValidateWritePath(op.Target)
	case OpDeleteFile:
		level, err = v.ValidateDeletePath(op.Target)
	case OpExecuteCmd:
		level, err = v.ValidateCommand(op.Target)
	case OpWebSearch:
		if v.policy.AllowNetwork {
			level = PermissionLow
		} else {
			level = PermissionDenied
			err = fmt.Errorf("network access is disabled")
		}
	case OpNetworkAccess:
		if v.policy.AllowNetwork {
			level = PermissionMedium
		} else {
			level = PermissionDenied
			err = fmt.Errorf("network access is disabled")
		}
	case OpProcessKill:
		level = PermissionHigh
	case OpProcessStart:
		level = PermissionMedium
	default:
		level = PermissionMedium
	}

	op.Level = level
	return err
}

// isWithinProject checks if a path is within the project directory
func (v *Validator) isWithinProject(absPath string) bool {
	return strings.HasPrefix(absPath, v.projectDirAbs)
}

// isTrustedPath checks if a path is within any trusted path
func (v *Validator) isTrustedPath(absPath string) bool {
	for _, trusted := range v.trustedPathsAbs {
		if strings.HasPrefix(absPath, trusted) {
			return true
		}
	}
	return false
}

// isDeniedPath checks if a path is within any denied path
func (v *Validator) isDeniedPath(absPath string) bool {
	for _, denied := range v.deniedPathsAbs {
		if strings.HasPrefix(absPath, denied) {
			return true
		}
	}
	return false
}

// GetProjectDir returns the absolute project directory path
func (v *Validator) GetProjectDir() string {
	return v.projectDirAbs
}
