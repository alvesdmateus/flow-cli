package codegen

import (
	"fmt"
	"strings"
)

// DocGenerator generates documentation from source code.
type DocGenerator struct {
	templates map[string]*DocTemplate
}

// DocTemplate defines documentation format for a language.
type DocTemplate struct {
	Language    string
	FuncDoc     string // Template for function documentation
	ClassDoc    string // Template for class documentation
	ParamDoc    string // Template for parameter documentation
	ReturnDoc   string // Template for return documentation
	ExampleDoc  string // Template for example documentation
	FileDoc     string // Template for file-level documentation
	CommentStart string
	CommentEnd   string
	CommentLine  string
}

// DocStyle represents documentation style.
type DocStyle string

const (
	DocStyleGoDoc  DocStyle = "godoc"
	DocStyleJSDoc  DocStyle = "jsdoc"
	DocStylePyDoc  DocStyle = "pydoc"
	DocStyleRustDoc DocStyle = "rustdoc"
)

// GeneratedDoc represents generated documentation.
type GeneratedDoc struct {
	Language    string
	Style       DocStyle
	Content     string
	Functions   []FunctionDoc
	Classes     []ClassDoc
	FileDoc     string
}

// FunctionDoc represents documentation for a function.
type FunctionDoc struct {
	Name        string
	Description string
	Params      []ParamDoc
	Returns     []ReturnDoc
	Examples    []string
	Throws      []string
	Deprecated  string
	Since       string
	See         []string
	Raw         string
}

// ParamDoc represents documentation for a parameter.
type ParamDoc struct {
	Name        string
	Type        string
	Description string
	Optional    bool
	Default     string
}

// ReturnDoc represents documentation for a return value.
type ReturnDoc struct {
	Type        string
	Description string
}

// ClassDoc represents documentation for a class.
type ClassDoc struct {
	Name        string
	Description string
	Properties  []PropertyDoc
	Methods     []FunctionDoc
	Examples    []string
}

// PropertyDoc represents documentation for a property.
type PropertyDoc struct {
	Name        string
	Type        string
	Description string
	ReadOnly    bool
}

// NewDocGenerator creates a new documentation generator.
func NewDocGenerator() *DocGenerator {
	dg := &DocGenerator{
		templates: make(map[string]*DocTemplate),
	}

	// Register default templates
	dg.RegisterTemplate("go", goDocTemplate())
	dg.RegisterTemplate("python", pythonDocTemplate())
	dg.RegisterTemplate("javascript", jsDocTemplate())
	dg.RegisterTemplate("typescript", tsDocTemplate())
	dg.RegisterTemplate("rust", rustDocTemplate())

	return dg
}

// RegisterTemplate registers a documentation template.
func (dg *DocGenerator) RegisterTemplate(lang string, template *DocTemplate) {
	dg.templates[lang] = template
}

// GenerateFunctionDoc generates documentation for a function.
func (dg *DocGenerator) GenerateFunctionDoc(fn FunctionInfo, language string) (string, error) {
	template, ok := dg.templates[language]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	doc := FunctionDoc{
		Name:        fn.Name,
		Description: generateDescription(fn.Name, "function"),
		Params:      make([]ParamDoc, 0),
		Returns:     make([]ReturnDoc, 0),
	}

	// Generate param docs
	for _, param := range fn.Params {
		doc.Params = append(doc.Params, ParamDoc{
			Name:        param.Name,
			Type:        param.Type,
			Description: generateParamDescription(param.Name, param.Type),
		})
	}

	// Generate return docs
	for _, ret := range fn.Returns {
		doc.Returns = append(doc.Returns, ReturnDoc{
			Type:        ret,
			Description: "The result of the operation",
		})
	}

	return dg.renderFunctionDoc(doc, template), nil
}

// GenerateClassDoc generates documentation for a class.
func (dg *DocGenerator) GenerateClassDoc(cls ClassInfo, language string) (string, error) {
	template, ok := dg.templates[language]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	doc := ClassDoc{
		Name:        cls.Name,
		Description: generateDescription(cls.Name, "class"),
		Properties:  make([]PropertyDoc, 0),
		Methods:     make([]FunctionDoc, 0),
	}

	// Generate property docs
	for _, prop := range cls.Properties {
		doc.Properties = append(doc.Properties, PropertyDoc{
			Name:        prop.Name,
			Type:        prop.Type,
			Description: generateParamDescription(prop.Name, prop.Type),
		})
	}

	// Generate method docs
	for _, method := range cls.Methods {
		methodDoc := FunctionDoc{
			Name:        method.Name,
			Description: generateDescription(method.Name, "method"),
		}
		for _, param := range method.Params {
			methodDoc.Params = append(methodDoc.Params, ParamDoc{
				Name:        param.Name,
				Type:        param.Type,
				Description: generateParamDescription(param.Name, param.Type),
			})
		}
		doc.Methods = append(doc.Methods, methodDoc)
	}

	return dg.renderClassDoc(doc, template), nil
}

