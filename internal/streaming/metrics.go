package streaming

import "sync/atomic"

// Metrics exposes counters and gauges for streaming operations.
// The fields are intentionally minimal to keep dependencies light while
// still enabling consumption by Prometheus-style collectors.
type Metrics struct {
	activeStreams   atomic.Int64
	acquireTimeouts atomic.Int64
}

// NewMetrics constructs an empty Metrics collection.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// IncActiveStreams increments the active stream gauge.
func (m *Metrics) IncActiveStreams() {
	if m == nil {
		return
	}
	m.activeStreams.Add(1)
}

// DecActiveStreams decrements the active stream gauge.
func (m *Metrics) DecActiveStreams() {
	if m == nil {
		return
	}
	m.activeStreams.Add(-1)
}

// ActiveStreams reports the number of currently active streams.
func (m *Metrics) ActiveStreams() int64 {
	if m == nil {
		return 0
	}
	return m.activeStreams.Load()
}

// IncAcquireTimeouts increments the acquire timeout counter.
func (m *Metrics) IncAcquireTimeouts() {
	if m == nil {
		return
	}
	m.acquireTimeouts.Add(1)
}

// AcquireTimeouts reports the total number of acquire timeouts.
func (m *Metrics) AcquireTimeouts() int64 {
	if m == nil {
		return 0
	}
	return m.acquireTimeouts.Load()
}
