// Package metrics defines Prometheus metric interfaces for V-MCU (Phase 7.4).
//
// The engine records metrics via the Recorder interface.
// The runtime layer provides a concrete implementation using
// github.com/prometheus/client_golang (not imported here to avoid
// adding the dependency to the engine package).
package metrics

import "time"

// Recorder is the metrics interface for V-MCU instrumentation.
// The runtime layer provides a Prometheus-backed implementation.
type Recorder interface {
	// RecordCycleCompleted increments vmcu_titration_cycles_total{final_gate}.
	RecordCycleCompleted(finalGate string)

	// RecordChannelLatency records per-channel evaluation latency.
	// vmcu_channel_{a,b,c}_latency_ms histogram.
	RecordChannelLatency(channel string, duration time.Duration)

	// RecordArbiterLatency records arbiter evaluation latency.
	// vmcu_arbiter_latency_ms histogram.
	RecordArbiterLatency(duration time.Duration)

	// RecordTraceWriteLatency records SafetyTrace write latency.
	// vmcu_safety_trace_write_latency_ms histogram.
	RecordTraceWriteLatency(duration time.Duration)

	// RecordGateBlocked increments vmcu_gate_blocked_total{channel,gate}.
	RecordGateBlocked(channel, gate string)

	// RecordHoldDataTriggered increments vmcu_hold_data_triggered_total{rule}.
	RecordHoldDataTriggered(rule string)

	// RecordCacheRefresh increments vmcu_cache_refresh_total{source}.
	RecordCacheRefresh(source string)

	// SetCacheAge sets vmcu_cache_age_seconds{source} gauge.
	SetCacheAge(source string, age time.Duration)
}

// NoopRecorder is a no-op implementation for testing and when metrics are disabled.
type NoopRecorder struct{}

func (NoopRecorder) RecordCycleCompleted(string)                   {}
func (NoopRecorder) RecordChannelLatency(string, time.Duration)    {}
func (NoopRecorder) RecordArbiterLatency(time.Duration)            {}
func (NoopRecorder) RecordTraceWriteLatency(time.Duration)         {}
func (NoopRecorder) RecordGateBlocked(string, string)              {}
func (NoopRecorder) RecordHoldDataTriggered(string)                {}
func (NoopRecorder) RecordCacheRefresh(string)                     {}
func (NoopRecorder) SetCacheAge(string, time.Duration)             {}
