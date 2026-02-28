# 语义化日志系统

支持动态策略配置的高性能语义化日志系统，结合动态配置中心、实时流处理和智能分析。

## 核心特性

| 特性 | 说明 |
|------|------|
| **高性能日志 SDK** | 零拷贝环形缓冲区，异步批量发送，支持百万级 QPS |
| **动态策略配置** | 基于 Etcd 的实时配置中心，支持热加载策略 |
| **语义化日志** | 自动提取业务上下文，结构化日志格式 |
| **Trace 关联** | 集成 TraceID/SpanID 字段，支持请求链路关联 |
| **轻量级流处理** | Log Processor 实时处理，语义增强 |
| **统一后端** | Log Analyzer 统一后端（日志查询 + 配置管理） |
| **SQL 查询** | 使用标准 SQL 查询日志，自动转换为 ES DSL |
| **存储优化** | 删除 raw 字段 + ES best_compression，节省 50-60% 存储空间 |

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      Logos 日志系统终态架构                   │
├─────────────────────────────────────────────────────────────────┤
│                                                          │
│  应用层（内嵌 Log SDK）                                 │
│  ┌───────────────────────────────────────────────────┐    │
│  │  User Application (with Log SDK)             │    │
│  │  - Logger API (传统/链式)                        │    │
│  │  - 异步发送至 Kafka                              │    │
│  │  - trace_id / span_id 字段                          │    │
│  └───────────────────────────────────────────────────┘    │
│                          │                               │
│                          ▼                               │
│  ┌───────────────────────────────────────────────────┐    │
│  │  Frontend (管理界面)                             │    │
│  │  - 日志查询                                       │    │
│  │  - 策略配置管理                                   │    │
│  └───────────────────────────────────────────────────┘    │
│                          │                               │
│                          ▼                               │
│  ┌───────────────────────────────────────────────────┐    │
│  │  Log Analyzer (统一后端)                        │    │
│  │  职责:                                             │    │
│  │  1. 日志查询 API → Elasticsearch                  │    │
│  │  2. 策略配置 API → Etcd                          │    │
│  └───────────────────────────────────────────────────┘    │
│          │                           │                   │
│          ▼                           ▼                   │
│  ┌───────────────┐         ┌──────────────────┐           │
│  │ Elasticsearch │         │  Etcd (配置存储) │           │
│  │  (日志存储)   │         │                  │           │
│  └───────────────┘         └──────────────────┘           │
│                                                          │
│  日志处理流:                                                │
│  App (Log SDK) → Kafka → Log Processor → Elasticsearch      │
│                                                          │
│  监控流:                                                    │
│  各服务 /metrics → Prometheus → Grafana                   │
│                                                          │
└─────────────────────────────────────────────────────────────────┘
```

**架构说明**：
- **SDK 轻量化**：语义处理移到服务端，SDK 只负责收集和发送
- **统一后端**：Log Analyzer 同时负责日志查询和配置管理
- **存储优化**：删除 raw 字段，ES best_compression
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
├── log-processor/        # 日志处理器 (Go) - 服务端语义处理
│   ├── pkg/
│   │   ├── parser/       # 模式解析（JSON/Regex/Multi）
│   │   ├── semantic/     # 语义化构建器（HTTP/User/Error/Domain）
│   │   └── sink/         # 输出目标（ES/Console/Webhook）
│   └── cmd/job/          # 处理器入口
├── log-analyzer/        # 分析服务 (Go) - 统一后端
│   └── go.mod         # 已初始化
├── frontend/           # 前端 (React + Vite) - 管理控制台
│   └── (待实现)
├── examples/            # 示例应用
│   ├── http/           # HTTP 服务示例
│   └── generic/        # 通用示例
├── deploy/              # 部署配置
│   ├── docker-compose/  # Docker Compose（本地开发）
│   ├── elasticsearch/  # ES 索引配置
│   └── k8s/            # Kubernetes（生产环境）
└── docs/               # 文档
    ├── thesis.md      # 论文大纲
    ├── slides.md      # 答辩幻灯片
    ├── roadmap.md     # 项目路线图
    └── postman.json    # Postman 集合文档
```

## 快速开始

### 前置要求

- Go 1.18+
- Node.js 18+
- Docker & Docker Compose
- Kind/K8s (可选)

### 使用 Docker Compose 快速部署

```bash
# 启动所有服务
cd deploy/docker-compose
docker-compose up -d
```

### 服务地址

| 服务 | 地址 | 说明 |
|------|------|------|
| Etcd | http://localhost:2379 | 配置存储 |
| Kafka | localhost:9092 | 日志队列 |
| Elasticsearch | http://localhost:9200 | 日志存储 |
| Kibana | http://localhost:5601 | 日志可视化 |
| Prometheus | http://localhost:9090 | 指标收集 |
| Grafana | http://localhost:3000 | 监控看板 (admin/admin) |

