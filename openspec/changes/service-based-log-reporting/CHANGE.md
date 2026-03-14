# Change: Service-Based Log Reporting API

## Overview

为 Log-Analyzer 服务添加基于服务的日志存储和报告功能，支持按服务查询日志、TOP 行号统计和 TOP 模式识别。

## Motivation

当前前端已实现服务选择功能和日志报告页面，但后端 API 不支持：
1. 按服务和组件过滤规则
2. 按服务存储和查询日志
3. TOP 行号统计接口
4. TOP 模式识别接口

这导致前端只能使用模拟数据，无法与实际后端集成。

## Goals

1. 在规则模型中添加 `service` 和 `component` 字段
2. 更新规则 API 支持按服务和组件过滤
3. 添加日志存储模型，支持按服务存储日志
4. 实现 TOP 行号统计接口
5. 实现 TOP 模式识别接口
6. 添加日志报告聚合接口

## Non-Goals

1. 不改变现有 ETCD 同步机制（保持向后兼容）
2. 不修改日志 SDK 的日志生成逻辑（后续单独实现）

## Proposed Solution

### 1. 数据模型变更

#### models/rule.go - 添加服务字段
```go
type Rule struct {
    ID          string    `gorm:"primaryKey"`
    Name        string    `gorm:"not null"`
    Description string
    Enabled     bool      `gorm:"default:true"`
    Priority    int       `gorm:"default:0"`
    Service     string    `gorm:"index"`  // 新增
    Component   string    `gorm:"index"`  // 新增：sdk 或 processor
    Conditions  []Condition
    Actions     []Action
    Version     int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### models/log_entry.go - 新增日志存储模型
```go
type LogEntry struct {
    ID          uint      `gorm:"primaryKey"`
    Service     string    `gorm:"index:not null"`
    Component   string    `gorm:"index"`
    Timestamp   time.Time `gorm:"index"`
    Level       string    `gorm:"index"`
    Message     string
    Path        string    `gorm:"index"`  // 日志所在文件路径
    Function    string    `gorm:"index"`  // 日志所在函数
    LineNumber  int       `gorm:"index"`  // 日志行号
    TraceID     string    `gorm:"index"`
    UserID      string
    Fields      gorm.datatypes.JSON  // 额外字段
    CreatedAt   time.Time
}
```

#### models/log_report.go - 新增报告模型
```go
type LogReport struct {
    Service     string    `gorm:"primaryKey"`
    Component   string    `gorm:"primaryKey"`
    From        time.Time `gorm:"index"`
    To          time.Time
    TotalLogs   int
    TopLines    gorm.datatypes.JSON  // 存储 TOP 行号统计
    TopPatterns gorm.datatypes.JSON  // 存储 TOP 模式统计
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 2. API 变更

#### GET /api/v1/rules - 支持服务过滤
```go
// ListRules 获取规则列表
func (h *RuleHandler) ListRules(c *gin.Context) {
    service := c.Query("service")
    component := c.Query("component")

    query := h.db.Preload("Conditions").Preload("Actions")
    if service != "" {
        query = query.Where("service = ?", service)
    }
    if component != "" {
        query = query.Where("component = ?", component)
    }

    var rules []models.Rule
    if err := query.Find(&rules).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rules"})
        return
    }

    c.JSON(http.StatusOK, rules)
}
```

#### POST /api/v1/rules - 支持服务字段
```go
type RuleRequest struct {
    Name        string              `json:"name" binding:"required"`
    Description string              `json:"description"`
    Enabled     bool                `json:"enabled"`
    Priority    int                 `json:"priority"`
    Service     string              `json:"service"`  // 新增
    Component   string              `json:"component"` // 新增
    Conditions  []ConditionRequest  `json:"conditions"`
    Actions     []ActionRequest     `json:"actions"`
}
```

#### GET /api/v1/report/:service/top-lines - 新增 TOP 行号接口
```go
// GetTopLines 获取 TOP 日志行号统计
func (h *ReportHandler) GetTopLines(c *gin.Context) {
    service := c.Param("service")
    component := c.Query("component")
    from := c.Query("from")
    to := c.Query("to")
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

    // 构建查询
    query := h.db.Model(&models.LogEntry{}).
        Select("path, function, line_number, COUNT(*) as count").
        Where("service = ?", service)

    if component != "" {
        query = query.Where("component = ?", component)
    }
    if from != "" {
        query = query.Where("timestamp >= ?", from)
    }
    if to != "" {
        query = query.Where("timestamp <= ?", to)
    }

    var results []struct {
        Path       string
        Function   string
        LineNumber int
        Count      int
    }

    query.Group("path, function, line_number").
        Order("count DESC").
        Limit(limit).
        Scan(&results)

    // 计算总数和百分比
    var total int64
    h.db.Model(&models.LogEntry{}).Where("service = ?", service).Count(&total)

    topLines := make([]gin.H, len(results))
    for i, r := range results {
        topLines[i] = gin.H{
            "line_number": r.LineNumber,
            "file":        r.Path,
            "function":    r.Function,
            "count":       r.Count,
            "percentage":  float64(r.Count) / float64(total) * 100,
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "service":     service,
        "total_logs":  total,
        "top_lines":   topLines,
    })
}
```

#### GET /api/v1/report/:service/top-patterns - 新增 TOP 模式接口
```go
// GetTopPatterns 获取 TOP 日志模式统计
func (h *ReportHandler) GetTopPatterns(c *gin.Context) {
    service := c.Param("service")
    component := c.Query("component")
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

    // 获取日志
    var logs []models.LogEntry
    query := h.db.Where("service = ?", service)
    if component != "" {
        query = query.Where("component = ?", component)
    }
    query.Find(&logs)

    // 使用 PatternMiner 进行模式识别
    miner := analysis.NewPatternMiner()
    entries := make([]analysis.LogEntry, len(logs))
    for i, log := range logs {
        entries[i] = analysis.LogEntry{
            Message:   log.Message,
            Timestamp: log.Timestamp,
            Level:     log.Level,
        }
    }

    patterns := miner.AnalyzePatterns(entries)

    // 排序并取 TOP N
    sort.Slice(patterns, func(i, j int) bool {
        return patterns[i].Frequency > patterns[j].Frequency
    })

    if len(patterns) > limit {
        patterns = patterns[:limit]
    }

    // 计算总数
    var total int64
    h.db.Model(&models.LogEntry{}).Where("service = ?", service).Count(&total)

    topPatterns := make([]gin.H, len(patterns))
    for i, p := range patterns {
        // 获取示例日志
        var sampleLogs []models.LogEntry
        h.db.Where("service = ? AND message LIKE ?", service, "%"+p.Template+"%").
            Limit(3).Find(&sampleLogs)

        samples := make([]string, len(sampleLogs))
        for j, s := range sampleLogs {
            samples[j] = s.Message
        }

        topPatterns[i] = gin.H{
            "pattern":     p.Template,
            "count":       p.Frequency,
            "percentage":  float64(p.Frequency) / float64(total) * 100,
            "sample_logs": samples,
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "service":      service,
        "total_logs":   total,
        "top_patterns": topPatterns,
    })
}
```

#### GET /api/v1/report/:service - 新增完整报告接口
```go
// GetReport 获取完整日志报告
func (h *ReportHandler) GetReport(c *gin.Context) {
    service := c.Param("service")
    component := c.Query("component")

    // 并行获取 TOP 行号和 TOP 模式
    var topLines, topPatterns gin.H
    var wg sync.WaitGroup

    wg.Add(2)
    go func() {
        defer wg.Done()
        topLines = h.getTopLines(service, component)
    }()
    go func() {
        defer wg.Done()
        topPatterns = h.getTopPatterns(service, component)
    }()

    wg.Wait()

    c.JSON(http.StatusOK, gin.H{
        "service":      service,
        "total_logs":   topLines["total_logs"],
        "top_lines":    topLines["top_lines"],
        "top_patterns": topPatterns["top_patterns"],
    })
}
```

### 3. 路由配置

#### cmd/server/main.go
```go
func (s *Server) setupRoutes() {
    // ... 现有代码 ...

    // 规则 API
    rules := v1.Group("/rules")
    {
        rules.GET("", s.ruleHandler.ListRules)
        rules.POST("", s.ruleHandler.CreateRule)
        rules.GET("/:id", s.ruleHandler.GetRule)
        rules.PUT("/:id", s.ruleHandler.UpdateRule)
        rules.DELETE("/:id", s.ruleHandler.DeleteRule)
    }

    // 报告 API - 新增
    reports := v1.Group("/report")
    {
        reports.GET("/:service", s.reportHandler.GetReport)
        reports.GET("/:service/top-lines", s.reportHandler.GetTopLines)
        reports.GET("/:service/top-patterns", s.reportHandler.GetTopPatterns)
    }

    // 日志摄入 API - 新增
    logs := v1.Group("/logs")
    {
        logs.POST("", s.logHandler.IngestLog)      // 摄入单条日志
        logs.POST("/batch", s.logHandler.IngestBatch) // 批量摄入
        logs.GET("", s.logHandler.QueryLogs)       // 查询日志
    }
}
```

### 4. 文件结构

```
log-analyzer/
├── internal/
│   ├── handlers/
│   │   ├── rules.go          # 修改
│   │   ├── report.go         # 新增
│   │   └── logs.go           # 新增
│   ├── models/
│   │   ├── rule.go           # 修改
│   │   ├── log_entry.go      # 新增
│   │   └── log_report.go     # 新增
│   └── migrations/
│       └── migrate.go        # 修改
└── cmd/
    └── server/
        └── main.go           # 修改
```

## Testing Strategy

### 单元测试
1. `handlers/report_test.go` - 测试报告 API
2. `handlers/rules_test.go` - 测试服务过滤
3. `models/log_entry_test.go` - 测试日志模型

### 集成测试
1. `tests/report_api_test.go` - 测试完整报告流程
2. `tests/service_filtering_test.go` - 测试服务隔离

### 验收测试
1. `tests/acceptance_test.go` - 端到端测试

## Migration Plan

1. **Phase 1**: 添加数据模型和迁移脚本
2. **Phase 2**: 实现规则服务过滤
3. **Phase 3**: 实现日志摄入 API
4. **Phase 4**: 实现 TOP 行号接口
5. **Phase 5**: 实现 TOP 模式接口
6. **Phase 6**: 添加测试和文档

## Backwards Compatibility

- 现有规则 API 保持兼容，`service` 和 `component` 为可选字段
- ETCD 同步路径暂时保持硬编码，后续支持动态命名空间

## Open Questions

1. 是否需要支持日志的 TTL 自动清理？
2. TOP 模式的模式识别算法是否需要可配置？
3. 是否需要支持实时流式日志摄入（如 WebSocket）？

## Implementation Status

### Completed (2026-03-14)

✅ **Phase 1**: 数据模型变更
- Rule 模型添加 `service` 和 `component` 字段
- 新增 LogEntry 模型用于日志存储
- 新增 LogReport 模型用于报告存储
- 统一 Rule 模型（pkg/rule/rule.go）添加 Priority/Service/Component 字段

✅ **Phase 2**: 规则 API 服务过滤
- ListRules 支持 `?service=` 和 `?component=` 查询参数
- CreateRule/UpdateRule 支持 service/component 字段
- ETCD 同步路径使用动态服务命名空间

✅ **Phase 3**: 日志摄入 API
- POST /api/v1/logs - 单条日志摄入
- POST /api/v1/logs/batch - 批量日志摄入
- POST /api/v1/logs/query - 日志查询

✅ **Phase 4**: TOP 行号接口
- GET /api/v1/report/:service/top-lines
- 支持 component 过滤和时间范围
- 返回文件、函数、行号、次数、百分比

✅ **Phase 5**: TOP 模式接口（使用 Drain 算法）
- GET /api/v1/report/:service/top-patterns
- GET /api/v1/report/:service - 完整报告
- 基于 Drain 算法的模式识别

✅ **Phase 6**: 测试和文档
- 所有现有测试通过
- 前端构建成功

## Drain Algorithm Implementation

### 算法原理

Drain 是一种在线日志解析算法，使用树形结构来高效匹配日志模板：

1. **令牌化**：将日志消息分割为令牌
2. **变量检测**：识别数字、UUID、IP 地址、时间戳、Hex 字符串
3. **树搜索**：在 Drain 树中查找最佳匹配节点
4. **模板更新**：匹配成功则增加频率，失败则创建新模板

### 数据结构

```
DrainTree
├── root (DrainNode)
│   ├── Children (map[string]*DrainNode)
│   │   ├── "ERROR" -> Node
│   │   ├── "*" -> Node (wildcard)
│   │   └── ...
│   └── PatternIDs ([]string)
├── templates (map[string]*LogPattern)
├── maxDepth (int) = 4
└── similarity (float64) = 0.8
```

### 占位符格式

Drain 算法使用 `<*>` 作为变量占位符：

| 原始日志 | 解析模式 |
|---------|---------|
| `Error 123: connection failed` | `Error <*>: connection failed` |
| `User abc-123 logged in` | `User <*> logged in` |
| `Request from 192.168.1.1` | `Request from <*>` |

### API 响应示例

```json
GET /api/v1/report/api-gateway/top-patterns
{
  "service": "api-gateway",
  "total_logs": 125847,
  "top_patterns": [
    {
      "pattern": "Request received from <*>",
      "count": 25000,
      "percentage": 19.9,
      "sample_logs": [
        "Request received from 192.168.1.100",
        "Request received from 10.0.0.50"
      ]
    }
  ]
}
```

## Testing Results

```bash
# Log Analyzer Tests
cd log-analyzer && go test ./...
ok  github.com/log-system/log-analyzer/internal/analysis
ok  github.com/log-system/log-analyzer/internal/handlers
ok  github.com/log-system/log-analyzer/tests

# Frontend Build
cd frontend && npm run build
✓ 35 modules transformed
dist/assets/index-*.js   237.50 kB │ gzip: 70.47 kB
```

## Files Changed

| File | Change |
|------|--------|
| `pkg/rule/rule.go` | Add Priority/Service/Component fields |
| `log-analyzer/internal/models/models.go` | Add Service/Component to Rule, add LogEntry/LogReport |
| `log-analyzer/internal/handlers/rules.go` | Support service filtering, update ETCD path |
| `log-analyzer/internal/handlers/report.go` | New file - ReportHandler implementation |
| `log-analyzer/internal/analysis/mining.go` | Implement Drain algorithm |
| `log-analyzer/internal/migrations/migrate.go` | Add LogEntry/LogReport to migrations |
| `log-analyzer/cmd/server/main.go` | Add ReportHandler, new routes |
