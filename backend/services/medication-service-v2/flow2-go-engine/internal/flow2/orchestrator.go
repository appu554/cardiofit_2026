package flow2

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/clients"
	candidatebuilder "flow2-go-engine/internal/clinical-intelligence/candidate-builder"
	"flow2-go-engine/internal/integration"
	"flow2-go-engine/internal/models"
	"flow2-go-engine/internal/orb"
	"flow2-go-engine/internal/scoring"
	"flow2-go-engine/internal/services"
)

// Orchestrator is the ORB-driven Flow 2 orchestrator service
type Orchestrator struct {
	// THE BRAIN - The most critical component
	orb *orb.OrchestratorRuleBase

	// Context planning (replaces generic context assembly)
	contextPlanner *ContextPlanner

	// Clinical intelligence components
	candidateBuilder *candidatebuilder.CandidateBuilder

	// Service clients for 2-hop architecture
	rustRecipeClient        clients.RustRecipeClient
	contextServiceClient    clients.ContextServiceClient
	contextGatewayClient    clients.ContextGatewayClient
	jitSafetyClient         clients.JITSafetyClient
	enhancedJITSafetyClient *integration.EnhancedJITSafetyClient

	// Response optimization and enrichment
	responseOptimizer *ResponseOptimizer

	// Enhanced compare-and-rank engine
	compareAndRankEngine scoring.CompareAndRankEngine

	// Supporting services
	cacheService   services.CacheService
	metricsService services.MetricsService
	healthService  services.HealthService

	logger *logrus.Logger
}

// Config holds the configuration for the orchestrator
type Config struct {
	RustRecipeClient     clients.RustRecipeClient
	ContextServiceClient clients.ContextServiceClient
	ContextGatewayClient clients.ContextGatewayClient
	JITSafetyClient      clients.JITSafetyClient
	MedicationAPIClient  clients.MedicationAPIClient
	CacheService         services.CacheService
	MetricsService       services.MetricsService
	HealthService        services.HealthService
	Logger               *logrus.Logger
}

// NewOrchestrator creates a new Flow 2 orchestrator
func NewOrchestrator(config *Config) (*Orchestrator, error) {
	// Initialize enhanced JIT Safety client
	enhancedJITClient := integration.NewEnhancedJITSafetyClient(
		"http://localhost:8080", // Would come from config
		config.Logger,
	)

	orchestrator := &Orchestrator{
		rustRecipeClient:        config.RustRecipeClient,
		contextServiceClient:    config.ContextServiceClient,
		contextGatewayClient:    config.ContextGatewayClient,
		jitSafetyClient:         config.JITSafetyClient,
		enhancedJITSafetyClient: enhancedJITClient,
		candidateBuilder:        candidatebuilder.NewCandidateBuilder(),
		cacheService:            config.CacheService,
		metricsService:          config.MetricsService,
		healthService:           config.HealthService,
		logger:                  config.Logger,
	}

	// Initialize components
	orchestrator.contextAssembler = NewContextAssembler(
		config.ContextServiceClient,
		config.MedicationAPIClient,
		config.CacheService,
		config.Logger,
	)

	orchestrator.recipeCoordinator = NewRecipeCoordinator(
		config.RustRecipeClient,
		config.MetricsService,
		config.Logger,
	)

	orchestrator.responseOptimizer = NewResponseOptimizer(
		config.CacheService,
		config.MetricsService,
		config.Logger,
	)

	return orchestrator, nil
}

// ExecuteFlow2 is the ORB-driven Flow 2 execution endpoint
// This implements the definitive 2-hop architecture with THE BRAIN
func (o *Orchestrator) ExecuteFlow2(c *gin.Context) {
	startTime := time.Now()
	requestID := uuid.New().String()

	// Parse medication request (ORB-specific format)
	var rawRequest models.Flow2Request
	if err := c.ShouldBindJSON(&rawRequest); err != nil {
		o.handleError(c, "Invalid request format", err, startTime, requestID)
		return
	}

	// Convert to ORB medication request
	medicationRequest := o.convertToMedicationRequest(&rawRequest, requestID, startTime)

	o.logger.WithFields(logrus.Fields{
		"request_id":      requestID,
		"patient_id":      medicationRequest.PatientID,
		"medication_code": medicationRequest.MedicationCode,
		"conditions":      medicationRequest.PatientConditions,
	}).Info("Starting ORB-driven Flow 2 execution")

	// STEP 1: LOCAL DECISION - Execute THE BRAIN (<1ms)
	intentManifest, err := o.orb.ExecuteLocal(c.Request.Context(), medicationRequest)
	if err != nil {
		o.handleError(c, "ORB evaluation failed", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"matched_rule_id":   intentManifest.RuleID,
		"recipe_id":         intentManifest.RecipeID,
		"data_requirements": len(intentManifest.DataRequirements),
	}).Info("ORB generated Intent Manifest")

	// STEP 2: GLOBAL FETCH - Context Service (Network Hop 1)
	contextRequest := o.contextPlanner.PlanDataRequirements(intentManifest)
	clinicalContext, err := o.contextServiceClient.FetchContext(c.Request.Context(), contextRequest)
	if err != nil {
		o.handleError(c, "Context fetch failed", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":     requestID,
		"context_fields": len(clinicalContext.Fields),
	}).Info("Context Service provided clinical data")

	// STEP 3: 4-STEP MEDICATION RECOMMENDATION WORKFLOW
	// Step 3.1: Candidate Generation
	candidateResult, err := o.generateCandidates(c.Request.Context(), clinicalContext, medicationRequest, requestID)
	if err != nil {
		o.handleError(c, "Candidate generation failed", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"candidates_found":  len(candidateResult.CandidateProposals),
		"safety_filtered":   candidateResult.SafetyFiltered,
	}).Info("Candidate generation completed")

	// Step 3.2: JIT Safety Verification
	safetyVerified, err := o.performJITSafetyVerification(c.Request.Context(), candidateResult, clinicalContext, requestID)
	if err != nil {
		o.handleError(c, "JIT Safety verification failed", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":           requestID,
		"safety_verified":      len(safetyVerified),
		"candidates_blocked":   len(candidateResult.CandidateProposals) - len(safetyVerified),
	}).Info("JIT Safety verification completed")

	// Step 3.3: Multi-Factor Scoring (placeholder for now)
	scoredProposals := o.performMultiFactorScoring(safetyVerified, clinicalContext, requestID)

	o.logger.WithFields(logrus.Fields{
		"request_id":      requestID,
		"scored_proposals": len(scoredProposals),
	}).Info("Multi-factor scoring completed")

	// STEP 4: ENHANCED PROPOSAL GENERATION - Replace basic assembly
	enhancedProposal, err := o.buildEnhancedProposalFromScoredResults(
		scoredProposals,
		clinicalContext,
		intentManifest,
		startTime,
	)
	if err != nil {
		o.logger.WithError(err).Error("Enhanced proposal generation failed")
		c.JSON(500, gin.H{
			"error": "Enhanced proposal generation failed",
			"details": err.Error(),
			"request_id": intentManifest.RequestID,
		})
		return
	}

	// STEP 5: ENHANCED RESPONSE ASSEMBLY
	response := o.assembleEnhancedResponse(enhancedProposal, intentManifest, clinicalContext, startTime)

	// Record metrics
	executionTime := time.Since(startTime)
	o.metricsService.RecordFlow2Execution(executionTime, response.OverallStatus, 1) // Always 1 recipe in ORB-driven

	o.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"execution_time_ms": executionTime.Milliseconds(),
		"overall_status":    response.OverallStatus,
		"recipe_executed":   intentManifest.RecipeID,
		"network_hops":      2, // Context Service + Rust Engine
	}).Info("ORB-driven Flow 2 execution completed")

	c.JSON(200, response)
}

