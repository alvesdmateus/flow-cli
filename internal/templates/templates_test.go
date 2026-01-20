package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetAllTemplates(t *testing.T) {
	templates := GetAllTemplates()

	// Should have at least the core templates
	if len(templates) < 5 {
		t.Errorf("expected at least 5 templates, got %d", len(templates))
	}

	// Check for expected templates
	expectedNames := []string{"go", "python", "node", "rust", "empty"}
	for _, name := range expectedNames {
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find template %q", name)
		}
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"go", true},
		{"python", true},
		{"node", true},
		{"rust", true},
		{"empty", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := GetTemplate(tt.name)
			if tt.expected && tmpl == nil {
				t.Errorf("expected to find template %q, got nil", tt.name)
			}
			if !tt.expected && tmpl != nil {
				t.Errorf("expected nil for template %q, got %+v", tt.name, tmpl)
			}
		})
	}
}

func TestTemplateStructure(t *testing.T) {
	for _, tmpl := range GetAllTemplates() {
		t.Run(tmpl.Name, func(t *testing.T) {
			// Name should not be empty
			if tmpl.Name == "" {
				t.Error("template name should not be empty")
			}

			// Description should not be empty
			if tmpl.Description == "" {
				t.Error("template description should not be empty")
			}

			// Language should not be empty
			if tmpl.Language == "" {
				t.Error("template language should not be empty")
			}

			// Should have at least one file
			if len(tmpl.Files) == 0 {
				t.Error("template should have at least one file")
			}

			// All templates should include flow config files
			hasFlowConfig := false
			hasFlowignore := false
			for _, f := range tmpl.Files {
				if strings.Contains(f.Path, ".flow/config.yaml") {
					hasFlowConfig = true
				}
				if strings.Contains(f.Path, ".flowignore") {
					hasFlowignore = true
				}
			}
			if !hasFlowConfig {
				t.Error("template should include .flow/config.yaml")
			}
			if !hasFlowignore {
				t.Error("template should include .flowignore")
			}
		})
	}
}

func TestScaffolderScaffold(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "flow-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := &ProjectInfo{
		Name:        "test-project",
		Description: "A test project",
		Author:      "Test Author",
		Path:        tmpDir,
	}

	scaffolder := NewScaffolder(tmpDir, info)

	// Test with empty template (minimal files)
	emptyTmpl := GetTemplate("empty")
	if emptyTmpl == nil {
		t.Fatal("empty template not found")
	}

	files, err := scaffolder.Scaffold(emptyTmpl)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	// Should have created files
	if len(files) == 0 {
		t.Error("expected files to be created")
	}

	// Check that files exist
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("file was not created: %s", f)
		}
	}

	// Check flow config exists
	flowConfigPath := filepath.Join(tmpDir, ".flow", "config.yaml")
	if _, err := os.Stat(flowConfigPath); os.IsNotExist(err) {
		t.Error(".flow/config.yaml was not created")
	}

	// Check flowignore exists
	flowignorePath := filepath.Join(tmpDir, ".flowignore")
	if _, err := os.Stat(flowignorePath); os.IsNotExist(err) {
		t.Error(".flowignore was not created")
	}
}

func TestScaffolderTemplateVariables(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flow-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := &ProjectInfo{
		Name:        "my-awesome-project",
		Description: "An awesome description",
		Author:      "John Doe",
		Path:        tmpDir,
	}

	scaffolder := NewScaffolder(tmpDir, info)

	goTmpl := GetTemplate("go")
	if goTmpl == nil {
		t.Fatal("go template not found")
	}

	_, err = scaffolder.Scaffold(goTmpl)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	// Check that template variables were replaced in go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	if !strings.Contains(string(content), "my-awesome-project") {
		t.Error("go.mod should contain project name")
	}

	// Check main.go
	mainGoPath := filepath.Join(tmpDir, "main.go")
	mainContent, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}

	if !strings.Contains(string(mainContent), "Hello, my-awesome-project") {
		t.Error("main.go should contain greeting with project name")
	}
}

func TestScaffolderGoTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flow-test-go-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := &ProjectInfo{
		Name:        "mygoproject",
		Description: "A Go project",
		Author:      "Developer",
		Path:        tmpDir,
	}

	scaffolder := NewScaffolder(tmpDir, info)
	goTmpl := GetTemplate("go")

	files, err := scaffolder.Scaffold(goTmpl)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	// Check expected files
	expectedFiles := []string{
		"go.mod",
		"main.go",
		"Makefile",
		".gitignore",
		"README.md",
		".flow/config.yaml",
		".flowignore",
	}

	for _, expected := range expectedFiles {
		found := false
		for _, f := range files {
			if strings.HasSuffix(f, expected) || strings.HasSuffix(f, filepath.FromSlash(expected)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file %s to be created", expected)
		}
	}
}

func TestScaffolderPythonTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flow-test-python-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := &ProjectInfo{
		Name:        "mypythonproject",
		Description: "A Python project",
		Author:      "Developer",
		Path:        tmpDir,
	}

	scaffolder := NewScaffolder(tmpDir, info)
	pythonTmpl := GetTemplate("python")

	files, err := scaffolder.Scaffold(pythonTmpl)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	// Should create pyproject.toml
	pyprojectPath := filepath.Join(tmpDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); os.IsNotExist(err) {
		t.Error("pyproject.toml was not created")
	}

	// Check pyproject.toml content
	content, err := os.ReadFile(pyprojectPath)
	if err != nil {
		t.Fatalf("failed to read pyproject.toml: %v", err)
	}

	if !strings.Contains(string(content), "mypythonproject") {
		t.Error("pyproject.toml should contain project name")
	}

	if len(files) == 0 {
		t.Error("expected files to be created")
	}
}

func TestScaffolderNodeTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flow-test-node-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := &ProjectInfo{
		Name:        "mynodeproject",
		Description: "A Node.js project",
		Author:      "Developer",
		Path:        tmpDir,
	}

	scaffolder := NewScaffolder(tmpDir, info)
	nodeTmpl := GetTemplate("node")

	files, err := scaffolder.Scaffold(nodeTmpl)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	// Should create package.json
	packagePath := filepath.Join(tmpDir, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Error("package.json was not created")
	}

	// Should create tsconfig.json
	tsconfigPath := filepath.Join(tmpDir, "tsconfig.json")
	if _, err := os.Stat(tsconfigPath); os.IsNotExist(err) {
		t.Error("tsconfig.json was not created")
	}

	if len(files) == 0 {
		t.Error("expected files to be created")
	}
}

func TestScaffolderRustTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flow-test-rust-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := &ProjectInfo{
		Name:        "myrustproject",
		Description: "A Rust project",
		Author:      "Developer",
		Path:        tmpDir,
	}

	scaffolder := NewScaffolder(tmpDir, info)
	rustTmpl := GetTemplate("rust")

	files, err := scaffolder.Scaffold(rustTmpl)
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	// Should create Cargo.toml
	cargoPath := filepath.Join(tmpDir, "Cargo.toml")
	if _, err := os.Stat(cargoPath); os.IsNotExist(err) {
		t.Error("Cargo.toml was not created")
	}

	// Should create src/main.rs
	mainPath := filepath.Join(tmpDir, "src", "main.rs")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Error("src/main.rs was not created")
	}

	if len(files) == 0 {
		t.Error("expected files to be created")
	}
}

func TestCreateFlowConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flow-test-config-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info := &ProjectInfo{
		Name:        "existing-project",
		Description: "An existing project",
		Path:        tmpDir,
	}

	scaffolder := NewScaffolder(tmpDir, info)

	files, err := scaffolder.CreateFlowConfig()
	if err != nil {
		t.Fatalf("CreateFlowConfig failed: %v", err)
	}

	// Should only create flow config files
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}

	// Check .flow/config.yaml exists
	flowConfigPath := filepath.Join(tmpDir, ".flow", "config.yaml")
	if _, err := os.Stat(flowConfigPath); os.IsNotExist(err) {
		t.Error(".flow/config.yaml was not created")
	}

	// Check .flowignore exists
	flowignorePath := filepath.Join(tmpDir, ".flowignore")
	if _, err := os.Stat(flowignorePath); os.IsNotExist(err) {
		t.Error(".flowignore was not created")
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple/path", "simple" + string(filepath.Separator) + "path"},
		{"path/with/multiple/parts", "path" + string(filepath.Separator) + "with" + string(filepath.Separator) + "multiple" + string(filepath.Separator) + "parts"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePath(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
