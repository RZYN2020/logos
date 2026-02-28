// Package config 提供配置管理单元测试
package config_test

import (
	"testing"
	"time"

	"github.com/log-system/log-processor/pkg/config"
)

func TestFilterConfigSerialization(t *testing.T) {
	cfg := &config.FilterConfig{
		ID:       "test-filter",
		Enabled:  true,
		Priority: 10,
		Service:  "test-service",
		Rules: []config.FilterRule{
			{
				Name:    "test-rule",
				Field:   "message",
				Pattern: "test.*",
				Action:  config.ActionAllow,
			},
		},
		UpdatedAt: time.Now(),
	}

	if cfg.ID != "test-filter" {
		t.Errorf("Expected ID 'test-filter', got '%s'", cfg.ID)
	}

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if len(cfg.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(cfg.Rules))
	}
}

func TestFilterRuleCompile(t *testing.T) {
	rule := config.FilterRule{
		Name:    "test-rule",
		Field:   "message",
		Pattern: "test.*",
	}

	err := rule.Compile()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !rule.Match("testing") {
		t.Error("Expected 'testing' to match")
	}

	if rule.Match("other") {
		t.Error("Expected 'other' not to match")
	}
}

func TestFilterRuleInvalidPattern(t *testing.T) {
	rule := config.FilterRule{
		Name:    "test-rule",
		Field:   "message",
		Pattern: "[invalid",
	}

	err := rule.Compile()
	if err == nil {
		t.Error("Expected error for invalid pattern")
	}
}

func TestFilterActionString(t *testing.T) {
	tests := []struct {
		action   config.FilterAction
		expected string
	}{
		{config.ActionAllow, "allow"},
		{config.ActionDrop, "drop"},
		{config.ActionMark, "mark"},
	}

	for _, test := range tests {
		result := test.action.String()
		if result != test.expected {
			t.Errorf("Expected %v.String() to be '%s', got '%s'", test.action, test.expected, result)
		}
	}
}

func TestFilterRuleMatch(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{"test.*", "testing", true},
		{"test.*", "other", false},
		{"ERROR.*", "ERROR: something failed", true},
		{"ERROR.*", "INFO: all good", false},
	}

	for _, test := range tests {
		rule := config.FilterRule{
			Name:    "test-rule",
			Field:   "message",
			Pattern: test.pattern,
		}

		if err := rule.Compile(); err != nil {
			t.Fatalf("Failed to compile pattern: %v", err)
		}

		got := rule.Match(test.input)
		if got != test.want {
			t.Errorf("Rule.Match(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestEtcdConfig(t *testing.T) {
	cfg := config.DefaultEtcdConfig()

	if len(cfg.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(cfg.Endpoints))
	}

	if cfg.Endpoints[0] != "localhost:2379" {
		t.Errorf("Expected endpoint 'localhost:2379', got '%s'", cfg.Endpoints[0])
	}

	if cfg.DialTimeout != 5*time.Second {
		t.Errorf("Expected dial timeout 5s, got %v", cfg.DialTimeout)
	}
}

func TestProcessorConfig(t *testing.T) {
	cfg := &config.ProcessorConfig{
		Version:   "1.0.0",
		UpdatedAt: time.Now(),
		Filters: []config.FilterConfig{
			{
				ID:      "filter-1",
				Enabled: true,
				Rules:   []config.FilterRule{},
			},
		},
		Parsers: []config.ParserConfig{
			{
				Name:    "json-parser",
				Type:    "json",
				Enabled: true,
			},
		},
		Transforms: []config.TransformRule{
			{
				SourceField: "message",
				TargetField: "extracted",
				Extractor:   "regex",
			},
		},
	}

	if cfg.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", cfg.Version)
	}

	if len(cfg.Filters) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(cfg.Filters))
	}

	if len(cfg.Parsers) != 1 {
		t.Errorf("Expected 1 parser, got %d", len(cfg.Parsers))
	}

	if len(cfg.Transforms) != 1 {
		t.Errorf("Expected 1 transform rule, got %d", len(cfg.Transforms))
	}
}
