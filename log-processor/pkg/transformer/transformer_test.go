// Package transformer 提供转换器集成测试
package transformer_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/log-system/log-processor/pkg/analyzer"
	"github.com/log-system/log-processor/pkg/parser"
	"github.com/log-system/log-processor/pkg/transformer"
)

// === Basic Transformer Tests ===

func TestTransformer_BasicTransform(t *testing.T) {
	tf := transformer.NewTransformer()

	parsed := &parser.ParsedLog{
		Timestamp: time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "Test message",
		Service:   "test-service",
		TraceID:   "trace123",
		SpanID:    "span456",
		Fields:    map[string]interface{}{"custom": "value"},
		Raw:       `{"level":"INFO","message":"Test message"}`,
		Format:    parser.FormatJSON,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", result.Level)
	}
	if result.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", result.Message)
	}
	if result.Service != "test-service" {
		t.Errorf("Expected service 'test-service', got %s", result.Service)
	}
	if result.TraceID != "trace123" {
		t.Errorf("Expected trace_id 'trace123', got %s", result.TraceID)
	}
	if result.Format != "json" {
		t.Errorf("Expected format 'json', got %s", result.Format)
	}
}

func TestTransformer_FieldsPreservation(t *testing.T) {
	tf := transformer.NewTransformer()

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test",
		Fields: map[string]interface{}{
			"string_field": "value",
			"int_field":    42,
			"float_field":  3.14,
			"bool_field":   true,
			"array_field":  []interface{}{1, 2, 3},
			"nested_field": map[string]interface{}{"key": "value"},
		},
		Raw:    "test",
		Format: parser.FormatJSON,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Fields["string_field"] != "value" {
		t.Errorf("Expected string_field to be preserved")
	}
	if result.Fields["int_field"] != 42 {
		t.Errorf("Expected int_field to be preserved")
	}
	if result.Fields["float_field"] != 3.14 {
		t.Errorf("Expected float_field to be preserved")
	}
	if result.Fields["bool_field"] != true {
		t.Errorf("Expected bool_field to be preserved")
	}
	if result.Fields["array_field"] == nil {
		t.Errorf("Expected array_field to be preserved")
	}
	if result.Fields["nested_field"] == nil {
		t.Errorf("Expected nested_field to be preserved")
	}
}

// === Rule Management Tests ===

func TestTransformer_AddRule(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "test_rule",
		SourceField: "message",
		TargetField: "extracted_value",
		Extractor:   "regex",
		Config:      map[string]interface{}{"pattern": `value: (\w+)`},
		Enabled:     true,
	}

	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Unexpected error adding rule: %v", err)
	}
}

