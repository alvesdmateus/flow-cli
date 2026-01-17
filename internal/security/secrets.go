package security

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretType represents the type of secret detected.
type SecretType string

const (
	SecretAPIKey       SecretType = "api_key"
	SecretPassword     SecretType = "password"
	SecretPrivateKey   SecretType = "private_key"
	SecretToken        SecretType = "token"
	SecretCredential   SecretType = "credential"
	SecretConnection   SecretType = "connection_string"
	SecretCertificate  SecretType = "certificate"
	SecretGeneric      SecretType = "generic"
)

// SecretFinding represents a detected secret.
type SecretFinding struct {
	Type        SecretType
	Description string
	File        string
	Line        int
	Column      int
	Match       string    // The matched text (redacted)
	Confidence  float64   // 0.0 to 1.0
	Pattern     string    // Name of the pattern that matched
}

// SecretScanner scans files for potential secrets.
type SecretScanner struct {
	patterns    []secretPattern
	ignorePaths []string
	maxFileSize int64
}

type secretPattern struct {
	name        string
	secretType  SecretType
	regex       *regexp.Regexp
	description string
	confidence  float64
}

// NewSecretScanner creates a new secret scanner with default patterns.
func NewSecretScanner() *SecretScanner {
	ss := &SecretScanner{
		patterns:    make([]secretPattern, 0),
		maxFileSize: 10 * 1024 * 1024, // 10MB default
		ignorePaths: []string{
			".git",
			"node_modules",
			"vendor",
			"__pycache__",
			".venv",
			"venv",
			".idea",
			".vscode",
		},
	}

	ss.addDefaultPatterns()
	return ss
}

