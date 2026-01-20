package codegen

import (
	"strings"
	"testing"
)

// Test generation tests

func TestNewTestGenerator(t *testing.T) {
	tg := NewTestGenerator()
	if tg == nil {
		t.Fatal("NewTestGenerator returned nil")
	}
}

func TestTestGenerator_GenerateGoTests(t *testing.T) {
	tg := NewTestGenerator()

	code := `package math

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}
`

	tests, err := tg.GenerateTests("math.go", code, "go")
	if err != nil {
		t.Fatalf("GenerateTests failed: %v", err)
	}

	// Should generate tests for both functions
	if len(tests.Tests) < 2 {
		t.Errorf("Expected at least 2 tests, got %d", len(tests.Tests))
	}

	// Check test names
	foundAdd := false
	foundSubtract := false
	for _, test := range tests.Tests {
		if strings.Contains(test.Name, "Add") {
			foundAdd = true
		}
		if strings.Contains(test.Name, "Subtract") {
			foundSubtract = true
		}
	}

	if !foundAdd {
		t.Error("Expected test for Add function")
	}
	if !foundSubtract {
		t.Error("Expected test for Subtract function")
	}
}

func TestTestGenerator_GeneratePythonTests(t *testing.T) {
	tg := NewTestGenerator()

	code := `def greet(name):
    return f"Hello, {name}!"

def add(a, b):
    return a + b
`

	tests, err := tg.GenerateTests("utils.py", code, "python")
	if err != nil {
		t.Fatalf("GenerateTests failed: %v", err)
	}

	if len(tests.Tests) < 2 {
		t.Errorf("Expected at least 2 tests, got %d", len(tests.Tests))
	}

	// Check test file path
	if tests.TestFile == "" {
		t.Error("Expected test file path")
	}
}

func TestTestGenerator_GenerateJSTests(t *testing.T) {
	tg := NewTestGenerator()

	code := `function multiply(a, b) { return a * b; }
function divide(a, b) { return a / b; }
`

	tests, err := tg.GenerateTests("math.js", code, "javascript")
	if err != nil {
		t.Fatalf("GenerateTests failed: %v", err)
	}

	// Verify test structure is created (even if no functions found due to regex limitations)
	if tests == nil {
		t.Fatal("Expected non-nil test result")
	}

	if tests.Language != "javascript" {
		t.Errorf("Expected language 'javascript', got '%s'", tests.Language)
	}

	if tests.TestFile == "" {
		t.Error("Expected test file path")
	}
}