func TestTransformer_AddRule_Validation(t *testing.T) {
	tf := transformer.NewTransformer()

	tests := []struct {
		name    string
		rule    transformer.TransformRule
		wantErr bool
	}{
		{
			name: "Valid rule",
			rule: transformer.TransformRule{
				Name: "valid", SourceField: "msg", TargetField: "out", Extractor: "direct", Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			rule: transformer.TransformRule{
				SourceField: "msg", TargetField: "out", Extractor: "direct",
			},
			wantErr: true,
		},
		{
			name: "Empty source",
			rule: transformer.TransformRule{
				Name: "test", TargetField: "out", Extractor: "direct",
			},
			wantErr: true,
		},
		{
			name: "Empty target",
			rule: transformer.TransformRule{
				Name: "test", SourceField: "msg", Extractor: "direct",
			},
			wantErr: true,
		},
		{
			name: "Empty extractor",
			rule: transformer.TransformRule{
				Name: "test", SourceField: "msg", TargetField: "out",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tf.AddRule(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestTransformer_RemoveRule(t *testing.T) {
	tf := transformer.NewTransformer()

	// Add a rule
	rule := transformer.TransformRule{
		Name: "to_remove", SourceField: "msg", TargetField: "out", Extractor: "direct", Enabled: true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Remove it
	err = tf.RemoveRule("to_remove")
	if err != nil {
		t.Fatalf("Failed to remove rule: %v", err)
	}

	// Try to remove non-existent rule
	err = tf.RemoveRule("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent rule")
	}
}

func TestTransformer_ApplyRules(t *testing.T) {
	tf := transformer.NewTransformer()

	rules := []transformer.TransformRule{
		{Name: "rule1", SourceField: "msg", TargetField: "out1", Extractor: "direct", Enabled: true},
		{Name: "rule2", SourceField: "raw", TargetField: "out2", Extractor: "direct", Enabled: true},
	}

	err := tf.ApplyRules(rules)
	if err != nil {
		t.Fatalf("Unexpected error applying rules: %v", err)
	}
}

func TestTransformer_ApplyRules_InvalidRule(t *testing.T) {
	tf := transformer.NewTransformer()

	rules := []transformer.TransformRule{
		{Name: "valid", SourceField: "msg", TargetField: "out", Extractor: "direct", Enabled: true},
		{Name: "", SourceField: "msg", TargetField: "out", Extractor: "direct", Enabled: true}, // Invalid
	}

	err := tf.ApplyRules(rules)
	if err == nil {
		t.Error("Expected error for invalid rule")
	}
}

// === Extractor Tests ===

func TestTransformer_RegexExtractor(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "extract_ip",
		SourceField: "message",
		TargetField: "ip_address",
		Extractor:   "regex",
		Config:      map[string]interface{}{"pattern": `(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`},
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Connection from 192.168.1.100 established",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["ip_address"] != "192.168.1.100" {
		t.Errorf("Expected ip_address '192.168.1.100', got %v", result.ExtractedFields["ip_address"])
	}
}

func TestTransformer_RegexExtractor_NamedGroups(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "extract_http",
		SourceField: "message",
		TargetField: "http_info",
		Extractor:   "regex",
		Config:      map[string]interface{}{"pattern": `(?P<method>GET|POST)\s+(?P<path>/[\w/]+)`},
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Request: GET /api/users received",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	httpInfo, ok := result.ExtractedFields["http_info"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected http_info to be a map")
	}

	if httpInfo["method"] != "GET" {
		t.Errorf("Expected method 'GET', got %v", httpInfo["method"])
	}
	if httpInfo["path"] != "/api/users" {
		t.Errorf("Expected path '/api/users', got %v", httpInfo["path"])
	}
}

func TestTransformer_TemplateExtractor(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "format_message",
		SourceField: "message",
		TargetField: "formatted",
		Extractor:   "template",
		Config:      map[string]interface{}{"template": "Log: {{source}}"},
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test message",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["formatted"] != "Log: Test message" {
		t.Errorf("Expected formatted 'Log: Test message', got %v", result.ExtractedFields["formatted"])
	}
}

func TestTransformer_DirectExtractor(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "copy_message",
		SourceField: "message",
		TargetField: "message_copy",
		Extractor:   "direct",
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Original message",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["message_copy"] != "Original message" {
		t.Errorf("Expected message_copy 'Original message', got %v", result.ExtractedFields["message_copy"])
	}
}

func TestTransformer_LowercaseExtractor(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "lowercase_level",
		SourceField: "level",
		TargetField: "level_lower",
		Extractor:   "lowercase",
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   "test",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["level_lower"] != "error" {
		t.Errorf("Expected level_lower 'error', got %v", result.ExtractedFields["level_lower"])
	}
}

func TestTransformer_UppercaseExtractor(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "uppercase_message",
		SourceField: "message",
		TargetField: "message_upper",
		Extractor:   "uppercase",
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "hello world",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["message_upper"] != "HELLO WORLD" {
		t.Errorf("Expected message_upper 'HELLO WORLD', got %v", result.ExtractedFields["message_upper"])
	}
}

func TestTransformer_SplitExtractor(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "split_tags",
		SourceField: "message",
		TargetField: "tags",
		Extractor:   "split",
		Config:      map[string]interface{}{"delimiter": ","},
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "tag1,tag2,tag3",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	tags, ok := result.ExtractedFields["tags"].([]string)
	if !ok {
		t.Fatalf("Expected tags to be []string")
	}

	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}
}

