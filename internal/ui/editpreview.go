package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FileChange represents a pending change to a file
type FileChange struct {
	Path       string
	OldContent string
	NewContent string
	IsNew      bool
	IsDelete   bool
}

// EditPreview manages multi-file edit previews
type EditPreview struct {
	changes     []FileChange
	workDir     string
	maxLines    int
	showContext int
}

// NewEditPreview creates a new edit preview manager
func NewEditPreview(workDir string) *EditPreview {
	return &EditPreview{
		changes:     make([]FileChange, 0),
		workDir:     workDir,
		maxLines:    100,
		showContext: 3,
	}
}

// AddChange adds a file change to the preview
func (ep *EditPreview) AddChange(path, oldContent, newContent string) {
	isNew := oldContent == ""
	isDelete := newContent == ""

	ep.changes = append(ep.changes, FileChange{
		Path:       path,
		OldContent: oldContent,
		NewContent: newContent,
		IsNew:      isNew,
		IsDelete:   isDelete,
	})
}

// AddNewFile adds a new file to the preview
func (ep *EditPreview) AddNewFile(path, content string) {
	ep.AddChange(path, "", content)
}

// AddModification adds a file modification to the preview
func (ep *EditPreview) AddModification(path, oldContent, newContent string) {
	ep.AddChange(path, oldContent, newContent)
}

// AddDeletion adds a file deletion to the preview
func (ep *EditPreview) AddDeletion(path, oldContent string) {
	ep.AddChange(path, oldContent, "")
}

// Clear clears all pending changes
func (ep *EditPreview) Clear() {
	ep.changes = make([]FileChange, 0)
}

// Changes returns the list of pending changes
func (ep *EditPreview) Changes() []FileChange {
	return ep.changes
}

// HasChanges returns true if there are pending changes
func (ep *EditPreview) HasChanges() bool {
	return len(ep.changes) > 0
}

// Summary styles
var (
	summaryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	fileHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("75"))

	newFileBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("28")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Render("NEW")

	modifiedFileBadge = lipgloss.NewStyle().
				Background(lipgloss.Color("172")).
				Foreground(lipgloss.Color("0")).
				Padding(0, 1).
				Render("MOD")

	deletedFileBadge = lipgloss.NewStyle().
				Background(lipgloss.Color("160")).
				Foreground(lipgloss.Color("255")).
				Padding(0, 1).
				Render("DEL")

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	addedStatsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	removedStatsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))
)

// ShowSummary displays a summary of all pending changes
func (ep *EditPreview) ShowSummary() {
	if !ep.HasChanges() {
		fmt.Println(dimStyle.Render("No pending changes"))
		return
	}

	var totalAdded, totalRemoved int
	var newFiles, modFiles, delFiles int

	fmt.Println()
	PrintTitle(fmt.Sprintf("Pending Changes (%d files)", len(ep.changes)))
	fmt.Println()

	for _, change := range ep.changes {
		var badge string
		if change.IsNew {
			badge = newFileBadge
			newFiles++
		} else if change.IsDelete {
			badge = deletedFileBadge
			delFiles++
		} else {
			badge = modifiedFileBadge
			modFiles++
		}

		// Calculate stats
		added, removed := countChanges(change.OldContent, change.NewContent)
		totalAdded += added
		totalRemoved += removed

		// Make path relative if possible
		displayPath := change.Path
		if ep.workDir != "" {
			if rel, err := filepath.Rel(ep.workDir, change.Path); err == nil {
				displayPath = rel
			}
		}

		// Display file entry
		stats := fmt.Sprintf("%s +%d %s -%d",
			addedStatsStyle.Render(""),
			added,
			removedStatsStyle.Render(""),
			removed)

		fmt.Printf("  %s %s %s\n",
			badge,
			fileHeaderStyle.Render(displayPath),
			statsStyle.Render(stats))
	}

	fmt.Println()

	// Summary line
	summary := fmt.Sprintf("Summary: %s new, %s modified, %s deleted | %s insertions, %s deletions",
		addedStatsStyle.Render(fmt.Sprintf("%d", newFiles)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(fmt.Sprintf("%d", modFiles)),
		removedStatsStyle.Render(fmt.Sprintf("%d", delFiles)),
		addedStatsStyle.Render(fmt.Sprintf("+%d", totalAdded)),
		removedStatsStyle.Render(fmt.Sprintf("-%d", totalRemoved)))

	fmt.Println(summaryBoxStyle.Render(summary))
	fmt.Println()
}

// ShowFullPreview displays detailed diffs for all changes
func (ep *EditPreview) ShowFullPreview() {
	if !ep.HasChanges() {
		fmt.Println(dimStyle.Render("No pending changes"))
		return
	}

	for i, change := range ep.changes {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 60))
		}

		// Make path relative if possible
		displayPath := change.Path
		if ep.workDir != "" {
			if rel, err := filepath.Rel(ep.workDir, change.Path); err == nil {
				displayPath = rel
			}
		}

		if change.IsDelete {
			ep.showDeletePreview(displayPath, change.OldContent)
		} else {
			ShowDiff(displayPath, change.OldContent, change.NewContent)
		}
	}
}

