package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetector_ScanLine(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name     string
		line     string
		wantType SecretType
		wantRule string
	}{
		{
			name:     "AWS Access Key",
			line:     `aws_access_key = "AKIAIOSFODNN7REALKEY"`,
			wantType: SecretTypeAPIKey,
			wantRule: "aws-access-key",
		},
		{
			name:     "GitHub PAT",
			line:     `token := "ghp_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8"`,
			wantType: SecretTypeToken,
			wantRule: "github-token",
		},
		{
			name:     "GitLab PAT",
			line:     `GITLAB_TOKEN=glpat-A1B2C3D4E5F6G7H8I9J0`,
			wantType: SecretTypeToken,
			wantRule: "gitlab-token",
		},
		{
			name:     "Stripe Secret Key",
			line:     `stripe.Key = "sk_live_A1B2C3D4E5F6G7H8I9J0K1L2"`,
			wantType: SecretTypeAPIKey,
			wantRule: "stripe-secret-key",
		},
		{
			name:     "RSA Private Key",
			line:     `-----BEGIN RSA PRIVATE KEY-----`,
			wantType: SecretTypePrivateKey,
			wantRule: "private-key-rsa",
		},
		{
			name:     "OpenSSH Private Key",
			line:     `-----BEGIN OPENSSH PRIVATE KEY-----`,
			wantType: SecretTypePrivateKey,
			wantRule: "private-key-openssh",
		},
		{
			name:     "PostgreSQL Connection",
			line:     `DATABASE_URL=postgres://admin:s3cr3tP4ss@localhost:5432/db`,
			wantType: SecretTypeConnectionString,
			wantRule: "postgres-connection",
		},
		{
			name:     "MongoDB Connection",
			line:     `MONGO_URI="mongodb+srv://admin:s3cr3tP4ss@cluster.mongodb.net"`,
			wantType: SecretTypeConnectionString,
			wantRule: "mongodb-connection",
		},
		{
			name:     "Google API Key",
			line:     `apiKey: "AIzaSyC1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q"`,
			wantType: SecretTypeAPIKey,
			wantRule: "google-api-key",
		},
		{
			name:     "Slack Token",
			line:     `SLACK_TOKEN=xoxb-1234567890-1234567890123-A1B2C3D4E5F6G7H8`,
			wantType: SecretTypeToken,
			wantRule: "slack-token",
		},
		{
			name:     "SendGrid API Key",
			line:     `sendgrid_key = "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"`,
			wantType: SecretTypeAPIKey,
			wantRule: "sendgrid-api-key",
		},
		{
			name:     "NPM Token",
			line:     `//registry.npmjs.org/:_authToken=npm_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8`,
			wantType: SecretTypeToken,
			wantRule: "npm-token",
		},
		{
			name:     "JWT Token",
			line:     `token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`,
			wantType: SecretTypeToken,
			wantRule: "jwt-token",
		},
		{
			name:     "Generic API Key",
			line:     `api_key = "A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6"`,
			wantType: SecretTypeAPIKey,
			wantRule: "generic-api-key",
		},
		{
			name:     "Generic Password",
			line:     `password = "SuperS3cr3tP4ssw0rd"`,
			wantType: SecretTypePassword,
			wantRule: "generic-password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.ScanLine(tt.line, 1)
			if len(findings) == 0 {
				t.Errorf("Expected to find secret, got none")
				return
			}

			found := false
			for _, f := range findings {
				if f.Rule == tt.wantRule {
					found = true
					if f.Type != tt.wantType {
						t.Errorf("Expected type %s, got %s", tt.wantType, f.Type)
					}
					break
				}
			}

			if !found {
				t.Errorf("Expected rule %s to match", tt.wantRule)
			}
		})
	}
}

