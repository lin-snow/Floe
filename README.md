# Floe

Workflow &amp;&amp; DSL MVP

---

# 大致技术方案 v1(已实现)

下面是一份面向可实现性的**详细技术方案**（包含 DSL schema、runtime 模块、并发/内存策略、变量替换、trace 格式、测试/验收要点与实现要点），你可以直接拿去实现或交接给同事。

我会把信息分成：目标与边界 → DSL 规范 → Runtime 架构 → 并发与内存细节 → 变量替换与工具接口 → 日志/Trace/可视化 → 验收/Demo 要点 → 实现步骤清单 & 交付产物 清单。直接上手即可。

# 目标与边界（Scope）

- 目标：实现一个**能解析 YAML 结构化 DSL、并发执行 parallel nodes、共享 thread-safe 全局 memory、并能跑通至少一个完整 example 的 Golang MVP runtime**。
- 不在此 MVP 范围：复杂调度优化、分布式执行、权限/沙箱、复杂错误恢复、插件治理、向量数据库等。
- 可加强但可控的点：DSL 保留 `input/output` 映射、node-level metadata（timeout/retry），以及详尽的 trace，便于 AI IDE 分析与自动生成补全。

# DSL（YAML）规范（v0.1 结构化草案）

设计原则：结构化、显式 input/output、node id 唯一、parallel 为一等节点、memory 路径支持（global.\*）。

示例：

```
version: "0.1"

workflow:
  id: example_flow
  memory:
    type: global

  inputs:
    endpoint: "https://api.example.com/search?q=quantum"

  nodes:
    - id: fetch_info
      type: task
      tool: http_get
      input:
        url: "${inputs.endpoint}"
      output: "global.search_result"

    - id: process_parallel
      type: parallel
      branches:
        - id: summary
          type: task
          tool: summarizer
          input:
            text: "${global.search_result}"
          output: "global.summary"

        - id: extract_terms
          type: task
          tool: keyword_extractor
          input:
            text: "${global.search_result}"
          output: "global.keywords"

    - id: final_step
      type: task
      tool: composer
      input:
        summary: "${global.summary}"
        keywords: "${global.keywords}"
      output: "global.final"

  outputs:
    result: "${global.final}"
```

说明：

- `inputs`：workflow 的初始变量命名空间（也可来自 memory）。
- 每个 `task` 明确 `tool`、`input`（支持 `${path}` 插值）、`output`（写入 memory 的 path，如 `global.xxx`）。
- `parallel` 含 `branches` 子节点数组，branches 内节点可以是 task 或 nested parallel（MVP 可以先限制只一层）。
- Memory 路径约定：`global.xxx.yyy`（用点分层）。
- Node metadata（可选）: `timeout`, `retry: {count, interval}`, `meta: {}`。

# Runtime 总体架构（模块化）

模块划分（单进程、单二进制）：

1. `dsl/parser`
   - 解析 YAML → 内存中的 AST / ExecutionPlan（节点对象、依赖关系、memory refs）。
   - 校验：唯一 id、output 路径合法性、未解析变量警告。
2. `engine/executor`
   - 核心执行器，接收 ExecutionPlan，按顺序/并行执行 nodes。
   - 节点执行器 NodeExecutor：负责执行单个 node（包括参数插值、调用 tool、写 output、打 trace）。
3. `memory/store`
   - 全局 Memory 实现（见下文），提供 Set/Get/GetWithPath、Snapshot、Thread-safe）。
4. `tools/adapter`
   - Tool 注册表：`map[string]ToolFunc`，ToolFunc(ctx, inputs) -> (outputs, err)。
   - 默认提供几个 demo mock 工具：http_get（可用 net/http）、summarizer（mock）、keyword_extractor（mock）、composer（合并输出并格式化）。
5. `runtime/trace`
   - 结构化日志与 trace（JSON），用于 AI IDE 可视化与追踪。每个 node 产出 trace entry。
6. `cli` / `server`
   - 最小 CLI：`run --file example.yaml`。可选 HTTP 接口返回 trace。

# 并发与 Memory 细节（线程安全）

## Memory 设计（建议实现）

```
type Memory struct {
    mu sync.RWMutex
    data map[string]interface{}
}
```

方法：

- `Get(path string) (interface{}, bool)` // path 支持 "global.a.b"
- `Set(path string, value interface{}) error`
- `Merge(path string, value map[string]interface{}) error` // 可选
- `Snapshot() map[string]interface{}`

