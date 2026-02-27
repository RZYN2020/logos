package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// 创建 API 处理器
	router := http.NewServeMux()

	// 健康检查
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// 就绪检查
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	// API路由
	router.HandleFunc("/api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		config := map[string]interface{}{
			"version": "1.0.0",
			"services": []string{
				"log-processor",
				"log-analyzer",
				"log-sdk",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	})

	// 启动服务器
	port := ":8080"
	log.Printf("Config Server starting on %s", port)
	fmt.Println(`
╔─────────────────────────────────────────────╗
║                                                   ║
║   🚀  Log System Config Server                    ║
║                                                   ║
║   📡 API:        http://localhost:8080/api/v1      ║
║   💊 Health:      http://localhost:8080/health    ║
║   📋 System:      http://localhost:8080/info     ║
║                                                   ║
╚─────────────────────────────────────────────╝
`)

	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
