package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for vibe.

To load completions:

Bash:
  $ source <(vibe completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ vibe completion bash > /etc/bash_completion.d/vibe
  # macOS:
  $ vibe completion bash > $(brew --prefix)/etc/bash_completion.d/vibe

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ vibe completion zsh > "${fpath[1]}/_vibe"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ vibe completion fish | source

  # To load completions for each session, execute once:
  $ vibe completion fish > ~/.config/fish/completions/vibe.fish

PowerShell:
  PS> vibe completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> vibe completion powershell > vibe.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
