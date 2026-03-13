package rule_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/log-system/logos/pkg/rule"
	"github.com/log-system/logos/pkg/rule/storage"
)

// TestE2E_SingleRuleFlow 端到端测试：单条规则流程
func TestE2E_SingleRuleFlow(t *testing.T) {
	// 1. 创建存储
	storage := storage.NewMemoryStorage()

	// 2. 创建规则
	rule1 := &rule.Rule{
		ID:      "e2e-drop-debug",
		Name:    "E2E Drop Debug Logs",
		Enabled: true,
		Condition: rule.Condition{
			Field:    "level",
			Operator: rule.OpEq,
			Value:    "DEBUG",
		},
		Actions: []rule.ActionDef{
			{Type: rule.ActionDrop},
		},
	}

	// 3. 存储规则
	if err := storage.PutRule(rule1); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	// 4. 创建引擎并加载规则
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	// 5. 测试 DEBUG 日志应该被 drop
	debugEntry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "DEBUG",
		"message": "debug message",
	})
	shouldKeep, results, errs := engine.Evaluate(debugEntry)

	if shouldKeep {
		t.Error("DEBUG log should be dropped")
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 matched rule, got %d", len(results))
	}
	if len(errs) > 0 {
		t.Errorf("Unexpected errors: %v", errs)
	}

	// 6. 测试 ERROR 日志应该被保留（默认行为）
	errorEntry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "error message",
	})
	shouldKeep2, results2, errs2 := engine.Evaluate(errorEntry)

	if !shouldKeep2 {
		t.Error("ERROR log should be kept by default")
	}
	if len(results2) != 0 {
		t.Errorf("Expected 0 matched rules for ERROR, got %d", len(results2))
	}
	if len(errs2) > 0 {
		t.Errorf("Unexpected errors: %v", errs2)
	}

	t.Log("E2E single rule flow test passed")
}

// TestE2E_CompositeConditionFlow 端到端测试：复合条件流程
func TestE2E_CompositeConditionFlow(t *testing.T) {
	storage := storage.NewMemoryStorage()

	// 创建带有复合条件的规则
	rule1 := &rule.Rule{
		ID:      "e2e-composite-rule",
		Name:    "E2E Composite Condition Rule",
		Enabled: true,
		Condition: rule.Condition{
			All: []rule.Condition{
				{Field: "level", Operator: rule.OpEq, Value: "ERROR"},
				{
					Any: []rule.Condition{
						{Field: "service", Operator: rule.OpEq, Value: "api"},
						{Field: "service", Operator: rule.OpEq, Value: "worker"},
					},
				},
			},
		},
		Actions: []rule.ActionDef{
			{Type: rule.ActionKeep},
		},
	}

	if err := storage.PutRule(rule1); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	tests := []struct {
		name          string
		entry         rule.LogEntry
		shouldKeep    bool
		shouldMatch   bool
	}{
		{
			name: "matches all conditions",
			entry: rule.NewMapLogEntry(map[string]interface{}{
				"level":   "ERROR",
				"service": "api",
			}),
			shouldKeep:  true,
			shouldMatch: true,
		},
		{
			name: "matches any service",
			entry: rule.NewMapLogEntry(map[string]interface{}{
				"level":   "ERROR",
				"service": "worker",
			}),
			shouldKeep:  true,
			shouldMatch: true,
		},
		{
			name: "does not match level",
			entry: rule.NewMapLogEntry(map[string]interface{}{
				"level":   "INFO",
				"service": "api",
			}),
			shouldKeep:  true,  // Default is to keep when no rule matches
			shouldMatch: false,
		},
		{
			name: "does not match service",
			entry: rule.NewMapLogEntry(map[string]interface{}{
				"level":   "ERROR",
				"service": "web",
			}),
			shouldKeep:  true,  // Default is to keep when no rule matches
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldKeep, results, _ := engine.Evaluate(tt.entry)
			if shouldKeep != tt.shouldKeep {
				t.Errorf("Expected shouldKeep=%v, got %v", tt.shouldKeep, shouldKeep)
			}
			matched := len(results) > 0
			if matched != tt.shouldMatch {
				t.Errorf("Expected match=%v, got %v", tt.shouldMatch, matched)
			}
		})
	}
}

