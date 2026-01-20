package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ChangeType represents the type of file change
type ChangeType int

const (
	ChangeCreate ChangeType = iota
	ChangeModify
	ChangeDelete
)

func (c ChangeType) String() string {
	switch c {
	case ChangeCreate:
		return "create"
	case ChangeModify:
		return "modify"
	case ChangeDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// HistoryEntry represents a single change in the history
type HistoryEntry struct {
	ID          int
	Type        ChangeType
	Path        string
	OldContent  string
	NewContent  string
	Timestamp   time.Time
	Description string
}

// ChangeHistory manages undo/redo history for file changes
type ChangeHistory struct {
	entries    []HistoryEntry
	undone     []HistoryEntry // Stack of undone entries for redo
	maxHistory int
	nextID     int
	workDir    string
	mu         sync.Mutex
}

// NewChangeHistory creates a new change history manager
func NewChangeHistory(workDir string) *ChangeHistory {
	return &ChangeHistory{
		entries:    make([]HistoryEntry, 0),
		undone:     make([]HistoryEntry, 0),
		maxHistory: 100,
		nextID:     1,
		workDir:    workDir,
	}
}

// RecordCreate records a file creation
func (ch *ChangeHistory) RecordCreate(path, content, description string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.addEntry(HistoryEntry{
		ID:          ch.nextID,
		Type:        ChangeCreate,
		Path:        path,
		OldContent:  "",
		NewContent:  content,
		Timestamp:   time.Now(),
		Description: description,
	})
	ch.nextID++

	// Clear redo stack when new change is made
	ch.undone = make([]HistoryEntry, 0)
}

// RecordModify records a file modification
func (ch *ChangeHistory) RecordModify(path, oldContent, newContent, description string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.addEntry(HistoryEntry{
		ID:          ch.nextID,
		Type:        ChangeModify,
		Path:        path,
		OldContent:  oldContent,
		NewContent:  newContent,
		Timestamp:   time.Now(),
		Description: description,
	})
	ch.nextID++

	// Clear redo stack when new change is made
	ch.undone = make([]HistoryEntry, 0)
}

// RecordDelete records a file deletion
func (ch *ChangeHistory) RecordDelete(path, oldContent, description string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.addEntry(HistoryEntry{
		ID:          ch.nextID,
		Type:        ChangeDelete,
		Path:        path,
		OldContent:  oldContent,
		NewContent:  "",
		Timestamp:   time.Now(),
		Description: description,
	})
	ch.nextID++

	// Clear redo stack when new change is made
	ch.undone = make([]HistoryEntry, 0)
}

func (ch *ChangeHistory) addEntry(entry HistoryEntry) {
	ch.entries = append(ch.entries, entry)

	// Trim history if it exceeds max
	if len(ch.entries) > ch.maxHistory {
		ch.entries = ch.entries[len(ch.entries)-ch.maxHistory:]
	}
}

// CanUndo returns true if there are changes to undo
func (ch *ChangeHistory) CanUndo() bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	return len(ch.entries) > 0
}

// CanRedo returns true if there are changes to redo
func (ch *ChangeHistory) CanRedo() bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	return len(ch.undone) > 0
}

// Undo reverts the last change
func (ch *ChangeHistory) Undo() (*HistoryEntry, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.entries) == 0 {
		return nil, fmt.Errorf("nothing to undo")
	}

	// Pop the last entry
	entry := ch.entries[len(ch.entries)-1]
	ch.entries = ch.entries[:len(ch.entries)-1]

	// Apply the reverse change
	if err := ch.applyReverse(entry); err != nil {
		// Put it back if we failed
		ch.entries = append(ch.entries, entry)
		return nil, fmt.Errorf("failed to undo: %w", err)
	}

	// Add to redo stack
	ch.undone = append(ch.undone, entry)

	return &entry, nil
}

// Redo re-applies a previously undone change
func (ch *ChangeHistory) Redo() (*HistoryEntry, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.undone) == 0 {
		return nil, fmt.Errorf("nothing to redo")
	}

	// Pop from redo stack
	entry := ch.undone[len(ch.undone)-1]
	ch.undone = ch.undone[:len(ch.undone)-1]

	// Apply the change
	if err := ch.applyChange(entry); err != nil {
		// Put it back if we failed
		ch.undone = append(ch.undone, entry)
		return nil, fmt.Errorf("failed to redo: %w", err)
	}

	// Add back to history
	ch.entries = append(ch.entries, entry)

	return &entry, nil
}