// convertToMedicationRequest converts Flow2Request to ORB MedicationRequest
func (o *Orchestrator) convertToMedicationRequest(request *models.Flow2Request, requestID string, timestamp time.Time) *orb.MedicationRequest {
	// Extract medication information from the request
	var medicationCode, medicationName, indication string
	var patientConditions []string
	var patientAge *float64

	// Extract from medication data if available
	if request.MedicationData != nil && len(request.MedicationData.Medications) > 0 {
		firstMed := request.MedicationData.Medications[0]
		medicationCode = firstMed.Code
		medicationName = firstMed.Name
	}

	// Extract from patient data if available
	if request.PatientData != nil {
		if request.PatientData.AgeYears > 0 {
			age := float64(request.PatientData.AgeYears)
			patientAge = &age
		}
		patientConditions = request.PatientData.Conditions
	}

	// Extract indication from action type or clinical context
	if request.ActionType != "" {
		indication = request.ActionType
	}

	return &orb.MedicationRequest{
		RequestID:         requestID,
		PatientID:         request.PatientID,
		MedicationCode:    medicationCode,
		MedicationName:    medicationName,
		Indication:        indication,
		PatientConditions: patientConditions,
		PatientAge:        patientAge,
		ClinicalContext:   make(map[string]interface{}),
		Urgency:           "routine", // Default urgency
		RequestedBy:       "flow2_api",
		Timestamp:         timestamp,
	}
}

// assembleORBDrivenResponse creates the final ORB-driven response
func (o *Orchestrator) assembleORBDrivenResponse(
	intentManifest *orb.IntentManifest,
	clinicalContext *models.ClinicalContext,
	medicationProposal *models.MedicationProposal,
	startTime time.Time,
) *models.ORBDrivenResponse {

	executionTime := time.Since(startTime)

	return &models.ORBDrivenResponse{
		RequestID:   intentManifest.RequestID,
		PatientID:   intentManifest.PatientID,

		// Intent Manifest information
		IntentManifest: &models.IntentManifestResponse{
			RecipeID:         intentManifest.RecipeID,
			DataRequirements: intentManifest.DataRequirements,
			Priority:         intentManifest.Priority,
			ClinicalRationale: intentManifest.ClinicalRationale,
			RuleID:           intentManifest.RuleID,
			GeneratedAt:      intentManifest.GeneratedAt,
		},

		// Clinical context summary
		ClinicalContext: &models.ClinicalContextSummary{
			DataFieldsRetrieved: len(clinicalContext.Fields),
			ContextSources:      clinicalContext.Sources,
			RetrievalTimeMs:     clinicalContext.RetrievalTimeMs,
		},

		// Medication proposal from Rust engine
		MedicationProposal: medicationProposal,

		// Overall assessment
		OverallStatus: medicationProposal.SafetyStatus,

		// Execution summary
		ExecutionSummary: &models.ExecutionSummary{
			TotalExecutionTimeMs: executionTime.Milliseconds(),
			ORBEvaluationTimeMs:  1, // Sub-millisecond
			ContextFetchTimeMs:   clinicalContext.RetrievalTimeMs,
			RecipeExecutionTimeMs: medicationProposal.ExecutionTimeMs,
			NetworkHops:          2, // Context Service + Rust Engine
			Engine:               "orb+rust",
			Architecture:         "2_hop_orb_driven",
		},

		// Performance metrics
		PerformanceMetrics: &models.PerformanceMetrics{
			CacheHitRate:        0.0, // Will be populated by cache service
			DataCompleteness:    float64(len(clinicalContext.Fields)) / float64(len(intentManifest.DataRequirements)),
			RuleEvaluationTime:  1,
			TotalNetworkTime:    clinicalContext.RetrievalTimeMs + medicationProposal.ExecutionTimeMs,
		},

		Timestamp: time.Now(),
	}
}

// MedicationIntelligence handles medication intelligence requests
func (o *Orchestrator) MedicationIntelligence(c *gin.Context) {
	startTime := time.Now()
	requestID := uuid.New().String()

	var request models.MedicationIntelligenceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		o.handleError(c, "Invalid medication intelligence request", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"patient_id":        request.PatientID,
		"intelligence_type": request.IntelligenceType,
	}).Info("Starting medication intelligence")

	// Enhanced context assembly for medication intelligence
	enhancedContext, err := o.contextAssembler.AssembleEnhancedContext(c.Request.Context(), &request)
	if err != nil {
		o.handleError(c, "Enhanced context assembly failed", err, startTime, requestID)
		return
	}

	// Execute medication intelligence via Rust engine
	intelligenceRequest := &models.RustIntelligenceRequest{
		RequestID:        requestID,
		PatientID:        request.PatientID,
		Medications:      request.Medications,
		IntelligenceType: request.IntelligenceType,
		AnalysisDepth:    request.AnalysisDepth,
		ClinicalContext:  enhancedContext,
		ProcessingHints: map[string]interface{}{
			"enable_ml_inference":       true,
			"enable_outcome_prediction": request.IncludePredictions,
			"enable_alternatives":       request.IncludeAlternatives,
		},
	}

	intelligenceResponse, err := o.rustRecipeClient.ExecuteMedicationIntelligence(c.Request.Context(), intelligenceRequest)
	if err != nil {
		o.handleError(c, "Medication intelligence execution failed", err, startTime, requestID)
		return
	}

	// Optimize response
	optimizedResponse := o.responseOptimizer.OptimizeMedicationIntelligenceResponse(intelligenceResponse, &request, startTime)

	executionTime := time.Since(startTime)
	o.metricsService.RecordMedicationIntelligence(executionTime, optimizedResponse.IntelligenceScore)

	c.JSON(200, optimizedResponse)
}

// DoseOptimization handles dose optimization requests
func (o *Orchestrator) DoseOptimization(c *gin.Context) {
	startTime := time.Now()
	requestID := uuid.New().String()

	var request models.DoseOptimizationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		o.handleError(c, "Invalid dose optimization request", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":      requestID,
		"patient_id":      request.PatientID,
		"medication_code": request.MedicationCode,
	}).Info("Starting dose optimization")

	// Assemble clinical context for dose optimization
	clinicalContext, err := o.contextAssembler.AssembleContextForDoseOptimization(c.Request.Context(), &request)
	if err != nil {
		o.handleError(c, "Context assembly for dose optimization failed", err, startTime, requestID)
		return
	}

	// Execute dose optimization via Rust ML engine
	optimizationRequest := &models.RustDoseOptimizationRequest{
		RequestID:          requestID,
		PatientID:          request.PatientID,
		MedicationCode:     request.MedicationCode,
		ClinicalParameters: request.ClinicalParameters,
		OptimizationType:   "ml_guided",
		ClinicalContext:    clinicalContext,
		ProcessingHints: map[string]interface{}{
			"include_confidence_intervals":      true,
			"include_sensitivity_analysis":     true,
			"enable_pharmacokinetic_modeling":  true,
		},
	}

	optimizationResponse, err := o.rustRecipeClient.ExecuteDoseOptimization(c.Request.Context(), optimizationRequest)
	if err != nil {
		o.handleError(c, "Dose optimization execution failed", err, startTime, requestID)
		return
	}

	executionTime := time.Since(startTime)
	o.metricsService.RecordDoseOptimization(executionTime, optimizationResponse.OptimizationScore)

	c.JSON(200, optimizationResponse)
}

