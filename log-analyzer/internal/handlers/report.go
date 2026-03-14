// Package handlers 报告处理器
package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/log-system/log-analyzer/internal/analysis"
	"github.com/log-system/log-analyzer/internal/models"
	"gorm.io/gorm"
)

// ReportHandler 报告处理器
type ReportHandler struct {
	db *gorm.DB
}

// NewReportHandler 创建报告处理器
func NewReportHandler(db *gorm.DB) *ReportHandler {
	return &ReportHandler{
		db: db,
	}
}

// GetTopLines 获取 TOP 日志行号统计
func (h *ReportHandler) GetTopLines(c *gin.Context) {
	service := c.Param("service")
	component := c.Query("component")
	from := c.Query("from")
	to := c.Query("to")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// 构建查询
	query := h.db.Model(&models.LogEntry{}).
		Select("path, function, line_number, COUNT(*) as count").
		Where("service = ?", service)

	if component != "" {
		query = query.Where("component = ?", component)
	}
	if from != "" {
		query = query.Where("timestamp >= ?", from)
	}
	if to != "" {
		query = query.Where("timestamp <= ?", to)
	}

	var results []struct {
		Path       string
		Function   string
		LineNumber int
		Count      int
	}

	query.Group("path, function, line_number").
		Order("count DESC").
		Limit(limit).
		Scan(&results)

	// 计算总数
	var total int64
	h.db.Model(&models.LogEntry{}).Where("service = ?", service).Count(&total)

	topLines := make([]gin.H, len(results))
	for i, r := range results {
		// 计算百分比（避免除零）
		percentage := 0.0
		if total > 0 {
			percentage = float64(r.Count) / float64(total) * 100
		}
		topLines[i] = gin.H{
			"line_number": r.LineNumber,
			"file":        r.Path,
			"function":    r.Function,
			"count":       r.Count,
			"percentage":  percentage,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"service":    service,
		"total_logs": total,
		"top_lines":  topLines,
	})
}

// GetTopPatterns 获取 TOP 日志模式统计
func (h *ReportHandler) GetTopPatterns(c *gin.Context) {
	service := c.Param("service")
	component := c.Query("component")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// 获取日志
	var logs []models.LogEntry
	query := h.db.Where("service = ?", service)
	if component != "" {
		query = query.Where("component = ?", component)
	}
	query.Find(&logs)

	// 使用 PatternMiner 进行模式识别
	miner := analysis.NewPatternMiner()
	entries := make([]analysis.LogEntry, len(logs))
	for i, log := range logs {
		entries[i] = analysis.LogEntry{
			Message:   log.Message,
			Timestamp: log.Timestamp,
			Level:     log.Level,
		}
	}

	patterns := miner.AnalyzePatterns(entries)

	// 排序并取 TOP N
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	// 计算总数
	var total int64
	h.db.Model(&models.LogEntry{}).Where("service = ?", service).Count(&total)

	topPatterns := make([]gin.H, len(patterns))
	for i, p := range patterns {
		// 获取示例日志（转义 LIKE 特殊字符）
		escapedPattern := escapeLikePattern(p.Pattern)
		var sampleLogs []models.LogEntry
		h.db.Where("service = ? AND message LIKE ?", service, "%"+escapedPattern+"%").
			Limit(3).Find(&sampleLogs)

		samples := make([]string, len(sampleLogs))
		for j, s := range sampleLogs {
			samples[j] = s.Message
		}

		// 计算百分比（避免除零）
		percentage := 0.0
		if total > 0 {
			percentage = float64(p.Frequency) / float64(total) * 100
		}

		topPatterns[i] = gin.H{
			"pattern":     p.Pattern,
			"count":       p.Frequency,
			"percentage":  percentage,
			"sample_logs": samples,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"service":      service,
		"total_logs":   total,
		"top_patterns": topPatterns,
	})
}

// GetReport 获取完整日志报告
func (h *ReportHandler) GetReport(c *gin.Context) {
	service := c.Param("service")
	component := c.Query("component")

	// 并行获取 TOP 行号和 TOP 模式
	var topLinesResult, topPatternsResult gin.H
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		topLinesResult = h.getTopLinesData(service, component)
	}()
	go func() {
		defer wg.Done()
		topPatternsResult = h.getTopPatternsData(service, component)
	}()

	wg.Wait()

	// 计算时间范围
	now := time.Now()
	timeRange := gin.H{
		"from": now.Add(-24 * time.Hour).Format(time.RFC3339),
		"to":   now.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, gin.H{
		"service":      service,
		"total_logs":   topLinesResult["total_logs"],
		"time_range":   timeRange,
		"top_lines":    topLinesResult["top_lines"],
		"top_patterns": topPatternsResult["top_patterns"],
	})
}

// getTopLinesData 获取 TOP 行号数据（内部方法）
func (h *ReportHandler) getTopLinesData(service, component string) gin.H {
	query := h.db.Model(&models.LogEntry{}).
		Select("path, function, line_number, COUNT(*) as count").
		Where("service = ?", service)

	if component != "" {
		query = query.Where("component = ?", component)
	}

	var results []struct {
		Path       string
		Function   string
		LineNumber int
		Count      int
	}

	query.Group("path, function, line_number").
		Order("count DESC").
		Limit(10).
		Scan(&results)

	// 计算总数
	var total int64
	h.db.Model(&models.LogEntry{}).Where("service = ?", service).Count(&total)

	topLines := make([]gin.H, len(results))
	for i, r := range results {
		// 计算百分比（避免除零）
		percentage := 0.0
		if total > 0 {
			percentage = float64(r.Count) / float64(total) * 100
		}
		topLines[i] = gin.H{
			"line_number": r.LineNumber,
			"file":        r.Path,
			"function":    r.Function,
			"count":       r.Count,
			"percentage":  percentage,
		}
	}

	return gin.H{
		"total_logs": total,
		"top_lines":  topLines,
	}
}

