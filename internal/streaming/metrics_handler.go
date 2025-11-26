package streaming

import (
	"fmt"
	"net/http"
	"strings"
)

// MetricsHandler exposes streaming metrics using a Prometheus-compatible text format.
func MetricsHandler(metrics *Metrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		builder := &strings.Builder{}
		writeMetric(builder, "streaming_active_streams", "gauge", metrics.ActiveStreams())
		writeMetric(builder, "streaming_limit_exceeded_total", "counter", metrics.LimitExceeded())
		writeMetric(builder, "streaming_acquire_timeouts_total", "counter", metrics.AcquireTimeouts())

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(builder.String()))
	})
}

func writeMetric(builder *strings.Builder, name, metricType string, value int64) {
	fmt.Fprintf(builder, "# TYPE %s %s\n", name, metricType)
	fmt.Fprintf(builder, "%s %d\n", name, value)
}
