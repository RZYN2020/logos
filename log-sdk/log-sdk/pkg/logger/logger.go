package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/log-system/log-sdk/pkg/async"
	"github.com/log-system/log-sdk/pkg/rule"
)

// Hook 是日志钩子接口，在实际打印日志前被调用
type Hook interface {
	OnLog(entry LogEntry) bool
}

// Func 允许使用函数作为 Hook
type Func func(entry LogEntry) bool

func (f Func) OnLog(entry LogEntry) bool {
	return f(entry)
}

// LevelHook 创建一个根据日志级别过滤的 Hook
func LevelHook(minLevel Level) Hook {
	return Func(func(entry LogEntry) bool {
		switch entry.Level {
		case "DEBUG":
			return minLevel <= LevelDebug
		case "INFO":
			return minLevel <= LevelInfo
		case "WARN":
			return minLevel <= LevelWarn
		case "ERROR":
			return minLevel <= LevelError
		case "FATAL":
			return minLevel <= LevelFatal
		case "PANIC":
			return minLevel <= LevelPanic
		default:
			return true
		}
	})
}

// RegexHook 创建一个基于字段正则匹配的 Hook
func RegexHook(field, pattern string) Hook {
	compiled := regexp.MustCompile(pattern)
	return Func(func(entry LogEntry) bool {
		var value string
		switch field {
		case "cluster":
			value = entry.Cluster
		case "pod":
			value = entry.Pod
		case "file":
			value = entry.File
		case "message":
			value = entry.Message
		case "service":
			value = entry.Service
		case "level":
			value = entry.Level
		default:
			// 尝试从 Fields 中获取
			if v, ok := entry.Fields[field]; ok {
				value = fmt.Sprintf("%v", v)
			} else {
				return true // 字段不存在时不过滤
			}
		}
		return compiled.MatchString(value)
	})
}

