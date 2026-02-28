# 解析器和转换规则使用文档

## 概述

Log Processor 支持多种日志格式的解析和转换。本文档介绍如何使用各种解析器和转换规则。

## 解析器

### 1. JSON 解析器

自动解析 JSON 格式的日志。

**输入示例:**
```json
{
  "timestamp": "2026-02-28T12:00:00Z",
  "level": "INFO",
  "message": "User login successful",
  "service": "auth-service",
  "user_id": "user123"
}
```

**输出字段:**
- `timestamp`: 解析后的时间戳
- `level`: 日志级别
- `message`: 日志消息
- `service`: 服务名
- `user_id`: 自定义字段

### 2. KeyValue 解析器

解析 key=value 格式的日志。

**输入示例:**
```
timestamp=2026-02-28T12:00:00Z level=INFO message="test message" service=api user_id=123
```

**支持的格式:**
- `key=value`
- `key: value`
- `key="value with spaces"`

### 3. Syslog 解析器

解析 RFC 3164 格式的 Syslog 日志。

**输入示例:**
```
<34>Feb 28 12:00:00 myhost myservice[1234]: Test message
```

**提取字段:**
- 优先级
- 时间戳
- 主机名
- 进程名
- PID
- 消息

### 4. Apache/Nginx 解析器

解析 Web 服务器访问日志。

**输入示例:**
```
127.0.0.1 - - [28/Feb/2026:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234 "-" "Mozilla/5.0"
```

**提取字段:**
- `client_ip`: 客户端 IP
- `http_method`: HTTP 方法
- `http_path`: URL 路径
- `http_status`: 状态码
- `response_size`: 响应大小
- `user_agent`: 用户代理

### 5. 非结构化日志解析器

解析纯文本日志，提取关键信息。

**输入示例:**
```
2026-02-28 12:00:00 ERROR Database connection failed: timeout after 30s for user admin from 192.168.1.1
```

**提取信息:**
- 时间戳
- 日志级别
- IP 地址
- URL
- 错误详情
- 关键词

## 转换规则

### 1. 正则提取器

从字段中使用正则表达式提取值。

**配置示例:**
```json
{
  "name": "extract-http-method",
  "source_field": "message",
  "target_field": "http_method",
  "extractor": "regex",
  "config": {
    "pattern": "(GET|POST|PUT|DELETE)\\s+"
  },
  "enabled": true
}
```

### 2. 模板提取器

使用模板转换字段值。

**配置示例:**
```json
{
  "name": "format-log",
  "source_field": "service",
  "target_field": "log_source",
  "extractor": "template",
  "config": {
    "template": "Service: {{source}}"
  },
  "enabled": true
}
```

### 3. 直接提取

直接复制字段值。

**配置示例:**
```json
{
  "name": "copy-message",
  "source_field": "message",
  "target_field": "original_message",
  "extractor": "direct",
  "enabled": true
}
```

### 4. 大小写转换

**配置示例:**
```json
{
  "name": "to-lowercase",
  "source_field": "level",
  "target_field": "level_lower",
  "extractor": "lowercase",
  "enabled": true
}
```

### 5. 分割提取器

将字符串分割为数组。

**配置示例:**
```json
{
  "name": "split-tags",
  "source_field": "tags",
  "target_field": "tag_array",
  "extractor": "split",
  "config": {
    "delimiter": ","
  },
  "enabled": true
}
```

## 常见日志格式解析示例

### Go 日志
```
2026/02/28 12:00:00 INFO main.go:42: Application started
```

### Java 日志
```
2026-02-28 12:00:00.123 INFO  [main] com.example.App - Application started
```

### Python 日志
```
2026-02-28 12:00:00,123 - INFO - root - Application started
```

### Node.js 日志 (Winston)
```json
{"level":"info","message":"Application started","timestamp":"2026-02-28T12:00:00.000Z"}
```

## 性能优化最佳实践

### 1. 解析器选择

- 优先使用特定格式解析器（JSON、KeyValue 等）
- 非结构化解析器作为最后手段
- 使用格式检测缓存

### 2. 过滤规则优化

- 将高优先级规则放在前面
- 使用具体的正则表达式
- 避免过多的回溯

### 3. 转换规则优化

- 预编译正则表达式
- 减少规则数量
- 使用更高效的提取器

### 4. 资源配置

```yaml
# 推荐配置
resources:
  requests:
    memory: "1Gi"
    cpu: "1"
  limits:
    memory: "4Gi"
    cpu: "4"
```

## 配置管理操作指南

### 添加过滤规则

```bash
# 使用 etcdctl
etcdctl put /log-processor/filters/my-filter '{
  "id": "my-filter",
  "enabled": true,
  "priority": 10,
  "rules": [{
    "name": "drop-debug",
    "field": "level",
    "pattern": "DEBUG",
    "action": "drop"
  }]
}'
```

### 查看当前配置

```bash
# 列出所有过滤器
etcdctl get --prefix /log-processor/filters/

# 查看特定过滤器
etcdctl get /log-processor/filters/my-filter
```

### 删除过滤规则

```bash
etcdctl del /log-processor/filters/my-filter
```

### 验证规则

```bash
# 查看日志确认规则加载
kubectl logs -l app=log-processor | grep "filter"
```

## 故障排查

### 解析失败

1. 检查日志格式是否符合预期
2. 启用调试日志查看详细错误
3. 测试解析器单独运行

### 过滤不生效

1. 确认规则已加载（查看日志）
2. 检查规则优先级
3. 验证正则表达式语法

### 转换错误

1. 检查源字段是否存在
2. 验证提取器配置
3. 测试转换规则
