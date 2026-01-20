package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information - set by build flags
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information about flow-cli.`,
	Run:   runVersion,
}

var shortVersion bool

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&shortVersion, "short", "s", false, "Print only the version number")
}

func runVersion(cmd *cobra.Command, args []string) {
	if shortVersion {
		fmt.Println(Version)
		return
	}

	fmt.Printf("flow-cli %s\n", Version)
	fmt.Println()
	fmt.Printf("  Commit:     %s\n", Commit)
	fmt.Printf("  Built:      %s\n", BuildDate)
	fmt.Printf("  Go version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// GetVersionInfo returns full version info as a formatted string
func GetVersionInfo() string {
	return fmt.Sprintf("flow-cli %s (%s) built %s", Version, Commit[:min(7, len(Commit))], BuildDate)
}