实现要点：

- `Get/Set` 通过 `strings.Split(path, ".")` 逐级遍历 map，写操作需要 mu.Lock()，读操作 mu.RLock()。
- 为避免并发写导致 map race（map 不是线程安全），对顶层写保护即可，但因为我们要支持 nested writes，执行 Set 时应保证创建所需中间 map，整个 Set 操作在 Lock 中完成。
- 读操作尽量用 RLock，但如果需要返回可修改的引用（map），请返回深拷贝或仅返回不可变副本。

## 并发模型

- 遇到 `parallel` 节点：executor 为每个 branch 启动一个 goroutine（使用 sync.WaitGroup）；所有 branch 共享同一个 Memory 实例（线程安全）。
- Node 执行顺序：按 DSL 顺序，遇 parallel 阻塞等待所有 branches 完成再继续到下一个 node（join）。
- 如果走 DAG（future 扩展）：可以通过构建依赖图再调度。MVP 保持线性 + parallel 容易实现。

## 并发注意点

- 写冲突：设计约定让每个 node 写入自己的 output path（例如 `global.branch1.result`），避免同一路径并发写。若必须写同一路径，runtime 应以 last-write-wins 或返回错误为策略（MVP 建议报错/警告）。
- 提供 `atomic` 操作（可选）：`memory.AtomicUpdate(path, func(old) new)`，但这可先放到后续迭代。

# 变量插值（Variable Interpolation）

实现规则：

- 语法：`${path}`，path 可以是 `inputs.x`、`global.a.b`、`outputs.prevNode`（建议限定到 `inputs` 和 `global`）。
- 在 NodeExecutor 执行前，对 node.input 的字符串参数进行解析与替换（递归处理 map[string]interface{}、[]interface{}）。
- 使用正则 `\$\{([^}]+)\}` 查找占位符，替换为 memory.Get(path) 的字符串形式（如果值不是字符串，序列化为 JSON）。
- 如果有未解析的占位符或 path 不存在，抛出执行错误或让 runtime 把值当作空字符串并记录 warning（MVP 推荐：执行失败并记录 trace 错误，便于 debug）。

# Tool 接口与适配

定义：

```
type ToolFunc func(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)
```

