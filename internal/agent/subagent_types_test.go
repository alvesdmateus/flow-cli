package agent

import (
	"testing"
)

func TestAllSubagentTypes(t *testing.T) {
	types := AllSubagentTypes()

	expected := []SubagentType{
		SubagentExplorer,
		SubagentCoder,
		SubagentReviewer,
		SubagentPlanner,
		SubagentResearch,
	}

	if len(types) != len(expected) {
		t.Errorf("expected %d types, got %d", len(expected), len(types))
	}

	for _, exp := range expected {
		found := false
		for _, got := range types {
			if got == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected type %s not found in AllSubagentTypes()", exp)
		}
	}
}

func TestIsValidSubagentType(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"explorer", true},
		{"coder", true},
		{"reviewer", true},
		{"planner", true},
		{"research", true},
		{"invalid", false},
		{"", false},
		{"Explorer", false}, // case-sensitive
	}

	for _, tc := range tests {
		result := IsValidSubagentType(tc.input)
		if result != tc.expected {
			t.Errorf("IsValidSubagentType(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestDefaultToolsForType(t *testing.T) {
	tests := []struct {
		subagentType SubagentType
		mustInclude  []string
		mustExclude  []string
	}{
		{
			SubagentExplorer,
			[]string{"read_file", "list_files", "grep_search"},
			[]string{"write_file", "edit_file", "run_command"},
		},
		{
			SubagentCoder,
			[]string{"read_file", "write_file", "edit_file", "run_command"},
			[]string{"web_search"},
		},
		{
			SubagentReviewer,
			[]string{"read_file", "list_files", "git_diff"},
			[]string{"write_file", "run_command"},
		},
		{
			SubagentPlanner,
			[]string{"read_file", "web_search"},
			[]string{"write_file", "run_command"},
		},
		{
			SubagentResearch,
			[]string{"web_search", "fetch_url"},
			[]string{"write_file", "run_command"},
		},
	}

	for _, tc := range tests {
		tools := DefaultToolsForType(tc.subagentType)
		toolSet := make(map[string]bool)
		for _, tool := range tools {
			toolSet[tool] = true
		}

		for _, must := range tc.mustInclude {
			if !toolSet[must] {
				t.Errorf("%s should include tool %s", tc.subagentType, must)
			}
		}

		for _, exclude := range tc.mustExclude {
			if toolSet[exclude] {
				t.Errorf("%s should NOT include tool %s", tc.subagentType, exclude)
			}
		}
	}
}

func TestDefaultMaxTurnsForType(t *testing.T) {
	tests := []struct {
		subagentType SubagentType
		minTurns     int
	}{
		{SubagentExplorer, 5},
		{SubagentCoder, 10},
		{SubagentReviewer, 5},
		{SubagentPlanner, 5},
		{SubagentResearch, 5},
	}

	for _, tc := range tests {
		turns := DefaultMaxTurnsForType(tc.subagentType)
		if turns < tc.minTurns {
			t.Errorf("%s should have at least %d turns, got %d", tc.subagentType, tc.minTurns, turns)
		}
	}
}

func TestDefaultTokenBudgetForType(t *testing.T) {
	tests := []struct {
		subagentType SubagentType
		minBudget    int
	}{
		{SubagentExplorer, 10000},
		{SubagentCoder, 20000},
		{SubagentReviewer, 10000},
		{SubagentPlanner, 15000},
		{SubagentResearch, 5000},
	}

	for _, tc := range tests {
		budget := DefaultTokenBudgetForType(tc.subagentType)
		if budget < tc.minBudget {
			t.Errorf("%s should have at least %d token budget, got %d", tc.subagentType, tc.minBudget, budget)
		}
	}
}

func TestSubagentConfig_Defaults(t *testing.T) {
	// Test that empty config fields are handled properly
	config := SubagentConfig{
		Type: SubagentExplorer,
		Task: "Test task",
	}

	if config.TokenBudget != 0 {
		t.Error("default TokenBudget should be 0 (to be filled by NewSubagent)")
	}

	if config.MaxTurns != 0 {
		t.Error("default MaxTurns should be 0 (to be filled by NewSubagent)")
	}

	if len(config.AllowedTools) != 0 {
		t.Error("default AllowedTools should be empty (to be filled by NewSubagent)")
	}
}

func TestSubagentResult_Fields(t *testing.T) {
	result := SubagentResult{
		Success:       true,
		Summary:       "Task completed",
		Details:       "Full details here...",
		FilesModified: []string{"file1.go", "file2.go"},
		TokensUsed:    5000,
		Error:         nil,
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.Summary != "Task completed" {
		t.Errorf("unexpected Summary: %s", result.Summary)
	}

	if len(result.FilesModified) != 2 {
		t.Errorf("expected 2 modified files, got %d", len(result.FilesModified))
	}

	if result.TokensUsed != 5000 {
		t.Errorf("expected TokensUsed 5000, got %d", result.TokensUsed)
	}
}
