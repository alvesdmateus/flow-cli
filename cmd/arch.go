package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateus/flow-cli/internal/agent"
	"github.com/mateus/flow-cli/internal/config"
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/sandbox"
	"github.com/mateus/flow-cli/internal/search"
	"github.com/mateus/flow-cli/internal/tools"
	"github.com/mateus/flow-cli/internal/ui"
)

var archCmd = &cobra.Command{
	Use:   "arch [description]",
	Short: "Enter architecture mode for planning",
	Long: `Enter architecture mode to plan an implementation before coding.

In this mode, the AI will:
1. Ask clarifying questions to understand your requirements
2. Analyze the current codebase structure
3. Create a detailed implementation plan
4. Wait for your approval before making changes

Examples:
  flow arch "Add user authentication"
  flow arch "Refactor the database layer"
  flow arch "Create a REST API for products"`,
	RunE: runArch,
}

func init() {
	rootCmd.AddCommand(archCmd)
}

func runArch(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create LLM client
	llmClient, err := llm.NewClient(
		config.GetLLMProvider(),
		config.GetLLMEndpoint(),
	)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Check connection with spinner
	spinner := ui.SpinnerConnecting(config.GetLLMEndpoint())
	spinner.Start()
	if err := llmClient.Ping(ctx); err != nil {
		spinner.StopWithError("Connection failed")
		return fmt.Errorf("cannot connect to LLM service at %s: %w", config.GetLLMEndpoint(), err)
	}
	spinner.StopWithSuccess("Connected")

	// Determine model
	model := modelFlag
	if model == "" {
		model = config.GetLLMModel()
	}

	// If still no model, prompt user to select one
	if model == "" {
		models, err := llmClient.ListModels(ctx)
		if err != nil {
			return fmt.Errorf("failed to list models: %w", err)
		}

		if len(models) == 0 {
			return fmt.Errorf("no models available")
		}

		modelNames := make([]string, len(models))
		for i, m := range models {
			modelNames[i] = m.Name
		}

		selected, err := ui.SelectModel(modelNames)
		if err != nil {
			return fmt.Errorf("model selection cancelled: %w", err)
		}
		model = selected
	}

	// Print welcome
	ui.PrintWelcomeArch()

	// Get initial request
	var request string
	if len(args) > 0 {
		request = strings.Join(args, " ")
	} else {
		fmt.Println("What would you like to build or implement?")
		fmt.Println()
		fmt.Print("📝 Your request: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		request = strings.TrimSpace(input)
	}

	if request == "" {
		return fmt.Errorf("no request provided")
	}

	// Create planner
	planner := agent.NewPlanner(agent.PlannerConfig{
		LLMClient: llmClient,
		Model:     model,
	})

	// Create handler
	handler := &ArchHandler{}

	// Start planning
	ui.PrintInfo("Starting architecture planning...")
	fmt.Println()

	plan, err := planner.StartPlanning(ctx, request, handler)
	if err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}

	// Plan review loop - allows modifications
	for {
		// Display the plan
		displayPlan(plan)

		// Ask for approval
		result, feedback, err := askPlanApproval()
		if err != nil {
			return err
		}

		switch result {
		case planApproved:
			// Plan approved - transition to execution
			planner.ApprovePlan()
			ui.PrintSuccess("Plan approved!")
			fmt.Println()
			goto executeplan

		case planNeedsModification:
			if feedback == "" {
				// No feedback, show plan again
				continue
			}
			// Modify the plan based on feedback
			plan, err = planner.ModifyPlan(ctx, feedback, handler)
			if err != nil {
				return fmt.Errorf("failed to modify plan: %w", err)
			}
			fmt.Println()
			ui.PrintSuccess("Plan has been revised based on your feedback.")
			fmt.Println()
			continue

		case planCancelled:
			ui.PrintInfo("Planning cancelled.")
			return nil
		}
	}

executeplan:

	// Ask if user wants to proceed to chat for execution
	proceed, err := ui.Confirm("Would you like to start implementing the plan now?")
	if err != nil {
		return err
	}

	if proceed {
		return runPlanExecution(ctx, llmClient, model, plan)
	}

	ui.PrintInfo("Plan saved. Run 'flow chat' to continue implementation later.")
	return nil
}

func displayPlan(plan *agent.Plan) {
	// Convert to display format
	display := ui.PlanDisplay{
		Title:       plan.Title,
		Summary:     plan.Summary,
		Goals:       plan.Goals,
		Constraints: plan.Constraints,
		Phase:       plan.Phase.String(),
		Tasks:       make([]ui.TaskDisplay, len(plan.Tasks)),
	}

	for i, task := range plan.Tasks {
		display.Tasks[i] = ui.TaskDisplay{
			ID:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			Priority:    task.Priority,
			Files:       task.Files,
			Completed:   task.Completed,
		}
	}

	fmt.Println(ui.FormatPlan(display))

	// Print summary
	fileCount := 0
	for _, task := range plan.Tasks {
		fileCount += len(task.Files)
	}
	ui.PrintPlanSummary(len(plan.Tasks), fileCount)
}

// planApprovalResult represents the result of asking for plan approval
type planApprovalResult int

const (
	planApproved planApprovalResult = iota
	planNeedsModification
	planCancelled
)

func askPlanApproval() (planApprovalResult, string, error) {
	ui.PrintPlanApprovalPrompt()

	choice, err := ui.AskQuestion("What would you like to do with this plan?", []string{
		"Approve and proceed",
		"Request modifications",
		"Cancel planning",
	})
	if err != nil {
		return planCancelled, "", err
	}

	switch choice {
	case "Approve and proceed":
		return planApproved, "", nil
	case "Request modifications":
		fmt.Println()
		feedback, err := ui.PromptInput("What changes would you like to make to the plan?")
		if err != nil {
			return planCancelled, "", err
		}
		if strings.TrimSpace(feedback) == "" {
			ui.PrintInfo("No feedback provided. Keeping the current plan.")
			return planNeedsModification, "", nil
		}
		return planNeedsModification, feedback, nil
	default:
		return planCancelled, "", nil
	}
}

