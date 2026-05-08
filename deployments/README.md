# 在线投票系统部署指南

## 项目概述

这是一个基于Go语言开发的在线投票系统，采用微服务架构，支持高并发访问和实时更新。

## 系统架构

### 服务组件
- **httpserver**: HTTP网关服务，处理前端请求并转发给gRPC服务
- **grpcserver**: 业务逻辑服务，处理投票核心逻辑
- **redis**: 数据存储服务，存储投票数据
- **prometheus**: 监控指标收集
- **grafana**: 监控数据可视化

### 部署方式
- **容器化**: 使用Docker进行容器化
- **编排**: 使用Kubernetes进行服务编排
- **监控**: Prometheus + Grafana

## 快速开始

### 1. 本地开发运行

#### 启动Redis
```bash
# Windows
redis-server --port 6379
```

#### 启动gRPC服务
```powershell
cd D:\CStudy\voting-app\vote-system
$env:GRPC_ADDR=":9090"
$env:REDIS_ADDR="127.0.0.1:6379"
go run ./cmd/grpcserver
```

#### 启动HTTP服务
```powershell
cd D:\CStudy\voting-app\vote-system
$env:HTTP_ADDR=":8080"
$env:GRPC_ADDR="127.0.0.1:9090"
go run ./cmd/httpserver
```

#### 访问应用
浏览器访问: http://localhost:8080/

### 2. Docker部署

#### 构建镜像
```bash
cd D:\CStudy\voting-app\vote-system\build\package
./docker-build.sh
```

#### 运行容器
```bash
# 启动Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine

# 启动gRPC服务
docker run -d --name grpcserver -p 9090:9090 \
  -e GRPC_ADDR=:9090 \
  -e REDIS_ADDR=redis:6379 \
  grpcserver:latest

# 启动HTTP服务
docker run -d --name httpserver -p 8080:8080 \
  -e HTTP_ADDR=:8080 \
  -e GRPC_ADDR=grpcserver:9090 \
  httpserver:latest
```

### 3. Kubernetes部署

#### 环境要求
- Kubernetes集群 (minikube, kind, EKS, GKE等)
- kubectl命令行工具
- helm (可选)

#### 部署步骤
```bash
# 应用所有配置
cd D:\CStudy\voting-app\vote-system\deployments\k8s
./apply-all.sh
```

#### 服务访问
- 投票应用: http://voting-app.local/
- Prometheus: http://prometheus-service:9090
- Grafana: http://grafana-service:3000 (admin/admin123)

#### 端口转发（本地访问）
```bash
# 投票应用
kubectl port-forward -n voting-app service/httpserver-service 8080:8080

# Prometheus
kubectl port-forward -n voting-app service/prometheus-service 9090:9090

# Grafana
kubectl port-forward -n voting-app service/grafana-service 3000:3000
```

## 配置说明

### 环境变量

#### httpserver
- `HTTP_ADDR`: HTTP服务监听地址 (默认: :8080)
- `GRPC_ADDR`: gRPC服务地址 (默认: :9090)
- `HEALTH_ADDR`: 健康检查服务地址 (默认: :8081)

#### grpcserver
- `GRPC_ADDR`: gRPC服务监听地址 (默认: :9090)
- `REDIS_ADDR`: Redis服务地址 (默认: 空，使用内存存储)
- `REDIS_PASSWORD`: Redis密码 (可选)
- `HEALTH_ADDR`: 健康检查服务地址 (默认: :9091)

### 健康检查端点
- `/health`: 基本健康检查
- `/ready`: 就绪检查
- `/metrics`: Prometheus指标端点

## 监控指标

系统收集以下指标：

### HTTP指标
- `http_requests_total`: HTTP请求总数
- `http_request_duration_seconds`: HTTP请求持续时间

### gRPC指标
- `grpc_requests_total`: gRPC请求总数
- `grpc_request_duration_seconds`: gRPC请求持续时间

### Redis指标
- `redis_operations_total`: Redis操作总数

### 业务指标
- `active_polls_count`: 活跃投票数
- `total_votes_count`: 总投票数

## API文档

主要HTTP接口：

- `POST /polls/createPoll`: 创建投票
- `GET /polls/close/:id`: 关闭投票
- `GET /polls/delete/:id`: 删除投票
- `POST /votes/:poll_id/vote`: 投票
- `DELETE /votes/:poll_id/vote`: 撤销投票
- `GET /polls/search`: 搜索投票
- `GET /polls/get_polls_by_vote`: 获取投票统计

所有接口都需要请求头 `X-User-Id`。

## 故障排查

### 常见问题

1. **连接失败**
   - 检查Redis服务是否启动
   - 检查服务端口是否被占用

2. **部署失败**
   - 检查Kubernetes集群状态
   - 查看Pod日志: `kubectl logs -n voting-app <pod-name>`

3. **监控数据不显示**
   - 检查Prometheus配置
   - 确认指标端点可访问

### 日志查看

```bash
# 查看所有Pod日志
kubectl logs -n voting-app -l app=<app-name>

# 实时查看日志
kubectl logs -n voting-app -l app=<app-name> -f
```

## 性能优化

### Redis优化
- 使用Redis Cluster支持大规模数据
- 配置适当的持久化策略
- 设置合理的内存限制

### Kubernetes优化
- 根据负载调整Pod数量
- 配置资源限制和请求
- 使用HPA (Horizontal Pod Autoscaler)

### 监控优化
- 调整Prometheus采集频率
- 配置合理的 retention policy
- 使用Grafana Dashboard进行可视化

## 扩展功能

### 可选扩展
- 用户认证和授权
- 投票结果导出
- 邮件通知
- 数据分析报告

### 部署扩展
- 多区域部署
- 负载均衡配置
- CDN集成
- 数据库备份策略