func TestTransformer_SplitExtractor_TrimSpaces(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "split_tags",
		SourceField: "message",
		TargetField: "tags",
		Extractor:   "split",
		Config:      map[string]interface{}{"delimiter": ","},
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   " tag1 , tag2 , tag3 ",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	tags, ok := result.ExtractedFields["tags"].([]string)
	if !ok {
		t.Fatalf("Expected tags to be []string")
	}

	for _, tag := range tags {
		if tag != "tag1" && tag != "tag2" && tag != "tag3" {
			t.Errorf("Expected trimmed tag, got %s", tag)
		}
	}
}

// === Analysis Integration Tests ===

func TestTransformer_WithAnalysisResult(t *testing.T) {
	tf := transformer.NewTransformer()

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test message with IP 192.168.1.1",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	analysis := &analyzer.AnalysisResult{
		Entities: []analyzer.Entity{
			{Type: "IP_ADDRESS", Value: "192.168.1.1", Start: 20, End: 31},
		},
		Keywords:   []string{"test", "message", "IP"},
		Language:   "en",
		Category:   "general",
		Sentiment:  analyzer.SentimentResult{Score: 0.5, Label: "positive", Mixed: false},
		AnalyzedAt: time.Now(),
	}

	result, err := tf.Transform(parsed, analysis)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check entities are extracted
	foundEntity := false
	for k, v := range result.ExtractedFields {
		if k == "keywords" {
			foundEntity = true
			keywords, ok := v.([]string)
			if !ok {
				t.Errorf("Expected keywords to be []string")
			}
			if len(keywords) != 3 {
				t.Errorf("Expected 3 keywords, got %d", len(keywords))
			}
		}
	}
	if !foundEntity {
		t.Error("Expected keywords to be extracted")
	}
}

func TestTransformer_AnalysisSentiment(t *testing.T) {
	tf := transformer.NewTransformer()

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	analysis := &analyzer.AnalysisResult{
		Sentiment: analyzer.SentimentResult{Score: -0.8, Label: "negative", Mixed: false},
		Language:  "en",
	}

	result, err := tf.Transform(parsed, analysis)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Add rule to extract sentiment
	rule := transformer.TransformRule{
		Name:        "sentiment_label",
		SourceField: "sentiment_label",
		TargetField: "sentiment_out",
		Extractor:   "direct",
		Enabled:     true,
	}
	tf.AddRule(rule)

	// Re-transform
	result, err = tf.Transform(parsed, analysis)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["sentiment_out"] != "negative" {
		t.Errorf("Expected sentiment 'negative', got %v", result.ExtractedFields["sentiment_out"])
	}
}

// === Enabled/Disabled Rule Tests ===

func TestTransformer_DisabledRule(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "disabled_rule",
		SourceField: "message",
		TargetField: "should_not_exist",
		Extractor:   "direct",
		Enabled:     false, // Disabled
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, ok := result.ExtractedFields["should_not_exist"]; ok {
		t.Error("Expected disabled rule not to produce output")
	}
}

// === JSON Loading Tests ===

func TestTransformer_LoadRulesFromJSON(t *testing.T) {
	tf := transformer.NewTransformer()

	jsonData := []byte(`{
		"rules": [
			{
				"name": "json_rule",
				"source_field": "message",
				"target_field": "extracted",
				"extractor": "regex",
				"config": {"pattern": "(\\w+)"},
				"enabled": true
			}
		],
		"version": "1.0.0",
		"updated_at": "2026-02-28T12:00:00Z"
	}`)

	err := tf.LoadRulesFromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to load rules from JSON: %v", err)
	}
}

