// holddata_responder.go implements the HOLD_DATA response flow (Phase 5.3 / SA-05).
//
// When Channel B returns HOLD_DATA:
//  1. V-MCU logs anomalous value to SafetyTrace (done in RunCycle)
//  2. V-MCU sends KB-20 re-validation: POST /patient/:id/labs/:lab_id/flag-anomaly
//  3. V-MCU notifies KB-19: DATA_ANOMALY_DETECTED event
//  4. This titration cycle is deferred (no dose change — arbiter blocks)
//  5. Next cycle: if re-validated → proceed; if confirmed error → KB-20 marks ANOMALOUS
package events

import "context"

// AnomalyFlagger flags lab values as anomalous in KB-20.
// Implements: POST /patient/:id/labs/:lab_id/flag-anomaly
type AnomalyFlagger interface {
	// FlagAnomaly sends an anomaly flag to KB-20 for the given lab entry.
	// Sets ValidationStatus = 'FLAGGED' with FlagReason = 'ANOMALY_FLAGGED_BY_VMCU'.
	// Triggers clinical review notification via KB-19.
	FlagAnomaly(ctx context.Context, patientID string, labID string, rule string) error
}

// HoldDataResponder orchestrates the HOLD_DATA response flow.
type HoldDataResponder struct {
	flagger   AnomalyFlagger
	publisher EventPublisher
}

// NewHoldDataResponder creates a responder wired to KB-20 and KB-19.
func NewHoldDataResponder(flagger AnomalyFlagger, publisher EventPublisher) *HoldDataResponder {
	return &HoldDataResponder{
		flagger:   flagger,
		publisher: publisher,
	}
}

// Respond executes the full HOLD_DATA response flow.
// Called by the runtime layer when a cycle result has IsAnomaly=true.
func (r *HoldDataResponder) Respond(ctx context.Context, patientID, labID, ruleFired, anomalyLab string) error {
	// Step 1: Flag the anomalous lab in KB-20
	if r.flagger != nil && labID != "" {
		if err := r.flagger.FlagAnomaly(ctx, patientID, labID, ruleFired); err != nil {
			return err
		}
	}

	// Step 2: Publish DATA_ANOMALY_DETECTED to KB-19
	if r.publisher != nil {
		event := Event{
			Type:      EventDataAnomalyDetected,
			PatientID: patientID,
			Source:    "V-MCU",
			Payload: map[string]interface{}{
				"rule_fired":  ruleFired,
				"anomaly_lab": anomalyLab,
			},
		}
		if err := r.publisher.Publish(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
