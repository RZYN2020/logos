// Package analysis 日志分析算法
package analysis

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Service   string
	Message   string
	TraceID   string
	UserID    string
	Fields    map[string]interface{}
}

// LogPattern 日志模式
type LogPattern struct {
	ID          string
	Pattern     string
	Description string
	Frequency   int
	Severity    string // high/medium/low
	Examples    []string
	CreatedAt   time.Time
}

// LogCluster 日志聚类
type LogCluster struct {
	ID         string
	Center     string
	Size       int
	Similarity float64
	Members    []string
	CreatedAt  time.Time
}

// PatternMiner 模式挖掘器
type PatternMiner struct {
	patterns map[string]*LogPattern
}

// NewPatternMiner 创建模式挖掘器
func NewPatternMiner() *PatternMiner {
	return &PatternMiner{
		patterns: make(map[string]*LogPattern),
	}
}

// ExtractPattern 从日志消息中提取模式
func (pm *PatternMiner) ExtractPattern(message string) string {
	// 替换变量部分为占位符
	pattern := message

	// 替换数字
	pattern = replacePattern(pattern, `\d+`, "{num}")

	// 替换 UUID
	pattern = replacePattern(pattern, `[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`, "{uuid}")

	// 替换 IP 地址
	pattern = replacePattern(pattern, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "{ip}")

	// 替换时间戳
	pattern = replacePattern(pattern, `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`, "{timestamp}")

	// 替换 hex 字符串
	pattern = replacePattern(pattern, `[a-f0-9]{8,}`, "{hex}")

	return pattern
}

// AnalyzePatterns 分析日志模式
func (pm *PatternMiner) AnalyzePatterns(entries []LogEntry) []LogPattern {
	patternCount := make(map[string]int)
	patternExamples := make(map[string][]string)

	for _, entry := range entries {
		pattern := pm.ExtractPattern(entry.Message)
		patternCount[pattern]++
		if len(patternExamples[pattern]) < 3 {
			patternExamples[pattern] = append(patternExamples[pattern], entry.Message)
		}
	}

	var patterns []LogPattern
	for pattern, count := range patternCount {
		severity := "low"
		if count > 100 {
			severity = "high"
		} else if count > 50 {
			severity = "medium"
		}

		hash := sha256.Sum256([]byte(pattern))
		patterns = append(patterns, LogPattern{
			ID:          fmt.Sprintf("%x", hash[:8]),
			Pattern:     pattern,
			Frequency:   count,
			Severity:    severity,
			Examples:    patternExamples[pattern],
			CreatedAt:   time.Now(),
		})
	}

	// 按频率排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	return patterns
}

// DetectAnomalies 检测异常日志
func (pm *PatternMiner) DetectAnomalies(entries []LogEntry, baseline []LogEntry) []LogEntry {
	baselinePatterns := make(map[string]bool)
	for _, entry := range baseline {
		pattern := pm.ExtractPattern(entry.Message)
		baselinePatterns[pattern] = true
	}

	var anomalies []LogEntry
	for _, entry := range entries {
		pattern := pm.ExtractPattern(entry.Message)
		if !baselinePatterns[pattern] {
			anomalies = append(anomalies, entry)
		}
	}

	return anomalies
}

// ClusterLogs 聚类日志
func (pm *PatternMiner) ClusterLogs(entries []LogEntry, threshold float64) []LogCluster {
	patternMap := make(map[string][]string)

	// 按模式分组
	for _, entry := range entries {
		pattern := pm.ExtractPattern(entry.Message)
		patternMap[pattern] = append(patternMap[pattern], entry.Message)
	}

	var clusters []LogCluster
	for pattern, members := range patternMap {
		if len(members) >= 2 {
			hash := sha256.Sum256([]byte(pattern))
			clusters = append(clusters, LogCluster{
				ID:         fmt.Sprintf("%x", hash[:8]),
				Center:     pattern,
				Size:       len(members),
				Similarity: 1.0,
				Members:    members,
				CreatedAt:  time.Now(),
			})
		}
	}

	// 按大小排序
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Size > clusters[j].Size
	})

	return clusters
}

// RecommendRules 基于模式推荐规则
func (pm *PatternMiner) RecommendRules(patterns []LogPattern) []RuleRecommendation {
	var recommendations []RuleRecommendation

	for _, pattern := range patterns {
		if pattern.Severity == "high" || pattern.Severity == "medium" {
			rec := RuleRecommendation{
				PatternID:   pattern.ID,
				Name:        fmt.Sprintf("Auto-generated rule for: %s", truncateString(pattern.Pattern, 50)),
				Description: fmt.Sprintf("Automatically generated from pattern with frequency %d", pattern.Frequency),
				Priority:    getPriorityFromSeverity(pattern.Severity),
				Conditions: []ConditionRecommendation{
					{
						Field:    "message",
						Operator: "contains",
						Value:    extractKeywords(pattern.Pattern),
					},
				},
				Action: ActionRecommendation{
					Type: "filter",
					Config: map[string]interface{}{
						"sampling": 1.0,
						"priority": getPriorityFromSeverity(pattern.Severity),
					},
				},
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// RuleRecommendation 规则推荐
type RuleRecommendation struct {
	PatternID   string
	Name        string
	Description string
	Priority    int
	Conditions  []ConditionRecommendation
	Action      ActionRecommendation
}

// ConditionRecommendation 条件推荐
type ConditionRecommendation struct {
	Field    string
	Operator string
	Value    interface{}
}

// ActionRecommendation 动作推荐
type ActionRecommendation struct {
	Type   string
	Config map[string]interface{}
}

// 辅助函数
func replacePattern(s, pattern, replacement string) string {
	// 简单实现，实际项目中应使用 regexp
	return s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func getPriorityFromSeverity(severity string) int {
	switch severity {
	case "high":
		return 1
	case "medium":
		return 2
	default:
		return 3
	}
}

func extractKeywords(pattern string) string {
	// 提取关键词（移除占位符）
	keywords := strings.ReplaceAll(pattern, "{num}", "")
	keywords = strings.ReplaceAll(keywords, "{uuid}", "")
	keywords = strings.ReplaceAll(keywords, "{ip}", "")
	keywords = strings.ReplaceAll(keywords, "{timestamp}", "")
	keywords = strings.ReplaceAll(keywords, "{hex}", "")
	return strings.TrimSpace(keywords)
}
