package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProposalGenerationRequest represents a request for medication proposal generation
type ProposalGenerationRequest struct {
	WorkflowID      uuid.UUID                  `json:"workflow_id"`
	PatientID       string                     `json:"patient_id"`
	SnapshotData    interface{}                `json:"snapshot_data"`
	ClinicalData    interface{}                `json:"clinical_data"`
	ProposalParams  *ProposalGenerationParams  `json:"proposal_params"`
	RequestedBy     string                     `json:"requested_by"`
	RequestedAt     time.Time                  `json:"requested_at"`
}

// ProposalGenerationResult represents the result of proposal generation
type ProposalGenerationResult struct {
	GenerationID        uuid.UUID                       `json:"generation_id"`
	WorkflowID          uuid.UUID                       `json:"workflow_id"`
	PatientID           string                          `json:"patient_id"`
	Proposals           []*MedicationProposal           `json:"proposals"`
	GenerationMetrics   *ProposalGenerationMetrics      `json:"generation_metrics"`
	QualityAssessment   *ProposalQualityAssessment      `json:"quality_assessment"`
	FHIRValidation      *FHIRValidationResult           `json:"fhir_validation,omitempty"`
	AlternativeAnalysis *AlternativeAnalysis            `json:"alternative_analysis,omitempty"`
	Warnings            []ProposalWarning               `json:"warnings,omitempty"`
	GeneratedAt         time.Time                       `json:"generated_at"`
}

// ProposalGenerationMetrics contains metrics for proposal generation
type ProposalGenerationMetrics struct {
	TotalGenerationTime     time.Duration          `json:"total_generation_time"`
	ProposalAnalysisTime    time.Duration          `json:"proposal_analysis_time"`
	SafetyCheckTime         time.Duration          `json:"safety_check_time"`
	FHIRValidationTime      time.Duration          `json:"fhir_validation_time"`
	AlternativeAnalysisTime time.Duration          `json:"alternative_analysis_time"`
	ProposalsGenerated      int                    `json:"proposals_generated"`
	ProposalsValidated      int                    `json:"proposals_validated"`
	SafetyChecksPerformed   int                    `json:"safety_checks_performed"`
	ComponentMetrics        map[string]interface{} `json:"component_metrics"`
}

// ProposalQualityAssessment represents quality assessment of generated proposals
type ProposalQualityAssessment struct {
	OverallQuality          float64                        `json:"overall_quality"`
	ClinicalAppropriatenesss float64                       `json:"clinical_appropriateness"`
	SafetyScore             float64                        `json:"safety_score"`
	EvidenceQuality         float64                        `json:"evidence_quality"`
	FHIRCompliance          float64                        `json:"fhir_compliance"`
	ProposalQualities       map[uuid.UUID]float64          `json:"proposal_qualities"`
	QualityFactors          []QualityFactor                `json:"quality_factors"`
	AssessmentTimestamp     time.Time                      `json:"assessment_timestamp"`
}

// QualityFactor represents a factor affecting proposal quality
type QualityFactor struct {
	Factor      string  `json:"factor"`
	Impact      string  `json:"impact"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

// FHIRValidationResult represents FHIR validation results
type FHIRValidationResult struct {
	ValidationID        uuid.UUID                      `json:"validation_id"`
	OverallValid        bool                           `json:"overall_valid"`
	ResourceValidations []FHIRResourceValidation       `json:"resource_validations"`
	ValidationErrors    []FHIRValidationError          `json:"validation_errors"`
	ValidationWarnings  []FHIRValidationWarning        `json:"validation_warnings"`
	FHIRVersion         string                         `json:"fhir_version"`
	ValidationProfile   string                         `json:"validation_profile"`
	ValidatedAt         time.Time                      `json:"validated_at"`
}

// FHIRResourceValidation represents validation of a single FHIR resource
type FHIRResourceValidation struct {
	ResourceType    string                     `json:"resource_type"`
	ResourceID      string                     `json:"resource_id"`
	Valid           bool                       `json:"valid"`
	Errors          []string                   `json:"errors,omitempty"`
	Warnings        []string                   `json:"warnings,omitempty"`
	ProfilesUsed    []string                   `json:"profiles_used"`
	ValidationTime  time.Duration              `json:"validation_time"`
}

// FHIRValidationError represents a FHIR validation error
type FHIRValidationError struct {
	ErrorID     uuid.UUID `json:"error_id"`
	Severity    string    `json:"severity"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Location    string    `json:"location"`
	ResourceType string   `json:"resource_type,omitempty"`
	ResourceID  string    `json:"resource_id,omitempty"`
}