func runPlanExecution(ctx context.Context, llmClient llm.Client, model string, plan *agent.Plan) error {
	ui.PrintInfo("Transitioning to implementation mode...")
	fmt.Println()

	// Create search client (optional)
	var searchClient search.Client
	if config.IsSearchEnabled() {
		var searchErr error
		searchClient, searchErr = search.NewClient(
			config.GetSearchProvider(),
			config.GetSearchEndpoint(),
		)
		if searchErr != nil {
			ui.PrintWarning(fmt.Sprintf("Search disabled: %v", searchErr))
		}
	}

	// Create security policy
	policy := sandbox.DefaultPolicy()
	policy.AutoApprove = config.IsAutoApprove() || autoApprove

	// Create permission manager
	permissions, err := sandbox.NewManager(policy, ui.RequestApproval)
	if err != nil {
		return fmt.Errorf("failed to create permission manager: %w", err)
	}

	// Create tool registry
	workDir, _ := os.Getwd()
	toolReg, err := tools.SetupRegistry(tools.SetupOptions{
		Permissions:  permissions,
		SearchClient: searchClient,
		WorkDir:      workDir,
	})
	if err != nil {
		return fmt.Errorf("failed to setup tools: %w", err)
	}

	// Create agent with plan context
	systemPrompt := fmt.Sprintf(`You are implementing an approved plan.

Plan Title: %s
Summary: %s

Tasks to complete:
%s

Work through each task in order, asking for approval before making changes.
After completing each task, mark it as done and move to the next.
Be thorough but efficient.`, plan.Title, plan.Summary, formatTaskList(plan.Tasks))

	chatAgent := agent.New(agent.Config{
		LLMClient:    llmClient,
		ToolReg:      toolReg,
		Model:        model,
		Temperature:  0.7,
		MaxTurns:     10,
		SystemPrompt: systemPrompt,
	})

	fmt.Println("Starting implementation. Type your instructions or press Enter to let the AI proceed.")
	fmt.Println()

	// Run chat loop for execution
	return runChatLoop(ctx, chatAgent)
}

func formatTaskList(tasks []agent.PlanTask) string {
	var b strings.Builder
	for i, task := range tasks {
		status := "[ ]"
		if task.Completed {
			status = "[x]"
		}
		b.WriteString(fmt.Sprintf("%d. %s %s", i+1, status, task.Title))
		if task.Description != "" {
			b.WriteString(fmt.Sprintf("\n   %s", task.Description))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ArchHandler implements agent.PlannerHandler
type ArchHandler struct {
	spinner *ui.Spinner
}

func (h *ArchHandler) OnPhaseChange(phase agent.PlanPhase) {
	// Stop any running spinner
	if h.spinner != nil {
		h.spinner.Stop()
	}
	ui.PrintPhaseChange(phase.String())

	// Start a new spinner for the phase
	h.spinner = ui.SpinnerProcessing(phase.String())
	h.spinner.Start()
}

func (h *ArchHandler) OnQuestion(question agent.ClarifyingQuestion) (string, error) {
	// Stop spinner for user interaction
	if h.spinner != nil {
		h.spinner.Stop()
		h.spinner = nil
	}

	// Display question
	display := ui.QuestionDisplay{
		Question: question.Question,
		Context:  question.Context,
		Options:  question.Options,
	}
	fmt.Println(ui.FormatQuestion(display))

	// Get answer using multiple choice
	if len(question.Options) > 0 {
		answer, err := ui.AskQuestion("Select an option:", question.Options)
		if err != nil {
			return "", err
		}
		// Restart spinner after user answers
		h.spinner = ui.SpinnerThinking()
		h.spinner.Start()
		return answer, nil
	}

	// Free-form input
	fmt.Print("Your answer: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Restart spinner after user answers
	h.spinner = ui.SpinnerThinking()
	h.spinner.Start()
	return strings.TrimSpace(input), nil
}

func (h *ArchHandler) OnPlanUpdate(plan *agent.Plan) {
	// Stop spinner when plan is ready to display
	if h.spinner != nil {
		h.spinner.Stop()
		h.spinner = nil
	}
}

func (h *ArchHandler) OnTaskStart(task agent.PlanTask) {
	// Stop any running spinner
	if h.spinner != nil {
		h.spinner.Stop()
	}
	fmt.Printf("\n🔧 Starting: %s\n", task.Title)
	h.spinner = ui.SpinnerProcessing(task.Title)
	h.spinner.Start()
}

func (h *ArchHandler) OnTaskComplete(task agent.PlanTask, success bool) {
	if h.spinner != nil {
		h.spinner.Stop()
		h.spinner = nil
	}
	if success {
		fmt.Printf("✅ Completed: %s\n", task.Title)
	} else {
		fmt.Printf("❌ Failed: %s\n", task.Title)
	}
}

func (h *ArchHandler) OnMessage(message string) {
	// Update spinner message if running, otherwise print
	if h.spinner != nil {
		h.spinner.UpdateMessage(message)
	} else {
		ui.PrintInfo(message)
	}
}

func (h *ArchHandler) OnError(err error) {
	if err == nil {
		return
	}
	if h.spinner != nil {
		h.spinner.StopWithError(err.Error())
		h.spinner = nil
	} else {
		ui.PrintError(err.Error())
	}
}
