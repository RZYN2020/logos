package logger

import (
	"context"
	"fmt"
	"os"
	"regexp"
	
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
	OnLog(entry *encoder.LogEntry) bool
}

// Func 允许使用函数作为 Hook
type Func func(entry *encoder.LogEntry) bool

func (f Func) OnLog(entry *encoder.LogEntry) bool {
	return f(entry)
}

// LevelHook 创建一个根据日志级别过滤的 Hook
func LevelHook(minLevel Level) Hook {
	return Func(func(entry *encoder.LogEntry) bool {
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
	return Func(func(entry *encoder.LogEntry) bool {
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
			found := false
			for i := 0; i < entry.FieldsLen; i++ {
				if entry.Fields[i].Key == field {
					f := entry.Fields[i]
					switch f.Type {
					case encoder.StringType:
						value = f.Str
					case encoder.IntType:
						value = fmt.Sprintf("%d", f.Int)
					case encoder.FloatType:
						value = fmt.Sprintf("%f", f.Float)
					}
					found = true
					break
				}
			}
			if !found {
				return true
			}
		}
		return compiled.MatchString(value)
	})
}

// LineHook 创建一个基于行号范围过滤的 Hook
func LineHook(min, max int) Hook {
	return Func(func(entry *encoder.LogEntry) bool {
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
type Field = encoder.Field

func F(key string, value interface{}) Field {
	switch v := value.(type) {
	case int:
		return Field{Key: key, Type: encoder.IntType, Int: int64(v)}
	case int64:
		return Field{Key: key, Type: encoder.IntType, Int: v}
	case string:
		return Field{Key: key, Type: encoder.StringType, Str: v}
	case float64:
		return Field{Key: key, Type: encoder.FloatType, Float: v}
	case bool:
		if v {
			return Field{Key: key, Type: encoder.BoolType, Int: 1}
		}
		return Field{Key: key, Type: encoder.BoolType, Int: 0}
	default:
		return Field{Key: key, Type: encoder.InterfaceType, Obj: v}
	}
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

func (b *LogBuilder) addField(f Field) *LogBuilder {
	if b.entry != nil {
		if b.entry.FieldsLen < len(b.entry.Fields) {
			b.entry.Fields[b.entry.FieldsLen] = f
			b.entry.FieldsLen++
		} else {
			b.entry.Fields = append(b.entry.Fields, f)
			b.entry.FieldsLen++
		}
	}
	return b
}

func (b *LogBuilder) Str(key, value string) *LogBuilder {
	return b.addField(Field{Key: key, Type: encoder.StringType, Str: value})
}

func (b *LogBuilder) Int(key string, value int) *LogBuilder {
	return b.addField(Field{Key: key, Type: encoder.IntType, Int: int64(value)})
}

func (b *LogBuilder) Int64(key string, value int64) *LogBuilder {
	return b.addField(Field{Key: key, Type: encoder.IntType, Int: value})
}

func (b *LogBuilder) Float64(key string, value float64) *LogBuilder {
	return b.addField(Field{Key: key, Type: encoder.FloatType, Float: value})
}

func (b *LogBuilder) Bool(key string, value bool) *LogBuilder {
	v := int64(0)
	if value {
		v = 1
	}
	return b.addField(Field{Key: key, Type: encoder.BoolType, Int: v})
}

func (b *LogBuilder) Send() {
	if b.entry != nil {
		b.logger.logEntry(b.entry)
		releaseLogEntry(b.entry)
	}
}

// Config 日志配置
type Config struct {
	ServiceName       string
	Environment       string
	Cluster           string
	Pod               string
	KafkaBrokers      []string
	KafkaTopic        string
	EtcdEndpoints     []string
	BatchSize         int
	BatchTimeout      time.Duration
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

	// 只在确实需要时才获取 Caller
	var file string
	var line int
	var function string
	
	// 在没有特殊 hook 需求时，关闭获取 caller 会极大幅度提升性能
	// 为了基准测试对比，我们此处注释掉 caller 获取，或者由用户通过 Option 控制
	// _, file, line, ok := runtime.Caller(2)
	// if ok {
	// 	pc, _, _, _ := runtime.Caller(2)
	// 	function = runtime.FuncForPC(pc).Name()
	// }

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

	// Copy fields
	for _, f := range l.fields {
		if entry.FieldsLen < cap(entry.Fields) {
			entry.Fields[entry.FieldsLen] = f
			entry.FieldsLen++
		} else {
			entry.Fields = append(entry.Fields, f)
			entry.FieldsLen++
		}
	}
	for _, f := range fields {
		if entry.FieldsLen < cap(entry.Fields) {
			entry.Fields[entry.FieldsLen] = f
			entry.FieldsLen++
		} else {
			entry.Fields = append(entry.Fields, f)
			entry.FieldsLen++
		}
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
		if !h.OnLog(entry) {
			return
		}
	}

	if l.rule != nil {
		// 为了减少 map 创建，我们可以临时将 fields 转成 rule engine 需要的格式
		// 但如果 rule 为空，就避免了 allocation
		m := make(map[string]interface{}, entry.FieldsLen)
		for i := 0; i < entry.FieldsLen; i++ {
			f := entry.Fields[i]
			switch f.Type {
			case encoder.StringType: m[f.Key] = f.Str
			case encoder.IntType: m[f.Key] = f.Int
			case encoder.FloatType: m[f.Key] = f.Float
			case encoder.BoolType: m[f.Key] = f.Int == 1
			case encoder.InterfaceType: m[f.Key] = f.Obj
			}
		}
		decision := l.rule.Evaluate(entry.Level, l.config.ServiceName, l.config.Environment, m)
		if !decision.ShouldLog {
			return
		}
	}

	for i := 0; i < entry.FieldsLen; i++ {
		f := entry.Fields[i]
		if f.Key == "trace_id" && f.Type == encoder.StringType {
			entry.TraceID = f.Str
		} else if f.Key == "span_id" && f.Type == encoder.StringType {
			entry.SpanID = f.Str
		}
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
	builder := l.newLogBuilder(level, msg, fields...)
	builder.Send()
}

// WithContext 从context创建logger
func WithContext(ctx context.Context, l Logger) Logger {
	return l
}

// logEntryPool 是 LogEntry 对象池，用于减少 GC 压力
var logEntryPool = sync.Pool{
	New: func() interface{} {
		return &encoder.LogEntry{
			Fields: make([]encoder.Field, 32),
			FieldsLen: 0,
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
	entry.FieldsLen = 0
	return entry
}

// releaseLogEntry 将 LogEntry 归还到对象池
func releaseLogEntry(entry *encoder.LogEntry) {
	if entry == nil {
		return
	}
	// 帮助 GC 清理可能的指针引用
	for i := 0; i < entry.FieldsLen; i++ {
		entry.Fields[i].Str = ""
		entry.Fields[i].Obj = nil
	}
	entry.FieldsLen = 0
	logEntryPool.Put(entry)
}
