// Package filter 提供日志过滤功能
package filter

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/log-system/log-processor/pkg/config"
)

// FilterEngine 过滤引擎接口
type FilterEngine interface {
	ApplyFilters(entry *ParsedLog) FilterResult
	AddFilter(cfg *config.FilterConfig) error
	RemoveFilter(id string) error
	ReloadFilters() error
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
