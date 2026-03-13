// 快速测试 - 不依赖 Kafka/Etcd，只使用控制台输出
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/log-system/log-sdk/pkg/logger"
)

func main() {
	fmt.Println("=== Logos 日志系统快速测试 ===\n")

	// 创建仅使用控制台输出的 Logger（不连接 Kafka/Etcd）
	log := logger.New(logger.Config{
		ServiceName:       "quick-test",
		Environment:       "dev",
		Cluster:           "local",
		Pod:               "test-pod-1",
		FallbackToConsole: true, // 强制使用控制台输出
		MaxBufferSize:     100,
	})
	defer log.Close()

	fmt.Println("✓ Logger 初始化成功")
	fmt.Println("  Service: quick-test")
	fmt.Println("  Output: Console (fallback mode)\n")

	// 等待用户输入
	fmt.Println("按 Ctrl+C 退出，或输入命令进行测试：")
	fmt.Println("  1 - 测试传统 API")
	fmt.Println("  2 - 测试链式 API")
	fmt.Println("  3 - 测试字段继承")
	fmt.Println("  4 - 测试 Hook 过滤")
	fmt.Println("  5 - 运行完整演示\n")

	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 简单的交互式测试
	go func() {
		for {
			var input string
			fmt.Print("> ")
			fmt.Scanln(&input)

			switch input {
			case "1":
				testTraditionalAPI(log)
			case "2":
				testChainAPI(log)
			case "3":
				testWithFields(log)
			case "4":
				testHooks(log)
			case "5":
				runFullDemo(log)
			default:
				fmt.Println("未知命令，请输入 1-5")
			}
		}
	}()

	<-sigChan
	fmt.Println("\n测试结束")
}

func testTraditionalAPI(log logger.Logger) {
	fmt.Println("\n--- 传统 API 测试 ---")
	log.Printf("这是一条 Printf 日志: %s", time.Now().Format(time.RFC3339))
	log.Println("这是一条 Println 日志")
	log.Info("订单创建成功",
		logger.F("order_id", "ORD-"+time.Now().Format("20060102150405")),
		logger.F("user_id", "user-123"),
		logger.F("amount", 99.99),
	)
	log.Error("数据库连接失败",
		logger.F("error", "connection timeout"),
		logger.F("retry", 3),
	)
	fmt.Println("✓ 传统 API 测试完成")
}

func testChainAPI(log logger.Logger) {
	fmt.Println("\n--- 链式 API 测试 ---")
	log.Info("用户登录成功").
		Str("username", "john.doe").
		Str("ip", "192.168.1.100").
		Int("login_count", 15).
		Send()

	log.Warn("高延迟警告").
		Str("endpoint", "/api/v1/orders").
		Float64("latency_ms", 1250.5).
		Int("threshold_ms", 1000).
		Send()

	log.Error("支付失败").
		Str("order_id", "ORD-98765").
		Str("error_code", "INSUFFICIENT_FUNDS").
		Float64("amount", 500.00).
		Send()
	fmt.Println("✓ 链式 API 测试完成")
}

func testWithFields(log logger.Logger) {
	fmt.Println("\n--- 字段继承测试 ---")
	requestLog := log.With(
		logger.F("request_id", "req-"+time.Now().Format("20060102150405")),
		logger.F("trace_id", "trace-abc-123-xyz"),
		logger.F("service", "payment-service"),
	)

	requestLog.Info("开始处理请求").
		Str("user_id", "user-456").
		Send()

	requestLog.Info("支付授权成功").
		Str("auth_code", "AUTH-789").
		Int("amount", 1000).
		Send()
	fmt.Println("✓ 字段继承测试完成 - 所有日志都包含 request_id/trace_id/service")
}

func testHooks(log logger.Logger) {
	fmt.Println("\n--- Hook 过滤测试 ---")
	// 只记录 WARN 及以上级别
	logWithFilter := log.AddHook(logger.LevelHook(logger.LevelWarn))

	fmt.Println("  测试 LevelHook (只记录 WARN+):")
	fmt.Println("  - DEBUG 日志会被过滤...")
	logWithFilter.Debug("这条 DEBUG 日志会被过滤").Send()
	fmt.Println("  - WARN 日志会通过...")
	logWithFilter.Warn("这条 WARN 日志会被记录").
		Str("reason", "level_filter_test").
		Send()
	fmt.Println("✓ Hook 过滤测试完成")
}

func runFullDemo(log logger.Logger) {
	fmt.Println("\n=== 完整演示 ===")

	// 模拟一个 HTTP 请求处理流程
	fmt.Println("\n1. 模拟 HTTP 请求处理...")
	reqID := "req-" + time.Now().Format("20060102150405")
	traceID := "trace-" + fmt.Sprintf("%x", time.Now().UnixNano())

	reqLog := log.With(
		logger.F("request_id", reqID),
		logger.F("trace_id", traceID),
		logger.F("endpoint", "/api/v1/order"),
		logger.F("method", "POST"),
	)

	reqLog.Info("收到请求").
		Str("client_ip", "10.0.1.50").
		Str("user_agent", "Mozilla/5.0").
		Send()

	// 模拟业务逻辑
	time.Sleep(50 * time.Millisecond)
	reqLog.Info("验证用户身份").
		Str("user_id", "user-789").
		Bool("authenticated", true).
		Send()

	// 模拟数据库操作
	time.Sleep(100 * time.Millisecond)
	reqLog.Info("创建订单").
		Str("order_id", "ORD-"+time.Now().Format("20060102150405")).
		Float64("amount", 299.99).
		Str("currency", "CNY").
		Send()

	// 模拟成功响应
	time.Sleep(30 * time.Millisecond)
	reqLog.Info("请求完成").
		Int("status_code", 201).
		Int("duration_ms", 185).
		Send()

	fmt.Println("\n✓ 完整演示完成！")
	fmt.Println("  查看上面的日志输出，观察：")
	fmt.Println("  - 结构化字段")
	fmt.Println("  - 字段继承 (request_id/trace_id)")
	fmt.Println("  - 时间戳")
	fmt.Println("  - 日志级别")
}
