# Floe

**Floe** 是一个轻量级的、基于 Golang 的 Agentic 工作流引擎。它专为构建 AI Agent 工作流而设计，支持 YAML 定义、并发执行、动态路由、实时 TUI 监控和详细的执行跟踪。

## ✨ 核心特性 (v0.5)

- **声明式工作流**: 使用 YAML 定义工作流，清晰易读。
- **并发执行**: 支持 `parallel` 步骤，自动管理并发与结果聚合。
- **动态路由**: 支持基于条件 (`when`) 和动态指针 (`next`) 的复杂流程控制。
- **实时 TUI**: 内置终端用户界面，支持实时监控执行状态、查看日志和变量。
- **事件驱动**: 基于事件流的运行时架构，支持解耦的监控与交互。
- **执行跟踪**: 自动生成 `trace.json`，完整记录输入、输出、路由决策和错误信息。
- **容错机制**: 支持重试 (Retry)、超时 (Timeout) 和降级 (Fallback) 策略。

## 🚀 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/lin-snow/Floe.git
cd Floe

# 编译
go build -o floe.exe ./cmd/floe
```

### 使用指南

Floe 提供了命令行工具来管理和运行工作流。

#### 1. 运行工作流 (Headless 模式)

适用于后台任务或脚本集成。

```bash
./floe.exe run example/05_conditionals_routing.yaml
```

#### 2. 启动 TUI (交互模式)

启动终端界面，实时可视化执行过程。

```bash
# 指定文件运行
./floe.exe tui --file example/05_conditionals_routing.yaml

# 交互式选择文件
./floe.exe tui
```

**TUI 操作:**

- `↑/↓`: 浏览步骤列表
- `q` / `Ctrl+C`: 退出

### 编写工作流

创建一个 `workflow.yaml` 文件：

```yaml
workflow:
  name: demo-workflow
  steps:
    - id: step1
      type: task
      tool: summarize
      input:
        text: "Hello Floe"
      output: global.result
      next: step2

    - id: step2
      type: task
      tool: http_get
      when: "${global.result} != ''" # 条件执行
      input:
        url: "https://api.example.com"
```

## 🏗️ 架构设计

Floe 采用模块化分层架构，核心组件如下：

### 1. 命令行接口 (cmd/floe)

- **Entry Point**: 基于 `cobra` 构建的 CLI 入口。
- **Commands**:
  - `run`: 启动 Headless 运行时。
  - `tui`: 初始化 TUI 应用并启动运行时。

### 2. 运行时 (runtime)

- **WorkflowRuntime**: 核心引擎，管理生命周期。
- **Scheduler**: 动态调度器，解析 `next` 指针和 `when` 条件，计算下一步骤。
- **Superstep**: 并发执行单元，确保步骤间的隔离性。
- **Event System**: (`internal/runtime_integration`) 基于 Channel 的事件总线，解耦运行时与 UI。

### 3. 用户界面 (internal/tui)

- **Framework**: 基于 `bubbletea` (ELM 架构) 和 `lipgloss` (样式)。
- **Components**:
  - **Step List**: 显示步骤状态 (Running, Executed, Skipped, Failed)。
  - **Details Panel**: 实时显示日志、输入输出和错误信息。
  - **Variables**: 监控全局内存变化。

### 4. 核心模块

- **DSL**: (`dsl/`) YAML 解析器，支持变量插值语法 `${var}`。
- **Memory**: (`memory/`) 线程安全的键值存储，支持点号路径访问 (`user.name`)。
- **Tools**: (`tools/`) 可扩展的工具接口，内置 HTTP、JSON 解析等工具。
- **Expr**: (`expr/`) 安全的表达式求值引擎，用于条件判断和动态路由。

## 📂 目录结构

```
Floe/
├── cmd/floe/           # CLI 入口 (main, root, run, tui)
├── dsl/                # YAML 解析与结构定义
├── example/            # 示例工作流
├── expr/               # 表达式求值引擎
├── internal/
│   ├── runtime_integration/ # 运行时事件定义
│   └── tui/            # 终端 UI 实现 (App, Model, Layout)
├── memory/             # 全局内存管理
├── runtime/            # 核心执行引擎 (Runtime, Scheduler, Trace)
├── tools/              # 工具接口与实现
└── main.go             # (Legacy) 旧入口，保留用于兼容
```
