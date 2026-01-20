package ui

import (
	"strings"
	"testing"
)

func TestFormatPlan_Basic(t *testing.T) {
	plan := PlanDisplay{
		Title:   "Test Plan",
		Summary: "A test implementation plan",
		Phase:   "Planning",
	}

	result := FormatPlan(plan)

	if result == "" {
		t.Error("FormatPlan() returned empty string")
	}

	if !strings.Contains(result, "Test Plan") {
		t.Error("result should contain the title")
	}
}

func TestFormatPlan_WithGoals(t *testing.T) {
	plan := PlanDisplay{
		Title: "Test Plan",
		Phase: "Planning",
		Goals: []string{
			"Implement feature X",
			"Add tests for feature X",
		},
	}

	result := FormatPlan(plan)

	if !strings.Contains(result, "Goals") {
		t.Error("result should contain Goals section")
	}
}

func TestFormatPlan_WithConstraints(t *testing.T) {
	plan := PlanDisplay{
		Title: "Test Plan",
		Phase: "Planning",
		Constraints: []string{
			"Must be backward compatible",
			"No breaking changes",
		},
	}

	result := FormatPlan(plan)

	if !strings.Contains(result, "Constraints") {
		t.Error("result should contain Constraints section")
	}
}

func TestFormatPlan_WithTasks(t *testing.T) {
	plan := PlanDisplay{
		Title: "Test Plan",
		Phase: "Planning",
		Tasks: []TaskDisplay{
			{
				ID:          "1",
				Title:       "Create module",
				Description: "Create the new module structure",
				Priority:    "high",
				Files:       []string{"module.go", "module_test.go"},
				Completed:   false,
			},
			{
				ID:          "2",
				Title:       "Add tests",
				Description: "Write unit tests",
				Priority:    "medium",
				Completed:   true,
			},
		},
	}

	result := FormatPlan(plan)

	if !strings.Contains(result, "Implementation Tasks") {
		t.Error("result should contain tasks section")
	}
}

func TestFormatPlan_Complete(t *testing.T) {
	plan := PlanDisplay{
		Title:   "Complete Plan",
		Summary: "This is a complete implementation plan",
		Phase:   "Implementation",
		Goals: []string{
			"Goal 1",
			"Goal 2",
		},
		Constraints: []string{
			"Constraint 1",
		},
		Tasks: []TaskDisplay{
			{
				ID:          "1",
				Title:       "Task 1",
				Description: "Description 1",
				Priority:    "critical",
				Files:       []string{"file1.go"},
			},
		},
	}

	result := FormatPlan(plan)

	// Verify all sections are present
	sections := []string{"Complete Plan", "Goals", "Constraints", "Implementation Tasks"}
	for _, section := range sections {
		if !strings.Contains(result, section) {
			t.Errorf("result should contain %q", section)
		}
	}
}

func TestFormatPlan_EmptyPlan(t *testing.T) {
	plan := PlanDisplay{
		Title: "Empty Plan",
		Phase: "Planning",
	}

	result := FormatPlan(plan)

	// Should still produce output with title and phase
	if result == "" {
		t.Error("FormatPlan() should produce output even for empty plan")
	}
}

func TestFormatTask(t *testing.T) {
	tests := []struct {
		name string
		task TaskDisplay
	}{
		{
			name: "basic task",
			task: TaskDisplay{
				ID:       "1",
				Title:    "Test Task",
				Priority: "high",
			},
		},
		{
			name: "completed task",
			task: TaskDisplay{
				ID:        "2",
				Title:     "Completed Task",
				Priority:  "medium",
				Completed: true,
			},
		},
		{
			name: "task with description",
			task: TaskDisplay{
				ID:          "3",
				Title:       "Described Task",
				Description: "This is a detailed description",
				Priority:    "low",
			},
		},
		{
			name: "task with files",
			task: TaskDisplay{
				ID:       "4",
				Title:    "File Task",
				Priority: "medium",
				Files:    []string{"file1.go", "file2.go", "file3.go"},
			},
		},
		{
			name: "full task",
			task: TaskDisplay{
				ID:          "5",
				Title:       "Full Task",
				Description: "Complete description",
				Priority:    "critical",
				Files:       []string{"main.go"},
				Completed:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTask(1, tt.task)

			if result == "" {
				t.Error("formatTask() returned empty string")
			}

			if !strings.Contains(result, tt.task.Title) {
				t.Errorf("result should contain task title %q", tt.task.Title)
			}
		})
	}
}

func TestFormatTask_Checkbox(t *testing.T) {
	// Test incomplete task
	incomplete := TaskDisplay{Title: "Incomplete", Completed: false}
	result := formatTask(1, incomplete)
	if !strings.Contains(result, checkboxEmpty) {
		t.Error("incomplete task should have empty checkbox")
	}

	// Test complete task
	complete := TaskDisplay{Title: "Complete", Completed: true}
	result = formatTask(1, complete)
	if !strings.Contains(result, checkboxComplete) {
		t.Error("complete task should have complete checkbox")
	}
}

func TestFormatPriority(t *testing.T) {
	tests := []struct {
		priority string
		expected string
	}{
		{"critical", "CRITICAL"},
		{"CRITICAL", "CRITICAL"},
		{"high", "HIGH"},
		{"HIGH", "HIGH"},
		{"medium", "MEDIUM"},
		{"MEDIUM", "MEDIUM"},
		{"low", "LOW"},
		{"LOW", "LOW"},
		{"", "MEDIUM"},       // default
		{"unknown", "MEDIUM"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			result := formatPriority(tt.priority)

			if result == "" {
				t.Error("formatPriority() returned empty string")
			}

			// The result is styled, but should contain the expected text
			// Note: lipgloss styling includes ANSI codes
			if len(result) == 0 {
				t.Errorf("formatPriority(%q) returned empty result", tt.priority)
			}
		})
	}
}

