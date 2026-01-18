package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/mateus/flow-cli/internal/context"
	"github.com/mateus/flow-cli/internal/ui"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage chat sessions",
	Long: `View, resume, and manage saved chat sessions.

Sessions are automatically saved during chat and can be resumed later.`,
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved sessions",
	RunE:  runSessionList,
}

var sessionResumeCmd = &cobra.Command{
	Use:   "resume [session-id]",
	Short: "Resume a saved session",
	Long: `Resume a previously saved chat session.

If no session ID is provided, shows a selection menu.`,
	RunE: runSessionResume,
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <session-id>",
	Short: "Delete a saved session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionDelete,
}

var sessionClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all saved sessions",
	RunE:  runSessionClear,
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionResumeCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)
	sessionCmd.AddCommand(sessionClearCmd)
}

func runSessionList(cmd *cobra.Command, args []string) error {
	sessions, err := context.ListSessions("")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ui.PrintInfo("No saved sessions found.")
		fmt.Println("Start a chat with 'flow chat' to create a session.")
		return nil
	}

	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	fmt.Println(headerStyle.Render("Saved Sessions"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	for _, s := range sessions {
		// Format time
		timeAgo := formatTimeAgo(s.UpdatedAt)

		// Print session info
		fmt.Printf("%s %s\n",
			idStyle.Render(s.ID[:min(8, len(s.ID))]),
			titleStyle.Render(truncate(s.Title, 45)),
		)
		fmt.Printf("   %s | %s | %d messages\n",
			dimStyle.Render(timeAgo),
			dimStyle.Render(s.Model),
			s.MessageCount,
		)
		fmt.Println()
	}

	fmt.Println(dimStyle.Render(fmt.Sprintf("Total: %d sessions", len(sessions))))
	fmt.Println()
	fmt.Println("Use 'flow session resume <id>' to continue a session.")

	return nil
}

func runSessionResume(cmd *cobra.Command, args []string) error {
	sessions, err := context.ListSessions("")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ui.PrintInfo("No saved sessions found.")
		return nil
	}

	var selectedSession *context.SessionInfo

	if len(args) > 0 {
		// Find by ID prefix
		sessionID := args[0]
		for _, s := range sessions {
			if strings.HasPrefix(s.ID, sessionID) {
				selectedSession = &s
				break
			}
		}
		if selectedSession == nil {
			return fmt.Errorf("session not found: %s", sessionID)
		}
	} else {
		// Show selection menu
		options := make([]string, len(sessions))
		for i, s := range sessions {
			timeAgo := formatTimeAgo(s.UpdatedAt)
			options[i] = fmt.Sprintf("[%s] %s (%s)",
				s.ID[:min(8, len(s.ID))],
				truncate(s.Title, 35),
				timeAgo,
			)
		}

		choice, err := ui.AskQuestion("Select a session to resume:", options)
		if err != nil {
			return err
		}

		// Find the selected session
		for i, opt := range options {
			if opt == choice {
				selectedSession = &sessions[i]
				break
			}
		}
	}

	if selectedSession == nil {
		return fmt.Errorf("no session selected")
	}

	ui.PrintSuccess(fmt.Sprintf("Resuming session: %s", selectedSession.Title))
	fmt.Println()

	// Set global flag to resume this session
	resumeSessionPath = selectedSession.FilePath

	// Run chat with the resumed session
	return runChat(cmd, nil)
}

func runSessionDelete(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	sessions, err := context.ListSessions("")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	var targetSession *context.SessionInfo
	for _, s := range sessions {
		if strings.HasPrefix(s.ID, sessionID) {
			targetSession = &s
			break
		}
	}

	if targetSession == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Confirm deletion
	confirmed, err := ui.Confirm(fmt.Sprintf("Delete session '%s'?", targetSession.Title))
	if err != nil {
		return err
	}

	if !confirmed {
		ui.PrintInfo("Deletion cancelled.")
		return nil
	}

	if err := context.DeleteSession(targetSession.FilePath); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	ui.PrintSuccess("Session deleted.")
	return nil
}

func runSessionClear(cmd *cobra.Command, args []string) error {
	sessions, err := context.ListSessions("")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ui.PrintInfo("No sessions to delete.")
		return nil
	}

	// Confirm deletion
	confirmed, err := ui.Confirm(fmt.Sprintf("Delete all %d sessions? This cannot be undone.", len(sessions)))
	if err != nil {
		return err
	}

	if !confirmed {
		ui.PrintInfo("Deletion cancelled.")
		return nil
	}

	deleted := 0
	for _, s := range sessions {
		if err := context.DeleteSession(s.FilePath); err == nil {
			deleted++
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Deleted %d sessions.", deleted))
	return nil
}

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
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// resumeSessionPath is set by session resume to load a specific session
var resumeSessionPath string
