# Log Processor ETL 增强架构文档

## 系统架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Log Processor Pipeline                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐          │
│  │  Kafka   │───▶│  Parser  │───▶│  Filter  │───▶│ Analyzer │          │
│  │  Consumer│    │  Engine  │    │  Engine  │    │  Engine  │          │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘          │
│                                                     │                    │
│                                                     ▼                    │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐          │
│  │Elastic   │◀───│ Semantic │◀───│Transform │◀───│  Metrics │          │
│  │ Search   │    │ Enricher │    │  Engine  │    │  Engine  │          │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘          │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    ETCD Configuration Store                       │   │
│  │  /log-processor/filters/    /log-processor/parsers/               │   │
│  │  /log-processor/transforms/   /log-processor/config               │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. Parser Engine（解析引擎）

**职责**: 自动检测日志格式并解析为结构化数据

**组件**:
- `FormatDetector`: 格式检测器，支持 JSON、KeyValue、Syslog、Apache、Nginx、Unstructured
- `ExtendedMultiParser`: 多格式解析器，支持自动切换
- `ParserScheduler`: 解析器调度器，支持缓存和性能统计
- `UnstructuredParser`: 非结构化日志解析器，支持实体提取

**接口**:
```go
type Parser interface {
    Parse(raw []byte) (*ParsedLog, error)
    SupportsFormat(format FormatType) bool
}
```

### 2. Filter Engine（过滤引擎）

**职责**: 根据 ETCD 配置进行服务端日志过滤

**组件**:
- `FilterEngineImpl`: 过滤引擎实现
- `RegexFilter`: 正则表达式过滤器
- `CompositeFilter`: 复合条件过滤器
- `FilterMetadata`: 过滤元数据记录

**配置结构**:
```json
{
  "id": "filter-1",
  "enabled": true,
  "priority": 10,
  "service": "api-gateway",
  "rules": [
    {
      "name": "drop-health-check",
      "field": "message",
      "pattern": "GET /healthz",
      "action": "drop"
    }
  ]
}
```

### 3. Analyzer Engine（文本分析引擎）

**职责**: 分析非结构化文本，提取关键信息

**组件**:
- `TextAnalyzerImpl`: 文本分析器
- `EntityExtractors`: 实体提取器（IP、URL、邮箱、时间戳、错误模式）
- `SentimentAnalyzer`: 情感分析器
- `LanguageDetector`: 语言检测器

**分析结果**:
```go
type AnalysisResult struct {
    Entities   []Entity
    Keywords   []string
    KeyPhrases []string
    Sentiment  SentimentResult
    Language   string
    Category   string
}
```

### 4. Transform Engine（转换引擎）

**职责**: 应用转换规则，将解析结果映射到目标字段

**组件**:
- `TransformerImpl`: 转换器实现
- `ExtractorFuncs`: 提取函数（regex、template、direct、split 等）

**规则类型**:
- `regex`: 正则表达式提取
- `template`: 模板转换
- `direct`: 直接复制
- `split`: 字符串分割

### 5. Semantic Enricher（语义增强器）

**职责**: 添加业务语义和上下文信息

**组件**:
- `Builder`: 语义构建器
- `FieldExtractors`: 字段提取器（HTTP、用户、错误、文本分析）
- `ContextEnrichers`: 上下文增强器（业务域、租户、业务属性）

**增强字段**:
- HTTP 信息（method、path、status）
- 用户 ID
- 错误类型
- 业务域
- 租户 ID
- 情感分析结果

### 6. Config Manager（配置管理器）

**职责**: 从 ETCD 加载和管理配置

**组件**:
- `EtcdConfigManager`: ETCD 配置管理器
- `EtcdClient`: ETCD 客户端
- `FilterConfig`: 过滤配置
- `TransformRule`: 转换规则

**功能**:
- 初始加载
- 周期性刷新
- 变更监听
- 热加载

### 7. Metrics Engine（指标引擎）

**职责**: 收集和暴露性能指标

**组件**:
- `Metrics`: 性能指标
- `LatencyTracker`: 延迟追踪器
- `Counter`: 计数器

