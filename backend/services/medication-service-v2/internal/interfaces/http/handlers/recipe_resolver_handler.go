package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
)

// RecipeResolverHandler handles HTTP requests for recipe resolution
type RecipeResolverHandler struct {
	resolverService  entities.RecipeResolverService
	templateService  services.RecipeTemplateService
	cacheService     *services.RecipeCacheService
	ruleEngine       *services.ConditionalRuleEngine
	performanceTarget time.Duration
}

// RecipeResolutionRequest represents HTTP request for recipe resolution
type RecipeResolutionRequest struct {
	RecipeID       string                 `json:"recipe_id" binding:"required"`
	PatientContext PatientContextDTO      `json:"patient_context" binding:"required"`
	Options        ResolutionOptionsDTO   `json:"options,omitempty"`
	CorrelationID  string                 `json:"correlation_id,omitempty"`
}

// PatientContextDTO represents patient context in HTTP requests
type PatientContextDTO struct {
	PatientID        string                  `json:"patient_id" binding:"required"`
	Age              int                     `json:"age" binding:"required"`
	Weight           float64                 `json:"weight" binding:"required"`
	Height           float64                 `json:"height" binding:"required"`
	Gender           string                  `json:"gender" binding:"required"`
	PregnancyStatus  bool                    `json:"pregnancy_status"`
	RenalFunction    *RenalFunctionDTO       `json:"renal_function,omitempty"`
	HepaticFunction  *HepaticFunctionDTO     `json:"hepatic_function,omitempty"`
	Allergies        []AllergyDTO            `json:"allergies,omitempty"`
	Conditions       []ConditionDTO          `json:"conditions,omitempty"`
	CurrentMedications []CurrentMedicationDTO `json:"current_medications,omitempty"`
	LabResults       map[string]LabValueDTO  `json:"lab_results,omitempty"`
	Demographics     DemographicsDTO         `json:"demographics"`
	EncounterContext EncounterContextDTO     `json:"encounter_context"`
}

// ResolutionOptionsDTO represents resolution options
type ResolutionOptionsDTO struct {
	UseCache           bool   `json:"use_cache"`
	CacheTTLSeconds    int    `json:"cache_ttl_seconds"`
	SkipFreshnessCheck bool   `json:"skip_freshness_check"`
	ValidationLevel    string `json:"validation_level"`
	IncludeMetadata    bool   `json:"include_metadata"`
	ParallelProcessing bool   `json:"parallel_processing"`
	TimeoutMs          int64  `json:"timeout_ms"`
}

// Supporting DTOs
type RenalFunctionDTO struct {
	CreatinineClearance float64   `json:"creatinine_clearance"`
	SerumCreatinine     float64   `json:"serum_creatinine"`
	EGFR               float64   `json:"egfr"`
	Stage              string    `json:"stage"`
	LastUpdated        time.Time `json:"last_updated"`
}

type HepaticFunctionDTO struct {
	ALT         float64   `json:"alt"`
	AST         float64   `json:"ast"`
	Bilirubin   float64   `json:"bilirubin"`
	Albumin     float64   `json:"albumin"`
	ChildPugh   string    `json:"child_pugh_class"`
	LastUpdated time.Time `json:"last_updated"`
}

type AllergyDTO struct {
	Allergen    string    `json:"allergen"`
	Reaction    string    `json:"reaction"`
	Severity    string    `json:"severity"`
	Type        string    `json:"type"`
	OnsetDate   time.Time `json:"onset_date"`
	Verified    bool      `json:"verified"`
}

type ConditionDTO struct {
	Code        string    `json:"code"`
	System      string    `json:"system"`
	Display     string    `json:"display"`
	Status      string    `json:"status"`
	OnsetDate   time.Time `json:"onset_date"`
	Severity    string    `json:"severity"`
	IsPrimary   bool      `json:"is_primary"`
}

type CurrentMedicationDTO struct {
	Code          string    `json:"code"`
	System        string    `json:"system"`
	Display       string    `json:"display"`
	Dosage        string    `json:"dosage"`
	Frequency     string    `json:"frequency"`
	Route         string    `json:"route"`
	StartDate     time.Time `json:"start_date"`
	IsActive      bool      `json:"is_active"`
	PrescribedBy  string    `json:"prescribed_by"`
}

