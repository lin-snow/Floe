package main

import (
	"log"

	"github.com/spf13/cobra"

	"floe/dsl"
	"floe/runtime"
)

var runCmd = &cobra.Command{
	Use:   "run [workflow_file]",
	Short: "Run a workflow in headless mode",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]

		// 1. Parse DSL
		workflow, err := dsl.ParseWorkflow(filename)
		if err != nil {
			log.Fatalf("Failed to parse workflow: %v", err)
		}

		// 2. Initialize Runtime
		rt := runtime.NewRuntime(workflow)

		// 3. Run Workflow
		if err := rt.Run(); err != nil {
			log.Fatalf("Workflow execution failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
