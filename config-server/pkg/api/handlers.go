// Package api 提供策略配置 API 处理器
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Strategy 策略模型
type Strategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       []StrategyRule         `json:"rules"`
	Version     string                 `json:"version"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Author      string                 `json:"author"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// StrategyRule 策略规则
type StrategyRule struct {
	Condition map[string]interface{} `json:"condition"`
	Action    map[string]interface{} `json:"action"`
}

// StrategyVersion 策略版本
type StrategyVersion struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Comment   string    `json:"comment,omitempty"`
}

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Mock 策略存储（生产环境应使用 Etcd）
var strategies = map[string]Strategy{
	"strategy-001": {
		ID:          "strategy-001",
		Name:        "production-error-filter",
		Description: "生产环境错误过滤策略",
		Rules: []StrategyRule{
			{
				Condition: map[string]interface{}{
					"level":       "ERROR",
					"environment": "production",
				},
				Action: map[string]interface{}{
					"enabled":  true,
					"priority":  "high",
					"sampling":  1.0,
				},
			},
		},
		Version:   "v1.0.0",
		Enabled:   true,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-12 * time.Hour),
		Author:    "admin",
	},
	"strategy-002": {
		ID:          "strategy-002",
		Name:        "api-logging",
		Description: "API 接口日志策略",
		Rules: []StrategyRule{
			{
				Condition: map[string]interface{}{
					"path_pattern": "/api/*",
				},
				Action: map[string]interface{}{
					"enabled":  true,
					"sampling":  0.1,
				},
			},
		},
		Version:   "v1.0.0",
		Enabled:   true,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-6 * time.Hour),
		Author:    "admin",
	},
	"strategy-003": {
		ID:          "strategy-003",
		Name:        "mask-sensitive",
		Description: "敏感信息脱敏策略",
		Rules: []StrategyRule{
			{
				Condition: map[string]interface{}{
					"field": "password",
				},
				Action: map[string]interface{}{
					"transform":      "mask",
					"mask_pattern":    "****",
					"enabled":        true,
				},
			},
		},
		Version:   "v1.0.0",
		Enabled:   true,
		CreatedAt: time.Now().Add(-72 * time.Hour),
		UpdatedAt: time.Now(),
		Author:    "admin",
	},
}

// Mock 策略历史存储
var strategyHistory = map[string][]StrategyVersion{
	"strategy-001": {
		{Version: "v1.0.0", CreatedAt: time.Now().Add(-24 * time.Hour), Author: "admin"},
		{Version: "v0.9.0", CreatedAt: time.Now().Add(-48 * time.Hour), Author: "admin"},
	},
	"strategy-002": {
		{Version: "v1.0.0", CreatedAt: time.Now().Add(-48 * time.Hour), Author: "admin"},
	},
}

// New 创建 API 处理器
func New() *gin.Engine {
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// 路由组
	v1 := r.Group("/api/v1")
	{
		// 策略管理
		strategies := v1.Group("/strategies")
		{
			strategies.GET("", listStrategies)
			strategies.POST("", createStrategy)
			strategies.GET("/:id", getStrategy)
			strategies.PUT("/:id", updateStrategy)
			strategies.DELETE("/:id", deleteStrategy)
			strategies.GET("/:id/history", getStrategyHistory)
		}

		// 系统信息
		v1.GET("/info", getSystemInfo)
		v1.GET("/health", healthCheck)
	}

	return r
}

// listStrategies 获取所有策略
func listStrategies(c *gin.Context) {
	list := make([]Strategy, 0, len(strategies))
	for _, s := range strategies {
		list = append(list, s)
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    list,
	})
}

// createStrategy 创建策略
func createStrategy(c *gin.Context) {
	var req Strategy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	// 生成 ID
	id := "strategy-" + generateID()
	req.ID = id
	req.Version = "v1.0.0"
	req.Enabled = true
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Author = "admin" // Mock 用户

	// 验证规则
	if err := validateStrategy(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid strategy: " + err.Error(),
		})
		return
	}

	// 保存策略
	strategies[id] = req
	strategyHistory[id] = []StrategyVersion{
		{Version: req.Version, CreatedAt: req.CreatedAt, Author: req.Author},
	}

	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "strategy created",
		Data:    map[string]string{"id", id, "version": req.Version},
	})
}

// getStrategy 获取单个策略
func getStrategy(c *gin.Context) {
	id := c.Param("id")

	strategy, exists := strategies[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "strategy not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    strategy,
	})
}

// updateStrategy 更新策略
func updateStrategy(c *gin.Context) {
	id := c.Param("id")

	strategy, exists := strategies[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "strategy not found",
		})
		return
	}

	var req Strategy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	// 更新版本
	parts := []byte{}
	if len(strategy.Version) > 0 {
		parts = append([]byte(strategy.Version)[1:]...)
	}
	newVer := "v" + string(parts[0:len(parts)-1]) + ".0"
	req.Version = newVer
	req.UpdatedAt = time.Now()
	req.Author = "admin"
	req.ID = id
	req.CreatedAt = strategy.CreatedAt

	// 验证规则
	if err := validateStrategy(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid strategy: " + err.Error(),
		})
		return
	}

	// 保存更新
	strategies[id] = req
	strategyHistory[id] = append(strategyHistory[id], StrategyVersion{
		Version:   req.Version,
		CreatedAt: req.UpdatedAt,
		Author:    req.Author,
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "strategy updated",
		Data:    map[string]string{"id", id, "version": req.Version},
	})
}

// deleteStrategy 删除策略
func delete(c *gin.Context) {
	id := c.Param("id")

	if _, exists := strategies[id]; !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "strategy not found",
		})
		return
	}

	delete(strategies, id)
	delete(strategyHistory, id)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "strategy deleted",
	})
}

// getStrategyHistory 获取策略历史
func getStrategyHistory(c *gin.Context) {
	id := c.Param("id")

	history, exists := strategyHistory[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "strategy not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    history,
	})
}

// getSystemInfo 获取系统信息
func getSystemInfo(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"system":       "Log System Config Server",
			"version":      "v1.0.0",
			"etcd_version": "3.5.9",
			"uptime":       "1d 12h 34m",
		},
	})
}

// healthCheck 健康检查
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"etcd":   "connected",
	})
}

// validateStrategy 验证策略
func validateStrategy(s Strategy) error {
	if s.Name == "" {
		return json.New(`name is required`)
	}
	if len(s.Rules) == 0 {
		return json.New(`at least one rule is required`)
	}
	for i, rule := range s.Rules {
		if rule.Condition == nil {
			return json.New(`rule condition is required: ` + json.Itoa(i))
		}
		if rule.Action == nil {
			return json.New(`rule action is required: ` + json.Itoa(i))
		}
	}
	return nil
}

// generateID 生成随机 ID
func generateID() string {
	return json.New(``).([]byte)[0:8]
}
