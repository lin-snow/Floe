package dsl

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Workflow struct {
	Name   string       `mapstructure:"name"`
	Memory MemoryConfig `mapstructure:"memory"`
	Steps  []Step       `mapstructure:"steps"`
}

type MemoryConfig struct {
	Initial map[string]interface{} `mapstructure:"initial"`
}

// ErrorConfig 定义步骤的错误处理策略。
type ErrorConfig struct {
	Strategy  string `mapstructure:"strategy"`   // 策略: retry, fail, ignore, fallback
	Retries   int    `mapstructure:"retries"`    // 重试次数
	DelayMs   int    `mapstructure:"delay_ms"`   // 重试延迟 (毫秒)
	TimeoutMs int    `mapstructure:"timeout_ms"` // 超时时间 (毫秒)
	Fallback  string `mapstructure:"fallback"`   // Fallback 步骤 ID
}

// Step 代表工作流中的一个步骤。
// 它可以是一个简单的任务（Task），也可以是一个包含分支的并行步骤（Parallel）。
type Step struct {
	ID       string                 `mapstructure:"id"`       // 步骤的唯一标识符
	Type     string                 `mapstructure:"type"`     // 步骤类型：task 或 parallel
	Tool     string                 `mapstructure:"tool"`     // 使用的工具名称（仅 task 类型）
	Input    map[string]interface{} `mapstructure:"input"`    // 输入参数，支持变量插值
	Output   string                 `mapstructure:"output"`   // 输出结果存储的内存路径
	Branches []Step                 `mapstructure:"branches"` // 并行分支（仅 parallel 类型）
	Next     interface{}            `mapstructure:"next"`     // 下一步骤的 ID 或路由配置 (string | map)
	When     string                 `mapstructure:"when"`     // 执行条件表达式
	Messages map[string]string      `mapstructure:"messages"` // 步骤产生的消息，用于消息传递
	Error    ErrorConfig            `mapstructure:"error"`    // 错误处理配置
}

// NextType defines the type of the Next field
type NextType int

const (
	NextStatic NextType = iota
	NextMap
	NextExpr
)

// NormalizedNext holds the normalized next configuration
type NormalizedNext struct {
	Type   NextType
	Static string
	Map    map[string]string
	Expr   string
}

// NormalizeNext parses the Next field into a NormalizedNext struct
func NormalizeNext(next interface{}) (*NormalizedNext, error) {
	if next == nil {
		return nil, nil
	}

	switch v := next.(type) {
	case string:
		if strings.HasPrefix(v, "${") {
			return &NormalizedNext{Type: NextExpr, Expr: v}, nil
		}
		return &NormalizedNext{Type: NextStatic, Static: v}, nil
	case map[string]interface{}:
		// Convert map[string]interface{} to map[string]string
		m := make(map[string]string)
		for k, val := range v {
			strVal, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("next map values must be strings, got %T", val)
			}
			m[k] = strVal
		}
		return &NormalizedNext{Type: NextMap, Map: m}, nil
	case map[string]string:
		return &NormalizedNext{Type: NextMap, Map: v}, nil
	default:
		return nil, fmt.Errorf("unsupported type for next: %T", next)
	}
}

func ParseWorkflow(filename string) (*Workflow, error) {
	v := viper.New()
	v.SetConfigFile(filename)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var wf Workflow
	if err := v.UnmarshalKey("workflow", &wf); err != nil {
		return nil, err
	}

	return &wf, nil
}
