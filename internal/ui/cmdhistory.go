package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// CommandEntry represents a command in history
type CommandEntry struct {
	Command   string
	Timestamp time.Time
	Count     int // Number of times this command was used
}

// CommandHistory manages command history with persistence
type CommandHistory struct {
	entries     []CommandEntry
	maxEntries  int
	historyFile string
	mu          sync.RWMutex
}

// NewCommandHistory creates a new command history manager
func NewCommandHistory(historyFile string) *CommandHistory {
	ch := &CommandHistory{
		entries:     make([]CommandEntry, 0),
		maxEntries:  1000,
		historyFile: historyFile,
	}
	ch.load()
	return ch
}

// Add adds a command to the history
func (ch *CommandHistory) Add(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Check if command already exists
	for i, entry := range ch.entries {
		if entry.Command == command {
			// Update existing entry
			ch.entries[i].Timestamp = time.Now()
			ch.entries[i].Count++
			// Move to end (most recent)
			entry := ch.entries[i]
			ch.entries = append(ch.entries[:i], ch.entries[i+1:]...)
			ch.entries = append(ch.entries, entry)
			ch.save()
			return
		}
	}

	// Add new entry
	ch.entries = append(ch.entries, CommandEntry{
		Command:   command,
		Timestamp: time.Now(),
		Count:     1,
	})

	// Trim if over max
	if len(ch.entries) > ch.maxEntries {
		ch.entries = ch.entries[len(ch.entries)-ch.maxEntries:]
	}

	ch.save()
}

// GetRecent returns the n most recent commands
func (ch *CommandHistory) GetRecent(n int) []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if n <= 0 || n > len(ch.entries) {
		n = len(ch.entries)
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = ch.entries[len(ch.entries)-1-i].Command
	}
	return result
}

// Search performs a fuzzy search on command history
func (ch *CommandHistory) Search(query string) []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if query == "" {
		return ch.GetRecent(20)
	}

	query = strings.ToLower(query)

	type scoredEntry struct {
		command string
		score   int
	}

	var matches []scoredEntry

	for _, entry := range ch.entries {
		score := fuzzyMatch(strings.ToLower(entry.Command), query)
		if score > 0 {
			// Boost score by frequency
			score += entry.Count
			matches = append(matches, scoredEntry{entry.Command, score})
		}
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Return top 20 matches
	limit := 20
	if len(matches) < limit {
		limit = len(matches)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = matches[i].command
	}

	return result
}

// fuzzyMatch returns a score for how well the pattern matches the text
func fuzzyMatch(text, pattern string) int {
	if pattern == "" {
		return 1
	}

	// Exact match gets highest score
	if text == pattern {
		return 1000
	}

	// Contains match
	if strings.Contains(text, pattern) {
		// Prefer matches at word boundaries
		if strings.HasPrefix(text, pattern) {
			return 500
		}
		// Check for word boundary match
		words := strings.Fields(text)
		for _, word := range words {
			if strings.HasPrefix(word, pattern) {
				return 400
			}
		}
		return 300
	}

	// Fuzzy character match
	score := 0
	patternIdx := 0
	consecutive := 0
	lastMatchIdx := -1

	for i := 0; i < len(text) && patternIdx < len(pattern); i++ {
		if text[i] == pattern[patternIdx] {
			score += 10
			patternIdx++

			// Bonus for consecutive matches
			if lastMatchIdx == i-1 {
				consecutive++
				score += consecutive * 5
			} else {
				consecutive = 0
			}
			lastMatchIdx = i

			// Bonus for matching at word start
			if i == 0 || text[i-1] == ' ' || text[i-1] == '/' || text[i-1] == '-' || text[i-1] == '_' {
				score += 20
			}
		}
	}

	// Only return score if we matched all pattern characters
	if patternIdx == len(pattern) {
		return score
	}

	return 0
}

// Clear clears all history
func (ch *CommandHistory) Clear() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.entries = make([]CommandEntry, 0)
	ch.save()
}

// load loads history from file
func (ch *CommandHistory) load() {
	if ch.historyFile == "" {
		return
	}

	file, err := os.Open(ch.historyFile)
	if err != nil {
		return // File doesn't exist yet
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse line: timestamp|count|command
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			// Old format: just command
			ch.entries = append(ch.entries, CommandEntry{
				Command:   line,
				Timestamp: time.Now(),
				Count:     1,
			})
			continue
		}

		ts, _ := time.Parse(time.RFC3339, parts[0])
		count := 1
		if c, err := parseInt(parts[1]); err == nil && c > 0 {
			count = c
		}

		ch.entries = append(ch.entries, CommandEntry{
			Command:   parts[2],
			Timestamp: ts,
			Count:     count,
		})
	}
}

