// Package filter 提供过滤引擎单元测试
package filter_test

import (
	"testing"

	"github.com/log-system/log-processor/pkg/config"
	"github.com/log-system/log-processor/pkg/filter"
)

func TestFilterEngineBasic(t *testing.T) {
	engine := filter.NewFilterEngine()

	// 添加过滤配置
	cfg := &config.FilterConfig{
		ID:       "test-filter",
		Enabled:  true,
		Priority: 10,
		Rules: []config.FilterRule{
			{
				Name:    "drop-error",
				Field:   "message",
				Pattern: "ERROR.*",
				Action:  config.ActionDrop,
			},
		},
	}

	err := engine.AddFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	// 测试过滤
	entry := &filter.ParsedLog{
		Message: "ERROR: something failed",
		Level:   "ERROR",
	}

	result := engine.ApplyFilters(entry)
	if result.ShouldKeep {
		t.Error("Expected log to be dropped")
	}
	if result.Action != config.ActionDrop {
		t.Errorf("Expected action Drop, got %v", result.Action)
	}
}

func TestFilterEngineAllow(t *testing.T) {
	engine := filter.NewFilterEngine()

	cfg := &config.FilterConfig{
		ID:       "test-filter",
		Enabled:  true,
		Priority: 10,
		Rules: []config.FilterRule{
			{
				Name:    "allow-info",
				Field:   "level",
				Pattern: "INFO.*",
				Action:  config.ActionAllow,
			},
		},
	}

	err := engine.AddFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	entry := &filter.ParsedLog{
		Message: "All good",
		Level:   "INFO",
	}

	result := engine.ApplyFilters(entry)
	if !result.ShouldKeep {
		t.Error("Expected log to be kept")
	}
}

func TestFilterEngineMark(t *testing.T) {
	engine := filter.NewFilterEngine()

	cfg := &config.FilterConfig{
		ID:       "test-filter",
		Enabled:  true,
		Priority: 10,
		Rules: []config.FilterRule{
			{
				Name:    "mark-warning",
				Field:   "level",
				Pattern: "WARN.*",
				Action:  config.ActionMark,
			},
		},
	}

	err := engine.AddFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	entry := &filter.ParsedLog{
		Message: "Something might be wrong",
		Level:   "WARNING",
	}

	result := engine.ApplyFilters(entry)
	if !result.ShouldKeep {
		t.Error("Expected log to be kept")
	}
	if result.Action != config.ActionMark {
		t.Errorf("Expected action Mark, got %v", result.Action)
	}
	if _, ok := result.Metadata["marked_by"]; !ok {
		t.Error("Expected metadata to contain marked_by")
	}
}

func TestFilterEngineDisabled(t *testing.T) {
	engine := filter.NewFilterEngine()

	cfg := &config.FilterConfig{
		ID:       "test-filter",
		Enabled:  false,
		Priority: 10,
		Rules: []config.FilterRule{
			{
				Name:    "drop-all",
				Field:   "message",
				Pattern: ".*",
				Action:  config.ActionDrop,
			},
		},
	}

	err := engine.AddFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	entry := &filter.ParsedLog{
		Message: "Any message",
	}

	result := engine.ApplyFilters(entry)
	if !result.ShouldKeep {
		t.Error("Expected log to be kept when filter is disabled")
	}
}

func TestFilterEngineRemove(t *testing.T) {
	engine := filter.NewFilterEngine()

	cfg := &config.FilterConfig{
		ID:       "test-filter",
		Enabled:  true,
		Priority: 10,
		Rules: []config.FilterRule{
			{
				Name:    "drop-error",
				Field:   "message",
				Pattern: "ERROR.*",
				Action:  config.ActionDrop,
			},
		},
	}

	err := engine.AddFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	err = engine.RemoveFilter("test-filter")
	if err != nil {
		t.Fatalf("Failed to remove filter: %v", err)
	}

	entry := &filter.ParsedLog{
		Message: "ERROR: something failed",
	}

	result := engine.ApplyFilters(entry)
	if !result.ShouldKeep {
		t.Error("Expected log to be kept after filter removal")
	}
}

func TestFilterMetadata(t *testing.T) {
	metadata := filter.NewFilterMetadata()

	entry := &filter.ParsedLog{
		Service: "test-service",
		Message: "test message",
	}

	result := filter.FilterResult{
		ShouldKeep:  true,
		Action:      config.ActionAllow,
		MatchedRule: "test-rule",
	}

	metadata.Record(entry, result, "test-filter")

	stats := metadata.GetStats()
	if stats.TotalProcessed != 1 {
		t.Errorf("Expected TotalProcessed to be 1, got %d", stats.TotalProcessed)
	}
	if stats.TotalKept != 1 {
		t.Errorf("Expected TotalKept to be 1, got %d", stats.TotalKept)
	}
}

