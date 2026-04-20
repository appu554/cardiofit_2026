package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

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
		_ = s.db.DB.Create(&verdict).Error
		_ = s.db.DB.Create(&entry).Error
	}
	c.JSON(http.StatusOK, gin.H{"verdict": verdict, "ledger_entry": entry})
}

// GET /api/v1/kb26/governance/ledger — return ledger entries with chain-validity status.
func (s *Server) getLedger(c *gin.Context) {
	if s.ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "governance ledger not initialized"})
		return
	}

	entries := s.ledger.Entries()
	ok, brokenIdx, _ := s.ledger.VerifyChain()
	c.JSON(http.StatusOK, gin.H{
		"entries":          entries,
		"chain_valid":      ok,
		"first_broken_idx": brokenIdx,
		"total":            len(entries),
	})
}
