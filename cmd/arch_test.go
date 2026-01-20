package cmd

import (
	"testing"

	"github.com/mateus/flow-cli/internal/agent"
)

func TestArchCmd_Structure(t *testing.T) {
	if archCmd == nil {
		t.Fatal("archCmd is nil")
	}

	if archCmd.Use != "arch [description]" {
		t.Errorf("archCmd.Use = %q, want %q", archCmd.Use, "arch [description]")
	}

	if archCmd.Short == "" {
		t.Error("archCmd.Short should not be empty")
	}

	if archCmd.Long == "" {
		t.Error("archCmd.Long should not be empty")
	}

	if archCmd.RunE == nil {
		t.Error("archCmd.RunE should not be nil")
	}
}

func TestArchCmd_ParentIsRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "arch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("archCmd should be a subcommand of rootCmd")
	}
}

func TestArchHandler_Interface(t *testing.T) {
	handler := &ArchHandler{}

	// Test that handler methods don't panic

	// OnPhaseChange
	handler.OnPhaseChange(agent.PhaseGathering)
	handler.OnPhaseChange(agent.PhaseAnalyzing)
	handler.OnPhaseChange(agent.PhaseDesigning)

	// OnPlanUpdate with nil plan
	handler.OnPlanUpdate(nil)

	// OnMessage
	handler.OnMessage("test message")

	// OnError
	handler.OnError(nil)
}

func TestArchHandler_OnPhaseChange(t *testing.T) {
	handler := &ArchHandler{}

	phases := []agent.PlanPhase{
		agent.PhaseGathering,
		agent.PhaseAnalyzing,
		agent.PhaseDesigning,
		agent.PhaseComplete,
	}

	for _, phase := range phases {
		// Each phase change should stop previous spinner and start new one
		handler.OnPhaseChange(phase)
	}

	// Clean up - stop any running spinner
	if handler.spinner != nil {
		handler.spinner.Stop()
	}
}

func TestArchHandler_OnTaskStart(t *testing.T) {
	handler := &ArchHandler{}

	task := agent.PlanTask{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "Test description",
	}

	handler.OnTaskStart(task)

	// Clean up
	if handler.spinner != nil {
		handler.spinner.Stop()
	}
}

func TestArchHandler_OnTaskComplete(t *testing.T) {
	handler := &ArchHandler{}

	task := agent.PlanTask{
		ID:    "task-1",
		Title: "Test Task",
	}

	// Test success
	handler.OnTaskComplete(task, true)

	// Test failure
	handler.OnTaskComplete(task, false)
}

func TestArchHandler_OnMessage_WithSpinner(t *testing.T) {
	handler := &ArchHandler{}

	// Start a phase to create spinner
	handler.OnPhaseChange(agent.PhaseGathering)

	// Message should update spinner
	handler.OnMessage("Processing...")

	// Clean up
	if handler.spinner != nil {
		handler.spinner.Stop()
	}
}

func TestArchHandler_OnMessage_WithoutSpinner(t *testing.T) {
	handler := &ArchHandler{}

	// Message without spinner should just print
	handler.OnMessage("No spinner message")
}

func TestArchHandler_OnError_WithSpinner(t *testing.T) {
	handler := &ArchHandler{}

	// Start a phase to create spinner
	handler.OnPhaseChange(agent.PhaseAnalyzing)

	// Error should stop spinner
	handler.OnError(nil)
}

func TestArchHandler_OnError_WithoutSpinner(t *testing.T) {
	handler := &ArchHandler{}

	// Error without spinner should just print
	handler.OnError(nil)
}

func TestPlanApprovalResult_Constants(t *testing.T) {
	// Verify constants have expected values
	if planApproved != 0 {
		t.Errorf("planApproved = %d, want 0", planApproved)
	}
	if planNeedsModification != 1 {
		t.Errorf("planNeedsModification = %d, want 1", planNeedsModification)
	}
	if planCancelled != 2 {
		t.Errorf("planCancelled = %d, want 2", planCancelled)
	}
}

func TestFormatTaskList(t *testing.T) {
	tasks := []agent.PlanTask{
		{
			ID:          "1",
			Title:       "First Task",
			Description: "Do the first thing",
			Completed:   false,
		},
		{
			ID:          "2",
			Title:       "Second Task",
			Description: "Do the second thing",
			Completed:   true,
		},
		{
			ID:        "3",
			Title:     "Third Task",
			Completed: false,
		},
	}

	result := formatTaskList(tasks)

	// Should contain task numbers
	if !containsSubstring(result, "1.") {
		t.Error("formatTaskList should contain task number 1")
	}
	if !containsSubstring(result, "2.") {
		t.Error("formatTaskList should contain task number 2")
	}

	// Should contain task titles
	if !containsSubstring(result, "First Task") {
		t.Error("formatTaskList should contain 'First Task'")
	}
	if !containsSubstring(result, "Second Task") {
		t.Error("formatTaskList should contain 'Second Task'")
	}

	// Completed task should have [x]
	if !containsSubstring(result, "[x]") {
		t.Error("formatTaskList should show [x] for completed tasks")
	}

	// Incomplete tasks should have [ ]
	if !containsSubstring(result, "[ ]") {
		t.Error("formatTaskList should show [ ] for incomplete tasks")
	}
}

func TestFormatTaskList_Empty(t *testing.T) {
	result := formatTaskList([]agent.PlanTask{})
	if result != "" {
		t.Errorf("formatTaskList([]) = %q, want empty string", result)
	}
}

func TestDisplayPlan(t *testing.T) {
	plan := &agent.Plan{
		Title:       "Test Plan",
		Summary:     "This is a test plan",
		Goals:       []string{"Goal 1", "Goal 2"},
		Constraints: []string{"Constraint 1"},
		Phase:       agent.PhaseDesigning,
		Tasks: []agent.PlanTask{
			{
				ID:          "1",
				Title:       "Task 1",
				Description: "Do task 1",
				Priority:    "high",
				Files:       []string{"file1.go"},
			},
		},
	}

	// Should complete without panic
	displayPlan(plan)
}

// Helper function
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
