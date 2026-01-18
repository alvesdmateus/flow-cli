package analysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SymbolKind represents the type of symbol
type SymbolKind string

const (
	SymbolFunction  SymbolKind = "function"
	SymbolMethod    SymbolKind = "method"
	SymbolType      SymbolKind = "type"
	SymbolStruct    SymbolKind = "struct"
	SymbolInterface SymbolKind = "interface"
	SymbolConst     SymbolKind = "const"
	SymbolVar       SymbolKind = "var"
	SymbolClass     SymbolKind = "class"
	SymbolImport    SymbolKind = "import"
)

// Symbol represents a code symbol (function, type, etc.)
type Symbol struct {
	Name      string     `json:"name"`
	Kind      SymbolKind `json:"kind"`
	File      string     `json:"file"`
	Line      int        `json:"line"`
	EndLine   int        `json:"end_line,omitempty"`
	Column    int        `json:"column,omitempty"`
	Signature string     `json:"signature,omitempty"`
	Doc       string     `json:"doc,omitempty"`
	Parent    string     `json:"parent,omitempty"` // For methods: receiver type
	Exported  bool       `json:"exported"`
	Language  string     `json:"language"`
}

// FileOutline represents the structure of a file
type FileOutline struct {
	File     string   `json:"file"`
	Language string   `json:"language"`
	Symbols  []Symbol `json:"symbols"`
	Imports  []string `json:"imports,omitempty"`
	Package  string   `json:"package,omitempty"`
}

// LanguageParser defines the interface for language-specific parsers
type LanguageParser interface {
	Parse(filename string, content []byte) (*FileOutline, error)
	Language() string
	Extensions() []string
}

// Analyzer performs code analysis
type Analyzer struct {
	parsers map[string]LanguageParser
}

// NewAnalyzer creates a new code analyzer
func NewAnalyzer() *Analyzer {
	a := &Analyzer{
		parsers: make(map[string]LanguageParser),
	}

	// Register built-in parsers
	a.RegisterParser(&GoParser{})
	a.RegisterParser(&PythonParser{})
	a.RegisterParser(&JavaScriptParser{})
	a.RegisterParser(&TypeScriptParser{})
	a.RegisterParser(&RustParser{})

	return a
}

// RegisterParser adds a language parser
func (a *Analyzer) RegisterParser(p LanguageParser) {
	for _, ext := range p.Extensions() {
		a.parsers[ext] = p
	}
}

// GetParser returns the parser for a file extension
func (a *Analyzer) GetParser(ext string) LanguageParser {
	return a.parsers[ext]
}

// ParseFile parses a file and returns its outline
func (a *Analyzer) ParseFile(filename string) (*FileOutline, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	parser := a.parsers[ext]
	if parser == nil {
		return nil, fmt.Errorf("no parser for extension: %s", ext)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return parser.Parse(filename, content)
}

// ParseDirectory parses all supported files in a directory
func (a *Analyzer) ParseDirectory(dir string, recursive bool) ([]*FileOutline, error) {
	var outlines []*FileOutline

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if info.IsDir() {
			if !recursive && path != dir {
				return filepath.SkipDir
			}
			// Skip hidden directories and common non-code directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if a.parsers[ext] != nil {
			outline, err := a.ParseFile(path)
			if err == nil {
				outlines = append(outlines, outline)
			}
		}

		return nil
	}

	if err := filepath.Walk(dir, walkFn); err != nil {
		return nil, err
	}

	return outlines, nil
}

// SymbolResult contains a symbol with its file context
type SymbolResult struct {
	Symbol Symbol `json:"symbol"`
	File   string `json:"file"`
}

// FindSymbol searches for a symbol by name across outlines
func FindSymbol(outlines []*FileOutline, name string) []SymbolResult {
	var results []SymbolResult

	for _, outline := range outlines {
		for _, sym := range outline.Symbols {
			if sym.Name == name {
				// Ensure symbol has file info
				sym.File = outline.File
				results = append(results, SymbolResult{
					Symbol: sym,
					File:   outline.File,
				})
			}
		}
	}

	return results
}

// FindReferences finds all references to a symbol (basic implementation)
func (a *Analyzer) FindReferences(dir string, symbolName string) ([]Symbol, error) {
	var refs []Symbol

	// Simple grep-based reference finding
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if a.parsers[ext] == nil {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.Contains(line, symbolName) {
				refs = append(refs, Symbol{
					Name:     symbolName,
					Kind:     "reference",
					File:     path,
					Line:     i + 1,
					Language: a.parsers[ext].Language(),
				})
			}
		}

		return nil
	})

	return refs, err
}

