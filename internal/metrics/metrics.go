package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vote_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vote_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vote_grpc_requests_total",
			Help: "Total number of gRPC requests.",
		},
		[]string{"method", "code"},
	)

	GRPCRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vote_grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "code"},
	)

	GRPCErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vote_grpc_errors_total",
			Help: "Total number of failed gRPC requests.",
		},
		[]string{"method", "code"},
	)
)

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		statusCode := strconv.Itoa(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, statusCode).Inc()
		HTTPRequestDurationSeconds.WithLabelValues(c.Request.Method, path, statusCode).Observe(time.Since(start).Seconds())
	}
}

func UnaryServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)

	code := status.Code(err).String()
	method := info.FullMethod
	GRPCRequestsTotal.WithLabelValues(method, code).Inc()
	GRPCRequestDurationSeconds.WithLabelValues(method, code).Observe(time.Since(start).Seconds())
	if err != nil {
		GRPCErrorsTotal.WithLabelValues(method, code).Inc()
	}

	return resp, err
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
