package runtime

import (
	"fmt"
	"sync"
	"time"

	"floe/dsl"
	"floe/expr"
	"floe/memory"
)

// WorkflowRuntime 是工作流执行的运行时环境。
// 它管理工作流的生命周期、内存状态、调度和执行跟踪。
type WorkflowRuntime struct {
	workflow  *dsl.Workflow  // 工作流定义
	memory    *memory.Memory // 全局内存
	scheduler Scheduler      // 调度器
	trace     *Trace         // 执行跟踪
}

// NewRuntime 创建一个新的 WorkflowRuntime 实例。
// 它会初始化内存，并加载工作流定义的初始变量。
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

// Run 开始执行工作流。
// 它使用 Superstep 模式：调度 -> 执行 -> 合并结果，直到没有更多步骤可执行。
func (r *WorkflowRuntime) Run() error {
	fmt.Printf("Starting workflow: %s\n", r.workflow.Name)

	executedSteps := make(map[string]bool)
	var lastResults []StepResult

	for {
		activeSteps, routingTraces := r.scheduler.NextSteps(r.memory, executedSteps, lastResults)

		// Update routing info in trace for previous steps
		if len(routingTraces) > 0 {
			for i := len(r.trace.Steps) - 1; i >= 0; i-- {
				stepName := r.trace.Steps[i].StepName
				if rt, ok := routingTraces[stepName]; ok {
					r.trace.Steps[i].Routing = rt
					delete(routingTraces, stepName) // Remove to avoid double update (though unlikely)
				}
			}
		}

		if len(activeSteps) == 0 {
			break
		}

		// Filter steps based on 'When' condition
		var stepsToExecute []dsl.Step
		var skippedResults []StepResult
		conditionTraces := make(map[string]*ConditionTrace)

		for _, step := range activeSteps {
			shouldRun := true
			var condTrace *ConditionTrace

			if step.When != "" {
				result, err := expr.EvaluateBool(step.When, r.memory)
				condTrace = &ConditionTrace{Raw: step.When, Result: result}
				conditionTraces[step.ID] = condTrace

				if err != nil {
					fmt.Printf("Error evaluating condition for step %s: %v\n", step.ID, err)
					shouldRun = false
				} else {
					shouldRun = result
				}
			}

			if shouldRun {
				stepsToExecute = append(stepsToExecute, step)
			} else {
				skippedResults = append(skippedResults, StepResult{
					NodeName:  step.ID,
					Status:    "skipped",
					Condition: condTrace,
				})
			}
		}

		fmt.Printf("Superstep: Executing %d steps (Skipped: %d)...\n", len(stepsToExecute), len(skippedResults))

		var results []StepResult
		if len(stepsToExecute) > 0 {
			results = r.runSuperstep(stepsToExecute)
		}

		results = append(results, skippedResults...)

		// Attach condition info to executed results if missing (runSuperstep doesn't set it)
		for i := range results {
			if results[i].Condition == nil {
				if ct, ok := conditionTraces[results[i].NodeName]; ok {
					results[i].Condition = ct
				}
			}
			// Also set Status to "executed" if empty (for non-skipped steps)
			if results[i].Status == "" {
				results[i].Status = "executed"
			}
		}

		r.mergeResults(results, executedSteps)

		lastResults = results
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
		}

		// Record Trace
		r.trace.Steps = append(r.trace.Steps, TraceEvent{
			StepName:  res.NodeName,
			Input:     r.memory.Snapshot(),
			Output:    res.Output,
			Messages:  res.Messages,
			Timestamp: time.Now(),
			Error:     res.ErrorMsg,
			Retries:   res.Retries,
			Strategy:  res.Strategy,
			Fallback:  res.Fallback,
			Ignored:   res.Ignored,
			Status:    res.Status,
			Condition: res.Condition,
			Routing:   res.Routing,
		})

		if res.Output != nil {
			step := r.findStepByID(res.NodeName)
			if step != nil && step.Output != "" {
				_ = r.memory.Set(step.Output, res.Output)
			} else {
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
