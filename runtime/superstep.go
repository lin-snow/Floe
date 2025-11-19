package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"floe/dsl"
	"floe/internal/runtime_integration"
	"floe/tools"
)

type StepResult struct {
	NodeName  string
	Output    interface{}
	Messages  map[string]interface{}
	Err       error
	Retries   int
	Ignored   bool
	Fallback  string
	Strategy  string
	ErrorMsg  string
	Status    string          // executed | skipped
	Condition *ConditionTrace // Condition trace info
	Routing   *RoutingTrace   // Routing trace info
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
	r.Emit(runtime_integration.NewEvent(runtime_integration.EventStepStart, map[string]interface{}{
		"step_id": step.ID,
		"tool":    step.Tool,
	}))

	var finalErr error
	var output interface{}
	var messages map[string]interface{}

	retries := 0
	maxRetries := step.Error.Retries
	delay := time.Duration(step.Error.DelayMs) * time.Millisecond
	timeout := time.Duration(step.Error.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 24 * time.Hour // Default no timeout (effectively)
	}

	for {
		// 1. Resolve Inputs
		input := make(map[string]interface{})
		for k, v := range step.Input {
			if strVal, ok := v.(string); ok {
				input[k] = r.memory.ResolveInterpolation(strVal)
			} else {
				input[k] = v
			}
		}

		// 2. Execute with Timeout
		var err error
		output, err = r.runWithTimeout(step, input, timeout)

		if err == nil {
			// Success
			finalErr = nil
			break
		}

		// Handle Error
		action := handleError(step, err)

		if action.Type == ActionRetry {
			if retries < maxRetries {
				retries++
				time.Sleep(delay)
				continue
			}
			// Retries exhausted. Check if fallback is configured.
			if step.Error.Fallback != "" {
				return StepResult{
					NodeName: step.ID,
					Err:      fmt.Errorf("max retries exceeded, triggering fallback: %w", err),
					Fallback: step.Error.Fallback,
					Strategy: "retry-fallback",
					ErrorMsg: err.Error(),
					Retries:  retries,
					Status:   "executed",
				}
			}
			// Fall through to fail
			finalErr = fmt.Errorf("max retries exceeded: %w", err)
		} else if action.Type == ActionIgnore {
			// Ignore error
			return StepResult{
				NodeName: step.ID,
				Err:      nil, // Clear error so runtime continues
				Ignored:  true,
				ErrorMsg: err.Error(),
				Strategy: "ignore",
				Status:   "executed",
			}
		} else if action.Type == ActionFallback {
			// Return fallback step name
			return StepResult{
				NodeName: step.ID,
				Err:      fmt.Errorf("fallback triggered: %w", err),
				Fallback: action.FallbackStepName,
				Strategy: "fallback",
				ErrorMsg: err.Error(),
				Status:   "executed",
			}
		} else {
			// Fail
			finalErr = err
		}
		break
	}

	if finalErr != nil {
		return StepResult{
			NodeName: step.ID,
			Err:      finalErr,
			Retries:  retries,
			Strategy: "fail",
			ErrorMsg: finalErr.Error(),
			Status:   "executed",
		}
	}

	// 4. Resolve Messages
	messages = make(map[string]interface{})
	for k, v := range step.Messages {
		messages[k] = r.memory.ResolveInterpolation(v)
	}

	return StepResult{
		NodeName: step.ID,
		Output:   output,
		Messages: messages,
		Err:      nil,
		Retries:  retries,
		Status:   "executed",
	}
}

func (r *WorkflowRuntime) runWithTimeout(step *dsl.Step, input map[string]interface{}, timeout time.Duration) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		val interface{}
		err error
	}
	ch := make(chan result, 1)

	go func() {
		// 2. Get Tool (only if not parallel)
		if step.Type == "parallel" {
			err := r.executeParallel(step)
			ch <- result{nil, err}
			return
		}

		tool, err := tools.Get(step.Tool)
		if err != nil {
			ch <- result{nil, err}
			return
		}

		// 3. Execute Tool
		out, err := tool.Run(ctx, input)
		ch <- result{out, err}
	}()

	select {
	case res := <-ch:
		return res.val, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
