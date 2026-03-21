package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-patient-profile/internal/models"
)

// validatePatientID extracts and validates the patient ID path parameter.
func validatePatientID(c *gin.Context) (string, bool) {
	patientID := c.Param("id")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient id is required"})
		return "", false
	}
	return patientID, true
}

// submitMealSignal handles POST /patient/:id/signals/meal (S4 Meal Log).
func (s *Server) submitMealSignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.MealSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventMealLog, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventMealLog), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventMealLog, "patient_id": patientID})
}

// submitActivitySignal handles POST /patient/:id/signals/activity (S16 Activity).
func (s *Server) submitActivitySignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.ActivitySignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventActivityLog, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventActivityLog), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventActivityLog, "patient_id": patientID})
}

// submitWaistSignal handles POST /patient/:id/signals/waist (S15 Waist).
func (s *Server) submitWaistSignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.WaistSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventWaistMeasurement, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventWaistMeasurement), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventWaistMeasurement, "patient_id": patientID})
}

// submitAdherenceSignal handles POST /patient/:id/signals/adherence (S20 Adherence).
func (s *Server) submitAdherenceSignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.AdherenceSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventAdherenceReport, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventAdherenceReport), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventAdherenceReport, "patient_id": patientID})
}

// submitSymptomSignal handles POST /patient/:id/signals/symptom (S18 Symptom).
func (s *Server) submitSymptomSignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.SymptomSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventSymptomReport, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventSymptomReport), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventSymptomReport, "patient_id": patientID})
}

// submitAdverseEventSignal handles POST /patient/:id/signals/adverse-event (S19 Adverse Event).
func (s *Server) submitAdverseEventSignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.AdverseEventSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventAdverseEvent, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventAdverseEvent), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventAdverseEvent, "patient_id": patientID})
}

// submitResolutionSignal handles POST /patient/:id/signals/resolution (S21 Resolution).
func (s *Server) submitResolutionSignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.ResolutionSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventResolutionReport, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventResolutionReport), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventResolutionReport, "patient_id": patientID})
}

// submitHospitalisationSignal handles POST /patient/:id/signals/hospitalisation (S22 Hospitalisation).
func (s *Server) submitHospitalisationSignal(c *gin.Context) {
	patientID, ok := validatePatientID(c)
	if !ok {
		return
	}
	var req models.HospitalisationSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.eventBus.Publish(models.EventHospitalisation, patientID, req)
	s.logger.Info("Signal ingested", zap.String("type", models.EventHospitalisation), zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{"success": true, "signal": models.EventHospitalisation, "patient_id": patientID})
}
