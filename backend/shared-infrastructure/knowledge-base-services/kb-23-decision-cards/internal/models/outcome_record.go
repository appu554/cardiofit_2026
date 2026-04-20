package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OutcomeSource identifies where an outcome record came from.
type OutcomeSource string

const (
	OutcomeSourceHospitalDischarge OutcomeSource = "HOSPITAL_DISCHARGE"
	OutcomeSourceClaimsFeed        OutcomeSource = "CLAIMS_FEED"
	OutcomeSourceMortalityRegistry OutcomeSource = "MORTALITY_REGISTRY"
	OutcomeSourceClinicianConfirm  OutcomeSource = "CLINICIAN_CONFIRMATION"
	OutcomeSourceFacilityReport    OutcomeSource = "FACILITY_REPORT"
)

// ReconciliationStatus tracks whether an outcome has been resolved across sources.
type ReconciliationStatus string

const (
	ReconciliationPending    ReconciliationStatus = "PENDING"
	ReconciliationResolved   ReconciliationStatus = "RESOLVED"
	ReconciliationConflicted ReconciliationStatus = "CONFLICTED"
	ReconciliationHorizonExp ReconciliationStatus = "HORIZON_EXPIRED"
)

// OutcomeRecord is a single outcome observation for one patient from one source.
// Multiple OutcomeRecords for the same (patient, outcome_type) are reconciled into
// a single authoritative record by OutcomeIngestionService.
type OutcomeRecord struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID      string    `gorm:"size:100;index;index:idx_or_patient_type,priority:1;not null" json:"patient_id"`
	LifecycleID    *uuid.UUID `gorm:"type:uuid;index" json:"lifecycle_id,omitempty"` // nil if no alert was generated before the outcome arrived (e.g., registry sweep predating any lifecycle)
	CohortID       string    `gorm:"size:60;index" json:"cohort_id,omitempty"`
	OutcomeType    string    `gorm:"size:60;index;index:idx_or_patient_type,priority:2;not null" json:"outcome_type"` // READMISSION_30D, ADMISSION_90D, MORTALITY_30D, etc.
	OutcomeOccurred bool     `gorm:"not null" json:"outcome_occurred"`
	OccurredAt     *time.Time `json:"occurred_at,omitempty"`
	Source         string    `gorm:"size:40;index;not null" json:"source"`
	SourceRecordID string    `gorm:"size:200" json:"source_record_id,omitempty"`
	// Feed-supplied idempotency key. When set, POST /outcomes/ingest with a
	// duplicate key returns the existing record instead of creating a new one.
	// Required for at-least-once claims/discharge feeds to avoid duplicate
	// reconciliation passes. uniqueIndex allows multiple NULL values (standard
	// SQL), so legacy records without a key are unaffected.
	IdempotencyKey  string     `gorm:"size:128;uniqueIndex:idx_or_idem_key" json:"idempotency_key,omitempty"`
	Reconciliation string    `gorm:"size:20;index;not null;default:'PENDING'" json:"reconciliation"`
	ReconciledID   *uuid.UUID `gorm:"type:uuid" json:"reconciled_id,omitempty"` // points to authoritative record after reconciliation
	IngestedAt     time.Time `gorm:"autoCreateTime" json:"ingested_at"`
	Notes          string    `gorm:"type:text" json:"notes,omitempty"`
}

func (OutcomeRecord) TableName() string { return "outcome_records" }

// BeforeCreate generates a UUID primary key if not already set.
// Mirrors the pattern used by DecisionCard, MCUGateHistory, etc., and ensures
// SQLite-backed test fixtures work without gen_random_uuid() support.
func (o *OutcomeRecord) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
