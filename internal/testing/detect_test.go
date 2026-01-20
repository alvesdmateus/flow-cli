package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectType(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "flow-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		files    []string
		expected ProjectType
	}{
		{
			name:     "Go project with go.mod",
			files:    []string{"go.mod"},
			expected: ProjectGo,
		},
		{
			name:     "Go project with .go files",
			files:    []string{"main.go"},
			expected: ProjectGo,
		},
		{
			name:     "Python project with pyproject.toml",
			files:    []string{"pyproject.toml"},
			expected: ProjectPython,
		},
		{
			name:     "Python project with requirements.txt",
			files:    []string{"requirements.txt"},
			expected: ProjectPython,
		},
		{
			name:     "Python project with setup.py",
			files:    []string{"setup.py"},
			expected: ProjectPython,
		},
		{
			name:     "Node project with package.json",
			files:    []string{"package.json"},
			expected: ProjectNode,
		},
		{
			name:     "Rust project with Cargo.toml",
			files:    []string{"Cargo.toml"},
			expected: ProjectRust,
		},
		{
			name:     "Unknown project",
			files:    []string{"readme.txt"},
			expected: ProjectUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a subdirectory for this test
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("failed to create test dir: %v", err)
			}

			// Create the files
			for _, f := range tt.files {
				filePath := filepath.Join(testDir, f)
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file %s: %v", f, err)
				}
			}

			// Detect project type
			result := DetectProjectType(testDir)
			if result != tt.expected {
				t.Errorf("DetectProjectType() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetTestFilePath(t *testing.T) {
	tests := []struct {
		sourcePath  string
		projectType ProjectType
		expected    string
	}{
		{
			sourcePath:  filepath.Join("project", "main.go"),
			projectType: ProjectGo,
			expected:    filepath.Join("project", "main_test.go"),
		},
		{
			sourcePath:  filepath.Join("project", "pkg", "utils.go"),
			projectType: ProjectGo,
			expected:    filepath.Join("project", "pkg", "utils_test.go"),
		},
		{
			sourcePath:  filepath.Join("project", "app.py"),
			projectType: ProjectPython,
			expected:    filepath.Join("project", "test_app.py"),
		},
		{
			sourcePath:  filepath.Join("project", "src", "index.ts"),
			projectType: ProjectNode,
			expected:    filepath.Join("project", "src", "index.test.ts"),
		},
		{
			sourcePath:  filepath.Join("project", "src", "lib.rs"),
			projectType: ProjectRust,
			expected:    filepath.Join("project", "src", "lib_test.rs"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.sourcePath, func(t *testing.T) {
			result := GetTestFilePath(tt.sourcePath, tt.projectType)
			if result != tt.expected {
				t.Errorf("GetTestFilePath(%q, %v) = %q, expected %q",
					tt.sourcePath, tt.projectType, result, tt.expected)
			}
		})
	}
}

func TestGetTestCommand(t *testing.T) {
	tests := []struct {
		projectType ProjectType
		expectedCmd string
		expectedLen int // minimum number of args
	}{
		{ProjectGo, "go", 1},
		{ProjectPython, "pytest", 0},
		{ProjectNode, "npm", 1},
		{ProjectRust, "cargo", 1},
		{ProjectUnknown, "", 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.projectType), func(t *testing.T) {
			cmd, args := GetTestCommand(tt.projectType)
			if cmd != tt.expectedCmd {
				t.Errorf("GetTestCommand(%v) cmd = %q, expected %q",
					tt.projectType, cmd, tt.expectedCmd)
			}
			if len(args) < tt.expectedLen {
				t.Errorf("GetTestCommand(%v) args len = %d, expected >= %d",
					tt.projectType, len(args), tt.expectedLen)
			}
		})
	}
}

func TestGetCoverageArgs(t *testing.T) {
	tests := []struct {
		projectType ProjectType
		expectArgs  bool
	}{
		{ProjectGo, true},
		{ProjectPython, true},
		{ProjectNode, true},
		{ProjectRust, false}, // Rust coverage needs external tool
	}

	for _, tt := range tests {
		t.Run(string(tt.projectType), func(t *testing.T) {
			args := GetCoverageArgs(tt.projectType)
			if tt.expectArgs && len(args) == 0 {
				t.Errorf("GetCoverageArgs(%v) expected args, got none", tt.projectType)
			}
		})
	}
}

func TestProjectTypeString(t *testing.T) {
	tests := []struct {
		pt       ProjectType
		expected string
	}{
		{ProjectGo, "go"},
		{ProjectPython, "python"},
		{ProjectNode, "node"},
		{ProjectRust, "rust"},
		{ProjectUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.pt.String() != tt.expected {
				t.Errorf("ProjectType.String() = %q, expected %q", tt.pt.String(), tt.expected)
			}
		})
	}
}
