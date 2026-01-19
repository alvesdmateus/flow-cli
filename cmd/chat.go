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
	flowcontext "github.com/mateus/flow-cli/internal/context"
	"github.com/mateus/flow-cli/internal/llm"
	"github.com/mateus/flow-cli/internal/sandbox"
	"github.com/mateus/flow-cli/internal/search"
	"github.com/mateus/flow-cli/internal/tools"
	"github.com/mateus/flow-cli/internal/ui"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with the AI assistant.

The assistant can help you with coding tasks, answer questions,
read and write files, execute commands, and search the web.

Examples:
  flow chat
  flow chat --model llama3:8b`,
	RunE: runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)
}

func runChat(cmd *cobra.Command, args []string) error {
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

	// Check connection
	ui.PrintInfo("Connecting to LLM service...")
	if err := llmClient.Ping(ctx); err != nil {
		return fmt.Errorf("cannot connect to LLM service at %s: %w", config.GetLLMEndpoint(), err)
	}

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
			return fmt.Errorf("no models available. Please pull a model first with: ollama pull <model>")
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

	// Create search client (optional)
	var searchClient search.Client
	if config.IsSearchEnabled() {
		searchClient, _ = search.NewClient(
			config.GetSearchProvider(),
			config.GetSearchEndpoint(),
		)
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

	// Create agent
	chatAgent := agent.New(agent.Config{
		LLMClient:   llmClient,
		ToolReg:     toolReg,
		Model:       model,
		Temperature: 0.7,
		MaxTurns:    10,
	})

	// Load resumed session if specified
	if resumeSessionPath != "" {
		ctxManager := chatAgent.GetContextManager()
		if err := ctxManager.Load(resumeSessionPath); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not load session: %v", err))
		} else {
			ui.PrintSuccess("Session resumed successfully")
		}
		resumeSessionPath = "" // Clear for next time
	}

	// Print welcome message
	printWelcome(model, workDir)

	// Run interactive chat loop
	err = runChatLoop(ctx, chatAgent)

	// Auto-save session on exit
	if chatAgent.GetContextManager().MessageCount() > 0 {
		if saveErr := chatAgent.GetContextManager().SaveToDefaultDir(); saveErr != nil {
			ui.PrintWarning(fmt.Sprintf("Could not save session: %v", saveErr))
		}
	}

	return err
}

func printWelcome(model, workDir string) {
	fmt.Println()
	ui.PrintTitle("╭─────────────────────────────────────────────╮")
	ui.PrintTitle("│           Welcome to flow-cli               │")
	ui.PrintTitle("╰─────────────────────────────────────────────╯")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("  Model:   %s", model))
	ui.PrintInfo(fmt.Sprintf("  Project: %s", workDir))
	fmt.Println()
	fmt.Println("  Type your message and press Enter to send.")
	fmt.Println("  Commands: /help, /clear, /model, /quit")
	fmt.Println()
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
}

func runChatLoop(ctx context.Context, chatAgent *agent.Agent) error {
	reader := bufio.NewReader(os.Stdin)
	handler := &ConsoleHandler{}

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			fmt.Println("\nGoodbye!")
			return nil
		default:
		}

		// Print prompt
		fmt.Print("\033[1;36mYou:\033[0m ")

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil // EOF or error, exit gracefully
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if handleCommand(input, chatAgent) {
				continue
			}
			if input == "/quit" || input == "/exit" {
				fmt.Println("Goodbye!")
				return nil
			}
		}

		// Process message
		fmt.Println()
		fmt.Print("\033[1;35mAssistant:\033[0m ")

		err = chatAgent.ProcessMessage(ctx, input, handler)
		if err != nil {
			if ctx.Err() != nil {
				return nil // Context cancelled
			}
			ui.PrintError(err.Error())
		}

		fmt.Println()
	}
}

func handleCommand(input string, chatAgent *agent.Agent) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return true
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/help", "/h":
		printHelp()
		return true

	case "/clear", "/c":
		chatAgent.Clear()
		// Clear screen
		fmt.Print("\033[2J\033[H")
		ui.PrintSuccess("Conversation cleared")
		fmt.Println()
		return true

	case "/model", "/m":
		return handleModelSwitch(chatAgent)


	case "/status", "/s":
		ctxManager := chatAgent.GetContextManager()
		conv := ctxManager.GetConversation()
		fmt.Printf("\nConversation: %s\n", conv.Title)
		fmt.Printf("Session ID: %s\n", ctxManager.GetID())
		fmt.Printf("Messages: %d\n", ctxManager.MessageCount())
		fmt.Printf("Estimated tokens: ~%d\n\n", ctxManager.EstimateTokens())
		return true

	case "/save":
		ctxManager := chatAgent.GetContextManager()
		if ctxManager.MessageCount() == 0 {
			ui.PrintInfo("No messages to save.")
			return true
		}
		if err := ctxManager.SaveToDefaultDir(); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to save session: %v", err))
		} else {
			ui.PrintSuccess(fmt.Sprintf("Session saved (ID: %s)", ctxManager.GetID()[:8]))
		}
		return true

	case "/sessions":
		sessions, err := flowcontext.ListSessions("")
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to list sessions: %v", err))
			return true
		}
		if len(sessions) == 0 {
			ui.PrintInfo("No saved sessions.")
		} else {
			fmt.Printf("\nSaved sessions (%d):\n", len(sessions))
			for i, s := range sessions {
				if i >= 5 {
					fmt.Printf("  ... and %d more. Use 'flow session list' to see all.\n", len(sessions)-5)
					break
				}
				fmt.Printf("  [%s] %s\n", s.ID[:8], s.Title)
			}
			fmt.Println()
		}
		return true

	case "/quit", "/exit", "/q":
		return false // Signal to exit

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Type /help for available commands")
		return true
	}
}

func printHelp() {
	help := `
