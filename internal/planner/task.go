package planner

import (
	"fmt"
	"sync"
	"time"
)

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskRunning    TaskStatus = "running"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
	TaskSkipped    TaskStatus = "skipped"
	TaskBlocked    TaskStatus = "blocked"
)

// TaskPriority represents the priority of a task.
type TaskPriority int

const (
	PriorityLow TaskPriority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Task represents a unit of work in a plan.
type Task struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Status       TaskStatus             `json:"status"`
	Priority     TaskPriority           `json:"priority"`
	Dependencies []string               `json:"dependencies"` // IDs of tasks this depends on
	Subtasks     []*Task                `json:"subtasks,omitempty"`
	ParentID     string                 `json:"parent_id,omitempty"`
	EstimatedMS  int64                  `json:"estimated_ms,omitempty"`
	ActualMS     int64                  `json:"actual_ms,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Result       interface{}            `json:"result,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Retries      int                    `json:"retries"`
	MaxRetries   int                    `json:"max_retries"`

	// Execution
	Action     TaskAction `json:"-"`
	Rollback   TaskAction `json:"-"`
	Validate   TaskAction `json:"-"`
}

// TaskAction is a function that executes a task.
type TaskAction func(ctx *TaskContext) error

// TaskContext provides context for task execution.
type TaskContext struct {
	Task       *Task
	Plan       *Plan
	Results    map[string]interface{} // Results from previous tasks
	Logger     func(string)
	OnProgress func(percent int, message string)
}

// NewTask creates a new task.
func NewTask(id, name string) *Task {
	return &Task{
		ID:         id,
		Name:       name,
		Status:     TaskPending,
		Priority:   PriorityMedium,
		Metadata:   make(map[string]interface{}),
		MaxRetries: 0,
	}
}

// WithDescription sets the task description.
func (t *Task) WithDescription(desc string) *Task {
	t.Description = desc
	return t
}

// WithPriority sets the task priority.
func (t *Task) WithPriority(p TaskPriority) *Task {
	t.Priority = p
	return t
}

// WithDependencies sets the task dependencies.
func (t *Task) WithDependencies(deps ...string) *Task {
	t.Dependencies = deps
	return t
}

// WithAction sets the task action.
func (t *Task) WithAction(action TaskAction) *Task {
	t.Action = action
	return t
}

// WithRollback sets the rollback action.
func (t *Task) WithRollback(rollback TaskAction) *Task {
	t.Rollback = rollback
	return t
}

// WithValidation sets the validation action.
func (t *Task) WithValidation(validate TaskAction) *Task {
	t.Validate = validate
	return t
}

// WithRetries sets the max retries.
func (t *Task) WithRetries(max int) *Task {
	t.MaxRetries = max
	return t
}

// AddSubtask adds a subtask.
func (t *Task) AddSubtask(subtask *Task) *Task {
	subtask.ParentID = t.ID
	t.Subtasks = append(t.Subtasks, subtask)
	return t
}

// IsBlocked returns true if task is blocked by dependencies.
func (t *Task) IsBlocked(completedTasks map[string]bool) bool {
	for _, dep := range t.Dependencies {
		if !completedTasks[dep] {
			return true
		}
	}
	return false
}

// CanRun returns true if task can run.
func (t *Task) CanRun(completedTasks map[string]bool) bool {
	return t.Status == TaskPending && !t.IsBlocked(completedTasks)
}