// save persists history to file
func (ch *CommandHistory) save() {
	if ch.historyFile == "" {
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(ch.historyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	file, err := os.Create(ch.historyFile)
	if err != nil {
		return
	}
	defer file.Close()

	for _, entry := range ch.entries {
		line := entry.Timestamp.Format(time.RFC3339) + "|" +
			intToString(entry.Count) + "|" +
			entry.Command + "\n"
		file.WriteString(line)
	}
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// HistoryNavigator provides up/down navigation through history
type HistoryNavigator struct {
	history  *CommandHistory
	position int
	current  string // Current input before navigation
	buffer   string // Current buffer during navigation
	matches  []string
	query    string
}

// NewHistoryNavigator creates a new history navigator
func NewHistoryNavigator(history *CommandHistory) *HistoryNavigator {
	return &HistoryNavigator{
		history:  history,
		position: -1,
	}
}

// Reset resets the navigator state
func (hn *HistoryNavigator) Reset(current string) {
	hn.current = current
	hn.buffer = current
	hn.position = -1
	hn.matches = nil
	hn.query = ""
}

// Up navigates to the previous (older) command
func (hn *HistoryNavigator) Up() string {
	if hn.matches == nil {
		hn.matches = hn.history.GetRecent(100)
	}

	if len(hn.matches) == 0 {
		return hn.buffer
	}

	if hn.position < len(hn.matches)-1 {
		hn.position++
		hn.buffer = hn.matches[hn.position]
	}

	return hn.buffer
}

// Down navigates to the next (newer) command
func (hn *HistoryNavigator) Down() string {
	if hn.position > 0 {
		hn.position--
		hn.buffer = hn.matches[hn.position]
	} else if hn.position == 0 {
		hn.position = -1
		hn.buffer = hn.current
	}

	return hn.buffer
}

// Search updates the search query and returns matches
func (hn *HistoryNavigator) Search(query string) []string {
	hn.query = query
	hn.matches = hn.history.Search(query)
	hn.position = -1
	return hn.matches
}

// SelectMatch selects a match by index
func (hn *HistoryNavigator) SelectMatch(index int) string {
	if index >= 0 && index < len(hn.matches) {
		return hn.matches[index]
	}
	return hn.buffer
}

// ShowHistoryPopup displays an interactive history search popup
func (ch *CommandHistory) ShowHistoryPopup() (string, error) {
	entries := ch.GetRecent(20)
	if len(entries) == 0 {
		PrintInfo("No command history")
		return "", nil
	}

	options := make([]string, len(entries))
	for i, e := range entries {
		// Truncate long commands
		if len(e) > 60 {
			e = e[:57] + "..."
		}
		options[i] = e
	}

	selected, err := SelectOption("Command history:", options)
	if err != nil {
		return "", err
	}

	// Find full command
	for _, entry := range ch.entries {
		if strings.HasPrefix(entry.Command, selected) || entry.Command == selected {
			return entry.Command, nil
		}
	}

	return selected, nil
}

// FuzzyFinder provides an interactive fuzzy search interface
type FuzzyFinder struct {
	items    []string
	query    string
	selected int
	matches  []int
}

// NewFuzzyFinder creates a new fuzzy finder
func NewFuzzyFinder(items []string) *FuzzyFinder {
	matches := make([]int, len(items))
	for i := range items {
		matches[i] = i
	}
	return &FuzzyFinder{
		items:    items,
		matches:  matches,
		selected: 0,
	}
}

// SetQuery updates the search query
func (ff *FuzzyFinder) SetQuery(query string) {
	ff.query = query
	ff.updateMatches()
}

func (ff *FuzzyFinder) updateMatches() {
	if ff.query == "" {
		ff.matches = make([]int, len(ff.items))
		for i := range ff.items {
			ff.matches[i] = i
		}
		ff.selected = 0
		return
	}

	type scored struct {
		index int
		score int
	}

	var matches []scored
	query := strings.ToLower(ff.query)

	for i, item := range ff.items {
		score := fuzzyMatch(strings.ToLower(item), query)
		if score > 0 {
			matches = append(matches, scored{i, score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	ff.matches = make([]int, len(matches))
	for i, m := range matches {
		ff.matches[i] = m.index
	}

	ff.selected = 0
}

// MoveUp moves the selection up
func (ff *FuzzyFinder) MoveUp() {
	if ff.selected > 0 {
		ff.selected--
	}
}

// MoveDown moves the selection down
func (ff *FuzzyFinder) MoveDown() {
	if ff.selected < len(ff.matches)-1 {
		ff.selected++
	}
}

// Selected returns the currently selected item
func (ff *FuzzyFinder) Selected() string {
	if ff.selected >= 0 && ff.selected < len(ff.matches) {
		return ff.items[ff.matches[ff.selected]]
	}
	return ""
}

// Matches returns the current list of matching indices
func (ff *FuzzyFinder) Matches() []int {
	return ff.matches
}

// Render renders the fuzzy finder display
func (ff *FuzzyFinder) Render(maxDisplay int) string {
	var sb strings.Builder

	// Show query
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	sb.WriteString(promptStyle.Render("> "))
	sb.WriteString(ff.query)
	sb.WriteString("\n\n")

	if len(ff.matches) == 0 {
		sb.WriteString(dimStyle.Render("  No matches"))
		return sb.String()
	}

	// Show matches
	start := 0
	if ff.selected >= maxDisplay {
		start = ff.selected - maxDisplay + 1
	}

	end := start + maxDisplay
	if end > len(ff.matches) {
		end = len(ff.matches)
	}

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("75")).
		Bold(true)

	normalStyle := lipgloss.NewStyle()

	for i := start; i < end; i++ {
		item := ff.items[ff.matches[i]]
		if i == ff.selected {
			sb.WriteString(selectedStyle.Render("▸ " + item))
		} else {
			sb.WriteString(normalStyle.Render("  " + item))
		}
		sb.WriteString("\n")
	}

	// Show count
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render(intToString(len(ff.matches)) + "/" + intToString(len(ff.items)) + " matches"))

	return sb.String()
}