// ShowFilePreview displays the diff for a specific file
func (ep *EditPreview) ShowFilePreview(index int) error {
	if index < 0 || index >= len(ep.changes) {
		return fmt.Errorf("invalid file index: %d", index)
	}

	change := ep.changes[index]
	displayPath := change.Path
	if ep.workDir != "" {
		if rel, err := filepath.Rel(ep.workDir, change.Path); err == nil {
			displayPath = rel
		}
	}

	if change.IsDelete {
		ep.showDeletePreview(displayPath, change.OldContent)
	} else {
		ShowDiff(displayPath, change.OldContent, change.NewContent)
	}

	return nil
}

func (ep *EditPreview) showDeletePreview(path, content string) {
	fmt.Println()
	fmt.Println(removedLineStyle.Render("- Deleted: " + path))
	fmt.Println()

	lines := strings.Split(content, "\n")
	maxLines := 20
	if len(lines) > maxLines {
		for i, line := range lines[:maxLines] {
			fmt.Printf("%s %s\n",
				lineNumberStyle.Render(fmt.Sprintf("%d", i+1)),
				removedLineStyle.Render("- "+line))
		}
		fmt.Printf("%s ... (%d more lines)\n",
			lineNumberStyle.Render(""),
			len(lines)-maxLines)
	} else {
		for i, line := range lines {
			fmt.Printf("%s %s\n",
				lineNumberStyle.Render(fmt.Sprintf("%d", i+1)),
				removedLineStyle.Render("- "+line))
		}
	}
	fmt.Println()
}