func TestTransformer_LoadRulesFromJSON_InvalidJSON(t *testing.T) {
	tf := transformer.NewTransformer()

	jsonData := []byte(`{invalid json}`)

	err := tf.LoadRulesFromJSON(jsonData)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// === Export Rules Tests ===

func TestTransformer_ExportRules(t *testing.T) {
	tf := transformer.NewTransformer()

	rules := []transformer.TransformRule{
		{Name: "rule1", SourceField: "msg", TargetField: "out1", Extractor: "direct", Enabled: true},
		{Name: "rule2", SourceField: "raw", TargetField: "out2", Extractor: "direct", Enabled: false},
	}

	err := tf.ApplyRules(rules)
	if err != nil {
		t.Fatalf("Failed to apply rules: %v", err)
	}

	exported, err := tf.ExportRules()
	if err != nil {
		t.Fatalf("Failed to export rules: %v", err)
	}

	if len(exported) != 2 {
		t.Errorf("Expected 2 exported rules, got %d", len(exported))
	}
}

// === Field Source Tests ===

func TestTransformer_SourceFromParsedFields(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "from_fields",
		SourceField: "custom_field",
		TargetField: "custom_output",
		Extractor:   "direct",
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Fields:    map[string]interface{}{"custom_field": "custom_value"},
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["custom_output"] != "custom_value" {
		t.Errorf("Expected custom_output 'custom_value', got %v", result.ExtractedFields["custom_output"])
	}
}

// === Edge Cases Tests ===

func TestTransformer_EmptyMessage(t *testing.T) {
	tf := transformer.NewTransformer()

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "",
		Raw:       "",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Message != "" {
		t.Errorf("Expected empty message")
	}
}

func TestTransformer_NilAnalysis(t *testing.T) {
	tf := transformer.NewTransformer()

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "test",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not panic with nil analysis
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestTransformer_MultipleRules(t *testing.T) {
	tf := transformer.NewTransformer()

	rules := []transformer.TransformRule{
		{Name: "rule1", SourceField: "message", TargetField: "out1", Extractor: "direct", Enabled: true},
		{Name: "rule2", SourceField: "level", TargetField: "out2", Extractor: "lowercase", Enabled: true},
		{Name: "rule3", SourceField: "raw", TargetField: "out3", Extractor: "uppercase", Enabled: true},
	}

	err := tf.ApplyRules(rules)
	if err != nil {
		t.Fatalf("Failed to apply rules: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   "test message",
		Raw:       "raw data",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ExtractedFields["out1"] != "test message" {
		t.Errorf("Expected out1 'test message', got %v", result.ExtractedFields["out1"])
	}
	if result.ExtractedFields["out2"] != "error" {
		t.Errorf("Expected out2 'error', got %v", result.ExtractedFields["out2"])
	}
	if result.ExtractedFields["out3"] != "RAW DATA" {
		t.Errorf("Expected out3 'RAW DATA', got %v", result.ExtractedFields["out3"])
	}
}

func TestTransformer_RegexNoMatch(t *testing.T) {
	tf := transformer.NewTransformer()

	rule := transformer.TransformRule{
		Name:        "extract_ip",
		SourceField: "message",
		TargetField: "ip_address",
		Extractor:   "regex",
		Config:      map[string]interface{}{"pattern": `(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`},
		Enabled:     true,
	}
	err := tf.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	parsed := &parser.ParsedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "No IP address in this message",
		Raw:       "test",
		Format:    parser.FormatUnstructured,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return nil for no match
	if result.ExtractedFields["ip_address"] != nil {
		t.Errorf("Expected nil for no match, got %v", result.ExtractedFields["ip_address"])
	}
}

func TestTransformer_TransformedLogJSONSerialization(t *testing.T) {
	tf := transformer.NewTransformer()

	parsed := &parser.ParsedLog{
		Timestamp: time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "Test message",
		Service:   "test-service",
		Fields:    map[string]interface{}{"key": "value"},
		Raw:       "test",
		Format:    parser.FormatJSON,
	}

	result, err := tf.Transform(parsed, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Verify JSON is valid
	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
}