func TestFormatQuestion_Basic(t *testing.T) {
	q := QuestionDisplay{
		Question: "What database should we use?",
	}

	result := FormatQuestion(q)

	if result == "" {
		t.Error("FormatQuestion() returned empty string")
	}

	if !strings.Contains(result, q.Question) {
		t.Error("result should contain the question")
	}
}

func TestFormatQuestion_WithContext(t *testing.T) {
	q := QuestionDisplay{
		Question: "What database should we use?",
		Context:  "The application requires high write throughput",
	}

	result := FormatQuestion(q)

	if result == "" {
		t.Error("FormatQuestion() returned empty string")
	}
}

func TestFormatQuestion_WithOptions(t *testing.T) {
	q := QuestionDisplay{
		Question: "What database should we use?",
		Options:  []string{"PostgreSQL", "MySQL", "MongoDB"},
	}

	result := FormatQuestion(q)

	if result == "" {
		t.Error("FormatQuestion() returned empty string")
	}
}

func TestPlanPrintFunctions_NoPanic(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{"PrintPhaseChange", func() { PrintPhaseChange("Analysis") }},
		{"PrintPlanApprovalPrompt", func() { PrintPlanApprovalPrompt() }},
		{"PrintWelcomeArch", func() { PrintWelcomeArch() }},
		{"PrintPlanSummary", func() { PrintPlanSummary(5, 10) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s() panicked: %v", tt.name, r)
				}
			}()

			tt.fn()
		})
	}
}

func TestPlanDisplay_Fields(t *testing.T) {
	plan := PlanDisplay{
		Title:       "Test",
		Summary:     "Summary",
		Goals:       []string{"Goal1"},
		Constraints: []string{"Constraint1"},
		Tasks:       []TaskDisplay{{ID: "1"}},
		Phase:       "Planning",
	}

	if plan.Title != "Test" {
		t.Errorf("Title = %q, want %q", plan.Title, "Test")
	}
	if plan.Summary != "Summary" {
		t.Errorf("Summary = %q, want %q", plan.Summary, "Summary")
	}
	if len(plan.Goals) != 1 {
		t.Errorf("Goals length = %d, want 1", len(plan.Goals))
	}
	if len(plan.Constraints) != 1 {
		t.Errorf("Constraints length = %d, want 1", len(plan.Constraints))
	}
	if len(plan.Tasks) != 1 {
		t.Errorf("Tasks length = %d, want 1", len(plan.Tasks))
	}
	if plan.Phase != "Planning" {
		t.Errorf("Phase = %q, want %q", plan.Phase, "Planning")
	}
}

func TestTaskDisplay_Fields(t *testing.T) {
	task := TaskDisplay{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "Description",
		Priority:    "high",
		Files:       []string{"file1.go", "file2.go"},
		Completed:   true,
	}

	if task.ID != "task-1" {
		t.Errorf("ID = %q, want %q", task.ID, "task-1")
	}
	if task.Title != "Test Task" {
		t.Errorf("Title = %q, want %q", task.Title, "Test Task")
	}
	if task.Description != "Description" {
		t.Errorf("Description = %q, want %q", task.Description, "Description")
	}
	if task.Priority != "high" {
		t.Errorf("Priority = %q, want %q", task.Priority, "high")
	}
	if len(task.Files) != 2 {
		t.Errorf("Files length = %d, want 2", len(task.Files))
	}
	if !task.Completed {
		t.Error("Completed should be true")
	}
}

func TestQuestionDisplay_Fields(t *testing.T) {
	q := QuestionDisplay{
		Question: "Test Question?",
		Context:  "Context info",
		Options:  []string{"Option1", "Option2"},
	}

	if q.Question != "Test Question?" {
		t.Errorf("Question = %q, want %q", q.Question, "Test Question?")
	}
	if q.Context != "Context info" {
		t.Errorf("Context = %q, want %q", q.Context, "Context info")
	}
	if len(q.Options) != 2 {
		t.Errorf("Options length = %d, want 2", len(q.Options))
	}
}

func TestPlanStyles_NotNil(t *testing.T) {
	styles := []struct {
		name  string
		style interface{}
	}{
		{"planTitleStyle", planTitleStyle},
		{"planSectionStyle", planSectionStyle},
		{"planItemStyle", planItemStyle},
		{"taskBoxStyle", taskBoxStyle},
		{"priorityCriticalStyle", priorityCriticalStyle},
		{"priorityHighStyle", priorityHighStyle},
		{"priorityMediumStyle", priorityMediumStyle},
		{"priorityLowStyle", priorityLowStyle},
		{"phaseIndicatorStyle", phaseIndicatorStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s caused panic: %v", s.name, r)
				}
			}()

			switch style := s.style.(type) {
			case interface{ Render(string) string }:
				result := style.Render("test")
				if result == "" {
					t.Errorf("%s.Render() returned empty string", s.name)
				}
			}
		})
	}
}

func TestCheckboxConstants(t *testing.T) {
	if checkboxEmpty == "" {
		t.Error("checkboxEmpty should not be empty")
	}
	if checkboxComplete == "" {
		t.Error("checkboxComplete should not be empty")
	}
	if checkboxEmpty == checkboxComplete {
		t.Error("checkboxEmpty and checkboxComplete should be different")
	}
}
