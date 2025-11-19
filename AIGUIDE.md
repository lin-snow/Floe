# **Floe v0.4 — AI IDE Implementation Guide**

目标是在现有 v0.3 基础上，为 Floe 增加 **条件执行（Conditionals）** 和 **动态分支（Dynamic Routing）** 功能，同时保持 DSL 的结构化风格、运行时的超步模型、以及可追踪性。

---

## **1. 目标功能**

在 v0.4，你需要让 Floe 支持：

### **1. Conditions（if）**

允许每个 Step 指定 `when` 字段，判断当前内存、消息、输入等是否满足条件，若不满足则跳过该 step。

示例：

```
steps:
  fetch_user:
    action: http.get
    args:
      url: "https://api/user"

  process_premium_user:
    when: "${memory.user.type} == 'premium'"
    action: logic.processPremium
```

### **2. Dynamic next routing（动态分支）**

允许 Step 的 `next` 字段动态生成。

示例：

```
next:
  on_success: "notify"
  on_failure: "retry_fetch"
```

或 DSL 表示：

```
next:
  route: "${memory.fetch.status == 200 ? 'process' : 'retry'}"
```

### **3. Runtime 变化**

Scheduler 增加：

- 条件判断逻辑
- 动态路由解析
- 生成新的活跃 node 集合

Superstep 模型保持不变。

### **4. Trace 扩展**

在 trace 中记录：

- 条件是否被触发
- 路由的计算结果
- 被跳过的 steps

---

## **2. DSL 需要你实现的变动**

扩展 `Step` 结构：

```
type Step struct {
    Name      string            `yaml:"-"`
    Action    string            `yaml:"action"`
    Args      map[string]any    `yaml:"args"`
    When      string            `yaml:"when"`
    Next      any               `yaml:"next"` // string | map[string]string
    Messages  map[string]any    `yaml:"messages"`
}
```

新增的部分：

1. **When 字段**：字符串表达式，true 执行，false 跳过。
2. **Next 字段**可允许：
   - string: 固定 next
   - map: 动态选择 next
   - expression: 返回 step 名

表达式解析仍沿用 `${ }` 模式。

---

## **3. Runtime 详细要求**

### **3.1 Condition Logic（在调度前进行）**

在 scheduler 中增加：

- EvaluateCondition(step, memory) → bool
- 若 false，则将该 step 标记为 “skipped” 并写入 trace

你需要在 scheduler 开始本轮 superstep 调度前判断哪些 step 应执行。

### **3.2 Dynamic Routing**

在解析 `next` 时需要支持三类：

- **string**：直接返回
- **map**：基于运行结果或 messages 判断属于哪个 key
- **expression**：返回 step 名称

运行时的规则：

优先级 = map.route > string > expression

### **3.3 超步模型集成**

保持 superstep：

- single step 不变
- 合并 outputs → 更新 memory
- 合并 messages → 更新 memory
- 根据 next 结果生成下一轮 active set

---

## **4. Trace 扩展要求**

trace.json 每个 step 需要新增字段：

```
{
  "step": "process_premium_user",
  "status": "skipped | executed",
  "condition": {
    "raw": "${memory.user.type} == 'premium'",
    "result": true
  },
  "routing": {
    "raw": "memory.fetch.status == 200 ? 'process' : 'retry'",
    "result": "process"
  }
}
```
