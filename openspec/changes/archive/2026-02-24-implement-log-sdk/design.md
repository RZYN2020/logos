# Log SDK 详细设计

## 1. 接口设计

### Logger API

```go
// Logger 是日志记录的核心接口
type Logger interface {
	// 传统打印方式
	Printf(format string, args ...interface{})
	Println(args ...interface{})
	Print(args ...interface{})

	// 强类型链式打印方式
	Debug(msg string, fields ...Field) *LogBuilder
	Info(msg string, fields ...Field) *LogBuilder
	Warn(msg string, fields ...Field) *LogBuilder
	Error(msg string, fields ...Field) *LogBuilder
	Fatal(msg string, fields ...Field) *LogBuilder
	Panic(msg string, fields ...Field) *LogBuilder

	// 上下文和字段管理
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger

	// Hook 管理
	AddHook(hook Hook) Logger

	// 生命周期管理
	Close() error
}

// LogBuilder 用于强类型链式打印
type LogBuilder struct {
	logger *loggerImpl
	entry  LogEntry
}

func (b *LogBuilder) Str(key, value string) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Int(key string, value int) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Int64(key string, value int64) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Float64(key string, value float64) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Bool(key string, value bool) *LogBuilder {
	b.entry.Fields[key] = value
	return b
}

func (b *LogBuilder) Send() {
	b.logger.logEntry(b.entry)
}
```

### Log Entry 结构

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
	// 内部字段（用于 Hook 过滤）
	File          string                 `json:"-"`  // 文件路径
	Line          int                    `json:"-"`  // 行号
	Function      string                 `json:"-"`  // 函数名
}
```

## 2. Hook 系统

### 接口定义

```go
type Hook interface {
	OnLog(entry LogEntry) bool
}
```

### 内置 Hook

```go
// 等级过滤 Hook
func LevelHook(minLevel Level) Hook {
	return func(entry LogEntry) bool {
		return entry.Level >= minLevel.String()
	}
}

// 行号过滤 Hook
func LineHook(min, max int) Hook {
	return func(entry LogEntry) bool {
		return entry.Line >= min && entry.Line <= max
	}
}

// 正则表达式过滤 Hook
func RegexHook(field string, pattern string) Hook {
	reg := regexp.MustCompile(pattern)
	return func(entry LogEntry) bool {
		switch field {
		case "cluster":
			return reg.MatchString(entry.Cluster)
		case "pod":
			return reg.MatchString(entry.Pod)
		case "file":
			return reg.MatchString(entry.File)
		default:
			return true
		}
	}
}
```

## 3. 策略引擎

### 配置结构

```go
type StrategyConfig struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	Name       string    `json:"name"`
	Condition  Condition `json:"condition"`
	Action     Action    `json:"action"`
}

type Condition struct {
	Level       string `json:"level,omitempty"`       // 日志级别
	Cluster     string `json:"cluster,omitempty"`     // 集群 ID
	Pod         string `json:"pod,omitempty"`         // Pod ID
	Environment string `json:"environment,omitempty"` // 环境
}

type Action struct {
	Enabled   bool    `json:"enabled"`                // 是否启用该规则
	Sampling  float64 `json:"sampling"`               // 采样率 (0.0 - 1.0)
}
```

### Etcd 配置管理

```go
type Engine struct {
	etcdEndpoints []string
	client        *clientv3.Client
	watcher       clientv3.Watcher
	config        StrategyConfig
	mu            sync.RWMutex
}

func NewEngine(endpoints []string) (*Engine, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		return nil, err
	}

	engine := &Engine{
		etcdEndpoints: endpoints,
		client:        client,
	}

	// 加载初始配置
	if err := engine.loadConfig(); err != nil {
		return nil, err
	}

	// 启动配置监控
	go engine.watchConfig()

	return engine, nil
}

func (e *Engine) loadConfig() error {
	resp, err := e.client.Get(context.Background(), "log-strategy")
	if err != nil {
		return err
	}

	var config StrategyConfig
	if err := json.Unmarshal(resp.Kvs[0].Value, &config); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = config
	return nil
}

func (e *Engine) watchConfig() {
	rch := e.client.Watch(context.Background(), "log-strategy")
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				var config StrategyConfig
				if err := json.Unmarshal(ev.Kv.Value, &config); err != nil {
					continue
				}
				e.mu.Lock()
				e.config = config
				e.mu.Unlock()
			}
		}
	}
}
```

## 4. 高性能实现

### 对象池

```go
var logEntryPool sync.Pool

func getLogEntry() *LogEntry {
	if entry, ok := logEntryPool.Get().(*LogEntry); ok {
		return entry
	}
	return &LogEntry{}
}

func putLogEntry(entry *LogEntry) {
	// 重置字段
	entry.Fields = map[string]interface{}{}
	logEntryPool.Put(entry)
}
```

### 异步发送

```go
type Producer struct {
	writer    *kafka.Writer
	buffer    chan []byte
	workerWg  sync.WaitGroup
	closeChan chan struct{}
}

func NewProducer(brokers []string, topic string) *Producer {
	p := &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
		buffer:    make(chan []byte, 10000),
		closeChan: make(chan struct{}),
	}

	// 启动工作协程
	for i := 0; i < 5; i++ {
		p.workerWg.Add(1)
		go p.worker()
	}

	return p
}

func (p *Producer) worker() {
	defer p.workerWg.Done()
	for {
		select {
		case data := <-p.buffer:
			err := p.writer.WriteMessages(context.Background(),
				kafka.Message{Value: data})
			if err != nil {
				println("Failed to send log to Kafka:", err.Error())
			}
		case <-p.closeChan:
			return
		}
	}
}
```

## 5. 使用示例

### 基础用法

```go
package main

import (
	"github.com/log-system/log-sdk/pkg/logger"
)

func main() {
	// 初始化 Logger
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

	// 基本日志
	log.Info("Hello World",
		logger.F("user_id", "123"),
		logger.F("product_id", "abc"))

	// 错误日志
	err = someFunction()
	if err != nil {
		log.Error("Failed to execute function",
			logger.F("error", err.Error()),
			logger.F("function", "someFunction"))
	}
}
```

### 高级配置

```go
package main

import (
	"github.com/log-system/log-sdk/pkg/logger"
	"github.com/log-system/log-sdk/pkg/hook"
	"time"
)

func main() {
	// 初始化配置
	log, err := logger.New(logger.Config{
		ServiceName:      "api-service",
		Environment:      "production",
		Cluster:          "cluster-1",
		Pod:              "api-pod-123",
		EtcdEndpoints:    []string{"http://localhost:2379"},
		BatchSize:        100,
		BatchTimeout:     100 * time.Millisecond,
		FallbackToConsole: true,
	})

	// 添加自定义 Hook
	log.AddHook(hook.LevelHook(logger.LevelWarn))
	log.AddHook(hook.RegexHook("pod", "api-*"))

	// 发送日志
	log.Warn("High latency",
		logger.F("latency", "1.5s"),
		logger.F("path", "/api/v1/orders"))
}
```