// --- Go Parser ---

// GoParser parses Go source files
type GoParser struct{}

func (p *GoParser) Language() string {
	return "go"
}

func (p *GoParser) Extensions() []string {
	return []string{".go"}
}

func (p *GoParser) Parse(filename string, content []byte) (*FileOutline, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	outline := &FileOutline{
		File:     filename,
		Language: "go",
		Package:  file.Name.Name,
	}

	// Extract imports
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		outline.Imports = append(outline.Imports, path)
	}

	// Extract symbols
	ast.Inspect(file, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			sym := Symbol{
				Name:     decl.Name.Name,
				Kind:     SymbolFunction,
				File:     filename,
				Line:     fset.Position(decl.Pos()).Line,
				EndLine:  fset.Position(decl.End()).Line,
				Exported: ast.IsExported(decl.Name.Name),
				Language: "go",
			}

			// Check if it's a method
			if decl.Recv != nil && len(decl.Recv.List) > 0 {
				sym.Kind = SymbolMethod
				if t := decl.Recv.List[0].Type; t != nil {
					sym.Parent = exprToString(t)
				}
			}

			// Build signature
			sym.Signature = buildGoFuncSignature(decl)

			// Extract doc comment
			if decl.Doc != nil {
				sym.Doc = strings.TrimSpace(decl.Doc.Text())
			}

			outline.Symbols = append(outline.Symbols, sym)

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sym := Symbol{
						Name:     s.Name.Name,
						Kind:     SymbolType,
						File:     filename,
						Line:     fset.Position(s.Pos()).Line,
						Exported: ast.IsExported(s.Name.Name),
						Language: "go",
					}

					// Determine specific type kind
					switch s.Type.(type) {
					case *ast.StructType:
						sym.Kind = SymbolStruct
					case *ast.InterfaceType:
						sym.Kind = SymbolInterface
					}

					if decl.Doc != nil {
						sym.Doc = strings.TrimSpace(decl.Doc.Text())
					}

					outline.Symbols = append(outline.Symbols, sym)

				case *ast.ValueSpec:
					kind := SymbolVar
					if decl.Tok == token.CONST {
						kind = SymbolConst
					}

					for _, name := range s.Names {
						sym := Symbol{
							Name:     name.Name,
							Kind:     kind,
							File:     filename,
							Line:     fset.Position(name.Pos()).Line,
							Exported: ast.IsExported(name.Name),
							Language: "go",
						}
						outline.Symbols = append(outline.Symbols, sym)
					}
				}
			}
		}
		return true
	})

	return outline, nil
}

// exprToString converts an AST expression to a string
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}

// buildGoFuncSignature builds a function signature string
func buildGoFuncSignature(decl *ast.FuncDecl) string {
	var sb strings.Builder
	sb.WriteString("func ")

	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		sb.WriteString("(")
		sb.WriteString(exprToString(decl.Recv.List[0].Type))
		sb.WriteString(") ")
	}

	sb.WriteString(decl.Name.Name)
	sb.WriteString("(")

	if decl.Type.Params != nil {
		params := []string{}
		for _, field := range decl.Type.Params.List {
			typeStr := exprToString(field.Type)
			if len(field.Names) == 0 {
				params = append(params, typeStr)
			} else {
				for _, name := range field.Names {
					params = append(params, name.Name+" "+typeStr)
				}
			}
		}
		sb.WriteString(strings.Join(params, ", "))
	}

	sb.WriteString(")")

	if decl.Type.Results != nil && len(decl.Type.Results.List) > 0 {
		sb.WriteString(" ")
		if len(decl.Type.Results.List) > 1 {
			sb.WriteString("(")
		}
		results := []string{}
		for _, field := range decl.Type.Results.List {
			results = append(results, exprToString(field.Type))
		}
		sb.WriteString(strings.Join(results, ", "))
		if len(decl.Type.Results.List) > 1 {
			sb.WriteString(")")
		}
	}

	return sb.String()
}

// --- Python Parser (regex-based) ---

// PythonParser parses Python source files
type PythonParser struct{}

func (p *PythonParser) Language() string {
	return "python"
}

func (p *PythonParser) Extensions() []string {
	return []string{".py"}
}

var (
	pyFuncPattern   = regexp.MustCompile(`(?m)^(\s*)def\s+(\w+)\s*\(([^)]*)\)`)
	pyClassPattern  = regexp.MustCompile(`(?m)^class\s+(\w+)(?:\s*\([^)]*\))?:`)
	pyImportPattern = regexp.MustCompile(`(?m)^(?:from\s+\S+\s+)?import\s+(.+)$`)
)

