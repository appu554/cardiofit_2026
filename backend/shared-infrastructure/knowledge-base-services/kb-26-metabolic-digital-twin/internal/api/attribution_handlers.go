package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// POST /api/v1/kb26/attribution/run — run attribution for one consolidated record.
// Body: AttributionInput JSON; returns AttributionVerdict + ledger entry.
func (s *Server) runAttribution(c *gin.Context) {
	if s.ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "governance ledger not initialized"})
		return
	}

	var in services.AttributionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	verdict := services.ComputeAttribution(in)

	payload, _ := json.Marshal(verdict)
	entry, err := s.ledger.AppendEntry("ATTRIBUTION_RUN", verdict.ID.String(), string(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	verdict.LedgerEntryID = &entry.ID

	if s.db != nil && s.db.DB != nil {
		// Sprint 2a Task 3: single transaction so a verdict cannot persist
		// with a LedgerEntryID pointing at a ledger row that was rolled back.
		// If either Create fails, both roll back.
		txErr := s.db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&verdict).Error; err != nil {
				s.logger.Warn("failed to persist attribution verdict (txn will roll back)",
					zap.Error(err),
					zap.String("verdict_id", verdict.ID.String()))
				return err
			}
			if err := tx.Create(&entry).Error; err != nil {
				s.logger.Error("failed to persist ledger entry (txn will roll back)",
					zap.Error(err),
					zap.Int64("seq", entry.Sequence),
					zap.String("verdict_id", verdict.ID.String()))
				return err
			}
			return nil
		})
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist attribution (transaction rolled back): " + txErr.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"verdict": verdict, "ledger_entry": entry})
}

// GET /api/v1/kb26/attribution/:patientId — return the patient's attribution
// verdict history, most recent first.
//
// Query params:
//   - limit: max records (default 50, max 500)
//
// Returns 200 {"patient_id": ..., "verdicts": [...], "total": N, "limit": L}.
// Returns 400 if patient_id is empty/whitespace.
func (s *Server) getAttributionByPatient(c *gin.Context) {
	patientID := strings.TrimSpace(c.Param("patientId"))
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id is required"})
		return
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	verdicts := []models.AttributionVerdict{}
	if s.db != nil && s.db.DB != nil {
		if err := s.db.DB.
			Where("patient_id = ?", patientID).
			Order("computed_at DESC").
			Limit(limit).
			Find(&verdicts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"verdicts":   verdicts,
		"total":      len(verdicts),
		"limit":      limit,
	})
}

// GET /api/v1/kb26/governance/ledger — return ledger entries with chain-validity status.
func (s *Server) getLedger(c *gin.Context) {
	if s.ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "governance ledger not initialized"})
		return
	}

	entries, ok, brokenIdx := s.ledger.Snapshot()
	c.JSON(http.StatusOK, gin.H{
		"entries":          entries,
		"chain_valid":      ok,
		"first_broken_idx": brokenIdx,
		"total":            len(entries),
	})
}