func TestFilterEngineServiceFilter(t *testing.T) {
	engine := filter.NewFilterEngine()

	cfg := &config.FilterConfig{
		ID:       "test-filter",
		Enabled:  true,
		Priority: 10,
		Service:  "specific-service",
		Rules: []config.FilterRule{
			{
				Name:    "drop-error",
				Field:   "message",
				Pattern: "ERROR.*",
				Action:  config.ActionDrop,
			},
		},
	}

	err := engine.AddFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to add filter: %v", err)
	}

	// 测试不同服务名的日志
	entry := &filter.ParsedLog{
		Service: "other-service",
		Message: "ERROR: something failed",
	}

	result := engine.ApplyFilters(entry)
	if !result.ShouldKeep {
		t.Error("Expected log to be kept for different service")
	}

	// 测试匹配服务名的日志
	entry.Service = "specific-service"
	result = engine.ApplyFilters(entry)
	if result.ShouldKeep {
		t.Error("Expected log to be dropped for matching service")
	}
}

func TestFilterEnginePriority(t *testing.T) {
	engine := filter.NewFilterEngine()

	// 添加多个不同优先级的过滤器
	cfg1 := &config.FilterConfig{
		ID:       "high-priority",
		Enabled:  true,
		Priority: 100,
		Rules: []config.FilterRule{
			{
				Name:    "drop-critical",
				Field:   "level",
				Pattern: "CRITICAL",
				Action:  config.ActionDrop,
			},
		},
	}

	cfg2 := &config.FilterConfig{
		ID:       "low-priority",
		Enabled:  true,
		Priority: 10,
		Rules: []config.FilterRule{
			{
				Name:    "allow-all",
				Field:   "message",
				Pattern: ".*",
				Action:  config.ActionAllow,
			},
		},
	}

	engine.AddFilter(cfg1)
	engine.AddFilter(cfg2)

	entry := &filter.ParsedLog{
		Level:   "CRITICAL",
		Message: "Critical error",
	}

	result := engine.ApplyFilters(entry)
	if result.ShouldKeep {
		t.Error("Expected log to be dropped by high priority filter")
	}
	if result.MatchedRule != "drop-critical" {
		t.Errorf("Expected matched rule 'drop-critical', got '%s'", result.MatchedRule)
	}
}

func TestFilterMetadataStats(t *testing.T) {
	metadata := filter.NewFilterMetadata()

	// 记录多条日志
	for i := 0; i < 10; i++ {
		entry := &filter.ParsedLog{
			Service: "test-service",
			Message: "test message",
		}

		result := filter.FilterResult{
			ShouldKeep: i%2 == 0,
			Action:     config.ActionAllow,
		}

		metadata.Record(entry, result, "test-filter")
	}

	stats := metadata.GetStats()
	if stats.TotalProcessed != 10 {
		t.Errorf("Expected TotalProcessed to be 10, got %d", stats.TotalProcessed)
	}
	if stats.TotalKept != 5 {
		t.Errorf("Expected TotalKept to be 5, got %d", stats.TotalKept)
	}
	if stats.TotalDropped != 5 {
		t.Errorf("Expected TotalDropped to be 5, got %d", stats.TotalDropped)
	}
}

func TestFilterMetadataClear(t *testing.T) {
	metadata := filter.NewFilterMetadata()

	// 记录一些数据
	entry := &filter.ParsedLog{
		Service: "test-service",
	}
	result := filter.FilterResult{ShouldKeep: true}
	metadata.Record(entry, result, "test-filter")

	// 清空
	metadata.Clear()

	stats := metadata.GetStats()
	if stats.TotalProcessed != 0 {
		t.Errorf("Expected TotalProcessed to be 0 after clear, got %d", stats.TotalProcessed)
	}
}

func TestFilterMetadataExportMetrics(t *testing.T) {
	metadata := filter.NewFilterMetadata()

	entry := &filter.ParsedLog{Service: "test"}
	result := filter.FilterResult{ShouldKeep: true, Action: config.ActionAllow}
	metadata.Record(entry, result, "test")

	metrics := metadata.ExportMetrics()

	if val, ok := metrics["total_processed"].(int64); !ok || val != 1 {
		t.Errorf("Expected total_processed to be 1, got %v", metrics["total_processed"])
	}
	if val, ok := metrics["total_kept"].(int64); !ok || val != 1 {
		t.Errorf("Expected total_kept to be 1, got %v", metrics["total_kept"])
	}
}

func TestCompositeFilter(t *testing.T) {
	composite := filter.NewCompositeFilter()

	cond := filter.CompositeCondition{
		Name:     "test-condition",
		Operator: filter.OpAnd,
		Conditions: []filter.SingleCondition{
			{
				Field:    "level",
				Operator: "eq",
				Value:    "ERROR",
			},
			{
				Field:    "message",
				Operator: "contains",
				Value:    "failed",
			},
		},
		Action: config.ActionDrop,
	}

	err := composite.AddCondition(cond)
	if err != nil {
		t.Fatalf("Failed to add condition: %v", err)
	}

	entry := &filter.ParsedLog{
		Level:   "ERROR",
		Message: "Operation failed",
	}

	result := composite.Evaluate(entry)
	if result.ShouldKeep {
		t.Error("Expected log to be dropped")
	}

	// 测试不匹配的情况
	entry.Message = "Operation succeeded"
	result = composite.Evaluate(entry)
	if !result.ShouldKeep {
		t.Error("Expected log to be kept")
	}
}
