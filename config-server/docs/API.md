# Config Server API 文档

## 概述

Config Server 提供集中的配置管理功能，支持解析器配置、转换规则、过滤器配置和策略管理。

## 基础信息

- **基础 URL**: `http://localhost:8080/api/v1`
- **认证**: 通过 `X-Auth-User` 头部传递用户标识（可选，默认为 `admin`）
- **响应格式**: 所有响应均为 JSON 格式

## 统一响应格式

所有 API 响应都遵循以下格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

| 字段 | 类型 | 描述 |
|------|------|------|
| code | int | 响应码，0 表示成功 |
| message | string | 响应消息 |
| data | object/array | 响应数据 |

## 健康检查

### GET /health

检查服务健康状态。

**响应示例**:
```json
{
  "status": "healthy"
}
```

## 系统信息

### GET /api/v1/info

获取系统信息和配置统计。

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "system": "Log System Config Server",
    "version": "v1.0.0",
    "uptime": "2026-02-28T12:00:00Z",
    "config_stats": {
      "strategies": 3,
      "parsers": 6,
      "transforms": 1,
      "filters": 2,
      "watchers": 0
    }
  }
}
```

---

## 解析器配置管理

### GET /api/v1/parsers

获取所有解析器配置。

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "parser-json",
      "name": "JSON Parser",
      "type": "json",
      "enabled": true,
      "priority": 100,
      "config": {
        "strict": true
      },
      "version": "v1.0.0",
      "created_at": "2026-02-26T12:00:00Z",
      "updated_at": "2026-02-27T12:00:00Z",
      "author": "admin"
    }
  ]
}
```

### GET /api/v1/parsers/:id

获取单个解析器配置。

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "parser-json",
    "name": "JSON Parser",
    "type": "json",
    "enabled": true,
    "priority": 100,
    "config": {
      "strict": true
    },
    "version": "v1.0.0"
  }
}
```

### POST /api/v1/parsers

创建新的解析器配置。

**请求体**:
```json
{
  "name": "Custom JSON Parser",
  "type": "json",
  "enabled": true,
  "priority": 90,
  "config": {
    "strict": false,
    "allow_nested": true
  }
}
```

**支持的类型**:
- `json` - JSON 格式解析
- `key_value` - 键值对格式解析
- `syslog` - Syslog 格式解析
- `apache` - Apache 日志格式解析
- `nginx` - Nginx 日志格式解析
- `unstructured` - 非结构化文本解析

**响应示例**:
```json
{
  "code": 0,
  "message": "parser config created",
  "data": {
    "id": "parser-abc123",
    "version": "v1.0.0"
  }
}
```

### PUT /api/v1/parsers/:id

更新解析器配置。自动递增版本号。

**请求体**: 与 POST 相同

**响应示例**:
```json
{
  "code": 0,
  "message": "parser config updated",
  "data": {
    "id": "parser-abc123",
    "version": "v1.1.0"
  }
}
```

### DELETE /api/v1/parsers/:id

删除解析器配置。

**响应示例**:
```json
{
  "code": 0,
  "message": "parser config deleted"
}
```

### GET /api/v1/parsers/:id/history

获取解析器配置版本历史。

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "version": "v1.0.0",
      "created_at": "2026-02-26T12:00:00Z",
      "author": "admin",
      "content_hash": "a1b2c3d4e5f6...",
      "comment": "Initial version"
    },
    {
      "version": "v1.1.0",
      "created_at": "2026-02-27T12:00:00Z",
      "author": "admin",
      "content_hash": "f6e5d4c3b2a1...",
      "comment": "Updated config"
    }
  ]
}
```

---

## 转换规则管理

### GET /api/v1/transforms

获取所有转换规则配置。

### GET /api/v1/transforms/:id

获取单个转换规则配置。

### POST /api/v1/transforms

创建转换规则。

**请求体**:
```json
{
  "name": "HTTP Log Transformation",
  "description": "HTTP 日志字段提取规则",
  "enabled": true,
  "priority": 100,
  "service": "web-service",
  "rules": [
    {
      "id": "rule-001",
      "source_field": "message",
      "target_field": "http_method",
      "extractor": "regex",
      "config": {
        "pattern": "(GET|POST|PUT|DELETE)\\s+"
      },
      "on_error": "skip"
    },
    {
      "id": "rule-002",
      "source_field": "message",
      "target_field": "http_path",
      "extractor": "regex",
      "config": {
        "pattern": "\\s+(/[^\\s]+)\\s+"
      },
      "on_error": "skip"
    }
  ]
}
```

**支持的提取器类型**:
- `regex` - 正则表达式提取
- `template` - 模板提取
- `jsonpath` - JSONPath 提取
- `direct` - 直接复制
- `lowercase` - 转小写
- `uppercase` - 转大写
- `split` - 分割字符串

### PUT /api/v1/transforms/:id

更新转换规则。

### DELETE /api/v1/transforms/:id

