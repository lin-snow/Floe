# Floe v0.5 TUI 技术方案

目标：为 Floe 添加基于终端的交互界面（TUI），用于可视化 workflow、实时观察执行、简单交互（单步/暂停/查看变量）。技术栈：`cobra`（CLI）、`lipgloss`（样式/布局）、`huh`（输入/表单）。后期可扩展到更复杂交互（bubbletea）。

---

## 一、总体设计概述（1 层图）

- CLI（cobra）负责命令入口与参数解析。命令：`floe tui --file <yaml> [--trace <trace.json>]`、`floe run`、`floe show-trace` 等。
- Runtime（已有 Go 进程）继续执行 workflow 并**向 TUI 以事件流方式推送**执行事件；TUI 订阅事件并渲染界面。若 runtime 与 TUI 运行在同一进程，可通过内部 channel 通信；若是独立进程，使用本地 websocket 或 Unix socket（v0.5 可先实现同进程 channel 模式）。
- Trace（trace.json）作为持久化/回放数据源。TUI 可在“回放模式”读取 trace.json 渲染历史执行。
- UI 层（lipgloss）渲染布局，huh 负责交互输入（选择、确认、简单表单）。

---

## 二、依赖（版本建议）

在 `go.mod` 中添加：

- `github.com/spf13/cobra`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/huh` （或 `github.com/muesli/huh`，确认你选用的库名）
  （可选后续）`github.com/charmbracelet/bubbletea`（若要更复杂交互）

---

## 三、项目目录建议（增量式）

```
/cmd/floe/main.go        # cobra CLI root
/cmd/floe/tui_cmd.go     # tui 子命令
/internal/tui/
    app.go               # 启动、事件总线、main loop
    layout.go            # lipgloss 布局与样式
    widgets.go           # 各组件（step list, details, variables）
    input.go             # huh 基本输入封装
    render.go            # 渲染相关工具
/internal/runtime_integration/
    events.go            # 事件类型定义、adapter（channel/socket）
/example/
    example.yaml
```

---

## 四、事件模型（关键契约）

定义 TUI 与 runtime 之间的事件结构（同进程 channel 或 JSON-over-socket）。

```
type EventType string
const (
    EventWorkflowStarted EventType = "workflow_started"
    EventSuperstepStart  EventType = "superstep_start"
    EventStepStart       EventType = "step_start"
    EventStepEnd         EventType = "step_end"
    EventStepSkipped     EventType = "step_skipped"
    EventMemoryUpdate    EventType = "memory_update"
    EventWorkflowEnd     EventType = "workflow_end"
    EventTraceSnapshot   EventType = "trace_snapshot"
    EventLog             EventType = "log"
)

type Event struct {
    Type      EventType
    Timestamp time.Time
    Payload   map[string]interface{}   // 尽量 JSON-serializable
}
```

**重要**：Payload 约定要明确，例如 `step_start` 包含 `step_id`, `input_snapshot`；`step_end` 包含 `step_id`, `output`, `status`。

实现要求：

- runtime 在合适位置（step start/end、superstep boundary、memory change） `sendEvent(event)`。
- TUI 提供 `Subscribe()` 以获得事件流；若是 run+TUI 在同进程，将 channel 传入；若是独立进程，提供 socket adapter（留作 v0.6）。

---

## 五、TUI 布局草案（lipgloss）

目标是清晰、低学习成本、信息密度高。采用三列＋底部状态栏。

```
+---------------------------------------------+
| 左栏 (25%) | 中间 (50%)         | 右栏 25%  |
| Steps List | Step Detail / Trace | Variables |
|            | (log & messages)    | & Messages |
+---------------------------------------------+
| Bottom: Controls: [q] quit [p] pause [n] next|
+---------------------------------------------+
```

组件说明：

- **Steps List**：树或序列化视图，支持高亮当前 step、标记 executed/skipped/failed（颜色区分）。
- **Step Detail**：显示输入、输出、messages、condition、routing（来自 trace/事件）。
- **Variables**：global memory snapshot（可折叠展示 JSON 树）。
- **Bottom Controls**：提示键位；当处于“单步”或“paused”时显示当前状态。

样式用 lipgloss 封装到 `layout.go`，提供 `StyleStepExecuted`, `StyleStepSkipped`, `StyleHighlight`, `StylePanel` 等。

---

## 六、交互模型（huh）

v0.5 的交互不复杂，主要覆盖：

- 选择文件（若 CLI 未指定）；
- 在 Steps List 中上下移动并按 Enter 查看详情；
- 启动/暂停/单步/恢复；
- 在运行时可按 `v` 查看 variables，`t` 切换 trace 回放模式；
- 在 paused 状态允许对某个 global variable 进行临时修改（用于调试），确认后更新 memory（通过 runtime 接口）。

示例输入流程（用 huh 组合）：

```
choice := huh.NewSelect("Choose file", &filePath).
    Options(fileList...).
    Run()
```

实现约束：

- 输入操作不要阻塞事件接收；用 goroutine 做 blocking 输入并与主渲染循环通过 channel 交互。

---

## 七、实现细节（关键点、陷阱与建议）

### 七点关键实现说明

1. **事件到 UI 的异步消费**
   - 主渲染循环必须从事件 channel 非阻塞读取并调用 render 更新，避免 UI 卡死。
   - 使用缓冲 channel（例如 100）防止短期事件高峰丢失。
2. **渲染频率控制**
   - 为避免过度重绘，采用节流：例如每 100ms 刷新一次（除非有 step 状态变化重要事件则立即刷新）。
3. **Snapshot vs Live**
   - 对 memory 的展示使用 `Snapshot()`（deep copy）以防渲染时数据被 runtime 修改引发 race。
   - 若与 runtime 同进程，确保 `Snapshot()` 在 Memory 内部做 RLock。
4. **回放模式**
   - 实现一个 `LoadTrace(path)`，读取 trace.json 并以事件序列回放。
   - 回放速度可控（速率、暂停、跳转到时间点）。
5. **单步 & Pause**
   - 当用户按 pause：runtime 应支持 `Pause()` 或 TUI 采用“控制运行”方式（若 runtime 不能 pause，TUI 可模拟：在 superstep boundary 自动停止并等待用户继续）。
   - 推荐在 runtime 增加 `ControlChannel`：`type ControlCmd string` 支持 `Pause`, `Resume`, `StepOnce`.
6. **错误显示**
   - 把 step error、expression eval error 等在 Step Detail 高亮展示，并在 Bottom Controls 显示提示。
7. **日志与滚动**
   - Step Detail 内部提供日志滚动区（lipgloss 渲染 + 简单缓冲），并支持 `PgUp/PgDn` 键。

---

## 八、运行模式（两种，v0.5 支持模式一）

**模式 A（嵌入式）**：`floe tui --file X.yaml` 在同一进程内，TUI 启动 runtime 并订阅内部 event channel（推荐 v0.5 优先实现）。
优点：无需 IPC；实现简单。