func (dg *DocGenerator) renderFunctionDoc(doc FunctionDoc, template *DocTemplate) string {
	var sb strings.Builder

	switch template.Language {
	case "go":
		sb.WriteString(fmt.Sprintf("// %s %s\n", doc.Name, doc.Description))
		for _, param := range doc.Params {
			sb.WriteString(fmt.Sprintf("// %s: %s\n", param.Name, param.Description))
		}
		if len(doc.Returns) > 0 {
			sb.WriteString(fmt.Sprintf("// Returns: %s\n", doc.Returns[0].Description))
		}

	case "python":
		sb.WriteString(`"""`)
		sb.WriteString(doc.Description)
		sb.WriteString("\n\n")
		if len(doc.Params) > 0 {
			sb.WriteString("Args:\n")
			for _, param := range doc.Params {
				sb.WriteString(fmt.Sprintf("    %s", param.Name))
				if param.Type != "" {
					sb.WriteString(fmt.Sprintf(" (%s)", param.Type))
				}
				sb.WriteString(fmt.Sprintf(": %s\n", param.Description))
			}
			sb.WriteString("\n")
		}
		if len(doc.Returns) > 0 {
			sb.WriteString("Returns:\n")
			sb.WriteString(fmt.Sprintf("    %s: %s\n", doc.Returns[0].Type, doc.Returns[0].Description))
		}
		sb.WriteString(`"""`)

	case "javascript", "typescript":
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", doc.Description))
		sb.WriteString(" *\n")
		for _, param := range doc.Params {
			sb.WriteString(fmt.Sprintf(" * @param {%s} %s - %s\n", param.Type, param.Name, param.Description))
		}
		if len(doc.Returns) > 0 {
			sb.WriteString(fmt.Sprintf(" * @returns {%s} %s\n", doc.Returns[0].Type, doc.Returns[0].Description))
		}
		sb.WriteString(" */")

	case "rust":
		sb.WriteString("/// ")
		sb.WriteString(doc.Description)
		sb.WriteString("\n///\n")
		if len(doc.Params) > 0 {
			sb.WriteString("/// # Arguments\n///\n")
			for _, param := range doc.Params {
				sb.WriteString(fmt.Sprintf("/// * `%s` - %s\n", param.Name, param.Description))
			}
		}
		if len(doc.Returns) > 0 {
			sb.WriteString("///\n/// # Returns\n///\n")
			sb.WriteString(fmt.Sprintf("/// %s\n", doc.Returns[0].Description))
		}
	}

	return sb.String()
}

func (dg *DocGenerator) renderClassDoc(doc ClassDoc, template *DocTemplate) string {
	var sb strings.Builder

	switch template.Language {
	case "go":
		sb.WriteString(fmt.Sprintf("// %s %s\n", doc.Name, doc.Description))

	case "python":
		sb.WriteString(`"""`)
		sb.WriteString(doc.Description)
		sb.WriteString("\n\n")
		if len(doc.Properties) > 0 {
			sb.WriteString("Attributes:\n")
			for _, prop := range doc.Properties {
				sb.WriteString(fmt.Sprintf("    %s (%s): %s\n", prop.Name, prop.Type, prop.Description))
			}
		}
		sb.WriteString(`"""`)

	case "javascript", "typescript":
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", doc.Description))
		sb.WriteString(" *\n")
		for _, prop := range doc.Properties {
			sb.WriteString(fmt.Sprintf(" * @property {%s} %s - %s\n", prop.Type, prop.Name, prop.Description))
		}
		sb.WriteString(" */")

	case "rust":
		sb.WriteString("/// ")
		sb.WriteString(doc.Description)
		sb.WriteString("\n///\n")
		if len(doc.Properties) > 0 {
			sb.WriteString("/// # Fields\n///\n")
			for _, prop := range doc.Properties {
				sb.WriteString(fmt.Sprintf("/// * `%s` - %s\n", prop.Name, prop.Description))
			}
		}
	}

	return sb.String()
}

func generateDescription(name, kind string) string {
	// Convert camelCase or PascalCase to sentence
	words := splitCamelCase(name)
	if len(words) == 0 {
		return fmt.Sprintf("A %s", kind)
	}

	// Lowercase all words
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}

	// Capitalize first word
	words[0] = strings.Title(words[0])

	return strings.Join(words, " ")
}

func generateParamDescription(name, typeName string) string {
	words := splitCamelCase(name)
	if len(words) == 0 {
		return "A parameter"
	}

	// Lowercase all words
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}

	desc := "The " + strings.Join(words, " ")

	// Add type hint
	if typeName != "" {
		switch strings.ToLower(typeName) {
		case "string":
			desc += " as a string"
		case "int", "int32", "int64", "number":
			desc += " as a number"
		case "bool", "boolean":
			desc += " flag"
		case "[]string":
			desc += " as a list of strings"
		case "[]int":
			desc += " as a list of numbers"
		}
	}

	return desc
}