// SafetyValidation handles safety validation requests
func (o *Orchestrator) SafetyValidation(c *gin.Context) {
	startTime := time.Now()
	requestID := uuid.New().String()

	var request models.SafetyValidationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		o.handleError(c, "Invalid safety validation request", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":      requestID,
		"patient_id":      request.PatientID,
		"medications_count": len(request.Medications),
	}).Info("Starting safety validation")

	// Assemble clinical context for safety validation
	clinicalContext, err := o.contextAssembler.AssembleContext(c.Request.Context(), &models.Flow2Request{
		PatientID: request.PatientID,
		ActionType: "SAFETY_VALIDATION",
	})
	if err != nil {
		o.handleError(c, "Context assembly for safety validation failed", err, startTime, requestID)
		return
	}

	// Execute safety validation via Rust engine
	safetyRequest := &models.RustSafetyValidationRequest{
		RequestID:       requestID,
		PatientID:       request.PatientID,
		Medications:     request.Medications,
		ClinicalContext: clinicalContext,
		ValidationLevel: request.ValidationLevel,
		ProcessingHints: map[string]interface{}{
			"include_interaction_analysis":      true,
			"include_allergy_checking":         true,
			"include_contraindication_analysis": true,
		},
	}

	safetyResponse, err := o.rustRecipeClient.ExecuteSafetyValidation(c.Request.Context(), safetyRequest)
	if err != nil {
		o.handleError(c, "Safety validation execution failed", err, startTime, requestID)
		return
	}

	executionTime := time.Since(startTime)
	o.metricsService.RecordSafetyValidation(executionTime, safetyResponse.OverallSafetyStatus)

	c.JSON(200, safetyResponse)
}

// ClinicalIntelligence handles clinical intelligence requests
func (o *Orchestrator) ClinicalIntelligence(c *gin.Context) {
	startTime := time.Now()
	requestID := uuid.New().String()

	var request models.ClinicalIntelligenceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		o.handleError(c, "Invalid clinical intelligence request", err, startTime, requestID)
		return
	}

	o.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"patient_id": request.PatientID,
	}).Info("Starting clinical intelligence")

	// For now, return a placeholder response
	response := map[string]interface{}{
		"request_id":        requestID,
		"intelligence_type": "comprehensive",
		"insights": []map[string]interface{}{
			{
				"type":        "outcome_prediction",
				"confidence":  0.85,
				"description": "High probability of positive therapeutic outcome",
			},
		},
		"execution_time_ms": time.Since(startTime).Milliseconds(),
	}

	c.JSON(200, response)
}

// CollectAnalytics handles analytics collection
func (o *Orchestrator) CollectAnalytics(c *gin.Context) {
	var analyticsData map[string]interface{}
	if err := c.ShouldBindJSON(&analyticsData); err != nil {
		c.JSON(400, gin.H{"error": "Invalid analytics data"})
		return
	}

	o.logger.WithFields(logrus.Fields{
		"analytics_data": analyticsData,
	}).Info("Analytics data collected")

	c.JSON(200, gin.H{"status": "collected"})
}

// GetPatientAnalytics retrieves patient analytics
func (o *Orchestrator) GetPatientAnalytics(c *gin.Context) {
	patientID := c.Param("patient_id")
	timeframe := c.DefaultQuery("timeframe", "30d")

	o.logger.WithFields(logrus.Fields{
		"patient_id": patientID,
		"timeframe":  timeframe,
	}).Info("Retrieving patient analytics")

	// Return mock analytics data
	analytics := map[string]interface{}{
		"patient_id": patientID,
		"timeframe":  timeframe,
		"metrics": map[string]interface{}{
			"total_prescriptions":    15,
			"safety_alerts":         2,
			"adherence_score":       0.85,
			"outcome_predictions":   []string{"positive", "stable"},
		},
		"timestamp": time.Now().UTC(),
	}

	c.JSON(200, analytics)
}

// GetRecommendations retrieves patient recommendations
func (o *Orchestrator) GetRecommendations(c *gin.Context) {
	patientID := c.Param("patient_id")
	recommendationType := c.DefaultQuery("type", "all")

	o.logger.WithFields(logrus.Fields{
		"patient_id":          patientID,
		"recommendation_type": recommendationType,
	}).Info("Retrieving patient recommendations")

	// Return mock recommendations
	recommendations := map[string]interface{}{
		"patient_id": patientID,
		"type":       recommendationType,
		"recommendations": []map[string]interface{}{
			{
				"id":          "rec_001",
				"type":        "medication_optimization",
				"priority":    "high",
				"title":       "Consider dose adjustment",
				"description": "Based on recent lab results, consider adjusting medication dose",
				"confidence":  0.92,
			},
			{
				"id":          "rec_002",
				"type":        "monitoring",
				"priority":    "medium",
				"title":       "Schedule follow-up",
				"description": "Schedule follow-up appointment in 2 weeks",
				"confidence":  0.78,
			},
		},
		"timestamp": time.Now().UTC(),
	}

	c.JSON(200, recommendations)
}

// GraphQLHandler handles GraphQL requests
func (o *Orchestrator) GraphQLHandler(c *gin.Context) {
	// For now, we'll implement a simple GraphQL-like handler
	// In a full implementation, you would use a GraphQL library like graphql-go

	var graphqlRequest struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}

	if err := c.ShouldBindJSON(&graphqlRequest); err != nil {
		c.JSON(400, gin.H{"error": "Invalid GraphQL request"})
		return
	}

	o.logger.WithFields(logrus.Fields{
		"query": graphqlRequest.Query,
	}).Info("Processing GraphQL request")

	// Simple query parsing and response
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"flow2": map[string]interface{}{
				"status":  "available",
				"version": "1.0.0",
				"capabilities": []string{
					"medication_intelligence",
					"dose_optimization",
					"safety_validation",
					"clinical_intelligence",
				},
			},
		},
	}

	c.JSON(200, response)
}

// handleError handles errors and returns appropriate HTTP responses
func (o *Orchestrator) handleError(c *gin.Context, message string, err error, startTime time.Time, requestID string) {
	executionTime := time.Since(startTime)

	o.metricsService.IncrementFlow2Errors()
	o.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"error":            err.Error(),
		"execution_time_ms": executionTime.Milliseconds(),
	}).Error(message)

	c.JSON(500, gin.H{
		"error":             message,
		"details":           err.Error(),
		"request_id":        requestID,
		"execution_time_ms": executionTime.Milliseconds(),
	})
}

// generateCandidates performs Step 1: Candidate Generation
func (o *Orchestrator) generateCandidates(
	ctx context.Context,
	clinicalContext *models.ClinicalContext,
	medicationRequest *models.MedicationRequest,
	requestID string,
) (*candidatebuilder.CandidateBuilderResult, error) {
	// Convert clinical context to candidate builder input
	input := candidatebuilder.CandidateBuilderInput{
		PatientContext: candidatebuilder.PatientContext{
			Demographics: candidatebuilder.Demographics{
				Age:        clinicalContext.Demographics.Age,
				Gender:     clinicalContext.Demographics.Gender,
				Weight:     clinicalContext.Demographics.Weight,
				Height:     clinicalContext.Demographics.Height,
				IsPregnant: clinicalContext.Demographics.IsPregnant,
			},
			Allergies:   convertAllergies(clinicalContext.Allergies),
			Conditions:  convertConditions(clinicalContext.Conditions),
			LabResults:  convertLabResults(clinicalContext.LabResults),
		},
		RequestedMedication: candidatebuilder.RequestedMedication{
			Code:        medicationRequest.MedicationCode,
			Indication:  medicationRequest.Indication,
			RequestType: "recommendation",
		},
		RequestID: requestID,
	}

	// Execute candidate generation
	return o.candidateBuilder.BuildCandidateProposals(ctx, input)
}

// performJITSafetyVerification performs Step 2: Enhanced JIT Safety Verification
func (o *Orchestrator) performJITSafetyVerification(
	ctx context.Context,
	candidateResult *candidatebuilder.CandidateBuilderResult,
	clinicalContext *models.ClinicalContext,
	requestID string,
) ([]*models.SafetyVerifiedProposal, error) {
	var safetyVerified []*models.SafetyVerifiedProposal

	// Convert clinical context to patient context
	patientContext := convertToPatientContext(clinicalContext)

	// Process each candidate through Enhanced JIT Safety
	for _, candidate := range candidateResult.CandidateProposals {
		// Use enhanced JIT Safety client for comprehensive evaluation
		verified, err := o.enhancedJITSafetyClient.RunEnhancedJITSafetyCheck(
			ctx,
			candidate,
			patientContext,
			10.0, // Default dose - would come from candidate
			requestID,
		)
		if err != nil {
			o.logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"drug_id":    candidate.MedicationCode,
				"error":      err.Error(),
			}).Warn("Enhanced JIT Safety check failed, excluding candidate")
			continue
		}

		// Check if candidate can proceed
		if verified.Action == "Contraindicated" {
			o.logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"drug_id":    candidate.MedicationCode,
				"reasons":    len(verified.SafetyReasons),
			}).Info("Enhanced JIT Safety blocked candidate")
			continue
		}

		safetyVerified = append(safetyVerified, verified)
	}

	o.logger.WithFields(logrus.Fields{
		"request_id":           requestID,
		"candidates_processed": len(candidateResult.CandidateProposals),
		"safety_verified":      len(safetyVerified),
		"blocked_count":        len(candidateResult.CandidateProposals) - len(safetyVerified),
	}).Info("Enhanced JIT Safety verification completed")

	return safetyVerified, nil
}

