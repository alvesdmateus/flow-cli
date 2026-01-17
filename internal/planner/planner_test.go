package planner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== Task Tests ====================

func TestNewTask(t *testing.T) {
	task := NewTask("task-1", "Test Task")

	if task.ID != "task-1" {
		t.Errorf("Expected ID 'task-1', got %s", task.ID)
	}
	if task.Name != "Test Task" {
		t.Errorf("Expected Name 'Test Task', got %s", task.Name)
	}
	if task.Status != TaskPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}
}

func TestTaskBuilder(t *testing.T) {
	task := NewTaskBuilder("task-1", "Build Test").
		Description("A test task").
		Priority(PriorityHigh).
		DependsOn("dep-1", "dep-2").
		Retries(3).
		Meta("key", "value").
		Build()

	if task.Description != "A test task" {
		t.Errorf("Description not set correctly")
	}
	if task.Priority != PriorityHigh {
		t.Errorf("Priority not set correctly")
	}
	if len(task.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task.Dependencies))
	}
	if task.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", task.MaxRetries)
	}
	if task.Metadata["key"] != "value" {
		t.Errorf("Metadata not set correctly")
	}
}

func TestTaskSubtasks(t *testing.T) {
	parent := NewTask("parent", "Parent Task")
	child1 := NewTask("child-1", "Child 1")
	child2 := NewTask("child-2", "Child 2")

	parent.AddSubtask(child1).AddSubtask(child2)

	if len(parent.Subtasks) != 2 {
		t.Errorf("Expected 2 subtasks, got %d", len(parent.Subtasks))
	}
	if child1.ParentID != "parent" {
		t.Errorf("Child 1 parent ID not set")
	}
}

func TestTaskBlocking(t *testing.T) {
	task := NewTask("task-1", "Task").WithDependencies("dep-1", "dep-2")

	completed := map[string]bool{}
	if !task.IsBlocked(completed) {
		t.Error("Task should be blocked with no completed dependencies")
	}

	completed["dep-1"] = true
	if !task.IsBlocked(completed) {
		t.Error("Task should be blocked with partial dependencies")
	}

	completed["dep-2"] = true
	if task.IsBlocked(completed) {
		t.Error("Task should not be blocked with all dependencies completed")
	}
}

func TestTaskDecomposer(t *testing.T) {
	decomposer := NewTaskDecomposer()

	task := NewTask("edit-1", "Edit File")
	task.Metadata["type"] = "file_edit"

	subtasks, err := decomposer.Decompose(task)
	if err != nil {
		t.Fatalf("Decompose failed: %v", err)
	}

	if len(subtasks) != 4 {
		t.Errorf("Expected 4 subtasks for file_edit, got %d", len(subtasks))
	}
}

// ==================== Plan Tests ====================

func TestNewPlan(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")

	if plan.ID != "plan-1" {
		t.Errorf("Expected ID 'plan-1', got %s", plan.ID)
	}
	if plan.Status != TaskPending {
		t.Errorf("Expected pending status")
	}
}

func TestPlanAddTask(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")
	task := NewTask("task-1", "Task 1")

	plan.AddTask(task)

	if len(plan.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(plan.Tasks))
	}
	if plan.GetTask("task-1") == nil {
		t.Error("GetTask should return the task")
	}
}

func TestPlanProgress(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")
	plan.AddTask(NewTask("task-1", "Task 1"))
	plan.AddTask(NewTask("task-2", "Task 2"))

	if plan.GetProgress() != 0 {
		t.Errorf("Expected 0%% progress, got %d%%", plan.GetProgress())
	}

	plan.Tasks[0].Status = TaskCompleted
	if plan.GetProgress() != 50 {
		t.Errorf("Expected 50%% progress, got %d%%", plan.GetProgress())
	}

	plan.Tasks[1].Status = TaskCompleted
	if plan.GetProgress() != 100 {
		t.Errorf("Expected 100%% progress, got %d%%", plan.GetProgress())
	}
}

func TestPlanStats(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")
	plan.AddTask(NewTask("task-1", "Task 1"))
	plan.AddTask(NewTask("task-2", "Task 2"))
	plan.AddTask(NewTask("task-3", "Task 3"))

	plan.Tasks[0].Status = TaskCompleted
	plan.Tasks[1].Status = TaskRunning
	plan.Tasks[2].Status = TaskFailed

	stats := plan.GetStats()

	if stats.Total != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total)
	}
	if stats.Completed != 1 {
		t.Errorf("Expected 1 completed, got %d", stats.Completed)
	}
	if stats.Running != 1 {
		t.Errorf("Expected 1 running, got %d", stats.Running)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", stats.Failed)
	}
}

