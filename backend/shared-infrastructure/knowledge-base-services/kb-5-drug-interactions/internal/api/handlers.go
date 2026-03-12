package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-drug-interactions/internal/models"
	"kb-drug-interactions/internal/services"
)

// InteractionHandlers handles drug interaction related endpoints
type InteractionHandlers struct {
	interactionService *services.InteractionService
	// Enhanced engines for comprehensive clinical analysis
	integrationService *services.EnhancedIntegrationService
	pgxEngine          *services.PharmacogenomicEngine
	classEngine        *services.ClassInteractionEngine
	modifierEngine     *services.FoodAlcoholHerbalEngine
	matrixService      *services.EnhancedInteractionMatrixService
}

// NewInteractionHandlers creates handlers with all engines for comprehensive analysis
func NewInteractionHandlers(
	interactionService *services.InteractionService,
	integrationService *services.EnhancedIntegrationService,
	pgxEngine *services.PharmacogenomicEngine,
	classEngine *services.ClassInteractionEngine,
	modifierEngine *services.FoodAlcoholHerbalEngine,
	matrixService *services.EnhancedInteractionMatrixService,
) *InteractionHandlers {
	return &InteractionHandlers{
		interactionService: interactionService,
		integrationService: integrationService,
		pgxEngine:          pgxEngine,
		classEngine:        classEngine,
		modifierEngine:     modifierEngine,
		matrixService:      matrixService,
	}
}

// checkInteractions handles POST /api/v1/interactions/check
func (h *InteractionHandlers) checkInteractions(c *gin.Context) {
	var request models.InteractionCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Perform interaction check
	response, err := h.interactionService.CheckInteractions(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to check interactions", "CHECK_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, response, map[string]interface{}{
		"total_interactions": len(response.InteractionsFound),
		"highest_severity":   response.GetHighestSeverity(),
		"has_contraindications": response.HasContraindication(),
	})
}

// batchCheckInteractions handles POST /api/v1/interactions/batch-check
func (h *InteractionHandlers) batchCheckInteractions(c *gin.Context) {
	var request models.BatchInteractionCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid batch request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Process batch requests
	responses, err := h.interactionService.BatchCheckInteractions(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to process batch check", "BATCH_CHECK_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Calculate batch statistics
	totalInteractions := 0
	hasErrors := false
	for _, response := range responses {
		totalInteractions += len(response.InteractionsFound)
		if response.Summary.TotalInteractions == 0 && len(response.CheckedDrugs) > 1 {
			hasErrors = true
		}
	}

	sendSuccess(c, responses, map[string]interface{}{
		"batch_size":         len(request.Requests),
		"total_interactions": totalInteractions,
		"has_errors":        hasErrors,
	})
}

// getInteractionDetails handles GET /api/v1/interactions/:interaction_id
func (h *InteractionHandlers) getInteractionDetails(c *gin.Context) {
	interactionID := c.Param("interaction_id")
	if interactionID == "" {
		sendError(c, http.StatusBadRequest, "Interaction ID is required", "MISSING_INTERACTION_ID", nil)
		return
	}

	interaction, err := h.interactionService.GetInteractionDetails(interactionID)
	if err != nil {
		if err.Error() == "record not found" {
			sendError(c, http.StatusNotFound, "Interaction not found", "INTERACTION_NOT_FOUND", map[string]interface{}{
				"interaction_id": interactionID,
			})
			return
		}
		sendError(c, http.StatusInternalServerError, "Failed to get interaction details", "GET_FAILED", nil)
		return
	}

	sendSuccess(c, interaction, nil)
}

// getPatientInteractionHistory handles GET /api/v1/interactions/patient/:patient_id/history
func (h *InteractionHandlers) getPatientInteractionHistory(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "Patient ID is required", "MISSING_PATIENT_ID", nil)
		return
	}

	// Parse query parameters
	request := models.PatientInteractionHistoryRequest{
		PatientID:       patientID,
		IncludeResolved: parseBoolQuery(c, "include_resolved", false),
		Limit:           parseIntQuery(c, "limit", 50),
		Offset:          parseIntQuery(c, "offset", 0),
	}

	// Parse date filters
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			request.StartDate = startDate
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			request.EndDate = endDate
		}
	}

	// Parse severity filter
	if severityStr := c.Query("severity"); severityStr != "" {
		request.SeverityFilter = []string{severityStr}
	}

	alerts, total, err := h.interactionService.GetPatientInteractionHistory(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get patient history", "HISTORY_FAILED", nil)
		return
	}

	// Calculate pagination
	totalPages := int((total + int64(request.Limit) - 1) / int64(request.Limit))

	sendSuccess(c, alerts, map[string]interface{}{
		"pagination": map[string]interface{}{
			"page":        (request.Offset / request.Limit) + 1,
			"limit":       request.Limit,
			"total":       total,
			"total_pages": totalPages,
		},
		"patient_id": patientID,
	})
}