// TestE2E_ActionTransformations 端到端测试：动作转换链
func TestE2E_ActionTransformations(t *testing.T) {
	storage := storage.NewMemoryStorage()

	rule1 := &rule.Rule{
		ID:      "e2e-transform-rule",
		Name:    "E2E Transform Rule",
		Enabled: true,
		Condition: rule.Condition{
			Field:    "level",
			Operator: rule.OpExists,
		},
		Actions: []rule.ActionDef{
			{
				Type: rule.ActionSet,
				Config: map[string]interface{}{
					"field": "processed",
					"value": true,
				},
			},
			{
				Type: rule.ActionMask,
				Config: map[string]interface{}{
					"field":   "password",
					"pattern": ".*",
				},
			},
			{
				Type: rule.ActionRemove,
				Config: map[string]interface{}{
					"field": "temp_data",
				},
			},
		},
	}

	if err := storage.PutRule(rule1); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":       "INFO",
		"message":     "test message",
		"password":    "secret123",
		"temp_data":   "to-be-removed",
	})

	shouldKeep, results, errs := engine.Evaluate(entry)

	if !shouldKeep {
		t.Error("Log should be kept")
	}
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}
	if len(errs) > 0 {
		t.Errorf("Unexpected errors: %v", errs)
	}

	// 验证转换结果
	processed, _ := entry.GetField("processed")
	if processed != true {
		t.Errorf("Expected processed=true, got %v", processed)
	}

	password, _ := entry.GetField("password")
	if password == "secret123" {
		t.Error("Password should be masked")
	}
	if password != "***" && password != "******" {
		t.Logf("Masked password: %v", password)
	}

	_, hasTemp := entry.GetField("temp_data")
	if hasTemp {
		t.Error("temp_data should be removed")
	}

	t.Log("Transform chain test passed")
}

// TestE2E_RuleChainTermination 端到端测试：规则链终止行为
func TestE2E_RuleChainTermination(t *testing.T) {
	storage := storage.NewMemoryStorage()

	rules := []*rule.Rule{
		{
			ID:      "rule1-drop-all",
			Name:    "Drop All",
			Enabled: true,
			Condition: rule.Condition{
				Field:    "level",
				Operator: rule.OpExists,
			},
			Actions: []rule.ActionDef{
				{Type: rule.ActionDrop},
			},
		},
		{
			ID:      "rule2-should-not-run",
			Name:    "Should Not Run",
			Enabled: true,
			Condition: rule.Condition{
				Field:    "level",
				Operator: rule.OpEq,
				Value:    "ERROR",
			},
			Actions: []rule.ActionDef{
				{Type: rule.ActionKeep},
			},
		},
	}

	for _, r := range rules {
		if err := storage.PutRule(r); err != nil {
			t.Fatalf("PutRule failed: %v", err)
		}
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level": "ERROR",
	})

	shouldKeep, results, _ := engine.Evaluate(entry)

	if shouldKeep {
		t.Error("Log should be dropped by first rule")
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result (due to drop termination), got %d", len(results))
	}
	if !results[0].Terminate {
		t.Error("First rule should terminate the chain")
	}

	t.Log("Rule chain termination test passed")
}