func (ss *SecretScanner) addDefaultPatterns() {
	patterns := []struct {
		name        string
		secretType  SecretType
		pattern     string
		description string
		confidence  float64
	}{
		// AWS
		{
			name:        "aws_access_key",
			secretType:  SecretAPIKey,
			pattern:     `(?i)(aws_access_key_id|aws_access_key|aws_key_id|aws_key)\s*[=:]\s*['"]?([A-Z0-9]{20})['"]?`,
			description: "AWS Access Key ID",
			confidence:  0.9,
		},
		{
			name:        "aws_secret_key",
			secretType:  SecretAPIKey,
			pattern:     `(?i)(aws_secret_access_key|aws_secret_key|aws_secret)\s*[=:]\s*['"]?([A-Za-z0-9/+=]{40})['"]?`,
			description: "AWS Secret Access Key",
			confidence:  0.9,
		},
		{
			name:        "aws_key_pattern",
			secretType:  SecretAPIKey,
			pattern:     `AKIA[0-9A-Z]{16}`,
			description: "AWS Access Key ID pattern",
			confidence:  0.95,
		},

		// GitHub
		{
			name:        "github_token",
			secretType:  SecretToken,
			pattern:     `ghp_[a-zA-Z0-9]{36}`,
			description: "GitHub Personal Access Token",
			confidence:  0.95,
		},
		{
			name:        "github_oauth",
			secretType:  SecretToken,
			pattern:     `gho_[a-zA-Z0-9]{36}`,
			description: "GitHub OAuth Token",
			confidence:  0.95,
		},
		{
			name:        "github_app",
			secretType:  SecretToken,
			pattern:     `(?:ghs|ghr)_[a-zA-Z0-9]{36}`,
			description: "GitHub App Token",
			confidence:  0.95,
		},

		// GitLab
		{
			name:        "gitlab_token",
			secretType:  SecretToken,
			pattern:     `glpat-[a-zA-Z0-9\-]{20}`,
			description: "GitLab Personal Access Token",
			confidence:  0.95,
		},

		// Slack
		{
			name:        "slack_token",
			secretType:  SecretToken,
			pattern:     `xox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`,
			description: "Slack Token",
			confidence:  0.9,
		},
		{
			name:        "slack_webhook",
			secretType:  SecretToken,
			pattern:     `https://hooks\.slack\.com/services/T[a-zA-Z0-9_]+/B[a-zA-Z0-9_]+/[a-zA-Z0-9_]+`,
			description: "Slack Webhook URL",
			confidence:  0.95,
		},

		// Google
		{
			name:        "google_api_key",
			secretType:  SecretAPIKey,
			pattern:     `AIza[0-9A-Za-z\-_]{35}`,
			description: "Google API Key",
			confidence:  0.9,
		},
		{
			name:        "google_oauth_id",
			secretType:  SecretCredential,
			pattern:     `[0-9]+-[a-z0-9_]{32}\.apps\.googleusercontent\.com`,
			description: "Google OAuth Client ID",
			confidence:  0.85,
		},

		// Stripe
		{
			name:        "stripe_key",
			secretType:  SecretAPIKey,
			pattern:     `(?:sk|pk)_(?:test|live)_[a-zA-Z0-9]{24,}`,
			description: "Stripe API Key",
			confidence:  0.95,
		},

		// Twilio
		{
			name:        "twilio_key",
			secretType:  SecretAPIKey,
			pattern:     `SK[a-fA-F0-9]{32}`,
			description: "Twilio API Key",
			confidence:  0.85,
		},

		// Private Keys
		{
			name:        "private_key_rsa",
			secretType:  SecretPrivateKey,
			pattern:     `-----BEGIN RSA PRIVATE KEY-----`,
			description: "RSA Private Key",
			confidence:  0.99,
		},
		{
			name:        "private_key_openssh",
			secretType:  SecretPrivateKey,
			pattern:     `-----BEGIN OPENSSH PRIVATE KEY-----`,
			description: "OpenSSH Private Key",
			confidence:  0.99,
		},
		{
			name:        "private_key_ec",
			secretType:  SecretPrivateKey,
			pattern:     `-----BEGIN EC PRIVATE KEY-----`,
			description: "EC Private Key",
			confidence:  0.99,
		},
		{
			name:        "private_key_pgp",
			secretType:  SecretPrivateKey,
			pattern:     `-----BEGIN PGP PRIVATE KEY BLOCK-----`,
			description: "PGP Private Key",
			confidence:  0.99,
		},
		{
			name:        "private_key_generic",
			secretType:  SecretPrivateKey,
			pattern:     `-----BEGIN (?:ENCRYPTED )?PRIVATE KEY-----`,
			description: "Private Key",
			confidence:  0.99,
		},

		// Database
		{
			name:        "postgres_uri",
			secretType:  SecretConnection,
			pattern:     `postgres(?:ql)?://[^\s'"]+:[^\s'"]+@[^\s'"]+`,
			description: "PostgreSQL Connection String",
			confidence:  0.9,
		},
		{
			name:        "mysql_uri",
			secretType:  SecretConnection,
			pattern:     `mysql://[^\s'"]+:[^\s'"]+@[^\s'"]+`,
			description: "MySQL Connection String",
			confidence:  0.9,
		},
		{
			name:        "mongodb_uri",
			secretType:  SecretConnection,
			pattern:     `mongodb(?:\+srv)?://[^\s'"]+:[^\s'"]+@[^\s'"]+`,
			description: "MongoDB Connection String",
			confidence:  0.9,
		},
		{
			name:        "redis_uri",
			secretType:  SecretConnection,
			pattern:     `redis://[^\s'"]*:[^\s'"]+@[^\s'"]+`,
			description: "Redis Connection String",
			confidence:  0.9,
		},

		// Generic Patterns
		{
			name:        "generic_api_key",
			secretType:  SecretAPIKey,
			pattern:     `(?i)(?:api[_-]?key|apikey)\s*[=:]\s*['"]?([a-zA-Z0-9_\-]{20,})['"]?`,
			description: "Generic API Key",
			confidence:  0.7,
		},
		{
			name:        "generic_secret",
			secretType:  SecretGeneric,
			pattern:     `(?i)(?:secret|secret[_-]?key)\s*[=:]\s*['"]?([a-zA-Z0-9_\-]{16,})['"]?`,
			description: "Generic Secret",
			confidence:  0.6,
		},
		{
			name:        "generic_password",
			secretType:  SecretPassword,
			pattern:     `(?i)(?:password|passwd|pwd)\s*[=:]\s*['"]?([^\s'"]{8,})['"]?`,
			description: "Password",
			confidence:  0.6,
		},
		{
			name:        "generic_token",
			secretType:  SecretToken,
			pattern:     `(?i)(?:auth[_-]?token|access[_-]?token|bearer)\s*[=:]\s*['"]?([a-zA-Z0-9_\-\.]{20,})['"]?`,
			description: "Generic Token",
			confidence:  0.7,
		},

		// JWT
		{
			name:        "jwt_token",
			secretType:  SecretToken,
			pattern:     `eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`,
			description: "JWT Token",
			confidence:  0.85,
		},

		// NPM
		{
			name:        "npm_token",
			secretType:  SecretToken,
			pattern:     `//registry\.npmjs\.org/:_authToken=([a-zA-Z0-9\-_]+)`,
			description: "NPM Token",
			confidence:  0.95,
		},

		// SendGrid
		{
			name:        "sendgrid_key",
			secretType:  SecretAPIKey,
			pattern:     `SG\.[a-zA-Z0-9_\-]{22}\.[a-zA-Z0-9_\-]{43}`,
			description: "SendGrid API Key",
			confidence:  0.95,
		},

		// Mailgun
		{
			name:        "mailgun_key",
			secretType:  SecretAPIKey,
			pattern:     `key-[a-zA-Z0-9]{32}`,
			description: "Mailgun API Key",
			confidence:  0.85,
		},

		// Heroku
		{
			name:        "heroku_key",
			secretType:  SecretAPIKey,
			pattern:     `[hH]eroku[a-zA-Z0-9_\-]*[aA][pP][iI][_\-]?[kK][eE][yY][^\w]*[=:][^\w]*['"]?([a-fA-F0-9\-]{36})['"]?`,
			description: "Heroku API Key",
			confidence:  0.8,
		},

		// High entropy strings (potential secrets)
		{
			name:        "high_entropy_hex",
			secretType:  SecretGeneric,
			pattern:     `['"=:]\s*([a-fA-F0-9]{32,64})\s*['"]?`,
			description: "High entropy hex string",
			confidence:  0.5,
		},
	}

	for _, p := range patterns {
		re, err := regexp.Compile(p.pattern)
		if err != nil {
			continue
		}
		ss.patterns = append(ss.patterns, secretPattern{
			name:        p.name,
			secretType:  p.secretType,
			regex:       re,
			description: p.description,
			confidence:  p.confidence,
		})
	}
}

