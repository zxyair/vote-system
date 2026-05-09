# Docker 构建说明

## 单独构建镜像

```powershell
docker build -f build/docker/Dockerfile.redis -t vote-redis:latest .
docker build -f build/docker/Dockerfile.httpserver -t vote-httpserver:latest .
docker build -f build/docker/Dockerfile.grpcserver -t vote-grpcserver:latest .
```

## 本地容器联调

```powershell
docker compose up --build
```

访问地址：

- Web 页面：http://127.0.0.1:8080/
- HTTP 健康检查：http://127.0.0.1:8080/healthz

`docker-compose.yml` 中 Redis 默认密码为 `redis123`，与当前本地测试环境保持一致。Redis 使用 `vote-redis:latest` 镜像，只暴露在 compose 内部网络，不占用宿主机 `6379` 端口。

Redis 配置文件位于 `build/docker/redis.conf`，compose 使用 `redis-data` 命名卷保存 Redis 数据。
