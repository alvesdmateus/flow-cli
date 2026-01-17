package planner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PlanExecutor executes plans with validation and rollback support.
type PlanExecutor struct {
	plan             *Plan
	graph            *DependencyGraph
	results          map[string]interface{}
	completedTasks   map[string]bool
	executedTasks    []*Task // For rollback order
	autoRollback     bool
	validateAfterEach bool
	parallel         bool
	maxParallel      int
	logger           func(string)
	onTaskStart      func(*Task)
	onTaskComplete   func(*Task, error)
	onProgress       func(int, string)
	mu               sync.RWMutex
}

// ExecutorConfig contains configuration for plan execution.
type ExecutorConfig struct {
	AutoRollback      bool
	ValidateAfterEach bool
	Parallel          bool
	MaxParallel       int
}

// DefaultExecutorConfig returns sensible defaults.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		AutoRollback:      true,
		ValidateAfterEach: true,
		Parallel:          false,
		MaxParallel:       4,
	}
}

// NewPlanExecutor creates a new plan executor.
func NewPlanExecutor(plan *Plan, config ExecutorConfig) *PlanExecutor {
	graph := NewDependencyGraph()
	for _, task := range plan.Tasks {
		graph.AddTask(task)
	}

	return &PlanExecutor{
		plan:              plan,
		graph:             graph,
		results:           make(map[string]interface{}),
		completedTasks:    make(map[string]bool),
		executedTasks:     make([]*Task, 0),
		autoRollback:      config.AutoRollback,
		validateAfterEach: config.ValidateAfterEach,
		parallel:          config.Parallel,
		maxParallel:       config.MaxParallel,
	}
}

// SetLogger sets the logger function.
func (e *PlanExecutor) SetLogger(logger func(string)) {
	e.logger = logger
}

// OnTaskStart sets a callback for task start.
func (e *PlanExecutor) OnTaskStart(fn func(*Task)) {
	e.onTaskStart = fn
}

// OnTaskComplete sets a callback for task completion.
func (e *PlanExecutor) OnTaskComplete(fn func(*Task, error)) {
	e.onTaskComplete = fn
}

// OnProgress sets a callback for progress updates.
func (e *PlanExecutor) OnProgress(fn func(int, string)) {
	e.onProgress = fn
}

// Execute runs the plan.
func (e *PlanExecutor) Execute(ctx context.Context) error {
	// Validate graph first
	if err := e.graph.Validate(); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}

	now := time.Now()
	e.plan.StartedAt = &now
	e.plan.Status = TaskRunning

	var err error
	if e.parallel {
		err = e.executeParallel(ctx)
	} else {
		err = e.executeSequential(ctx)
	}

	completedAt := time.Now()
	e.plan.CompletedAt = &completedAt

	if err != nil {
		e.plan.Status = TaskFailed
		if e.autoRollback {
			if rollbackErr := e.Rollback(ctx); rollbackErr != nil {
				e.log("Rollback failed: %v", rollbackErr)
			}
		}
		return err
	}

	e.plan.Status = TaskCompleted
	return nil
}

func (e *PlanExecutor) executeSequential(ctx context.Context) error {
	sorted, err := e.graph.TopologicalSort()
	if err != nil {
		return err
	}

	for _, task := range sorted {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := e.executeTask(ctx, task); err != nil {
			return err
		}
	}

	return nil
}

func (e *PlanExecutor) executeParallel(ctx context.Context) error {
	levels, err := e.graph.GetExecutionLevels()
	if err != nil {
		return err
	}

	for _, level := range levels {
		if err := e.executeLevelParallel(ctx, level); err != nil {
			return err
		}
	}

	return nil
}