// Plan represents a collection of tasks to execute.
type Plan struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tasks       []*Task                `json:"tasks"`
	Status      TaskStatus             `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	taskIndex   map[string]*Task
	mu          sync.RWMutex
}

// NewPlan creates a new plan.
func NewPlan(id, name string) *Plan {
	return &Plan{
		ID:        id,
		Name:      name,
		Tasks:     make([]*Task, 0),
		Status:    TaskPending,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
		taskIndex: make(map[string]*Task),
	}
}

// AddTask adds a task to the plan.
func (p *Plan) AddTask(task *Task) *Plan {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Tasks = append(p.Tasks, task)
	p.taskIndex[task.ID] = task
	p.indexSubtasks(task)
	return p
}

func (p *Plan) indexSubtasks(task *Task) {
	for _, subtask := range task.Subtasks {
		p.taskIndex[subtask.ID] = subtask
		p.indexSubtasks(subtask)
	}
}

// GetTask returns a task by ID.
func (p *Plan) GetTask(id string) *Task {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.taskIndex[id]
}

// GetReadyTasks returns tasks that are ready to run.
func (p *Plan) GetReadyTasks() []*Task {
	p.mu.RLock()
	defer p.mu.RUnlock()

	completed := p.getCompletedTaskIDs()
	var ready []*Task

	for _, task := range p.Tasks {
		if task.CanRun(completed) {
			ready = append(ready, task)
		}
	}

	return ready
}

func (p *Plan) getCompletedTaskIDs() map[string]bool {
	completed := make(map[string]bool)
	for _, task := range p.Tasks {
		if task.Status == TaskCompleted {
			completed[task.ID] = true
		}
	}
	return completed
}

// IsComplete returns true if all tasks are completed.
func (p *Plan) IsComplete() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, task := range p.Tasks {
		if task.Status != TaskCompleted && task.Status != TaskSkipped {
			return false
		}
	}
	return true
}

// HasFailed returns true if any task has failed.
func (p *Plan) HasFailed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, task := range p.Tasks {
		if task.Status == TaskFailed {
			return true
		}
	}
	return false
}

// GetProgress returns the progress percentage.
func (p *Plan) GetProgress() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.Tasks) == 0 {
		return 100
	}

	completed := 0
	for _, task := range p.Tasks {
		if task.Status == TaskCompleted || task.Status == TaskSkipped {
			completed++
		}
	}

	return (completed * 100) / len(p.Tasks)
}

// GetStats returns statistics about the plan.
func (p *Plan) GetStats() PlanStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PlanStats{
		Total: len(p.Tasks),
	}

	for _, task := range p.Tasks {
		switch task.Status {
		case TaskPending:
			stats.Pending++
		case TaskRunning:
			stats.Running++
		case TaskCompleted:
			stats.Completed++
		case TaskFailed:
			stats.Failed++
		case TaskSkipped:
			stats.Skipped++
		case TaskBlocked:
			stats.Blocked++
		}
	}

	return stats
}

// PlanStats contains statistics about a plan.
type PlanStats struct {
	Total     int
	Pending   int
	Running   int
	Completed int
	Failed    int
	Skipped   int
	Blocked   int
}

// TaskDecomposer breaks down complex tasks into subtasks.
type TaskDecomposer struct {
	strategies map[string]DecompositionStrategy
	mu         sync.RWMutex
}

// DecompositionStrategy defines how to decompose a task type.
type DecompositionStrategy func(task *Task) ([]*Task, error)

// NewTaskDecomposer creates a new task decomposer.
func NewTaskDecomposer() *TaskDecomposer {
	td := &TaskDecomposer{
		strategies: make(map[string]DecompositionStrategy),
	}

	// Register default strategies
	td.RegisterStrategy("file_edit", fileEditStrategy)
	td.RegisterStrategy("code_refactor", codeRefactorStrategy)
	td.RegisterStrategy("feature_add", featureAddStrategy)
	td.RegisterStrategy("bug_fix", bugFixStrategy)

	return td
}

// RegisterStrategy registers a decomposition strategy.
func (td *TaskDecomposer) RegisterStrategy(taskType string, strategy DecompositionStrategy) {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.strategies[taskType] = strategy
}

// Decompose breaks down a task using the appropriate strategy.
func (td *TaskDecomposer) Decompose(task *Task) ([]*Task, error) {
	td.mu.RLock()
	taskType, ok := task.Metadata["type"].(string)
	if !ok {
		taskType = "generic"
	}
	strategy, found := td.strategies[taskType]
	td.mu.RUnlock()

	if !found {
		// Return as single task if no strategy
		return []*Task{task}, nil
	}

	return strategy(task)
}

// Default decomposition strategies

func fileEditStrategy(task *Task) ([]*Task, error) {
	subtasks := []*Task{
		NewTask(task.ID+"-read", "Read file").
			WithDescription("Read the current file contents").
			WithPriority(PriorityHigh),
		NewTask(task.ID+"-backup", "Create backup").
			WithDescription("Create a backup of the file").
			WithDependencies(task.ID + "-read"),
		NewTask(task.ID+"-edit", "Apply edits").
			WithDescription("Apply the requested edits").
			WithDependencies(task.ID + "-backup"),
		NewTask(task.ID+"-validate", "Validate changes").
			WithDescription("Validate the changes are correct").
			WithDependencies(task.ID + "-edit"),
	}
	return subtasks, nil
}

func codeRefactorStrategy(task *Task) ([]*Task, error) {
	subtasks := []*Task{
		NewTask(task.ID+"-analyze", "Analyze code").
			WithDescription("Analyze the code to understand structure").
			WithPriority(PriorityHigh),
		NewTask(task.ID+"-plan", "Plan refactoring").
			WithDescription("Create a refactoring plan").
			WithDependencies(task.ID + "-analyze"),
		NewTask(task.ID+"-backup", "Backup files").
			WithDescription("Backup affected files").
			WithDependencies(task.ID + "-plan"),
		NewTask(task.ID+"-refactor", "Apply refactoring").
			WithDescription("Apply the refactoring changes").
			WithDependencies(task.ID + "-backup"),
		NewTask(task.ID+"-test", "Run tests").
			WithDescription("Run tests to verify refactoring").
			WithDependencies(task.ID + "-refactor"),
		NewTask(task.ID+"-cleanup", "Cleanup").
			WithDescription("Clean up temporary files and artifacts").
			WithDependencies(task.ID + "-test"),
	}
	return subtasks, nil
}

func featureAddStrategy(task *Task) ([]*Task, error) {
	subtasks := []*Task{
		NewTask(task.ID+"-design", "Design feature").
			WithDescription("Design the feature architecture").
			WithPriority(PriorityHigh),
		NewTask(task.ID+"-deps", "Add dependencies").
			WithDescription("Add required dependencies").
			WithDependencies(task.ID + "-design"),
		NewTask(task.ID+"-implement", "Implement feature").
			WithDescription("Implement the feature code").
			WithDependencies(task.ID + "-deps"),
		NewTask(task.ID+"-test", "Write tests").
			WithDescription("Write tests for the feature").
			WithDependencies(task.ID + "-implement"),
		NewTask(task.ID+"-docs", "Update documentation").
			WithDescription("Update documentation").
			WithDependencies(task.ID + "-test"),
	}
	return subtasks, nil
}

func bugFixStrategy(task *Task) ([]*Task, error) {
	subtasks := []*Task{
		NewTask(task.ID+"-reproduce", "Reproduce bug").
			WithDescription("Reproduce the bug to understand it").
			WithPriority(PriorityCritical),
		NewTask(task.ID+"-diagnose", "Diagnose root cause").
			WithDescription("Find the root cause of the bug").
			WithDependencies(task.ID + "-reproduce"),
		NewTask(task.ID+"-fix", "Apply fix").
			WithDescription("Apply the bug fix").
			WithDependencies(task.ID + "-diagnose"),
		NewTask(task.ID+"-test", "Test fix").
			WithDescription("Test that the fix works").
			WithDependencies(task.ID + "-fix"),
		NewTask(task.ID+"-regression", "Check for regressions").
			WithDescription("Ensure no regressions were introduced").
			WithDependencies(task.ID + "-test"),
	}
	return subtasks, nil
}

// TaskBuilder helps build complex tasks fluently.
type TaskBuilder struct {
	task *Task
}

// NewTaskBuilder creates a new task builder.
func NewTaskBuilder(id, name string) *TaskBuilder {
	return &TaskBuilder{
		task: NewTask(id, name),
	}
}

// Description sets the description.
func (b *TaskBuilder) Description(desc string) *TaskBuilder {
	b.task.Description = desc
	return b
}

// Priority sets the priority.
func (b *TaskBuilder) Priority(p TaskPriority) *TaskBuilder {
	b.task.Priority = p
	return b
}

// DependsOn adds dependencies.
func (b *TaskBuilder) DependsOn(deps ...string) *TaskBuilder {
	b.task.Dependencies = append(b.task.Dependencies, deps...)
	return b
}

// Action sets the action.
func (b *TaskBuilder) Action(action TaskAction) *TaskBuilder {
	b.task.Action = action
	return b
}

// OnRollback sets the rollback action.
func (b *TaskBuilder) OnRollback(rollback TaskAction) *TaskBuilder {
	b.task.Rollback = rollback
	return b
}

// OnValidate sets the validation action.
func (b *TaskBuilder) OnValidate(validate TaskAction) *TaskBuilder {
	b.task.Validate = validate
	return b
}

// Retries sets max retries.
func (b *TaskBuilder) Retries(max int) *TaskBuilder {
	b.task.MaxRetries = max
	return b
}

// Meta adds metadata.
func (b *TaskBuilder) Meta(key string, value interface{}) *TaskBuilder {
	b.task.Metadata[key] = value
	return b
}

// Subtask adds a subtask.
func (b *TaskBuilder) Subtask(subtask *Task) *TaskBuilder {
	b.task.AddSubtask(subtask)
	return b
}

// Build returns the built task.
func (b *TaskBuilder) Build() *Task {
	return b.task
}

// String returns the string representation of task status.
func (s TaskStatus) String() string {
	return string(s)
}

// String returns the string representation of task priority.
func (p TaskPriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return fmt.Sprintf("priority(%d)", p)
	}
}
