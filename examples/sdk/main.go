// SDK 使用示例，演示日志 SDK 的两种 API 风格
package main

import (
	"time"

	"github.com/log-system/log-sdk/pkg/logger"
)

func main() {
	// 初始化 Logger
	log := logger.New(logger.Config{
		ServiceName:       "sdk-example",
		Environment:       "dev",
		Cluster:           "local",
		Pod:               "sdk-example-pod-1",
		KafkaBrokers:      []string{"localhost:9092"},
		KafkaTopic:        "logs",
		EtcdEndpoints:     []string{"http://localhost:2379"},
		BatchSize:         100,
		BatchTimeout:      100,
		FallbackToConsole: true,
		MaxBufferSize:     10000,
	})
	defer log.Close()

	// ========== 风格 1: 传统打印方式 (类似标准 log 包) ==========
	demoTraditionalStyle(log)

	// ========== 风格 2: 强类型链式打印方式 (类似 Zap/ZeroLog) ==========
	demoChainStyle(log)

	// ========== Hook 过滤示例 ==========
	demoHookFiltering(log)

	// ========== With 字段继承示例 ==========
	demoWithFields(log)
}

// demoTraditionalStyle 演示传统打印风格
func demoTraditionalStyle(log logger.Logger) {
	// 类似于标准 log 包的用法
	log.Printf("User %s logged in at %s", "john.doe", time.Now().Format("15:04:05"))
	log.Println("Application started successfully")
	log.Print("Debug mode enabled")

	// 带字段的结构化日志（传统风格）
	log.Info("Order created",
		logger.F("order_id", "ORD-12345"),
		logger.F("user_id", "user-789"),
		logger.F("amount", 99.99),
	)

	log.Error("Database connection failed",
		logger.F("error", "connection timeout"),
		logger.F("retry_count", 3),
	)
}

// demoChainStyle 演示强类型链式风格
func demoChainStyle(log logger.Logger) {
	// Info 级别日志
	log.Info("User logged in").
		Str("username", "john.doe").
		Str("ip", "192.168.1.1").
		Int("login_count", 42).
		Send()

	// Error 级别日志，带更多字段
	log.Error("Payment failed").
		Str("order_id", "ORD-67890").
		Str("user_id", "user-123").
		Float64("amount", 199.99).
		Str("currency", "USD").
		Str("error_code", "PAYMENT_DECLINED").
		Send()

	// Debug 级别日志，带布尔值
	log.Debug("Feature flag check").
		Str("feature", "new_checkout").
		Bool("enabled", true).
		Int64("rollout_percentage", 50).
		Send()

	// Warn 级别日志
	log.Warn("High latency detected").
		Str("endpoint", "/api/v1/orders").
		Float64("latency_ms", 1500.5).
		Int("threshold_ms", 1000).
		Send()
}

// demoHookFiltering 演示 Hook 过滤功能
func demoHookFiltering(log logger.Logger) {
	// 添加 LevelHook - 只记录 WARN 及以上级别
	logWithLevelFilter := log.AddHook(logger.LevelHook(logger.LevelWarn))

	// 这条 DEBUG 日志会被过滤掉（不会输出）
	logWithLevelFilter.Debug("This debug will be filtered").
		Str("reason", "level too low").
		Send()

	// 这条 WARN 日志会通过
	logWithLevelFilter.Warn("This warning will be logged").
		Str("reason", "level sufficient").
		Send()

	// 添加 LineHook - 只记录特定行号范围的日志
	logWithLineFilter := log.AddHook(logger.LineHook(50, 100))

	// 这条日志行号不在范围内，会被过滤
	logWithLineFilter.Info("This may be filtered based on line number").
		Send()
}

// demoWithFields 演示 With 字段继承
func demoWithFields(log logger.Logger) {
	// 创建一个带默认字段的 Logger
	requestLog := log.With(
		logger.F("request_id", "req-abc-123"),
		logger.F("trace_id", "trace-xyz-789"),
		logger.F("service", "payment-service"),
	)

	// 所有后续日志都会自动包含上面的字段
	requestLog.Info("Processing payment").
		Str("user_id", "user-456").
		Send()

	requestLog.Info("Payment authorized").
		Str("auth_code", "AUTH-999").
		Int("amount", 5000).
		Send()

	requestLog.Error("Payment failed").
		Str("error", "insufficient funds").
		Send()

	// 传统风格也同样继承字段
	requestLog.Printf("Request completed in %dms", 150)
}
