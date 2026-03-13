## 1. 创建共享包结构

- [x] 1.1 在项目根目录创建 `pkg/rule` 目录
- [x] 1.2 在 `pkg/rule` 下创建子目录：`actions/`、`storage/`
- [x] 1.3 创建 `go.mod`（如果需要）或更新项目根目录的 `go.mod`

## 2. 实现数据模型

- [x] 2.1 在 `rule.go` 中实现 `Rule`、`Condition`、`Action`、`LogEntry`、`RuleResult` 类型
- [x] 2.2 添加必要的 JSON 序列化/反序列化方法
- [x] 2.3 添加数据验证方法

## 3. 实现 Condition 匹配器

- [x] 3.1 在 `condition.go` 中实现单条件匹配（所有 14+ 种操作符）
- [x] 3.2 实现复合条件匹配（`all`/`any`/`not`，支持任意嵌套）
- [x] 3.3 实现字段访问器（支持点号访问嵌套字段 `fields.xxx`）
- [x] 3.4 添加 `Operator` 接口和注册机制

## 4. 实现 Action 执行器

- [x] 4.1 在 `action.go` 中实现 Action 执行框架
- [x] 4.2 在 `actions/` 目录中实现流控制动作：`keep.go`、`drop.go`、`sample.go`
- [x] 4.3 在 `actions/` 目录中实现转换动作：`mask.go`、`truncate.go`、`extract.go`、`rename.go`、`remove.go`、`set.go`
- [x] 4.4 在 `actions/` 目录中实现元数据动作：`mark.go`
- [x] 4.5 添加 `Action` 接口和注册机制

## 5. 实现规则引擎

- [x] 5.1 在 `engine.go` 中实现 `RuleEngine` 结构体
- [x] 5.2 实现规则加载和排序逻辑
- [x] 5.3 实现单条规则评估逻辑
- [x] 5.4 实现规则链评估逻辑（按顺序执行、终止性动作处理）
- [x] 5.5 实现 best-effort 错误处理策略
- [x] 5.6 实现审计/监控记录（匹配记录、错误记录）

## 6. 实现存储适配器

- [x] 6.1 在 `storage/` 目录中实现 `memory.go`（内存存储，用于测试）
- [x] 6.2 在 `storage/` 目录中实现 `etcd.go`（ETCD 存储、支持 Watch）
- [x] 6.3 实现热加载机制（Watch + 定时刷新兜底）
- [x] 6.4 实现 ETCD 命名空间支持（`/rules/clients/`、`/rules/defaults/`）

## 7. 集成到 Log SDK

- [x] 7.1 移除或重构 `log-sdk/pkg/strategy/` 包
- [x] 7.2 更新 `log-sdk/pkg/logger/` 以集成新的规则引擎
- [x] 7.3 添加配置选项（Service Name、Environment、ETCD 端点）
- [x] 7.4 在发送日志前评估规则

## 8. 集成到 Log Processor

- [x] 8.1 移除或重构 `log-processor/pkg/filter/` 包
- [x] 8.2 更新 `log-processor/pkg/config/` 以集成新的规则引擎
- [x] 8.3 更新主处理流程以使用规则引擎
- [x] 8.4 添加配置选项（Service Name、Environment、ETCD 端点）

## 9. 集成到 Log Analyzer

- [x] 9.1 更新 `log-analyzer/internal/models/` 以使用新的统一规则模型
- [x] 9.2 更新 `log-analyzer/internal/handlers/rules.go` API 处理器
- [x] 9.3 实现规则到 ETCD 的同步逻辑
- [ ] 9.4 更新数据库迁移脚本（可选，现有迁移已兼容）
- [x] 9.5 更新前端组件（如需要）

## 10. 测试和文档

- [x] 10.1 编写单元测试（Condition 匹配、Action 执行、规则引擎）
- [x] 10.2 编写集成测试（ETCD 存储、热加载）
- [x] 10.3 编写端到端测试（跨组件）
- [x] 10.4 添加性能基准测试
- [x] 10.5 更新 README 和文档
- [x] 10.6 创建迁移指南
