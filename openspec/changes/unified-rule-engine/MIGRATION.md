# 统一规则引擎迁移指南

本文档指导您从旧的策略/过滤系统迁移到新的统一规则引擎。

## 迁移概述

Logos 平台原本有三套不同的规则系统：
- **Log SDK**: `Hook` + `Strategy`
- **Log Processor**: `FilterRule` + `CompositeCondition`
- **Log Analyzer**: `Rule` (带 version)

现在统一为单一 `Rule = Condition + Action` 模型。

## 迁移阶段

### 阶段 1: 双写阶段

在此阶段，新旧系统并行运行，规则同时写入旧表和新 ETCD 路径。

#### Log Analyzer API 迁移

```go
// 旧代码：只写入数据库
func (h *RuleHandler) CreateRule(c *gin.Context) {
    // ... 创建规则到数据库
    h.db.Create(rule)
}

// 新代码：同时写入数据库和 ETCD
func (h *RuleHandler) CreateRule(c *gin.Context) {
    // ... 创建规则到数据库
    h.db.Create(rule)

    // 同步到 ETCD（统一规则格式）
    unifiedRule := rule.ToUnifiedRule()
    key := "/rules/clients/analyzer.default/sdk/" + rule.ID
    h.etcdCli.Put(ctx, key, unifiedRule)
}
```

### 阶段 2: 切换阶段

启用新规则引擎，逐步关闭旧系统。

#### Log SDK 迁移

**旧的配置方式：**

```go
// 旧代码 - 使用 strategy 包
import "github.com/log-system/log-sdk/pkg/strategy"

engine, err := strategy.NewEngine(cfg.EtcdEndpoints)
decision := engine.Evaluate(level, service, environment, fields)
if !decision.ShouldLog {
    return // 被策略过滤
}
```

**新的配置方式：**

```go
// 新代码 - 使用 rule 包
import "github.com/log-system/log-sdk/pkg/rule"

engine, err := rule.NewEngine(rule.Config{
    ServiceName:   cfg.ServiceName,
    Environment:   cfg.Environment,
    EtcdEndpoints: cfg.EtcdEndpoints,
})
```

#### Log Processor 迁移

**旧的配置方式：**

```go
// 旧代码 - 使用 filter 包
import "github.com/log-system/log-processor/pkg/filter"

filterEngine := filter.NewFilterEngine()
filterEngine.LoadFilters(configMgr)

result := filterEngine.ApplyFilters(entry)
if !result.ShouldKeep {
    return // 被过滤
}
```

**新的配置方式：**

```go
// 新代码 - 使用统一规则引擎
import (
    "github.com/log-system/log-processor/pkg/rule"
    "github.com/log-system/logos/pkg/rule"
)

engine, err := rule.NewEngine(rule.Config{
    ServiceName:   "log-processor",
    Environment:   getEnvString("ENVIRONMENT", "dev"),
    EtcdEndpoints: cfg.EtcdEndpoints,
})

// 转换日志条目并评估
entry := rule.NewMapLogEntry(map[string]interface{}{
    "level":   parsed.Level,
    "message": parsed.Message,
    "service": parsed.Service,
})
shouldKeep, _, _ := engine.Evaluate(entry)
if !shouldKeep {
    return // 被规则过滤
}
```

### 阶段 3: 清理阶段

移除旧规则相关代码和 ETCD 键。

## 规则 DSL 迁移

### 旧格式（Log SDK Strategy）

```json
{
  "level": "ERROR",
  "service": "api",
  "action": "drop"
}
```

### 新格式（统一规则引擎）

```json
{
  "id": "rule-001",
  "name": "Drop API Errors",
  "enabled": true,
  "condition": {
    "all": [
      {
        "field": "level",
        "operator": "eq",
        "value": "ERROR"
      },
      {
        "field": "service",
        "operator": "eq",
        "value": "api"
      }
    ]
  },
  "actions": [
    {
      "type": "drop"
    }
  ]
}
```

### 操作符映射表

| 旧操作符 | 新操作符 | 说明 |
|----------|----------|------|
| `=` | `eq` | 等于 |
| `!=` | `ne` | 不等于 |
| `>` | `gt` | 大于 |
| `<` | `lt` | 小于 |
| `>=` | `ge` | 大于等于 |
| `<=` | `le` | 小于等于 |
| `contains` | `contains` | 包含 |
| `regex` | `matches` | 正则匹配 |
| `in` | `in` | 在集合中 |

### 动作映射表

| 旧动作 | 新动作 | 说明 |
|--------|--------|------|
| `drop` | `drop` | 丢弃日志 |
| `keep` | `keep` | 保留日志 |
| `mark` | `mark` | 标记日志 |
| `mask` | `mask` | 掩码敏感数据 |
| - | `truncate` | 截断字段 |
| - | `extract` | 提取字段 |
| - | `rename` | 重命名字段 |
| - | `remove` | 删除字段 |
| - | `set` | 设置字段 |
| - | `sample` | 采样日志 |

## ETCD 命名空间迁移

### 旧命名空间

```
/analyzer/config/rules/{ruleID}
/processor/config/filters/{filterID}
/sdk/config/strategies/{strategyID}
```

### 新命名空间

```
/rules/clients/{service_name}.{environment}/sdk/{ruleID}
/rules/clients/{service_name}.{environment}/processor/{ruleID}
/rules/defaults/processor/{ruleID}
```

**示例：**

```
/rules/clients/api.prod/sdk/001-drop-debug
/rules/clients/api.prod/processor/001-mask-passwords
/rules/defaults/processor/001-add-timestamp
```

## 迁移检查清单

### Log SDK

- [ ] 更新导入：`strategy` → `rule`
- [ ] 更新配置：添加 `ServiceName` 和 `Environment`
- [ ] 验证规则评估逻辑
- [ ] 测试 ETCD 热加载

### Log Processor

- [ ] 更新导入：`filter` → `rule`
- [ ] 更新主处理流程
- [ ] 测试规则过滤效果
- [ ] 验证性能基准

### Log Analyzer

- [ ] 更新 models 转换方法
- [ ] 更新 handlers ETCD 同步
- [ ] 测试规则 CRUD
- [ ] 验证规则同步

## 回滚计划

如果迁移过程中遇到问题，可以回滚到旧系统：

1. 设置环境变量 `USE_LEGACY_STRATEGY=true`
2. 重启服务
3. 旧系统将自动加载旧的规则配置

## 性能影响

新规则引擎的性能基准：

- 单条件评估：~50ns/op
- 复合条件（3 层嵌套）：~200ns/op
- 完整规则链（10 条规则）：~2μs/op
- 带转换动作：~500ns/op per action

## 常见问题

### Q: 旧的规则配置会自动转换吗？

A: 不会自动转换。需要手动将旧规则按照新 DSL 格式重新创建。Log Analyzer 提供了 `ToUnifiedRule()` 方法辅助转换。

### Q: 迁移期间会影响现有服务吗？

A: 双写阶段不会影响现有服务。切换阶段需要重启服务。

### Q: 如何验证迁移成功？

A: 运行集成测试并检查日志是否正常输出，验证规则是否正确生效。

## 联系支持

如有问题，请联系 Logos 团队或提交 Issue。
