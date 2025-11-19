package runtime

import (
	"floe/dsl"
	"floe/memory"
)

type Scheduler interface {
	NextSteps(mem *memory.Memory, executedSteps map[string]bool, lastResults []StepResult) []dsl.Step
}

type BasicScheduler struct {
	workflow *dsl.Workflow
}

func NewBasicScheduler(wf *dsl.Workflow) *BasicScheduler {
	return &BasicScheduler{workflow: wf}
}

// NextSteps 决定下一个 Superstep 应该执行哪些步骤。
func (s *BasicScheduler) NextSteps(mem *memory.Memory, executedSteps map[string]bool, lastResults []StepResult) []dsl.Step {
	var nextSteps []dsl.Step

	// 1. Check for Fallbacks from last superstep
	for _, res := range lastResults {
		if res.Fallback != "" {
			// If fallback is triggered, we ONLY execute the fallback step(s) in the next superstep?
			// Or we prioritize them. The requirement says: "fallback 触发后，下一 superstep 只执行 fallback step"
			// So if we find any fallback, we return it immediately.
			fallbackStep := s.findStep(res.Fallback)
			if fallbackStep != nil {
				nextSteps = append(nextSteps, *fallbackStep)
			}
		}
	}

	if len(nextSteps) > 0 {
		return nextSteps
	}

	// 2. Normal Flow
	// If it's the first step (empty executedSteps), find the first step in the list.
	if len(executedSteps) == 0 {
		if len(s.workflow.Steps) > 0 {
			return []dsl.Step{s.workflow.Steps[0]}
		}
		return nil
	}

	// Find steps that are pointed to by 'next' from recently executed steps
	// But wait, we need to know which steps were just finished in the *last* superstep.
	// The current interface `NextSteps(mem, executed)` doesn't explicitly pass "last finished steps".
	// However, we can infer it or change the interface.
	// For MVP v0.2, we just looked for steps that haven't been executed yet but are "next".
	// Let's refine the interface to accept `lastResults`.

	for _, res := range lastResults {
		// If step finished successfully (or ignored error), proceed to Next
		if res.Err == nil || res.Ignored {
			currentStep := s.findStep(res.NodeName)
			if currentStep != nil && currentStep.Next != "" {
				nextStep := s.findStep(currentStep.Next)
				if nextStep != nil && !executedSteps[nextStep.ID] {
					nextSteps = append(nextSteps, *nextStep)
				}
			} else if currentStep != nil {
				// Sequential default: if no 'next' defined, find the one after it in the list
				// This is a bit implicit. Let's stick to explicit 'next' or linear list if 'next' is missing.
				// For v0.2 we implemented linear search if next is empty.
				idx := s.findStepIndex(res.NodeName)
				if idx != -1 && idx+1 < len(s.workflow.Steps) {
					nextStep := &s.workflow.Steps[idx+1]
					if !executedSteps[nextStep.ID] {
						nextSteps = append(nextSteps, *nextStep)
					}
				}
			}
		}
	}

	return nextSteps
}

func (s *BasicScheduler) findStep(id string) *dsl.Step {
	for _, step := range s.workflow.Steps {
		if step.ID == id {
			return &step
		}
	}
	return nil
}

func (s *BasicScheduler) findStepIndex(id string) int {
	for i, step := range s.workflow.Steps {
		if step.ID == id {
			return i
		}
	}
	return -1
}
