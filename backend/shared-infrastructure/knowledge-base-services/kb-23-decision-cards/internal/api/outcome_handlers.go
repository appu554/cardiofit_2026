package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
		q := s.db.DB.Where("patient_id = ? AND outcome_type = ?", incoming.PatientID, incoming.OutcomeType)
		if incoming.LifecycleID != nil {
			q = q.Where("lifecycle_id = ?", *incoming.LifecycleID)
		}
		if err := q.Find(&prior).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load prior records: " + err.Error()})
			return
		}
		records = append(prior, incoming)
	}

	authoritative, err := services.ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reconciliation failed: " + err.Error()})
		return
	}

	if s.db != nil && s.db.DB != nil {
		if err := s.db.DB.Create(&authoritative).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist record: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"record": authoritative})
}
