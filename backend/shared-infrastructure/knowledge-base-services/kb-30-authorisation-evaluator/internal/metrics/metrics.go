// Package metrics owns the kb-30 Prometheus metric registry.
//
// The authorisation evaluator's primary SLO is per-evaluation latency
// (v3 §11 line 624: p95 < 500ms V1, < 200ms V2). The EvaluationLatency
// histogram is the source of truth for that SLO. Labels:
//   - outcome: "allow" | "deny" | "error"
//
// Use ObserveEvaluation at the API boundary to record one observation
// per request.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// EvaluationLatency tracks /v1/authorise request handling latency in
// seconds, broken down by outcome. The bucket boundaries are tuned for
// kb-30's SLO range (sub-millisecond cache hits up to multi-second
// degraded paths).
var EvaluationLatency = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "kb30",
		Subsystem: "authorise",
		Name:      "evaluation_latency_seconds",
		Help:      "Latency of /v1/authorise request handling, by outcome.",
		Buckets: []float64{
			0.001, 0.005, 0.010, 0.025, 0.050,
			0.100, 0.200, 0.500, 1.000, 2.500, 5.000,
		},
	},
	[]string{"outcome"},
)

// EvaluationOutcome enumerates the histogram label values.
const (
	OutcomeAllow = "allow"
	OutcomeDeny  = "deny"
	OutcomeError = "error"
)

// ObserveEvaluation records one latency observation. Call from a deferred
// timer at the start of /v1/authorise handling.
func ObserveEvaluation(outcome string, durationSeconds float64) {
	EvaluationLatency.WithLabelValues(outcome).Observe(durationSeconds)
}
