package codegen

import (
	"fmt"
	"regexp"
	"strings"
)

// BugFixSuggester analyzes error messages and suggests fixes.
type BugFixSuggester struct {
	patterns map[string][]ErrorPattern
}

// ErrorPattern represents a pattern for matching errors.
type ErrorPattern struct {
	Pattern     *regexp.Regexp
	Description string
	SuggestFix  func(matches []string, context string) *BugFixSuggestion
}

// BugFixSuggestion represents a suggested fix for an error.
type BugFixSuggestion struct {
	ErrorType   string
	Description string
	Explanation string
	Fixes       []ProposedFix
	References  []string
}

// ProposedFix represents a specific fix option.
type ProposedFix struct {
	Description string
	Code        string
	Confidence  float64 // 0.0 to 1.0
	IsPreferred bool
}

// NewBugFixSuggester creates a new bug fix suggester.
func NewBugFixSuggester() *BugFixSuggester {
	bfs := &BugFixSuggester{
		patterns: make(map[string][]ErrorPattern),
	}

	// Register patterns for different languages
	bfs.registerGoPatterns()
	bfs.registerPythonPatterns()
	bfs.registerJavaScriptPatterns()
	bfs.registerTypeScriptPatterns()
	bfs.registerGenericPatterns()

	return bfs
}

// SuggestFix analyzes an error message and suggests fixes.
func (bfs *BugFixSuggester) SuggestFix(errorMsg, language, codeContext string) *BugFixSuggestion {
	// Try language-specific patterns first
	if patterns, ok := bfs.patterns[language]; ok {
		for _, pattern := range patterns {
			if matches := pattern.Pattern.FindStringSubmatch(errorMsg); matches != nil {
				return pattern.SuggestFix(matches, codeContext)
			}
		}
	}

	// Fall back to generic patterns
	for _, pattern := range bfs.patterns["generic"] {
		if matches := pattern.Pattern.FindStringSubmatch(errorMsg); matches != nil {
			return pattern.SuggestFix(matches, codeContext)
		}
	}

	return nil
}

