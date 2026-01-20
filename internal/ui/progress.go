package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ProgressStyle defines the visual style for progress indicators
type ProgressStyle int

const (
	ProgressStyleSpinner ProgressStyle = iota
	ProgressStyleBar
	ProgressStyleDots
	ProgressStylePulse
)

var (
	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75"))

	progressCompleteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46"))

	progressBarFilled = lipgloss.NewStyle().
				Foreground(lipgloss.Color("75")).
				Render("█")

	progressBarEmpty = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("░")
)

// Spinner frames for animation (used by MultiProgress)
var defaultSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ProgressBar provides a determinate progress bar
type ProgressBar struct {
	total     int
	current   int
	width     int
	message   string
	output    io.Writer
	startTime time.Time
	mu        sync.Mutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, message string) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     40,
		message:   message,
		output:    os.Stdout,
		startTime: time.Now(),
	}
}

// SetWidth sets the progress bar width
func (pb *ProgressBar) SetWidth(width int) *ProgressBar {
	pb.width = width
	return pb
}

// Increment advances the progress bar by one
func (pb *ProgressBar) Increment() {
	pb.mu.Lock()
	pb.current++
	if pb.current > pb.total {
		pb.current = pb.total
	}
	pb.mu.Unlock()
	pb.render()
}

// Set sets the current progress value
func (pb *ProgressBar) Set(value int) {
	pb.mu.Lock()
	pb.current = value
	if pb.current < 0 {
		pb.current = 0
	}
	if pb.current > pb.total {
		pb.current = pb.total
	}
	pb.mu.Unlock()
	pb.render()
}

// SetMessage updates the progress bar message
func (pb *ProgressBar) SetMessage(message string) {
	pb.mu.Lock()
	pb.message = message
	pb.mu.Unlock()
	pb.render()
}

// Complete marks the progress as complete
func (pb *ProgressBar) Complete() {
	pb.mu.Lock()
	pb.current = pb.total
	pb.mu.Unlock()
	pb.render()
	fmt.Fprintln(pb.output)
}

// render updates the progress bar display
func (pb *ProgressBar) render() {
	pb.mu.Lock()
	current := pb.current
	total := pb.total
	message := pb.message
	elapsed := time.Since(pb.startTime)
	pb.mu.Unlock()

	if total == 0 {
		return
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(pb.width))
	empty := pb.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Calculate ETA
	var eta string
	if current > 0 && current < total {
		rate := float64(current) / elapsed.Seconds()
		remaining := float64(total-current) / rate
		eta = fmt.Sprintf(" ETA: %s", formatDuration(time.Duration(remaining)*time.Second))
	}

	fmt.Fprintf(pb.output, "\r\033[K%s [%s] %d/%d (%.0f%%)%s",
		message,
		progressStyle.Render(bar),
		current,
		total,
		percentage*100,
		dimStyle.Render(eta))
}

// MultiProgress manages multiple progress indicators
type MultiProgress struct {
	items   []*ProgressItem
	output  io.Writer
	running bool
	done    chan struct{}
	mu      sync.Mutex
}

// ProgressItem represents a single item in multi-progress
type ProgressItem struct {
	Name    string
	Status  string
	Done    bool
	Error   bool
	Current int
	Total   int
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		items:  make([]*ProgressItem, 0),
		output: os.Stdout,
	}
}

// AddItem adds a new progress item
func (mp *MultiProgress) AddItem(name string) *ProgressItem {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	item := &ProgressItem{
		Name:   name,
		Status: "pending",
	}
	mp.items = append(mp.items, item)
	return item
}

// Start begins rendering the multi-progress display
func (mp *MultiProgress) Start() {
	mp.mu.Lock()
	if mp.running {
		mp.mu.Unlock()
		return
	}
	mp.running = true
	mp.done = make(chan struct{})
	mp.mu.Unlock()

	// Initial render
	mp.render()

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-mp.done:
				return
			case <-ticker.C:
				mp.render()
			}
		}
	}()
}

// Stop stops the multi-progress display
func (mp *MultiProgress) Stop() {
	mp.mu.Lock()
	if !mp.running {
		mp.mu.Unlock()
		return
	}
	mp.running = false
	close(mp.done)
	mp.mu.Unlock()

	// Final render
	mp.render()
	fmt.Fprintln(mp.output)
}

// UpdateItem updates a progress item's status
func (mp *MultiProgress) UpdateItem(item *ProgressItem, status string) {
	mp.mu.Lock()
	item.Status = status
	mp.mu.Unlock()
}

// CompleteItem marks an item as complete
func (mp *MultiProgress) CompleteItem(item *ProgressItem) {
	mp.mu.Lock()
	item.Done = true
	item.Status = "done"
	mp.mu.Unlock()
}

// FailItem marks an item as failed
func (mp *MultiProgress) FailItem(item *ProgressItem, err string) {
	mp.mu.Lock()
	item.Error = true
	item.Status = err
	mp.mu.Unlock()
}

func (mp *MultiProgress) render() {
	mp.mu.Lock()
	items := make([]*ProgressItem, len(mp.items))
	copy(items, mp.items)
	mp.mu.Unlock()

	// Move cursor up and clear lines
	if len(items) > 0 {
		fmt.Fprintf(mp.output, "\033[%dA", len(items))
	}

	for _, item := range items {
		var icon string
		var style lipgloss.Style

		if item.Done {
			icon = "✓"
			style = progressCompleteStyle
		} else if item.Error {
			icon = "✗"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		} else {
			icon = defaultSpinnerFrames[int(time.Now().UnixMilli()/100)%len(defaultSpinnerFrames)]
			style = progressStyle
		}

		fmt.Fprintf(mp.output, "\033[K%s %s: %s\n",
			style.Render(icon),
			item.Name,
			dimStyle.Render(item.Status))
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// WithProgress runs a function with a spinner
func WithProgress(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()

	err := fn()

	if err != nil {
		spinner.StopWithError(err.Error())
		return err
	}

	spinner.StopWithMessage(message)
	return nil
}

// WithProgressBar runs a function with a progress bar
func WithProgressBar(message string, total int, fn func(update func(int))) error {
	bar := NewProgressBar(total, message)
	bar.render()

	fn(func(current int) {
		bar.Set(current)
	})

	bar.Complete()
	return nil
}
