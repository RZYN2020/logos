# Log Analyzer 部署和运维文档

## 系统要求

- Go 1.25+
- PostgreSQL 14+
- ETCD 3.5+
- Node.js 18+ (前端)

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| PORT | 服务端口 | 8080 |
| DATABASE_URL | PostgreSQL 连接字符串 | host=localhost user=postgres password=postgres dbname=logAnalyzer port=5432 sslmode=disable |
| ETCD_ENDPOINTS | ETCD 地址 | localhost:2379 |
| ES_ADDRESSES | Elasticsearch 地址 | http://localhost:9200 |
| JWT_SECRET | JWT 密钥 | logos-default-secret-key-change-in-production |
| LOG_LEVEL | 日志级别 | info |

## 本地开发部署

### 1. 启动依赖服务

```bash
# 启动 PostgreSQL
docker run -d --name postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=logAnalyzer \
  -p 5432:5432 \
  postgres:14

# 启动 ETCD
docker run -d --name etcd \
  -p 2379:2379 \
  quay.io/coreos/etcd:v3.5.9 \
  /usr/local/bin/etcd \
  --advertise-client-urls http://0.0.0.0:2379 \
  --listen-client-urls http://0.0.0.0:2379
```

### 2. 构建后端

```bash
cd log-analyzer

# 下载依赖
go mod tidy

# 构建
go build -o log-analyzer ./cmd/server

# 运行
./log-analyzer
```

### 3. 构建前端

```bash
cd frontend

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
```

## Docker 部署

### 后端 Dockerfile

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/log-analyzer ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/log-analyzer .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
CMD ["./log-analyzer"]
```

### 前端 Dockerfile

```dockerfile
FROM node:18-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: logAnalyzer
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  etcd:
    image: quay.io/coreos/etcd:v3.5.9
    command: /usr/local/bin/etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379
    ports:
      - "2379:2379"

  log-analyzer:
    build: ./log-analyzer
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/logAnalyzer?sslmode=disable
      ETCD_ENDPOINTS: etcd:2379
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - etcd

  frontend:
    build: ./frontend
    ports:
      - "80:80"
    depends_on:
      - log-analyzer

volumes:
  postgres_data:
```

部署：

```bash
docker-compose up -d
```

## Kubernetes 部署

### 配置 ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: log-analyzer-config
data:
  PORT: "8080"
  LOG_LEVEL: "info"
```

### 配置 Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: log-analyzer-secret
type: Opaque
stringData:
  DATABASE_URL: "postgres://postgres:password@postgres:5432/logAnalyzer?sslmode=disable"
  ETCD_ENDPOINTS: "etcd:2379"
  JWT_SECRET: "your-secret-key-here"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: log-analyzer
spec:
  replicas: 3
  selector:
    matchLabels:
      app: log-analyzer
  template:
    metadata:
      labels:
        app: log-analyzer
    spec:
      containers:
      - name: log-analyzer
        image: log-analyzer:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: log-analyzer-config
        - secretRef:
            name: log-analyzer-secret
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: log-analyzer-service
spec:
  selector:
    app: log-analyzer
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 运维监控

### 健康检查端点

- `/health` - 健康检查（负载均衡使用）
- `/ready` - 就绪检查（服务发现使用）

### 日志

日志输出到 stdout，建议配置日志收集：

```bash
# 使用 journald
./log-analyzer 2>&1 | journalctl -u log-analyzer -f

# 使用 file
./log-analyzer >> /var/log/log-analyzer.log 2>&1
```

### 指标监控

建议配置以下监控指标：

1. **HTTP 请求指标**
   - 请求总数
   - 请求延迟（p50, p95, p99）
   - 错误率

2. **数据库指标**
   - 连接池使用情况
   - 查询延迟
   - 慢查询数量

3. **ETCD 指标**
   - 连接状态
   - 读写延迟
   - 配置分发成功率

### 告警规则

建议配置以下告警：

1. 服务不可用（健康检查失败）
2. 错误率超过 5%
3. 响应时间超过 1 秒
4. 数据库连接失败
5. ETCD 连接失败

## 备份和恢复

### 数据库备份

```bash
# 备份
pg_dump "postgresql://postgres:password@localhost:5432/logAnalyzer" > backup.sql

# 恢复
psql "postgresql://postgres:password@localhost:5432/logAnalyzer" < backup.sql
```

### ETCD 备份

```bash
# 备份
etcdctl snapshot save backup.db

# 恢复
etcdctl snapshot restore backup.db
```

## 性能优化

### 数据库优化

1. 创建索引
```sql
CREATE INDEX idx_rules_name ON rules(name);
CREATE INDEX idx_rule_versions ON rule_versions(rule_id);
```

2. 连接池配置
```go
// 在 main.go 中配置
db.SetMaxIdleConns(10)
db.SetMaxOpenConns(100)
db.SetConnMaxLifetime(time.Hour)
```

### 缓存策略

1. 规则列表缓存（5 分钟）
2. 系统信息缓存（1 分钟）

### 水平扩展

通过增加副本数实现水平扩展：

```bash
kubectl scale deployment log-analyzer --replicas=5
```

## 故障排查

### 常见问题

1. **无法连接数据库**
   - 检查 DATABASE_URL 配置
   - 确认 PostgreSQL 服务运行
   - 检查网络连接

2. **无法连接 ETCD**
   - 检查 ETCD_ENDPOINTS 配置
   - 确认 ETCD 服务运行
   - 使用 `etcdctl endpoint health` 检查状态

3. **JWT 认证失败**
   - 检查 JWT_SECRET 配置
   - 确认 token 未过期
   - 检查 Authorization header 格式

### 日志级别

调整 LOG_LEVEL 获取更多调试信息：

```bash
export LOG_LEVEL=debug
./log-analyzer
```

## 安全建议

1. **修改默认 JWT 密钥**
   ```bash
   export JWT_SECRET=$(openssl rand -base64 32)
   ```

2. **修改默认管理员密码**
   - 首次登录后立即修改

3. **启用 HTTPS**
   - 在生产环境中使用反向代理（如 Nginx）配置 HTTPS

4. **配置防火墙**
   - 只开放必要的端口（80/443）

5. **定期更新依赖**
   ```bash
   go get -u ./...
   npm update
   ```