╭─────────────────────────────────────────────────────────╮
│                    flow-cli Help                        │
├─────────────────────────────────────────────────────────┤
│ Commands:                                               │
│   /help, /h     - Show this help message                │
│   /clear, /c    - Clear conversation history            │
│   /model, /m    - Switch to a different model           │
│   /status, /s   - Show conversation status              │
│   /save         - Save current session                  │
│   /sessions     - List saved sessions                   │
│   /quit, /q     - Exit the chat (auto-saves)            │
│                                                         │
│ Session Management:                                     │
│   Sessions are auto-saved when you exit.                │
│   Use 'flow session list' to see all sessions.          │
│   Use 'flow session resume' to continue a session.      │
│                                                         │
│ Tips:                                                   │
│   • Be specific about what you want to accomplish       │
│   • The AI will ask for permission before changes       │
│   • Use /clear to start a fresh conversation            │
│                                                         │
│ Examples:                                               │
│   "Read the main.go file and explain what it does"      │
│   "Create a new file called utils.go with a helper"     │
│   "Run the tests and fix any failures"                  │
│   "Search the web for Go error handling best practices" │
╰─────────────────────────────────────────────────────────╯
`
	fmt.Println(help)
}

// handleModelSwitch allows switching models during chat
func handleModelSwitch(chatAgent *agent.Agent) bool {
	ctx := context.Background()
	llmClient := chatAgent.GetLLMClient()

	// Get available models
	models, err := llmClient.ListModels(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to list models: %v", err))
		return true
	}

	if len(models) == 0 {
		ui.PrintInfo("No models available.")
		return true
	}

	// Get current model for display
	currentModel := chatAgent.GetContextManager().GetModel()

	// Build model list with current indicator
	modelNames := make([]string, len(models))
	for i, m := range models {
		if m.Name == currentModel {
			modelNames[i] = m.Name + " (current)"
		} else {
			modelNames[i] = m.Name
		}
	}

	fmt.Println()
	selected, err := ui.SelectModel(modelNames)
	if err != nil {
		ui.PrintInfo("Model selection cancelled.")
		return true
	}

	// Remove " (current)" suffix if present
	selected = strings.TrimSuffix(selected, " (current)")

	if selected == currentModel {
		ui.PrintInfo("Already using this model.")
		return true
	}

	// Switch the model
	chatAgent.SetModel(selected)
	ui.PrintSuccess(fmt.Sprintf("Switched to model: %s", selected))
	fmt.Println()
	return true
}

// ConsoleHandler implements agent.ResponseHandler for console output
type ConsoleHandler struct{}

func (h *ConsoleHandler) OnStreamStart() {}

func (h *ConsoleHandler) OnStreamChunk(chunk string) {
	fmt.Print(chunk)
}

func (h *ConsoleHandler) OnStreamEnd() {
	fmt.Println()
}

func (h *ConsoleHandler) OnToolStart(name, desc string) {
	fmt.Printf("\n\033[1;33m[🔧 %s]\033[0m %s\n", name, desc)
}

func (h *ConsoleHandler) OnToolEnd(name string, success bool, result string) {
	if success {
		fmt.Printf("\033[1;32m[✓ %s]\033[0m\n", name)
	} else {
		fmt.Printf("\033[1;31m[✗ %s]\033[0m %s\n", name, result)
	}
}

func (h *ConsoleHandler) OnThinking(msg string) {
	fmt.Printf("\033[90m⏳ %s\033[0m\n", msg)
}

func (h *ConsoleHandler) OnError(err error) {
	fmt.Printf("\033[1;31mError: %s\033[0m\n", err.Error())
}
