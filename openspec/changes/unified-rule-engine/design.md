## Context

当前 Logos 平台有五套不同的规则系统：
- Log SDK: `Hook` + `Strategy`
- Log Processor: `FilterRule` + `CompositeCondition`
- Log Analyzer: `Rule` (带 version)

这些系统使用不同的 DSL，存储在不同的 ETCD 路径，导致配置分散、维护成本高、用户体验混乱。

## Goals / Non-Goals

**Goals:**
- 统一所有规则为单一 `Rule = Condition + Action` 模型
- 创建共享的 `pkg/rule` 包，被所有组件引用
- 支持 ETCD 热加载和按 `{service_name}.{environment}` 隔离
- 提供 10+ 种标准 Operator 和 12+ 种标准 Action
- 支持 Action/Operator 扩展机制
- 提供完整的迁移路径

**Non-Goals:**
- 不改变日志的 Kafka 传输格式
- 不实现查询时的规则应用
- 不改变现有 Log Entry 数据模型（除了增加字段访问支持）

## Decisions

### Decision 1: Condition DSL - Explicit with Nested Support
**选择**: 使用显式的 `{"field": "...", "operator": "...", "value": "..."}` 格式，支持 `all`/`any`/`not` 任意嵌套

**理由**:
- 显式格式更清晰，类型安全
- 嵌套支持提供强大的表达能力
- 与我们探索模式中的决策一致

**替代方案**: 简化格式 `{"level": "ERROR"}` - 不够灵活

---

### Decision 2: ETCD 命名空间 - Service + Environment 隔离
**选择**: `/rules/clients/{service_name}.{environment}/{sdk,processor}/{ruleID}` + `/rules/defaults/processor/`

**理由**:
- 清晰的隔离边界
- 支持默认规则
- 与我们探索模式中的决策一致

**替代方案**: 单一扁平命名空间 - 不支持多租户隔离

---

### Decision 3: Action 错误处理 - Best-Effort
**选择**: Action 执行失败时记录日志，跳过当前 Action，继续执行后续 Action

**理由**:
- 符合 "best-effort" 理念
- 不因为一个 Action 失败而丢弃整个日志
- 错误记录可用于监控

**替代方案**: Fail-Fast - 过于严格，可能导致日志丢失

---

### Decision 4: 规则排序 - ETCD 键名顺序
**选择**: 按 ETCD 键名字典序执行，建议使用 `001-`, `002-` 前缀

**理由**:
- 简单，无需额外字段
- 与我们探索模式中的决策一致

**替代方案**: Priority 字段 - 增加复杂度

---

### Decision 5: 新包位置 - `logos/pkg/rule`
**选择**: 在项目根目录创建 `pkg/rule` 包，被所有组件共享引用

**理由**:
- 避免循环依赖
- 清晰的共享包位置
- Go 项目的常见模式

**替代方案**: 放在某个组件目录下 - 会导致依赖混乱

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| 迁移期间旧规则可能失效 | 高 | 提供双写/双读阶段，逐步迁移 |
| ETCD Watch 可能丢失事件 | 中 | 同时支持定时刷新 (30s) 作为兜底 |
| 新包可能引入循环依赖 | 中 | 保持 `pkg/rule` 无外部依赖（除了标准库和 etcd 客户端） |
| Action 执行可能影响性能 | 中 | 提供性能基准测试，正则缓存 |

## Migration Plan

### Phase 1: 创建新包
- 创建 `pkg/rule` 包
- 实现数据模型、Condition 匹配器、Action 执行器
- 实现 ETCD 存储适配器

### Phase 2: 双写阶段
- Log Analyzer API 同时写入旧表和新 ETCD 路径
- Log SDK/Processor 同时支持旧配置和新配置（配置开关）

### Phase 3: 切换阶段
- 默认启用新规则引擎
- 提供迁移工具

### Phase 4: 清理阶段
- 移除旧规则相关代码
- 移除旧 ETCD 键

## Open Questions

1. 是否需要提供规则从旧格式到新格式的自动迁移工具？
2. 性能目标是什么？（单条规则评估延迟预期）
3. 是否需要规则验证 DSL？