// overrideAlert handles PUT /api/v1/interactions/alerts/:alert_id/override
func (h *InteractionHandlers) overrideAlert(c *gin.Context) {
	alertIDStr := c.Param("alert_id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid alert ID format", "INVALID_ALERT_ID", nil)
		return
	}

	var request models.AlertOverrideRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid override request", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Set alert ID from URL parameter
	request.AlertID = alertID

	err = h.interactionService.OverrideInteractionAlert(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to override alert", "OVERRIDE_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, map[string]interface{}{
		"alert_id":      alertID,
		"status":        "overridden",
		"overridden_by": request.OverriddenBy,
		"timestamp":     time.Now().UTC(),
	}, nil)
}

// quickCheck handles GET /api/v1/interactions/quick-check
func (h *InteractionHandlers) quickCheck(c *gin.Context) {
	drugCodesStr := c.Query("drugs")
	if drugCodesStr == "" {
		sendError(c, http.StatusBadRequest, "Drug codes parameter is required", "MISSING_DRUGS", nil)
		return
	}

	// Parse comma-separated drug codes
	drugCodes := []string{}
	for _, code := range parseStringSliceQuery(c, "drugs") {
		if code != "" {
			drugCodes = append(drugCodes, code)
		}
	}

	if len(drugCodes) < 2 {
		sendError(c, http.StatusBadRequest, "At least 2 drug codes required", "INSUFFICIENT_DRUGS", nil)
		return
	}

	// Create quick check request
	request := models.InteractionCheckRequest{
		DrugCodes:           drugCodes,
		CheckType:           "quick",
		SeverityFilter:      parseStringSliceQuery(c, "severity"),
		IncludeAlternatives: parseBoolQuery(c, "alternatives", false),
		IncludeMonitoring:   parseBoolQuery(c, "monitoring", false),
	}

	response, err := h.interactionService.CheckInteractions(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Quick check failed", "QUICK_CHECK_FAILED", nil)
		return
	}

	// Return simplified response for quick checks
	sendSuccess(c, map[string]interface{}{
		"drugs_checked":      response.CheckedDrugs,
		"interactions_found": len(response.InteractionsFound),
		"highest_severity":   response.GetHighestSeverity(),
		"has_contraindications": response.HasContraindication(),
		"risk_score":         response.Summary.RiskScore,
		"interactions":       response.InteractionsFound,
		"recommendations":    response.Recommendations,
	}, map[string]interface{}{
		"cache_hit": response.CacheHit,
		"timestamp": response.CheckTimestamp,
	})
}

// searchInteractions handles GET /api/v1/interactions/search
func (h *InteractionHandlers) searchInteractions(c *gin.Context) {
	drugCode := c.Query("drug")
	severity := c.Query("severity")
	limit := parseIntQuery(c, "limit", 20)

	if drugCode == "" && severity == "" {
		sendError(c, http.StatusBadRequest, "Either drug code or severity filter is required", "MISSING_SEARCH_PARAMS", nil)
		return
	}

	// This would typically involve a more complex search service
	// For now, return a simple response indicating the search capability
	sendSuccess(c, map[string]interface{}{
		"message": "Interaction search functionality",
		"filters": map[string]interface{}{
			"drug":     drugCode,
			"severity": severity,
			"limit":    limit,
		},
	}, nil)
}

