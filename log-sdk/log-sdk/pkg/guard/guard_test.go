package guard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/log-system/log-sdk/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestGinMiddleware(t *testing.T) {
	// 初始化 gin 测试模式
	gin.SetMode(gin.TestMode)

	// 创建测试 logger
	log := logger.New(logger.Config{
		ServiceName:       "test-service",
		Environment:       "test",
		FallbackToConsole: true,
	})
	defer log.Close()

	// 创建测试路由
	router := gin.New()
	router.Use(GinMiddleware(log))

	// 测试正常请求
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 执行请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "test-trace-123")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGinMiddleware_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log := logger.New(logger.Config{
		ServiceName:       "test-service",
		Environment:       "test",
		FallbackToConsole: true,
	})
	defer log.Close()

	router := gin.New()
	router.Use(GinMiddleware(log))

	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGinMiddleware_WithUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log := logger.New(logger.Config{
		ServiceName:       "test-service",
		Environment:       "test",
		FallbackToConsole: true,
	})
	defer log.Close()

	router := gin.New()
	router.Use(GinMiddleware(log))

	router.GET("/auth", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.JSON(http.StatusOK, gin.H{"authenticated": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