- 工具注册：`RegisterTool("http_get", HttpGetTool)`。
- Node 的 `tool` 字段直接 map 到 ToolFunc，传入经过插值的 inputs。Tool 返回 outputs，runtime 根据 `output` 字段写入 memory。
- Demo Tools：实现 `http_get`（简单 GET，返回 body string）、`summarizer`（mock，返回 "summary of ..."`）、`keyword_extractor`（mock，返回 list）、`composer`（合并成最终文本）。

# Trace / Logging / AI IDE 辅助信息

为了让 AI IDE 能更好理解运行过程，设计结构化 trace，每个节点产出一条 entry，例如：

```
{
  "node_id": "fetch_info",
  "type": "task",
  "tool": "http_get",
  "input": {"url": "https://..."},
  "start_time": "...",
  "end_time": "...",
  "status": "success",
  "output_path": "global.search_result",
  "output_preview": "大型文本片段...",
  "error": null
}
```

- 最终生成的 `workflow_trace.json` 包含 nodes list 与 memory snapshot，便于 demo 与 AI IDE 回放、调试、诊断。AI IDE 可据此建议 DSL 优化或替换 tool。

# 错误处理与可观察性（MVP 级）

- Node 执行异常：写入 trace（status=failed, error=...），并中断 workflow（可选配置：`continue_on_error: true`）。MVP 默认中断并返回错误。
- 日志：console + JSON trace 文件输出。
- 提供 `--dry-run` 选项：解析 DSL、验证变量引用、输出 execution plan，不执行工具调用。

# 安全/资源限制（MVP 简化）

- 对 http_get 做简单超时（context with timeout）和最大响应大小限制。
- 并发数量默认不限（依赖运行环境），但可以给 CLI 加 `--max-parallel` 参数做简单 throttling（实现可选）。

# 不要在本次 MVP 中实现（明确划界）

- 不要做分布式 worker、永久队列、checkpoint/replay、复杂 retry/backoff 策略、向量检索、权限控制、插件市场、编辑器集成（IDE 只是消费 trace）。
- 不要做深度语义解析或自然语言层次的 node（你已决定结构化 DSL，所以不做混合语义）。

# 最小实现步骤清单（可直接交付的工作项）

（下面是“要做什么”清单，不给时间预算）

1. 建立 repo skeleton：`cmd/`, `pkg/dsl/`, `pkg/engine/`, `pkg/memory/`, `pkg/tools/`, `pkg/trace/`。
2. 实现 YAML parser → ExecutionPlan（包含节点序列、parallel info、inputs）。
3. 实现 Memory（带 path 支持的 Get/Set，配合 RWMutex）。
4. 实现 Tool registry 与 3 个 demo tools（http_get、summarizer、keyword_extractor、composer）。
5. 实现 NodeExecutor（插值、调用 tool、写 output、trace）。
6. 实现 ParallelExecutor（goroutine + sync.WaitGroup），共享 Memory。
7. CLI：读取 YAML，run，输出 trace JSON & memory snapshot。
8. 基本单元测试（Memory、插值、parallel branch）。
9. 准备 example.yaml 与 demo 指令、以及 README（如何运行 & demo 流程说明）。

---

# AI IDE Prompt(已解决)

我将给你一个完整的技术方案，请按照这个方案 _从零创建一个可运行的 MVP 实现_。

要求你立即进入工程执行模式，产出可直接编译运行的 Golang 代码，最终形成一个小型但可执行的 Workflow Runtime + DSL Parser。

所有代码需要符合以下规则：

- 语言：Golang（最新稳定版）
- 构建方式：Go modules
- 目录结构清晰，包含：
  - `/dsl`（YAML 定义与解析）
  - `/runtime`（执行引擎）
  - `/memory`（线程安全 memory）
  - `/example`（一个完整 example.yaml 文件）
  - `main.go`（加载 DSL → 运行 workflow）
- 最终可运行：`go run . example/example.yaml`

你需要严格按以下“技术方案”构建：

---

### 【技术方案】

#### 1. DSL（YAML）结构（固定）

```
workflow:
  name: sample
  memory:
    initial:
      endpoint: "https://example.com"
  steps:
    - id: fetch
      type: task
      tool: http_get
      input:
        url: ${endpoint}
      output: global.raw

    - id: parallel_processing
      type: parallel
      branches:
        - id: parse
          type: task
          tool: parse_json
          input:
            source: ${global.raw}
          output: global.parsed

        - id: summarize
          type: task
          tool: summarize
          input:
            text: ${global.raw}
          output: global.summary
```

约束：

- 只有两种 node：`task`、`parallel`
- parallel 中的 branches 是一组任务，必须并发执行
- 每个 task 都有 input/output
- input 中支持 `${var}` 插值
- output 是 memory 路径（如：global.xxx）

---

#### 2. Memory（线程安全）

Memory 是一个结构化 key–value 存储，有路径支持：

- 数据结构：`map[string]interface{}`
- 使用 `sync.RWMutex` 实现线程安全
- 提供方法：
  - `Set(path string, value interface{})`
  - `Get(path string) (interface{}, error)`
  - `ResolveInterpolation(str string) string`（`${var}` 替换）

---

#### 3. Runtime 执行模型

WorkflowRuntime 需具备：

1. `LoadWorkflow(yamlFile string)`
2. `Run() error`
3. 顺序执行步骤
4. task：执行工具（tool registry）
5. parallel：使用 goroutine 并发执行 branches
   - 用 WaitGroup 等待所有分支结束
   - 所有分支共享同一 memory

---

#### 4. Tool 系统（最简）

需内置 3 个 tool，用 interface 统一：

- `http_get` → GET 请求，返回 string
- `parse_json` → 解析 JSON 返回 map[string]interface{}
- `summarize` → 简单做个“统计字数/行数”的 mock summary

工具注册方式示例：

```
type Tool interface {
    Run(input map[string]interface{}) (interface{}, error)
}

var ToolRegistry map[string]Tool
```

---

#### 5. 文件结构（强制）

```
/memory
  memory.go
/runtime
  runtime.go
  task.go
  parallel.go
/dsl
  parser.go
/tools
  http.go
  json.go
  summarize.go
/example
  example.yaml
main.go
go.mod
```

---

### 【最终要求】

你需要生成：

- 完整代码文件（分目录输出）
- 一个可直接运行的 example.yaml
- main.go 能加载 example.yaml 并执行整个 workflow

所有文件必须在我提出“继续”之前连续生成，不要做解释。

**当你理解以上全部方案后，请回复：
“已就绪，请提供我开始生成代码的信号。”**

**指令结束。**
