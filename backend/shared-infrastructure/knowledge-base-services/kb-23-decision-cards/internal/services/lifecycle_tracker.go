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
	db            *gorm.DB // nil-safe: when nil, operates purely in-memory (for tests)
	log           *zap.Logger
	defaultCohort string // stamped onto every RecordT0 when no explicit cohort is given
}

// NewLifecycleTracker creates a tracker. Both db and log may be nil for testing.
func NewLifecycleTracker(db *gorm.DB, log *zap.Logger) *LifecycleTracker {
	if log == nil {
		log = zap.NewNop()
	}
	return &LifecycleTracker{db: db, log: log}
}

// SetDefaultCohort sets the cohort stamped on lifecycles that don't carry
// their own. Configured per KB-23 deployment (e.g. an HCF-pilot instance
// defaults to "hcf_catalyst_chf"); empty string means "unassigned".
func (t *LifecycleTracker) SetDefaultCohort(cohort string) {
	t.defaultCohort = cohort
}

// RecordT0 creates a new DetectionLifecycle at the detection moment.
// cohort may be empty; when empty the tracker's defaultCohort is used.
func (t *LifecycleTracker) RecordT0(
	detectionType, detectionSubtype, patientID, tier, sourceService, cohort string,
	cardID, escalationID *uuid.UUID,
) *models.DetectionLifecycle {
	if cohort == "" {
		cohort = t.defaultCohort
	}
	lc := &models.DetectionLifecycle{
		ID:               uuid.New(),
		DetectionType:    detectionType,
		DetectionSubtype: detectionSubtype,
		PatientID:        patientID,
		TierAtDetection:  tier,
		CohortID:         cohort,
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
//
// T3 semantics — "action initiated", NOT "action completed". The timestamp
// captured here is the moment the clinician signals intent (e.g. clicks
// CALL_PATIENT in the worklist, marks RECHECK_VITALS on the ward round).
// The actual call/visit/recheck typically happens minutes to hours later.
//
// Implication for Gap 19 response metrics:
//   - ActionLatencyMs = T3 − T2 measures intent-to-act latency, i.e. how
//     fast the clinician commits to a response after acknowledging.
//   - It does NOT measure how fast the patient is actually helped.
//
// If a future "action completed" signal is needed (e.g. call actually
// placed, visit documented, dose administered), introduce a new T4 stage
// rather than redefining T3 — callers already depend on the intent-to-act
// interpretation.
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

// FindMostRecentActionedByPatient returns the most recent lifecycle for a
// patient that has been actioned but not yet resolved. Used by the T4 bridge
// from KB-26: when an acute event resolves ("weight back to baseline") we
// attribute that outcome confirmation to the most recent actioned detection
// for the patient, optionally narrowed by detection type. This is the
// deliberate-but-crude Sprint 1 attribution; a future attribution engine
// can supersede it without changing callers.
//
// Returns gorm.ErrRecordNotFound when nothing matches.
func (t *LifecycleTracker) FindMostRecentActionedByPatient(patientID, detectionType string) (*models.DetectionLifecycle, error) {
	if t.db == nil {
		return nil, fmt.Errorf("no database")
	}
	q := t.db.Where("patient_id = ? AND actioned_at IS NOT NULL AND resolved_at IS NULL", patientID)
	if detectionType != "" {
		q = q.Where("detection_type = ?", detectionType)
	}
	var lc models.DetectionLifecycle
	if err := q.Order("actioned_at DESC").First(&lc).Error; err != nil {
		return nil, err
	}
	return &lc, nil
}
