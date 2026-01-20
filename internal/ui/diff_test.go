package ui

import (
	"testing"
)

func TestDiffLineType_Constants(t *testing.T) {
	// Verify constants have distinct values
	types := map[DiffLineType]string{
		DiffContext: "DiffContext",
		DiffAdded:   "DiffAdded",
		DiffRemoved: "DiffRemoved",
	}

	seen := make(map[DiffLineType]bool)
	for diffType := range types {
		if seen[diffType] {
			t.Errorf("duplicate DiffLineType value: %v", diffType)
		}
		seen[diffType] = true
	}
}

func TestDiffLine_Fields(t *testing.T) {
	line := DiffLine{
		Type:    DiffAdded,
		Content: "new line",
		OldNum:  0,
		NewNum:  5,
	}

	if line.Type != DiffAdded {
		t.Errorf("Type = %v, want DiffAdded", line.Type)
	}

	if line.Content != "new line" {
		t.Errorf("Content = %q, want %q", line.Content, "new line")
	}

	if line.NewNum != 5 {
		t.Errorf("NewNum = %d, want 5", line.NewNum)
	}
}

func TestGenerateDiff_IdenticalContent(t *testing.T) {
	content := "line1\nline2\nline3"

	diff := generateDiff(content, content)

	// All lines should be context (unchanged)
	for i, line := range diff {
		if line.Type != DiffContext {
			t.Errorf("line[%d].Type = %v, want DiffContext", i, line.Type)
		}
	}
}

func TestGenerateDiff_AddedLines(t *testing.T) {
	oldContent := "line1\nline2"
	newContent := "line1\nline2\nline3"

	diff := generateDiff(oldContent, newContent)

	// Last line should be added
	hasAdded := false
	for _, line := range diff {
		if line.Type == DiffAdded && line.Content == "line3" {
			hasAdded = true
			break
		}
	}

	if !hasAdded {
		t.Error("expected to find added line 'line3'")
	}
}

func TestGenerateDiff_RemovedLines(t *testing.T) {
	oldContent := "line1\nline2\nline3"
	newContent := "line1\nline2"

	diff := generateDiff(oldContent, newContent)

	// Should have a removed line
	hasRemoved := false
	for _, line := range diff {
		if line.Type == DiffRemoved && line.Content == "line3" {
			hasRemoved = true
			break
		}
	}

	if !hasRemoved {
		t.Error("expected to find removed line 'line3'")
	}
}

func TestGenerateDiff_ModifiedLines(t *testing.T) {
	oldContent := "line1\nold line\nline3"
	newContent := "line1\nnew line\nline3"

	diff := generateDiff(oldContent, newContent)

	// Should have both removed and added for the modified line
	hasRemoved := false
	hasAdded := false

	for _, line := range diff {
		if line.Type == DiffRemoved && line.Content == "old line" {
			hasRemoved = true
		}
		if line.Type == DiffAdded && line.Content == "new line" {
			hasAdded = true
		}
	}

	if !hasRemoved {
		t.Error("expected to find removed 'old line'")
	}
	if !hasAdded {
		t.Error("expected to find added 'new line'")
	}
}

func TestGenerateDiff_EmptyOld(t *testing.T) {
	oldContent := ""
	newContent := "line1\nline2"

	diff := generateDiff(oldContent, newContent)

	// Should have some added lines
	addedCount := 0
	for _, line := range diff {
		if line.Type == DiffAdded {
			addedCount++
		}
	}

	if addedCount == 0 {
		t.Error("expected at least one added line")
	}
}

func TestGenerateDiff_EmptyNew(t *testing.T) {
	oldContent := "line1\nline2"
	newContent := ""

	diff := generateDiff(oldContent, newContent)

	// Should have some removed lines
	removedCount := 0
	for _, line := range diff {
		if line.Type == DiffRemoved {
			removedCount++
		}
	}

	if removedCount == 0 {
		t.Error("expected at least one removed line")
	}
}

func TestGenerateDiff_ComplexChanges(t *testing.T) {
	oldContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello")
}`

	newContent := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0)
}`

	diff := generateDiff(oldContent, newContent)

	if len(diff) == 0 {
		t.Error("diff should not be empty for different content")
	}

	// Count changes
	added := 0
	removed := 0
	context := 0

	for _, line := range diff {
		switch line.Type {
		case DiffAdded:
			added++
		case DiffRemoved:
			removed++
		case DiffContext:
			context++
		}
	}

	if added == 0 {
		t.Error("expected some added lines")
	}
	if removed == 0 {
		t.Error("expected some removed lines")
	}
}

func TestFindLineAhead_Found(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}

	result := findLineAhead("c", lines, 0, 5)

	if result != 2 {
		t.Errorf("findLineAhead() = %d, want 2", result)
	}
}

func TestFindLineAhead_NotFound(t *testing.T) {
	lines := []string{"a", "b", "c"}

	result := findLineAhead("z", lines, 0, 5)

	if result != -1 {
		t.Errorf("findLineAhead() = %d, want -1", result)
	}
}

