## Context

Log Processor 作为 Logos 平台的核心 ETL 组件，负责从 Kafka 中消费原始日志数据，进行解析、清洗和语义增强，最终写入 Elasticsearch。当前 Log Processor 对结构化 JSON 日志的处理能力较强，但对非结构化文本日志的处理能力有限。

随着业务的发展，大量系统仍在产生非结构化的文本日志，这些日志难以进行有效的分析和查询。为了提升平台的整体价值，需要增强 Log Processor 的非结构化日志处理能力。

## Goals / Non-Goals

**Goals:**
- 实现强大的非结构化日志解析能力
- 支持从文本日志中提取结构化信息
- 提供可配置的转换规则
- 保持与现有系统架构的兼容性
- 确保高吞吐量和低延迟

**Non-Goals:**
- 实现完整的自然语言处理系统
- 处理图像、音频等非文本日志
- 替代现有的结构化日志处理能力

## Decisions

### 解析引擎架构

**决策**: 使用分层解析架构

**原因**:
- 支持多种解析策略
- 易于扩展新的解析方法
- 可以根据日志类型选择最佳解析策略

**架构**:
```
┌─────────────────────────────────────────┐
│        Log Processor Pipeline           │
├─────────────────────────────────────────┤
│ 1. Raw Log Input                        │
│ 2. Format Detection & Classification    │
│ 3. Parser Selection & Execution         │
│ 4. Text Analysis & Extraction          │
│ 5. Structured Transformation            │
│ 6. Semantic Enrichment                  │
│ 7. Output to Elasticsearch              │
└─────────────────────────────────────────┘
```

### 文本分析策略

**决策**: 组合使用规则引擎和轻量级 NLP 技术

**原因**:
- 规则引擎对于常见模式的解析效率高、准确率高
- 轻量级 NLP 技术可以处理更复杂的文本模式
- 平衡了性能和功能的需求

**技术选型**:
- 规则引擎: 自定义实现，支持正则表达式和模式匹配
- NLP: 使用 Go 语言的轻量级 NLP 库（如 go-ego/gse 中文分词、github.com/jdkato/prose 英文处理）

### 格式检测方法

**决策**: 使用启发式规则和机器学习分类器

**原因**:
- 启发式规则对于常见格式的检测快速有效
- 机器学习分类器可以提高对复杂格式的检测准确率
- 结合使用可以达到最佳效果

**实现**:
- 启发式规则: 基于关键字、模式匹配
- 分类器: 使用朴素贝叶斯或 SVM 分类器，离线训练

### 转换规则管理

**决策**: 使用配置驱动的转换规则

**原因**:
- 支持动态配置和更新
- 无需重启服务即可应用新规则
- 降低运维成本

**存储方式**:
- 转换规则存储在 Etcd 中
- 通过 Config Server 进行管理
- 支持版本控制和回滚

## Risks / Trade-offs

### 风险 1: 解析性能下降

**描述**: 复杂的解析和分析过程可能导致性能下降

**缓解措施**:
- 使用流式处理和异步解析
- 对解析器进行性能优化
- 支持解析策略的动态调整
- 提供性能监控和告警

### 风险 2: 解析准确率不足

**描述**: 某些复杂的非结构化日志可能无法准确解析

**缓解措施**:
- 提供解析质量评估和反馈机制
- 支持人工标记和修正
- 持续优化解析规则和算法

### 风险 3: 配置复杂度增加

**描述**: 大量的解析和转换规则可能导致配置复杂

**缓解措施**:
- 提供可视化的规则配置界面
- 支持模板化和复用
- 提供规则推荐和自动生成功能

## Architecture Design

### 模块架构

```
┌─────────────────────────────────────────────────────┐
│                 Log Processor                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│ ┌──────────────┐  ┌──────────────┐  ┌──────────────┐│
│ │  Kafka Reader│  │  ETCD Config │  │  Metrics     ││
│ │              │  │  (配置拉取)  │  │              ││
│ └──────┬───────┘  └──────┬───────┘  └──────┬───────┘│
│        │                 │                  │        │
│        └────────────┬────┴──────────────────┘        │
│                     ▼                                 │
│            ┌──────────────┐                           │
│            │  Preprocessor│                           │
│            │  (格式检测)   │                           │
│            └──────┬───────┘                           │
│                   ▼                                   │
│            ┌──────────────┐                           │
│            │  Parser      │                           │
│            │  (解析引擎)   │                           │
│            └──────┬───────┘                           │
│                   ▼                                   │
│            ┌──────────────┐                           │
│            │  Text Analyzer│                          │
│            │  (文本分析)   │                          │
│            └──────┬───────┘                           │
│                   ▼                                   │
│            ┌──────────────┐                           │
│            │  Filter      │                           │
│            │  (过滤引擎)   │                           │
│            └──────┬───────┘                           │
│                   ▼                                   │
│            ┌──────────────┐                           │
│            │  Transformer │                           │
│            │  (转换器)     │                           │
│            └──────┬───────┘                           │
│                   ▼                                   │
│            ┌──────────────┐                           │
│            │  Semantic    │                           │
│            │  Enricher    │                           │
│            └──────┬───────┘                           │
│                   ▼                                   │
│            ┌──────────────┐                           │
│            │  Elasticsearch│                          │
│            │  Writer      │                           │
│            └──────────────┘                           │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 核心组件设计

#### 1. ETCD 配置管理器 (ETCD Config Manager)

**职责**: 从 ETCD 拉取和管理过滤配置

```go
type FilterConfig struct {
    ID         string       `json:"id"`
    Enabled    bool         `json:"enabled"`
    Priority   int          `json:"priority"`
    Service    string       `json:"service,omitempty"`
    Environment string      `json:"environment,omitempty"`
    Rules      []FilterRule `json:"rules"`
}

