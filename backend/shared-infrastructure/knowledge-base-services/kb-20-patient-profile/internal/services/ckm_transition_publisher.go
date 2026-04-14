package services

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// CKMTransitionPublisher records CKM substage transitions on the patient
// profile and publishes a CKM_STAGE_TRANSITION event via the event bus
// outbox. Phase 6 P6-6: enables KB-23 to react to stage changes (e.g.
// 4c transitions trigger MandatoryMedChecker for GDMT gap detection).
//
// Scope note — the publisher is the abstraction proof for Decision 9
// (event-driven cross-service handoff). Recomputing the stage from raw
// patient data via ClassifyCKMStage is a separate Phase 6 follow-up that
// requires assembling 25+ fields of CKMClassifierInput from the patient
// profile + new clinical event data. Once that recomputation service
// exists, it calls PublishStageTransition with the result; until then,
// the publisher can be invoked directly via HTTP for testing or by a
// future FHIR Condition sync that hasn't been built yet.
type CKMTransitionPublisher struct {
	db       *gorm.DB
	eventBus *EventBus
	logger   *zap.Logger
}

// NewCKMTransitionPublisher wires the dependencies.
func NewCKMTransitionPublisher(db *gorm.DB, eventBus *EventBus, logger *zap.Logger) *CKMTransitionPublisher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CKMTransitionPublisher{db: db, eventBus: eventBus, logger: logger}
}

// PublishStageTransition compares the new stage against the patient's
// current CKMStageV2, and on change: persists the new stage to the
// patient profile and publishes a CKM_STAGE_TRANSITION event. Returns
// (transitioned, error) so callers know whether the event fired.
//
// When the new stage matches the current stage, the call is a no-op
// (returns (false, nil)) — this prevents false-positive event spam
// when reclassification happens repeatedly with no actual change.
func (p *CKMTransitionPublisher) PublishStageTransition(
	patientID string,
	newStage string,
	rationale string,
	hfType string,
	triggeredByEventType string,
	triggeredByEventID string,
) (bool, error) {
	if patientID == "" {
		return false, nil
	}

	var profile models.PatientProfile
	if err := p.db.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return false, err
	}

	fromStage := profile.CKMStageV2
	if fromStage == newStage {
		// No-op: stage unchanged. This is the most common path and
		// prevents false-positive transition events.
		return false, nil
	}

	now := time.Now().UTC()
	if err := p.db.Model(&profile).
		Update("ckm_stage_v2", newStage).Error; err != nil {
		p.logger.Warn("failed to persist CKM stage transition",
			zap.String("patient_id", patientID),
			zap.String("from_stage", fromStage),
			zap.String("to_stage", newStage),
			zap.Error(err))
		return false, err
	}

	payload := models.CKMStageTransitionPayload{
		FromStage:            fromStage,
		ToStage:              newStage,
		TransitionDate:       now,
		StagingRationale:     rationale,
		HFType:               hfType,
		TriggeredByEventType: triggeredByEventType,
		TriggeredByEventID:   triggeredByEventID,
	}
	if p.eventBus != nil {
		p.eventBus.Publish(models.EventCKMStageTransition, patientID, payload)
	}

	p.logger.Info("CKM stage transition published",
		zap.String("patient_id", patientID),
		zap.String("from_stage", fromStage),
		zap.String("to_stage", newStage),
		zap.String("triggered_by", triggeredByEventType),
	)
	return true, nil
}
