// Package integration 集成测试
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/log-system/log-analyzer/internal/etcd"
	"github.com/log-system/log-analyzer/internal/handlers"
	"github.com/log-system/log-analyzer/internal/middleware"
	"github.com/log-system/log-analyzer/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestServer 测试服务器
type TestServer struct {
	router          *gin.Engine
	db              *gorm.DB
	authConfig      *middleware.AuthConfig
	authHandler     *handlers.AuthHandler
	ruleHandler     *handlers.RuleHandler
	analysisHandler *handlers.AnalysisHandler
	reportHandler   *handlers.ReportHandler
}

// setupTestServer 设置测试服务器
func setupTestServer(t *testing.T) *TestServer {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// 迁移表
	err = db.AutoMigrate(
		&models.Rule{}, &models.Condition{}, &models.Action{}, &models.RuleVersion{},
		&models.LogEntry{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// 创建认证配置
	authConfig := middleware.NewAuthConfig()
	authHandler := handlers.NewAuthHandler(authConfig)

	// 创建规则处理器
	var etcdCli *etcd.Client // nil for testing
	ruleHandler := handlers.NewRuleHandler(db, etcdCli)

	// 创建分析处理器
	analysisHandler := handlers.NewAnalysisHandler()

	// 创建报告处理器
	reportHandler := handlers.NewReportHandler(db)

	// 设置路由
	router := gin.Default()

	// 认证路由（公开）
	router.POST("/api/v1/auth/login", authHandler.Login)
	router.POST("/api/v1/auth/register", authHandler.Register)

	// 认证中间件
	auth := router.Group("/api/v1")
	auth.Use(authConfig.AuthMiddleware())
	{
		// 规则 API
		auth.GET("/rules", ruleHandler.ListRules)
		auth.POST("/rules", ruleHandler.CreateRule)
		auth.GET("/rules/:id", ruleHandler.GetRule)
		auth.PUT("/rules/:id", ruleHandler.UpdateRule)
		auth.DELETE("/rules/:id", ruleHandler.DeleteRule)
		auth.POST("/rules/:id/validate", ruleHandler.ValidateRule)
		auth.POST("/rules/:id/test", ruleHandler.TestRule)

		// 分析 API
		auth.POST("/analysis/mine", analysisHandler.MinePatterns)
		auth.POST("/analysis/recommend", analysisHandler.RecommendRules)

		// 报告 API
		auth.GET("/report/:service/top-lines", reportHandler.GetTopLines)
		auth.GET("/report/:service/top-patterns", reportHandler.GetTopPatterns)
		auth.GET("/report/:service", reportHandler.GetReport)

		// 日志摄入 API
		auth.POST("/logs", reportHandler.IngestLog)
		auth.POST("/logs/batch", reportHandler.IngestBatch)
		auth.POST("/logs/query", reportHandler.QueryLogs)
	}

	return &TestServer{
		router:          router,
		db:              db,
		authConfig:      authConfig,
		authHandler:     authHandler,
		ruleHandler:     ruleHandler,
		analysisHandler: analysisHandler,
		reportHandler:   reportHandler,
	}
}

// login  Helper 函数获取 token
func (s *TestServer) login(t *testing.T) string {
	loginJSON := `{"username": "admin", "password": "admin123"}`
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(loginJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	return response["token"].(string)
}

// TestIntegration_FullWorkflow 完整工作流集成测试
func TestIntegration_FullWorkflow(t *testing.T) {
	server := setupTestServer(t)
	token := server.login(t)

	// 1. 创建规则
	t.Run("CreateRule", func(t *testing.T) {
		ruleJSON := `{
			"name": "integration-test-rule",
			"description": "Integration test",
			"enabled": true,
			"priority": 1,
			"conditions": [
				{"field": "level", "operator": "=", "value": "ERROR"}
			],
			"actions": [
				{"type": "filter", "config": {"sampling": 1.0}}
			]
		}`

		req, _ := http.NewRequest("POST", "/api/v1/rules", bytes.NewBufferString(ruleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	// 2. 获取规则列表
	t.Run("ListRules", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var rules []models.Rule
		_ = json.Unmarshal(w.Body.Bytes(), &rules)
		assert.Greater(t, len(rules), 0)
	})

	// 3. 验证规则
	t.Run("ValidateRule", func(t *testing.T) {
		// 先获取规则 ID
		var rules []models.Rule
		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		_ = json.Unmarshal(w.Body.Bytes(), &rules)

		if len(rules) > 0 {
			ruleID := rules[0].ID

			req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/rules/%s/validate", ruleID), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w = httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			assert.Equal(t, true, response["valid"])
		}
	})

	// 4. 测试规则
	t.Run("TestRule", func(t *testing.T) {
		var rules []models.Rule
		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		_ = json.Unmarshal(w.Body.Bytes(), &rules)

		if len(rules) > 0 {
			ruleID := rules[0].ID

			req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/rules/%s/test", ruleID), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w = httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	// 5. 更新规则
	t.Run("UpdateRule", func(t *testing.T) {
		var rules []models.Rule
		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		_ = json.Unmarshal(w.Body.Bytes(), &rules)

		if len(rules) > 0 {
			ruleID := rules[0].ID

			updateJSON := `{
				"name": "updated-rule",
				"description": "Updated description",
				"enabled": false,
				"priority": 2,
				"conditions": [],
				"actions": []
			}`

			req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/rules/%s", ruleID), bytes.NewBufferString(updateJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w = httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	// 6. 删除规则
	t.Run("DeleteRule", func(t *testing.T) {
		var rules []models.Rule
		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		_ = json.Unmarshal(w.Body.Bytes(), &rules)

		if len(rules) > 0 {
			ruleID := rules[0].ID

			req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/v1/rules/%s", ruleID), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w = httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		}
	})
}

// TestIntegration_LogAnalysis 日志分析集成测试
func TestIntegration_LogAnalysis(t *testing.T) {
	server := setupTestServer(t)
	token := server.login(t)

	// 测试模式挖掘
	t.Run("MinePatterns", func(t *testing.T) {
		requestJSON := `{
			"logs": [
				{"timestamp": "2024-01-01T00:00:00Z", "level": "ERROR", "service": "api", "message": "Connection failed"},
				{"timestamp": "2024-01-01T00:00:01Z", "level": "ERROR", "service": "api", "message": "Connection failed"},
				{"timestamp": "2024-01-01T00:00:02Z", "level": "INFO", "service": "api", "message": "Request processed"}
			]
		}`

		req, _ := http.NewRequest("POST", "/api/v1/analysis/mine", bytes.NewBufferString(requestJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["patterns"])
	})

	// 测试规则推荐
	t.Run("RecommendRules", func(t *testing.T) {
		// 生成 51 条日志以触发 medium severity (>50)
		logs := make([]map[string]interface{}, 51)
		for i := 0; i < 51; i++ {
			logs[i] = map[string]interface{}{
				"timestamp": "2024-01-01T00:00:00Z",
				"level":     "ERROR",
				"service":   "api",
				"message":   "Payment failed",
			}
		}

		requestBody, _ := json.Marshal(map[string]interface{}{
			"logs":          logs,
			"min_frequency": 1,
		})

		req, _ := http.NewRequest("POST", "/api/v1/analysis/recommend", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["recommendations"])
		assert.Greater(t, len(response["recommendations"].([]interface{})), 0)
	})
}

// TestIntegration_Authentication 认证集成测试
func TestIntegration_Authentication(t *testing.T) {
	server := setupTestServer(t)

	// 测试登录
	t.Run("Login", func(t *testing.T) {
		loginJSON := `{"username": "admin", "password": "admin123"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["token"])
		assert.NotNil(t, response["user"])
	})

	// 测试注册
	t.Run("Register", func(t *testing.T) {
		registerJSON := `{"username": "testuser", "password": "testpass123", "email": "test@example.com"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(registerJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["token"])
	})

	// 测试无认证访问
	t.Run("UnauthorizedAccess", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/rules", nil)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestIntegration_LogIngestion 日志摄入集成测试
func TestIntegration_LogIngestion(t *testing.T) {
	server := setupTestServer(t)
	token := server.login(t)

	// 测试单条日志摄入
	t.Run("IngestLog", func(t *testing.T) {
		logJSON := `{
			"service": "test-service",
			"component": "sdk",
			"level": "ERROR",
			"message": "Connection failed: timeout",
			"path": "/app/handler.go",
			"function": "HandleRequest",
			"line_number": 42,
			"trace_id": "trace-123",
			"user_id": "user-456",
			"fields": {"key": "value"}
		}`

		req, _ := http.NewRequest("POST", "/api/v1/logs", bytes.NewBufferString(logJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["id"])
	})

	// 测试批量日志摄入
	t.Run("IngestLogsBatch", func(t *testing.T) {
		logs := []map[string]interface{}{
			{
				"service":     "test-service",
				"component":   "sdk",
				"level":       "INFO",
				"message":     "Request processed",
				"path":        "/app/handler.go",
				"function":    "HandleRequest",
				"line_number": 50,
				"timestamp":   time.Now().Format(time.RFC3339),
			},
			{
				"service":     "test-service",
				"component":   "sdk",
				"level":       "WARN",
				"message":     "Slow response detected",
				"path":        "/app/handler.go",
				"function":    "HandleRequest",
				"line_number": 55,
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		}

		body, _ := json.Marshal(map[string]interface{}{"logs": logs})
		req, _ := http.NewRequest("POST", "/api/v1/logs/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.Greater(t, int(response["ingested"].(float64)), 0)
	})

	// 测试日志查询
	t.Run("QueryLogs", func(t *testing.T) {
		queryJSON := `{
			"service": "test-service",
			"limit": 10
		}`

		req, _ := http.NewRequest("POST", "/api/v1/logs/query", bytes.NewBufferString(queryJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["total"])
		assert.NotNil(t, response["logs"])
	})
}

// TestIntegration_LogReport 日志报告集成测试
func TestIntegration_LogReport(t *testing.T) {
	server := setupTestServer(t)
	token := server.login(t)

	// 先摄入一些测试日志
	logs := []map[string]interface{}{
		{"service": "report-test", "component": "sdk", "level": "ERROR", "message": "Connection failed", "path": "/app/db.go", "function": "Connect", "line_number": 100},
		{"service": "report-test", "component": "sdk", "level": "ERROR", "message": "Connection failed", "path": "/app/db.go", "function": "Connect", "line_number": 100},
		{"service": "report-test", "component": "sdk", "level": "ERROR", "message": "Connection failed", "path": "/app/db.go", "function": "Connect", "line_number": 100},
		{"service": "report-test", "component": "sdk", "level": "INFO", "message": "Request processed successfully", "path": "/app/handler.go", "function": "Handle", "line_number": 50},
		{"service": "report-test", "component": "sdk", "level": "INFO", "message": "Request processed successfully", "path": "/app/handler.go", "function": "Handle", "line_number": 50},
	}

	body, _ := json.Marshal(map[string]interface{}{"logs": logs})
	req, _ := http.NewRequest("POST", "/api/v1/logs/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// 测试 TOP 行号统计
	t.Run("GetTopLines", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/report/report-test/top-lines", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["total_logs"])
		assert.NotNil(t, response["top_lines"])
		topLines := response["top_lines"].([]interface{})
		assert.Greater(t, len(topLines), 0)
	})

	// 测试 TOP 模式统计
	t.Run("GetTopPatterns", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/report/report-test/top-patterns", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["total_logs"])
		assert.NotNil(t, response["top_patterns"])
		topPatterns := response["top_patterns"].([]interface{})
		assert.Greater(t, len(topPatterns), 0)
	})

	// 测试完整报告
	t.Run("GetReport", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/report/report-test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["total_logs"])
		assert.NotNil(t, response["top_lines"])
		assert.NotNil(t, response["top_patterns"])
	})
}