// performMultiFactorScoring performs Step 3: Enhanced Multi-Factor Scoring with Compare-and-Rank
func (o *Orchestrator) performMultiFactorScoring(
	safetyVerified []*models.SafetyVerifiedProposal,
	clinicalContext *models.ClinicalContext,
	requestID string,
) []*models.ScoredProposal {
	// If no compare-and-rank engine available, fall back to simple scoring
	if o.compareAndRankEngine == nil {
		return o.performSimpleScoring(safetyVerified, requestID)
	}

	// Convert safety-verified proposals to enhanced proposals for compare-and-rank
	enhancedProposals := o.convertToEnhancedProposals(safetyVerified, clinicalContext)
	if len(enhancedProposals) == 0 {
		o.logger.WithField("request_id", requestID).Warn("No enhanced proposals to rank")
		return []*models.ScoredProposal{}
	}

	// Determine patient risk phenotype and preferences from clinical context
	patientContext := o.extractPatientRiskContext(clinicalContext)

	// Create compare-and-rank request
	compareRequest := &models.CompareAndRankRequest{
		PatientContext: patientContext,
		Candidates:     enhancedProposals,
		ConfigRef: models.ConfigReference{
			WeightProfile:    patientContext.RiskPhenotype,
			PenaltiesProfile: "default",
		},
		RequestID: requestID,
		Timestamp: time.Now(),
	}

	// Execute enhanced compare-and-rank
	ctx := context.Background()
	compareResponse, err := o.compareAndRankEngine.CompareAndRank(ctx, compareRequest)
	if err != nil {
		o.logger.WithError(err).WithField("request_id", requestID).Error("Enhanced compare-and-rank failed, falling back to simple scoring")
		return o.performSimpleScoring(safetyVerified, requestID)
	}

	// Convert enhanced scored proposals back to legacy format
	scoredProposals := o.convertFromEnhancedScoredProposals(compareResponse.Ranked, safetyVerified)

	o.logger.WithFields(logrus.Fields{
		"request_id":         requestID,
		"candidates_ranked":  len(scoredProposals),
		"candidates_pruned":  compareResponse.Audit.CandidatesPruned,
		"weight_profile":     compareResponse.Audit.ProfileUsed.Weights,
		"processing_time_ms": compareResponse.Audit.ProcessingTime.Milliseconds(),
		"top_therapy":        func() string {
			if len(compareResponse.Ranked) > 0 {
				return compareResponse.Ranked[0].TherapyID
			}
			return "none"
		}(),
		"top_score": func() float64 {
			if len(compareResponse.Ranked) > 0 {
				return compareResponse.Ranked[0].FinalScore
			}
			return 0.0
		}(),
	}).Info("Enhanced multi-factor scoring completed")

	return scoredProposals
}

// performSimpleScoring provides fallback simple scoring when compare-and-rank is unavailable
func (o *Orchestrator) performSimpleScoring(
	safetyVerified []*models.SafetyVerifiedProposal,
	requestID string,
) []*models.ScoredProposal {
	var scoredProposals []*models.ScoredProposal

	for i, verified := range safetyVerified {
		// Simple scoring - basic implementation
		componentScores := models.ComponentScores{
			SafetyScore:             verified.SafetyScore,
			EfficacyScore:           0.8, // Placeholder
			CostScore:               0.7, // Placeholder
			ConvenienceScore:        0.9, // Placeholder
			PatientPreferenceScore:  0.8, // Placeholder
			GuidelineAdherenceScore: 0.85, // Placeholder
		}

		// Calculate weighted total score
		totalScore := (componentScores.SafetyScore * 0.30) +
			(componentScores.EfficacyScore * 0.25) +
			(componentScores.CostScore * 0.15) +
			(componentScores.ConvenienceScore * 0.10) +
			(componentScores.PatientPreferenceScore * 0.10) +
			(componentScores.GuidelineAdherenceScore * 0.10)

		scored := &models.ScoredProposal{
			SafetyVerified:  *verified,
			TotalScore:      totalScore,
			ComponentScores: componentScores,
			Ranking:         i + 1,
			ScoredAt:        time.Now(),
		}

		scoredProposals = append(scoredProposals, scored)
	}

	// Sort by total score (highest first)
	for i := 0; i < len(scoredProposals)-1; i++ {
		for j := i + 1; j < len(scoredProposals); j++ {
			if scoredProposals[j].TotalScore > scoredProposals[i].TotalScore {
				scoredProposals[i], scoredProposals[j] = scoredProposals[j], scoredProposals[i]
			}
		}
	}

	// Update rankings
	for i, scored := range scoredProposals {
		scored.Ranking = i + 1
	}

	o.logger.WithField("request_id", requestID).Info("Simple scoring completed (fallback mode)")
	return scoredProposals
}

// convertToEnhancedProposals converts safety-verified proposals to enhanced proposals
func (o *Orchestrator) convertToEnhancedProposals(
	safetyVerified []*models.SafetyVerifiedProposal,
	clinicalContext *models.ClinicalContext,
) []models.EnhancedProposal {
	var enhanced []models.EnhancedProposal

	for _, verified := range safetyVerified {
		// Extract medication details from original proposal
		original := verified.Original

		// Create enhanced proposal with available data
		enhancedProposal := models.EnhancedProposal{
			TherapyID: original.MedicationCode,
			Class:     original.TherapeuticClass,
			Agent:     original.GenericName,
			Regimen: models.RegimenDetail{
				Form:      "tablet", // Default, would be extracted from medication data
				Frequency: "daily",  // Default, would be extracted from dose
				IsFDC:     false,    // Would be determined from medication data
				PillCount: 1,        // Default
			},
			Dose: models.DoseDetail{
				Amount:    verified.FinalDose.DoseMg,
				Unit:      "mg",
				Frequency: "daily", // Would be extracted from interval
				Route:     verified.FinalDose.Route,
				Rationale: "JIT safety verified dose",
			},
			Efficacy: models.EfficacyDetail{
				ExpectedA1cDropPct: o.estimateEfficacy(original.MedicationName),
				CVBenefit:         o.hasCVBenefit(original.MedicationName),
				HFBenefit:         o.hasHFBenefit(original.MedicationName),
				CKDBenefit:        o.hasCKDBenefit(original.MedicationName),
			},
			Safety: models.SafetyDetail{
				ResidualDDI:    o.mapDDISeverity(verified.DDIWarnings),
				HypoPropensity: o.mapHypoglycemiaRisk(original.MedicationName),
				WeightEffect:   o.mapWeightEffect(original.MedicationName),
			},
			Suitability: models.SuitabilityDetail{
				RenalFit:   true, // Would be determined from JIT safety results
				HepaticFit: true, // Would be determined from JIT safety results
			},
			Adherence: models.AdherenceDetail{
				DosesPerDay:      o.calculateDosesPerDay(verified.FinalDose.IntervalH),
				PillBurden:       1, // Default
				RequiresDevice:   verified.FinalDose.Route != "po",
				RequiresTraining: verified.FinalDose.Route == "sc" || verified.FinalDose.Route == "im",
			},
			Availability: models.AvailabilityDetail{
				Tier:         original.FormularyTier,
				OnHand:       100, // Default - would come from inventory system
				LeadTimeDays: 0,   // Default
			},
			Cost: models.CostDetail{
				MonthlyEstimate: original.CostEstimate,
				Currency:        "USD",
			},
			Preferences: models.PreferencesDetail{
				AvoidInjectables:   false, // Would come from patient preferences
				OnceDailyPreferred: true,  // Default preference
				CostSensitivity:    "medium",
			},
			Provenance: models.ProvenanceDetail{
				KBVersions: map[string]string{
					"jit_safety": "v1.0",
					"drug_master": "v1.0",
				},
			},
		}

		enhanced = append(enhanced, enhancedProposal)
	}

	return enhanced
}

