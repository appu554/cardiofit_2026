package services

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// SafetyEventRecorder persists safety events to the safety_events
// table alongside the existing event-bus publish. Phase 8 P8-5: the
// summary-context service reads from this table to populate the
// confounder flags (IsAcuteIll, HasRecentTransfusion,
// HasRecentHypoglycaemia) that gate MCU clinical rules.
//
// Kept narrow on purpose — the recorder is a write-only sink. It
// does not subscribe to the event bus; instead, callers that already
// publish safety events (lab_service.deriveEGFR, lab_service.processACR,
// etc.) invoke Record alongside their eventBus.Publish call. This
// keeps the Kafka flow unchanged while adding a persistent, queryable
// audit trail.
type SafetyEventRecorder struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewSafetyEventRecorder wires the dependencies.
func NewSafetyEventRecorder(db *gorm.DB, logger *zap.Logger) *SafetyEventRecorder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SafetyEventRecorder{db: db, logger: logger}
}

// Record writes a SafetyEvent row. Returns an error on DB failure
// so callers can log at warn level, but the caller should NOT abort
// the surrounding clinical operation on record failure — the Kafka
// event still fires, downstream consumers still see the alert, and
// the only thing lost is the confounder-flag lookup for this
// specific event. Defensive failure-isolation matches the pattern
// used by stampMedicationChange and lab_service.updateCKDStatus.
func (r *SafetyEventRecorder) Record(
	patientID string,
	eventType string,
	severity string,
	description string,
	observedAt time.Time,
) error {
	if r == nil || r.db == nil {
		return nil
	}
	if patientID == "" || eventType == "" {
		return nil
	}
	event := &models.SafetyEvent{
		ID:          uuid.New(),
		PatientID:   patientID,
		EventType:   eventType,
		Severity:    severity,
		Description: description,
		ObservedAt:  observedAt,
	}
	if err := r.db.Create(event).Error; err != nil {
		r.logger.Warn("failed to persist safety event",
			zap.String("patient_id", patientID),
			zap.String("event_type", eventType),
			zap.Error(err))
		return err
	}
	return nil
}

// RecordLabEvent is the lab-derived variant that fills in LabType /
// OldValue / NewValue from a SafetyAlertPayload. Used by lab_service
// safety event publish paths (EGFR critical, potassium high, etc.).
func (r *SafetyEventRecorder) RecordLabEvent(
	patientID string,
	eventType string,
	severity string,
	description string,
	labType string,
	oldValue string,
	newValue string,
	observedAt time.Time,
) error {
	if r == nil || r.db == nil {
		return nil
	}
	if patientID == "" || eventType == "" {
		return nil
	}
	event := &models.SafetyEvent{
		ID:          uuid.New(),
		PatientID:   patientID,
		EventType:   eventType,
		Severity:    severity,
		Description: description,
		LabType:     labType,
		OldValue:    oldValue,
		NewValue:    newValue,
		ObservedAt:  observedAt,
	}
	if err := r.db.Create(event).Error; err != nil {
		r.logger.Warn("failed to persist lab safety event",
			zap.String("patient_id", patientID),
			zap.String("event_type", eventType),
			zap.String("lab_type", labType),
			zap.Error(err))
		return err
	}
	return nil
}

// ConfounderFlags derives the three flags the MCU gate manager
// reads from a sliding window of recent safety events:
//
//   - IsAcuteIll:             ACUTE_ILLNESS within 7 days
//   - HasRecentTransfusion:   BLOOD_TRANSFUSION within 90 days
//   - HasRecentHypoglycaemia: HYPO_EVENT within 30 days (any severity
//                             for simplicity; the MCU gate manager's
//                             downstream logic can tighten further)
//
// Returns zero-valued flags on nil recorder / nil db so the summary-
// context service degrades cleanly when the recorder is not wired.
// Phase 8 P8-5.
func (r *SafetyEventRecorder) ConfounderFlags(patientID string, now time.Time) (isAcuteIll, hasRecentTransfusion, hasRecentHypoglycaemia bool) {
	if r == nil || r.db == nil || patientID == "" {
		return false, false, false
	}

	isAcuteIll = r.hasRecentEvent(patientID, models.SafetyEventAcuteIllness, now.AddDate(0, 0, -7))
	hasRecentTransfusion = r.hasRecentEvent(patientID, models.SafetyEventBloodTransfusion, now.AddDate(0, 0, -90))
	hasRecentHypoglycaemia = r.hasRecentEvent(patientID, models.SafetyEventHypoEvent, now.AddDate(0, 0, -30))
	return isAcuteIll, hasRecentTransfusion, hasRecentHypoglycaemia
}

// hasRecentEvent returns true if the patient has at least one
// SafetyEvent of the given type observed at or after the given
// cutoff time. A DB error returns false — defensive failure isolation.
func (r *SafetyEventRecorder) hasRecentEvent(patientID, eventType string, since time.Time) bool {
	var count int64
	err := r.db.Model(&models.SafetyEvent{}).
		Where("patient_id = ? AND event_type = ? AND observed_at >= ?", patientID, eventType, since).
		Count(&count).Error
	if err != nil {
		r.logger.Warn("safety event query failed",
			zap.String("patient_id", patientID),
			zap.String("event_type", eventType),
			zap.Error(err))
		return false
	}
	return count > 0
}
