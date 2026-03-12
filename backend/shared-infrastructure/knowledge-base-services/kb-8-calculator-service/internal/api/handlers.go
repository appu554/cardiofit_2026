package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-8-calculator-service/internal/models"
)

// CalculateEGFRRequest is the request body for eGFR calculation.
type CalculateEGFRRequest struct {
	SerumCreatinine float64    `json:"serumCreatinine" binding:"required,gt=0"`
	AgeYears        int        `json:"ageYears" binding:"required,gt=0,lte=120"`
	Sex             models.Sex `json:"sex" binding:"required"`
}

// CalculateCrClRequest is the request body for CrCl calculation.
type CalculateCrClRequest struct {
	SerumCreatinine float64    `json:"serumCreatinine" binding:"required,gt=0"`
	AgeYears        int        `json:"ageYears" binding:"required,gt=0,lte=120"`
	Sex             models.Sex `json:"sex" binding:"required"`
	WeightKg        float64    `json:"weightKg" binding:"required,gt=0"`
}

// CalculateBMIRequest is the request body for BMI calculation.
type CalculateBMIRequest struct {
	WeightKg  float64       `json:"weightKg" binding:"required,gt=0"`
	HeightCm  float64       `json:"heightCm" binding:"required,gt=0"`
	Region    models.Region `json:"region,omitempty"`
	Ethnicity string        `json:"ethnicity,omitempty"`
}

// calculateEGFRHandler handles POST /api/v1/calculate/egfr
//
// @Summary Calculate eGFR
// @Description Calculates estimated glomerular filtration rate using CKD-EPI 2021 race-free equation
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body CalculateEGFRRequest true "eGFR calculation parameters"
// @Success 200 {object} models.EGFRResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/egfr [post]
func (s *Server) calculateEGFRHandler(c *gin.Context) {
	var req CalculateEGFRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Convert to internal params
	params := &models.EGFRParams{
		SerumCreatinine: req.SerumCreatinine,
		AgeYears:        req.AgeYears,
		Sex:             req.Sex,
	}

	// Calculate
	result, err := s.service.CalculateEGFR(c.Request.Context(), params)
	if err != nil {
		switch err {
		case models.ErrInvalidCreatinine:
			respondError(c, http.StatusBadRequest, "INVALID_CREATININE", err.Error())
		case models.ErrInvalidAge:
			respondError(c, http.StatusBadRequest, "INVALID_AGE", err.Error())
		case models.ErrInvalidSex:
			respondError(c, http.StatusBadRequest, "INVALID_SEX", err.Error())
		default:
			respondError(c, http.StatusInternalServerError, "CALCULATION_ERROR", err.Error())
		}
		return
	}

	respondSuccess(c, result)
}

// calculateCrClHandler handles POST /api/v1/calculate/crcl
//
// @Summary Calculate CrCl
// @Description Calculates creatinine clearance using Cockcroft-Gault equation
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body CalculateCrClRequest true "CrCl calculation parameters"
// @Success 200 {object} models.CrClResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/crcl [post]
func (s *Server) calculateCrClHandler(c *gin.Context) {
	var req CalculateCrClRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Convert to internal params
	params := &models.CrClParams{
		SerumCreatinine: req.SerumCreatinine,
		AgeYears:        req.AgeYears,
		Sex:             req.Sex,
		WeightKg:        req.WeightKg,
	}

	// Calculate
	result, err := s.service.CalculateCrCl(c.Request.Context(), params)
	if err != nil {
		switch err {
		case models.ErrInvalidCreatinine:
			respondError(c, http.StatusBadRequest, "INVALID_CREATININE", err.Error())
		case models.ErrInvalidAge:
			respondError(c, http.StatusBadRequest, "INVALID_AGE", err.Error())
		case models.ErrInvalidSex:
			respondError(c, http.StatusBadRequest, "INVALID_SEX", err.Error())
		case models.ErrInvalidWeight:
			respondError(c, http.StatusBadRequest, "INVALID_WEIGHT", err.Error())
		default:
			respondError(c, http.StatusInternalServerError, "CALCULATION_ERROR", err.Error())
		}
		return
	}

	respondSuccess(c, result)
}

// calculateBMIHandler handles POST /api/v1/calculate/bmi
//
// @Summary Calculate BMI
// @Description Calculates body mass index with Western and Asian (WHO Asia-Pacific) categorization
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body CalculateBMIRequest true "BMI calculation parameters"
// @Success 200 {object} models.BMIResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/bmi [post]
func (s *Server) calculateBMIHandler(c *gin.Context) {
	var req CalculateBMIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Convert to internal params
	params := &models.BMIParams{
		WeightKg:  req.WeightKg,
		HeightCm:  req.HeightCm,
		Region:    req.Region,
		Ethnicity: req.Ethnicity,
	}

	// Calculate
	result, err := s.service.CalculateBMI(c.Request.Context(), params)
	if err != nil {
		switch err {
		case models.ErrInvalidWeight:
			respondError(c, http.StatusBadRequest, "INVALID_WEIGHT", err.Error())
		case models.ErrInvalidHeight:
			respondError(c, http.StatusBadRequest, "INVALID_HEIGHT", err.Error())
		default:
			respondError(c, http.StatusInternalServerError, "CALCULATION_ERROR", err.Error())
		}
		return
	}

	respondSuccess(c, result)
}

