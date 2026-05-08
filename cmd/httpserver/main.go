package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	votingv1 "vote-system/internal/gen/voting/v1"
	"vote-system/internal/http/router"
	"vote-system/internal/monitor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gin-gonic/gin"
)

func main() {
	httpAddr := getenv("HTTP_ADDR", ":8080")
	grpcAddr := getenv("GRPC_ADDR", ":9090")
	healthAddr := getenv("HEALTH_ADDR", ":8081")

	// 初始化监控指标
	monitor.InitMetrics()

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc dial %s: %v", grpcAddr, err)
	}
	defer conn.Close()

	client := votingv1.NewVotingServiceClient(conn)
	r := router.SetupRouter(client)

	// 添加监控端点
	r.GET("/metrics", func(c *gin.Context) {
		monitor.GetPrometheusHandler().ServeHTTP(c.Writer, c.Request)
	})

	// 创建主HTTP服务器
	srv := &http.Server{
		Addr:    httpAddr,
		Handler: r,
	}

	// 创建健康检查HTTP服务器
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// 检查gRPC连接
		if conn != nil {
			// 尝试ping gRPC服务器
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// 简单的gRPC健康检查
			resp, err := client.Ping(ctx, &votingv1.PingRequest{})
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("gRPC server unavailable"))
				return
			}
			if resp.Status != "ok" {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("gRPC server not ready"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})

	healthServer := &http.Server{
		Addr:    healthAddr,
		Handler: healthMux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("HTTP server running %s (grpc=%s)", httpAddr, grpcAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http serve error: %v", err)
		}
	}()

	go func() {
		log.Printf("Health check server running %s", healthAddr)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("health server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down servers")

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		healthServer.Shutdown(shutdownCtx)
		srv.Shutdown(shutdownCtx)
	}()

	select {
	case <-shutdownCtx.Done():
		log.Println("shutdown timeout")
	case <-time.After(5 * time.Second):
		log.Println("forced shutdown")
	}

}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
