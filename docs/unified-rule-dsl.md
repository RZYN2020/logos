# Unified Rule DSL - 语法语义设计

## 概述

Unified Rule DSL 是 Logos 平台统一的日志处理规则语言，采用 `Rule = Condition + Action` 模型。

**设计哲学**：MIT - Simple, Composable, Do One Thing Well

---

## 目录

1. [数据模型](#数据模型)
2. [Condition DSL](#condition-dsl)
3. [Action DSL](#action-dsl)
4. [ETCD 命名空间](#etcd-命名空间)
5. [执行语义](#执行语义)
6. [扩展机制](#扩展机制)
7. [完整示例](#完整示例)

---

## 数据模型

### Rule（规则）

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

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 全局唯一标识符 |
| `name` | string | 是 | 人类可读名称 |
| `description` | string | 否 | 规则描述 |
| `enabled` | boolean | 是 | 是否启用 |
| `match` | Condition | 是 | 匹配条件 |
| `actions` | Action[] | 是 | 执行动作列表 |

### Condition（条件）

条件支持**单条件**和**复合条件**两种模式。

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

复合条件**支持任意嵌套**。

### Action（动作）

```json
{
  "type": "mask",
  "config": { ... }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 动作类型 |
| `config` | object | 否 | 动作配置 |

---

## Condition DSL

### 字段访问

条件可以访问以下日志字段：

| 字段名 | 说明 |
|--------|------|
| `level` | 日志级别 (DEBUG/INFO/WARN/ERROR/FATAL/PANIC) |
| `message` | 日志消息 |
| `service` | 服务名 |
| `environment` | 环境 |
| `cluster` | 集群名 |
| `pod` | Pod 名 |
| `trace_id` | 追踪 ID |
| `span_id` | Span ID |
| `raw` | 原始日志字符串 |
| `fields.xxx` | 自定义字段（支持点号访问嵌套） |

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

---

## Action DSL

### 标准动作类型

#### 流控制动作 (Flow Control)

| 类型 | 说明 | 配置 | 终止性 |
|------|------|------|--------|
| `keep` | 保留日志并终止后续规则 | 无 | 是 |
| `drop` | 丢弃日志并立即终止 | 无 | 是 |
| `sample` | 采样通过 | `rate`: 0.0-1.0 | 否 |

##### `keep` - 保留日志
```json
{"type": "keep"}
```
- 保留当前日志
- 终止后续规则执行

##### `drop` - 丢弃日志
```json
{"type": "drop"}
```
- 丢弃当前日志
- 立即终止规则链

##### `sample` - 采样
```json
{
  "type": "sample",
  "config": {
    "rate": 0.1
  }
}
```
- `rate`：采样率，0.0-1.0
- 不终止规则链（除非后续有 `keep`/`drop`）

---

#### 转换动作 (Transformation)

| 类型 | 说明 | 终止性 |
|------|------|--------|
| `mask` | 脱敏替换 | 否 |
| `truncate` | 裁剪长度 | 否 |
| `extract` | 提取字段 | 否 |
| `rename` | 重命名字段 | 否 |
| `remove` | 移除字段 | 否 |
| `set` | 设置字段值 | 否 |

##### `mask` - 脱敏替换
```json
{
  "type": "mask",
  "config": {
    "fields": ["message", "fields.user_email"],
    "pattern": "(?<=password=)\\w+",
    "replacement": "******"
  }
}
```
- `fields`：要脱敏的字段列表
- `pattern`：正则表达式（Go RE2）
- `replacement`：替换字符串

##### `truncate` - 裁剪长度
```json
{
  "type": "truncate",
  "config": {
    "field": "message",
    "max_length": 1000,
    "suffix": "..."
  }
}
```
- `field`：要裁剪的字段
- `max_length`：最大长度
- `suffix`：后缀（可选，默认 "..."）

##### `extract` - 提取字段
```json
{
  "type": "extract",
  "config": {
    "source": "message",
    "target": "request_id",
    "pattern": "request_id=(\\w+)"
  }
}
```
- `source`：源字段
- `target`：目标字段
- `pattern`：正则表达式（第一个捕获组的值）

##### `rename` - 重命名字段
```json
{
  "type": "rename",
  "config": {
    "from": "old_field",
    "to": "new_field"
  }
}
```
- `from`：原字段名
- `to`：新字段名

##### `remove` - 移除字段
```json
{
  "type": "remove",
  "config": {
    "fields": ["password", "secret"]
  }
}
```
- `fields`：要移除的字段列表

##### `set` - 设置字段值
```json
{
  "type": "set",
  "config": {
    "field": "processed_by",
    "value": "unified-rule-engine"
  }
}
```
- `field`：字段名
- `value`：字段值（支持任意类型）

---

#### 元数据动作 (Metadata)

| 类型 | 说明 | 终止性 |
|------|------|--------|
| `mark` | 添加标记 | 否 |

##### `mark` - 添加标记
```json
{
  "type": "mark",
  "config": {
    "tags": ["sensitive", "needs-review"],
    "metadata": {
      "reviewed": false,
      "priority": "high"
    }
  }
}
```
- `tags`：标签列表（可选）
- `metadata`：元数据键值对（可选）

---

### 动作链执行

Actions 按照定义的**顺序依次执行**：

```json
"actions": [
  {"type": "mask", "config": {...}},
  {"type": "truncate", "config": {...}},
  {"type": "keep"}
]
```

**执行语义**：
1. 每个 Action 接收前一个 Action 修改后的日志条目
2. Action 执行返回 `error` 时：
   - 记录错误日志
   - 跳过当前 Action
   - 继续执行后续 Action
3. 遇到 `keep` 或 `drop` 时：
   - 立即终止规则链
   - 不再执行后续 Action

---

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

### 完整键路径示例

```
/rules/clients/payment-service.production/sdk/mask-sensitive-data
/rules/clients/payment-service.production/processor/drop-debug
/rules/defaults/processor/common-sampling
```

---

## 执行语义

### 规则加载顺序

每个客户端加载的规则按以下顺序：

1. **SDK 规则**：
   - 从 `/rules/clients/{client_id}/sdk/` 加载

2. **Processor 规则**：
   - 从 `/rules/clients/{client_id}/processor/` 加载
   - 从 `/rules/defaults/processor/` 加载

### 规则排序

规则按**文件中定义的顺序**依次执行（从上到下）。

建议在键名中包含优先级前缀以确保顺序：
```
001-high-priority-rule
002-medium-priority-rule
003-low-priority-rule
```

### 单条规则执行流程

```
┌─────────────────────────────────────────────────────────┐
│  1. 评估 Condition                                      │
│     - 不匹配 → 跳过当前规则，继续下一条                 │
│     - 匹配 → 继续执行                                   │
└────────────────────┬────────────────────────────────────┘
                     │ 匹配
                     ▼
┌─────────────────────────────────────────────────────────┐
│  2. 顺序执行 Actions                                    │
│     ┌─────────────────────────────────────────────┐   │
│     │ 对于每个 Action:                            │   │
│     │  - 执行 Action                              │   │
│     │  - 成功？→ 应用修改，继续下一个 Action      │   │
│     │  - 失败？→ 记录日志，跳过，继续下一个       │   │
│     │  - 是 keep/drop？→ 终止规则链              │   │
│     └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 结果语义

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

### 审计与监控

规则引擎需要记录以下可观测性数据：

| 指标 | 说明 |
|------|------|
| `rules_total` | 规则总数 |
| `rules_enabled` | 启用的规则数 |
| `evaluations_total` | 评估次数 |
| `matches_total` | 匹配次数 |
| `actions_executed_total` | 动作执行次数 |
| `actions_failed_total` | 动作失败次数 |
| `logs_kept_total` | 保留日志数 |
| `logs_dropped_total` | 丢弃日志数 |

---

## 扩展机制

### 自定义操作符 (Custom Operators)

实现 `Operator` 接口：

```go
type Operator interface {
    Name() string
    Evaluate(fieldValue interface{}, conditionValue interface{}) bool
}

// 注册
rule.RegisterOperator(&MyCustomOperator{})
```

### 自定义动作 (Custom Actions)

实现 `Action` 接口：

```go
type Action interface {
    Type() string
    Execute(entry *LogEntry, config map[string]interface{}) (*LogEntry, error)
}

// 注册
rule.RegisterAction(&MyCustomAction{})
```

---

## 完整示例

### 示例 1：生产环境丢弃 DEBUG 日志

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

### 示例 2：敏感数据脱敏

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

### 示例 3：健康检查日志采样

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

---

## 附录

### A. Go RE2 正则语法参考

参见：https://github.com/google/re2/wiki/Syntax

### B. 保留字段列表

- `level`
- `message`
- `service`
- `environment`
- `cluster`
- `pod`
- `trace_id`
- `span_id`
- `raw`
- `fields.*`

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2026-03-13 | 初始版本 |
