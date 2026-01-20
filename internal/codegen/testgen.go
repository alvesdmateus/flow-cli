package codegen

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// TestGenerator generates tests from implementation code.
type TestGenerator struct {
	templates map[string]*TestTemplate
	analyzers map[string]CodeAnalyzer
}

// TestTemplate defines a test template for a language.
type TestTemplate struct {
	Language     string
	FilePattern  string // e.g., "_test.go", ".test.js"
	TestFunc     string // Template for test function
	TestClass    string // Template for test class (if applicable)
	Imports      string // Required imports
	Setup        string // Setup code
	Teardown     string // Teardown code
	Assertion    string // Assertion template
	TableDriven  string // Table-driven test template
}

// CodeAnalyzer extracts information from source code.
type CodeAnalyzer interface {
	ExtractFunctions(code string) []FunctionInfo
	ExtractClasses(code string) []ClassInfo
	ExtractMethods(code string) []MethodInfo
}

// FunctionInfo contains information about a function.
type FunctionInfo struct {
	Name       string
	Params     []ParamInfo
	Returns    []string
	IsExported bool
	IsAsync    bool
	Comments   string
	StartLine  int
	EndLine    int
}

// ParamInfo contains information about a parameter.
type ParamInfo struct {
	Name string
	Type string
}

// ClassInfo contains information about a class.
type ClassInfo struct {
	Name       string
	Methods    []MethodInfo
	Properties []PropertyInfo
	IsExported bool
}

// MethodInfo contains information about a method.
type MethodInfo struct {
	Name       string
	Receiver   string
	Params     []ParamInfo
	Returns    []string
	IsExported bool
	IsStatic   bool
}

// PropertyInfo contains information about a property.
type PropertyInfo struct {
	Name string
	Type string
}

// TestCase represents a generated test case.
type TestCase struct {
	Name        string
	Description string
	Input       []string
	Expected    string
	Setup       string
	Assertion   string
}

// GeneratedTest represents a complete generated test.
type GeneratedTest struct {
	Language   string
	SourceFile string
	TestFile   string
	Imports    []string
	Setup      string
	Tests      []GeneratedTestFunc
	Teardown   string
}

// GeneratedTestFunc represents a generated test function.
type GeneratedTestFunc struct {
	Name        string
	TargetFunc  string
	Description string
	Cases       []TestCase
	Code        string
}

// NewTestGenerator creates a new test generator.
func NewTestGenerator() *TestGenerator {
	tg := &TestGenerator{
		templates: make(map[string]*TestTemplate),
		analyzers: make(map[string]CodeAnalyzer),
	}

	// Register default templates
	tg.RegisterTemplate("go", goTestTemplate())
	tg.RegisterTemplate("python", pythonTestTemplate())
	tg.RegisterTemplate("javascript", jsTestTemplate())
	tg.RegisterTemplate("typescript", tsTestTemplate())

	// Register analyzers
	tg.analyzers["go"] = &GoAnalyzer{}
	tg.analyzers["python"] = &PythonAnalyzer{}
	tg.analyzers["javascript"] = &JSAnalyzer{}
	tg.analyzers["typescript"] = &JSAnalyzer{} // Same as JS

	return tg
}

// RegisterTemplate registers a test template for a language.
func (tg *TestGenerator) RegisterTemplate(lang string, template *TestTemplate) {
	tg.templates[lang] = template
}

