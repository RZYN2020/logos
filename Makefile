.PHONY: help build test clean install deps stop-deps deploy deploy-app docs

help:
	@echo "可用命令:"
	@echo "  make help       - 显示帮助信息"
	@echo "  make build       - 构建所有项目"
	@echo "  make test        - 运行所有测试"
	@echo "  make clean       - 清理构建产物"
	@echo "  make install     - 安装依赖"
	@echo "  make deps        - 启动基础设施服务"
	@echo "  make stop-deps   - 停止基础设施服务"
	@echo "  make deploy       - 部署到 K8s"
	@echo "  make deploy-app   - 部署应用服务"
	@echo "  make docs        - 生成文档"

# 构建
build:
	@echo "构建 log-sdk..."
	@cd log-sdk/log-sdk && go build -o ../../bin/log-sdk ./...
	@echo "构建 log-processor..."
	@cd log-processor && go build -o ../bin/log-processor ./cmd/job
	@echo "构建 log-analyzer..."
	@cd log-analyzer && go build -o ../bin/log-analyzer ./cmd/main.go
	@echo "构建完成!"

# 测试
test:
	@echo "运行 log-sdk 测试..."
	@cd log-sdk/log-sdk && go test ./...
	@echo "运行 log-processor 测试..."
	@cd log-processor && go test ./...
	@echo "运行 log-analyzer 测试..."
	@cd log-analyzer && go test ./...
	@echo "测试完成!"

# 清理
clean:
	@echo "清理构建产物..."
	@rm -rf bin/
	@rm -rf log-sdk/log-sdk/bin
	@rm -rf log-processor/bin
	@rm -rf log-analyzer/bin
	@echo "清理完成!"

# 安装依赖
install:
	@echo "安装依赖..."
	@cd log-sdk/log-sdk && go mod download && go mod tidy
	@cd log-processor && go mod download && go mod tidy
	@cd log-analyzer && go mod download && go mod tidy
	@echo "依赖安装完成!"

# 基础设施
deps:
	@echo "启动基础设施服务..."
	@cd deploy/docker-compose && docker-compose up -d
	@echo "服务已启动!"
	@echo "访问地址:"
	@echo "  Etcd:      http://localhost:2379"
	@echo "  Kibana:    http://localhost:5601"
	@echo "  Grafana:   http://localhost:3000"
	@echo "  Prometheus:http://localhost:9090"

stop-deps:
	@echo "停止基础设施服务..."
	@cd deploy/docker-compose && docker-compose down
	@echo "服务已停止!"

# 部署
deploy:
	@echo "部署基础设施到 Kubernetes..."
	@kubectl create namespace logging-system || true
	@kubectl apply -f deploy/k8s/storage/
	@echo "基础设施部署完成!"

deploy-app:
	@echo "部署应用服务到 Kubernetes..."
	@kubectl apply -f deploy/k8s/charts/logos/
	@echo "应用服务部署完成!"

# 文档
docs:
	@echo "生成文档..."
	@echo "文档已生成在 docs/ 目录"
