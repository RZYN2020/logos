// Package config 提供过滤配置的结构化定义
package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// FilterConfig 过滤配置
type FilterConfig struct {
	ID          string       `json:"id"`
	Enabled     bool         `json:"enabled"`
	Priority    int          `json:"priority"`
	Service     string       `json:"service,omitempty"`
	Environment string       `json:"environment,omitempty"`
	Rules       []FilterRule `json:"rules"`
	UpdatedAt   time.Time    `json:"updated_at,omitempty"`
}

// FilterRule 过滤规则
type FilterRule struct {
	Name    string       `json:"name"`
	Field   string       `json:"field"`             // 要匹配的字段 (message, raw, level, etc.)
	Pattern string       `json:"pattern"`           // 正则表达式模式
	Action  FilterAction `json:"action"`            // 匹配后的动作
	compiled *regexp.Regexp                        // 编译后的正则表达式
}

// FilterAction 过滤动作
type FilterAction int

const (
	// ActionAllow 允许通过
	ActionAllow FilterAction = iota
	// ActionDrop 丢弃日志
	ActionDrop
	// ActionMark 标记日志（添加元数据）
	ActionMark
)

// String 返回动作的字符串表示
func (a FilterAction) String() string {
	switch a {
	case ActionAllow:
		return "allow"
	case ActionDrop:
		return "drop"
	case ActionMark:
		return "mark"
	default:
		return "unknown"
	}
}

// MarshalJSON 自定义 JSON 序列化
func (a FilterAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON 自定义 JSON 反序列化
func (a *FilterAction) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "allow":
		*a = ActionAllow
	case "drop":
		*a = ActionDrop
	case "mark":
		*a = ActionMark
	default:
		return fmt.Errorf("unknown filter action: %s", s)
	}

	return nil
}

// Compile 编译正则表达式
func (r *FilterRule) Compile() error {
	compiled, err := regexp.Compile(r.Pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern '%s': %w", r.Pattern, err)
	}
	r.compiled = compiled
	return nil
}

// Match 检查日志字段是否匹配规则
func (r *FilterRule) Match(value string) bool {
	if r.compiled == nil {
		if err := r.Compile(); err != nil {
			return false
		}
	}
	return r.compiled.MatchString(value)
}

// FilterResult 过滤结果
type FilterResult struct {
	ShouldKeep  bool                 `json:"should_keep"`
	Action      FilterAction         `json:"action"`
	MatchedRule string               `json:"matched_rule,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FilterEvent 配置变更事件
type FilterEvent struct {
	Type   EventType
	Config *FilterConfig
}

// ParserConfig 解析器配置
type ParserConfig struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// TransformRule 转换规则
type TransformRule struct {
	SourceField string            `json:"source_field"`
	TargetField string            `json:"target_field"`
	Extractor   string            `json:"extractor"` // regex, template, jsonpath
	Config      map[string]interface{} `json:"config,omitempty"`
}

// ProcessorConfig 处理器完整配置
type ProcessorConfig struct {
	Filters      []FilterConfig  `json:"filters,omitempty"`
	Parsers      []ParserConfig  `json:"parsers,omitempty"`
	Transforms   []TransformRule `json:"transform_rules,omitempty"`
	UpdatedAt    time.Time       `json:"updated_at,omitempty"`
	Version      string          `json:"version,omitempty"`
}

// ETCD 键路径常量
const (
	FilterConfigPrefix   = "/log-processor/filters/"
	ParserConfigPrefix   = "/log-processor/parsers/"
	TransformConfigPrefix = "/log-processor/transforms/"
	ProcessorConfigKey   = "/log-processor/config"
)
