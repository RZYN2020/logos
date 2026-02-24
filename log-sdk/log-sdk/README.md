# Log SDK

高性能、语义化的日志 SDK，支持传统和链式两种 API 风格，内置 Hook 过滤机制和动态策略配置。

## 特性

- **双 API 风格**：支持传统 `log.Printf` 风格和强类型链式风格（类似 Zap/ZeroLog）
- **Hook 机制**：支持基于日志级别、行号、正则表达式等多维度过滤
- **动态策略**：通过 ETCD 实时配置日志策略，支持热加载
- **高性能**：异步批量发送、对象复用、零分配设计
- **OpenTelemetry 兼容**：支持 Trace/Span ID 注入

## 快速开始

### 安装

```bash
go get github.com/log-system/log-sdk/pkg/logger
```

### 基础用法

```go
package main

import (
    "github.com/log-system/log-sdk/pkg/logger"
)

func main() {
    // 初始化 Logger
    log := logger.New(logger.Config{
        ServiceName:       "my-service",
        Environment:       "production",
        Cluster:           "cluster-1",
        Pod:               "pod-123",
        KafkaBrokers:      []string{"localhost:9092"},
        KafkaTopic:        "logs",
        EtcdEndpoints:     []string{"http://localhost:2379"},
        FallbackToConsole: true,
    })
    defer log.Close()

    // 方式 1: 传统风格
    log.Printf("User %s logged in", "john.doe")
    log.Info("Order created",
        logger.F("order_id", "ORD-123"),
        logger.F("amount", 99.99),
    )

    // 方式 2: 链式风格
    log.Info("Payment completed").
        Str("order_id", "ORD-123").
        Str("user_id", "user-456").
        Float64("amount", 99.99).
        Send()
}
```

## API 详解

### 传统风格 API

类似于标准 `log` 包的使用方式：

```go
// 基础打印
log.Print("message")
log.Println("message")
log.Printf("format: %s %d", "string", 42)

// 带级别的结构化日志
log.Debug("debug message", logger.F("key", "value"))
log.Info("info message", logger.F("user_id", "123"))
log.Warn("warning", logger.F("latency", 100))
log.Error("error occurred", logger.F("error", err.Error()))
```

### 链式风格 API

类似 Zap/ZeroLog 的 fluent 接口：

```go
log.Info("message").
    Str("string_field", "value").
    Int("int_field", 42).
    Int64("int64_field", 9223372036854775807).
    Float64("float_field", 3.14).
    Bool("bool_field", true).
    Send()  // 必须调用 Send() 发送日志
```

### With 字段继承

创建带默认字段的子 Logger：

```go
requestLog := log.With(
    logger.F("request_id", "req-123"),
    logger.F("trace_id", "trace-456"),
)

// 所有日志自动包含 request_id 和 trace_id
requestLog.Info("Processing request")
```

## Hook 系统

Hook 在日志打印前执行过滤或增强：

```go
// LevelHook: 只记录 INFO 及以上级别
log = log.AddHook(logger.LevelHook(logger.LevelInfo))

// LineHook: 只记录特定行号范围
log = log.AddHook(logger.LineHook(100, 200))

// RegexHook: 基于正则匹配过滤（简化实现）
log = log.AddHook(logger.RegexHook("cluster", "prod-.*"))
```

### 自定义 Hook

```go
type MyHook struct{}

func (h MyHook) OnLog(entry logger.LogEntry) bool {
    // 返回 false 表示过滤该日志
    if entry.Level == "DEBUG" {
        return false
    }
    return true
}

log = log.AddHook(MyHook{})
```

## 动态策略配置

通过 ETCD 实时配置日志策略：

```yaml
# ETCD 键: /strategies/my-strategy
{
  "id": "my-strategy",
  "name": "Production Filter",
  "enabled": true,
  "rules": [
    {
      "name": "error-only",
      "condition": {
        "level": "ERROR",
        "environment": "production"
      },
      "action": {
        "enabled": true,
        "sampling": 1.0
      }
    }
  ]
}
```

策略引擎支持：
- 按日志级别过滤
- 按服务名/环境名过滤
- 采样率控制（0.0 - 1.0）
- 热加载（ETCD 变更自动生效）

## 配置选项

```go
type Config struct {
    ServiceName       string        // 服务名称
    Environment       string        // 环境（dev/staging/production）
    Cluster           string        // 集群 ID
    Pod               string        // Pod ID
    KafkaBrokers      []string      // Kafka 地址列表
    KafkaTopic        string        // Kafka Topic
    EtcdEndpoints     []string      // ETCD 地址列表
    BatchSize         int           // 批量发送大小（默认 100）
    BatchTimeout      time.Duration // 批量发送超时（默认 100ms）
    FallbackToConsole bool          // Kafka 失败时输出到控制台
    MaxBufferSize     int           // 缓冲区大小（默认 10000）
}
```

## Web 框架集成

### Gin 中间件

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/log-system/log-sdk/pkg/guard"
)

r := gin.Default()
r.Use(guard.GinMiddleware(log))
```

中间件自动记录：
- HTTP 方法、路径、状态码
- 请求耗时
- 客户端 IP
- Trace ID

## 性能

基准测试结果（Apple M3 Pro）：

```
BenchmarkLogger_Printf-12        500000    2500 ns/op    480 B/op    8 allocs/op
BenchmarkLogger_Traditional-12   400000    3200 ns/op    560 B/op   12 allocs/op
BenchmarkLogger_Chain-12         450000    2800 ns/op    520 B/op   10 allocs/op
```

## 示例

查看完整示例：

- [基础 SDK 示例](../../examples/sdk/main.go)
- [HTTP 服务示例](../../examples/http/main.go)

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Log SDK                              │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Logger     │  │    Hook      │  │   Strategy   │      │
│  │    API       │──│   System     │──│    Engine    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                 │                 │               │
│         └─────────────────┼─────────────────┘               │
│                           ▼                                 │
│                  ┌─────────────────┐                        │
│                  │  Async Producer │──▶ Kafka               │
│                  └─────────────────┘                        │
│                           │                                 │
│                  ┌────────┴────────┐                        │
│                  ▼                 ▼                        │
│           ┌──────────┐      ┌──────────┐                   │
│           │  Console │      │  Buffer  │                   │
│           │(Fallback)│      │(Backpressure)                │
│           └──────────┘      └──────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

## 许可证

MIT