// FHIRValidationWarning represents a FHIR validation warning
type FHIRValidationWarning struct {
	WarningID   uuid.UUID `json:"warning_id"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Location    string    `json:"location"`
	ResourceType string   `json:"resource_type,omitempty"`
	ResourceID  string    `json:"resource_id,omitempty"`
}

// AlternativeAnalysis represents analysis of alternative medications
type AlternativeAnalysis struct {
	AnalysisID          uuid.UUID                      `json:"analysis_id"`
	AlternativesFound   int                            `json:"alternatives_found"`
	AlternativeProposals []AlternativeMedicationProposal `json:"alternative_proposals"`
	ComparisonMatrix    []MedicationComparison         `json:"comparison_matrix"`
	RecommendationRanking []RankingEntry               `json:"recommendation_ranking"`
	AnalysisTimestamp   time.Time                      `json:"analysis_timestamp"`
}

// AlternativeMedicationProposal represents an alternative medication proposal
type AlternativeMedicationProposal struct {
	ProposalID          uuid.UUID              `json:"proposal_id"`
	MedicationName      string                 `json:"medication_name"`
	Rationale           string                 `json:"rationale"`
	Advantages          []string               `json:"advantages"`
	Disadvantages       []string               `json:"disadvantages"`
	ClinicalEvidence    []EvidenceSource       `json:"clinical_evidence"`
	SafetyProfile       *SafetyProfile         `json:"safety_profile"`
	CostEffectiveness   *CostEffectivenessAnalysis `json:"cost_effectiveness,omitempty"`
	QualityScore        float64                `json:"quality_score"`
}

// MedicationComparison represents a comparison between medications
type MedicationComparison struct {
	PrimaryMedication   string                         `json:"primary_medication"`
	AlternativeMedication string                       `json:"alternative_medication"`
	ComparisonFactors   []ComparisonFactor             `json:"comparison_factors"`
	OverallScore        float64                        `json:"overall_score"`
	Recommendation      string                         `json:"recommendation"`
}

// ComparisonFactor represents a factor in medication comparison
type ComparisonFactor struct {
	Factor      string  `json:"factor"`
	Primary     string  `json:"primary"`
	Alternative string  `json:"alternative"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Evidence    string  `json:"evidence,omitempty"`
}

// RankingEntry represents a ranking entry for recommendations
type RankingEntry struct {
	ProposalID  uuid.UUID `json:"proposal_id"`
	Medication  string    `json:"medication"`
	Rank        int       `json:"rank"`
	Score       float64   `json:"score"`
	Rationale   string    `json:"rationale"`
}

// SafetyProfile represents the safety profile of a medication
type SafetyProfile struct {
	OverallSafety       string                 `json:"overall_safety"`
	CommonSideEffects   []string               `json:"common_side_effects"`
	SeriousSideEffects  []string               `json:"serious_side_effects"`
	Contraindications   []string               `json:"contraindications"`
	DrugInteractions    []string               `json:"drug_interactions"`
	MonitoringRequirements []MonitoringRequirement `json:"monitoring_requirements"`
	SafetyScore         float64                `json:"safety_score"`
}

// CostEffectivenessAnalysis represents cost-effectiveness analysis
type CostEffectivenessAnalysis struct {
	EstimatedCost       float64                `json:"estimated_cost"`
	CostCategory        string                 `json:"cost_category"`
	EffectivenessScore  float64                `json:"effectiveness_score"`
	CostEffectivenessRatio float64             `json:"cost_effectiveness_ratio"`
	InsuranceCoverage   string                 `json:"insurance_coverage,omitempty"`
	GenericAvailable    bool                   `json:"generic_available"`
}

