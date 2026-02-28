// Package api 提供策略配置和解析器配置 API 处理器
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ============== 策略配置相关 ==============

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

// ============== 解析器配置相关 ==============

// ParserConfig 解析器配置
type ParserConfig struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // json, key_value, syslog, apache, nginx, unstructured
	Enabled   bool                   `json:"enabled"`
	Priority  int                    `json:"priority"`
	Config    map[string]interface{} `json:"config,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Author    string                 `json:"author"`
	Version   string                 `json:"version"`
}

// ParserConfigVersion 解析器配置版本
type ParserConfigVersion struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Comment   string    `json:"comment,omitempty"`
	Content   string    `json:"content_hash"`
}

// ============== 转换规则配置相关 ==============

// TransformRuleConfig 转换规则配置
type TransformRuleConfig struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       []TransformRule        `json:"rules"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	Service     string                 `json:"service,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Author      string                 `json:"author"`
	Version     string                 `json:"version"`
}

// TransformRule 转换规则
type TransformRule struct {
	ID            string                 `json:"id"`
	SourceField   string                 `json:"source_field"`
	TargetField   string                 `json:"target_field"`
	Extractor     string                 `json:"extractor"` // regex, template, jsonpath, direct, lowercase, uppercase, split
	Config        map[string]interface{} `json:"config,omitempty"`
	OnError       string                 `json:"on_error,omitempty"` // skip, fail, default
	DefaultValue  interface{}            `json:"default_value,omitempty"`
}

// TransformRuleVersion 转换规则版本
type TransformRuleVersion struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Comment   string    `json:"comment,omitempty"`
}

// ============== 过滤器配置相关 ==============

// FilterConfig 过滤器配置
type FilterConfig struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Enabled     bool         `json:"enabled"`
	Priority    int          `json:"priority"`
	Service     string       `json:"service,omitempty"`
	Environment string       `json:"environment,omitempty"`
	Rules       []FilterRule `json:"rules"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
}

// FilterRule 过滤规则
type FilterRule struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Field   string       `json:"field"`
	Pattern string       `json:"pattern"`
	Action  FilterAction `json:"action"`
}

// FilterAction 过滤动作
type FilterAction int

const (
	ActionAllow FilterAction = iota
	ActionDrop
	ActionMark
)

// FilterConfigVersion 过滤器配置版本
type FilterConfigVersion struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Comment   string    `json:"comment,omitempty"`
}

// ============== 响应格式 ==============

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ============== 数据存储 ==============

var (
	// 策略存储
	strategies = make(map[string]Strategy)
	// 策略历史存储
	strategyHistory = make(map[string][]StrategyVersion)
	// 解析器配置存储
	parserConfigs = make(map[string]ParserConfig)
	// 解析器配置历史存储
	parserConfigHistory = make(map[string][]ParserConfigVersion)
	// 转换规则存储
	transformRules = make(map[string]TransformRuleConfig)
	// 转换规则历史存储
	transformRuleHistory = make(map[string][]TransformRuleVersion)
	// 过滤器配置存储
	filterConfigs = make(map[string]FilterConfig)
	// 过滤器配置历史存储
	filterConfigHistory = make(map[string][]FilterConfigVersion)
	// 配置版本推送通道
	configWatchers = make(map[string]chan ConfigUpdateEvent)
	// 互斥锁
	mu sync.RWMutex
)

