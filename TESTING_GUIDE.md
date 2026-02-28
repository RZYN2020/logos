# Logos 日志系统 - 测试指南

本指南将教你如何完整地启动和测试 Logos 日志系统。

## 目录
- [环境要求](#环境要求)
- [快速开始](#快速开始)
- [基础设施启动](#基础设施启动)
- [服务验证](#服务验证)
- [SDK 测试](#sdk-测试)
- [全链路测试](#全链路测试)
- [常见问题](#常见问题)

---

## 环境要求

在开始之前，请确保你已经安装了以下工具：

| 工具 | 版本要求 |
|------|----------|
| Go | 1.18+ |
| Docker | 20.10+ |
| Docker Compose | 2.0+ |

### 检查环境

```bash
# 检查 Go 版本
go version

# 检查 Docker 版本
docker --version

# 检查 Docker Compose 版本
docker-compose --version
```

---

## 快速开始

### 1. 克隆项目（如果需要）
```bash
cd /Users/eka/Code/logos
```

### 2. 启动基础设施服务

```bash
cd deploy/docker-compose
docker-compose up -d
```

这会启动以下服务：
- **Etcd**: 配置中心 (:2379)
- **ZooKeeper**: Kafka 协调 (:2181)
- **Kafka**: 消息队列 (:9092, :9093)
- **Elasticsearch**: 日志存储 (:9200, :9300)
- **Kibana**: 日志可视化 (:5601)
- **Prometheus**: 指标收集 (:9090)
- **Grafana**: 监控看板 (:3000)

### 3. 等待服务启动

服务启动需要约 2-3 分钟。检查服务状态：

```bash
cd deploy/docker-compose
docker-compose ps
```

所有服务的状态应该显示为 `Up`。

---

## 基础设施启动

### 详细步骤

#### 步骤 1: 进入 docker-compose 目录
```bash
cd /Users/eka/Code/logos/deploy/docker-compose
```

#### 步骤 2: 启动服务
```bash
docker-compose up -d
```

输出示例：
```
Creating network "docker-compose_logging" with the default driver
Creating volume "docker-compose_etcd0-data" with default driver
Creating volume "docker-compose_kafka0-data" with default driver
Creating volume "docker-compose_es0-data" with default driver
Creating volume "docker-compose_prometheus-data" with default driver
Creating volume "docker-compose_grafana-data" with default driver
Creating etcd-0 ... done
Creating zookeeper-0 ... done
Creating elasticsearch-0 ... done
Creating prometheus ... done
Creating kafka-0 ... done
Creating kibana ... done
Creating grafana ... done
```

#### 步骤 3: 查看服务日志（可选）
```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f kafka-0
docker-compose logs -f elasticsearch-0
```

#### 步骤 4: 停止服务（如需要）
```bash
# 停止服务但保留数据
docker-compose stop

# 停止并删除容器
docker-compose down

# 停止并删除容器和数据卷（清空所有数据）
docker-compose down -v
```

---

## 服务验证

服务启动后，验证每个服务是否正常工作。

### 1. 验证 Etcd

```bash
curl http://localhost:2379/health
```

预期输出：
```json
{"health":"true","reason":""}
```

### 2. 验证 Kafka

#### 2.1 查看 Kafka Topic 列表
```bash
docker exec kafka-0 kafka-topics --list --bootstrap-server localhost:9092
```

#### 2.2 创建测试 Topic
```bash
docker exec kafka-0 kafka-topics --create \
  --topic logs \
  --partitions 1 \
  --replication-factor 1 \
  --bootstrap-server localhost:9092
```

#### 2.3 发送测试消息
```bash
docker exec -i kafka-0 kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic logs <<EOF
{"level":"INFO","message":"Hello from Kafka","service":"test"}
EOF
```

#### 2.4 消费测试消息
```bash
docker exec kafka-0 kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic logs \
  --from-beginning \
  --max-messages 1
```

按 `Ctrl+C` 停止消费。

### 3. 验证 Elasticsearch

#### 3.1 检查集群健康
```bash
curl 'http://localhost:9200/_cluster/health?pretty'
```

预期输出：
```json
{
  "cluster_name" : "docker-cluster",
  "status" : "green",
  "timed_out" : false,
  "number_of_nodes" : 1,
  "number_of_data_nodes" : 1,
  "active_primary_shards" : 0,
  "active_shards" : 0,
  "relocating_shards" : 0,
  "initializing_shards" : 0,
  "unassigned_shards" : 0,
  "delayed_unassigned_shards" : 0,
  "number_of_pending_tasks" : 0,
  "number_of_in_flight_fetch" : 0,
  "task_max_waiting_in_queue_millis" : 0,
  "active_shards_percent_as_number" : 100.0
}
```

#### 3.2 查看索引列表
```bash
curl 'http://localhost:9200/_cat/indices?v'
```

#### 3.3 创建测试索引
```bash
curl -X PUT 'http://localhost:9200/test-index'
```

#### 3.4 删除测试索引
```bash
curl -X DELETE 'http://localhost:9200/test-index'
```

### 4. 验证 Kibana

在浏览器中打开: http://localhost:5601

首次访问时，Kibana 会询问是否配置索引模式，可以稍后配置。

### 5. 验证 Grafana

在浏览器中打开: http://localhost:3000

默认登录凭据：
- 用户名: `admin`
- 密码: `admin`

首次登录后会要求修改密码。

### 6. 验证 Prometheus

在浏览器中打开: http://localhost:9090

可以在 Graph 页面执行查询，例如：
- `up` - 查看监控目标状态

---

## SDK 测试

### 1. 构建 SDK

```bash
cd /Users/eka/Code/logos/log-sdk/log-sdk
go build ./...
```

### 2. 运行 SDK 单元测试

```bash
cd /Users/eka/Code/logos/log-sdk/log-sdk
go test -v ./pkg/logger
```

预期输出示例：
```
=== RUN   TestNew
--- PASS: TestNew (0.00s)
=== RUN   TestLogger_Info
--- PASS: TestLogger_Info (0.00s)
=== RUN   TestLogger_Error
--- PASS: TestLogger_Error (0.00s)
PASS
ok      github.com/log-system/log-sdk/pkg/logger     0.002s
```

### 3. 运行所有 SDK 测试

```bash
cd /Users/eka/Code/logos/log-sdk/log-sdk
go test -v ./...
```

### 4. 运行 SDK 基准测试

```bash
cd /Users/eka/Code/logos/log-sdk/log-sdk
go test -bench="Benchmark" -benchmem ./pkg/logger
```

这会展示 SDK 的性能指标，包括：
- 每秒操作数
- 每次操作的内存分配
- 每次操作的分配次数

### 5. 运行示例程序

#### 5.1 基础 SDK 示例

```bash
cd /Users/eka/Code/logos/examples
go run sdk/main.go
```

这个示例会展示：
- 传统 API 风格（类似标准 log 包）
- 链式 API 风格（类似 Zap/ZeroLog）
- Hook 过滤功能
- With 字段继承

#### 5.2 HTTP 服务示例

```bash
cd /Users/eka/Code/logos/examples
go run http/main.go
```

然后在另一个终端测试：
```bash
# 测试 ping 端点
curl http://localhost:8080/ping

# 测试 hello 端点
curl http://localhost:8080/hello/world

# 测试创建订单
curl -X POST http://localhost:8080/order \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u123","product_id":"p456","amount":99.99}'
```

---

## 全链路测试

### 测试架构

```
┌─────────────────────────────────────────────────────────────┐
│  应用 (SDK)                    →  Kafka                      │
│  (生成结构化日志)                 (消息队列)                  │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Log Processor                →  Elasticsearch              │
│  (消费、解析、语义增强)          (日志存储)                  │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Log Analyzer / Kibana         →  用户界面                  │
│  (查询、分析、可视化)            (展示日志)                  │
└─────────────────────────────────────────────────────────────┘
```

### 步骤 1: 创建 Kafka Topic

```bash
docker exec kafka-0 kafka-topics --create \
  --topic logs \
  --partitions 1 \
  --replication-factor 1 \
  --bootstrap-server localhost:9092
```

如果 Topic 已存在会报错，可以忽略或先删除：
```bash
docker exec kafka-0 kafka-topics --delete \
  --topic logs \
  --bootstrap-server localhost:9092
```

### 步骤 2: 启动 Kafka 消费者（终端 1）

保持这个终端运行，观察消息：

```bash
docker exec kafka-0 kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic logs \
  --from-beginning
```

### 步骤 3: 发送测试日志（终端 2）

创建一个简单的测试程序：

```bash
cat > /tmp/test-log.go << 'EOF'
package main

import (
	"github.com/log-system/log-sdk/pkg/logger"
)

func main() {
	log := logger.New(logger.Config{
		ServiceName:       "full-demo",
		Environment:       "test",
		Cluster:           "local",
		Pod:               "test-pod-1",
		KafkaBrokers:      []string{"localhost:9092"},
		KafkaTopic:        "logs",
		FallbackToConsole: true,
		MaxBufferSize:     1000,
	})
	defer log.Close()

	// 发送测试日志
	log.Info("全链路测试开始").
		Str("test_id", "test-001").
		Send()

	log.Warn("这是一条警告日志").
		Str("component", "database").
		Int("retry_count", 3).
		Send()

	log.Error("这是一条错误日志").
		Str("error", "connection failed").
		Str("host", "db01.example.com").
		Send()

	log.Info("全链路测试完成")
}
EOF
```

注意：由于 SDK 的完整测试需要正确的模块路径配置，我们可以用更简单的方式测试。

### 步骤 3 (简化版): 直接用 Kafka 命令行工具测试

```bash
# 发送多条 JSON 格式的日志
docker exec -i kafka-0 kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic logs <<EOF
{"level":"INFO","timestamp":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","message":"用户登录","service":"auth-service","user_id":"u1001","ip":"192.168.1.100"}
{"level":"INFO","timestamp":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","message":"订单创建","service":"order-service","order_id":"ORD-001","user_id":"u1001","amount":299.99}
{"level":"WARN","timestamp":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","message":"响应时间过长","service":"api-gateway","endpoint":"/api/orders","duration_ms":1250}
{"level":"ERROR","timestamp":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","message":"支付失败","service":"payment-service","order_id":"ORD-002","error":"INSUFFICIENT_FUNDS"}
EOF
```

### 步骤 4: 在消费者终端观察

回到终端 1，你应该能看到刚才发送的日志消息。

### 步骤 5: 写入 Elasticsearch

创建测试索引并写入日志：

```bash
# 创建索引
curl -X PUT 'http://localhost:9200/logs-test' -H 'Content-Type: application/json' -d '{
  "settings": {
    "index": {
      "codec": "best_compression"
    }
  },
  "mappings": {
    "properties": {
      "timestamp": {"type": "date"},
      "level": {"type": "keyword"},
      "message": {"type": "text"},
      "service": {"type": "keyword"},
      "user_id": {"type": "keyword"},
      "order_id": {"type": "keyword"}
    }
  }
}'

# 写入日志文档
curl -X POST 'http://localhost:9200/logs-test/_doc' -H 'Content-Type: application/json' -d '{
  "timestamp": "'"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"'",
  "level": "INFO",
  "message": "用户登录",
  "service": "auth-service",
  "user_id": "u1001",
  "ip": "192.168.1.100"
}'

curl -X POST 'http://localhost:9200/logs-test/_doc' -H 'Content-Type: application/json' -d '{
  "timestamp": "'"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"'",
  "level": "ERROR",
  "message": "支付失败",
  "service": "payment-service",
  "order_id": "ORD-001",
  "error": "INSUFFICIENT_FUNDS"
}'

# 刷新索引使文档可搜索
curl -X POST 'http://localhost:9200/logs-test/_refresh'
```

### 步骤 6: 从 Elasticsearch 查询

```bash
# 搜索所有文档
curl 'http://localhost:9200/logs-test/_search?pretty'

# 只搜索 ERROR 级别日志
curl 'http://localhost:9200/logs-test/_search?pretty' -H 'Content-Type: application/json' -d '{
  "query": {
    "term": {
      "level": "ERROR"
    }
  }
}'

# 按时间范围查询
curl 'http://localhost:9200/logs-test/_search?pretty' -H 'Content-Type: application/json' -d '{
  "query": {
    "range": {
      "timestamp": {
        "gte": "now-1h"
      }
    }
  },
  "sort": [
    {"timestamp": "desc"}
  ]
}'
```

### 步骤 7: 在 Kibana 中查看

1. 打开 http://localhost:5601
2. 进入 "Management" → "Index Patterns"
3. 点击 "Create index pattern"
4. 输入 `logs-test` 作为索引模式
5. 选择 `timestamp` 作为时间字段
6. 点击 "Create index pattern"
7. 进入 "Discover" 页面查看日志

---

## 常见问题

### Q: Docker 容器启动失败怎么办？

A: 检查以下几点：
1. 端口是否被占用：`lsof -i :2379 -i :9092 -i :9200`
2. Docker 资源是否足够：Docker Desktop → Settings → Resources
3. 查看容器日志：`docker-compose logs <service-name>`

### Q: Kafka 连接失败？

A: 确认以下几点：
1. Kafka 容器是否正常运行：`docker-compose ps kafka-0`
2. 等待 Kafka 完全启动（约 30-60 秒）
3. 检查 Kafka 日志：`docker-compose logs kafka-0`

### Q: Elasticsearch 健康状态是 yellow？

A: 这是正常的，因为我们只有一个节点，副本无法分配。可以生产环境使用 3 个节点的集群。

### Q: 如何清理所有数据重新开始？

A:
```bash
cd deploy/docker-compose
docker-compose down -v
docker-compose up -d
```

### Q: SDK 测试时提示找不到模块？

A: 确保在正确的目录下，并且 Go 模块已正确设置：
```bash
cd /Users/eka/Code/logos/log-sdk/log-sdk
go mod tidy
```

### Q: 如何查看所有服务的资源使用情况？

A:
```bash
docker stats
```

---

## 下一步

完成基础测试后，你可以：

1. **启动 Log Processor**: 消费 Kafka 日志并写入 Elasticsearch
2. **启动 Log Analyzer**: 提供日志查询 API
3. **部署到 Kubernetes**: 使用 `deploy/k8s/` 下的配置
4. **查看文档**: `docs/` 目录下有更详细的架构和设计文档

---

## 附录

### 快速参考命令

```bash
# 进入项目根目录
cd /Users/eka/Code/logos

# 启动所有基础设施
cd deploy/docker-compose && docker-compose up -d

# 查看服务状态
cd deploy/docker-compose && docker-compose ps

# 查看服务日志
cd deploy/docker-compose && docker-compose logs -f

# 停止服务
cd deploy/docker-compose && docker-compose down

# 运行 SDK 测试
cd log-sdk/log-sdk && go test -v ./...

# 运行 SDK 基准测试
cd log-sdk/log-sdk && go test -bench="Benchmark" -benchmem ./pkg/logger
```

### 服务地址速查

| 服务 | 地址 | 说明 |
|------|------|------|
| Etcd | http://localhost:2379 | 配置中心 |
| Kafka | localhost:9092 | 消息队列（内部） |
| Kafka | localhost:9093 | 消息队列（外部） |
| Elasticsearch | http://localhost:9200 | 日志存储 |
| Kibana | http://localhost:5601 | 日志可视化 |
| Prometheus | http://localhost:9090 | 指标收集 |
| Grafana | http://localhost:3000 | 监控看板 (admin/admin) |

---

祝测试顺利！如有问题，请查看项目 README.md 或提交 Issue。
