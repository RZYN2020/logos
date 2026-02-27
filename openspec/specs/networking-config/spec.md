# networking-config Specification

## Purpose
TBD - created by archiving change kubernetes-framework-setup. Update Purpose after archive.
## Requirements
### Requirement: Kubernetes Network Policies must be configured
系统 SHALL 配置 Kubernetes Network Policies 以控制服务间的通信。

#### Scenario: Kafka communication policy
- **WHEN** Log SDK 或 Log Processor 尝试访问 Kafka
- **THEN** Network Policy 允许通信
- **AND** 其他服务的访问被拒绝

#### Scenario: Elasticsearch communication policy
- **WHEN** Log Processor 或 Log Analyzer 尝试访问 Elasticsearch
- **THEN** Network Policy 允许通信
- **AND** 其他服务的访问被拒绝

### Requirement: Services must be exposed through Ingress
系统 SHALL 配置 Kubernetes Ingress 以暴露外部服务。

#### Scenario: Expose Frontend
- **WHEN** 用户访问前端界面
- **THEN** Ingress 路由请求到 Frontend Service
- **AND** 配置路径为 /

#### Scenario: Expose Config Server
- **WHEN** 用户访问配置服务器
- **THEN** Ingress 路由请求到 Config Server Service
- **AND** 配置路径为 /api/v1

### Requirement: External services must be accessible
系统 SHALL 配置 Kubernetes Service 以暴露需要外部访问的服务。

#### Scenario: Kafka external access
- **WHEN** Log SDK 从外部访问 Kafka
- **THEN** Kafka Service 暴露外部端口
- **AND** 配置为 NodePort 或 LoadBalancer 类型

#### Scenario: Elasticsearch external access
- **WHEN** 用户从外部访问 Elasticsearch
- **THEN** Elasticsearch Service 暴露外部端口
- **AND** 配置为 NodePort 或 LoadBalancer 类型

