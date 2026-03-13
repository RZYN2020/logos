# Unified Rule Engine Specification

## Purpose

统一规则引擎为 Logos 平台提供单一、一致的日志处理规则系统。该引擎采用 `Rule = Condition + Action` 模型，支持在 Log SDK、Log Processor 和 Log Analyzer 之间共享规则定义。

## Core Principles

1. **MIT Philosophy** - Simple, composable, do one thing well
2. **Uniform DSL** - 同一套语法在所有组件中使用
3. **Hot Reload** - 规则变更实时生效
4. **Extensible** - 易于添加新的 Condition Operator 和 Action Type
5. **Client Isolation** - 不同客户端通过 Service Name + Environment 隔离配置

## Core Concepts

### Rule

规则是日志处理的基本单元，由匹配条件和执行动作组成。

```json
{
  "id": "unique-rule-id",
  "name": "规则名称",
  "description": "规则描述（可选）",
  "enabled": true,
  "match": { ... },
  "actions": [ ... ]
}
```

### Condition

条件支持单条件和复合条件两种模式，**支持任意嵌套**。

#### 单条件模式

```json
{
  "field": "level",
  "operator": "eq",
  "value": "ERROR"
}
```

#### 复合条件模式

```json
{
  "all": [ ... ]
}

// 或

{
  "any": [ ... ]
}

// 或

{
  "not": { ... }
}
```

### Action

动作定义匹配成功后对日志执行的操作。

```json
{
  "type": "mask",
  "config": { ... }
}
```

### Log Entry

日志条目是规则处理的输入。

```go
type LogEntry struct {
    Timestamp     time.Time              `json:"timestamp"`
    Level         string                 `json:"level"`
    Message       string                 `json:"message"`
    Service       string                 `json:"service"`
    Environment   string                 `json:"environment,omitempty"`
    Cluster       string                 `json:"cluster,omitempty"`
    Pod           string                 `json:"pod,omitempty"`
    TraceID       string                 `json:"trace_id,omitempty"`
    SpanID        string                 `json:"span_id,omitempty"`
    Fields        map[string]interface{} `json:"fields,omitempty"`
    Raw           string                 `json:"raw,omitempty"`
}
```

**字段访问**：
- 标准字段：`level`, `message`, `service`, `environment`, `cluster`, `pod`, `trace_id`, `span_id`, `raw`
- 自定义字段：`fields.xxx`（支持点号访问嵌套字段）

### Rule Result

规则执行结果。

```go
type RuleResult struct {
    // 规则是否匹配
    Matched bool

    // 是否保留日志
    ShouldKeep bool

    // 修改后的日志条目（未修改则为 nil）
    ModifiedEntry *LogEntry

    // 匹配的规则 ID（用于审计）
    MatchedRule string

    // 执行的动作列表（用于审计）
    ExecutedActions []string
}
```

## Condition Operators

### 比较操作符 (Comparison Operators)

| 操作符 | 说明 | 值类型 | 示例 |
|--------|------|--------|------|
| `eq` | 等于 | string/number | `{"field":"level","operator":"eq","value":"ERROR"}` |
| `ne` | 不等于 | string/number | `{"field":"level","operator":"ne","value":"DEBUG"}` |
| `gt` | 大于 | number | `{"field":"status_code","operator":"gt","value":500}` |
| `lt` | 小于 | number | `{"field":"latency_ms","operator":"lt","value":100}` |
| `ge` | 大于等于 | number | `{"field":"latency_ms","operator":"ge","value":1000}` |
| `le` | 小于等于 | number | `{"field":"status_code","operator":"le","value":499}` |

### 字符串操作符 (String Operators)

| 操作符 | 说明 | 值类型 | 示例 |
|--------|------|--------|------|
| `contains` | 包含字符串 | string | `{"field":"message","operator":"contains","value":"error"}` |
| `starts_with` | 开头匹配 | string | `{"field":"message","operator":"starts_with","value":"ERROR"}` |
| `ends_with` | 结尾匹配 | string | `{"field":"message","operator":"ends_with","value":"failed"}` |
| `matches` | 正则匹配 | string | `{"field":"message","operator":"matches","value":"password=\\w+"}` |

