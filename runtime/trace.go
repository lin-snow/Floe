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
	Error     string                 `json:"error,omitempty"`    // 错误信息
	Retries   int                    `json:"retries,omitempty"`  // 重试次数
	Strategy  string                 `json:"strategy,omitempty"` // 错误处理策略
	Fallback  string                 `json:"fallback,omitempty"` // Fallback 步骤
	Ignored   bool                   `json:"ignored,omitempty"`  // 是否忽略错误
}

func (r *WorkflowRuntime) SaveTrace(path string) error {
	data, err := json.MarshalIndent(r.trace, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
