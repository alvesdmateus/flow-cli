package codegen

import (
	"fmt"
	"regexp"
	"strings"
)

// RefactoringType represents a type of refactoring.
type RefactoringType string

const (
	RefactorExtractFunction  RefactoringType = "extract_function"
	RefactorExtractVariable  RefactoringType = "extract_variable"
	RefactorRename           RefactoringType = "rename"
	RefactorInlineVariable   RefactoringType = "inline_variable"
	RefactorExtractConstant  RefactoringType = "extract_constant"
	RefactorMoveToFile       RefactoringType = "move_to_file"
	RefactorExtractInterface RefactoringType = "extract_interface"
)

// RefactoringSuggestion represents a suggested refactoring.
type RefactoringSuggestion struct {
	Type        RefactoringType
	Description string
	Location    CodeLocation
	Severity    string // "info", "warning", "suggestion"
	Code        string
	Fix         *RefactoringFix
}

// CodeLocation represents a location in code.
type CodeLocation struct {
	File      string
	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
}

// RefactoringFix represents an automated fix for a refactoring.
type RefactoringFix struct {
	Description string
	Before      string
	After       string
	Edits       []TextEdit
}

// TextEdit represents a text edit operation.
type TextEdit struct {
	Location CodeLocation
	NewText  string
}

// RefactoringAnalyzer analyzes code for refactoring opportunities.
type RefactoringAnalyzer struct {
	rules []RefactoringRule
}

// RefactoringRule defines a rule for detecting refactoring opportunities.
type RefactoringRule interface {
	Name() string
	Analyze(code string, language string) []RefactoringSuggestion
}

// NewRefactoringAnalyzer creates a new refactoring analyzer.
func NewRefactoringAnalyzer() *RefactoringAnalyzer {
	return &RefactoringAnalyzer{
		rules: []RefactoringRule{
			&LongFunctionRule{MaxLines: 50},
			&DuplicateCodeRule{MinLines: 5},
			&MagicNumberRule{},
			&LongParameterListRule{MaxParams: 5},
			&DeepNestingRule{MaxDepth: 4},
			&LargeClassRule{MaxMethods: 20},
			&UnusedVariableRule{},
		},
	}
}

// Analyze analyzes code for refactoring opportunities.
func (ra *RefactoringAnalyzer) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	for _, rule := range ra.rules {
		suggestions = append(suggestions, rule.Analyze(code, language)...)
	}

	return suggestions
}

// AddRule adds a custom refactoring rule.
func (ra *RefactoringAnalyzer) AddRule(rule RefactoringRule) {
	ra.rules = append(ra.rules, rule)
}

// LongFunctionRule detects functions that are too long.
type LongFunctionRule struct {
	MaxLines int
}

func (r *LongFunctionRule) Name() string {
	return "long_function"
}

func (r *LongFunctionRule) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	// Detect function boundaries based on language
	var funcPattern *regexp.Regexp
	switch language {
	case "go":
		funcPattern = regexp.MustCompile(`(?m)^func\s+(\w+)?\s*\([^)]*\)`)
	case "python":
		funcPattern = regexp.MustCompile(`(?m)^def\s+(\w+)\s*\(`)
	case "javascript", "typescript":
		funcPattern = regexp.MustCompile(`(?m)(function\s+(\w+)|(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>))`)
	default:
		return suggestions
	}

	lines := strings.Split(code, "\n")
	matches := funcPattern.FindAllStringIndex(code, -1)

	for i, match := range matches {
		startLine := countLines(code[:match[0]])
		var endLine int

		// Estimate function end (simplistic - would need proper parsing for accuracy)
		if i+1 < len(matches) {
			endLine = countLines(code[:matches[i+1][0]]) - 1
		} else {
			endLine = len(lines)
		}

		funcLines := endLine - startLine
		if funcLines > r.MaxLines {
			// Extract function name
			funcMatch := funcPattern.FindStringSubmatch(code[match[0]:])
			funcName := "anonymous"
			if len(funcMatch) > 1 && funcMatch[1] != "" {
				funcName = funcMatch[1]
			}

			suggestions = append(suggestions, RefactoringSuggestion{
				Type:        RefactorExtractFunction,
				Description: fmt.Sprintf("Function '%s' is %d lines long (max: %d). Consider extracting smaller functions.", funcName, funcLines, r.MaxLines),
				Location: CodeLocation{
					StartLine: startLine,
					EndLine:   endLine,
				},
				Severity: "warning",
			})
		}
	}

	return suggestions
}

