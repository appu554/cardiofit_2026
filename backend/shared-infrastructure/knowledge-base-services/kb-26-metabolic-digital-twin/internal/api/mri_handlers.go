package api

import (
	"net/http"
	"strconv"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// getMRI returns the current MRI score + domain sub-scores for a patient.
// GET /api/v1/kb26/mri/:patientId
func (s *Server) getMRI(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found", "TWIN_NOT_FOUND", nil)
		return
	}

	input := services.TwinToMRIScorerInput(twin)
	history := s.mriScorer.GetHistoryScores(patientID)
	result := s.mriScorer.ComputeMRI(input, history)

	// Persist score
	persisted, err := s.mriScorer.PersistScore(patientID, result, &twin.ID)
	if err != nil {
		s.logger.Error("failed to persist MRI score", zap.Error(err))
	}

	// Phase 6 P6-3: compute the decomposed domain trajectory automatically
	// after every MRI persistence. Best-effort — failures log a warning and
	// continue (the MRI response is the source of truth; trajectory is a
	// derived analytic that downstream consumers receive via the Kafka
	// publisher invoked inside TrajectoryEngine.Compute). Closes the
	// Module 13 empty velocity gap by ensuring trajectory runs on every
	// natural MRI recomputation, not just on explicit GET requests.
	s.computeAndPersistTrajectory(patientID)

	resp := gin.H{
		"score":      result.Score,
		"category":   result.Category,
		"trend":      result.Trend,
		"top_driver": result.TopDriver,
		"domains":    result.Domains,
	}
	if persisted != nil {
		resp["id"] = persisted.ID.String()
		resp["computed_at"] = persisted.ComputedAt
	}

	sendSuccess(c, resp, map[string]interface{}{
		"patient_id": patientID.String(),
	})
}

// getMRIHistory returns the MRI time-series for sparkline rendering.
// GET /api/v1/kb26/mri/:patientId/history?limit=12
func (s *Server) getMRIHistory(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	limit := 12
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	scores, err := s.mriScorer.GetHistory(patientID, limit)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "failed to retrieve MRI history", "MRI_HISTORY_ERROR", nil)
		return
	}

	sendSuccess(c, scores, map[string]interface{}{
		"patient_id": patientID.String(),
		"count":      len(scores),
	})
}

// getMRIDecomposition returns per-signal z-scores showing what's driving the MRI.
// GET /api/v1/kb26/mri/:patientId/decomposition
func (s *Server) getMRIDecomposition(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found", "TWIN_NOT_FOUND", nil)
		return
	}

	input := services.TwinToMRIScorerInput(twin)
	result := s.mriScorer.ComputeMRI(input, nil)

	sendSuccess(c, gin.H{
		"score":      result.Score,
		"category":   result.Category,
		"top_driver": result.TopDriver,
		"domains":    result.Domains,
	}, map[string]interface{}{
		"patient_id": patientID.String(),
	})
}

// simulateMRI projects MRI change under a proposed intervention.
// POST /api/v1/kb26/mri/simulate
func (s *Server) simulateMRI(c *gin.Context) {
	var req struct {
		PatientID    string              `json:"patient_id" binding:"required"`
		Intervention models.Intervention `json:"intervention" binding:"required"`
		Days         int                 `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request", "INVALID_REQUEST", nil)
		return
	}
	if req.Days <= 0 {
		req.Days = 30
	} else if req.Days > 365 {
		req.Days = 365
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found", "NOT_FOUND", nil)
		return
	}

	// Current MRI
	currentInput := services.TwinToMRIScorerInput(twin)
	currentResult := s.mriScorer.ComputeMRI(currentInput, nil)

	// Run simulation
	initial := services.TwinToSimState(twin)
	projected := services.RunSimulation(initial, req.Intervention, req.Days)

	// Compute projected MRI from final simulation state
	if len(projected) > 0 {
		final := projected[len(projected)-1]
		projectedInput := biomarkersToMRIInput(final, currentInput)
		projectedResult := s.mriScorer.ComputeMRI(projectedInput, nil)

		sendSuccess(c, gin.H{
			"current_mri":   currentResult.Score,
			"projected_mri": projectedResult.Score,
			"projection_days": req.Days,
			"domain_changes": gin.H{
				"glucose":          projectedResult.Domains[0].Scaled - currentResult.Domains[0].Scaled,
				"body_composition": projectedResult.Domains[1].Scaled - currentResult.Domains[1].Scaled,
				"cardiovascular":   projectedResult.Domains[2].Scaled - currentResult.Domains[2].Scaled,
				"behavioral":       projectedResult.Domains[3].Scaled - currentResult.Domains[3].Scaled,
			},
			"confidence": s.calibrator.GetPatientConfidence(patientID),
		}, nil)
		return
	}

	sendError(c, http.StatusInternalServerError, "simulation produced no results", "SIM_ERROR", nil)
}

// biomarkersToMRIInput maps projected biomarkers to MRIScorerInput.
// Behavioral signals (steps, protein, sleep) are carried from the current input
// since the ODE doesn't project those directly.
func biomarkersToMRIInput(projected models.ProjectedState, current services.MRIScorerInput) services.MRIScorerInput {
	return services.MRIScorerInput{
		FBG:         projected.FBG,
		PPBG:        projected.PPBG,
		HbA1cTrend:  current.HbA1cTrend,
		WaistCm:     projected.WaistCm,
		WeightTrend: current.WeightTrend,
		MuscleSTS:   current.MuscleSTS,
		SBP:         projected.SBP,
		SBPTrend:    current.SBPTrend,
		BPDipping:   current.BPDipping,
		Steps:       current.Steps,
		ProteinGKg:  current.ProteinGKg,
		SleepScore:  current.SleepScore,
		Sex:         current.Sex,
		BMI:         current.BMI,
	}
}
