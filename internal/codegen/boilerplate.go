package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

// BoilerplateGenerator generates common code patterns.
type BoilerplateGenerator struct {
	templates map[string]*template.Template
}

// BoilerplateType represents a type of boilerplate code.
type BoilerplateType string

const (
	BoilerplateCRUD       BoilerplateType = "crud"
	BoilerplateAPI        BoilerplateType = "api"
	BoilerplateModel      BoilerplateType = "model"
	BoilerplateRepository BoilerplateType = "repository"
	BoilerplateService    BoilerplateType = "service"
	BoilerplateHandler    BoilerplateType = "handler"
	BoilerplateTest       BoilerplateType = "test"
)

// BoilerplateConfig contains configuration for boilerplate generation.
type BoilerplateConfig struct {
	Type       BoilerplateType
	Language   string
	EntityName string
	Fields     []FieldSpec
	Options    BoilerplateOptions
}

// FieldSpec describes a field in an entity.
type FieldSpec struct {
	Name     string
	Type     string
	JSONTag  string
	DBTag    string
	Required bool
	Primary  bool
}

// BoilerplateOptions contains additional options for generation.
type BoilerplateOptions struct {
	PackageName    string
	IncludeTests   bool
	IncludeCreate  bool
	IncludeRead    bool
	IncludeUpdate  bool
	IncludeDelete  bool
	IncludeList    bool
	UseInterfaces  bool
	DatabaseType   string // "sql", "mongodb", "memory"
	HTTPFramework  string // "stdlib", "gin", "echo", "chi"
}

// GeneratedBoilerplate represents generated boilerplate code.
type GeneratedBoilerplate struct {
	Type     BoilerplateType
	Language string
	Files    []GeneratedFile
}

// GeneratedFile represents a single generated file.
type GeneratedFile struct {
	Name    string
	Path    string
	Content string
}

// NewBoilerplateGenerator creates a new boilerplate generator.
func NewBoilerplateGenerator() *BoilerplateGenerator {
	bg := &BoilerplateGenerator{
		templates: make(map[string]*template.Template),
	}
	return bg
}

// Generate generates boilerplate code based on configuration.
func (bg *BoilerplateGenerator) Generate(config BoilerplateConfig) (*GeneratedBoilerplate, error) {
	// Set defaults
	if config.Options.PackageName == "" {
		config.Options.PackageName = strings.ToLower(config.EntityName)
	}

	switch config.Language {
	case "go":
		return bg.generateGo(config)
	case "python":
		return bg.generatePython(config)
	case "typescript", "javascript":
		return bg.generateTypeScript(config)
	default:
		return nil, fmt.Errorf("unsupported language: %s", config.Language)
	}
}

func (bg *BoilerplateGenerator) generateGo(config BoilerplateConfig) (*GeneratedBoilerplate, error) {
	result := &GeneratedBoilerplate{
		Type:     config.Type,
		Language: "go",
		Files:    make([]GeneratedFile, 0),
	}

	switch config.Type {
	case BoilerplateCRUD:
		return bg.generateGoCRUD(config)
	case BoilerplateModel:
		result.Files = append(result.Files, bg.generateGoModel(config))
	case BoilerplateRepository:
		result.Files = append(result.Files, bg.generateGoRepository(config))
	case BoilerplateService:
		result.Files = append(result.Files, bg.generateGoService(config))
	case BoilerplateHandler:
		result.Files = append(result.Files, bg.generateGoHandler(config))
	case BoilerplateAPI:
		return bg.generateGoAPI(config)
	}

	return result, nil
}

func (bg *BoilerplateGenerator) generateGoCRUD(config BoilerplateConfig) (*GeneratedBoilerplate, error) {
	result := &GeneratedBoilerplate{
		Type:     BoilerplateCRUD,
		Language: "go",
		Files:    make([]GeneratedFile, 0),
	}

	// Generate all CRUD components
	result.Files = append(result.Files, bg.generateGoModel(config))
	result.Files = append(result.Files, bg.generateGoRepository(config))
	result.Files = append(result.Files, bg.generateGoService(config))
	result.Files = append(result.Files, bg.generateGoHandler(config))

	if config.Options.IncludeTests {
		result.Files = append(result.Files, bg.generateGoRepositoryTest(config))
	}

	return result, nil
}

