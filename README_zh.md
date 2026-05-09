# 在线投票系统

## 项目介绍

这是一个基于 Go、gRPC、Redis、Docker 和 Kubernetes 的在线投票系统，提供投票创建、投票、撤销投票、结果查询和实时结果刷新能力。

系统链路：

```text
Web 前端 -> HTTP 服务 -> gRPC 服务 -> Redis
```

主要能力：

- 创建、查询、关闭和删除投票。
- 用户投票与撤销投票。
- 防止同一用户重复投票。
- 使用 Redis Lua 脚本保证投票和撤销操作的原子性。
- 支持 SSE 实时刷新投票结果。
- 支持 Docker Compose 和 kind 本机 Kubernetes 部署。

## 技术栈

| 模块 | 技术 |
|---|---|
| 前端 | HTML、CSS、JavaScript |
| HTTP 服务 | Go、Gin |
| RPC | gRPC、Protocol Buffers |
| 存储 | Redis |
| 容器化 | Docker、Docker Compose |
| 编排 | Kubernetes、kind |
| 可观测性 | Prometheus、Grafana |

## 本地直接运行

前提：本机 Redis 已启动，地址为 `127.0.0.1:6379`，密码为 `redis123`。

启动 gRPC 服务：

```powershell
$env:GRPC_ADDR=":9090"
$env:METRICS_ADDR=":9091"
$env:REDIS_ADDR="127.0.0.1:6379"
$env:REDIS_PASSWORD="redis123"
go run ./cmd/grpcserver
```

启动 HTTP 服务：

```powershell
$env:HTTP_ADDR=":8080"
$env:GRPC_ADDR="127.0.0.1:9090"
go run ./cmd/httpserver
```

访问：

```text
http://localhost:8080/
```

## Docker Compose 部署

构建并启动服务：

```powershell
docker compose up --build
```

访问：

```text
http://localhost:8080/
```

停止服务：

```powershell
docker compose down
```

## Kubernetes 本机部署

当前 Kubernetes 部署方案使用 kind。

创建集群：

```powershell
kind create cluster --name vote-system --config deployments/kind/kind-config.yaml
```

构建镜像：

```powershell
docker build -f build/docker/Dockerfile.redis -t vote-redis:latest .
docker build -f build/docker/Dockerfile.grpcserver -t vote-grpcserver:latest .
docker build -f build/docker/Dockerfile.httpserver -t vote-httpserver:latest .
```

导入镜像：

```powershell
kind load docker-image vote-redis:latest --name vote-system
kind load docker-image vote-grpcserver:latest --name vote-system
kind load docker-image vote-httpserver:latest --name vote-system
```

部署应用：

```powershell
kubectl apply -k deployments/k8s/
```

查看状态：

```powershell
kubectl get pods -n vote-system -o wide
kubectl get svc -n vote-system
```

访问 Web：

```text
http://localhost:30080/
```

## 可观测性入口

Kubernetes 部署包含 Prometheus 和 Grafana。

Prometheus：

```powershell
kubectl port-forward -n vote-system svc/prometheus 19090:9090
```

```text
http://localhost:19090/
```

Grafana：

```powershell
kubectl port-forward -n vote-system svc/grafana 30300:3000
```

```text
http://localhost:30300/
```

默认账号：

```text
admin / admin123
```
