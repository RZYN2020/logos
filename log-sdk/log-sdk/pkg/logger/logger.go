package logger

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/log-system/log-sdk/pkg/async"
	"github.com/log-system/log-sdk/pkg/strategy"
)

// Level 日志级别
type Level int8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	case LevelPanic:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// Field 结构化日志字段
type Field struct {
	Key   string
	Value interface{}
}

func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// LogEntry 日志条目（简化版，只包含基础字段）
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Panic(msg string, fields ...Field)
	With(fields ...Field) Logger
	Close() error
}

// Config 日志配置
type Config struct {
	ServiceName   string
	Environment   string
	KafkaBrokers  []string
	KafkaTopic    string
	EtcdEndpoints []string
	BatchSize     int
	BatchTimeout  time.Duration
	// 降级配置
	FallbackToConsole bool
	MaxBufferSize     int
}

// loggerImpl 日志器实现
type loggerImpl struct {
	config   Config
	producer *async.Producer
	strategy *strategy.Engine
	fields   []Field
	mu       sync.RWMutex
	closed   bool
}

// New 创建日志器
func New(cfg Config) Logger {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 100 * time.Millisecond
	}
	if cfg.MaxBufferSize == 0 {
		cfg.MaxBufferSize = 10000
	}

	// 创建异步生产者
	producer := async.NewProducer(cfg.KafkaBrokers, cfg.BatchSize, cfg.BatchTimeout)

	// 创建策略引擎（如果配置了etcd）
	var engine *strategy.Engine
	if len(cfg.EtcdEndpoints) > 0 {
		var err error
		engine, err = strategy.NewEngine(cfg.EtcdEndpoints)
		if err != nil {
			// 策略引擎失败不影响日志记录
			println("Failed to create strategy engine:", err.Error())
		}
	}

	return &loggerImpl{
		config:   cfg,
		producer: producer,
		strategy: engine,
		fields:   make([]Field, 0),
	}
}

func (l *loggerImpl) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

func (l *loggerImpl) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

func (l *loggerImpl) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

func (l *loggerImpl) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

func (l *loggerImpl) Fatal(msg string, fields ...Field) {
	l.log(LevelFatal, msg, fields...)
	os.Exit(1)
}

func (l *loggerImpl) Panic(msg string, fields ...Field) {
	l.log(LevelPanic, msg, fields...)
	panic(msg)
}

func (l *loggerImpl) With(fields ...Field) Logger {
	newLogger := &loggerImpl{
		config:   l.config,
		producer: l.producer,
		strategy: l.strategy,
		fields:   append(l.fields, fields...),
	}
	return newLogger
}

func (l *loggerImpl) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// 关闭策略引擎
	if l.strategy != nil {
		l.strategy.Close()
	}

	// 关闭生产者
	return l.producer.Close()
}

func (l *loggerImpl) log(level Level, msg string, fields ...Field) {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	// 合并字段
	allFields := make(map[string]interface{})
	for _, f := range l.fields {
		allFields[f.Key] = f.Value
	}
	for _, f := range fields {
		allFields[f.Key] = f.Value
	}

	// 策略评估
	if l.strategy != nil {
		decision := l.strategy.Evaluate(level.String(), l.config.ServiceName, l.config.Environment, allFields)
		if !decision.ShouldLog {
			return // 被策略过滤
		}
	}

	// 构建日志条目（轻量级，不包含复杂语义处理）
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   msg,
		Service:   l.config.ServiceName,
		Fields:    allFields,
	}

	// 从context提取trace信息（如果有）
	if traceID, ok := allFields["trace_id"]; ok {
		entry.TraceID = traceID.(string)
	}
	if spanID, ok := allFields["span_id"]; ok {
		entry.SpanID = spanID.(string)
	}

	// 序列化
	data, err := json.Marshal(entry)
	if err != nil {
		// 降级到控制台
		if l.config.FallbackToConsole {
			println("LOG ERROR:", err.Error())
		}
		return
	}

	// 异步发送
	msg2 := async.LogMessage{
		Topic: l.config.KafkaTopic,
		Key:   entry.Service,
		Value: data,
		Headers: map[string]string{
			"level":   entry.Level,
			"service": entry.Service,
		},
	}

	if err := l.producer.Send(msg2); err != nil {
		// 发送失败，降级到控制台
		if l.config.FallbackToConsole {
			println(string(data))
		}
	}
}

// WithContext 从context创建logger
func WithContext(ctx context.Context, l Logger) Logger {
	// 从context提取trace信息
	// 简化实现，实际应集成OpenTelemetry
	return l
}
