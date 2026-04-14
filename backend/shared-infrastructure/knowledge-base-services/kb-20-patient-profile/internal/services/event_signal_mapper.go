package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/signals"
	"kb-patient-profile/internal/models"
)

// SignalType is an alias to the shared signals package type.
type SignalType = signals.SignalType

// MappedEvent is the result of mapping a KB-20 outbox entry to either
// a ClinicalSignalEnvelope or a ClinicalStateChangeEnvelope.
type MappedEvent struct {
	Signal      *MappedSignal
	StateChange *MappedStateChange
}

// MappedSignal holds a signal envelope ready for Kafka publishing.
type MappedSignal struct {
	EventID    uuid.UUID
	PatientID  string
	SignalType SignalType
	Priority   bool
	Timestamp  time.Time
	LOINCCode  string
	Payload    json.RawMessage
}

// MappedStateChange holds a state change envelope ready for Kafka publishing.
type MappedStateChange struct {
	EventID    uuid.UUID
	PatientID  string
	ChangeType string
	Timestamp  time.Time
	Payload    json.RawMessage
}

// EventSignalMapper maps KB-20 outbox EventType + Payload → MappedEvent.
type EventSignalMapper struct {
	labTypeMap map[string]labMapping
}

type labMapping struct {
	signalType SignalType
	loincCode  string
	priority   func(float64) bool
}

// NewEventSignalMapper creates a mapper with the full lab/alert/state-change mapping table.
func NewEventSignalMapper() *EventSignalMapper {
	return &EventSignalMapper{
		labTypeMap: map[string]labMapping{
			"FBG": {
				signalType: signals.SignalFBG,
				loincCode:  "1558-6",
				priority:   func(v float64) bool { return v < 4.0 || v > 20.0 },
			},
			"PPBG": {
				signalType: signals.SignalPPBG,
				loincCode:  "87422-2",
				priority:   func(v float64) bool { return v < 4.0 },
			},
			"HBA1C": {
				signalType: signals.SignalHbA1c,
				loincCode:  "4548-4",
			},
			"SBP": {
				signalType: signals.SignalSBP,
				loincCode:  "8480-6",
				priority:   func(v float64) bool { return v > 180 || v < 90 },
			},
			"DBP": {
				signalType: signals.SignalDBP,
				loincCode:  "8462-4",
			},
			"HEART_RATE": {
				signalType: signals.SignalHR,
				loincCode:  "8867-4",
				priority:   func(v float64) bool { return v < 40 || v > 150 },
			},
			"CREATININE": {
				signalType: signals.SignalCreatinine,
				loincCode:  "2160-0",
				priority:   func(v float64) bool { return v > 10.0 },
			},
			"ACR": {
				signalType: signals.SignalACR,
				loincCode:  "9318-7",
				priority:   func(v float64) bool { return v > 300 },
			},
			"POTASSIUM": {
				signalType: signals.SignalPotassium,
				loincCode:  "6298-4",
				priority:   func(v float64) bool { return v > 5.5 || v < 3.0 },
			},
			"WEIGHT": {
				signalType: signals.SignalWeight,
				loincCode:  "29463-7",
			},
			"TOTAL_CHOLESTEROL": {
				signalType: signals.SignalLipidPanel,
				loincCode:  "2093-3",
			},
			"HDL": {
				signalType: signals.SignalLipidPanel,
				loincCode:  "2085-9",
			},
		},
	}
}

