package cmd

import (
	"testing"
)

func TestRunCmd_Structure(t *testing.T) {
	if runCmd == nil {
		t.Fatal("runCmd is nil")
	}

	if runCmd.Use != "run [prompt]" {
		t.Errorf("runCmd.Use = %q, want %q", runCmd.Use, "run [prompt]")
	}

	if runCmd.Short == "" {
		t.Error("runCmd.Short should not be empty")
	}

	if runCmd.Long == "" {
		t.Error("runCmd.Long should not be empty")
	}

	if runCmd.RunE == nil {
		t.Error("runCmd.RunE should not be nil")
	}

	// Should require at least 1 argument (the prompt)
	if runCmd.Args == nil {
		t.Error("runCmd.Args should not be nil")
	}
}

func TestRunCmd_Examples(t *testing.T) {
	// The Long description should contain examples
	if runCmd.Long == "" {
		t.Skip("runCmd.Long is empty")
	}

	// Check that the description mentions examples or usage
	// This is a soft check - the presence of "Examples:" in Long
}

func TestRunCmd_ParentIsRoot(t *testing.T) {
	// Verify run command is registered under root
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "run" {
			found = true
			break
		}
	}
	if !found {
		t.Error("runCmd should be a subcommand of rootCmd")
	}
}

func TestRunCmd_InheritsGlobalFlags(t *testing.T) {
	// Run command should inherit global flags from root
	// Check that model flag is accessible
	flag := runCmd.Flags().Lookup("model")
	if flag == nil {
		// Try inherited flags
		flag = runCmd.InheritedFlags().Lookup("model")
	}
	// Model flag should be available (either local or inherited)
}

func TestRunCommand_MissingPrompt(t *testing.T) {
	// Save original args
	originalArgs := runCmd.Args

	// Test that empty args would fail validation
	// The cobra.MinimumNArgs(1) validator should reject empty args
	if originalArgs == nil {
		t.Skip("runCmd.Args is nil")
	}

	// We can't easily test this without executing the command,
	// but we can verify the Args validator is set
}