func (p *PythonParser) Parse(filename string, content []byte) (*FileOutline, error) {
	outline := &FileOutline{
		File:     filename,
		Language: "python",
	}

	lines := strings.Split(string(content), "\n")
	var currentClass string

	for i, line := range lines {
		lineNum := i + 1

		// Check for class definition
		if matches := pyClassPattern.FindStringSubmatch(line); matches != nil {
			currentClass = matches[1]
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolClass,
				File:     filename,
				Line:     lineNum,
				Exported: !strings.HasPrefix(matches[1], "_"),
				Language: "python",
			})
			continue
		}

		// Check for function/method definition
		if matches := pyFuncPattern.FindStringSubmatch(line); matches != nil {
			indent := matches[1]
			name := matches[2]
			params := matches[3]

			sym := Symbol{
				Name:      name,
				Kind:      SymbolFunction,
				File:      filename,
				Line:      lineNum,
				Signature: fmt.Sprintf("def %s(%s)", name, params),
				Exported:  !strings.HasPrefix(name, "_"),
				Language:  "python",
			}

			// If indented, it's likely a method
			if len(indent) > 0 && currentClass != "" {
				sym.Kind = SymbolMethod
				sym.Parent = currentClass
			}

			outline.Symbols = append(outline.Symbols, sym)
			continue
		}

		// Check for imports
		if matches := pyImportPattern.FindStringSubmatch(line); matches != nil {
			outline.Imports = append(outline.Imports, strings.TrimSpace(matches[1]))
		}

		// Reset class context on unindented non-empty line
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(line, "#") {
			if !strings.HasPrefix(line, "class ") && !strings.HasPrefix(line, "def ") {
				currentClass = ""
			}
		}
	}

	return outline, nil
}

// --- JavaScript Parser (regex-based) ---

// JavaScriptParser parses JavaScript source files
type JavaScriptParser struct{}

func (p *JavaScriptParser) Language() string {
	return "javascript"
}

func (p *JavaScriptParser) Extensions() []string {
	return []string{".js", ".jsx", ".mjs"}
}

