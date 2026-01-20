package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("test message")

	if s == nil {
		t.Fatal("NewSpinner() returned nil")
	}

	if s.message != "test message" {
		t.Errorf("message = %q, want %q", s.message, "test message")
	}

	if len(s.frames) == 0 {
		t.Error("frames should not be empty")
	}

	if s.interval == 0 {
		t.Error("interval should not be zero")
	}
}

func TestNewSpinner_WithOptions(t *testing.T) {
	customFrames := []string{"a", "b", "c"}
	customInterval := 200 * time.Millisecond
	customStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	s := NewSpinner("test",
		WithFrames(customFrames),
		WithInterval(customInterval),
		WithStyle(customStyle),
	)

	if len(s.frames) != 3 {
		t.Errorf("frames length = %d, want 3", len(s.frames))
	}

	if s.interval != customInterval {
		t.Errorf("interval = %v, want %v", s.interval, customInterval)
	}
}

func TestSpinner_StartStop(t *testing.T) {
	s := NewSpinner("test")

	// Start the spinner
	s.Start()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	if !s.running {
		t.Error("spinner should be running after Start()")
	}

	// Stop the spinner
	s.Stop()

	if s.running {
		t.Error("spinner should not be running after Stop()")
	}
}

func TestSpinner_DoubleStart(t *testing.T) {
	s := NewSpinner("test")

	s.Start()
	s.Start() // Should not panic or create issues

	time.Sleep(50 * time.Millisecond)

	s.Stop()
}

func TestSpinner_DoubleStop(t *testing.T) {
	s := NewSpinner("test")

	s.Start()
	time.Sleep(50 * time.Millisecond)

	s.Stop()
	s.Stop() // Should not panic
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	s := NewSpinner("test")

	// Should not panic
	s.Stop()
}

func TestSpinner_UpdateMessage(t *testing.T) {
	s := NewSpinner("initial message")

	s.UpdateMessage("updated message")

	if s.message != "updated message" {
		t.Errorf("message = %q, want %q", s.message, "updated message")
	}
}

func TestSpinner_UpdateMessageWhileRunning(t *testing.T) {
	s := NewSpinner("initial")
	s.Start()

	time.Sleep(50 * time.Millisecond)

	s.UpdateMessage("updated")

	s.mu.Lock()
	msg := s.message
	s.mu.Unlock()

	if msg != "updated" {
		t.Errorf("message = %q, want %q", msg, "updated")
	}

	s.Stop()
}

func TestStartSpinner(t *testing.T) {
	s := StartSpinner("test")

	if s == nil {
		t.Fatal("StartSpinner() returned nil")
	}

	if !s.running {
		t.Error("spinner should be running after StartSpinner()")
	}

	s.Stop()
}

func TestSpinnerConnecting(t *testing.T) {
	s := SpinnerConnecting("localhost:8080")

	if s == nil {
		t.Fatal("SpinnerConnecting() returned nil")
	}

	if s.message == "" {
		t.Error("message should not be empty")
	}
}

func TestSpinnerThinking(t *testing.T) {
	s := SpinnerThinking()

	if s == nil {
		t.Fatal("SpinnerThinking() returned nil")
	}

	if s.message != "Thinking..." {
		t.Errorf("message = %q, want %q", s.message, "Thinking...")
	}
}

func TestSpinnerProcessing(t *testing.T) {
	s := SpinnerProcessing("task")

	if s == nil {
		t.Fatal("SpinnerProcessing() returned nil")
	}

	if s.message == "" {
		t.Error("message should not be empty")
	}
}

func TestSpinnerLoading(t *testing.T) {
	s := SpinnerLoading("resource")

	if s == nil {
		t.Fatal("SpinnerLoading() returned nil")
	}

	if s.message == "" {
		t.Error("message should not be empty")
	}
}

func TestFrameConstants(t *testing.T) {
	// Test that frame constants are not empty
	if len(DotsFrames) == 0 {
		t.Error("DotsFrames should not be empty")
	}

	if len(LineFrames) == 0 {
		t.Error("LineFrames should not be empty")
	}

	if len(BouncingFrames) == 0 {
		t.Error("BouncingFrames should not be empty")
	}

	if len(GrowingFrames) == 0 {
		t.Error("GrowingFrames should not be empty")
	}
}

func TestWithFrames_EmptySlice(t *testing.T) {
	s := NewSpinner("test", WithFrames([]string{}))

	// Should keep default frames when empty slice provided
	if len(s.frames) == 0 {
		t.Error("frames should not be empty when empty slice provided")
	}
}