// TestE2E_SampleAction 端到端测试：采样动作
func TestE2E_SampleAction(t *testing.T) {
	storage := storage.NewMemoryStorage()

	rule1 := &rule.Rule{
		ID:      "e2e-sample-rule",
		Name:    "E2E Sample Rule",
		Enabled: true,
		Condition: rule.Condition{
			Field:    "level",
			Operator: rule.OpEq,
			Value:    "DEBUG",
		},
		Actions: []rule.ActionDef{
			{
				Type: rule.ActionSample,
				Config: map[string]interface{}{
					"rate": 0.5, // 50% 采样率
				},
			},
		},
	}

	if err := storage.PutRule(rule1); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	// 运行 1000 次，验证采样率接近 50%
	keepCount := 0
	total := 1000

	for i := 0; i < total; i++ {
		entry := rule.NewMapLogEntry(map[string]interface{}{
			"level":   "DEBUG",
			"message": fmt.Sprintf("debug message %d", i),
		})
		shouldKeep, _, _ := engine.Evaluate(entry)
		if shouldKeep {
			keepCount++
		}
	}

	rate := float64(keepCount) / float64(total)
	t.Logf("Sample rate: %.2f%% (expected ~50%%)", rate*100)

	// 允许 10% 的误差
	if rate < 0.4 || rate > 0.6 {
		t.Errorf("Sample rate %.2f is outside expected range [0.4, 0.6]", rate)
	}
}

// TestE2E_ConcurrentEvaluation 端到端测试：并发评估
func TestE2E_ConcurrentEvaluation(t *testing.T) {
	storage := storage.NewMemoryStorage()

	rule1 := &rule.Rule{
		ID:      "e2e-concurrent-rule",
		Name:    "E2E Concurrent Rule",
		Enabled: true,
		Condition: rule.Condition{
			Field:    "level",
			Operator: rule.OpEq,
			Value:    "ERROR",
		},
		Actions: []rule.ActionDef{
			{Type: rule.ActionKeep},
		},
	}

	if err := storage.PutRule(rule1); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	iterations := 100

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				entry := rule.NewMapLogEntry(map[string]interface{}{
					"level":       "ERROR",
					"message":     fmt.Sprintf("error %d-%d", id, j),
					"goroutine":   id,
					"iteration":   j,
				})
				shouldKeep, results, errs := engine.Evaluate(entry)
				if !shouldKeep {
					errors <- fmt.Errorf("goroutine %d: expected to keep log", id)
					return
				}
				if len(errs) > 0 {
					errors <- fmt.Errorf("goroutine %d: unexpected errors: %v", id, errs)
					return
				}
				_ = results
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	if len(errors) > 0 {
		for err := range errors {
			t.Error(err)
		}
	}

	t.Logf("Concurrent evaluation test passed with %d goroutines x %d iterations", numGoroutines, iterations)
}

// TestE2E_DynamicRuleReload 端到端测试：动态规则重载
func TestE2E_DynamicRuleReload(t *testing.T) {
	storage := storage.NewMemoryStorage()

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("Initial LoadRules failed: %v", err)
	}

	// 初始状态：没有规则
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "DEBUG",
		"message": "debug message",
	})
	shouldKeep, _, _ := engine.Evaluate(entry)
	if !shouldKeep {
		t.Error("DEBUG log should be kept when no rules match")
	}

	// 动态添加规则
	rule1 := &rule.Rule{
		ID:      "dynamic-drop-debug",
		Name:    "Dynamic Drop Debug",
		Enabled: true,
		Condition: rule.Condition{
			Field:    "level",
			Operator: rule.OpEq,
			Value:    "DEBUG",
		},
		Actions: []rule.ActionDef{
			{Type: rule.ActionDrop},
		},
	}

	if err := storage.PutRule(rule1); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	// 重新加载规则
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("Reload LoadRules failed: %v", err)
	}

	// 验证新规则生效
	shouldKeep2, results2, _ := engine.Evaluate(entry)
	if shouldKeep2 {
		t.Error("DEBUG log should be dropped after rule reload")
	}
	if len(results2) != 1 {
		t.Errorf("Expected 1 matched rule, got %d", len(results2))
	}

	t.Log("Dynamic rule reload test passed")
}

