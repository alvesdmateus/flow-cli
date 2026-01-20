package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathGuard provides path traversal protection with symlink resolution.
type PathGuard struct {
	// AllowedRoots are the directories that can be accessed
	AllowedRoots []string

	// DeniedPaths are paths that should never be accessed
	DeniedPaths []string

	// FollowSymlinks determines if symlinks should be resolved
	FollowSymlinks bool

	// MaxSymlinkDepth is the maximum depth of symlink resolution
	MaxSymlinkDepth int
}

// NewPathGuard creates a new path guard with default settings.
func NewPathGuard(allowedRoots []string) *PathGuard {
	resolved := make([]string, 0, len(allowedRoots))
	for _, root := range allowedRoots {
		if abs, err := filepath.Abs(root); err == nil {
			resolved = append(resolved, abs)
		}
	}

	return &PathGuard{
		AllowedRoots:    resolved,
		DeniedPaths:     defaultDeniedPaths(),
		FollowSymlinks:  true,
		MaxSymlinkDepth: 10,
	}
}

func defaultDeniedPaths() []string {
	home, _ := os.UserHomeDir()
	denied := []string{
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".gnupg"),
		filepath.Join(home, ".aws"),
		filepath.Join(home, ".azure"),
		filepath.Join(home, ".config", "gcloud"),
		filepath.Join(home, ".kube"),
		filepath.Join(home, ".docker", "config.json"),
		filepath.Join(home, ".npmrc"),
		filepath.Join(home, ".pypirc"),
		filepath.Join(home, ".netrc"),
		filepath.Join(home, ".git-credentials"),
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
	}
	return denied
}

// PathValidation contains the result of path validation.
type PathValidation struct {
	Original     string
	Resolved     string
	IsAllowed    bool
	IsDenied     bool
	IsSymlink    bool
	SymlinkDepth int
	Reason       string
	Warnings     []string
}

// ValidatePath validates a path for safety.
func (pg *PathGuard) ValidatePath(path string) *PathValidation {
	result := &PathValidation{
		Original: path,
		Warnings: make([]string, 0),
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		result.IsDenied = true
		result.Reason = fmt.Sprintf("cannot resolve path: %v", err)
		return result
	}

	// Clean the path to remove .. and .
	cleanPath := filepath.Clean(absPath)
	result.Resolved = cleanPath

	// Check for path traversal attempts in the original path
	if pg.hasTraversalAttempt(path) {
		result.Warnings = append(result.Warnings, "path contains traversal sequences")
	}

	// Resolve symlinks if enabled
	if pg.FollowSymlinks {
		realPath, depth, err := pg.resolveSymlinks(cleanPath)
		if err != nil {
			result.IsDenied = true
			result.Reason = fmt.Sprintf("symlink resolution failed: %v", err)
			return result
		}
		if realPath != cleanPath {
			result.IsSymlink = true
			result.SymlinkDepth = depth
			result.Resolved = realPath
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("path is a symlink (depth: %d)", depth))
		}
	}

	// Check denied paths
	for _, denied := range pg.DeniedPaths {
		if pg.isSubpath(result.Resolved, denied) {
			result.IsDenied = true
			result.Reason = "path is in denied list"
			return result
		}
	}

	// Check allowed roots
	if len(pg.AllowedRoots) > 0 {
		allowed := false
		for _, root := range pg.AllowedRoots {
			if pg.isSubpath(result.Resolved, root) {
				allowed = true
				break
			}
		}
		if !allowed {
			result.IsAllowed = false
			result.Reason = "path is outside allowed directories"
			return result
		}
	}

	result.IsAllowed = true
	return result
}

