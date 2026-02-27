// Package main Log SDK服务入口
// 提供日志发送、配置管理API
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Config SDK配置
type Config struct {
	Port     string
	LogLevel string
}

// LogRequest 日志请求
type LogRequest struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// SDKServer HTTP服务器
type SDKServer struct {
	config Config
	mux    *http.ServeMux
}

// NewSDKServer 创建服务器
func NewSDKServer(cfg Config) *SDKServer {
	s := &SDKServer{
		config: cfg,
		mux:    http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes 设置路由
func (s *SDKServer) setupRoutes() {
	s.mux.HandleFunc("/healthz", s.healthHandler)
	s.mux.HandleFunc("/readyz", s.readyHandler)
	s.mux.HandleFunc("/api/v1/log", s.logHandler)
	s.mux.HandleFunc("/api/v1/batch", s.batchHandler)
}

// healthHandler 健康检查
func (s *SDKServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// readyHandler 就绪检查
func (s *SDKServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// logHandler 发送单条日志
func (s *SDKServer) logHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 模拟处理日志
	log.Printf("Received log: [%s] %s - %s", req.Level, req.Service, req.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// batchHandler 批量发送日志
func (s *SDKServer) batchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqs []LogRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 模拟处理日志
	log.Printf("Received %d logs in batch", len(reqs))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "ok",
		"processed":  fmt.Sprintf("%d", len(reqs)),
	})
}

// Start 启动服务器
func (s *SDKServer) Start() error {
	log.Printf("Starting Log SDK server on port %s", s.config.Port)
	return http.ListenAndServe(":"+s.config.Port, s.mux)
}

func main() {
	// 加载配置
	cfg := Config{
		Port:     getEnvString("PORT", "8080"),
		LogLevel: getEnvString("LOG_LEVEL", "info"),
	}

	// 创建服务器
	server := NewSDKServer(cfg)

	// 启动服务器
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
