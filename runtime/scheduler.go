package runtime

import (
	"floe/dsl"
	"floe/memory"
)

type Scheduler interface {
	NextSteps(memory *memory.Memory, executedSteps map[string]bool) []dsl.Step
}

type BasicScheduler struct {
	workflow *dsl.Workflow
}

func NewBasicScheduler(wf *dsl.Workflow) *BasicScheduler {
	return &BasicScheduler{workflow: wf}
}

func (s *BasicScheduler) NextSteps(memory *memory.Memory, executedSteps map[string]bool) []dsl.Step {
	var nextSteps []dsl.Step

	// Simple logic:
	// 1. If no steps executed, start with the first one.
	// 2. If steps executed, find steps that are pointed to by 'next' or are sequential.
	// For MVP v0.2, we can stick to a simple sequence if 'next' is not present,
	// or follow 'next' if present.

	// Let's try to find steps that haven't been executed yet.
	// And if a step specifies 'next', we should respect it.

	// Current simplified logic:
	// Iterate through all steps. If a step is not executed, check if it's ready.
	// For linear sequence:
	// - First step is always ready if not executed.
	// - Subsequent steps are ready if previous is executed.

	// But we want to support 'next'.
	// Let's maintain a set of "ready" steps?
	// Or just look at the last executed step?

	// Let's assume for now we just want to run the next available step in the list that hasn't run.
	// UNLESS the previous step had a 'next' pointer.

	// Actually, the AIGUIDE says:
	// "MVP 版本中：如果 step 没有执行过 → 执行; 如果 step 有 next 字段 → 在下一 superstep 启动 next"

	// This implies we need to know what was the *last* executed step to know what's next if 'next' is used.
	// But "NextSteps" is stateless in the interface (except memory/executedSteps).

	// Let's refine:
	// We can iterate through the steps list.
	// If we find a step that is NOT executed:
	//   Check if the PREVIOUS step (if any) was executed.
	//   If previous executed:
	//      Did previous have 'next'? If so, is this step the target?
	//      If previous didn't have 'next', then this step is the natural next.
	//   If no previous step (it's the first one), it's ready.

	// Limitation: This doesn't handle branching/DAG well yet, but fits the linear + parallel MVP.

	for i, step := range s.workflow.Steps {
		if executedSteps[step.ID] {
			continue
		}

		// This step is not executed. Is it ready?
		if i == 0 {
			return []dsl.Step{step}
		}

		prevStep := s.workflow.Steps[i-1]
		if executedSteps[prevStep.ID] {
			// Previous was executed.
			// Check if previous had a 'next' directive
			if prevStep.Next != "" {
				if prevStep.Next == step.ID {
					return []dsl.Step{step}
				}
				// If next points elsewhere, we skip this one (it's skipped or waiting for another path)
				// But wait, if next points elsewhere, we need to find THAT step.
				target := s.findStepByID(prevStep.Next)
				if target != nil && !executedSteps[target.ID] {
					return []dsl.Step{*target}
				}
			} else {
				// No 'next' directive, so natural sequence
				return []dsl.Step{step}
			}
		}
	}

	return nextSteps
}

func (s *BasicScheduler) findStepByID(id string) *dsl.Step {
	for _, step := range s.workflow.Steps {
		if step.ID == id {
			return &step
		}
	}
	return nil
}