// ScanFile scans a single file for secrets.
func (ss *SecretScanner) ScanFile(path string) ([]SecretFinding, error) {
	// Check file size
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.Size() > ss.maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes", info.Size())
	}

	// Skip binary files
	if ss.isBinaryFile(path) {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var findings []SecretFinding
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, pattern := range ss.patterns {
			matches := pattern.regex.FindAllStringIndex(line, -1)
			for _, match := range matches {
				// Redact the actual secret
				matchText := line[match[0]:match[1]]
				redacted := ss.redact(matchText)

				findings = append(findings, SecretFinding{
					Type:        pattern.secretType,
					Description: pattern.description,
					File:        path,
					Line:        lineNum,
					Column:      match[0] + 1,
					Match:       redacted,
					Confidence:  pattern.confidence,
					Pattern:     pattern.name,
				})
			}
		}
	}

	return findings, scanner.Err()
}

// ScanDirectory scans a directory recursively for secrets.
func (ss *SecretScanner) ScanDirectory(dir string) ([]SecretFinding, error) {
	var allFindings []SecretFinding

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip ignored directories
		if info.IsDir() {
			base := filepath.Base(path)
			for _, ignored := range ss.ignorePaths {
				if base == ignored {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Scan the file
		findings, err := ss.ScanFile(path)
		if err != nil {
			return nil // Skip files that fail to scan
		}

		allFindings = append(allFindings, findings...)
		return nil
	})

	return allFindings, err
}

// ScanContent scans content (string) for secrets.
func (ss *SecretScanner) ScanContent(content string, filename string) []SecretFinding {
	var findings []SecretFinding
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for _, pattern := range ss.patterns {
			matches := pattern.regex.FindAllStringIndex(line, -1)
			for _, match := range matches {
				matchText := line[match[0]:match[1]]
				redacted := ss.redact(matchText)

				findings = append(findings, SecretFinding{
					Type:        pattern.secretType,
					Description: pattern.description,
					File:        filename,
					Line:        lineNum + 1,
					Column:      match[0] + 1,
					Match:       redacted,
					Confidence:  pattern.confidence,
					Pattern:     pattern.name,
				})
			}
		}
	}

	return findings
}

// isBinaryFile checks if a file appears to be binary.
func (ss *SecretScanner) isBinaryFile(path string) bool {
	// Check extension
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".ico": true, ".bmp": true, ".webp": true, ".svg": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mkv": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true,
		".7z": true, ".pdf": true, ".doc": true, ".docx": true,
		".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".wasm": true, ".pyc": true, ".class": true,
	}

	if binaryExts[ext] {
		return true
	}

	// Check first bytes for binary content
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	return false
}

// redact redacts a secret value for safe display.
func (ss *SecretScanner) redact(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}

	// Show first 4 and last 4 characters
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}

// AddIgnorePath adds a path to ignore during scanning.
func (ss *SecretScanner) AddIgnorePath(path string) {
	ss.ignorePaths = append(ss.ignorePaths, path)
}

// SetMaxFileSize sets the maximum file size to scan.
func (ss *SecretScanner) SetMaxFileSize(size int64) {
	ss.maxFileSize = size
}

// FilterByConfidence filters findings by minimum confidence.
func FilterByConfidence(findings []SecretFinding, minConfidence float64) []SecretFinding {
	var filtered []SecretFinding
	for _, f := range findings {
		if f.Confidence >= minConfidence {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// FilterByType filters findings by secret type.
func FilterByType(findings []SecretFinding, secretType SecretType) []SecretFinding {
	var filtered []SecretFinding
	for _, f := range findings {
		if f.Type == secretType {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
