package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mateus/vibe-cli/internal/config"
	"github.com/mateus/vibe-cli/internal/logging"
)

var (
	cfgFile     string
	autoApprove bool
	verbose     bool
	debug       bool
	modelFlag   string
)

var rootCmd = &cobra.Command{
	Use:   "vibe",
	Short: "A self-hosted AI coding assistant",
	Long: `vibe-cli is a coding assistant CLI that uses self-hosted LLMs
to suggest, architect, plan, and create artifacts for your projects.

It operates with a permission-first model, always asking for approval
before making changes unless explicitly configured otherwise.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Setup logging based on flags
		if debug {
			logging.SetLevel(logging.LevelTrace)
			logging.Debug("Debug logging enabled")
		} else if verbose {
			logging.SetLevel(logging.LevelDebug)
			logging.Debug("Verbose logging enabled")
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vibe.yaml)")
	rootCmd.PersistentFlags().BoolVar(&autoApprove, "auto-approve", false, "automatically approve all actions without prompting")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output (more detailed than verbose)")
	rootCmd.PersistentFlags().StringVarP(&modelFlag, "model", "m", "", "model to use for LLM requests")

	_ = viper.BindPFlag("security.auto_approve", rootCmd.PersistentFlags().Lookup("auto-approve"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error getting home directory:", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".vibe")
	}

	viper.SetEnvPrefix("VIBE")
	viper.AutomaticEnv()

	config.SetDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Error reading config file:", err)
		}
	}
}
