package server

import (
	"context"
	"time"

	"vote-system/internal/obs"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func MetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err).String()
		obs.ObserveGRPC(info.FullMethod, code, time.Since(start))
		return resp, err
	}
}
