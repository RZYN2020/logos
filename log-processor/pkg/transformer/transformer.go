// Package transformer 提供日志结构化转换功能
package transformer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/log-system/log-processor/pkg/analyzer"
	"github.com/log-system/log-processor/pkg/parser"
)

// Transformer 转换器接口
type Transformer interface {
	Transform(parsed *parser.ParsedLog, analysis *analyzer.AnalysisResult) (*TransformedLog, error)
	ApplyRules(rules []TransformRule) error
	AddRule(rule TransformRule) error
	RemoveRule(name string) error
}

// TransformedLog 转换后的日志
type TransformedLog struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        string                 `json:"level"`
	Message      string                 `json:"message"`
	Service      string                 `json:"service"`
	TraceID      string                 `json:"trace_id,omitempty"`
	SpanID       string                 `json:"span_id,omitempty"`
	Fields       map[string]interface{} `json:"fields,omitempty"`
	Raw          string                 `json:"raw,omitempty"`
	Format       string                 `json:"format,omitempty"`
	ExtractedFields map[string]interface{} `json:"extracted_fields,omitempty"`
}

// TransformRule 转换规则
type TransformRule struct {
	Name        string                 `json:"name"`
	SourceField string                 `json:"source_field"`
	TargetField string                 `json:"target_field"`
	Extractor   string                 `json:"extractor"` // regex, template, jsonpath, direct
	Config      map[string]interface{} `json:"config,omitempty"`
	Enabled     bool                   `json:"enabled"`
	pattern     *regexp.Regexp         // 编译后的正则
}

// TransformerImpl 转换器实现
type TransformerImpl struct {
	mu      sync.RWMutex
	rules   []TransformRule
	funcs   map[string]ExtractorFunc
}

// ExtractorFunc 提取函数类型
type ExtractorFunc func(source string, config map[string]interface{}) (interface{}, error)

// NewTransformer 创建转换器
func NewTransformer() *TransformerImpl {
	t := &TransformerImpl{
		rules: make([]TransformRule, 0),
		funcs: make(map[string]ExtractorFunc),
	}

	// 注册默认提取器
	t.registerDefaultExtractors()

	return t
}

// registerDefaultExtractors 注册默认提取器
func (t *TransformerImpl) registerDefaultExtractors() {
	t.funcs["regex"] = extractByRegex
	t.funcs["template"] = extractByTemplate
	t.funcs["direct"] = extractDirect
	t.funcs["lowercase"] = extractLowercase
	t.funcs["uppercase"] = extractUppercase
	t.funcs["split"] = extractSplit
}