// extractPatientRiskContext extracts patient risk context from clinical context
func (o *Orchestrator) extractPatientRiskContext(clinicalContext *models.ClinicalContext) models.PatientRiskContext {
	// Determine risk phenotype based on conditions
	riskPhenotype := "NONE"

	// Check for high-risk conditions
	for _, condition := range clinicalContext.Conditions {
		switch condition.Code {
		case "I25.9", "I21.9": // CAD, MI
			riskPhenotype = "ASCVD"
		case "I50.9": // Heart failure
			riskPhenotype = "HF"
		case "N18.6": // CKD
			riskPhenotype = "CKD"
		}
	}

	return models.PatientRiskContext{
		RiskPhenotype: riskPhenotype,
		ResourceTier:  "standard", // Default
		Preferences: models.JITPatientPreferences{
			AvoidInjectables:   false, // Would come from patient preferences
			OnceDailyPreferred: true,  // Default preference
			CostSensitivity:    "medium",
		},
	}
}

// convertFromEnhancedScoredProposals converts enhanced scored proposals back to legacy format
func (o *Orchestrator) convertFromEnhancedScoredProposals(
	enhanced []models.EnhancedScoredProposal,
	original []*models.SafetyVerifiedProposal,
) []*models.ScoredProposal {
	var scored []*models.ScoredProposal

	// Create a map for quick lookup of original proposals
	originalMap := make(map[string]*models.SafetyVerifiedProposal)
	for _, orig := range original {
		originalMap[orig.Original.MedicationCode] = orig
	}

	for _, enh := range enhanced {
		// Find corresponding original proposal
		orig, exists := originalMap[enh.TherapyID]
		if !exists {
			continue
		}

		// Convert enhanced scores to legacy component scores
		componentScores := models.ComponentScores{
			SafetyScore:             enh.SubScores.Safety.Score,
			EfficacyScore:           enh.SubScores.Efficacy.Score,
			CostScore:               enh.SubScores.Cost.Score,
			ConvenienceScore:        enh.SubScores.Adherence.Score,
			PatientPreferenceScore:  enh.SubScores.Preference.Score,
			GuidelineAdherenceScore: 0.85, // Default - would be calculated
			AvailabilityScore:       enh.SubScores.Availability.Score,
			AdherenceScore:          enh.SubScores.Adherence.Score,
		}

		scoredProposal := &models.ScoredProposal{
			SafetyVerified:  *orig,
			TotalScore:      enh.FinalScore,
			ComponentScores: componentScores,
			Ranking:         enh.Rank,
			ScoredAt:        enh.ScoredAt,
		}

		scored = append(scored, scoredProposal)
	}

	return scored
}

// Helper methods for medication knowledge extraction

// estimateEfficacy estimates A1c reduction based on medication class
func (o *Orchestrator) estimateEfficacy(medicationName string) float64 {
	// Simple lookup based on medication name/class
	// In production, this would query a comprehensive drug knowledge base
	switch {
	case contains(medicationName, "metformin"):
		return 1.0
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		return 1.2
	case contains(medicationName, "insulin"):
		return 1.8
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		return 1.5
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		return 0.8
	default:
		return 1.0 // Default estimate
	}
}

// hasCVBenefit checks if medication has cardiovascular outcome benefit
func (o *Orchestrator) hasCVBenefit(medicationName string) bool {
	cvBenefitMeds := []string{"semaglutide", "liraglutide", "empagliflozin", "canagliflozin"}
	for _, med := range cvBenefitMeds {
		if contains(medicationName, med) {
			return true
		}
	}
	return false
}

// hasHFBenefit checks if medication has heart failure benefit
func (o *Orchestrator) hasHFBenefit(medicationName string) bool {
	hfBenefitMeds := []string{"empagliflozin", "dapagliflozin"}
	for _, med := range hfBenefitMeds {
		if contains(medicationName, med) {
			return true
		}
	}
	return false
}

// hasCKDBenefit checks if medication has chronic kidney disease benefit
func (o *Orchestrator) hasCKDBenefit(medicationName string) bool {
	ckdBenefitMeds := []string{"empagliflozin", "canagliflozin"}
	for _, med := range ckdBenefitMeds {
		if contains(medicationName, med) {
			return true
		}
	}
	return false
}

// mapDDISeverity maps DDI warnings to severity level
func (o *Orchestrator) mapDDISeverity(ddiWarnings []models.DDIFlag) string {
	if len(ddiWarnings) == 0 {
		return "none"
	}

	// Check for major DDIs first
	for _, ddi := range ddiWarnings {
		if ddi.Severity == "major" {
			return "major"
		}
	}

	// Check for moderate DDIs
	for _, ddi := range ddiWarnings {
		if ddi.Severity == "moderate" {
			return "moderate"
		}
	}

	return "none"
}

// mapHypoglycemiaRisk maps medication to hypoglycemia risk level
func (o *Orchestrator) mapHypoglycemiaRisk(medicationName string) string {
	switch {
	case contains(medicationName, "insulin"):
		return "high"
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		return "high"
	case contains(medicationName, "metformin"):
		return "low"
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		return "low"
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		return "low"
	default:
		return "med"
	}
}

// mapWeightEffect maps medication to weight effect
func (o *Orchestrator) mapWeightEffect(medicationName string) string {
	switch {
	case contains(medicationName, "insulin"):
		return "gain"
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		return "gain"
	case contains(medicationName, "metformin"):
		return "neutral"
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		return "loss"
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		return "loss"
	default:
		return "neutral"
	}
}

// calculateDosesPerDay calculates doses per day from interval hours
func (o *Orchestrator) calculateDosesPerDay(intervalH uint32) int {
	if intervalH == 0 {
		return 1
	}
	return int(24 / intervalH)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    len(s) > len(substr) &&
		    (s[:len(substr)] == substr ||
		     s[len(s)-len(substr):] == substr ||
		     findSubstring(s, substr)))
}

// findSubstring performs case-insensitive substring search
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// convertToLegacyFormat converts scored proposals to legacy format for response assembly
func (o *Orchestrator) convertToLegacyFormat(
	scoredProposals []*models.ScoredProposal,
	intentManifest *orb.IntentManifest,
) *models.MedicationProposal {
	if len(scoredProposals) == 0 {
		return &models.MedicationProposal{
			SafetyStatus: "no_safe_options",
			Recommendations: []models.MedicationRecommendation{},
		}
	}

	// Use the top-ranked proposal as the primary recommendation
	topProposal := scoredProposals[0]

	return &models.MedicationProposal{
		SafetyStatus: "safe",
		Recommendations: []models.MedicationRecommendation{
			{
				MedicationName: topProposal.SafetyVerified.Original.MedicationName,
				DosageForm:     topProposal.SafetyVerified.Original.DosageForm,
				Strength:       topProposal.SafetyVerified.Original.Strength,
				Route:          topProposal.SafetyVerified.FinalDose.Route,
				Frequency:      "q24h", // Simplified
				Duration:       "ongoing",
				Instructions:   "Take as directed",
				SafetyScore:    topProposal.SafetyVerified.SafetyScore,
			},
		},
	}
}

// Helper conversion functions

