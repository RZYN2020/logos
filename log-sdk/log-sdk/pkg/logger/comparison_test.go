package logger

import (
	"log"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/rs/zerolog"
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

// BenchmarkLogosChain 测试当前 Logos SDK 链式调用的性能
func BenchmarkLogosChain(b *testing.B) {
	l := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: false,
	})
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
