package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	votingv1 "vote-system/internal/gen/voting/v1"
	grpcserver "vote-system/internal/grpc/server"
	"vote-system/internal/obs"
	"vote-system/internal/service"
	memorystore "vote-system/internal/store/memory"
	redisstore "vote-system/internal/store/redis"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	addr := getenv("GRPC_ADDR", ":9090")
	metricsAddr := getenv("METRICS_ADDR", ":2112")
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
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
	obs.RegisterAll()
	svc := service.New(store)

	s := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.MetricsUnaryInterceptor()))
	votingv1.RegisterVotingServiceServer(s, grpcserver.New(svc))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("gRPC server running %s", addr)
		if err := s.Serve(lis); err != nil {
			log.Printf("grpc serve error: %v", err)
		}
	}()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		srv := &http.Server{Addr: metricsAddr, Handler: mux}
		log.Printf("metrics server running %s", metricsAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics serve error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gRPC server")
	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		s.Stop()
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
