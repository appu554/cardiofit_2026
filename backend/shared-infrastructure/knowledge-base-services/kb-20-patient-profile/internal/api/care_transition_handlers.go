package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/models"
	"kb-patient-profile/internal/services"
)

// registerDischarge handles POST /api/v1/patient/:id/discharge.
// Accepts a manual (or FHIR-forwarded) discharge registration, validates
// via DischargeDetector, checks for duplicates, persists, and returns 201.
func (s *Server) registerDischarge(c *gin.Context) {
	patientID := c.Param("id")

	var req struct {
		DischargeDate    string `json:"discharge_date" binding:"required"`
		FacilityName     string `json:"facility_name"`
		FacilityType     string `json:"facility_type"`
		PrimaryDiagnosis string `json:"primary_diagnosis"`
		LengthOfStayDays int    `json:"length_of_stay_days"`
		Disposition      string `json:"disposition"`
		Source           string `json:"source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	dischargeDate, err := time.Parse(time.RFC3339, req.DischargeDate)
	if err != nil {
		// Try date-only format as fallback
		dischargeDate, err = time.Parse("2006-01-02", req.DischargeDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid discharge_date format, expected RFC3339 or YYYY-MM-DD"})
			return
		}
	}

	// Default source to MANUAL if not provided
	source := req.Source
	if source == "" {
		source = models.SourceManual
	}

	input := services.DischargeInput{
		PatientID:        patientID,
		DischargeDate:    dischargeDate,
		Source:           source,
		FacilityName:     req.FacilityName,
		FacilityType:     req.FacilityType,
		PrimaryDiagnosis: req.PrimaryDiagnosis,
		LengthOfStayDays: req.LengthOfStayDays,
		Disposition:      req.Disposition,
	}

	detector := services.NewDischargeDetector()

	transition, err := detector.DetectDischarge(input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	// Check for duplicates against existing active transitions
	var existing models.CareTransition
	if err := s.db.DB.Where("patient_id = ? AND transition_state = ?", patientID, string(models.TransitionActive)).
		Order("discharge_date DESC").First(&existing).Error; err == nil {
		if detector.IsDuplicate(&existing, input) {
			c.JSON(http.StatusConflict, gin.H{
				"error":                   "duplicate discharge detected within 24 hours of existing transition",
				"existing_transition_id":  existing.ID,
				"existing_discharge_date": existing.DischargeDate,
			})
			return
		}
	}

	// Persist
	if err := s.db.DB.Create(transition).Error; err != nil {
		s.logger.Error("failed to persist care transition: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save transition"})
		return
	}

	c.JSON(http.StatusCreated, transition)
}

// getActiveTransition handles GET /api/v1/patient/:id/transition.
// Returns the current active care transition for the patient, or 404.
func (s *Server) getActiveTransition(c *gin.Context) {
	patientID := c.Param("id")

	var transition models.CareTransition
	if err := s.db.DB.Where("patient_id = ? AND transition_state = ?", patientID, string(models.TransitionActive)).
		Order("discharge_date DESC").First(&transition).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active transition found for patient"})
		return
	}

	c.JSON(http.StatusOK, transition)
}

// getTransitionMilestones handles GET /api/v1/patient/:id/transition/milestones.
// Returns the milestone schedule for the patient's active transition.
func (s *Server) getTransitionMilestones(c *gin.Context) {
	patientID := c.Param("id")

	// Find active transition
	var transition models.CareTransition
	if err := s.db.DB.Where("patient_id = ? AND transition_state = ?", patientID, string(models.TransitionActive)).
		Order("discharge_date DESC").First(&transition).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active transition found for patient"})
		return
	}

	// Fetch milestones
	var milestones []models.TransitionMilestone
	if err := s.db.DB.Where("transition_id = ?", transition.ID).
		Order("scheduled_for ASC").Find(&milestones).Error; err != nil {
		s.logger.Error("failed to query transition milestones: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve milestones"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transition_id": transition.ID,
		"patient_id":    patientID,
		"milestones":    milestones,
	})
}
