// Package main 日志分析器服务入口
// 提供日志查询、规则配置和管理 API
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/log-system/log-analyzer/internal/etcd"
	"github.com/log-system/log-analyzer/internal/handlers"
	"github.com/log-system/log-analyzer/internal/middleware"
	"github.com/log-system/log-analyzer/internal/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config 服务器配置
type Config struct {
	ESAddresses   []string
	PostgresDSN   string
	ETCDEndpoints []string
	Port          string
	LogLevel      string
}

// Server HTTP 服务器
type Server struct {
	config          Config
	engine          *gin.Engine
	db              *gorm.DB
	etcdCli         *etcd.Client
	ruleHandler     *handlers.RuleHandler
	analysisHandler *handlers.AnalysisHandler
	reportHandler   *handlers.ReportHandler
	authHandler     *handlers.AuthHandler
	alertHandler    *handlers.AlertHandler
	authConfig      *middleware.AuthConfig
}

// NewServer 创建服务器
func NewServer(cfg Config) (*Server, error) {
	// 初始化数据库
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 初始化 ETCD 客户端
	etcdCli, err := etcd.NewClient(etcd.Config{
		Endpoints:   cfg.ETCDEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	// 自动迁移数据模型
	if err := migrations.Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// 创建认证配置和处理器
	authConfig := middleware.NewAuthConfig()
	authHandler := handlers.NewAuthHandler(authConfig)

	// 创建处理器
	ruleHandler := handlers.NewRuleHandler(db, etcdCli)
	analysisHandler := handlers.NewAnalysisHandler()
	reportHandler := handlers.NewReportHandler(db)
	alertHandler := handlers.NewAlertHandler(db)

	s := &Server{
		config:          cfg,
		engine:          gin.Default(),
		db:              db,
		etcdCli:         etcdCli,
		ruleHandler:     ruleHandler,
		analysisHandler: analysisHandler,
		reportHandler:   reportHandler,
		authHandler:     authHandler,
		alertHandler:    alertHandler,
		authConfig:      authConfig,
	}

	s.setupRoutes()
	return s, nil
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// CORS 中间件
	s.engine.Use(middleware.CorsMiddleware())

	// 健康检查
	s.engine.GET("/health", s.healthHandler)
	s.engine.GET("/ready", s.readyHandler)

	// API v1 路由组
	v1 := s.engine.Group("/api/v1")
	{
		// 系统信息
		v1.GET("/info", s.systemInfoHandler)

		// 日志查询 API
		v1.POST("/query", s.queryHandler)
		v1.GET("/search", s.searchHandler)
		v1.POST("/aggregate", s.aggregateHandler)

		// 认证 API (公开)
		v1.POST("/auth/login", s.authHandler.Login)
		v1.POST("/auth/register", s.authHandler.Register)

		// 认证 API (需要认证)
		auth := v1.Group("")
		auth.Use(s.authConfig.AuthMiddleware())
		{
			// 用户管理
			auth.GET("/user", s.authHandler.GetCurrentUser)
			auth.PUT("/user/password", s.authHandler.ChangePassword)
			auth.GET("/users", s.authConfig.RequireRole("admin"), s.authHandler.ListUsers)

			// 规则管理 API
			auth.GET("/rules", s.ruleHandler.ListRules)
			auth.GET("/rules/:id", s.ruleHandler.GetRule)
			auth.POST("/rules", s.ruleHandler.CreateRule)
			auth.PUT("/rules/:id", s.ruleHandler.UpdateRule)
			auth.DELETE("/rules/:id", s.ruleHandler.DeleteRule)
			auth.GET("/rules/:id/history", s.ruleHandler.GetRuleHistory)
			auth.POST("/rules/:id/rollback/:version", s.ruleHandler.RollbackRule)
			auth.POST("/rules/:id/validate", s.ruleHandler.ValidateRule)
			auth.POST("/rules/:id/test", s.ruleHandler.TestRule)
			auth.GET("/rules/export", s.ruleHandler.ExportRules)
			auth.POST("/rules/import", s.ruleHandler.ImportRules)

			// 策略管理 API (保留兼容)
			auth.GET("/strategies", s.ruleHandler.ListRules)
			auth.GET("/strategies/:id", s.ruleHandler.GetRule)
			auth.POST("/strategies", s.ruleHandler.CreateRule)
			auth.PUT("/strategies/:id", s.ruleHandler.UpdateRule)
			auth.DELETE("/strategies/:id", s.ruleHandler.DeleteRule)

			// 日志分析 API
			auth.POST("/analysis/mine", s.analysisHandler.MinePatterns)
			auth.POST("/analysis/anomalies", s.analysisHandler.DetectAnomalies)
			auth.POST("/analysis/cluster", s.analysisHandler.ClusterLogs)
			auth.POST("/analysis/recommend", s.analysisHandler.RecommendRules)
			auth.GET("/analysis/pattern-types", s.analysisHandler.GetPatternTypes)

			// 日志报告 API - 新增
			auth.GET("/report/:service", s.reportHandler.GetReport)
			auth.GET("/report/:service/top-lines", s.reportHandler.GetTopLines)
			auth.GET("/report/:service/top-patterns", s.reportHandler.GetTopPatterns)

			// 日志摄入 API - 新增
			auth.POST("/logs", s.reportHandler.IngestLog)
			auth.POST("/logs/batch", s.reportHandler.IngestBatch)
			auth.POST("/logs/query", s.reportHandler.QueryLogs)

			// 告警管理 API - 新增
			auth.GET("/alerts/rules", s.alertHandler.ListAlertRules)
			auth.POST("/alerts/rules", s.alertHandler.CreateAlertRule)
			auth.PUT("/alerts/rules/:id", s.alertHandler.UpdateAlertRule)
			auth.DELETE("/alerts/rules/:id", s.alertHandler.DeleteAlertRule)
			auth.GET("/alerts/history", s.alertHandler.ListAlertHistory)
			auth.PUT("/alerts/history/:id/resolve", s.alertHandler.ResolveAlert)
		}
	}
}

// healthHandler 健康检查
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// readyHandler 就绪检查
func (s *Server) readyHandler(c *gin.Context) {
	// 检查数据库连接
	sqlDB, err := s.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// systemInfoHandler 系统信息
func (s *Server) systemInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"system":       "Log Analyzer",
		"version":      "1.0.0",
		"etcd_version": "3.5.9",
		"uptime":       time.Since(startTime).String(),
	})
}

