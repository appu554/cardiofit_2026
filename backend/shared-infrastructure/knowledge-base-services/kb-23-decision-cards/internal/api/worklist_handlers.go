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

// Gap 18 Clinician Worklist — Sprint 1 Core Engine.
//
// Sprint 2 deferrals (documented per review):
//   - Redis cache + 5-min TTL (currently inline fetch — acceptable for 500-patient pilot)
//   - WebSocket push for SAFETY events (Gap 15 notification covers this for now)
//   - WhatsApp summary generator for India GP + ASHA
//   - Shift handover generator for aged care nurses (ISBAR format)
//   - PMS integration for Australian GPs (Best Practice / Medical Director)
//   - Facility aggregation service for aged care managers
//   - Kafka event consumer for cache invalidation
//   - Offline capability for ASHA workers
//   - Full YAML config loading (persona configs hardcoded matching YAML values)

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

	// Select persona config based on clinician role.
	persona := personaConfigForRole(role)

	// Sort and tier
	view := services.SortAndTierWorklist(allItems, persona.MaxItems)

	// Apply persona filter
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

	// Gap 15 loop closure: when a worklist action resolves or acknowledges
	// an item, update the underlying escalation state so the 30-minute
	// timeout cancels and T2/T3 timestamps are recorded.
	if req.ActionCode == "ACKNOWLEDGE" || result.UpdatedItem.ResolutionState == models.ResolutionResolved {
		// Find and update pending escalation for this patient
		var pendingEsc models.EscalationEvent
		if err := s.db.DB.Where("patient_id = ? AND current_state IN (?, ?)",
			req.PatientID, string(models.StatePending), string(models.StateDelivered)).
			Order("created_at DESC").First(&pendingEsc).Error; err == nil {
			now := time.Now()
			pendingEsc.AcknowledgedAt = &now
			pendingEsc.AcknowledgedBy = req.ClinicianID
			pendingEsc.CurrentState = string(models.StateAcknowledged)
			s.db.DB.Save(&pendingEsc)
			s.log.Info("worklist→escalation loop closed: acknowledged",
				zap.String("patient_id", req.PatientID),
				zap.String("escalation_id", pendingEsc.ID.String()))
		}
	}
	if result.UpdatedItem.ResolutionState == models.ResolutionInProgress {
		// Record T3 (action taken) on the underlying escalation
		var actedEsc models.EscalationEvent
		if err := s.db.DB.Where("patient_id = ? AND current_state = ?",
			req.PatientID, string(models.StateAcknowledged)).
			Order("created_at DESC").First(&actedEsc).Error; err == nil {
			now := time.Now()
			actedEsc.ActedAt = &now
			actedEsc.ActionType = req.ActionCode
			actedEsc.ActionDetail = req.Notes
			actedEsc.CurrentState = string(models.StateActed)
			s.db.DB.Save(&actedEsc)
			s.log.Info("worklist→escalation loop closed: action recorded",
				zap.String("patient_id", req.PatientID),
				zap.String("action", req.ActionCode))
		}
	}

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

// personaConfigForRole returns the persona configuration matching the
// clinician's role. Values match persona_definitions.yaml. In Sprint 2,
// this will load from YAML at startup; for Sprint 1, hardcoded to ensure
// persona differentiation works through the API.
func personaConfigForRole(role string) services.PersonaConfig {
	switch role {
	case "HCF_CARE_MANAGER":
		return services.PersonaConfig{
			MaxItems:      15,
			Scope:         "ASSIGNED_PANEL",
			Actions:       []string{"CALL_PATIENT", "SCHEDULE_CLINIC", "ESCALATE_TO_GP", "DEFER", "ACKNOWLEDGE"},
			PrimaryAction: "CALL_PATIENT",
		}
	case "AGED_CARE_NURSE":
		return services.PersonaConfig{
			MaxItems:      20,
			Scope:         "FACILITY",
			Actions:       []string{"RECHECK_VITALS", "CALL_GP", "MEDICATION_HOLD", "ACKNOWLEDGE", "HANDOVER_NOTE"},
			PrimaryAction: "RECHECK_VITALS",
		}
	case "AUSTRALIA_GP":
		return services.PersonaConfig{
			MaxItems:      25,
			Scope:         "ASSIGNED_PANEL",
			Actions:       []string{"MEDICATION_REVIEW", "SCHEDULE_APPOINTMENT", "TELEHEALTH", "REFERRAL", "ACKNOWLEDGE"},
			PrimaryAction: "MEDICATION_REVIEW",
		}
	case "INDIA_GP":
		return services.PersonaConfig{
			MaxItems:      15,
			Scope:         "ASSIGNED_PANEL",
			Actions:       []string{"CALL_PATIENT", "TELECONSULT", "ASHA_OUTREACH", "PRESCRIPTION_REVIEW", "DEFER"},
			PrimaryAction: "CALL_PATIENT",
		}
	case "ASHA_WORKER":
		return services.PersonaConfig{
			MaxItems:      10,
			Scope:         "VILLAGE",
			Actions:       []string{"VISIT_TODAY", "VISIT_TOMORROW", "CALL_ANM", "RECORD_VITALS"},
			PrimaryAction: "VISIT_TODAY",
			Language:       "hi-IN",
		}
	default:
		return services.PersonaConfig{
			MaxItems:      20,
			Scope:         "ASSIGNED_PANEL",
			Actions:       []string{"ACKNOWLEDGE", "CALL_PATIENT", "DEFER", "DISMISS"},
			PrimaryAction: "CALL_PATIENT",
		}
	}
}