type LabValueDTO struct {
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	ReferenceRange string `json:"reference_range"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	IsAbnormal  bool      `json:"is_abnormal"`
}

type DemographicsDTO struct {
	Race          string `json:"race"`
	Ethnicity     string `json:"ethnicity"`
	Language      string `json:"language"`
	MaritalStatus string `json:"marital_status"`
	Insurance     string `json:"insurance"`
}

type EncounterContextDTO struct {
	EncounterID   string    `json:"encounter_id"`
	ProviderID    string    `json:"provider_id" binding:"required"`
	Specialty     string    `json:"specialty"`
	EncounterType string    `json:"encounter_type"`
	FacilityID    string    `json:"facility_id"`
	Date          time.Time `json:"date"`
	Urgency       string    `json:"urgency"`
}

// RecipeResolutionResponse represents HTTP response for recipe resolution
type RecipeResolutionResponse struct {
	Status           string                     `json:"status"`
	Resolution       *entities.RecipeResolution `json:"resolution"`
	ProcessingTimeMs int64                      `json:"processing_time_ms"`
	CacheUsed        bool                       `json:"cache_used"`
	MeetsPerformanceTarget bool               `json:"meets_performance_target"`
	Errors           []ErrorDTO                 `json:"errors,omitempty"`
	Warnings         []string                   `json:"warnings,omitempty"`
	CorrelationID    string                     `json:"correlation_id,omitempty"`
}

// ErrorDTO represents an error in HTTP responses
type ErrorDTO struct {
	Code        string      `json:"code"`
	Message     string      `json:"message"`
	Field       string      `json:"field,omitempty"`
	Phase       string      `json:"phase,omitempty"`
	Severity    string      `json:"severity"`
	Recoverable bool        `json:"recoverable"`
	Details     interface{} `json:"details,omitempty"`
}

// NewRecipeResolverHandler creates a new recipe resolver handler
func NewRecipeResolverHandler(
	resolverService entities.RecipeResolverService,
	templateService services.RecipeTemplateService,
	cacheService *services.RecipeCacheService,
	ruleEngine *services.ConditionalRuleEngine,
) *RecipeResolverHandler {
	return &RecipeResolverHandler{
		resolverService:   resolverService,
		templateService:   templateService,
		cacheService:      cacheService,
		ruleEngine:        ruleEngine,
		performanceTarget: 10 * time.Millisecond,
	}
}

// ResolveRecipe handles POST /api/v1/recipes/{id}/resolve
func (h *RecipeResolverHandler) ResolveRecipe(c *gin.Context) {
	startTime := time.Now()

	// Parse recipe ID from path
	recipeIDStr := c.Param("id")
	recipeID, err := uuid.Parse(recipeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid recipe ID format",
		})
		return
	}

	// Parse request body
	var req RecipeResolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format: " + err.Error(),
		})
		return
	}

	// Override recipe ID from path
	req.RecipeID = recipeID.String()

	// Set correlation ID if not provided
	if req.CorrelationID == "" {
		req.CorrelationID = uuid.New().String()
	}

	// Convert DTO to domain entities
	domainReq, err := h.convertToDomainRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request data: " + err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Resolve recipe
	resolution, err := h.resolverService.ResolveRecipe(ctx, *domainReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":         "error",
			"error":          err.Error(),
			"correlation_id": req.CorrelationID,
		})
		return
	}

	// Calculate processing time
	processingTime := time.Since(startTime)
	meetsTarget := processingTime <= h.performanceTarget

	// Build response
	response := RecipeResolutionResponse{
		Status:                 "success",
		Resolution:             resolution,
		ProcessingTimeMs:       processingTime.Milliseconds(),
		CacheUsed:              false, // This would be set by the resolver service
		MeetsPerformanceTarget: meetsTarget,
		CorrelationID:          req.CorrelationID,
	}

	// Add performance warning if target not met
	if !meetsTarget {
		response.Warnings = append(response.Warnings, 
			"Processing time exceeded target of 10ms")
	}

	c.JSON(http.StatusOK, response)
}

// GetResolverHealth handles GET /api/v1/resolver/health
func (h *RecipeResolverHandler) GetResolverHealth(c *gin.Context) {
	ctx := c.Request.Context()

	// Check cache health
	cacheHealth := h.cacheService.GetCacheHealth(ctx)

	// Get cache statistics
	cacheStats, err := h.cacheService.GetCacheStatistics(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Failed to get cache statistics",
		})
		return
	}

	health := map[string]interface{}{
		"status":             "healthy",
		"performance_target": h.performanceTarget.String(),
		"cache_health":       cacheHealth,
		"cache_statistics":   cacheStats,
		"features": map[string]bool{
			"recipe_resolution":     true,
			"field_merging":        true,
			"conditional_rules":    true,
			"protocol_resolvers":   true,
			"caching":              true,
			"performance_tracking": true,
		},
		"timestamp": time.Now(),
	}

	c.JSON(http.StatusOK, health)
}

// GetProtocolResolvers handles GET /api/v1/resolver/protocols
func (h *RecipeResolverHandler) GetProtocolResolvers(c *gin.Context) {
	protocols := map[string]interface{}{
		"hypertension-standard": map[string]interface{}{
			"name":        "Hypertension Standard Protocol",
			"version":     "1.0.0",
			"description": "Standard hypertension management protocol",
			"features": []string{
				"age_based_targets",
				"ckd_considerations",
				"diabetes_modifications",
				"pregnancy_safety",
			},
		},
		"diabetes-management": map[string]interface{}{
			"name":        "Diabetes Management Protocol",
			"version":     "1.0.0",
			"description": "Comprehensive diabetes management protocol",
			"features": []string{
				"hba1c_based_intensity",
				"age_based_targets",
				"renal_function_adjustments",
				"heart_failure_considerations",
			},
		},
		"pediatric-standard": map[string]interface{}{
			"name":        "Pediatric Standard Protocol",
			"version":     "1.0.0",
			"description": "Standard pediatric dosing and safety protocol",
			"features": []string{
				"age_based_dosing",
				"weight_based_calculations",
				"organ_maturity_considerations",
				"formulation_preferences",
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"protocols": protocols,
		"count":     len(protocols),
	})
}

// ClearCache handles POST /api/v1/resolver/cache/clear
func (h *RecipeResolverHandler) ClearCache(c *gin.Context) {
	var req struct {
		Type      string `json:"type"` // patient, recipe, protocol, all
		ID        string `json:"id,omitempty"`
		PatientID string `json:"patient_id,omitempty"`
		RecipeID  string `json:"recipe_id,omitempty"`
		ProtocolID string `json:"protocol_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	ctx := c.Request.Context()

	switch req.Type {
	case "patient":
		if req.PatientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"error":  "patient_id is required for patient cache invalidation",
			})
			return
		}
		err := h.cacheService.InvalidatePatientCache(ctx, req.PatientID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

	case "recipe":
		if req.RecipeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"error":  "recipe_id is required for recipe cache invalidation",
			})
			return
		}
		recipeID, err := uuid.Parse(req.RecipeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"error":  "Invalid recipe ID format",
			})
			return
		}
		err = h.cacheService.InvalidateRecipeCache(ctx, recipeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

	case "protocol":
		if req.ProtocolID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"error":  "protocol_id is required for protocol cache invalidation",
			})
			return
		}
		err := h.cacheService.InvalidateProtocolCache(ctx, req.ProtocolID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid cache type. Must be: patient, recipe, protocol, or all",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Cache cleared successfully",
		"type":    req.Type,
	})
}

