// Package analysis 日志分析算法
// 实现 Drain 算法用于日志模式解析
package analysis

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
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

// DrainNode Drain 算法树节点
type DrainNode struct {
	Children   map[string]*DrainNode // 子节点映射
	PatternIDs []string              // 此节点的模板 ID 列表
	Depth      int                   // 节点深度
	Template   string                // 日志模板
}

// DrainTree Drain 算法树
type DrainTree struct {
	root       *DrainNode
	templates  map[string]*LogPattern // 模板 ID -> LogPattern
	maxDepth   int                    // 树最大深度
	similarity float64                // 相似度阈值
	mu         sync.RWMutex
}

// NewDrainTree 创建 Drain 树
func NewDrainTree(maxDepth int, similarity float64) *DrainTree {
	return &DrainTree{
		root:       &DrainNode{Children: make(map[string]*DrainNode)},
		templates:  make(map[string]*LogPattern),
		maxDepth:   maxDepth,
		similarity: similarity,
	}
}

// AddLog 添加日志并返回匹配的模式
func (dt *DrainTree) AddLog(message string) string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 提取令牌
	tokens := tokenize(message)
	if len(tokens) == 0 {
		return message
	}

	// 在树中查找最佳匹配
	node := dt.findBestNode(tokens)
	if node == nil {
		// 创建新模板
		pattern := dt.extractPattern(tokens)
		patternID := dt.createTemplate(pattern, message)
		dt.insertToTree(tokens, patternID, len(tokens))
		return pattern
	}

	// 更新现有模板
	dt.templates[node.PatternIDs[0]].Frequency++
	if len(dt.templates[node.PatternIDs[0]].Examples) < 3 {
		dt.templates[node.PatternIDs[0]].Examples = append(
			dt.templates[node.PatternIDs[0]].Examples,
			message,
		)
	}
	return dt.templates[node.PatternIDs[0]].Pattern
}

// tokenize 将消息分割为令牌
func tokenize(message string) []string {
	// 使用空格和特殊字符分割
	re := regexp.MustCompile(`[\s,;:=\[\]{}()]+`)
	parts := re.Split(message, -1)
	var tokens []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			tokens = append(tokens, strings.TrimSpace(p))
		}
	}
	return tokens
}

// extractPattern 从令牌提取模式
func (dt *DrainTree) extractPattern(tokens []string) string {
	var pattern []string
	for _, token := range tokens {
		if isVariable(token) {
			pattern = append(pattern, "<*>")
		} else {
			pattern = append(pattern, token)
		}
	}
	return strings.Join(pattern, " ")
}

// 预编译正则表达式（性能优化）
var (
	regexDigit    = regexp.MustCompile(`^\d+$`)
	regexUUID     = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	regexIP       = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	regexTimestamp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	regexHex      = regexp.MustCompile(`^[a-f0-9]{8,}$`)
)

// isVariable 判断是否为变量
func isVariable(token string) bool {
	// 数字
	if regexDigit.MatchString(token) {
		return true
	}
	// UUID
	if regexUUID.MatchString(token) {
		return true
	}
	// IP 地址
	if regexIP.MatchString(token) {
		return true
	}
	// 时间戳
	if regexTimestamp.MatchString(token) {
		return true
	}
	// Hex 字符串
	if regexHex.MatchString(token) {
		return true
	}
	return false
}

// createTemplate 创建新模板
func (dt *DrainTree) createTemplate(pattern, example string) string {
	hash := sha256.Sum256([]byte(pattern))
	patternID := fmt.Sprintf("%x", hash[:8])

	dt.templates[patternID] = &LogPattern{
		ID:        patternID,
		Pattern:   pattern,
		Frequency: 1,
		Examples:  []string{example},
		CreatedAt: time.Now(),
	}
	return patternID
}

// insertToTree 插入到树中
func (dt *DrainTree) insertToTree(tokens []string, patternID string, depth int) {
	node := dt.root
	for i, token := range tokens {
		if i >= dt.maxDepth {
			break
		}
		key := token
		if isVariable(token) {
			key = "*"
		}
		if node.Children[key] == nil {
			node.Children[key] = &DrainNode{
				Children:   make(map[string]*DrainNode),
				Depth:      i + 1,
				PatternIDs: []string{},
			}
		}
		node = node.Children[key]
	}
	node.PatternIDs = append(node.PatternIDs, patternID)
}

// findBestNode 查找最佳匹配节点
func (dt *DrainTree) findBestNode(tokens []string) *DrainNode {
	node := dt.root
	for i, token := range tokens {
		if i >= dt.maxDepth {
			break
		}
		key := token
		if isVariable(token) {
			key = "*"
		}
		if next := node.Children[key]; next != nil {
			node = next
		} else if wildcard := node.Children["*"]; wildcard != nil {
			node = wildcard
		} else {
			return nil
		}
	}
	if len(node.PatternIDs) > 0 {
		return node
	}
	return nil
}

// GetPatterns 获取所有模式
func (dt *DrainTree) GetPatterns() []LogPattern {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	patterns := make([]LogPattern, 0, len(dt.templates))
	for _, p := range dt.templates {
		patterns = append(patterns, *p)
	}

	// 按频率排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	return patterns
}

// PatternMiner 模式挖掘器（使用 Drain 算法）
type PatternMiner struct {
	tree *DrainTree
	mu   sync.RWMutex
}

// NewPatternMiner 创建模式挖掘器
func NewPatternMiner() *PatternMiner {
	return &PatternMiner{
		tree: NewDrainTree(4, 0.8), // 默认最大深度 4，相似度 0.8
	}
}

// ExtractPattern 从日志消息中提取模式（使用 Drain 算法）
func (pm *PatternMiner) ExtractPattern(message string) string {
	return pm.tree.AddLog(message)
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
	// 提取关键词（移除 Drain 算法的 <*> 占位符）
	keywords := strings.ReplaceAll(pattern, "<*>", "")
	return strings.TrimSpace(keywords)
}