// getInteractionStatistics handles GET /api/v1/interactions/statistics
func (h *InteractionHandlers) getInteractionStatistics(c *gin.Context) {
	days := parseIntQuery(c, "days", 30)
	
	// This would typically query analytics data
	// For now, return mock statistics
	stats := map[string]interface{}{
		"total_checks_last_30_days":    1250,
		"interactions_found":           340,
		"contraindicated_interactions": 15,
		"major_interactions":          125,
		"moderate_interactions":       180,
		"minor_interactions":          20,
		"most_common_interactions": []map[string]interface{}{
			{
				"drug_a":     "Warfarin",
				"drug_b":     "Aspirin",
				"severity":   "major",
				"frequency":  45,
			},
			{
				"drug_a":     "Simvastatin",
				"drug_b":     "Gemfibrozil",
				"severity":   "contraindicated",
				"frequency":  12,
			},
		},
		"alert_override_rate": 0.08,
		"compliance_rate":     0.92,
		"period_days":        days,
	}

	sendSuccess(c, stats, nil)
}

// Helper functions

func sendSuccess(c *gin.Context, data interface{}, meta map[string]interface{}) {
	response := gin.H{
		"success": true,
		"data":    data,
	}

	if meta != nil {
		response["meta"] = meta
	}

	c.JSON(http.StatusOK, response)
}

