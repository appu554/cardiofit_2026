package review

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// ReviewHandler
// ---------------------------------------------------------------------------

type ReviewHandler struct {
	queue  *Queue
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewReviewHandler(db *pgxpool.Pool, logger *zap.Logger) *ReviewHandler {
	return &ReviewHandler{
		queue:  NewQueue(db, logger),
		db:     db,
		logger: logger,
	}
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

type SubmitReviewRequest struct {
	HardStopCount int     `json:"hard_stop_count"`
	SoftFlagCount int     `json:"soft_flag_count"`
	Age           int     `json:"age"`
	MedCount      int     `json:"med_count"`
	EGFRValue     float64 `json:"egfr_value"`
}

type ClarificationRequest struct {
	SlotNames []string `json:"slot_names"`
	Notes     string   `json:"notes"`
}

type EscalateRequest struct {
	Reason string `json:"reason"`
	Notes  string `json:"notes"`
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// HandleSubmitReview accepts a review submission for a given encounter.
func (h *ReviewHandler) HandleSubmitReview(c *gin.Context) {
	encounterID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	var req SubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify intake is completed before submitting for review.
	var state string
	err = h.db.QueryRow(c.Request.Context(), `
		SELECT state FROM enrollments WHERE encounter_id = $1`,
		encounterID,
	).Scan(&state)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enrollment not found"})
		return
	}
	if state != "INTAKE_COMPLETED" {
		c.JSON(http.StatusConflict, gin.H{"error": "enrollment not in INTAKE_COMPLETED state"})
		return
	}

	// Classify risk.
	risk := ClassifyRisk(RiskClassificationInput{
		HardStopCount: req.HardStopCount,
		SoftFlagCount: req.SoftFlagCount,
		Age:           req.Age,
		MedCount:      req.MedCount,
		EGFRValue:     req.EGFRValue,
	})

	// Build and submit entry.
	entry := ReviewEntry{
		EncounterID: encounterID,
		RiskStratum: risk,
	}

	// Pull patient and tenant from enrollment row.
	_ = h.db.QueryRow(c.Request.Context(), `
		SELECT patient_id, tenant_id FROM enrollments WHERE encounter_id = $1`,
		encounterID,
	).Scan(&entry.PatientID, &entry.TenantID)

	result, err := h.queue.Submit(c.Request.Context(), entry)
	if err != nil {
		h.logger.Error("failed to submit review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit review"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// HandleApprove approves a review entry and transitions enrollment to ENROLLED.
func (h *ReviewHandler) HandleApprove(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}

	reviewerID := extractReviewerID(c)
	if reviewerID == uuid.Nil {
		return // response already sent
	}

	if err := h.queue.Approve(c.Request.Context(), entryID, reviewerID); err != nil {
		h.logger.Error("failed to approve", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve"})
		return
	}

	// Transition enrollment to ENROLLED — look up encounter_id by entry ID.
	var encounterID uuid.UUID
	row := h.db.QueryRow(c.Request.Context(), `
		SELECT encounter_id FROM review_queue WHERE id = $1`, entryID)
	if row.Scan(&encounterID) == nil {
		_, _ = h.db.Exec(c.Request.Context(), `
			UPDATE enrollments SET state = 'ENROLLED' WHERE encounter_id = $1`,
			encounterID,
		)
	}

	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

// HandleRequestClarification reverts enrollment and marks the entry.
func (h *ReviewHandler) HandleRequestClarification(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}

	reviewerID := extractReviewerID(c)
	if reviewerID == uuid.Nil {
		return
	}

	var req ClarificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.queue.RequestClarification(c.Request.Context(), entryID, reviewerID, req.Notes); err != nil {
		h.logger.Error("failed to request clarification", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to request clarification"})
		return
	}

	// Revert enrollment to INTAKE_IN_PROGRESS.
	var encounterID uuid.UUID
	row := h.db.QueryRow(c.Request.Context(), `
		SELECT encounter_id FROM review_queue WHERE id = $1`, entryID)
	if row.Scan(&encounterID) == nil {
		_, _ = h.db.Exec(c.Request.Context(), `
			UPDATE enrollments SET state = 'INTAKE_IN_PROGRESS' WHERE encounter_id = $1`,
			encounterID,
		)
	}

	c.JSON(http.StatusOK, gin.H{"status": "clarification_requested"})
}

// HandleEscalate escalates a review entry.
func (h *ReviewHandler) HandleEscalate(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}

	reviewerID := extractReviewerID(c)
	if reviewerID == uuid.Nil {
		return
	}

	var req EscalateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.queue.Escalate(c.Request.Context(), entryID, reviewerID, req.Notes); err != nil {
		h.logger.Error("failed to escalate", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to escalate"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "escalated"})
}

// extractReviewerID reads the reviewer UUID from the X-User-ID header.
// On failure it writes an error response and returns uuid.Nil.
func extractReviewerID(c *gin.Context) uuid.UUID {
	header := c.GetHeader("X-User-ID")
	if header == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing X-User-ID header"})
		return uuid.Nil
	}
	id, err := uuid.Parse(header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-User-ID header"})
		return uuid.Nil
	}
	return id
}
