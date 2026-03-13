package rule_test

import (
	"testing"

	"github.com/log-system/logos/pkg/rule"
)

// BenchmarkConditionEvaluate 基准测试：单条件评估
func BenchmarkConditionEvaluate(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Field:    "level",
		Operator: rule.OpEq,
		Value:    "ERROR",
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "test message",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkConditionEvaluateContains 基准测试：contains 操作符
func BenchmarkConditionEvaluateContains(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Field:    "message",
		Operator: rule.OpContains,
		Value:    "error",
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "connection error occurred in module",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkConditionEvaluateMatches 基准测试：regex matches 操作符
func BenchmarkConditionEvaluateMatches(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Field:    "message",
		Operator: rule.OpMatches,
		Value:    "error.*occurred",
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "connection error occurred in module",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkCompositeConditionAll 基准测试：all 复合条件
func BenchmarkCompositeConditionAll(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		All: []rule.Condition{
			{Field: "level", Operator: rule.OpEq, Value: "ERROR"},
			{Field: "service", Operator: rule.OpEq, Value: "api"},
			{Field: "environment", Operator: rule.OpEq, Value: "prod"},
		},
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":       "ERROR",
		"service":     "api",
		"environment": "prod",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkCompositeConditionAny 基准测试：any 复合条件
func BenchmarkCompositeConditionAny(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Any: []rule.Condition{
			{Field: "level", Operator: rule.OpEq, Value: "ERROR"},
			{Field: "level", Operator: rule.OpEq, Value: "PANIC"},
			{Field: "level", Operator: rule.OpEq, Value: "FATAL"},
		},
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level": "INFO",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkNestedCondition 基准测试：嵌套复合条件
func BenchmarkNestedCondition(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		All: []rule.Condition{
			{Field: "level", Operator: rule.OpEq, Value: "ERROR"},
			{
				Any: []rule.Condition{
					{Field: "service", Operator: rule.OpEq, Value: "api"},
					{Field: "service", Operator: rule.OpEq, Value: "worker"},
				},
			},
			{
				Not: &rule.Condition{
					Field:    "environment",
					Operator: rule.OpEq,
					Value:    "dev",
				},
			},
		},
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":       "ERROR",
		"service":     "api",
		"environment": "prod",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkRuleEngineSingleRule 基准测试：单条规则评估
func BenchmarkRuleEngineSingleRule(b *testing.B) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	engine.SetRules([]*rule.Rule{
		{
			ID:      "rule1",
			Name:    "Error Filter",
			Enabled: true,
			Condition: rule.Condition{
				Field:    "level",
				Operator: rule.OpEq,
				Value:    "ERROR",
			},
			Actions: []rule.ActionDef{
				{Type: rule.ActionDrop},
			},
		},
	})

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "test",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.Evaluate(entry)
	}
}

// BenchmarkRuleEngineMultipleRules 基准测试：多条规则评估
func BenchmarkRuleEngineMultipleRules(b *testing.B) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})

	// 创建 10 条规则
	rules := make([]*rule.Rule, 10)
	for i := 0; i < 10; i++ {
		rules[i] = &rule.Rule{
			ID:      string(rune('0' + i)),
			Name:    "Rule",
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
	}

	engine.SetRules(rules)

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "test",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.Evaluate(entry)
	}
}

// BenchmarkRuleEngineWithTransform 基准测试：带转换动作的规则评估
func BenchmarkRuleEngineWithTransform(b *testing.B) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	engine.SetRules([]*rule.Rule{
		{
			ID:      "rule1",
			Name:    "Mask Password",
			Enabled: true,
			Condition: rule.Condition{
				Field:    "message",
				Operator: rule.OpContains,
				Value:    "password",
			},
			Actions: []rule.ActionDef{
				{
					Type: rule.ActionMask,
					Config: map[string]interface{}{
						"field": "message",
					},
				},
			},
		},
	})

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "INFO",
		"message": "user password is secret123",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.Evaluate(entry)
	}
}

// BenchmarkRuleEngineWithSetAction 基准测试：set 动作性能
func BenchmarkRuleEngineWithSetAction(b *testing.B) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	engine.SetRules([]*rule.Rule{
		{
			ID:      "rule1",
			Name:    "Add Environment",
			Enabled: true,
			Condition: rule.Condition{
				Field:    "level",
				Operator: rule.OpExists,
			},
			Actions: []rule.ActionDef{
				{
					Type: rule.ActionSet,
					Config: map[string]interface{}{
						"field": "environment",
						"value": "production",
					},
				},
			},
		},
	})

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level": "INFO",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.Evaluate(entry)
	}
}

