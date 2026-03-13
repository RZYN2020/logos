// 简单的日志全链路演示
// 1. Log SDK 生成日志 → Kafka
// 2. 从 Kafka 消费并展示
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/log-system/log-sdk/pkg/logger"
	"github.com/segmentio/kafka-go"
)

func main() {
	fmt.Println("=== Logos 日志系统 - 全链路演示 ===\n")

	// 配置
	kafkaBrokers := []string{"localhost:9092"}
	kafkaTopic := "logs"

	// 步骤 1: 初始化 Logger
	fmt.Println("[1/4] 初始化 Log SDK...")
	logSDK := logger.New(logger.Config{
		ServiceName:       "demo-service",
		Environment:       "demo",
		Cluster:           "local",
		Pod:               "demo-pod-1",
		KafkaBrokers:      kafkaBrokers,
		KafkaTopic:        kafkaTopic,
		EtcdEndpoints:     []string{"http://localhost:2379"},
		BatchSize:         5,
		BatchTimeout:      200,
		FallbackToConsole: true,
		MaxBufferSize:     1000,
	})
	defer logSDK.Close()
	fmt.Println("  ✓ Log SDK 初始化完成\n")

	// 步骤 2: 启动 Kafka 消费者
	fmt.Println("[2/4] 启动 Kafka 消费者...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  kafkaBrokers,
		Topic:    kafkaTopic,
		GroupID:  "demo-group",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	msgChan := make(chan kafka.Message, 100)

	go func() {
		fmt.Println("  ✓ 消费者已启动\n")
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() == nil {
						fmt.Printf("  读取错误: %v\n", err)
					}
					return
				}
				msgChan <- msg
			}
		}
	}()

	// 步骤 3: 生成测试日志
	fmt.Println("[3/4] 生成测试日志...")
	go generateLogs(logSDK)

	// 步骤 4: 展示消费到的日志
	fmt.Println("[4/4] 消费并展示日志:\n")
	fmt.Println("  " + strings.Repeat("═", 80))
	fmt.Printf("  %-3s | %-5s | %-20s | %s\n", "#", "LEVEL", "SERVICE", "MESSAGE")
	fmt.Println("  " + strings.Repeat("─", 80))

	receivedCount := 0
	timeout := time.After(10 * time.Second)
	done := false

	for !done {
		select {
		case msg := <-msgChan:
			receivedCount++
			displayMsg(msg, receivedCount)

		case <-timeout:
			done = true
			fmt.Println("  " + strings.Repeat("─", 80))
			fmt.Printf("  共收到 %d 条日志\n", receivedCount)
		}
	}

	// 显示服务信息
	fmt.Println("\n=== 服务访问地址 ===")
	fmt.Println("  ✓ Etcd:         http://localhost:2379")
	fmt.Println("  ✓ Kafka:        localhost:9092")
	fmt.Println("  ✓ Elasticsearch: http://localhost:9200")
	checkES("http://localhost:9200")
	fmt.Println("  ✓ Kibana:       http://localhost:5601")
	fmt.Println("  ✓ Grafana:      http://localhost:3000 (admin/admin)")
	fmt.Println("  ✓ Prometheus:   http://localhost:9090")

	fmt.Println("\n=== 手工测试命令 ===")
	fmt.Println("  1. 查看 Kafka Topic:")
	fmt.Println("     docker exec kafka-0 kafka-topics --list --bootstrap-server localhost:9092")
	fmt.Println("\n  2. 查看 Kafka 消息:")
	fmt.Println("     docker exec kafka-0 kafka-console-consumer --bootstrap-server localhost:9092 --topic logs --from-beginning")
	fmt.Println("\n  3. 查看 Elasticsearch 索引:")
	fmt.Println("     curl 'http://localhost:9200/_cat/indices?v'")
	fmt.Println("\n  4. 查看 Etcd 状态:")
	fmt.Println("     curl http://localhost:2379/health")

	fmt.Println("\n按 Ctrl+C 退出...")

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n再见！")
}

func generateLogs(log logger.Logger) {
	time.Sleep(800 * time.Millisecond)
	fmt.Println("  ✓ 开始发送日志...\n")

	// 示例 1: 传统 API
	log.Printf("系统启动, 时间: %s", time.Now().Format(time.RFC3339))
	log.Println("服务初始化完成")

	time.Sleep(200 * time.Millisecond)

	// 示例 2: 结构化日志（传统风格）
	log.Info("用户登录",
		logger.F("user_id", "u12345"),
		logger.F("username", "张三"),
		logger.F("ip", "192.168.1.100"),
	)

	time.Sleep(200 * time.Millisecond)

	// 示例 3: 链式 API
	log.Warn("高延迟警告").
		Str("endpoint", "/api/orders").
		Str("method", "POST").
		Float64("latency_ms", 1250.5).
		Int("threshold", 1000).
		Send()

	time.Sleep(200 * time.Millisecond)

	// 示例 4: 错误日志
	log.Error("数据库连接失败").
		Str("db_host", "db01.example.com").
		Int("db_port", 5432).
		Str("error", "connection refused").
		Int("retry_count", 3).
		Send()

	time.Sleep(200 * time.Millisecond)

	// 示例 5: With 字段继承
	reqLog := log.With(
		logger.F("request_id", "req-abc-123"),
		logger.F("trace_id", "trace-xyz-789"),
	)

	reqLog.Info("处理请求").
		Str("path", "/api/payment").
		Send()

	reqLog.Info("支付成功").
		Str("order_id", "ORD-001").
		Float64("amount", 99.99).
		Send()

	time.Sleep(300 * time.Millisecond)
	log.Info("测试日志发送完成")
}

func displayMsg(msg kafka.Message, count int) {
	var logEntry map[string]interface{}
	if err := json.Unmarshal(msg.Value, &logEntry); err == nil {
		level, _ := logEntry["level"].(string)
		service, _ := logEntry["service"].(string)
		message, _ := logEntry["message"].(string)

		levelColor := getLevelColor(level)
		reset := "\033[0m"

		fmt.Printf("  %-3d | %s%-5s%s | %-20s | %s\n",
			count, levelColor, level, reset, service, message)
	} else {
		fmt.Printf("  %-3d | %s\n", count, string(msg.Value))
	}
}

func getLevelColor(level string) string {
	switch level {
	case "DEBUG":
		return "\033[36m"
	case "INFO":
		return "\033[32m"
	case "WARN":
		return "\033[33m"
	case "ERROR":
		return "\033[31m"
	case "FATAL":
		return "\033[35m"
	default:
		return "\033[37m"
	}
}

func checkES(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url+"/_cluster/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

}
