package sandbox

import (
	"testing"
)

func TestPermissionLevel_String(t *testing.T) {
	tests := []struct {
		level    PermissionLevel
		expected string
	}{
		{PermissionNone, "none"},
		{PermissionLow, "low"},
		{PermissionMedium, "medium"},
		{PermissionHigh, "high"},
		{PermissionDenied, "denied"},
		{PermissionLevel(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("PermissionLevel(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestDefaultPolicy(t *testing.T) {
	policy := DefaultPolicy()

	if policy == nil {
		t.Fatal("DefaultPolicy() returned nil")
	}

	// Check that project dir is set (should be current directory)
	if policy.ProjectDir == "" {
		t.Error("DefaultPolicy().ProjectDir is empty")
	}

	// Check that denied paths are set
	if len(policy.DeniedPaths) == 0 {
		t.Error("DefaultPolicy().DeniedPaths is empty")
	}

	// Check that allowed commands are set
	if len(policy.AllowedCommands) == 0 {
		t.Error("DefaultPolicy().AllowedCommands is empty")
	}

	// Check that blocked commands are set
	if len(policy.BlockedCommands) == 0 {
		t.Error("DefaultPolicy().BlockedCommands is empty")
	}

	// Check default values
	if policy.AutoApprove != false {
		t.Error("DefaultPolicy().AutoApprove should be false")
	}

	if policy.AllowNetwork != true {
		t.Error("DefaultPolicy().AllowNetwork should be true")
	}
}

func TestDefaultPolicy_DeniedPathsContainSensitive(t *testing.T) {
	policy := DefaultPolicy()

	// Check that sensitive directories are denied
	sensitivePatterns := []string{".ssh", ".gnupg", ".aws", ".azure", ".kube"}

	for _, pattern := range sensitivePatterns {
		found := false
		for _, denied := range policy.DeniedPaths {
			if containsPath(denied, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultPolicy().DeniedPaths should contain path with %q", pattern)
		}
	}
}

func TestDefaultPolicy_AllowedCommandsContainSafe(t *testing.T) {
	policy := DefaultPolicy()

	// Check that safe commands are allowed
	safeCommands := []string{"go build", "go test", "npm install", "git status", "ls"}

	for _, cmd := range safeCommands {
		found := false
		for _, allowed := range policy.AllowedCommands {
			if allowed == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultPolicy().AllowedCommands should contain %q", cmd)
		}
	}
}

func TestDefaultPolicy_BlockedCommandsContainDangerous(t *testing.T) {
	policy := DefaultPolicy()

	// Check that dangerous commands are blocked
	dangerousCommands := []string{"rm -rf /", "sudo", "chmod 777"}

	for _, cmd := range dangerousCommands {
		found := false
		for _, blocked := range policy.BlockedCommands {
			if blocked == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultPolicy().BlockedCommands should contain %q", cmd)
		}
	}
}

func TestNewOperation(t *testing.T) {
	op := NewOperation(OpReadFile, "/path/to/file", "Reading a file")

	if op == nil {
		t.Fatal("NewOperation() returned nil")
	}

	if op.Type != OpReadFile {
		t.Errorf("op.Type = %v, want %v", op.Type, OpReadFile)
	}

	if op.Target != "/path/to/file" {
		t.Errorf("op.Target = %q, want %q", op.Target, "/path/to/file")
	}

	if op.Description != "Reading a file" {
		t.Errorf("op.Description = %q, want %q", op.Description, "Reading a file")
	}

	if op.Level != PermissionMedium {
		t.Errorf("op.Level = %v, want %v (default)", op.Level, PermissionMedium)
	}

	if op.Details == nil {
		t.Error("op.Details should not be nil")
	}
}

func TestOperation_WithDetail(t *testing.T) {
	op := NewOperation(OpWriteFile, "/path", "desc")

	result := op.WithDetail("size", "1024")

	// Should return same operation (for chaining)
	if result != op {
		t.Error("WithDetail() should return same operation for chaining")
	}

	// Should have the detail
	if op.Details["size"] != "1024" {
		t.Errorf("op.Details[\"size\"] = %q, want %q", op.Details["size"], "1024")
	}
}

func TestOperation_WithDetail_Multiple(t *testing.T) {
	op := NewOperation(OpExecuteCmd, "ls", "listing").
		WithDetail("cwd", "/home").
		WithDetail("timeout", "30s")

	if len(op.Details) != 2 {
		t.Errorf("expected 2 details, got %d", len(op.Details))
	}

	if op.Details["cwd"] != "/home" {
		t.Errorf("op.Details[\"cwd\"] = %q, want %q", op.Details["cwd"], "/home")
	}

	if op.Details["timeout"] != "30s" {
		t.Errorf("op.Details[\"timeout\"] = %q, want %q", op.Details["timeout"], "30s")
	}
}

func TestOperation_WithLevel(t *testing.T) {
	op := NewOperation(OpDeleteFile, "/path", "desc")

	result := op.WithLevel(PermissionHigh)

	// Should return same operation (for chaining)
	if result != op {
		t.Error("WithLevel() should return same operation for chaining")
	}

	// Should have updated level
	if op.Level != PermissionHigh {
		t.Errorf("op.Level = %v, want %v", op.Level, PermissionHigh)
	}
}

func TestOperationType_Values(t *testing.T) {
	// Ensure operation types have expected string values
	tests := []struct {
		opType   OperationType
		expected string
	}{
		{OpReadFile, "read_file"},
		{OpWriteFile, "write_file"},
		{OpDeleteFile, "delete_file"},
		{OpCreateDir, "create_directory"},
		{OpListDir, "list_directory"},
		{OpExecuteCmd, "execute_command"},
		{OpWebSearch, "web_search"},
		{OpNetworkAccess, "network_access"},
		{OpProcessKill, "process_kill"},
		{OpProcessStart, "process_start"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.opType) != tt.expected {
				t.Errorf("OperationType = %q, want %q", tt.opType, tt.expected)
			}
		})
	}
}

// Helper function to check if a path contains a pattern
func containsPath(path, pattern string) bool {
	for i := 0; i <= len(path)-len(pattern); i++ {
		if path[i:i+len(pattern)] == pattern {
			return true
		}
	}
	return false
}
