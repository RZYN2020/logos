// Package filter 提供日志过滤功能，使用统一规则引擎
package filter

import (
	"github.com/log-system/log-processor/pkg/rule"
	unifiedRule "github.com/log-system/logos/pkg/rule"
)

// FilterEngine 过滤引擎接口
type FilterEngine interface {
	ApplyFilters(entry *ParsedLog) FilterResult
	Close() error
}

// ParsedLog 解析后的日志
type ParsedLog struct {
	Timestamp interface{} // time.Time
	Level     string
	Message   string
	Service   string
	TraceID   string
	SpanID    string
	Fields    map[string]interface{}
	Raw       string
}

// FilterResult 过滤结果
type FilterResult struct {
	ShouldKeep  bool
	Action      string
	MatchedRule string
	Metadata    map[string]interface{}
}

// RuleFilterEngine 基于统一规则引擎的过滤器
type RuleFilterEngine struct {
	engine *rule.Engine
}

// NewRuleFilterEngine 创建规则过滤引擎
func NewRuleFilterEngine(engine *rule.Engine) *RuleFilterEngine {
	return &RuleFilterEngine{
		engine: engine,
	}
}

// ApplyFilters 应用规则过滤
func (e *RuleFilterEngine) ApplyFilters(entry *ParsedLog) FilterResult {
	result := FilterResult{
		ShouldKeep: true,
		Action:     "allow",
		Metadata:   make(map[string]interface{}),
	}

	if e.engine == nil {
		return result
	}

	// 转换为统一规则引擎的 LogEntry
	entryData := make(map[string]interface{})
	entryData["level"] = entry.Level
	entryData["message"] = entry.Message
	entryData["service"] = entry.Service
	entryData["trace_id"] = entry.TraceID
	entryData["span_id"] = entry.SpanID
	entryData["raw"] = entry.Raw
	for k, v := range entry.Fields {
		entryData[k] = v
	}

	logEntry := unifiedRule.NewMapLogEntry(entryData)

	// 评估规则
	shouldKeep, results, _ := e.engine.Evaluate(logEntry)
	result.ShouldKeep = shouldKeep

	// 填充结果信息
	if len(results) > 0 {
		lastResult := results[len(results)-1]
		result.MatchedRule = lastResult.RuleName
		result.Metadata["rule_id"] = lastResult.RuleID
		result.Metadata["actions"] = lastResult.Actions
	}

	return result
}

// Close 关闭引擎
func (e *RuleFilterEngine) Close() error {
	if e.engine != nil {
		return e.engine.Close()
	}
	return nil
}
