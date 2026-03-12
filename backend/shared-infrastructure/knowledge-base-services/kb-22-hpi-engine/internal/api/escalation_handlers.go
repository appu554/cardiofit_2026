package api

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-22-hpi-engine/internal/models"
	"kb-22-hpi-engine/internal/services"
)

// ---------------------------------------------------------------------------
// BAY-10: /v1/session/escalate — SCE force-escalation webhook
// ---------------------------------------------------------------------------

// EscalationRequest is the request body for POST /v1/session/escalate.
type EscalationRequest struct {
	SessionID    uuid.UUID   `json:"session_id" binding:"required"`
	ReasonCode   string      `json:"reason_code" binding:"required"`
	Evidence     models.JSONB `json:"evidence_snapshot"`
	UrgencyLevel string      `json:"urgency_level" binding:"required"`
}

func (s *Server) escalateSessionHandler(c *gin.Context) {
	var req EscalationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// Validate urgency level
	switch req.UrgencyLevel {
	case "IMMEDIATE", "URGENT", "ROUTINE":
		// valid
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":          "invalid urgency_level",
			"urgency_level":  req.UrgencyLevel,
			"allowed_values": []string{"IMMEDIATE", "URGENT", "ROUTINE"},
		})
		return
	}

	// Load the session
	session, err := s.SessionService.GetSession(c.Request.Context(), req.SessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found", "session_id": req.SessionID.String()})
		return
	}

	// Only ACTIVE or SUSPENDED sessions can be escalated
	if session.Status != models.StatusActive && session.Status != models.StatusSuspended {
		c.JSON(http.StatusConflict, gin.H{
			"error":      "session cannot be escalated in current state",
			"status":     session.Status,
			"session_id": req.SessionID.String(),
		})
		return
	}

	// Transition to SAFETY_ESCALATED
	if err := s.SessionService.EscalateSession(c.Request.Context(), req.SessionID); err != nil {
		s.Log.Error("BAY-10: failed to escalate session",
			zap.String("session_id", req.SessionID.String()),
			zap.String("reason_code", req.ReasonCode),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to escalate session", "details": err.Error()})
		return
	}

	s.Log.Warn("BAY-10: session force-escalated via SCE webhook",
		zap.String("session_id", req.SessionID.String()),
		zap.String("reason_code", req.ReasonCode),
		zap.String("urgency_level", req.UrgencyLevel),
	)

	c.JSON(http.StatusOK, gin.H{
		"session_id":    req.SessionID,
		"status":        models.StatusSafetyEscalated,
		"reason_code":   req.ReasonCode,
		"urgency_level": req.UrgencyLevel,
		"message":       "session escalated successfully",
	})
}

// ---------------------------------------------------------------------------
// B01: /v1/session/multi-init — Multi-node session initialization
// ---------------------------------------------------------------------------

// MultiInitRequest is the request body for POST /v1/session/multi-init.
type MultiInitRequest struct {
	PatientID uuid.UUID `json:"patient_id" binding:"required"`
	NodeIDs   []string  `json:"node_ids" binding:"required"`
}

// MultiInitResponse contains the linked sessions created for multi-complaint patients.
type MultiInitResponse struct {
	LinkedGroupID uuid.UUID              `json:"linked_group_id"`
	Sessions      []MultiInitSessionInfo `json:"sessions"`
}

type MultiInitSessionInfo struct {
	SessionID uuid.UUID `json:"session_id"`
	NodeID    string    `json:"node_id"`
	Status    string    `json:"status"`
}

func (s *Server) multiInitHandler(c *gin.Context) {
	var req MultiInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	if len(req.NodeIDs) < 2 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "multi-init requires at least 2 node_ids"})
		return
	}
	if len(req.NodeIDs) > 5 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "multi-init allows at most 5 node_ids"})
		return
	}

	// Validate all nodes exist
	for i, nodeID := range req.NodeIDs {
		if node := s.NodeLoader.Get(nodeID); node == nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "unknown node_id",
				"index":   i,
				"node_id": nodeID,
			})
			return
		}
	}

	// Check for duplicates
	seen := make(map[string]bool, len(req.NodeIDs))
	for _, nodeID := range req.NodeIDs {
		if seen[nodeID] {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "duplicate node_id",
				"node_id": nodeID,
			})
			return
		}
		seen[nodeID] = true
	}

	linkedGroupID := uuid.New()
	sessions := make([]MultiInitSessionInfo, 0, len(req.NodeIDs))

	for _, nodeID := range req.NodeIDs {
		createReq := models.CreateSessionRequest{
			PatientID: req.PatientID,
			NodeID:    nodeID,
		}

		session, err := s.SessionService.CreateSession(c.Request.Context(), createReq)
		if err != nil {
			s.Log.Error("B01: failed to create session in multi-init",
				zap.String("node_id", nodeID),
				zap.String("patient_id", req.PatientID.String()),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to create session",
				"node_id": nodeID,
				"details": err.Error(),
			})
			return
		}

		sessions = append(sessions, MultiInitSessionInfo{
			SessionID: session.SessionID,
			NodeID:    nodeID,
			Status:    string(session.Status),
		})
	}

	s.Log.Info("B01: multi-node session group created",
		zap.String("linked_group_id", linkedGroupID.String()),
		zap.String("patient_id", req.PatientID.String()),
		zap.Int("node_count", len(req.NodeIDs)),
	)

	c.JSON(http.StatusCreated, MultiInitResponse{
		LinkedGroupID: linkedGroupID,
		Sessions:      sessions,
	})
}