var (
	jsFuncPattern   = regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)`)
	jsArrowPattern  = regexp.MustCompile(`(?m)^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\([^)]*\)\s*=>`)
	jsClassPattern  = regexp.MustCompile(`(?m)^(?:export\s+)?class\s+(\w+)`)
	jsMethodPattern = regexp.MustCompile(`(?m)^\s+(?:async\s+)?(\w+)\s*\([^)]*\)\s*\{`)
	jsImportPattern = regexp.MustCompile(`(?m)^import\s+.+\s+from\s+['"]([^'"]+)['"]`)
)

func (p *JavaScriptParser) Parse(filename string, content []byte) (*FileOutline, error) {
	outline := &FileOutline{
		File:     filename,
		Language: "javascript",
	}

	lines := strings.Split(string(content), "\n")
	var inClass bool
	var currentClass string

	for i, line := range lines {
		lineNum := i + 1

		// Check for class
		if matches := jsClassPattern.FindStringSubmatch(line); matches != nil {
			inClass = true
			currentClass = matches[1]
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolClass,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(line, "export"),
				Language: "javascript",
			})
			continue
		}

		// Check for function
		if matches := jsFuncPattern.FindStringSubmatch(line); matches != nil {
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:      matches[1],
				Kind:      SymbolFunction,
				File:      filename,
				Line:      lineNum,
				Signature: fmt.Sprintf("function %s(%s)", matches[1], matches[2]),
				Exported:  strings.HasPrefix(line, "export"),
				Language:  "javascript",
			})
			continue
		}

		// Check for arrow function
		if matches := jsArrowPattern.FindStringSubmatch(line); matches != nil {
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolFunction,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(line, "export"),
				Language: "javascript",
			})
			continue
		}

		// Check for method (when in class)
		if inClass {
			if matches := jsMethodPattern.FindStringSubmatch(line); matches != nil {
				name := matches[1]
				if name != "constructor" && name != "if" && name != "for" && name != "while" {
					outline.Symbols = append(outline.Symbols, Symbol{
						Name:     name,
						Kind:     SymbolMethod,
						File:     filename,
						Line:     lineNum,
						Parent:   currentClass,
						Language: "javascript",
					})
				}
			}
		}

		// Check for imports
		if matches := jsImportPattern.FindStringSubmatch(line); matches != nil {
			outline.Imports = append(outline.Imports, matches[1])
		}

		// Simple class end detection
		if inClass && strings.TrimSpace(line) == "}" {
			inClass = false
			currentClass = ""
		}
	}

	return outline, nil
}

// --- TypeScript Parser (extends JavaScript) ---

// TypeScriptParser parses TypeScript source files
type TypeScriptParser struct {
	JavaScriptParser
}

func (p *TypeScriptParser) Language() string {
	return "typescript"
}

func (p *TypeScriptParser) Extensions() []string {
	return []string{".ts", ".tsx"}
}

var (
	tsInterfacePattern = regexp.MustCompile(`(?m)^(?:export\s+)?interface\s+(\w+)`)
	tsTypePattern      = regexp.MustCompile(`(?m)^(?:export\s+)?type\s+(\w+)\s*=`)
)

func (p *TypeScriptParser) Parse(filename string, content []byte) (*FileOutline, error) {
	// Start with JavaScript parsing
	outline, err := p.JavaScriptParser.Parse(filename, content)
	if err != nil {
		return nil, err
	}

	outline.Language = "typescript"

	// Add TypeScript-specific patterns
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		lineNum := i + 1

		// Check for interface
		if matches := tsInterfacePattern.FindStringSubmatch(line); matches != nil {
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolInterface,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(line, "export"),
				Language: "typescript",
			})
		}

		// Check for type alias
		if matches := tsTypePattern.FindStringSubmatch(line); matches != nil {
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolType,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(line, "export"),
				Language: "typescript",
			})
		}
	}

	return outline, nil
}

// --- Rust Parser (regex-based) ---

// RustParser parses Rust source files
type RustParser struct{}

func (p *RustParser) Language() string {
	return "rust"
}

func (p *RustParser) Extensions() []string {
	return []string{".rs"}
}

var (
	rustFnPattern     = regexp.MustCompile(`(?m)^(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`)
	rustStructPattern = regexp.MustCompile(`(?m)^(?:pub\s+)?struct\s+(\w+)`)
	rustEnumPattern   = regexp.MustCompile(`(?m)^(?:pub\s+)?enum\s+(\w+)`)
	rustTraitPattern  = regexp.MustCompile(`(?m)^(?:pub\s+)?trait\s+(\w+)`)
	rustImplPattern   = regexp.MustCompile(`(?m)^impl(?:<[^>]+>)?\s+(?:(\w+)\s+for\s+)?(\w+)`)
	rustUsePattern    = regexp.MustCompile(`(?m)^use\s+(.+);`)
)

func (p *RustParser) Parse(filename string, content []byte) (*FileOutline, error) {
	outline := &FileOutline{
		File:     filename,
		Language: "rust",
	}

	lines := strings.Split(string(content), "\n")
	var currentImpl string

	for i, line := range lines {
		lineNum := i + 1

		// Check for impl block
		if matches := rustImplPattern.FindStringSubmatch(line); matches != nil {
			currentImpl = matches[2]
			continue
		}

		// Check for function
		if matches := rustFnPattern.FindStringSubmatch(line); matches != nil {
			sym := Symbol{
				Name:     matches[1],
				Kind:     SymbolFunction,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(strings.TrimSpace(line), "pub"),
				Language: "rust",
			}
			if currentImpl != "" {
				sym.Kind = SymbolMethod
				sym.Parent = currentImpl
			}
			outline.Symbols = append(outline.Symbols, sym)
			continue
		}

		// Check for struct
		if matches := rustStructPattern.FindStringSubmatch(line); matches != nil {
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolStruct,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(strings.TrimSpace(line), "pub"),
				Language: "rust",
			})
			continue
		}

		// Check for enum
		if matches := rustEnumPattern.FindStringSubmatch(line); matches != nil {
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolType,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(strings.TrimSpace(line), "pub"),
				Language: "rust",
			})
			continue
		}

		// Check for trait
		if matches := rustTraitPattern.FindStringSubmatch(line); matches != nil {
			outline.Symbols = append(outline.Symbols, Symbol{
				Name:     matches[1],
				Kind:     SymbolInterface,
				File:     filename,
				Line:     lineNum,
				Exported: strings.HasPrefix(strings.TrimSpace(line), "pub"),
				Language: "rust",
			})
			continue
		}

		// Check for use (imports)
		if matches := rustUsePattern.FindStringSubmatch(line); matches != nil {
			outline.Imports = append(outline.Imports, matches[1])
		}

		// Reset impl context on closing brace at start of line
		if strings.TrimSpace(line) == "}" {
			currentImpl = ""
		}
	}

	return outline, nil
}
