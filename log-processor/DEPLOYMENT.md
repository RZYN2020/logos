# Log Processor 部署和操作文档

## 概述

Log Processor 是 Logos 平台的核心 ETL 组件，负责从 Kafka 消费日志数据，经过解析、过滤、文本分析、转换和语义增强后写入 Elasticsearch。

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Log Processor Pipeline                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Kafka ──▶ Parser ──▶ Filter ──▶ Analyzer ──▶ Transformer  │
│                                                              │
│                      ▼                                       │
│              Semantic Enricher                               │
│                      ▼                                       │
│              Elasticsearch                                   │
│                                                              │
│  ETCD (配置管理) ─────────────────────────────────┘          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 处理流程

1. **解析 (Parser)**: 自动检测日志格式并解析为结构化数据
   - JSON 格式
   - KeyValue 格式
   - Syslog 格式
   - Apache/Nginx 日志
   - 非结构化文本

2. **过滤 (Filter)**: 根据 ETCD 配置进行服务端过滤
   - 正则表达式匹配
   - 复合条件过滤
   - 允许/丢弃/标记动作

3. **文本分析 (Analyzer)**: 提取关键信息
   - 实体识别（IP、URL、邮箱等）
   - 关键词提取
   - 情感分析
   - 语言检测

4. **转换 (Transformer)**: 应用转换规则
   - 正则提取
   - 模板转换
   - 字段映射

5. **语义增强 (Semantic)**: 添加业务语义
   - HTTP 信息提取
   - 用户信息提取
   - 错误标记
   - 业务域推断

## 部署

### Docker 部署

```bash
# 构建镜像
docker build -t logos-registry/log-processor:latest ./log-processor

# 运行容器
docker run -d \
  --name log-processor \
  -e KAFKA_BROKERS=kafka:9092 \
  -e KAFKA_TOPIC=logs \
  -e ES_ADDRESSES=http://elasticsearch:9200 \
  -e ETCD_ENDPOINTS=etcd:2379 \
  -e ENABLE_FILTERING=true \
  -e ENABLE_TRANSFORM=true \
  logos-registry/log-processor:latest
```

### Kubernetes 部署

```bash
# 应用配置
kubectl apply -f deploy/kubernetes/deployment.yaml

# 查看状态
kubectl get pods -n logos -l app=log-processor

# 查看日志
kubectl logs -n logos -l app=log-processor -f
```

## 配置选项

### 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| KAFKA_BROKERS | Kafka  broker 地址 | localhost:9092 |
| KAFKA_TOPIC | Kafka 主题 | logs |
| KAFKA_GROUP | Kafka 消费组 | log-processor |
| ES_ADDRESSES | Elasticsearch 地址 | http://localhost:9200 |
| ES_INDEX | Elasticsearch 索引 | logs |
| BATCH_SIZE | 批量写入大小 | 100 |
| BATCH_TIMEOUT | 批量超时时间 | 5s |
| ETCD_ENDPOINTS | ETCD 端点 | localhost:2379 |
| ENABLE_FILTERING | 启用过滤 | false |
| ENABLE_TRANSFORM | 启用转换 | false |

### ETCD 配置格式

过滤配置存储在 ETCD 的 `/log-processor/filters/` 路径下：

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
    },
    {
      "name": "mark-error",
      "field": "raw",
      "pattern": "ERROR.*Timeout",
      "action": "mark"
    }
  ]
}
```

## 监控指标

### Prometheus 指标

- `log_parse_count_total`: 解析总数
- `log_parse_success_total`: 解析成功数
- `log_parse_failure_total`: 解析失败数
- `log_parse_latency_seconds`: 解析延迟
- `log_filter_count_total`: 过滤总数
- `log_filter_dropped_total`: 被丢弃的日志数
- `log_transform_count_total`: 转换总数
- `log_write_count_total`: 写入总数
- `log_write_latency_seconds`: 写入延迟

### 访问指标

```bash
curl http://localhost:9090/metrics
```

## 操作指南

### 添加过滤规则

```bash
# 使用 etcdctl 添加规则
etcdctl put /log-processor/filters/filter-1 '{
  "id": "filter-1",
  "enabled": true,
  "priority": 10,
  "rules": [
    {
      "name": "drop-debug",
      "field": "level",
      "pattern": "DEBUG",
      "action": "drop"
    }
  ]
}'
```

### 查看处理统计

```bash
# 查看实时日志
kubectl logs -n logos -l app=log-processor | grep "Flushed"

# 查看指标
kubectl port-forward -n logos svc/log-processor 9090:9090
curl http://localhost:9090/metrics
```

### 故障排查

**问题：日志处理延迟高**

1. 检查 Kafka 消费延迟
2. 检查 Elasticsearch 写入性能
3. 检查 CPU 和内存使用率
4. 考虑增加 replicas

**问题：过滤规则不生效**

1. 确认 ETCD 配置格式正确
2. 检查规则优先级
3. 查看日志确认规则加载

```bash
kubectl logs -n logos -l app=log-processor | grep "filter"
```

**问题：内存使用过高**

1. 检查 batch size 配置
2. 检查并行处理配置
3. 考虑启用内存限制

## 性能优化

### 推荐配置

- **小集群** (< 1000 logs/s): 1-2 replicas, 512Mi memory
- **中集群** (1000-10000 logs/s): 3 replicas, 1Gi memory
- **大集群** (> 10000 logs/s): 5+ replicas, 2Gi memory

### 调优参数

```yaml
resources:
  requests:
    memory: "1Gi"
    cpu: "1"
  limits:
    memory: "4Gi"
    cpu: "4"
```

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-02-28 | 初始版本，支持增强 ETL 功能 |
