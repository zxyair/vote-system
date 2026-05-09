# Kubernetes 部署说明

## 部署资源

```powershell
kubectl apply -f deployments/k8s/
```

也可以使用 kustomize：

```powershell
kubectl apply -k deployments/k8s/
```

## 查看状态

```powershell
kubectl get nodes
kubectl get pods -n vote-system -o wide
kubectl get svc -n vote-system
kubectl get deploy -n vote-system
```

## 访问方式

当前最小交付版本使用 `NodePort` 暴露 HTTP 服务。

本机 kind 访问地址：

```text
http://localhost:30080/
```

通用 NodePort 访问方式：

```text
http://任意节点公网IP:30080/
```

## 镜像要求

Deployment 默认使用以下镜像：

```text
vote-redis:latest
vote-grpcserver:latest
vote-httpserver:latest
```

如果镜像没有推送到镜像仓库，需要先把镜像导入每台 Kubernetes 节点的 containerd：

```bash
sudo ctr -n k8s.io images import vote-redis.tar
sudo ctr -n k8s.io images import vote-grpcserver.tar
sudo ctr -n k8s.io images import vote-httpserver.tar
```

## 验收标准

- `httpserver` 为 3 个 Ready Pod。
- `grpcserver` 为 3 个 Ready Pod。
- `redis` 为 1 个 Ready Pod。
- `prometheus` 为 1 个 Ready Pod。
- `grafana` 为 1 个 Ready Pod。
- `http://任意节点公网IP:30080/healthz` 返回 200。
- Web 页面可以完成创建投票、投票、撤销投票和查看结果。

## 可观测性

当前清单包含一个轻量 Prometheus Deployment 和一个 Grafana Deployment。Prometheus 会通过 Pod 注解自动抓取：

- `httpserver`：`/metrics`，端口 `8080`
- `grpcserver`：`/metrics`，端口 `9091`

本地访问 Prometheus：

```powershell
kubectl port-forward -n vote-system svc/prometheus 19090:9090
```

浏览器打开：

```text
http://localhost:19090/
```

查看抓取目标：

```text
http://localhost:19090/targets
```

常用 PromQL：

```promql
vote_http_requests_total
vote_http_request_duration_seconds_count
vote_grpc_requests_total
vote_grpc_request_duration_seconds_count
vote_grpc_errors_total
```

本地访问 Grafana：

```powershell
kubectl port-forward -n vote-system svc/grafana 30300:3000
```

浏览器打开：

```text
http://localhost:30300/
```

默认账号：

```text
admin / admin123
```

Grafana 会自动加载 Prometheus 数据源和 `Vote System Observability` 看板。该看板包含：

- HTTP 请求速率。
- HTTP p50/p95 延迟。
- gRPC 请求速率。
- gRPC p50/p95 延迟。
- gRPC 错误速率。
- Go goroutine 数。
