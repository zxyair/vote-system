package obs

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusHandler 返回Prometheus HTTP处理器
func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}