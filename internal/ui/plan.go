package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	planTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	planSectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginTop(1)

	planItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	taskBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginTop(1)

	priorityCriticalStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("196")).
				Foreground(lipgloss.Color("255")).
				Padding(0, 1).
				Bold(true)

	priorityHighStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("208")).
				Foreground(lipgloss.Color("255")).
				Padding(0, 1)

	priorityMediumStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("220")).
				Foreground(lipgloss.Color("0")).
				Padding(0, 1)

	priorityLowStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("245")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)

	phaseIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	checkboxEmpty    = "☐"
	checkboxComplete = "☑"
)

// PlanDisplay represents a displayable plan
type PlanDisplay struct {
	Title       string
	Summary     string
	Goals       []string
	Constraints []string
	Tasks       []TaskDisplay
	Phase       string
}

// TaskDisplay represents a displayable task
type TaskDisplay struct {
	ID          string
	Title       string
	Description string
	Priority    string
	Files       []string
	Completed   bool
}

// FormatPlan formats a plan for terminal display
func FormatPlan(plan PlanDisplay) string {
	var b strings.Builder

	// Header box
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(60)

	header := planTitleStyle.Render("📋 " + plan.Title)
	if plan.Summary != "" {
		header += "\n\n" + planItemStyle.Render(plan.Summary)
	}
	header += "\n\n" + phaseIndicatorStyle.Render("Phase: "+plan.Phase)

	b.WriteString(headerBox.Render(header))
	b.WriteString("\n\n")

	// Goals section
	if len(plan.Goals) > 0 {
		b.WriteString(planSectionStyle.Render("🎯 Goals"))
		b.WriteString("\n")
		for _, goal := range plan.Goals {
			b.WriteString(planItemStyle.Render("  • " + goal))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Constraints section
	if len(plan.Constraints) > 0 {
		b.WriteString(planSectionStyle.Render("⚠️ Constraints"))
		b.WriteString("\n")
		for _, constraint := range plan.Constraints {
			b.WriteString(planItemStyle.Render("  • " + constraint))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Tasks section
	if len(plan.Tasks) > 0 {
		b.WriteString(planSectionStyle.Render("📝 Implementation Tasks"))
		b.WriteString("\n")

		for i, task := range plan.Tasks {
			taskContent := formatTask(i+1, task)
			b.WriteString(taskBoxStyle.Render(taskContent))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatTask formats a single task
func formatTask(num int, task TaskDisplay) string {
	var b strings.Builder

	// Checkbox and number
	checkbox := checkboxEmpty
	if task.Completed {
		checkbox = checkboxComplete
	}

	// Priority badge
	priorityBadge := formatPriority(task.Priority)

	// Title line
	b.WriteString(fmt.Sprintf("%s %d. %s  %s\n",
		checkbox,
		num,
		titleStyle.Render(task.Title),
		priorityBadge,
	))

	// Description
	if task.Description != "" {
		b.WriteString(dimStyle.Render("   " + task.Description))
		b.WriteString("\n")
	}

	// Files
	if len(task.Files) > 0 {
		b.WriteString(dimStyle.Render("   Files: " + strings.Join(task.Files, ", ")))
		b.WriteString("\n")
	}

	return b.String()
}

// formatPriority returns a styled priority badge
func formatPriority(priority string) string {
	switch strings.ToLower(priority) {
	case "critical":
		return priorityCriticalStyle.Render("CRITICAL")
	case "high":
		return priorityHighStyle.Render("HIGH")
	case "medium":
		return priorityMediumStyle.Render("MEDIUM")
	case "low":
		return priorityLowStyle.Render("LOW")
	default:
		return priorityMediumStyle.Render("MEDIUM")
	}
}

// PrintPhaseChange prints a phase change notification
func PrintPhaseChange(phase string) {
	phaseBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 2)

	fmt.Println()
	fmt.Println(phaseBox.Render(phaseIndicatorStyle.Render("→ " + phase)))
	fmt.Println()
}

// PrintPlanApprovalPrompt prints the plan approval prompt
func PrintPlanApprovalPrompt() {
	fmt.Println()
	fmt.Println(planSectionStyle.Render("What would you like to do?"))
	fmt.Println()
}

// QuestionDisplay formats a clarifying question
type QuestionDisplay struct {
	Question string
	Context  string
	Options  []string
}

// FormatQuestion formats a question for display
func FormatQuestion(q QuestionDisplay) string {
	var b strings.Builder

	questionBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2)

	content := titleStyle.Render("❓ " + q.Question)
	if q.Context != "" {
		content += "\n\n" + dimStyle.Render(q.Context)
	}

	b.WriteString(questionBox.Render(content))

	return b.String()
}

// PrintWelcomeArch prints the architecture mode welcome
func PrintWelcomeArch() {
	welcomeBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(60)

	content := `🏗️  Architecture Mode

In this mode, I'll help you plan before you build:

1. 📝 Gather requirements through questions
2. 🔍 Analyze the current codebase
3. 📋 Create a detailed implementation plan
4. ✅ Get your approval before any changes

This ensures we build the right thing, the right way.`

	fmt.Println()
	fmt.Println(welcomeBox.Render(content))
	fmt.Println()
}

// PrintPlanSummary prints a brief plan summary
func PrintPlanSummary(taskCount int, estimatedFiles int) {
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("46")).
		Padding(0, 2)

	content := fmt.Sprintf("📊 Plan Summary: %d tasks, ~%d files affected",
		taskCount, estimatedFiles)

	fmt.Println(summaryBox.Render(successStyle.Render(content)))
	fmt.Println()
}
