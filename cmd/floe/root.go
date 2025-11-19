package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "floe",
	Short: "Floe is a lightweight, agentic workflow engine",
	Long: `Floe is a workflow engine designed for AI agents. 
It supports defining workflows in YAML, executing them with a runtime, 
and visualizing the execution via TUI.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be defined here
}
