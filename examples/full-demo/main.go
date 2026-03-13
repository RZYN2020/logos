// 完整的日志全链路演示
// 1. Log SDK 生成日志 → Kafka
// 2. 消费 Kafka 消息并展示
// 3. 写入 Elasticsearch
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	fmt.Println("=== Logos 日志系统 - 全链路演示 ===")

	// 配置
	kafkaBrokers := []string{"localhost:9092"}
	kafkaTopic := "logs"
	esURL := "http://localhost:9200"

	// 步骤 1: 初始化 Logger
	fmt.Println("[1/5] 初始化 Log SDK...")
	logSDK := logger.New(logger.Config{
		ServiceName:       "full-demo-service",
		Environment:       "demo",
		Cluster:           "local",
		Pod:               "demo-pod-1",
		KafkaBrokers:      kafkaBrokers,
		KafkaTopic:        kafkaTopic,
		EtcdEndpoints:     []string{"http://localhost:2379"},
		BatchSize:         10,
		BatchTimeout:      500,
		FallbackToConsole: true,
		MaxBufferSize:     1000,
	})
	defer logSDK.Close()
	fmt.Println("  ✓ Log SDK 初始化完成")

	// 步骤 2: 启动 Kafka 消费者（模拟 Log Processor）
	fmt.Println("[2/5] 启动 Kafka 消费者（模拟 Log Processor）...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  kafkaBrokers,
		Topic:    kafkaTopic,
		GroupID:  "demo-consumer-group",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	// 用于接收消费到的消息
	msgChan := make(chan kafka.Message, 100)

	go func() {
		fmt.Println("  ✓ 消费者已启动，等待消息...")
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() == nil {
						log.Printf("读取消息错误: %v", err)
					}
					return
				}
				msgChan <- msg
			}
		}
	}()

	// 步骤 3: 生成测试日志
	fmt.Println("[3/5] 生成测试日志...")
	go generateTestLogs(logSDK)

	// 步骤 4: 展示消费到的日志
	fmt.Println("[4/5] 展示从 Kafka 消费到的日志:")
	fmt.Println("  " + strings.Repeat("─", 80))

	receivedCount := 0
	timeout := time.After(15 * time.Second)
	done := false

	for !done {
		select {
		case msg := <-msgChan:
			receivedCount++
			displayLogMessage(msg, receivedCount)

		case <-timeout:
			done = true
			fmt.Println("  " + strings.Repeat("─", 80))
			fmt.Printf("\n  超时，共收到 %d 条消息\n", receivedCount)
		}
	}

	// 步骤 5: 检查 Elasticsearch
	fmt.Println("\n[5/5] 检查 Elasticsearch...")
	checkElasticsearch(esURL)

	// 等待用户退出
	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("\n服务访问地址:")
	fmt.Println("  - Etcd:         http://localhost:2379")
	fmt.Println("  - Kafka:        localhost:9092")
	fmt.Println("  - Elasticsearch: http://localhost:9200")
	fmt.Println("  - Kibana:       http://localhost:5601")
	fmt.Println("  - Grafana:      http://localhost:3000 (admin/admin)")
	fmt.Println("  - Prometheus:   http://localhost:9090")

	fmt.Println("\n按 Ctrl+C 退出...")

	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n再见！")
}

func generateTestLogs(log logger.Logger) {
	// 等待一小会儿让消费者启动
	time.Sleep(1 * time.Second)

	fmt.Println("  ✓ 开始发送日志到 Kafka...")

	// 生成各种类型的日志
	log.Info("应用启动成功").
		Str("version", "1.0.0").
		Str("commit", "abc123").
		Send()

	time.Sleep(100 * time.Millisecond)

	// 模拟用户登录
	log.Info("用户登录").
		Str("user_id", "user-001").
		Str("username", "张三").
		Str("ip", "192.168.1.100").
		Str("user_agent", "Chrome/120.0").
		Send()

	time.Sleep(100 * time.Millisecond)

	// 模拟订单创建
	log.Info("订单创建").
		Str("order_id", "ORD-2026-0228-0001").
		Str("user_id", "user-001").
		Float64("amount", 299.99).
		Str("currency", "CNY").
		Str("product_id", "PROD-001").
		Send()

	time.Sleep(100 * time.Millisecond)

	// 模拟警告
	log.Warn("响应时间过长").
		Str("endpoint", "/api/v1/orders").
		Str("method", "POST").
		Int("status_code", 201).
		Float64("duration_ms", 1250.5).
		Int("threshold_ms", 1000).
		Send()

	time.Sleep(100 * time.Millisecond)

	// 模拟错误
	log.Error("支付失败").
		Str("order_id", "ORD-2026-0228-0002").
		Str("user_id", "user-002").
		Float64("amount", 500.00).
		Str("error_code", "INSUFFICIENT_FUNDS").
		Str("error_message", "账户余额不足").
		Send()

	time.Sleep(100 * time.Millisecond)

	// 批量生成更多日志
	for i := 1; i <= 5; i++ {
		log.Info("批量测试日志").
			Int("batch_id", i).
			Str("type", "test").
			Str("timestamp", time.Now().Format(time.RFC3339)).
			Send()
		time.Sleep(50 * time.Millisecond)
	}

	log.Info("测试日志发送完成").
		Int("total_sent", 10).
		Send()
}

func displayLogMessage(msg kafka.Message, count int) {
	var logEntry map[string]any
	if err := json.Unmarshal(msg.Value, &logEntry); err == nil {
		level, _ := logEntry["level"].(string)
		message, _ := logEntry["message"].(string)
		service, _ := logEntry["service"].(string)

		levelColor := getLevelColor(level)
		resetColor := "\033[0m"

		fmt.Printf("  [%d] %s%-5s%s | %-20s | %s\n",
			count, levelColor, level, resetColor, service, message)
	} else {
		fmt.Printf("  [%d] Raw: %s\n", count, string(msg.Value))
	}
}

func getLevelColor(level string) string {
	switch level {
	case "DEBUG":
		return "\033[36m" // cyan
	case "INFO":
		return "\033[32m" // green
	case "WARN":
		return "\033[33m" // yellow
	case "ERROR":
		return "\033[31m" // red
	case "FATAL":
		return "\033[35m" // magenta
	default:
		return "\033[37m" // white
	}
}

func checkElasticsearch(url string) {
	// 检查 ES 是否健康
	resp, err := fetchURL(url + "/_cluster/health")
	if err != nil {
		fmt.Printf("  ✗ Elasticsearch 连接失败: %v\n", err)
		return
	}

	var health map[string]any
	if err := json.Unmarshal([]byte(resp), &health); err == nil {
		status, _ := health["status"].(string)
		fmt.Printf("  ✓ Elasticsearch 集群状态: %s\n", status)
	}

	// 检查索引
	resp, err = fetchURL(url + "/_cat/indices?v")
	if err == nil {
		fmt.Println("  索引列表:")
		fmt.Println("    " + strings.ReplaceAll(resp, "\n", "\n    "))
	}
}

func fetchURL(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// 辅助函数使用标准库 strings 包
