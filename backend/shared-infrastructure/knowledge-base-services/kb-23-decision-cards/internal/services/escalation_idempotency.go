package services

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kb-23-decision-cards/internal/models"
)

// AcknowledgePendingEscalation atomically finds the most recent pending or
// delivered escalation for the patient, applies the tracker's T2 transition
// inside a row-locking transaction, and persists. The state predicate in the
// WHERE clause is what provides idempotency: a second concurrent caller sees
// current_state already equal to ACKNOWLEDGED and First() returns
// ErrRecordNotFound, so the transaction no-ops rather than overwriting T2.
//
// Concurrency:
//   - On Postgres, SELECT ... FOR UPDATE serialises writers at the row level,
//     giving both correctness and throughput.
//   - On SQLite (tests), the state predicate alone guarantees idempotency:
//     writes are globally serialised by SQLite, and the second caller's
//     First() returns ErrRecordNotFound once the first transaction commits.
//
// Returns (nil, gorm.ErrRecordNotFound) when no eligible escalation exists —
// this is the expected "already acknowledged" path. Callers should treat it
// as a no-op, NOT as an error worth surfacing.
func AcknowledgePendingEscalation(db *gorm.DB, tracker *AcknowledgmentTracker, patientID, clinicianID string) (*models.EscalationEvent, error) {
	if db == nil || tracker == nil {
		return nil, nil
	}
	var event models.EscalationEvent
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("patient_id = ? AND current_state IN (?, ?)",
				patientID, string(models.StatePending), string(models.StateDelivered)).
			Order("created_at DESC").First(&event).Error; err != nil {
			return err
		}
		tracker.RecordAcknowledgment(&event, clinicianID)
		return tx.Save(&event).Error
	})
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// RecordActionOnAcknowledgedEscalation is the T3 sibling of
// AcknowledgePendingEscalation — same transactional discipline, same
// "already actioned" idempotency semantics via state predicate.
//
// T3 semantics note: the timestamp written here is "action initiated" (the
// moment the clinician tapped the action button), not "action completed".
// See LifecycleTracker.RecordT3 for the full contract.
func RecordActionOnAcknowledgedEscalation(db *gorm.DB, tracker *AcknowledgmentTracker, patientID, actionCode, notes string) (*models.EscalationEvent, error) {
	if db == nil || tracker == nil {
		return nil, nil
	}
	var event models.EscalationEvent
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("patient_id = ? AND current_state = ?",
				patientID, string(models.StateAcknowledged)).
			Order("created_at DESC").First(&event).Error; err != nil {
			return err
		}
		tracker.RecordAction(&event, actionCode, notes)
		return tx.Save(&event).Error
	})
	if err != nil {
		return nil, err
	}
	return &event, nil
}
