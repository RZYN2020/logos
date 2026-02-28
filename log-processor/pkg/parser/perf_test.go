// Package parser 提供解析器性能基准测试
package parser_test

import (
	"testing"
	"time"

	"github.com/log-system/log-processor/pkg/parser"
)

// BenchmarkJSONParser 测试 JSON 解析器性能
func BenchmarkJSONParser(b *testing.B) {
	p := parser.NewJSONParser()
	logData := []byte(`{
		"timestamp": "2026-02-28T12:00:00Z",
		"level": "INFO",
		"message": "User login successful",
		"service": "auth-service",
		"trace_id": "abc123",
		"span_id": "span456",
		"user_id": "user789"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultiParser 测试多格式解析器性能
func BenchmarkMultiParser(b *testing.B) {
	p := parser.NewMultiParser()
	logData := []byte(`{"timestamp": "2026-02-28T12:00:00Z", "level": "INFO", "message": "test"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkExtendedMultiParser 测试扩展多格式解析器性能
func BenchmarkExtendedMultiParser(b *testing.B) {
	p := parser.NewExtendedMultiParser()
	logData := []byte(`{"timestamp": "2026-02-28T12:00:00Z", "level": "INFO", "message": "test"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkKeyValueParser 测试 KeyValue 解析器性能
func BenchmarkKeyValueParser(b *testing.B) {
	p := parser.NewKeyValueParser()
	logData := []byte(`timestamp=2026-02-28T12:00:00Z level=INFO message="test message" service=api`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSyslogParser 测试 Syslog 解析器性能
func BenchmarkSyslogParser(b *testing.B) {
	p := parser.NewSyslogParser()
	logData := []byte(`<34>Feb 28 12:00:00 myhost myservice[1234]: Test message`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkApacheParser 测试 Apache 解析器性能
func BenchmarkApacheParser(b *testing.B) {
	p := parser.NewApacheParser()
	logData := []byte(`127.0.0.1 - frank [28/Feb/2026:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNginxParser 测试 Nginx 解析器性能
func BenchmarkNginxParser(b *testing.B) {
	p := parser.NewNginxParser()
	logData := []byte(`127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnstructuredParser 测试非结构化解析器性能
func BenchmarkUnstructuredParser(b *testing.B) {
	p := parser.NewUnstructuredParser()
	logData := []byte(`2026-02-28 12:00:00 ERROR Database connection failed: timeout after 30s for user admin from 192.168.1.1`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParserScheduler 测试解析器调度器性能
func BenchmarkParserScheduler(b *testing.B) {
	s := parser.NewParserScheduler()
	logData := []byte(`{"timestamp": "2026-02-28T12:00:00Z", "level": "INFO", "message": "test"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLatency 测试解析延迟
func BenchmarkParseLatency(b *testing.B) {
	p := parser.NewExtendedMultiParser()
	logData := []byte(`{"timestamp": "2026-02-28T12:00:00Z", "level": "INFO", "message": "test"}`)

	latencies := make([]time.Duration, 0, b.N)

	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
		latencies = append(latencies, time.Since(start))
	}

	// 计算平均延迟
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	b.Logf("Average parse latency: %v", total/time.Duration(b.N))
}

// BenchmarkMemoryUsage 测试内存使用
func BenchmarkMemoryUsage(b *testing.B) {
	p := parser.NewExtendedMultiParser()
	logData := []byte(`{"timestamp": "2026-02-28T12:00:00Z", "level": "INFO", "message": "test", "data": "some extra data to increase size"}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := p.Parse(logData)
		if err != nil {
			b.Fatal(err)
		}
	}
}