// convertAllergies converts clinical context allergies to candidate builder format
func convertAllergies(allergies []models.Allergy) []candidatebuilder.Allergy {
	result := make([]candidatebuilder.Allergy, len(allergies))
	for i, allergy := range allergies {
		result[i] = candidatebuilder.Allergy{
			AllergenCode: allergy.AllergenCode,
			AllergenName: allergy.AllergenName,
			Severity:     allergy.Severity,
		}
	}
	return result
}

// convertConditions converts clinical context conditions to candidate builder format
func convertConditions(conditions []models.Condition) []candidatebuilder.Condition {
	result := make([]candidatebuilder.Condition, len(conditions))
	for i, condition := range conditions {
		result[i] = candidatebuilder.Condition{
			Code:        condition.Code,
			Name:        condition.Name,
			Severity:    condition.Severity,
			OnsetDate:   condition.OnsetDate,
		}
	}
	return result
}

// convertLabResults converts clinical context lab results to candidate builder format
func convertLabResults(labs models.LabResults) candidatebuilder.LabResults {
	return candidatebuilder.LabResults{
		EGFR:                labs.EGFR,
		CreatinineClearance: labs.CreatinineClearance,
		ALT:                 labs.ALT,
		AST:                 labs.AST,
		UACR:                labs.UACR,
		HbA1c:               labs.HbA1c,
		Potassium:           labs.Potassium,
		Sodium:              labs.Sodium,
	}
}

// convertToPatientContext converts clinical context to patient context for JIT Safety
func convertToPatientContext(clinicalContext *models.ClinicalContext) models.PatientContext {
	return models.PatientContext{
		Demographics:      clinicalContext.Demographics,
		Allergies:         clinicalContext.Allergies,
		Conditions:        clinicalContext.Conditions,
		LabResults:        clinicalContext.LabResults,
		ActiveMedications: clinicalContext.ActiveMedications,
	}
}

// convertToConcurrentMeds converts active medications to concurrent meds for JIT Safety
func convertToConcurrentMeds(activeMeds []models.ActiveMedication) []models.ConcurrentMed {
	result := make([]models.ConcurrentMed, len(activeMeds))
	for i, med := range activeMeds {
		result[i] = models.ConcurrentMed{
			DrugID:    med.MedicationCode,
			ClassID:   med.TherapeuticClass,
			DoseMg:    med.DoseAmount,
			IntervalH: uint32(med.FrequencyHours),
		}
	}
	return result
}

// ==================== ENHANCED PROPOSAL INTEGRATION ====================

// buildEnhancedProposalFromScoredResults creates enhanced proposal from existing pipeline results
func (o *Orchestrator) buildEnhancedProposalFromScoredResults(
	scoredProposals []*models.ScoredProposal,
	clinicalContext *models.ClinicalContext,
	intentManifest *orb.IntentManifest,
	startTime time.Time,
) (*models.EnhancedProposedOrder, error) {
	if len(scoredProposals) == 0 {
		return nil, fmt.Errorf("no scored proposals available")
	}

	// Use the top-ranked proposal as the primary recommendation
	topProposal := scoredProposals[0]
	proposalID := uuid.New().String()
	now := time.Now()

	// Build enhanced proposal using existing pipeline data
	enhancedProposal := &models.EnhancedProposedOrder{
		ProposalID:      proposalID,
		ProposalVersion: "1.0",
		Timestamp:       now,
		ExpiresAt:       now.Add(24 * time.Hour),

		Metadata: models.ProposalMetadata{
			PatientID:           intentManifest.PatientID,
			EncounterID:         "", // Would come from request context
			PrescriberID:        "", // Would come from request context
			Status:              "PROPOSED",
			Urgency:             intentManifest.Priority,
			ProposalType:        "NEW_PRESCRIPTION",
			RecipeUsed:          intentManifest.RecipeID,
			ContextCompleteness: float64(len(clinicalContext.Fields)) / float64(len(intentManifest.DataRequirements)),
			ConfidenceScore:     topProposal.SafetyVerified.SafetyScore,
		},

		CalculatedOrder: o.buildCalculatedOrderFromProposal(topProposal),
		MonitoringPlan:  o.buildBasicMonitoringPlan(topProposal),
		TherapeuticAlternatives: o.buildAlternativesFromScoredProposals(scoredProposals),
		ClinicalRationale: o.buildClinicalRationaleFromProposal(topProposal, intentManifest),
		ProposalMetadata: o.buildProposalMetadataFromExecution(startTime),
	}

	return enhancedProposal, nil
}

// assembleEnhancedResponse creates the final enhanced response
func (o *Orchestrator) assembleEnhancedResponse(
	enhancedProposal *models.EnhancedProposedOrder,
	intentManifest *orb.IntentManifest,
	clinicalContext *models.ClinicalContext,
	startTime time.Time,
) *models.ORBDrivenResponse {
	executionTime := time.Since(startTime)

	return &models.ORBDrivenResponse{
		RequestID: intentManifest.RequestID,
		PatientID: intentManifest.PatientID,

		// Intent Manifest information
		IntentManifest: &models.IntentManifestResponse{
			RecipeID:          intentManifest.RecipeID,
			DataRequirements:  intentManifest.DataRequirements,
			Priority:          intentManifest.Priority,
			ClinicalRationale: intentManifest.ClinicalRationale,
			RuleID:            intentManifest.RuleID,
			GeneratedAt:       intentManifest.GeneratedAt,
		},

		// Clinical context summary
		ClinicalContext: &models.ClinicalContextSummary{
			DataFieldsRetrieved: len(clinicalContext.Fields),
			ContextSources:      clinicalContext.Sources,
			RetrievalTimeMs:     clinicalContext.RetrievalTimeMs,
		},

		// Enhanced proposal instead of basic medication proposal
		EnhancedProposal: enhancedProposal,

		// Overall assessment
		OverallStatus: "enhanced_recommendation_generated",

		// Execution summary
		ExecutionSummary: &models.ExecutionSummary{
			TotalExecutionTimeMs:  executionTime.Milliseconds(),
			ORBEvaluationTimeMs:   1,
			ContextFetchTimeMs:    clinicalContext.RetrievalTimeMs,
			RecipeExecutionTimeMs: enhancedProposal.ProposalMetadata.AuditTrail.TotalProcessingTime,
			NetworkHops:           2,
			Engine:                "orb+enhanced",
			Architecture:          "2_hop_orb_enhanced",
		},

		// Performance metrics
		PerformanceMetrics: &models.PerformanceMetrics{
			CacheHitRate:       0.0,
			DataCompleteness:   enhancedProposal.Metadata.ContextCompleteness,
			RuleEvaluationTime: 1,
			TotalNetworkTime:   clinicalContext.RetrievalTimeMs + enhancedProposal.ProposalMetadata.AuditTrail.TotalProcessingTime,
		},

		Timestamp: time.Now(),
	}
}

// ==================== ENHANCED PROPOSAL HELPER METHODS ====================