// DuplicateCodeRule detects duplicate code blocks.
type DuplicateCodeRule struct {
	MinLines int
}

func (r *DuplicateCodeRule) Name() string {
	return "duplicate_code"
}

func (r *DuplicateCodeRule) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	lines := strings.Split(code, "\n")
	if len(lines) < r.MinLines*2 {
		return suggestions
	}

	// Simple duplicate detection: look for repeated blocks
	blockHashes := make(map[string][]int)

	for i := 0; i <= len(lines)-r.MinLines; i++ {
		block := strings.Join(lines[i:i+r.MinLines], "\n")
		block = strings.TrimSpace(block)
		if block == "" || len(block) < 50 { // Skip empty or very short blocks
			continue
		}

		// Normalize whitespace for comparison
		normalized := normalizeCode(block)
		blockHashes[normalized] = append(blockHashes[normalized], i+1)
	}

	seen := make(map[string]bool)
	for hash, lineNums := range blockHashes {
		if len(lineNums) > 1 && !seen[hash] {
			seen[hash] = true
			suggestions = append(suggestions, RefactoringSuggestion{
				Type:        RefactorExtractFunction,
				Description: fmt.Sprintf("Duplicate code block found at lines %v. Consider extracting to a shared function.", lineNums),
				Location: CodeLocation{
					StartLine: lineNums[0],
					EndLine:   lineNums[0] + r.MinLines,
				},
				Severity: "suggestion",
			})
		}
	}

	return suggestions
}

// MagicNumberRule detects magic numbers that should be constants.
type MagicNumberRule struct{}

func (r *MagicNumberRule) Name() string {
	return "magic_number"
}

func (r *MagicNumberRule) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	// Pattern for numbers that aren't 0, 1, 2 (commonly acceptable)
	magicPattern := regexp.MustCompile(`\b([3-9]|[1-9][0-9]+)\b`)
	skipPattern := regexp.MustCompile(`(const|#define|final|readonly|\[\d+\]|:\d+|"\d+"|'\d+')`)

	lines := strings.Split(code, "\n")
	for i, line := range lines {
		// Skip constant declarations and array indices
		if skipPattern.MatchString(line) {
			continue
		}

		matches := magicPattern.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			num := line[match[2]:match[3]]
			// Skip common acceptable numbers
			if num == "10" || num == "100" || num == "1000" {
				continue
			}

			suggestions = append(suggestions, RefactoringSuggestion{
				Type:        RefactorExtractConstant,
				Description: fmt.Sprintf("Magic number '%s' found. Consider extracting to a named constant.", num),
				Location: CodeLocation{
					StartLine: i + 1,
					EndLine:   i + 1,
					StartCol:  match[2],
					EndCol:    match[3],
				},
				Severity: "info",
				Code:     line,
			})
		}
	}

	// Limit suggestions to avoid noise
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// LongParameterListRule detects functions with too many parameters.
type LongParameterListRule struct {
	MaxParams int
}

func (r *LongParameterListRule) Name() string {
	return "long_parameter_list"
}

func (r *LongParameterListRule) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	var funcPattern *regexp.Regexp
	switch language {
	case "go":
		funcPattern = regexp.MustCompile(`func\s+(\w+)?\s*\(([^)]*)\)`)
	case "python":
		funcPattern = regexp.MustCompile(`def\s+(\w+)\s*\(([^)]*)\)`)
	case "javascript", "typescript":
		funcPattern = regexp.MustCompile(`(?:function\s+(\w+)|(\w+)\s*=\s*(?:async\s+)?function)\s*\(([^)]*)\)`)
	default:
		return suggestions
	}

	matches := funcPattern.FindAllStringSubmatch(code, -1)
	indices := funcPattern.FindAllStringIndex(code, -1)

	for i, match := range matches {
		var params string
		var funcName string

		switch language {
		case "go", "python":
			funcName = match[1]
			params = match[2]
		case "javascript", "typescript":
			if match[1] != "" {
				funcName = match[1]
			} else {
				funcName = match[2]
			}
			params = match[3]
		}

		if funcName == "" {
			funcName = "anonymous"
		}

		// Count parameters
		paramCount := countParams(params)
		if paramCount > r.MaxParams {
			lineNum := countLines(code[:indices[i][0]])
			suggestions = append(suggestions, RefactoringSuggestion{
				Type:        RefactorExtractFunction,
				Description: fmt.Sprintf("Function '%s' has %d parameters (max: %d). Consider using a parameter object or builder pattern.", funcName, paramCount, r.MaxParams),
				Location: CodeLocation{
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Severity: "warning",
			})
		}
	}

	return suggestions
}

