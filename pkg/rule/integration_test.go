package rule_test

import (
	"testing"

	"github.com/log-system/logos/pkg/rule"
	"github.com/log-system/logos/pkg/rule/storage"
)

// TestMemoryStorage 测试内存存储
func TestMemoryStorage(t *testing.T) {
	s := storage.NewMemoryStorage()

	// 测试添加规则
	r := &rule.Rule{
		ID:      "test-rule-1",
		Name:    "Test Rule",
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

	err := s.PutRule(r)
	if err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	// 测试加载规则
	rules, err := s.LoadRules()
	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	// 测试删除规则
	err = s.DeleteRule("test-rule-1")
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	rules, err = s.LoadRules()
	if err != nil {
		t.Fatalf("LoadRules after delete failed: %v", err)
	}

	if len(rules) != 0 {
		t.Errorf("Expected 0 rules after delete, got %d", len(rules))
	}
}

// TestRuleEngineWithMemoryStorage 测试规则引擎与内存存储集成
func TestRuleEngineWithMemoryStorage(t *testing.T) {
	s := storage.NewMemoryStorage()

	// 添加多条规则
	rules := []*rule.Rule{
		{
			ID:      "001-drop-debug",
			Name:    "Drop Debug Logs",
			Enabled: true,
			Condition: rule.Condition{
				Field:    "level",
				Operator: rule.OpEq,
				Value:    "DEBUG",
			},
			Actions: []rule.ActionDef{
				{Type: rule.ActionDrop},
			},
		},
		{
			ID:      "002-keep-errors",
			Name:    "Keep Error Logs",
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
		{
			ID:      "003-mask-passwords",
			Name:    "Mask Passwords",
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
	}

	for _, r := range rules {
		if err := s.PutRule(r); err != nil {
			t.Fatalf("PutRule failed: %v", err)
		}
	}

	// 创建引擎并加载规则
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(s); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	// 测试规则 1: DEBUG 日志应该被 drop
	entry1 := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "DEBUG",
		"message": "debug message",
	})
	shouldKeep1, _, _ := engine.Evaluate(entry1)
	if shouldKeep1 {
		t.Error("DEBUG log should be dropped")
	}

	// 测试规则 2: ERROR 日志应该被 keep
	entry2 := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "ERROR",
		"message": "error message",
	})
	shouldKeep2, results, _ := engine.Evaluate(entry2)
	if !shouldKeep2 {
		t.Error("ERROR log should be kept")
	}
	if len(results) == 0 {
		t.Error("Expected matching results")
	}

	// 测试规则 3: 包含 password 的消息应该被 mask
	entry3 := rule.NewMapLogEntry(map[string]interface{}{
		"level":   "INFO",
		"message": "user password is secret",
	})
	shouldKeep3, _, _ := engine.Evaluate(entry3)
	if !shouldKeep3 {
		t.Error("INFO log should be kept by default")
	}
	// 检查是否被 mask
	maskedValue, _ := entry3.GetField("message")
	if maskedValue == "user password is secret" {
		t.Error("Password should be masked")
	}
	t.Logf("Masked message: %v", maskedValue)
}

// TestRuleEngineHotReload 测试热加载（模拟）
func TestRuleEngineHotReload(t *testing.T) {
	s := storage.NewMemoryStorage()
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})

	// 初始加载
	if err := engine.LoadRules(s); err != nil {
		t.Fatalf("Initial LoadRules failed: %v", err)
	}

	// 添加新规则
	newRule := &rule.Rule{
		ID:      "new-rule",
		Name:    "New Rule",
		Enabled: true,
		Condition: rule.Condition{
			Field:    "service",
			Operator: rule.OpEq,
			Value:    "test-service",
		},
		Actions: []rule.ActionDef{
			{Type: rule.ActionMark},
		},
	}

	if err := s.PutRule(newRule); err != nil {
		t.Fatalf("PutRule failed: %v", err)
	}

	// 重新加载
	if err := engine.LoadRules(s); err != nil {
		t.Fatalf("Reload LoadRules failed: %v", err)
	}

	// 验证新规则生效
	entry := rule.NewMapLogEntry(map[string]interface{}{
		"service": "test-service",
	})
	_, results, _ := engine.Evaluate(entry)

	if len(results) == 0 {
		t.Error("New rule should be effective after reload")
	}
}

