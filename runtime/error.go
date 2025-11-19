package runtime

import "floe/dsl"

// ActionType 定义错误处理动作的类型。
type ActionType int

const (
	ActionRetry    ActionType = iota // 重试
	ActionIgnore                     // 忽略错误
	ActionFail                       // 失败 (终止工作流)
	ActionFallback                   // 执行 Fallback 步骤
)

// ErrorAction 描述针对特定错误应采取的动作。
type ErrorAction struct {
	Type             ActionType
	FallbackStepName string
}

// handleError 根据步骤的错误配置决定采取的动作。
func handleError(step *dsl.Step, err error) ErrorAction {
	config := step.Error

	switch config.Strategy {
	case "retry":
		return ErrorAction{Type: ActionRetry}
	case "ignore":
		return ErrorAction{Type: ActionIgnore}
	case "fallback":
		return ErrorAction{Type: ActionFallback, FallbackStepName: config.Fallback}
	case "fail":
		fallthrough
	default:
		return ErrorAction{Type: ActionFail}
	}
}
