package obs

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SSE连接指标
	SSEActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sse_active_connections",
		Help: "当前活跃的SSE连接数",
	})

	SSEUserConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sse_user_connections",
		Help: "每个用户的SSE连接数",
	})

	SSEConnectionsPerUser = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sse_connections_per_user",
		Help: "每个用户的连接数分布",
	}, []string{"user_id"})

	// SSE消息指标
	SSEMessagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sse_messages_sent_total",
		Help: "发送的SSE消息总数",
	})

	SSEMessagesDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sse_messages_dropped_total",
		Help: "丢弃的SSE消息总数（缓冲区满）",
	})

	// SSE错误指标
	SSEErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sse_errors_total",
		Help: "SSE错误总数",
	}, []string{"type", "user_id"})

	// SSE性能指标
	SSEMessageLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "sse_message_latency_seconds",
		Help:    "SSE消息延迟时间（秒）",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
	})

	SSEBufferUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sse_buffer_usage_bytes",
		Help: "SSE缓冲区使用字节数",
	})

	// Poll相关指标
	PollInvalidates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "poll_invalidates_total",
		Help: "Poll失效消息总数",
	})

	PollInvalidatesPerPoll = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "poll_invalidates_per_poll_total",
		Help: "每个Poll的失效消息总数",
	}, []string{"poll_id"})
)

// SSE操作跟踪
func TrackSSEOp(userID string, operation string, success bool, duration float64) {
	if success {
		SSEMessagesSent.Inc()
		SSEMessageLatency.Observe(duration)
	} else {
		SSEErrors.WithLabelValues(operation, userID).Inc()
	}
}

// 记录连接状态变化
func RecordConnectionChange(userID string, delta int) {
	current := SSEConnectionsPerUser.WithLabelValues(userID)
	current.Add(float64(delta))

	// 更新总连接数
	SSEActiveConnections.Add(float64(delta))
}

// 记录缓冲区使用
func RecordBufferUsage(userID string, bufferSize int, used int) {
	SSEBufferUsage.Set(float64(used))
}

// 记录消息丢弃
func RecordMessageDropped(userID string) {
	SSEMessagesDropped.Inc()
	SSEErrors.WithLabelValues("drop", userID).Inc()
}

// 记录Poll失效
func RecordPollInvalidate(pollID string) {
	PollInvalidates.Inc()
	PollInvalidatesPerPoll.WithLabelValues(pollID).Inc()
}