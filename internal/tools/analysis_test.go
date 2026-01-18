package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeOutlineTool_Execute(t *testing.T) {
	// Create temp directory with test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main

import "fmt"

// Person represents a person
type Person struct {
	Name string
	Age  int
}

// NewPerson creates a new Person
func NewPerson(name string, age int) *Person {
	return &Person{Name: name, Age: age}
}

func main() {
	p := NewPerson("Alice", 30)
	fmt.Println(p.Name)
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewCodeOutlineTool(tmpDir)

	// Test basic info
	if tool.Name() != "code_outline" {
		t.Errorf("Expected name 'code_outline', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}

	// Test execution with relative path
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "test.go",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Check output contains expected elements
	output := result.Output
	if !strings.Contains(output, "Language: go") {
		t.Error("Output should contain language")
	}
	if !strings.Contains(output, "Package: main") {
		t.Error("Output should contain package")
	}
	if !strings.Contains(output, "Person") {
		t.Error("Output should contain Person struct")
	}
	if !strings.Contains(output, "NewPerson") {
		t.Error("Output should contain NewPerson function")
	}

	// Test execution with absolute path
	result, err = tool.Execute(context.Background(), map[string]any{
		"path": testFile,
	})

	if err != nil {
		t.Fatalf("Execute with absolute path failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success with absolute path")
	}
}

func TestCodeOutlineTool_Execute_MissingPath(t *testing.T) {
	tool := NewCodeOutlineTool(".")

	result, err := tool.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing path")
	}

	if !strings.Contains(result.Error, "path") {
		t.Error("Error should mention missing path")
	}
}

func TestCodeOutlineTool_Execute_UnsupportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.xyz")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewCodeOutlineTool(tmpDir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "test.xyz",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for unsupported file type")
	}
}