func splitCamelCase(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// Default documentation templates

func goDocTemplate() *DocTemplate {
	return &DocTemplate{
		Language:     "go",
		CommentLine:  "//",
		FuncDoc:      "// {{.Name}} {{.Description}}",
		ClassDoc:     "// {{.Name}} {{.Description}}",
		ParamDoc:     "// {{.Name}}: {{.Description}}",
		ReturnDoc:    "// Returns: {{.Description}}",
	}
}

func pythonDocTemplate() *DocTemplate {
	return &DocTemplate{
		Language:     "python",
		CommentStart: `"""`,
		CommentEnd:   `"""`,
		FuncDoc: `"""{{.Description}}

Args:
{{range .Params}}    {{.Name}}{{if .Type}} ({{.Type}}){{end}}: {{.Description}}
{{end}}
Returns:
{{range .Returns}}    {{.Type}}: {{.Description}}
{{end}}"""`,
	}
}

func jsDocTemplate() *DocTemplate {
	return &DocTemplate{
		Language:     "javascript",
		CommentStart: "/**",
		CommentEnd:   " */",
		CommentLine:  " *",
		FuncDoc: `/**
 * {{.Description}}
 *{{range .Params}}
 * @param {{if .Type}}{{"{"}}{{.Type}}{{"}"}} {{end}}{{.Name}} - {{.Description}}{{end}}{{range .Returns}}
 * @returns {{if .Type}}{{"{"}}{{.Type}}{{"}"}} {{end}}{{.Description}}{{end}}
 */`,
	}
}

func tsDocTemplate() *DocTemplate {
	return &DocTemplate{
		Language:     "typescript",
		CommentStart: "/**",
		CommentEnd:   " */",
		CommentLine:  " *",
		FuncDoc: `/**
 * {{.Description}}
 *{{range .Params}}
 * @param {{if .Type}}{{"{"}}{{.Type}}{{"}"}} {{end}}{{.Name}} - {{.Description}}{{end}}{{range .Returns}}
 * @returns {{if .Type}}{{"{"}}{{.Type}}{{"}"}} {{end}}{{.Description}}{{end}}
 */`,
	}
}

func rustDocTemplate() *DocTemplate {
	return &DocTemplate{
		Language:    "rust",
		CommentLine: "///",
		FuncDoc: `/// {{.Description}}
///
/// # Arguments
///{{range .Params}}
/// * ` + "`{{.Name}}`" + ` - {{.Description}}{{end}}
///
/// # Returns
///{{range .Returns}}
/// {{.Description}}{{end}}`,
	}
}

// DocInserter inserts documentation into existing code.
type DocInserter struct {
	generator *DocGenerator
}

// NewDocInserter creates a new documentation inserter.
func NewDocInserter() *DocInserter {
	return &DocInserter{
		generator: NewDocGenerator(),
	}
}

// InsertDocs inserts documentation for undocumented functions.
func (di *DocInserter) InsertDocs(code, language string) (string, error) {
	// Get appropriate analyzer
	analyzer := di.getAnalyzer(language)
	if analyzer == nil {
		return code, fmt.Errorf("unsupported language: %s", language)
	}

	functions := analyzer.ExtractFunctions(code)
	result := code

	// Process in reverse order to maintain line positions
	for i := len(functions) - 1; i >= 0; i-- {
		fn := functions[i]
		if !fn.IsExported {
			continue
		}

		// Check if already documented
		if di.hasDocumentation(code, fn, language) {
			continue
		}

		// Generate documentation
		doc, err := di.generator.GenerateFunctionDoc(fn, language)
		if err != nil {
			continue
		}

		// Insert documentation before function
		result = di.insertBefore(result, fn.StartLine, doc)
	}

	return result, nil
}

func (di *DocInserter) getAnalyzer(language string) CodeAnalyzer {
	switch language {
	case "go":
		return &GoAnalyzer{}
	case "python":
		return &PythonAnalyzer{}
	case "javascript", "typescript":
		return &JSAnalyzer{}
	default:
		return nil
	}
}

func (di *DocInserter) hasDocumentation(code string, fn FunctionInfo, language string) bool {
	lines := strings.Split(code, "\n")
	if fn.StartLine <= 0 || fn.StartLine > len(lines) {
		return false
	}

	// Check lines before function
	startIdx := fn.StartLine - 2
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < fn.StartLine-1; i++ {
		line := strings.TrimSpace(lines[i])
		switch language {
		case "go":
			if strings.HasPrefix(line, "//") {
				return true
			}
		case "python":
			if strings.Contains(line, `"""`) || strings.Contains(line, `'''`) {
				return true
			}
		case "javascript", "typescript":
			if strings.Contains(line, "/**") || strings.Contains(line, "*/") {
				return true
			}
		case "rust":
			if strings.HasPrefix(line, "///") {
				return true
			}
		}
	}

	return false
}

func (di *DocInserter) insertBefore(code string, lineNum int, doc string) string {
	lines := strings.Split(code, "\n")
	if lineNum <= 0 || lineNum > len(lines)+1 {
		return code
	}

	idx := lineNum - 1
	newLines := make([]string, 0, len(lines)+strings.Count(doc, "\n")+1)
	newLines = append(newLines, lines[:idx]...)
	newLines = append(newLines, doc)
	newLines = append(newLines, lines[idx:]...)

	return strings.Join(newLines, "\n")
}