// GetCacheStatistics handles GET /api/v1/resolver/cache/statistics
func (h *RecipeResolverHandler) GetCacheStatistics(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.cacheService.GetCacheStatistics(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	memoryUsage, err := h.cacheService.GetMemoryUsage(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"statistics":   stats,
		"memory_usage": memoryUsage,
		"healthy":      h.cacheService.IsCacheHealthy(),
	})
}

// EvaluateRules handles POST /api/v1/resolver/rules/evaluate
func (h *RecipeResolverHandler) EvaluateRules(c *gin.Context) {
	var req struct {
		ProtocolID     string            `json:"protocol_id" binding:"required"`
		PatientContext PatientContextDTO `json:"patient_context" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format: " + err.Error(),
		})
		return
	}

	// Convert patient context
	patientContext, err := h.convertPatientContext(req.PatientContext)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid patient context: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// Evaluate rules
	results, err := h.ruleEngine.EvaluateRules(ctx, req.ProtocolID, patientContext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"protocol_id": req.ProtocolID,
		"results":     results,
		"count":       len(results),
	})
}

// Helper methods

func (h *RecipeResolverHandler) convertToDomainRequest(req RecipeResolutionRequest) (*entities.RecipeResolutionRequest, error) {
	recipeID, err := uuid.Parse(req.RecipeID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid recipe ID")
	}

	patientContext, err := h.convertPatientContext(req.PatientContext)
	if err != nil {
		return nil, errors.Wrap(err, "invalid patient context")
	}

	options := h.convertResolutionOptions(req.Options)

	return &entities.RecipeResolutionRequest{
		RecipeID:       recipeID,
		PatientContext: patientContext,
		Options:        options,
		CorrelationID:  req.CorrelationID,
	}, nil
}

func (h *RecipeResolverHandler) convertPatientContext(dto PatientContextDTO) (entities.PatientContext, error) {
	// Convert allergies
	allergies := make([]entities.Allergy, len(dto.Allergies))
	for i, a := range dto.Allergies {
		allergies[i] = entities.Allergy{
			Allergen:    a.Allergen,
			Reaction:    a.Reaction,
			Severity:    a.Severity,
			Type:        a.Type,
			OnsetDate:   a.OnsetDate,
			Verified:    a.Verified,
		}
	}

	// Convert conditions
	conditions := make([]entities.Condition, len(dto.Conditions))
	for i, c := range dto.Conditions {
		conditions[i] = entities.Condition{
			Code:        c.Code,
			System:      c.System,
			Display:     c.Display,
			Status:      c.Status,
			OnsetDate:   c.OnsetDate,
			Severity:    c.Severity,
			IsPrimary:   c.IsPrimary,
		}
	}

	// Convert current medications
	currentMeds := make([]entities.CurrentMedication, len(dto.CurrentMedications))
	for i, m := range dto.CurrentMedications {
		currentMeds[i] = entities.CurrentMedication{
			Code:          m.Code,
			System:        m.System,
			Display:       m.Display,
			Dosage:        m.Dosage,
			Frequency:     m.Frequency,
			Route:         m.Route,
			StartDate:     m.StartDate,
			IsActive:      m.IsActive,
			PrescribedBy:  m.PrescribedBy,
		}
	}

	// Convert lab results
	labResults := make(map[string]entities.LabValue)
	for key, lab := range dto.LabResults {
		labResults[key] = entities.LabValue{
			Value:       lab.Value,
			Unit:        lab.Unit,
			ReferenceRange: lab.ReferenceRange,
			Status:      lab.Status,
			Timestamp:   lab.Timestamp,
			IsAbnormal:  lab.IsAbnormal,
		}
	}

	// Convert renal function
	var renalFunc *entities.RenalFunction
	if dto.RenalFunction != nil {
		renalFunc = &entities.RenalFunction{
			CreatinineClearance: dto.RenalFunction.CreatinineClearance,
			SerumCreatinine:     dto.RenalFunction.SerumCreatinine,
			eGFR:               dto.RenalFunction.EGFR,
			Stage:              dto.RenalFunction.Stage,
			LastUpdated:        dto.RenalFunction.LastUpdated,
		}
	}

	// Convert hepatic function
	var hepaticFunc *entities.HepaticFunction
	if dto.HepaticFunction != nil {
		hepaticFunc = &entities.HepaticFunction{
			ALT:         dto.HepaticFunction.ALT,
			AST:         dto.HepaticFunction.AST,
			Bilirubin:   dto.HepaticFunction.Bilirubin,
			Albumin:     dto.HepaticFunction.Albumin,
			ChildPugh:   dto.HepaticFunction.ChildPugh,
			LastUpdated: dto.HepaticFunction.LastUpdated,
		}
	}

	return entities.PatientContext{
		PatientID:        dto.PatientID,
		Age:              dto.Age,
		Weight:           dto.Weight,
		Height:           dto.Height,
		Gender:           dto.Gender,
		PregnancyStatus:  dto.PregnancyStatus,
		RenalFunction:    renalFunc,
		HepaticFunction:  hepaticFunc,
		Allergies:        allergies,
		Conditions:       conditions,
		CurrentMedications: currentMeds,
		LabResults:       labResults,
		Demographics: entities.Demographics{
			Race:          dto.Demographics.Race,
			Ethnicity:     dto.Demographics.Ethnicity,
			Language:      dto.Demographics.Language,
			MaritalStatus: dto.Demographics.MaritalStatus,
			Insurance:     dto.Demographics.Insurance,
		},
		EncounterContext: entities.EncounterContext{
			EncounterID:   dto.EncounterContext.EncounterID,
			ProviderID:    dto.EncounterContext.ProviderID,
			Specialty:     dto.EncounterContext.Specialty,
			EncounterType: dto.EncounterContext.EncounterType,
			FacilityID:    dto.EncounterContext.FacilityID,
			Date:          dto.EncounterContext.Date,
			Urgency:       dto.EncounterContext.Urgency,
		},
	}, nil
}

func (h *RecipeResolverHandler) convertResolutionOptions(dto ResolutionOptionsDTO) entities.ResolutionOptions {
	validationLevel := entities.ValidationLevelBasic
	switch dto.ValidationLevel {
	case "none":
		validationLevel = entities.ValidationLevelNone
	case "strict":
		validationLevel = entities.ValidationLevelStrict
	case "critical":
		validationLevel = entities.ValidationLevelCritical
	}

	cacheTTL := time.Duration(dto.CacheTTLSeconds) * time.Second
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute // Default TTL
	}

	return entities.ResolutionOptions{
		UseCache:           dto.UseCache,
		CacheTTL:           cacheTTL,
		SkipFreshnessCheck: dto.SkipFreshnessCheck,
		ValidationLevel:    validationLevel,
		IncludeMetadata:    dto.IncludeMetadata,
		ParallelProcessing: dto.ParallelProcessing,
		TimeoutMs:          dto.TimeoutMs,
	}
}