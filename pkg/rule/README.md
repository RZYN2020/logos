# Unified Rule Engine

统一规则引擎是 Logos 平台的核心组件，提供强大的日志过滤、转换和增强功能。

## 特性

- **统一模型**: 单一 `Rule = Condition + Action` 模型，替代原有的三套规则系统
- **丰富的操作符**: 14+ 种条件操作符（eq, ne, gt, lt, ge, le, contains, starts_with, ends_with, matches, in, not_in, exists, not_exists）
- **复合条件**: 支持 `all`/`any`/`not` 任意嵌套
- **灵活的动作**: 10+ 种内置动作（keep, drop, sample, mask, truncate, extract, rename, remove, set, mark）
- **热加载**: 基于 ETCD Watch 的规则热加载，30 秒定时刷新兜底
- **可扩展**: 支持自定义操作符和动作注册
- **多租户隔离**: 按 `{service_name}.{environment}` 隔离规则配置

## 快速开始

### 安装

```bash
go get github.com/log-system/logos/pkg/rule
```

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/log-system/logos/pkg/rule"
)

func main() {
    // 创建规则引擎
    engine := rule.NewRuleEngine(rule.RuleEngineConfig{
        EnableAudit: true,
        EnableStats: true,
    })

    // 定义规则
    r := &rule.Rule{
        ID:      "drop-debug",
        Name:    "Drop Debug Logs",
        Enabled: true,
        Condition: rule.Condition{
            Field:    "level",
            Operator: rule.OpEq,
            Value:    "DEBUG",
        },
        Actions: []rule.ActionDef{
            {Type: rule.ActionDrop},
        },
    }

    // 加载规则
    engine.SetRules([]*rule.Rule{r})

    // 评估日志
    entry := rule.NewMapLogEntry(map[string]interface{}{
        "level":   "DEBUG",
        "message": "debug message",
    })

    shouldKeep, results, errors := engine.Evaluate(entry)

    fmt.Printf("Should keep: %v\n", shouldKeep)
    fmt.Printf("Matched rules: %d\n", len(results))
    fmt.Printf("Errors: %v\n", errors)
}
```

## 条件 DSL

### 单条件

```json
{
  "field": "level",
  "operator": "eq",
  "value": "ERROR"
}
```

### 复合条件（AND）

```json
{
  "all": [
    {"field": "level", "operator": "eq", "value": "ERROR"},
    {"field": "service", "operator": "eq", "value": "api"}
  ]
}
```

### 复合条件（OR）

```json
{
  "any": [
    {"field": "level", "operator": "eq", "value": "ERROR"},
    {"field": "level", "operator": "eq", "value": "PANIC"}
  ]
}
```

### 复合条件（NOT）

```json
{
  "not": {
    "field": "environment",
    "operator": "eq",
    "value": "dev"
  }
}
```

### 嵌套复合条件

```json
{
  "all": [
    {"field": "level", "operator": "eq", "value": "ERROR"},
    {
      "any": [
        {"field": "service", "operator": "eq", "value": "api"},
        {"field": "service", "operator": "eq", "value": "worker"}
      ]
    }
  ]
}
```

## 动作类型

### 流控制动作

| 动作 | 说明 | 配置 |
|------|------|------|
| `keep` | 保留日志并终止规则链 | - |
| `drop` | 丢弃日志并终止规则链 | - |
| `sample` | 按采样率保留日志 | `rate`: 采样率 (0.0-1.0) |

### 转换动作

| 动作 | 说明 | 配置 |
|------|------|------|
| `mask` | 掩码敏感数据 | `field`: 字段名，`pattern`: 正则（可选） |
| `truncate` | 截断长字段 | `field`: 字段名，`max_length`: 最大长度，`suffix`: 后缀 |
| `extract` | 提取子串到新字段 | `source_field`: 源字段，`target_field`: 目标字段，`pattern`: 正则 |
| `rename` | 重命名字段 | `from`: 原字段名，`to`: 新字段名 |
| `remove` | 删除字段 | `field`: 字段名 或 `fields`: 字段列表 |
| `set` | 设置字段值 | `field`: 字段名，`value`: 值 |

### 元数据动作

| 动作 | 说明 | 配置 |
|------|------|------|
| `mark` | 添加标记元数据 | `field`: 标记字段名，`value`: 标记值，`reason`: 原因 |

## 使用 ETCD 存储

```go
package main

