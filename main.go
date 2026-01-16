package main

import (
	"os"

	"github.com/mateus/vibe-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
