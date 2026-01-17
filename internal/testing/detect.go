package testing

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectProjectType determines the project type based on files in the directory.
func DetectProjectType(dir string) ProjectType {
	// Check for Go
	if fileExists(filepath.Join(dir, "go.mod")) || hasFileWithExt(dir, ".go") {
		return ProjectGo
	}

	// Check for Python
	if fileExists(filepath.Join(dir, "pyproject.toml")) ||
		fileExists(filepath.Join(dir, "setup.py")) ||
		fileExists(filepath.Join(dir, "requirements.txt")) ||
		hasFileWithExt(dir, ".py") {
		return ProjectPython
	}

	// Check for Node.js
	if fileExists(filepath.Join(dir, "package.json")) {
		return ProjectNode
	}

	// Check for Rust
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return ProjectRust
	}

	return ProjectUnknown
}

// GetTestFilePath returns the appropriate test file path for a source file.
func GetTestFilePath(sourcePath string, projectType ProjectType) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	switch projectType {
	case ProjectGo:
		// Go: foo.go -> foo_test.go
		return filepath.Join(dir, name+"_test.go")

	case ProjectPython:
		// Python: foo.py -> test_foo.py (in same dir or tests/ dir)
		testsDir := filepath.Join(filepath.Dir(dir), "tests")
		if dirExists(testsDir) {
			return filepath.Join(testsDir, "test_"+name+".py")
		}
		return filepath.Join(dir, "test_"+name+".py")

	case ProjectNode:
		// Node: foo.ts -> foo.test.ts or foo.spec.ts
		// Check if there's a __tests__ directory
		testsDir := filepath.Join(dir, "__tests__")
		if dirExists(testsDir) {
			return filepath.Join(testsDir, name+".test"+ext)
		}
		return filepath.Join(dir, name+".test"+ext)

	case ProjectRust:
		// Rust: typically in same file or in tests/ directory
		testsDir := filepath.Join(filepath.Dir(dir), "tests")
		if dirExists(testsDir) {
			return filepath.Join(testsDir, name+"_test.rs")
		}
		return filepath.Join(dir, name+"_test.rs")

	default:
		return filepath.Join(dir, name+"_test"+ext)
	}
}

// GetTestCommand returns the test command for the project type.
func GetTestCommand(projectType ProjectType) (string, []string) {
	switch projectType {
	case ProjectGo:
		return "go", []string{"test"}
	case ProjectPython:
		return "pytest", []string{}
	case ProjectNode:
		return "npm", []string{"test"}
	case ProjectRust:
		return "cargo", []string{"test"}
	default:
		return "", nil
	}
}

// GetCoverageArgs returns additional arguments for coverage.
func GetCoverageArgs(projectType ProjectType) []string {
	switch projectType {
	case ProjectGo:
		return []string{"-cover", "-coverprofile=coverage.out"}
	case ProjectPython:
		return []string{"--cov", "--cov-report=term-missing"}
	case ProjectNode:
		return []string{"--", "--coverage"}
	case ProjectRust:
		// cargo-tarpaulin for coverage
		return []string{}
	default:
		return nil
	}
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// hasFileWithExt checks if the directory contains any file with the given extension.
func hasFileWithExt(dir, ext string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ext) {
			return true
		}
	}
	return false
}
