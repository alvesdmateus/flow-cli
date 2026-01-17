package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tests for EditPreview

func TestNewEditPreview(t *testing.T) {
	ep := NewEditPreview("/test/dir")
	if ep == nil {
		t.Fatal("expected EditPreview to be created")
	}
	if ep.workDir != "/test/dir" {
		t.Errorf("expected workDir '/test/dir', got '%s'", ep.workDir)
	}
	if len(ep.changes) != 0 {
		t.Error("expected empty changes")
	}
}

func TestEditPreview_AddChange(t *testing.T) {
	ep := NewEditPreview("")

	ep.AddChange("test.go", "", "new content")
	if len(ep.changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(ep.changes))
	}

	change := ep.changes[0]
	if !change.IsNew {
		t.Error("expected IsNew to be true")
	}
	if change.Path != "test.go" {
		t.Errorf("expected path 'test.go', got '%s'", change.Path)
	}
}

func TestEditPreview_AddModification(t *testing.T) {
	ep := NewEditPreview("")

	ep.AddModification("test.go", "old", "new")
	if len(ep.changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(ep.changes))
	}

	change := ep.changes[0]
	if change.IsNew {
		t.Error("expected IsNew to be false")
	}
	if change.IsDelete {
		t.Error("expected IsDelete to be false")
	}
	if change.OldContent != "old" {
		t.Errorf("expected OldContent 'old', got '%s'", change.OldContent)
	}
}

func TestEditPreview_AddDeletion(t *testing.T) {
	ep := NewEditPreview("")

	ep.AddDeletion("test.go", "content")
	if len(ep.changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(ep.changes))
	}

	change := ep.changes[0]
	if !change.IsDelete {
		t.Error("expected IsDelete to be true")
	}
}

func TestEditPreview_Clear(t *testing.T) {
	ep := NewEditPreview("")
	ep.AddNewFile("a.go", "content a")
	ep.AddNewFile("b.go", "content b")

	ep.Clear()
	if ep.HasChanges() {
		t.Error("expected no changes after clear")
	}
}

func TestCountChanges(t *testing.T) {
	tests := []struct {
		name            string
		old             string
		new             string
		expectedAdded   int
		expectedRemoved int
	}{
		{
			name:            "new file",
			old:             "",
			new:             "line1\nline2\nline3",
			expectedAdded:   3,
			expectedRemoved: 0,
		},
		{
			name:            "deleted file",
			old:             "line1\nline2",
			new:             "",
			expectedAdded:   0,
			expectedRemoved: 2,
		},
		{
			name:            "modified file",
			old:             "line1\nline2",
			new:             "line1\nline3",
			expectedAdded:   1,
			expectedRemoved: 1,
		},
		{
			name:            "no changes",
			old:             "same",
			new:             "same",
			expectedAdded:   0,
			expectedRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := countChanges(tt.old, tt.new)
			if added != tt.expectedAdded {
				t.Errorf("expected added %d, got %d", tt.expectedAdded, added)
			}
			if removed != tt.expectedRemoved {
				t.Errorf("expected removed %d, got %d", tt.expectedRemoved, removed)
			}
		})
	}
}

// Tests for ChangeHistory

func TestNewChangeHistory(t *testing.T) {
	ch := NewChangeHistory("/test")
	if ch == nil {
		t.Fatal("expected ChangeHistory to be created")
	}
	if ch.CanUndo() {
		t.Error("expected CanUndo to be false initially")
	}
	if ch.CanRedo() {
		t.Error("expected CanRedo to be false initially")
	}
}

func TestChangeHistory_RecordCreate(t *testing.T) {
	ch := NewChangeHistory("")
	ch.RecordCreate("/test/file.go", "content", "created file")

	if !ch.CanUndo() {
		t.Error("expected CanUndo to be true after recording")
	}

	entries := ch.History(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Type != ChangeCreate {
		t.Errorf("expected type ChangeCreate, got %v", entry.Type)
	}
	if entry.Path != "/test/file.go" {
		t.Errorf("expected path '/test/file.go', got '%s'", entry.Path)
	}
}

func TestChangeHistory_RecordModify(t *testing.T) {
	ch := NewChangeHistory("")
	ch.RecordModify("/test/file.go", "old", "new", "modified file")

	entries := ch.History(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Type != ChangeModify {
		t.Errorf("expected type ChangeModify, got %v", entry.Type)
	}
	if entry.OldContent != "old" {
		t.Errorf("expected OldContent 'old', got '%s'", entry.OldContent)
	}
}

func TestChangeHistory_RecordDelete(t *testing.T) {
	ch := NewChangeHistory("")
	ch.RecordDelete("/test/file.go", "content", "deleted file")

	entries := ch.History(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Type != ChangeDelete {
		t.Errorf("expected type ChangeDelete, got %v", entry.Type)
	}
}

func TestChangeHistory_Clear(t *testing.T) {
	ch := NewChangeHistory("")
	ch.RecordCreate("/test/a.go", "a", "")
	ch.RecordCreate("/test/b.go", "b", "")

	ch.Clear()
	if ch.CanUndo() {
		t.Error("expected CanUndo to be false after clear")
	}
	if ch.CanRedo() {
		t.Error("expected CanRedo to be false after clear")
	}
}

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		ct       ChangeType
		expected string
	}{
		{ChangeCreate, "create"},
		{ChangeModify, "modify"},
		{ChangeDelete, "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.ct.String() != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, tt.ct.String())
			}
		})
	}
}

