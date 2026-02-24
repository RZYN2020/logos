# Log SDK 实现方案

## 概述

实现一个高性能、语义化的日志 SDK，API 设计类似于 Zap 或 ZeroLog，支持动态策略配置和可扩展的 Hook 机制。

## 核心特性

### 1. API 设计
- **传统打印方式**：类似 log.Printf，直接打印字符串
  ```go
  log.Printf("Hello %s, you have %d new messages", name, count)
  ```

- **强类型链式打印方式**：类似 Zap/ZeroLog，支持链式调用和结构化字段
  ```go
  log.Info("User logged in").
      Str("username", "john.doe").
      Int("login_count", 42).
      Float64("balance", 123.45).
      Send()
  ```

- 支持结构化日志字段（key-value 对）
- 支持日志级别（Debug/Info/Warn/Error/Fatal/Panic）
- 支持上下文传播（通过 context.Context）

### 2. Hook 机制
在实际打印日志之前，允许用户自定义 Hook 进行过滤或增强：
- **过滤维度**：
  - 集群 ID（Cluster ID）
  - Pod ID
  - 日志等级（Level）
  - 行号（Line Number）
  - 文件路径（File Path）
  - 函数名（Function Name）

- **Hook 接口**：
```go
type Hook interface {
  OnLog(entry LogEntry) bool  // 返回 false 表示过滤该日志
}
```

### 3. 动态策略配置
- 使用 Etcd 作为配置中心
- 定时监控 Etcd 配置变更，支持热加载
- 策略规则格式：
```yaml
rules:
  - name: "production-error-filter"
    condition:
      level: "ERROR"
      environment: "production"
      cluster: "cluster-1"
      pod: "api-*"
    action:
      enabled: true
      sampling: 1.0
```

### 4. 高性能特性
- 对象池（Object Pool）实现零分配
- 使用接口（Interface）实现高性能
- 异步批量发送到 Kafka
- 环形缓冲区防止阻塞

## 架构设计

```
┌───────────────────────────────────────────────────────────────────┐
│                         Log SDK Architecture                     │
├───────────────────────────────────────────────────────────────────┤
│ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │
│ │  Logger  │ │  Hook    │ │ Strategy │ │ Async    │ │ Encoder  │ │
│ │  API     │ │  System  │ │  Engine  │ │ Producer │ │  (JSON)  │ │
│ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ │
│          │          │          │          │          │           │
│          └──────────┼──────────┼──────────┼──────────┘           │
│                     ▼          ▼          ▼                        │
│             ┌──────────────────────────────────┐                  │
│             │       Log Entry Builder          │                  │
│             └──────────────────────────────────┘                  │
│                        │                                          │
│                        ▼                                          │
│             ┌──────────────────────────────────┐                  │
│             │      Etcd Config Watcher         │                  │
│             └──────────────────────────────────┘                  │
└───────────────────────────────────────────────────────────────────┘
```

## 目录结构

```
log-sdk/
└── log-sdk/
    ├── pkg/
    │   ├── logger/          # 核心 Logger API
    │   │   ├── logger.go    # 主接口
    │   │   └── options.go   # 配置选项
    │   ├── hook/            # Hook 系统
    │   │   ├── hook.go      # 接口定义
    │   │   └── builtin.go   # 内置 Hook
    │   ├── strategy/        # 策略引擎
    │   │   ├── engine.go    # 策略评估
    │   │   └── etcd.go      # Etcd 配置管理
    │   ├── async/           # 异步 I/O
    │   │   ├── producer.go  # Kafka 生产者
    │   │   └── buffer.go    # 环形缓冲区
    │   └── encoder/         # 编码
    │       └── json.go      # JSON 编码器
    ├── cmd/                 # 命令行工具
    └── go.mod
```

## 实现计划

### Phase 1: 基础 API (2 days)
1. 实现 Logger 接口和基础结构
2. 实现 Hook 系统
3. 实现简单的编码和打印功能

### Phase 2: 高性能优化 (3 days)
1. 实现对象池
2. 实现异步发送和环形缓冲区
3. 性能基准测试

### Phase 3: 策略引擎 (3 days)
1. 实现 Etcd 配置监控
2. 实现策略评估引擎
3. 实现热加载机制

### Phase 4: 集成测试 (2 days)
1. 编写单元测试
2. 编写集成测试
3. 测试 Etcd 配置变更

### Phase 5: 示例 (1 day)
1. 完善 examples/http/main.go
2. 创建 examples/sdk/main.go
3. 编写 SDK 使用文档

## 依赖

- Go 1.25+
- github.com/segmentio/kafka-go
- go.etcd.io/etcd/client/v3
- github.com/gin-gonic/gin (for examples)

## 风险评估

- **Etcd 不可用**：降级到默认策略
- **Kafka 不可用**：缓冲区满时丢弃日志
- **性能下降**：监控系统资源使用，自动调整策略