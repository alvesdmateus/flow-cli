package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/mateus/flow-cli/internal/sandbox"
)

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	riskLowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	riskMediumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	riskHighStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// SelectModel prompts the user to select a model from available options
func SelectModel(models []string) (string, error) {
	if len(models) == 0 {
		return "", fmt.Errorf("no models available")
	}

	if len(models) == 1 {
		return models[0], nil
	}

	options := make([]huh.Option[string], len(models))
	for i, m := range models {
		options[i] = huh.NewOption(m, m)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a model").
				Options(options...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

// Confirm prompts the user for yes/no confirmation
func Confirm(message string) (bool, error) {
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(message).
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	)

	err := form.Run()
	if err != nil {
		return false, err
	}

	return confirmed, nil
}

// MultiSelect prompts the user to select multiple options
func MultiSelect(title string, options []string) ([]string, error) {
	if len(options) == 0 {
		return nil, nil
	}

	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Options(opts...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return selected, nil
}

// AskQuestion presents a question with multiple choice options
func AskQuestion(question string, choices []string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices provided")
	}

	options := make([]huh.Option[string], len(choices))
	for i, c := range choices {
		options[i] = huh.NewOption(c, c)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(question).
				Options(options...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

// PrintTitle prints a styled title
func PrintTitle(text string) {
	fmt.Println(titleStyle.Render(text))
}

// PrintError prints a styled error message
func PrintError(text string) {
	fmt.Println(errorStyle.Render("Error: " + text))
}

// PrintSuccess prints a styled success message
func PrintSuccess(text string) {
	fmt.Println(successStyle.Render(text))
}

// PrintInfo prints a styled info message
func PrintInfo(text string) {
	fmt.Println(infoStyle.Render(text))
}

// PrintWarning prints a styled warning message
func PrintWarning(text string) {
	fmt.Println(warningStyle.Render("Warning: " + text))
}

// RequestApproval prompts the user to approve an operation
func RequestApproval(op *sandbox.Operation) (bool, error) {
	// Build operation details display
	var details strings.Builder

	// Operation type and description
	details.WriteString(fmt.Sprintf("Operation: %s\n", titleStyle.Render(string(op.Type))))
	details.WriteString(fmt.Sprintf("Description: %s\n", op.Description))

	// Target
	if op.Target != "" {
		details.WriteString(fmt.Sprintf("Target: %s\n", infoStyle.Render(op.Target)))
	}

	// Additional details
	if len(op.Details) > 0 {
		details.WriteString("\nDetails:\n")
		for k, v := range op.Details {
			details.WriteString(fmt.Sprintf("  %s: %s\n", dimStyle.Render(k), v))
		}
	}

	// Risk level
	riskLabel := formatRiskLevel(op.Level)
	details.WriteString(fmt.Sprintf("\nRisk Level: %s", riskLabel))

	// Print the operation box
	fmt.Println()
	fmt.Println(boxStyle.Render(details.String()))
	fmt.Println()

	// Prompt for approval
	var approved bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Do you want to allow this operation?").
				Affirmative("Yes, allow").
				Negative("No, deny").
				Value(&approved),
		),
	)

	err := form.Run()
	if err != nil {
		return false, err
	}

	return approved, nil
}

// formatRiskLevel returns a styled risk level string
func formatRiskLevel(level sandbox.PermissionLevel) string {
	switch level {
	case sandbox.PermissionNone:
		return dimStyle.Render("none")
	case sandbox.PermissionLow:
		return riskLowStyle.Render("LOW")
	case sandbox.PermissionMedium:
		return riskMediumStyle.Render("MEDIUM")
	case sandbox.PermissionHigh:
		return riskHighStyle.Render("HIGH")
	case sandbox.PermissionDenied:
		return errorStyle.Render("DENIED")
	default:
		return dimStyle.Render("unknown")
	}
}

// ConfirmWithDetails shows details and asks for confirmation
func ConfirmWithDetails(title string, details map[string]string, message string) (bool, error) {
	var detailsStr strings.Builder

	detailsStr.WriteString(titleStyle.Render(title) + "\n\n")

	for k, v := range details {
		detailsStr.WriteString(fmt.Sprintf("%s: %s\n", dimStyle.Render(k), v))
	}

	fmt.Println()
	fmt.Println(boxStyle.Render(detailsStr.String()))
	fmt.Println()

	return Confirm(message)
}

// PromptInput prompts the user for text input
func PromptInput(prompt string) (string, error) {
	var input string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(prompt).
				Value(&input),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return input, nil
}

// PromptInputWithDefault prompts the user for text input with a default value
func PromptInputWithDefault(prompt, defaultValue string) (string, error) {
	input := defaultValue
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(prompt).
				Value(&input).
				Placeholder(defaultValue),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	if input == "" {
		return defaultValue, nil
	}

	return input, nil
}

// FormatTemplate formats a template name and description for display.
func FormatTemplate(name, description string) string {
	return fmt.Sprintf("%s - %s", titleStyle.Render(name), dimStyle.Render(description))
}

// ChatMode represents the current chat mode
type ChatMode string

const (
	ModeChatNormal ChatMode = "chat"
	ModeArch       ChatMode = "arch"
)

// RenderInputPrompt returns a styled input prompt for the given mode
func RenderInputPrompt(mode ChatMode) string {
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).
		Bold(true)

	return fmt.Sprintf("%s %s ",
		modeStyle.Render(fmt.Sprintf("[%s]", mode)),
		promptStyle.Render(">"))
}

// RenderAssistantHeader returns a styled header for assistant output
func RenderAssistantHeader() string {
	return ""  // No header needed in the new design
}

// RenderSeparator returns a visual separator line
func RenderSeparator() string {
	return dimStyle.Render(strings.Repeat("─", 50))
}

// PrintWelcomeInit prints a welcome message for vibe init.
func PrintWelcomeInit() {
	fmt.Println()
	fmt.Println(titleStyle.Render("Vibe Project Initialization"))
	fmt.Println(dimStyle.Render("Create a new project or configure vibe for an existing one"))
	fmt.Println()
}

// SelectOption prompts the user to select from a list of options
func SelectOption(title string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	if len(options) == 1 {
		return options[0], nil
	}

	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(opts...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}
