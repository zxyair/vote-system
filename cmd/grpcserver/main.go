package main

import (
	"context"
	"log"
	"net"
	"os"

	votingv1 "vote-system/internal/gen/voting/v1"
	grpcserver "vote-system/internal/grpc/server"
	"vote-system/internal/service"
	memorystore "vote-system/internal/store/memory"
	redisstore "vote-system/internal/store/redis"

	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	grpcAddr := getenv("GRPC_ADDR", ":9090")
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// 创建gRPC服务器
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", grpcAddr, err)
	}

	var store service.Store = memorystore.New()
	if redisAddr != "" {
		rdb := goredis.NewClient(&goredis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
		})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			log.Fatalf("redis ping %s: %v", redisAddr, err)
		}
		store = redisstore.New(rdb)
		log.Printf("using redis store %s", redisAddr)
	} else {
		log.Printf("REDIS_ADDR not set, using memory store")
	}
	svc := service.New(store)

	// 创建gRPC服务器
	s := grpc.NewServer()
	votingv1.RegisterVotingServiceServer(s, grpcserver.New(svc))

	// 启动gRPC服务器
	log.Printf("gRPC server running %s", grpcAddr)
	if err := s.Serve(lis); err != nil {
		log.Printf("grpc serve error: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}