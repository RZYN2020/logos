// Package filter 提供日志过滤功能，使用统一规则引擎
package filter

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/log-system/log-processor/pkg/config"
	"github.com/log-system/log-processor/pkg/rule"
	unifiedRule "github.com/log-system/logos/pkg/rule"
)

// FilterEngine 过滤引擎接口
type FilterEngine interface {
	ApplyFilters(entry *ParsedLog) FilterResult
	AddFilter(cfg *config.FilterConfig) error
	RemoveFilter(id string) error
	ReloadFilters() error
	Close() error
}

// ParsedLog 解析后的日志
type ParsedLog struct {
	Timestamp time.Time
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
	Action      config.FilterAction
	MatchedRule string
	Metadata    map[string]interface{}
}

// FilterEngineImpl 过滤引擎实现
type FilterEngineImpl struct {
	mu          sync.RWMutex
	filters     map[string]*config.FilterConfig
	regexCache  map[string]*regexp.Regexp // 编译后的正则缓存
	regexCacheMu sync.RWMutex
}

// NewFilterEngine 创建过滤引擎
func NewFilterEngine() *FilterEngineImpl {
	return &FilterEngineImpl{
		filters:    make(map[string]*config.FilterConfig),
		regexCache: make(map[string]*regexp.Regexp),
	}
}

