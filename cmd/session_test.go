package cmd

import (
	"testing"
	"time"
)

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{
			name:     "just now",
			time:     time.Now().Add(-30 * time.Second),
			contains: "just now",
		},
		{
			name:     "1 minute ago",
			time:     time.Now().Add(-1 * time.Minute),
			contains: "1 minute ago",
		},
		{
			name:     "multiple minutes ago",
			time:     time.Now().Add(-5 * time.Minute),
			contains: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     time.Now().Add(-1 * time.Hour),
			contains: "1 hour ago",
		},
		{
			name:     "multiple hours ago",
			time:     time.Now().Add(-3 * time.Hour),
			contains: "3 hours ago",
		},
		{
			name:     "yesterday",
			time:     time.Now().Add(-25 * time.Hour),
			contains: "yesterday",
		},
		{
			name:     "multiple days ago",
			time:     time.Now().Add(-3 * 24 * time.Hour),
			contains: "3 days ago",
		},
		{
			name:     "more than a week",
			time:     time.Now().Add(-10 * 24 * time.Hour),
			contains: "2006", // Should show date format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if tt.contains == "2006" {
				// For dates, just check it's not one of the relative formats
				if result == "just now" || result == "yesterday" {
					t.Errorf("formatTimeAgo() = %q, expected a date format", result)
				}
			} else {
				if result != tt.contains {
					t.Errorf("formatTimeAgo() = %q, want %q", result, tt.contains)
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "no truncation needed",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			input:  "exact",
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "needs truncation",
			input:  "this is a long string",
			maxLen: 10,
			want:   "this is...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "very short maxLen",
			input:  "hello world",
			maxLen: 5,
			want:   "he...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSessionCmd_Structure(t *testing.T) {
	if sessionCmd == nil {
		t.Fatal("sessionCmd is nil")
	}

	if sessionCmd.Use != "session" {
		t.Errorf("sessionCmd.Use = %q, want %q", sessionCmd.Use, "session")
	}

	// Check subcommands exist
	subcommands := sessionCmd.Commands()
	expectedSubs := []string{"list", "resume", "delete", "clear"}

	for _, expected := range expectedSubs {
		found := false
		for _, sub := range subcommands {
			if sub.Use == expected || sub.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sessionCmd missing subcommand %q", expected)
		}
	}
}

func TestSessionListCmd_Structure(t *testing.T) {
	if sessionListCmd == nil {
		t.Fatal("sessionListCmd is nil")
	}

	if sessionListCmd.Use != "list" {
		t.Errorf("sessionListCmd.Use = %q, want %q", sessionListCmd.Use, "list")
	}

	if sessionListCmd.RunE == nil {
		t.Error("sessionListCmd.RunE should not be nil")
	}
}

func TestSessionResumeCmd_Structure(t *testing.T) {
	if sessionResumeCmd == nil {
		t.Fatal("sessionResumeCmd is nil")
	}

	if sessionResumeCmd.Use != "resume [session-id]" {
		t.Errorf("sessionResumeCmd.Use = %q, want %q", sessionResumeCmd.Use, "resume [session-id]")
	}
}

func TestSessionDeleteCmd_Structure(t *testing.T) {
	if sessionDeleteCmd == nil {
		t.Fatal("sessionDeleteCmd is nil")
	}

	if sessionDeleteCmd.Use != "delete <session-id>" {
		t.Errorf("sessionDeleteCmd.Use = %q, want %q", sessionDeleteCmd.Use, "delete <session-id>")
	}

	// Should require exactly 1 argument
	if sessionDeleteCmd.Args == nil {
		t.Error("sessionDeleteCmd.Args should not be nil")
	}
}

func TestSessionClearCmd_Structure(t *testing.T) {
	if sessionClearCmd == nil {
		t.Fatal("sessionClearCmd is nil")
	}

	if sessionClearCmd.Use != "clear" {
		t.Errorf("sessionClearCmd.Use = %q, want %q", sessionClearCmd.Use, "clear")
	}
}
