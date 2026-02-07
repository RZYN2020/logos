# 语义化日志系统

支持动态策略配置的高性能语义化日志系统，结合动态配置中心、实时流处理和智能分析。

## 核心特性

| 特性 | 说明 |
|------|------|
| **高性能日志 SDK** | 零拷贝环形缓冲区，异步批量发送，支持百万级 QPS |
| **动态策略配置** | 基于 Etcd 的实时配置中心，支持热加载策略 |
| **语义化日志** | 自动提取业务上下文，结构化日志格式 |
| **全链路可观测性** | 集成 OpenTelemetry，自动关联 TraceID/SpanID |
| **实时流处理** | Flink 实时处理，模式解析，异常检测 |
| **SQL 查询** | 使用标准 SQL 查询日志，自动转换为 ES DSL |
| **自动报告** | 定时分析日志模式，自动生成分析报告 |

## 系统架构

```
应用层 (内嵌 SDK)
    ├─ pkg/logger   - Logger API
    ├─ pkg/guard    - 中间件拦截器
    ├─ pkg/strategy - 动态策略引擎
    ├─ pkg/async    - 异步 I/O
    └─ pkg/encoder  - 编码器
          ↓
配置层
    ├─ Etcd           - 配置中心
    ├─ Config Server    - 策略管理 API
    └─ Frontend       - 管理面板
          ↓
缓冲层
    └─ Kafka          - 消息队列
          ↓
处理层
    └─ Log Processor
        ├─ pkg/parser   - 模式解析
        ├─ pkg/semantic - 语义增强
        ├─ pkg/enricher - 上下文富化
        └─ pkg/sink     - 输出目标
          ↓
存储层
    ├─ Elasticsearch   - 日志存储
    ├─ PostgreSQL     - 元数据存储
    └─ Redis         - 缓存
          ↓
分析层
    └─ Log Analyzer
        ├─ SQL 查询引擎
        └─ 自动报告生成
```

**架构改进**：
- **SDK 轻量化**：语义处理移到服务端，SDK 只负责收集和发送
- **模块化设计**：SDK 五大核心模块清晰分离，便于维护
- **轻量级 Processor**：替代 Flink，降低运维复杂度
- **背压机制**：缓冲区满时自动丢弃，避免阻塞应用
- **降级策略**：Kafka 不可用时自动降级到控制台输出

## 项目结构

```
├── log-sdk/              # 日志 SDK (Go) - 轻量级客户端
│   └── log-sdk/
│       ├── pkg/
│       │   ├── logger/   # Logger API
│       │   ├── guard/    # Guard 拦截器（中间件）
│       │   ├── strategy/ # 策略引擎（采样、过滤、Etcd Watch）
│       │   ├── async/    # 异步 I/O（Kafka Producer + 背压）
│       │   └── encoder/  # 编码器（JSON）
│       └── cmd/          # 命令行工具
│           ├── logger/    # Logger 命令
│           └── config/    # Config 命令
├── log-processor/        # 日志处理器 (Go) - 服务端语义处理
│   ├── pkg/
│   │   ├── parser/       # 模式解析（JSON/Regex/Multi）
│   │   ├── semantic/     # 语义化构建器（HTTP/User/Error/Domain）
│   │   ├── enricher/     # 上下文增强
│   │   └── sink/         # 输出目标（ES/Console/Webhook）
│   └── cmd/job/          # 处理器入口
├── log-analyzer/        # 分析服务 (Go)
│   ├── api/            # SQL 查询 API
│   ├── storage/        # 数据访问层
│   └── cmd/server/     # 服务启动
├── config-server/       # 配置服务 (Go) - 策略管理 API
│   ├── pkg/api/       # API 处理器
│   └── cmd/main.go   # 服务入口
├── frontend/           # 前端 (React + Vite) - 管理控制台
│   ├── src/
│   │   ├── api/       # API 客户端
│   │   ├── components/ # React 组件
│   │   ├── App.tsx   # 主应用
│   │   └── main.tsx  # 入口
│   ├── index.html
│   └── package.json
├── examples/            # 示例应用
│   ├── http/           # HTTP 服务示例
│   └── generic/        # 通用示例
├── deploy/              # 部署配置
│   ├── docker-compose/  # Docker Compose
│   └── k8s/            # Kubernetes
├── docs/               # 文档
│   ├── README.md       # 完整文档
│   ├── thesis.md      # 论文大纲
│   ├── slides.md      # 答辩幻灯片
│   ├── roadmap.md     # 项目路线图
│   └── postman.json    # Postman 集合文档
```

## 快速开始

### 前置要求

- Go 1.18+
- Node.js 18+
- Docker & Docker Compose
- Kind/K8s (可选)

### 使用 Docker Compose 快速部署