// DeepNestingRule detects deeply nested code.
type DeepNestingRule struct {
	MaxDepth int
}

func (r *DeepNestingRule) Name() string {
	return "deep_nesting"
}

func (r *DeepNestingRule) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	lines := strings.Split(code, "\n")
	for i, line := range lines {
		// Count indentation depth (simplified)
		depth := countIndentDepth(line, language)
		if depth > r.MaxDepth {
			suggestions = append(suggestions, RefactoringSuggestion{
				Type:        RefactorExtractFunction,
				Description: fmt.Sprintf("Code is nested %d levels deep (max: %d). Consider extracting to reduce nesting.", depth, r.MaxDepth),
				Location: CodeLocation{
					StartLine: i + 1,
					EndLine:   i + 1,
				},
				Severity: "warning",
				Code:     line,
			})
		}
	}

	// Limit suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// LargeClassRule detects classes with too many methods.
type LargeClassRule struct {
	MaxMethods int
}

func (r *LargeClassRule) Name() string {
	return "large_class"
}

func (r *LargeClassRule) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	switch language {
	case "go":
		// For Go, detect structs with many methods
		structPattern := regexp.MustCompile(`type\s+(\w+)\s+struct`)
		methodPattern := regexp.MustCompile(`func\s+\([^)]+\s+\*?(\w+)\)`)

		structs := structPattern.FindAllStringSubmatch(code, -1)
		methods := methodPattern.FindAllStringSubmatch(code, -1)

		methodCounts := make(map[string]int)
		for _, m := range methods {
			methodCounts[m[1]]++
		}

		for _, s := range structs {
			structName := s[1]
			if count := methodCounts[structName]; count > r.MaxMethods {
				suggestions = append(suggestions, RefactoringSuggestion{
					Type:        RefactorExtractInterface,
					Description: fmt.Sprintf("Struct '%s' has %d methods (max: %d). Consider splitting into smaller types.", structName, count, r.MaxMethods),
					Location:    CodeLocation{StartLine: 1},
					Severity:    "warning",
				})
			}
		}

	case "python", "javascript", "typescript":
		classPattern := regexp.MustCompile(`class\s+(\w+)`)
		matches := classPattern.FindAllStringSubmatch(code, -1)
		indices := classPattern.FindAllStringIndex(code, -1)

		for i, match := range matches {
			className := match[1]
			startIdx := indices[i][0]

			// Find class end (simplified - next class or end of file)
			var endIdx int
			if i+1 < len(indices) {
				endIdx = indices[i+1][0]
			} else {
				endIdx = len(code)
			}

			classCode := code[startIdx:endIdx]

			// Count methods
			var methodPattern *regexp.Regexp
			if language == "python" {
				methodPattern = regexp.MustCompile(`(?m)^\s+def\s+\w+`)
			} else {
				methodPattern = regexp.MustCompile(`(?m)^\s+(?:async\s+)?(?:\w+\s*)?\(`)
			}

			methodCount := len(methodPattern.FindAllString(classCode, -1))
			if methodCount > r.MaxMethods {
				lineNum := countLines(code[:startIdx])
				suggestions = append(suggestions, RefactoringSuggestion{
					Type:        RefactorExtractInterface,
					Description: fmt.Sprintf("Class '%s' has %d methods (max: %d). Consider splitting into smaller classes.", className, methodCount, r.MaxMethods),
					Location: CodeLocation{
						StartLine: lineNum,
					},
					Severity: "warning",
				})
			}
		}
	}

	return suggestions
}

// UnusedVariableRule detects potentially unused variables.
type UnusedVariableRule struct{}