// GenerateTests generates tests for the given source code.
func (tg *TestGenerator) GenerateTests(sourceFile, code, language string) (*GeneratedTest, error) {
	template, ok := tg.templates[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	analyzer, ok := tg.analyzers[language]
	if !ok {
		return nil, fmt.Errorf("no analyzer for language: %s", language)
	}

	// Extract functions
	functions := analyzer.ExtractFunctions(code)

	// Generate test file path
	testFile := tg.generateTestFilePath(sourceFile, template)

	// Generate tests
	result := &GeneratedTest{
		Language:   language,
		SourceFile: sourceFile,
		TestFile:   testFile,
		Imports:    tg.generateImports(template, sourceFile),
		Setup:      template.Setup,
		Tests:      make([]GeneratedTestFunc, 0),
		Teardown:   template.Teardown,
	}

	for _, fn := range functions {
		if !fn.IsExported {
			continue // Only test exported functions
		}

		testFunc := tg.generateTestFunc(fn, template)
		result.Tests = append(result.Tests, testFunc)
	}

	return result, nil
}

func (tg *TestGenerator) generateTestFilePath(sourceFile string, template *TestTemplate) string {
	ext := filepath.Ext(sourceFile)
	base := strings.TrimSuffix(sourceFile, ext)
	return base + template.FilePattern
}

func (tg *TestGenerator) generateImports(template *TestTemplate, sourceFile string) []string {
	imports := []string{}
	if template.Imports != "" {
		imports = append(imports, template.Imports)
	}
	return imports
}

func (tg *TestGenerator) generateTestFunc(fn FunctionInfo, template *TestTemplate) GeneratedTestFunc {
	testFunc := GeneratedTestFunc{
		Name:        "Test" + fn.Name,
		TargetFunc:  fn.Name,
		Description: fmt.Sprintf("Tests the %s function", fn.Name),
		Cases:       tg.generateTestCases(fn),
	}

	// Generate code based on template
	testFunc.Code = tg.generateTestCode(testFunc, fn, template)

	return testFunc
}

func (tg *TestGenerator) generateTestCases(fn FunctionInfo) []TestCase {
	cases := []TestCase{}

	// Generate basic test case
	cases = append(cases, TestCase{
		Name:        "basic",
		Description: "Basic test case",
		Input:       tg.generateDefaultInputs(fn.Params),
		Expected:    "expected",
	})

	// Generate edge cases based on parameter types
	for _, param := range fn.Params {
		edgeCases := tg.generateEdgeCases(param)
		cases = append(cases, edgeCases...)
	}

	return cases
}

func (tg *TestGenerator) generateDefaultInputs(params []ParamInfo) []string {
	inputs := make([]string, len(params))
	for i, param := range params {
		inputs[i] = tg.getDefaultValue(param.Type)
	}
	return inputs
}

func (tg *TestGenerator) getDefaultValue(typeName string) string {
	switch strings.ToLower(typeName) {
	case "string":
		return `"test"`
	case "int", "int32", "int64", "number":
		return "1"
	case "float", "float32", "float64":
		return "1.0"
	case "bool", "boolean":
		return "true"
	case "[]string":
		return `[]string{"a", "b"}`
	case "[]int":
		return "[]int{1, 2, 3}"
	default:
		if strings.HasPrefix(typeName, "[]") {
			return "nil"
		}
		if strings.HasPrefix(typeName, "*") {
			return "nil"
		}
		if strings.HasPrefix(typeName, "map") {
			return "nil"
		}
		return "nil"
	}
}

func (tg *TestGenerator) generateEdgeCases(param ParamInfo) []TestCase {
	cases := []TestCase{}

	switch strings.ToLower(param.Type) {
	case "string":
		cases = append(cases, TestCase{
			Name:        param.Name + "_empty",
			Description: "Empty string for " + param.Name,
			Input:       []string{`""`},
		})
	case "int", "int32", "int64", "number":
		cases = append(cases, TestCase{
			Name:        param.Name + "_zero",
			Description: "Zero value for " + param.Name,
			Input:       []string{"0"},
		})
		cases = append(cases, TestCase{
			Name:        param.Name + "_negative",
			Description: "Negative value for " + param.Name,
			Input:       []string{"-1"},
		})
	case "[]string", "[]int":
		cases = append(cases, TestCase{
			Name:        param.Name + "_empty",
			Description: "Empty slice for " + param.Name,
			Input:       []string{"nil"},
		})
	}

	return cases
}

func (tg *TestGenerator) generateTestCode(testFunc GeneratedTestFunc, fn FunctionInfo, template *TestTemplate) string {
	var sb strings.Builder

	switch template.Language {
	case "go":
		sb.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testFunc.Name))
		if len(testFunc.Cases) > 1 {
			// Table-driven test
			sb.WriteString("\ttests := []struct {\n")
			sb.WriteString("\t\tname string\n")
			for i, param := range fn.Params {
				sb.WriteString(fmt.Sprintf("\t\targ%d %s\n", i, param.Type))
			}
			if len(fn.Returns) > 0 {
				sb.WriteString(fmt.Sprintf("\t\twant %s\n", fn.Returns[0]))
			}
			sb.WriteString("\t}{\n")
			for _, tc := range testFunc.Cases {
				sb.WriteString(fmt.Sprintf("\t\t{%q, ", tc.Name))
				sb.WriteString(strings.Join(tc.Input, ", "))
				sb.WriteString(", /* expected */},\n")
			}
			sb.WriteString("\t}\n\n")
			sb.WriteString("\tfor _, tt := range tests {\n")
			sb.WriteString("\t\tt.Run(tt.name, func(t *testing.T) {\n")
			sb.WriteString(fmt.Sprintf("\t\t\tgot := %s(", fn.Name))
			for i := range fn.Params {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("tt.arg%d", i))
			}
			sb.WriteString(")\n")
			sb.WriteString("\t\t\tif got != tt.want {\n")
			sb.WriteString("\t\t\t\tt.Errorf(\"%s() = %v, want %v\", got, tt.want)\n")
			sb.WriteString("\t\t\t}\n")
			sb.WriteString("\t\t})\n")
			sb.WriteString("\t}\n")
		} else {
			// Simple test
			sb.WriteString(fmt.Sprintf("\tresult := %s(", fn.Name))
			if len(testFunc.Cases) > 0 && len(testFunc.Cases[0].Input) > 0 {
				sb.WriteString(strings.Join(testFunc.Cases[0].Input, ", "))
			}
			sb.WriteString(")\n")
			sb.WriteString("\t// TODO: Add assertions\n")
			sb.WriteString("\t_ = result\n")
		}
		sb.WriteString("}\n")

	case "python":
		sb.WriteString(fmt.Sprintf("def test_%s():\n", strings.ToLower(fn.Name)))
		sb.WriteString(fmt.Sprintf("    result = %s(", fn.Name))
		if len(testFunc.Cases) > 0 && len(testFunc.Cases[0].Input) > 0 {
			sb.WriteString(strings.Join(testFunc.Cases[0].Input, ", "))
		}
		sb.WriteString(")\n")
		sb.WriteString("    # TODO: Add assertions\n")
		sb.WriteString("    assert result is not None\n")

	case "javascript", "typescript":
		sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))
		for _, tc := range testFunc.Cases {
			sb.WriteString(fmt.Sprintf("  it('%s', () => {\n", tc.Description))
			sb.WriteString(fmt.Sprintf("    const result = %s(", fn.Name))
			sb.WriteString(strings.Join(tc.Input, ", "))
			sb.WriteString(");\n")
			sb.WriteString("    // TODO: Add assertions\n")
			sb.WriteString("    expect(result).toBeDefined();\n")
			sb.WriteString("  });\n")
		}
		sb.WriteString("});\n")
	}

	return sb.String()
}

