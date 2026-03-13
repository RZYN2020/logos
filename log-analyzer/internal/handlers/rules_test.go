// Package handlers 规则处理器测试
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/log-system/log-analyzer/internal/etcd"
	"github.com/log-system/log-analyzer/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&models.Rule{}, &models.Condition{}, &models.Action{}, &models.RuleVersion{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

// setupTestHandler 创建测试处理器
func setupTestHandler(t *testing.T) (*RuleHandler, *gin.Engine) {
	db := setupTestDB(t)

	// 创建 mock etcd 客户端（nil 用于测试）
	var etcdCli *etcd.Client

	handler := NewRuleHandler(db, etcdCli)

	// 设置路由
	router := gin.Default()
	router.POST("/rules", handler.CreateRule)
	router.GET("/rules/:id", handler.GetRule)
	router.PUT("/rules/:id", handler.UpdateRule)
	router.DELETE("/rules/:id", handler.DeleteRule)
	router.GET("/rules", handler.ListRules)
	router.POST("/rules/:id/validate", handler.ValidateRule)
	router.POST("/rules/:id/test", handler.TestRule)

	return handler, router
}

// TestCreateRule 测试创建规则
func TestCreateRule(t *testing.T) {
	_, router := setupTestHandler(t)

	ruleJSON := `{
		"name": "test-rule",
		"description": "test description",
		"enabled": true,
		"priority": 1,
		"conditions": [
			{"field": "level", "operator": "=", "value": "ERROR"}
		],
		"actions": [
			{"type": "filter", "config": {"sampling": 1.0}}
		]
	}`

	req, _ := http.NewRequest("POST", "/rules", bytes.NewBufferString(ruleJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["id"])
	assert.Equal(t, "1", response["version"])
}

// TestGetRule 测试获取规则
func TestGetRule(t *testing.T) {
	handler, router := setupTestHandler(t)
	db := handler.db

	// 创建测试数据
	rule := &models.Rule{
		ID:          "test-get-001",
		Name:        "test-rule",
		Description: "test description",
		Enabled:     true,
		Priority:    1,
		Version:     1,
	}
	db.Create(rule)

	req, _ := http.NewRequest("GET", "/rules/test-get-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.Rule
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "test-get-001", response.ID)
	assert.Equal(t, "test-rule", response.Name)
}

// TestListRules 测试获取规则列表
func TestListRules(t *testing.T) {
	handler, router := setupTestHandler(t)
	db := handler.db

	// 创建测试数据
	rules := []models.Rule{
		{ID: "test-list-001", Name: "rule-1", Version: 1},
		{ID: "test-list-002", Name: "rule-2", Version: 1},
	}
	for i := range rules {
		db.Create(&rules[i])
	}

	req, _ := http.NewRequest("GET", "/rules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Rule
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 2, len(response))
}

// TestUpdateRule 测试更新规则
func TestUpdateRule(t *testing.T) {
	handler, router := setupTestHandler(t)
	db := handler.db

	// 创建测试数据
	rule := &models.Rule{
		ID:          "test-update-001",
		Name:        "test-rule",
		Description: "original description",
		Enabled:     true,
		Priority:    1,
		Version:     1,
	}
	db.Create(rule)

	updateJSON := `{
		"name": "updated-rule",
		"description": "updated description",
		"enabled": false,
		"priority": 2,
		"conditions": [],
		"actions": []
	}`

	req, _ := http.NewRequest("PUT", "/rules/test-update-001", bytes.NewBufferString(updateJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证数据库中的数据已更新
	var updatedRule models.Rule
	db.First(&updatedRule, "id = ?", "test-update-001")
	assert.Equal(t, "updated-rule", updatedRule.Name)
	assert.Equal(t, "updated description", updatedRule.Description)
	assert.Equal(t, false, updatedRule.Enabled)
	assert.Equal(t, 2, updatedRule.Priority)
	assert.Equal(t, 2, updatedRule.Version) // 版本号应该增加
}

// TestDeleteRule 测试删除规则
func TestDeleteRule(t *testing.T) {
	handler, router := setupTestHandler(t)
	db := handler.db

	// 创建测试数据
	rule := &models.Rule{
		ID:   "test-delete-001",
		Name: "test-rule",
	}
	db.Create(rule)

	req, _ := http.NewRequest("DELETE", "/rules/test-delete-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证数据库中的数据已删除
	var deletedRule models.Rule
	result := db.First(&deletedRule, "id = ?", "test-delete-001")
	assert.Error(t, result.Error)
	assert.Equal(t, "record not found", result.Error.Error())
}

// TestValidateRule 测试验证规则
func TestValidateRule(t *testing.T) {
	handler, router := setupTestHandler(t)
	db := handler.db

	// 创建有效的测试数据
	rule := &models.Rule{
		ID:       "test-validate-001",
		Name:     "test-rule",
		Enabled:  true,
		Version:  1,
		Conditions: []models.Condition{
			{ID: "cond-validate-001", RuleID: "test-validate-001", Field: "level", Operator: "=", Value: "ERROR"},
		},
		Actions: []models.Action{
			{ID: "act-validate-001", RuleID: "test-validate-001", Type: "filter", Config: models.JSONMap{"sampling": 1.0}},
		},
	}
	db.Create(rule)
	db.Create(&rule.Conditions[0])
	db.Create(&rule.Actions[0])

	req, _ := http.NewRequest("POST", "/rules/test-validate-001/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, true, response["valid"])
}

// TestTestRule 测试测试规则
func TestTestRule(t *testing.T) {
	handler, router := setupTestHandler(t)
	db := handler.db

	// 创建测试数据
	rule := &models.Rule{
		ID:       "test-rule-001",
		Name:     "test-rule",
		Enabled:  true,
		Version:  1,
		Conditions: []models.Condition{
			{ID: "cond-test-001", RuleID: "test-rule-001", Field: "level", Operator: "=", Value: "ERROR"},
		},
	}
	db.Create(rule)
	db.Create(&rule.Conditions[0])

	req, _ := http.NewRequest("POST", "/rules/test-rule-001/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["matched"])
}