**正则语法**：Go RE2 语法

### 集合操作符 (Collection Operators)

| 操作符 | 说明 | 值类型 | 示例 |
|--------|------|--------|------|
| `in` | 在集合中 | array | `{"field":"level","operator":"in","value":["ERROR","WARN"]}` |
| `not_in` | 不在集合中 | array | `{"field":"service","operator":"not_in","value":["health-check"]}` |

### 存在操作符 (Existence Operators)

| 操作符 | 说明 | 值类型 | 示例 |
|--------|------|--------|------|
| `exists` | 字段存在 | 无 | `{"field":"trace_id","operator":"exists"}` |
| `not_exists` | 字段不存在 | 无 | `{"field":"password","operator":"not_exists"}` |

### 复合条件操作符 (Logical Operators)

| 操作符 | 说明 | 示例 |
|--------|------|------|
| `all` | AND - 所有条件都满足 | `{"all": [cond1, cond2]}` |
| `any` | OR - 任一条件满足 | `{"any": [cond1, cond2]}` |
| `not` | NOT - 条件不满足 | `{"not": cond}` |

**嵌套示例**：
```json
{
  "all": [
    {"field": "level", "operator": "eq", "value": "ERROR"},
    {
      "any": [
        {"field": "service", "operator": "eq", "value": "payment"},
        {"field": "service", "operator": "eq", "value": "user"}
      ]
    },
    {
      "not": {
        "field": "message",
        "operator": "contains",
        "value": "expected error"
      }
    }
  ]
}
```

## Action Types

### 流控制动作 (Flow Control)

| 类型 | 说明 | 配置 | 终止性 |
|------|------|------|--------|
| `keep` | 保留日志并终止后续规则 | 无 | 是 |
| `drop` | 丢弃日志并立即终止 | 无 | 是 |
| `sample` | 采样通过 | `rate`: 0.0-1.0 | 否 |

### 转换动作 (Transformation)

| 类型 | 说明 | 终止性 |
|------|------|--------|
| `mask` | 脱敏替换 | 否 |
| `truncate` | 裁剪长度 | 否 |
| `extract` | 提取字段 | 否 |
| `rename` | 重命名字段 | 否 |
| `remove` | 移除字段 | 否 |
| `set` | 设置字段值 | 否 |

### 元数据动作 (Metadata)

| 类型 | 说明 | 终止性 |
|------|------|--------|
| `mark` | 添加标记 | 否 |

### 动作链执行语义

Actions 按照定义的**顺序依次执行**：

- 每个 Action 接收前一个 Action 修改后的日志条目
- Action 执行返回 Go `error` 时：
  - **记录错误日志**（只记录，不暴露在结果中）
  - 跳过当前 Action
  - 继续执行后续 Action
- 遇到 `keep` 或 `drop` 时：
  - 立即终止规则链
  - 不再执行后续 Action

**匹配记录可以上报**用于审计和监控。

## ETCD 命名空间

### 键路径结构

```
/rules/
  ├── clients/
  │   ├── {service_name}.{environment}/
  │   │   ├── sdk/
  │   │   │   └── {rule_id}
  │   │   └── processor/
  │   │       └── {rule_id}
  │   └── ...
  │
  └── defaults/
      └── processor/
          └── {rule_id}
```

### 客户端标识符

客户端标识符格式：`{service_name}.{environment}`

示例：
- `payment-service.production`
- `user-service.staging`
- `api-gateway.development`

### 规则加载顺序

每个客户端加载的规则按以下顺序：

1. **SDK 规则**：
   - 从 `/rules/clients/{client_id}/sdk/` 加载

2. **Processor 规则**：
   - 从 `/rules/clients/{client_id}/processor/` 加载
   - 从 `/rules/defaults/processor/` 加载

### 规则排序

规则按**文件中定义的顺序**依次执行（从上到下）。

建议在键名中包含前缀以确保顺序：
```
001-high-priority-rule
002-medium-priority-rule
003-low-priority-rule
```

## 执行点

支持两个执行点：

| 执行点 | 位置 | 用途 |
|--------|------|------|
| `sdk` | Log SDK（发送到 Kafka 前） | 节省带宽、在客户端脱敏敏感数据 |
| `processor` | Log Processor（从 Kafka 消费后） | 处理非 SDK 来源的日志、过滤、丰富 |