// Render generates the complete test file content.
func (gt *GeneratedTest) Render() string {
	var sb strings.Builder

	// Add imports
	for _, imp := range gt.Imports {
		sb.WriteString(imp)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Add setup
	if gt.Setup != "" {
		sb.WriteString(gt.Setup)
		sb.WriteString("\n\n")
	}

	// Add tests
	for _, test := range gt.Tests {
		sb.WriteString(test.Code)
		sb.WriteString("\n")
	}

	// Add teardown
	if gt.Teardown != "" {
		sb.WriteString(gt.Teardown)
		sb.WriteString("\n")
	}

	return sb.String()
}

// Default templates

func goTestTemplate() *TestTemplate {
	return &TestTemplate{
		Language:    "go",
		FilePattern: "_test.go",
		Imports:     `import "testing"`,
		TestFunc: `func Test{{.Name}}(t *testing.T) {
	{{.Body}}
}`,
		TableDriven: `func Test{{.Name}}(t *testing.T) {
	tests := []struct {
		name string
		{{.Fields}}
	}{
		{{.Cases}}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			{{.Body}}
		})
	}
}`,
		Assertion: `if got != want {
	t.Errorf("{{.Name}}() = %v, want %v", got, want)
}`,
	}
}

func pythonTestTemplate() *TestTemplate {
	return &TestTemplate{
		Language:    "python",
		FilePattern: "_test.py",
		Imports:     "import pytest",
		TestFunc: `def test_{{.Name}}():
    {{.Body}}`,
		Assertion: "assert result == expected",
	}
}

func jsTestTemplate() *TestTemplate {
	return &TestTemplate{
		Language:    "javascript",
		FilePattern: ".test.js",
		Imports:     "",
		TestFunc: `describe('{{.Name}}', () => {
  it('{{.Description}}', () => {
    {{.Body}}
  });
});`,
		Assertion: "expect(result).toEqual(expected);",
	}
}

func tsTestTemplate() *TestTemplate {
	return &TestTemplate{
		Language:    "typescript",
		FilePattern: ".test.ts",
		Imports:     "",
		TestFunc: `describe('{{.Name}}', () => {
  it('{{.Description}}', () => {
    {{.Body}}
  });
});`,
		Assertion: "expect(result).toEqual(expected);",
	}
}

// GoAnalyzer analyzes Go source code.
type GoAnalyzer struct{}

func (a *GoAnalyzer) ExtractFunctions(code string) []FunctionInfo {
	var functions []FunctionInfo

	// Match function declarations
	funcRegex := regexp.MustCompile(`func\s+(\w+)\s*\(([^)]*)\)\s*(?:\(([^)]*)\)|(\w+))?\s*\{`)
	matches := funcRegex.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		if match[2] == -1 {
			continue
		}

		name := code[match[2]:match[3]]
		params := ""
		if match[4] != -1 {
			params = code[match[4]:match[5]]
		}

		fn := FunctionInfo{
			Name:       name,
			Params:     parseGoParams(params),
			IsExported: name[0] >= 'A' && name[0] <= 'Z',
		}

		// Parse returns
		if match[6] != -1 {
			fn.Returns = []string{code[match[6]:match[7]]}
		} else if match[8] != -1 {
			fn.Returns = []string{code[match[8]:match[9]]}
		}

		functions = append(functions, fn)
	}

	return functions
}