func TestFindLineAhead_BeyondLookahead(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e", "f"}

	// Looking for "f" but lookahead is only 3
	result := findLineAhead("f", lines, 0, 3)

	if result != -1 {
		t.Errorf("findLineAhead() = %d, want -1", result)
	}
}

func TestFindLineAhead_FromMiddle(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}

	result := findLineAhead("d", lines, 2, 5)

	if result != 3 {
		t.Errorf("findLineAhead() = %d, want 3", result)
	}
}

func TestFindLineAhead_AtBoundary(t *testing.T) {
	lines := []string{"a", "b", "c"}

	// Start at end
	result := findLineAhead("c", lines, 2, 5)

	if result != 2 {
		t.Errorf("findLineAhead() = %d, want 2", result)
	}
}

func TestShowDiff_NoPanic(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		oldContent string
		newContent string
	}{
		{"new file", "test.go", "", "package main"},
		{"modified file", "test.go", "old", "new"},
		{"no changes", "test.go", "same", "same"},
		{"empty both", "test.go", "", ""},
		{"multiline new", "test.go", "", "line1\nline2\nline3"},
		{"large new file", "test.go", "", generateLargeContent(50)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ShowDiff() panicked: %v", r)
				}
			}()

			err := ShowDiff(tt.path, tt.oldContent, tt.newContent)
			if err != nil {
				t.Errorf("ShowDiff() error: %v", err)
			}
		})
	}
}

func TestShowFileDiff(t *testing.T) {
	// ShowFileDiff is a wrapper for ShowDiff
	err := ShowFileDiff("test.go", "old", "new")
	if err != nil {
		t.Errorf("ShowFileDiff() error: %v", err)
	}
}

func TestDisplayDiff_NoPanic(t *testing.T) {
	tests := []struct {
		name  string
		lines []DiffLine
	}{
		{"empty", []DiffLine{}},
		{"all context", []DiffLine{
			{Type: DiffContext, Content: "line1", OldNum: 1, NewNum: 1},
			{Type: DiffContext, Content: "line2", OldNum: 2, NewNum: 2},
		}},
		{"mixed", []DiffLine{
			{Type: DiffContext, Content: "context", OldNum: 1, NewNum: 1},
			{Type: DiffRemoved, Content: "removed", OldNum: 2},
			{Type: DiffAdded, Content: "added", NewNum: 2},
			{Type: DiffContext, Content: "context2", OldNum: 3, NewNum: 3},
		}},
		{"all added", []DiffLine{
			{Type: DiffAdded, Content: "line1", NewNum: 1},
			{Type: DiffAdded, Content: "line2", NewNum: 2},
		}},
		{"all removed", []DiffLine{
			{Type: DiffRemoved, Content: "line1", OldNum: 1},
			{Type: DiffRemoved, Content: "line2", OldNum: 2},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("displayDiff() panicked: %v", r)
				}
			}()

			displayDiff(tt.lines)
		})
	}
}

func TestPrintDiffLine_NoPanic(t *testing.T) {
	tests := []struct {
		name string
		line DiffLine
	}{
		{"context", DiffLine{Type: DiffContext, Content: "context line", OldNum: 1, NewNum: 1}},
		{"added", DiffLine{Type: DiffAdded, Content: "added line", NewNum: 2}},
		{"removed", DiffLine{Type: DiffRemoved, Content: "removed line", OldNum: 3}},
		{"empty content", DiffLine{Type: DiffContext, Content: "", NewNum: 1}},
		{"long content", DiffLine{Type: DiffAdded, Content: "this is a very long line that contains many characters to test the display", NewNum: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printDiffLine() panicked: %v", r)
				}
			}()

			printDiffLine(tt.line)
		})
	}
}

func TestDiffStyles_NotNil(t *testing.T) {
	styles := []struct {
		name  string
		style interface{}
	}{
		{"addedLineStyle", addedLineStyle},
		{"removedLineStyle", removedLineStyle},
		{"contextLineStyle", contextLineStyle},
		{"lineNumberStyle", lineNumberStyle},
		{"newFileStyle", newFileStyle},
		{"modifiedFileStyle", modifiedFileStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
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

// Helper to generate large content
func generateLargeContent(lines int) string {
	content := ""
	for i := 0; i < lines; i++ {
		content += "line content here\n"
	}
	return content
}

func TestGenerateDiff_LineNumbers(t *testing.T) {
	oldContent := "line1\nline2\nline3"
	newContent := "line1\nmodified\nline3"

	diff := generateDiff(oldContent, newContent)

	// Verify line numbers are set correctly
	for _, line := range diff {
		switch line.Type {
		case DiffAdded:
			if line.NewNum == 0 {
				t.Error("added line should have NewNum set")
			}
		case DiffRemoved:
			if line.OldNum == 0 {
				t.Error("removed line should have OldNum set")
			}
		case DiffContext:
			if line.OldNum == 0 || line.NewNum == 0 {
				t.Error("context line should have both line numbers set")
			}
		}
	}
}
