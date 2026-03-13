// Package handlers HTTP 处理器
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/log-system/log-analyzer/internal/etcd"
	"github.com/log-system/log-analyzer/internal/models"
	"gorm.io/gorm"
)

// RuleHandler 规则处理器
type RuleHandler struct {
	db       *gorm.DB
	etcdCli  *etcd.Client
}

// NewRuleHandler 创建规则处理器
func NewRuleHandler(db *gorm.DB, etcdCli *etcd.Client) *RuleHandler {
	return &RuleHandler{
		db:      db,
		etcdCli: etcdCli,
	}
}

// RuleRequest 规则创建/更新请求
type RuleRequest struct {
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	Enabled     bool                `json:"enabled"`
	Priority    int                 `json:"priority"`
	Conditions  []ConditionRequest  `json:"conditions"`
	Actions     []ActionRequest     `json:"actions"`
}

// ConditionRequest 条件请求
type ConditionRequest struct {
	Field    string      `json:"field" binding:"required"`
	Operator string      `json:"operator" binding:"required"`
	Value    interface{} `json:"value" binding:"required"`
}

// ActionRequest 动作请求
type ActionRequest struct {
	Type   string                 `json:"type" binding:"required"`
	Config map[string]interface{} `json:"config"`
}

// CreateRule 创建规则
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req RuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ruleID := uuid.New().String()
	rule := &models.Rule{
		ID:          ruleID,
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		Priority:    req.Priority,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 创建条件和动作
	for _, cond := range req.Conditions {
		rule.Conditions = append(rule.Conditions, models.Condition{
			ID:       uuid.New().String(),
			RuleID:   ruleID,
			Field:    cond.Field,
			Operator: cond.Operator,
			Value:    cond.Value,
		})
	}

	for _, act := range req.Actions {
		rule.Actions = append(rule.Actions, models.Action{
			ID:     uuid.New().String(),
			RuleID: ruleID,
			Type:   act.Type,
			Config: act.Config,
		})
	}

	if err := h.db.Create(rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rule"})
		return
	}

	// 同步到 ETCD
	if err := h.syncRuleToEtcd(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync to etcd"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      ruleID,
		"version": "1",
	})
}

// GetRule 获取规则详情
func (h *RuleHandler) GetRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule models.Rule
	if err := h.db.Preload("Conditions").Preload("Actions").First(&rule, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// ListRules 获取规则列表
func (h *RuleHandler) ListRules(c *gin.Context) {
	var rules []models.Rule
	if err := h.db.Preload("Conditions").Preload("Actions").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rules"})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// UpdateRule 更新规则
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule models.Rule
	if err := h.db.First(&rule, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var req RuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新规则
	rule.Name = req.Name
	rule.Description = req.Description
	rule.Enabled = req.Enabled
	rule.Priority = req.Priority
	rule.Version++
	rule.UpdatedAt = time.Now()

	// 删除旧的条件和动作
	h.db.Where("rule_id = ?", ruleID).Delete(&models.Condition{})
	h.db.Where("rule_id = ?", ruleID).Delete(&models.Action{})

	// 创建新的条件和动作
	for _, cond := range req.Conditions {
		rule.Conditions = append(rule.Conditions, models.Condition{
			ID:       uuid.New().String(),
			RuleID:   ruleID,
			Field:    cond.Field,
			Operator: cond.Operator,
			Value:    cond.Value,
		})
	}

	for _, act := range req.Actions {
		rule.Actions = append(rule.Actions, models.Action{
			ID:     uuid.New().String(),
			RuleID: ruleID,
			Type:   act.Type,
			Config: act.Config,
		})
	}

	if err := h.db.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rule"})
		return
	}

	// 创建版本记录
	version := &models.RuleVersion{
		ID:      uuid.New().String(),
		RuleID:  ruleID,
		Version: rule.Version,
		Author:  c.GetHeader("X-User"),
		CreatedAt: time.Now(),
	}
	content, _ := json.Marshal(rule)
	json.Unmarshal(content, &version.Content)
	h.db.Create(version)

	// 同步到 ETCD
	if err := h.syncRuleToEtcd(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync to etcd"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      ruleID,
		"version": strconv.Itoa(rule.Version),
	})
}

