package sandbox

import (
	"os"
	"path/filepath"
)

// PermissionLevel represents the level of permission required for an operation
type PermissionLevel int

const (
	// PermissionNone - no permission required (e.g., reading allowed files)
	PermissionNone PermissionLevel = iota
	// PermissionLow - low risk, auto-approved in permissive mode
	PermissionLow
	// PermissionMedium - medium risk, requires approval unless auto-approve
	PermissionMedium
	// PermissionHigh - high risk, always requires explicit approval
	PermissionHigh
	// PermissionDenied - operation is never allowed
	PermissionDenied
)

// PermissivityMode defines the level of user approval required
type PermissivityMode string

const (
	// PermissivityAutoAccept - accept all changes (except denied operations)
	PermissivityAutoAccept PermissivityMode = "auto-accept"
	// PermissivityRevisionAlways - review every change
	PermissivityRevisionAlways PermissivityMode = "revision-always"
	// PermissivityRevisionBig - review significant changes (default)
	PermissivityRevisionBig PermissivityMode = "revision-big"
	// PermissivityRevisionArchitecture - review only architectural changes
	PermissivityRevisionArchitecture PermissivityMode = "revision-architecture"
)

// ValidPermissivityModes returns all valid permissivity mode values
func ValidPermissivityModes() []PermissivityMode {
	return []PermissivityMode{
		PermissivityAutoAccept,
		PermissivityRevisionAlways,
		PermissivityRevisionBig,
		PermissivityRevisionArchitecture,
	}
}

// IsValidPermissivityMode checks if a string is a valid permissivity mode
func IsValidPermissivityMode(mode string) bool {
	for _, valid := range ValidPermissivityModes() {
		if string(valid) == mode {
			return true
		}
	}
	return false
}

// String returns a human-readable name for the permission level
func (p PermissionLevel) String() string {
	switch p {
	case PermissionNone:
		return "none"
	case PermissionLow:
		return "low"
	case PermissionMedium:
		return "medium"
	case PermissionHigh:
		return "high"
	case PermissionDenied:
		return "denied"
	default:
		return "unknown"
	}
}

// OperationType represents the type of operation being performed
type OperationType string

const (
	OpReadFile      OperationType = "read_file"
	OpWriteFile     OperationType = "write_file"
	OpDeleteFile    OperationType = "delete_file"
	OpCreateDir     OperationType = "create_directory"
	OpListDir       OperationType = "list_directory"
	OpExecuteCmd    OperationType = "execute_command"
	OpWebSearch     OperationType = "web_search"
	OpNetworkAccess OperationType = "network_access"
	OpProcessKill   OperationType = "process_kill"
	OpProcessStart  OperationType = "process_start"
)

// Operation represents an action that requires permission
type Operation struct {
	Type        OperationType
	Description string
	Target      string            // File path, command, URL, etc.
	Details     map[string]string // Additional context
	Level       PermissionLevel
}

// Policy defines the security policy for the sandbox
type Policy struct {
	// ProjectDir is the base directory for file operations
	ProjectDir string

	// TrustedPaths are additional paths that can be read
	TrustedPaths []string

	// DeniedPaths are paths that should never be accessed
	DeniedPaths []string

	// AllowedCommands are command prefixes that are always allowed
	AllowedCommands []string

	// BlockedCommands are command patterns that are never allowed
	BlockedCommands []string

	// AutoApprove skips confirmation for medium-risk operations
	// Deprecated: Use Permissivity instead
	AutoApprove bool

	// Permissivity controls the level of user approval required
	// Options: auto-accept, revision-always, revision-big, revision-architecture
	// Default: revision-big
	Permissivity PermissivityMode

	// AllowNetwork permits network operations (web search, fetch)
	AllowNetwork bool

	// Classification holds thresholds for change classification
	Classification ClassificationConfig
}

// DefaultPolicy creates a policy with sensible defaults
func DefaultPolicy() *Policy {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	return &Policy{
		ProjectDir:   cwd,
		TrustedPaths: []string{},
		DeniedPaths: []string{
			filepath.Join(home, ".ssh"),
			filepath.Join(home, ".gnupg"),
			filepath.Join(home, ".aws"),
			filepath.Join(home, ".azure"),
			filepath.Join(home, ".config", "gcloud"),
			filepath.Join(home, ".kube"),
			filepath.Join(home, ".docker", "config.json"),
		},
		AllowedCommands: []string{
			"go build", "go test", "go run", "go mod", "go fmt", "go vet",
			"npm install", "npm test", "npm run", "npm start", "npm ci",
			"yarn install", "yarn test", "yarn run", "yarn start",
			"pnpm install", "pnpm test", "pnpm run",
			"cargo build", "cargo test", "cargo run", "cargo check",
			"make", "cmake",
			"git status", "git diff", "git log", "git branch", "git show",
			"ls", "dir", "pwd", "echo", "cat", "head", "tail", "wc",
			"grep", "find", "which", "where", "type",
		},
		BlockedCommands: []string{
			"rm -rf /",
			"rm -rf ~",
			"rm -rf $HOME",
			"sudo",
			"su ",
			"chmod 777",
			"chown",
			"mkfs",
			"dd if=",
			"> /dev/",
			":(){ :|:& };:",
			"curl | sh",
			"curl | bash",
			"wget | sh",
			"wget | bash",
		},
		AutoApprove:    false,
		Permissivity:   PermissivityRevisionBig, // Default: review significant changes
		AllowNetwork:   true,
		Classification: DefaultClassificationConfig(),
	}
}

// NewOperation creates a new operation with the given parameters
func NewOperation(opType OperationType, target, description string) *Operation {
	return &Operation{
		Type:        opType,
		Target:      target,
		Description: description,
		Details:     make(map[string]string),
		Level:       PermissionMedium, // Default, will be adjusted by validator
	}
}

// WithDetail adds a detail to the operation
func (o *Operation) WithDetail(key, value string) *Operation {
	o.Details[key] = value
	return o
}

// WithLevel sets the permission level
func (o *Operation) WithLevel(level PermissionLevel) *Operation {
	o.Level = level
	return o
}