func (r *UnusedVariableRule) Name() string {
	return "unused_variable"
}

func (r *UnusedVariableRule) Analyze(code, language string) []RefactoringSuggestion {
	var suggestions []RefactoringSuggestion

	// This is a simplified check - proper implementation would need scope analysis
	var varPattern *regexp.Regexp
	switch language {
	case "go":
		varPattern = regexp.MustCompile(`(?m)^\s*(\w+)\s*:=`)
	case "python":
		varPattern = regexp.MustCompile(`(?m)^\s*(\w+)\s*=(?!=)`)
	case "javascript", "typescript":
		varPattern = regexp.MustCompile(`(?:let|const|var)\s+(\w+)\s*=`)
	default:
		return suggestions
	}

	matches := varPattern.FindAllStringSubmatch(code, -1)
	indices := varPattern.FindAllStringIndex(code, -1)

	for i, match := range matches {
		varName := match[1]
		if varName == "_" || varName == "err" {
			continue
		}

		// Check if variable is used elsewhere (after declaration)
		declEnd := indices[i][1]
		restOfCode := code[declEnd:]

		// Count occurrences
		usePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(varName) + `\b`)
		uses := usePattern.FindAllString(restOfCode, -1)

		if len(uses) == 0 {
			lineNum := countLines(code[:indices[i][0]])
			suggestions = append(suggestions, RefactoringSuggestion{
				Type:        RefactorInlineVariable,
				Description: fmt.Sprintf("Variable '%s' appears to be unused after declaration.", varName),
				Location: CodeLocation{
					StartLine: lineNum,
				},
				Severity: "info",
			})
		}
	}

	// Limit suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// Refactorer applies refactoring operations to code.
type Refactorer struct {
	language string
}

// NewRefactorer creates a new refactorer.
func NewRefactorer(language string) *Refactorer {
	return &Refactorer{language: language}
}

// ExtractFunction extracts a code block into a new function.
func (r *Refactorer) ExtractFunction(code string, startLine, endLine int, newFuncName string) (string, error) {
	lines := strings.Split(code, "\n")
	if startLine < 1 || endLine > len(lines) {
		return "", fmt.Errorf("invalid line range: %d-%d", startLine, endLine)
	}

	// Extract the block
	extractedLines := lines[startLine-1 : endLine]
	extractedCode := strings.Join(extractedLines, "\n")

	// Detect variables used in the block
	vars := r.detectUsedVariables(extractedCode)

	// Generate the new function
	var newFunc string
	var funcCall string

	switch r.language {
	case "go":
		params := strings.Join(vars, ", ")
		newFunc = fmt.Sprintf("func %s(%s) {\n%s\n}\n", newFuncName, params, extractedCode)
		funcCall = fmt.Sprintf("%s(%s)", newFuncName, params)
	case "python":
		params := strings.Join(vars, ", ")
		newFunc = fmt.Sprintf("def %s(%s):\n%s\n", newFuncName, params, indentCode(extractedCode, "    "))
		funcCall = fmt.Sprintf("%s(%s)", newFuncName, params)
	case "javascript", "typescript":
		params := strings.Join(vars, ", ")
		newFunc = fmt.Sprintf("function %s(%s) {\n%s\n}\n", newFuncName, params, extractedCode)
		funcCall = fmt.Sprintf("%s(%s);", newFuncName, params)
	default:
		return "", fmt.Errorf("unsupported language: %s", r.language)
	}

	// Replace the extracted block with the function call
	newLines := make([]string, 0, len(lines)-len(extractedLines)+2)
	newLines = append(newLines, lines[:startLine-1]...)
	newLines = append(newLines, funcCall)
	newLines = append(newLines, lines[endLine:]...)

	// Add the new function at the end
	result := strings.Join(newLines, "\n")
	result += "\n\n" + newFunc

	return result, nil
}

// Rename renames a symbol throughout the code.
func (r *Refactorer) Rename(code, oldName, newName string) (string, int) {
	// Use word boundary matching to avoid partial replacements
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(oldName) + `\b`)
	matches := pattern.FindAllStringIndex(code, -1)
	result := pattern.ReplaceAllString(code, newName)
	return result, len(matches)
}