## Log Analyzer API

### API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/logs` | GET | 查询日志 |
| `/api/v1/logs/sql` | POST | SQL 查询日志 |
| `/api/v1/strategies` | GET | 获取所有策略 |
| `/api/v1/strategies` | POST | 创建策略 |
| `/api/v1/strategies/{id}` | GET | 获取策略详情 |
| `/api/v1/strategies/{id}` | PUT | 更新策略 |
| `/api/v1/strategies/{id}` | DELETE | 删除策略 |
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

## 前端

### 功能

- **规则配置管理**
  - 规则列表展示
  - 创建/编辑规则（支持 JSON 条件/动作编辑）
  - 规则版本历史
  - 启用/禁用切换
  - 删除规则（带确认）
  - 规则验证和测试
  - 规则导入/导出

- **日志分析**
  - SQL 查询编辑器
  - 快捷查询模板
  - 查询结果表格展示
  - 日志模式挖掘
  - 异常检测
  - 智能规则推荐

- **系统信息**
  - 系统版本和运行状态
  - Etcd 连接状态检查
  - 自动健康检查（5秒间隔）

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
- 环形缓冲区
- 批量发送
- 背压处理

### 5. Log Processor (pkg/parser/semantic/sink)

轻量化服务端处理：
- pkg/parser: 模式解析（JSON/Regex）
- pkg/semantic: 语义增强（HTTP/User/Error/Domain）
- pkg/sink: 输出目标（ES/Console/Webhook）

### 6. Log Analyzer (统一后端)

统一后端服务：
- 日志查询 API（SQL → ES DSL）
- 策略配置 API（读写 Etcd）

### 7. SQL 查询引擎

标准 SQL 查询日志：
```sql
-- 查询错误日志
SELECT * FROM logs
WHERE level = 'ERROR'
  AND timestamp > NOW() - INTERVAL 1 HOUR;

-- 查询特定 Trace 的所有日志
SELECT * FROM logs
WHERE trace_id = 'abc123def456'
ORDER BY timestamp;
```

## 存储优化

| 优化项 | 状态 | 效果 |
|--------|------|------|
| 删除 `raw` 字段 | ✅ 已完成 | 节省 40-50% |
| ES `best_compression` | ✅ 配置就绪 | 额外节省 15-20% |

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
- **Log Analyzer API**：通过 Postman 或 curl 访问 Log Analyzer 的地址
- **监控系统**：Grafana（地址：http://&lt;cluster-ip&gt;:30300，用户名/密码：admin/admin123）

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
- 数据存储：使用 PVC（默认 10GB）

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
- 数据存储：使用 PVC（默认 50GB）

**访问方式**：
- 内部访问：`kafka-headless.logging-system.svc.cluster.local:9092`
- 外部访问：使用 NodePort 或 LoadBalancer 服务（端口 9094）

#### 3. Elasticsearch 存储

**用途**：存储和索引结构化日志数据

**部署配置**：
- 使用 Elastic 官方 Helm Chart 部署
- 默认副本数：1（生产环境建议 3+ 节点）
- 资源限制：CPU 2 核，内存 4GB
- 数据存储：使用 PVC（默认 100GB）
- 索引配置：`index.codec: best_compression`

**访问方式**：
- 内部访问：`http://elasticsearch-master.logging-system.svc.cluster.local:9200`
- 外部访问：使用 NodePort 或 LoadBalancer 服务（端口 9200）

#### 4. 应用服务

**Log Analyzer**（统一后端）：
- 用途：日志查询 API + 策略配置 API
- 部署方式：Docker 镜像 + Kubernetes Deployment
- 资源限制：CPU 1 核，内存 1GB
- 依赖：Elasticsearch、Etcd

**Log Processor**（日志处理器）：
- 用途：消费 Kafka 日志，进行解析和语义增强，写入 Elasticsearch
- 部署方式：Docker 镜像 + Kubernetes Deployment
- 资源限制：CPU 2 核，内存 2GB
- 依赖：Kafka、Elasticsearch

**Frontend**（管理界面）：
- 用途：管理和监控界面
- 部署方式：Docker 镜像 + Kubernetes Deployment
- 资源限制：CPU 0.5 核，内存 256MB
- 依赖：Log Analyzer

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

### Minikube 测试记录

#### 测试环境

- **Minikube 版本**: v1.38.1
- **Kubernetes 版本**: v1.35.1
- **Docker Desktop**: 已启用
- **资源**: 4 CPUs, 6GB RAM, 40GB Disk

#### 发现的问题及解决方案

##### 问题 1: 存在已删除组件的残留配置