// applyReverse applies the reverse of a change
func (ch *ChangeHistory) applyReverse(entry HistoryEntry) error {
	switch entry.Type {
	case ChangeCreate:
		// Reverse of create is delete
		return os.Remove(entry.Path)

	case ChangeModify:
		// Reverse of modify is restoring old content
		return os.WriteFile(entry.Path, []byte(entry.OldContent), 0644)

	case ChangeDelete:
		// Reverse of delete is create with old content
		dir := filepath.Dir(entry.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return os.WriteFile(entry.Path, []byte(entry.OldContent), 0644)

	default:
		return fmt.Errorf("unknown change type: %v", entry.Type)
	}
}

// applyChange applies a change (for redo)
func (ch *ChangeHistory) applyChange(entry HistoryEntry) error {
	switch entry.Type {
	case ChangeCreate:
		dir := filepath.Dir(entry.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return os.WriteFile(entry.Path, []byte(entry.NewContent), 0644)

	case ChangeModify:
		return os.WriteFile(entry.Path, []byte(entry.NewContent), 0644)

	case ChangeDelete:
		return os.Remove(entry.Path)

	default:
		return fmt.Errorf("unknown change type: %v", entry.Type)
	}
}

// History returns recent history entries
func (ch *ChangeHistory) History(limit int) []HistoryEntry {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if limit <= 0 || limit > len(ch.entries) {
		limit = len(ch.entries)
	}

	// Return most recent entries (reversed order)
	result := make([]HistoryEntry, limit)
	for i := 0; i < limit; i++ {
		result[i] = ch.entries[len(ch.entries)-1-i]
	}

	return result
}

// Clear clears all history
func (ch *ChangeHistory) Clear() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.entries = make([]HistoryEntry, 0)
	ch.undone = make([]HistoryEntry, 0)
}

// ShowHistory displays the change history
func (ch *ChangeHistory) ShowHistory(limit int) {
	entries := ch.History(limit)

	if len(entries) == 0 {
		fmt.Println(dimStyle.Render("No history available"))
		return
	}

	fmt.Println()
	PrintTitle("Change History")
	fmt.Println()

	for i, entry := range entries {
		displayPath := entry.Path
		if ch.workDir != "" {
			if rel, err := filepath.Rel(ch.workDir, entry.Path); err == nil {
				displayPath = rel
			}
		}

		var typeStyle string
		switch entry.Type {
		case ChangeCreate:
			typeStyle = addedStatsStyle.Render("[CREATE]")
		case ChangeModify:
			typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[MODIFY]")
		case ChangeDelete:
			typeStyle = removedStatsStyle.Render("[DELETE]")
		}

		timeAgo := formatTimeAgo(entry.Timestamp)

		fmt.Printf("  %d. %s %s\n",
			i+1,
			typeStyle,
			fileHeaderStyle.Render(displayPath))

		if entry.Description != "" {
			fmt.Printf("     %s\n", dimStyle.Render(entry.Description))
		}
		fmt.Printf("     %s\n", dimStyle.Render(timeAgo))
		fmt.Println()
	}

	// Show undo/redo status
	ch.mu.Lock()
	canUndo := len(ch.entries) > 0
	canRedo := len(ch.undone) > 0
	ch.mu.Unlock()

	var status []string
	if canUndo {
		status = append(status, "Ctrl+Z to undo")
	}
	if canRedo {
		status = append(status, "Ctrl+Y to redo")
	}

	if len(status) > 0 {
		fmt.Println(dimStyle.Render("  " + joinStrings(status, " | ")))
	}
}

// formatTimeAgo formats a timestamp as a relative time string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// UndoInteractive provides an interactive undo experience
func (ch *ChangeHistory) UndoInteractive() error {
	if !ch.CanUndo() {
		PrintInfo("Nothing to undo")
		return nil
	}

	// Show what will be undone
	ch.mu.Lock()
	lastEntry := ch.entries[len(ch.entries)-1]
	ch.mu.Unlock()

	displayPath := lastEntry.Path
	if ch.workDir != "" {
		if rel, err := filepath.Rel(ch.workDir, lastEntry.Path); err == nil {
			displayPath = rel
		}
	}

	fmt.Printf("\nUndo %s: %s\n", lastEntry.Type, displayPath)
	if lastEntry.Description != "" {
		fmt.Printf("  %s\n", dimStyle.Render(lastEntry.Description))
	}

	confirmed, err := Confirm("Proceed with undo?")
	if err != nil {
		return err
	}

	if !confirmed {
		PrintInfo("Undo cancelled")
		return nil
	}

	entry, err := ch.Undo()
	if err != nil {
		return err
	}

	PrintSuccess(fmt.Sprintf("Undone: %s %s", entry.Type, displayPath))
	return nil
}

// RedoInteractive provides an interactive redo experience
func (ch *ChangeHistory) RedoInteractive() error {
	if !ch.CanRedo() {
		PrintInfo("Nothing to redo")
		return nil
	}

	// Show what will be redone
	ch.mu.Lock()
	lastUndone := ch.undone[len(ch.undone)-1]
	ch.mu.Unlock()

	displayPath := lastUndone.Path
	if ch.workDir != "" {
		if rel, err := filepath.Rel(ch.workDir, lastUndone.Path); err == nil {
			displayPath = rel
		}
	}

	fmt.Printf("\nRedo %s: %s\n", lastUndone.Type, displayPath)
	if lastUndone.Description != "" {
		fmt.Printf("  %s\n", dimStyle.Render(lastUndone.Description))
	}

	confirmed, err := Confirm("Proceed with redo?")
	if err != nil {
		return err
	}

	if !confirmed {
		PrintInfo("Redo cancelled")
		return nil
	}

	entry, err := ch.Redo()
	if err != nil {
		return err
	}

	PrintSuccess(fmt.Sprintf("Redone: %s %s", entry.Type, displayPath))
	return nil
}
