# Log SDK Specification

## 1. Overview

The Log SDK is a lightweight, high-performance logging library designed for distributed systems. It provides a simple API for capturing structured logs, supports dynamic policy configuration via Etcd, and integrates with distributed tracing systems like OpenTelemetry.

## 2. Core Concepts

### 2.1 Log Levels

| Level      | Description                          | Severity Number |
|------------|--------------------------------------|-----------------|
| Debug      | Debugging information                | 5               |
| Info       | Informational messages               | 9               |
| Warn       | Warning conditions                   | 13              |
| Error      | Error conditions                     | 17              |
| Fatal      | Fatal errors (causes program exit)   | 21              |
| Panic      | Panic conditions (causes panic)      | 25              |

### 2.2 Log Entry Structure

```go
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Service       string                 `json:"service"`
	Cluster       string                 `json:"cluster,omitempty"`
	Pod           string                 `json:"pod,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	// Internal fields
	File          string                 `json:"-"`
	Line          int                    `json:"-"`
	Function      string                 `json:"-"`
}
```

### 2.3 Hook Interface

```go
type Hook interface {
	OnLog(entry LogEntry) bool
}
```

## 3. Configuration

### 3.1 Basic Config

```go
type Config struct {
	ServiceName         string
	Environment         string
	Cluster             string
	Pod                 string
	KafkaBrokers        []string
	KafkaTopic          string
	EtcdEndpoints       []string
	BatchSize           int
	BatchTimeout        time.Duration
	FallbackToConsole   bool
	MaxBufferSize       int
}
```

### 3.2 Etcd Configuration

```yaml
rules:
  - name: "production-error-filter"
    condition:
      level: "ERROR"
      environment: "production"
      cluster: "cluster-1"
      pod: "api-*"
    action:
      enabled: true
      sampling: 1.0
```

## 4. Usage

### 4.1 Basic Usage

```go
package main

import (
	"github.com/log-system/log-sdk/pkg/logger"
)

func main() {
	log, err := logger.New(logger.Config{
		ServiceName: "example-service",
		Environment: "dev",
		Cluster:     "local",
		Pod:         "example-pod-1",
	})
	if err != nil {
		panic(err)
	}
	defer log.Close()

	log.Info("Hello World",
		logger.F("user_id", "123"),
		logger.F("product_id", "abc"))

	err = someFunction()
	if err != nil {
		log.Error("Failed to execute function",
			logger.F("error", err.Error()),
			logger.F("function", "someFunction"))
	}
}
```

### 4.2 Custom Hooks

```go
package main

import (
	"github.com/log-system/log-sdk/pkg/logger"
	"github.com/log-system/log-sdk/pkg/hook"
	"time"
)

func main() {
	log, err := logger.New(logger.Config{
		ServiceName: "api-service",
		Environment: "production",
		Cluster:     "cluster-1",
		Pod:         "api-pod-123",
		EtcdEndpoints:    []string{"http://localhost:2379"},
	})

	log.AddHook(hook.LevelHook(logger.LevelWarn))
	log.AddHook(hook.RegexHook("pod", "api-*"))

	log.Warn("High latency",
		logger.F("latency", "1.5s"),
		logger.F("path", "/api/v1/orders"))
}
```

### 4.3 Context Propagation

```go
func handleRequest(c *gin.Context) {
	// Extract context from request
	ctx := c.Request.Context()

	// Create log entry with context
	log.WithContext(ctx).Info("Handling request",
		logger.F("method", c.Request.Method),
		logger.F("path", c.Request.URL.Path))
}
```

## 5. Performance Characteristics

- **Zero Allocation**: Object pool for log entries
- **High Throughput**: Asynchronous batch sending
- **Backpressure Protection**: Ring buffer prevents blocking
- **Low Latency**: Efficient encoding and network handling

## 6. Error Handling

- **Kafka Unavailable**: Fallback to console
- **Etcd Unavailable**: Use default strategy
- **Buffer Full**: Discard oldest messages

## 7. Integration

### 7.1 With Gin

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/log-system/log-sdk/pkg/guard"
	"github.com/log-system/log-sdk/pkg/logger"
)

func main() {
	router := gin.Default()

	log, err := logger.New(logger.Config{
		ServiceName: "api-service",
	})
	if err != nil {
		panic(err)
	}
	defer log.Close()

	router.Use(guard.New(log))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	router.Run(":8080")
}
```

### 7.2 With Etcd

```go
log, err := logger.New(logger.Config{
	EtcdEndpoints: []string{"http://localhost:2379"},
})
if err != nil {
	panic(err)
}
```
