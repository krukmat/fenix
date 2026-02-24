// Task 4.9 — NFR-030: Metrics tests
package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler_PrometheusFormat(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	MetricsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "fenixcrm_requests_total") {
		t.Errorf("body missing fenixcrm_requests_total: %s", body)
	}
	if !strings.Contains(body, "fenixcrm_uptime_seconds") {
		t.Errorf("body missing fenixcrm_uptime_seconds: %s", body)
	}
	if !strings.Contains(body, "# TYPE fenixcrm_requests_total counter") {
		t.Errorf("body missing TYPE declaration: %s", body)
	}
}

func TestMetricsHandler_ContentType(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	MetricsHandler(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "text/plain; version=0.0.4" {
		t.Errorf("Content-Type = %s; want text/plain; version=0.0.4", ct)
	}
}

func TestMetricsCollector_IncRequests(t *testing.T) {
	t.Parallel()
	initial := Metrics.RequestsTotal()
	Metrics.IncRequests()
	after := Metrics.RequestsTotal()
	if after != initial+1 {
		t.Errorf("RequestsTotal after IncRequests = %d; want %d", after, initial+1)
	}
}

func TestMetricsCollector_IncErrors(t *testing.T) {
	t.Parallel()
	initial := Metrics.RequestErrors()
	Metrics.IncErrors()
	after := Metrics.RequestErrors()
	if after != initial+1 {
		t.Errorf("RequestErrors after IncErrors = %d; want %d", after, initial+1)
	}
}
