package logger

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/log-system/log-sdk/pkg/async"
	"github.com/log-system/log-sdk/pkg/encoder"
	"github.com/log-system/log-sdk/pkg/guard"
	"github.com/log-system/log-sdk/pkg/rule"
)

// Hook 是日志钩子接口，在实际打印日志前被调用
type Hook interface {
	OnLog(entry encoder.LogEntry) bool
}

// Func 允许使用函数作为 Hook
type Func func(entry encoder.LogEntry) bool

func (f Func) OnLog(entry encoder.LogEntry) bool {
	return f(entry)
}

// LevelHook 创建一个根据日志级别过滤的 Hook
func LevelHook(minLevel Level) Hook {
	return Func(func(entry encoder.LogEntry) bool {
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
	return Func(func(entry encoder.LogEntry) bool {
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
	return Func(func(entry encoder.LogEntry) bool {
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

// Logger 日志接口
type Logger interface {
	Printf(format string, args ...interface{})
	Println(args ...interface{})
	Print(args ...interface{})

	Debug(msg string, fields ...Field) *LogBuilder
	Info(msg string, fields ...Field) *LogBuilder
	Warn(msg string, fields ...Field) *LogBuilder
	Error(msg string, fields ...Field) *LogBuilder
	Fatal(msg string, fields ...Field) *LogBuilder
	Panic(msg string, fields ...Field) *LogBuilder

	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger

	AddHook(h Hook) Logger
	Close() error
}

// LogBuilder 用于强类型链式打印
type LogBuilder struct {
	logger *loggerImpl
	entry  *encoder.LogEntry
}

func (b *LogBuilder) Str(key, value string) *LogBuilder {
	if b.entry != nil {
		b.entry.Fields[key] = value
	}
	return b
}

func (b *LogBuilder) Int(key string, value int) *LogBuilder {
	if b.entry != nil {
		b.entry.Fields[key] = value
	}
	return b
}

func (b *LogBuilder) Int64(key string, value int64) *LogBuilder {
	if b.entry != nil {
		b.entry.Fields[key] = value
	}
	return b
}

func (b *LogBuilder) Float64(key string, value float64) *LogBuilder {
	if b.entry != nil {
		b.entry.Fields[key] = value
	}
	return b
}

func (b *LogBuilder) Bool(key string, value bool) *LogBuilder {
	if b.entry != nil {
		b.entry.Fields[key] = value
	}
	return b
}

func (b *LogBuilder) Send() {
	if b.entry != nil {
		b.logger.logEntry(b.entry)
		releaseLogEntry(b.entry)
	}
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
	FallbackToConsole bool
	MaxBufferSize     int
	RateLimit         int64
}

// loggerImpl 日志器实现
type loggerImpl struct {
	config   Config
	producer *async.Producer
	enc      *encoder.JSONEncoder
	guard    *guard.TokenBucketGuard
	rule     *rule.Engine
	fields   []Field
	hooks    []Hook
	closed   atomic.Bool
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
	if cfg.RateLimit == 0 {
		cfg.RateLimit = 10000 // default 10k ops/sec
	}

	producer := async.NewProducer(cfg.KafkaBrokers, cfg.BatchSize, cfg.BatchTimeout)
	enc := encoder.DefaultJSONEncoder()
	tokenGuard := guard.NewTokenBucketGuard(cfg.RateLimit, cfg.RateLimit, time.Second)

	var engine *rule.Engine
	if len(cfg.EtcdEndpoints) > 0 {
		var err error
		engine, err = rule.NewEngine(rule.Config{ServiceName: cfg.ServiceName, Environment: cfg.Environment, EtcdEndpoints: cfg.EtcdEndpoints})
		if err != nil {
			println("Failed to create rule engine:", err.Error())
		}
	}

	return &loggerImpl{
		config:   cfg,
		producer: producer,
		enc:      enc,
		guard:    tokenGuard,
		rule:     engine,
		fields:   make([]Field, 0),
		hooks:    make([]Hook, 0),
	}
}

func (l *loggerImpl) Printf(format string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(format, args...))
}

func (l *loggerImpl) Println(args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintln(args...))
}

func (l *loggerImpl) Print(args ...interface{}) {
	l.log(LevelInfo, fmt.Sprint(args...))
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
	return l.newLogBuilder(LevelFatal, msg, fields...)
}

func (l *loggerImpl) Panic(msg string, fields ...Field) *LogBuilder {
	return l.newLogBuilder(LevelPanic, msg, fields...)
}

func (l *loggerImpl) newLogBuilder(level Level, msg string, fields ...Field) *LogBuilder {
	if l.closed.Load() {
		return &LogBuilder{}
	}

	_, file, line, ok := runtime.Caller(2)
	function := "unknown"
	if ok {
		pc, _, _, _ := runtime.Caller(2)
		function = runtime.FuncForPC(pc).Name()
	}

	entry := acquireLogEntry()
	entry.Timestamp = time.Now().UTC()
	entry.Level = level.String()
	entry.Message = msg
	entry.Service = l.config.ServiceName
	entry.Cluster = l.config.Cluster
	entry.Pod = l.config.Pod
	entry.File = file
	entry.Line = line
	entry.Function = function

	for _, f := range l.fields {
		entry.Fields[f.Key] = f.Value
	}
	for _, f := range fields {
		entry.Fields[f.Key] = f.Value
	}

	return &LogBuilder{
		logger: l,
		entry:  entry,
	}
}

func (l *loggerImpl) logEntry(entry *encoder.LogEntry) {
	if !l.guard.Allow() {
		return
	}

	for _, h := range l.hooks {
		if !h.OnLog(*entry) {
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

	data, err := l.enc.EncodeToBytes(*entry)
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

func (l *loggerImpl) With(fields ...Field) Logger {
	newLogger := &loggerImpl{
		config:   l.config,
		producer: l.producer,
		enc:      l.enc,
		guard:    l.guard,
		rule:     l.rule,
		fields:   append(l.fields, fields...),
		hooks:    append([]Hook(nil), l.hooks...),
	}
	return newLogger
}

func (l *loggerImpl) AddHook(h Hook) Logger {
	newLogger := &loggerImpl{
		config:   l.config,
		producer: l.producer,
		enc:      l.enc,
		guard:    l.guard,
		rule:     l.rule,
		fields:   append([]Field(nil), l.fields...),
		hooks:    append(append([]Hook(nil), l.hooks...), h),
	}
	return newLogger
}

func (l *loggerImpl) WithContext(ctx context.Context) Logger {
	return l
}

func (l *loggerImpl) Close() error {
	if l.closed.Swap(true) {
		return nil
	}

	if l.rule != nil {
		l.rule.Close()
	}

	return l.producer.Close()
}

func (l *loggerImpl) log(level Level, msg string, fields ...Field) {
	if l.closed.Load() {
		return
	}

	if !l.guard.Allow() {
		return
	}

	_, file, line, ok := runtime.Caller(2)
	function := "unknown"
	if ok {
		pc, _, _, _ := runtime.Caller(2)
		function = runtime.FuncForPC(pc).Name()
	}

	entry := acquireLogEntry()
	defer releaseLogEntry(entry)

	entry.Timestamp = time.Now().UTC()
	entry.Level = level.String()
	entry.Message = msg
	entry.Service = l.config.ServiceName
	entry.Cluster = l.config.Cluster
	entry.Pod = l.config.Pod
	entry.File = file
	entry.Line = line
	entry.Function = function

	for _, f := range l.fields {
		entry.Fields[f.Key] = f.Value
	}
	for _, f := range fields {
		entry.Fields[f.Key] = f.Value
	}

	for _, h := range l.hooks {
		if !h.OnLog(*entry) {
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

	data, err := l.enc.EncodeToBytes(*entry)
	if err != nil {
		if l.config.FallbackToConsole {
			println("LOG ERROR:", err.Error())
		}
		return
	}

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
		if l.config.FallbackToConsole {
			println(string(data))
		}
	}
}

// WithContext 从context创建logger
func WithContext(ctx context.Context, l Logger) Logger {
	return l
}

// logEntryPool 是 LogEntry 对象池，用于减少 GC 压力
var logEntryPool = sync.Pool{
	New: func() interface{} {
		return &encoder.LogEntry{
			Fields: make(map[string]interface{}, 8),
		}
	},
}

// acquireLogEntry 从对象池获取 LogEntry
func acquireLogEntry() *encoder.LogEntry {
	entry := logEntryPool.Get().(*encoder.LogEntry)
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
func releaseLogEntry(entry *encoder.LogEntry) {
	if entry == nil {
		return
	}
	for k := range entry.Fields {
		delete(entry.Fields, k)
	}
	logEntryPool.Put(entry)
}
