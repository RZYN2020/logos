// Package handlers 分析处理器
package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/log-system/log-analyzer/internal/analysis"
)

// AnalysisHandler 分析处理器
type AnalysisHandler struct {
	miner *analysis.PatternMiner
}

// NewAnalysisHandler 创建分析处理器
func NewAnalysisHandler() *AnalysisHandler {
	return &AnalysisHandler{
		miner: analysis.NewPatternMiner(),
	}
}

// MiningRequest 挖掘请求
type MiningRequest struct {
	Logs      []analysis.LogEntry `json:"logs" binding:"required"`
	TimeRange struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"time_range"`
}

// MinePatterns 挖掘日志模式
func (h *AnalysisHandler) MinePatterns(c *gin.Context) {
	var req MiningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patterns := h.miner.AnalyzePatterns(req.Logs)

	c.JSON(http.StatusOK, gin.H{
		"patterns": patterns,
		"total":    len(patterns),
	})
}

// DetectAnomaliesRequest 异常检测请求
type DetectAnomaliesRequest struct {
	CurrentLogs  []analysis.LogEntry `json:"current_logs" binding:"required"`
	BaselineLogs []analysis.LogEntry `json:"baseline_logs"`
}

// DetectAnomalies 检测异常日志
func (h *AnalysisHandler) DetectAnomalies(c *gin.Context) {
	var req DetectAnomaliesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	anomalies := h.miner.DetectAnomalies(req.CurrentLogs, req.BaselineLogs)

	c.JSON(http.StatusOK, gin.H{
		"anomalies": anomalies,
		"total":     len(anomalies),
	})
}

// ClusterLogsRequest 聚类请求
type ClusterLogsRequest struct {
	Logs      []analysis.LogEntry `json:"logs" binding:"required"`
	Threshold float64             `json:"threshold"`
}

// ClusterLogs 聚类日志
func (h *AnalysisHandler) ClusterLogs(c *gin.Context) {
	var req ClusterLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	threshold := req.Threshold
	if threshold == 0 {
		threshold = 0.8
	}

	clusters := h.miner.ClusterLogs(req.Logs, threshold)

	c.JSON(http.StatusOK, gin.H{
		"clusters": clusters,
		"total":    len(clusters),
	})
}

// RecommendRulesRequest 规则推荐请求
type RecommendRulesRequest struct {
	Logs     []analysis.LogEntry `json:"logs" binding:"required"`
	MinFreq  int                 `json:"min_frequency"`
}

// RecommendRules 推荐规则
func (h *AnalysisHandler) RecommendRules(c *gin.Context) {
	var req RecommendRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	minFreq := req.MinFreq
	if minFreq == 0 {
		minFreq = 10
	}

	patterns := h.miner.AnalyzePatterns(req.Logs)

	// 过滤低频模式
	var filteredPatterns []analysis.LogPattern
	for _, p := range patterns {
		if p.Frequency >= minFreq {
			filteredPatterns = append(filteredPatterns, p)
		}
	}

	recommendations := h.miner.RecommendRules(filteredPatterns)

	// 按优先级排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority < recommendations[j].Priority
	})

	c.JSON(http.StatusOK, gin.H{
		"recommendations": recommendations,
		"total":           len(recommendations),
	})
}

// GetPatternTypes 获取模式类型
func (h *AnalysisHandler) GetPatternTypes(c *gin.Context) {
	patternTypes := []gin.H{
		{
			"name":        "error_spike",
			"description": "错误日志突然增加",
			"severity":    "high",
		},
		{
			"name":        "new_pattern",
			"description": "出现新的日志模式",
			"severity":    "medium",
		},
		{
			"name":        "frequency_change",
			"description": "日志频率显著变化",
			"severity":    "medium",
		},
		{
			"name":        "slow_response",
			"description": "响应时间变慢",
			"severity":    "high",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"pattern_types": patternTypes,
	})
}