// ---------------------------------------------------------------------------
// A01/BAY-10: /v1/node/validate — Pre-deployment YAML validation
// ---------------------------------------------------------------------------

// NodeValidateRequest is the request body for POST /v1/node/validate.
type NodeValidateRequest struct {
	NodeYAML string `json:"node_yaml" binding:"required"`
}

// ValidationCheck represents a single validation check result.
type ValidationCheck struct {
	Check   string `json:"check"`
	Passed  bool   `json:"passed"`
	Details string `json:"details,omitempty"`
}

func (s *Server) validateNodeHandler(c *gin.Context) {
	var req NodeValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	var node models.NodeDefinition
	if err := yaml.Unmarshal([]byte(req.NodeYAML), &node); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"checks": []ValidationCheck{{Check: "yaml_parse", Passed: false, Details: err.Error()}},
		})
		return
	}

	checks := runNodeValidation(&node)
	allPassed := true
	for _, check := range checks {
		if !check.Passed {
			allPassed = false
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   allPassed,
		"node_id": node.NodeID,
		"version": node.Version,
		"checks":  checks,
	})
}

// runNodeValidation performs comprehensive validation of a NodeDefinition.
func runNodeValidation(node *models.NodeDefinition) []ValidationCheck {
	var checks []ValidationCheck

	// 1. Schema completeness
	checks = append(checks, checkSchemaCompleteness(node))

	// 2. LR bounds check
	checks = append(checks, checkLRBounds(node))

	// 3. Red flag presence
	checks = append(checks, checkRedFlagPresence(node))

	// 4. Differential prior sum
	checks = append(checks, checkPriorSum(node))

	// 5. Question reference integrity
	checks = append(checks, checkQuestionReferenceIntegrity(node))

	// 6. Contradiction pair validity
	checks = append(checks, checkContradictionPairs(node))

	return checks
}

func checkSchemaCompleteness(node *models.NodeDefinition) ValidationCheck {
	var missing []string
	if node.NodeID == "" {
		missing = append(missing, "node_id")
	}
	if node.Version == "" {
		missing = append(missing, "version")
	}
	if len(node.Differentials) == 0 {
		missing = append(missing, "differentials")
	}
	if len(node.Questions) == 0 {
		missing = append(missing, "questions")
	}
	if len(missing) > 0 {
		return ValidationCheck{
			Check:   "schema_completeness",
			Passed:  false,
			Details: "missing required fields: " + strings.Join(missing, ", "),
		}
	}
	return ValidationCheck{Check: "schema_completeness", Passed: true}
}

func checkLRBounds(node *models.NodeDefinition) ValidationCheck {
	var violations []string
	for _, q := range node.Questions {
		for diffID, lr := range q.LRPositive {
			if lr < 0.01 {
				violations = append(violations, fmt.Sprintf("%s LR+[%s]=%.4f < 0.01", q.ID, diffID, lr))
			}
			if lr > 100.0 {
				violations = append(violations, fmt.Sprintf("%s LR+[%s]=%.1f > 100", q.ID, diffID, lr))
			}
		}
		for diffID, lr := range q.LRNegative {
			if lr < 0.01 {
				violations = append(violations, fmt.Sprintf("%s LR-[%s]=%.4f < 0.01", q.ID, diffID, lr))
			}
			if lr > 100.0 {
				violations = append(violations, fmt.Sprintf("%s LR-[%s]=%.1f > 100", q.ID, diffID, lr))
			}
		}
	}
	if len(violations) > 0 {
		return ValidationCheck{
			Check:   "lr_bounds",
			Passed:  false,
			Details: fmt.Sprintf("%d violations: %s", len(violations), strings.Join(violations[:min(3, len(violations))], "; ")),
		}
	}
	return ValidationCheck{Check: "lr_bounds", Passed: true}
}

func checkRedFlagPresence(node *models.NodeDefinition) ValidationCheck {
	for _, trigger := range node.SafetyTriggers {
		if trigger.Severity == "IMMEDIATE" {
			return ValidationCheck{Check: "red_flag_presence", Passed: true}
		}
	}
	return ValidationCheck{
		Check:   "red_flag_presence",
		Passed:  false,
		Details: "no safety trigger with severity=IMMEDIATE found; at least one red flag required",
	}
}