// queryHandler 查询日志
func (s *Server) queryHandler(c *gin.Context) {
	var req struct {
		Query   string            `json:"query"`
		Filters map[string]string `json:"filters"`
		Limit   int               `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认 limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	// 模拟查询结果（实际应连接 Elasticsearch）
	// 这里返回示例数据用于演示
	entries := []map[string]interface{}{
		{
			"timestamp": "2024-01-22T10:00:00Z",
			"level":     "INFO",
			"service":   "api-gateway",
			"message":   "Request received",
			"trace_id":  "7a3c9f8d5e2b1a4",
		},
		{
			"timestamp": "2024-01-22T10:00:01Z",
			"level":     "ERROR",
			"service":   "payment-service",
			"message":   "Payment processing failed",
			"trace_id":  "7a3c9f8d5e2b1a5",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(entries),
		"entries": entries,
	})
}

// searchHandler 搜索日志
func (s *Server) searchHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing query parameter"})
		return
	}

	// 模拟搜索结果
	entries := []map[string]interface{}{
		{
			"timestamp": "2024-01-22T10:00:00Z",
			"level":     "INFO",
			"service":   "api-gateway",
			"message":   query,
			"trace_id":  "7a3c9f8d5e2b1a4",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   1,
		"entries": entries,
	})
}

// aggregateHandler 聚合分析
func (s *Server) aggregateHandler(c *gin.Context) {
	var req struct {
		Field    string   `json:"field"`
		Interval string   `json:"interval"`
		Filters  []string `json:"filters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 模拟聚合分析结果
	c.JSON(http.StatusOK, gin.H{
		"total_logs":   1000,
		"error_rate":   0.05,
		"top_services": []string{"api-gateway", "user-service", "payment-service"},
		"by_level": map[string]int{
			"INFO":  700,
			"WARN":  150,
			"ERROR": 100,
			"FATAL": 30,
			"PANIC": 20,
		},
	})
}

var startTime = time.Now()

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.engine,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		s.etcdCli.Close()
	}()

	log.Printf("Starting log analyzer server on port %s", s.config.Port)
	return server.ListenAndServe()
}

// 环境变量辅助函数
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvStrings(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// 简单处理：按逗号分割
		return []string{value}
	}
	return defaultValue
}

func main() {
	// 加载配置
	cfg := Config{
		ESAddresses:   getEnvStrings("ES_ADDRESSES", []string{"http://localhost:9200"}),
		PostgresDSN:   getEnvString("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=logAnalyzer port=5432 sslmode=disable"),
		ETCDEndpoints: getEnvStrings("ETCD_ENDPOINTS", []string{"localhost:2379"}),
		Port:          getEnvString("PORT", "8080"),
		LogLevel:      getEnvString("LOG_LEVEL", "info"),
	}

	// 创建服务器
	server, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 设置优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// 启动服务器
	if err := server.Start(ctx); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