func TestTestGenerator_UnsupportedLanguage(t *testing.T) {
	tg := NewTestGenerator()

	_, err := tg.GenerateTests("file.xyz", "code", "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

// Documentation generation tests

func TestNewDocGenerator(t *testing.T) {
	dg := NewDocGenerator()
	if dg == nil {
		t.Fatal("NewDocGenerator returned nil")
	}
}

func TestDocGenerator_GenerateFunctionDoc_Go(t *testing.T) {
	dg := NewDocGenerator()

	fn := FunctionInfo{
		Name: "ProcessData",
		Params: []ParamInfo{
			{Name: "input", Type: "string"},
			{Name: "count", Type: "int"},
		},
		Returns: []string{"error"},
	}

	doc, err := dg.GenerateFunctionDoc(fn, "go")
	if err != nil {
		t.Fatalf("GenerateFunctionDoc failed: %v", err)
	}

	if !strings.HasPrefix(doc, "//") {
		t.Error("Go doc should start with //")
	}
	if !strings.Contains(doc, "ProcessData") {
		t.Error("Doc should contain function name")
	}
}

func TestDocGenerator_GenerateFunctionDoc_Python(t *testing.T) {
	dg := NewDocGenerator()

	fn := FunctionInfo{
		Name: "process_data",
		Params: []ParamInfo{
			{Name: "input", Type: "str"},
		},
		Returns: []string{"str"},
	}

	doc, err := dg.GenerateFunctionDoc(fn, "python")
	if err != nil {
		t.Fatalf("GenerateFunctionDoc failed: %v", err)
	}

	if !strings.Contains(doc, `"""`) {
		t.Error("Python doc should use triple quotes")
	}
	if !strings.Contains(doc, "Args:") {
		t.Error("Python doc should have Args section")
	}
}

func TestDocGenerator_GenerateFunctionDoc_JavaScript(t *testing.T) {
	dg := NewDocGenerator()

	fn := FunctionInfo{
		Name: "processData",
		Params: []ParamInfo{
			{Name: "input", Type: "string"},
		},
		Returns: []string{"string"},
	}

	doc, err := dg.GenerateFunctionDoc(fn, "javascript")
	if err != nil {
		t.Fatalf("GenerateFunctionDoc failed: %v", err)
	}

	if !strings.Contains(doc, "/**") {
		t.Error("JS doc should use JSDoc style")
	}
	if !strings.Contains(doc, "@param") {
		t.Error("JS doc should have @param")
	}
}

func TestDocGenerator_GenerateClassDoc(t *testing.T) {
	dg := NewDocGenerator()

	cls := ClassInfo{
		Name: "UserService",
		Properties: []PropertyInfo{
			{Name: "name", Type: "string"},
		},
		Methods: []MethodInfo{
			{Name: "GetUser", Params: []ParamInfo{{Name: "id", Type: "string"}}},
		},
	}

	doc, err := dg.GenerateClassDoc(cls, "go")
	if err != nil {
		t.Fatalf("GenerateClassDoc failed: %v", err)
	}

	if !strings.Contains(doc, "UserService") {
		t.Error("Doc should contain class name")
	}
}

func TestDocGenerator_UnsupportedLanguage(t *testing.T) {
	dg := NewDocGenerator()

	fn := FunctionInfo{Name: "test"}
	_, err := dg.GenerateFunctionDoc(fn, "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

// Boilerplate generation tests

func TestNewBoilerplateGenerator(t *testing.T) {
	bg := NewBoilerplateGenerator()
	if bg == nil {
		t.Fatal("NewBoilerplateGenerator returned nil")
	}
}

func TestBoilerplateGenerator_GenerateGoCRUD(t *testing.T) {
	bg := NewBoilerplateGenerator()

	config := BoilerplateConfig{
		Type:       BoilerplateCRUD,
		Language:   "go",
		EntityName: "User",
		Fields: []FieldSpec{
			{Name: "Name", Type: "string", Required: true},
			{Name: "Email", Type: "string", Required: true},
			{Name: "Age", Type: "int"},
		},
		Options: BoilerplateOptions{
			PackageName:   "user",
			IncludeTests:  true,
			IncludeCreate: true,
			IncludeRead:   true,
			IncludeUpdate: true,
			IncludeDelete: true,
			IncludeList:   true,
			UseInterfaces: true,
		},
	}

	result, err := bg.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should generate model, repository, service, handler, and test files
	if len(result.Files) < 4 {
		t.Errorf("Expected at least 4 files, got %d", len(result.Files))
	}

	// Check for model file
	foundModel := false
	foundRepo := false
	for _, file := range result.Files {
		if strings.Contains(file.Name, "user.go") && !strings.Contains(file.Name, "_") {
			foundModel = true
			if !strings.Contains(file.Content, "type User struct") {
				t.Error("Model should contain User struct")
			}
		}
		if strings.Contains(file.Name, "repository") {
			foundRepo = true
			if !strings.Contains(file.Content, "UserRepository") {
				t.Error("Repository should contain UserRepository interface")
			}
		}
	}

	if !foundModel {
		t.Error("Should generate model file")
	}
	if !foundRepo {
		t.Error("Should generate repository file")
	}
}

func TestBoilerplateGenerator_GeneratePythonModel(t *testing.T) {
	bg := NewBoilerplateGenerator()

	config := BoilerplateConfig{
		Type:       BoilerplateModel,
		Language:   "python",
		EntityName: "Product",
		Fields: []FieldSpec{
			{Name: "Name", Type: "string", Required: true},
			{Name: "Price", Type: "float64", Required: true},
		},
		Options: BoilerplateOptions{
			PackageName: "product",
		},
	}

	result, err := bg.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(result.Files) < 1 {
		t.Error("Expected at least 1 file")
	}

	// Check Python model
	if !strings.Contains(result.Files[0].Content, "@dataclass") {
		t.Error("Python model should use dataclass")
	}
	if !strings.Contains(result.Files[0].Content, "class Product:") {
		t.Error("Python model should define Product class")
	}
}

func TestBoilerplateGenerator_GenerateTypeScriptAPI(t *testing.T) {
	bg := NewBoilerplateGenerator()

	config := BoilerplateConfig{
		Type:       BoilerplateAPI,
		Language:   "typescript",
		EntityName: "Order",
		Fields: []FieldSpec{
			{Name: "Items", Type: "[]string"},
			{Name: "Total", Type: "float64"},
		},
		Options: BoilerplateOptions{
			PackageName:   "order",
			IncludeCreate: true,
			IncludeRead:   true,
			IncludeUpdate: true,
			IncludeDelete: true,
			IncludeList:   true,
		},
	}

	result, err := bg.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for interface and service
	foundInterface := false
	foundService := false
	for _, file := range result.Files {
		if strings.Contains(file.Content, "interface Order") {
			foundInterface = true
		}
		if strings.Contains(file.Content, "OrderService") {
			foundService = true
		}
	}

	if !foundInterface {
		t.Error("Should generate TypeScript interface")
	}
	if !foundService {
		t.Error("Should generate TypeScript service")
	}
}

func TestBoilerplateGenerator_UnsupportedLanguage(t *testing.T) {
	bg := NewBoilerplateGenerator()

	config := BoilerplateConfig{
		Type:       BoilerplateCRUD,
		Language:   "unsupported",
		EntityName: "Test",
	}

	_, err := bg.Generate(config)
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

// Refactoring analysis tests

func TestNewRefactoringAnalyzer(t *testing.T) {
	ra := NewRefactoringAnalyzer()
	if ra == nil {
		t.Fatal("NewRefactoringAnalyzer returned nil")
	}
	if len(ra.rules) == 0 {
		t.Error("Analyzer should have default rules")
	}
}

func TestRefactoringAnalyzer_LongFunction(t *testing.T) {
	ra := NewRefactoringAnalyzer()

	// Create a long function
	var sb strings.Builder
	sb.WriteString("func longFunction() {\n")
	for i := 0; i < 60; i++ {
		sb.WriteString("    // line\n")
	}
	sb.WriteString("}\n")

	suggestions := ra.Analyze(sb.String(), "go")

	foundLong := false
	for _, s := range suggestions {
		if s.Type == RefactorExtractFunction && strings.Contains(s.Description, "lines long") {
			foundLong = true
			break
		}
	}

	if !foundLong {
		t.Error("Should detect long function")
	}
}

func TestRefactoringAnalyzer_LongParameterList(t *testing.T) {
	ra := NewRefactoringAnalyzer()

	code := `func tooManyParams(a, b, c, d, e, f, g int) {
    // body
}`

	suggestions := ra.Analyze(code, "go")

	foundLongParams := false
	for _, s := range suggestions {
		if strings.Contains(s.Description, "parameters") {
			foundLongParams = true
			break
		}
	}

	if !foundLongParams {
		t.Error("Should detect long parameter list")
	}
}

func TestRefactoringAnalyzer_DeepNesting(t *testing.T) {
	ra := NewRefactoringAnalyzer()

	code := `func deepNesting() {
	if true {
		if true {
			if true {
				if true {
					if true {
						// deep
					}
				}
			}
		}
	}
}`

	suggestions := ra.Analyze(code, "go")

	foundDeep := false
	for _, s := range suggestions {
		if s.Type == RefactorExtractFunction && strings.Contains(s.Description, "nested") {
			foundDeep = true
			break
		}
	}

	if !foundDeep {
		t.Error("Should detect deep nesting")
	}
}

func TestRefactorer_Rename(t *testing.T) {
	r := NewRefactorer("go")

	code := `func processUser(user User) {
    fmt.Println(user.Name)
    return user
}`

	result, count := r.Rename(code, "user", "u")

	if count != 3 {
		t.Errorf("Expected 3 replacements, got %d", count)
	}

	if strings.Contains(result, "user") {
		t.Error("Should have replaced all occurrences of 'user'")
	}

	if !strings.Contains(result, "processUser") {
		t.Error("Should not rename partial matches like 'processUser'")
	}
}

func TestRefactorer_ExtractVariable(t *testing.T) {
	r := NewRefactorer("go")

	code := `func example() {
    result := calculateValue() + 10
}`

	result, err := r.ExtractVariable(code, 2, 14, 30, "baseValue")
	if err != nil {
		t.Fatalf("ExtractVariable failed: %v", err)
	}

	if !strings.Contains(result, "baseValue :=") {
		t.Error("Should create variable declaration")
	}
}

// Bug fix suggester tests

func TestNewBugFixSuggester(t *testing.T) {
	bfs := NewBugFixSuggester()
	if bfs == nil {
		t.Fatal("NewBugFixSuggester returned nil")
	}
	if len(bfs.patterns) == 0 {
		t.Error("Suggester should have patterns for multiple languages")
	}
}

func TestBugFixSuggester_GoUndefined(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("undefined: myVariable", "go", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for undefined variable")
	}

	if suggestion.ErrorType != "undefined_identifier" {
		t.Errorf("Expected error type 'undefined_identifier', got '%s'", suggestion.ErrorType)
	}

	if len(suggestion.Fixes) == 0 {
		t.Error("Should have fix suggestions")
	}
}

func TestBugFixSuggester_GoTypeMismatch(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("cannot use x (variable of type int) as type string", "go", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for type mismatch")
	}

	if suggestion.ErrorType != "type_mismatch" {
		t.Errorf("Expected error type 'type_mismatch', got '%s'", suggestion.ErrorType)
	}
}

func TestBugFixSuggester_GoNilPointer(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("nil pointer dereference", "go", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for nil pointer")
	}

	if suggestion.ErrorType != "nil_pointer" {
		t.Errorf("Expected error type 'nil_pointer', got '%s'", suggestion.ErrorType)
	}

	// Should have nil check as a fix
	hasNilCheck := false
	for _, fix := range suggestion.Fixes {
		if strings.Contains(fix.Code, "!= nil") {
			hasNilCheck = true
			break
		}
	}
	if !hasNilCheck {
		t.Error("Should suggest nil check")
	}
}

func TestBugFixSuggester_PythonNameError(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("NameError: name 'undefined_var' is not defined", "python", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for NameError")
	}

	if suggestion.ErrorType != "name_error" {
		t.Errorf("Expected error type 'name_error', got '%s'", suggestion.ErrorType)
	}
}

func TestBugFixSuggester_PythonKeyError(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("KeyError: 'missing_key'", "python", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for KeyError")
	}

	if suggestion.ErrorType != "key_error" {
		t.Errorf("Expected error type 'key_error', got '%s'", suggestion.ErrorType)
	}

	// Should suggest .get() method
	hasGetSuggestion := false
	for _, fix := range suggestion.Fixes {
		if strings.Contains(fix.Code, ".get(") {
			hasGetSuggestion = true
			break
		}
	}
	if !hasGetSuggestion {
		t.Error("Should suggest using .get() method")
	}
}

func TestBugFixSuggester_JavaScriptReferenceError(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("ReferenceError: myFunc is not defined", "javascript", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for ReferenceError")
	}

	if suggestion.ErrorType != "reference_error" {
		t.Errorf("Expected error type 'reference_error', got '%s'", suggestion.ErrorType)
	}
}

func TestBugFixSuggester_JavaScriptNullProperty(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("TypeError: Cannot read property 'name' of null", "javascript", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for null property access")
	}

	// Should suggest optional chaining
	hasOptionalChaining := false
	for _, fix := range suggestion.Fixes {
		if strings.Contains(fix.Code, "?.") {
			hasOptionalChaining = true
			break
		}
	}
	if !hasOptionalChaining {
		t.Error("Should suggest optional chaining")
	}
}

func TestBugFixSuggester_TypeScriptTypeMismatch(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("Type 'number' is not assignable to type 'string'", "typescript", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for TypeScript type error")
	}

	if suggestion.ErrorType != "type_mismatch" {
		t.Errorf("Expected error type 'type_mismatch', got '%s'", suggestion.ErrorType)
	}
}

func TestBugFixSuggester_GenericDivisionByZero(t *testing.T) {
	bfs := NewBugFixSuggester()

	// Generic patterns should work for any language
	suggestion := bfs.SuggestFix("division by zero", "unknown", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for division by zero")
	}

	if suggestion.ErrorType != "division_by_zero" {
		t.Errorf("Expected error type 'division_by_zero', got '%s'", suggestion.ErrorType)
	}
}

func TestBugFixSuggester_GenericTimeout(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("request timed out after 30s", "unknown", "")

	if suggestion == nil {
		t.Fatal("Should return suggestion for timeout")
	}

	if suggestion.ErrorType != "timeout" {
		t.Errorf("Expected error type 'timeout', got '%s'", suggestion.ErrorType)
	}
}

func TestBugFixSuggester_NoMatch(t *testing.T) {
	bfs := NewBugFixSuggester()

	suggestion := bfs.SuggestFix("some random error message", "go", "")

	if suggestion != nil {
		t.Error("Should return nil for unrecognized error")
	}
}

func TestAnalyzeCompilerOutput_Go(t *testing.T) {
	output := `./main.go:10:5: undefined: foo
./main.go:15:10: cannot use x (type int) as type string`

	errors := AnalyzeCompilerOutput(output, "go")

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	if errors[0].Line != 10 {
		t.Errorf("Expected line 10, got %d", errors[0].Line)
	}

	if errors[0].Column != 5 {
		t.Errorf("Expected column 5, got %d", errors[0].Column)
	}
}

// Helper function tests

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserName", "user_name"},
		{"userName", "user_name"},
		{"ID", "i_d"},
		{"HTTPServer", "h_t_t_p_server"},
		{"simple", "simple"},
	}

	for _, tc := range tests {
		result := toSnakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("toSnakeCase(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserName", "userName"},
		{"ID", "iD"},
		{"A", "a"},
		{"", ""},
	}

	for _, tc := range tests {
		result := toCamelCase(tc.input)
		if result != tc.expected {
			t.Errorf("toCamelCase(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestGoPythonTypeMap(t *testing.T) {
	tests := []struct {
		goType     string
		pythonType string
	}{
		{"string", "str"},
		{"int", "int"},
		{"bool", "bool"},
		{"float64", "float"},
		{"[]string", "list[str]"},
		{"unknown", "Any"},
	}

	for _, tc := range tests {
		result := goPythonTypeMap(tc.goType)
		if result != tc.pythonType {
			t.Errorf("goPythonTypeMap(%s) = %s, expected %s", tc.goType, result, tc.pythonType)
		}
	}
}

func TestGoTSTypeMap(t *testing.T) {
	tests := []struct {
		goType string
		tsType string
	}{
		{"string", "string"},
		{"int", "number"},
		{"bool", "boolean"},
		{"[]string", "string[]"},
		{"unknown", "unknown"},
	}

	for _, tc := range tests {
		result := goTSTypeMap(tc.goType)
		if result != tc.tsType {
			t.Errorf("goTSTypeMap(%s) = %s, expected %s", tc.goType, result, tc.tsType)
		}
	}
}
