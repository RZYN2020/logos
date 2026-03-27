package logger

import (
	"testing"
	"github.com/log-system/log-sdk/pkg/encoder"
	"time"
)

// TestNew 测试 Logger 创建
func TestNew(t *testing.T) {
	cfg := Config{
		ServiceName:       "test-service",
		Environment:       "test",
		Cluster:           "test-cluster",
		Pod:               "test-pod-1",
		KafkaBrokers:      []string{"localhost:9092"},
		KafkaTopic:        "test-logs",
		BatchSize:         100,
		BatchTimeout:      100 * time.Millisecond,
		FallbackToConsole: true,
		MaxBufferSize:     1000,
	}

	log := New(cfg)
	if log == nil {
		t.Fatal("New() returned nil")
	}

	// 测试关闭
	if err := log.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestLogger_Printf 测试传统打印方式
func TestLogger_Printf(t *testing.T) {
	log := New(Config{
		ServiceName:       "test",
		FallbackToConsole: true,
	})
	defer log.Close()

	// 测试 Printf
	log.Printf("Test message: %s, %d", "string", 42)

	// 测试 Println
	log.Println("Test println message")

	// 测试 Print
	log.Print("Test print message")
}

// TestLogger_TraditionalStyle 测试传统风格 API
func TestLogger_TraditionalStyle(t *testing.T) {
	log := New(Config{
		ServiceName:       "test-service",
		FallbackToConsole: true,
	})
	defer log.Close()

	// 测试各级别的传统风格调用
	log.Debug("Debug message", F("key", "value"))
	log.Info("Info message", F("user_id", "123"))
	log.Warn("Warn message", F("warning", "test"))
	log.Error("Error message", F("error", "test error"))
}

// TestLogger_ChainStyle 测试链式风格 API
func TestLogger_ChainStyle(t *testing.T) {
	log := New(Config{
		ServiceName:       "test-service",
		FallbackToConsole: true,
	})
	defer log.Close()

	// 测试链式调用
	log.Info("Chain test").
		Str("string_field", "value").
		Int("int_field", 42).
		Int64("int64_field", 9223372036854775807).
		Float64("float_field", 3.14159).
		Bool("bool_field", true).
		Send()
}

// TestLogBuilder_Methods 测试 LogBuilder 各个方法
func TestLogBuilder_Methods(t *testing.T) {
	log := New(Config{
		ServiceName:       "test",
		FallbackToConsole: true,
	})
	defer log.Close()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "Str",
			fn: func() {
				log.Info("test").Str("key", "value").Send()
			},
		},
		{
			name: "Int",
			fn: func() {
				log.Info("test").Int("count", 100).Send()
			},
		},
		{
			name: "Int64",
			fn: func() {
				log.Info("test").Int64("big_num", 123456789012345).Send()
			},
		},
		{
			name: "Float64",
			fn: func() {
				log.Info("test").Float64("pi", 3.14159).Send()
			},
		},
		{
			name: "Bool",
			fn: func() {
				log.Info("test").Bool("enabled", true).Send()
			},
		},
		{
			name: "Mixed",
			fn: func() {
				log.Info("test").
					Str("a", "b").
					Int("c", 1).
					Float64("d", 1.5).
					Bool("e", false).
					Send()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 不应该 panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Test %s panicked: %v", tt.name, r)
				}
			}()
			tt.fn()
		})
	}
}

// TestLogger_With 测试 With 字段继承
func TestLogger_With(t *testing.T) {
	log := New(Config{
		ServiceName:       "test",
		FallbackToConsole: true,
	})
	defer log.Close()

	// 创建带默认字段的 logger
	logWithFields := log.With(
		F("request_id", "req-123"),
		F("trace_id", "trace-456"),
	)

	// 测试传统风格继承字段
	logWithFields.Info("test with fields")

	// 测试链式风格继承字段
	logWithFields.Info("chain with fields").Str("extra", "value").Send()
}

// TestLevelHook 测试 LevelHook
func TestLevelHook(t *testing.T) {
	tests := []struct {
		name      string
		minLevel  Level
		entryLevel string
		wantPass  bool
	}{
		{"DEBUG pass DEBUG", LevelDebug, "DEBUG", true},
		{"DEBUG pass INFO", LevelDebug, "INFO", true},
		{"INFO block DEBUG", LevelInfo, "DEBUG", false},
		{"INFO pass INFO", LevelInfo, "INFO", true},
		{"INFO pass ERROR", LevelInfo, "ERROR", true},
		{"ERROR block INFO", LevelError, "INFO", false},
		{"ERROR pass ERROR", LevelError, "ERROR", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := LevelHook(tt.minLevel)
			entry := encoder.LogEntry{Level: tt.entryLevel}

			got := hook.OnLog(entry)
			if got != tt.wantPass {
				t.Errorf("LevelHook(%v).OnLog(%v) = %v, want %v",
					tt.minLevel, tt.entryLevel, got, tt.wantPass)
			}
		})
	}
}

// TestLineHook 测试 LineHook
func TestLineHook(t *testing.T) {
	tests := []struct {
		name    string
		minLine int
		maxLine int
		entryLine int
		wantPass bool
	}{
		{"line in range", 10, 20, 15, true},
		{"line at min", 10, 20, 10, true},
		{"line at max", 10, 20, 20, true},
		{"line below range", 10, 20, 5, false},
		{"line above range", 10, 20, 25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := LineHook(tt.minLine, tt.maxLine)
			entry := encoder.LogEntry{Line: tt.entryLine}

			got := hook.OnLog(entry)
			if got != tt.wantPass {
				t.Errorf("LineHook(%d, %d).OnLog(%d) = %v, want %v",
					tt.minLine, tt.maxLine, tt.entryLine, got, tt.wantPass)
			}
		})
	}
}

// TestRegexHook 测试 RegexHook
func TestRegexHook(t *testing.T) {
	// 注意：当前 RegexHook 是简化实现，始终返回 true
	// 这里测试接口兼容性
	hook := RegexHook("cluster", "prod-.*")
	entry := encoder.LogEntry{Cluster: "prod-cluster-1"}

	// 简化实现始终返回 true
	if !hook.OnLog(entry) {
		t.Error("RegexHook should return true in simplified implementation")
	}
}

// TestLogger_AddHook 测试 AddHook 方法
func TestLogger_AddHook(t *testing.T) {
	log := New(Config{
		ServiceName:       "test",
		FallbackToConsole: true,
	})
	defer log.Close()

	// 添加 hook
	logWithHook := log.AddHook(LevelHook(LevelInfo))
	if logWithHook == nil {
		t.Error("AddHook() returned nil")
	}

	// 测试添加 hook 后的日志记录
	logWithHook.Info("test with hook")
}

// TestField 测试 Field 创建
func TestField(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "key", "value"},
		{"int", "count", 42},
		{"float", "pi", 3.14},
		{"bool", "enabled", true},
		{"struct", "data", "struct_value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := F(tt.key, tt.value)
			if f.Key != tt.key {
				t.Errorf("F() Key = %v, want %v", f.Key, tt.key)
			}
			if f.Value != tt.value {
				t.Errorf("F() Value = %v, want %v", f.Value, tt.value)
			}
		})
	}
}

// TestLevel_String 测试 Level 字符串转换
func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{LevelPanic, "PANIC"},
		{Level(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmarks are in performance_test.go
