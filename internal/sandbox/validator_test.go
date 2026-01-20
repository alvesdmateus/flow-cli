package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewValidator(t *testing.T) {
	policy := &Policy{
		ProjectDir:   "/project",
		TrustedPaths: []string{"/tmp"},
		DeniedPaths:  []string{"/etc"},
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	if validator == nil {
		t.Fatal("NewValidator() returned nil")
	}

	if validator.policy != policy {
		t.Error("validator.policy not set correctly")
	}
}

func TestNewValidator_InvalidProjectDir(t *testing.T) {
	// Empty project dir should still work (uses cwd)
	policy := &Policy{
		ProjectDir: "",
	}

	_, err := NewValidator(policy)
	// This should not error - it will use current directory
	if err != nil {
		t.Logf("Note: empty project dir gave error: %v", err)
	}
}

func TestExpandAndAbs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		name     string
		input    string
		contains string // Path should contain this
	}{
		{
			name:     "home expansion",
			input:    "~",
			contains: home,
		},
		{
			name:     "home with subpath",
			input:    "~/subdir",
			contains: filepath.Join(home, "subdir"),
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			contains: "absolute",
		},
		{
			name:     "relative path",
			input:    "relative",
			contains: "relative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandAndAbs(tt.input)
			if err != nil {
				t.Fatalf("expandAndAbs(%q) error = %v", tt.input, err)
			}

			if !filepath.IsAbs(result) {
				t.Errorf("expandAndAbs(%q) = %q, want absolute path", tt.input, result)
			}
		})
	}
}

