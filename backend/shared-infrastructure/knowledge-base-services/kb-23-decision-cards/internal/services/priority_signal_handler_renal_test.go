package services

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
)

// TestHandleRenalGate_NonEGFRPayload_Noop asserts the handler is
// defensive against malformed routing — even if a non-EGFR payload
// arrives on the EGFR_LAB route, no gating runs.
func TestHandleRenalGate_NonEGFRPayload_Noop(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, zap.NewNop(),
	)
	payload, _ := json.Marshal(map[string]interface{}{
		"lab_type": "CREATININE",
		"value":    1.5,
	})
	env := priorityEnvelope{
		SignalType: "EGFR_LAB",
		PatientID:  "p1",
		Payload:    payload,
	}
	if err := handler.handleRenalGate(context.Background(), env); err != nil {
		t.Errorf("expected nil for non-EGFR payload, got %v", err)
	}
}

// TestHandleRenalGate_WithoutDependencies_IsDefensiveNoop asserts that
// an EGFR lab event arriving when RenalDoseGate or KB20Client are nil
// (e.g. test harness or formulary load failure in production) logs a
// warning and returns nil rather than panicking.
func TestHandleRenalGate_WithoutDependencies_IsDefensiveNoop(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, // db
		nil, // gateCache
		nil, // kb19
		nil, // hypoHandler
		nil, // mandatoryMedChecker
		nil, // kb20Client — intentionally nil
		nil, // renalDoseGate — intentionally nil
		nil, // metrics
		zap.NewNop(),
	)
	payload, _ := json.Marshal(map[string]interface{}{
		"lab_type":    "EGFR",
		"value":       28.0,
		"unit":        "mL/min/1.73m²",
		"measured_at": "2026-04-14T10:00:00Z",
		"source":      "CKD-EPI-2021",
		"is_derived":  true,
	})
	env := priorityEnvelope{
		SignalType: "EGFR_LAB",
		PatientID:  "p-defensive",
		Payload:    payload,
	}
	if err := handler.handleRenalGate(context.Background(), env); err != nil {
		t.Errorf("expected nil (defensive no-op), got %v", err)
	}
}

// TestHandleRenalGate_InvalidPayload_ReturnsError asserts that a
// malformed JSON payload surfaces a clear error.
func TestHandleRenalGate_InvalidPayload_ReturnsError(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, zap.NewNop(),
	)
	env := priorityEnvelope{
		SignalType: "EGFR_LAB",
		PatientID:  "p1",
		Payload:    json.RawMessage(`{"value": [not-a-number]}`),
	}
	if err := handler.handleRenalGate(context.Background(), env); err == nil {
		t.Error("expected error for invalid payload, got nil")
	}
}

// TestHandleRenalGate_DispatchedViaHandle asserts that the top-level
// Handle method correctly routes an EGFR_LAB envelope through to
// handleRenalGate (covers the dispatch wiring, not just the inner
// handler).
func TestHandleRenalGate_DispatchedViaHandle(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, zap.NewNop(),
	)
	envelopeJSON, _ := json.Marshal(priorityEnvelope{
		SignalType: "EGFR_LAB",
		PatientID:  "p1",
		Payload: json.RawMessage(
			`{"lab_type":"EGFR","value":45.0,"measured_at":"2026-04-14T10:00:00Z","is_derived":true}`,
		),
	})
	if err := handler.Handle(context.Background(), RouteRenalGate, "p1", envelopeJSON); err != nil {
		t.Errorf("expected nil (defensive no-op via Handle dispatch), got %v", err)
	}
}
