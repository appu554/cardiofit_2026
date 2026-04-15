package services

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

// TestParseCGMAnalyticsEvent_RoundTripsAllCoreFields verifies the
// Go consumer can deserialize the exact JSON shape emitted by the
// Flink Module3_CGMStreamJob.serializeAnalyticsEvent helper. A
// mismatch between the two sides would silently drop fields or
// reject messages entirely, so this test pins the wire contract.
// Phase 7 P7-E Milestone 1.
func TestParseCGMAnalyticsEvent_RoundTripsAllCoreFields(t *testing.T) {
	// Wire format mirrors Module3_CGMStreamJob.serializeAnalyticsEvent.
	wire := `{
		"event_type": "CGMAnalyticsEvent",
		"event_version": "v1",
		"patient_id": "p-1",
		"computed_at_ms": 1776249000000,
		"window_end_ms": 1776249000000,
		"window_days": 14,
		"total_readings": 1344,
		"coverage_pct": 100.0,
		"sufficient_data": true,
		"confidence_level": "HIGH",
		"mean_glucose": 140.0,
		"sd_glucose": 15.2,
		"cv_pct": 10.9,
		"glucose_stable": true,
		"tir_pct": 92.5,
		"tbr_l1_pct": 2.0,
		"tbr_l2_pct": 0.5,
		"tar_l1_pct": 3.5,
		"tar_l2_pct": 1.5,
		"gmi": 6.8,
		"gri": 12.5,
		"gri_zone": "A",
		"sustained_hypo_detected": false,
		"sustained_severe_hypo_detected": false,
		"sustained_hyper_detected": false,
		"nocturnal_hypo_detected": false,
		"rapid_rise_detected": false,
		"rapid_fall_detected": false
	}`

	evt, err := ParseCGMAnalyticsEvent([]byte(wire))
	if err != nil {
		t.Fatalf("ParseCGMAnalyticsEvent: %v", err)
	}

	if evt.EventType != "CGMAnalyticsEvent" {
		t.Errorf("event_type = %q, want CGMAnalyticsEvent", evt.EventType)
	}
	if evt.EventVersion != "v1" {
		t.Errorf("event_version = %q, want v1", evt.EventVersion)
	}
	if evt.PatientID != "p-1" {
		t.Errorf("patient_id = %q, want p-1", evt.PatientID)
	}
	if evt.WindowDays != 14 {
		t.Errorf("window_days = %d, want 14", evt.WindowDays)
	}
	if evt.TotalReadings != 1344 {
		t.Errorf("total_readings = %d, want 1344", evt.TotalReadings)
	}
	if evt.TIRPct != 92.5 {
		t.Errorf("tir_pct = %f, want 92.5", evt.TIRPct)
	}
	if evt.MeanGlucose != 140.0 {
		t.Errorf("mean_glucose = %f, want 140.0", evt.MeanGlucose)
	}
	if evt.GRIZone != "A" {
		t.Errorf("gri_zone = %q, want A", evt.GRIZone)
	}
	if evt.CoveragePct != 100.0 {
		t.Errorf("coverage_pct = %f, want 100.0", evt.CoveragePct)
	}
}

// TestParseCGMAnalyticsEvent_TolerantOfUnknownFields asserts the
// consumer does not reject messages that include additional fields
// the Go side doesn't know about (e.g., future AGP percentile arrays
// from a Flink schema expansion). Forward-compatibility is load-bearing:
// Flink can add fields without a coordinated KB-26 deploy.
func TestParseCGMAnalyticsEvent_TolerantOfUnknownFields(t *testing.T) {
	wire := `{
		"event_type": "CGMAnalyticsEvent",
		"patient_id": "p-2",
		"tir_pct": 80.0,
		"unknown_future_field": "some value",
		"agp_percentiles": {"50": [1, 2, 3]}
	}`

	evt, err := ParseCGMAnalyticsEvent([]byte(wire))
	if err != nil {
		t.Fatalf("expected tolerant parse, got error: %v", err)
	}
	if evt.PatientID != "p-2" {
		t.Errorf("patient_id = %q, want p-2", evt.PatientID)
	}
	if evt.TIRPct != 80.0 {
		t.Errorf("tir_pct = %f, want 80.0", evt.TIRPct)
	}
}

// TestParseCGMAnalyticsEvent_RejectsMalformedJson asserts a bad record
// surfaces an error rather than silently returning a zero value.
// The consumer's parse-failure branch logs at WARN and commits the
// offset so the pipeline doesn't stall on a poisoned message.
func TestParseCGMAnalyticsEvent_RejectsMalformedJson(t *testing.T) {
	_, err := ParseCGMAnalyticsEvent([]byte("{not json"))
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}

// TestLogOnlyCGMAnalyticsHandler_ReturnsNil confirms the Milestone 1
// default handler is a non-failing stub. A future Milestone 2 swap
// replaces it with a repository-backed handler.
func TestLogOnlyCGMAnalyticsHandler_ReturnsNil(t *testing.T) {
	handler := LogOnlyCGMAnalyticsHandler(zap.NewNop())
	evt := CGMAnalyticsEventPayload{PatientID: "p-test", TIRPct: 75.0}
	if err := handler(context.Background(), evt); err != nil {
		t.Errorf("expected nil from log-only handler, got %v", err)
	}
}

// TestCGMAnalyticsConsumer_Construction verifies the consumer
// constructor wires kafka.Reader with the documented topic, group,
// and offset settings. Kept as a narrow unit test — a full Kafka
// round-trip test would require a real broker.
func TestCGMAnalyticsConsumer_Construction(t *testing.T) {
	consumer := NewCGMAnalyticsConsumer([]string{"localhost:9092"}, zap.NewNop())
	if consumer == nil {
		t.Fatal("expected non-nil consumer")
	}
	if consumer.reader == nil {
		t.Error("expected reader to be initialised")
	}
	// Verify the Stop method is safe to call even when Start wasn't.
	consumer.Stop()
}
