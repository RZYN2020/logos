// Package models 数据模型定义
package models

import (
	"time"

	"github.com/google/uuid"
	unifiedRule "github.com/log-system/logos/pkg/rule"
)

// Rule 规则配置模型
type Rule struct {
	ID          string                 `json:"id" gorm:"primaryKey"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	Conditions  []Condition            `json:"conditions" gorm:"foreignKey:RuleID"`
	Actions     []Action               `json:"actions" gorm:"foreignKey:RuleID"`
	Version     int                    `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Condition 规则条件
type Condition struct {
	ID       string      `json:"id" gorm:"primaryKey"`
	RuleID   string      `json:"rule_id" gorm:"index"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value" gorm:"type:jsonb"`
}

// Action 规则动作
type Action struct {
	ID     string                 `json:"id" gorm:"primaryKey"`
	RuleID string                 `json:"rule_id" gorm:"index"`
	Type   string                 `json:"type"` // filter/drop/transform
	Config map[string]interface{} `json:"config,omitempty" gorm:"type:jsonb"`
}

// RuleVersion 规则版本历史
type RuleVersion struct {
	ID        string                 `json:"id" gorm:"primaryKey"`
	RuleID    string                 `json:"rule_id" gorm:"index"`
	Version   int                    `json:"version"`
	Content   map[string]interface{} `json:"content" gorm:"type:jsonb"`
	Author    string                 `json:"author"`
	Comment   string                 `json:"comment,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
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
	ID        string                 `json:"id" gorm:"primaryKey"`
	StrategyID string                `json:"strategy_id" gorm:"index"`
	Condition map[string]interface{} `json:"condition" gorm:"type:jsonb"`
	Action    map[string]interface{} `json:"action" gorm:"type:jsonb"`
}

// LogPattern 日志模式 (用于日志挖掘结果)
type LogPattern struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Pattern     string    `json:"pattern"`
	Description string    `json:"description,omitempty"`
	Frequency   int       `json:"frequency"`
	Severity    string    `json:"severity"` // high/medium/low
	Examples    []string  `json:"examples" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"created_at"`
}

// LogCluster 日志聚类
type LogCluster struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Center     string    `json:"center"`
	Size       int       `json:"size"`
	Similarity float64   `json:"similarity"`
	Members    []string  `json:"members" gorm:"type:jsonb"`
	CreatedAt  time.Time `json:"created_at"`
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
			Config: act.Config,
		})
	}

	return &unifiedRule.Rule{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Enabled:     r.Enabled,
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
	r.Version++
	r.UpdatedAt = time.Now()

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
			Config: act.Config,
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
