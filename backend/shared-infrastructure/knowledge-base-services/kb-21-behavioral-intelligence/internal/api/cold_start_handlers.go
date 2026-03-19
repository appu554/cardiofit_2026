package api

import (
	"net/http"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// submitIntakeProfile records a patient's intake signals for cold-start phenotype assignment.
// POST /api/v1/patient/:patient_id/intake-profile
func (s *Server) submitIntakeProfile(c *gin.Context) {
	if s.coldStartEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Cold-start profiling not enabled", "FEATURE_DISABLED", nil)
		return
	}

	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patient_id is required", "INVALID_REQUEST", nil)
		return
	}

	var body struct {
		AgeBand              string  `json:"age_band"`
		EducationLevel       string  `json:"education_level"`
		SmartphoneLiteracy   string  `json:"smartphone_literacy"`
		SelfEfficacy         float64 `json:"self_efficacy"`
		FamilyStructure      string  `json:"family_structure"`
		EmploymentStatus     string  `json:"employment_status"`
		PriorProgramSuccess  *bool   `json:"prior_program_success"`
		FirstResponseLatency int64   `json:"first_response_latency_ms"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", "VALIDATION_ERROR",
			map[string]interface{}{"error": err.Error()})
		return
	}

	intake := models.IntakeProfile{
		PatientID:            patientID,
		AgeBand:              body.AgeBand,
		EducationLevel:       body.EducationLevel,
		SmartphoneLiteracy:   body.SmartphoneLiteracy,
		SelfEfficacy:         body.SelfEfficacy,
		FamilyStructure:      body.FamilyStructure,
		EmploymentStatus:     body.EmploymentStatus,
		PriorProgramSuccess:  body.PriorProgramSuccess,
		FirstResponseLatency: body.FirstResponseLatency,
		CollectedAt:          time.Now().UTC(),
	}

	// Upsert
	if err := s.db.DB.Where("patient_id = ?", patientID).
		Assign(intake).FirstOrCreate(&intake).Error; err != nil {
		s.logger.Error("intake profile save failed", zap.Error(err))
		sendError(c, http.StatusInternalServerError, "Failed to save intake profile", "SAVE_FAILED", nil)
		return
	}

	// Assign phenotype
	phenotype := s.coldStartEngine.AssignPhenotype(intake)

	sendSuccess(c, gin.H{
		"patient_id": patientID,
		"phenotype":  phenotype,
		"intake":     intake,
	}, nil)
}

// getColdStartPhenotype returns the assigned cold-start phenotype for a patient.
// GET /api/v1/patient/:patient_id/cold-start-phenotype
func (s *Server) getColdStartPhenotype(c *gin.Context) {
	if s.coldStartEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Cold-start profiling not enabled", "FEATURE_DISABLED", nil)
		return
	}

	patientID := c.Param("patient_id")
	phenotype, err := s.coldStartEngine.GetOrAssignPhenotype(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get phenotype", "QUERY_FAILED", nil)
		return
	}

	priors := s.coldStartEngine.GetPhenotypePriors(phenotype)

	sendSuccess(c, gin.H{
		"patient_id": patientID,
		"phenotype":  phenotype,
		"priors":     priors,
	}, nil)
}
