#!/bin/bash

# 应用所有Kubernetes配置
echo "开始部署Kubernetes资源..."

# 创建命名空间
kubectl apply -f namespace.yaml

# 部署Redis
kubectl apply -f redis-pvc.yaml
kubectl apply -f redis-deployment.yaml

# 等待Redis就绪
echo "等待Redis服务就绪..."
kubectl wait --for=condition=ready pod -l app=redis -n voting-app --timeout=60s

# 部署gRPC服务
kubectl apply -f grpc-deployment.yaml

# 等待gRPC服务就绪
echo "等待gRPC服务就绪..."
kubectl wait --for=condition=ready pod -l app=grpcserver -n voting-app --timeout=60s

# 部署HTTP服务
kubectl apply -f http-deployment.yaml

# 等待HTTP服务就绪
echo "等待HTTP服务就绪..."
kubectl wait --for=condition=ready pod -l app=httpserver -n voting-app --timeout=60s

# 部署监控组件
echo "部署Prometheus..."
kubectl apply -f prometheus-pvc.yaml
kubectl apply -f prometheus-configmap.yaml
kubectl apply -f prometheus-deployment.yaml

echo "等待Prometheus就绪..."
kubectl wait --for=condition=ready pod -l app=prometheus -n voting-app --timeout=60s

echo "部署Grafana..."
kubectl apply -f grafana-pvc.yaml
kubectl apply -f grafana-deployment.yaml

echo "等待Grafana就绪..."
kubectl wait --for=condition=ready pod -l app=grafana -n voting-app --timeout=60s

# 部署Ingress
kubectl apply -f ingress.yaml

echo "部署完成！"
echo "服务访问地址：http://voting-app.local/"
echo ""
echo "监控服务地址："
echo "- Prometheus: http://prometheus-service:9090"
echo "- Grafana: http://grafana-service:3000 (admin/admin123)"
echo ""
echo "或者使用端口转发："
echo "kubectl port-forward -n voting-app service/httpserver-service 8080:8080"
echo "kubectl port-forward -n voting-app service/prometheus-service 9090:9090"
echo "kubectl port-forward -n voting-app service/grafana-service 3000:3000"
echo ""
echo "然后访问："
echo "- 投票应用: http://localhost:8080/"
echo "- Prometheus: http://localhost:9090"
echo "- Grafana: http://localhost:3000"