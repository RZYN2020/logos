package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/log-system/log-sdk/pkg/api"
)

func main() {
	// 创建 API 处理器
	router := api.New()

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
