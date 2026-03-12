package api

import (
	"net/http"
	"time"

	"kb-7-terminology/internal/cdss"
	"kb-7-terminology/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// CDSS HTTP Handlers
// ============================================================================
// HTTP handlers for Clinical Decision Support System endpoints.
// These handlers enable patient-level evaluation against clinical value sets.

// CDSSHandlers contains the CDSS-related handlers
type CDSSHandlers struct {
	factBuilder    cdss.FactBuilder
	evaluator      cdss.CDSSEvaluator
	alertGenerator cdss.AlertGenerator
	ruleRepository cdss.RuleRepository // Database repository for persistent clinical rules
	logger         *logrus.Logger
}

// NewCDSSHandlers creates a new CDSSHandlers instance
func NewCDSSHandlers(
	factBuilder cdss.FactBuilder,
	evaluator cdss.CDSSEvaluator,
	alertGenerator cdss.AlertGenerator,
	logger *logrus.Logger,
) *CDSSHandlers {
	return &CDSSHandlers{
		factBuilder:    factBuilder,
		evaluator:      evaluator,
		alertGenerator: alertGenerator,
		logger:         logger,
	}
}

// NewCDSSHandlersWithRepository creates a new CDSSHandlers instance with database repository support
func NewCDSSHandlersWithRepository(
	factBuilder cdss.FactBuilder,
	evaluator cdss.CDSSEvaluator,
	alertGenerator cdss.AlertGenerator,
	ruleRepository cdss.RuleRepository,
	logger *logrus.Logger,
) *CDSSHandlers {
	return &CDSSHandlers{
		factBuilder:    factBuilder,
		evaluator:      evaluator,
		alertGenerator: alertGenerator,
		ruleRepository: ruleRepository,
		logger:         logger,
	}
}

// ============================================================================
// Fact Builder Endpoint
// ============================================================================

// BuildFactsRequest represents the request body for building facts
type BuildFactsRequest struct {
	PatientID   string                          `json:"patient_id" binding:"required"`
	EncounterID string                          `json:"encounter_id,omitempty"`
	Bundle      *models.FHIRBundle              `json:"bundle,omitempty"`
	Conditions  []models.FHIRCondition          `json:"conditions,omitempty"`
	Observations []models.FHIRObservation       `json:"observations,omitempty"`
	Medications []models.FHIRMedicationRequest  `json:"medications,omitempty"`
	Procedures  []models.FHIRProcedure          `json:"procedures,omitempty"`
	Allergies   []models.FHIRAllergyIntolerance `json:"allergies,omitempty"`
	Options     *models.FactBuilderOptions      `json:"options,omitempty"`
}

// BuildFacts handles POST /v1/cdss/facts/build
// Extracts clinical facts from FHIR resources
func (h *CDSSHandlers) BuildFacts(c *gin.Context) {
	var req BuildFactsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Convert to FactBuilderRequest
	factBuilderReq := &models.FactBuilderRequest{
		PatientID:   req.PatientID,
		EncounterID: req.EncounterID,
		Bundle:      req.Bundle,
		Conditions:  req.Conditions,
		Observations: req.Observations,
		Medications: req.Medications,
		Procedures:  req.Procedures,
		Allergies:   req.Allergies,
		Options:     req.Options,
	}

	// Build facts
	ctx := c.Request.Context()
	response, err := h.factBuilder.BuildFactsFromRequest(ctx, factBuilderReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to build facts")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to build facts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// CDSS Evaluation Endpoints
// ============================================================================

// EvaluateRequest represents the request body for CDSS evaluation
type EvaluateRequest struct {
	PatientID          string                          `json:"patient_id" binding:"required"`
	EncounterID        string                          `json:"encounter_id,omitempty"`
	FactSet            *models.PatientFactSet          `json:"fact_set,omitempty"`
	Bundle             *models.FHIRBundle              `json:"bundle,omitempty"`
	Conditions         []models.FHIRCondition          `json:"conditions,omitempty"`
	Observations       []models.FHIRObservation        `json:"observations,omitempty"`
	Medications        []models.FHIRMedicationRequest  `json:"medications,omitempty"`
	Procedures         []models.FHIRProcedure          `json:"procedures,omitempty"`
	Allergies          []models.FHIRAllergyIntolerance `json:"allergies,omitempty"`
	FactBuilderOptions *models.FactBuilderOptions      `json:"fact_builder_options,omitempty"`
	Options            *models.CDSSEvaluationOptions   `json:"options,omitempty"`
}

// EvaluatePatient handles POST /v1/cdss/evaluate
// Full CDSS evaluation with fact extraction and alert generation
func (h *CDSSHandlers) EvaluatePatient(c *gin.Context) {
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Convert to CDSSEvaluationRequest
	evalReq := &models.CDSSEvaluationRequest{
		PatientID:          req.PatientID,
		EncounterID:        req.EncounterID,
		FactSet:            req.FactSet,
		Bundle:             req.Bundle,
		Conditions:         req.Conditions,
		Observations:       req.Observations,
		Medications:        req.Medications,
		Procedures:         req.Procedures,
		Allergies:          req.Allergies,
		FactBuilderOptions: req.FactBuilderOptions,
		Options:            req.Options,
	}

	// Evaluate patient
	ctx := c.Request.Context()
	response, err := h.evaluator.EvaluatePatient(ctx, evalReq)
	if err != nil {
		h.logger.WithError(err).Error("CDSS evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "CDSS evaluation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// EvaluateFactsRequest represents the request body for evaluating pre-built facts
type EvaluateFactsRequest struct {
	PatientID   string                        `json:"patient_id" binding:"required"`
	EncounterID string                        `json:"encounter_id,omitempty"`
	FactSet     *models.PatientFactSet        `json:"fact_set" binding:"required"`
	Options     *models.CDSSEvaluationOptions `json:"options,omitempty"`
}

// EvaluateFacts handles POST /v1/cdss/evaluate/facts
// Evaluates pre-built facts against clinical value sets
func (h *CDSSHandlers) EvaluateFacts(c *gin.Context) {
	var req EvaluateFactsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Convert to CDSSEvaluationRequest
	evalReq := &models.CDSSEvaluationRequest{
		PatientID:   req.PatientID,
		EncounterID: req.EncounterID,
		FactSet:     req.FactSet,
		Options:     req.Options,
	}

	// Evaluate facts
	ctx := c.Request.Context()
	response, err := h.evaluator.EvaluatePatient(ctx, evalReq)
	if err != nil {
		h.logger.WithError(err).Error("Fact evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Fact evaluation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// Alert Generation Endpoint
// ============================================================================

// GenerateAlertsRequest represents the request body for alert generation
type GenerateAlertsRequest struct {
	PatientID         string                       `json:"patient_id" binding:"required"`
	EncounterID       string                       `json:"encounter_id,omitempty"`
	EvaluationResults []models.EvaluationResult    `json:"evaluation_results" binding:"required"`
	FactSet           *models.PatientFactSet       `json:"fact_set,omitempty"`
	Options           *models.AlertGenerationOptions `json:"options,omitempty"`
}

// GenerateAlerts handles POST /v1/cdss/alerts/generate
// Generates clinical alerts from evaluation results
func (h *CDSSHandlers) GenerateAlerts(c *gin.Context) {
	var req GenerateAlertsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Convert to AlertGenerationRequest
	alertReq := &models.AlertGenerationRequest{
		PatientID:         req.PatientID,
		EncounterID:       req.EncounterID,
		EvaluationResults: req.EvaluationResults,
		FactSet:           req.FactSet,
		Options:           req.Options,
	}

	// Generate alerts
	ctx := c.Request.Context()
	response, err := h.alertGenerator.GenerateAlerts(ctx, alertReq)
	if err != nil {
		h.logger.WithError(err).Error("Alert generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Alert generation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// Health Check Endpoint
// ============================================================================

// CDSSHealthResponse represents the health status of CDSS components
type CDSSHealthResponse struct {
	Status          string            `json:"status"`
	Timestamp       time.Time         `json:"timestamp"`
	Components      map[string]string `json:"components"`
	Version         string            `json:"version"`
	PipelineEnabled string            `json:"pipeline_enabled"`
}

// CDSSHealth handles GET /v1/cdss/health
// Returns health status of CDSS subsystem
func (h *CDSSHandlers) CDSSHealth(c *gin.Context) {
	response := CDSSHealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Components: map[string]string{
			"fact_builder":    "available",
			"evaluator":       "available",
			"alert_generator": "available",
		},
		Version:         "1.0.0",
		PipelineEnabled: "THREE-CHECK",
	}

	// Check each component
	if h.factBuilder == nil {
		response.Components["fact_builder"] = "unavailable"
		response.Status = "degraded"
	}
	if h.evaluator == nil {
		response.Components["evaluator"] = "unavailable"
		response.Status = "degraded"
	}
	if h.alertGenerator == nil {
		response.Components["alert_generator"] = "unavailable"
		response.Status = "degraded"
	}

	statusCode := http.StatusOK
	if response.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// ============================================================================
// Quick Validate Endpoint
// ============================================================================

// QuickValidateRequest represents a simplified validation request
type QuickValidateRequest struct {
	PatientID   string `json:"patient_id"`
	Code        string `json:"code" binding:"required"`
	System      string `json:"system" binding:"required"`
	Display     string `json:"display,omitempty"`
	ValueSetIDs []string `json:"value_set_ids,omitempty"`
}

// QuickValidate handles POST /v1/cdss/validate
// Quick validation of a single code against value sets
func (h *CDSSHandlers) QuickValidate(c *gin.Context) {
	var req QuickValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Create a single fact for evaluation
	fact := models.ClinicalFact{
		ID:       "quick-validate",
		FactType: models.FactTypeCondition,
		Status:   models.FactStatusActive,
		Code:     req.Code,
		System:   req.System,
		Display:  req.Display,
	}

	// Set up options
	options := models.DefaultCDSSEvaluationOptions()
	options.GenerateAlerts = false
	options.IncludeDetails = true
	if len(req.ValueSetIDs) > 0 {
		options.ValueSetIDs = req.ValueSetIDs
	}

	// Evaluate the single fact
	ctx := c.Request.Context()
	result, err := h.evaluator.EvaluateFact(ctx, &fact, options)
	if err != nil {
		h.logger.WithError(err).Error("Quick validation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Quick validation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"code":    req.Code,
		"system":  req.System,
		"matched": result.Matched,
		"matched_value_sets": result.MatchedValueSets,
		"evaluation_time_ms": result.EvaluationTimeMs,
	})
}

// ============================================================================
// Domain Information Endpoint
// ============================================================================

// GetClinicalDomains handles GET /v1/cdss/domains
// Returns list of supported clinical domains
func (h *CDSSHandlers) GetClinicalDomains(c *gin.Context) {
	domains := models.AllClinicalDomains()

	domainInfo := make([]gin.H, 0, len(domains))
	for _, domain := range domains {
		indicators := models.GetIndicatorsByDomain(domain)
		indicatorNames := make([]string, 0, len(indicators))
		for _, ind := range indicators {
			indicatorNames = append(indicatorNames, ind.Name)
		}

		domainInfo = append(domainInfo, gin.H{
			"domain":      domain.String(),
			"indicators":  indicatorNames,
			"indicator_count": len(indicators),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"domains": domainInfo,
		"total":   len(domains),
	})
}

// ============================================================================
// Clinical Indicators Endpoint
// ============================================================================

// GetClinicalIndicators handles GET /v1/cdss/indicators
// Returns list of supported clinical indicators
func (h *CDSSHandlers) GetClinicalIndicators(c *gin.Context) {
	// Filter by domain if specified
	domainFilter := c.Query("domain")

	indicators := make([]gin.H, 0)
	for id, indicator := range models.ClinicalIndicatorRegistry {
		if domainFilter != "" && string(indicator.Domain) != domainFilter {
			continue
		}

		indicators = append(indicators, gin.H{
			"id":              id,
			"name":            indicator.Name,
			"description":     indicator.Description,
			"domain":          indicator.Domain.String(),
			"severity":        indicator.Severity.String(),
			"value_sets":      indicator.ValueSets,
			"recommendations": indicator.Recommendations,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"indicators": indicators,
		"total":      len(indicators),
	})
}

// ============================================================================
// Severity Mapping Endpoint
// ============================================================================

// GetSeverityMapping handles GET /v1/cdss/severity-mapping
// Returns value set to severity mappings
func (h *CDSSHandlers) GetSeverityMapping(c *gin.Context) {
	mappings := make([]gin.H, 0, len(models.ValueSetSeverityMapping))

	for valueSetID, severity := range models.ValueSetSeverityMapping {
		domain := models.GetDomainForValueSet(valueSetID)
		mappings = append(mappings, gin.H{
			"value_set_id": valueSetID,
			"severity":     severity.String(),
			"priority":     severity.Priority(),
			"domain":       domain.String(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"mappings": mappings,
		"total":    len(mappings),
	})
}

// ============================================================================
// Clinical Rules Management Endpoints
// ============================================================================

// SeedClinicalRules handles POST /v1/cdss/rules/seed
// Seeds the default clinical rules from in-memory definitions to the database
// This enables persistent rule storage for customization and versioning
func (h *CDSSHandlers) SeedClinicalRules(c *gin.Context) {
	if h.ruleRepository == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Rule repository not configured - database persistence not available",
		})
		return
	}

	ctx := c.Request.Context()

	// Get default clinical rules from in-memory definitions
	defaultRules := cdss.GetDefaultClinicalRules()

	// Save rules to database using batch operation
	if err := h.ruleRepository.SaveRules(ctx, defaultRules); err != nil {
		h.logger.WithError(err).Error("Failed to seed clinical rules to database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to seed clinical rules: " + err.Error(),
		})
		return
	}

	h.logger.WithField("rule_count", len(defaultRules)).Info("Successfully seeded clinical rules to database")

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Clinical rules seeded to database successfully",
		"rules_count": len(defaultRules),
		"description": "Default clinical rules have been migrated from in-memory to PostgreSQL",
		"rules": func() []gin.H {
			rulesSummary := make([]gin.H, len(defaultRules))
			for i, rule := range defaultRules {
				rulesSummary[i] = gin.H{
					"id":       rule.ID,
					"name":     rule.Name,
					"domain":   rule.Domain,
					"severity": rule.Severity,
					"enabled":  rule.Enabled,
					"priority": rule.Priority,
				}
			}
			return rulesSummary
		}(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetClinicalRules handles GET /v1/cdss/rules
// Returns all clinical rules from the database (or defaults if DB empty)
func (h *CDSSHandlers) GetClinicalRules(c *gin.Context) {
	ctx := c.Request.Context()

	var rules []cdss.ClinicalRule
	var source string

	// Try to get rules from database if repository is configured
	if h.ruleRepository != nil {
		dbRules, err := h.ruleRepository.GetAllEnabledRules(ctx)
		if err == nil && len(dbRules) > 0 {
			rules = dbRules
			source = "database"
		}
	}

	// Fall back to in-memory defaults if database empty or unavailable
	if len(rules) == 0 {
		rules = cdss.GetDefaultClinicalRules()
		source = "in_memory_defaults"
	}

	// Filter by domain if specified
	domainFilter := c.Query("domain")
	if domainFilter != "" {
		filteredRules := make([]cdss.ClinicalRule, 0)
		for _, rule := range rules {
			if string(rule.Domain) == domainFilter {
				filteredRules = append(filteredRules, rule)
			}
		}
		rules = filteredRules
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"source":  source,
		"rules":   rules,
		"total":   len(rules),
	})
}

// GetClinicalRuleByID handles GET /v1/cdss/rules/:id
// Returns a specific clinical rule by ID
func (h *CDSSHandlers) GetClinicalRuleByID(c *gin.Context) {
	ruleID := c.Param("id")

	ctx := c.Request.Context()

	// Try database first
	if h.ruleRepository != nil {
		rule, err := h.ruleRepository.GetRuleByID(ctx, ruleID)
		if err == nil && rule != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"source":  "database",
				"rule":    rule,
			})
			return
		}
	}

	// Fall back to in-memory defaults
	for _, rule := range cdss.GetDefaultClinicalRules() {
		if rule.ID == ruleID {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"source":  "in_memory_defaults",
				"rule":    rule,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   "Rule not found",
		"rule_id": ruleID,
	})
}
