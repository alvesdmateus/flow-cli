package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spinner provides a console spinner for indicating progress
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	style    lipgloss.Style

	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
	doneChan chan struct{}
}

// SpinnerOption configures a spinner
type SpinnerOption func(*Spinner)

// Default spinner frames
var (
	DotsFrames    = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	LineFrames    = []string{"|", "/", "-", "\\"}
	BouncingFrames = []string{"⠁", "⠂", "⠄", "⠂"}
	GrowingFrames = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}
)

// WithFrames sets custom spinner frames
func WithFrames(frames []string) SpinnerOption {
	return func(s *Spinner) {
		if len(frames) > 0 {
			s.frames = frames
		}
	}
}

// WithInterval sets the animation interval
func WithInterval(d time.Duration) SpinnerOption {
	return func(s *Spinner) {
		s.interval = d
	}
}

// WithStyle sets the spinner style
func WithStyle(style lipgloss.Style) SpinnerOption {
	return func(s *Spinner) {
		s.style = style
	}
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string, opts ...SpinnerOption) *Spinner {
	s := &Spinner{
		message:  message,
		frames:   DotsFrames,
		interval: 80 * time.Millisecond,
		style:    lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.doneChan = make(chan struct{})
	s.mu.Unlock()

	go s.run()
}

// Stop stops the spinner animation and clears the line
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	<-s.doneChan

	// Clear the spinner line
	fmt.Print("\r\033[K")
}

// StopWithMessage stops the spinner and displays a final message
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	fmt.Println(message)
}

// StopWithSuccess stops the spinner and displays a success message
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	fmt.Println(successStyle.Render("✓ " + message))
}

// StopWithError stops the spinner and displays an error message
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	fmt.Println(errorStyle.Render("✗ " + message))
}

// UpdateMessage updates the spinner message while running
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

func (s *Spinner) run() {
	defer close(s.doneChan)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frameIdx := 0

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.Lock()
			frame := s.frames[frameIdx]
			message := s.message
			s.mu.Unlock()

			// Print spinner with message
			fmt.Printf("\r\033[K%s %s", s.style.Render(frame), message)

			frameIdx = (frameIdx + 1) % len(s.frames)
		}
	}
}

// StartSpinner is a convenience function to start a spinner with a message
func StartSpinner(message string) *Spinner {
	s := NewSpinner(message)
	s.Start()
	return s
}

// Convenience functions for common spinner patterns

// SpinnerConnecting creates a spinner for connection operations
func SpinnerConnecting(endpoint string) *Spinner {
	return NewSpinner(
		fmt.Sprintf("Connecting to %s...", endpoint),
		WithFrames(DotsFrames),
	)
}

// SpinnerThinking creates a spinner for LLM thinking operations
func SpinnerThinking() *Spinner {
	return NewSpinner(
		"Thinking...",
		WithFrames(DotsFrames),
		WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("212"))),
	)
}

// SpinnerProcessing creates a spinner for processing operations
func SpinnerProcessing(task string) *Spinner {
	return NewSpinner(
		fmt.Sprintf("Processing %s...", task),
		WithFrames(BouncingFrames),
	)
}

// SpinnerLoading creates a spinner for loading operations
func SpinnerLoading(resource string) *Spinner {
	return NewSpinner(
		fmt.Sprintf("Loading %s...", resource),
		WithFrames(GrowingFrames),
		WithInterval(100*time.Millisecond),
	)
}
