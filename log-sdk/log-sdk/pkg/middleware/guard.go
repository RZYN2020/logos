package middleware

import (
	"time"
	"github.com/gin-gonic/gin"
)

type Logger interface {
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
}

func GinMiddleware(log Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		userID, _ := c.Get("user_id")
		if userID == nil {
			userID = "anonymous"
		}

		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = c.GetString("trace_id")
		}

		fields := map[string]interface{}{
			"http_method": method,
			"http_path":   path,
			"http_status": status,
			"duration_ms": duration.Milliseconds(),
			"client_ip":   c.ClientIP(),
			"user_id":     userID,
			"trace_id":    traceID,
		}

		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		if status >= 500 {
			log.Error("HTTP request", fields)
		} else if status >= 400 {
			log.Warn("HTTP request", fields)
		} else {
			log.Info("HTTP request", fields)
		}
	}
}