func TestDetector_NoFalsePositives(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name string
		line string
	}{
		{
			name: "Example placeholder",
			line: `api_key = "your_api_key_here"`,
		},
		{
			name: "Environment variable reference",
			line: `api_key = "${API_KEY}"`,
		},
		{
			name: "Template variable",
			line: `api_key = "{{.ApiKey}}"`,
		},
		{
			name: "Test/fake key",
			line: `test_api_key = "fake_key_for_testing_12345"`,
		},
		{
			name: "Comment with example",
			line: `// Example: api_key = "sk_live_example123456789012345"`,
		},
		{
			name: "Placeholder in docs",
			line: `# Set your api_key = "<your-api-key-here>"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := d.ScanLine(tt.line, 1)
			if len(findings) > 0 {
				t.Errorf("Expected no findings (false positive), got %d: %v", len(findings), findings)
			}
		})
	}
}

func TestDetector_ScanContent(t *testing.T) {
	d := NewDetector()

	content := `package main

const (
	// This is fine
	appName = "myapp"

	// This is NOT fine - hardcoded secret
	apiKey = "ghp_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8"

	// Another secret
	dbURL = "postgres://admin:s3cr3tP4ss@db.internal.com:5432/app"
)
`

	findings := d.ScanContent(content)

	if len(findings) < 2 {
		t.Errorf("Expected at least 2 findings, got %d", len(findings))
	}

	// Check that we found the GitHub token
	foundGH := false
	for _, f := range findings {
		if f.Rule == "github-token" {
			foundGH = true
			if f.Line != 8 {
				t.Errorf("Expected GitHub token on line 8, got line %d", f.Line)
			}
		}
	}
	if !foundGH {
		t.Error("Expected to find GitHub token")
	}
}

func TestDetector_ScanFile(t *testing.T) {
	// Create a temp file with secrets
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config.go")

	content := `package config

var (
	APIKey = "AKIAIOSFODNN7REALKEY"
	DBUrl = "postgres://admin:s3cr3tP4ss@db.internal.com:5432/app"
)
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	d := NewDetector()
	findings, err := d.ScanFile(testFile)
	if err != nil {
		t.Fatalf("ScanFile failed: %v", err)
	}

	if len(findings) == 0 {
		t.Error("Expected to find secrets in file")
	}

	// Check file path is set
	for _, f := range findings {
		if f.File != testFile {
			t.Errorf("Expected file %s, got %s", testFile, f.File)
		}
	}
}

func TestDetector_ScanDirectory(t *testing.T) {
	// Create a temp directory structure with secrets
	tmpDir := t.TempDir()

	// Create a file with secrets
	secretFile := filepath.Join(tmpDir, "config.go")
	secretContent := `package config
var APIKey = "ghp_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8"
`
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("Failed to write secret file: %v", err)
	}

	// Create a clean file
	cleanFile := filepath.Join(tmpDir, "main.go")
	cleanContent := `package main
func main() {
	fmt.Println("Hello")
}
`
	if err := os.WriteFile(cleanFile, []byte(cleanContent), 0644); err != nil {
		t.Fatalf("Failed to write clean file: %v", err)
	}

	// Create an ignored directory
	ignoredDir := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(ignoredDir, 0755); err != nil {
		t.Fatalf("Failed to create ignored dir: %v", err)
	}
	ignoredFile := filepath.Join(ignoredDir, "secret.js")
	if err := os.WriteFile(ignoredFile, []byte(`var key = "ghp_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8"`), 0644); err != nil {
		t.Fatalf("Failed to write ignored file: %v", err)
	}

	d := NewDetector()
	findings, err := d.ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// Should find secret in config.go but not in node_modules
	if len(findings) == 0 {
		t.Error("Expected to find secrets")
	}

	for _, f := range findings {
		if filepath.Base(filepath.Dir(f.File)) == "node_modules" {
			t.Error("Should not scan node_modules directory")
		}
	}
}

func TestRedact(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234****6789"},
		{"ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", "ghp_****xxxx"},
	}

	for _, tt := range tests {
		result := redact(tt.input)
		if result != tt.expected {
			t.Errorf("redact(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestHasCriticalFindings(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityLow},
		{Severity: SeverityMedium},
	}

	if HasCriticalFindings(findings) {
		t.Error("Should not have critical findings")
	}

	findings = append(findings, Finding{Severity: SeverityCritical})
	if !HasCriticalFindings(findings) {
		t.Error("Should have critical findings")
	}
}

func TestFilterBySeverity(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityLow},
		{Severity: SeverityMedium},
		{Severity: SeverityHigh},
		{Severity: SeverityCritical},
	}

	filtered := FilterBySeverity(findings, SeverityHigh)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 findings, got %d", len(filtered))
	}

	for _, f := range filtered {
		if f.Severity != SeverityHigh && f.Severity != SeverityCritical {
			t.Errorf("Unexpected severity: %s", f.Severity)
		}
	}
}

func TestGroupByFile(t *testing.T) {
	findings := []Finding{
		{File: "a.go", Line: 1},
		{File: "a.go", Line: 2},
		{File: "b.go", Line: 1},
	}

	grouped := GroupByFile(findings)

	if len(grouped) != 2 {
		t.Errorf("Expected 2 files, got %d", len(grouped))
	}

	if len(grouped["a.go"]) != 2 {
		t.Errorf("Expected 2 findings for a.go, got %d", len(grouped["a.go"]))
	}

	if len(grouped["b.go"]) != 1 {
		t.Errorf("Expected 1 finding for b.go, got %d", len(grouped["b.go"]))
	}
}
