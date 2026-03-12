package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// Session handlers
// ---------------------------------------------------------------------------

// createSessionHandler handles POST /sessions.
// Binds a CreateSessionRequest, validates the node_id exists, and creates
// a new HPI session via SessionService.
func (s *Server) createSessionHandler(c *gin.Context) {
	var req models.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// Verify the requested node exists in loaded definitions
	if node := s.NodeLoader.Get(req.NodeID); node == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "unknown node_id",
			"node_id": req.NodeID,
		})
		return
	}

	session, err := s.SessionService.CreateSession(c.Request.Context(), req)
	if err != nil {
		s.Log.Error("failed to create session",
			zap.String("node_id", req.NodeID),
			zap.String("patient_id", req.PatientID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// getSessionHandler handles GET /sessions/:id.
// Returns the full session state including current question and top differentials.
func (s *Server) getSessionHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id", "details": err.Error()})
		return
	}

	session, err := s.SessionService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		s.Log.Error("failed to get session", zap.String("session_id", sessionID.String()), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found", "session_id": sessionID.String()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// submitAnswerHandler handles POST /sessions/:id/answers.
// Binds a SubmitAnswerRequest, applies Bayesian update, evaluates safety
// triggers, and returns the next question or completion state.
func (s *Server) submitAnswerHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id", "details": err.Error()})
		return
	}

	var req models.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// Validate answer value is one of the accepted values
	switch models.AnswerValue(req.AnswerValue) {
	case models.AnswerYes, models.AnswerNo, models.AnswerPata:
		// valid
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":          "invalid answer_value",
			"answer_value":   req.AnswerValue,
			"allowed_values": []string{string(models.AnswerYes), string(models.AnswerNo), string(models.AnswerPata)},
		})
		return
	}

	response, err := s.SessionService.SubmitAnswer(c.Request.Context(), sessionID, req)
	if err != nil {
		s.Log.Error("failed to submit answer",
			zap.String("session_id", sessionID.String()),
			zap.String("question_id", req.QuestionID),
			zap.Error(err),
		)
		// Distinguish between conflict (wrong state) and internal errors
		if isConflictError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "session_id": sessionID.String()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit answer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// suspendSessionHandler handles POST /sessions/:id/suspend.
// Transitions a session to SUSPENDED status for later resumption.
func (s *Server) suspendSessionHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id", "details": err.Error()})
		return
	}

	if err := s.SessionService.SuspendSession(c.Request.Context(), sessionID); err != nil {
		s.Log.Error("failed to suspend session", zap.String("session_id", sessionID.String()), zap.Error(err))
		if isConflictError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "session_id": sessionID.String()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to suspend session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session_id": sessionID, "status": models.StatusSuspended})
}

// resumeSessionHandler handles POST /sessions/:id/resume.
// Resumes a suspended session, performing R-04 stratum drift detection.
func (s *Server) resumeSessionHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id", "details": err.Error()})
		return
	}

	response, err := s.SessionService.ResumeSession(c.Request.Context(), sessionID)
	if err != nil {
		s.Log.Error("failed to resume session", zap.String("session_id", sessionID.String()), zap.Error(err))
		if isConflictError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "session_id": sessionID.String()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resume session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// completeSessionHandler handles POST /sessions/:id/complete.
// Finalises the session, writes the DifferentialSnapshot, and publishes
// the HPICompleteEvent to KB-23 and KB-19.
func (s *Server) completeSessionHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id", "details": err.Error()})
		return
	}

	if err := s.SessionService.CompleteSession(c.Request.Context(), sessionID); err != nil {
		s.Log.Error("failed to complete session", zap.String("session_id", sessionID.String()), zap.Error(err))
		if isConflictError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "session_id": sessionID.String()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session completed", "session_id": sessionID.String()})
}

// ---------------------------------------------------------------------------
// Differential & Safety handlers
// ---------------------------------------------------------------------------

// getDifferentialHandler handles GET /sessions/:id/differential.
// Returns the current ranked differential list for an active session.
func (s *Server) getDifferentialHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id", "details": err.Error()})
		return
	}

	differential, err := s.SessionService.GetDifferential(c.Request.Context(), sessionID)
	if err != nil {
		s.Log.Error("failed to get differential", zap.String("session_id", sessionID.String()), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "differential not found", "session_id": sessionID.String()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":    sessionID,
		"differentials": differential,
	})
}

// getSafetyFlagsHandler handles GET /sessions/:id/safety.
// Returns all safety flags raised during the session.
func (s *Server) getSafetyFlagsHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id", "details": err.Error()})
		return
	}

	flags, err := s.SessionService.GetSafetyFlags(c.Request.Context(), sessionID)
	if err != nil {
		s.Log.Error("failed to get safety flags", zap.String("session_id", sessionID.String()), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found", "session_id": sessionID.String()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":   sessionID,
		"safety_flags": flags,
	})
}