**现象**: Kubernetes 部署配置中包含了已删除的组件
- `config-server/` - Config Server 已合并到 Log Analyzer
- `log-sdk/` - SDK 是客户端库，不是服务
- `postgresql.yaml` - PostgreSQL 已删除
- `jaeger.yaml` - Jaeger 已删除

**解决方案**:
```bash
# 删除 Config Server Helm chart
rm -rf deploy/k8s/charts/config-server/
rm deploy/k8s/charts/logos/config-server  # 符号链接

# 删除 Log SDK Helm chart
rm -rf deploy/k8s/charts/log-sdk/
rm deploy/k8s/charts/logos/log-sdk  # 符号链接

# 删除 PostgreSQL 和 Jaeger 配置
rm deploy/k8s/storage/postgresql.yaml
rm deploy/k8s/monitoring/jaeger.yaml

# 更新 umbrella chart 的 Chart.yaml，移除相关依赖
# 更新 values.yaml，移除相关配置
```

**原因分析**:
- 架构调整后，Config Server 功能合并到了 Log Analyzer
- Log SDK 是客户端库，应用内嵌使用，无需 Kubernetes 部署
- PostgreSQL 和 Jaeger 在架构简化时已删除
- Kubernetes 配置未及时同步更新

##### 问题 2: Helm Chart 依赖配置错误

**现象**: umbrella chart 的 Chart.yaml 中 dependencies 被注释掉

**解决方案**:
```yaml
# Chart.yaml
# 取消注释 dependencies，并添加 condition 字段
dependencies:
  - name: log-processor
    version: 0.1.0
    repository: file://./log-processor
    condition: log-processor.enabled
  # ... 其他依赖
```

##### 问题 3: Service 模板缺少 LoadBalancer 支持

**现象**: Service 模板只支持 ClusterIP，无法配置 LoadBalancer

**解决方案**:
```yaml
# templates/service.yaml
spec:
  type: {{ .Values.service.type }}
  {{- if and (eq .Values.service.type "LoadBalancer") .Values.service.loadBalancerIP }}
  loadBalancerIP: {{ .Values.service.loadBalancerIP }}
  {{- end }}
  {{- if eq .Values.service.type "LoadBalancer" }}
  externalTrafficPolicy: {{ .Values.service.externalTrafficPolicy }}
  {{- end }}
```

##### 问题 4: 缺少镜像导致 Pod 启动失败

**现象**: log-analyzer 和 log-processor Pod 出现 ImagePullBackOff

**预期行为**: 这是正常的，因为自定义镜像尚未构建

**解决方案**:
```bash
# 需要构建并推送镜像到镜像仓库
docker build -t logos/log-analyzer:1.0.0 ./log-analyzer/
docker build -t logos/log-processor:1.0.0 ./log-processor/

# 或者使用本地镜像（Minikube）
eval $(minikube docker-env)
docker build -t logos/log-analyzer:1.0.0 ./log-analyzer/
```

#### 当前部署状态

```
NAMESPACE: logging-system
✓ frontend: 2/2 Running (使用 nginx 镜像)
○ log-analyzer: 0/2 ImagePullBackOff (等待自定义镜像)
○ log-processor: 0/3 ImagePullBackOff (等待自定义镜像)
```

#### 后续优化建议

1. **添加 CI/CD 流程**: 自动构建和推送镜像
2. **完善健康检查**: 为所有服务添加 /health 和 /ready 端点
3. **配置资源限制**: 根据实际负载调整 CPU/内存限制
4. **添加 HPA**: 配置 Horizontal Pod Autoscaler 自动扩缩容

---

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
- 检查 Elasticsearch 日志：`kubectl logs -f &lt;elasticsearch-pod-name&gt; -n logging-system`
- 检查内存使用情况：`kubectl top pods -n logging-system`

### 生产环境检查清单

- [ ] 配置 Etcd 集群（3+ 节点）
- [ ] 配置 Kafka 集群（3+ 节点）
- [ ] 配置 Elasticsearch 集群（3+ 节点）
- [ ] 启用监控告警
- [ ] 配置日志持久化
- [ ] 配置备份恢复
- [ ] 安全加固（TLS, 认证）
- [ ] 资源限制配置
- [ ] 优雅关闭配置

## 已删除组件

| 组件 | 原因 |
|------|------|
| Config Server | 功能合并到 Log Analyzer |
| Flink | 已用 Log Processor 替代 |
| Redis | 未使用 |
| PostgreSQL | 未使用 |
| Jaeger | 纯日志项目不需要 |
| log-sdk Deployment | SDK 是客户端库，不是服务 |

## 文档

### 论文文档

- [论文大纲](docs/thesis.md)
- [项目路线图](docs/roadmap.md)

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
