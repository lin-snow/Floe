package runtime

import (
	"context"
	"fmt"
	"sync"

	"floe/dsl"
	"floe/memory"
	"floe/tools"
)

type WorkflowRuntime struct {
	workflow *dsl.Workflow
	memory   *memory.Memory
}

func NewRuntime(wf *dsl.Workflow) *WorkflowRuntime {
	mem := memory.NewMemory()
	if wf.Memory.Initial != nil {
		for k, v := range wf.Memory.Initial {
			mem.Set(k, v)
		}
	}
	return &WorkflowRuntime{
		workflow: wf,
		memory:   mem,
	}
}

func (r *WorkflowRuntime) Run() error {
	fmt.Printf("Starting workflow: %s\n", r.workflow.Name)
	for _, step := range r.workflow.Steps {
		if err := r.executeStep(&step); err != nil {
			return err
		}
	}
	fmt.Println("Workflow completed successfully.")
	return nil
}

func (r *WorkflowRuntime) executeStep(step *dsl.Step) error {
	fmt.Printf("Executing step: %s (Type: %s)\n", step.ID, step.Type)

	switch step.Type {
	case "task":
		return r.executeTask(step)
	case "parallel":
		return r.executeParallel(step)
	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}
}

func (r *WorkflowRuntime) executeTask(step *dsl.Step) error {
	// 1. Resolve Inputs
	input := make(map[string]interface{})
	for k, v := range step.Input {
		if strVal, ok := v.(string); ok {
			input[k] = r.memory.ResolveInterpolation(strVal)
		} else {
			input[k] = v
		}
	}

	// 2. Get Tool
	tool, err := tools.Get(step.Tool)
	if err != nil {
		return err
	}

	// 3. Execute Tool
	// TODO: Add timeout from metadata if available
	result, err := tool.Run(context.Background(), input)
	if err != nil {
		return fmt.Errorf("step %s failed: %w", step.ID, err)
	}

	// 4. Write Output
	if step.Output != "" {
		if err := r.memory.Set(step.Output, result); err != nil {
			return fmt.Errorf("failed to set output for step %s: %w", step.ID, err)
		}
	}

	// Trace log (simplified)
	fmt.Printf("  [Task] %s finished. Output: %v\n", step.ID, result)
	return nil
}

func (r *WorkflowRuntime) executeParallel(step *dsl.Step) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(step.Branches))

	for _, branch := range step.Branches {
		wg.Add(1)
		go func(b dsl.Step) {
			defer wg.Done()
			// Recursive execution for branches
			// Note: Branches in DSL are defined as Steps, but here we treat them as individual tasks or nested steps.
			// The DSL structure in parser.go defines Branches as []Step.
			// So we can just call executeStep.
			// However, the README example shows branches as a list of tasks.
			// Let's assume branches are just steps.
			if err := r.executeStep(&b); err != nil {
				errChan <- err
			}
		}(branch)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err // Return first error
		}
	}

	return nil
}