func (e *PlanExecutor) executeLevelParallel(ctx context.Context, taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}

	// Get tasks
	tasks := make([]*Task, 0, len(taskIDs))
	for _, id := range taskIDs {
		if task := e.plan.GetTask(id); task != nil {
			tasks = append(tasks, task)
		}
	}

	// Execute in parallel with limit
	semaphore := make(chan struct{}, e.maxParallel)
	errChan := make(chan error, len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := e.executeTask(ctx, t); err != nil {
				errChan <- err
			}
		}(task)
	}

	wg.Wait()
	close(errChan)

	// Return first error
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *PlanExecutor) executeTask(ctx context.Context, task *Task) error {
	// Check if already completed
	e.mu.RLock()
	if e.completedTasks[task.ID] {
		e.mu.RUnlock()
		return nil
	}
	e.mu.RUnlock()

	// Notify start
	if e.onTaskStart != nil {
		e.onTaskStart(task)
	}

	now := time.Now()
	task.StartedAt = &now
	task.Status = TaskRunning

	e.log("Starting task: %s", task.Name)

	// Create task context
	taskCtx := &TaskContext{
		Task:    task,
		Plan:    e.plan,
		Results: e.getResults(),
		Logger:  func(msg string) { e.log("[%s] %s", task.Name, msg) },
		OnProgress: func(percent int, msg string) {
			if e.onProgress != nil {
				e.onProgress(percent, fmt.Sprintf("[%s] %s", task.Name, msg))
			}
		},
	}

	// Execute with retries
	var execErr error
	for attempt := 0; attempt <= task.MaxRetries; attempt++ {
		if attempt > 0 {
			e.log("Retrying task %s (attempt %d/%d)", task.Name, attempt+1, task.MaxRetries+1)
			task.Retries = attempt
		}

		execErr = e.runTaskAction(ctx, task, taskCtx)
		if execErr == nil {
			break
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}

	if execErr != nil {
		task.Status = TaskFailed
		task.Error = execErr.Error()
		if e.onTaskComplete != nil {
			e.onTaskComplete(task, execErr)
		}
		return fmt.Errorf("task %s failed: %w", task.Name, execErr)
	}

	// Validate if configured
	if e.validateAfterEach && task.Validate != nil {
		e.log("Validating task: %s", task.Name)
		if validErr := task.Validate(taskCtx); validErr != nil {
			task.Status = TaskFailed
			task.Error = "validation failed: " + validErr.Error()
			if e.onTaskComplete != nil {
				e.onTaskComplete(task, validErr)
			}
			return fmt.Errorf("task %s validation failed: %w", task.Name, validErr)
		}
	}

	// Mark completed
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	task.Status = TaskCompleted
	task.ActualMS = completedAt.Sub(now).Milliseconds()

	e.mu.Lock()
	e.completedTasks[task.ID] = true
	e.executedTasks = append(e.executedTasks, task)
	if task.Result != nil {
		e.results[task.ID] = task.Result
	}
	e.mu.Unlock()

	e.log("Completed task: %s", task.Name)

	// Notify completion
	if e.onTaskComplete != nil {
		e.onTaskComplete(task, nil)
	}

	// Update progress
	if e.onProgress != nil {
		progress := e.plan.GetProgress()
		e.onProgress(progress, fmt.Sprintf("Completed: %s", task.Name))
	}

	return nil
}

func (e *PlanExecutor) runTaskAction(ctx context.Context, task *Task, taskCtx *TaskContext) error {
	if task.Action == nil {
		return nil // No action to run
	}

	// Run action with panic recovery
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		done <- task.Action(taskCtx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Rollback undoes executed tasks in reverse order.
func (e *PlanExecutor) Rollback(ctx context.Context) error {
	e.mu.RLock()
	tasks := make([]*Task, len(e.executedTasks))
	copy(tasks, e.executedTasks)
	e.mu.RUnlock()

	e.log("Rolling back %d tasks", len(tasks))

	// Rollback in reverse order
	var lastErr error
	for i := len(tasks) - 1; i >= 0; i-- {
		task := tasks[i]
		if task.Rollback == nil {
			e.log("Skipping rollback for %s (no rollback action)", task.Name)
			continue
		}

		e.log("Rolling back: %s", task.Name)

		taskCtx := &TaskContext{
			Task:    task,
			Plan:    e.plan,
			Results: e.getResults(),
			Logger:  func(msg string) { e.log("[ROLLBACK %s] %s", task.Name, msg) },
		}

		if err := task.Rollback(taskCtx); err != nil {
			e.log("Rollback failed for %s: %v", task.Name, err)
			lastErr = err
			// Continue rolling back other tasks
		} else {
			e.log("Rolled back: %s", task.Name)
		}
	}

	return lastErr
}

func (e *PlanExecutor) getResults() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	results := make(map[string]interface{})
	for k, v := range e.results {
		results[k] = v
	}
	return results
}

func (e *PlanExecutor) log(format string, args ...interface{}) {
	if e.logger != nil {
		e.logger(fmt.Sprintf(format, args...))
	}
}

// GetResults returns the results map.
func (e *PlanExecutor) GetResults() map[string]interface{} {
	return e.getResults()
}

// GetResult returns a specific result.
func (e *PlanExecutor) GetResult(taskID string) (interface{}, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result, ok := e.results[taskID]
	return result, ok
}

// ValidationResult contains the result of a validation check.
type ValidationResult struct {
	TaskID  string
	Valid   bool
	Message string
	Error   error
}

// Validator provides validation for task execution.
type Validator struct {
	checks map[string]ValidationCheck
	mu     sync.RWMutex
}

// ValidationCheck is a function that validates a task result.
type ValidationCheck func(ctx *TaskContext) error

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{
		checks: make(map[string]ValidationCheck),
	}
}

// RegisterCheck registers a validation check for a task type.
func (v *Validator) RegisterCheck(taskType string, check ValidationCheck) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.checks[taskType] = check
}

