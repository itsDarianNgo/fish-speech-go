package streaming

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandlerExportsPrometheusText(t *testing.T) {
	metrics := NewMetrics()
	metrics.IncActiveStreams()
	metrics.IncLimitExceeded()
	metrics.IncAcquireTimeouts()
	metrics.IncAcquireTimeouts()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/metrics", nil)

	MetricsHandler(metrics).ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "streaming_active_streams 1") {
		t.Fatalf("expected active streams metric, got: %s", body)
	}
	if !strings.Contains(body, "streaming_limit_exceeded_total 1") {
		t.Fatalf("expected limit exceeded metric, got: %s", body)
	}
	if !strings.Contains(body, "streaming_acquire_timeouts_total 2") {
		t.Fatalf("expected acquire timeout metric, got: %s", body)
	}
}