func (bg *BoilerplateGenerator) generateGoModel(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString(fmt.Sprintf("package %s\n\n", config.Options.PackageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"time\"\n")
	sb.WriteString(")\n\n")

	// Entity struct
	sb.WriteString(fmt.Sprintf("// %s represents a %s entity.\n", config.EntityName, entityLower))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", config.EntityName))

	for _, field := range config.Fields {
		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = toSnakeCase(field.Name)
		}

		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"", field.Name, field.Type, jsonTag))
		if field.DBTag != "" {
			sb.WriteString(fmt.Sprintf(" db:\"%s\"", field.DBTag))
		}
		sb.WriteString("`\n")
	}

	// Add common fields if not present
	hasID := false
	hasCreatedAt := false
	hasUpdatedAt := false
	for _, field := range config.Fields {
		if field.Name == "ID" {
			hasID = true
		}
		if field.Name == "CreatedAt" {
			hasCreatedAt = true
		}
		if field.Name == "UpdatedAt" {
			hasUpdatedAt = true
		}
	}

	if !hasID {
		sb.WriteString("\tID        string    `json:\"id\" db:\"id\"`\n")
	}
	if !hasCreatedAt {
		sb.WriteString("\tCreatedAt time.Time `json:\"created_at\" db:\"created_at\"`\n")
	}
	if !hasUpdatedAt {
		sb.WriteString("\tUpdatedAt time.Time `json:\"updated_at\" db:\"updated_at\"`\n")
	}

	sb.WriteString("}\n\n")

	// Create request struct
	sb.WriteString(fmt.Sprintf("// Create%sRequest represents a request to create a %s.\n", config.EntityName, entityLower))
	sb.WriteString(fmt.Sprintf("type Create%sRequest struct {\n", config.EntityName))
	for _, field := range config.Fields {
		if field.Primary {
			continue
		}
		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = toSnakeCase(field.Name)
		}
		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", field.Name, field.Type, jsonTag))
	}
	sb.WriteString("}\n\n")

	// Update request struct
	sb.WriteString(fmt.Sprintf("// Update%sRequest represents a request to update a %s.\n", config.EntityName, entityLower))
	sb.WriteString(fmt.Sprintf("type Update%sRequest struct {\n", config.EntityName))
	for _, field := range config.Fields {
		if field.Primary {
			continue
		}
		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = toSnakeCase(field.Name)
		}
		// Make update fields pointers for partial updates
		sb.WriteString(fmt.Sprintf("\t%s *%s `json:\"%s,omitempty\"`\n", field.Name, field.Type, jsonTag))
	}
	sb.WriteString("}\n")

	return GeneratedFile{
		Name:    fmt.Sprintf("%s.go", entityLower),
		Path:    fmt.Sprintf("%s/%s.go", config.Options.PackageName, entityLower),
		Content: sb.String(),
	}
}