func (a *GoAnalyzer) ExtractClasses(code string) []ClassInfo {
	return nil // Go doesn't have classes
}

func (a *GoAnalyzer) ExtractMethods(code string) []MethodInfo {
	var methods []MethodInfo

	// Match method declarations
	methodRegex := regexp.MustCompile(`func\s+\((\w+)\s+\*?(\w+)\)\s+(\w+)\s*\(([^)]*)\)\s*(?:\(([^)]*)\)|(\w+))?\s*\{`)
	matches := methodRegex.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		receiver := ""
		if match[4] != -1 {
			receiver = code[match[4]:match[5]]
		}
		name := code[match[6]:match[7]]
		params := ""
		if match[8] != -1 {
			params = code[match[8]:match[9]]
		}

		method := MethodInfo{
			Name:       name,
			Receiver:   receiver,
			Params:     parseGoParams(params),
			IsExported: name[0] >= 'A' && name[0] <= 'Z',
		}

		// Parse returns
		if match[10] != -1 {
			method.Returns = []string{code[match[10]:match[11]]}
		} else if match[12] != -1 {
			method.Returns = []string{code[match[12]:match[13]]}
		}

		methods = append(methods, method)
	}

	return methods
}

func parseGoParams(params string) []ParamInfo {
	if strings.TrimSpace(params) == "" {
		return nil
	}

	var result []ParamInfo
	parts := strings.Split(params, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle "name type" or just "type"
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			result = append(result, ParamInfo{
				Name: fields[0],
				Type: strings.Join(fields[1:], " "),
			})
		} else if len(fields) == 1 {
			result = append(result, ParamInfo{
				Type: fields[0],
			})
		}
	}

	return result
}

// PythonAnalyzer analyzes Python source code.
type PythonAnalyzer struct{}

func (a *PythonAnalyzer) ExtractFunctions(code string) []FunctionInfo {
	var functions []FunctionInfo

	// Match function definitions
	funcRegex := regexp.MustCompile(`def\s+(\w+)\s*\(([^)]*)\)(?:\s*->\s*(\w+))?\s*:`)
	matches := funcRegex.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		name := code[match[2]:match[3]]
		params := ""
		if match[4] != -1 {
			params = code[match[4]:match[5]]
		}

		fn := FunctionInfo{
			Name:       name,
			Params:     parsePythonParams(params),
			IsExported: !strings.HasPrefix(name, "_"),
			IsAsync:    strings.Contains(code[max(0, match[0]-10):match[0]], "async"),
		}

		// Parse return type
		if match[6] != -1 {
			fn.Returns = []string{code[match[6]:match[7]]}
		}

		functions = append(functions, fn)
	}

	return functions
}

