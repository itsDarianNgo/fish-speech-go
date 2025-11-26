package api

import "sync/atomic"

// Metrics exposes counters and gauges for the API layer.
// The struct can be wrapped by Prometheus collectors when integrating
// with monitoring pipelines.
type Metrics struct {
	activeStreams          atomic.Int64
	limitExceededResponses atomic.Int64
	acquireTimeouts        atomic.Int64
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

// ActiveStreams reports the number of active streams currently being processed.
func (m *Metrics) ActiveStreams() int64 {
	if m == nil {
		return 0
	}
	return m.activeStreams.Load()
}

// IncLimitExceeded increments the counter for limit-exceeded responses.
func (m *Metrics) IncLimitExceeded() {
	if m == nil {
		return
	}
	m.limitExceededResponses.Add(1)
}

// LimitExceededResponses reports how many requests returned limit-exceeded errors.
func (m *Metrics) LimitExceededResponses() int64 {
	if m == nil {
		return 0
	}
	return m.limitExceededResponses.Load()
}

// IncAcquireTimeout increments the counter for acquire timeouts.
func (m *Metrics) IncAcquireTimeout() {
	if m == nil {
		return
	}
	m.acquireTimeouts.Add(1)
}

// AcquireTimeouts reports how many requests failed due to acquire timeout.
func (m *Metrics) AcquireTimeouts() int64 {
	if m == nil {
		return 0
	}
	return m.acquireTimeouts.Load()
}