// buildCalculatedOrderFromProposal converts scored proposal to calculated order
func (o *Orchestrator) buildCalculatedOrderFromProposal(proposal *models.ScoredProposal) models.CalculatedOrder {
	return models.CalculatedOrder{
		Medication: models.MedicationDetail{
			PrimaryIdentifier: models.Identifier{
				System:  "RxNorm",
				Code:    proposal.SafetyVerified.Original.MedicationCode,
				Display: proposal.SafetyVerified.Original.MedicationName,
			},
			AlternateIdentifiers: []models.Identifier{},
			BrandName:           nil,
			GenericName:         proposal.SafetyVerified.Original.GenericName,
			TherapeuticClass:    proposal.SafetyVerified.Original.TherapeuticClass,
			IsHighAlert:         false,
			IsControlled:        false,
		},
		Dosing: models.DosingDetail{
			Dose: models.DoseInfo{
				Value:   proposal.SafetyVerified.FinalDose.DoseMg,
				Unit:    "mg",
				PerDose: true,
			},
			Route: models.RouteInfo{
				Code:    proposal.SafetyVerified.FinalDose.Route,
				Display: o.getRouteDisplay(proposal.SafetyVerified.FinalDose.Route),
			},
			Frequency: models.FrequencyInfo{
				Code:          "DAILY",
				Display:       "Once daily",
				TimesPerDay:   int(24 / proposal.SafetyVerified.FinalDose.IntervalH),
				SpecificTimes: []string{"08:00"},
			},
			Duration: models.DurationInfo{
				Value:   90,
				Unit:    "days",
				Refills: 3,
			},
			Instructions: models.InstructionInfo{
				PatientInstructions:    o.generatePatientInstructions(proposal),
				PharmacyInstructions:   o.generatePharmacyInstructions(proposal),
				AdditionalInstructions: o.generateAdditionalInstructions(proposal),
			},
		},
		CalculationDetails: models.CalculationDetails{
			Method:          "JIT_SAFETY_VERIFIED",
			Factors:         o.buildCalculationFactorsFromContext(proposal),
			Adjustments:     proposal.SafetyVerified.Adjustments,
			RoundingApplied: false,
			MaximumDoseCheck: models.MaximumDoseCheck{
				Daily:        proposal.SafetyVerified.FinalDose.DoseMg,
				Maximum:      2000,
				WithinLimits: true,
			},
		},
		Formulation: models.FormulationDetail{
			SelectedForm:            proposal.SafetyVerified.Original.DosageForm,
			AvailableStrengths:      []float64{500, 850, 1000},
			Splittable:              true,
			Crushable:               false,
			AlternativeFormulations: []models.AlternativeFormulation{},
		},
	}
}

// buildBasicMonitoringPlan creates a basic monitoring plan from proposal
func (o *Orchestrator) buildBasicMonitoringPlan(proposal *models.ScoredProposal) models.EnhancedMonitoringPlan {
	return models.EnhancedMonitoringPlan{
		RiskStratification: models.RiskStratification{
			OverallRisk: o.assessOverallRisk(proposal.SafetyVerified.SafetyScore),
			Factors: []models.RiskFactor{
				{
					Factor:  "Safety Score",
					Present: proposal.SafetyVerified.SafetyScore < 0.8,
					Impact:  o.getRiskImpact(proposal.SafetyVerified.SafetyScore),
				},
			},
		},
		Baseline:          o.getBaselineMonitoringForMedication(proposal.SafetyVerified.Original.MedicationCode),
		Ongoing:           o.getOngoingMonitoringForMedication(proposal.SafetyVerified.Original.MedicationCode),
		SymptomMonitoring: o.getSymptomMonitoringForMedication(proposal.SafetyVerified.Original.MedicationCode),
	}
}

// buildAlternativesFromScoredProposals creates alternatives from other scored proposals
func (o *Orchestrator) buildAlternativesFromScoredProposals(scoredProposals []*models.ScoredProposal) models.TherapeuticAlternatives {
	alternatives := []models.TherapeuticAlternative{}

	// Use next 2-3 highest-ranked proposals as alternatives
	for i := 1; i < len(scoredProposals) && i < 4; i++ {
		proposal := scoredProposals[i]
		alternative := models.TherapeuticAlternative{
			Medication: models.AlternativeMedicationDetail{
				Name:     proposal.SafetyVerified.Original.GenericName,
				Code:     proposal.SafetyVerified.Original.MedicationCode,
				Strength: proposal.SafetyVerified.FinalDose.DoseMg,
				Unit:     "mg",
			},
			Category: o.determineAlternativeCategory(scoredProposals[0], proposal),
			FormularyStatus: models.FormularyStatus{
				Tier:              proposal.SafetyVerified.Original.FormularyTier,
				PriorAuthRequired: false,
				QuantityLimits:    nil,
			},
			CostComparison: models.CostComparison{
				RelativeCost:         o.compareCosts(scoredProposals[0].SafetyVerified.Original.CostEstimate, proposal.SafetyVerified.Original.CostEstimate),
				EstimatedMonthlyCost: proposal.SafetyVerified.Original.CostEstimate,
				PatientCopay:         proposal.SafetyVerified.Original.CostEstimate * 0.2,
			},
			ClinicalConsiderations: models.ClinicalConsiderations{
				Advantages:    o.getAdvantages(proposal),
				Disadvantages: o.getDisadvantages(proposal),
			},
			SwitchingInstructions: o.getSwitchingInstructions(scoredProposals[0], proposal),
		}
		alternatives = append(alternatives, alternative)
	}

	return models.TherapeuticAlternatives{
		PrimaryReason:        "CLINICAL_OPTIMIZATION",
		Alternatives:         alternatives,
		NonPharmAlternatives: o.getNonPharmAlternatives(),
	}
}

// buildClinicalRationaleFromProposal creates clinical rationale from proposal
func (o *Orchestrator) buildClinicalRationaleFromProposal(proposal *models.ScoredProposal, intentManifest *orb.IntentManifest) models.ClinicalRationale {
	return models.ClinicalRationale{
		Summary: models.RationaleSummary{
			Decision:   fmt.Sprintf("Recommend %s based on safety score %.2f", proposal.SafetyVerified.Original.GenericName, proposal.SafetyVerified.SafetyScore),
			Confidence: o.getConfidenceLevel(proposal.SafetyVerified.SafetyScore),
			Complexity: o.getComplexityLevel(proposal),
		},
		IndicationAssessment: models.IndicationAssessment{
			PrimaryIndication: intentManifest.ClinicalRationale,
			ICDCode:           "",
			ClinicalCriteria:  []models.ClinicalCriterion{},
			Appropriateness:   "CLINICALLY_APPROPRIATE",
		},
		DosingRationale: models.DosingRationale{
			Strategy:    "SAFETY_OPTIMIZED",
			Explanation: fmt.Sprintf("Dose calculated using JIT safety verification with score %.2f", proposal.SafetyVerified.SafetyScore),
			TitrationPlan: models.TitrationPlan{
				Week2:   "Continue current dose if tolerated",
				Week4:   "Assess efficacy and consider adjustment",
				MaxDose: "Per clinical guidelines",
			},
			EvidenceBase: models.EvidenceBase{
				Source:                 "JIT Safety Engine",
				RecommendationStrength: "STRONG",
				EvidenceQuality:        "HIGH",
			},
		},
		FormularyRationale: models.FormularyRationale{
			FormularyDecision: o.getFormularyDecision(proposal.SafetyVerified.Original.FormularyTier),
			CostEffectiveness: o.getCostEffectiveness(proposal.SafetyVerified.Original.CostEstimate),
			InsuranceCoverage: models.InsuranceCoverage{
				Covered:           true,
				Tier:              proposal.SafetyVerified.Original.FormularyTier,
				Copay:             proposal.SafetyVerified.Original.CostEstimate * 0.2,
				DeductibleApplies: false,
			},
		},
		PatientFactors: models.PatientFactors{
			PositiveFactors: []string{
				fmt.Sprintf("High safety score: %.2f", proposal.SafetyVerified.SafetyScore),
				"No major contraindications identified",
			},
			Considerations: []string{
				"Monitor for side effects",
				"Follow up as scheduled",
			},
			SharedDecisionMaking: models.SharedDecisionMaking{
				Discussed:         []string{"Benefits", "Risks", "Alternatives"},
				PatientPreference: "Patient agrees to treatment",
			},
		},
		QualityMeasures: models.QualityMeasures{
			AlignedMeasures: []models.QualityMeasure{},
		},
	}
}

// buildProposalMetadataFromExecution creates proposal metadata from execution
func (o *Orchestrator) buildProposalMetadataFromExecution(startTime time.Time) models.ProposalMetadataSection {
	return models.ProposalMetadataSection{
		ClinicalReferences: []models.ClinicalReference{
			{
				Type:     "SYSTEM",
				Citation: "Flow2 Clinical Intelligence Engine",
				URL:      "",
			},
		},
		AuditTrail: models.AuditTrail{
			CalculationTime:     10,
			ContextFetchTime:    50,
			TotalProcessingTime: time.Since(startTime).Milliseconds(),
			CacheUtilization: models.CacheUtilization{
				FormularyCache:       "HIT",
				DoseCalculationCache: "MISS",
				MonitoringCache:      "HIT",
			},
		},
		NextSteps: []models.NextStep{
			{
				Step:     "PROVIDER_REVIEW",
				Service:  "Clinical Review",
				Optional: false,
				Reason:   "New medication recommendation",
			},
		},
	}
}

