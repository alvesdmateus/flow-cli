package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mateus/flow-cli/internal/templates"
	"github.com/mateus/flow-cli/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new project or configure flow for an existing project",
	Long: `Initialize a new project with scaffolding or configure flow for an existing project.

When run without arguments in an empty directory, it will prompt you to select
a project template and create the appropriate structure.

When run in an existing project, it will create .flow/config.yaml for
project-level flow configuration.

Examples:
  flow init                    # Interactive initialization in current directory
  flow init my-project         # Create new project in my-project directory
  flow init --template go      # Use Go template
  flow init --list-templates   # List available templates
  flow init --existing         # Configure flow for existing project only`,
	RunE: runInit,
}

var (
	templateName   string
	listTemplates  bool
	existingOnly   bool
	skipPrompts    bool
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&templateName, "template", "t", "", "Project template to use")
	initCmd.Flags().BoolVarP(&listTemplates, "list-templates", "l", false, "List available templates")
	initCmd.Flags().BoolVarP(&existingOnly, "existing", "e", false, "Only create flow config for existing project")
	initCmd.Flags().BoolVarP(&skipPrompts, "yes", "y", false, "Skip confirmation prompts and use defaults")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Handle --list-templates
	if listTemplates {
		return listAvailableTemplates()
	}

	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Get absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if directory exists
	dirExists := true
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		dirExists = false
	}

	// If --existing flag, just create flow config
	if existingOnly {
		if !dirExists {
			return fmt.Errorf("directory does not exist: %s", absPath)
		}
		return initFlowConfig(absPath)
	}

	// If directory doesn't exist, create it
	if !dirExists {
		if !skipPrompts {
			confirmed, err := ui.Confirm(fmt.Sprintf("Create new project at %s?", absPath))
			if err != nil {
				return err
			}
			if !confirmed {
				ui.PrintInfo("Initialization cancelled.")
				return nil
			}
		}

		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Check if directory is empty (excluding hidden files)
	isEmpty, err := isDirEmpty(absPath)
	if err != nil {
		return fmt.Errorf("failed to check directory: %w", err)
	}

	// Determine template to use
	tmpl, err := selectTemplate(isEmpty)
	if err != nil {
		return err
	}

	// Gather project info
	info, err := gatherProjectInfo(absPath, tmpl)
	if err != nil {
		return err
	}

	// Preview and confirm
	if !skipPrompts && tmpl.Name != "empty" {
		if err := previewAndConfirm(info, tmpl); err != nil {
			return err
		}
	}

	// Scaffold the project
	ui.PrintInfo("Initializing project...")
	fmt.Println()

	scaffolder := templates.NewScaffolder(absPath, info)
	files, err := scaffolder.Scaffold(tmpl)
	if err != nil {
		return fmt.Errorf("failed to scaffold project: %w", err)
	}

	// Print created files
	ui.PrintSuccess("Project initialized successfully!")
	fmt.Println()
	fmt.Println("Created files:")
	for _, f := range files {
		relPath, _ := filepath.Rel(absPath, f)
		fmt.Printf("  %s\n", relPath)
	}

	// Print next steps
	fmt.Println()
	printNextSteps(absPath, tmpl, info)

	return nil
}

func listAvailableTemplates() error {
	ui.PrintTitle("Available Templates")
	fmt.Println()

	for _, tmpl := range templates.GetAllTemplates() {
		fmt.Printf("  %s\n", ui.FormatTemplate(tmpl.Name, tmpl.Description))
		if len(tmpl.Features) > 0 {
			for _, f := range tmpl.Features {
				fmt.Printf("    - %s\n", f)
			}
		}
		fmt.Println()
	}

	fmt.Println("Use 'flow init --template <name>' to use a template.")
	return nil
}

func isDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		// Skip hidden files and common non-project files
		name := entry.Name()
		if name[0] == '.' {
			continue
		}
		return false, nil
	}

	return true, nil
}

