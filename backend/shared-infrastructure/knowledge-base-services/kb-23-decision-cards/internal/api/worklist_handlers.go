package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
	"kb-23-decision-cards/internal/services"
)

// GET /api/v1/worklist?clinician_id=X&role=HCF_CARE_MANAGER&patient_ids=P1,P2,P3
// Returns the PAI-sorted, persona-filtered worklist.
func (s *Server) getWorklist(c *gin.Context) {
	clinicianID := c.Query("clinician_id")
	role := c.Query("role")
	patientIDsRaw := c.Query("patient_ids")

	if clinicianID == "" || role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clinician_id and role required"})
		return
	}

	// Parse patient IDs (comma-separated)
	var assignedPatientIDs []string
	if patientIDsRaw != "" {
		assignedPatientIDs = strings.Split(patientIDsRaw, ",")
	}

	// For Sprint 1: build mock clinical states from the patient IDs.
	// In production, this would fetch PAI from KB-26, cards + escalations
	// from local DB, and profiles from KB-20.
	// For now, query local cards and escalations for each patient.
	var allItems []models.WorklistItem

	for _, patientID := range assignedPatientIDs {
		// Query active cards for this patient
		var cards []models.DecisionCard
		s.db.DB.Where("patient_id = ? AND status = ?", patientID, "ACTIVE").
			Order("created_at DESC").Limit(5).Find(&cards)

		// Query active escalations for this patient
		var escalations []models.EscalationEvent
		s.db.DB.Where("patient_id = ? AND current_state IN (?, ?)",
			patientID, "PENDING", "DELIVERED").
			Order("created_at DESC").Limit(3).Find(&escalations)

		// Build clinical state from available data
		state := services.PatientClinicalState{
			PatientID:   patientID,
			PatientName: patientID, // placeholder — would come from KB-20
		}

		// Map escalations
		for _, esc := range escalations {
			state.ActiveEscalations = append(state.ActiveEscalations, services.EscalationInfo{
				Tier:            esc.EscalationTier,
				State:           esc.CurrentState,
				PrimaryReason:   esc.PrimaryReason,
				SuggestedAction: esc.SuggestedAction,
			})
		}

		// Map cards
		for _, card := range cards {
			state.ActiveCards = append(state.ActiveCards, services.CardInfo{
				CardID:           card.CardID.String(),
				ClinicianSummary: card.ClinicianSummary,
				MCUGate:          string(card.MCUGate),
			})
		}

		// Aggregate to one worklist item
		if item := services.AggregateWorklistItem(state); item != nil {
			allItems = append(allItems, *item)
		}
	}

	// Sort and tier
	maxItems := 20 // default; persona-specific in production
	view := services.SortAndTierWorklist(allItems, maxItems)

	// Apply persona filter
	persona := services.PersonaConfig{
		MaxItems:      maxItems,
		Scope:         "ASSIGNED_PANEL",
		Actions:       []string{"ACKNOWLEDGE", "CALL_PATIENT", "DEFER", "DISMISS"},
		PrimaryAction: "CALL_PATIENT",
	}
	view.Items = services.ApplyPersonaFilter(view.Items, assignedPatientIDs, persona)
	view.ClinicianID = clinicianID
	view.PersonaType = role
	view.TotalCount = len(view.Items)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": view})
}

// POST /api/v1/worklist/action
// Executes a one-tap action on a worklist item.
func (s *Server) handleWorklistAction(c *gin.Context) {
	var req models.WorklistActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a minimal WorklistItem for resolution
	item := &models.WorklistItem{
		PatientID:       req.PatientID,
		ResolutionState: models.ResolutionPending,
	}

	result := services.HandleWorklistAction(item, req)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	// If feedback was generated (DISMISS), persist it
	// (in Sprint 2, this would write to a feedback table)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"resolution": result.UpdatedItem.ResolutionState,
		"feedback":   result.Feedback,
	})
}

// POST /api/v1/worklist/feedback
// Records clinician feedback for trust calibration.
func (s *Server) recordWorklistFeedback(c *gin.Context) {
	var feedback models.WorklistFeedback
	if err := c.ShouldBindJSON(&feedback); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	feedback.SubmittedAt = time.Now()

	// In Sprint 2, persist to feedback table for closed-loop learning
	s.log.Info("worklist feedback recorded",
		zap.String("patient_id", feedback.PatientID),
		zap.String("clinician_id", feedback.ClinicianID),
		zap.String("type", feedback.FeedbackType))

	c.JSON(http.StatusOK, gin.H{"success": true})
}