// SuggestMultipleFixes analyzes multiple error messages.
func (bfs *BugFixSuggester) SuggestMultipleFixes(errors []ErrorInfo, codeContext string) []*BugFixSuggestion {
	var suggestions []*BugFixSuggestion

	for _, err := range errors {
		if suggestion := bfs.SuggestFix(err.Message, err.Language, codeContext); suggestion != nil {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// ErrorInfo contains information about an error.
type ErrorInfo struct {
	Message  string
	Language string
	File     string
	Line     int
	Column   int
}

// Go error patterns
func (bfs *BugFixSuggester) registerGoPatterns() {
	bfs.patterns["go"] = []ErrorPattern{
		{
			Pattern:     regexp.MustCompile(`undefined: (\w+)`),
			Description: "Undefined identifier",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				identifier := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "undefined_identifier",
					Description: fmt.Sprintf("'%s' is not defined", identifier),
					Explanation: "The identifier is used but not declared in the current scope.",
					Fixes: []ProposedFix{
						{
							Description: fmt.Sprintf("Declare variable '%s'", identifier),
							Code:        fmt.Sprintf("var %s TYPE", identifier),
							Confidence:  0.7,
						},
						{
							Description: fmt.Sprintf("Import package containing '%s'", identifier),
							Code:        fmt.Sprintf("import \"package/%s\"", strings.ToLower(identifier)),
							Confidence:  0.5,
						},
						{
							Description: "Check for typo in identifier name",
							Code:        "",
							Confidence:  0.8,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`cannot use (.+) \(.*type (.+)\) as type (.+)`),
			Description: "Type mismatch",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				value, actualType, expectedType := matches[1], matches[2], matches[3]
				return &BugFixSuggestion{
					ErrorType:   "type_mismatch",
					Description: fmt.Sprintf("Cannot use %s (type %s) as type %s", value, actualType, expectedType),
					Explanation: "The value type doesn't match the expected type.",
					Fixes: []ProposedFix{
						{
							Description: fmt.Sprintf("Convert to %s", expectedType),
							Code:        fmt.Sprintf("%s(%s)", expectedType, value),
							Confidence:  0.8,
							IsPreferred: true,
						},
						{
							Description: "Change the variable declaration type",
							Code:        fmt.Sprintf("var x %s = ...", actualType),
							Confidence:  0.6,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`imported and not used: "(.+)"`),
			Description: "Unused import",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				pkg := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "unused_import",
					Description: fmt.Sprintf("Package '%s' is imported but not used", pkg),
					Explanation: "Go requires all imports to be used.",
					Fixes: []ProposedFix{
						{
							Description: "Remove the unused import",
							Code:        fmt.Sprintf("// Remove: import \"%s\"", pkg),
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Use blank identifier to ignore",
							Code:        fmt.Sprintf("import _ \"%s\"", pkg),
							Confidence:  0.7,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`(\w+) declared (?:but|and) not used`),
			Description: "Unused variable",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				varName := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "unused_variable",
					Description: fmt.Sprintf("Variable '%s' is declared but not used", varName),
					Explanation: "Go requires all declared variables to be used.",
					Fixes: []ProposedFix{
						{
							Description: "Remove the unused variable",
							Code:        fmt.Sprintf("// Remove: %s := ...", varName),
							Confidence:  0.8,
						},
						{
							Description: "Use blank identifier",
							Code:        fmt.Sprintf("_ = %s", varName),
							Confidence:  0.9,
							IsPreferred: true,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`cannot assign to (.+)`),
			Description: "Cannot assign",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				target := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "invalid_assignment",
					Description: fmt.Sprintf("Cannot assign to '%s'", target),
					Explanation: "The target of the assignment is not assignable (may be a constant, function, or immutable value).",
					Fixes: []ProposedFix{
						{
							Description: "Use a mutable variable instead",
							Code:        fmt.Sprintf("var mutable%s = %s // then modify mutable%s", strings.Title(target), target, strings.Title(target)),
							Confidence:  0.7,
						},
						{
							Description: "Check if you need a pointer",
							Code:        fmt.Sprintf("*%s = value // if %s is a pointer", target, target),
							Confidence:  0.6,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`nil pointer dereference`),
			Description: "Nil pointer dereference",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "nil_pointer",
					Description: "Attempted to dereference a nil pointer",
					Explanation: "A pointer was nil when you tried to access its value.",
					Fixes: []ProposedFix{
						{
							Description: "Add nil check before dereferencing",
							Code:        "if ptr != nil {\n    // use *ptr\n}",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Initialize the pointer before use",
							Code:        "ptr = &SomeStruct{}",
							Confidence:  0.8,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`index out of range \[(\d+)\] with length (\d+)`),
			Description: "Index out of range",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				index, length := matches[1], matches[2]
				return &BugFixSuggestion{
					ErrorType:   "index_out_of_range",
					Description: fmt.Sprintf("Index %s is out of range (length %s)", index, length),
					Explanation: "You tried to access an element beyond the slice/array bounds.",
					Fixes: []ProposedFix{
						{
							Description: "Check bounds before accessing",
							Code:        fmt.Sprintf("if i < len(slice) {\n    value := slice[i]\n}"),
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Use a range loop instead",
							Code:        "for i, v := range slice {\n    // use v\n}",
							Confidence:  0.8,
						},
					},
				}
			},
		},
	}
}

// Python error patterns
func (bfs *BugFixSuggester) registerPythonPatterns() {
	bfs.patterns["python"] = []ErrorPattern{
		{
			Pattern:     regexp.MustCompile(`NameError: name '(\w+)' is not defined`),
			Description: "Name not defined",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				name := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "name_error",
					Description: fmt.Sprintf("Name '%s' is not defined", name),
					Explanation: "The variable or function is used before it's defined.",
					Fixes: []ProposedFix{
						{
							Description: fmt.Sprintf("Define '%s' before using it", name),
							Code:        fmt.Sprintf("%s = None  # or appropriate value", name),
							Confidence:  0.7,
						},
						{
							Description: "Import the module containing this name",
							Code:        fmt.Sprintf("from module import %s", name),
							Confidence:  0.6,
						},
						{
							Description: "Check for typo in the name",
							Code:        "",
							Confidence:  0.8,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`TypeError: (.+) takes (\d+) positional arguments? but (\d+) (?:was|were) given`),
			Description: "Wrong number of arguments",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				funcName := matches[1]
				expected, given := matches[2], matches[3]
				return &BugFixSuggestion{
					ErrorType:   "argument_count",
					Description: fmt.Sprintf("%s takes %s arguments but %s were given", funcName, expected, given),
					Explanation: "The function was called with the wrong number of arguments.",
					Fixes: []ProposedFix{
						{
							Description: fmt.Sprintf("Call with %s arguments", expected),
							Code:        fmt.Sprintf("%s(arg1, arg2, ...)  # %s arguments", funcName, expected),
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Check the function signature",
							Code:        fmt.Sprintf("help(%s)  # to see expected arguments", funcName),
							Confidence:  0.7,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`IndentationError: (.+)`),
			Description: "Indentation error",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				detail := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "indentation_error",
					Description: fmt.Sprintf("Indentation error: %s", detail),
					Explanation: "Python uses indentation to define code blocks.",
					Fixes: []ProposedFix{
						{
							Description: "Use consistent 4-space indentation",
							Code:        "# Use 4 spaces for each indentation level",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Convert tabs to spaces",
							Code:        "# Run: python -m tabnanny file.py",
							Confidence:  0.8,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`AttributeError: '(\w+)' object has no attribute '(\w+)'`),
			Description: "Missing attribute",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				objType, attr := matches[1], matches[2]
				return &BugFixSuggestion{
					ErrorType:   "attribute_error",
					Description: fmt.Sprintf("'%s' object has no attribute '%s'", objType, attr),
					Explanation: "The object doesn't have the attribute or method you're trying to access.",
					Fixes: []ProposedFix{
						{
							Description: "Check if the attribute name is correct",
							Code:        fmt.Sprintf("dir(obj)  # List all attributes of %s", objType),
							Confidence:  0.8,
						},
						{
							Description: "Use getattr with a default",
							Code:        fmt.Sprintf("getattr(obj, '%s', default_value)", attr),
							Confidence:  0.7,
						},
						{
							Description: "Use hasattr to check first",
							Code:        fmt.Sprintf("if hasattr(obj, '%s'):\n    obj.%s", attr, attr),
							Confidence:  0.7,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`KeyError: '?(.+)'?`),
			Description: "Key not found",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				key := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "key_error",
					Description: fmt.Sprintf("Key '%s' not found in dictionary", key),
					Explanation: "The key doesn't exist in the dictionary.",
					Fixes: []ProposedFix{
						{
							Description: "Use .get() with a default value",
							Code:        fmt.Sprintf("value = dict.get('%s', default_value)", key),
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Check if key exists first",
							Code:        fmt.Sprintf("if '%s' in dict:\n    value = dict['%s']", key, key),
							Confidence:  0.8,
						},
						{
							Description: "Use setdefault to provide default",
							Code:        fmt.Sprintf("value = dict.setdefault('%s', default_value)", key),
							Confidence:  0.7,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`IndexError: list index out of range`),
			Description: "List index out of range",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "index_error",
					Description: "List index is out of range",
					Explanation: "You tried to access a list element that doesn't exist.",
					Fixes: []ProposedFix{
						{
							Description: "Check list length before accessing",
							Code:        "if i < len(my_list):\n    value = my_list[i]",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Use try/except to handle gracefully",
							Code:        "try:\n    value = my_list[i]\nexcept IndexError:\n    value = default",
							Confidence:  0.8,
						},
					},
				}
			},
		},
	}
}

// JavaScript error patterns
func (bfs *BugFixSuggester) registerJavaScriptPatterns() {
	bfs.patterns["javascript"] = []ErrorPattern{
		{
			Pattern:     regexp.MustCompile(`ReferenceError: (\w+) is not defined`),
			Description: "Reference error",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				name := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "reference_error",
					Description: fmt.Sprintf("'%s' is not defined", name),
					Explanation: "The variable or function hasn't been declared.",
					Fixes: []ProposedFix{
						{
							Description: fmt.Sprintf("Declare '%s' with let or const", name),
							Code:        fmt.Sprintf("const %s = value;", name),
							Confidence:  0.8,
							IsPreferred: true,
						},
						{
							Description: "Import the module",
							Code:        fmt.Sprintf("import { %s } from 'module';", name),
							Confidence:  0.6,
						},
						{
							Description: "Check for typo in the name",
							Code:        "",
							Confidence:  0.7,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`TypeError: Cannot read propert(?:y|ies) (?:of|'(\w+)' of) (null|undefined)`),
			Description: "Cannot read property of null/undefined",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "null_property_access",
					Description: "Cannot read property of null or undefined",
					Explanation: "You tried to access a property on a value that is null or undefined.",
					Fixes: []ProposedFix{
						{
							Description: "Use optional chaining",
							Code:        "const value = obj?.property;",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Add null check",
							Code:        "if (obj != null) {\n  const value = obj.property;\n}",
							Confidence:  0.8,
						},
						{
							Description: "Use nullish coalescing with default",
							Code:        "const value = obj?.property ?? defaultValue;",
							Confidence:  0.9,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`TypeError: (\w+) is not a function`),
			Description: "Not a function",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				name := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "not_a_function",
					Description: fmt.Sprintf("'%s' is not a function", name),
					Explanation: "You tried to call something that isn't a function.",
					Fixes: []ProposedFix{
						{
							Description: "Check the value type before calling",
							Code:        fmt.Sprintf("if (typeof %s === 'function') {\n  %s();\n}", name, name),
							Confidence:  0.8,
						},
						{
							Description: "Check the import/require statement",
							Code:        fmt.Sprintf("// Verify: import { %s } from 'module';", name),
							Confidence:  0.7,
						},
						{
							Description: "Check if accessing from the correct object",
							Code:        fmt.Sprintf("object.%s(); // if %s is a method", name, name),
							Confidence:  0.6,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`SyntaxError: Unexpected token (\S+)`),
			Description: "Syntax error",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				token := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "syntax_error",
					Description: fmt.Sprintf("Unexpected token '%s'", token),
					Explanation: "There's a syntax error in your code.",
					Fixes: []ProposedFix{
						{
							Description: "Check for missing brackets or parentheses",
							Code:        "// Verify matching: { }, ( ), [ ]",
							Confidence:  0.8,
						},
						{
							Description: "Check for missing semicolons or commas",
							Code:        "// Check previous line for missing ; or ,",
							Confidence:  0.7,
						},
						{
							Description: "Verify JSON syntax if parsing JSON",
							Code:        "// Use JSON.parse() with try/catch",
							Confidence:  0.6,
						},
					},
				}
			},
		},
	}
}

// TypeScript error patterns
func (bfs *BugFixSuggester) registerTypeScriptPatterns() {
	bfs.patterns["typescript"] = []ErrorPattern{
		{
			Pattern:     regexp.MustCompile(`Type '(.+)' is not assignable to type '(.+)'`),
			Description: "Type not assignable",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				actualType, expectedType := matches[1], matches[2]
				return &BugFixSuggestion{
					ErrorType:   "type_mismatch",
					Description: fmt.Sprintf("Type '%s' is not assignable to type '%s'", actualType, expectedType),
					Explanation: "The value's type doesn't match the expected type.",
					Fixes: []ProposedFix{
						{
							Description: "Add type assertion",
							Code:        fmt.Sprintf("value as %s", expectedType),
							Confidence:  0.7,
						},
						{
							Description: "Change the variable type",
							Code:        fmt.Sprintf("const x: %s = value;", actualType),
							Confidence:  0.6,
						},
						{
							Description: "Fix the value to match expected type",
							Code:        fmt.Sprintf("// Ensure value is of type %s", expectedType),
							Confidence:  0.8,
							IsPreferred: true,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`Property '(\w+)' does not exist on type '(.+)'`),
			Description: "Property doesn't exist",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				prop, typeName := matches[1], matches[2]
				return &BugFixSuggestion{
					ErrorType:   "missing_property",
					Description: fmt.Sprintf("Property '%s' does not exist on type '%s'", prop, typeName),
					Explanation: "You're accessing a property that doesn't exist on the type.",
					Fixes: []ProposedFix{
						{
							Description: "Add the property to the type definition",
							Code:        fmt.Sprintf("interface %s {\n  %s: unknown;\n}", typeName, prop),
							Confidence:  0.7,
						},
						{
							Description: "Use type assertion",
							Code:        fmt.Sprintf("(obj as any).%s", prop),
							Confidence:  0.5,
						},
						{
							Description: "Check if the property name is correct",
							Code:        "",
							Confidence:  0.8,
							IsPreferred: true,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`Object is possibly '(null|undefined)'`),
			Description: "Possibly null/undefined",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				nullType := matches[1]
				return &BugFixSuggestion{
					ErrorType:   "possibly_null",
					Description: fmt.Sprintf("Object is possibly '%s'", nullType),
					Explanation: "TypeScript detected that this value might be null or undefined.",
					Fixes: []ProposedFix{
						{
							Description: "Add null check",
							Code:        "if (obj != null) {\n  // use obj\n}",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Use optional chaining",
							Code:        "obj?.property",
							Confidence:  0.9,
						},
						{
							Description: "Use non-null assertion (if certain)",
							Code:        "obj!.property",
							Confidence:  0.6,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`Argument of type '(.+)' is not assignable to parameter of type '(.+)'`),
			Description: "Argument type mismatch",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				actualType, expectedType := matches[1], matches[2]
				return &BugFixSuggestion{
					ErrorType:   "argument_type_mismatch",
					Description: fmt.Sprintf("Argument of type '%s' is not assignable to parameter of type '%s'", actualType, expectedType),
					Explanation: "The function argument type doesn't match the parameter type.",
					Fixes: []ProposedFix{
						{
							Description: "Convert the argument to the expected type",
							Code:        fmt.Sprintf("func(arg as %s)", expectedType),
							Confidence:  0.7,
						},
						{
							Description: "Fix the argument value",
							Code:        fmt.Sprintf("// Ensure argument matches type %s", expectedType),
							Confidence:  0.8,
							IsPreferred: true,
						},
						{
							Description: "Update the function parameter type",
							Code:        fmt.Sprintf("function func(param: %s | %s)", expectedType, actualType),
							Confidence:  0.6,
						},
					},
				}
			},
		},
	}
}

// Generic patterns that apply to multiple languages
func (bfs *BugFixSuggester) registerGenericPatterns() {
	bfs.patterns["generic"] = []ErrorPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)division by zero`),
			Description: "Division by zero",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "division_by_zero",
					Description: "Division by zero error",
					Explanation: "You tried to divide a number by zero.",
					Fixes: []ProposedFix{
						{
							Description: "Check if divisor is zero before dividing",
							Code:        "if (divisor != 0) {\n  result = dividend / divisor;\n}",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Use a default value when divisor is zero",
							Code:        "result = divisor != 0 ? dividend / divisor : 0;",
							Confidence:  0.8,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)stack overflow`),
			Description: "Stack overflow",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "stack_overflow",
					Description: "Stack overflow error",
					Explanation: "Usually caused by infinite recursion or very deep recursion.",
					Fixes: []ProposedFix{
						{
							Description: "Add a base case to recursive function",
							Code:        "if (baseCondition) {\n  return baseValue;\n}",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Convert recursion to iteration",
							Code:        "// Use a loop instead of recursive calls",
							Confidence:  0.8,
						},
						{
							Description: "Increase stack size (if appropriate)",
							Code:        "// Language-specific stack size configuration",
							Confidence:  0.4,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)out of memory|memory allocation failed`),
			Description: "Out of memory",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "out_of_memory",
					Description: "Out of memory error",
					Explanation: "The application ran out of available memory.",
					Fixes: []ProposedFix{
						{
							Description: "Process data in smaller chunks",
							Code:        "// Use streaming or pagination",
							Confidence:  0.8,
							IsPreferred: true,
						},
						{
							Description: "Release unused resources",
							Code:        "// Close files, clear caches, etc.",
							Confidence:  0.7,
						},
						{
							Description: "Optimize data structures",
							Code:        "// Use more memory-efficient structures",
							Confidence:  0.6,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)timeout|timed out|deadline exceeded`),
			Description: "Timeout",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "timeout",
					Description: "Operation timed out",
					Explanation: "The operation took longer than the allowed time.",
					Fixes: []ProposedFix{
						{
							Description: "Increase timeout value",
							Code:        "// Set a higher timeout: timeout: 30000",
							Confidence:  0.7,
						},
						{
							Description: "Optimize the slow operation",
							Code:        "// Profile and optimize the code",
							Confidence:  0.8,
							IsPreferred: true,
						},
						{
							Description: "Add retry with exponential backoff",
							Code:        "// Retry with increasing delays",
							Confidence:  0.7,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)permission denied|access denied|unauthorized`),
			Description: "Permission denied",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "permission_denied",
					Description: "Permission denied",
					Explanation: "The operation requires permissions that aren't available.",
					Fixes: []ProposedFix{
						{
							Description: "Check file/resource permissions",
							Code:        "// Verify read/write permissions",
							Confidence:  0.8,
							IsPreferred: true,
						},
						{
							Description: "Run with elevated privileges",
							Code:        "// Run as admin/root if appropriate",
							Confidence:  0.5,
						},
						{
							Description: "Check authentication credentials",
							Code:        "// Verify API keys, tokens, etc.",
							Confidence:  0.7,
						},
					},
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)connection refused|ECONNREFUSED`),
			Description: "Connection refused",
			SuggestFix: func(matches []string, context string) *BugFixSuggestion {
				return &BugFixSuggestion{
					ErrorType:   "connection_refused",
					Description: "Connection refused",
					Explanation: "The server refused the connection or isn't running.",
					Fixes: []ProposedFix{
						{
							Description: "Verify the server is running",
							Code:        "// Check if the service is started",
							Confidence:  0.9,
							IsPreferred: true,
						},
						{
							Description: "Check host and port configuration",
							Code:        "// Verify: host=localhost, port=8080",
							Confidence:  0.8,
						},
						{
							Description: "Check firewall settings",
							Code:        "// Ensure the port is not blocked",
							Confidence:  0.6,
						},
					},
				}
			},
		},
	}
}

// AnalyzeCompilerOutput parses compiler output and extracts errors.
func AnalyzeCompilerOutput(output, language string) []ErrorInfo {
	var errors []ErrorInfo

	var patterns map[string]*regexp.Regexp
	switch language {
	case "go":
		patterns = map[string]*regexp.Regexp{
			"error": regexp.MustCompile(`(.+):(\d+):(\d+): (.+)`),
		}
	case "python":
		patterns = map[string]*regexp.Regexp{
			"error": regexp.MustCompile(`File "(.+)", line (\d+).+\n.+\n(\w+Error: .+)`),
		}
	case "javascript", "typescript":
		patterns = map[string]*regexp.Regexp{
			"error": regexp.MustCompile(`(.+)\((\d+),(\d+)\): error .+: (.+)`),
		}
	default:
		return errors
	}

	for name, pattern := range patterns {
		if name == "error" {
			matches := pattern.FindAllStringSubmatch(output, -1)
			for _, match := range matches {
				var info ErrorInfo
				info.Language = language

				switch language {
				case "go":
					info.File = match[1]
					fmt.Sscanf(match[2], "%d", &info.Line)
					fmt.Sscanf(match[3], "%d", &info.Column)
					info.Message = match[4]
				case "python":
					info.File = match[1]
					fmt.Sscanf(match[2], "%d", &info.Line)
					info.Message = match[3]
				case "javascript", "typescript":
					info.File = match[1]
					fmt.Sscanf(match[2], "%d", &info.Line)
					fmt.Sscanf(match[3], "%d", &info.Column)
					info.Message = match[4]
				}

				errors = append(errors, info)
			}
		}
	}

	return errors
}
