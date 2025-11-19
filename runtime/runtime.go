package runtime

import (
	"fmt"
	"sync"
	"time"

	"floe/dsl"
	"floe/memory"
)

type WorkflowRuntime struct {
	workflow  *dsl.Workflow
	memory    *memory.Memory
	scheduler Scheduler
	trace     *Trace
}

func NewRuntime(wf *dsl.Workflow) *WorkflowRuntime {
	mem := memory.NewMemory()
	if wf.Memory.Initial != nil {
		for k, v := range wf.Memory.Initial {
			mem.Set(k, v)
		}
	}
	return &WorkflowRuntime{
		workflow:  wf,
		memory:    mem,
		scheduler: NewBasicScheduler(wf),
		trace:     &Trace{Steps: []TraceEvent{}},
	}
}

func (r *WorkflowRuntime) Run() error {
	fmt.Printf("Starting workflow: %s\n", r.workflow.Name)

	executedSteps := make(map[string]bool)

	for {
		activeSteps := r.scheduler.NextSteps(r.memory, executedSteps)
		if len(activeSteps) == 0 {
			break
		}

		fmt.Printf("Superstep: Executing %d steps...\n", len(activeSteps))
		results := r.runSuperstep(activeSteps)
		r.mergeResults(results, executedSteps)
	}

	fmt.Println("Workflow completed successfully.")

	// Save trace
	if err := r.SaveTrace("trace.json"); err != nil {
		fmt.Printf("Warning: failed to save trace: %v\n", err)
	}

	return nil
}

func (r *WorkflowRuntime) mergeResults(results []StepResult, executedSteps map[string]bool) {
	for _, res := range results {
		executedSteps[res.NodeName] = true

		if res.Err != nil {
			fmt.Printf("Error in step %s: %v\n", res.NodeName, res.Err)
			// TODO: Handle error policy (stop or continue)
			// For now, we just log and continue, but maybe we should stop?
			// MVP v0.1 stopped. v0.2 guide says "memory 更新在 superstep 结束后统一合并".
			continue
		}

		// Record Trace
		r.trace.Steps = append(r.trace.Steps, TraceEvent{
			StepName: res.NodeName,
			Input:    r.memory.Snapshot(), // Snapshot BEFORE merge? Or AFTER? Guide says "In superstep execution complete". Usually input is what was used.
			// But here we are merging output.
			// Let's record Output and Messages.
			Output:    res.Output,
			Messages:  res.Messages,
			Timestamp: time.Now(),
		})

		if res.Output != nil {
			// If output path is defined in step, we should use it.
			// But StepResult doesn't have the output path.
			// We need to look up the step definition or pass it in StepResult.
			// For simplicity, let's use the convention from AIGUIDE: "global." + NodeName
			// BUT the DSL has an 'output' field. We should respect it.
			// Let's find the step to get the output path.
			step := r.findStepByID(res.NodeName)
			if step != nil && step.Output != "" {
				_ = r.memory.Set(step.Output, res.Output)
			} else {
				// Fallback or default?
				_ = r.memory.Set("global."+res.NodeName, res.Output)
			}
		}

		for k, v := range res.Messages {
			_ = r.memory.Set("messages."+k, v)
		}
	}
}

func (r *WorkflowRuntime) findStepByID(id string) *dsl.Step {
	for _, step := range r.workflow.Steps {
		if step.ID == id {
			return &step
		}
	}
	return nil
}

// executeParallel is used by superstep.go for legacy "parallel" step types
func (r *WorkflowRuntime) executeParallel(step *dsl.Step) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(step.Branches))

	for _, branch := range step.Branches {
		wg.Add(1)
		go func(b dsl.Step) {
			defer wg.Done()
			res := r.executeSingleStep(&b)
			if res.Err != nil {
				errChan <- res.Err
				return
			}
			// Write output
			if res.Output != nil {
				if b.Output != "" {
					_ = r.memory.Set(b.Output, res.Output)
				} else {
					_ = r.memory.Set("global."+b.ID, res.Output)
				}
			}
			// Write messages
			for k, v := range res.Messages {
				_ = r.memory.Set("messages."+k, v)
			}
		}(branch)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
