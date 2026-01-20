package secrets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretType represents the type of secret detected
type SecretType string

const (
	SecretTypeAPIKey       SecretType = "api_key"
	SecretTypePrivateKey   SecretType = "private_key"
	SecretTypePassword     SecretType = "password"
	SecretTypeToken        SecretType = "token"
	SecretTypeCredential   SecretType = "credential"
	SecretTypeConnectionString SecretType = "connection_string"
	SecretTypeGeneric      SecretType = "generic"
)

// Finding represents a detected secret
type Finding struct {
	Type        SecretType `json:"type"`
	File        string     `json:"file"`
	Line        int        `json:"line"`
	Column      int        `json:"column"`
	Match       string     `json:"match"`       // The matched text (redacted)
	Description string     `json:"description"`
	Severity    Severity   `json:"severity"`
	Rule        string     `json:"rule"`
}

// Severity represents the severity of a finding
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Rule defines a secret detection rule
type Rule struct {
	ID          string
	Description string
	Type        SecretType
	Severity    Severity
	Pattern     *regexp.Regexp
	Keywords    []string // Keywords that must be present (case-insensitive)
	Entropy     float64  // Minimum entropy threshold (0 to disable)
}

// Detector detects secrets in files and content
type Detector struct {
	rules       []Rule
	allowedExts map[string]bool
	ignoredDirs map[string]bool
}

// NewDetector creates a new secrets detector with default rules
func NewDetector() *Detector {
	d := &Detector{
		allowedExts: map[string]bool{
			".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
			".rb": true, ".php": true, ".java": true, ".cs": true, ".rs": true,
			".yaml": true, ".yml": true, ".json": true, ".xml": true, ".toml": true,
			".env": true, ".ini": true, ".cfg": true, ".conf": true, ".config": true,
			".sh": true, ".bash": true, ".zsh": true, ".ps1": true, ".bat": true, ".cmd": true,
			".sql": true, ".md": true, ".txt": true, ".properties": true,
		},
		ignoredDirs: map[string]bool{
			".git": true, "node_modules": true, "vendor": true, ".venv": true,
			"__pycache__": true, "dist": true, "build": true, ".idea": true,
			".vscode": true, "target": true, "bin": true, "obj": true,
		},
	}
	d.rules = defaultRules()
	return d
}