查询时的规则执行不在范围内。

## Package Architecture

```
logos/
└── pkg/
    └── rule/                  # 统一规则包（被所有组件引用）
        ├── rule.go            # 数据模型
        ├── engine.go          # 规则评估引擎
        ├── condition.go       # 条件匹配器
        ├── action.go          # 动作执行器
        ├── actions/           # 标准 Action 实现
        │   ├── keep.go
        │   ├── drop.go
        │   ├── mask.go
        │   ├── truncate.go
        │   ├── sample.go
        │   ├── extract.go
        │   ├── rename.go
        │   ├── remove.go
        │   ├── set.go
        │   └── mark.go
        ├── context.go         # 规则执行上下文
        ├── registry.go        # Action/Operator 注册中心
        └── storage/           # 存储适配器
            ├── etcd.go
            ├── memory.go
            └── database.go
```

## Requirements

### Requirement: Rule Engine must evaluate conditions correctly

系统 SHALL 正确评估规则条件，支持单条件和复合条件（AND/OR/NOT），复合条件支持任意嵌套。

#### Scenario: Single condition match

- **WHEN** 日志条目满足单条件规则
- **THEN** 规则匹配成功
- **AND** 执行规则定义的 Actions

#### Scenario: ALL conditions match

- **WHEN** 规则使用 `all` 复合条件
- **AND** 所有子条件都满足
- **THEN** 规则匹配成功

#### Scenario: ANY condition matches

- **WHEN** 规则使用 `any` 复合条件
- **AND** 至少一个子条件满足
- **THEN** 规则匹配成功

#### Scenario: NOT condition

- **WHEN** 规则使用 `not` 条件
- **AND** 子条件不满足
- **THEN** 规则匹配成功

#### Scenario: Nested conditions

- **WHEN** 规则包含嵌套的复合条件
- **AND** 所有嵌套条件按语义正确评估
- **THEN** 规则匹配结果正确

### Requirement: Rule Engine must execute actions in order with best-effort error handling

系统 SHALL 按照规则定义的顺序执行 Actions，采用 best-effort 错误处理策略。

#### Scenario: Multiple actions

- **WHEN** 规则定义了多个 Actions
- **THEN** Actions 按照定义的顺序依次执行
- **AND** 每个 Action 可以修改日志条目
- **AND** 后续 Action 接收前一个 Action 修改后的条目

#### Scenario: Action fails with error

- **WHEN** Action 执行返回 Go `error`
- **THEN** 错误被记录到日志
- **AND** 当前 Action 被跳过
- **AND** 继续执行后续 Action
- **AND** 已成功的 Action 修改保留

#### Scenario: Drop action terminates immediately

- **WHEN** 执行 `drop` Action
- **THEN** 立即终止规则链
- **AND** 不再执行后续 Rules

#### Scenario: Keep action terminates rule chain

- **WHEN** 执行 `keep` Action
- **THEN** 终止后续 Rules
- **AND** 保留当前日志条目

#### Scenario: Match records are reported

- **WHEN** 规则匹配或 Action 执行
- **THEN** 匹配记录可以上报用于审计
- **AND** 错误记录可以上报用于监控

### Requirement: Rule Engine must support client isolation via service name and environment

系统 SHALL 通过 Service Name + Environment 隔离不同客户端的规则配置。

#### Scenario: Client ID format

- **WHEN** 客户端初始化规则引擎
- **THEN** 客户端标识符格式为 `{service_name}.{environment}`

#### Scenario: SDK rules loading

- **WHEN** Log SDK 加载规则
- **THEN** 从 `/rules/clients/{client_id}/sdk/` 加载规则

#### Scenario: Processor rules loading

- **WHEN** Log Processor 加载规则
- **THEN** 从 `/rules/clients/{client_id}/processor/` 加载规则
- **AND** 从 `/rules/defaults/processor/` 加载规则

### Requirement: Rule Engine must execute rules in definition order

系统 SHALL 按照规则在 ETCD 中的定义顺序依次执行。

#### Scenario: Rules execute in key order