删除转换规则。

### GET /api/v1/transforms/:id/history

获取转换规则版本历史。

---

## 过滤器配置管理

### GET /api/v1/filters

获取所有过滤器配置。

### GET /api/v1/filters/:id

获取单个过滤器配置。

### POST /api/v1/filters

创建过滤器配置。

**请求体**:
```json
{
  "name": "Production Debug Filter",
  "description": "生产环境过滤 DEBUG 日志",
  "enabled": true,
  "priority": 50,
  "environment": "production",
  "rules": [
    {
      "id": "rule-001",
      "name": "drop-debug",
      "field": "level",
      "pattern": "^DEBUG$",
      "action": 1
    }
  ]
}
```

**动作类型**:
- `0` - Allow（允许）
- `1` - Drop（丢弃）
- `2` - Mark（标记）

### PUT /api/v1/filters/:id

更新过滤器配置。

### DELETE /api/v1/filters/:id

删除过滤器配置。

### GET /api/v1/filters/:id/history

获取过滤器版本历史。

---

## 策略管理

### GET /api/v1/strategies

获取所有策略。

### GET /api/v1/strategies/:id

获取单个策略。

### POST /api/v1/strategies

创建策略。

**请求体**:
```json
{
  "name": "production-error-filter",
  "description": "生产环境错误过滤策略",
  "rules": [
    {
      "condition": {
        "level": "ERROR",
        "environment": "production"
      },
      "action": {
        "enabled": true,
        "priority": "high",
        "sampling": 1.0
      }
    }
  ]
}
```

### PUT /api/v1/strategies/:id

更新策略。

### DELETE /api/v1/strategies/:id

删除策略。

### GET /api/v1/strategies/:id/history

获取策略版本历史。

---

## 配置验证

### POST /api/v1/validate/parser

验证解析器配置。

**请求体**: ParserConfig 对象

**响应示例 (成功)**:
```json
{
  "code": 0,
  "message": "validation passed",
  "data": {
    "valid": true
  }
}
```

**响应示例 (失败)**:
```json
{
  "code": 400,
  "message": "validation failed: invalid type: unknown",
  "data": {
    "error": "invalid type: unknown"
  }
}
```

### POST /api/v1/validate/transform

验证转换规则配置。

### POST /api/v1/validate/filter

验证过滤器配置。

---

## 配置变更监听

### GET /api/v1/watch

监听配置变更（长轮询）。

**查询参数**:
- `type` - 配置类型（可选）：`parser`, `transform`, `filter`, `strategy`
- `id` - 配置 ID（可选）

**请求示例**:
```
GET /api/v1/watch?type=parser&id=parser-json
```

**响应示例**:
```json
{
  "code": 0,
  "message": "config update",
  "data": {
    "type": "updated",
    "config_type": "parser",
    "id": "parser-json",
    "data": {...},
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

**事件类型**:
- `created` - 配置创建
- `updated` - 配置更新
- `deleted` - 配置删除

**超时**: 30 秒无更新后返回超时响应

---

## 错误码

| HTTP 状态码 | 描述 |
|------------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 404 | 资源未找到 |
| 500 | 服务器内部错误 |

## 使用示例

### cURL 示例

```bash
# 获取所有解析器配置
curl http://localhost:8080/api/v1/parsers

# 创建解析器配置
curl -X POST http://localhost:8080/api/v1/parsers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Parser",
    "type": "json",
    "enabled": true,
    "priority": 50
  }'

# 验证配置
curl -X POST http://localhost:8080/api/v1/validate/parser \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Parser",
    "type": "json"
  }'

# 监听配置变更
curl "http://localhost:8080/api/v1/watch?type=parser"
```

### Go 示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type ParserConfig struct {
    Name     string `json:"name"`
    Type     string `json:"type"`
    Enabled  bool   `json:"enabled"`
    Priority int    `json:"priority"`
}

func main() {
    // 创建解析器配置
    config := ParserConfig{
        Name:     "test-parser",
        Type:     "json",
        Enabled:  true,
        Priority: 50,
    }

    body, _ := json.Marshal(config)
    resp, err := http.Post(
        "http://localhost:8080/api/v1/parsers",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    fmt.Printf("Status: %d\n", resp.StatusCode)
}
```

## 版本控制

所有配置都支持版本控制：

- 初始版本为 `v1.0.0`
- 每次更新自动递增 minor 版本号（v1.0.0 -> v1.1.0）
- 历史版本记录包含：版本号、创建时间、作者、内容哈希

## 最佳实践

1. **配置命名**: 使用有意义的名称，如 `parser-json-strict`
2. **优先级设置**: 设置合理的优先级，确保配置按预期顺序应用
3. **验证先行**: 在生产环境应用前，先使用验证 API 检查配置
4. **版本管理**: 在配置变更时添加注释说明变更内容
5. **监听变更**: 使用 watch API 实时感知配置变更