```bash
# 启动所有基础设施服务
make deps

# 查看服务状态
cd deploy/docker-compose
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 服务地址

| 服务 | 地址 |
|------|------|
| Etcd | http://localhost:2379 |
| Kafka | localhost:9092 |
| Elasticsearch | http://localhost:9200 |
| Kibana | http://localhost:5601 |
| Log Processor | http://localhost:9091 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |
| Jaeger | http://localhost:16686 |
| **Config API** | http://localhost:8080/api/v1 |
| **Frontend** | http://localhost:5173 |

## 配置服务

### API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/strategies` | GET | 获取所有策略 |
| `/api/v1/strategies` | POST | 创建策略 |
| `/api/v1/strategies/{id}` | GET | 获取策略详情 |
| `/api/v1/strategies/{id}` | PUT | 更新策略 |
| `/api/v1/strategies/{id}` | DELETE | 删除策略 |
| `/api/v1/strategies/{id}/history` | GET | 获取策略历史 |
| `/api/v1/info` | GET | 获取系统信息 |
| `/api/v1/health` | GET | 健康检查 |

### 策略规则 DSL

```yaml
rules:
  - name: "production-error-filter"
    condition:
      level: ERROR
      environment: production
    action:
      enabled: true
      priority: high
      sampling: 1.0
```

### 运行配置服务

```bash
cd config-server
go mod init github.com/log-system/config-server
go get github.com/gin-gonic/gin
go run cmd/main.go
```

## 前端

### 功能

- **策略配置管理**
  - 策略列表展示
  - 创建/编辑策略（支持 JSON 规则编辑）
  - 策略版本历史
  - 启用/禁用切换
  - 删除策略（带确认）

- **日志分析**
  - SQL 查询编辑器
  - 快捷查询模板
  - 查询结果表格展示
  - 导出结果功能

- **系统信息**
  - 系统版本和运行状态
  - Etcd 连接状态检查
  - 自动健康检查（5秒间隔）

### 运行前端

```bash
cd frontend
npm install
npm run dev
```

## 核心组件

### 1. pkg/logger (Logger API)

高性能日志接口，支持：
- 结构化日志字段
- 多种日志级别
- 上下文传递
- 性能优化（零拷贝）

### 2. pkg/guard (Guard 拦截器)

HTTP 中间件自动记录：
- 请求方法、路径、状态码
- 响应时间
- 客户端 IP、User ID
- TraceID/SpanID 注入

### 3. pkg/strategy (策略引擎)

动态策略配置，支持：
- 日志级别过滤
- 路由策略
- 采样策略
- 脱敏策略
- Etcd Watch 热加载

### 4. pkg/async (异步 I/O)

高性能异步发送：
- Kafka Producer 封装
- Worker Pool
- 环形缓冲区
- 批量发送
- 背压处理

### 5. pkg/encoder (编码器)

日志格式编码：
- JSON 编码器
- 可扩展编码器接口

### 6. Log Processor (pkg/parser/semantic/enricher/sink)

轻量化服务端处理：
- pkg/parser: 模式解析（JSON/Regex）
- pkg/semantic: 语义增强（HTTP/User/Error/Domain）
- pkg/enricher: 上下文富化
- pkg/sink: 输出目标（ES/Console/Webhook）

### 7. SQL 查询引擎

标准 SQL 查询日志：
```sql
-- 查询错误日志
SELECT * FROM logs
WHERE level = 'ERROR'
  AND timestamp > NOW() - INTERVAL 1 HOUR;

-- 查询特定用户的请求
SELECT user_id, COUNT(*) as count
FROM logs
WHERE event_type = 'request'
GROUP BY user_id
ORDER BY count DESC
LIMIT 10;
```

## 开发指南

### 初始化依赖

```bash
make install
```

### 构建项目

```bash
make build
```

### 运行测试

```bash
make test
```

### 运行示例

```bash
# HTTP 服务示例
cd examples/http
go run main.go

# 访问测试
curl http://localhost:8080/ping
```

## 部署

### Kubernetes 部署

```bash
# 创建命名空间
kubectl create namespace logging

# 部署基础设施
make deploy

# 部署应用服务
make deploy-app
```

### 生产环境检查清单

- [ ] 配置 Etcd 集群（3+ 节点）
- [ ] 配置 Kafka 集群（3+ 节点）
- [ ] 配置 Elasticsearch 集群（3+ 节点）
- [ ] 配置 Flink 高可用
- [ ] 启用监控告警
- [ ] 配置日志持久化
- [ ] 配置备份恢复
- [ ] 安全加固（TLS, 认证）
- [ ] 资源限制配置
- [ ] 优雅关闭配置

## 文档

### 完整文档

查看 [docs/README.md](docs/README.md)，包含：
- 策略配置管理 API 文档
- 日志分析页面说明
- SDK 使用指南
- 部署运维手册

### 论文文档

- [论文大纲](thesis.md)
- [项目路线图](roadmap.md)

### 答辩幻灯片

查看 [docs/slides.md](docs/slides.md)，可用 Typora 或 Slidev 打开并导出为 PDF。

## Postman 导入

1. 打开 Postman
2. Import `docs/postman.json`
3. 选择环境变量（development/staging/production）
4. 开始测试 API

## 许可证

[MIT License](LICENSE)

## 联系方式

- Issues: [GitHub Issues](https://github.com/log-system/logos/issues)
- Discussions: [GitHub Discussions](https://github.com/log-system/logos/discussions)