// ApplyChanges applies all pending changes to disk
func (ep *EditPreview) ApplyChanges() error {
	for _, change := range ep.changes {
		if change.IsDelete {
			if err := os.Remove(change.Path); err != nil {
				return fmt.Errorf("failed to delete %s: %w", change.Path, err)
			}
		} else {
			// Ensure directory exists
			dir := filepath.Dir(change.Path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			if err := os.WriteFile(change.Path, []byte(change.NewContent), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", change.Path, err)
			}
		}
	}
	return nil
}

// ApplyChange applies a specific change by index
func (ep *EditPreview) ApplyChange(index int) error {
	if index < 0 || index >= len(ep.changes) {
		return fmt.Errorf("invalid change index: %d", index)
	}

	change := ep.changes[index]

	if change.IsDelete {
		if err := os.Remove(change.Path); err != nil {
			return fmt.Errorf("failed to delete %s: %w", change.Path, err)
		}
	} else {
		dir := filepath.Dir(change.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(change.Path, []byte(change.NewContent), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", change.Path, err)
		}
	}

	return nil
}

// RemoveChange removes a change from the preview without applying it
func (ep *EditPreview) RemoveChange(index int) error {
	if index < 0 || index >= len(ep.changes) {
		return fmt.Errorf("invalid change index: %d", index)
	}

	ep.changes = append(ep.changes[:index], ep.changes[index+1:]...)
	return nil
}

// PromptApproval shows an interactive approval prompt for changes
func (ep *EditPreview) PromptApproval() (bool, error) {
	if !ep.HasChanges() {
		return false, nil
	}

	ep.ShowSummary()

	options := []string{
		"Apply all changes",
		"Review each file",
		"Cancel",
	}

	choice, err := SelectOption("What would you like to do?", options)
	if err != nil {
		return false, err
	}

	switch choice {
	case "Apply all changes":
		return true, nil
	case "Review each file":
		return ep.reviewEachFile()
	default:
		return false, nil
	}
}

func (ep *EditPreview) reviewEachFile() (bool, error) {
	approved := make([]bool, len(ep.changes))

	for i := range ep.changes {
		change := ep.changes[i]

		displayPath := change.Path
		if ep.workDir != "" {
			if rel, err := filepath.Rel(ep.workDir, change.Path); err == nil {
				displayPath = rel
			}
		}

		fmt.Printf("\n--- File %d/%d: %s ---\n", i+1, len(ep.changes), displayPath)

		if err := ep.ShowFilePreview(i); err != nil {
			return false, err
		}

		options := []string{
			"Accept this change",
			"Skip this change",
			"Accept all remaining",
			"Cancel all",
		}

		choice, err := SelectOption("", options)
		if err != nil {
			return false, err
		}

		switch choice {
		case "Accept this change":
			approved[i] = true
		case "Skip this change":
			approved[i] = false
		case "Accept all remaining":
			for j := i; j < len(ep.changes); j++ {
				approved[j] = true
			}
			goto done
		case "Cancel all":
			return false, nil
		}
	}

done:
	// Apply approved changes in reverse order to preserve indices
	appliedCount := 0
	for i := len(approved) - 1; i >= 0; i-- {
		if approved[i] {
			if err := ep.ApplyChange(i); err != nil {
				PrintError(fmt.Sprintf("Failed to apply change: %v", err))
			} else {
				appliedCount++
			}
		}
	}

	if appliedCount > 0 {
		PrintSuccess(fmt.Sprintf("Applied %d change(s)", appliedCount))
	}

	return appliedCount > 0, nil
}

// countChanges counts added and removed lines between old and new content
func countChanges(oldContent, newContent string) (added, removed int) {
	if oldContent == "" {
		// New file - all lines are added
		return len(strings.Split(newContent, "\n")), 0
	}
	if newContent == "" {
		// Deleted file - all lines are removed
		return 0, len(strings.Split(oldContent, "\n"))
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Build a set of old lines
	oldSet := make(map[string]int)
	for _, line := range oldLines {
		oldSet[line]++
	}

	// Count lines in new that aren't in old
	for _, line := range newLines {
		if oldSet[line] > 0 {
			oldSet[line]--
		} else {
			added++
		}
	}

	// Count lines in old that aren't in new
	newSet := make(map[string]int)
	for _, line := range newLines {
		newSet[line]++
	}

	for _, line := range oldLines {
		if newSet[line] > 0 {
			newSet[line]--
		} else {
			removed++
		}
	}

	return added, removed
}

// GenerateUnifiedDiff generates a unified diff format string for all changes
func (ep *EditPreview) GenerateUnifiedDiff() string {
	var sb strings.Builder

	for _, change := range ep.changes {
		displayPath := change.Path
		if ep.workDir != "" {
			if rel, err := filepath.Rel(ep.workDir, change.Path); err == nil {
				displayPath = rel
			}
		}

		if change.IsNew {
			sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", displayPath, displayPath))
			sb.WriteString("new file mode 100644\n")
			sb.WriteString(fmt.Sprintf("--- /dev/null\n"))
			sb.WriteString(fmt.Sprintf("+++ b/%s\n", displayPath))

			lines := strings.Split(change.NewContent, "\n")
			sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
			for _, line := range lines {
				sb.WriteString("+" + line + "\n")
			}
		} else if change.IsDelete {
			sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", displayPath, displayPath))
			sb.WriteString("deleted file mode 100644\n")
			sb.WriteString(fmt.Sprintf("--- a/%s\n", displayPath))
			sb.WriteString("+++ /dev/null\n")

			lines := strings.Split(change.OldContent, "\n")
			sb.WriteString(fmt.Sprintf("@@ -1,%d +0,0 @@\n", len(lines)))
			for _, line := range lines {
				sb.WriteString("-" + line + "\n")
			}
		} else {
			sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", displayPath, displayPath))
			sb.WriteString(fmt.Sprintf("--- a/%s\n", displayPath))
			sb.WriteString(fmt.Sprintf("+++ b/%s\n", displayPath))

			// Generate hunks
			oldLines := strings.Split(change.OldContent, "\n")
			newLines := strings.Split(change.NewContent, "\n")
			hunks := generateHunks(oldLines, newLines)

			for _, hunk := range hunks {
				sb.WriteString(hunk)
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// generateHunks creates unified diff hunks
func generateHunks(oldLines, newLines []string) []string {
	var hunks []string

	diffLines := generateDiff(strings.Join(oldLines, "\n"), strings.Join(newLines, "\n"))

	// Find regions of changes
	type region struct {
		start, end int
	}

	var regions []region
	inChange := false
	changeStart := 0

	for i, line := range diffLines {
		if line.Type != DiffContext {
			if !inChange {
				inChange = true
				changeStart = i
			}
		} else {
			if inChange {
				regions = append(regions, region{changeStart, i})
				inChange = false
			}
		}
	}
	if inChange {
		regions = append(regions, region{changeStart, len(diffLines)})
	}

	// Generate hunk for each region
	const contextLines = 3

	for _, r := range regions {
		var hunk strings.Builder

		// Calculate hunk bounds with context
		hunkStart := r.start - contextLines
		if hunkStart < 0 {
			hunkStart = 0
		}
		hunkEnd := r.end + contextLines
		if hunkEnd > len(diffLines) {
			hunkEnd = len(diffLines)
		}

		// Calculate line numbers
		var oldStart, oldCount, newStart, newCount int
		for i := hunkStart; i < hunkEnd; i++ {
			line := diffLines[i]
			switch line.Type {
			case DiffContext:
				if oldStart == 0 {
					oldStart = line.OldNum
				}
				if newStart == 0 {
					newStart = line.NewNum
				}
				oldCount++
				newCount++
			case DiffRemoved:
				if oldStart == 0 {
					oldStart = line.OldNum
				}
				oldCount++
			case DiffAdded:
				if newStart == 0 {
					newStart = line.NewNum
				}
				newCount++
			}
		}

		if oldStart == 0 {
			oldStart = 1
		}
		if newStart == 0 {
			newStart = 1
		}

		hunk.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount))

		for i := hunkStart; i < hunkEnd; i++ {
			line := diffLines[i]
			switch line.Type {
			case DiffContext:
				hunk.WriteString(" " + line.Content + "\n")
			case DiffRemoved:
				hunk.WriteString("-" + line.Content + "\n")
			case DiffAdded:
				hunk.WriteString("+" + line.Content + "\n")
			}
		}

		hunks = append(hunks, hunk.String())
	}

	return hunks
}
