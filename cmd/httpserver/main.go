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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	httpAddr := getenv("HTTP_ADDR", ":8080")
	grpcAddr := getenv("GRPC_ADDR", ":9090")

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc dial %s: %v", grpcAddr, err)
	}
	defer conn.Close()

	client := votingv1.NewVotingServiceClient(conn)
	r := router.SetupRouter(client)

	srv := &http.Server{
		Addr:    httpAddr,
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("HTTP server running %s (grpc=%s)", httpAddr, grpcAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http serve error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down HTTP server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
