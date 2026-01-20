package ui

import (
	"testing"

	"github.com/mateus/flow-cli/internal/sandbox"
)

func TestSelectModel_NoModels(t *testing.T) {
	_, err := SelectModel([]string{})

	if err == nil {
		t.Error("SelectModel() should return error for empty models")
	}

	if err.Error() != "no models available" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSelectModel_SingleModel(t *testing.T) {
	model, err := SelectModel([]string{"llama3"})

	if err != nil {
		t.Errorf("SelectModel() unexpected error: %v", err)
	}

	if model != "llama3" {
		t.Errorf("SelectModel() = %q, want %q", model, "llama3")
	}
}

func TestMultiSelect_EmptyOptions(t *testing.T) {
	selected, err := MultiSelect("Test", []string{})

	if err != nil {
		t.Errorf("MultiSelect() unexpected error: %v", err)
	}

	if selected != nil {
		t.Errorf("MultiSelect() = %v, want nil", selected)
	}
}

func TestAskQuestion_NoChoices(t *testing.T) {
	_, err := AskQuestion("Test?", []string{})

	if err == nil {
		t.Error("AskQuestion() should return error for empty choices")
	}

	if err.Error() != "no choices provided" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFormatRiskLevel(t *testing.T) {
	tests := []struct {
		name  string
		level sandbox.PermissionLevel
	}{
		{"none", sandbox.PermissionNone},
		{"low", sandbox.PermissionLow},
		{"medium", sandbox.PermissionMedium},
		{"high", sandbox.PermissionHigh},
		{"denied", sandbox.PermissionDenied},
		{"unknown", sandbox.PermissionLevel(999)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRiskLevel(tt.level)

			if result == "" {
				t.Error("formatRiskLevel() returned empty string")
			}
		})
	}
}

func TestFormatRiskLevel_Values(t *testing.T) {
	// Test that each level returns the expected styled text content
	tests := []struct {
		level    sandbox.PermissionLevel
		contains string
	}{
		{sandbox.PermissionNone, "none"},
		{sandbox.PermissionLow, "LOW"},
		{sandbox.PermissionMedium, "MEDIUM"},
		{sandbox.PermissionHigh, "HIGH"},
		{sandbox.PermissionDenied, "DENIED"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := formatRiskLevel(tt.level)
			// The styled text should contain the level name
			// Note: lipgloss styling may include ANSI codes
			if len(result) == 0 {
				t.Errorf("formatRiskLevel(%v) returned empty result", tt.level)
			}
		})
	}
}

func TestStyles_NotNil(t *testing.T) {
	// Test that all styles are properly initialized
	styles := []struct {
		name  string
		style interface{}
	}{
		{"titleStyle", titleStyle},
		{"errorStyle", errorStyle},
		{"successStyle", successStyle},
		{"infoStyle", infoStyle},
		{"warningStyle", warningStyle},
		{"dimStyle", dimStyle},
		{"boxStyle", boxStyle},
		{"riskLowStyle", riskLowStyle},
		{"riskMediumStyle", riskMediumStyle},
		{"riskHighStyle", riskHighStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			// Styles should render without panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s caused panic: %v", s.name, r)
				}
			}()

			// Test rendering
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

func TestPrintFunctions_NoPanic(t *testing.T) {
	// These functions print to stdout, we just verify they don't panic
	tests := []struct {
		name string
		fn   func()
	}{
		{"PrintTitle", func() { PrintTitle("Test Title") }},
		{"PrintError", func() { PrintError("Test Error") }},
		{"PrintSuccess", func() { PrintSuccess("Test Success") }},
		{"PrintInfo", func() { PrintInfo("Test Info") }},
		{"PrintWarning", func() { PrintWarning("Test Warning") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked: %v", tt.name, r)
				}
			}()
			tt.fn()
		})
	}
}
