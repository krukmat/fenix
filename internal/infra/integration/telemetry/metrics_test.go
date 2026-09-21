package telemetry

import (
	"strings"
	"testing"
	"time"
)

func TestRegistryPrometheusIncludesProviderLifecycleAndPendingMetrics(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.ObserveProvider("vel", "verify", "success", 250*time.Millisecond)
	registry.IncLifecycle("verification", "verified")
	registry.SetPending(3)

	body := registry.Prometheus()
	for _, expected := range []string{
		`fenixcrm_integration_provider_calls_total{provider="vel",operation="verify",outcome="success"} 1`,
		`fenixcrm_integration_lifecycle_total{stage="verification",outcome="verified"} 1`,
		"fenixcrm_evidence_pending 3",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics missing %q:\n%s", expected, body)
		}
	}
}