- **WHEN** 存在多个规则
- **THEN** 按 ETCD 键名字典序执行
- **AND** 建议使用 `001-`, `002-` 前缀确保顺序

### Requirement: Rule Engine must support hot reload from ETCD

系统 SHALL 支持从 ETCD 热加载规则变更。

#### Scenario: Rule added

- **WHEN** 新 Rule 写入 ETCD
- **THEN** Rule Engine 自动加载新 Rule
- **AND** 新 Rule 立即生效

#### Scenario: Rule updated

- **WHEN** 现有 Rule 在 ETCD 中更新
- **THEN** Rule Engine 自动重新加载该 Rule
- **AND** 更新后的 Rule 立即生效

#### Scenario: Rule deleted

- **WHEN** Rule 从 ETCD 中删除
- **THEN** Rule Engine 自动移除该 Rule
- **AND** 该 Rule 不再生效

### Requirement: Rule Engine must be shareable across components

系统 SHALL 提供可被 Log SDK、Log Processor 和 Log Analyzer 共享使用的规则包。

#### Scenario: Log SDK uses rule engine

- **WHEN** Log SDK 初始化
- **AND** 配置了 ETCD 端点
- **THEN** Log SDK 加载规则引擎
- **AND** 在发送日志前评估规则

#### Scenario: Log Processor uses rule engine

- **WHEN** Log Processor 消费日志
- **THEN** 使用同一规则包评估规则
- **AND** 应用相同的规则逻辑

#### Scenario: Log Analyzer manages rules

- **WHEN** 用户通过 Log Analyzer API 创建/更新规则
- **THEN** 规则存储到数据库
- **AND** 规则同步到 ETCD
- **AND** Log SDK 和 Log Processor 自动加载新规则

### Requirement: Rule Engine must support extensible actions and operators

系统 SHALL 支持通过注册机制扩展自定义 Action 和 Operator。

#### Scenario: Register custom action

- **WHEN** 组件注册自定义 Action Type
- **THEN** 规则引擎可以执行该自定义 Action
- **AND** 自定义 Action 可以访问和修改日志条目

#### Scenario: Register custom operator

- **WHEN** 组件注册自定义 Condition Operator
- **THEN** 规则引擎可以使用该自定义 Operator 评估条件

## DSL Examples

### Example 1: 生产环境丢弃 DEBUG 日志

```json
{
  "id": "drop-debug-in-prod",
  "name": "生产环境丢弃 DEBUG 日志",
  "description": "在生产环境中丢弃所有 DEBUG 级别的日志以节省成本",
  "enabled": true,
  "match": {
    "all": [
      {"field": "level", "operator": "eq", "value": "DEBUG"},
      {"field": "environment", "operator": "eq", "value": "production"}
    ]
  },
  "actions": [
    {"type": "drop"}
  ]
}
```

### Example 2: 敏感数据脱敏

```json
{
  "id": "mask-sensitive-data",
  "name": "敏感数据脱敏",
  "description": "对密码、token 等敏感数据进行脱敏",
  "enabled": true,
  "match": {
    "any": [
      {"field": "service", "operator": "eq", "value": "payment-service"},
      {"field": "service", "operator": "eq", "value": "user-service"}
    ]
  },
  "actions": [
    {
      "type": "mask",
      "config": {
        "fields": ["message"],
        "pattern": "(?<=password=)\\w+",
        "replacement": "******"
      }
    },
    {
      "type": "mask",
      "config": {
        "fields": ["message"],
        "pattern": "(?<=token=)[\\w-]+",
        "replacement": "***REDACTED***"
      }
    },
    {
      "type": "remove",
      "config": {
        "fields": ["fields.password", "fields.secret"]
      }
    },
    {"type": "keep"}
  ]
}
```

### Example 3: 健康检查日志采样

```json
{
  "id": "sample-health-checks",
  "name": "健康检查日志采样",
  "description": "对高频健康检查日志进行 10% 采样以减少噪音",
  "enabled": true,
  "match": {
    "field": "message",
    "operator": "contains",
    "value": "/health"
  },
  "actions": [
    {
      "type": "sample",
      "config": {
        "rate": 0.1
      }
    }
  ]
}
```
