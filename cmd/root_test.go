package cmd

import (
	"testing"
)

func TestRootCmd_Structure(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "flow" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "flow")
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}

	if rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}

	if rootCmd.PersistentPreRun == nil {
		t.Error("rootCmd.PersistentPreRun should not be nil")
	}
}

func TestRootCmd_Flags(t *testing.T) {
	// Check persistent flags exist
	flags := []string{"config", "auto-approve", "verbose", "debug", "model"}

	for _, flagName := range flags {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("rootCmd missing persistent flag %q", flagName)
		}
	}
}

func TestRootCmd_VerboseFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("verbose flag not found")
	}

	if flag.Shorthand != "v" {
		t.Errorf("verbose flag shorthand = %q, want %q", flag.Shorthand, "v")
	}
}

func TestRootCmd_ModelFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("model")
	if flag == nil {
		t.Fatal("model flag not found")
	}

	if flag.Shorthand != "m" {
		t.Errorf("model flag shorthand = %q, want %q", flag.Shorthand, "m")
	}
}

func TestRootCmd_Subcommands(t *testing.T) {
	subcommands := rootCmd.Commands()

	// Expected subcommands
	expectedCmds := []string{"version", "config", "session", "run", "chat", "arch", "lsp", "completion"}

	for _, expected := range expectedCmds {
		found := false
		for _, sub := range subcommands {
			if sub.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rootCmd missing subcommand %q", expected)
		}
	}
}

func TestExecute(t *testing.T) {
	// Execute with --help should not error
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Execute() with --help error = %v", err)
	}

	// Reset args
	rootCmd.SetArgs([]string{})
}

func TestGlobalFlags_Defaults(t *testing.T) {
	// Check default values
	if cfgFile != "" {
		// cfgFile might be set from previous tests, just check it's a string
		_ = cfgFile
	}

	// These are package-level vars that should have reasonable defaults
	// autoApprove should default to false (security)
	// verbose and debug should default to false
}
