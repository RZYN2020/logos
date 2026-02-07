package guard

import (
	"github.com/gin-gonic/gin"
	"github.com/log-system/log-sdk/pkg/logger"
	"time"
)

// GinMiddleware Gin框架的日志拦截器
func GinMiddleware(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 记录日志
		duration := time.Since(start)
		status := c.Writer.Status()

		// 提取用户ID（如果有）
		userID, _ := c.Get("user_id")
		if userID == nil {
			userID = "anonymous"
		}

		// 提取trace信息（如果有）
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = c.GetString("trace_id")
		}

		// 构建日志字段
		fields := []logger.Field{
			logger.F("http_method", method),
			logger.F("http_path", path),
			logger.F("http_status", status),
			logger.F("duration_ms", duration.Milliseconds()),
			logger.F("client_ip", c.ClientIP()),
			logger.F("user_id", userID),
			logger.F("trace_id", traceID),
		}

		// 添加错误信息
		if len(c.Errors) > 0 {
			fields = append(fields, logger.F("errors", c.Errors.String()))
		}

		// 根据状态码选择日志级别
		if status >= 500 {
			log.Error("HTTP request", fields...)
		} else if status >= 400 {
			log.Warn("HTTP request", fields...)
		} else {
			log.Info("HTTP request", fields...)
		}
	}
}

// HTTPMiddleware 标准http.Handler的拦截器（用于非Gin框架）
// 简化实现，实际使用时需要适配具体框架
