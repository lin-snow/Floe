# v0.2 版本技术方案与指导

# Floe Runtime v0.2 技术方案

**目标：在保持当前 MVP 结构的前提下，引入 Pregel/LangGraph 的关键能力，构建可扩展的智能执行引擎基础架构。**
本版本属于渐进式增强，而非重构。

---

# 1. Superstep Runtime（轻量 Pregel 调度层）

## 1.1 目标

将当前“顺序执行 + 平行 step”升级为**基于超步 (superstep) 的统一调度模型**，为之后的图结构、并发执行和状态一致性奠定基础。

## 1.2 新增结构

```
type StepResult struct {
    NodeName string
    Output   any
    Messages map[string]any
    Err      error
}
```

新增执行循环：

```
func (r *WorkflowRuntime) Run() error {
    for {
        activeSteps := r.scheduler.NextSteps(r.memory)

        if len(activeSteps) == 0 {
            break
        }

        results := r.runSuperstep(activeSteps)
        r.mergeResults(results)
    }

    return nil
}
```

## 1.3 关键变化

- 所有步骤都由 scheduler 决定是否执行
- 同一 superstep 内的节点可并行执行（goroutine）
- memory 更新在 superstep 结束后统一合并（隔离 + 确定性）

---

# 2. Message Passing（轻量 Aggregator）

## 2.1 目标

允许节点输出消息（messages），并在 superstep 合并阶段写入全局 memory。该能力是未来多代理沟通、自动链路推导的核心。

## 2.2 DSL 扩展：节点输出格式

```
steps:
  fetch:
    type: task
    tool: http_get
    output: global.raw
    messages:
      nextUrl: "${global.raw.id}"
```

## 2.3 runtime 输入结构

在已有 StepResult 增加 Messages 字段：

```
Messages map[string]any
```

## 2.4 合并逻辑

```
func (r *WorkflowRuntime) mergeResults(results []StepResult) {
    for _, res := range results {
        if res.Output != nil {
            _ = r.memory.Set("global."+res.NodeName, res.Output)
        }
        for k, v := range res.Messages {
            _ = r.memory.Set("messages."+k, v)
        }
    }
}
```

---

# 3. Execution Trace（可调试性增强）

## 3.1 目标

为每个 step 记录输入 / 输出状态，用于调试、可视化、重放（replay）。

## 3.2 数据结构

```
type Trace struct {
    Steps []TraceEvent
}

type TraceEvent struct {
    StepName  string
    Input     map[string]any
    Output    any
    Messages  map[string]any
    Timestamp time.Time
}
```

## 3.3 记录位置

在 superstep 执行完成：

```
r.trace.Steps = append(r.trace.Steps, TraceEvent{
    StepName: res.NodeName,
    Input:    r.memory.Snapshot(),
    Output:   res.Output,
    Messages: res.Messages,
    Timestamp: time.Now(),
})
```

## 3.4 导出

提供：

```
func (r *WorkflowRuntime) SaveTrace(path string) error
```

输出 trace.json。

---

# 4. DSL v0.2：图结构与流转关系

## 4.1 目标

为 runtime 提供拓扑结构依据，让调度器根据依赖关系决定执行顺序。

## 4.2 DSL 改进方案

### 方案 A：每个 step 指定 next

最小改动：

```
steps:
  parse:
    type: task
    next: summarize

  summarize:
    type: task
```

### 方案 B：显式 edges（未来版本）

```
edges:
  - from: parse
    to: summarize
```

## 4.3 调度器

新增：

```
type Scheduler interface {
    NextSteps(memory *Memory) []dsl.Step
}
```

MVP 版本中：

- 如果 step 没有执行过 → 执行
- 如果 step 有 next 字段 → 在下一 superstep 启动 next

这保留灵活性，也允许以后支持循环、反馈、并行 DAG。

---

# 5. 目录结构（增量更新）

```
/runtime
    runtime.go
    scheduler.go         ← 新增
    superstep.go         ← 新增
    trace.go             ← 新增

/dsl
    parser.go
    model.go             ← 扩展 step/edges 结构

/memory
    memory.go

/tools
    ...
```
