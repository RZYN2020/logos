package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/log-system/log-analyzer/internal/models"
	"gorm.io/gorm"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	db *gorm.DB
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{
		db: db,
	}
}

// AlertRuleRequest 告警规则请求
type AlertRuleRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	Service     string         `json:"service" binding:"required"`
	Condition   models.JSONMap `json:"condition" binding:"required"`
	Threshold   int            `json:"threshold"`
	Window      int            `json:"window"`
	Channels    models.JSONMap `json:"channels" binding:"required"`
}

// CreateAlertRule 创建告警规则
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	var req AlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := &models.AlertRule{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		Service:     req.Service,
		Condition:   req.Condition,
		Threshold:   req.Threshold,
		Window:      req.Window,
		Channels:    req.Channels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.Create(rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create alert rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListAlertRules 获取告警规则列表
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
	service := c.Query("service")

	query := h.db.Model(&models.AlertRule{})
	if service != "" {
		query = query.Where("service = ?", service)
	}

	var rules []models.AlertRule
	if err := query.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alert rules"})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// UpdateAlertRule 更新告警规则
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule models.AlertRule
	if err := h.db.First(&rule, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert rule not found"})
		return
	}

	var req AlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.Name = req.Name
	rule.Description = req.Description
	rule.Enabled = req.Enabled
	rule.Service = req.Service
	rule.Condition = req.Condition
	rule.Threshold = req.Threshold
	rule.Window = req.Window
	rule.Channels = req.Channels
	rule.UpdatedAt = time.Now()

	if err := h.db.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteAlertRule 删除告警规则
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	if err := h.db.Delete(&models.AlertRule{}, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete alert rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert rule deleted"})
}

// ListAlertHistory 获取告警历史
func (h *AlertHandler) ListAlertHistory(c *gin.Context) {
	service := c.Query("service")

	query := h.db.Model(&models.AlertHistory{}).Order("trigger_time desc")
	if service != "" {
		query = query.Where("service = ?", service)
	}

	var history []models.AlertHistory
	if err := query.Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alert history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// ResolveAlert 处理告警
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")

	if err := h.db.Model(&models.AlertHistory{}).Where("id = ?", alertID).Update("status", "resolved").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert resolved"})
}