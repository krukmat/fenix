// Package telemetry provides dependency-free metrics for cross-platform integrations.
package telemetry

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type providerKey struct {
	provider  string
	operation string
	outcome   string
}

type lifecycleKey struct {
	stage   string
	outcome string
}

type providerValue struct {
	calls   int64
	latency time.Duration
}

// Registry stores integration counters independently from provider implementations.
type Registry struct {
	mu        sync.Mutex
	providers map[providerKey]providerValue
	lifecycle map[lifecycleKey]int64
	pending   atomic.Int64
}

// Default is the process-wide integration telemetry registry exposed by /metrics.
var Default = NewRegistry()

// NewRegistry creates an empty telemetry registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[providerKey]providerValue),
		lifecycle: make(map[lifecycleKey]int64),
	}
}

// ObserveProvider records one provider operation and its cumulative latency.
func (r *Registry) ObserveProvider(provider, operation, outcome string, elapsed time.Duration) {
	if r == nil {
		return
	}
	key := providerKey{provider: provider, operation: operation, outcome: outcome}
	r.mu.Lock()
	value := r.providers[key]
	value.calls++
	value.latency += elapsed
	r.providers[key] = value
	r.mu.Unlock()
}

// IncLifecycle records one durable evidence lifecycle transition attempt/outcome.
func (r *Registry) IncLifecycle(stage, outcome string) {
	if r == nil {
		return
	}
	key := lifecycleKey{stage: stage, outcome: outcome}
	r.mu.Lock()
	r.lifecycle[key]++
	r.mu.Unlock()
}

// SetPending sets the current number of non-terminal durable evidence deliveries.
func (r *Registry) SetPending(value int64) {
	if r != nil {
		r.pending.Store(value)
	}
}

// Prometheus renders integration metrics in the existing dependency-free text format.
func (r *Registry) Prometheus() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	providers := cloneProviders(r.providers)
	lifecycle := cloneLifecycle(r.lifecycle)
	r.mu.Unlock()

	var output strings.Builder
	renderProviderMetrics(&output, providers)
	renderLifecycleMetrics(&output, lifecycle)
	fmt.Fprintf(&output, "# HELP fenixcrm_evidence_pending Durable evidence deliveries awaiting terminal verification\n")
	fmt.Fprintf(&output, "# TYPE fenixcrm_evidence_pending gauge\n")
	fmt.Fprintf(&output, "fenixcrm_evidence_pending %d\n", r.pending.Load())
	return output.String()
}

func cloneProviders(source map[providerKey]providerValue) map[providerKey]providerValue {
	cloned := make(map[providerKey]providerValue, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneLifecycle(source map[lifecycleKey]int64) map[lifecycleKey]int64 {
	cloned := make(map[lifecycleKey]int64, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func renderProviderMetrics(output *strings.Builder, values map[providerKey]providerValue) {
	fmt.Fprintf(output, "# HELP fenixcrm_integration_provider_calls_total Cross-platform provider calls\n")
	fmt.Fprintf(output, "# TYPE fenixcrm_integration_provider_calls_total counter\n")
	fmt.Fprintf(output, "# HELP fenixcrm_integration_provider_latency_seconds_total Cumulative provider call latency\n")
	fmt.Fprintf(output, "# TYPE fenixcrm_integration_provider_latency_seconds_total counter\n")
	keys := sortedProviderKeys(values)
	for _, key := range keys {
		labels := providerLabels(key)
		value := values[key]
		fmt.Fprintf(output, "fenixcrm_integration_provider_calls_total{%s} %d\n", labels, value.calls)
		fmt.Fprintf(
			output,
			"fenixcrm_integration_provider_latency_seconds_total{%s} %.6f\n",
			labels,
			value.latency.Seconds(),
		)
	}
}

func renderLifecycleMetrics(output *strings.Builder, values map[lifecycleKey]int64) {
	fmt.Fprintf(output, "# HELP fenixcrm_integration_lifecycle_total Durable evidence lifecycle outcomes\n")
	fmt.Fprintf(output, "# TYPE fenixcrm_integration_lifecycle_total counter\n")
	keys := sortedLifecycleKeys(values)
	for _, key := range keys {
		fmt.Fprintf(
			output,
			"fenixcrm_integration_lifecycle_total{stage=%q,outcome=%q} %d\n",
			key.stage,
			key.outcome,
			values[key],
		)
	}
}

func sortedProviderKeys(values map[providerKey]providerValue) []providerKey {
	keys := make([]providerKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return providerLabels(keys[i]) < providerLabels(keys[j])
	})
	return keys
}

func sortedLifecycleKeys(values map[lifecycleKey]int64) []lifecycleKey {
	keys := make([]lifecycleKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := keys[i].stage + "\x00" + keys[i].outcome
		right := keys[j].stage + "\x00" + keys[j].outcome
		return left < right
	})
	return keys
}

func providerLabels(key providerKey) string {
	return fmt.Sprintf("provider=%q,operation=%q,outcome=%q", key.provider, key.operation, key.outcome)
}
