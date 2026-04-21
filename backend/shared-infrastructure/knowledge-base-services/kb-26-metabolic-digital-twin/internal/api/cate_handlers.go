package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-26-metabolic-digital-twin/internal/models"
)

// GET /api/v1/kb26/cate/:id — read a single CATEEstimate by ID.
// Returns the persisted estimate produced by a prior POST /cate/estimate call.
// Sprint 1 note: POST /cate/estimate itself ships as a 501 stub below, so until
// Sprint 2 wires cross-service data fetch, this endpoint will only find rows
// populated by tests or manual inserts.
func (s *Server) getCATEEstimate(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid: " + idStr})
		return
	}
	if s.db == nil || s.db.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var est models.CATEEstimate
	if err := s.db.DB.First(&est, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CATE estimate not found"})
		return
	}
	c.JSON(http.StatusOK, est)
}

// GET /api/v1/kb26/cate/calibration/summary/:cohortId — return the calibration
// summary for a given cohort × intervention × horizon. Query params:
//   - intervention (required): intervention ID
//   - horizon (optional, default 30): horizon in days
//
// Joins Gap 22 CATEEstimates with Gap 21 AttributionVerdicts via the calibration
// monitor (Task 6) and returns {MatchedPairs, MeanAbsDiff, Status, AlarmTriggered}.
func (s *Server) getCalibrationSummary(c *gin.Context) {
	cohort := strings.TrimSpace(c.Param("cohortId"))
	if cohort == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cohort_id is required"})
		return
	}
	intervention := strings.TrimSpace(c.Query("intervention"))
	if intervention == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "intervention query parameter is required"})
		return
	}
	horizon := 30
	if raw := c.Query("horizon"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 3650 {
			horizon = n
		}
	}
	if s.cateMonitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Gap 22 CATE services not wired (monitor nil)"})
		return
	}
	sum, err := s.cateMonitor.ComputeCalibrationSummary(cohort, intervention, horizon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sum)
}

// POST /api/v1/kb26/cate/estimate — Sprint 1 stub. The real implementation
// requires fetching a training cohort from KB-23's consolidated_alert_records +
// outcome_records tables, which is a cross-service data fetch deferred to
// Sprint 2 (Python KB-28 service behind the same CATEEstimate contract).
//
// Returns 501 Not Implemented with a structured body identifying the deferral.
// Integration tests that exercise the baseline learner should call
// services.EstimateFromCohort directly (see baseline_cate_learner_test.go).
func (s *Server) postCATEEstimate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":          "POST /cate/estimate is a Sprint 1 stub",
		"sprint_1_scope": "CATE contract + baseline learner + overlap + calibration monitor are functional in-process; HTTP-path estimation deferred to Sprint 2.",
		"sprint_2_plan":  "KB-28 Python service or KB-23 gRPC client reads training rows from consolidated_alert_records + outcome_records and invokes services.EstimateFromCohort.",
		"workaround":     "For Sprint 1 testing, call services.EstimateFromCohort(rows, patientID, features, band) directly.",
	})
}
