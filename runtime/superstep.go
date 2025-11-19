package runtime

import (
	"context"
	"sync"

	"floe/dsl"
	"floe/tools"
)

type StepResult struct {
	NodeName string
	Output   interface{}
	Messages map[string]interface{}
	Err      error
}

func (r *WorkflowRuntime) runSuperstep(steps []dsl.Step) []StepResult {
	var wg sync.WaitGroup
	results := make([]StepResult, len(steps))

	for i, step := range steps {
		wg.Add(1)
		go func(idx int, s dsl.Step) {
			defer wg.Done()
			res := r.executeSingleStep(&s)
			results[idx] = res
		}(i, step)
	}

	wg.Wait()
	return results
}

func (r *WorkflowRuntime) executeSingleStep(step *dsl.Step) StepResult {
	// 1. Resolve Inputs
	input := make(map[string]interface{})
	for k, v := range step.Input {
		if strVal, ok := v.(string); ok {
			input[k] = r.memory.ResolveInterpolation(strVal)
		} else {
			input[k] = v
		}
	}

	// 2. Get Tool (only if not parallel)
	var tool tools.Tool
	var err error

	if step.Type != "parallel" {
		tool, err = tools.Get(step.Tool)
		if err != nil {
			return StepResult{NodeName: step.ID, Err: err}
		}
	}

	// 3. Execute Tool
	var output interface{}

	if step.Type == "parallel" {
		err = r.executeParallel(step)
	} else {
		output, err = tool.Run(context.Background(), input)
	}

	if err != nil {
		return StepResult{NodeName: step.ID, Err: err}
	}

	// 4. Resolve Messages
	messages := make(map[string]interface{})
	for k, v := range step.Messages {
		messages[k] = r.memory.ResolveInterpolation(v)
	}

	return StepResult{
		NodeName: step.ID,
		Output:   output,
		Messages: messages,
		Err:      nil,
	}
}