func sendError(c *gin.Context, statusCode int, message, code string, details interface{}) {
	response := gin.H{
		"success": false,
		"error": gin.H{
			"message": message,
			"code":    code,
		},
	}

	if details != nil {
		response["error"].(gin.H)["details"] = details
	}

	c.JSON(statusCode, response)
}

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func parseBoolQuery(c *gin.Context, key string, defaultValue bool) bool {
	if value := c.Query(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func parseStringSliceQuery(c *gin.Context, key string) []string {
	if value := c.Query(key); value != "" {
		return []string{value} // Simple implementation - could be enhanced to support comma-separated values
	}
	return []string{}
}

// ============================================================================
// PHASE 2: Enhanced Engine Endpoints
// ============================================================================

// comprehensiveAnalysis handles POST /api/v1/interactions/comprehensive
// This endpoint orchestrates ALL engines for complete clinical analysis
func (h *InteractionHandlers) comprehensiveAnalysis(c *gin.Context) {
	if h.integrationService == nil {
		sendError(c, http.StatusServiceUnavailable, "Comprehensive analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	var request struct {
		DrugCodes       []string                    `json:"drug_codes" binding:"required,min=2"`
		PatientContext  *models.PatientContext      `json:"patient_context,omitempty"`
		ModifierContext *services.ModifierContext   `json:"modifier_context,omitempty"`
		DatasetVersion  string                      `json:"dataset_version,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Set default dataset version if not provided
	datasetVersion := request.DatasetVersion
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	// Build analysis request with proper types
	analysisRequest := services.ComprehensiveInteractionRequest{
		DrugCodes:      request.DrugCodes,
		DatasetVersion: datasetVersion,
	}

	// Add patient context if provided
	if request.PatientContext != nil {
		analysisRequest.PatientContext = *request.PatientContext
	}

	// Add modifier context if provided
	if request.ModifierContext != nil {
		analysisRequest.ModifierContext = *request.ModifierContext
	}

	// Perform comprehensive analysis
	response, err := h.integrationService.PerformComprehensiveAnalysis(c.Request.Context(), analysisRequest)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to perform comprehensive analysis", "ANALYSIS_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, response, map[string]interface{}{
		"engines_used":        []string{"pgx", "class", "modifier", "matrix"},
		"analysis_type":       "comprehensive",
		"has_critical_alerts": len(response.CriticalAlerts) > 0,
	})
}

// foodInteractions handles POST /api/v1/interactions/food
// Analyzes food, alcohol, and herbal supplement interactions
func (h *InteractionHandlers) foodInteractions(c *gin.Context) {
	if h.modifierEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Food interaction analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	var request struct {
		DrugCodes       []string                  `json:"drug_codes" binding:"required,min=1"`
		ModifierContext services.ModifierContext  `json:"modifier_context" binding:"required"`
		DatasetVersion  string                    `json:"dataset_version,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Set default dataset version if not provided
	datasetVersion := request.DatasetVersion
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	// Evaluate food/alcohol/herbal interactions with context and version
	results, err := h.modifierEngine.EvaluateModifierInteractions(
		c.Request.Context(),
		request.DrugCodes,
		request.ModifierContext,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to evaluate food interactions", "EVALUATION_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, results, map[string]interface{}{
		"total_interactions": len(results),
		"analysis_type":      "food_alcohol_herbal",
	})
}

// classInteractions handles POST /api/v1/interactions/class
// Analyzes drug class-based interactions (ATC classification)
func (h *InteractionHandlers) classInteractions(c *gin.Context) {
	if h.classEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Class interaction analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	var request struct {
		DrugCodes      []string `json:"drug_codes" binding:"required,min=2"`
		DatasetVersion string   `json:"dataset_version,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Set default dataset version if not provided
	datasetVersion := request.DatasetVersion
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	// Evaluate class-based interactions with context and version
	results, err := h.classEngine.EvaluateClassInteractions(
		c.Request.Context(),
		request.DrugCodes,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to evaluate class interactions", "EVALUATION_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Also check for Triple Whammy pattern (ACE-I + Diuretic + NSAID)
	tripleWhammyResult, tripleWhammyDetected := h.classEngine.EvaluateTripleWhammy(
		c.Request.Context(),
		request.DrugCodes,
		datasetVersion,
	)

	sendSuccess(c, map[string]interface{}{
		"class_interactions": results,
		"triple_whammy":      tripleWhammyResult,
	}, map[string]interface{}{
		"total_interactions": len(results),
		"analysis_type":      "drug_class",
		"triple_whammy_risk": tripleWhammyDetected,
	})
}

// cypProfile handles GET /api/v1/cyp/profile/:drug_code
// Returns CYP enzyme metabolism profile for a specific drug
func (h *InteractionHandlers) cypProfile(c *gin.Context) {
	if h.pgxEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "PGx analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	drugCode := c.Param("drug_code")
	if drugCode == "" {
		sendError(c, http.StatusBadRequest, "Drug code is required", "MISSING_DRUG_CODE", nil)
		return
	}

	// Get dataset version from query params (optional)
	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	// Get patient PGx markers from query params (optional)
	patientPGX := make(map[string]string)
	if cyp2d6 := c.Query("cyp2d6"); cyp2d6 != "" {
		patientPGX["CYP2D6"] = cyp2d6
	}
	if cyp2c19 := c.Query("cyp2c19"); cyp2c19 != "" {
		patientPGX["CYP2C19"] = cyp2c19
	}
	if slco1b1 := c.Query("slco1b1"); slco1b1 != "" {
		patientPGX["SLCO1B1"] = slco1b1
	}
	if cyp3a5 := c.Query("cyp3a5"); cyp3a5 != "" {
		patientPGX["CYP3A5"] = cyp3a5
	}

	// Evaluate drug metabolism with context and version
	metabolismResult, err := h.pgxEngine.EvaluateDrugMetabolism(
		c.Request.Context(),
		drugCode,
		patientPGX,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to evaluate drug metabolism", "EVALUATION_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, metabolismResult, map[string]interface{}{
		"drug_code":       drugCode,
		"analysis_type":   "cyp_metabolism",
		"has_patient_pgx": len(patientPGX) > 0,
	})
}

// cypEnzymeInteractions handles GET /api/v1/cyp/interactions/:enzyme
// Returns drugs that interact via a specific CYP enzyme
func (h *InteractionHandlers) cypEnzymeInteractions(c *gin.Context) {
	if h.pgxEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "PGx analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	enzyme := c.Param("enzyme")
	if enzyme == "" {
		sendError(c, http.StatusBadRequest, "Enzyme identifier is required", "MISSING_ENZYME", nil)
		return
	}

	// Get dataset version from query params (optional)
	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	// Get supported PGx markers with context and version
	supportedMarkers, err := h.pgxEngine.GetSupportedPGXMarkers(c.Request.Context(), datasetVersion)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve supported markers", "MARKERS_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Validate enzyme is a supported PGx marker (check map keys)
	phenotypes, markerValid := supportedMarkers[enzyme]

	if !markerValid {
		// Get list of supported gene names for error response
		supportedGenes := make([]string, 0, len(supportedMarkers))
		for gene := range supportedMarkers {
			supportedGenes = append(supportedGenes, gene)
		}
		sendError(c, http.StatusBadRequest, "Invalid or unsupported enzyme", "INVALID_ENZYME", map[string]interface{}{
			"enzyme":          enzyme,
			"supported_genes": supportedGenes,
		})
		return
	}

	// Return enzyme-specific interaction information
	sendSuccess(c, map[string]interface{}{
		"enzyme":            enzyme,
		"phenotypes":        phenotypes,
		"supported_markers": supportedMarkers,
		"description":       getEnzymeDescription(enzyme),
	}, map[string]interface{}{
		"analysis_type": "enzyme_profile",
	})
}

// getEnzymeDescription returns a clinical description for a CYP enzyme
func getEnzymeDescription(enzyme string) map[string]interface{} {
	descriptions := map[string]map[string]interface{}{
		"CYP2D6": {
			"name":        "Cytochrome P450 2D6",
			"metabolizes": []string{"codeine", "tramadol", "tamoxifen", "metoprolol", "many antidepressants"},
			"phenotypes":  []string{"poor_metabolizer", "intermediate_metabolizer", "normal_metabolizer", "ultra_rapid_metabolizer"},
			"clinical_significance": "Affects ~25% of all medications. Poor metabolizers may experience toxicity; ultra-rapid metabolizers may have reduced efficacy.",
		},
		"CYP2C19": {
			"name":        "Cytochrome P450 2C19",
			"metabolizes": []string{"clopidogrel", "omeprazole", "diazepam", "citalopram"},
			"phenotypes":  []string{"poor_metabolizer", "intermediate_metabolizer", "normal_metabolizer", "rapid_metabolizer", "ultra_rapid_metabolizer"},
			"clinical_significance": "Critical for clopidogrel activation. Poor metabolizers have increased cardiovascular risk.",
		},
		"SLCO1B1": {
			"name":        "Organic anion transporting polypeptide 1B1",
			"metabolizes": []string{"simvastatin", "atorvastatin", "rosuvastatin", "other statins"},
			"phenotypes":  []string{"normal_function", "decreased_function", "poor_function"},
			"clinical_significance": "SLCO1B1*5 variant increases statin myopathy risk. Consider lower doses or alternative statins.",
		},
		"CYP3A5": {
			"name":        "Cytochrome P450 3A5",
			"metabolizes": []string{"tacrolimus", "cyclosporine", "sirolimus"},
			"phenotypes":  []string{"poor_metabolizer", "intermediate_metabolizer", "normal_metabolizer"},
			"clinical_significance": "Affects immunosuppressant dosing. Expressers require higher tacrolimus doses.",
		},
	}

	if desc, exists := descriptions[enzyme]; exists {
		return desc
	}
	return map[string]interface{}{
		"name":        enzyme,
		"description": "Enzyme information not available",
	}
}

// ============================================================================
// PHASE 3: Drug-Disease, Allergy, and Duplicate Therapy Endpoints
// ============================================================================

// Phase3Handlers handles the new clinical safety engines
type Phase3Handlers struct {
	drugDiseaseEngine     *services.DrugDiseaseEngine
	allergyEngine         *services.AllergyEngine
	duplicateTherapyEngine *services.DuplicateTherapyEngine
}

// NewPhase3Handlers creates handlers for Phase 3 engines
func NewPhase3Handlers(
	drugDiseaseEngine *services.DrugDiseaseEngine,
	allergyEngine *services.AllergyEngine,
	duplicateTherapyEngine *services.DuplicateTherapyEngine,
) *Phase3Handlers {
	return &Phase3Handlers{
		drugDiseaseEngine:      drugDiseaseEngine,
		allergyEngine:          allergyEngine,
		duplicateTherapyEngine: duplicateTherapyEngine,
	}
}

// ============================================================================
// Drug-Disease Contraindication Handlers
// ============================================================================

// checkDrugDiseaseContraindications handles POST /api/v1/contraindications/disease
// Checks drugs against patient's disease conditions for contraindications
func (h *Phase3Handlers) checkDrugDiseaseContraindications(c *gin.Context) {
	if h.drugDiseaseEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Drug-disease analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	var request services.DrugDiseaseCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Get dataset version from query params or request body
	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "2025Q4" // Default to Phase 3 clinical data version
	}

	// Evaluate drug-disease contraindications
	results, err := h.drugDiseaseEngine.EvaluateDrugDiseaseContraindications(
		c.Request.Context(),
		request,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to evaluate drug-disease contraindications", "EVALUATION_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Count by severity
	contraindicatedCount := 0
	majorCount := 0
	for _, result := range results {
		switch result.Severity {
		case models.SeverityContraindicated:
			contraindicatedCount++
		case models.SeverityMajor:
			majorCount++
		}
	}

	sendSuccess(c, results, map[string]interface{}{
		"total_contraindications": len(results),
		"contraindicated_count":   contraindicatedCount,
		"major_count":             majorCount,
		"analysis_type":           "drug_disease",
	})
}

// getContraindicatedDiseases handles GET /api/v1/contraindications/drug/:drug_code/diseases
// Returns all diseases contraindicated for a specific drug
func (h *Phase3Handlers) getContraindicatedDiseases(c *gin.Context) {
	if h.drugDiseaseEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Drug-disease analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	drugCode := c.Param("drug_code")
	if drugCode == "" {
		sendError(c, http.StatusBadRequest, "Drug code is required", "MISSING_DRUG_CODE", nil)
		return
	}

	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	contraindications, err := h.drugDiseaseEngine.GetContraindicatedDiseases(
		c.Request.Context(),
		drugCode,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get contraindicated diseases", "QUERY_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, contraindications, map[string]interface{}{
		"drug_code":     drugCode,
		"total_results": len(contraindications),
	})
}

// getContraindicatedDrugs handles GET /api/v1/contraindications/disease/:disease_code/drugs
// Returns all drugs contraindicated for a specific disease
func (h *Phase3Handlers) getContraindicatedDrugs(c *gin.Context) {
	if h.drugDiseaseEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Drug-disease analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	diseaseCode := c.Param("disease_code")
	if diseaseCode == "" {
		sendError(c, http.StatusBadRequest, "Disease code is required", "MISSING_DISEASE_CODE", nil)
		return
	}

	codeSystem := c.Query("code_system") // ICD-10 or SNOMED-CT
	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	contraindications, err := h.drugDiseaseEngine.GetContraindicatedDrugs(
		c.Request.Context(),
		diseaseCode,
		codeSystem,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get contraindicated drugs", "QUERY_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, contraindications, map[string]interface{}{
		"disease_code":  diseaseCode,
		"code_system":   codeSystem,
		"total_results": len(contraindications),
	})
}

// ============================================================================
// Allergy Cross-Reactivity Handlers
// ============================================================================

// checkAllergyRisk handles POST /api/v1/allergy/check
// Checks drugs against patient's documented allergies for cross-reactivity
func (h *Phase3Handlers) checkAllergyRisk(c *gin.Context) {
	if h.allergyEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Allergy analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	var request services.AllergyCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	// Evaluate allergy risk
	results, err := h.allergyEngine.EvaluateAllergyRisk(
		c.Request.Context(),
		request,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to evaluate allergy risk", "EVALUATION_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Count by alert level
	criticalCount := 0
	highCount := 0
	for _, result := range results {
		switch result.AlertLevel {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
	}

	sendSuccess(c, results, map[string]interface{}{
		"total_alerts":   len(results),
		"critical_count": criticalCount,
		"high_count":     highCount,
		"analysis_type":  "allergy_cross_reactivity",
	})
}

// getCrossReactivity handles GET /api/v1/allergy/cross-reactivity/:allergen
// Returns cross-reactivity information for a specific allergen
func (h *Phase3Handlers) getCrossReactivity(c *gin.Context) {
	if h.allergyEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Allergy analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	allergenCode := c.Param("allergen")
	if allergenCode == "" {
		sendError(c, http.StatusBadRequest, "Allergen code is required", "MISSING_ALLERGEN", nil)
		return
	}

	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	crossReactivity, err := h.allergyEngine.GetCrossReactivity(
		c.Request.Context(),
		allergenCode,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get cross-reactivity", "QUERY_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, crossReactivity, map[string]interface{}{
		"allergen_code": allergenCode,
		"total_classes": len(crossReactivity),
	})
}

// getCommonCrossReactivities handles GET /api/v1/allergy/common-patterns
// Returns well-known cross-reactivity patterns
func (h *Phase3Handlers) getCommonCrossReactivities(c *gin.Context) {
	if h.allergyEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Allergy analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	patterns, err := h.allergyEngine.GetCommonCrossReactivities(
		c.Request.Context(),
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get common patterns", "QUERY_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, patterns, map[string]interface{}{
		"total_patterns": len(patterns),
		"description":    "Common cross-reactivity patterns for clinical reference",
	})
}

// ============================================================================
// Duplicate Therapy Handlers
// ============================================================================

// checkDuplicateTherapy handles POST /api/v1/duplicates/check
// Checks for duplicate therapy in a medication list
func (h *Phase3Handlers) checkDuplicateTherapy(c *gin.Context) {
	if h.duplicateTherapyEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Duplicate therapy analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	var request services.DuplicateTherapyCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	// Check for duplicate therapy
	results, err := h.duplicateTherapyEngine.CheckDuplicateTherapy(
		c.Request.Context(),
		request,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to check duplicate therapy", "EVALUATION_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Count by type
	exactCount := 0
	classCount := 0
	for _, result := range results {
		switch result.DuplicateType {
		case "exact":
			exactCount++
		case "therapeutic_class":
			classCount++
		}
	}

	sendSuccess(c, results, map[string]interface{}{
		"total_duplicates":      len(results),
		"exact_duplicates":      exactCount,
		"class_duplicates":      classCount,
		"analysis_type":         "duplicate_therapy",
	})
}

// getDrugTherapeuticClasses handles GET /api/v1/duplicates/classes/:drug_code
// Returns all therapeutic classifications for a drug
func (h *Phase3Handlers) getDrugTherapeuticClasses(c *gin.Context) {
	if h.duplicateTherapyEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Duplicate therapy analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	drugCode := c.Param("drug_code")
	if drugCode == "" {
		sendError(c, http.StatusBadRequest, "Drug code is required", "MISSING_DRUG_CODE", nil)
		return
	}

	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	classes, err := h.duplicateTherapyEngine.GetDrugTherapeuticClasses(
		c.Request.Context(),
		drugCode,
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get therapeutic classes", "QUERY_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Also get ATC hierarchy
	hierarchy, _ := h.duplicateTherapyEngine.GetATCHierarchy(
		c.Request.Context(),
		drugCode,
		datasetVersion,
	)

	sendSuccess(c, map[string]interface{}{
		"drug_code":     drugCode,
		"classes":       classes,
		"atc_hierarchy": hierarchy,
	}, map[string]interface{}{
		"total_classes": len(classes),
	})
}

// getCommonDuplicateClasses handles GET /api/v1/duplicates/common-classes
// Returns commonly duplicated therapeutic classes
func (h *Phase3Handlers) getCommonDuplicateClasses(c *gin.Context) {
	if h.duplicateTherapyEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Duplicate therapy analysis not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	datasetVersion := c.Query("dataset_version")
	if datasetVersion == "" {
		datasetVersion = "v1.0"
	}

	rules, err := h.duplicateTherapyEngine.GetCommonDuplicateClasses(
		c.Request.Context(),
		datasetVersion,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get duplicate classes", "QUERY_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, rules, map[string]interface{}{
		"total_classes": len(rules),
		"description":   "Therapeutic classes commonly flagged for duplicate therapy",
	})
}