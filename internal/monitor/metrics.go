package monitor

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP请求计数
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP请求持续时间
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// gRPC请求计数
	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	// gRPC请求持续时间
	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Redis操作计数
	redisOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	// 活跃投票数
	activePolls = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_polls_count",
			Help: "Number of active polls",
		},
		[]string{},
	)

	// 总投票数
	totalVotes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "total_votes_count",
			Help: "Total number of votes",
		},
		[]string{},
	)
)

// InitMetrics 初始化指标
func InitMetrics() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		grpcRequestsTotal,
		grpcRequestDuration,
		redisOperationsTotal,
		activePolls,
		totalVotes,
	)
}

// HTTPRequestDuration 记录HTTP请求持续时间
func HTTPRequestDuration(method, endpoint string, start time.Time) {
	httpRequestDuration.WithLabelValues(method, endpoint).Observe(time.Since(start).Seconds())
}

// IncHTTPRequestCount 增加HTTP请求计数
func IncHTTPRequestCount(method, endpoint string, status int) {
	httpRequestsTotal.WithLabelValues(method, endpoint, strconv.Itoa(status)).Inc()
}

// GRPCRequestDuration 记录gRPC请求持续时间
func GRPCRequestDuration(method string, start time.Time) {
	grpcRequestDuration.WithLabelValues(method).Observe(time.Since(start).Seconds())
}

// IncGRPCRequestCount 增加gRPC请求计数
func IncGRPCRequestCount(method string, status string) {
	grpcRequestsTotal.WithLabelValues(method, status).Inc()
}

// IncRedisOperation 增加Redis操作计数
func IncRedisOperation(operation, status string) {
	redisOperationsTotal.WithLabelValues(operation, status).Inc()
}

// SetActivePolls 设置活跃投票数
func SetActivePolls(count int) {
	activePolls.WithLabelValues().Set(float64(count))
}

// IncTotalVotes 增加总投票数
func IncTotalVotes() {
	totalVotes.WithLabelValues().Inc()
}

// GetPrometheusHandler 获取Prometheus HTTP处理器
func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}