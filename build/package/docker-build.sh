#!/bin/bash

# Docker构建脚本
echo "开始构建Docker镜像..."

# 构建grpcserver镜像
echo "构建grpcserver镜像..."
docker build -f Dockerfile.grpcserver -t grpcserver:latest .

# 构建httpserver镜像
echo "构建httpserver镜像..."
docker build -f Dockerfile.httpserver -t httpserver:latest .

echo "Docker镜像构建完成！"

# 显示镜像列表
echo "已构建的镜像："
docker images | grep -E "(grpcserver|httpserver)"