type FilterRule struct {
    Name       string       `json:"name"`
    Field      string       `json:"field"`           // 要匹配的字段 (message, raw, etc.)
    Pattern    string       `json:"pattern"`         // 正则表达式模式
    Action     FilterAction `json:"action"`          // 匹配后的动作
}

type FilterAction int

const (
    ActionAllow FilterAction = iota
    ActionDrop
    ActionMark
)

type ConfigManager interface {
    LoadFilters() ([]*FilterConfig, error)
    WatchFilters() (<-chan FilterEvent, <-chan error)
    GetActiveFilters() []*FilterConfig
}

type FilterEvent struct {
    Type    EventType
    Config  *FilterConfig
}
```

#### 2. 过滤引擎 (Filter Engine)

**职责**: 对日志进行服务端过滤

```go
type FilterEngine interface {
    ApplyFilters(entry *ParsedLog) FilterResult
    AddFilter(config *FilterConfig) error
    RemoveFilter(id string) error
    ReloadFilters() error
}

type FilterResult struct {
    ShouldKeep  bool
    Action      FilterAction
    MatchedRule string
    Metadata    map[string]interface{}
}

type RegexFilter struct {
    config *FilterConfig
    compiledPatterns []*regexp.Regexp
}

func NewRegexFilter(config *FilterConfig) (*RegexFilter, error)
func (f *RegexFilter) Match(entry *ParsedLog) bool
```

#### 3. 格式检测器 (Format Detector)

**职责**: 检测日志格式，确定使用的解析策略

```go
type FormatDetector interface {
    Detect(log []byte) (FormatType, float64)
    RegisterFormat(format FormatType, detector FormatDetectorFunc)
}

type FormatType string

const (
    FormatJSON        FormatType = "json"
    FormatKeyValue    FormatType = "key_value"
    FormatSyslog      FormatType = "syslog"
    FormatApache      FormatType = "apache"
    FormatNginx       FormatType = "nginx"
    FormatUnstructured FormatType = "unstructured"
)
```

#### 2. 解析器接口 (Parser)

**职责**: 解析原始日志，提取结构化信息

```go
type Parser interface {
    Parse(log []byte) (*ParsedLog, error)
    SupportsFormat(format FormatType) bool
    GetName() string
}

type ParsedLog struct {
    Timestamp time.Time
    Level     string
    Message   string
    Service   string
    TraceID   string
    SpanID    string
    Fields    map[string]interface{}
    Raw       string
    Format    FormatType
}
```

#### 3. 文本分析器 (Text Analyzer)

**职责**: 分析非结构化文本，提取关键信息

```go
type TextAnalyzer interface {
    Analyze(text string) (*TextAnalysisResult, error)
}

type TextAnalysisResult struct {
    Entities     []Entity
    Keywords     []string
    Sentiment    Sentiment
    KeyPhrases   []string
    Language     string
}

type Entity struct {
    Type  string
    Value string
    Score float64
    Start int
    End   int
}
```

#### 4. 结构化转换器 (Transformer)

**职责**: 将解析和分析结果转换为标准结构化格式

```go
type Transformer interface {
    Transform(parsed *ParsedLog, analysis *TextAnalysisResult) (*semantic.EnrichedLog, error)
    ApplyRules(rules []TransformationRule)
}

type TransformationRule struct {
    SourceField string
    TargetField string
    Transformer string
    Config      map[string]interface{}
}
```

### 配置管理

**ETCD 过滤配置结构**:

```yaml
filters:
  - id: "filter-1"
    enabled: true
    priority: 10
    service: "order-service"
    environment: "production"
    rules:
      - name: "drop-health-check"
        field: "message"
        pattern: "GET /healthz"
        action: "drop"
      - name: "mark-error"
        field: "raw"
        pattern: "ERROR.*Timeout"
        action: "mark"

  - id: "filter-2"
    enabled: true
    priority: 5
    rules:
      - name: "allow-only-json"
        field: "format"
        pattern: "json"
        action: "allow"

parsers:
  - name: "json-parser"
    type: "json"
    enabled: true
  - name: "key-value-parser"
    type: "key_value"
    enabled: true
    config:
      delimiter: "="
  - name: "unstructured-parser"
    type: "unstructured"
    enabled: true
    config:
      min_confidence: 0.8

transform_rules:
  - source_field: "message"
    target_field: "http_method"
    extractor: "regex"
    config:
      pattern: "(GET|POST|PUT|DELETE|HEAD)\\s+"
  - source_field: "message"
    target_field: "http_path"
    extractor: "regex"
    config:
      pattern: "\\s+(\\/[\\w\\-\\/]+)\\s+"
```

## Performance Considerations

### 并行处理

- 使用 goroutine 池进行并行解析
- 解析器选择和执行的流水线化
- 结果合并和批量处理

### 内存优化

- 重用对象池，避免频繁内存分配
- 流式处理，减少内存占用
- 解析过程中的临时对象管理

### 缓存机制

- 格式检测结果缓存
- 解析规则编译缓存
- 常用转换规则的结果缓存

## Monitoring and Metrics

需要添加以下指标：

- 解析器性能指标：解析时间、吞吐量
- 解析质量指标：成功率、准确率、失败率
- 格式分布指标：各格式日志的比例
- 资源使用指标：内存、CPU 使用率
- 错误统计：解析错误类型分布
