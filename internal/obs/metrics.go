package obs

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "voting",
			Subsystem: "httpserver",
			Name:      "requests_total",
			Help:      "Total HTTP requests handled by httpserver.",
		},
		[]string{"method", "route", "code"},
	)

	HTTPServerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "voting",
			Subsystem: "httpserver",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration (seconds).",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	GRPCRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "voting",
			Subsystem: "grpcserver",
			Name:      "requests_total",
			Help:      "Total gRPC requests handled by grpcserver.",
		},
		[]string{"method", "code"},
	)

	GRPCServerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "voting",
			Subsystem: "grpcserver",
			Name:      "request_duration_seconds",
			Help:      "gRPC request duration (seconds).",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	VoteOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "voting",
			Subsystem: "business",
			Name:      "vote_ops_total",
			Help:      "Business vote operations total (vote/undo/create/close/delete outcomes).",
		},
		[]string{"op", "result"},
	)
)

func RegisterAll() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPServerDuration,
		GRPCRequestsTotal,
		GRPCServerDuration,
		VoteOpsTotal,
	)
}

func ObserveHTTP(method, route, code string, dur time.Duration) {
	if route == "" {
		route = "unknown"
	}
	HTTPRequestsTotal.WithLabelValues(method, route, code).Inc()
	HTTPServerDuration.WithLabelValues(method, route).Observe(dur.Seconds())
}

func ObserveGRPC(method, code string, dur time.Duration) {
	GRPCRequestsTotal.WithLabelValues(method, code).Inc()
	GRPCServerDuration.WithLabelValues(method).Observe(dur.Seconds())
}

func VoteOp(op, result string) {
	VoteOpsTotal.WithLabelValues(op, result).Inc()
}
