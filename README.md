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

#### 前置要求

1. **Kubernetes 集群**：可以使用以下方式之一：
   - **Minikube**：本地单节点 Kubernetes 集群
   - **云服务商**：AWS EKS、GCP GKE、Azure AKS 等
   - **本地集群**：Kind 或 K3s

2. **Helm**：版本 3.x+（用于部署 Helm Chart）

3. **kubectl**：与 Kubernetes 集群版本匹配的命令行工具

#### 部署步骤

##### 1. 部署 Logos Platform

```bash
cd deploy/k8s/scripts
chmod +x deploy.sh
./deploy.sh
```

##### 2. 验证部署

```bash
# 查看命名空间下的所有资源
kubectl get all -n logging-system
kubectl get all -n monitoring

# 检查 Pod 状态
kubectl get pods -n logging-system -w

# 查看服务访问地址
kubectl get services -n logging-system
kubectl get services -n monitoring
```

##### 3. 访问应用

- **前端界面**：通过浏览器访问 Frontend 服务的 NodePort 或 LoadBalancer 地址
- **配置服务器 API**：通过 Postman 或 curl 访问 Config Server 的地址（端口 8080）
- **监控系统**：Grafana（地址：http://<cluster-ip>:30300，用户名/密码：admin/admin123）
- **分布式追踪**：Jaeger UI（地址：http://<cluster-ip>:31686）

#### 卸载

```bash
cd deploy/k8s/scripts
./undeploy.sh
```

### 组件搭建说明

#### 1. Etcd 配置中心

**用途**：存储和监控策略配置的动态变更

**部署配置**：
- 使用 Bitnami Helm Chart 部署
- 默认副本数：1（生产环境建议 3+ 节点）
- 资源限制：CPU 0.5 核，内存 512MB
- 数据存储：使用 PVC（默认 8GB）

**访问方式**：
- 内部访问：`etcd.logging-system.svc.cluster.local:2379`
- 外部访问：使用 NodePort 或 LoadBalancer 服务

#### 2. Kafka 消息队列

**用途**：缓冲和传输高吞吐量的日志数据

**部署配置**：
- 使用 Bitnami Helm Chart 部署
- 包含 ZooKeeper（用于 Kafka 协调）
- 默认副本数：1（生产环境建议 3+ 节点）
- 资源限制：CPU 1 核，内存 2GB
- 数据存储：使用 PVC（默认 10GB）

**访问方式**：
- 内部访问：`kafka-headless.logging-system.svc.cluster.local:9092`
- 外部访问：使用 NodePort 或 LoadBalancer 服务（端口 9094）

#### 3. Elasticsearch 存储

**用途**：存储和索引结构化日志数据

**部署配置**：
- 使用 Elastic 官方 Helm Chart 部署
- 默认副本数：1（生产环境建议 3+ 节点）
- 资源限制：CPU 2 核，内存 4GB
- 数据存储：使用 PVC（默认 30GB）

**访问方式**：
- 内部访问：`http://elasticsearch-master.logging-system.svc.cluster.local:9200`
- 外部访问：使用 NodePort 或 LoadBalancer 服务（端口 9200）

#### 4. 应用服务

**配置服务器**（Config Server）：
- 用途：管理策略配置的 API 服务
- 部署方式：Docker 镜像 + Kubernetes Deployment
- 资源限制：CPU 0.5 核，内存 512MB
- 依赖：Etcd

**日志处理器**（Log Processor）：
- 用途：消费 Kafka 日志，进行解析和语义增强，写入 Elasticsearch
- 部署方式：Docker 镜像 + Kubernetes Deployment
- 资源限制：CPU 1 核，内存 1GB
- 依赖：Kafka、Elasticsearch

**前端应用**（Frontend）：
- 用途：管理和监控界面
- 部署方式：Docker 镜像 + Kubernetes Deployment
- 资源限制：CPU 0.2 核，内存 256MB
- 依赖：Config Server

#### 5. 监控和日志收集

**Prometheus**：
- 用途：指标收集和查询
- 部署方式：Helm Chart
- 资源限制：CPU 1 核，内存 2GB
- 存储：使用 PVC（默认 8GB）

**Grafana**：
- 用途：指标可视化
- 部署方式：Helm Chart
- 资源限制：CPU 0.5 核，内存 512MB
- 访问：NodePort 3000（默认用户名/密码：admin/admin）

**Jaeger**：
- 用途：分布式追踪
- 部署方式：Helm Chart
- 资源限制：CPU 0.5 核，内存 1GB
- 访问：NodePort 16686

### 常见问题

#### 1. 服务无法访问

**解决方案**：
- 检查 Pod 状态是否正常：`kubectl get pods -n logging-system`
- 检查服务是否已创建：`kubectl get services -n logging-system`
- 检查网络策略是否允许访问：`kubectl get networkpolicies -n logging-system`

#### 2. 存储卷无法挂载

**解决方案**：
- 检查存储类是否已配置：`kubectl get storageclasses`
- 检查 PVC 是否已绑定：`kubectl get pvc -n logging-system`
- 检查存储资源是否充足

#### 3. Kafka 连接失败

**解决方案**：
- 检查 Kafka Pod 状态：`kubectl get pods -n logging-system`
- 检查 Kafka 服务是否正常：`kubectl port-forward service/kafka 9092:9092 -n logging-system`
- 使用 Kafka 客户端工具测试连接

#### 4. Elasticsearch 健康检查失败

**解决方案**：
- 检查 Elasticsearch Pod 状态：`kubectl get pods -n logging-system`
- 检查 Elasticsearch 日志：`kubectl logs -f <elasticsearch-pod-name> -n logging-system`
- 检查内存使用情况：`kubectl top pods -n logging-system`

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
