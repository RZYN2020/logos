package logger

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkLogger_Printf benchmarks the printf-style API
func BenchmarkLogger_Printf(b *testing.B) {
	log := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: true,
	})
	defer log.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Printf("Benchmark message %d with %s and %f", i, "string", 3.14)
			i++
		}
	})
}

// BenchmarkLogger_TraditionalStyle benchmarks traditional style with fields
func BenchmarkLogger_TraditionalStyle(b *testing.B) {
	log := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: true,
	})
	defer log.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Info("Benchmark message",
				F("iteration", i),
				F("user_id", "user-123"),
				F("action", "login"),
			)
			i++
		}
	})
}

// BenchmarkLogger_ChainStyle benchmarks chain-style API
func BenchmarkLogger_ChainStyle(b *testing.B) {
	log := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: true,
	})
	defer log.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			log.Info("Benchmark message").
				Int("iteration", i).
				Str("user_id", "user-123").
				Str("action", "login").
				Send()
			i++
		}
	})
}

// BenchmarkLogger_With benchmarks logger with pre-defined fields
func BenchmarkLogger_With(b *testing.B) {
	log := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: true,
	})
	defer log.Close()

	requestLog := log.With(
		F("request_id", "req-abc-123"),
		F("trace_id", "trace-xyz-789"),
		F("service", "payment-service"),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			requestLog.Info("Processing request",
				F("user_id", "user-456"),
				F("amount", 100),
			)
			i++
		}
	})
}

// BenchmarkLogger_LevelFilter benchmarks level filtering
func BenchmarkLogger_LevelFilter(b *testing.B) {
	log := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: true,
	})
	defer log.Close()

	// Add level filter - only WARN and above
	filteredLog := log.AddHook(LevelHook(LevelWarn))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// These should be filtered out
			filteredLog.Debug("Debug message").Send()
			filteredLog.Info("Info message").Send()
			// This should pass through
			filteredLog.Warn("Warning message").Send()
		}
	})
}

// BenchmarkLogEntryPool benchmarks object pool performance
func BenchmarkLogEntryPool_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			entry := acquireLogEntry()
			entry.Level = "INFO"
			entry.Message = "benchmark"
			entry.Fields["key"] = "value"
			releaseLogEntry(entry)
		}
	})
}

// BenchmarkLogger_Memory benchmarks memory allocations
func BenchmarkLogger_Memory(b *testing.B) {
	log := New(Config{
		ServiceName:       "benchmark",
		FallbackToConsole: true,
	})
	defer log.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		log.Info("Memory benchmark").
			Str("iteration", fmt.Sprintf("%d", i)).
			Int("count", i).
			Float64("pi", 3.14159).
			Bool("enabled", true).
			Send()
	}
}

// TestConcurrentLogging tests concurrent logging safety
func TestConcurrentLogging(t *testing.T) {
	log := New(Config{
		ServiceName:       "concurrent-test",
		FallbackToConsole: true,
	})
	defer log.Close()

	const numGoroutines = 100
	const numLogsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numLogsPerGoroutine; j++ {
				log.Info("Concurrent log").
					Int("goroutine", id).
					Int("iteration", j).
					Send()
			}
		}(i)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for concurrent logs")
	}
}

// TestConcurrentLogging_WithHooks tests concurrent logging with hooks
func TestConcurrentLogging_WithHooks(t *testing.T) {
	log := New(Config{
		ServiceName:       "concurrent-hook-test",
		FallbackToConsole: true,
	})
	defer log.Close()

	// Add hooks
	log = log.AddHook(LevelHook(LevelInfo))
	log = log.AddHook(LineHook(1, 10000))

	const numGoroutines = 50
	const numLogsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numLogsPerGoroutine; j++ {
				log.Warn("Concurrent with hooks").
					Int("goroutine", id).
					Int("iteration", j).
					Send()
			}
		}(i)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for concurrent logs with hooks")
	}
}

// TestConcurrentLogging_WithChaining tests concurrent logger chaining
func TestConcurrentLogging_WithChaining(t *testing.T) {
	log := New(Config{
		ServiceName:       "concurrent-chain-test",
		FallbackToConsole: true,
	})
	defer log.Close()

	const numGoroutines = 50
	const numLogsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			// Each goroutine creates its own sub-logger
			requestLog := log.With(
				F("goroutine_id", id),
				F("request_id", fmt.Sprintf("req-%d", id)),
			)

			for j := 0; j < numLogsPerGoroutine; j++ {
				requestLog.Info("Chained log").
					Int("iteration", j).
					Send()
			}
		}(i)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for concurrent chained logs")
	}
}

// BenchmarkComparison compares different logging approaches
func BenchmarkComparison(b *testing.B) {
	b.Run("Printf", func(b *testing.B) {
		log := New(Config{
			ServiceName:       "benchmark",
			FallbackToConsole: true,
		})
		defer log.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			log.Printf("Message %d %s %f", i, "test", 3.14)
		}
	})

	b.Run("Traditional", func(b *testing.B) {
		log := New(Config{
			ServiceName:       "benchmark",
			FallbackToConsole: true,
		})
		defer log.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			log.Info("Message", F("i", i), F("s", "test"), F("f", 3.14))
		}
	})

	b.Run("Chain", func(b *testing.B) {
		log := New(Config{
			ServiceName:       "benchmark",
			FallbackToConsole: true,
		})
		defer log.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			log.Info("Message").Int("i", i).Str("s", "test").Float64("f", 3.14).Send()
		}
	})
}

// BenchmarkHookOverhead benchmarks hook filtering overhead
func BenchmarkHookOverhead(b *testing.B) {
	b.Run("NoHooks", func(b *testing.B) {
		log := New(Config{
			ServiceName:       "benchmark",
			FallbackToConsole: true,
		})
		defer log.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			log.Info("Message").Int("i", i).Send()
		}
	})

	b.Run("OneHook", func(b *testing.B) {
		log := New(Config{
			ServiceName:       "benchmark",
			FallbackToConsole: true,
		})
		defer log.Close()

		log = log.AddHook(LevelHook(LevelDebug))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			log.Info("Message").Int("i", i).Send()
		}
	})

	b.Run("ThreeHooks", func(b *testing.B) {
		log := New(Config{
			ServiceName:       "benchmark",
			FallbackToConsole: true,
		})
		defer log.Close()

		log = log.AddHook(LevelHook(LevelDebug))
		log = log.AddHook(LineHook(1, 100000))
		log = log.AddHook(Func(func(entry LogEntry) bool { return true }))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			log.Info("Message").Int("i", i).Send()
		}
	})
}