// TestRuleEngineTermination 测试终止性动作
func TestRuleEngineTermination(t *testing.T) {
	s := storage.NewMemoryStorage()

	// 创建会终止的规则链
	rules := []*rule.Rule{
		{
			ID:      "001-drop-all",
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
			ID:      "002-should-not-run",
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
		if err := s.PutRule(r); err != nil {
			t.Fatalf("PutRule failed: %v", err)
		}
	}

	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
	if err := engine.LoadRules(s); err != nil {
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
		t.Errorf("Expected only 1 rule result (due to drop termination), got %d", len(results))
	}

	if results[0].Terminate {
		t.Log("Drop action correctly terminated rule chain")
	}
}

// TestCompositeConditionDeepNesting 测试深层嵌套复合条件
func TestCompositeConditionDeepNesting(t *testing.T) {
	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})

	// 创建深层嵌套条件
	deepCondition := rule.Condition{
		All: []rule.Condition{
			{
				Field:    "level",
				Operator: rule.OpEq,
				Value:    "ERROR",
			},
			{
				Any: []rule.Condition{
					{
						Field:    "service",
						Operator: rule.OpEq,
						Value:    "api",
					},
					{
						All: []rule.Condition{
							{
								Field:    "service",
								Operator: rule.OpEq,
								Value:    "worker",
							},
							{
								Not: &rule.Condition{
									Field:    "environment",
									Operator: rule.OpEq,
									Value:    "dev",
								},
							},
						},
					},
				},
			},
		},
	}

	rule1 := &rule.Rule{
		ID:        "complex-rule",
		Name:      "Complex Rule",
		Enabled:   true,
		Condition: deepCondition,
		Actions: []rule.ActionDef{
			{Type: rule.ActionKeep},
		},
	}

	engine.SetRules([]*rule.Rule{rule1})

	// 测试场景 1: 匹配所有条件
	entry1 := rule.NewMapLogEntry(map[string]interface{}{
		"level":       "ERROR",
		"service":     "api",
		"environment": "prod",
	})
	shouldKeep1, _, _ := engine.Evaluate(entry1)
	if !shouldKeep1 {
		t.Error("Entry1 should match (level=ERROR, service=api)")
	}

	// 测试场景 2: 匹配嵌套 any 的第二分支
	entry2 := rule.NewMapLogEntry(map[string]interface{}{
		"level":       "ERROR",
		"service":     "worker",
		"environment": "prod",
	})
	shouldKeep2, _, _ := engine.Evaluate(entry2)
	if !shouldKeep2 {
		t.Error("Entry2 should match (level=ERROR, service=worker, env!=dev)")
	}

	// 测试场景 3: 不匹配
	entry3 := rule.NewMapLogEntry(map[string]interface{}{
		"level":       "INFO",
		"service":     "api",
		"environment": "prod",
	})
	shouldKeep3, results3, _ := engine.Evaluate(entry3)
	if shouldKeep3 && len(results3) == 0 {
		// 这是正确的：没有规则匹配，默认保留日志
		t.Log("Entry3: no rules matched, default to keep")
	}
	if len(results3) > 0 {
		t.Error("Entry3 should not match any rules")
	}
}

// TestRuleValidation 测试规则验证
func TestRuleValidation(t *testing.T) {
	tests := []struct {
		name    string
		rule    *rule.Rule
		wantErr bool
	}{
		{
			name: "valid rule",
			rule: &rule.Rule{
				ID:      "rule1",
				Name:    "Test",
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
			wantErr: false,
		},
		{
			name: "missing ID",
			rule: &rule.Rule{
				Name:    "Test",
				Enabled: true,
				Condition: rule.Condition{
					Field:    "level",
					Operator: rule.OpEq,
					Value:    "ERROR",
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			rule: &rule.Rule{
				ID:      "rule1",
				Enabled: true,
				Condition: rule.Condition{
					Field:    "level",
					Operator: rule.OpEq,
					Value:    "ERROR",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid condition",
			rule: &rule.Rule{
				ID:      "rule1",
				Name:    "Test",
				Enabled: true,
				Condition: rule.Condition{
					Field:    "",
					Operator: "",
					Value:    "ERROR",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Rule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