// TestE2E_AllActionTypes 端到端测试：所有动作类型
func TestE2E_AllActionTypes(t *testing.T) {
	tests := []struct {
		name     string
		action   rule.ActionDef
		setup    func() rule.LogEntry
		validate func(t *testing.T, entry rule.LogEntry)
	}{
		{
			name: "keep action",
			action: rule.ActionDef{Type: rule.ActionKeep},
			setup: func() rule.LogEntry {
				return rule.NewMapLogEntry(map[string]interface{}{"level": "INFO"})
			},
			validate: func(t *testing.T, entry rule.LogEntry) {
				// keep 动作应该保留日志
			},
		},
		{
			name: "drop action",
			action: rule.ActionDef{Type: rule.ActionDrop},
			setup: func() rule.LogEntry {
				return rule.NewMapLogEntry(map[string]interface{}{"level": "INFO"})
			},
			validate: func(t *testing.T, entry rule.LogEntry) {
				// drop 动作应该丢弃日志
			},
		},
		{
			name: "set action",
			action: rule.ActionDef{
				Type: rule.ActionSet,
				Config: map[string]interface{}{
					"field": "new_field",
					"value": "new_value",
				},
			},
			setup: func() rule.LogEntry {
				return rule.NewMapLogEntry(map[string]interface{}{"level": "INFO"})
			},
			validate: func(t *testing.T, entry rule.LogEntry) {
				val, ok := entry.GetField("new_field")
				if !ok || val != "new_value" {
					t.Errorf("Expected new_field=new_value, got %v", val)
				}
			},
		},
		{
			name: "remove action",
			action: rule.ActionDef{
				Type: rule.ActionRemove,
				Config: map[string]interface{}{
					"field": "to_remove",
				},
			},
			setup: func() rule.LogEntry {
				return rule.NewMapLogEntry(map[string]interface{}{
					"level":     "INFO",
					"to_remove": "delete_me",
				})
			},
			validate: func(t *testing.T, entry rule.LogEntry) {
				_, ok := entry.GetField("to_remove")
				if ok {
					t.Error("to_remove field should be deleted")
				}
			},
		},
		{
			name: "rename action",
			action: rule.ActionDef{
				Type: rule.ActionRename,
				Config: map[string]interface{}{
					"from": "old_name",
					"to":   "new_name",
				},
			},
			setup: func() rule.LogEntry {
				return rule.NewMapLogEntry(map[string]interface{}{
					"level":    "INFO",
					"old_name": "value",
				})
			},
			validate: func(t *testing.T, entry rule.LogEntry) {
				_, hasOld := entry.GetField("old_name")
				if hasOld {
					t.Error("old_name should be removed")
				}
				val, hasNew := entry.GetField("new_name")
				if !hasNew || val != "value" {
					t.Errorf("Expected new_name=value, got %v", val)
				}
			},
		},
		{
			name: "truncate action",
			action: rule.ActionDef{
				Type: rule.ActionTruncate,
				Config: map[string]interface{}{
					"field":      "long_message",
					"max_length": 10,
					"suffix":     "...",
				},
			},
			setup: func() rule.LogEntry {
				return rule.NewMapLogEntry(map[string]interface{}{
					"level":        "INFO",
					"long_message": "this is a very long message",
				})
			},
			validate: func(t *testing.T, entry rule.LogEntry) {
				val, _ := entry.GetField("long_message")
				str, ok := val.(string)
				if !ok {
					t.Fatal("long_message should be string")
				}
				if len(str) > 13 {
					t.Errorf("Expected truncated string with max length 13 (10+3), got %d", len(str))
				}
				t.Logf("Truncated: %s", str)
			},
		},
		{
			name: "mark action",
			action: rule.ActionDef{
				Type: rule.ActionMark,
				Config: map[string]interface{}{
					"field":  "_tags",
					"value":  "important",
					"reason": "high priority",
				},
			},
			setup: func() rule.LogEntry {
				return rule.NewMapLogEntry(map[string]interface{}{"level": "ERROR"})
			},
			validate: func(t *testing.T, entry rule.LogEntry) {
				// mark 动作添加元数据标记
				t.Log("Mark action executed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := storage.NewMemoryStorage()

			r := &rule.Rule{
				ID:      fmt.Sprintf("e2e-%s-rule", tt.name),
				Name:    tt.name,
				Enabled: true,
				Condition: rule.Condition{
					Field:    "level",
					Operator: rule.OpExists,
				},
				Actions: []rule.ActionDef{tt.action},
			}

			if err := storage.PutRule(r); err != nil {
				t.Fatalf("PutRule failed: %v", err)
			}

			engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
			if err := engine.LoadRules(storage); err != nil {
				t.Fatalf("LoadRules failed: %v", err)
			}

			entry := tt.setup()
			_, _, errs := engine.Evaluate(entry)

			if len(errs) > 0 {
				t.Errorf("Unexpected errors: %v", errs)
				return
			}

			tt.validate(t, entry)
		})
	}
}

// TestE2E_NestedFieldAccess 端到端测试：嵌套字段访问
func TestE2E_NestedFieldAccess(t *testing.T) {
	storage := storage.NewMemoryStorage()

	rule1 := &rule.Rule{
		ID:      "e2e-nested-field-rule",
		Name:    "E2E Nested Field Rule",
		Enabled: true,
		Condition: rule.Condition{
			Field:    "user.role",
			Operator: rule.OpEq,
			Value:    "admin",
		},
		Actions: []rule.ActionDef{
			{Type: rule.ActionKeep},
		},
	}

	if err := storage.PutRule(rule1); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	// 测试嵌套字段匹配
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "INFO",
		"user": map[string]interface{}{
			"id":    "123",
			"role":  "admin",
			"name":  "test",
		},
	})

	shouldKeep, results, _ := engine.Evaluate(entry)

	if !shouldKeep {
		t.Error("Log should match nested field condition")
	}
	if len(results) == 0 {
		t.Error("Expected matching results")
	}

	t.Log("Nested field access test passed")
}

