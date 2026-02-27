// Package main 日志分析器服务入口
// 提供日志查询、分析和聚合API
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config 分析器配置
type Config struct {
	ESAddresses []string
	Port        string
	LogLevel    string
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp       time.Time              `json:"timestamp"`
	Level           string                 `json:"level"`
	Message         string                 `json:"message"`
	Service         string                 `json:"service"`
	TraceID         string                 `json:"trace_id"`
	SpanID          string                 `json:"span_id"`
	HTTPMethod      string                 `json:"http_method,omitempty"`
	HTTPPath        string                 `json:"http_path,omitempty"`
	HTTPStatus      int                    `json:"http_status,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	BusinessDomain  string                 `json:"business_domain,omitempty"`
	TenantID        string                 `json:"tenant_id,omitempty"`
	IsError         bool                   `json:"is_error"`
	ErrorType       string                 `json:"error_type,omitempty"`
	Fields          map[string]interface{} `json:"fields,omitempty"`
}

// QueryRequest 查询请求
type QueryRequest struct {
	Query     string            `json:"query"`
	Filters   map[string]string `json:"filters,omitempty"`
	TimeRange struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"time_range"`
	Limit int `json:"limit"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	Total   int         `json:"total"`
	Entries []LogEntry  `json:"entries"`
}

// Server HTTP服务器
type Server struct {
	config Config
	mux    *http.ServeMux
}

// NewServer 创建服务器
func NewServer(cfg Config) *Server {
	s := &Server{
		config: cfg,
		mux:    http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.healthHandler)
	s.mux.HandleFunc("/ready", s.readyHandler)
	s.mux.HandleFunc("/api/v1/query", s.queryHandler)
	s.mux.HandleFunc("/api/v1/search", s.searchHandler)
	s.mux.HandleFunc("/api/v1/aggregate", s.aggregateHandler)
}

// healthHandler 健康检查
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// readyHandler 就绪检查
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// queryHandler 查询日志
func (s *Server) queryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 模拟查询结果
	resp := QueryResponse{
		Total: 0,
		Entries: []LogEntry{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// searchHandler 搜索日志
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	// 模拟搜索结果
	resp := QueryResponse{
		Total: 0,
		Entries: []LogEntry{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// aggregateHandler 聚合分析
func (s *Server) aggregateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 模拟聚合结果
	result := map[string]interface{}{
		"total_logs": 0,
		"error_rate": 0.0,
		"top_services": []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("Starting log analyzer server on port %s", s.config.Port)
	return server.ListenAndServe()
}

func main() {
	// 加载配置
	cfg := Config{
		ESAddresses: getEnvStrings("ES_ADDRESSES", []string{"http://localhost:9200"}),
		Port:        getEnvString("PORT", "8080"),
		LogLevel:    getEnvString("LOG_LEVEL", "info"),
	}

	// 创建服务器
	server := NewServer(cfg)

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

// 环境变量辅助函数
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvStrings(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		var result []string
		if err := json.Unmarshal([]byte(value), &result); err == nil {
			return result
		}
		return []string{value}
	}
	return defaultValue
}