// Map converts a KB-20 EventOutboxEntry to a MappedEvent.
// Returns (nil, nil) if the event type is not mapped (silently skipped).
func (m *EventSignalMapper) Map(entry models.EventOutboxEntry) (*MappedEvent, error) {
	switch entry.EventType {
	case models.EventLabResult:
		return m.mapLabResult(entry)
	case models.EventOrthostaticAlert:
		return m.mapAlertSignal(entry, signals.SignalOrthostatic, true)
	case models.EventBPSevereAlert, models.EventBPUrgencyAlert:
		return m.mapAlertSignal(entry, signals.SignalSBP, true)
	case models.EventBPAlert:
		return m.mapAlertSignal(entry, signals.SignalSBP, false)
	case models.EventGlucoseTrajectoryChange:
		return m.mapAlertSignal(entry, signals.SignalGlucoseCV, false)
	case models.EventACRWorsening:
		return m.mapACRWorsening(entry)
	case models.EventMedicationChange:
		return m.mapStateChange(entry, "MEDICATION_CHANGE")
	case models.EventStratumChange:
		return m.mapStateChange(entry, "STRATUM_CHANGE")
	case models.EventProtocolActivated:
		return m.mapStateChange(entry, "PROTOCOL_ACTIVATED")
	case models.EventProtocolTransitioned:
		return m.mapStateChange(entry, "PROTOCOL_TRANSITIONED")
	case models.EventProtocolGraduated:
		return m.mapStateChange(entry, "PROTOCOL_GRADUATED")
	case models.EventProtocolEscalated:
		return m.mapStateChange(entry, "PROTOCOL_ESCALATED")
	case models.EventMedicationThresholdCrossed:
		return m.mapAlertSignal(entry, signals.SignalCreatinine, true)

	// Patient-reported signal events (S4, S15, S16, S18-S22)
	case models.EventMealLog:
		return m.mapAlertSignal(entry, signals.SignalMealLog, false)
	case models.EventActivityLog:
		return m.mapAlertSignal(entry, signals.SignalActivity, false)
	case models.EventWaistMeasurement:
		return m.mapAlertSignal(entry, signals.SignalWaist, false)
	case models.EventAdherenceReport:
		return m.mapAlertSignal(entry, signals.SignalAdherence, false)
	case models.EventSymptomReport:
		return m.mapAlertSignal(entry, signals.SignalSymptom, false)
	case models.EventAdverseEvent:
		return m.mapAlertSignal(entry, signals.SignalAdverseEvent, true) // priority
	case models.EventResolutionReport:
		return m.mapAlertSignal(entry, signals.SignalResolution, false)
	case models.EventHospitalisation:
		return m.mapAlertSignal(entry, signals.SignalHospitalisation, true) // priority

	// Phase 6 P6-6: CKM substage transitions are routed through the
	// priority pipeline so KB-23 can react to 4c transitions with
	// MandatoryMedChecker (GDMT gap detection).
	case models.EventCKMStageTransition:
		return m.mapAlertSignal(entry, signals.SignalCKMStageTransition, true) // priority

	default:
		return nil, nil
	}
}

func (m *EventSignalMapper) mapLabResult(entry models.EventOutboxEntry) (*MappedEvent, error) {
	var lab models.LabResultPayload
	if err := json.Unmarshal(entry.Payload, &lab); err != nil {
		return nil, fmt.Errorf("unmarshal lab payload: %w", err)
	}
	mapping, ok := m.labTypeMap[lab.LabType]
	if !ok {
		return nil, nil
	}
	priority := false
	if mapping.priority != nil {
		priority = mapping.priority(lab.Value)
	}
	return &MappedEvent{
		Signal: &MappedSignal{
			EventID:    entry.ID,
			PatientID:  entry.PatientID,
			SignalType: mapping.signalType,
			Priority:   priority,
			Timestamp:  entry.CreatedAt,
			LOINCCode:  mapping.loincCode,
			Payload:    entry.Payload,
		},
	}, nil
}

func (m *EventSignalMapper) mapAlertSignal(entry models.EventOutboxEntry, st SignalType, priority bool) (*MappedEvent, error) {
	return &MappedEvent{
		Signal: &MappedSignal{
			EventID:    entry.ID,
			PatientID:  entry.PatientID,
			SignalType: st,
			Priority:   priority,
			Timestamp:  entry.CreatedAt,
			Payload:    entry.Payload,
		},
	}, nil
}

func (m *EventSignalMapper) mapACRWorsening(entry models.EventOutboxEntry) (*MappedEvent, error) {
	var acr models.ACRWorseningPayload
	if err := json.Unmarshal(entry.Payload, &acr); err != nil {
		return nil, fmt.Errorf("unmarshal ACR worsening: %w", err)
	}
	priority := acr.CurrentCategory == "A3"
	return &MappedEvent{
		Signal: &MappedSignal{
			EventID:    entry.ID,
			PatientID:  entry.PatientID,
			SignalType: signals.SignalACR,
			Priority:   priority,
			Timestamp:  entry.CreatedAt,
			LOINCCode:  "9318-7",
			Payload:    entry.Payload,
		},
	}, nil
}

func (m *EventSignalMapper) mapStateChange(entry models.EventOutboxEntry, changeType string) (*MappedEvent, error) {
	return &MappedEvent{
		StateChange: &MappedStateChange{
			EventID:    entry.ID,
			PatientID:  entry.PatientID,
			ChangeType: changeType,
			Timestamp:  entry.CreatedAt,
			Payload:    entry.Payload,
		},
	}, nil
}