// calculateBatchHandler handles POST /api/v1/calculate/batch
//
// @Summary Batch Calculate
// @Description Performs multiple calculations in a single request
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body models.BatchCalculatorRequest true "Batch calculation parameters"
// @Success 200 {object} models.SimpleBatchResponse
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/batch [post]
func (s *Server) calculateBatchHandler(c *gin.Context) {
	var req models.BatchCalculatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Validate batch size
	if len(req.Calculators) == 0 {
		respondError(c, http.StatusBadRequest, "EMPTY_BATCH", "at least one calculator must be requested")
		return
	}
	if len(req.Calculators) > s.cfg.MaxBatchSize {
		respondError(c, http.StatusBadRequest, "BATCH_TOO_LARGE",
			"batch size exceeds maximum allowed")
		return
	}

	// Calculate batch
	result, err := s.service.CalculateBatch(c.Request.Context(), &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "BATCH_ERROR", err.Error())
		return
	}

	respondSuccess(c, result)
}

// listCalculatorsHandler handles GET /api/v1/calculators
//
// @Summary List Calculators
// @Description Returns list of available calculators with their metadata
// @Tags info
// @Produce json
// @Success 200 {array} models.CalculatorInfo
// @Router /api/v1/calculators [get]
func (s *Server) listCalculatorsHandler(c *gin.Context) {
	calculators := s.service.GetAvailableCalculators()
	respondSuccess(c, gin.H{
		"calculators": calculators,
		"count":       len(calculators),
	})
}

// ==================== P1 Calculator Handlers ====================

// calculateSOFAHandler handles POST /api/v1/calculate/sofa
//
// @Summary Calculate SOFA Score
// @Description Calculates Sequential Organ Failure Assessment score for ICU mortality prediction
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body models.SOFAParams true "SOFA calculation parameters"
// @Success 200 {object} models.SOFAResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/sofa [post]
func (s *Server) calculateSOFAHandler(c *gin.Context) {
	var params models.SOFAParams
	if err := c.ShouldBindJSON(&params); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	result, err := s.service.CalculateSOFA(c.Request.Context(), &params)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "CALCULATION_ERROR", err.Error())
		return
	}

	respondSuccess(c, result)
}

// calculateQSOFAHandler handles POST /api/v1/calculate/qsofa
//
// @Summary Calculate qSOFA Score
// @Description Calculates quick SOFA score for bedside sepsis screening
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body models.QSOFAParams true "qSOFA calculation parameters"
// @Success 200 {object} models.QSOFAResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/qsofa [post]
func (s *Server) calculateQSOFAHandler(c *gin.Context) {
	var params models.QSOFAParams
	if err := c.ShouldBindJSON(&params); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	result, err := s.service.CalculateQSOFA(c.Request.Context(), &params)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "CALCULATION_ERROR", err.Error())
		return
	}

	respondSuccess(c, result)
}

// calculateCHA2DS2VAScHandler handles POST /api/v1/calculate/cha2ds2vasc
//
// @Summary Calculate CHA₂DS₂-VASc Score
// @Description Calculates stroke risk score for atrial fibrillation anticoagulation decisions
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body models.CHA2DS2VAScParams true "CHA2DS2-VASc calculation parameters"
// @Success 200 {object} models.CHA2DS2VAScResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/cha2ds2vasc [post]
func (s *Server) calculateCHA2DS2VAScHandler(c *gin.Context) {
	var params models.CHA2DS2VAScParams
	if err := c.ShouldBindJSON(&params); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	result, err := s.service.CalculateCHA2DS2VASc(c.Request.Context(), &params)
	if err != nil {
		switch err {
		case models.ErrInvalidAge:
			respondError(c, http.StatusBadRequest, "INVALID_AGE", err.Error())
		case models.ErrInvalidSex:
			respondError(c, http.StatusBadRequest, "INVALID_SEX", err.Error())
		default:
			respondError(c, http.StatusInternalServerError, "CALCULATION_ERROR", err.Error())
		}
		return
	}

	respondSuccess(c, result)
}

// calculateHASBLEDHandler handles POST /api/v1/calculate/hasbled
//
// @Summary Calculate HAS-BLED Score
// @Description Calculates major bleeding risk score for anticoagulation decisions
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body models.HASBLEDParams true "HAS-BLED calculation parameters"
// @Success 200 {object} models.HASBLEDResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/hasbled [post]
func (s *Server) calculateHASBLEDHandler(c *gin.Context) {
	var params models.HASBLEDParams
	if err := c.ShouldBindJSON(&params); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	result, err := s.service.CalculateHASBLED(c.Request.Context(), &params)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "CALCULATION_ERROR", err.Error())
		return
	}

	respondSuccess(c, result)
}

// calculateASCVDHandler handles POST /api/v1/calculate/ascvd
//
// @Summary Calculate ASCVD 10-Year Risk
// @Description Calculates 10-year atherosclerotic cardiovascular disease risk using Pooled Cohort Equations
// @Tags calculators
// @Accept json
// @Produce json
// @Param request body models.ASCVDParams true "ASCVD calculation parameters"
// @Success 200 {object} models.ASCVDResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/calculate/ascvd [post]
func (s *Server) calculateASCVDHandler(c *gin.Context) {
	var params models.ASCVDParams
	if err := c.ShouldBindJSON(&params); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	result, err := s.service.CalculateASCVD(c.Request.Context(), &params)
	if err != nil {
		switch err {
		case models.ErrInvalidAge:
			respondError(c, http.StatusBadRequest, "INVALID_AGE", err.Error())
		case models.ErrInvalidSex:
			respondError(c, http.StatusBadRequest, "INVALID_SEX", err.Error())
		default:
			respondError(c, http.StatusBadRequest, "CALCULATION_ERROR", err.Error())
		}
		return
	}

	respondSuccess(c, result)
}