// ==================== Graph Tests ====================

func TestDependencyGraphAdd(t *testing.T) {
	g := NewDependencyGraph()

	task1 := NewTask("task-1", "Task 1")
	task2 := NewTask("task-2", "Task 2").WithDependencies("task-1")

	g.AddTask(task1)
	g.AddTask(task2)

	if g.Size() != 2 {
		t.Errorf("Expected 2 nodes, got %d", g.Size())
	}

	deps := g.GetDependencies("task-2")
	if len(deps) != 1 || deps[0] != "task-1" {
		t.Errorf("Dependencies not set correctly")
	}
}

func TestDependencyGraphCycle(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEdge("a", "b")
	g.AddEdge("b", "c")

	if g.HasCycle() {
		t.Error("Should not have cycle")
	}

	g.AddEdge("c", "a")
	if !g.HasCycle() {
		t.Error("Should detect cycle")
	}
}

func TestDependencyGraphTopologicalSort(t *testing.T) {
	g := NewDependencyGraph()

	task1 := NewTask("task-1", "Task 1")
	task2 := NewTask("task-2", "Task 2").WithDependencies("task-1")
	task3 := NewTask("task-3", "Task 3").WithDependencies("task-1")
	task4 := NewTask("task-4", "Task 4").WithDependencies("task-2", "task-3")

	g.AddTask(task1)
	g.AddTask(task2)
	g.AddTask(task3)
	g.AddTask(task4)

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(sorted) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(sorted))
	}

	// task-1 should be first
	if sorted[0].ID != "task-1" {
		t.Errorf("Expected task-1 first, got %s", sorted[0].ID)
	}

	// task-4 should be last
	if sorted[len(sorted)-1].ID != "task-4" {
		t.Errorf("Expected task-4 last, got %s", sorted[len(sorted)-1].ID)
	}
}

func TestDependencyGraphExecutionLevels(t *testing.T) {
	g := NewDependencyGraph()

	task1 := NewTask("task-1", "Task 1")
	task2 := NewTask("task-2", "Task 2").WithDependencies("task-1")
	task3 := NewTask("task-3", "Task 3").WithDependencies("task-1")
	task4 := NewTask("task-4", "Task 4").WithDependencies("task-2", "task-3")

	g.AddTask(task1)
	g.AddTask(task2)
	g.AddTask(task3)
	g.AddTask(task4)

	levels, err := g.GetExecutionLevels()
	if err != nil {
		t.Fatalf("GetExecutionLevels failed: %v", err)
	}

	if len(levels) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(levels))
	}

	// Level 0: task-1
	if len(levels[0]) != 1 {
		t.Errorf("Expected 1 task at level 0, got %d", len(levels[0]))
	}

	// Level 1: task-2, task-3 (can run in parallel)
	if len(levels[1]) != 2 {
		t.Errorf("Expected 2 tasks at level 1, got %d", len(levels[1]))
	}

	// Level 2: task-4
	if len(levels[2]) != 1 {
		t.Errorf("Expected 1 task at level 2, got %d", len(levels[2]))
	}
}

func TestDependencyGraphCriticalPath(t *testing.T) {
	g := NewDependencyGraph()

	g.AddTask(NewTask("a", "A"))
	g.AddTask(NewTask("b", "B").WithDependencies("a"))
	g.AddTask(NewTask("c", "C").WithDependencies("a"))
	g.AddTask(NewTask("d", "D").WithDependencies("b"))
	g.AddTask(NewTask("e", "E").WithDependencies("c"))
	g.AddTask(NewTask("f", "F").WithDependencies("d", "e"))

	path, err := g.GetCriticalPath()
	if err != nil {
		t.Fatalf("GetCriticalPath failed: %v", err)
	}

	// Critical path should be a -> b -> d -> f or a -> c -> e -> f (both length 4)
	if len(path) != 4 {
		t.Errorf("Expected critical path length 4, got %d", len(path))
	}
}

// ==================== Executor Tests ====================