// DeleteRule 删除规则
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("id")

	if err := h.db.Delete(&models.Rule{ID: ruleID}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule"})
		return
	}

	// 从 ETCD 删除 - 使用统一规则的命名空间
	key := "/rules/clients/analyzer.default/sdk/" + ruleID
	h.etcdCli.Delete(c.Request.Context(), key)

	c.JSON(http.StatusOK, gin.H{"message": "rule deleted"})
}

// GetRuleHistory 获取规则历史
func (h *RuleHandler) GetRuleHistory(c *gin.Context) {
	ruleID := c.Param("id")

	var versions []models.RuleVersion
	if err := h.db.Where("rule_id = ?", ruleID).Order("version DESC").Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get history"})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// RollbackRule 回滚规则版本
func (h *RuleHandler) RollbackRule(c *gin.Context) {
	ruleID := c.Param("id")
	versionStr := c.Param("version")

	version, _ := strconv.Atoi(versionStr)

	var versionRecord models.RuleVersion
	if err := h.db.Where("rule_id = ? AND version = ?", ruleID, version).First(&versionRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	var rule models.Rule
	if err := h.db.First(&rule, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	// 恢复规则内容
	content, _ := json.Marshal(versionRecord.Content)
	json.Unmarshal(content, &rule)
	rule.Version++
	rule.UpdatedAt = time.Now()

	if err := h.db.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rollback"})
		return
	}

	// 同步到 ETCD
	if err := h.syncRuleToEtcd(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync to etcd"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      ruleID,
		"version": strconv.Itoa(rule.Version),
	})
}

// ValidateRule 验证规则
func (h *RuleHandler) ValidateRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule models.Rule
	if err := h.db.Preload("Conditions").Preload("Actions").First(&rule, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	// 验证规则逻辑
	errors := []string{}

	// 验证条件
	for _, cond := range rule.Conditions {
		if cond.Field == "" {
			errors = append(errors, "condition field cannot be empty")
		}
		if cond.Operator == "" {
			errors = append(errors, "condition operator cannot be empty")
		}
	}

	// 验证动作
	for _, act := range rule.Actions {
		if act.Type == "" {
			errors = append(errors, "action type cannot be empty")
		}
		if act.Type != "filter" && act.Type != "drop" && act.Type != "transform" {
			errors = append(errors, "invalid action type")
		}
	}

	if len(errors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "errors": errors})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// TestRule 测试规则
func (h *RuleHandler) TestRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule models.Rule
	if err := h.db.Preload("Conditions").Preload("Actions").First(&rule, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	// 模拟测试数据
	testData := map[string]interface{}{
		"level":   "ERROR",
		"service": "test-service",
	}

	// 测试条件匹配
	matched := true
	for _, cond := range rule.Conditions {
		if cond.Field == "level" && cond.Value != testData["level"] {
			matched = false
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"matched": matched,
		"test_data": testData,
	})
}

// ExportRules 导出规则
func (h *RuleHandler) ExportRules(c *gin.Context) {
	var rules []models.Rule
	if err := h.db.Preload("Conditions").Preload("Actions").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// ImportRules 导入规则
func (h *RuleHandler) ImportRules(c *gin.Context) {
	var req struct {
		Rules []models.Rule `json:"rules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imported := 0
	for _, rule := range req.Rules {
		rule.ID = uuid.New().String()
		rule.CreatedAt = time.Now()
		rule.UpdatedAt = time.Now()

		if err := h.db.Create(&rule).Error; err != nil {
			continue
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{"imported": imported})
}

// syncRuleToEtcd 同步规则到 ETCD
func (h *RuleHandler) syncRuleToEtcd(rule *models.Rule) error {
	ctx := context.Background()

	// 转换为统一规则模型
	unifiedRule := rule.ToUnifiedRule()

	// 序列化为 JSON
	data, err := json.Marshal(unifiedRule)
	if err != nil {
		return err
	}

	// 写入 ETCD - 使用统一规则的命名空间
	// 格式：/rules/clients/{service_name}.{environment}/sdk/{ruleID}
	// 这里使用 analyzer 作为默认服务
	key := "/rules/clients/analyzer.default/sdk/" + rule.ID
	return h.etcdCli.Put(ctx, key, string(data))
}
