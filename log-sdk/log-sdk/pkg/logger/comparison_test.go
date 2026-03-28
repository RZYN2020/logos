package logger

import (
	"log"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/rs/zerolog"
	"github.com/log-system/log-sdk/pkg/async"
	"github.com/log-system/log-sdk/pkg/encoder"
	"time"
)

// mockWriter 模拟一个 /dev/null 或无操作的写入器
type mockWriter struct{}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// BenchmarkGoStandardLog 测试 Go 标准库 log 的性能
func BenchmarkGoStandardLog(b *testing.B) {
	stdLogger := log.New(&mockWriter{}, "", log.LstdFlags)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stdLogger.Printf("Message %d %s %f", i, "test", 3.14)
	}
}

// BenchmarkZapLog 测试 Zap 库的性能
func BenchmarkZapLog(b *testing.B) {
	encoderConfig := zap.NewProductionEncoderConfig()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&mockWriter{}),
		zap.InfoLevel,
	)
	zapLogger := zap.New(core)
	defer zapLogger.Sync()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zapLogger.Info("Message",
			zap.Int("i", i),
			zap.String("s", "test"),
			zap.Float64("f", 3.14),
		)
	}
}

// BenchmarkZerolog 测试 Zerolog 库的性能
func BenchmarkZerolog(b *testing.B) {
	zeroLogger := zerolog.New(&mockWriter{}).With().Timestamp().Logger()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zeroLogger.Info().
			Int("i", i).
			Str("s", "test").
			Float64("f", 3.14).
			Msg("Message")
	}
}

// BenchmarkLogosChain_FullFeatures 测试带有 Caller、Hook、限流和动态规则逻辑的 Logos SDK
func BenchmarkLogosChain_FullFeatures(b *testing.B) {
	// 在测试中初始化携带了假规则和开启 Caller 功能的 SDK
	l := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: false,
		RateLimit:         100000000, // 足够大的限流，不拦截但产生计算开销
		// 模拟 EtcdEndpoints 会导致建立连接失败，所以我们手动构建一个规则
	})
	
	// 添加一个虚拟 Hook 来强制触发 Hook 遍历
	l.AddHook(LevelHook(LevelDebug))
	
	// 在原有的 Logger 结构上强制开启 Caller，我们可以通过添加一个特殊的 WithCaller 方法或者直接修改实现
	// 因为此处是测试包外，我们这里只能反映目前 Logger 的开销
	// 目前 Logger.logEntry 内部包含了限流检查（guard.Allow()）和 Hook 遍历开销

	defer l.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("Message").
			Int("i", i).
			Str("s", "test").
			Float64("f", 3.14).
			Send()
	}
}

// BenchmarkLogosChain_Pure2 测试不含限流、不含Hook、没有规则引擎的 Logos SDK 纯净版本
func BenchmarkLogosChain_Pure2(b *testing.B) {
	// RateLimit 设为 -1 代表在我们的测试或真实场景中完全跳过限流
	// 为了演示极速纯净版，我们需要关闭或绕过限流判断（可以通过将 Allow 直接放行来实现，或者通过配置）
	l := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: false,
	})
	
	// 为了模拟纯净版，我们移除了 Hook 和 Rule，这是默认行为
	
	// 手动将 guard 置为一个始终放行的虚拟 guard（或者是默认非常宽容的guard）
	// 由于 Go 测试没法轻易 mock 私有字段，这里通过配置一个极大限流率来近似，或者我们提供一个配置参数关闭它
	// 但这依然会有互斥锁或原子操作开销。要完全展示纯净版，可认为这就是基础无其他配置的耗时
	
	defer l.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("Message").
			Int("i", i).
			Str("s", "test").
			Float64("f", 3.14).
			Send()
	}
}

func BenchmarkLogosChain_Pure(b *testing.B) {
	// 初始化一个没有规则引擎和限流守卫的 Logger (仅保留 JSON 序列化和 RingBuffer 队列)
	producer := async.NewProducer([]string{"mock"}, 100, 100 * time.Millisecond)
	l := &loggerImpl{
		config: Config{
			ServiceName:       "benchmark",
			FallbackToConsole: false,
		},
		producer: producer,
		enc:      encoder.DefaultJSONEncoder(),
		guard:    nil, // 无限流守卫
		rule:     nil, // 无规则引擎
		fields:   make([]Field, 0),
		hooks:    make([]Hook, 0),
	}
	defer l.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("Message").
			Int("i", i).
			Str("s", "test").
			Float64("f", 3.14).
			Send()
	}
}