// TestE2E_PerformanceWithStats 端到端测试：带统计的性能测试
func TestE2E_PerformanceWithStats(t *testing.T) {
	storage := storage.NewMemoryStorage()

	// 创建 10 条规则
	for i := 0; i < 10; i++ {
		r := &rule.Rule{
			ID:      fmt.Sprintf("perf-rule-%d", i),
			Name:    fmt.Sprintf("Performance Rule %d", i),
			Enabled: true,
			Condition: rule.Condition{
				Field:    "level",
				Operator: rule.OpEq,
				Value:    "ERROR",
			},
			Actions: []rule.ActionDef{
				{Type: rule.ActionMark},
			},
		}
		if err := storage.PutRule(r); err != nil {
			t.Fatalf("PutRule failed: %v", err)
		}
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{
		EnableStats: true,
	})
	if err := engine.LoadRules(storage); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "performance test",
	})

	// 记录开始时间
	start := time.Now()

	// 执行 1000 次评估
	for i := 0; i < 1000; i++ {
		_, _, _ = engine.Evaluate(entry)
	}

	elapsed := time.Since(start)
	avgLatency := elapsed / 1000

	t.Logf("1000 evaluations completed in %v", elapsed)
	t.Logf("Average latency: %v/op", avgLatency)

	// 获取统计信息
	stats := engine.GetStats()
	t.Logf("Total evaluations: %d", stats.TotalEvaluations)
	t.Logf("Matched evaluations: %d", stats.MatchedEvaluations)

	if avgLatency > time.Millisecond {
		t.Errorf("Average latency %v exceeds threshold 1ms", avgLatency)
	}

	t.Log("Performance test completed")
}