func selectTemplate(isEmpty bool) (*templates.Template, error) {
	// If template specified via flag
	if templateName != "" {
		tmpl := templates.GetTemplate(templateName)
		if tmpl == nil {
			return nil, fmt.Errorf("unknown template: %s\nRun 'flow init --list-templates' to see available templates", templateName)
		}
		return tmpl, nil
	}

	// If using --yes flag, use empty template for non-empty dirs
	if skipPrompts {
		if isEmpty {
			return templates.GetTemplate("go"), nil // Default to Go for empty dirs
		}
		return templates.GetTemplate("empty"), nil
	}

	// Interactive selection
	if !isEmpty {
		ui.PrintInfo("Directory is not empty. Using minimal initialization.")
		return templates.GetTemplate("empty"), nil
	}

	// Show template selection
	allTemplates := templates.GetAllTemplates()
	options := make([]string, len(allTemplates))
	for i, t := range allTemplates {
		options[i] = fmt.Sprintf("%s - %s", t.Name, t.Description)
	}

	choice, err := ui.AskQuestion("Select a project template:", options)
	if err != nil {
		return nil, err
	}

	// Parse selection
	for _, t := range allTemplates {
		if fmt.Sprintf("%s - %s", t.Name, t.Description) == choice {
			return &t, nil
		}
	}

	return templates.GetTemplate("empty"), nil
}

func gatherProjectInfo(targetDir string, tmpl *templates.Template) (*templates.ProjectInfo, error) {
	info := &templates.ProjectInfo{
		Path: targetDir,
	}

	// Default project name from directory
	info.Name = filepath.Base(targetDir)

	if skipPrompts {
		info.Description = fmt.Sprintf("A %s project created with flow-cli", tmpl.Language)
		info.Author = ""
		return info, nil
	}

	// Prompt for project name
	name, err := ui.PromptInputWithDefault("Project name:", info.Name)
	if err != nil {
		return nil, err
	}
	info.Name = name

	// Prompt for description
	desc, err := ui.PromptInputWithDefault("Description:", "")
	if err != nil {
		return nil, err
	}
	info.Description = desc

	// Prompt for author (optional)
	author, err := ui.PromptInputWithDefault("Author:", "")
	if err != nil {
		return nil, err
	}
	info.Author = author

	return info, nil
}

func previewAndConfirm(info *templates.ProjectInfo, tmpl *templates.Template) error {
	fmt.Println()
	ui.PrintTitle("Project Summary")
	fmt.Println()
	fmt.Printf("  Name:        %s\n", info.Name)
	fmt.Printf("  Path:        %s\n", info.Path)
	fmt.Printf("  Template:    %s\n", tmpl.Name)
	if info.Description != "" {
		fmt.Printf("  Description: %s\n", info.Description)
	}
	if info.Author != "" {
		fmt.Printf("  Author:      %s\n", info.Author)
	}
	fmt.Println()

	fmt.Println("Files to be created:")
	for _, f := range tmpl.Files {
		fmt.Printf("  %s\n", f.Path)
	}
	fmt.Println()

	confirmed, err := ui.Confirm("Proceed with initialization?")
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("initialization cancelled")
	}

	return nil
}

func initFlowConfig(targetDir string) error {
	ui.PrintInfo("Initializing flow configuration for existing project...")
	fmt.Println()

	info := &templates.ProjectInfo{
		Name: filepath.Base(targetDir),
		Path: targetDir,
	}

	scaffolder := templates.NewScaffolder(targetDir, info)
	files, err := scaffolder.CreateFlowConfig()
	if err != nil {
		return fmt.Errorf("failed to create flow config: %w", err)
	}

	ui.PrintSuccess("Flow configuration created!")
	fmt.Println()
	fmt.Println("Created files:")
	for _, f := range files {
		relPath, _ := filepath.Rel(targetDir, f)
		fmt.Printf("  %s\n", relPath)
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit .flow/config.yaml to customize settings")
	fmt.Println("  2. Edit .flowignore to exclude files from indexing")
	fmt.Println("  3. Run 'flow chat' to start working with flow")

	return nil
}

func printNextSteps(targetDir string, tmpl *templates.Template, info *templates.ProjectInfo) {
	fmt.Println("Next steps:")

	// Change to directory if not current
	if targetDir != "." {
		relPath, err := filepath.Rel(".", targetDir)
		if err == nil && relPath != "." {
			fmt.Printf("  cd %s\n", relPath)
		}
	}

	// Template-specific next steps
	switch tmpl.Name {
	case "go":
		fmt.Println("  go mod tidy")
		fmt.Println("  go build")
	case "python":
		fmt.Println("  python -m venv .venv")
		fmt.Println("  source .venv/bin/activate  # or .venv\\Scripts\\activate on Windows")
		fmt.Println("  pip install -r requirements.txt")
	case "node":
		fmt.Println("  npm install")
		fmt.Println("  npm run dev")
	case "rust":
		fmt.Println("  cargo build")
		fmt.Println("  cargo run")
	}

	// Common next steps
	fmt.Println()
	fmt.Println("  flow chat    # Start chatting with flow")
	fmt.Println("  flow arch    # Plan your implementation")
}