// hasTraversalAttempt checks if a path contains traversal sequences.
func (pg *PathGuard) hasTraversalAttempt(path string) bool {
	// Check for various traversal patterns
	patterns := []string{
		"..",
		"..\\",
		"../",
		"%2e%2e",      // URL encoded ..
		"%2e%2e%2f",   // URL encoded ../
		"%2e%2e%5c",   // URL encoded ..\
		"..%2f",       // Mixed encoding
		"..%5c",       // Mixed encoding
		"%252e%252e",  // Double URL encoded
		"....//",      // Double dot with extra slashes
		"..../",       // Extra dots
	}

	lowerPath := strings.ToLower(path)
	for _, pattern := range patterns {
		if strings.Contains(lowerPath, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check for null byte injection
	if strings.Contains(path, "\x00") {
		return true
	}

	return false
}

// resolveSymlinks resolves symlinks and returns the real path.
func (pg *PathGuard) resolveSymlinks(path string) (string, int, error) {
	depth := 0
	currentPath := path

	for depth < pg.MaxSymlinkDepth {
		info, err := os.Lstat(currentPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Path doesn't exist yet, return the cleaned path
				return filepath.Clean(currentPath), depth, nil
			}
			return "", depth, err
		}

		if info.Mode()&os.ModeSymlink == 0 {
			// Not a symlink, we're done
			return currentPath, depth, nil
		}

		// Resolve the symlink
		target, err := os.Readlink(currentPath)
		if err != nil {
			return "", depth, err
		}

		// Handle relative symlinks
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(currentPath), target)
		}

		currentPath = filepath.Clean(target)
		depth++
	}

	return "", depth, fmt.Errorf("symlink depth exceeded maximum of %d", pg.MaxSymlinkDepth)
}

// isSubpath checks if path is within or equal to root.
func (pg *PathGuard) isSubpath(path, root string) bool {
	// Clean and normalize both paths
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)

	// Check if path starts with root
	if cleanPath == cleanRoot {
		return true
	}

	// Ensure root ends with separator for prefix matching
	if !strings.HasSuffix(cleanRoot, string(filepath.Separator)) {
		cleanRoot += string(filepath.Separator)
	}

	return strings.HasPrefix(cleanPath, cleanRoot)
}

// SafeJoin safely joins path components, preventing traversal.
func (pg *PathGuard) SafeJoin(base string, parts ...string) (string, error) {
	// Start with the base
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("cannot resolve base path: %w", err)
	}

	// Join all parts
	fullPath := absBase
	for _, part := range parts {
		// Check for traversal in each part
		if pg.hasTraversalAttempt(part) {
			return "", fmt.Errorf("path component contains traversal attempt: %s", part)
		}
		fullPath = filepath.Join(fullPath, part)
	}

	// Clean and verify
	cleanPath := filepath.Clean(fullPath)

	// Ensure the result is still within the base
	if !pg.isSubpath(cleanPath, absBase) {
		return "", fmt.Errorf("path escapes base directory")
	}

	return cleanPath, nil
}

// NormalizePath normalizes a path for safe comparison.
func (pg *PathGuard) NormalizePath(path string) (string, error) {
	// Convert to absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Clean the path
	cleanPath := filepath.Clean(absPath)

	// Resolve symlinks if enabled
	if pg.FollowSymlinks {
		realPath, _, err := pg.resolveSymlinks(cleanPath)
		if err != nil {
			return "", err
		}
		return realPath, nil
	}

	return cleanPath, nil
}

// IsHiddenFile checks if a file is hidden (starts with .).
func (pg *PathGuard) IsHiddenFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".")
}

// IsSensitiveFile checks if a file is potentially sensitive.
func (pg *PathGuard) IsSensitiveFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	// Sensitive file names
	sensitiveNames := map[string]bool{
		".env":              true,
		".env.local":        true,
		".env.production":   true,
		".env.development":  true,
		"secrets.yaml":      true,
		"secrets.yml":       true,
		"secrets.json":      true,
		"credentials.json":  true,
		"credentials.yaml":  true,
		"service-account.json": true,
		"private.key":       true,
		"private.pem":       true,
		"id_rsa":            true,
		"id_dsa":            true,
		"id_ecdsa":          true,
		"id_ed25519":        true,
		".htpasswd":         true,
		".netrc":            true,
		".pgpass":           true,
	}

	if sensitiveNames[base] {
		return true
	}

	// Sensitive extensions
	sensitiveExts := map[string]bool{
		".pem":      true,
		".key":      true,
		".p12":      true,
		".pfx":      true,
		".jks":      true,
		".keystore": true,
	}

	if sensitiveExts[ext] {
		return true
	}

	// Check for patterns in path
	lowerPath := strings.ToLower(path)
	sensitivePatterns := []string{
		"password",
		"secret",
		"credential",
		"private",
		"apikey",
		"api_key",
		"api-key",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

// AddAllowedRoot adds a new allowed root directory.
func (pg *PathGuard) AddAllowedRoot(root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	pg.AllowedRoots = append(pg.AllowedRoots, abs)
	return nil
}

// AddDeniedPath adds a new denied path.
func (pg *PathGuard) AddDeniedPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	pg.DeniedPaths = append(pg.DeniedPaths, abs)
	return nil
}
