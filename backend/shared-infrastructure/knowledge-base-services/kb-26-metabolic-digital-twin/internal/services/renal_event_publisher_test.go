package services

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// mockPublisher — records published messages for test assertions
// ---------------------------------------------------------------------------

type publishedMessage struct {
	Topic   string
	Key     string
	Payload []byte
}

type mockPublisher struct {
	messages []publishedMessage
}

func (m *mockPublisher) Publish(topic string, key string, payload []byte) error {
	m.messages = append(m.messages, publishedMessage{
		Topic:   topic,
		Key:     key,
		Payload: payload,
	})
	return nil
}

// ---------------------------------------------------------------------------
// TestRapidDecline_PublishesEvent
// ---------------------------------------------------------------------------

func TestRapidDecline_PublishesEvent(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewRenalEventPublisher(mock, "renal-events")

	trajectory := &EGFRTrajectoryResult{
		Slope:           -8.0,
		Classification:  "RAPID_DECLINE",
		IsRapidDecliner: true,
		DataPoints:      5,
		SpanDays:        365,
		LatestEGFR:      40,
		RSquared:        0.92,
	}

	err := pub.EvaluateAndPublish("patient-001", trajectory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.messages) < 1 {
		t.Fatal("expected at least 1 event to be published, got 0")
	}

	// Verify the rapid decline event is present.
	foundRapidDecline := false
	for _, msg := range mock.messages {
		var evt RenalEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}
		if evt.EventType == "RENAL_RAPID_DECLINE" {
			foundRapidDecline = true
			if evt.Severity != "CRITICAL" {
				t.Errorf("expected severity CRITICAL, got %s", evt.Severity)
			}
			if evt.PatientID != "patient-001" {
				t.Errorf("expected patient_id patient-001, got %s", evt.PatientID)
			}
			if evt.EGFR != 40 {
				t.Errorf("expected eGFR 40, got %.1f", evt.EGFR)
			}
			if evt.Slope != -8.0 {
				t.Errorf("expected slope -8.0, got %.1f", evt.Slope)
			}
		}
	}

	if !foundRapidDecline {
		t.Error("RENAL_RAPID_DECLINE event not found among published events")
	}
}

// ---------------------------------------------------------------------------
// TestStableTrajectory_NoEvents
// ---------------------------------------------------------------------------

func TestStableTrajectory_NoEvents(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewRenalEventPublisher(mock, "renal-events")

	trajectory := &EGFRTrajectoryResult{
		Slope:           -0.5,
		Classification:  "STABLE",
		IsRapidDecliner: false,
		DataPoints:      4,
		SpanDays:        300,
		LatestEGFR:      65,
		RSquared:        0.60,
	}

	err := pub.EvaluateAndPublish("patient-002", trajectory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.messages) != 0 {
		t.Errorf("expected 0 events for stable trajectory, got %d", len(mock.messages))
		for i, msg := range mock.messages {
			var evt RenalEvent
			json.Unmarshal(msg.Payload, &evt)
			t.Logf("  event[%d]: type=%s severity=%s details=%s", i, evt.EventType, evt.Severity, evt.Details)
		}
	}
}

// ---------------------------------------------------------------------------
// TestThresholdApproaching_PublishesWarning
// ---------------------------------------------------------------------------

func TestThresholdApproaching_PublishesWarning(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewRenalEventPublisher(mock, "renal-events")

	// eGFR=35 with slope -3/year: time to 30 = 5/3 = ~20 months (>12, skip),
	// but time to 25 = 10/3 = ~40 months (>12, skip), time to 20 = 15/3 = ~60 months.
	// Use eGFR=32 with slope -3/year: time to 30 = 2/3 = 8 months → WARNING.
	trajectory := &EGFRTrajectoryResult{
		Slope:           -3.0,
		Classification:  "MODERATE_DECLINE",
		IsRapidDecliner: false,
		DataPoints:      4,
		SpanDays:        365,
		LatestEGFR:      32,
		RSquared:        0.85,
	}

	err := pub.EvaluateAndPublish("patient-003", trajectory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// eGFR=32, slope=-3: METFORMIN(30): 2/3*12=8mo→WARNING, SULFONYLUREA(30): same,
	// MRA(30): same, FINERENONE(25): 7/3*12=28mo→skip, SGLT2i(20): 12/3*12=48mo→skip
	if len(mock.messages) != 3 {
		t.Errorf("expected 3 threshold-approaching events, got %d", len(mock.messages))
	}

	for _, msg := range mock.messages {
		var evt RenalEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if evt.EventType != "RENAL_THRESHOLD_APPROACHING" {
			t.Errorf("expected RENAL_THRESHOLD_APPROACHING, got %s", evt.EventType)
		}
		if evt.Severity != "WARNING" {
			t.Errorf("expected WARNING severity, got %s", evt.Severity)
		}
	}
}

// ---------------------------------------------------------------------------
// TestThresholdApproaching_CriticalWhenUnder3Months
// ---------------------------------------------------------------------------

func TestThresholdApproaching_CriticalWhenUnder3Months(t *testing.T) {
	mock := &mockPublisher{}
	pub := NewRenalEventPublisher(mock, "renal-events")

	// eGFR=31 with slope -10/year: time to 30 = 1/10*12 = 1.2 months → CRITICAL
	trajectory := &EGFRTrajectoryResult{
		Slope:           -10.0,
		Classification:  "RAPID_DECLINE",
		IsRapidDecliner: true,
		DataPoints:      5,
		SpanDays:        365,
		LatestEGFR:      31,
		RSquared:        0.95,
	}

	err := pub.EvaluateAndPublish("patient-004", trajectory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at least the rapid decline event plus threshold events.
	foundCriticalThreshold := false
	for _, msg := range mock.messages {
		var evt RenalEvent
		json.Unmarshal(msg.Payload, &evt)
		if evt.EventType == "RENAL_THRESHOLD_APPROACHING" && evt.Severity == "CRITICAL" {
			foundCriticalThreshold = true
		}
	}

	if !foundCriticalThreshold {
		t.Error("expected at least one CRITICAL RENAL_THRESHOLD_APPROACHING event")
	}
}