// ProposalWarning represents a warning during proposal generation
type ProposalWarning struct {
	WarningID   uuid.UUID `json:"warning_id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Context     string    `json:"context"`
	ProposalID  *uuid.UUID `json:"proposal_id,omitempty"`
	Actionable  bool      `json:"actionable"`
	Timestamp   time.Time `json:"timestamp"`
}

// ProposalGenerationConfig contains configuration for proposal generation
type ProposalGenerationConfig struct {
	MaxProposals            int           `mapstructure:"max_proposals" default:"5"`
	MinQualityThreshold     float64       `mapstructure:"min_quality_threshold" default:"0.7"`
	EnableAlternativeAnalysis bool        `mapstructure:"enable_alternative_analysis" default:"true"`
	EnableFHIRValidation    bool          `mapstructure:"enable_fhir_validation" default:"true"`
	FHIRValidationProfile   string        `mapstructure:"fhir_validation_profile" default:"us-core"`
	EnableCostAnalysis      bool          `mapstructure:"enable_cost_analysis" default:"false"`
	ProposalGenerationTimeout time.Duration `mapstructure:"proposal_generation_timeout" default:"15s"`
	SafetyCheckTimeout      time.Duration `mapstructure:"safety_check_timeout" default:"5s"`
	FHIRValidationTimeout   time.Duration `mapstructure:"fhir_validation_timeout" default:"5s"`
	AlternativeAnalysisTimeout time.Duration `mapstructure:"alternative_analysis_timeout" default:"10s"`
	EnableParallelGeneration bool         `mapstructure:"enable_parallel_generation" default:"true"`
	MaxConcurrentGenerators  int          `mapstructure:"max_concurrent_generators" default:"3"`
	RequireEvidenceBased     bool         `mapstructure:"require_evidence_based" default:"true"`
}

// ProposalGenerationService generates medication proposals based on clinical intelligence
type ProposalGenerationService struct {
	// External service clients
	rustEngineClient        RustEngineClient
	knowledgeBaseClients    map[string]KnowledgeBaseClient
	fhirValidationClient    FHIRValidationClient
	
	// Supporting services
	clinicalIntelligenceService *ClinicalIntelligenceService
	auditService               *AuditService
	metricsService            *MetricsService
	cacheService              *CacheService
	
	// Proposal generators
	generators                 map[string]ProposalGenerator
	
	// Configuration and logging
	config                    ProposalGenerationConfig
	logger                    *zap.Logger
	
	// Internal state
	generationStats           *GenerationStatistics
}

// ProposalGenerator interface for different types of proposal generators
type ProposalGenerator interface {
	GenerateProposals(ctx context.Context, request *ProposalGeneratorRequest) ([]*MedicationProposal, error)
	GetGeneratorInfo() ProposalGeneratorInfo
	IsHealthy() bool
}

// ProposalGeneratorRequest represents a request to a proposal generator
type ProposalGeneratorRequest struct {
	PatientData     interface{}                `json:"patient_data"`
	ClinicalData    interface{}                `json:"clinical_data"`
	GenerationType  string                     `json:"generation_type"`
	Parameters      map[string]interface{}     `json:"parameters"`
	Context         map[string]interface{}     `json:"context"`
}

// ProposalGeneratorInfo contains information about a proposal generator
type ProposalGeneratorInfo struct {
	GeneratorID     string            `json:"generator_id"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Capabilities    []string          `json:"capabilities"`
	ProposalTypes   []string          `json:"proposal_types"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// FHIRValidationClient interface for FHIR validation
type FHIRValidationClient interface {
	ValidateFHIRResources(ctx context.Context, resources []FHIRResource, profile string) (*FHIRValidationResult, error)
	IsHealthy() bool
}

// GenerationStatistics tracks proposal generation statistics
type GenerationStatistics struct {
	TotalRequests           int64         `json:"total_requests"`
	SuccessfulGenerations   int64         `json:"successful_generations"`
	FailedGenerations       int64         `json:"failed_generations"`
	TotalProposalsGenerated int64         `json:"total_proposals_generated"`
	AverageGenerationTime   time.Duration `json:"average_generation_time"`
	LastGenerationTime      time.Duration `json:"last_generation_time"`
	LastRequestAt           time.Time     `json:"last_request_at"`
	QualityDistribution     map[string]int64 `json:"quality_distribution"`
}

// NewProposalGenerationService creates a new proposal generation service
func NewProposalGenerationService(
	rustEngineClient RustEngineClient,
	knowledgeBaseClients map[string]KnowledgeBaseClient,
	fhirValidationClient FHIRValidationClient,
	clinicalIntelligenceService *ClinicalIntelligenceService,
	auditService *AuditService,
	metricsService *MetricsService,
	cacheService *CacheService,
	config ProposalGenerationConfig,
	logger *zap.Logger,
) *ProposalGenerationService {
	service := &ProposalGenerationService{
		rustEngineClient:            rustEngineClient,
		knowledgeBaseClients:        knowledgeBaseClients,
		fhirValidationClient:        fhirValidationClient,
		clinicalIntelligenceService: clinicalIntelligenceService,
		auditService:                auditService,
		metricsService:             metricsService,
		cacheService:               cacheService,
		config:                     config,
		logger:                     logger,
		generators:                 make(map[string]ProposalGenerator),
		generationStats:            &GenerationStatistics{
			QualityDistribution: make(map[string]int64),
		},
	}
	
	// Initialize proposal generators
	service.initializeGenerators()
	
	return service
}

// GenerateProposals generates medication proposals based on clinical data
func (p *ProposalGenerationService) GenerateProposals(
	ctx context.Context,
	request *ProposalGenerationRequest,
) (*ProposalGenerationResult, error) {
	generationStart := time.Now()
	generationID := uuid.New()
	
	p.logger.Info("Starting medication proposal generation",
		zap.String("generation_id", generationID.String()),
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("patient_id", request.PatientID),
	)
	
	// Audit generation start
	if p.auditService != nil {
		p.auditEvent(ctx, "proposal_generation_started", request.RequestedBy, map[string]interface{}{
			"generation_id": generationID.String(),
			"workflow_id":   request.WorkflowID.String(),
			"patient_id":    request.PatientID,
		})
	}
	
	result := &ProposalGenerationResult{
		GenerationID: generationID,
		WorkflowID:   request.WorkflowID,
		PatientID:    request.PatientID,
		Proposals:    []*MedicationProposal{},
		Warnings:     []ProposalWarning{},
		GeneratedAt:  time.Now(),
	}
	
	// Step 1: Analyze clinical data and extract proposal requirements
	analysisStart := time.Now()
	proposalRequirements, err := p.analyzeClinicalData(ctx, request.SnapshotData, request.ClinicalData)
	analysisTime := time.Since(analysisStart)
	
	if err != nil {
		p.logger.Error("Clinical data analysis failed",
			zap.String("generation_id", generationID.String()),
			zap.Error(err),
		)
		return result, fmt.Errorf("clinical data analysis failed: %w", err)
	}
	
	// Step 2: Generate medication proposals
	generationTime := time.Now()
	proposals, genErr := p.generateMedicationProposals(ctx, proposalRequirements, request.ProposalParams)
	proposalGenerationTime := time.Since(generationTime)
	
	if genErr != nil {
		p.logger.Warn("Proposal generation encountered errors", zap.Error(genErr))
		result.Warnings = append(result.Warnings, ProposalWarning{
			WarningID: uuid.New(),
			Type:      "generation_error",
			Severity:  "medium",
			Message:   fmt.Sprintf("Proposal generation failed: %v", genErr),
			Context:   "proposal_generation",
			Actionable: true,
			Timestamp: time.Now(),
		})
	} else {
		result.Proposals = proposals
	}
	
	// Step 3: Perform safety checks on proposals
	safetyStart := time.Now()
	safetyErr := p.performProposalSafetyChecks(ctx, result.Proposals, proposalRequirements)
	safetyCheckTime := time.Since(safetyStart)
	
	if safetyErr != nil {
		p.logger.Warn("Safety checks failed", zap.Error(safetyErr))
		result.Warnings = append(result.Warnings, ProposalWarning{
			WarningID: uuid.New(),
			Type:      "safety_check_error",
			Severity:  "high",
			Message:   fmt.Sprintf("Safety checks failed: %v", safetyErr),
			Context:   "safety_checks",
			Actionable: true,
			Timestamp: time.Now(),
		})
	}
	
	// Step 4: Validate FHIR compliance if enabled
	var fhirValidationTime time.Duration
	if p.config.EnableFHIRValidation && request.ProposalParams != nil && 
	   request.ProposalParams.FHIRCompliance != nil &&
	   request.ProposalParams.FHIRCompliance.ValidateResources {
		
		fhirStart := time.Now()
		fhirValidation, fhirErr := p.performFHIRValidation(ctx, result.Proposals, request.ProposalParams.FHIRCompliance)
		fhirValidationTime = time.Since(fhirStart)
		
		if fhirErr != nil {
			p.logger.Warn("FHIR validation failed", zap.Error(fhirErr))
			result.Warnings = append(result.Warnings, ProposalWarning{
				WarningID: uuid.New(),
				Type:      "fhir_validation_error",
				Severity:  "medium",
				Message:   fmt.Sprintf("FHIR validation failed: %v", fhirErr),
				Context:   "fhir_validation",
				Actionable: false,
				Timestamp: time.Now(),
			})
		} else {
			result.FHIRValidation = fhirValidation
		}
	}
	
	// Step 5: Perform alternative analysis if enabled
	var alternativeAnalysisTime time.Duration
	if p.config.EnableAlternativeAnalysis && request.ProposalParams != nil &&
	   request.ProposalParams.IncludeAlternatives {
		
		altStart := time.Now()
		alternativeAnalysis, altErr := p.performAlternativeAnalysis(ctx, result.Proposals, proposalRequirements)
		alternativeAnalysisTime = time.Since(altStart)
		
		if altErr != nil {
			p.logger.Warn("Alternative analysis failed", zap.Error(altErr))
			result.Warnings = append(result.Warnings, ProposalWarning{
				WarningID: uuid.New(),
				Type:      "alternative_analysis_error",
				Severity:  "low",
				Message:   fmt.Sprintf("Alternative analysis failed: %v", altErr),
				Context:   "alternative_analysis",
				Actionable: false,
				Timestamp: time.Now(),
			})
		} else {
			result.AlternativeAnalysis = alternativeAnalysis
		}
	}
	
	// Step 6: Assess proposal quality
	qualityAssessment := p.assessProposalQuality(result.Proposals, result.FHIRValidation)
	result.QualityAssessment = qualityAssessment
	
	// Step 7: Filter and rank proposals based on quality
	filteredProposals := p.filterAndRankProposals(result.Proposals, qualityAssessment, request.ProposalParams)
	result.Proposals = filteredProposals
	
	// Build generation metrics
	totalGenerationTime := time.Since(generationStart)
	result.GenerationMetrics = &ProposalGenerationMetrics{
		TotalGenerationTime:     totalGenerationTime,
		ProposalAnalysisTime:    analysisTime,
		SafetyCheckTime:         safetyCheckTime,
		FHIRValidationTime:      fhirValidationTime,
		AlternativeAnalysisTime: alternativeAnalysisTime,
		ProposalsGenerated:      len(result.Proposals),
		ProposalsValidated:      p.countValidatedProposals(result.Proposals),
		SafetyChecksPerformed:   len(result.Proposals), // Simplified count
		ComponentMetrics:        make(map[string]interface{}),
	}
	
	// Update generation statistics
	p.updateGenerationStats(totalGenerationTime, len(result.Proposals), qualityAssessment.OverallQuality, genErr == nil)
	
	// Audit generation completion
	if p.auditService != nil {
		p.auditEvent(ctx, "proposal_generation_completed", request.RequestedBy, map[string]interface{}{
			"generation_id":     generationID.String(),
			"proposals_count":   len(result.Proposals),
			"overall_quality":   qualityAssessment.OverallQuality,
			"generation_time":   totalGenerationTime.String(),
			"warnings_count":    len(result.Warnings),
		})
	}
	
	// Update metrics
	if p.metricsService != nil {
		p.metricsService.RecordProposalGeneration(
			totalGenerationTime,
			len(result.Proposals),
			qualityAssessment.OverallQuality,
			len(result.Warnings),
		)
	}
	
	p.logger.Info("Completed medication proposal generation",
		zap.String("generation_id", generationID.String()),
		zap.Duration("generation_time", totalGenerationTime),
		zap.Int("proposals_count", len(result.Proposals)),
		zap.Float64("overall_quality", qualityAssessment.OverallQuality),
		zap.Int("warnings_count", len(result.Warnings)),
	)
	
	return result, nil
}

// analyzeClinicalData analyzes clinical data to extract proposal requirements
func (p *ProposalGenerationService) analyzeClinicalData(
	ctx context.Context,
	snapshotData interface{},
	clinicalData interface{},
) (*ProposalRequirements, error) {
	// Extract clinical findings and intelligence results
	var clinicalFindings *ClinicalFindings
	var clinicalResult *ClinicalIntelligenceResult
	
	// Parse clinical intelligence result if available
	if clinicalData != nil {
		if result, ok := clinicalData.(*ClinicalIntelligenceResult); ok {
			clinicalResult = result
			clinicalFindings = result.ClinicalFindings
		}
	}
	
	// If no clinical findings, try to extract from snapshot
	if clinicalFindings == nil && snapshotData != nil {
		findings, err := p.extractClinicalFindings(snapshotData)
		if err != nil {
			return nil, fmt.Errorf("failed to extract clinical findings: %w", err)
		}
		clinicalFindings = findings
	}
	
	// Build proposal requirements
	requirements := &ProposalRequirements{
		PatientProfile:      p.buildPatientProfile(clinicalFindings),
		ClinicalIndications: p.extractClinicalIndications(clinicalFindings, clinicalResult),
		Contraindications:   p.extractContraindications(clinicalFindings, clinicalResult),
		SafetyRequirements:  p.buildSafetyRequirements(clinicalFindings, clinicalResult),
		QualityRequirements: p.buildQualityRequirements(clinicalResult),
		Context:             make(map[string]interface{}),
	}
	
	return requirements, nil
}

// generateMedicationProposals generates medication proposals based on requirements
func (p *ProposalGenerationService) generateMedicationProposals(
	ctx context.Context,
	requirements *ProposalRequirements,
	params *ProposalGenerationParams,
) ([]*MedicationProposal, error) {
	var allProposals []*MedicationProposal
	var generationErrors []error
	
	// Determine proposal types to generate
	proposalTypes := []string{"standard"}
	if params != nil && len(params.ProposalTypes) > 0 {
		proposalTypes = params.ProposalTypes
	}
	
	// Generate proposals using different generators
	for _, proposalType := range proposalTypes {
		if generator, exists := p.generators[proposalType]; exists && generator.IsHealthy() {
			generatorRequest := &ProposalGeneratorRequest{
				PatientData:    requirements.PatientProfile,
				ClinicalData:   requirements.ClinicalIndications,
				GenerationType: proposalType,
				Parameters:     p.buildGeneratorParameters(params),
				Context:        requirements.Context,
			}
			
			proposals, err := generator.GenerateProposals(ctx, generatorRequest)
			if err != nil {
				p.logger.Warn("Generator failed",
					zap.String("generator_type", proposalType),
					zap.Error(err),
				)
				generationErrors = append(generationErrors, err)
				continue
			}
			
			allProposals = append(allProposals, proposals...)
		}
	}
	
	// Use Rust engine as backup/additional generator
	if p.rustEngineClient != nil && p.rustEngineClient.IsHealthy() {
		rustProposals, err := p.generateWithRustEngine(ctx, requirements)
		if err != nil {
			p.logger.Warn("Rust engine proposal generation failed", zap.Error(err))
			generationErrors = append(generationErrors, err)
		} else {
			allProposals = append(allProposals, rustProposals...)
		}
	}
	
	// Limit proposals based on configuration
	if len(allProposals) > p.config.MaxProposals {
		// Sort by quality score and take top proposals
		sort.Slice(allProposals, func(i, j int) bool {
			return allProposals[i].QualityScore > allProposals[j].QualityScore
		})
		allProposals = allProposals[:p.config.MaxProposals]
	}
	
	// Return error if no proposals were generated
	if len(allProposals) == 0 {
		if len(generationErrors) > 0 {
			return nil, fmt.Errorf("no proposals generated, errors: %v", generationErrors)
		}
		return nil, fmt.Errorf("no proposals generated")
	}
	
	return allProposals, nil
}

// performProposalSafetyChecks performs safety checks on generated proposals
func (p *ProposalGenerationService) performProposalSafetyChecks(
	ctx context.Context,
	proposals []*MedicationProposal,
	requirements *ProposalRequirements,
) error {
	for _, proposal := range proposals {
		// Check against contraindications
		for _, contraindication := range requirements.Contraindications {
			if p.checkContraindication(proposal, contraindication) {
				proposal.SafetyAlerts = append(proposal.SafetyAlerts, SafetyAlert{
					AlertID:   uuid.New(),
					Severity:  contraindication.Severity,
					Message:   fmt.Sprintf("Contraindication: %s", contraindication.Description),
					Source:    "proposal_safety_check",
					Timestamp: time.Now(),
				})
			}
		}
		
		// Check drug interactions
		if len(requirements.PatientProfile.CurrentMedications) > 0 {
			interactions := p.checkDrugInteractions(proposal, requirements.PatientProfile.CurrentMedications)
			proposal.DrugInteractions = append(proposal.DrugInteractions, interactions...)
		}
		
		// Check allergies
		if len(requirements.PatientProfile.Allergies) > 0 {
			allergyAlerts := p.checkAllergies(proposal, requirements.PatientProfile.Allergies)
			proposal.AllergyAlerts = append(proposal.AllergyAlerts, allergyAlerts...)
		}
	}
	
	return nil
}

// performFHIRValidation validates FHIR resources in proposals
func (p *ProposalGenerationService) performFHIRValidation(
	ctx context.Context,
	proposals []*MedicationProposal,
	fhirParams *FHIRComplianceParams,
) (*FHIRValidationResult, error) {
	if p.fhirValidationClient == nil {
		return nil, fmt.Errorf("FHIR validation client not available")
	}
	
	var allResources []FHIRResource
	
	// Extract FHIR resources from proposals
	for _, proposal := range proposals {
		if len(proposal.FHIRResources) > 0 {
			allResources = append(allResources, proposal.FHIRResources...)
		}
	}
	
	if len(allResources) == 0 {
		return &FHIRValidationResult{
			ValidationID: uuid.New(),
			OverallValid: true,
			ResourceValidations: []FHIRResourceValidation{},
			ValidationErrors: []FHIRValidationError{},
			ValidationWarnings: []FHIRValidationWarning{},
			FHIRVersion: fhirParams.FHIRVersion,
			ValidationProfile: p.config.FHIRValidationProfile,
			ValidatedAt: time.Now(),
		}, nil
	}
	
	// Perform validation
	profile := p.config.FHIRValidationProfile
	if fhirParams.FHIRVersion != "" {
		profile = fhirParams.FHIRVersion
	}
	
	return p.fhirValidationClient.ValidateFHIRResources(ctx, allResources, profile)
}

// performAlternativeAnalysis performs alternative medication analysis
func (p *ProposalGenerationService) performAlternativeAnalysis(
	ctx context.Context,
	proposals []*MedicationProposal,
	requirements *ProposalRequirements,
) (*AlternativeAnalysis, error) {
	analysis := &AlternativeAnalysis{
		AnalysisID:            uuid.New(),
		AlternativeProposals:  []AlternativeMedicationProposal{},
		ComparisonMatrix:      []MedicationComparison{},
		RecommendationRanking: []RankingEntry{},
		AnalysisTimestamp:     time.Now(),
	}
	
	// For each primary proposal, find alternatives
	for _, proposal := range proposals {
		alternatives, err := p.findAlternatives(ctx, proposal, requirements)
		if err != nil {
			p.logger.Warn("Failed to find alternatives",
				zap.String("medication", proposal.MedicationName),
				zap.Error(err),
			)
			continue
		}
		
		analysis.AlternativeProposals = append(analysis.AlternativeProposals, alternatives...)
		
		// Create comparisons
		for _, alternative := range alternatives {
			comparison := p.compareMedications(proposal, &alternative, requirements)
			analysis.ComparisonMatrix = append(analysis.ComparisonMatrix, comparison)
		}
	}
	
	analysis.AlternativesFound = len(analysis.AlternativeProposals)
	
	// Create recommendation ranking
	analysis.RecommendationRanking = p.rankRecommendations(proposals, analysis.AlternativeProposals)
	
	return analysis, nil
}

// Helper methods and supporting functions

// ProposalRequirements represents requirements for proposal generation
type ProposalRequirements struct {
	PatientProfile      *PatientProfile           `json:"patient_profile"`
	ClinicalIndications []ClinicalIndication      `json:"clinical_indications"`
	Contraindications   []Contraindication        `json:"contraindications"`
	SafetyRequirements  *SafetyRequirements       `json:"safety_requirements"`
	QualityRequirements *QualityRequirements      `json:"quality_requirements"`
	Context             map[string]interface{}    `json:"context"`
}

// PatientProfile represents a patient profile for proposal generation
type PatientProfile struct {
	Demographics        map[string]interface{} `json:"demographics"`
	CurrentMedications  []string               `json:"current_medications"`
	Allergies           []string               `json:"allergies"`
	Conditions          []string               `json:"conditions"`
	VitalSigns          map[string]interface{} `json:"vital_signs"`
	LabResults          map[string]interface{} `json:"lab_results"`
	RiskFactors         []string               `json:"risk_factors"`
}

// ClinicalIndication represents a clinical indication for medication
type ClinicalIndication struct {
	Condition   string  `json:"condition"`
	Severity    string  `json:"severity"`
	Priority    int     `json:"priority"`
	Evidence    string  `json:"evidence"`
	Confidence  float64 `json:"confidence"`
}

// SafetyRequirements represents safety requirements for proposals
type SafetyRequirements struct {
	RequiredChecks      []string               `json:"required_checks"`
	SafetyThreshold     float64                `json:"safety_threshold"`
	MonitoringRequired  bool                   `json:"monitoring_required"`
	SpecialPopulation   string                 `json:"special_population,omitempty"`
	Context             map[string]interface{} `json:"context"`
}

// QualityRequirements represents quality requirements for proposals
type QualityRequirements struct {
	MinQualityScore     float64                `json:"min_quality_score"`
	EvidenceRequired    bool                   `json:"evidence_required"`
	GuidelineCompliance bool                   `json:"guideline_compliance"`
	Context             map[string]interface{} `json:"context"`
}

func (p *ProposalGenerationService) extractClinicalFindings(data interface{}) (*ClinicalFindings, error) {
	// Implementation to extract clinical findings from snapshot data
	// This would be similar to the implementation in ClinicalIntelligenceService
	return &ClinicalFindings{}, nil
}

func (p *ProposalGenerationService) buildPatientProfile(findings *ClinicalFindings) *PatientProfile {
	profile := &PatientProfile{
		Demographics:       make(map[string]interface{}),
		CurrentMedications: []string{},
		Allergies:          []string{},
		Conditions:         []string{},
		VitalSigns:         make(map[string]interface{}),
		LabResults:         make(map[string]interface{}),
		RiskFactors:        []string{},
	}
	
	if findings != nil {
		// Extract medications
		for _, med := range findings.ActiveMedications {
			profile.CurrentMedications = append(profile.CurrentMedications, med.Name)
		}
		
		// Extract allergies
		for _, allergy := range findings.Allergies {
			profile.Allergies = append(profile.Allergies, allergy.Allergen)
		}
		
		// Extract conditions from diagnoses
		for _, diagnosis := range findings.PrimaryDiagnoses {
			profile.Conditions = append(profile.Conditions, diagnosis.Display)
		}
		
		// Extract risk factors
		for _, risk := range findings.RiskFactors {
			profile.RiskFactors = append(profile.RiskFactors, risk.Factor)
		}
	}
	
	return profile
}

func (p *ProposalGenerationService) extractClinicalIndications(findings *ClinicalFindings, clinicalResult *ClinicalIntelligenceResult) []ClinicalIndication {
	var indications []ClinicalIndication
	
	if findings != nil {
		// Extract indications from primary diagnoses
		for _, diagnosis := range findings.PrimaryDiagnoses {
			indication := ClinicalIndication{
				Condition:  diagnosis.Display,
				Severity:   diagnosis.Severity,
				Priority:   1,
				Evidence:   fmt.Sprintf("Primary diagnosis: %s", diagnosis.Code),
				Confidence: diagnosis.Confidence,
			}
			indications = append(indications, indication)
		}
	}
	
	if clinicalResult != nil {
		// Extract indications from clinical recommendations
		for _, recommendation := range clinicalResult.ClinicalRecommendations {
			if recommendation.Type == "medication_recommendation" {
				indication := ClinicalIndication{
					Condition:  recommendation.Title,
					Severity:   "moderate", // Default
					Priority:   recommendation.Priority,
					Evidence:   recommendation.Rationale,
					Confidence: 0.8, // Default confidence
				}
				indications = append(indications, indication)
			}
		}
	}
	
	return indications
}

func (p *ProposalGenerationService) extractContraindications(findings *ClinicalFindings, clinicalResult *ClinicalIntelligenceResult) []Contraindication {
	var contraindications []Contraindication
	
	if clinicalResult != nil && clinicalResult.SafetyChecks != nil {
		for _, contraindicationAlert := range clinicalResult.SafetyChecks.ContraindicationAlerts {
			contraindication := Contraindication{
				ContraindicationID: contraindicationAlert.AlertID,
				Type:               contraindicationAlert.ContraindicationType,
				Description:        contraindicationAlert.Description,
				Severity:           contraindicationAlert.Severity,
				Evidence:           []string{contraindicationAlert.Condition},
				Override:           contraindicationAlert.Override,
			}
			contraindications = append(contraindications, contraindication)
		}
	}
	
	return contraindications
}

func (p *ProposalGenerationService) buildSafetyRequirements(findings *ClinicalFindings, clinicalResult *ClinicalIntelligenceResult) *SafetyRequirements {
	requirements := &SafetyRequirements{
		RequiredChecks:      []string{"drug_interactions", "allergies", "contraindications"},
		SafetyThreshold:     0.8,
		MonitoringRequired:  false,
		Context:             make(map[string]interface{}),
	}
	
	// Adjust based on patient profile
	if findings != nil {
		if len(findings.Allergies) > 0 {
			requirements.RequiredChecks = append(requirements.RequiredChecks, "allergy_check")
		}
		
		if len(findings.ActiveMedications) > 3 {
			requirements.RequiredChecks = append(requirements.RequiredChecks, "polypharmacy_check")
		}
	}
	
	return requirements
}

func (p *ProposalGenerationService) buildQualityRequirements(clinicalResult *ClinicalIntelligenceResult) *QualityRequirements {
	requirements := &QualityRequirements{
		MinQualityScore:     p.config.MinQualityThreshold,
		EvidenceRequired:    p.config.RequireEvidenceBased,
		GuidelineCompliance: true,
		Context:             make(map[string]interface{}),
	}
	
	return requirements
}

func (p *ProposalGenerationService) buildGeneratorParameters(params *ProposalGenerationParams) map[string]interface{} {
	parameters := make(map[string]interface{})
	
	if params != nil {
		parameters["max_proposals"] = params.MaxProposals
		parameters["quality_threshold"] = params.QualityThreshold
		parameters["include_alternatives"] = params.IncludeAlternatives
	}
	
	return parameters
}

func (p *ProposalGenerationService) generateWithRustEngine(ctx context.Context, requirements *ProposalRequirements) ([]*MedicationProposal, error) {
	// Implementation would call Rust engine to generate proposals
	// For now, return empty slice
	return []*MedicationProposal{}, nil
}

func (p *ProposalGenerationService) assessProposalQuality(proposals []*MedicationProposal, fhirValidation *FHIRValidationResult) *ProposalQualityAssessment {
	assessment := &ProposalQualityAssessment{
		ProposalQualities:   make(map[uuid.UUID]float64),
		QualityFactors:      []QualityFactor{},
		AssessmentTimestamp: time.Now(),
	}
	
	var totalQuality float64
	var validProposals int
	
	for _, proposal := range proposals {
		quality := proposal.QualityScore
		assessment.ProposalQualities[proposal.ProposalID] = quality
		
		if quality > 0 {
			totalQuality += quality
			validProposals++
		}
	}
	
	if validProposals > 0 {
		assessment.OverallQuality = totalQuality / float64(validProposals)
	}
	
	// Set component scores
	assessment.ClinicalAppropriatenesss = assessment.OverallQuality // Simplified
	assessment.SafetyScore = assessment.OverallQuality
	assessment.EvidenceQuality = assessment.OverallQuality
	
	// FHIR compliance score
	assessment.FHIRCompliance = 1.0 // Default
	if fhirValidation != nil {
		if fhirValidation.OverallValid {
			assessment.FHIRCompliance = 0.95
		} else {
			assessment.FHIRCompliance = 0.7
		}
	}
	
	return assessment
}

func (p *ProposalGenerationService) filterAndRankProposals(
	proposals []*MedicationProposal,
	quality *ProposalQualityAssessment,
	params *ProposalGenerationParams,
) []*MedicationProposal {
	// Filter by quality threshold
	var filteredProposals []*MedicationProposal
	threshold := p.config.MinQualityThreshold
	
	if params != nil && params.QualityThreshold > 0 {
		threshold = params.QualityThreshold
	}
	
	for _, proposal := range proposals {
		if proposal.QualityScore >= threshold {
			filteredProposals = append(filteredProposals, proposal)
		}
	}
	
	// Sort by quality score descending
	sort.Slice(filteredProposals, func(i, j int) bool {
		return filteredProposals[i].QualityScore > filteredProposals[j].QualityScore
	})
	
	// Limit number of proposals
	maxProposals := p.config.MaxProposals
	if params != nil && params.MaxProposals > 0 {
		maxProposals = params.MaxProposals
	}
	
	if len(filteredProposals) > maxProposals {
		filteredProposals = filteredProposals[:maxProposals]
	}
	
	return filteredProposals
}

func (p *ProposalGenerationService) countValidatedProposals(proposals []*MedicationProposal) int {
	count := 0
	for _, proposal := range proposals {
		for _, resource := range proposal.FHIRResources {
			if resource.Validated {
				count++
				break
			}
		}
	}
	return count
}

func (p *ProposalGenerationService) checkContraindication(proposal *MedicationProposal, contraindication Contraindication) bool {
	// Implementation would check if medication is contraindicated
	// This would involve checking medication name against contraindication type
	return false // Placeholder
}

func (p *ProposalGenerationService) checkDrugInteractions(proposal *MedicationProposal, currentMedications []string) []DrugInteraction {
	// Implementation would check for drug interactions
	return []DrugInteraction{} // Placeholder
}

func (p *ProposalGenerationService) checkAllergies(proposal *MedicationProposal, allergies []string) []AllergyAlert {
	// Implementation would check for allergy alerts
	return []AllergyAlert{} // Placeholder
}

func (p *ProposalGenerationService) findAlternatives(ctx context.Context, proposal *MedicationProposal, requirements *ProposalRequirements) ([]AlternativeMedicationProposal, error) {
	// Implementation would find alternative medications
	return []AlternativeMedicationProposal{}, nil
}

func (p *ProposalGenerationService) compareMedications(primary *MedicationProposal, alternative *AlternativeMedicationProposal, requirements *ProposalRequirements) MedicationComparison {
	// Implementation would compare medications
	return MedicationComparison{
		PrimaryMedication:     primary.MedicationName,
		AlternativeMedication: alternative.MedicationName,
		ComparisonFactors:     []ComparisonFactor{},
		OverallScore:          0.8,
		Recommendation:        "consider_alternative",
	}
}

func (p *ProposalGenerationService) rankRecommendations(proposals []*MedicationProposal, alternatives []AlternativeMedicationProposal) []RankingEntry {
	var ranking []RankingEntry
	
	// Rank primary proposals
	for i, proposal := range proposals {
		entry := RankingEntry{
			ProposalID: proposal.ProposalID,
			Medication: proposal.MedicationName,
			Rank:       i + 1,
			Score:      proposal.QualityScore,
			Rationale:  "Primary recommendation based on clinical analysis",
		}
		ranking = append(ranking, entry)
	}
	
	// Add alternatives with lower ranking
	for i, alternative := range alternatives {
		entry := RankingEntry{
			ProposalID: alternative.ProposalID,
			Medication: alternative.MedicationName,
			Rank:       len(proposals) + i + 1,
			Score:      alternative.QualityScore,
			Rationale:  alternative.Rationale,
		}
		ranking = append(ranking, entry)
	}
	
	return ranking
}

func (p *ProposalGenerationService) initializeGenerators() {
	// Initialize different types of proposal generators
	p.logger.Info("Initializing proposal generators")
	
	// This would initialize various generators like:
	// - StandardMedicationGenerator
	// - AlternativeMedicationGenerator
	// - EvidenceBasedGenerator
	// etc.
}

func (p *ProposalGenerationService) updateGenerationStats(
	generationTime time.Duration,
	proposalsCount int,
	qualityScore float64,
	success bool,
) {
	p.generationStats.TotalRequests++
	p.generationStats.LastGenerationTime = generationTime
	p.generationStats.LastRequestAt = time.Now()
	
	if success {
		p.generationStats.SuccessfulGenerations++
		p.generationStats.TotalProposalsGenerated += int64(proposalsCount)
	} else {
		p.generationStats.FailedGenerations++
	}
	
	// Update quality distribution
	qualityBucket := "low"
	if qualityScore >= 0.8 {
		qualityBucket = "high"
	} else if qualityScore >= 0.6 {
		qualityBucket = "medium"
	}
	p.generationStats.QualityDistribution[qualityBucket]++
	
	// Update average generation time
	if p.generationStats.TotalRequests > 0 {
		totalTime := p.generationStats.AverageGenerationTime * time.Duration(p.generationStats.TotalRequests-1)
		p.generationStats.AverageGenerationTime = (totalTime + generationTime) / time.Duration(p.generationStats.TotalRequests)
	}
}

func (p *ProposalGenerationService) auditEvent(ctx context.Context, eventType, actor string, data interface{}) {
	if p.auditService == nil {
		return
	}
	
	auditData, _ := json.Marshal(data)
	p.auditService.LogEvent(ctx, &AuditEvent{
		EventType: eventType,
		ActorID:   actor,
		Data:      string(auditData),
		Timestamp: time.Now(),
	})
}

// GetGenerationStatistics returns current generation statistics
func (p *ProposalGenerationService) GetGenerationStatistics() *GenerationStatistics {
	return p.generationStats
}

// IsHealthy returns the health status of the service
func (p *ProposalGenerationService) IsHealthy(ctx context.Context) bool {
	// Check Rust engine health
	if p.rustEngineClient != nil && !p.rustEngineClient.IsHealthy() {
		return false
	}
	
	// Check FHIR validation client health
	if p.fhirValidationClient != nil && !p.fhirValidationClient.IsHealthy() {
		return false
	}
	
	// Check knowledge base clients health
	for _, client := range p.knowledgeBaseClients {
		if !client.IsHealthy() {
			return false
		}
	}
	
	// Check proposal generators health
	for _, generator := range p.generators {
		if !generator.IsHealthy() {
			return false
		}
	}
	
	return true
}