// AddRule 添加转换规则
func (t *TransformerImpl) AddRule(rule TransformRule) error {
	// 验证规则
	if err := t.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// 编译正则（如果是 regex 提取器）
	if rule.Extractor == "regex" && rule.Config != nil {
		if pattern, ok := rule.Config["pattern"].(string); ok {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern: %w", err)
			}
			rule.pattern = compiled
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// 检查是否已存在同名规则
	for i, r := range t.rules {
		if r.Name == rule.Name {
			t.rules[i] = rule
			return nil
		}
	}

	t.rules = append(t.rules, rule)
	return nil
}

// validateRule 验证规则
func (t *TransformerImpl) validateRule(rule TransformRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	if rule.SourceField == "" {
		return fmt.Errorf("source_field cannot be empty")
	}
	if rule.TargetField == "" {
		return fmt.Errorf("target_field cannot be empty")
	}
	if rule.Extractor == "" {
		return fmt.Errorf("extractor cannot be empty")
	}
	return nil
}

// RemoveRule 删除转换规则
func (t *TransformerImpl) RemoveRule(name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, rule := range t.rules {
		if rule.Name == name {
			t.rules = append(t.rules[:i], t.rules[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", name)
}

// ApplyRules 应用转换规则
func (t *TransformerImpl) ApplyRules(rules []TransformRule) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, rule := range rules {
		if err := t.validateRule(rule); err != nil {
			return fmt.Errorf("invalid rule %s: %w", rule.Name, err)
		}

		// 编译正则
		if rule.Extractor == "regex" && rule.Config != nil {
			if pattern, ok := rule.Config["pattern"].(string); ok {
				compiled, err := regexp.Compile(pattern)
				if err != nil {
					return fmt.Errorf("invalid regex pattern in rule %s: %w", rule.Name, err)
				}
				rule.pattern = compiled
			}
		}

		t.rules = append(t.rules, rule)
	}

	return nil
}

// Transform 转换日志
func (t *TransformerImpl) Transform(parsed *parser.ParsedLog, analysis *analyzer.AnalysisResult) (*TransformedLog, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := &TransformedLog{
		Timestamp:       parsed.Timestamp,
		Level:           parsed.Level,
		Message:         parsed.Message,
		Service:         parsed.Service,
		TraceID:         parsed.TraceID,
		SpanID:          parsed.SpanID,
		Fields:          make(map[string]interface{}),
		Raw:             parsed.Raw,
		Format:          string(parsed.Format),
		ExtractedFields: make(map[string]interface{}),
	}

	// 复制原始 Fields
	for k, v := range parsed.Fields {
		result.Fields[k] = v
	}

	// 如果有分析结果，合并实体和关键词
	if analysis != nil {
		for _, entity := range analysis.Entities {
			key := fmt.Sprintf("entity_%s_%d", entity.Type, len(result.ExtractedFields))
			result.ExtractedFields[key] = entity.Value
		}

		if len(analysis.Keywords) > 0 {
			result.ExtractedFields["keywords"] = analysis.Keywords
		}
	}

	// 应用转换规则
	for _, rule := range t.rules {
		if !rule.Enabled {
			continue
		}

		// 获取源字段值
		sourceValue := t.getSourceValue(parsed, analysis, rule.SourceField)
		if sourceValue == "" {
			continue
		}

		// 应用提取器
		extractorFunc, ok := t.funcs[rule.Extractor]
		if !ok {
			continue
		}

		extracted, err := extractorFunc(sourceValue, rule.Config)
		if err != nil {
			continue
		}

		// 设置目标字段
		result.ExtractedFields[rule.TargetField] = extracted
	}

	return result, nil
}

// getSourceValue 获取源字段值
func (t *TransformerImpl) getSourceValue(parsed *parser.ParsedLog, analysis *analyzer.AnalysisResult, field string) string {
	// 标准字段
	switch field {
	case "message":
		return parsed.Message
	case "raw":
		return parsed.Raw
	case "level":
		return parsed.Level
	case "service":
		return parsed.Service
	case "trace_id":
		return parsed.TraceID
	case "span_id":
		return parsed.SpanID
	}

	// 从 Fields 中获取
	if parsed.Fields != nil {
		if v, ok := parsed.Fields[field]; ok {
			return fmt.Sprintf("%v", v)
		}
	}

	// 从分析结果中获取
	if analysis != nil {
		switch field {
		case "sentiment_score":
			return fmt.Sprintf("%f", analysis.Sentiment.Score)
		case "sentiment_label":
			return analysis.Sentiment.Label
		case "language":
			return analysis.Language
		case "category":
			return analysis.Category
		}

		// 从实体中获取
		for _, entity := range analysis.Entities {
			if entity.Type == field {
				return entity.Value
			}
		}
	}

	return ""
}

// extractByRegex 使用正则提取
func extractByRegex(source string, config map[string]interface{}) (interface{}, error) {
	patternStr, ok := config["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern is required for regex extractor")
	}

	pattern := regexp.MustCompile(patternStr)
	matches := pattern.FindStringSubmatch(source)

	if len(matches) == 0 {
		return nil, nil
	}

	// 如果有命名捕获组，返回 map
	groupNames := pattern.SubexpNames()
	if len(groupNames) > 1 {
		result := make(map[string]interface{})
		for i, name := range groupNames {
			if i != 0 && name != "" {
				result[name] = matches[i]
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	// 返回第一个捕获组
	if len(matches) > 1 {
		return matches[1], nil
	}

	return matches[0], nil
}

// extractByTemplate 使用模板提取
func extractByTemplate(source string, config map[string]interface{}) (interface{}, error) {
	template, ok := config["template"].(string)
	if !ok {
		return nil, fmt.Errorf("template is required for template extractor")
	}

	// 简单的模板替换
	result := template
	result = strings.ReplaceAll(result, "{{source}}", source)

	return result, nil
}

// extractDirect 直接提取
func extractDirect(source string, config map[string]interface{}) (interface{}, error) {
	return source, nil
}

// extractLowercase 转换为小写
func extractLowercase(source string, config map[string]interface{}) (interface{}, error) {
	return strings.ToLower(source), nil
}

// extractUppercase 转换为大写
func extractUppercase(source string, config map[string]interface{}) (interface{}, error) {
	return strings.ToUpper(source), nil
}

// extractSplit 分割字符串
func extractSplit(source string, config map[string]interface{}) (interface{}, error) {
	delimiter, ok := config["delimiter"].(string)
	if !ok {
		delimiter = ","
	}

	parts := strings.Split(source, delimiter)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result, nil
}

// RuleConfig 规则配置
type RuleConfig struct {
	Rules     []TransformRule `json:"rules"`
	Version   string          `json:"version,omitempty"`
	UpdatedAt time.Time       `json:"updated_at,omitempty"`
}

// LoadRulesFromJSON 从 JSON 加载规则
func (t *TransformerImpl) LoadRulesFromJSON(jsonData []byte) error {
	var config RuleConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return fmt.Errorf("failed to parse rule config: %w", err)
	}

	return t.ApplyRules(config.Rules)
}

// ExportRules 导出规则
func (t *TransformerImpl) ExportRules() ([]TransformRule, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 返回规则副本
	rules := make([]TransformRule, len(t.rules))
	copy(rules, t.rules)

	return rules, nil
}
