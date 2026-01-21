package sandbox

import (
	"path/filepath"
	"strings"
)

// ChangeSize represents the magnitude of a change
type ChangeSize int

const (
	// ChangeTrivial - less than 5 lines changed
	ChangeTrivial ChangeSize = iota
	// ChangeSmall - less than 20 lines changed
	ChangeSmall
	// ChangeMedium - 20-100 lines or 2-3 files
	ChangeMedium
	// ChangeLarge - more than 100 lines or 4+ files
	ChangeLarge
	// ChangeArchitectural - new packages, interfaces, core files
	ChangeArchitectural
)

// String returns a human-readable name for the change size
func (c ChangeSize) String() string {
	switch c {
	case ChangeTrivial:
		return "trivial"
	case ChangeSmall:
		return "small"
	case ChangeMedium:
		return "medium"
	case ChangeLarge:
		return "large"
	case ChangeArchitectural:
		return "architectural"
	default:
		return "unknown"
	}
}

// ClassificationConfig holds thresholds for change classification
type ClassificationConfig struct {
	SmallLineThreshold  int // Lines below this are "small" (default: 20)
	MediumLineThreshold int // Lines below this are "medium" (default: 100)
}

// DefaultClassificationConfig returns sensible defaults
func DefaultClassificationConfig() ClassificationConfig {
	return ClassificationConfig{
		SmallLineThreshold:  20,
		MediumLineThreshold: 100,
	}
}

// ChangeClassifier analyzes changes to determine their size and nature
type ChangeClassifier struct {
	config ClassificationConfig
}

// NewChangeClassifier creates a new classifier with the given config
func NewChangeClassifier(config ClassificationConfig) *ChangeClassifier {
	return &ChangeClassifier{config: config}
}

// ChangeInfo holds information about a file change
type ChangeInfo struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
	IsNewFile    bool
}

// ClassifyChange determines the size of a single file change
func (c *ChangeClassifier) ClassifyChange(info ChangeInfo) ChangeSize {
	// Check if it's an architectural change first
	if c.isArchitecturalFile(info.Path) {
		return ChangeArchitectural
	}

	// Check for new package/directory creation
	if info.IsNewFile && c.isNewPackage(info.Path) {
		return ChangeArchitectural
	}

	totalLines := info.LinesAdded + info.LinesRemoved

	if totalLines < 5 {
		return ChangeTrivial
	}
	if totalLines < c.config.SmallLineThreshold {
		return ChangeSmall
	}
	if totalLines < c.config.MediumLineThreshold {
		return ChangeMedium
	}
	return ChangeLarge
}

// ClassifyBatch determines the overall size of multiple changes
func (c *ChangeClassifier) ClassifyBatch(changes []ChangeInfo) ChangeSize {
	if len(changes) == 0 {
		return ChangeTrivial
	}

	// Track metrics
	totalLines := 0
	maxSize := ChangeTrivial
	hasArchitectural := false

	for _, change := range changes {
		size := c.ClassifyChange(change)
		totalLines += change.LinesAdded + change.LinesRemoved

		if size == ChangeArchitectural {
			hasArchitectural = true
		}
		if size > maxSize {
			maxSize = size
		}
	}

	// Architectural changes always return architectural
	if hasArchitectural {
		return ChangeArchitectural
	}

	// Multiple files increase the classification
	fileCount := len(changes)
	if fileCount >= 4 {
		if maxSize < ChangeLarge {
			maxSize = ChangeLarge
		}
	} else if fileCount >= 2 {
		if maxSize < ChangeMedium {
			maxSize = ChangeMedium
		}
	}

	return maxSize
}

// isArchitecturalFile checks if a file is considered architectural
func (c *ChangeClassifier) isArchitecturalFile(path string) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	// Core project files
	coreFiles := []string{
		"main.go",
		"go.mod",
		"go.sum",
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
		"Makefile",
		"CMakeLists.txt",
		"package.json",
		"Cargo.toml",
		"pyproject.toml",
		"setup.py",
	}

	for _, core := range coreFiles {
		if base == core {
			return true
		}
	}

	// Root config files
	if dir == "." || dir == "" {
		configExtensions := []string{".yaml", ".yml", ".json", ".toml"}
		for _, ext := range configExtensions {
			if strings.HasSuffix(base, ext) {
				return true
			}
		}
	}

	// Interface definitions in Go
	if strings.HasSuffix(path, ".go") {
		// Check if in cmd/ or internal/app/ directories
		if strings.Contains(path, "cmd/") || strings.Contains(path, "internal/app/") {
			return true
		}
	}

	// Common architectural directories
	architecturalDirs := []string{
		"cmd/",
		"internal/app/",
		"pkg/",
		"api/",
		"proto/",
		"schemas/",
	}

	for _, archDir := range architecturalDirs {
		if strings.Contains(path, archDir) {
			// Only flag as architectural if it's creating new structure
			return false // Individual files in these dirs need new file check
		}
	}

	return false
}

// isNewPackage checks if creating a new file would create a new package/directory
func (c *ChangeClassifier) isNewPackage(path string) bool {
	path = filepath.ToSlash(path)

	// Check if it's creating a new directory structure
	dir := filepath.Dir(path)

	// Creating files in cmd/, internal/*, or pkg/* directories
	if strings.HasPrefix(dir, "cmd/") ||
		strings.HasPrefix(dir, "internal/") ||
		strings.HasPrefix(dir, "pkg/") {
		// This is potentially creating a new package
		// In practice, we'd check if the directory already exists
		return true
	}

	return false
}

// IsInterfaceChange checks if the change involves interface definitions
func (c *ChangeClassifier) IsInterfaceChange(path string, content string) bool {
	if !strings.HasSuffix(path, ".go") {
		return false
	}

	// Look for interface declarations
	return strings.Contains(content, "type ") && strings.Contains(content, " interface {")
}