// LineHook 创建一个基于行号范围过滤的 Hook
func LineHook(min, max int) Hook {
	return Func(func(entry LogEntry) bool {
		return entry.Line >= min && entry.Line <= max
	})
}

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
	Cluster   string                 `json:"cluster,omitempty"`
	Pod       string                 `json:"pod,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	// 内部字段（用于 Hook 过滤）
	File      string                 `json:"-"`
	Line      int                    `json:"-"`
	Function  string                 `json:"-"`
}

// Logger 日志接口
type Logger interface {
	// 传统打印方式
	Printf(format string, args ...interface{})
	Println(args ...interface{})
	Print(args ...interface{})

	// 强类型链式打印方式
	Debug(msg string, fields ...Field) *LogBuilder
	Info(msg string, fields ...Field) *LogBuilder
	Warn(msg string, fields ...Field) *LogBuilder
	Error(msg string, fields ...Field) *LogBuilder
	Fatal(msg string, fields ...Field) *LogBuilder
	Panic(msg string, fields ...Field) *LogBuilder

	// 上下文和字段管理
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger

	// Hook 管理
	AddHook(h Hook) Logger

	// 生命周期管理
	Close() error
}

// LogBuilder 用于强类型链式打印
type LogBuilder struct {
	logger *loggerImpl
	entry  LogEntry
}

func (b *LogBuilder) Str(key, value string) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Int(key string, value int) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Int64(key string, value int64) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Float64(key string, value float64) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Bool(key string, value bool) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Send() {
	b.logger.logEntry(b.entry)
}

// Config 日志配置
type Config struct {
	ServiceName   string
	Environment   string
	Cluster       string
	Pod           string
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
	rule     *rule.Engine
	fields   []Field
	hooks    []Hook
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

	// 创建规则引擎（如果配置了etcd）
	var engine *rule.Engine
	if len(cfg.EtcdEndpoints) > 0 {
		var err error
		engine, err = rule.NewEngine(rule.Config{ServiceName: cfg.ServiceName, Environment: cfg.Environment, EtcdEndpoints: cfg.EtcdEndpoints})
		if err != nil {
			// 规则引擎失败不影响日志记录
			println("Failed to create rule engine:", err.Error())
		}
	}

	return &loggerImpl{
		config:   cfg,
		producer: producer,
		rule: engine,
		fields:   make([]Field, 0),
		hooks:    make([]Hook, 0),
	}
}

func (l *loggerImpl) Printf(format string, args ...interface{}) {
	l.log(LevelInfo, sprintf(format, args...))
}

func (l *loggerImpl) Println(args ...interface{}) {
	l.log(LevelInfo, sprintln(args...))
}

func (l *loggerImpl) Print(args ...interface{}) {
	l.log(LevelInfo, sprint(args...))
}

func (l *loggerImpl) Debug(msg string, fields ...Field) *LogBuilder {
	return l.newLogBuilder(LevelDebug, msg, fields...)
}

func (l *loggerImpl) Info(msg string, fields ...Field) *LogBuilder {
	return l.newLogBuilder(LevelInfo, msg, fields...)
}

func (l *loggerImpl) Warn(msg string, fields ...Field) *LogBuilder {
	return l.newLogBuilder(LevelWarn, msg, fields...)
}

func (l *loggerImpl) Error(msg string, fields ...Field) *LogBuilder {
	return l.newLogBuilder(LevelError, msg, fields...)
}

func (l *loggerImpl) Fatal(msg string, fields ...Field) *LogBuilder {
	builder := l.newLogBuilder(LevelFatal, msg, fields...)
	return &LogBuilder{
		logger: l,
		entry:  builder.entry,
	}
}

func (l *loggerImpl) Panic(msg string, fields ...Field) *LogBuilder {
	builder := l.newLogBuilder(LevelPanic, msg, fields...)
	return &LogBuilder{
		logger: l,
		entry:  builder.entry,
	}
}

func (l *loggerImpl) newLogBuilder(level Level, msg string, fields ...Field) *LogBuilder {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return &LogBuilder{} // 返回空 builder，Send() 会忽略
	}
	l.mu.RUnlock()

	_, file, line, ok := runtime.Caller(2)
	function := "unknown"
	if ok {
		pc, _, _, _ := runtime.Caller(2)
		function = runtime.FuncForPC(pc).Name()
	}

	allFields := make(map[string]interface{})
	for _, f := range l.fields {
		allFields[f.Key] = f.Value
	}
	for _, f := range fields {
		allFields[f.Key] = f.Value
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   msg,
		Service:   l.config.ServiceName,
		Cluster:   l.config.Cluster,
		Pod:       l.config.Pod,
		Fields:    allFields,
		File:      file,
		Line:      line,
		Function:  function,
	}

	return &LogBuilder{
		logger: l,
		entry:  entry,
	}
}

func (l *loggerImpl) logEntry(entry LogEntry) {
	for _, h := range l.hooks {
		if !h.OnLog(entry) {
			return
		}
	}

	if l.rule != nil {
		decision := l.rule.Evaluate(entry.Level, l.config.ServiceName, l.config.Environment, entry.Fields)
		if !decision.ShouldLog {
			return
		}
	}

	if traceID, ok := entry.Fields["trace_id"]; ok {
		entry.TraceID = traceID.(string)
	}
	if spanID, ok := entry.Fields["span_id"]; ok {
		entry.SpanID = spanID.(string)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		if l.config.FallbackToConsole {
			println("LOG ERROR:", err.Error())
		}
		return
	}

	msg := async.LogMessage{
		Topic: l.config.KafkaTopic,
		Key:   entry.Service,
		Value: data,
		Headers: map[string]string{
			"level":   entry.Level,
			"service": entry.Service,
		},
	}

	if err := l.producer.Send(msg); err != nil {
		if l.config.FallbackToConsole {
			println(string(data))
		}
	}

	if entry.Level == LevelFatal.String() {
		os.Exit(1)
	}

	if entry.Level == LevelPanic.String() {
		panic(entry.Message)
	}
}

func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func sprintln(args ...interface{}) string {
	return fmt.Sprintln(args...)
}

func sprint(args ...interface{}) string {
	return fmt.Sprint(args...)
}

func (l *loggerImpl) With(fields ...Field) Logger {
	newLogger := &loggerImpl{
		config:   l.config,
		producer: l.producer,
		rule: l.rule,
		fields:   append(l.fields, fields...),
		hooks:    append([]Hook(nil), l.hooks...),
	}
	return newLogger
}

func (l *loggerImpl) AddHook(h Hook) Logger {
	newLogger := &loggerImpl{
		config:   l.config,
		producer: l.producer,
		rule: l.rule,
		fields:   append([]Field(nil), l.fields...),
		hooks:    append(append([]Hook(nil), l.hooks...), h),
	}
	return newLogger
}

func (l *loggerImpl) WithContext(ctx context.Context) Logger {
	return l
}

func (l *loggerImpl) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// 关闭规则引擎
	if l.rule != nil {
		l.rule.Close()
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

	// 获取调用栈信息（用于行号、文件名、函数名）
	_, file, line, ok := runtime.Caller(2)
	function := "unknown"
	if ok {
		// 尝试获取函数名
		pc, _, _, _ := runtime.Caller(2)
		function = runtime.FuncForPC(pc).Name()
	}

	// 合并字段
	allFields := make(map[string]interface{})
	for _, f := range l.fields {
		allFields[f.Key] = f.Value
	}
	for _, f := range fields {
		allFields[f.Key] = f.Value
	}

	// 构建日志条目（轻量级，不包含复杂语义处理）
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   msg,
		Service:   l.config.ServiceName,
		Cluster:   l.config.Cluster,
		Pod:       l.config.Pod,
		Fields:    allFields,
		File:      file,
		Line:      line,
		Function:  function,
	}

	// Hook 过滤
	for _, h := range l.hooks {
		if !h.OnLog(entry) {
			return // 被 Hook 过滤
		}
	}

	// 规则评估
	if l.rule != nil {
		decision := l.rule.Evaluate(level.String(), l.config.ServiceName, l.config.Environment, allFields)
		if !decision.ShouldLog {
			return // 被策略过滤
		}
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

// logEntryPool 是 LogEntry 对象池，用于减少 GC 压力
var logEntryPool = sync.Pool{
	New: func() interface{} {
		return &LogEntry{
			Fields: make(map[string]interface{}, 8),
		}
	},
}

// acquireLogEntry 从对象池获取 LogEntry
func acquireLogEntry() *LogEntry {
	entry := logEntryPool.Get().(*LogEntry)
	// 重置字段
	entry.Timestamp = time.Time{}
	entry.Level = ""
	entry.Message = ""
	entry.File = ""
	entry.Line = 0
	entry.Function = ""
	for k := range entry.Fields {
		delete(entry.Fields, k)
	}
	return entry
}

// releaseLogEntry 将 LogEntry 归还到对象池
func releaseLogEntry(entry *LogEntry) {
	if entry == nil {
		return
	}
	// 清理字段避免内存泄漏
	for k := range entry.Fields {
		delete(entry.Fields, k)
	}
	logEntryPool.Put(entry)
}