// ExtractVariable extracts an expression into a variable.
func (r *Refactorer) ExtractVariable(code string, line int, startCol, endCol int, varName string) (string, error) {
	lines := strings.Split(code, "\n")
	if line < 1 || line > len(lines) {
		return "", fmt.Errorf("invalid line: %d", line)
	}

	targetLine := lines[line-1]
	if startCol < 0 || endCol > len(targetLine) {
		return "", fmt.Errorf("invalid column range: %d-%d", startCol, endCol)
	}

	expression := targetLine[startCol:endCol]

	var declaration string
	switch r.language {
	case "go":
		declaration = fmt.Sprintf("%s := %s", varName, expression)
	case "python":
		declaration = fmt.Sprintf("%s = %s", varName, expression)
	case "javascript", "typescript":
		declaration = fmt.Sprintf("const %s = %s;", varName, expression)
	default:
		return "", fmt.Errorf("unsupported language: %s", r.language)
	}

	// Replace expression with variable name
	newLine := targetLine[:startCol] + varName + targetLine[endCol:]
	lines[line-1] = newLine

	// Insert declaration before the line
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:line-1]...)
	newLines = append(newLines, declaration)
	newLines = append(newLines, lines[line-1:]...)

	return strings.Join(newLines, "\n"), nil
}

// Helper functions

func (r *Refactorer) detectUsedVariables(code string) []string {
	var varPattern *regexp.Regexp
	switch r.language {
	case "go":
		varPattern = regexp.MustCompile(`\b([a-z][a-zA-Z0-9]*)\b`)
	case "python":
		varPattern = regexp.MustCompile(`\b([a-z_][a-zA-Z0-9_]*)\b`)
	case "javascript", "typescript":
		varPattern = regexp.MustCompile(`\b([a-z_$][a-zA-Z0-9_$]*)\b`)
	default:
		return nil
	}

	// Find all variable-like identifiers
	matches := varPattern.FindAllString(code, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var vars []string
	for _, v := range matches {
		if !seen[v] && !isKeyword(v, r.language) {
			seen[v] = true
			vars = append(vars, v)
		}
	}

	return vars
}

func isKeyword(word, language string) bool {
	keywords := map[string][]string{
		"go":         {"if", "else", "for", "range", "return", "func", "var", "const", "type", "struct", "interface", "package", "import", "defer", "go", "select", "case", "default", "break", "continue", "fallthrough", "switch", "chan", "map", "make", "new", "nil", "true", "false"},
		"python":     {"if", "else", "elif", "for", "while", "return", "def", "class", "import", "from", "as", "try", "except", "finally", "with", "lambda", "and", "or", "not", "in", "is", "None", "True", "False", "pass", "break", "continue", "raise", "yield", "async", "await"},
		"javascript": {"if", "else", "for", "while", "return", "function", "var", "let", "const", "class", "import", "export", "default", "try", "catch", "finally", "throw", "new", "this", "super", "null", "undefined", "true", "false", "typeof", "instanceof", "async", "await", "yield"},
		"typescript": {"if", "else", "for", "while", "return", "function", "var", "let", "const", "class", "import", "export", "default", "try", "catch", "finally", "throw", "new", "this", "super", "null", "undefined", "true", "false", "typeof", "instanceof", "async", "await", "yield", "interface", "type", "enum", "namespace", "module", "declare", "readonly", "private", "public", "protected"},
	}

	for _, kw := range keywords[language] {
		if word == kw {
			return true
		}
	}
	return false
}

func countLines(s string) int {
	return strings.Count(s, "\n") + 1
}

func normalizeCode(code string) string {
	// Remove extra whitespace
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(strings.TrimSpace(code), " ")
}

func countParams(params string) int {
	params = strings.TrimSpace(params)
	if params == "" {
		return 0
	}
	// Simple count by commas (doesn't handle nested generics perfectly)
	return strings.Count(params, ",") + 1
}

func countIndentDepth(line, language string) int {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return 0
	}

	indent := len(line) - len(trimmed)

	switch language {
	case "python":
		return indent / 4 // Assume 4-space indent
	case "go":
		return strings.Count(line[:indent], "\t")
	default:
		return indent / 2 // Assume 2-space indent
	}
}

func indentCode(code, indent string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}