// getTopPatternsData 获取 TOP 模式数据（内部方法）
func (h *ReportHandler) getTopPatternsData(service, component string) gin.H {
	var logs []models.LogEntry
	query := h.db.Where("service = ?", service)
	if component != "" {
		query = query.Where("component = ?", component)
	}
	query.Find(&logs)

	// 使用 PatternMiner 进行模式识别
	miner := analysis.NewPatternMiner()
	entries := make([]analysis.LogEntry, len(logs))
	for i, log := range logs {
		entries[i] = analysis.LogEntry{
			Message:   log.Message,
			Timestamp: log.Timestamp,
			Level:     log.Level,
		}
	}

	patterns := miner.AnalyzePatterns(entries)

	// 排序并取 TOP 10
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	if len(patterns) > 10 {
		patterns = patterns[:10]
	}

	// 计算总数
	var total int64
	h.db.Model(&models.LogEntry{}).Where("service = ?", service).Count(&total)

	topPatterns := make([]gin.H, len(patterns))
	for i, p := range patterns {
		// 获取示例日志（转义 LIKE 特殊字符）
		escapedPattern := escapeLikePattern(p.Pattern)
		var sampleLogs []models.LogEntry
		h.db.Where("service = ? AND message LIKE ?", service, "%"+escapedPattern+"%").
			Limit(3).Find(&sampleLogs)

		samples := make([]string, len(sampleLogs))
		for j, s := range sampleLogs {
			samples[j] = s.Message
		}

		// 计算百分比（避免除零）
		percentage := 0.0
		if total > 0 {
			percentage = float64(p.Frequency) / float64(total) * 100
		}

		topPatterns[i] = gin.H{
			"pattern":     p.Pattern,
			"count":       p.Frequency,
			"percentage":  percentage,
			"sample_logs": samples,
		}
	}

	return gin.H{
		"total_logs":   total,
		"top_patterns": topPatterns,
	}
}

// IngestLogRequest 日志摄入请求
type IngestLogRequest struct {
	Service    string                 `json:"service" binding:"required"`
	Component  string                 `json:"component"`
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Path       string                 `json:"path"`
	Function   string                 `json:"function"`
	LineNumber int                    `json:"line_number"`
	TraceID    string                 `json:"trace_id"`
	UserID     string                 `json:"user_id"`
	Fields     map[string]interface{} `json:"fields"`
}

// IngestLog 摄入单条日志
func (h *ReportHandler) IngestLog(c *gin.Context) {
	var req IngestLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := models.LogEntry{
		Service:    req.Service,
		Component:  req.Component,
		Timestamp:  req.Timestamp,
		Level:      req.Level,
		Message:    req.Message,
		Path:       req.Path,
		Function:   req.Function,
		LineNumber: req.LineNumber,
		TraceID:    req.TraceID,
		UserID:     req.UserID,
		Fields:     models.JSONMap(req.Fields),
		CreatedAt:  time.Now(),
	}

	if err := h.db.Create(&entry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest log"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      entry.ID,
		"message": "log ingested",
	})
}

// IngestBatchRequest 批量日志摄入请求
type IngestBatchRequest struct {
	Logs []IngestLogRequest `json:"logs" binding:"required"`
}

// IngestBatch 批量摄入日志
func (h *ReportHandler) IngestBatch(c *gin.Context) {
	var req IngestBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entries := make([]models.LogEntry, len(req.Logs))
	for i, log := range req.Logs {
		entries[i] = models.LogEntry{
			Service:    log.Service,
			Component:  log.Component,
			Timestamp:  log.Timestamp,
			Level:      log.Level,
			Message:    log.Message,
			Path:       log.Path,
			Function:   log.Function,
			LineNumber: log.LineNumber,
			TraceID:    log.TraceID,
			UserID:     log.UserID,
			Fields:     models.JSONMap(log.Fields),
			CreatedAt:  time.Now(),
		}
	}

	if err := h.db.CreateInBatches(entries, 100).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest logs"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"ingested": len(entries),
		"message":  "logs ingested",
	})
}

// QueryLogsRequest 日志查询请求
type QueryLogsRequest struct {
	Service   string    `json:"service"`
	Component string    `json:"component"`
	Level     string    `json:"level"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// QueryLogs 查询日志
func (h *ReportHandler) QueryLogs(c *gin.Context) {
	var req QueryLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := h.db.Model(&models.LogEntry{})

	if req.Service != "" {
		query = query.Where("service = ?", req.Service)
	}
	if req.Component != "" {
		query = query.Where("component = ?", req.Component)
	}
	if req.Level != "" {
		query = query.Where("level = ?", req.Level)
	}
	if !req.From.IsZero() {
		query = query.Where("timestamp >= ?", req.From)
	}
	if !req.To.IsZero() {
		query = query.Where("timestamp <= ?", req.To)
	}

	// 分页
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := req.Offset

	var total int64
	query.Model(&models.LogEntry{}).Count(&total)

	var logs []models.LogEntry
	query.Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"logs":  logs,
	})
}

// escapeLikePattern 转义 LIKE 模式中的特殊字符
// SQL LIKE 中 % 和 _ 是特殊字符，需要用 ESCAPE 转义
func escapeLikePattern(s string) string {
	// 转义 % 为 \%
	s = strings.ReplaceAll(s, "%", "\\%")
	// 转义 _ 为 \_
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