func TestFindDefinitionTool_Execute(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create first file
	file1 := filepath.Join(tmpDir, "person.go")
	content1 := `package main

// Person represents a person
type Person struct {
	Name string
}

// NewPerson creates a new Person
func NewPerson(name string) *Person {
	return &Person{Name: name}
}
`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create second file
	file2 := filepath.Join(tmpDir, "utils.go")
	content2 := `package main

// Helper is a helper type
type Helper struct{}

// NewHelper creates a new Helper
func NewHelper() *Helper {
	return &Helper{}
}
`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewFindDefinitionTool(tmpDir)

	// Test basic info
	if tool.Name() != "find_definition" {
		t.Errorf("Expected name 'find_definition', got '%s'", tool.Name())
	}

	// Test finding Person
	result, err := tool.Execute(context.Background(), map[string]any{
		"symbol": "Person",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Output, "Person") {
		t.Error("Output should contain Person")
	}
	if !strings.Contains(result.Output, "person.go") {
		t.Error("Output should contain filename")
	}

	// Test finding with kind filter
	result, err = tool.Execute(context.Background(), map[string]any{
		"symbol": "NewPerson",
		"kind":   "function",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success")
	}
	if !strings.Contains(result.Output, "function") {
		t.Error("Output should show kind as function")
	}

	// Test finding non-existent symbol
	result, err = tool.Execute(context.Background(), map[string]any{
		"symbol": "NonExistent",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success even for no results")
	}
	if !strings.Contains(result.Output, "No definition found") {
		t.Error("Output should indicate no definition found")
	}
}

func TestFindDefinitionTool_Execute_MissingSymbol(t *testing.T) {
	tool := NewFindDefinitionTool(".")

	result, err := tool.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing symbol")
	}
}

func TestFindReferencesTool_Execute(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "main.go")
	content1 := `package main

func main() {
	p := NewPerson("Alice")
	fmt.Println(p.Name)
}
`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	file2 := filepath.Join(tmpDir, "person.go")
	content2 := `package main

type Person struct {
	Name string
}

func NewPerson(name string) *Person {
	return &Person{Name: name}
}

func (p *Person) Greet() string {
	return "Hello, " + p.Name
}
`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewFindReferencesTool(tmpDir)

	// Test basic info
	if tool.Name() != "find_references" {
		t.Errorf("Expected name 'find_references', got '%s'", tool.Name())
	}

	// Test finding references to Person
	result, err := tool.Execute(context.Background(), map[string]any{
		"symbol": "Person",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Should find multiple references
	if !strings.Contains(result.Output, "reference") {
		t.Error("Output should mention references")
	}

	// Test finding references with extension filter
	result, err = tool.Execute(context.Background(), map[string]any{
		"symbol":     "Name",
		"extensions": ".go",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success with extension filter")
	}

	// Test finding non-existent symbol
	result, err = tool.Execute(context.Background(), map[string]any{
		"symbol": "NonExistentSymbol12345",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "No references found") {
		t.Error("Output should indicate no references found")
	}
}

func TestListSymbolsTool_Execute(t *testing.T) {
	// Create temp directory with test file
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

type PublicType struct{}
type privateType struct{}

func PublicFunc() {}
func privateFunc() {}

const PublicConst = 1
const privateConst = 2

var PublicVar = "public"
var privateVar = "private"
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewListSymbolsTool(tmpDir)

	// Test basic info
	if tool.Name() != "list_symbols" {
		t.Errorf("Expected name 'list_symbols', got '%s'", tool.Name())
	}

	// Test listing all symbols
	result, err := tool.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Should find multiple symbols
	if !strings.Contains(result.Output, "PublicType") {
		t.Error("Output should contain PublicType")
	}
	if !strings.Contains(result.Output, "PublicFunc") {
		t.Error("Output should contain PublicFunc")
	}

	// Test filtering by kind
	result, err = tool.Execute(context.Background(), map[string]any{
		"kind": "function",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Func") {
		t.Error("Output should contain functions")
	}
	// Should not contain non-function symbols
	if strings.Contains(result.Output, "struct PublicType") {
		t.Error("Output should not contain structs when filtering by function")
	}

	// Test filtering by exported only
	result, err = tool.Execute(context.Background(), map[string]any{
		"exported_only": true,
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "PublicType") {
		t.Error("Output should contain PublicType")
	}
	if strings.Contains(result.Output, "privateType") {
		t.Error("Output should not contain privateType when filtering exported only")
	}
}

func TestCollectSourceFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various files
	goFile := filepath.Join(tmpDir, "main.go")
	pyFile := filepath.Join(tmpDir, "script.py")
	jsFile := filepath.Join(tmpDir, "app.js")
	txtFile := filepath.Join(tmpDir, "readme.txt")

	for _, f := range []string{goFile, pyFile, jsFile, txtFile} {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Create subdirectory with file
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	subFile := filepath.Join(subDir, "lib.ts")
	if err := os.WriteFile(subFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create node_modules (should be skipped)
	nodeDir := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(nodeDir, 0755); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}
	nodeFile := filepath.Join(nodeDir, "package.js")
	if err := os.WriteFile(nodeFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	files, err := collectSourceFiles(tmpDir)
	if err != nil {
		t.Fatalf("collectSourceFiles failed: %v", err)
	}

	// Should find go, py, js, ts but not txt or node_modules files
	expectedCount := 4
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d: %v", expectedCount, len(files), files)
	}

	// Verify txt file is not included
	for _, f := range files {
		if strings.HasSuffix(f, ".txt") {
			t.Error("Should not include .txt files")
		}
		if strings.Contains(f, "node_modules") {
			t.Error("Should not include files from node_modules")
		}
	}
}

func TestCollectSourceFilesWithExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "main.go")
	pyFile := filepath.Join(tmpDir, "script.py")

	for _, f := range []string{goFile, pyFile} {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Test with specific extension
	files, err := collectSourceFilesWithExtensions(tmpDir, []string{".go"})
	if err != nil {
		t.Fatalf("collectSourceFilesWithExtensions failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if !strings.HasSuffix(files[0], ".go") {
		t.Error("Should only include .go files")
	}

	// Test with extension without dot
	files, err = collectSourceFilesWithExtensions(tmpDir, []string{"py"})
	if err != nil {
		t.Fatalf("collectSourceFilesWithExtensions failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

func TestFormatOutline(t *testing.T) {
	// Verify formatOutline doesn't panic with various inputs
	// This is a basic smoke test

	// Create a test file to parse
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main

func Hello() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewCodeOutlineTool(tmpDir)
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "test.go",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Output == "" {
		t.Error("formatOutline should return non-empty string")
	}
}

func TestTruncateDoc(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."},
		{"with\nnewline", 20, "with newline"},
		{"  trimmed  ", 20, "trimmed"},
	}

	for _, tt := range tests {
		result := truncateDoc(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateDoc(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}
