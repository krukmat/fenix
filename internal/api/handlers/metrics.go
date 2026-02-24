package handlers

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// MetricsCollector holds atomic counters — safe for concurrent use.
// Task 4.9 — NFR-030: Prometheus-compatible metrics endpoint
type MetricsCollector struct {
	requestsTotal atomic.Int64
	requestErrors atomic.Int64
	startTime     time.Time
}

// Metrics is the global metrics collector instance
var Metrics = &MetricsCollector{startTime: time.Now()}

// IncRequests increments the total requests counter
func (m *MetricsCollector) IncRequests() { m.requestsTotal.Add(1) }

// IncErrors increments the total errors counter
func (m *MetricsCollector) IncErrors() { m.requestErrors.Add(1) }

// RequestsTotal returns the current request count
func (m *MetricsCollector) RequestsTotal() int64 { return m.requestsTotal.Load() }

// RequestErrors returns the current error count
func (m *MetricsCollector) RequestErrors() int64 { return m.requestErrors.Load() }

// UptimeSeconds returns the process uptime in seconds
func (m *MetricsCollector) UptimeSeconds() float64 {
	return time.Since(m.startTime).Seconds()
}

// mimePrometheus is the Prometheus text exposition format content type.
// Task 4.9 — NFR-030
const mimePrometheus = "text/plain; version=0.0.4"

// MetricsHandler returns Prometheus text format — Task 4.9 NFR-030
func MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(headerContentType, mimePrometheus)
	fmt.Fprintf(w, "# HELP fenixcrm_requests_total Total HTTP requests\n")
	fmt.Fprintf(w, "# TYPE fenixcrm_requests_total counter\n")
	fmt.Fprintf(w, "fenixcrm_requests_total %d\n", Metrics.RequestsTotal())
	fmt.Fprintf(w, "# HELP fenixcrm_request_errors_total Total HTTP errors (5xx)\n")
	fmt.Fprintf(w, "# TYPE fenixcrm_request_errors_total counter\n")
	fmt.Fprintf(w, "fenixcrm_request_errors_total %d\n", Metrics.RequestErrors())
	fmt.Fprintf(w, "# HELP fenixcrm_uptime_seconds Process uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE fenixcrm_uptime_seconds gauge\n")
	fmt.Fprintf(w, "fenixcrm_uptime_seconds %.2f\n", Metrics.UptimeSeconds())
}
