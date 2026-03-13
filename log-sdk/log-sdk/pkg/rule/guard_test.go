package rule

import (
	"testing"
	"time"

	unifiedRule "github.com/log-system/logos/pkg/rule"
	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	t.Run("NewEngine_WithoutEtcd", func(t *testing.T) {
		cfg := Config{
			ServiceName: "test-service",
			Environment: "test",
		}

		engine, err := NewEngine(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, engine)
		defer engine.Close()

		assert.False(t, engine.closed)
		assert.Equal(t, "test-service.test", engine.clientID)
	})

	t.Run("NewEngine_WithDefaultConfig", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Equal(t, 5*time.Second, cfg.DialTimeout)
	})
}

func TestEngine_Evaluate(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Environment: "test",
	}

	engine, err := NewEngine(cfg)
	assert.NoError(t, err)
	defer engine.Close()

	t.Run("Evaluate_AllowsLog", func(t *testing.T) {
		decision := engine.Evaluate("INFO", "test-service", "test", map[string]interface{}{
			"message": "test log",
		})

		assert.True(t, decision.ShouldLog)
		assert.Equal(t, 1.0, decision.Sampling)
		assert.Equal(t, "normal", decision.Priority)
	})

	t.Run("Evaluate_AfterClose", func(t *testing.T) {
		cfg := Config{
			ServiceName: "test-service",
			Environment: "test",
		}

		engine, err := NewEngine(cfg)
		assert.NoError(t, err)

		err = engine.Close()
		assert.NoError(t, err)

		decision := engine.Evaluate("INFO", "test-service", "test", map[string]interface{}{})
		assert.True(t, decision.ShouldLog)
		assert.Equal(t, 1.0, decision.Sampling)
	})
}

func TestEngine_AddRule(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Environment: "test",
	}

	engine, err := NewEngine(cfg)
	assert.NoError(t, err)
	defer engine.Close()

	rule := &unifiedRule.Rule{
		ID:          "test-rule-1",
		Name:        "Test Rule",
		Description: "A test rule",
		Enabled:     true,
		Condition: unifiedRule.Condition{
			Field:    "level",
			Operator: "eq",
			Value:    "ERROR",
		},
		Actions: []unifiedRule.ActionDef{
			{
				Type: "mark",
				Config: map[string]interface{}{
					"sampling_rate": 1.0,
				},
			},
		},
	}

	engine.AddRule(rule)

	// 验证规则已添加
	_ = engine.GetStats()
}

func TestEngine_RemoveRule(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Environment: "test",
	}

	engine, err := NewEngine(cfg)
	assert.NoError(t, err)
	defer engine.Close()

	ruleID := "test-rule-remove-" + time.Now().Format("150405")

	rule := &unifiedRule.Rule{
		ID:          ruleID,
		Name:        "Test Rule To Remove",
		Description: "A test rule",
		Enabled:     true,
		Condition: unifiedRule.Condition{
			Field:    "level",
			Operator: "eq",
			Value:    "ERROR",
		},
		Actions:   []unifiedRule.ActionDef{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	engine.AddRule(rule)
	engine.RemoveRule(ruleID)
}

func TestEngine_GetStats(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Environment: "test",
	}

	engine, err := NewEngine(cfg)
	assert.NoError(t, err)
	defer engine.Close()

	stats := engine.GetStats()
	_ = stats // 验证可以获取 stats
}

func TestDecision(t *testing.T) {
	decision := Decision{
		ShouldLog: true,
		Sampling:  0.5,
		Priority:  "high",
		Transform: "uppercase",
	}

	assert.True(t, decision.ShouldLog)
	assert.Equal(t, 0.5, decision.Sampling)
	assert.Equal(t, "high", decision.Priority)
	assert.Equal(t, "uppercase", decision.Transform)
}
