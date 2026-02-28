// Package filter 提供复合条件过滤功能
package filter

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/log-system/log-processor/pkg/config"
)

// CompositeFilter 复合条件过滤器
type CompositeFilter struct {
	mu          sync.RWMutex
	filters     map[string]*config.FilterConfig
	conditions  []CompositeCondition
}

// CompositeCondition 复合条件
type CompositeCondition struct {
	Name       string
	Conditions []SingleCondition
	Operator   LogicalOperator // AND 或 OR
	Action     config.FilterAction
}

// SingleCondition 单个条件
type SingleCondition struct {
	Field    string
	Operator string // eq, ne, contains, regex, gt, lt
	Value    string
}

// LogicalOperator 逻辑运算符
type LogicalOperator string

const (
	OpAnd LogicalOperator = "AND"
	OpOr  LogicalOperator = "OR"
)

// NewCompositeFilter 创建复合条件过滤器
func NewCompositeFilter() *CompositeFilter {
	return &CompositeFilter{
		filters:    make(map[string]*config.FilterConfig),
		conditions: make([]CompositeCondition, 0),
	}
}

// AddCondition 添加复合条件
func (f *CompositeFilter) AddCondition(cond CompositeCondition) error {
	// 验证条件
	if err := f.validateCondition(cond); err != nil {
		return fmt.Errorf("invalid condition: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.conditions = append(f.conditions, cond)
	return nil
}

// validateCondition 验证条件
func (f *CompositeFilter) validateCondition(cond CompositeCondition) error {
	if cond.Name == "" {
		return fmt.Errorf("condition name cannot be empty")
	}

	if len(cond.Conditions) == 0 {
		return fmt.Errorf("condition must have at least one single condition")
	}

	validOperators := map[string]bool{
		"eq":       true,
		"ne":       true,
		"contains": true,
		"regex":    true,
		"gt":       true,
		"lt":       true,
		"ge":       true,
		"le":       true,
	}

	for _, sc := range cond.Conditions {
		if !validOperators[sc.Operator] {
			return fmt.Errorf("invalid operator: %s", sc.Operator)
		}
	}

	return nil
}

// Evaluate 评估复合条件
func (f *CompositeFilter) Evaluate(entry *ParsedLog) FilterResult {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := FilterResult{
		ShouldKeep: true,
		Action:     config.ActionAllow,
		Metadata:   make(map[string]interface{}),
	}

	for _, cond := range f.conditions {
		if f.evaluateCondition(cond, entry) {
			result.MatchedRule = cond.Name
			result.Action = cond.Action

			switch cond.Action {
			case config.ActionDrop:
				result.ShouldKeep = false
				return result
			case config.ActionMark:
				result.Metadata["marked_by"] = cond.Name
				result.Metadata["mark_time"] = time.Now().Unix()
			}
		}
	}

	return result
}

// evaluateCondition 评估单个复合条件
func (f *CompositeFilter) evaluateCondition(cond CompositeCondition, entry *ParsedLog) bool {
	if len(cond.Conditions) == 0 {
		return false
	}

	results := make([]bool, 0, len(cond.Conditions))
	for _, sc := range cond.Conditions {
		results = append(results, f.evaluateSingleCondition(sc, entry))
	}

	// 根据逻辑运算符组合结果
	switch cond.Operator {
	case OpAnd:
		for _, r := range results {
			if !r {
				return false
			}
		}
		return true
	case OpOr:
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// evaluateSingleCondition 评估单个条件
func (f *CompositeFilter) evaluateSingleCondition(cond SingleCondition, entry *ParsedLog) bool {
	value := f.getFieldValue(entry, cond.Field)

	switch cond.Operator {
	case "eq":
		return value == cond.Value
	case "ne":
		return value != cond.Value
	case "contains":
		return strings.Contains(value, cond.Value)
	case "regex":
		compiled := f.getCompiledRegex(cond.Value)
		if compiled == nil {
			return false
		}
		return compiled.MatchString(value)
	case "gt":
		return value > cond.Value
	case "lt":
		return value < cond.Value
	case "ge":
		return value >= cond.Value
	case "le":
		return value <= cond.Value
	default:
		return false
	}
}

// getFieldValue 获取字段值
func (f *CompositeFilter) getFieldValue(entry *ParsedLog, field string) string {
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
		if entry.Fields != nil {
			if v, ok := entry.Fields[field]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
		return ""
	}
}

// getCompiledRegex 获取编译后的正则表达式（复用 FilterEngineImpl 的方法）
func (f *CompositeFilter) getCompiledRegex(pattern string) *regexp.Regexp {
	// 这里应该使用共享的正则缓存
	// 为简化实现，直接编译
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	return compiled
}
