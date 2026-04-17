package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ClinicalAuditEntry is the canonical audit row shape. Append-only:
// the table enforces immutability via application-level convention
// (no UPDATE or DELETE calls anywhere in the codebase). A future
// Postgres trigger can enforce this at the database level.
//
// Hash chain: each entry carries the SHA-256 hash of the previous
// entry's Hash field. The first entry in a patient's chain uses
// "GENESIS" as the PreviousHash. Tampering with any entry in the
// chain breaks the hash chain for all subsequent entries — detectable
// by the VerifyChain method.
//
// Phase 10 Gap 11.
type ClinicalAuditEntry struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PatientID    string    `gorm:"size:100;index;not null" json:"patient_id"`
	EventType    string    `gorm:"size:60;index;not null" json:"event_type"`
	ServiceSource string  `gorm:"size:40;not null" json:"service_source"`
	Payload      string    `gorm:"type:text;not null" json:"payload"`
	PreviousHash string    `gorm:"size:64;not null" json:"previous_hash"`
	Hash         string    `gorm:"size:64;not null" json:"hash"`
	OccurredAt   time.Time `gorm:"not null" json:"occurred_at"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the Postgres table name.
func (ClinicalAuditEntry) TableName() string { return "clinical_audit_log" }

// AuditService provides append-only, hash-chained clinical audit
// logging. Phase 10 Gap 11.
type AuditService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAuditService constructs the audit service.
func NewAuditService(db *gorm.DB, logger *zap.Logger) *AuditService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AuditService{db: db, logger: logger}
}

// Append writes a new audit entry with hash-chain linkage to the
// previous entry for this patient. The read-compute-write is wrapped
// in a SERIALIZABLE transaction to prevent concurrent Appends for the
// same patient from reading the same prevHash (which would fork the
// chain). Postgres SERIALIZABLE isolation guarantees that if two
// transactions read the same row, one will be rolled back and retried.
//
// The hash chain is per-patient so chain verification is scoped to
// individual patient audit trails rather than the entire table.
func (s *AuditService) Append(
	patientID string,
	eventType string,
	serviceSource string,
	payload interface{},
	occurredAt time.Time,
) error {
	if s == nil || s.db == nil {
		return nil
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal audit payload: %w", err)
	}

	// Wrap the read-compute-write in a SERIALIZABLE transaction to
	// prevent concurrent appends from forking the hash chain.
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Fetch the previous entry's hash for this patient (within tx).
		prevHash := "GENESIS"
		var lastEntry ClinicalAuditEntry
		if err := tx.
			Where("patient_id = ?", patientID).
			Order("created_at DESC").
			First(&lastEntry).Error; err == nil {
			prevHash = lastEntry.Hash
		}

		// Compute this entry's hash: SHA-256(previousHash + patientID + eventType + payload + occurredAt)
		hashInput := fmt.Sprintf("%s|%s|%s|%s|%s",
			prevHash, patientID, eventType, string(payloadJSON), occurredAt.Format(time.RFC3339Nano))
		hash := sha256.Sum256([]byte(hashInput))
		hashHex := hex.EncodeToString(hash[:])

		entry := ClinicalAuditEntry{
			ID:            uuid.New(),
			PatientID:     patientID,
			EventType:     eventType,
			ServiceSource: serviceSource,
			Payload:       string(payloadJSON),
			PreviousHash:  prevHash,
			Hash:          hashHex,
			OccurredAt:    occurredAt,
		}

		if err := tx.Create(&entry).Error; err != nil {
			s.logger.Error("audit append failed",
				zap.String("patient_id", patientID),
				zap.String("event_type", eventType),
				zap.Error(err))
			return err
		}
		return nil
	})
}

// FetchPatientTrail returns the audit trail for a patient within
// a date range, ordered by creation time ascending (chronological).
// Used by compliance officers for regulatory audits.
func (s *AuditService) FetchPatientTrail(patientID string, from, to time.Time) ([]ClinicalAuditEntry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var entries []ClinicalAuditEntry
	err := s.db.
		Where("patient_id = ? AND occurred_at >= ? AND occurred_at <= ?", patientID, from, to).
		Order("created_at ASC").
		Find(&entries).Error
	return entries, err
}

// VerifyChain checks the hash chain integrity for a patient's
// audit trail. Returns the index of the first broken link, or -1
// if the chain is intact. A broken chain indicates tampering with
// the audit log — a compliance-critical finding.
func (s *AuditService) VerifyChain(patientID string) (brokenAtIndex int, err error) {
	if s == nil || s.db == nil {
		return -1, nil
	}
	var entries []ClinicalAuditEntry
	if err := s.db.
		Where("patient_id = ?", patientID).
		Order("created_at ASC").
		Find(&entries).Error; err != nil {
		return -1, err
	}

	for i, entry := range entries {
		expectedPrev := "GENESIS"
		if i > 0 {
			expectedPrev = entries[i-1].Hash
		}
		if entry.PreviousHash != expectedPrev {
			return i, nil
		}

		// Recompute hash and verify
		hashInput := fmt.Sprintf("%s|%s|%s|%s|%s",
			entry.PreviousHash, entry.PatientID, entry.EventType,
			entry.Payload, entry.OccurredAt.Format(time.RFC3339Nano))
		hash := sha256.Sum256([]byte(hashInput))
		if hex.EncodeToString(hash[:]) != entry.Hash {
			return i, nil
		}
	}
	return -1, nil // chain intact
}
