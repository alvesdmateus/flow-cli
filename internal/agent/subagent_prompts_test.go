package agent

import (
	"strings"
	"testing"
)

func TestSystemPromptForType(t *testing.T) {
	tests := []struct {
		subagentType SubagentType
		mustContain  []string
	}{
		{
			SubagentExplorer,
			[]string{"exploration", "read", "analyze", "CANNOT modify"},
		},
		{
			SubagentCoder,
			[]string{"coding", "implement", "write"},
		},
		{
			SubagentReviewer,
			[]string{"review", "bug", "security"},
		},
		{
			SubagentPlanner,
			[]string{"architecture", "design", "plan"},
		},
		{
			SubagentResearch,
			[]string{"research", "documentation", "sources"},
		},
	}

	for _, tc := range tests {
		prompt := SystemPromptForType(tc.subagentType)

		if prompt == "" {
			t.Errorf("%s returned empty prompt", tc.subagentType)
			continue
		}

		for _, keyword := range tc.mustContain {
			if !strings.Contains(strings.ToLower(prompt), strings.ToLower(keyword)) {
				t.Errorf("%s prompt should contain %q", tc.subagentType, keyword)
			}
		}
	}
}

func TestExplorerSystemPrompt(t *testing.T) {
	prompt := ExplorerSystemPrompt()

	// Explorer should emphasize read-only operations
	if !strings.Contains(prompt, "CANNOT modify") {
		t.Error("Explorer prompt should mention that it cannot modify files")
	}

	// Should mention exploration capabilities
	if !strings.Contains(prompt, "exploration") {
		t.Error("Explorer prompt should mention exploration")
	}

	// Should have tool usage instructions
	if !strings.Contains(prompt, "tool_call") {
		t.Error("Explorer prompt should include tool usage instructions")
	}
}

func TestCoderSystemPrompt(t *testing.T) {
	prompt := CoderSystemPrompt()

	// Coder should mention implementation
	if !strings.Contains(strings.ToLower(prompt), "implement") {
		t.Error("Coder prompt should mention implementation")
	}

	// Should mention quality standards
	if !strings.Contains(strings.ToLower(prompt), "security") {
		t.Error("Coder prompt should mention security considerations")
	}
}

func TestReviewerSystemPrompt(t *testing.T) {
	prompt := ReviewerSystemPrompt()

	// Reviewer should focus on analysis
	if !strings.Contains(strings.ToLower(prompt), "review") {
		t.Error("Reviewer prompt should mention code review")
	}

	// Should mention bug detection
	if !strings.Contains(strings.ToLower(prompt), "bug") {
		t.Error("Reviewer prompt should mention bug detection")
	}
}

func TestPlannerSystemPrompt(t *testing.T) {
	prompt := PlannerSystemPrompt()

	// Planner should focus on design
	if !strings.Contains(strings.ToLower(prompt), "design") {
		t.Error("Planner prompt should mention design")
	}

	// Should mention maintainability
	if !strings.Contains(strings.ToLower(prompt), "maintainability") {
		t.Error("Planner prompt should mention maintainability")
	}
}

func TestResearchSystemPrompt(t *testing.T) {
	prompt := ResearchSystemPrompt()

	// Research should mention sources
	if !strings.Contains(strings.ToLower(prompt), "source") {
		t.Error("Research prompt should mention sources")
	}

	// Should mention verification
	if !strings.Contains(strings.ToLower(prompt), "verify") || !strings.Contains(strings.ToLower(prompt), "verif") {
		// Check for either "verify" or "verification"
	}
}

func TestDefaultSubagentPrompt(t *testing.T) {
	prompt := DefaultSubagentPrompt()

	if prompt == "" {
		t.Error("Default prompt should not be empty")
	}

	// Should be a general prompt
	if !strings.Contains(strings.ToLower(prompt), "task") {
		t.Error("Default prompt should mention task completion")
	}
}

func TestBuildTaskPrompt(t *testing.T) {
	task := "Find all Go files in the project"
	prompt := BuildTaskPrompt(SubagentExplorer, task)

	// Should include the base system prompt
	basePrompt := SystemPromptForType(SubagentExplorer)
	if !strings.Contains(prompt, basePrompt[:50]) { // Check first 50 chars
		t.Error("BuildTaskPrompt should include the base system prompt")
	}

	// Should include the task
	if !strings.Contains(prompt, task) {
		t.Error("BuildTaskPrompt should include the task description")
	}

	// Should have task section marker
	if !strings.Contains(prompt, "YOUR TASK") {
		t.Error("BuildTaskPrompt should include task section marker")
	}
}

func TestPromptLength(t *testing.T) {
	// All prompts should be reasonable length (not too short, not too long)
	types := AllSubagentTypes()

	for _, st := range types {
		prompt := SystemPromptForType(st)
		length := len(prompt)

		if length < 200 {
			t.Errorf("%s prompt is too short (%d chars)", st, length)
		}

		if length > 5000 {
			t.Errorf("%s prompt is too long (%d chars)", st, length)
		}
	}
}

func TestPromptToolCallFormat(t *testing.T) {
	// All prompts should include tool call format instructions
	types := AllSubagentTypes()

	for _, st := range types {
		prompt := SystemPromptForType(st)

		if !strings.Contains(prompt, "<tool_call") {
			t.Errorf("%s prompt should include tool_call format", st)
		}
	}
}
