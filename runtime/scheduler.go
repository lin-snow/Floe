package runtime

import (
	"fmt"

	"floe/dsl"
	"floe/expr"
	"floe/memory"
)

type Scheduler interface {
	NextSteps(mem *memory.Memory, executedSteps map[string]bool, lastResults []StepResult) ([]dsl.Step, map[string]*RoutingTrace)
}

type BasicScheduler struct {
	workflow *dsl.Workflow
}

func NewBasicScheduler(wf *dsl.Workflow) *BasicScheduler {
	return &BasicScheduler{workflow: wf}
}

// NextSteps 决定下一个 Superstep 应该执行哪些步骤。
func (s *BasicScheduler) NextSteps(mem *memory.Memory, executedSteps map[string]bool, lastResults []StepResult) ([]dsl.Step, map[string]*RoutingTrace) {
	var nextSteps []dsl.Step
	routingTraces := make(map[string]*RoutingTrace)

	// 1. Check for Fallbacks from last superstep
	for _, res := range lastResults {
		if res.Fallback != "" {
			fallbackStep := s.findStep(res.Fallback)
			if fallbackStep != nil {
				nextSteps = append(nextSteps, *fallbackStep)
			}
		}
	}

	if len(nextSteps) > 0 {
		return nextSteps, nil
	}

	// 2. Normal Flow
	// If it's the first step (empty executedSteps), find the first step in the list.
	if len(executedSteps) == 0 {
		if len(s.workflow.Steps) > 0 {
			return []dsl.Step{s.workflow.Steps[0]}, nil
		}
		return nil, nil
	}

	// 3. Follow 'Next' from last results
	for _, res := range lastResults {
		// If step finished successfully (or ignored error, or skipped), proceed to Next
		if res.Err == nil || res.Ignored || res.Status == "skipped" {
			currentStep := s.findStep(res.NodeName)
			if currentStep != nil {
				// Resolve Next
				nextID, err := s.resolveNext(currentStep, mem)

				// Record routing trace
				// We need the raw expression/map. NormalizeNext gives us the parsed version.
				// But we want the raw string from YAML?
				// step.Next is interface{}.
				// Let's just fmt.Sprint(step.Next) for Raw? Or use the normalized expr.
				// The requirement says "Raw config".
				rawRouting := fmt.Sprintf("%v", currentStep.Next)
				routingTraces[res.NodeName] = &RoutingTrace{
					Raw:    rawRouting,
					Result: nextID,
				}

				if err != nil {
					fmt.Printf("Error resolving next for step %s: %v\n", currentStep.ID, err)
					continue
				}

				if nextID != "" {
					nextStep := s.findStep(nextID)
					if nextStep != nil && !executedSteps[nextStep.ID] {
						nextSteps = append(nextSteps, *nextStep)
					}
				} else {
					// No explicit next, try sequential fallback
					idx := s.findStepIndex(res.NodeName)
					if idx != -1 && idx+1 < len(s.workflow.Steps) {
						nextStep := &s.workflow.Steps[idx+1]
						// Only add if not executed
						if !executedSteps[nextStep.ID] {
							nextSteps = append(nextSteps, *nextStep)
						}
					}
				}
			}
		}
	}

	return nextSteps, routingTraces
}

func (s *BasicScheduler) resolveNext(step *dsl.Step, mem *memory.Memory) (string, error) {
	norm, err := dsl.NormalizeNext(step.Next)
	if err != nil {
		return "", err
	}
	if norm == nil {
		return "", nil
	}

	switch norm.Type {
	case dsl.NextStatic:
		return norm.Static, nil
	case dsl.NextExpr:
		return expr.EvaluateString(norm.Expr, mem)
	case dsl.NextMap:
		// Iterate keys, evaluate as bool
		// We pick the first one we find that is true.
		for k, v := range norm.Map {
			matched, err := expr.EvaluateBool(k, mem)
			if err != nil {
				fmt.Printf("Error evaluating route condition '%s': %v\n", k, err)
				continue
			}
			if matched {
				return v, nil
			}
		}
		return "", nil // No match
	}
	return "", nil
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
