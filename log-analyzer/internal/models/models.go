// Package models 数据模型定义
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	unifiedRule "github.com/log-system/logos/pkg/rule"
)

// JSONMap 自定义类型，用于在 SQLite 中存储 JSON 数据
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 实现 sql.Scanner 接口
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	data, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(data, m)
}

// Rule 规则配置模型
type Rule struct {
	ID          string      `json:"id" gorm:"primaryKey"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Enabled     bool        `json:"enabled" gorm:"default:true"`
	Priority    int         `json:"priority" gorm:"default:0"`
	Service     string      `json:"service,omitempty" gorm:"index"`
	Component   string      `json:"component,omitempty" gorm:"index"` // sdk or processor
	Conditions  []Condition `json:"conditions" gorm:"foreignKey:RuleID"`
	Actions     []Action    `json:"actions" gorm:"foreignKey:RuleID"`
	Version     int         `json:"version"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Condition 规则条件
type Condition struct {
	ID       string      `json:"id" gorm:"primaryKey"`
	RuleID   string      `json:"rule_id" gorm:"index"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value" gorm:"type:text"`
}

// Action 规则动作
type Action struct {
	ID     string  `json:"id" gorm:"primaryKey"`
	RuleID string  `json:"rule_id" gorm:"index"`
	Type   string  `json:"type"` // filter/drop/transform
	Config JSONMap `json:"config,omitempty" gorm:"type:text"`
}

// RuleVersion 规则版本历史
type RuleVersion struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	RuleID    string    `json:"rule_id" gorm:"index"`
	Version   int       `json:"version"`
	Content   JSONMap   `json:"content" gorm:"type:text"`
	Author    string    `json:"author"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Strategy 策略模型 (用于策略组管理)
type Strategy struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Rules       []StrategyRule `json:"rules" gorm:"foreignKey:StrategyID"`
	Version     int            `json:"version"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Author      string         `json:"author"`
}

// StrategyRule 策略规则
type StrategyRule struct {
	ID         string  `json:"id" gorm:"primaryKey"`
	StrategyID string  `json:"strategy_id" gorm:"index"`
	Condition  JSONMap `json:"condition" gorm:"type:text"`
	Action     JSONMap `json:"action" gorm:"type:text"`
}

// LogPattern 日志模式 (用于日志挖掘结果)
type LogPattern struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Pattern     string    `json:"pattern"`
	Description string    `json:"description,omitempty"`
	Frequency   int       `json:"frequency"`
	Severity    string    `json:"severity"` // high/medium/low
	Examples    []string  `json:"examples" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
}

// LogCluster 日志聚类
type LogCluster struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Center     string    `json:"center"`
	Size       int       `json:"size"`
	Similarity float64   `json:"similarity"`
	Members    []string  `json:"members" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

// LogEntry 日志存储模型
type LogEntry struct {
	ID         uint      `gorm:"primaryKey"`
	Service    string    `gorm:"index;not null"`
	Component  string    `gorm:"index"`
	Timestamp  time.Time `gorm:"index"`
	Level      string    `gorm:"index"`
	Message    string    `gorm:"type:text"`
	Path       string    `gorm:"index"` // 日志所在文件路径
	Function   string    `gorm:"index"` // 日志所在函数
	LineNumber int       `gorm:"index"` // 日志行号
	TraceID    string    `gorm:"index"`
	UserID     string
	Fields     JSONMap `gorm:"type:text"` // 额外字段
	CreatedAt  time.Time
}

// LogReport 日志报告模型
type LogReport struct {
	Service     string    `gorm:"primaryKey"`
	Component   string    `gorm:"primaryKey"`
	From        time.Time `gorm:"index"`
	To          time.Time
	TotalLogs   int
	TopLines    JSONMap `gorm:"type:text"` // 存储 TOP 行号统计
	TopPatterns JSONMap `gorm:"type:text"` // 存储 TOP 模式统计
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ToUnifiedRule 将数据库规则转换为统一规则模型
func (r *Rule) ToUnifiedRule() *unifiedRule.Rule {
	// 构建复合条件
	var conditions []unifiedRule.Condition
	for _, cond := range r.Conditions {
		conditions = append(conditions, unifiedRule.Condition{
			Field:    cond.Field,
			Operator: cond.Operator,
			Value:    cond.Value,
		})
	}

	// 构建复合条件（使用 all 连接所有条件）
	var compositeCondition unifiedRule.Condition
	if len(conditions) == 1 {
		compositeCondition = conditions[0]
	} else if len(conditions) > 1 {
		compositeCondition = unifiedRule.Condition{
			All: conditions,
		}
	}

	// 构建动作
	var actions []unifiedRule.ActionDef
	for _, act := range r.Actions {
		actions = append(actions, unifiedRule.ActionDef{
			Type:   act.Type,
			Config: map[string]interface{}(act.Config),
		})
	}

	return &unifiedRule.Rule{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Enabled:     r.Enabled,
		Priority:    r.Priority,
		Service:     r.Service,
		Component:   r.Component,
		Condition:   compositeCondition,
		Actions:     actions,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// FromUnifiedRule 从统一规则模型转换为数据库规则
func (r *Rule) FromUnifiedRule(ur *unifiedRule.Rule) {
	r.ID = ur.ID
	r.Name = ur.Name
	r.Description = ur.Description
	r.Enabled = ur.Enabled
	r.Priority = ur.Priority
	r.Service = ur.Service
	r.Component = ur.Component
	r.Version++
	r.UpdatedAt = time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}

	// 清空旧的条件和动作
	r.Conditions = nil
	r.Actions = nil

	// 展平复合条件为简单条件列表
	var flatConditions []unifiedRule.Condition
	flattenCondition(ur.Condition, &flatConditions)

	// 转换为数据库条件
	for _, cond := range flatConditions {
		r.Conditions = append(r.Conditions, Condition{
			ID:       uuid.New().String(),
			RuleID:   ur.ID,
			Field:    cond.Field,
			Operator: cond.Operator,
			Value:    cond.Value,
		})
	}

	// 转换为数据库动作
	for _, act := range ur.Actions {
		r.Actions = append(r.Actions, Action{
			ID:     uuid.New().String(),
			RuleID: ur.ID,
			Type:   act.Type,
			Config: JSONMap(act.Config),
		})
	}
}

// flattenCondition 递归展平复合条件
func flattenCondition(cond unifiedRule.Condition, result *[]unifiedRule.Condition) {
	if cond.IsSingle() {
		*result = append(*result, cond)
		return
	}

	// 处理 all 条件
	for _, child := range cond.All {
		flattenCondition(child, result)
	}

	// 处理 any 条件（转换为多个规则或使用标记）
	for _, child := range cond.Any {
		flattenCondition(child, result)
	}

	// 处理 not 条件
	if cond.Not != nil {
		flattenCondition(*cond.Not, result)
	}
}
