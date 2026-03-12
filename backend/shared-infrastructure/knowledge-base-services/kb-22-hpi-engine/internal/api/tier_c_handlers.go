package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/services"
)

// tierCComputeHandler triggers Tier C empirical LR computation for a node/stratum.
// Requires ≥200 adjudicated cases. Returns a governance proposal (PENDING_REVIEW)
// that must be approved before LRs are applied.
//
// POST /api/v1/calibration/tier-c/compute
// Body: { "node_id": "P01_CHEST_PAIN", "stratum": "DM_HTN_base" }
func (s *Server) tierCComputeHandler(c *gin.Context) {
	var req struct {
		NodeID  string `json:"node_id" binding:"required"`
		Stratum string `json:"stratum" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proposal, err := s.TierCService.ComputeEmpiricalLRs(c.Request.Context(), req.NodeID, req.Stratum)
	if err != nil {
		s.Log.Warn("Tier C computation failed",
			zap.String("node_id", req.NodeID),
			zap.String("stratum", req.Stratum),
			zap.Error(err),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"proposal": proposal,
		"message":  "Tier C proposal computed — requires governance approval",
	})
}

// tierCApproveHandler approves a Tier C calibration proposal, creating
// immutable CalibrationEvent records for every LR adjustment.
//
// POST /api/v1/calibration/tier-c/approve
// Body: { "proposal": { ... }, "approved_by": "calibration_committee", "rationale": "quarterly review Q3 2026" }
func (s *Server) tierCApproveHandler(c *gin.Context) {
	var req struct {
		Proposal   services.TierCProposal `json:"proposal" binding:"required"`
		ApprovedBy string                 `json:"approved_by" binding:"required"`
		Rationale  string                 `json:"rationale" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, err := s.TierCService.ApproveProposal(
		c.Request.Context(),
		&req.Proposal,
		req.ApprovedBy,
		req.Rationale,
	)
	if err != nil {
		s.Log.Error("Tier C approval failed",
			zap.String("proposal_id", req.Proposal.ProposalID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Publish approval event for downstream consumers (E07 Flink)
	if s.EventPublisher != nil {
		s.EventPublisher.PublishCalibrationUpdate(
			req.Proposal.NodeID,
			req.Proposal.StratumLabel,
			"DATA_DRIVEN",
			req.Proposal.TotalCases,
		)
	}

	now := time.Now()
	c.JSON(http.StatusOK, gin.H{
		"status":         "approved",
		"events_created": len(events),
		"approved_by":    req.ApprovedBy,
		"approved_at":    now.Format(time.RFC3339),
	})
}