// LoadFilters 从配置管理器加载过滤配置
func (e *FilterEngineImpl) LoadFilters(manager config.ConfigManager) error {
	filters, err := manager.LoadFilters()
	if err != nil {
		return fmt.Errorf("failed to load filters: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, filter := range filters {
		e.filters[filter.ID] = filter
	}

	log.Printf("Loaded %d filters", len(filters))
	return nil
}

// AddFilter 添加过滤配置
func (e *FilterEngineImpl) AddFilter(cfg *config.FilterConfig) error {
	if cfg == nil {
		return fmt.Errorf("filter config cannot be nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.filters[cfg.ID] = cfg
	log.Printf("Added filter: %s", cfg.ID)
	return nil
}

// RemoveFilter 删除过滤配置
func (e *FilterEngineImpl) RemoveFilter(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.filters, id)
	log.Printf("Removed filter: %s", id)
	return nil
}

// ReloadFilters 重新加载过滤配置
func (e *FilterEngineImpl) ReloadFilters(manager config.ConfigManager) error {
	// 清空现有配置
	e.mu.Lock()
	e.filters = make(map[string]*config.FilterConfig)
	e.mu.Unlock()

	// 重新加载
	return e.LoadFilters(manager)
}

// Close 关闭引擎
func (e *FilterEngineImpl) Close() error {
	return nil
}

// ApplyFilters 应用所有过滤规则
func (e *FilterEngineImpl) ApplyFilters(entry *ParsedLog) FilterResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := FilterResult{
		ShouldKeep: true,
		Action:     config.ActionAllow,
		Metadata:   make(map[string]interface{}),
	}

	// 收集所有激活的过滤器
	var activeFilters []*config.FilterConfig
	for _, filter := range e.filters {
		if filter.Enabled {
			activeFilters = append(activeFilters, filter)
		}
	}

	// 按优先级排序（已在配置管理器中排序）
	for _, filter := range activeFilters {
		// 检查服务过滤
		if filter.Service != "" && filter.Service != entry.Service {
			continue
		}

		// 应用规则
		for _, rule := range filter.Rules {
			if e.matchRule(rule, entry) {
				result.MatchedRule = rule.Name
				result.Action = rule.Action

				switch rule.Action {
				case config.ActionDrop:
					result.ShouldKeep = false
					return result

				case config.ActionMark:
					// 标记日志
					result.Metadata["marked_by"] = rule.Name
					result.Metadata["mark_time"] = time.Now().Unix()

				case config.ActionAllow:
					result.ShouldKeep = true
				}
			}
		}
	}

	return result
}

// matchRule 匹配单条规则
func (e *FilterEngineImpl) matchRule(rule config.FilterRule, entry *ParsedLog) bool {
	// 获取要匹配的值
	value := e.getFieldValue(entry, rule.Field)
	if value == "" {
		return false
	}

	// 使用缓存或编译正则
	compiled := e.getCompiledRegex(rule.Pattern)
	if compiled == nil {
		return false
	}

	return compiled.MatchString(value)
}

// getFieldValue 获取字段值
func (e *FilterEngineImpl) getFieldValue(entry *ParsedLog, field string) string {
	switch field {
	case "message":
		return entry.Message
	case "raw":
		return entry.Raw
	case "level":
		return entry.Level
	case "service":
		return entry.Service
	case "trace_id":
		return entry.TraceID
	case "span_id":
		return entry.SpanID
	default:
		// 从 Fields 中获取
		if entry.Fields != nil {
			if v, ok := entry.Fields[field]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
		return ""
	}
}

// getCompiledRegex 获取编译后的正则表达式
func (e *FilterEngineImpl) getCompiledRegex(pattern string) *regexp.Regexp {
	// 先尝试从缓存获取
	e.regexCacheMu.RLock()
	if compiled, ok := e.regexCache[pattern]; ok {
		e.regexCacheMu.RUnlock()
		return compiled
	}
	e.regexCacheMu.RUnlock()

	// 编译并缓存
	e.regexCacheMu.Lock()
	defer e.regexCacheMu.Unlock()

	// 双重检查
	if compiled, ok := e.regexCache[pattern]; ok {
		return compiled
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("Failed to compile regex pattern '%s': %v", pattern, err)
		return nil
	}

	e.regexCache[pattern] = compiled
	return compiled
}

// RegexFilter 正则表达式过滤器
type RegexFilter struct {
	config         *config.FilterConfig
	compiledRules  []*compiledRule
}

type compiledRule struct {
	name     string
	field    string
	pattern  *regexp.Regexp
	action   config.FilterAction
}

// NewRegexFilter 创建正则表达式过滤器
func NewRegexFilter(cfg *config.FilterConfig) (*RegexFilter, error) {
	filter := &RegexFilter{
		config:        cfg,
		compiledRules: make([]*compiledRule, 0, len(cfg.Rules)),
	}

	for _, rule := range cfg.Rules {
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %w", rule.Pattern, err)
		}

		filter.compiledRules = append(filter.compiledRules, &compiledRule{
			name:    rule.Name,
			field:   rule.Field,
			pattern: pattern,
			action:  rule.Action,
		})
	}

	return filter, nil
}

// Match 检查日志是否匹配过滤器
func (f *RegexFilter) Match(entry *ParsedLog) bool {
	for _, rule := range f.compiledRules {
		value := getFieldValue(entry, rule.field)
		if rule.pattern.MatchString(value) {
			return true
		}
	}
	return false
}

// getFieldValue 辅助函数
func getFieldValue(entry *ParsedLog, field string) string {
	switch field {
	case "message":
		return entry.Message
	case "raw":
		return entry.Raw
	case "level":
		return entry.Level
	case "service":
		return entry.Service
	default:
		if entry.Fields != nil {
			if v, ok := entry.Fields[field]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
		return ""
	}
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
		Action:     config.ActionAllow,
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

// AddFilter 添加过滤配置（不适用于规则引擎）
func (e *RuleFilterEngine) AddFilter(cfg *config.FilterConfig) error {
	return fmt.Errorf("AddFilter not supported for RuleFilterEngine")
}

// RemoveFilter 删除过滤配置（不适用于规则引擎）
func (e *RuleFilterEngine) RemoveFilter(id string) error {
	return fmt.Errorf("RemoveFilter not supported for RuleFilterEngine")
}

// ReloadFilters 重新加载过滤配置（不适用于规则引擎）
func (e *RuleFilterEngine) ReloadFilters() error {
	return fmt.Errorf("ReloadFilters not supported for RuleFilterEngine")
}

// Close 关闭引擎
func (e *RuleFilterEngine) Close() error {
	if e.engine != nil {
		return e.engine.Close()
	}
	return nil
}

// LegacyFilterEngine 传统过滤器引擎（空实现）
type LegacyFilterEngine struct{}

// ApplyFilters 空实现
func (e *LegacyFilterEngine) ApplyFilters(entry *ParsedLog) FilterResult {
	return FilterResult{
		ShouldKeep: true,
		Action:     config.ActionAllow,
		Metadata:   make(map[string]interface{}),
	}
}

// AddFilter 空实现
func (e *LegacyFilterEngine) AddFilter(cfg *config.FilterConfig) error {
	return nil
}

// RemoveFilter 空实现
func (e *LegacyFilterEngine) RemoveFilter(id string) error {
	return nil
}

// ReloadFilters 空实现
func (e *LegacyFilterEngine) ReloadFilters() error {
	return nil
}

// Close 空实现
func (e *LegacyFilterEngine) Close() error {
	return nil
}