// defaultRules returns the default set of secret detection rules
func defaultRules() []Rule {
	return []Rule{
		// AWS
		{
			ID:          "aws-access-key",
			Description: "AWS Access Key ID",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`\b(AKIA[0-9A-Z]{16})\b`),
		},
		{
			ID:          "aws-secret-key",
			Description: "AWS Secret Access Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`(?i)(aws)?_?secret_?(access)?_?key['":\s=]*['"]?([A-Za-z0-9/+=]{40})['"]?`),
		},
		// GitHub
		{
			ID:          "github-token",
			Description: "GitHub Personal Access Token",
			Type:        SecretTypeToken,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`\b(ghp_[A-Za-z0-9]{36})\b`),
		},
		{
			ID:          "github-oauth",
			Description: "GitHub OAuth Token",
			Type:        SecretTypeToken,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`\b(gho_[A-Za-z0-9]{36})\b`),
		},
		{
			ID:          "github-app-token",
			Description: "GitHub App Token",
			Type:        SecretTypeToken,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`\b(ghu_[A-Za-z0-9]{36})\b`),
		},
		{
			ID:          "github-refresh-token",
			Description: "GitHub Refresh Token",
			Type:        SecretTypeToken,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`\b(ghr_[A-Za-z0-9]{36})\b`),
		},
		// GitLab
		{
			ID:          "gitlab-token",
			Description: "GitLab Personal Access Token",
			Type:        SecretTypeToken,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`\b(glpat-[A-Za-z0-9\-]{20,})\b`),
		},
		// Slack
		{
			ID:          "slack-token",
			Description: "Slack Token",
			Type:        SecretTypeToken,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`\b(xox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*)\b`),
		},
		{
			ID:          "slack-webhook",
			Description: "Slack Webhook URL",
			Type:        SecretTypeToken,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`https://hooks\.slack\.com/services/T[A-Z0-9]+/B[A-Z0-9]+/[A-Za-z0-9]+`),
		},
		// Google
		{
			ID:          "google-api-key",
			Description: "Google API Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`\bAIza[0-9A-Za-z\-_]{35}\b`),
		},
		// Stripe
		{
			ID:          "stripe-secret-key",
			Description: "Stripe Secret Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`\b(sk_live_[0-9a-zA-Z]{24,})\b`),
		},
		{
			ID:          "stripe-restricted-key",
			Description: "Stripe Restricted Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`\b(rk_live_[0-9a-zA-Z]{24,})\b`),
		},
		// Private Keys
		{
			ID:          "private-key-rsa",
			Description: "RSA Private Key",
			Type:        SecretTypePrivateKey,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`-----BEGIN RSA PRIVATE KEY-----`),
		},
		{
			ID:          "private-key-openssh",
			Description: "OpenSSH Private Key",
			Type:        SecretTypePrivateKey,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`-----BEGIN OPENSSH PRIVATE KEY-----`),
		},
		{
			ID:          "private-key-ec",
			Description: "EC Private Key",
			Type:        SecretTypePrivateKey,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`-----BEGIN EC PRIVATE KEY-----`),
		},
		{
			ID:          "private-key-pgp",
			Description: "PGP Private Key",
			Type:        SecretTypePrivateKey,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`),
		},
		// Database connection strings
		{
			ID:          "postgres-connection",
			Description: "PostgreSQL Connection String with Password",
			Type:        SecretTypeConnectionString,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`postgres(ql)?://[^:]+:[^@]+@[^/]+`),
		},
		{
			ID:          "mysql-connection",
			Description: "MySQL Connection String with Password",
			Type:        SecretTypeConnectionString,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`mysql://[^:]+:[^@]+@[^/]+`),
		},
		{
			ID:          "mongodb-connection",
			Description: "MongoDB Connection String with Password",
			Type:        SecretTypeConnectionString,
			Severity:    SeverityCritical,
			Pattern:     regexp.MustCompile(`mongodb(\+srv)?://[^:]+:[^@]+@[^/]+`),
		},
		// Generic patterns
		{
			ID:          "generic-api-key",
			Description: "Generic API Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey)['":\s=]+['"]?([A-Za-z0-9_\-]{20,})['"]?`),
		},
		{
			ID:          "generic-secret",
			Description: "Generic Secret",
			Type:        SecretTypeGeneric,
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`(?i)(secret|client[_-]?secret)['":\s=]+['"]?([A-Za-z0-9_\-]{20,})['"]?`),
		},
		{
			ID:          "generic-password",
			Description: "Hardcoded Password",
			Type:        SecretTypePassword,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)['":\s=]+['"]?([^\s'"]{8,})['"]?`),
		},
		{
			ID:          "generic-token",
			Description: "Generic Token",
			Type:        SecretTypeToken,
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`(?i)(access[_-]?token|auth[_-]?token|bearer)['":\s=]+['"]?([A-Za-z0-9_\-\.]{20,})['"]?`),
		},
		// JWT
		{
			ID:          "jwt-token",
			Description: "JSON Web Token",
			Type:        SecretTypeToken,
			Severity:    SeverityMedium,
			Pattern:     regexp.MustCompile(`\beyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*\b`),
		},
		// Heroku
		{
			ID:          "heroku-api-key",
			Description: "Heroku API Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`(?i)heroku[_-]?api[_-]?key['":\s=]+['"]?([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})['"]?`),
		},
		// SendGrid
		{
			ID:          "sendgrid-api-key",
			Description: "SendGrid API Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`\bSG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}\b`),
		},
		// Twilio
		{
			ID:          "twilio-api-key",
			Description: "Twilio API Key",
			Type:        SecretTypeAPIKey,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`\bSK[0-9a-fA-F]{32}\b`),
		},
		// npm
		{
			ID:          "npm-token",
			Description: "NPM Access Token",
			Type:        SecretTypeToken,
			Severity:    SeverityHigh,
			Pattern:     regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`),
		},
	}
}

// ScanFile scans a single file for secrets
func (d *Detector) ScanFile(path string) ([]Finding, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var findings []Finding
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lineFindings := d.ScanLine(line, lineNum)
		for i := range lineFindings {
			lineFindings[i].File = path
		}
		findings = append(findings, lineFindings...)
	}

	if err := scanner.Err(); err != nil {
		return findings, fmt.Errorf("error reading file: %w", err)
	}

	return findings, nil
}

// ScanLine scans a single line for secrets
func (d *Detector) ScanLine(line string, lineNum int) []Finding {
	var findings []Finding

	for _, rule := range d.rules {
		matches := rule.Pattern.FindAllStringIndex(line, -1)
		for _, match := range matches {
			// Skip if it looks like it's in a comment or is a placeholder
			if isLikelyFalsePositive(line, match[0]) {
				continue
			}

			matchText := line[match[0]:match[1]]
			findings = append(findings, Finding{
				Type:        rule.Type,
				Line:        lineNum,
				Column:      match[0] + 1,
				Match:       redact(matchText),
				Description: rule.Description,
				Severity:    rule.Severity,
				Rule:        rule.ID,
			})
		}
	}

	return findings
}

// ScanContent scans content string for secrets
func (d *Detector) ScanContent(content string) []Finding {
	var findings []Finding
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineFindings := d.ScanLine(line, i+1)
		findings = append(findings, lineFindings...)
	}

	return findings
}

// ScanDirectory recursively scans a directory for secrets
func (d *Detector) ScanDirectory(root string) ([]Finding, error) {
	var findings []Finding

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip ignored directories
		if info.IsDir() {
			if d.ignoredDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if !d.allowedExts[ext] && ext != "" {
			return nil
		}

		// Skip large files (> 1MB)
		if info.Size() > 1024*1024 {
			return nil
		}

		fileFindings, err := d.ScanFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		findings = append(findings, fileFindings...)
		return nil
	})

	return findings, err
}

// isLikelyFalsePositive checks if a match is likely a false positive
func isLikelyFalsePositive(line string, matchStart int) bool {
	lineLower := strings.ToLower(line)

	// Check if it's in a comment
	commentPrefixes := []string{"//", "#", "--", "/*", "*", "<!--"}
	trimmed := strings.TrimSpace(line)
	for _, prefix := range commentPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			// Allow if it's a real secret, not documentation
			if strings.Contains(lineLower, "example") ||
				strings.Contains(lineLower, "placeholder") ||
				strings.Contains(lineLower, "your_") ||
				strings.Contains(lineLower, "xxx") ||
				strings.Contains(lineLower, "todo") {
				return true
			}
		}
	}

	// Check for placeholder patterns
	placeholders := []string{
		"example", "placeholder", "your_", "xxx", "<your", "${", "{{",
		"test_", "fake_", "dummy_", "sample_", "demo_",
	}
	for _, ph := range placeholders {
		if strings.Contains(lineLower, ph) {
			return true
		}
	}

	return false
}

// redact redacts a secret, showing only prefix and suffix
func redact(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// AddRule adds a custom rule to the detector
func (d *Detector) AddRule(rule Rule) {
	d.rules = append(d.rules, rule)
}

// SetAllowedExtensions sets the allowed file extensions
func (d *Detector) SetAllowedExtensions(exts []string) {
	d.allowedExts = make(map[string]bool)
	for _, ext := range exts {
		d.allowedExts[ext] = true
	}
}

// SetIgnoredDirs sets the ignored directories
func (d *Detector) SetIgnoredDirs(dirs []string) {
	d.ignoredDirs = make(map[string]bool)
	for _, dir := range dirs {
		d.ignoredDirs[dir] = true
	}
}

// HasCriticalFindings returns true if any finding is critical severity
func HasCriticalFindings(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// HasHighOrCriticalFindings returns true if any finding is high or critical
func HasHighOrCriticalFindings(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityCritical || f.Severity == SeverityHigh {
			return true
		}
	}
	return false
}

// FilterBySeverity filters findings by minimum severity
func FilterBySeverity(findings []Finding, minSeverity Severity) []Finding {
	severityOrder := map[Severity]int{
		SeverityLow:      1,
		SeverityMedium:   2,
		SeverityHigh:     3,
		SeverityCritical: 4,
	}

	minLevel := severityOrder[minSeverity]
	var filtered []Finding

	for _, f := range findings {
		if severityOrder[f.Severity] >= minLevel {
			filtered = append(filtered, f)
		}
	}

	return filtered
}

// GroupByFile groups findings by file
func GroupByFile(findings []Finding) map[string][]Finding {
	grouped := make(map[string][]Finding)
	for _, f := range findings {
		grouped[f.File] = append(grouped[f.File], f)
	}
	return grouped
}
