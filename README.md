# Floe

# Floe Workflow Engine

Floe 是一个轻量级的、基于 Golang 的工作流执行引擎。它支持 YAML 定义的工作流，具备并发执行、变量插值、工具扩展和执行跟踪等功能。

## 架构与设计思路

Floe 的设计遵循模块化和可扩展的原则，主要包含以下核心组件：

### 1. DSL 解析 (dsl)

- **职责**: 负责解析 YAML 格式的工作流定义文件。
- **实现**: 使用 `spf13/viper` 库进行 YAML 解析。
- **核心结构**:
  - `Workflow`: 包含工作流名称、内存配置和步骤列表。
  - `Step`: 定义单个执行步骤，支持 `task` (任务) 和 `parallel` (并行) 两种类型。
  - `Next`: 支持流程控制，允许跳转到指定步骤。
  - `Messages`: 支持步骤间的消息传递。

### 2. 运行时环境 (runtime)

- **职责**: 负责工作流的生命周期管理、调度和执行。
- **核心组件**:
  - **Superstep Runtime**: 采用 Superstep (超步) 模式，将并发步骤作为一个整体执行，步骤间相互隔离，直到 Superstep 结束才合并结果。
  - **Scheduler**: 调度器，决定当前 Superstep 应该执行哪些步骤。支持基于 `next` 指针的流程控制。
  - **Executor**: 执行器，负责运行单个步骤，包括变量插值、工具调用和结果处理。
  - **Trace**: 跟踪系统，记录每个步骤的输入、输出、消息和耗时，最终生成 `trace.json` 用于调试和可视化。

### 3. 内存管理 (memory)

- **职责**: 提供线程安全的全局变量存储。
- **特性**:
  - **Thread-Safe**: 使用 `sync.RWMutex` 保证并发读写的安全性。
  - **Path Access**: 支持通过点号分隔的路径 (e.g., `global.user.name`) 访问和设置嵌套数据。
  - **Interpolation**: 支持 `${variable}` 语法的变量插值，在运行时动态替换字符串中的变量。

### 4. 工具系统 (tools)

- **职责**: 提供可扩展的工具执行能力。
- **接口**: 定义了 `Tool` 接口，所有工具必须实现 `Run` 方法。
- **内置工具**:
  - `http_get`: 执行 HTTP GET 请求。
  - `parse_json`: 解析 JSON 字符串。
  - `summarize`: 模拟文本摘要功能。
- **注册机制**: 通过 `Registry` 进行工具注册和查找，方便扩展新工具。

## 快速开始

### 1. 编写工作流 (example.yaml)

```yaml
workflow:
  name: demo
  steps:
    - id: hello
      type: task
      tool: summarize
      input:
        text: "Hello Floe"
      output: global.result
```

### 2. 运行

```bash
go run . example.yaml
```

### 3. 查看结果

执行完成后，会生成 `trace.json` 文件，包含详细的执行记录。

## 目录结构

- `/dsl`: YAML 解析逻辑
- `/runtime`: 核心执行引擎、调度器、Trace
- `/memory`: 内存管理
- `/tools`: 工具接口与实现
- `/example`: 示例工作流
- `main.go`: 程序入口