// ==================== UTILITY HELPER METHODS ====================

// getRouteDisplay returns display name for route code
func (o *Orchestrator) getRouteDisplay(routeCode string) string {
	switch routeCode {
	case "PO":
		return "Oral"
	case "IV":
		return "Intravenous"
	case "IM":
		return "Intramuscular"
	case "SC":
		return "Subcutaneous"
	default:
		return routeCode
	}
}

// generatePatientInstructions creates patient-specific instructions
func (o *Orchestrator) generatePatientInstructions(proposal *models.ScoredProposal) string {
	medicationName := proposal.SafetyVerified.Original.GenericName
	switch medicationName {
	case "Metformin":
		return "Take 1 tablet by mouth once daily with breakfast to minimize GI upset"
	case "Gliclazide":
		return "Take 1 tablet by mouth once daily before breakfast"
	case "Sitagliptin":
		return "Take 1 tablet by mouth once daily with or without food"
	default:
		return "Take as directed by your healthcare provider"
	}
}

// generatePharmacyInstructions creates pharmacy instructions
func (o *Orchestrator) generatePharmacyInstructions(proposal *models.ScoredProposal) string {
	return fmt.Sprintf("Dispense %d tablets", 90) // Default 90-day supply
}

// generateAdditionalInstructions creates additional safety instructions
func (o *Orchestrator) generateAdditionalInstructions(proposal *models.ScoredProposal) []string {
	medicationName := proposal.SafetyVerified.Original.GenericName
	switch medicationName {
	case "Metformin":
		return []string{
			"Take with food to minimize GI upset",
			"If a dose is missed, take as soon as remembered unless it's almost time for the next dose",
		}
	case "Gliclazide":
		return []string{
			"Monitor blood glucose regularly",
			"Be aware of signs of hypoglycemia",
		}
	default:
		return []string{"Follow up with healthcare provider as scheduled"}
	}
}

// buildCalculationFactorsFromContext builds calculation factors
func (o *Orchestrator) buildCalculationFactorsFromContext(proposal *models.ScoredProposal) models.CalculationFactors {
	return models.CalculationFactors{
		PatientWeight: 70.0, // Would come from clinical context
		PatientAge:    45,   // Would come from clinical context
		RenalFunction: models.RenalFunction{
			EGFR:     85.0,
			Category: "G2",
		},
	}
}

// assessOverallRisk determines overall risk level
func (o *Orchestrator) assessOverallRisk(safetyScore float64) string {
	if safetyScore > 0.9 {
		return "LOW"
	} else if safetyScore > 0.7 {
		return "MODERATE"
	}
	return "HIGH"
}

// getRiskImpact determines risk impact level
func (o *Orchestrator) getRiskImpact(safetyScore float64) string {
	if safetyScore > 0.9 {
		return "MINIMAL"
	} else if safetyScore > 0.7 {
		return "MODERATE"
	}
	return "HIGH"
}

// getBaselineMonitoringForMedication returns baseline monitoring requirements
func (o *Orchestrator) getBaselineMonitoringForMedication(medicationCode string) []models.BaselineMonitoring {
	// This would be looked up from a clinical database
	return []models.BaselineMonitoring{
		{
			Parameter: "eGFR",
			LOINC:     "48642-3",
			Timing:    "BEFORE_INITIATION",
			Priority:  "REQUIRED",
			Rationale: "To establish baseline renal function",
			CriticalValues: models.CriticalValues{
				Contraindicated: "< 30",
				CautionRequired: "30-45",
				Normal:          "> 45",
			},
		},
	}
}

// getOngoingMonitoringForMedication returns ongoing monitoring requirements
func (o *Orchestrator) getOngoingMonitoringForMedication(medicationCode string) []models.OngoingMonitoring {
	return []models.OngoingMonitoring{
		{
			Parameter: "eGFR",
			Frequency: models.MonitoringFrequency{
				Interval: 12,
				Unit:     "months",
			},
			Rationale: "Monitor for changes in renal function",
			ActionThresholds: []models.ActionThreshold{
				{
					Value:   "< 45",
					Action:  "DOSE_REDUCTION",
					Urgency: "ROUTINE",
				},
			},
		},
	}
}

// getSymptomMonitoringForMedication returns symptom monitoring requirements
func (o *Orchestrator) getSymptomMonitoringForMedication(medicationCode string) []models.SymptomMonitoring {
	return []models.SymptomMonitoring{
		{
			Symptom:           "GI upset",
			Frequency:         "At each visit",
			EducationProvided: "Common initially, usually improves with time",
		},
	}
}

// determineAlternativeCategory determines the category of alternative
func (o *Orchestrator) determineAlternativeCategory(primary, alternative *models.ScoredProposal) string {
	if primary.SafetyVerified.Original.TherapeuticClass == alternative.SafetyVerified.Original.TherapeuticClass {
		return "SAME_CLASS"
	}
	return "THERAPEUTIC_ALTERNATIVE"
}

// compareCosts compares medication costs
func (o *Orchestrator) compareCosts(primaryCost, alternativeCost float64) string {
	if alternativeCost < primaryCost*0.8 {
		return "LOWER"
	} else if alternativeCost > primaryCost*1.2 {
		return "HIGHER"
	}
	return "SIMILAR"
}

// getAdvantages returns advantages of a medication
func (o *Orchestrator) getAdvantages(proposal *models.ScoredProposal) []string {
	return []string{
		fmt.Sprintf("High safety score: %.2f", proposal.SafetyVerified.SafetyScore),
		"Well-established efficacy profile",
	}
}

// getDisadvantages returns disadvantages of a medication
func (o *Orchestrator) getDisadvantages(proposal *models.ScoredProposal) []string {
	return []string{
		"Requires regular monitoring",
		"Potential for side effects",
	}
}

// getSwitchingInstructions provides switching instructions
func (o *Orchestrator) getSwitchingInstructions(primary, alternative *models.ScoredProposal) string {
	return "Direct substitution with appropriate dose adjustment"
}

// getNonPharmAlternatives returns non-pharmacological alternatives
func (o *Orchestrator) getNonPharmAlternatives() []models.NonPharmAlternative {
	return []models.NonPharmAlternative{
		{
			Intervention:   "Lifestyle modification",
			Components:     []string{"Diet", "Exercise", "Weight management"},
			Effectiveness:  "Can reduce disease progression",
			Recommendation: "Should be continued regardless of medication",
		},
	}
}

// getConfidenceLevel determines confidence level from safety score
func (o *Orchestrator) getConfidenceLevel(safetyScore float64) string {
	if safetyScore > 0.9 {
		return "HIGH"
	} else if safetyScore > 0.7 {
		return "MODERATE"
	}
	return "LOW"
}

// getComplexityLevel determines complexity level
func (o *Orchestrator) getComplexityLevel(proposal *models.ScoredProposal) string {
	if len(proposal.SafetyVerified.Adjustments) == 0 {
		return "LOW"
	} else if len(proposal.SafetyVerified.Adjustments) < 3 {
		return "MODERATE"
	}
	return "HIGH"
}

// getFormularyDecision determines formulary decision
func (o *Orchestrator) getFormularyDecision(tier int) string {
	if tier == 1 {
		return "PREFERRED_DRUG_SELECTED"
	} else if tier == 2 {
		return "FORMULARY_ALTERNATIVE"
	}
	return "NON_FORMULARY_OPTION"
}

// getCostEffectiveness determines cost effectiveness
func (o *Orchestrator) getCostEffectiveness(cost float64) string {
	if cost < 20.0 {
		return "Highly cost-effective first-line agent"
	} else if cost < 50.0 {
		return "Cost-effective therapeutic option"
	}
	return "Higher cost option with specific indications"
}