// getSnapshotHandler handles GET /snapshots/:session_id.
// Returns the immutable DifferentialSnapshot created on session completion.
func (s *Server) getSnapshotHandler(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id", "details": err.Error()})
		return
	}

	snapshot, err := s.SessionService.GetSnapshot(c.Request.Context(), sessionID)
	if err != nil {
		s.Log.Error("failed to get snapshot", zap.String("session_id", sessionID.String()), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found", "session_id": sessionID.String()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// ---------------------------------------------------------------------------
// Node handlers
// ---------------------------------------------------------------------------

// listNodesHandler handles GET /nodes.
// Returns a summary list of all loaded HPI node definitions.
func (s *Server) listNodesHandler(c *gin.Context) {
	allNodes := s.NodeLoader.All()
	summaries := make([]gin.H, 0, len(allNodes))
	for _, node := range allNodes {
		summaries = append(summaries, gin.H{
			"node_id":        node.NodeID,
			"version":        node.Version,
			"strata":         node.StrataSupported,
			"question_count": len(node.Questions),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": summaries,
		"count": len(summaries),
	})
}

// getNodeHandler handles GET /nodes/:node_id.
// Returns the full node definition or 404 if not loaded.
func (s *Server) getNodeHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	node := s.NodeLoader.Get(nodeID)
	if node == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found", "node_id": nodeID})
		return
	}

	c.JSON(http.StatusOK, node)
}

// ---------------------------------------------------------------------------
// Calibration handlers
// ---------------------------------------------------------------------------

// calibrationFeedbackHandler handles POST /calibration/feedback.
// Receives clinician adjudication for a completed session and records
// concordance for LR recalibration.
func (s *Server) calibrationFeedbackHandler(c *gin.Context) {
	var req models.AdjudicationFeedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	record, err := s.CalibrationManager.SubmitFeedback(c.Request.Context(), req)
	if err != nil {
		s.Log.Error("failed to submit calibration feedback",
			zap.String("snapshot_id", req.SnapshotID.String()),
			zap.String("confirmed_diagnosis", req.ConfirmedDiagnosis),
			zap.Error(err),
		)
		if isConflictError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record feedback", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// calibrationStatusHandler handles GET /calibration/status/:node_id.
// Returns concordance metrics filtered by optional stratum and ckd_substage
// query parameters.
func (s *Server) calibrationStatusHandler(c *gin.Context) {
	nodeID := c.Param("node_id")

	// Verify node exists
	if node := s.NodeLoader.Get(nodeID); node == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found", "node_id": nodeID})
		return
	}

	stratum := c.Query("stratum")
	ckdSubstage := c.Query("ckd_substage")

	status, err := s.CalibrationManager.GetStatus(c.Request.Context(), nodeID, stratum, ckdSubstage)
	if err != nil {
		s.Log.Error("failed to get calibration status",
			zap.String("node_id", nodeID),
			zap.String("stratum", stratum),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get calibration status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// importGoldenHandler handles POST /calibration/import-golden.
// Bulk-imports golden dataset cases for synthetic concordance calculation.
func (s *Server) importGoldenHandler(c *gin.Context) {
	var req models.GoldenDatasetImport
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	if len(req.Cases) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "cases array must not be empty"})
		return
	}

	// Validate all referenced nodes exist
	for i, gc := range req.Cases {
		if node := s.NodeLoader.Get(gc.NodeID); node == nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "unknown node_id in case",
				"index":   i,
				"node_id": gc.NodeID,
			})
			return
		}
	}

	result, err := s.CalibrationManager.ImportGolden(c.Request.Context(), req)
	if err != nil {
		s.Log.Error("failed to import golden dataset",
			zap.Int("case_count", len(req.Cases)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import golden dataset", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isConflictError returns true if the error represents a state-machine conflict
// (e.g. trying to answer on a completed session).
func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Convention: service layer returns errors containing these phrases for
	// invalid state transitions.
	for _, phrase := range []string{
		"invalid state",
		"session is not active",
		"session already completed",
		"session is completed",
		"session is abandoned",
		"already adjudicated",
		"cannot suspend",
		"cannot resume",
	} {
		if containsIgnoreCase(msg, phrase) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	sLen := len(s)
	subLen := len(substr)
	if subLen > sLen {
		return false
	}
	for i := 0; i <= sLen-subLen; i++ {
		match := true
		for j := 0; j < subLen; j++ {
			sc := s[i+j]
			tc := substr[j]
			// ASCII lower
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