// Tests for CommandHistory

func TestNewCommandHistory(t *testing.T) {
	ch := NewCommandHistory("")
	if ch == nil {
		t.Fatal("expected CommandHistory to be created")
	}
}

func TestCommandHistory_Add(t *testing.T) {
	ch := NewCommandHistory("")

	ch.Add("test command")
	recent := ch.GetRecent(1)

	if len(recent) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recent))
	}
	if recent[0] != "test command" {
		t.Errorf("expected 'test command', got '%s'", recent[0])
	}
}

func TestCommandHistory_Add_Duplicate(t *testing.T) {
	ch := NewCommandHistory("")

	ch.Add("cmd1")
	ch.Add("cmd2")
	ch.Add("cmd1") // Duplicate - should move to end

	recent := ch.GetRecent(2)
	if recent[0] != "cmd1" {
		t.Error("expected cmd1 to be most recent")
	}
	if recent[1] != "cmd2" {
		t.Error("expected cmd2 to be second")
	}
}

func TestCommandHistory_Search(t *testing.T) {
	ch := NewCommandHistory("")

	ch.Add("go build")
	ch.Add("go test")
	ch.Add("npm install")
	ch.Add("go run main.go")

	results := ch.Search("go")
	if len(results) < 3 {
		t.Errorf("expected at least 3 results, got %d", len(results))
	}

	// All results should contain "go"
	for _, r := range results {
		if !strings.Contains(strings.ToLower(r), "go") {
			t.Errorf("expected result to contain 'go', got '%s'", r)
		}
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		text     string
		pattern  string
		minScore int
	}{
		{"exact match", "exact match", 500}, // Exact match
		{"go build", "go", 300},             // Contains
		{"go build", "gb", 10},              // Fuzzy
		{"something", "xyz", 0},             // No match
		{"", "test", 0},                     // Empty text
	}

	for _, tt := range tests {
		t.Run(tt.text+"_"+tt.pattern, func(t *testing.T) {
			score := fuzzyMatch(tt.text, tt.pattern)
			if score < tt.minScore {
				t.Errorf("expected score >= %d, got %d", tt.minScore, score)
			}
		})
	}
}

// Tests for Progress

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Loading...")
	if s == nil {
		t.Fatal("expected Spinner to be created")
	}
	if s.message != "Loading..." {
		t.Errorf("expected message 'Loading...', got '%s'", s.message)
	}
}

func TestSpinner_UpdateMessage(t *testing.T) {
	s := NewSpinner("Loading...")
	s.UpdateMessage("Almost done...")

	s.mu.Lock()
	msg := s.message
	s.mu.Unlock()

	if msg != "Almost done..." {
		t.Errorf("expected message 'Almost done...', got '%s'", msg)
	}
}

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(100, "Processing")
	if pb == nil {
		t.Fatal("expected ProgressBar to be created")
	}
	if pb.total != 100 {
		t.Errorf("expected total 100, got %d", pb.total)
	}
}

func TestProgressBar_Increment(t *testing.T) {
	pb := NewProgressBar(10, "Test")
	pb.Increment()
	pb.Increment()

	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 2 {
		t.Errorf("expected current 2, got %d", current)
	}
}

func TestProgressBar_Set(t *testing.T) {
	pb := NewProgressBar(100, "Test")
	pb.Set(50)

	pb.mu.Lock()
	current := pb.current
	pb.mu.Unlock()

	if current != 50 {
		t.Errorf("expected current 50, got %d", current)
	}
}