func checkPriorSum(node *models.NodeDefinition) ValidationCheck {
	// Priors is a map[stratum]float64 per differential.
	// Validate each stratum's prior sum independently.
	// Collect all strata referenced across differentials.
	stratumSums := make(map[string]float64)
	for _, d := range node.Differentials {
		for stratum, prior := range d.Priors {
			stratumSums[stratum] += prior
		}
	}

	if len(stratumSums) == 0 {
		return ValidationCheck{
			Check:   "prior_sum",
			Passed:  false,
			Details: "no priors defined in any differential",
		}
	}

	expectedSum := 1.0
	tolerance := 0.05
	if node.OtherBucketEnabled {
		expectedSum = 1.0 - node.OtherBucketPrior
		if node.OtherBucketPrior == 0 {
			expectedSum = 0.85 // default
		}
	}

	var violations []string
	for stratum, sum := range stratumSums {
		if math.Abs(sum-expectedSum) > tolerance {
			violations = append(violations, fmt.Sprintf(
				"stratum %s: sum=%.3f (expected ~%.2f)", stratum, sum, expectedSum,
			))
		}
	}

	if len(violations) > 0 {
		return ValidationCheck{
			Check:   "prior_sum",
			Passed:  false,
			Details: strings.Join(violations, "; "),
		}
	}

	// Report using first stratum as representative
	for stratum, sum := range stratumSums {
		return ValidationCheck{
			Check:   "prior_sum",
			Passed:  true,
			Details: fmt.Sprintf("stratum %s: sum=%.3f (expected ~%.2f)", stratum, sum, expectedSum),
		}
	}
	return ValidationCheck{Check: "prior_sum", Passed: true}
}

func checkQuestionReferenceIntegrity(node *models.NodeDefinition) ValidationCheck {
	questionIDs := make(map[string]bool, len(node.Questions))
	for _, q := range node.Questions {
		questionIDs[q.ID] = true
	}

	var missing []string
	for _, trigger := range node.SafetyTriggers {
		// Parse condition to extract question IDs
		atoms := extractAtoms(trigger.Condition)
		for _, atom := range atoms {
			parts := strings.SplitN(atom, "=", 2)
			if len(parts) == 2 {
				qid := strings.TrimSpace(parts[0])
				// Skip CM atoms (G8)
				if strings.HasPrefix(qid, "CM_") {
					continue
				}
				if !questionIDs[qid] {
					missing = append(missing, fmt.Sprintf("trigger %s references %s", trigger.ID, qid))
				}
			}
		}
	}

	if len(missing) > 0 {
		return ValidationCheck{
			Check:   "question_reference_integrity",
			Passed:  false,
			Details: strings.Join(missing, "; "),
		}
	}
	return ValidationCheck{Check: "question_reference_integrity", Passed: true}
}

func checkContradictionPairs(node *models.NodeDefinition) ValidationCheck {
	if len(node.ContradictionPairs) == 0 {
		return ValidationCheck{Check: "contradiction_pairs", Passed: true, Details: "no pairs defined"}
	}

	questionIDs := make(map[string]bool, len(node.Questions))
	for _, q := range node.Questions {
		questionIDs[q.ID] = true
	}

	var invalid []string
	for _, pair := range node.ContradictionPairs {
		if !questionIDs[pair.QuestionA] {
			invalid = append(invalid, fmt.Sprintf("pair %s: %s not found", pair.ID, pair.QuestionA))
		}
		if !questionIDs[pair.QuestionB] {
			invalid = append(invalid, fmt.Sprintf("pair %s: %s not found", pair.ID, pair.QuestionB))
		}
	}

	if len(invalid) > 0 {
		return ValidationCheck{
			Check:   "contradiction_pairs",
			Passed:  false,
			Details: strings.Join(invalid, "; "),
		}
	}
	return ValidationCheck{Check: "contradiction_pairs", Passed: true}
}

// extractAtoms parses a boolean condition into individual atoms.
func extractAtoms(condition string) []string {
	// Remove AND/OR operators and split
	condition = strings.ReplaceAll(condition, " AND ", "|")
	condition = strings.ReplaceAll(condition, " OR ", "|")
	parts := strings.Split(condition, "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// E01: Expert panel calibration handlers
// ---------------------------------------------------------------------------

func (s *Server) expertReviewHandler(c *gin.Context) {
	var review services.ExpertPanelReview
	if err := c.ShouldBindJSON(&review); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	event, err := s.ExpertPanelService.SubmitReview(c.Request.Context(), review)
	if err != nil {
		s.Log.Error("E01: expert review submission failed",
			zap.String("node_id", review.NodeID),
			zap.Error(err),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "expert review recorded",
		"event_id": event.EventID,
		"node_id":  review.NodeID,
	})
}

func (s *Server) expertReviewHistoryHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}

	history, err := s.ExpertPanelService.GetReviewHistory(c.Request.Context(), nodeID)
	if err != nil {
		s.Log.Error("E01: failed to get review history",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"node_id": nodeID,
		"reviews": history,
		"count":   len(history),
	})
}
