package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-23-decision-cards/internal/models"
	"kb-23-decision-cards/internal/services"
)

// POST /api/v1/outcomes/ingest — ingest one OutcomeRecord.
// Looks up existing records for the same (patient, outcome_type, lifecycle_id),
// reconciles them with the incoming record via services.ReconcileOutcomes,
// persists the authoritative record (and updates prior PENDING rows) in a
// single GORM transaction, and returns the authoritative record.
//
// Body: OutcomeRecord JSON. Required: patient_id, outcome_type, source.
// Optional: idempotency_key — when set, a duplicate POST with the same key
// returns the existing record without creating a new one (safe for at-least-once feeds).
// Returns 200 {"record": OutcomeRecord} on success.
// Returns 200 {"record": OutcomeRecord, "idempotent": true} on duplicate key.
// Returns 400 on malformed JSON or missing required fields.
// Returns 500 on persistence failure.
func (s *Server) ingestOutcome(c *gin.Context) {
	var incoming models.OutcomeRecord
	if err := c.ShouldBindJSON(&incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if incoming.PatientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id is required"})
		return
	}
	if incoming.OutcomeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "outcome_type is required"})
		return
	}
	if incoming.Source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source is required"})
		return
	}
	if incoming.IngestedAt.IsZero() {
		incoming.IngestedAt = time.Now().UTC()
	}

	// Idempotency check — if an earlier ingest with the same key already
	// produced an authoritative row, return that row unchanged. Short-circuits
	// reconciliation + persist and makes at-least-once feed delivery safe.
	if incoming.IdempotencyKey != "" && s.db != nil && s.db.DB != nil {
		var existing models.OutcomeRecord
		err := s.db.DB.Where("idempotency_key = ?", incoming.IdempotencyKey).First(&existing).Error
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"record": existing, "idempotent": true})
			return
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			if s.log != nil {
				s.log.Error("idempotency key lookup failed",
					zap.Error(err),
					zap.String("idempotency_key", incoming.IdempotencyKey))
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "idempotency lookup failed: " + err.Error()})
			return
		}
	}

	// Collect prior records for the same (patient, outcome_type, lifecycle_id)
	// tuple. When DB is unavailable (tests), we reconcile against the single
	// incoming record alone — equivalent to "first source" semantics.
	records := []models.OutcomeRecord{incoming}
	var priorIDs []uuid.UUID
	if s.db != nil && s.db.DB != nil {
		var prior []models.OutcomeRecord
		// Only include PENDING prior rows — rows already marked RESOLVED or
		// CONFLICTED from earlier ingest calls must not be pulled back into
		// a second reconciliation pass, otherwise the authoritative verdict
		// can be silently re-derived on every new source arrival.
		q := s.db.DB.
			Where("patient_id = ? AND outcome_type = ?", incoming.PatientID, incoming.OutcomeType).
			Where("reconciliation = ?", string(models.ReconciliationPending))
		// When LifecycleID is nil, the query spans ALL lifecycles for this
		// (patient, outcome_type). This is intentional for "global sweep"
		// ingest (e.g., mortality registry feeds that don't know about
		// alert lifecycles) but semantically undefined for feed sources
		// that SHOULD know the lifecycle. Sprint 3 Task 2 makes this explicit
		// via a scope discriminator.
		if incoming.LifecycleID != nil {
			q = q.Where("lifecycle_id = ?", *incoming.LifecycleID)
		}
		if err := q.Find(&prior).Error; err != nil {
			if s.log != nil {
				s.log.Error("failed to load prior outcome records for reconciliation",
					zap.Error(err),
					zap.String("patient_id", incoming.PatientID),
					zap.String("outcome_type", incoming.OutcomeType))
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load prior records: " + err.Error()})
			return
		}
		records = append(prior, incoming)
		for _, p := range prior {
			priorIDs = append(priorIDs, p.ID)
		}
	}

	authoritative, err := services.ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		if s.log != nil {
			s.log.Error("reconciliation failed during outcome ingest",
				zap.Error(err),
				zap.String("patient_id", incoming.PatientID),
				zap.String("outcome_type", incoming.OutcomeType),
				zap.Int("num_records", len(records)))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reconciliation failed: " + err.Error()})
		return
	}

	if s.db != nil && s.db.DB != nil {
		// Zero out the authoritative ID so BeforeCreate generates a fresh UUID.
		// ReconcileOutcomes returns records[0] as the base, which may carry an
		// already-persisted UUID from a prior DB row. Always writing a NEW row
		// as the authoritative ensures the prior rows can be updated in place
		// (pointing at the new authoritative ID) without a UNIQUE key collision.
		authoritative.ID = uuid.Nil

		// Transaction: Create authoritative + Update prior rows' ReconciledID
		// and Reconciliation status. If either fails, both roll back so the
		// table never ends up with a half-promoted prior row pointing at a
		// phantom authoritative.
		txErr := s.db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&authoritative).Error; err != nil {
				return err
			}
			if len(priorIDs) > 0 && authoritative.Reconciliation == string(models.ReconciliationResolved) {
				if err := tx.Model(&models.OutcomeRecord{}).
					Where("id IN ?", priorIDs).
					Updates(map[string]interface{}{
						"reconciled_id":  authoritative.ID,
						"reconciliation": string(models.ReconciliationResolved),
					}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			if s.log != nil {
				s.log.Error("failed to persist authoritative outcome record (txn rolled back)",
					zap.Error(txErr),
					zap.String("patient_id", incoming.PatientID),
					zap.String("outcome_type", incoming.OutcomeType))
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist record: " + txErr.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"record": authoritative})
}