// Validate runs validation for a task.
func (v *Validator) Validate(ctx *TaskContext) error {
	v.mu.RLock()
	taskType, ok := ctx.Task.Metadata["type"].(string)
	if !ok {
		taskType = "generic"
	}
	check, found := v.checks[taskType]
	v.mu.RUnlock()

	if !found {
		return nil // No validation check registered
	}

	return check(ctx)
}

// ValidateAll validates all completed tasks in a plan.
func (v *Validator) ValidateAll(plan *Plan) []ValidationResult {
	var results []ValidationResult

	for _, task := range plan.Tasks {
		if task.Status != TaskCompleted {
			continue
		}

		ctx := &TaskContext{
			Task: task,
			Plan: plan,
		}

		err := v.Validate(ctx)
		result := ValidationResult{
			TaskID: task.ID,
			Valid:  err == nil,
		}
		if err != nil {
			result.Error = err
			result.Message = err.Error()
		}
		results = append(results, result)
	}

	return results
}

// RollbackManager manages rollback operations.
type RollbackManager struct {
	snapshots map[string]interface{}
	actions   map[string]TaskAction
	mu        sync.Mutex
}

// NewRollbackManager creates a new rollback manager.
func NewRollbackManager() *RollbackManager {
	return &RollbackManager{
		snapshots: make(map[string]interface{}),
		actions:   make(map[string]TaskAction),
	}
}

// SaveSnapshot saves a snapshot for rollback.
func (rm *RollbackManager) SaveSnapshot(taskID string, data interface{}) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.snapshots[taskID] = data
}

// GetSnapshot retrieves a saved snapshot.
func (rm *RollbackManager) GetSnapshot(taskID string) (interface{}, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	data, ok := rm.snapshots[taskID]
	return data, ok
}

// RegisterRollback registers a rollback action for a task.
func (rm *RollbackManager) RegisterRollback(taskID string, action TaskAction) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.actions[taskID] = action
}

// ExecuteRollback executes rollback for a specific task.
func (rm *RollbackManager) ExecuteRollback(ctx *TaskContext) error {
	rm.mu.Lock()
	action, ok := rm.actions[ctx.Task.ID]
	rm.mu.Unlock()

	if !ok {
		return nil
	}

	return action(ctx)
}

// Clear clears all snapshots and actions.
func (rm *RollbackManager) Clear() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.snapshots = make(map[string]interface{})
	rm.actions = make(map[string]TaskAction)
}
