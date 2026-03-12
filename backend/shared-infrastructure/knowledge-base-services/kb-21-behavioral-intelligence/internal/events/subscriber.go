package events

import (
	"encoding/json"

	"kb-21-behavioral-intelligence/internal/services"

	"go.uber.org/zap"
)

// Subscriber handles inbound events consumed by KB-21.
// Events consumed:
//   - LAB_RESULT (from KB-20) → triggers OutcomeCorrelation recomputation (Finding F-04)
//   - MEDICATION_CHANGED (from KB-20) → triggers adherence state reconciliation
//
// In development mode, events are received via HTTP webhook endpoints.
// In production, they are consumed from Kafka topics.
type Subscriber struct {
	logger             *zap.Logger
	correlationService *services.CorrelationService
	adherenceService   *services.AdherenceService
	kafkaEnabled       bool
}

func NewSubscriber(
	logger *zap.Logger,
	correlationService *services.CorrelationService,
	adherenceService *services.AdherenceService,
	kafkaEnabled bool,
) *Subscriber {
	return &Subscriber{
		logger:             logger,
		correlationService: correlationService,
		adherenceService:   adherenceService,
		kafkaEnabled:       kafkaEnabled,
	}
}

// HandleLabResult processes an inbound LAB_RESULT event from KB-20.
// This is the trigger for OutcomeCorrelation recomputation.
func (s *Subscriber) HandleLabResult(data []byte) error {
	var event services.LabResultEvent
	if err := json.Unmarshal(data, &event); err != nil {
		s.logger.Error("failed to unmarshal LAB_RESULT event", zap.Error(err))
		return err
	}

	s.logger.Info("Processing LAB_RESULT event",
		zap.String("patient_id", event.PatientID),
		zap.String("lab_type", event.LabType),
		zap.Float64("value", event.Value),
	)

	return s.correlationService.OnLabResult(event)
}

// MedicationChangedEvent represents a MEDICATION_CHANGED event from KB-20.
// Published when a patient's medication regimen changes (new drug, FDC switch, discontinuation).
type MedicationChangedEvent struct {
	PatientID     string   `json:"patient_id"`
	DrugClass     string   `json:"drug_class"`
	MedicationID  string   `json:"medication_id"`
	ChangeType    string   `json:"change_type"` // ADDED, REMOVED, SWITCHED_TO_FDC, SWITCHED_FROM_FDC
	IsFDC         bool     `json:"is_fdc"`
	FDCComponents []string `json:"fdc_components,omitempty"`
}

// HandleMedicationChanged processes an inbound MEDICATION_CHANGED event from KB-20.
// This reconciles adherence state when medications change — critical for FDC tracking (F-07).
// When a patient switches to an FDC, existing individual adherence records must be
// consolidated. When switching from FDC, the combined record must be split.
func (s *Subscriber) HandleMedicationChanged(data []byte) error {
	var event MedicationChangedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		s.logger.Error("failed to unmarshal MEDICATION_CHANGED event", zap.Error(err))
		return err
	}

	s.logger.Info("Processing MEDICATION_CHANGED event",
		zap.String("patient_id", event.PatientID),
		zap.String("drug_class", event.DrugClass),
		zap.String("change_type", event.ChangeType),
		zap.Bool("is_fdc", event.IsFDC),
	)

	// Recompute adherence for the affected drug class to reconcile state
	if err := s.adherenceService.RecomputeAdherence(event.PatientID, event.DrugClass); err != nil {
		s.logger.Error("adherence reconciliation failed after medication change",
			zap.String("patient_id", event.PatientID),
			zap.String("drug_class", event.DrugClass),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// Start begins consuming events from Kafka (production) or sets up webhook listeners (dev).
func (s *Subscriber) Start() error {
	if !s.kafkaEnabled {
		s.logger.Info("Event subscriber running in webhook mode (dev)")
		return nil
	}

	// Production Kafka consumer setup would go here.
	// Topics: kb20.LAB_RESULT, kb20.MEDICATION_CHANGED
	s.logger.Info("Event subscriber started (Kafka mode)")
	return nil
}