import (
    "github.com/log-system/logos/pkg/rule"
    "github.com/log-system/logos/pkg/rule/storage"
)

func main() {
    // 创建 ETCD 存储
    etcdStorage, err := storage.NewETCDStorage(storage.ETCDStorageConfig{
        Endpoints:       []string{"localhost:2379"},
        Namespace:       "/rules/clients/api.prod/sdk",
        DialTimeout:     5 * time.Second,
        RefreshDuration: 30 * time.Second,
    })
    if err != nil {
        panic(err)
    }
    defer etcdStorage.Close()

    // 创建引擎并从 ETCD 加载规则
    engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
    if err := engine.LoadRules(etcdStorage); err != nil {
        panic(err)
    }

    // 规则会自动从 ETCD 热加载
    // ... 使用 engine.Evaluate() 评估日志
}
```

## ETCD 命名空间

规则存储在 ETCD 中的路径格式：

```
/rules/clients/{service_name}.{environment}/{component}/{rule_id}
```

示例：
- `/rules/clients/api.prod/sdk/001-drop-debug`
- `/rules/clients/api.prod/processor/001-mask-passwords`
- `/rules/defaults/processor/001-add-timestamp`

## 性能基准

运行基准测试：

```bash
cd pkg/rule
go test -bench=. -benchmem
```

典型性能数据：

| 操作 | 延迟 |
|------|------|
| 单条件评估 | ~50ns/op |
| 复合条件（3 层嵌套） | ~200ns/op |
| 完整规则链（10 条规则） | ~2μs/op |
| 带转换动作 | ~500ns/op per action |

## 测试

```bash
# 运行单元测试
go test -v ./...

# 运行集成测试
go test -v -run Integration

# 运行基准测试
go test -bench=. -benchmem
```

## 扩展

### 自定义操作符

```go
type MyOperator struct{}

func (o *MyOperator) Evaluate(actual, expected interface{}) (bool, error) {
    // 实现自定义比较逻辑
    return actual == expected, nil
}

// 注册操作符
evaluator := rule.NewConditionEvaluator()
evaluator.RegisterOperator("my_op", &MyOperator{})
```

### 自定义动作

```go
type MyAction struct{}

func (a *MyAction) Name() string {
    return "my_action"
}

func (a *MyAction) Execute(entry rule.LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
    // 实现自定义动作逻辑
    return true, nil, nil
}

// 注册动作
executor := rule.NewActionExecutor()
executor.RegisterAction(&MyAction{})
```

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Rule Engine                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │   Condition     │  │     Action      │                  │
│  │   Evaluator     │──▶│    Executor    │                  │
│  └─────────────────┘  └─────────────────┘                  │
│         │                    │                              │
│         ▼                    ▼                              │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │   Operators     │  │    Handlers     │                  │
│  │ - eq, ne, gt    │  │ - keep, drop    │                  │
│  │ - contains      │  │ - mask, set     │                  │
│  │ - matches, in   │  │ - extract, ...  │                  │
│  └─────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Storage Layer                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │   Memory        │  │     ETCD        │                  │
│  │   (testing)     │  │  (production)   │                  │
│  └─────────────────┘  └─────────────────┘                  │
│                        - Watch                             │
│                        - Refresh                           │
└─────────────────────────────────────────────────────────────┘
```

## 相关文件

- [MIGRATION.md](../../../openspec/changes/unified-rule-engine/MIGRATION.md) - 迁移指南
- [proposal.md](../../../openspec/changes/unified-rule-engine/proposal.md) - 提案文档
- [design.md](../../../openspec/changes/unified-rule-engine/design.md) - 设计文档
- [specs/](../../../openspec/changes/unified-rule-engine/specs/) - 规格说明
- [tasks.md](../../../openspec/changes/unified-rule-engine/tasks.md) - 任务清单

## 许可证

MIT
