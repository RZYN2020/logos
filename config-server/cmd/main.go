package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/log-system/log-processor/config-server/pkg/api"
)

var (
	port    string
	version string
)

func init() {
	flag.StringVar(&port, "port", ":8080", "Server port")
	flag.StringVar(&version, "version", "dev", "Server version")
	flag.Parse()
}

func main() {
	// 创建 API 处理器
	router := api.New()

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown error: %v", err)
		}
	}()

	// 启动服务器
	log.Printf("Config Server starting on %s (version: %s)", port, version)
	fmt.Printf(`
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   🚀  Log System Config Server                            ║
║                                                           ║
║   Version: %s                                          ║
║   Port: %s                                           ║
║                                                           ║
║   API Endpoints:                                          ║
║   ─────────────────────────────────────────────────────   ║
║   📋 Parsers:     GET/POST /api/v1/parsers                ║
║   🔧 Transforms:  GET/POST /api/v1/transforms             ║
║   🛡️  Filters:     GET/POST /api/v1/filters                ║
║   📜 Strategies:  GET/POST /api/v1/strategies             ║
║                                                           ║
║   Validation:                                             ║
║   ─────────────────────────────────────────────────────   ║
║   POST /api/v1/validate/parser                            ║
║   POST /api/v1/validate/transform                         ║
║   POST /api/v1/validate/filter                            ║
║                                                           ║
║   Watch Changes:                                          ║
║   ─────────────────────────────────────────────────────   ║
║   GET  /api/v1/watch?type=parser&id=<id>                  ║
║                                                           ║
║   Health & Info:                                          ║
║   ─────────────────────────────────────────────────────   ║
║   💊 Health:      GET /health                             ║
║   📊 Info:        GET /api/v1/info                        ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

`, version, port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
