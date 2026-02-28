// Package filter 提供过滤引擎端到端集成测试
package filter_test

import (
	"context"
	"sync"
	"testing"

	"github.com/log-system/log-processor/pkg/analyzer"
	"github.com/log-system/log-processor/pkg/config"
	"github.com/log-system/log-processor/pkg/filter"
	"github.com/log-system/log-processor/pkg/parser"
	"github.com/log-system/log-processor/pkg/transformer"
)

// TestEndToEnd_LogProcessing 端到端测试：完整的日志处理流程
func TestEndToEnd_LogProcessing(t *testing.T) {
	// 创建解析器
	p := parser.NewExtendedMultiParser()

	// 创建分析器
	a := analyzer.NewTextAnalyzer()

	// 创建转换器
	tf := transformer.NewTransformer()

	// 创建过滤引擎
	engine := filter.NewFilterEngine()

	// 配置过滤器
	err := engine.AddFilter(&config.FilterConfig{
		ID:      "e2e-filter",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "drop-debug",
				Field:   "level",
				Pattern: "^DEBUG$",
				Action:  config.ActionDrop,
			},
			{
				Name:    "mark-errors",
				Field:   "level",
				Pattern: "^ERROR$",
				Action:  config.ActionMark,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	// 测试用例
	tests := []struct {
		name          string
		rawLog        []byte
		wantKeep      bool
		wantMark      bool
		wantParseErr  bool
	}{
		{
			name:     "JSON Info Log",
			rawLog:   []byte(`{"level":"INFO","message":"User logged in"}`),
			wantKeep: true,
			wantMark: false,
		},
		{
			name:     "JSON Debug Log",
			rawLog:   []byte(`{"level":"DEBUG","message":"Debug info"}`),
			wantKeep: false, // Should be dropped
			wantMark: false,
		},
		{
			name:     "JSON Error Log",
			rawLog:   []byte(`{"level":"ERROR","message":"Database connection failed"}`),
			wantKeep: true,
			wantMark: true, // Should be marked
		},
		{
			name:     "Nginx Access Log",
			rawLog:   []byte(`127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api HTTP/1.1" 200 1234 "-" "curl"`),
			wantKeep: true,
			wantMark: false,
		},
		{
			name:     "Syslog",
			rawLog:   []byte(`<34>Feb 28 12:00:00 myhost myservice[1234]: Test message`),
			wantKeep: true,
			wantMark: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 解析
			parsed, err := p.Parse(tt.rawLog)
			if (err != nil) != tt.wantParseErr {
				t.Fatalf("Parse error = %v, wantParseErr = %v", err, tt.wantParseErr)
			}
			if err != nil {
				return // Skip rest of test if parse error expected
			}

			// 2. 分析
			analysis, err := a.Analyze(parsed.Message)
			if err != nil {
				t.Errorf("Analyze error: %v", err)
			}

			// 3. 转换
			transformed, err := tf.Transform(parsed, analysis)
			if err != nil {
				t.Errorf("Transform error: %v", err)
			}

			// 4. 过滤
			filterEntry := &filter.ParsedLog{
				Timestamp: transformed.Timestamp,
				Level:     transformed.Level,
				Message:   transformed.Message,
				Service:   transformed.Service,
				TraceID:   transformed.TraceID,
				SpanID:    transformed.SpanID,
				Fields:    transformed.Fields,
				Raw:       transformed.Raw,
			}

			result := engine.ApplyFilters(filterEntry)

			if result.ShouldKeep != tt.wantKeep {
				t.Errorf("ShouldKeep = %v, want %v", result.ShouldKeep, tt.wantKeep)
			}

			if tt.wantMark {
				if _, ok := result.Metadata["marked_by"]; !ok {
					t.Error("Expected log to be marked")
				}
			}
		})
	}
}

// TestEndToEnd_ConcurrentProcessing 端到端测试：并发处理
func TestEndToEnd_ConcurrentProcessing(t *testing.T) {
	engine := filter.NewFilterEngine()

	err := engine.AddFilter(&config.FilterConfig{
		ID:      "concurrent-filter",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "drop-spam",
				Field:   "message",
				Pattern: ".*spam.*",
				Action:  config.ActionDrop,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	logsPerGoroutine := 100

	keptCount := 0
	droppedCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				entry := &filter.ParsedLog{
					Message: "Normal log message",
					Level:   "INFO",
				}

				result := engine.ApplyFilters(entry)

				mu.Lock()
				if result.ShouldKeep {
					keptCount++
				} else {
					droppedCount++
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	total := keptCount + droppedCount
	expectedTotal := numGoroutines * logsPerGoroutine

	if total != expectedTotal {
		t.Errorf("Expected %d total logs, got %d", expectedTotal, total)
	}

	if keptCount != expectedTotal {
		t.Errorf("Expected all logs to be kept, got %d kept", keptCount)
	}
}

// TestEndToEnd_DynamicFilterReload 端到端测试：动态过滤规则重载
func TestEndToEnd_DynamicFilterReload(t *testing.T) {
	engine := filter.NewFilterEngine()

	// 初始配置
	err := engine.AddFilter(&config.FilterConfig{
		ID:      "dynamic-filter",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "drop-v1",
				Field:   "message",
				Pattern: ".*v1.*",
				Action:  config.ActionDrop,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add initial filter: %v", err)
	}

	// 测试初始配置
	entry := &filter.ParsedLog{
		Message: "This is v1 message",
		Level:   "INFO",
	}

	result := engine.ApplyFilters(entry)
	if result.ShouldKeep {
		t.Error("Expected v1 message to be dropped")
	}

	// 移除过滤器
	err = engine.RemoveFilter("dynamic-filter")
	if err != nil {
		t.Fatalf("Failed to remove filter: %v", err)
	}

	// 添加新配置
	err = engine.AddFilter(&config.FilterConfig{
		ID:      "dynamic-filter",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "drop-v2",
				Field:   "message",
				Pattern: ".*v2.*",
				Action:  config.ActionDrop,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add new filter: %v", err)
	}

	// 测试新配置 - v1 应该被保留
	result = engine.ApplyFilters(entry)
	if !result.ShouldKeep {
		t.Error("Expected v1 message to be kept after filter update")
	}

	// 测试新配置 - v2 应该被丢弃
	entry.Message = "This is v2 message"
	result = engine.ApplyFilters(entry)
	if result.ShouldKeep {
		t.Error("Expected v2 message to be dropped")
	}
}

// TestEndToEnd_ComplexFilterChain 端到端测试：复杂过滤链
func TestEndToEnd_ComplexFilterChain(t *testing.T) {
	engine := filter.NewFilterEngine()

	// 添加多个过滤器
	filters := []*config.FilterConfig{
		{
			ID:       "security-filter",
			Enabled:  true,
			Priority: 100,
			Rules: []config.FilterRule{
				{
					Name:    "drop-sensitive",
					Field:   "message",
					Pattern: ".*(password|secret|token).*",
					Action:  config.ActionDrop,
				},
			},
		},
		{
			ID:       "performance-filter",
			Enabled:  true,
			Priority: 50,
			Rules: []config.FilterRule{
				{
					Name:    "mark-slow",
					Field:   "message",
					Pattern: ".*(slow|timeout|latency).*",
					Action:  config.ActionMark,
				},
			},
		},
		{
			ID:       "error-filter",
			Enabled:  true,
			Priority: 10,
			Rules: []config.FilterRule{
				{
					Name:    "drop-debug-errors",
					Field:   "level",
					Pattern: "DEBUG",
					Action:  config.ActionDrop,
				},
			},
		},
	}

	for _, f := range filters {
		err := engine.AddFilter(f)
		if err != nil {
			t.Fatalf("Failed to add filter %s: %v", f.ID, err)
		}
	}

	tests := []struct {
		name     string
		entry    *filter.ParsedLog
		wantKeep bool
		wantMark bool
	}{
		{
			name: "Normal log",
			entry: &filter.ParsedLog{
				Message: "Request processed successfully",
				Level:   "INFO",
			},
			wantKeep: true,
			wantMark: false,
		},
		{
			name: "Sensitive log (should be dropped)",
			entry: &filter.ParsedLog{
				Message: "User password reset failed",
				Level:   "ERROR",
			},
			wantKeep: false,
			wantMark: false,
		},
		{
			name: "Slow log (should be marked)",
			entry: &filter.ParsedLog{
				Message: "Request took 5s - very slow",
				Level:   "WARN",
			},
			wantKeep: true,
			wantMark: true,
		},
		{
			name: "Debug log (should be dropped)",
			entry: &filter.ParsedLog{
				Message: "Debug information",
				Level:   "DEBUG",
			},
			wantKeep: false,
			wantMark: false,
		},
		{
			name: "Timeout log (should be marked)",
			entry: &filter.ParsedLog{
				Message: "Connection timeout occurred",
				Level:   "ERROR",
			},
			wantKeep: true,
			wantMark: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ApplyFilters(tt.entry)

			if result.ShouldKeep != tt.wantKeep {
				t.Errorf("ShouldKeep = %v, want %v", result.ShouldKeep, tt.wantKeep)
			}

			if tt.wantMark {
				if _, ok := result.Metadata["marked_by"]; !ok {
					t.Error("Expected log to be marked")
				}
			}
		})
	}
}

// TestEndToEnd_ServiceBasedFiltering 端到端测试：基于服务的过滤
func TestEndToEnd_ServiceBasedFiltering(t *testing.T) {
	engine := filter.NewFilterEngine()

	err := engine.AddFilter(&config.FilterConfig{
		ID:      "service-filter",
		Enabled: true,
		Service: "payment-service",
		Rules: []config.FilterRule{
			{
				Name:    "drop-payment-debug",
				Field:   "level",
				Pattern: "DEBUG",
				Action:  config.ActionDrop,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	tests := []struct {
		name     string
		entry    *filter.ParsedLog
		wantKeep bool
	}{
		{
			name: "Payment service debug (should be dropped)",
			entry: &filter.ParsedLog{
				Service: "payment-service",
				Level:   "DEBUG",
				Message: "Debug payment info",
			},
			wantKeep: false,
		},
		{
			name: "Payment service info (should be kept)",
			entry: &filter.ParsedLog{
				Service: "payment-service",
				Level:   "INFO",
				Message: "Payment processed",
			},
			wantKeep: true,
		},
		{
			name: "Other service debug (should be kept - not matching service)",
			entry: &filter.ParsedLog{
				Service: "other-service",
				Level:   "DEBUG",
				Message: "Debug other info",
			},
			wantKeep: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ApplyFilters(tt.entry)

			if result.ShouldKeep != tt.wantKeep {
				t.Errorf("ShouldKeep = %v, want %v", result.ShouldKeep, tt.wantKeep)
			}
		})
	}
}

// TestEndToEnd_ContextCancellation 端到端测试：上下文取消
func TestEndToEnd_ContextCancellation(t *testing.T) {
	engine := filter.NewFilterEngine()

	err := engine.AddFilter(&config.FilterConfig{
		ID:      "context-filter",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "test-rule",
				Field:   "message",
				Pattern: ".*",
				Action:  config.ActionAllow,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	entry := &filter.ParsedLog{
		Message: "Test message",
		Level:   "INFO",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// 过滤操作应该不受上下文影响（当前实现不支持上下文）
	_ = ctx
	result := engine.ApplyFilters(entry)

	if !result.ShouldKeep {
		t.Error("Expected log to be kept")
	}
}

// TestEndToEnd_FieldBasedFiltering 端到端测试：基于字段的过滤
func TestEndToEnd_FieldBasedFiltering(t *testing.T) {
	engine := filter.NewFilterEngine()

	err := engine.AddFilter(&config.FilterConfig{
		ID:      "field-filter",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "drop-specific-trace",
				Field:   "trace_id",
				Pattern: "^bad-trace.*",
				Action:  config.ActionDrop,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	tests := []struct {
		name     string
		entry    *filter.ParsedLog
		wantKeep bool
	}{
		{
			name: "Bad trace ID (should be dropped)",
			entry: &filter.ParsedLog{
				Message: "Request failed",
				TraceID: "bad-trace-123",
			},
			wantKeep: false,
		},
		{
			name: "Good trace ID (should be kept)",
			entry: &filter.ParsedLog{
				Message: "Request succeeded",
				TraceID: "good-trace-456",
			},
			wantKeep: true,
		},
		{
			name: "No trace ID (should be kept)",
			entry: &filter.ParsedLog{
				Message: "Request without trace",
			},
			wantKeep: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ApplyFilters(tt.entry)

			if result.ShouldKeep != tt.wantKeep {
				t.Errorf("ShouldKeep = %v, want %v", result.ShouldKeep, tt.wantKeep)
			}
		})
	}
}

// TestEndToEnd_MultipleFiltersSameField 端到端测试：同一字段的多个过滤器
func TestEndToEnd_MultipleFiltersSameField(t *testing.T) {
	engine := filter.NewFilterEngine()

	// 添加多个作用于同一字段的过滤器
	err := engine.AddFilter(&config.FilterConfig{
		ID:      "filter-1",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "drop-contains-a",
				Field:   "message",
				Pattern: ".*a.*",
				Action:  config.ActionDrop,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add filter 1: %v", err)
	}

	err = engine.AddFilter(&config.FilterConfig{
		ID:      "filter-2",
		Enabled: true,
		Rules: []config.FilterRule{
			{
				Name:    "drop-contains-b",
				Field:   "message",
				Pattern: ".*b.*",
				Action:  config.ActionDrop,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add filter 2: %v", err)
	}

	tests := []struct {
		name     string
		message  string
		wantKeep bool
	}{
		{"Contains a only", "apple", false},
		{"Contains b only", "banana", false},
		{"Contains both", "abc", false},
		{"Contains neither", "xyz", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &filter.ParsedLog{
				Message: tt.message,
				Level:   "INFO",
			}

			result := engine.ApplyFilters(entry)

			if result.ShouldKeep != tt.wantKeep {
				t.Errorf("ShouldKeep = %v, want %v", result.ShouldKeep, tt.wantKeep)
			}
		})
	}
}