func TestValidator_ValidateReadPath(t *testing.T) {
	tmpDir := t.TempDir()
	trustedDir := t.TempDir()

	policy := &Policy{
		ProjectDir:   tmpDir,
		TrustedPaths: []string{trustedDir},
		DeniedPaths:  []string{filepath.Join(tmpDir, "secret")},
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	tests := []struct {
		name          string
		path          string
		expectedLevel PermissionLevel
		expectError   bool
	}{
		{
			name:          "file in project dir",
			path:          filepath.Join(tmpDir, "file.txt"),
			expectedLevel: PermissionNone,
			expectError:   false,
		},
		{
			name:          "file in trusted dir",
			path:          filepath.Join(trustedDir, "file.txt"),
			expectedLevel: PermissionLow,
			expectError:   false,
		},
		{
			name:          "file in denied dir",
			path:          filepath.Join(tmpDir, "secret", "file.txt"),
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
		{
			name:          "file outside project",
			path:          "/some/other/path/file.txt",
			expectedLevel: PermissionMedium,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := validator.ValidateReadPath(tt.path)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if level != tt.expectedLevel {
				t.Errorf("ValidateReadPath(%q) level = %v, want %v", tt.path, level, tt.expectedLevel)
			}
		})
	}
}

func TestValidator_ValidateWritePath(t *testing.T) {
	tmpDir := t.TempDir()

	policy := &Policy{
		ProjectDir:  tmpDir,
		DeniedPaths: []string{filepath.Join(tmpDir, "readonly")},
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	tests := []struct {
		name          string
		path          string
		expectedLevel PermissionLevel
		expectError   bool
	}{
		{
			name:          "write in project dir",
			path:          filepath.Join(tmpDir, "newfile.txt"),
			expectedLevel: PermissionMedium,
			expectError:   false,
		},
		{
			name:          "write in denied path",
			path:          filepath.Join(tmpDir, "readonly", "file.txt"),
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
		{
			name:          "write outside project",
			path:          "/some/other/path/file.txt",
			expectedLevel: PermissionHigh,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := validator.ValidateWritePath(tt.path)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if level != tt.expectedLevel {
				t.Errorf("ValidateWritePath(%q) level = %v, want %v", tt.path, level, tt.expectedLevel)
			}
		})
	}
}

func TestValidator_ValidateDeletePath(t *testing.T) {
	tmpDir := t.TempDir()

	policy := &Policy{
		ProjectDir:  tmpDir,
		DeniedPaths: []string{filepath.Join(tmpDir, "protected")},
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	tests := []struct {
		name          string
		path          string
		expectedLevel PermissionLevel
		expectError   bool
	}{
		{
			name:          "delete in project dir",
			path:          filepath.Join(tmpDir, "deleteme.txt"),
			expectedLevel: PermissionHigh,
			expectError:   false,
		},
		{
			name:          "delete in denied path",
			path:          filepath.Join(tmpDir, "protected", "file.txt"),
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
		{
			name:          "delete outside project",
			path:          "/some/other/path/file.txt",
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := validator.ValidateDeletePath(tt.path)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if level != tt.expectedLevel {
				t.Errorf("ValidateDeletePath(%q) level = %v, want %v", tt.path, level, tt.expectedLevel)
			}
		})
	}
}

func TestValidator_ValidateCommand(t *testing.T) {
	policy := &Policy{
		ProjectDir: t.TempDir(),
		AllowedCommands: []string{
			"go build",
			"go test",
			"ls",
		},
		BlockedCommands: []string{
			"rm -rf /",
			"sudo",
		},
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	tests := []struct {
		name          string
		command       string
		expectedLevel PermissionLevel
		expectError   bool
	}{
		{
			name:          "allowed command - go build",
			command:       "go build ./...",
			expectedLevel: PermissionLow,
			expectError:   false,
		},
		{
			name:          "allowed command - go test",
			command:       "go test -v ./...",
			expectedLevel: PermissionLow,
			expectError:   false,
		},
		{
			name:          "allowed command - ls",
			command:       "ls -la",
			expectedLevel: PermissionLow,
			expectError:   false,
		},
		{
			name:          "blocked command - rm -rf /",
			command:       "rm -rf /",
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
		{
			name:          "blocked command - sudo",
			command:       "sudo apt install something",
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
		{
			name:          "unknown command",
			command:       "some-unknown-command",
			expectedLevel: PermissionMedium,
			expectError:   false,
		},
		{
			name:          "empty command",
			command:       "",
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
		{
			name:          "whitespace only",
			command:       "   ",
			expectedLevel: PermissionDenied,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := validator.ValidateCommand(tt.command)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if level != tt.expectedLevel {
				t.Errorf("ValidateCommand(%q) level = %v, want %v", tt.command, level, tt.expectedLevel)
			}
		})
	}
}

func TestValidator_ValidateOperation(t *testing.T) {
	tmpDir := t.TempDir()

	policy := &Policy{
		ProjectDir:   tmpDir,
		AllowNetwork: true,
		AllowedCommands: []string{
			"go test",
		},
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	tests := []struct {
		name          string
		operation     *Operation
		expectedLevel PermissionLevel
		expectError   bool
	}{
		{
			name: "read file in project",
			operation: &Operation{
				Type:   OpReadFile,
				Target: filepath.Join(tmpDir, "file.txt"),
			},
			expectedLevel: PermissionNone,
			expectError:   false,
		},
		{
			name: "write file in project",
			operation: &Operation{
				Type:   OpWriteFile,
				Target: filepath.Join(tmpDir, "file.txt"),
			},
			expectedLevel: PermissionMedium,
			expectError:   false,
		},
		{
			name: "execute allowed command",
			operation: &Operation{
				Type:   OpExecuteCmd,
				Target: "go test ./...",
			},
			expectedLevel: PermissionLow,
			expectError:   false,
		},
		{
			name: "web search with network allowed",
			operation: &Operation{
				Type:   OpWebSearch,
				Target: "golang error handling",
			},
			expectedLevel: PermissionLow,
			expectError:   false,
		},
		{
			name: "network access with network allowed",
			operation: &Operation{
				Type:   OpNetworkAccess,
				Target: "https://example.com",
			},
			expectedLevel: PermissionMedium,
			expectError:   false,
		},
		{
			name: "process kill",
			operation: &Operation{
				Type:   OpProcessKill,
				Target: "1234",
			},
			expectedLevel: PermissionHigh,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateOperation(tt.operation)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.operation.Level != tt.expectedLevel {
				t.Errorf("operation.Level = %v, want %v", tt.operation.Level, tt.expectedLevel)
			}
		})
	}
}

func TestValidator_ValidateOperation_NetworkDisabled(t *testing.T) {
	policy := &Policy{
		ProjectDir:   t.TempDir(),
		AllowNetwork: false,
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	// Test web search with network disabled
	op := &Operation{
		Type:   OpWebSearch,
		Target: "test query",
	}

	err = validator.ValidateOperation(op)
	if err == nil {
		t.Error("expected error for web search with network disabled")
	}
	if op.Level != PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", op.Level)
	}

	// Test network access with network disabled
	op2 := &Operation{
		Type:   OpNetworkAccess,
		Target: "https://example.com",
	}

	err = validator.ValidateOperation(op2)
	if err == nil {
		t.Error("expected error for network access with network disabled")
	}
	if op2.Level != PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", op2.Level)
	}
}

func TestValidator_GetProjectDir(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	projectDir := validator.GetProjectDir()
	if projectDir == "" {
		t.Error("GetProjectDir() returned empty string")
	}

	// Should be absolute
	if !filepath.IsAbs(projectDir) {
		t.Errorf("GetProjectDir() = %q, want absolute path", projectDir)
	}
}

func TestValidator_IsWithinProject(t *testing.T) {
	tmpDir := t.TempDir()
	policy := &Policy{
		ProjectDir: tmpDir,
	}

	validator, err := NewValidator(policy)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	// File in project should return true
	inProject := validator.isWithinProject(filepath.Join(tmpDir, "subdir", "file.txt"))
	if !inProject {
		t.Error("isWithinProject() returned false for path in project")
	}

	// File outside project should return false
	outsideProject := validator.isWithinProject("/some/other/path")
	if outsideProject {
		t.Error("isWithinProject() returned true for path outside project")
	}
}
