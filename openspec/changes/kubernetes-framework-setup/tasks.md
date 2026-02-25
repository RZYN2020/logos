## 1. Kubernetes 部署配置

- [x] 1.1 创建 Log SDK 的 Helm Chart
- [x] 1.2 创建 Config Server 的 Helm Chart
- [x] 1.3 创建 Log Processor 的 Helm Chart
- [x] 1.4 创建 Log Analyzer 的 Helm Chart
- [x] 1.5 创建 Frontend 的 Helm Chart

## 2. 网络和通信配置

- [x] 2.1 创建 Network Policies 配置文件
- [x] 2.2 创建 Ingress 资源配置
- [x] 2.3 配置服务间的通信策略
- [ ] 2.4 配置外部访问的 LoadBalancer 服务  # TODO

## 3. 存储配置

- [x] 3.1 创建 Etcd 的 PVC 配置
- [x] 3.2 创建 Kafka 的 PVC 配置
- [x] 3.3 创建 Elasticsearch 的 PVC 配置
- [x] 3.4 创建 PostgreSQL 的 PVC 配置
- [x] 3.5 配置存储类（StorageClasses）

## 4. 监控和日志收集

- [x] 4.1 部署 Prometheus 和配置刮取规则
- [x] 4.2 部署 Grafana 和预设监控面板
- [x] 4.3 部署 Jaeger 和配置追踪收集
- [x] 4.4 配置 Prometheus Alertmanager 告警规则
- [x] 4.5 部署 Fluentd 作为 DaemonSet

## 5. CI/CD 和部署工具

- [x] 5.1 创建项目级别的 Helm Chart（umbrella Chart）
- [x] 5.2 配置 values.yaml 支持多环境部署
- [x] 5.3 创建部署脚本和操作指南
- [ ] 5.4 测试 Helm Chart 部署过程  # TODO - helm not available

## 6. README 文档更新

- [x] 6.1 添加 Kubernetes 部署说明
- [x] 6.2 描述每个组件的搭建方法
- [x] 6.3 提供部署前的准备工作
- [x] 6.4 编写部署和验证步骤
- [x] 6.5 包含常见问题和解决方案

## 7. 验证和测试

- [ ] 7.1 在 Minikube 上测试完整部署流程  # TODO
- [ ] 7.2 验证服务间的通信和网络策略  # TODO
- [ ] 7.3 测试监控和日志收集系统  # TODO
- [ ] 7.4 进行压力测试和性能测试  # TODO
- [ ] 7.5 修复测试中发现的问题  # TODO