func TestExecutorSequential(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")

	var executed []string
	var mu atomic.Int32

	task1 := NewTask("task-1", "Task 1")
	task1.Action = func(ctx *TaskContext) error {
		executed = append(executed, "task-1")
		mu.Add(1)
		return nil
	}

	task2 := NewTask("task-2", "Task 2").WithDependencies("task-1")
	task2.Action = func(ctx *TaskContext) error {
		executed = append(executed, "task-2")
		mu.Add(1)
		return nil
	}

	plan.AddTask(task1).AddTask(task2)

	config := DefaultExecutorConfig()
	config.Parallel = false
	executor := NewPlanExecutor(plan, config)

	err := executor.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if mu.Load() != 2 {
		t.Errorf("Expected 2 tasks executed, got %d", mu.Load())
	}

	if executed[0] != "task-1" || executed[1] != "task-2" {
		t.Error("Tasks executed in wrong order")
	}
}

func TestExecutorRollback(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")

	var rollbackCalled bool

	task1 := NewTask("task-1", "Task 1")
	task1.Action = func(ctx *TaskContext) error {
		return nil
	}
	task1.Rollback = func(ctx *TaskContext) error {
		rollbackCalled = true
		return nil
	}

	task2 := NewTask("task-2", "Task 2").WithDependencies("task-1")
	task2.Action = func(ctx *TaskContext) error {
		return errors.New("task-2 failed")
	}

	plan.AddTask(task1).AddTask(task2)

	config := DefaultExecutorConfig()
	config.AutoRollback = true
	executor := NewPlanExecutor(plan, config)

	err := executor.Execute(context.Background())
	if err == nil {
		t.Error("Expected error from failing task")
	}

	if !rollbackCalled {
		t.Error("Expected rollback to be called")
	}
}

func TestExecutorValidation(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")

	task := NewTask("task-1", "Task 1")
	task.Action = func(ctx *TaskContext) error {
		return nil
	}
	task.Validate = func(ctx *TaskContext) error {
		return errors.New("validation failed")
	}

	plan.AddTask(task)

	config := DefaultExecutorConfig()
	config.ValidateAfterEach = true
	executor := NewPlanExecutor(plan, config)

	err := executor.Execute(context.Background())
	if err == nil {
		t.Error("Expected validation error")
	}

	if task.Status != TaskFailed {
		t.Errorf("Expected task status failed, got %s", task.Status)
	}
}

func TestExecutorRetries(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")

	attempts := 0
	task := NewTask("task-1", "Task 1").WithRetries(2)
	task.Action = func(ctx *TaskContext) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	plan.AddTask(task)

	config := DefaultExecutorConfig()
	executor := NewPlanExecutor(plan, config)

	err := executor.Execute(context.Background())
	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestExecutorContextCancellation(t *testing.T) {
	plan := NewPlan("plan-1", "Test Plan")

	task := NewTask("task-1", "Task 1")
	task.Action = func(ctx *TaskContext) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	plan.AddTask(task)

	config := DefaultExecutorConfig()
	executor := NewPlanExecutor(plan, config)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := executor.Execute(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// ==================== Validator Tests ====================

func TestValidator(t *testing.T) {
	validator := NewValidator()

	validator.RegisterCheck("file_write", func(ctx *TaskContext) error {
		// Validate file was written
		if ctx.Task.Result == nil {
			return errors.New("no result set")
		}
		return nil
	})

	task := NewTask("task-1", "Write File")
	task.Metadata["type"] = "file_write"
	task.Result = "file_path.txt"

	ctx := &TaskContext{Task: task}

	err := validator.Validate(ctx)
	if err != nil {
		t.Errorf("Validation should pass, got %v", err)
	}

	// Test failure
	task.Result = nil
	err = validator.Validate(ctx)
	if err == nil {
		t.Error("Validation should fail without result")
	}
}

// ==================== Learning Tests ====================

func TestLearningStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "learning.json")

	store, err := NewLearningStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	correction := &Correction{
		SessionID: "session-1",
		TaskID:    "task-1",
		TaskType:  "file_edit",
		Type:      CorrectionModify,
		Original:  "old content",
		Corrected: "new content",
		Context: map[string]interface{}{
			"operation": "edit",
			"target":    "test.go",
		},
	}

	store.RecordCorrection(correction)

	corrections := store.GetCorrections("session-1")
	if len(corrections) != 1 {
		t.Errorf("Expected 1 correction, got %d", len(corrections))
	}

	// Check pattern was created
	patterns := store.GetPatterns("file_edit")
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(patterns))
	}
}

func TestLearningStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "learning.json")

	// Create and populate store
	store1, _ := NewLearningStore(storePath)
	store1.RecordCorrection(&Correction{
		SessionID: "session-1",
		TaskID:    "task-1",
		TaskType:  "test",
		Type:      CorrectionUndo,
	})

	// Wait for async save
	time.Sleep(100 * time.Millisecond)

	// Load in new store
	store2, err := NewLearningStore(storePath)
	if err != nil {
		t.Fatalf("Failed to load store: %v", err)
	}

	corrections := store2.GetCorrections("session-1")
	if len(corrections) != 1 {
		t.Errorf("Expected 1 correction after reload, got %d", len(corrections))
	}
}

func TestSessionLearner(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "learning.json")
	store, _ := NewLearningStore(storePath)

	learner := NewSessionLearner("session-1", store)

	learner.RecordCorrection(
		"task-1",
		"file_edit",
		CorrectionReject,
		"original",
		"corrected",
		nil,
	)

	learner.RecordCorrection(
		"task-2",
		"file_edit",
		CorrectionReject,
		"original2",
		"corrected2",
		nil,
	)

	stats := learner.Stats()
	if stats.TotalCorrections != 2 {
		t.Errorf("Expected 2 corrections, got %d", stats.TotalCorrections)
	}

	if stats.ByType[CorrectionReject] != 2 {
		t.Errorf("Expected 2 rejections, got %d", stats.ByType[CorrectionReject])
	}

	// Check suggestions
	suggestions := learner.GetSuggestions("file_edit", nil)
	if len(suggestions) == 0 {
		t.Error("Expected suggestions after rejections")
	}
}

func TestPatternMatching(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "learning.json")
	store, _ := NewLearningStore(storePath)

	// Record corrections to build pattern
	for i := 0; i < 5; i++ {
		store.RecordCorrection(&Correction{
			SessionID: "session-1",
			TaskID:    "task-" + string(rune('0'+i)),
			TaskType:  "code_review",
			Type:      CorrectionModify,
			Context: map[string]interface{}{
				"language": "go",
				"target":   "main.go",
			},
		})
	}

	// Check that patterns were created
	allPatterns := store.GetPatterns("code_review")
	if len(allPatterns) == 0 {
		t.Error("Expected patterns to be created from corrections")
		return
	}

	// Check confidence is positive
	if allPatterns[0].Confidence <= 0 {
		t.Error("Expected positive confidence")
	}

	// Check corrections count
	if allPatterns[0].Corrections != 5 {
		t.Errorf("Expected 5 corrections in pattern, got %d", allPatterns[0].Corrections)
	}
}

// ==================== Integration Tests ====================

func TestFullPlanExecution(t *testing.T) {
	plan := NewPlan("plan-1", "Full Test")

	results := make(map[string]string)

	task1 := NewTask("task-1", "Setup")
	task1.Action = func(ctx *TaskContext) error {
		results["task-1"] = "done"
		ctx.Task.Result = "setup-complete"
		return nil
	}

	task2 := NewTask("task-2", "Process").WithDependencies("task-1")
	task2.Action = func(ctx *TaskContext) error {
		// Access previous result
		if ctx.Results["task-1"] != "setup-complete" {
			return errors.New("missing task-1 result")
		}
		results["task-2"] = "done"
		return nil
	}
	task2.Validate = func(ctx *TaskContext) error {
		return nil
	}

	task3 := NewTask("task-3", "Cleanup").WithDependencies("task-2")
	task3.Action = func(ctx *TaskContext) error {
		results["task-3"] = "done"
		return nil
	}

	plan.AddTask(task1).AddTask(task2).AddTask(task3)

	config := DefaultExecutorConfig()
	executor := NewPlanExecutor(plan, config)

	err := executor.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	if plan.Status != TaskCompleted {
		t.Errorf("Expected plan completed, got %s", plan.Status)
	}
}

func TestCleanup(t *testing.T) {
	// Clean up any test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Verify cleanup
	if _, err := os.Stat(testFile); err != nil {
		t.Error("Test file should exist before cleanup")
	}
}
