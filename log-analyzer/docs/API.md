# Log Analyzer API 文档

## 概述

Log Analyzer 提供日志查询、规则配置和分析功能的 RESTful API。

**Base URL**: `http://localhost:8080/api/v1`

## 认证

大部分 API 需要 JWT 认证。在请求头中添加：

```
Authorization: Bearer <token>
```

## 认证 API

### 登录

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

响应：

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "admin-001",
    "username": "admin",
    "email": "admin@logos.com",
    "roles": ["admin", "user"]
  }
}
```

### 注册

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "newuser",
  "password": "password123",
  "email": "user@example.com"
}
```

### 获取当前用户

```http
GET /api/v1/user
Authorization: Bearer <token>
```

### 修改密码

```http
PUT /api/v1/user/password
Authorization: Bearer <token>
Content-Type: application/json

{
  "old_password": "oldpass",
  "new_password": "newpass"
}
```

## 规则管理 API

### 获取规则列表

```http
GET /api/v1/rules
Authorization: Bearer <token>
```

响应：

```json
[
  {
    "id": "rule-001",
    "name": "error-log-filter",
    "description": "过滤错误日志",
    "enabled": true,
    "priority": 1,
    "version": 1,
    "conditions": [
      {
        "id": "cond-001",
        "rule_id": "rule-001",
        "field": "level",
        "operator": "=",
        "value": "ERROR"
      }
    ],
    "actions": [
      {
        "id": "act-001",
        "rule_id": "rule-001",
        "type": "filter",
        "config": {
          "sampling": 1.0
        }
      }
    ],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

### 获取规则详情

```http
GET /api/v1/rules/:id
Authorization: Bearer <token>
```

### 创建规则

```http
POST /api/v1/rules
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "error-log-filter",
  "description": "过滤错误日志",
  "enabled": true,
  "priority": 1,
  "conditions": [
    {
      "field": "level",
      "operator": "=",
      "value": "ERROR"
    }
  ],
  "actions": [
    {
      "type": "filter",
      "config": {
        "sampling": 1.0
      }
    }
  ]
}
```

响应：

```json
{
  "id": "rule-001",
  "version": 1
}
```

### 更新规则

```http
PUT /api/v1/rules/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "updated-rule-name",
  "description": "updated description",
  "enabled": false,
  "priority": 2,
  "conditions": [...],
  "actions": [...]
}
```

响应：

```json
{
  "id": "rule-001",
  "version": 2
}
```

### 删除规则

```http
DELETE /api/v1/rules/:id
Authorization: Bearer <token>
```

### 获取规则历史

```http
GET /api/v1/rules/:id/history
Authorization: Bearer <token>
```

响应：

```json
[
  {
    "id": "version-001",
    "rule_id": "rule-001",
    "version": 1,
    "content": {...},
    "author": "admin",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

### 回滚规则版本

```http
POST /api/v1/rules/:id/rollback/:version
Authorization: Bearer <token>
```

### 验证规则

```http
POST /api/v1/rules/:id/validate
Authorization: Bearer <token>
```

响应：

```json
{
  "valid": true
}
```

或

```json
{
  "valid": false,
  "errors": ["condition field cannot be empty"]
}
```

### 测试规则

```http
POST /api/v1/rules/:id/test
Authorization: Bearer <token>
```

响应：

```json
{
  "matched": true,
  "test_data": {
    "level": "ERROR",
    "service": "test-service"
  }
}
```

### 导出规则

```http
GET /api/v1/rules/export
Authorization: Bearer <token>
```

### 导入规则

```http
POST /api/v1/rules/import
Authorization: Bearer <token>
Content-Type: application/json

{
  "rules": [...]
}
```

响应：

```json
{
  "imported": 5
}
```

## 日志分析 API

### 挖掘日志模式

```http
POST /api/v1/analysis/mine
Authorization: Bearer <token>
Content-Type: application/json

{
  "logs": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "level": "ERROR",
      "service": "api",
      "message": "Connection failed"
    }
  ],
  "time_range": {
    "from": "2024-01-01T00:00:00Z",
    "to": "2024-01-01T23:59:59Z"
  }
}
```

响应：

```json
{
  "patterns": [
    {
      "id": "pattern-001",
      "pattern": "Connection failed",
      "frequency": 100,
      "severity": "high",
      "examples": ["Connection failed"],
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### 检测异常日志

```http
POST /api/v1/analysis/anomalies
Authorization: Bearer <token>
Content-Type: application/json

{
  "current_logs": [...],
  "baseline_logs": [...]
}
```

响应：

```json
{
  "anomalies": [...],
  "total": 5
}
```

### 聚类日志

```http
POST /api/v1/analysis/cluster
Authorization: Bearer <token>
Content-Type: application/json

{
  "logs": [...],
  "threshold": 0.8
}
```

响应：

```json
{
  "clusters": [
    {
      "id": "cluster-001",
      "center": "Connection failed",
      "size": 50,
      "similarity": 0.95,
      "members": [...],
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 3
}
```

### 推荐规则

```http
POST /api/v1/analysis/recommend
Authorization: Bearer <token>
Content-Type: application/json

{
  "logs": [...],
  "min_frequency": 10
}
```

响应：

```json
{
  "recommendations": [
    {
      "pattern_id": "pattern-001",
      "name": "Auto-generated rule for: Connection failed",
      "description": "Automatically generated from pattern with frequency 100",
      "priority": 1,
      "conditions": [
        {
          "field": "message",
          "operator": "contains",
          "value": "Connection failed"
        }
      ],
      "action": {
        "type": "filter",
        "config": {
          "sampling": 1.0,
          "priority": 1
        }
      }
    }
  ],
  "total": 1
}
```

### 获取模式类型

```http
GET /api/v1/analysis/pattern-types
Authorization: Bearer <token>
```

响应：

```json
{
  "pattern_types": [
    {
      "name": "error_spike",
      "description": "错误日志突然增加",
      "severity": "high"
    }
  ]
}
```

## 系统 API

### 健康检查

```http
GET /health
```

响应：

```json
{
  "status": "healthy"
}
```

### 就绪检查

```http
GET /ready
```

响应：

```json
{
  "status": "ready"
}
```

### 获取系统信息

```http
GET /api/v1/info
```

响应：

```json
{
  "system": "Log Analyzer",
  "version": "1.0.0",
  "etcd_version": "3.5.9",
  "uptime": "24h0m0s"
}
```

## 错误响应

所有错误返回统一格式：

```json
{
  "error": "error message"
}
```

常见 HTTP 状态码：

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 500 | 服务器内部错误 |

## 默认用户

系统启动时会自动创建默认管理员用户：

- 用户名：`admin`
- 密码：`admin123`

**请在生产环境中修改默认密码！**