**指标类型**:
- 解析指标（数量、成功率、延迟）
- 过滤指标（数量、丢弃率）
- 转换指标（数量、成功率）
- 写入指标（数量、延迟）

## 数据流

```
1. Kafka Message
       │
       ▼
2. Parser Engine
   - Detect format
   - Parse to ParsedLog
       │
       ▼
3. Filter Engine
   - Apply rules
   - Mark/Drop/Allow
       │
       ▼
4. Analyzer Engine (for unstructured)
   - Extract entities
   - Analyze sentiment
   - Detect language
       │
       ▼
5. Transform Engine
   - Apply rules
   - Extract fields
       │
       ▼
6. Semantic Enricher
   - Add business context
   - Infer attributes
       │
       ▼
7. Elasticsearch
   - Batch write
   - Index management
```

## 配置管理

### ETCD Key 结构

```
/log-processor/
├── filters/
│   ├── filter-1
│   ├── filter-2
│   └── ...
├── parsers/
│   ├── parser-1
│   └── ...
├── transforms/
│   ├── transform-1
│   └── ...
└── config
```

### 配置更新流程

```
1. 配置写入 ETCD
       │
       ▼
2. ETCD Watch 触发事件
       │
       ▼
3. ConfigManager 接收事件
       │
       ▼
4. 更新内存中的配置
       │
       ▼
5. FilterEngine 重新加载
       │
       ▼
6. 新日志使用新配置
```

## 性能优化

### 1. 对象池

重用 ParsedLog 对象，减少 GC 压力。

### 2. 正则缓存

预编译正则表达式，避免重复编译。

### 3. 格式检测缓存

缓存日志格式检测结果。

### 4. 批量处理

批量写入 Elasticsearch，提高吞吐量。

### 5. 并行处理

使用 goroutine 池并行处理日志。

## 监控和可观测性

### Prometheus 指标

| 指标 | 类型 | 描述 |
|------|------|------|
| `log_parse_count_total` | Counter | 解析总数 |
| `log_parse_success_total` | Counter | 解析成功数 |
| `log_parse_latency_seconds` | Histogram | 解析延迟 |
| `log_filter_count_total` | Counter | 过滤总数 |
| `log_filter_dropped_total` | Counter | 被丢弃的日志数 |
| `log_transform_count_total` | Counter | 转换总数 |
| `log_write_count_total` | Counter | 写入总数 |
| `log_write_latency_seconds` | Histogram | 写入延迟 |

### 日志

```json
{
  "timestamp": "2026-02-28T12:00:00Z",
  "level": "INFO",
  "message": "Flushed 100 logs to Elasticsearch",
  "service": "log-processor",
  "batch_size": 100,
  "duration_ms": 50
}
```

## 扩展性

### 添加新解析器

1. 实现 `Parser` 接口
2. 在 `ExtendedMultiParser` 中注册
3. 添加格式检测器

### 添加新过滤器

1. 实现 `FilterRule` 逻辑
2. 在 `FilterEngine` 中注册
3. 配置 ETCD 规则

### 添加新提取器

1. 实现 `ExtractorFunc`
2. 在 `Transformer` 中注册
3. 配置转换规则

## 容错机制

### 1. 解析失败处理

- 尝试多个解析器
- 降级到非结构化解析
- 记录错误日志

### 2. ETCD 连接失败

- 使用最后已知配置
- 定期重试连接
- 记录警告日志

### 3. Elasticsearch 写入失败

- 重试机制
- 降级到控制台输出
- 批量回退

## 安全考虑

### 1. 数据脱敏

- 敏感字段过滤
- PII 信息处理

### 2. 访问控制

- ETCD 认证
- Kubernetes RBAC

### 3. 传输加密

- TLS for ETCD
- TLS for Elasticsearch

## 未来改进

1. **机器学习集成**: 使用 ML 模型进行日志分类和异常检测
2. **动态采样**: 根据负载动态调整采样率
3. **多租户支持**: 支持租户级别的配置隔离
4. **流式处理**: 支持实时流式分析
