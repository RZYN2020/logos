// Package main 日志处理器入口
// 从Kafka消费日志，经过解析、语义增强后写入Elasticsearch
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/log-system/log-processor/pkg/parser"
	"github.com/log-system/log-processor/pkg/semantic"
	"github.com/log-system/log-processor/pkg/sink"
	"github.com/segmentio/kafka-go"
)

// Config 处理器配置
type Config struct {
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroup   string
	ESAddresses  []string
	ESIndex      string
	BatchSize    int
	BatchTimeout time.Duration
}

// Processor 日志处理器
type Processor struct {
	config    Config
	reader    *kafka.Reader
	parser    parser.Parser
	builder   *semantic.Builder
	sink      sink.Sink
	batch     []sink.LogEntry
	lastFlush time.Time
}

// NewProcessor 创建处理器
func NewProcessor(cfg Config) (*Processor, error) {
	// 创建Kafka Reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.KafkaBrokers,
		Topic:    cfg.KafkaTopic,
		GroupID:  cfg.KafkaGroup,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	// 创建解析器
	p := parser.NewMultiParser()

	// 创建语义构建器
	builder := semantic.NewBuilder()

	// 创建Sink
	s, err := sink.NewElasticsearchSink(cfg.ESAddresses, cfg.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create sink: %w", err)
	}

	return &Processor{
		config:    cfg,
		reader:    reader,
		parser:    p,
		builder:   builder,
		sink:      s,
		batch:     make([]sink.LogEntry, 0, cfg.BatchSize),
		lastFlush: time.Now(),
	}, nil
}

// Run 启动处理循环
func (p *Processor) Run(ctx context.Context) error {
	log.Println("Starting log processor...")

	for {
		select {
		case <-ctx.Done():
			return p.flush()

		default:
			// 读取消息
			msg, err := p.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return p.flush()
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			// 处理消息
			if err := p.processMessage(msg); err != nil {
				log.Printf("Error processing message: %v", err)
				continue
			}

			// 检查是否需要刷新
			if p.shouldFlush() {
				if err := p.flush(); err != nil {
					log.Printf("Error flushing batch: %v", err)
				}
			}
		}
	}
}

// processMessage 处理单条消息
func (p *Processor) processMessage(msg kafka.Message) error {
	// 解析日志
	parsed, err := p.parser.Parse(msg.Value)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// 转换为LogEntry
	entry := &semantic.LogEntry{
		Timestamp: parsed.Timestamp,
		Level:     parsed.Level,
		Message:   parsed.Message,
		Service:   parsed.Service,
		TraceID:   parsed.TraceID,
		SpanID:    parsed.SpanID,
		Fields:    parsed.Fields,
		Raw:       parsed.Raw,
	}

	// 语义增强
	ctx := context.Background()
	enriched := p.builder.Build(ctx, entry)

	// 转换为sink格式
	doc := map[string]interface{}{
		"timestamp":       enriched.Timestamp,
		"level":           enriched.Level,
		"message":         enriched.Message,
		"service":         enriched.Service,
		"trace_id":        enriched.TraceID,
		"span_id":         enriched.SpanID,
		"http_method":     enriched.HTTPMethod,
		"http_path":       enriched.HTTPPath,
		"http_status":     enriched.HTTPStatus,
		"user_id":         enriched.UserID,
		"business_domain": enriched.BusinessDomain,
		"tenant_id":       enriched.TenantID,
		"is_error":        enriched.IsError,
		"error_type":      enriched.ErrorType,
		"fields":          enriched.Fields,
		"raw":             enriched.Raw,
	}

	// 生成文档ID
	id := fmt.Sprintf("%s-%d", enriched.Service, enriched.Timestamp.UnixNano())

	// 添加到批次
	p.batch = append(p.batch, sink.LogEntry{
		Index:     p.config.ESIndex,
		ID:        id,
		Document:  doc,
		Timestamp: enriched.Timestamp,
	})

	return nil
}

// shouldFlush 检查是否需要刷新批次
func (p *Processor) shouldFlush() bool {
	return len(p.batch) >= p.config.BatchSize ||
		time.Since(p.lastFlush) >= p.config.BatchTimeout
}

// flush 刷新批次到ES
func (p *Processor) flush() error {
	if len(p.batch) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.sink.Write(ctx, p.batch); err != nil {
		return fmt.Errorf("sink write error: %w", err)
	}

	log.Printf("Flushed %d logs to Elasticsearch", len(p.batch))

	// 清空批次
	p.batch = p.batch[:0]
	p.lastFlush = time.Now()

	return nil
}

// Close 关闭处理器
func (p *Processor) Close() error {
	if err := p.flush(); err != nil {
		log.Printf("Error flushing on close: %v", err)
	}

	if err := p.reader.Close(); err != nil {
		return fmt.Errorf("error closing reader: %w", err)
	}

	if err := p.sink.Close(); err != nil {
		return fmt.Errorf("error closing sink: %w", err)
	}

	return nil
}

func main() {
	// 加载配置
	cfg := Config{
		KafkaBrokers: getEnvStrings("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaTopic:   getEnvString("KAFKA_TOPIC", "logs"),
		KafkaGroup:   getEnvString("KAFKA_GROUP", "log-processor"),
		ESAddresses:  getEnvStrings("ES_ADDRESSES", []string{"http://localhost:9200"}),
		ESIndex:      getEnvString("ES_INDEX", "logs"),
		BatchSize:    getEnvInt("BATCH_SIZE", 100),
		BatchTimeout: getEnvDuration("BATCH_TIMEOUT", 5*time.Second),
	}

	// 创建处理器
	processor, err := NewProcessor(cfg)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	// 设置优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// 启动处理
	if err := processor.Run(ctx); err != nil {
		log.Fatalf("Processor error: %v", err)
	}
}

// 环境变量辅助函数

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvStrings(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		var result []string
		if err := json.Unmarshal([]byte(value), &result); err == nil {
			return result
		}
		// 尝试逗号分隔
		return []string{value}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
