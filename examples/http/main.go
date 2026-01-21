// HTTP 服务示例，演示日志 SDK 使用
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"../log-sdk/log-sdk/pkg/logger"
	"../log-sdk/log-sdk/pkg/guard"
)

func main() {
	// 初始化日志 SDK
	logInstance := logger.New(logger.Config{
		EtcdEndpoints: []string{"localhost:2379"},
		KafkaBrokers:  []string{"localhost:9092"},
		KafkaTopic:    "logs-semantic",
		BatchSize:     1000,
		BatchTimeout:  100 * time.Millisecond,
		OTelEndpoint:  "http://localhost:4317",
		ServiceName:   "example-http",
	})
	defer logInstance.Close()

	// 创建 Gin 引擎
	r := gin.Default()

	// 添加日志拦截器
	r.Use(guard.GinMiddleware(logInstance))

	// 定义路由
	r.GET("/ping", handlePing)
	r.GET("/hello/:name", handleHello)
	r.POST("/order", handleOrder)

	// 启动服务
	log.Println("Starting HTTP server on :8080")
	r.Run(":8080")
}

func handlePing(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func handleHello(c *gin.Context) {
	name := c.Param("name")
	c.JSON(200, gin.H{
		"message": "Hello, " + name,
	})
}

func handleOrder(c *gin.Context) {
	type OrderRequest struct {
		UserID  string  `json:"user_id"`
		ProductID string  `json:"product_id"`
		Amount   float64 `json:"amount"`
	}

	var req OrderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 处理订单逻辑
	c.JSON(200, gin.H{
		"order_id":  "ORD-" + time.Now().Format("20060102150405"),
		"user_id":   req.UserID,
		"amount":    req.Amount,
		"status":    "created",
	})
}
