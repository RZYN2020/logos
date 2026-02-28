// Package models 数据模型定义
package models

import (
	"time"
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
