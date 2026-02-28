// Package main 日志处理器入口
// 从 Kafka 消费日志，经过解析、过滤、文本分析、转换、语义增强后写入 Elasticsearch
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

	"github.com/log-system/log-processor/pkg/analyzer"
	"github.com/log-system/log-processor/pkg/config"
	"github.com/log-system/log-processor/pkg/filter"
	"github.com/log-system/log-processor/pkg/parser"
	"github.com/log-system/log-processor/pkg/semantic"
	"github.com/log-system/log-processor/pkg/sink"
	"github.com/log-system/log-processor/pkg/transformer"
	"github.com/segmentio/kafka-go"
)

// Config 处理器配置
type Config struct {
	KafkaBrokers    []string
	KafkaTopic      string
	KafkaGroup      string
	ESAddresses     []string
	ESIndex         string
	BatchSize       int
	BatchTimeout    time.Duration
	EtcdEndpoints   []string
	EnableFiltering bool
	EnableTransform bool
}

// Processor 日志处理器
type Processor struct {
	config       Config
	reader       *kafka.Reader
	parser       *parser.ExtendedMultiParser
	filterEngine *filter.FilterEngineImpl
	analyzer     *analyzer.TextAnalyzerImpl
	transformer  *transformer.TransformerImpl
	builder      *semantic.Builder
	sink         sink.Sink
	configMgr    config.ConfigManager
	batch        []sink.LogEntry
	lastFlush    time.Time
}

// NewProcessor 创建处理器
func NewProcessor(cfg Config) (*Processor, error) {
	// 创建 Kafka Reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.KafkaBrokers,
		Topic:    cfg.KafkaTopic,
		GroupID:  cfg.KafkaGroup,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	// 创建扩展解析器
	p := parser.NewExtendedMultiParser()

	// 创建过滤器引擎
	filterEngine := filter.NewFilterEngine()

	// 创建文本分析器
	textAnalyzer := analyzer.NewTextAnalyzer()

	// 创建转换器
	transformer := transformer.NewTransformer()

	// 创建语义构建器
	builder := semantic.NewBuilder()

	// 创建配置管理器（如果启用了 ETCD）
	var configMgr config.ConfigManager
	if cfg.EnableFiltering && len(cfg.EtcdEndpoints) > 0 {
		mgrCfg := config.ConfigManagerConfig{
			EtcdEndpoints:   cfg.EtcdEndpoints,
			RefreshInterval: 30 * time.Second,
		}
		var err error
		configMgr, err = config.NewEtcdConfigManager(mgrCfg)
		if err != nil {
			log.Printf("Warning: failed to create config manager: %v", err)
		} else {
			// 加载过滤配置
			if err := filterEngine.LoadFilters(configMgr); err != nil {
				log.Printf("Warning: failed to load filters: %v", err)
			}
		}
	}

	// 创建 Sink
	s, err := sink.NewElasticsearchSink(cfg.ESAddresses, cfg.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create sink: %w", err)
	}

	return &Processor{
		config:       cfg,
		reader:       reader,
		parser:       p,
		filterEngine: filterEngine,
		analyzer:     textAnalyzer,
		transformer:  transformer,
		builder:      builder,
		sink:         s,
		configMgr:    configMgr,
		batch:        make([]sink.LogEntry, 0, cfg.BatchSize),
		lastFlush:    time.Now(),
	}, nil
}

// Run 启动处理循环
func (p *Processor) Run(ctx context.Context) error {
	log.Println("Starting log processor with enhanced ETL pipeline...")

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
	// 1. 解析日志
	parsed, err := p.parser.Parse(msg.Value)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// 2. 应用过滤（如果启用）
	if p.config.EnableFiltering && p.filterEngine != nil {
		filterEntry := &filter.ParsedLog{
			Timestamp: parsed.Timestamp,
			Level:     parsed.Level,
			Message:   parsed.Message,
			Service:   parsed.Service,
			TraceID:   parsed.TraceID,
			SpanID:    parsed.SpanID,
			Fields:    parsed.Fields,
			Raw:       parsed.Raw,
		}

		filterResult := p.filterEngine.ApplyFilters(filterEntry)
		if !filterResult.ShouldKeep {
			// 日志被过滤掉
			log.Printf("Log filtered by rule: %s", filterResult.MatchedRule)
			return nil
		}
	}

	// 3. 文本分析（用于非结构化日志）
	var analysis *analyzer.AnalysisResult
	if parsed.Format == parser.FormatUnstructured {
		analysis, _ = p.analyzer.Analyze(parsed.Message)
	}

	// 4. 转换（如果启用）
	var transformed *transformer.TransformedLog
	if p.config.EnableTransform {
		transformed, _ = p.transformer.Transform(parsed, analysis)
	}

	// 5. 转换为 LogEntry
	entry := &semantic.LogEntry{
		Timestamp:   parsed.Timestamp,
		Level:       parsed.Level,
		Message:     parsed.Message,
		Service:     parsed.Service,
		TraceID:     parsed.TraceID,
		SpanID:      parsed.SpanID,
		Fields:      parsed.Fields,
		Raw:         parsed.Raw,
		Environment: "",
		Host:        "",
	}

	// 如果有转换结果，使用转换后的字段
	if transformed != nil {
		for k, v := range transformed.ExtractedFields {
			entry.Fields[k] = v
		}
	}

	// 6. 语义增强
	ctx := context.Background()
	enriched := p.builder.Build(ctx, entry)

	// 7. 转换为 sink 格式
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
	}

	// 添加分析结果字段
	if analysis != nil {
		doc["sentiment_score"] = analysis.Sentiment.Score
		doc["sentiment_label"] = analysis.Sentiment.Label
		doc["language"] = analysis.Language
		doc["category"] = analysis.Category
		doc["keywords"] = analysis.Keywords
		doc["entities"] = analysis.Entities
	}

	// 生成文档 ID
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

// flush 刷新批次到 ES
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

	if p.configMgr != nil {
		if err := p.configMgr.Close(); err != nil {
			log.Printf("Error closing config manager: %v", err)
		}
	}

	return nil
}

func main() {
	// 加载配置
	cfg := Config{
		KafkaBrokers:    getEnvStrings("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaTopic:      getEnvString("KAFKA_TOPIC", "logs"),
		KafkaGroup:      getEnvString("KAFKA_GROUP", "log-processor"),
		ESAddresses:     getEnvStrings("ES_ADDRESSES", []string{"http://localhost:9200"}),
		ESIndex:         getEnvString("ES_INDEX", "logs"),
		BatchSize:       getEnvInt("BATCH_SIZE", 100),
		BatchTimeout:    getEnvDuration("BATCH_TIMEOUT", 5*time.Second),
		EtcdEndpoints:   getEnvStrings("ETCD_ENDPOINTS", []string{"localhost:2379"}),
		EnableFiltering: getEnvBool("ENABLE_FILTERING", false),
		EnableTransform: getEnvBool("ENABLE_TRANSFORM", false),
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch value {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}
