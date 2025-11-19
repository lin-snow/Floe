package main

import (
	"log"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"floe/dsl"
	"floe/internal/tui"
	"floe/runtime"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Run workflow with Terminal User Interface",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := cmd.Flags().GetString("file")

		if file == "" {
			// Interactive selection
			files, err := filepath.Glob("*.yaml")
			if err != nil {
				log.Fatal(err)
			}
			if len(files) == 0 {
				// Try example directory
				files, _ = filepath.Glob("example/*.yaml")
			}

			if len(files) == 0 {
				log.Fatal("No YAML files found in current or example directory")
			}

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select a workflow to run").
						Options(huh.NewOptions(files...)...).
						Value(&file),
				),
			)

			if err := form.Run(); err != nil {
				log.Fatal(err)
			}
		}

		// 1. Parse DSL
		workflow, err := dsl.ParseWorkflow(file)
		if err != nil {
			log.Fatalf("Failed to parse workflow: %v", err)
		}

		// 2. Initialize Runtime
		rt := runtime.NewRuntime(workflow)

		// 3. Start TUI
		app := tui.NewApp(rt)
		if err := app.Run(); err != nil {
			log.Fatalf("TUI failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	tuiCmd.Flags().StringP("file", "f", "", "Path to workflow YAML file")
}