// BenchmarkMapLogEntryGetField 基准测试：MapLogEntry 字段访问
func BenchmarkMapLogEntryGetField(b *testing.B) {
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "test",
		"nested": map[string]interface{}{
			"field1": "value1",
			"field2": "value2",
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = entry.GetField("level")
	}
}

// BenchmarkMapLogEntryGetFieldNested 基准测试：嵌套字段访问
func BenchmarkMapLogEntryGetFieldNested(b *testing.B) {
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "test",
		"nested": map[string]interface{}{
			"field1": "value1",
			"field2": "value2",
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = entry.GetField("nested.field1")
	}
}

// BenchmarkMapLogEntrySetField 基准测试：字段设置
func BenchmarkMapLogEntrySetField(b *testing.B) {
	baseData := map[string]interface{}{
		"level":   "ERROR",
		"message": "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := rule.NewMapLogEntry(cloneMap(baseData))
		_ = entry.SetField("new_field", "new_value")
	}
}

func cloneMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// BenchmarkRuleEngineStats 基准测试：统计性能
func BenchmarkRuleEngineStats(b *testing.B) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{
		EnableStats: true,
	})
	engine.SetRules([]*rule.Rule{
		{
			ID:      "rule1",
			Name:    "Test",
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
	})

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level": "ERROR",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.Evaluate(entry)
		if i%1000 == 0 {
			_ = engine.GetStats()
		}
	}
}

// BenchmarkRegexCache 基准测试：正则缓存性能
func BenchmarkRegexCache(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Field:    "message",
		Operator: rule.OpMatches,
		Value:    "^error.*\\d+$",
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"message": "error occurred 12345",
	})

	// 第一次编译正则
	_, _ = evaluator.Evaluate(condition, entry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkInOperator 基准测试：in 操作符
func BenchmarkInOperator(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Field:    "level",
		Operator: rule.OpIn,
		Value:    []interface{}{"ERROR", "PANIC", "FATAL"},
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level": "ERROR",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkStartsWithOperator 基准测试：starts_with 操作符
func BenchmarkStartsWithOperator(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Field:    "message",
		Operator: rule.OpStartsWith,
		Value:    "connection",
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"message": "connection timeout to server",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkEndsWithOperator 基准测试：ends_with 操作符
func BenchmarkEndsWithOperator(b *testing.B) {
	evaluator := rule.NewConditionEvaluator()
	condition := rule.Condition{
		Field:    "message",
		Operator: rule.OpEndsWith,
		Value:    "failed",
	}
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"message": "database connection failed",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, entry)
	}
}

// BenchmarkRuleEngineDeepNestedCondition 基准测试：深层嵌套条件评估
func BenchmarkRuleEngineDeepNestedCondition(b *testing.B) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})

	// 创建 5 层嵌套条件
	deepCondition := rule.Condition{
		All: []rule.Condition{
			{Field: "level", Operator: rule.OpEq, Value: "ERROR"},
			{
				Any: []rule.Condition{
					{Field: "service", Operator: rule.OpEq, Value: "api"},
					{
						All: []rule.Condition{
							{Field: "service", Operator: rule.OpEq, Value: "worker"},
							{
								Not: &rule.Condition{
									Field:    "env",
									Operator: rule.OpEq,
									Value:    "dev",
								},
							},
						},
					},
				},
			},
			{
				Any: []rule.Condition{
					{Field: "priority", Operator: rule.OpGt, Value: 5},
					{Field: "priority", Operator: rule.OpLe, Value: 1},
				},
			},
		},
	}

	engine.SetRules([]*rule.Rule{
		{
			ID:        "deep-rule",
			Name:      "Deep Nested Rule",
			Enabled:   true,
			Condition: deepCondition,
			Actions: []rule.ActionDef{
				{Type: rule.ActionKeep},
			},
		},
	})

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":    "ERROR",
		"service":  "api",
		"env":      "prod",
		"priority": 8,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.Evaluate(entry)
	}
}

// BenchmarkActionTransformations 基准测试：连续转换动作性能
func BenchmarkActionTransformations(b *testing.B) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	engine.SetRules([]*rule.Rule{
		{
			ID:      "transform-rule",
			Name:    "Transform Chain",
			Enabled: true,
			Condition: rule.Condition{
				Field:    "level",
				Operator: rule.OpExists,
			},
			Actions: []rule.ActionDef{
				{Type: rule.ActionSet, Config: map[string]interface{}{"field": "processed", "value": true}},
				{Type: rule.ActionMask, Config: map[string]interface{}{"field": "password", "pattern": ".*"}},
				{Type: rule.ActionRemove, Config: map[string]interface{}{"field": "temp"}},
			},
		},
	})

	entry := rule.NewMapLogEntry(map[string]interface{}{
		"level":    "INFO",
		"password": "secret123",
		"temp":     "delete-me",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = engine.Evaluate(entry)
	}
}