// ConfigUpdateEvent 配置更新事件
type ConfigUpdateEvent struct {
	Type      string      `json:"type"` // created, updated, deleted
	ConfigType string     `json:"config_type"` // parser, transform, filter, strategy
	ID        string      `json:"id"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// 初始化 mock 数据
func init() {
	// 初始化策略
	strategies["strategy-001"] = Strategy{
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
	}
	strategyHistory["strategy-001"] = []StrategyVersion{
		{Version: "v1.0.0", CreatedAt: time.Now().Add(-24 * time.Hour), Author: "admin"},
	}

	// 初始化解析器配置
	parserConfigs["parser-json"] = ParserConfig{
		ID:        "parser-json",
		Name:      "JSON Parser",
		Type:      "json",
		Enabled:   true,
		Priority:  100,
		Config:    map[string]interface{}{"strict": true},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	parserConfigs["parser-keyvalue"] = ParserConfig{
		ID:        "parser-keyvalue",
		Name:      "KeyValue Parser",
		Type:      "key_value",
		Enabled:   true,
		Priority:  90,
		Config:    map[string]interface{}{"delimiter": "=", "separator": " "},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	parserConfigs["parser-syslog"] = ParserConfig{
		ID:        "parser-syslog",
		Name:      "Syslog Parser",
		Type:      "syslog",
		Enabled:   true,
		Priority:  80,
		Config:    map[string]interface{}{"location": "UTC"},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	parserConfigs["parser-apache"] = ParserConfig{
		ID:        "parser-apache",
		Name:      "Apache Parser",
		Type:      "apache",
		Enabled:   true,
		Priority:  70,
		Config:    map[string]interface{}{},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	parserConfigs["parser-nginx"] = ParserConfig{
		ID:        "parser-nginx",
		Name:      "Nginx Parser",
		Type:      "nginx",
		Enabled:   true,
		Priority:  60,
		Config:    map[string]interface{}{},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	parserConfigs["parser-unstructured"] = ParserConfig{
		ID:        "parser-unstructured",
		Name:      "Unstructured Parser",
		Type:      "unstructured",
		Enabled:   true,
		Priority:  10,
		Config:    map[string]interface{}{"min_confidence": 0.8},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}

	for id, config := range parserConfigs {
		parserConfigHistory[id] = []ParserConfigVersion{
			{Version: config.Version, CreatedAt: config.CreatedAt, Author: config.Author},
		}
	}

	// 初始化转换规则
	transformRules["transform-http"] = TransformRuleConfig{
		ID:          "transform-http",
		Name:        "HTTP Log Transformation",
		Description: "HTTP 日志字段提取规则",
		Enabled:     true,
		Priority:    100,
		Service:     "web-service",
		Rules: []TransformRule{
			{
				ID:          "rule-001",
				SourceField: "message",
				TargetField: "http_method",
				Extractor:   "regex",
				Config:      map[string]interface{}{"pattern": "(GET|POST|PUT|DELETE|HEAD|PATCH|OPTIONS)\\s+"},
				OnError:     "skip",
			},
			{
				ID:          "rule-002",
				SourceField: "message",
				TargetField: "http_path",
				Extractor:   "regex",
				Config:      map[string]interface{}{"pattern": "\\s+(/[^\\s]+)\\s+"},
				OnError:     "skip",
			},
			{
				ID:          "rule-003",
				SourceField: "message",
				TargetField: "http_status",
				Extractor:   "regex",
				Config:      map[string]interface{}{"pattern": "\"\\s+(\\d{3})\\s+"},
				OnError:     "skip",
			},
		},
		CreatedAt: time.Now().Add(-72 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	transformRuleHistory["transform-http"] = []TransformRuleVersion{
		{Version: "v1.0.0", CreatedAt: time.Now().Add(-72 * time.Hour), Author: "admin"},
	}

	// 初始化过滤器配置
	filterConfigs["filter-debug"] = FilterConfig{
		ID:          "filter-debug",
		Name:        "Debug Log Filter",
		Description: "生产环境过滤 DEBUG 日志",
		Enabled:     true,
		Priority:    50,
		Environment: "production",
		Rules: []FilterRule{
			{
				ID:      "rule-001",
				Name:    "drop-debug",
				Field:   "level",
				Pattern: "^DEBUG$",
				Action:  ActionDrop,
			},
		},
		CreatedAt: time.Now().Add(-96 * time.Hour),
		UpdatedAt: time.Now().Add(-48 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	filterConfigs["filter-security"] = FilterConfig{
		ID:          "filter-security",
		Name:        "Security Filter",
		Description: "敏感信息过滤",
		Enabled:     true,
		Priority:    100,
		Rules: []FilterRule{
			{
				ID:      "rule-001",
				Name:    "drop-sensitive",
				Field:   "message",
				Pattern: ".*(password|secret|token|credential).*",
				Action:  ActionDrop,
			},
			{
				ID:      "rule-002",
				Name:    "mark-error",
				Field:   "level",
				Pattern: "^ERROR$",
				Action:  ActionMark,
			},
		},
		CreatedAt: time.Now().Add(-96 * time.Hour),
		UpdatedAt: time.Now().Add(-48 * time.Hour),
		Author:    "admin",
		Version:   "v1.0.0",
	}
	for id, config := range filterConfigs {
		filterConfigHistory[id] = []FilterConfigVersion{
			{Version: config.Version, CreatedAt: config.CreatedAt, Author: config.Author},
		}
	}
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

	// API v1 路由组
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

		// 解析器配置管理
		parsers := v1.Group("/parsers")
		{
			parsers.GET("", listParserConfigs)
			parsers.POST("", createParserConfig)
			parsers.GET("/:id", getParserConfig)
			parsers.PUT("/:id", updateParserConfig)
			parsers.DELETE("/:id", deleteParserConfig)
			parsers.GET("/:id/history", getParserConfigHistory)
		}

		// 转换规则管理
		transforms := v1.Group("/transforms")
		{
			transforms.GET("", listTransformRules)
			transforms.POST("", createTransformRule)
			transforms.GET("/:id", getTransformRule)
			transforms.PUT("/:id", updateTransformRule)
			transforms.DELETE("/:id", deleteTransformRule)
			transforms.GET("/:id/history", getTransformRuleHistory)
		}

		// 过滤器配置管理
		filters := v1.Group("/filters")
		{
			filters.GET("", listFilterConfigs)
			filters.POST("", createFilterConfig)
			filters.GET("/:id", getFilterConfig)
			filters.PUT("/:id", updateFilterConfig)
			filters.DELETE("/:id", deleteFilterConfig)
			filters.GET("/:id/history", getFilterConfigHistory)
		}

		// 配置验证
		v1.POST("/validate/parser", validateParserConfig)
		v1.POST("/validate/transform", validateTransformRule)
		v1.POST("/validate/filter", validateFilterConfig)

		// 配置推送通知 (WebSocket 长轮询)
		v1.GET("/watch", watchConfigChanges)

		// 系统信息
		v1.GET("/info", getSystemInfo)
		v1.GET("/health", healthCheck)
	}

	return r
}

// ============== 策略管理处理器 ==============

// listStrategies 获取所有策略
func listStrategies(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

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
	req.Author = getAuthorFromContext(c)

	// 验证规则
	if err := validateStrategy(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid strategy: " + err.Error(),
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// 保存策略
	strategies[id] = req
	strategyHistory[id] = []StrategyVersion{
		{Version: req.Version, CreatedAt: req.CreatedAt, Author: req.Author},
	}

	// 推送配置更新事件
	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "created",
		ConfigType: "strategy",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "strategy created",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// getStrategy 获取单个策略
func getStrategy(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

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

	mu.Lock()
	defer mu.Unlock()

	existing, exists := strategies[id]
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
	newVersion := incrementVersion(existing.Version)
	req.Version = newVersion
	req.UpdatedAt = time.Now()
	req.Author = getAuthorFromContext(c)
	req.ID = id
	req.CreatedAt = existing.CreatedAt

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

	// 推送配置更新事件
	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "updated",
		ConfigType: "strategy",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "strategy updated",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// deleteStrategy 删除策略
func deleteStrategy(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := strategies[id]; !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "strategy not found",
		})
		return
	}

	delete(strategies, id)
	history := strategyHistory[id]
	delete(strategyHistory, id)

	// 推送配置更新事件
	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "deleted",
		ConfigType: "strategy",
		ID:         id,
		Data:       map[string]interface{}{"history": history},
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "strategy deleted",
	})
}

// getStrategyHistory 获取策略历史
func getStrategyHistory(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

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

// ============== 解析器配置处理器 ==============

// listParserConfigs 获取所有解析器配置
func listParserConfigs(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]ParserConfig, 0, len(parserConfigs))
	for _, config := range parserConfigs {
		list = append(list, config)
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    list,
	})
}

// createParserConfig 创建解析器配置
func createParserConfig(c *gin.Context) {
	var req ParserConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	id := "parser-" + generateID()
	req.ID = id
	req.Version = "v1.0.0"
	req.Enabled = true
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Author = getAuthorFromContext(c)

	if err := validateParserConfigInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid parser config: " + err.Error(),
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	parserConfigs[id] = req
	parserConfigHistory[id] = []ParserConfigVersion{
		{Version: req.Version, CreatedAt: req.CreatedAt, Author: req.Author, Content: hashContent(req)},
	}

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "created",
		ConfigType: "parser",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "parser config created",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// getParserConfig 获取单个解析器配置
func getParserConfig(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	config, exists := parserConfigs[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "parser config not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    config,
	})
}

// updateParserConfig 更新解析器配置
func updateParserConfig(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	existing, exists := parserConfigs[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "parser config not found",
		})
		return
	}

	var req ParserConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	newVersion := incrementVersion(existing.Version)
	req.Version = newVersion
	req.UpdatedAt = time.Now()
	req.Author = getAuthorFromContext(c)
	req.ID = id
	req.CreatedAt = existing.CreatedAt

	if err := validateParserConfigInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid parser config: " + err.Error(),
		})
		return
	}

	parserConfigs[id] = req
	parserConfigHistory[id] = append(parserConfigHistory[id], ParserConfigVersion{
		Version:   req.Version,
		CreatedAt: req.UpdatedAt,
		Author:    req.Author,
		Content:   hashContent(req),
	})

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "updated",
		ConfigType: "parser",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "parser config updated",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// deleteParserConfig 删除解析器配置
func deleteParserConfig(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := parserConfigs[id]; !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "parser config not found",
		})
		return
	}

	history := parserConfigHistory[id]
	delete(parserConfigs, id)
	delete(parserConfigHistory, id)

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "deleted",
		ConfigType: "parser",
		ID:         id,
		Data:       map[string]interface{}{"history": history},
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "parser config deleted",
	})
}

// getParserConfigHistory 获取解析器配置历史
func getParserConfigHistory(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	history, exists := parserConfigHistory[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "parser config not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    history,
	})
}

// ============== 转换规则处理器 ==============

// listTransformRules 获取所有转换规则
func listTransformRules(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]TransformRuleConfig, 0, len(transformRules))
	for _, config := range transformRules {
		list = append(list, config)
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    list,
	})
}

// createTransformRule 创建转换规则
func createTransformRule(c *gin.Context) {
	var req TransformRuleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	id := "transform-" + generateID()
	req.ID = id
	req.Version = "v1.0.0"
	req.Enabled = true
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Author = getAuthorFromContext(c)

	if err := validateTransformRuleInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid transform rule: " + err.Error(),
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	transformRules[id] = req
	transformRuleHistory[id] = []TransformRuleVersion{
		{Version: req.Version, CreatedAt: req.CreatedAt, Author: req.Author},
	}

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "created",
		ConfigType: "transform",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "transform rule created",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// getTransformRule 获取单个转换规则
func getTransformRule(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	config, exists := transformRules[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "transform rule not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    config,
	})
}

// updateTransformRule 更新转换规则
func updateTransformRule(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	existing, exists := transformRules[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "transform rule not found",
		})
		return
	}

	var req TransformRuleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	newVersion := incrementVersion(existing.Version)
	req.Version = newVersion
	req.UpdatedAt = time.Now()
	req.Author = getAuthorFromContext(c)
	req.ID = id
	req.CreatedAt = existing.CreatedAt

	if err := validateTransformRuleInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid transform rule: " + err.Error(),
		})
		return
	}

	transformRules[id] = req
	transformRuleHistory[id] = append(transformRuleHistory[id], TransformRuleVersion{
		Version:   req.Version,
		CreatedAt: req.UpdatedAt,
		Author:    req.Author,
	})

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "updated",
		ConfigType: "transform",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "transform rule updated",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// deleteTransformRule 删除转换规则
func deleteTransformRule(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := transformRules[id]; !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "transform rule not found",
		})
		return
	}

	history := transformRuleHistory[id]
	delete(transformRules, id)
	delete(transformRuleHistory, id)

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "deleted",
		ConfigType: "transform",
		ID:         id,
		Data:       map[string]interface{}{"history": history},
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "transform rule deleted",
	})
}

// getTransformRuleHistory 获取转换规则历史
func getTransformRuleHistory(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	history, exists := transformRuleHistory[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "transform rule not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    history,
	})
}

// ============== 过滤器配置处理器 ==============

// listFilterConfigs 获取所有过滤器配置
func listFilterConfigs(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]FilterConfig, 0, len(filterConfigs))
	for _, config := range filterConfigs {
		list = append(list, config)
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    list,
	})
}

// createFilterConfig 创建过滤器配置
func createFilterConfig(c *gin.Context) {
	var req FilterConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	id := "filter-" + generateID()
	req.ID = id
	req.Version = "v1.0.0"
	req.Enabled = true
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Author = getAuthorFromContext(c)

	if err := validateFilterConfigInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid filter config: " + err.Error(),
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	filterConfigs[id] = req
	filterConfigHistory[id] = []FilterConfigVersion{
		{Version: req.Version, CreatedAt: req.CreatedAt, Author: req.Author},
	}

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "created",
		ConfigType: "filter",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "filter config created",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// getFilterConfig 获取单个过滤器配置
func getFilterConfig(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	config, exists := filterConfigs[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "filter config not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    config,
	})
}

// updateFilterConfig 更新过滤器配置
func updateFilterConfig(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	existing, exists := filterConfigs[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "filter config not found",
		})
		return
	}

	var req FilterConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	newVersion := incrementVersion(existing.Version)
	req.Version = newVersion
	req.UpdatedAt = time.Now()
	req.Author = getAuthorFromContext(c)
	req.ID = id
	req.CreatedAt = existing.CreatedAt

	if err := validateFilterConfigInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid filter config: " + err.Error(),
		})
		return
	}

	filterConfigs[id] = req
	filterConfigHistory[id] = append(filterConfigHistory[id], FilterConfigVersion{
		Version:   req.Version,
		CreatedAt: req.UpdatedAt,
		Author:    req.Author,
	})

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "updated",
		ConfigType: "filter",
		ID:         id,
		Data:       req,
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "filter config updated",
		Data:    map[string]string{"id": id, "version": req.Version},
	})
}

// deleteFilterConfig 删除过滤器配置
func deleteFilterConfig(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := filterConfigs[id]; !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "filter config not found",
		})
		return
	}

	history := filterConfigHistory[id]
	delete(filterConfigs, id)
	delete(filterConfigHistory, id)

	pushConfigUpdate(ConfigUpdateEvent{
		Type:       "deleted",
		ConfigType: "filter",
		ID:         id,
		Data:       map[string]interface{}{"history": history},
		Timestamp:  time.Now(),
	})

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "filter config deleted",
	})
}

// getFilterConfigHistory 获取过滤器配置历史
func getFilterConfigHistory(c *gin.Context) {
	id := c.Param("id")

	mu.RLock()
	defer mu.RUnlock()

	history, exists := filterConfigHistory[id]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "filter config not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    history,
	})
}

// ============== 配置验证处理器 ==============

// validateParserConfig 验证解析器配置
func validateParserConfig(c *gin.Context) {
	var req ParserConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	if err := validateParserConfigInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "validation failed: " + err.Error(),
			Data:    map[string]string{"error": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "validation passed",
		Data:    map[string]bool{"valid": true},
	})
}

// validateTransformRule 验证转换规则
func validateTransformRule(c *gin.Context) {
	var req TransformRuleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	if err := validateTransformRuleInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "validation failed: " + err.Error(),
			Data:    map[string]string{"error": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "validation passed",
		Data:    map[string]bool{"valid": true},
	})
}

// validateFilterConfig 验证过滤器配置
func validateFilterConfig(c *gin.Context) {
	var req FilterConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body: " + err.Error(),
		})
		return
	}

	if err := validateFilterConfigInternal(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "validation failed: " + err.Error(),
			Data:    map[string]string{"error": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "validation passed",
		Data:    map[string]bool{"valid": true},
	})
}

// ============== 配置推送 ==============

// watchConfigChanges 监听配置变化 (长轮询)
func watchConfigChanges(c *gin.Context) {
	configType := c.Query("type") // parser, transform, filter, strategy
	configID := c.Query("id")

	// 创建临时通道
	done := make(chan ConfigUpdateEvent, 1)
	watcherID := fmt.Sprintf("%s-%s-%d", configType, configID, time.Now().UnixNano())

	mu.Lock()
	configWatchers[watcherID] = done
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(configWatchers, watcherID)
		close(done)
		mu.Unlock()
	}()

	// 设置超时
	ctx := c.Request.Context()
	select {
	case <-ctx.Done():
		c.JSON(http.StatusOK, Response{
			Code:    0,
			Message: "connection closed",
		})
		return
	case event := <-done:
		// 过滤事件
		if configType != "" && event.ConfigType != configType {
			c.JSON(http.StatusOK, Response{
				Code:    0,
				Message: "event filtered",
			})
			return
		}
		if configID != "" && event.ID != configID {
			c.JSON(http.StatusOK, Response{
				Code:    0,
				Message: "event filtered",
			})
			return
		}
		c.JSON(http.StatusOK, Response{
			Code:    0,
			Message: "config update",
			Data:    event,
		})
	case <-time.After(30 * time.Second):
		c.JSON(http.StatusOK, Response{
			Code:    0,
			Message: "timeout",
		})
	}
}

// pushConfigUpdate 推送配置更新事件
func pushConfigUpdate(event ConfigUpdateEvent) {
	mu.Lock()
	defer mu.Unlock()

	for _, ch := range configWatchers {
		select {
		case ch <- event:
		default:
			// 通道已满，跳过
		}
	}
}

// ============== 系统信息 ==============

// getSystemInfo 获取系统信息
func getSystemInfo(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"system":        "Log System Config Server",
			"version":       "v1.0.0",
			"uptime":        time.Now().Format(time.RFC3339),
			"config_stats": gin.H{
				"strategies":   len(strategies),
				"parsers":      len(parserConfigs),
				"transforms":   len(transformRules),
				"filters":      len(filterConfigs),
				"watchers":     len(configWatchers),
			},
		},
	})
}

// healthCheck 健康检查
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// ============== 验证函数 ==============

func validateStrategy(s Strategy) error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(s.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}
	for i, rule := range s.Rules {
		if rule.Condition == nil {
			return fmt.Errorf("rule[%d] condition is required", i)
		}
		if rule.Action == nil {
			return fmt.Errorf("rule[%d] action is required", i)
		}
	}
	return nil
}

func validateParserConfigInternal(config ParserConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if config.Type == "" {
		return fmt.Errorf("type is required")
	}
	validTypes := map[string]bool{
		"json": true, "key_value": true, "syslog": true,
		"apache": true, "nginx": true, "unstructured": true,
	}
	if !validTypes[config.Type] {
		return fmt.Errorf("invalid type: %s", config.Type)
	}
	return nil
}

func validateTransformRuleInternal(config TransformRuleConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(config.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}
	validExtractors := map[string]bool{
		"regex": true, "template": true, "jsonpath": true,
		"direct": true, "lowercase": true, "uppercase": true, "split": true,
	}
	for i, rule := range config.Rules {
		if rule.SourceField == "" {
			return fmt.Errorf("rule[%d] source_field is required", i)
		}
		if rule.TargetField == "" {
			return fmt.Errorf("rule[%d] target_field is required", i)
		}
		if rule.Extractor == "" {
			return fmt.Errorf("rule[%d] extractor is required", i)
		}
		if !validExtractors[rule.Extractor] {
			return fmt.Errorf("rule[%d] invalid extractor: %s", i, rule.Extractor)
		}
		if rule.Extractor == "regex" {
			if pattern, ok := rule.Config["pattern"].(string); ok {
				if _, err := regexp.Compile(pattern); err != nil {
					return fmt.Errorf("rule[%d] invalid regex pattern: %v", i, err)
				}
			}
		}
	}
	return nil
}

func validateFilterConfigInternal(config FilterConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(config.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}
	for i, rule := range config.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule[%d] name is required", i)
		}
		if rule.Field == "" {
			return fmt.Errorf("rule[%d] field is required", i)
		}
		if rule.Pattern == "" {
			return fmt.Errorf("rule[%d] pattern is required", i)
		}
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("rule[%d] invalid regex pattern: %v", i, err)
		}
	}
	return nil
}

// ============== 工具函数 ==============

func incrementVersion(version string) string {
	if len(version) < 2 || version[0] != 'v' {
		return "v1.0.0"
	}

	// 解析版本号 v1.0.0 -> 1.0.0
	parts := version[1:]
	var major, minor, patch int
	fmt.Sscanf(parts, "%d.%d.%d", &major, &minor, &patch)

	// 增加 minor 版本
	minor++
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

func generateID() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = "abcdefghijklmnopqrstuvwxyz0123456789"[time.Now().UnixNano()%36]
	}
	return string(b)
}

func getAuthorFromContext(c *gin.Context) string {
	author := c.GetHeader("X-Auth-User")
	if author == "" {
		author = "admin"
	}
	return author
}

func hashContent(config ParserConfig) string {
	data, _ := json.Marshal(config)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
