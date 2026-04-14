package services

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
)

// TestHandleCKMTransition_NonFourCIsNoop asserts that CKM transitions to
// any stage other than "4c" produce no MandatoryMedChecker invocation
// and no error. Phase 6 P6-6.
func TestHandleCKMTransition_NonFourCIsNoop(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, // db
		nil, // gateCache
		nil, // kb19
		nil, // hypoHandler
		nil, // mandatoryMedChecker
		nil, // kb20Client
		nil, // renalDoseGate
		nil, // metrics
		zap.NewNop(),
	)

	cases := []struct {
		name    string
		toStage string
	}{
		{"3a no-op", "3a"},
		{"3b no-op", "3b"},
		{"4a no-op", "4a"},
		{"4b no-op", "4b"},
		{"empty no-op", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(map[string]interface{}{
				"from_stage": "3a",
				"to_stage":   tc.toStage,
			})
			env := priorityEnvelope{
				SignalType: "CKM_STAGE_TRANSITION",
				PatientID:  "p1",
				Payload:    payload,
			}
			if err := handler.handleCKMTransition(context.Background(), env); err != nil {
				t.Errorf("expected nil error for non-4c transition, got %v", err)
			}
		})
	}
}

// TestHandleCKMTransition_FourCWithoutDependencies_IsDefensiveNoop
// asserts that when MandatoryMedChecker or KB20Client are nil (e.g.
// in a misconfigured test harness), a 4c transition logs a warning
// and returns nil rather than panicking.
func TestHandleCKMTransition_FourCWithoutDependencies_IsDefensiveNoop(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, // db
		nil, // gateCache
		nil, // kb19
		nil, // hypoHandler
		nil, // mandatoryMedChecker — intentionally nil
		nil, // kb20Client — intentionally nil
		nil, // renalDoseGate
		nil, // metrics
		zap.NewNop(),
	)

	payload, _ := json.Marshal(map[string]interface{}{
		"from_stage": "4b",
		"to_stage":   "4c",
		"hf_type":    "HFrEF",
	})
	env := priorityEnvelope{
		SignalType: "CKM_STAGE_TRANSITION",
		PatientID:  "p-defensive",
		Payload:    payload,
	}

	if err := handler.handleCKMTransition(context.Background(), env); err != nil {
		t.Errorf("expected nil (defensive no-op), got %v", err)
	}
}

// TestHandleCKMTransition_InvalidPayload_ReturnsError asserts that a
// malformed payload surfaces a clear error rather than silently passing.
func TestHandleCKMTransition_InvalidPayload_ReturnsError(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, zap.NewNop(),
	)
	env := priorityEnvelope{
		SignalType: "CKM_STAGE_TRANSITION",
		PatientID:  "p1",
		Payload:    json.RawMessage(`{"to_stage": [not-a-string]}`),
	}
	if err := handler.handleCKMTransition(context.Background(), env); err == nil {
		t.Error("expected error for invalid payload, got nil")
	}
}

// TestHandleCKMTransition_DispatchedViaHandle asserts that the top-level
// Handle method correctly routes a CKM_STAGE_TRANSITION envelope through
// to handleCKMTransition (covers the dispatch wiring, not just the inner
// handler).
func TestHandleCKMTransition_DispatchedViaHandle(t *testing.T) {
	handler := NewPrioritySignalHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, zap.NewNop(),
	)
	envelopeJSON, _ := json.Marshal(priorityEnvelope{
		SignalType: "CKM_STAGE_TRANSITION",
		PatientID:  "p1",
		Payload:    json.RawMessage(`{"from_stage":"4b","to_stage":"3a"}`),
	})
	if err := handler.Handle(context.Background(), RouteCKMTransition, "p1", envelopeJSON); err != nil {
		t.Errorf("expected nil for non-4c via Handle dispatch, got %v", err)
	}
}
