package validation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateEvent reports any structural problem with e.
//
// Universal rules (apply to every event_type):
//   - ResidentID must be non-zero
//   - EventType must be one of the EventType* enum values
//   - OccurredAt must be non-zero
//   - ReportedByRef must be non-zero
//   - Severity, when set, must be one of EventSeverity*
//   - Each TriggeredStateChange.StateMachine, when present, must be one of
//     EventStateMachine*
//
// Per-event-type required-field rules (intentionally pragmatic — only the
// event types where MVP regulatory or causality semantics demand it get
// bespoke rules; everything else passes once the universal rules hold):
//
//   - fall                 — require Severity; require WitnessedByRefs
//                            non-empty OR DescriptionStructured non-empty
//   - medication_error     — require Severity; require >= 1
//                            RelatedMedicationUses ref
//   - adverse_drug_event   — require >= 1 RelatedMedicationUses ref
//   - death                — universal only (OccurredAt is mandatory anyway;
//                            no further fields required at the model layer)
//   - hospital_admission   — require Severity AND DescriptionStructured non-empty
//   - hospital_discharge   — require Severity AND DescriptionStructured non-empty
//
// All other event types (pressure_injury, behavioural_incident, GP_visit,
// admission_to_facility, rule_fire, etc.) require only the universal fields.
// Future tightening should land here, not in callers.
func ValidateEvent(e models.Event) error {
	if e.ResidentID == uuid.Nil {
		return errors.New("resident_id is required")
	}
	if !models.IsValidEventType(e.EventType) {
		return fmt.Errorf("invalid event_type %q", e.EventType)
	}
	if e.OccurredAt.IsZero() {
		return errors.New("occurred_at is required")
	}
	if e.ReportedByRef == uuid.Nil {
		return errors.New("reported_by_ref is required")
	}
	if e.Severity != "" && !models.IsValidEventSeverity(e.Severity) {
		return fmt.Errorf("invalid severity %q", e.Severity)
	}
	for i, tsc := range e.TriggeredStateChanges {
		if !models.IsValidEventStateMachine(tsc.StateMachine) {
			return fmt.Errorf("invalid triggered_state_changes[%d].state_machine %q", i, tsc.StateMachine)
		}
	}

	// Per-event-type required-field rules.
	switch e.EventType {
	case models.EventTypeFall:
		if e.Severity == "" {
			return errors.New("fall: severity is required")
		}
		if len(e.WitnessedByRefs) == 0 && len(e.DescriptionStructured) == 0 {
			return errors.New("fall: at least one witnessed_by_refs OR description_structured is required")
		}
	case models.EventTypeMedicationError:
		if e.Severity == "" {
			return errors.New("medication_error: severity is required")
		}
		if len(e.RelatedMedicationUses) == 0 {
			return errors.New("medication_error: at least one related_medication_uses ref is required")
		}
	case models.EventTypeAdverseDrugEvent:
		if len(e.RelatedMedicationUses) == 0 {
			return errors.New("adverse_drug_event: at least one related_medication_uses ref is required")
		}
	case models.EventTypeHospitalAdmission, models.EventTypeHospitalDischarge:
		if e.Severity == "" {
			return fmt.Errorf("%s: severity is required", e.EventType)
		}
		if len(e.DescriptionStructured) == 0 {
			return fmt.Errorf("%s: description_structured is required", e.EventType)
		}
	}

	return nil
}
