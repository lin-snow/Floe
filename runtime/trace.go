package runtime

import (
	"encoding/json"
	"os"
	"time"
)

type Trace struct {
	Steps []TraceEvent `json:"steps"`
}

type TraceEvent struct {
	StepName  string                 `json:"step_name"`
	Input     map[string]interface{} `json:"input"`
	Output    interface{}            `json:"output"`
	Messages  map[string]interface{} `json:"messages,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func (r *WorkflowRuntime) SaveTrace(path string) error {
	data, err := json.MarshalIndent(r.trace, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
