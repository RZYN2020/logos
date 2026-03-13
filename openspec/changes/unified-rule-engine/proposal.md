## Why

当前 Logos 平台存在五套不同的规则/DSL 系统（Hook、Strategy、FilterRule、CompositeCondition、Rule），分散在 Log SDK、Log Processor 和 Log Analyzer 三个组件中，配置不统一，维护成本高，用户体验混乱。

## What Changes

- 统一所有规则为单一的 `Rule = Condition + Action` 模型
- 创建共享的 `pkg/rule` 包，被所有组件引用
- 统一 ETCD 命名空间，支持按 `{service_name}.{environment}` 隔离客户端配置
- 实现 12+ 种标准 Action 类型（keep, drop, sample, mask, truncate, extract, rename, remove, set, mark）
- 实现 10+ 种 Condition Operator（eq, ne, gt, lt, ge, le, contains, starts_with, ends_with, matches, in, not_in, exists, not_exists）
- 支持复合条件任意嵌套（all/any/not）
- 实现 best-effort 错误处理策略
- 支持 Action/Operator 扩展机制
- 支持热加载规则变更

## Capabilities

### New Capabilities

- `unified-rule-engine`: 统一的规则引擎，提供 Condition + Action DSL，被 Log SDK、Log Processor 和 Log Analyzer 共享使用

### Modified Capabilities

- `log-sdk`: 移除 `strategy` 包，改用 `pkg/rule` 包
- `kubernetes-deployment`: 更新以适配新的统一规则引擎

## Impact

- **Affected code**: `log-sdk/`, `log-processor/`, `log-analyzer/`
- **New package**: `pkg/rule/`（统一规则包）
- **ETCD structure**: 迁移到新的 `/rules/` 命名空间
- **APIs**: Log Analyzer API 更新以支持统一规则