func (a *PythonAnalyzer) ExtractClasses(code string) []ClassInfo {
	var classes []ClassInfo

	classRegex := regexp.MustCompile(`class\s+(\w+)(?:\([^)]*\))?\s*:`)
	matches := classRegex.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		name := code[match[2]:match[3]]
		classes = append(classes, ClassInfo{
			Name:       name,
			IsExported: !strings.HasPrefix(name, "_"),
		})
	}

	return classes
}

func (a *PythonAnalyzer) ExtractMethods(code string) []MethodInfo {
	return nil // Methods are extracted as part of classes
}

func parsePythonParams(params string) []ParamInfo {
	if strings.TrimSpace(params) == "" {
		return nil
	}

	var result []ParamInfo
	parts := strings.Split(params, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "self" || part == "cls" {
			continue
		}

		// Handle "name: type" or "name: type = default" or just "name"
		if idx := strings.Index(part, ":"); idx != -1 {
			name := strings.TrimSpace(part[:idx])
			typePart := part[idx+1:]
			if eqIdx := strings.Index(typePart, "="); eqIdx != -1 {
				typePart = typePart[:eqIdx]
			}
			result = append(result, ParamInfo{
				Name: name,
				Type: strings.TrimSpace(typePart),
			})
		} else {
			if eqIdx := strings.Index(part, "="); eqIdx != -1 {
				part = part[:eqIdx]
			}
			result = append(result, ParamInfo{
				Name: strings.TrimSpace(part),
			})
		}
	}

	return result
}

// JSAnalyzer analyzes JavaScript/TypeScript source code.
type JSAnalyzer struct{}

func (a *JSAnalyzer) ExtractFunctions(code string) []FunctionInfo {
	var functions []FunctionInfo

	// Match function declarations
	patterns := []string{
		`function\s+(\w+)\s*\(([^)]*)\)`,              // function name()
		`(?:const|let|var)\s+(\w+)\s*=\s*\(([^)]*)\)\s*=>`, // const name = () =>
		`(?:const|let|var)\s+(\w+)\s*=\s*function\s*\(([^)]*)\)`, // const name = function()
		`(\w+)\s*:\s*\(([^)]*)\)\s*=>`,               // name: () => (object method)
	}

	for _, pattern := range patterns {
		funcRegex := regexp.MustCompile(pattern)
		matches := funcRegex.FindAllStringSubmatchIndex(code, -1)

		for _, match := range matches {
			if match[2] == -1 {
				continue
			}

			name := code[match[2]:match[3]]
			params := ""
			if match[4] != -1 {
				params = code[match[4]:match[5]]
			}

			fn := FunctionInfo{
				Name:       name,
				Params:     parseJSParams(params),
				IsExported: strings.Contains(code[max(0, match[0]-20):match[0]], "export"),
				IsAsync:    strings.Contains(code[max(0, match[0]-10):match[0]], "async"),
			}

			functions = append(functions, fn)
		}
	}

	return functions
}

func (a *JSAnalyzer) ExtractClasses(code string) []ClassInfo {
	var classes []ClassInfo

	classRegex := regexp.MustCompile(`class\s+(\w+)(?:\s+extends\s+\w+)?\s*\{`)
	matches := classRegex.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		name := code[match[2]:match[3]]
		classes = append(classes, ClassInfo{
			Name:       name,
			IsExported: strings.Contains(code[max(0, match[0]-20):match[0]], "export"),
		})
	}

	return classes
}

func (a *JSAnalyzer) ExtractMethods(code string) []MethodInfo {
	return nil
}

func parseJSParams(params string) []ParamInfo {
	if strings.TrimSpace(params) == "" {
		return nil
	}

	var result []ParamInfo
	parts := strings.Split(params, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle "name: type" or "name = default" or just "name"
		name := part
		typeName := ""

		if idx := strings.Index(part, ":"); idx != -1 {
			name = strings.TrimSpace(part[:idx])
			typePart := part[idx+1:]
			if eqIdx := strings.Index(typePart, "="); eqIdx != -1 {
				typePart = typePart[:eqIdx]
			}
			typeName = strings.TrimSpace(typePart)
		} else if idx := strings.Index(part, "="); idx != -1 {
			name = strings.TrimSpace(part[:idx])
		}

		result = append(result, ParamInfo{
			Name: name,
			Type: typeName,
		})
	}

	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
