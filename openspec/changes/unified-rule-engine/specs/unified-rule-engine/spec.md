## ADDED Requirements

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
