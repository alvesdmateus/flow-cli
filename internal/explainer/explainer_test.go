package explainer

import (
	"testing"
)

func TestNewExplainer(t *testing.T) {
	cfg := ExplainerConfig{
		LLMClient: nil, // Would be a mock in real tests
		Model:     "test-model",
		Detailed:  true,
	}

	exp := NewExplainer(cfg)

	if exp == nil {
		t.Fatal("expected explainer to be created")
	}
	if exp.model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", exp.model)
	}
	if !exp.detailed {
		t.Error("expected detailed to be true")
	}
}

func TestLanguageFromExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".go", "go"},
		{"go", "go"},
		{".py", "python"},
		{".js", "javascript"},
		{".ts", "typescript"},
		{".jsx", "jsx"},
		{".tsx", "tsx"},
		{".rs", "rust"},
		{".rb", "ruby"},
		{".java", "java"},
		{".c", "c"},
		{".cpp", "cpp"},
		{".cc", "cpp"},
		{".cxx", "cpp"},
		{".h", "cpp"},
		{".hpp", "cpp"},
		{".cs", "csharp"},
		{".php", "php"},
		{".swift", "swift"},
		{".kt", "kotlin"},
		{".scala", "scala"},
		{".sh", "bash"},
		{".bash", "bash"},
		{".sql", "sql"},
		{".html", "html"},
		{".css", "css"},
		{".json", "json"},
		{".yaml", "yaml"},
		{".yml", "yaml"},
		{".xml", "xml"},
		{".md", "markdown"},
		{".unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := languageFromExt(tt.ext)
			if result != tt.expected {
				t.Errorf("languageFromExt(%q) = %q, expected %q", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestExtractFunction_Go(t *testing.T) {
	code := `package main

func helper() {
	// helper function
}

func main() {
	x := 1
	fmt.Println(x)
}

func another() {
	// another function
}`

	result := extractFunction(code, "main", "go")

	if result == "" {
		t.Fatal("expected function to be extracted")
	}
	if result != "func main() {\n\tx := 1\n\tfmt.Println(x)\n}" {
		t.Errorf("unexpected extraction: %q", result)
	}
}

func TestExtractFunction_Python(t *testing.T) {
	code := `def helper():
    pass

def main():
    x = 1
    print(x)

def another():
    pass`

	result := extractFunction(code, "main", "python")

	if result == "" {
		t.Fatal("expected function to be extracted")
	}
	// Python extraction may include trailing newline
	expected := "def main():\n    x = 1\n    print(x)"
	if result != expected && result != expected+"\n" {
		t.Errorf("unexpected extraction: %q", result)
	}
}

func TestExtractFunction_JavaScript(t *testing.T) {
	code := `function helper() {
	return 1;
}

function main() {
	const x = 1;
	console.log(x);
}

const arrow = () => {};`

	result := extractFunction(code, "main", "javascript")

	if result == "" {
		t.Fatal("expected function to be extracted")
	}
	if result != "function main() {\n\tconst x = 1;\n\tconsole.log(x);\n}" {
		t.Errorf("unexpected extraction: %q", result)
	}
}

func TestExtractFunction_NotFound(t *testing.T) {
	code := `func main() {
	fmt.Println("hello")
}`

	result := extractFunction(code, "nonexistent", "go")

	if result != "" {
		t.Errorf("expected empty string for non-existent function, got %q", result)
	}
}

func TestExtractFunction_Rust(t *testing.T) {
	code := `fn helper() {
    println!("helper");
}

pub fn main() {
    let x = 1;
    println!("{}", x);
}

fn another() {
    // nothing
}`

	result := extractFunction(code, "main", "rust")

	if result == "" {
		t.Fatal("expected function to be extracted")
	}
	if result != "pub fn main() {\n    let x = 1;\n    println!(\"{}\", x);\n}" {
		t.Errorf("unexpected extraction: %q", result)
	}
}

func TestExtractFunctionBody_EmptyBraces(t *testing.T) {
	lines := []string{
		"func test() {",
		"}",
	}

	result := extractFunctionBody(lines, 0, "go")

	expected := "func test() {\n}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractPythonFunction_WithEmptyLines(t *testing.T) {
	lines := []string{
		"def test():",
		"    x = 1",
		"",
		"    y = 2",
		"    return x + y",
		"",
		"def next():",
		"    pass",
	}

	result := extractPythonFunction(lines, 0)

	if result == "" {
		t.Fatal("expected function to be extracted")
	}
	// Should include empty lines within function
	if result != "def test():\n    x = 1\n\n    y = 2\n    return x + y\n" {
		t.Errorf("unexpected extraction: %q", result)
	}
}

func TestExplainerConfig(t *testing.T) {
	cfg := ExplainerConfig{
		LLMClient: nil,
		Model:     "gpt-4",
		Detailed:  false,
	}

	if cfg.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got '%s'", cfg.Model)
	}
	if cfg.Detailed != false {
		t.Error("expected Detailed to be false")
	}
}
