package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	vibeContext "github.com/mateus/vibe-cli/internal/context"
	"github.com/mateus/vibe-cli/internal/llm"
)

// PlanPhase represents a phase in the planning workflow
type PlanPhase int

const (
	PhaseGathering PlanPhase = iota // Gathering requirements
	PhaseAnalyzing                  // Analyzing the codebase
	PhaseDesigning                  // Designing the solution
	PhaseReviewing                  // Reviewing with user
	PhaseExecuting                  // Executing the plan
	PhaseComplete                   // Planning complete
)

func (p PlanPhase) String() string {
	switch p {
	case PhaseGathering:
		return "Gathering Requirements"
	case PhaseAnalyzing:
		return "Analyzing Codebase"
	case PhaseDesigning:
		return "Designing Solution"
	case PhaseReviewing:
		return "Reviewing Plan"
	case PhaseExecuting:
		return "Executing Plan"
	case PhaseComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// PlanTask represents a task in the implementation plan
type PlanTask struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Priority     string   `json:"priority"` // critical, high, medium, low
	Files        []string `json:"files"`
	Dependencies []string `json:"dependencies"`
	Completed    bool     `json:"completed"`
}

// Plan represents an architecture/implementation plan
type Plan struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	Goals       []string   `json:"goals"`
	Constraints []string   `json:"constraints"`
	Tasks       []PlanTask `json:"tasks"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Phase       PlanPhase  `json:"phase"`
}

// ClarifyingQuestion represents a question to ask the user
type ClarifyingQuestion struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Context  string   `json:"context"`
	Required bool     `json:"required"`
}

// PlannerHandler handles planner events
type PlannerHandler interface {
	OnPhaseChange(phase PlanPhase)
	OnQuestion(question ClarifyingQuestion) (string, error)
	OnPlanUpdate(plan *Plan)
	OnTaskStart(task PlanTask)
	OnTaskComplete(task PlanTask, success bool)
	OnMessage(message string)
	OnError(err error)
}

// Planner orchestrates the architecture planning workflow
type Planner struct {
	llmClient  llm.Client
	ctxManager *vibeContext.Manager
	model      string
	plan       *Plan
	phase      PlanPhase
	answers    map[string]string // Collected answers to questions
}

// PlannerConfig holds planner configuration
type PlannerConfig struct {
	LLMClient llm.Client
	Model     string
}

// NewPlanner creates a new planner
func NewPlanner(cfg PlannerConfig) *Planner {
	ctxManager := vibeContext.NewManager(ArchitectureSystemPrompt(), 100)
	ctxManager.SetModel(cfg.Model)

	return &Planner{
		llmClient:  cfg.LLMClient,
		ctxManager: ctxManager,
		model:      cfg.Model,
		phase:      PhaseGathering,
		answers:    make(map[string]string),
	}
}

// StartPlanning begins the planning workflow
func (p *Planner) StartPlanning(ctx context.Context, request string, handler PlannerHandler) (*Plan, error) {
	// Initialize plan
	p.plan = &Plan{
		ID:        fmt.Sprintf("plan_%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Phase:     PhaseGathering,
	}

	// Phase 1: Gather requirements through questions
	handler.OnPhaseChange(PhaseGathering)
	handler.OnMessage("Let me understand your requirements better...")

	questions, err := p.generateClarifyingQuestions(ctx, request)
	if err != nil {
		handler.OnError(err)
		return nil, err
	}

	// Ask each question
	for _, q := range questions {
		answer, err := handler.OnQuestion(q)
		if err != nil {
			if q.Required {
				return nil, fmt.Errorf("required question not answered: %s", q.Question)
			}
			continue
		}
		p.answers[q.Question] = answer
	}

	// Phase 2: Analyze and design
	handler.OnPhaseChange(PhaseAnalyzing)
	handler.OnMessage("Analyzing requirements and designing solution...")

	plan, err := p.generatePlan(ctx, request)
	if err != nil {
		handler.OnError(err)
		return nil, err
	}

	p.plan = plan
	p.plan.Phase = PhaseDesigning

	// Phase 3: Review with user
	handler.OnPhaseChange(PhaseReviewing)
	handler.OnPlanUpdate(p.plan)

	return p.plan, nil
}

// generateClarifyingQuestions uses LLM to generate relevant questions
func (p *Planner) generateClarifyingQuestions(ctx context.Context, request string) ([]ClarifyingQuestion, error) {
	prompt := fmt.Sprintf(`Based on this request, generate 2-4 clarifying questions to better understand the requirements.

Request: %s

For each question, provide:
1. The question itself
2. 3-4 multiple choice options
3. Brief context about why this matters

Format your response as a list of questions, each with options.
Focus on:
- Technical approach choices
- Scope clarification
- Priority decisions
- Constraint identification

Keep questions focused and actionable.`, request)

	p.ctxManager.AddUserMessage(prompt)

	messages := p.ctxManager.GetMessages()
	response, err := p.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
		Model:       p.model,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}

	p.ctxManager.AddAssistantMessage(response, nil)

	// Parse questions from response
	questions := p.parseQuestions(response)

	// Add default questions if none generated
	if len(questions) == 0 {
		questions = p.getDefaultQuestions(request)
	}

	return questions, nil
}

// parseQuestions extracts questions from LLM response
func (p *Planner) parseQuestions(response string) []ClarifyingQuestion {
	var questions []ClarifyingQuestion

	// Simple parsing - look for numbered questions with options
	lines := strings.Split(response, "\n")
	var currentQ *ClarifyingQuestion

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for question markers
		if strings.HasPrefix(line, "Question") || strings.HasPrefix(line, "Q:") || strings.Contains(line, "?") {
			if currentQ != nil && len(currentQ.Options) > 0 {
				questions = append(questions, *currentQ)
			}
			currentQ = &ClarifyingQuestion{
				Question: cleanQuestion(line),
				Options:  make([]string, 0),
				Required: true,
			}
		} else if currentQ != nil {
			// Check for option markers
			if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") ||
				strings.HasPrefix(line, "a)") || strings.HasPrefix(line, "b)") ||
				strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") {
				option := cleanOption(line)
				if option != "" {
					currentQ.Options = append(currentQ.Options, option)
				}
			}
		}
	}

	// Add last question
	if currentQ != nil && len(currentQ.Options) > 0 {
		questions = append(questions, *currentQ)
	}

	return questions
}

// cleanQuestion removes prefixes from questions
func cleanQuestion(s string) string {
	s = strings.TrimPrefix(s, "Question:")
	s = strings.TrimPrefix(s, "Q:")
	s = strings.TrimSpace(s)
	// Remove numbering
	for i := 1; i <= 10; i++ {
		s = strings.TrimPrefix(s, fmt.Sprintf("%d.", i))
		s = strings.TrimPrefix(s, fmt.Sprintf("%d)", i))
	}
	return strings.TrimSpace(s)
}

// cleanOption removes prefixes from options
func cleanOption(s string) string {
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "•")
	s = strings.TrimPrefix(s, "*")
	for _, prefix := range []string{"a)", "b)", "c)", "d)", "1.", "2.", "3.", "4."} {
		s = strings.TrimPrefix(s, prefix)
	}
	return strings.TrimSpace(s)
}

// getDefaultQuestions returns fallback questions
func (p *Planner) getDefaultQuestions(request string) []ClarifyingQuestion {
	return []ClarifyingQuestion{
		{
			Question: "What is the priority for this implementation?",
			Options:  []string{"Critical - needs to be done now", "High - important but can wait", "Medium - nice to have", "Low - when time permits"},
			Context:  "Helps determine the level of detail and effort",
			Required: true,
		},
		{
			Question: "How should we approach the implementation?",
			Options:  []string{"Minimal - just the essentials", "Standard - balanced approach", "Comprehensive - full featured", "Incremental - start small, iterate"},
			Context:  "Determines the scope and complexity",
			Required: true,
		},
	}
}

// generatePlan creates the implementation plan
func (p *Planner) generatePlan(ctx context.Context, request string) (*Plan, error) {
	// Build context with answers
	var answersContext strings.Builder
	answersContext.WriteString("User's answers to clarifying questions:\n")
	for q, a := range p.answers {
		answersContext.WriteString(fmt.Sprintf("Q: %s\nA: %s\n\n", q, a))
	}

	prompt := fmt.Sprintf(`Based on this request and the user's answers, create a detailed implementation plan.

Original Request: %s

%s

Create a structured plan with:
1. A clear title and summary
2. Specific goals (what we're trying to achieve)
3. Constraints or considerations
4. Implementation tasks in priority order

For each task, include:
- A clear title
- Description of what needs to be done
- Which files will be affected
- Priority level (critical/high/medium/low)
- Dependencies on other tasks

Format the plan clearly so it can be reviewed and approved.`, request, answersContext.String())

	p.ctxManager.AddUserMessage(prompt)

	messages := p.ctxManager.GetMessages()
	response, err := p.llmClient.ChatSync(ctx, messages, llm.ChatOptions{
		Model:       p.model,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}

	p.ctxManager.AddAssistantMessage(response, nil)

	// Parse plan from response
	plan := p.parsePlan(response)
	plan.ID = p.plan.ID
	plan.CreatedAt = p.plan.CreatedAt
	plan.UpdatedAt = time.Now()

	return plan, nil
}

// parsePlan extracts plan structure from LLM response
func (p *Planner) parsePlan(response string) *Plan {
	plan := &Plan{
		Goals:       make([]string, 0),
		Constraints: make([]string, 0),
		Tasks:       make([]PlanTask, 0),
	}

	lines := strings.Split(response, "\n")
	section := ""
	taskCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)

		// Detect sections
		if strings.Contains(lower, "title:") || strings.HasPrefix(lower, "# ") {
			plan.Title = cleanSection(line)
			section = "title"
		} else if strings.Contains(lower, "summary") {
			section = "summary"
		} else if strings.Contains(lower, "goal") {
			section = "goals"
		} else if strings.Contains(lower, "constraint") || strings.Contains(lower, "consideration") {
			section = "constraints"
		} else if strings.Contains(lower, "task") || strings.Contains(lower, "implementation") {
			section = "tasks"
		} else {
			// Add content to current section
			switch section {
			case "summary":
				if plan.Summary == "" {
					plan.Summary = line
				} else {
					plan.Summary += " " + line
				}
			case "goals":
				if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
					plan.Goals = append(plan.Goals, cleanOption(line))
				}
			case "constraints":
				if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
					plan.Constraints = append(plan.Constraints, cleanOption(line))
				}
			case "tasks":
				// Check if this is a new task
				if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") ||
					strings.HasPrefix(line, "*") || isNumbered(line) {
					taskCount++
					task := PlanTask{
						ID:       fmt.Sprintf("task_%d", taskCount),
						Title:    cleanOption(line),
						Priority: "medium",
					}
					plan.Tasks = append(plan.Tasks, task)
				} else if len(plan.Tasks) > 0 {
					// Add as description to last task
					lastIdx := len(plan.Tasks) - 1
					if plan.Tasks[lastIdx].Description == "" {
						plan.Tasks[lastIdx].Description = line
					} else {
						plan.Tasks[lastIdx].Description += " " + line
					}
				}
			}
		}
	}

	// Set defaults if empty
	if plan.Title == "" {
		plan.Title = "Implementation Plan"
	}
	if plan.Summary == "" {
		plan.Summary = "Generated implementation plan"
	}

	return plan
}

func cleanSection(s string) string {
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(s, "Title:")
	return strings.TrimSpace(s)
}

func isNumbered(s string) bool {
	for i := 1; i <= 20; i++ {
		if strings.HasPrefix(s, fmt.Sprintf("%d.", i)) || strings.HasPrefix(s, fmt.Sprintf("%d)", i)) {
			return true
		}
	}
	return false
}

// GetPlan returns the current plan
func (p *Planner) GetPlan() *Plan {
	return p.plan
}

// GetPhase returns the current phase
func (p *Planner) GetPhase() PlanPhase {
	return p.phase
}

// ApprovePlan marks the plan as approved and ready for execution
func (p *Planner) ApprovePlan() {
	if p.plan != nil {
		p.plan.Phase = PhaseExecuting
		p.phase = PhaseExecuting
	}
}

// ArchitectureSystemPrompt returns the system prompt for architecture mode
func ArchitectureSystemPrompt() string {
	return `You are vibe-cli in Architecture Mode - a specialized planning assistant.

Your role is to help users plan and design software implementations BEFORE writing code.

In this mode, you should:
1. Ask clarifying questions to understand requirements fully
2. Analyze the existing codebase structure if relevant
3. Propose a clear, phased implementation plan
4. Identify potential challenges and constraints
5. Break down work into actionable tasks

Guidelines:
- Be thorough but concise
- Prioritize tasks logically
- Consider dependencies between tasks
- Identify files that will need changes
- Suggest testing strategies
- Think about error handling and edge cases

Format plans with clear structure:
- Title and summary
- Goals (what we're achieving)
- Constraints (limitations to consider)
- Tasks (ordered by priority with descriptions)

Always get user approval before proceeding to implementation.
Never skip the planning phase - good architecture leads to good code.`
}