func (bg *BoilerplateGenerator) generateGoRepository(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString(fmt.Sprintf("package %s\n\n", config.Options.PackageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"errors\"\n")
	sb.WriteString("\t\"sync\"\n")
	sb.WriteString("\t\"time\"\n\n")
	sb.WriteString("\t\"github.com/google/uuid\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("var (\n")
	sb.WriteString(fmt.Sprintf("\tErr%sNotFound = errors.New(\"%s not found\")\n", config.EntityName, entityLower))
	sb.WriteString(")\n\n")

	// Repository interface
	if config.Options.UseInterfaces {
		sb.WriteString(fmt.Sprintf("// %sRepository defines the interface for %s persistence.\n", config.EntityName, entityLower))
		sb.WriteString(fmt.Sprintf("type %sRepository interface {\n", config.EntityName))
		if config.Options.IncludeCreate {
			sb.WriteString(fmt.Sprintf("\tCreate(ctx context.Context, entity *%s) error\n", config.EntityName))
		}
		if config.Options.IncludeRead {
			sb.WriteString(fmt.Sprintf("\tGetByID(ctx context.Context, id string) (*%s, error)\n", config.EntityName))
		}
		if config.Options.IncludeUpdate {
			sb.WriteString(fmt.Sprintf("\tUpdate(ctx context.Context, entity *%s) error\n", config.EntityName))
		}
		if config.Options.IncludeDelete {
			sb.WriteString("\tDelete(ctx context.Context, id string) error\n")
		}
		if config.Options.IncludeList {
			sb.WriteString(fmt.Sprintf("\tList(ctx context.Context, offset, limit int) ([]*%s, error)\n", config.EntityName))
		}
		sb.WriteString("}\n\n")
	}

	// In-memory implementation
	sb.WriteString(fmt.Sprintf("// InMemory%sRepository is an in-memory implementation of %sRepository.\n", config.EntityName, config.EntityName))
	sb.WriteString(fmt.Sprintf("type InMemory%sRepository struct {\n", config.EntityName))
	sb.WriteString("\tmu    sync.RWMutex\n")
	sb.WriteString(fmt.Sprintf("\titems map[string]*%s\n", config.EntityName))
	sb.WriteString("}\n\n")

	// Constructor
	sb.WriteString(fmt.Sprintf("// NewInMemory%sRepository creates a new in-memory %s repository.\n", config.EntityName, entityLower))
	sb.WriteString(fmt.Sprintf("func NewInMemory%sRepository() *InMemory%sRepository {\n", config.EntityName, config.EntityName))
	sb.WriteString(fmt.Sprintf("\treturn &InMemory%sRepository{\n", config.EntityName))
	sb.WriteString(fmt.Sprintf("\t\titems: make(map[string]*%s),\n", config.EntityName))
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	// Create method
	if config.Options.IncludeCreate {
		sb.WriteString(fmt.Sprintf("// Create adds a new %s to the repository.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (r *InMemory%sRepository) Create(ctx context.Context, entity *%s) error {\n", config.EntityName, config.EntityName))
		sb.WriteString("\tr.mu.Lock()\n")
		sb.WriteString("\tdefer r.mu.Unlock()\n\n")
		sb.WriteString("\tif entity.ID == \"\" {\n")
		sb.WriteString("\t\tentity.ID = uuid.New().String()\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\tentity.CreatedAt = time.Now()\n")
		sb.WriteString("\tentity.UpdatedAt = time.Now()\n\n")
		sb.WriteString("\tr.items[entity.ID] = entity\n")
		sb.WriteString("\treturn nil\n")
		sb.WriteString("}\n\n")
	}

	// GetByID method
	if config.Options.IncludeRead {
		sb.WriteString(fmt.Sprintf("// GetByID retrieves a %s by its ID.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (r *InMemory%sRepository) GetByID(ctx context.Context, id string) (*%s, error) {\n", config.EntityName, config.EntityName))
		sb.WriteString("\tr.mu.RLock()\n")
		sb.WriteString("\tdefer r.mu.RUnlock()\n\n")
		sb.WriteString("\tentity, ok := r.items[id]\n")
		sb.WriteString("\tif !ok {\n")
		sb.WriteString(fmt.Sprintf("\t\treturn nil, Err%sNotFound\n", config.EntityName))
		sb.WriteString("\t}\n")
		sb.WriteString("\treturn entity, nil\n")
		sb.WriteString("}\n\n")
	}

	// Update method
	if config.Options.IncludeUpdate {
		sb.WriteString(fmt.Sprintf("// Update updates an existing %s.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (r *InMemory%sRepository) Update(ctx context.Context, entity *%s) error {\n", config.EntityName, config.EntityName))
		sb.WriteString("\tr.mu.Lock()\n")
		sb.WriteString("\tdefer r.mu.Unlock()\n\n")
		sb.WriteString("\tif _, ok := r.items[entity.ID]; !ok {\n")
		sb.WriteString(fmt.Sprintf("\t\treturn Err%sNotFound\n", config.EntityName))
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tentity.UpdatedAt = time.Now()\n")
		sb.WriteString("\tr.items[entity.ID] = entity\n")
		sb.WriteString("\treturn nil\n")
		sb.WriteString("}\n\n")
	}

	// Delete method
	if config.Options.IncludeDelete {
		sb.WriteString(fmt.Sprintf("// Delete removes a %s from the repository.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (r *InMemory%sRepository) Delete(ctx context.Context, id string) error {\n", config.EntityName))
		sb.WriteString("\tr.mu.Lock()\n")
		sb.WriteString("\tdefer r.mu.Unlock()\n\n")
		sb.WriteString("\tif _, ok := r.items[id]; !ok {\n")
		sb.WriteString(fmt.Sprintf("\t\treturn Err%sNotFound\n", config.EntityName))
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tdelete(r.items, id)\n")
		sb.WriteString("\treturn nil\n")
		sb.WriteString("}\n\n")
	}

	// List method
	if config.Options.IncludeList {
		sb.WriteString(fmt.Sprintf("// List retrieves a paginated list of %ss.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (r *InMemory%sRepository) List(ctx context.Context, offset, limit int) ([]*%s, error) {\n", config.EntityName, config.EntityName))
		sb.WriteString("\tr.mu.RLock()\n")
		sb.WriteString("\tdefer r.mu.RUnlock()\n\n")
		sb.WriteString(fmt.Sprintf("\tresult := make([]*%s, 0, limit)\n", config.EntityName))
		sb.WriteString("\ti := 0\n")
		sb.WriteString("\tfor _, entity := range r.items {\n")
		sb.WriteString("\t\tif i >= offset && len(result) < limit {\n")
		sb.WriteString("\t\t\tresult = append(result, entity)\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t\ti++\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\treturn result, nil\n")
		sb.WriteString("}\n")
	}

	return GeneratedFile{
		Name:    fmt.Sprintf("%s_repository.go", entityLower),
		Path:    fmt.Sprintf("%s/%s_repository.go", config.Options.PackageName, entityLower),
		Content: sb.String(),
	}
}

func (bg *BoilerplateGenerator) generateGoService(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString(fmt.Sprintf("package %s\n\n", config.Options.PackageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString(")\n\n")

	// Service struct
	sb.WriteString(fmt.Sprintf("// %sService provides business logic for %s operations.\n", config.EntityName, entityLower))
	sb.WriteString(fmt.Sprintf("type %sService struct {\n", config.EntityName))
	if config.Options.UseInterfaces {
		sb.WriteString(fmt.Sprintf("\trepo %sRepository\n", config.EntityName))
	} else {
		sb.WriteString(fmt.Sprintf("\trepo *InMemory%sRepository\n", config.EntityName))
	}
	sb.WriteString("}\n\n")

	// Constructor
	sb.WriteString(fmt.Sprintf("// New%sService creates a new %s service.\n", config.EntityName, entityLower))
	if config.Options.UseInterfaces {
		sb.WriteString(fmt.Sprintf("func New%sService(repo %sRepository) *%sService {\n", config.EntityName, config.EntityName, config.EntityName))
	} else {
		sb.WriteString(fmt.Sprintf("func New%sService(repo *InMemory%sRepository) *%sService {\n", config.EntityName, config.EntityName, config.EntityName))
	}
	sb.WriteString(fmt.Sprintf("\treturn &%sService{repo: repo}\n", config.EntityName))
	sb.WriteString("}\n\n")

	// Create method
	if config.Options.IncludeCreate {
		sb.WriteString(fmt.Sprintf("// Create creates a new %s.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (s *%sService) Create(ctx context.Context, req Create%sRequest) (*%s, error) {\n", config.EntityName, config.EntityName, config.EntityName))
		sb.WriteString(fmt.Sprintf("\tentity := &%s{\n", config.EntityName))
		for _, field := range config.Fields {
			if field.Primary {
				continue
			}
			sb.WriteString(fmt.Sprintf("\t\t%s: req.%s,\n", field.Name, field.Name))
		}
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tif err := s.repo.Create(ctx, entity); err != nil {\n")
		sb.WriteString("\t\treturn nil, err\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\treturn entity, nil\n")
		sb.WriteString("}\n\n")
	}

	// GetByID method
	if config.Options.IncludeRead {
		sb.WriteString(fmt.Sprintf("// GetByID retrieves a %s by ID.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (s *%sService) GetByID(ctx context.Context, id string) (*%s, error) {\n", config.EntityName, config.EntityName))
		sb.WriteString("\treturn s.repo.GetByID(ctx, id)\n")
		sb.WriteString("}\n\n")
	}

	// Update method
	if config.Options.IncludeUpdate {
		sb.WriteString(fmt.Sprintf("// Update updates an existing %s.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (s *%sService) Update(ctx context.Context, id string, req Update%sRequest) (*%s, error) {\n", config.EntityName, config.EntityName, config.EntityName))
		sb.WriteString("\tentity, err := s.repo.GetByID(ctx, id)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\treturn nil, err\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\t// Apply updates\n")
		for _, field := range config.Fields {
			if field.Primary {
				continue
			}
			sb.WriteString(fmt.Sprintf("\tif req.%s != nil {\n", field.Name))
			sb.WriteString(fmt.Sprintf("\t\tentity.%s = *req.%s\n", field.Name, field.Name))
			sb.WriteString("\t}\n")
		}
		sb.WriteString("\n\tif err := s.repo.Update(ctx, entity); err != nil {\n")
		sb.WriteString("\t\treturn nil, err\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\treturn entity, nil\n")
		sb.WriteString("}\n\n")
	}

	// Delete method
	if config.Options.IncludeDelete {
		sb.WriteString(fmt.Sprintf("// Delete removes a %s.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (s *%sService) Delete(ctx context.Context, id string) error {\n", config.EntityName))
		sb.WriteString("\treturn s.repo.Delete(ctx, id)\n")
		sb.WriteString("}\n\n")
	}

	// List method
	if config.Options.IncludeList {
		sb.WriteString(fmt.Sprintf("// List retrieves a paginated list of %ss.\n", entityLower))
		sb.WriteString(fmt.Sprintf("func (s *%sService) List(ctx context.Context, offset, limit int) ([]*%s, error) {\n", config.EntityName, config.EntityName))
		sb.WriteString("\treturn s.repo.List(ctx, offset, limit)\n")
		sb.WriteString("}\n")
	}

	return GeneratedFile{
		Name:    fmt.Sprintf("%s_service.go", entityLower),
		Path:    fmt.Sprintf("%s/%s_service.go", config.Options.PackageName, entityLower),
		Content: sb.String(),
	}
}

func (bg *BoilerplateGenerator) generateGoHandler(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString(fmt.Sprintf("package %s\n\n", config.Options.PackageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"strconv\"\n")
	sb.WriteString(")\n\n")

	// Handler struct
	sb.WriteString(fmt.Sprintf("// %sHandler handles HTTP requests for %s operations.\n", config.EntityName, entityLower))
	sb.WriteString(fmt.Sprintf("type %sHandler struct {\n", config.EntityName))
	sb.WriteString(fmt.Sprintf("\tservice *%sService\n", config.EntityName))
	sb.WriteString("}\n\n")

	// Constructor
	sb.WriteString(fmt.Sprintf("// New%sHandler creates a new %s handler.\n", config.EntityName, entityLower))
	sb.WriteString(fmt.Sprintf("func New%sHandler(service *%sService) *%sHandler {\n", config.EntityName, config.EntityName, config.EntityName))
	sb.WriteString(fmt.Sprintf("\treturn &%sHandler{service: service}\n", config.EntityName))
	sb.WriteString("}\n\n")

	// RegisterRoutes
	sb.WriteString("// RegisterRoutes registers the handler routes.\n")
	sb.WriteString(fmt.Sprintf("func (h *%sHandler) RegisterRoutes(mux *http.ServeMux) {\n", config.EntityName))
	basePath := fmt.Sprintf("/%ss", entityLower)
	if config.Options.IncludeCreate || config.Options.IncludeList {
		sb.WriteString(fmt.Sprintf("\tmux.HandleFunc(\"%s\", h.handleCollection)\n", basePath))
	}
	if config.Options.IncludeRead || config.Options.IncludeUpdate || config.Options.IncludeDelete {
		sb.WriteString(fmt.Sprintf("\tmux.HandleFunc(\"%s/\", h.handleSingle)\n", basePath))
	}
	sb.WriteString("}\n\n")

	// handleCollection
	if config.Options.IncludeCreate || config.Options.IncludeList {
		sb.WriteString(fmt.Sprintf("func (h *%sHandler) handleCollection(w http.ResponseWriter, r *http.Request) {\n", config.EntityName))
		sb.WriteString("\tswitch r.Method {\n")
		if config.Options.IncludeList {
			sb.WriteString("\tcase http.MethodGet:\n")
			sb.WriteString("\t\th.list(w, r)\n")
		}
		if config.Options.IncludeCreate {
			sb.WriteString("\tcase http.MethodPost:\n")
			sb.WriteString("\t\th.create(w, r)\n")
		}
		sb.WriteString("\tdefault:\n")
		sb.WriteString("\t\thttp.Error(w, \"Method not allowed\", http.StatusMethodNotAllowed)\n")
		sb.WriteString("\t}\n")
		sb.WriteString("}\n\n")
	}

	// handleSingle
	if config.Options.IncludeRead || config.Options.IncludeUpdate || config.Options.IncludeDelete {
		sb.WriteString(fmt.Sprintf("func (h *%sHandler) handleSingle(w http.ResponseWriter, r *http.Request) {\n", config.EntityName))
		sb.WriteString("\tswitch r.Method {\n")
		if config.Options.IncludeRead {
			sb.WriteString("\tcase http.MethodGet:\n")
			sb.WriteString("\t\th.getByID(w, r)\n")
		}
		if config.Options.IncludeUpdate {
			sb.WriteString("\tcase http.MethodPut:\n")
			sb.WriteString("\t\th.update(w, r)\n")
		}
		if config.Options.IncludeDelete {
			sb.WriteString("\tcase http.MethodDelete:\n")
			sb.WriteString("\t\th.delete(w, r)\n")
		}
		sb.WriteString("\tdefault:\n")
		sb.WriteString("\t\thttp.Error(w, \"Method not allowed\", http.StatusMethodNotAllowed)\n")
		sb.WriteString("\t}\n")
		sb.WriteString("}\n\n")
	}

	// create handler
	if config.Options.IncludeCreate {
		sb.WriteString(fmt.Sprintf("func (h *%sHandler) create(w http.ResponseWriter, r *http.Request) {\n", config.EntityName))
		sb.WriteString(fmt.Sprintf("\tvar req Create%sRequest\n", config.EntityName))
		sb.WriteString("\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {\n")
		sb.WriteString("\t\thttp.Error(w, err.Error(), http.StatusBadRequest)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tentity, err := h.service.Create(r.Context(), req)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
		sb.WriteString("\tw.WriteHeader(http.StatusCreated)\n")
		sb.WriteString("\tjson.NewEncoder(w).Encode(entity)\n")
		sb.WriteString("}\n\n")
	}

	// getByID handler
	if config.Options.IncludeRead {
		sb.WriteString(fmt.Sprintf("func (h *%sHandler) getByID(w http.ResponseWriter, r *http.Request) {\n", config.EntityName))
		sb.WriteString("\tid := extractID(r.URL.Path)\n")
		sb.WriteString("\tif id == \"\" {\n")
		sb.WriteString("\t\thttp.Error(w, \"ID required\", http.StatusBadRequest)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tentity, err := h.service.GetByID(r.Context(), id)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString(fmt.Sprintf("\t\tif err == Err%sNotFound {\n", config.EntityName))
		sb.WriteString("\t\t\thttp.Error(w, err.Error(), http.StatusNotFound)\n")
		sb.WriteString("\t\t\treturn\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
		sb.WriteString("\tjson.NewEncoder(w).Encode(entity)\n")
		sb.WriteString("}\n\n")
	}

	// update handler
	if config.Options.IncludeUpdate {
		sb.WriteString(fmt.Sprintf("func (h *%sHandler) update(w http.ResponseWriter, r *http.Request) {\n", config.EntityName))
		sb.WriteString("\tid := extractID(r.URL.Path)\n")
		sb.WriteString("\tif id == \"\" {\n")
		sb.WriteString("\t\thttp.Error(w, \"ID required\", http.StatusBadRequest)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString(fmt.Sprintf("\tvar req Update%sRequest\n", config.EntityName))
		sb.WriteString("\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {\n")
		sb.WriteString("\t\thttp.Error(w, err.Error(), http.StatusBadRequest)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tentity, err := h.service.Update(r.Context(), id, req)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString(fmt.Sprintf("\t\tif err == Err%sNotFound {\n", config.EntityName))
		sb.WriteString("\t\t\thttp.Error(w, err.Error(), http.StatusNotFound)\n")
		sb.WriteString("\t\t\treturn\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
		sb.WriteString("\tjson.NewEncoder(w).Encode(entity)\n")
		sb.WriteString("}\n\n")
	}

	// delete handler
	if config.Options.IncludeDelete {
		sb.WriteString(fmt.Sprintf("func (h *%sHandler) delete(w http.ResponseWriter, r *http.Request) {\n", config.EntityName))
		sb.WriteString("\tid := extractID(r.URL.Path)\n")
		sb.WriteString("\tif id == \"\" {\n")
		sb.WriteString("\t\thttp.Error(w, \"ID required\", http.StatusBadRequest)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tif err := h.service.Delete(r.Context(), id); err != nil {\n")
		sb.WriteString(fmt.Sprintf("\t\tif err == Err%sNotFound {\n", config.EntityName))
		sb.WriteString("\t\t\thttp.Error(w, err.Error(), http.StatusNotFound)\n")
		sb.WriteString("\t\t\treturn\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tw.WriteHeader(http.StatusNoContent)\n")
		sb.WriteString("}\n\n")
	}

	// list handler
	if config.Options.IncludeList {
		sb.WriteString(fmt.Sprintf("func (h *%sHandler) list(w http.ResponseWriter, r *http.Request) {\n", config.EntityName))
		sb.WriteString("\toffset, _ := strconv.Atoi(r.URL.Query().Get(\"offset\"))\n")
		sb.WriteString("\tlimit, _ := strconv.Atoi(r.URL.Query().Get(\"limit\"))\n")
		sb.WriteString("\tif limit <= 0 {\n")
		sb.WriteString("\t\tlimit = 10\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tentities, err := h.service.List(r.Context(), offset, limit)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n\n")
		sb.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
		sb.WriteString("\tjson.NewEncoder(w).Encode(entities)\n")
		sb.WriteString("}\n\n")
	}

	// Helper function
	sb.WriteString("// extractID extracts the ID from a URL path like /resources/{id}\n")
	sb.WriteString("func extractID(path string) string {\n")
	sb.WriteString("\tparts := strings.Split(path, \"/\")\n")
	sb.WriteString("\tif len(parts) < 3 {\n")
	sb.WriteString("\t\treturn \"\"\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn parts[len(parts)-1]\n")
	sb.WriteString("}\n")

	// Add strings import
	content := sb.String()
	content = strings.Replace(content, "import (\n\t\"encoding/json\"\n\t\"net/http\"\n\t\"strconv\"\n)\n\n", "import (\n\t\"encoding/json\"\n\t\"net/http\"\n\t\"strconv\"\n\t\"strings\"\n)\n\n", 1)

	return GeneratedFile{
		Name:    fmt.Sprintf("%s_handler.go", entityLower),
		Path:    fmt.Sprintf("%s/%s_handler.go", config.Options.PackageName, entityLower),
		Content: content,
	}
}

func (bg *BoilerplateGenerator) generateGoRepositoryTest(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString(fmt.Sprintf("package %s\n\n", config.Options.PackageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"testing\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString(fmt.Sprintf("func TestInMemory%sRepository_CRUD(t *testing.T) {\n", config.EntityName))
	sb.WriteString(fmt.Sprintf("\trepo := NewInMemory%sRepository()\n", config.EntityName))
	sb.WriteString("\tctx := context.Background()\n\n")

	if config.Options.IncludeCreate {
		sb.WriteString("\t// Test Create\n")
		sb.WriteString(fmt.Sprintf("\tentity := &%s{\n", config.EntityName))
		for _, field := range config.Fields {
			if field.Primary {
				continue
			}
			switch field.Type {
			case "string":
				sb.WriteString(fmt.Sprintf("\t\t%s: \"test_%s\",\n", field.Name, strings.ToLower(field.Name)))
			case "int", "int32", "int64":
				sb.WriteString(fmt.Sprintf("\t\t%s: 1,\n", field.Name))
			case "bool":
				sb.WriteString(fmt.Sprintf("\t\t%s: true,\n", field.Name))
			}
		}
		sb.WriteString("\t}\n\n")
		sb.WriteString("\terr := repo.Create(ctx, entity)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\tt.Fatalf(\"Create failed: %v\", err)\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\tif entity.ID == \"\" {\n")
		sb.WriteString("\t\tt.Error(\"Expected ID to be set after create\")\n")
		sb.WriteString("\t}\n\n")
	}

	if config.Options.IncludeRead {
		sb.WriteString("\t// Test GetByID\n")
		sb.WriteString("\tretrieved, err := repo.GetByID(ctx, entity.ID)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\tt.Fatalf(\"GetByID failed: %v\", err)\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\tif retrieved.ID != entity.ID {\n")
		sb.WriteString("\t\tt.Error(\"Retrieved entity ID mismatch\")\n")
		sb.WriteString("\t}\n\n")

		sb.WriteString("\t// Test GetByID not found\n")
		sb.WriteString("\t_, err = repo.GetByID(ctx, \"nonexistent\")\n")
		sb.WriteString(fmt.Sprintf("\tif err != Err%sNotFound {\n", config.EntityName))
		sb.WriteString(fmt.Sprintf("\t\tt.Errorf(\"Expected Err%sNotFound, got %%v\", err)\n", config.EntityName))
		sb.WriteString("\t}\n\n")
	}

	if config.Options.IncludeUpdate {
		sb.WriteString("\t// Test Update\n")
		if len(config.Fields) > 0 {
			field := config.Fields[0]
			switch field.Type {
			case "string":
				sb.WriteString(fmt.Sprintf("\tentity.%s = \"updated_%s\"\n", field.Name, strings.ToLower(field.Name)))
			case "int", "int32", "int64":
				sb.WriteString(fmt.Sprintf("\tentity.%s = 2\n", field.Name))
			case "bool":
				sb.WriteString(fmt.Sprintf("\tentity.%s = false\n", field.Name))
			}
		}
		sb.WriteString("\terr = repo.Update(ctx, entity)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\tt.Fatalf(\"Update failed: %v\", err)\n")
		sb.WriteString("\t}\n\n")
	}

	if config.Options.IncludeList {
		sb.WriteString("\t// Test List\n")
		sb.WriteString("\tentities, err := repo.List(ctx, 0, 10)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\tt.Fatalf(\"List failed: %v\", err)\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\tif len(entities) != 1 {\n")
		sb.WriteString("\t\tt.Errorf(\"Expected 1 entity, got %d\", len(entities))\n")
		sb.WriteString("\t}\n\n")
	}

	if config.Options.IncludeDelete {
		sb.WriteString("\t// Test Delete\n")
		sb.WriteString("\terr = repo.Delete(ctx, entity.ID)\n")
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\tt.Fatalf(\"Delete failed: %v\", err)\n")
		sb.WriteString("\t}\n\n")

		sb.WriteString("\t// Verify deleted\n")
		sb.WriteString("\t_, err = repo.GetByID(ctx, entity.ID)\n")
		sb.WriteString(fmt.Sprintf("\tif err != Err%sNotFound {\n", config.EntityName))
		sb.WriteString(fmt.Sprintf("\t\tt.Errorf(\"Expected Err%sNotFound after delete, got %%v\", err)\n", config.EntityName))
		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n")

	return GeneratedFile{
		Name:    fmt.Sprintf("%s_repository_test.go", entityLower),
		Path:    fmt.Sprintf("%s/%s_repository_test.go", config.Options.PackageName, entityLower),
		Content: sb.String(),
	}
}

func (bg *BoilerplateGenerator) generateGoAPI(config BoilerplateConfig) (*GeneratedBoilerplate, error) {
	result := &GeneratedBoilerplate{
		Type:     BoilerplateAPI,
		Language: "go",
		Files:    make([]GeneratedFile, 0),
	}

	// Generate model and handler
	result.Files = append(result.Files, bg.generateGoModel(config))
	result.Files = append(result.Files, bg.generateGoHandler(config))

	return result, nil
}

// Python generation
func (bg *BoilerplateGenerator) generatePython(config BoilerplateConfig) (*GeneratedBoilerplate, error) {
	result := &GeneratedBoilerplate{
		Type:     config.Type,
		Language: "python",
		Files:    make([]GeneratedFile, 0),
	}

	switch config.Type {
	case BoilerplateCRUD, BoilerplateModel:
		result.Files = append(result.Files, bg.generatePythonModel(config))
	}

	return result, nil
}

func (bg *BoilerplateGenerator) generatePythonModel(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString("from dataclasses import dataclass, field\n")
	sb.WriteString("from datetime import datetime\n")
	sb.WriteString("from typing import Optional\n")
	sb.WriteString("from uuid import uuid4\n\n\n")

	sb.WriteString("@dataclass\n")
	sb.WriteString(fmt.Sprintf("class %s:\n", config.EntityName))
	sb.WriteString(fmt.Sprintf("    \"\"\"%s entity.\"\"\"\n\n", config.EntityName))

	for _, f := range config.Fields {
		pyType := goPythonTypeMap(f.Type)
		if f.Required {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", toSnakeCase(f.Name), pyType))
		} else {
			sb.WriteString(fmt.Sprintf("    %s: Optional[%s] = None\n", toSnakeCase(f.Name), pyType))
		}
	}

	sb.WriteString("    id: str = field(default_factory=lambda: str(uuid4()))\n")
	sb.WriteString("    created_at: datetime = field(default_factory=datetime.utcnow)\n")
	sb.WriteString("    updated_at: datetime = field(default_factory=datetime.utcnow)\n")

	return GeneratedFile{
		Name:    fmt.Sprintf("%s.py", entityLower),
		Path:    fmt.Sprintf("%s/%s.py", config.Options.PackageName, entityLower),
		Content: sb.String(),
	}
}

// TypeScript generation
func (bg *BoilerplateGenerator) generateTypeScript(config BoilerplateConfig) (*GeneratedBoilerplate, error) {
	result := &GeneratedBoilerplate{
		Type:     config.Type,
		Language: "typescript",
		Files:    make([]GeneratedFile, 0),
	}

	switch config.Type {
	case BoilerplateCRUD, BoilerplateModel:
		result.Files = append(result.Files, bg.generateTSModel(config))
	case BoilerplateAPI:
		result.Files = append(result.Files, bg.generateTSModel(config))
		result.Files = append(result.Files, bg.generateTSService(config))
	}

	return result, nil
}

func (bg *BoilerplateGenerator) generateTSModel(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString(fmt.Sprintf("/**\n * %s entity interface.\n */\n", config.EntityName))
	sb.WriteString(fmt.Sprintf("export interface %s {\n", config.EntityName))

	for _, f := range config.Fields {
		tsType := goTSTypeMap(f.Type)
		if f.Required {
			sb.WriteString(fmt.Sprintf("  %s: %s;\n", toCamelCase(f.Name), tsType))
		} else {
			sb.WriteString(fmt.Sprintf("  %s?: %s;\n", toCamelCase(f.Name), tsType))
		}
	}

	sb.WriteString("  id: string;\n")
	sb.WriteString("  createdAt: Date;\n")
	sb.WriteString("  updatedAt: Date;\n")
	sb.WriteString("}\n\n")

	// Create request type
	sb.WriteString(fmt.Sprintf("/**\n * Request to create a %s.\n */\n", entityLower))
	sb.WriteString(fmt.Sprintf("export interface Create%sRequest {\n", config.EntityName))
	for _, f := range config.Fields {
		if f.Primary {
			continue
		}
		tsType := goTSTypeMap(f.Type)
		if f.Required {
			sb.WriteString(fmt.Sprintf("  %s: %s;\n", toCamelCase(f.Name), tsType))
		} else {
			sb.WriteString(fmt.Sprintf("  %s?: %s;\n", toCamelCase(f.Name), tsType))
		}
	}
	sb.WriteString("}\n\n")

	// Update request type
	sb.WriteString(fmt.Sprintf("/**\n * Request to update a %s.\n */\n", entityLower))
	sb.WriteString(fmt.Sprintf("export interface Update%sRequest {\n", config.EntityName))
	for _, f := range config.Fields {
		if f.Primary {
			continue
		}
		tsType := goTSTypeMap(f.Type)
		sb.WriteString(fmt.Sprintf("  %s?: %s;\n", toCamelCase(f.Name), tsType))
	}
	sb.WriteString("}\n")

	return GeneratedFile{
		Name:    fmt.Sprintf("%s.ts", entityLower),
		Path:    fmt.Sprintf("%s/%s.ts", config.Options.PackageName, entityLower),
		Content: sb.String(),
	}
}

func (bg *BoilerplateGenerator) generateTSService(config BoilerplateConfig) GeneratedFile {
	var sb strings.Builder
	entityLower := strings.ToLower(config.EntityName)

	sb.WriteString(fmt.Sprintf("import { %s, Create%sRequest, Update%sRequest } from './%s';\n\n",
		config.EntityName, config.EntityName, config.EntityName, entityLower))

	sb.WriteString(fmt.Sprintf("/**\n * Service for %s operations.\n */\n", entityLower))
	sb.WriteString(fmt.Sprintf("export class %sService {\n", config.EntityName))
	sb.WriteString("  private baseUrl: string;\n\n")

	sb.WriteString("  constructor(baseUrl: string) {\n")
	sb.WriteString("    this.baseUrl = baseUrl;\n")
	sb.WriteString("  }\n\n")

	if config.Options.IncludeCreate {
		sb.WriteString(fmt.Sprintf("  async create(request: Create%sRequest): Promise<%s> {\n", config.EntityName, config.EntityName))
		sb.WriteString(fmt.Sprintf("    const response = await fetch(`${this.baseUrl}/%ss`, {\n", entityLower))
		sb.WriteString("      method: 'POST',\n")
		sb.WriteString("      headers: { 'Content-Type': 'application/json' },\n")
		sb.WriteString("      body: JSON.stringify(request),\n")
		sb.WriteString("    });\n")
		sb.WriteString("    return response.json();\n")
		sb.WriteString("  }\n\n")
	}

	if config.Options.IncludeRead {
		sb.WriteString(fmt.Sprintf("  async getById(id: string): Promise<%s> {\n", config.EntityName))
		sb.WriteString(fmt.Sprintf("    const response = await fetch(`${this.baseUrl}/%ss/${id}`);\n", entityLower))
		sb.WriteString("    return response.json();\n")
		sb.WriteString("  }\n\n")
	}

	if config.Options.IncludeUpdate {
		sb.WriteString(fmt.Sprintf("  async update(id: string, request: Update%sRequest): Promise<%s> {\n", config.EntityName, config.EntityName))
		sb.WriteString(fmt.Sprintf("    const response = await fetch(`${this.baseUrl}/%ss/${id}`, {\n", entityLower))
		sb.WriteString("      method: 'PUT',\n")
		sb.WriteString("      headers: { 'Content-Type': 'application/json' },\n")
		sb.WriteString("      body: JSON.stringify(request),\n")
		sb.WriteString("    });\n")
		sb.WriteString("    return response.json();\n")
		sb.WriteString("  }\n\n")
	}

	if config.Options.IncludeDelete {
		sb.WriteString("  async delete(id: string): Promise<void> {\n")
		sb.WriteString(fmt.Sprintf("    await fetch(`${this.baseUrl}/%ss/${id}`, { method: 'DELETE' });\n", entityLower))
		sb.WriteString("  }\n\n")
	}

	if config.Options.IncludeList {
		sb.WriteString(fmt.Sprintf("  async list(offset = 0, limit = 10): Promise<%s[]> {\n", config.EntityName))
		sb.WriteString(fmt.Sprintf("    const response = await fetch(`${this.baseUrl}/%ss?offset=${offset}&limit=${limit}`);\n", entityLower))
		sb.WriteString("    return response.json();\n")
		sb.WriteString("  }\n")
	}

	sb.WriteString("}\n")

	return GeneratedFile{
		Name:    fmt.Sprintf("%s.service.ts", entityLower),
		Path:    fmt.Sprintf("%s/%s.service.ts", config.Options.PackageName, entityLower),
		Content: sb.String(),
	}
}

// Helper functions

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

func goPythonTypeMap(goType string) string {
	switch goType {
	case "string":
		return "str"
	case "int", "int32", "int64":
		return "int"
	case "float32", "float64":
		return "float"
	case "bool":
		return "bool"
	case "[]string":
		return "list[str]"
	case "[]int":
		return "list[int]"
	default:
		return "Any"
	}
}

func goTSTypeMap(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "[]string":
		return "string[]"
	case "[]int":
		return "number[]"
	default:
		return "unknown"
	}
}
