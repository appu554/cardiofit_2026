package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
	"kb-23-decision-cards/internal/services"
)

// POST /api/v1/outcomes/ingest — ingest one OutcomeRecord.
// Looks up existing records for the same (patient, outcome_type, lifecycle_id),
// reconciles them with the incoming record via services.ReconcileOutcomes,
// persists the authoritative record, and returns it.
//
// Body: OutcomeRecord JSON. Required: patient_id, outcome_type, source.
// Returns 200 {"record": OutcomeRecord} on success.
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

	// Collect prior records for the same (patient, outcome_type, lifecycle_id)
	// tuple. When DB is unavailable (tests), we reconcile against the single
	// incoming record alone — equivalent to "first source" semantics.
	records := []models.OutcomeRecord{incoming}
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
		// that SHOULD know the lifecycle. Sprint 3 should either require
		// LifecycleID for feed-side calls or add a "scope": "global"
		// discriminator to make the intent explicit in the request body.
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
		// Sprint 3 debt: this Create writes a new row, but the model's
		// ReconciledID field on the prior PENDING rows is NOT updated to
		// point at the authoritative row. A future Sprint 3 task should
		// wrap both the Create and the prior-rows ReconciledID update in a
		// single transaction (mirrors the kb-26 attribution_handlers.go
		// transaction-wrap work in Sprint 2a Task 3).
		if err := s.db.DB.Create(&authoritative).Error; err != nil {
			if s.log != nil {
				s.log.Error("failed to persist authoritative outcome record",
					zap.Error(err),
					zap.String("patient_id", incoming.PatientID),
					zap.String("outcome_type", incoming.OutcomeType))
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist record: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"record": authoritative})
}
