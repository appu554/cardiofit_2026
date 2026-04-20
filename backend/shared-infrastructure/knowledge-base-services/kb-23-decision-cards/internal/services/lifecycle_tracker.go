package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"kb-23-decision-cards/internal/models"
)

// LifecycleTracker manages the T0→T4 detection lifecycle state machine,
// computing inter-stage latencies at each transition.
type LifecycleTracker struct {
	db  *gorm.DB   // nil-safe: when nil, operates purely in-memory (for tests)
	log *zap.Logger
}

// NewLifecycleTracker creates a tracker. Both db and log may be nil for testing.
func NewLifecycleTracker(db *gorm.DB, log *zap.Logger) *LifecycleTracker {
	if log == nil {
		log = zap.NewNop()
	}
	return &LifecycleTracker{db: db, log: log}
}

// RecordT0 creates a new DetectionLifecycle at the detection moment.
func (t *LifecycleTracker) RecordT0(
	detectionType, detectionSubtype, patientID, tier, sourceService string,
	cardID, escalationID *uuid.UUID,
) *models.DetectionLifecycle {
	lc := &models.DetectionLifecycle{
		ID:               uuid.New(),
		DetectionType:    detectionType,
		DetectionSubtype: detectionSubtype,
		PatientID:        patientID,
		TierAtDetection:  tier,
		CurrentState:     string(models.LifecyclePendingNotification),
		DetectedAt:       time.Now().UTC(),
		CardID:           cardID,
		EscalationID:     escalationID,
		SourceService:    sourceService,
	}
	if t.db != nil {
		t.db.Create(lc)
	}
	return lc
}

// RecordT1 marks the detection as delivered to the notification channel.
func (t *LifecycleTracker) RecordT1(lc *models.DetectionLifecycle, deliveredAt time.Time) {
	lc.DeliveredAt = &deliveredAt
	lc.CurrentState = string(models.LifecycleNotified)
	latency := deliveredAt.Sub(lc.DetectedAt).Milliseconds()
	lc.DeliveryLatencyMs = &latency
	// If T2 was recorded out-of-order, compute AcknowledgmentLatencyMs now
	if lc.AcknowledgedAt != nil && lc.AcknowledgmentLatencyMs == nil {
		ackLatency := lc.AcknowledgedAt.Sub(deliveredAt).Milliseconds()
		lc.AcknowledgmentLatencyMs = &ackLatency
	}
	if t.db != nil {
		t.db.Save(lc)
	}
}

// RecordT2 marks the detection as acknowledged by a clinician.
// First-write-wins: when the same detection is delivered via multiple channels
// (Push + SMS + WhatsApp), the earliest acknowledgment is the canonical T2.
// Subsequent calls are no-ops so the latency isn't inflated by duplicate taps.
func (t *LifecycleTracker) RecordT2(lc *models.DetectionLifecycle, clinicianID string, acknowledgedAt time.Time) {
	if lc.AcknowledgedAt != nil {
		return
	}
	lc.AcknowledgedAt = &acknowledgedAt
	lc.AssignedClinicianID = clinicianID
	lc.CurrentState = string(models.LifecycleAcknowledged)
	if lc.DeliveredAt != nil {
		latency := acknowledgedAt.Sub(*lc.DeliveredAt).Milliseconds()
		lc.AcknowledgmentLatencyMs = &latency
	}
	if t.db != nil {
		t.db.Save(lc)
	}
}

// RecordT3 marks the detection as actioned by the clinician.
func (t *LifecycleTracker) RecordT3(lc *models.DetectionLifecycle, actionType, actionDetail string, actionedAt time.Time) {
	lc.ActionedAt = &actionedAt
	lc.ActionType = actionType
	lc.ActionDetail = actionDetail
	lc.CurrentState = string(models.LifecycleActioned)
	if lc.AcknowledgedAt != nil {
		latency := actionedAt.Sub(*lc.AcknowledgedAt).Milliseconds()
		lc.ActionLatencyMs = &latency
	}
	if t.db != nil {
		t.db.Save(lc)
	}
}

// RecordT4 marks the detection as resolved and computes total lifecycle latency.
func (t *LifecycleTracker) RecordT4(lc *models.DetectionLifecycle, outcomeDescription string, resolvedAt time.Time) {
	lc.ResolvedAt = &resolvedAt
	lc.OutcomeDescription = outcomeDescription
	lc.CurrentState = string(models.LifecycleResolved)
	if lc.ActionedAt != nil {
		latency := resolvedAt.Sub(*lc.ActionedAt).Milliseconds()
		lc.OutcomeLatencyMs = &latency
	}
	total := resolvedAt.Sub(lc.DetectedAt).Milliseconds()
	lc.TotalLatencyMs = &total
	if t.db != nil {
		t.db.Save(lc)
	}
}

// FindByEscalation retrieves a lifecycle by its escalation ID.
func (t *LifecycleTracker) FindByEscalation(escalationID uuid.UUID) (*models.DetectionLifecycle, error) {
	if t.db == nil {
		return nil, fmt.Errorf("no database")
	}
	var lc models.DetectionLifecycle
	if err := t.db.Where("escalation_id = ?", escalationID).First(&lc).Error; err != nil {
		return nil, err
	}
	return &lc, nil
}