func TestProgressBar_Set_Clamping(t *testing.T) {
	pb := NewProgressBar(100, "Test")

	pb.Set(-10)
	pb.mu.Lock()
	if pb.current != 0 {
		t.Error("expected current to be clamped to 0")
	}
	pb.mu.Unlock()

	pb.Set(200)
	pb.mu.Lock()
	if pb.current != 100 {
		t.Error("expected current to be clamped to 100")
	}
	pb.mu.Unlock()
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{2 * time.Second, "2.0s"},
		{90 * time.Second, "1m30s"},
		{2 * time.Hour, "2h0m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Tests for Markdown

func TestRenderMarkdown_Headers(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"# Header 1", "Header 1"},
		{"## Header 2", "Header 2"},
		{"### Header 3", "Header 3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain '%s', got '%s'", tt.contains, result)
			}
		})
	}
}

func TestRenderMarkdown_Lists(t *testing.T) {
	input := `- Item 1
- Item 2
* Item 3`

	result := RenderMarkdown(input)

	if !strings.Contains(result, "Item 1") {
		t.Error("expected result to contain 'Item 1'")
	}
	if !strings.Contains(result, "•") {
		t.Error("expected result to contain bullet character")
	}
}

func TestRenderMarkdown_CodeBlock(t *testing.T) {
	input := "```go\nfunc main() {}\n```"
	result := RenderMarkdown(input)

	if !strings.Contains(result, "func main") {
		t.Errorf("expected result to contain code, got '%s'", result)
	}
}

func TestLanguageFromExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{"go", "go"},
		{"py", "python"},
		{"js", "javascript"},
		{"ts", "typescript"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			keywords := map[string][]string{
				"go":         {"func", "package"},
				"python":     {"def", "class"},
				"javascript": {"function", "const"},
				"typescript": {"function", "interface"},
			}

			if _, ok := keywords[tt.expected]; !ok {
				t.Errorf("language '%s' not in keywords map", tt.expected)
			}
		})
	}
}

func TestTable_Render(t *testing.T) {
	table := NewTable("Name", "Value")
	table.AddRow("key1", "val1")
	table.AddRow("key2", "val2")

	result := table.Render()

	if !strings.Contains(result, "Name") {
		t.Error("expected table to contain header 'Name'")
	}
	if !strings.Contains(result, "key1") {
		t.Error("expected table to contain 'key1'")
	}
	if !strings.Contains(result, "val2") {
		t.Error("expected table to contain 'val2'")
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		time     time.Time
		contains string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "minutes ago"},
		{now.Add(-2 * time.Hour), "hours ago"},
		{now.Add(-48 * time.Hour), "days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected '%s' to contain '%s'", result, tt.contains)
			}
		})
	}
}

// Test FuzzyFinder

func TestNewFuzzyFinder(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	ff := NewFuzzyFinder(items)

	if ff == nil {
		t.Fatal("expected FuzzyFinder to be created")
	}
	if len(ff.matches) != 3 {
		t.Errorf("expected 3 matches, got %d", len(ff.matches))
	}
}

func TestFuzzyFinder_SetQuery(t *testing.T) {
	items := []string{"alpha", "beta", "gamma", "delta"}
	ff := NewFuzzyFinder(items)

	ff.SetQuery("a")
	if len(ff.matches) < 2 {
		t.Errorf("expected at least 2 matches for 'a', got %d", len(ff.matches))
	}
}

func TestFuzzyFinder_MoveSelection(t *testing.T) {
	items := []string{"a", "b", "c"}
	ff := NewFuzzyFinder(items)

	if ff.selected != 0 {
		t.Error("expected initial selection to be 0")
	}

	ff.MoveDown()
	if ff.selected != 1 {
		t.Errorf("expected selection 1 after MoveDown, got %d", ff.selected)
	}

	ff.MoveUp()
	if ff.selected != 0 {
		t.Errorf("expected selection 0 after MoveUp, got %d", ff.selected)
	}
}

func TestFuzzyFinder_Selected(t *testing.T) {
	items := []string{"first", "second", "third"}
	ff := NewFuzzyFinder(items)

	if ff.Selected() != "first" {
		t.Errorf("expected 'first', got '%s'", ff.Selected())
	}

	ff.MoveDown()
	if ff.Selected() != "second" {
		t.Errorf("expected 'second', got '%s'", ff.Selected())
	}
}

// Integration test for file operations
func TestEditPreview_ApplyChanges_Integration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "editpreview_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ep := NewEditPreview(tmpDir)

	// Add a new file
	newFilePath := filepath.Join(tmpDir, "new.txt")
	ep.AddNewFile(newFilePath, "new file content")

	// Apply changes
	if err := ep.ApplyChanges(); err != nil {
		t.Fatalf("failed to apply changes: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(newFilePath)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	if string(content) != "new file content" {
		t.Errorf("expected 'new file content', got '%s'", string(content))
	}
}
