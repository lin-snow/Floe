package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"floe/dsl"
	"floe/runtime"
)

func main() {
	// Parse command line arguments
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Usage: floe <workflow_file>")
		os.Exit(1)
	}

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
}
