package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"medication-service-v2/internal/infrastructure/clients"
)

// ClinicalIntelligenceRequest represents a request for clinical intelligence processing
type ClinicalIntelligenceRequest struct {
	WorkflowID      uuid.UUID                  `json:"workflow_id"`
	PatientID       string                     `json:"patient_id"`
	SnapshotData    interface{}                `json:"snapshot_data"`
	ClinicalParams  *ClinicalIntelligenceParams `json:"clinical_params"`
	RequestedBy     string                     `json:"requested_by"`
	RequestedAt     time.Time                  `json:"requested_at"`
}

// ClinicalIntelligenceResult represents the result of clinical intelligence processing
type ClinicalIntelligenceResult struct {
	ProcessingID        uuid.UUID                    `json:"processing_id"`
	WorkflowID          uuid.UUID                    `json:"workflow_id"`
	PatientID           string                       `json:"patient_id"`
	ClinicalFindings    *ClinicalFindings            `json:"clinical_findings"`
	RiskAssessment      *RiskAssessmentResult        `json:"risk_assessment"`
	RuleEngineResults   []RuleEngineResult           `json:"rule_engine_results"`
	ClinicalRecommendations []ClinicalRecommendation `json:"clinical_recommendations"`
	SafetyChecks        *SafetyCheckResults          `json:"safety_checks"`
	QualityScore        float64                      `json:"quality_score"`
	ProcessingMetrics   *ProcessingMetrics           `json:"processing_metrics"`
	Warnings            []ClinicalWarning            `json:"warnings,omitempty"`
	ProcessedAt         time.Time                    `json:"processed_at"`
}

// ClinicalFindings represents clinical findings extracted from patient data
type ClinicalFindings struct {
	PrimaryDiagnoses    []Diagnosis                `json:"primary_diagnoses"`
	SecondaryDiagnoses  []Diagnosis                `json:"secondary_diagnoses"`
	ActiveMedications   []MedicationSummary        `json:"active_medications"`
	Allergies           []AllergyInfo              `json:"allergies"`
	VitalSigns          *VitalSignsSummary         `json:"vital_signs"`
	LabResults          []LabResultSummary         `json:"lab_results"`
	RiskFactors         []RiskFactor               `json:"risk_factors"`
	ClinicalIndicators  map[string]interface{}     `json:"clinical_indicators"`
}

// Diagnosis represents a clinical diagnosis
type Diagnosis struct {
	Code        string    `json:"code"`
	System      string    `json:"system"`
	Display     string    `json:"display"`
	Severity    string    `json:"severity"`
	OnsetDate   *time.Time `json:"onset_date,omitempty"`
	Status      string    `json:"status"`
	Confidence  float64   `json:"confidence"`
}

// MedicationSummary represents a medication summary
type MedicationSummary struct {
	Name            string    `json:"name"`
	ActiveIngredient string   `json:"active_ingredient"`
	Dosage          string    `json:"dosage"`
	Frequency       string    `json:"frequency"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	Status          string    `json:"status"`
	Prescriber      string    `json:"prescriber,omitempty"`
}

// AllergyInfo represents allergy information
type AllergyInfo struct {
	Allergen    string    `json:"allergen"`
	AllergyType string    `json:"allergy_type"`
	Severity    string    `json:"severity"`
	Reaction    []string  `json:"reaction"`
	OnsetDate   *time.Time `json:"onset_date,omitempty"`
	Status      string    `json:"status"`
}

// VitalSignsSummary represents a summary of vital signs
type VitalSignsSummary struct {
	BloodPressure    *BloodPressureReading `json:"blood_pressure,omitempty"`
	HeartRate        *VitalReading         `json:"heart_rate,omitempty"`
	Temperature      *VitalReading         `json:"temperature,omitempty"`
	RespiratoryRate  *VitalReading         `json:"respiratory_rate,omitempty"`
	OxygenSaturation *VitalReading         `json:"oxygen_saturation,omitempty"`
	Weight           *VitalReading         `json:"weight,omitempty"`
	Height           *VitalReading         `json:"height,omitempty"`
	BMI              *VitalReading         `json:"bmi,omitempty"`
	LastUpdated      time.Time             `json:"last_updated"`
}

// BloodPressureReading represents a blood pressure reading
type BloodPressureReading struct {
	Systolic    float64   `json:"systolic"`
	Diastolic   float64   `json:"diastolic"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}

// VitalReading represents a single vital sign reading
type VitalReading struct {
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}

// LabResultSummary represents a lab result summary
type LabResultSummary struct {
	TestName        string    `json:"test_name"`
	TestCode        string    `json:"test_code"`
	Value           string    `json:"value"`
	Unit            string    `json:"unit,omitempty"`
	ReferenceRange  string    `json:"reference_range,omitempty"`
	Status          string    `json:"status"`
	AbnormalFlag    string    `json:"abnormal_flag,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	OrderedBy       string    `json:"ordered_by,omitempty"`
}

// RiskFactor represents a clinical risk factor
type RiskFactor struct {
	Factor      string    `json:"factor"`
	Category    string    `json:"category"`
	Severity    string    `json:"severity"`
	Value       string    `json:"value,omitempty"`
	Impact      string    `json:"impact"`
	IdentifiedAt time.Time `json:"identified_at"`
}

// RiskAssessmentResult represents the result of risk assessment
type RiskAssessmentResult struct {
	OverallRiskScore    float64                    `json:"overall_risk_score"`
	RiskCategory        string                     `json:"risk_category"`
	PrimaryRisks        []IdentifiedRisk           `json:"primary_risks"`
	SecondaryRisks      []IdentifiedRisk           `json:"secondary_risks"`
	RiskMitigations     []RiskMitigation           `json:"risk_mitigations"`
	AssessmentMetrics   map[string]float64         `json:"assessment_metrics"`
	AssessmentDate      time.Time                  `json:"assessment_date"`
}

// IdentifiedRisk represents an identified clinical risk
type IdentifiedRisk struct {
	RiskID          uuid.UUID `json:"risk_id"`
	RiskType        string    `json:"risk_type"`
	Description     string    `json:"description"`
	Probability     float64   `json:"probability"`
	Impact          string    `json:"impact"`
	Severity        string    `json:"severity"`
	Evidence        []string  `json:"evidence"`
	ClinicalBasis   string    `json:"clinical_basis"`
}

// RiskMitigation represents a risk mitigation strategy
type RiskMitigation struct {
	MitigationID    uuid.UUID `json:"mitigation_id"`
	RiskID          uuid.UUID `json:"risk_id"`
	Strategy        string    `json:"strategy"`
	Description     string    `json:"description"`
	Priority        string    `json:"priority"`
	EffectivenessScore float64 `json:"effectiveness_score"`
	Implementation  string    `json:"implementation"`
}

// RuleEngineResult represents the result from a clinical rule engine
type RuleEngineResult struct {
	EngineID        string                     `json:"engine_id"`
	EngineName      string                     `json:"engine_name"`
	RulesEvaluated  int                        `json:"rules_evaluated"`
	RulesFired      int                        `json:"rules_fired"`
	FiredRules      []FiredRule                `json:"fired_rules"`
	ProcessingTime  time.Duration              `json:"processing_time"`
	QualityScore    float64                    `json:"quality_score"`
	EngineMetrics   map[string]interface{}     `json:"engine_metrics"`
}

// FiredRule represents a rule that was triggered
type FiredRule struct {
	RuleID          string                     `json:"rule_id"`
	RuleName        string                     `json:"rule_name"`
	RuleType        string                     `json:"rule_type"`
	Priority        int                        `json:"priority"`
	Confidence      float64                    `json:"confidence"`
	Conditions      []RuleCondition            `json:"conditions"`
	Actions         []RuleAction               `json:"actions"`
	Evidence        []string                   `json:"evidence"`
	FiredAt         time.Time                  `json:"fired_at"`
}

// RuleCondition represents a condition that was evaluated
type RuleCondition struct {
	ConditionID     string      `json:"condition_id"`
	Field           string      `json:"field"`
	Operator        string      `json:"operator"`
	ExpectedValue   interface{} `json:"expected_value"`
	ActualValue     interface{} `json:"actual_value"`
	Satisfied       bool        `json:"satisfied"`
	Weight          float64     `json:"weight,omitempty"`
}

// RuleAction represents an action recommended by a rule
type RuleAction struct {
	ActionID        string                     `json:"action_id"`
	ActionType      string                     `json:"action_type"`
	Description     string                     `json:"description"`
	Parameters      map[string]interface{}     `json:"parameters"`
	Priority        int                        `json:"priority"`
	Automated       bool                       `json:"automated"`
}

// ClinicalRecommendation represents a clinical recommendation
type ClinicalRecommendation struct {
	RecommendationID uuid.UUID                     `json:"recommendation_id"`
	Type            string                         `json:"type"`
	Category        string                         `json:"category"`
	Title           string                         `json:"title"`
	Description     string                         `json:"description"`
	Rationale       string                         `json:"rationale"`
	Evidence        []EvidenceSource               `json:"evidence"`
	Strength        string                         `json:"strength"`
	Priority        int                            `json:"priority"`
	Actions         []RecommendedAction            `json:"actions"`
	Contraindications []Contraindication           `json:"contraindications,omitempty"`
	Monitoring      []MonitoringRequirement       `json:"monitoring,omitempty"`
	CreatedAt       time.Time                      `json:"created_at"`
}

// EvidenceSource represents a source of clinical evidence
type EvidenceSource struct {
	SourceID        string    `json:"source_id"`
	SourceType      string    `json:"source_type"`
	Citation        string    `json:"citation"`
	EvidenceLevel   string    `json:"evidence_level"`
	Quality         string    `json:"quality"`
	RelevanceScore  float64   `json:"relevance_score"`
}

// RecommendedAction represents a recommended clinical action
type RecommendedAction struct {
	ActionID        uuid.UUID                  `json:"action_id"`
	ActionType      string                     `json:"action_type"`
	Description     string                     `json:"description"`
	Instructions    string                     `json:"instructions"`
	Timing          string                     `json:"timing"`
	Duration        string                     `json:"duration,omitempty"`
	Parameters      map[string]interface{}     `json:"parameters"`
	Urgency         string                     `json:"urgency"`
}

// Contraindication represents a contraindication
type Contraindication struct {
	ContraindicationID uuid.UUID `json:"contraindication_id"`
	Type               string    `json:"type"`
	Description        string    `json:"description"`
	Severity           string    `json:"severity"`
	Evidence           []string  `json:"evidence"`
	Override           bool      `json:"override_possible"`
}

// MonitoringRequirement represents a monitoring requirement
type MonitoringRequirement struct {
	RequirementID   uuid.UUID `json:"requirement_id"`
	Parameter       string    `json:"parameter"`
	Frequency       string    `json:"frequency"`
	Duration        string    `json:"duration"`
	AlertThresholds map[string]interface{} `json:"alert_thresholds"`
	Rationale       string    `json:"rationale"`
}

// SafetyCheckResults represents safety check results
type SafetyCheckResults struct {
	OverallSafetyScore  float64               `json:"overall_safety_score"`
	SafetyAlerts        []SafetyAlert         `json:"safety_alerts"`
	DrugInteractions    []DrugInteraction     `json:"drug_interactions"`
	AllergyAlerts       []AllergyAlert        `json:"allergy_alerts"`
	ContraindicationAlerts []ContraindicationAlert `json:"contraindication_alerts"`
	DosageWarnings      []DosageWarning       `json:"dosage_warnings"`
	MonitoringAlerts    []MonitoringAlert     `json:"monitoring_alerts"`
	ChecksPerformed     []string              `json:"checks_performed"`
	CheckTimestamp      time.Time             `json:"check_timestamp"`
}

// ContraindicationAlert represents a contraindication alert
type ContraindicationAlert struct {
	AlertID             uuid.UUID `json:"alert_id"`
	ContraindicationType string    `json:"contraindication_type"`
	Medication          string    `json:"medication"`
	Condition           string    `json:"condition"`
	Severity            string    `json:"severity"`
	Description         string    `json:"description"`
	Recommendation      string    `json:"recommendation"`
	Override            bool      `json:"override_possible"`
}

// DosageWarning represents a dosage warning
type DosageWarning struct {
	WarningID       uuid.UUID `json:"warning_id"`
	Medication      string    `json:"medication"`
	WarningType     string    `json:"warning_type"`
	CurrentDosage   string    `json:"current_dosage"`
	RecommendedDosage string  `json:"recommended_dosage"`
	Reason          string    `json:"reason"`
	Severity        string    `json:"severity"`
}

// MonitoringAlert represents a monitoring alert
type MonitoringAlert struct {
	AlertID         uuid.UUID `json:"alert_id"`
	MonitoringType  string    `json:"monitoring_type"`
	Parameter       string    `json:"parameter"`
	Frequency       string    `json:"frequency"`
	Reason          string    `json:"reason"`
	Priority        string    `json:"priority"`
	LastChecked     *time.Time `json:"last_checked,omitempty"`
}

// ClinicalWarning represents a clinical warning
type ClinicalWarning struct {
	WarningID   uuid.UUID `json:"warning_id"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	Context     string    `json:"context"`
	Actionable  bool      `json:"actionable"`
	Timestamp   time.Time `json:"timestamp"`
}

// ProcessingMetrics represents processing metrics for clinical intelligence
type ProcessingMetrics struct {
	TotalProcessingTime     time.Duration          `json:"total_processing_time"`
	DataExtractionTime      time.Duration          `json:"data_extraction_time"`
	RuleEvaluationTime      time.Duration          `json:"rule_evaluation_time"`
	RiskAssessmentTime      time.Duration          `json:"risk_assessment_time"`
	SafetyCheckTime         time.Duration          `json:"safety_check_time"`
	RecommendationTime      time.Duration          `json:"recommendation_time"`
	DataPointsProcessed     int                    `json:"data_points_processed"`
	RulesEvaluated          int                    `json:"rules_evaluated"`
	RecommendationsGenerated int                   `json:"recommendations_generated"`
	ComponentMetrics        map[string]interface{} `json:"component_metrics"`
}

// ClinicalIntelligenceConfig contains configuration for clinical intelligence
type ClinicalIntelligenceConfig struct {
	EnableRuleEngines        []string      `mapstructure:"enable_rule_engines"`
	RustEngineURL           string        `mapstructure:"rust_engine_url" default:"http://localhost:8095"`
	KnowledgeBaseURLs       map[string]string `mapstructure:"knowledge_base_urls"`
	DefaultQualityThreshold  float64       `mapstructure:"default_quality_threshold" default:"0.8"`
	EnableRiskAssessment     bool          `mapstructure:"enable_risk_assessment" default:"true"`
	EnableSafetyChecks       bool          `mapstructure:"enable_safety_checks" default:"true"`
	MaxRuleEngines          int           `mapstructure:"max_rule_engines" default:"5"`
	RuleEvaluationTimeout   time.Duration `mapstructure:"rule_evaluation_timeout" default:"10s"`
	RiskAssessmentTimeout   time.Duration `mapstructure:"risk_assessment_timeout" default:"5s"`
	SafetyCheckTimeout      time.Duration `mapstructure:"safety_check_timeout" default:"5s"`
	EnableParallelProcessing bool          `mapstructure:"enable_parallel_processing" default:"true"`
	MaxConcurrentProcessors  int           `mapstructure:"max_concurrent_processors" default:"3"`
}

// ClinicalIntelligenceService provides clinical intelligence and rule evaluation capabilities
type ClinicalIntelligenceService struct {
	// High-performance clinical calculation service
	clinicalCalculationService *ClinicalCalculationService
	
	// External service clients
	rustEngineClient        *clients.RustClinicalEngineClient
	knowledgeBaseClients    map[string]KnowledgeBaseClient
	
	// Supporting services
	auditService           *AuditService
	metricsService        *MetricsService
	cacheService          *CacheService
	
	// Configuration and logging
	config                ClinicalIntelligenceConfig
	logger                *zap.Logger
	
	// Internal state and synchronization
	ruleEngines           map[string]RuleEngine
	processingStats       *ProcessingStatistics
	processingMutex       sync.RWMutex
}

// RuleEngine interface for rule evaluation engines
type RuleEngine interface {
	EvaluateRules(ctx context.Context, clinicalData interface{}) (*RuleEngineResult, error)
	GetEngineInfo() RuleEngineInfo
	IsHealthy() bool
}

// RuleEngineInfo contains information about a rule engine
type RuleEngineInfo struct {
	EngineID    string            `json:"engine_id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Capabilities []string         `json:"capabilities"`
	RuleCount   int               `json:"rule_count"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// KnowledgeBaseClient interface for knowledge base interactions
type KnowledgeBaseClient interface {
	QueryKnowledge(ctx context.Context, query KnowledgeQuery) (*KnowledgeResponse, error)
	GetEvidenceSources(ctx context.Context, topic string) ([]EvidenceSource, error)
	IsHealthy() bool
}

// RustEngineClient interface for Rust clinical engine
type RustEngineClient interface {
	EvaluateClinicalRules(ctx context.Context, request *ClinicalRuleRequest) (*ClinicalRuleResponse, error)
	AssessRisk(ctx context.Context, request *RiskAssessmentRequest) (*RiskAssessmentResponse, error)
	PerformSafetyChecks(ctx context.Context, request *SafetyCheckRequest) (*SafetyCheckResponse, error)
	IsHealthy() bool
}

// Knowledge base types
type KnowledgeQuery struct {
	QueryType   string                 `json:"query_type"`
	Topic       string                 `json:"topic"`
	Context     map[string]interface{} `json:"context"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type KnowledgeResponse struct {
	Results     []KnowledgeResult      `json:"results"`
	Confidence  float64                `json:"confidence"`
	Sources     []EvidenceSource       `json:"sources"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type KnowledgeResult struct {
	ResultID    string                 `json:"result_id"`
	Content     string                 `json:"content"`
	Confidence  float64                `json:"confidence"`
	Sources     []string               `json:"sources"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Rust engine request/response types
type ClinicalRuleRequest struct {
	PatientData     interface{}            `json:"patient_data"`
	Rules           []string               `json:"rules"`
	Context         map[string]interface{} `json:"context"`
	EvaluationMode  string                 `json:"evaluation_mode"`
}

type ClinicalRuleResponse struct {
	Results         []RuleEngineResult     `json:"results"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Metrics         map[string]interface{} `json:"metrics"`
}

type RiskAssessmentRequest struct {
	PatientData     interface{}            `json:"patient_data"`
	RiskFactors     []string               `json:"risk_factors"`
	AssessmentType  string                 `json:"assessment_type"`
	Context         map[string]interface{} `json:"context"`
}

type RiskAssessmentResponse struct {
	RiskAssessment  *RiskAssessmentResult  `json:"risk_assessment"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Metrics         map[string]interface{} `json:"metrics"`
}

type SafetyCheckRequest struct {
	PatientData     interface{}            `json:"patient_data"`
	ProposedMedications []string           `json:"proposed_medications"`
	CheckTypes      []string               `json:"check_types"`
	Context         map[string]interface{} `json:"context"`
}

type SafetyCheckResponse struct {
	SafetyResults   *SafetyCheckResults    `json:"safety_results"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Metrics         map[string]interface{} `json:"metrics"`
}

// ProcessingStatistics tracks processing statistics
type ProcessingStatistics struct {
	TotalRequests       int64     `json:"total_requests"`
	SuccessfulRequests  int64     `json:"successful_requests"`
	FailedRequests      int64     `json:"failed_requests"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	LastProcessingTime  time.Duration `json:"last_processing_time"`
	LastRequestAt       time.Time `json:"last_request_at"`
}

// NewClinicalIntelligenceService creates a new clinical intelligence service
func NewClinicalIntelligenceService(
	clinicalCalculationService *ClinicalCalculationService,
	rustEngineClient *clients.RustClinicalEngineClient,
	knowledgeBaseClients map[string]KnowledgeBaseClient,
	auditService *AuditService,
	metricsService *MetricsService,
	cacheService *CacheService,
	config ClinicalIntelligenceConfig,
	logger *zap.Logger,
) *ClinicalIntelligenceService {
	service := &ClinicalIntelligenceService{
		clinicalCalculationService: clinicalCalculationService,
		rustEngineClient:     rustEngineClient,
		knowledgeBaseClients: knowledgeBaseClients,
		auditService:         auditService,
		metricsService:      metricsService,
		cacheService:        cacheService,
		config:              config,
		logger:              logger,
		ruleEngines:         make(map[string]RuleEngine),
		processingStats:     &ProcessingStatistics{},
	}
	
	// Initialize rule engines
	service.initializeRuleEngines()
	
	return service
}

// ProcessClinicalIntelligence processes clinical intelligence for a patient
func (c *ClinicalIntelligenceService) ProcessClinicalIntelligence(
	ctx context.Context,
	request *ClinicalIntelligenceRequest,
) (*ClinicalIntelligenceResult, error) {
	processingStart := time.Now()
	processingID := uuid.New()
	
	c.logger.Info("Starting clinical intelligence processing",
		zap.String("processing_id", processingID.String()),
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("patient_id", request.PatientID),
	)
	
	// Audit processing start
	if c.auditService != nil {
		c.auditEvent(ctx, "clinical_intelligence_started", request.RequestedBy, map[string]interface{}{
			"processing_id": processingID.String(),
			"workflow_id":   request.WorkflowID.String(),
			"patient_id":    request.PatientID,
		})
	}
	
	result := &ClinicalIntelligenceResult{
		ProcessingID:        processingID,
		WorkflowID:          request.WorkflowID,
		PatientID:           request.PatientID,
		RuleEngineResults:   []RuleEngineResult{},
		ClinicalRecommendations: []ClinicalRecommendation{},
		Warnings:            []ClinicalWarning{},
		ProcessedAt:         time.Now(),
	}
	
	// Step 1: Extract clinical findings from snapshot data
	extractionStart := time.Now()
	clinicalFindings, err := c.extractClinicalFindings(ctx, request.SnapshotData)
	extractionTime := time.Since(extractionStart)
	
	if err != nil {
		c.logger.Error("Failed to extract clinical findings",
			zap.String("processing_id", processingID.String()),
			zap.Error(err),
		)
		return result, fmt.Errorf("clinical findings extraction failed: %w", err)
	}
	
	result.ClinicalFindings = clinicalFindings
	
	// Step 2: Perform risk assessment if enabled
	var riskAssessmentTime time.Duration
	if c.config.EnableRiskAssessment && request.ClinicalParams != nil && request.ClinicalParams.RiskAssessment != nil {
		riskStart := time.Now()
		riskAssessment, riskErr := c.performRiskAssessment(ctx, clinicalFindings, request.ClinicalParams.RiskAssessment)
		riskAssessmentTime = time.Since(riskStart)
		
		if riskErr != nil {
			c.logger.Warn("Risk assessment failed", zap.Error(riskErr))
			result.Warnings = append(result.Warnings, ClinicalWarning{
				WarningID: uuid.New(),
				Type:      "risk_assessment_error",
				Message:   fmt.Sprintf("Risk assessment failed: %v", riskErr),
				Severity:  "medium",
				Context:   "risk_assessment",
				Actionable: false,
				Timestamp: time.Now(),
			})
		} else {
			result.RiskAssessment = riskAssessment
		}
	}
	
	// Step 3: Evaluate clinical rules
	ruleEvaluationStart := time.Now()
	ruleResults, ruleErr := c.evaluateClinicalRules(ctx, clinicalFindings, request.ClinicalParams)
	ruleEvaluationTime := time.Since(ruleEvaluationStart)
	
	if ruleErr != nil {
		c.logger.Warn("Rule evaluation failed", zap.Error(ruleErr))
		result.Warnings = append(result.Warnings, ClinicalWarning{
			WarningID: uuid.New(),
			Type:      "rule_evaluation_error",
			Message:   fmt.Sprintf("Rule evaluation failed: %v", ruleErr),
			Severity:  "medium",
			Context:   "rule_evaluation",
			Actionable: false,
			Timestamp: time.Now(),
		})
	} else {
		result.RuleEngineResults = ruleResults
	}
	
	// Step 4: Perform safety checks if enabled
	var safetyCheckTime time.Duration
	if c.config.EnableSafetyChecks {
		safetyStart := time.Now()
		safetyResults, safetyErr := c.performSafetyChecks(ctx, clinicalFindings)
		safetyCheckTime = time.Since(safetyStart)
		
		if safetyErr != nil {
			c.logger.Warn("Safety checks failed", zap.Error(safetyErr))
			result.Warnings = append(result.Warnings, ClinicalWarning{
				WarningID: uuid.New(),
				Type:      "safety_check_error",
				Message:   fmt.Sprintf("Safety checks failed: %v", safetyErr),
				Severity:  "high",
				Context:   "safety_checks",
				Actionable: true,
				Timestamp: time.Now(),
			})
		} else {
			result.SafetyChecks = safetyResults
		}
	}
	
	// Step 5: Generate clinical recommendations
	recommendationStart := time.Now()
	recommendations, recErr := c.generateClinicalRecommendations(ctx, clinicalFindings, result.RiskAssessment, ruleResults)
	recommendationTime := time.Since(recommendationStart)
	
	if recErr != nil {
		c.logger.Warn("Clinical recommendations generation failed", zap.Error(recErr))
		result.Warnings = append(result.Warnings, ClinicalWarning{
			WarningID: uuid.New(),
			Type:      "recommendation_error",
			Message:   fmt.Sprintf("Recommendation generation failed: %v", recErr),
			Severity:  "medium",
			Context:   "recommendations",
			Actionable: false,
			Timestamp: time.Now(),
		})
	} else {
		result.ClinicalRecommendations = recommendations
	}
	
	// Calculate quality score
	result.QualityScore = c.calculateQualityScore(result)
	
	// Build processing metrics
	totalProcessingTime := time.Since(processingStart)
	result.ProcessingMetrics = &ProcessingMetrics{
		TotalProcessingTime:     totalProcessingTime,
		DataExtractionTime:      extractionTime,
		RuleEvaluationTime:      ruleEvaluationTime,
		RiskAssessmentTime:      riskAssessmentTime,
		SafetyCheckTime:         safetyCheckTime,
		RecommendationTime:      recommendationTime,
		DataPointsProcessed:     c.countDataPoints(clinicalFindings),
		RulesEvaluated:          c.countRulesEvaluated(ruleResults),
		RecommendationsGenerated: len(recommendations),
		ComponentMetrics:        make(map[string]interface{}),
	}
	
	// Update processing statistics
	c.updateProcessingStats(totalProcessingTime, err == nil)
	
	// Audit processing completion
	if c.auditService != nil {
		c.auditEvent(ctx, "clinical_intelligence_completed", request.RequestedBy, map[string]interface{}{
			"processing_id":    processingID.String(),
			"quality_score":    result.QualityScore,
			"processing_time":  totalProcessingTime.String(),
			"warnings_count":   len(result.Warnings),
		})
	}
	
	// Update metrics
	if c.metricsService != nil {
		c.metricsService.RecordClinicalIntelligenceProcessing(
			totalProcessingTime,
			result.QualityScore,
			len(result.Warnings),
		)
	}
	
	c.logger.Info("Completed clinical intelligence processing",
		zap.String("processing_id", processingID.String()),
		zap.Duration("processing_time", totalProcessingTime),
		zap.Float64("quality_score", result.QualityScore),
		zap.Int("warnings_count", len(result.Warnings)),
	)
	
	return result, nil
}

// extractClinicalFindings extracts clinical findings from snapshot data
func (c *ClinicalIntelligenceService) extractClinicalFindings(ctx context.Context, snapshotData interface{}) (*ClinicalFindings, error) {
	// This would parse the snapshot data and extract relevant clinical information
	// For now, we'll create a basic structure - this would be implemented based on the actual snapshot format
	
	findings := &ClinicalFindings{
		PrimaryDiagnoses:   []Diagnosis{},
		SecondaryDiagnoses: []Diagnosis{},
		ActiveMedications:  []MedicationSummary{},
		Allergies:          []AllergyInfo{},
		LabResults:         []LabResultSummary{},
		RiskFactors:        []RiskFactor{},
		ClinicalIndicators: make(map[string]interface{}),
	}
	
	// Parse snapshot data - this would be implemented based on actual data structure
	if snapshotMap, ok := snapshotData.(map[string]interface{}); ok {
		// Extract diagnoses
		if diagnosesData, exists := snapshotMap["diagnoses"]; exists {
			findings.PrimaryDiagnoses = c.parseDiagnoses(diagnosesData, "primary")
			findings.SecondaryDiagnoses = c.parseDiagnoses(diagnosesData, "secondary")
		}
		
		// Extract medications
		if medicationsData, exists := snapshotMap["medications"]; exists {
			findings.ActiveMedications = c.parseMedications(medicationsData)
		}
		
		// Extract allergies
		if allergiesData, exists := snapshotMap["allergies"]; exists {
			findings.Allergies = c.parseAllergies(allergiesData)
		}
		
		// Extract vital signs
		if vitalSignsData, exists := snapshotMap["vital_signs"]; exists {
			findings.VitalSigns = c.parseVitalSigns(vitalSignsData)
		}
		
		// Extract lab results
		if labResultsData, exists := snapshotMap["lab_results"]; exists {
			findings.LabResults = c.parseLabResults(labResultsData)
		}
		
		// Extract risk factors
		findings.RiskFactors = c.identifyRiskFactors(findings)
	}
	
	return findings, nil
}

// performRiskAssessment performs clinical risk assessment
func (c *ClinicalIntelligenceService) performRiskAssessment(
	ctx context.Context,
	findings *ClinicalFindings,
	params *RiskAssessmentParams,
) (*RiskAssessmentResult, error) {
	if !params.EnableRiskScoring {
		return nil, nil
	}
	
	// Create risk assessment request
	riskRequest := &RiskAssessmentRequest{
		PatientData:    findings,
		RiskFactors:    params.RiskFactors,
		AssessmentType: "comprehensive",
		Context: map[string]interface{}{
			"min_risk_threshold": params.MinRiskThreshold,
		},
	}
	
	// Call Rust engine for risk assessment
	response, err := c.rustEngineClient.AssessRisk(ctx, riskRequest)
	if err != nil {
		return nil, fmt.Errorf("rust engine risk assessment failed: %w", err)
	}
	
	return response.RiskAssessment, nil
}

// evaluateClinicalRules evaluates clinical rules using available engines
func (c *ClinicalIntelligenceService) evaluateClinicalRules(
	ctx context.Context,
	findings *ClinicalFindings,
	params *ClinicalIntelligenceParams,
) ([]RuleEngineResult, error) {
	var results []RuleEngineResult
	var engines []string
	
	// Determine which engines to use
	if params != nil && len(params.RuleEngines) > 0 {
		engines = params.RuleEngines
	} else {
		engines = c.config.EnableRuleEngines
	}
	
	// Evaluate rules using each engine
	for _, engineName := range engines {
		if engine, exists := c.ruleEngines[engineName]; exists && engine.IsHealthy() {
			result, err := engine.EvaluateRules(ctx, findings)
			if err != nil {
				c.logger.Warn("Rule engine evaluation failed",
					zap.String("engine", engineName),
					zap.Error(err),
				)
				continue
			}
			results = append(results, *result)
		}
	}
	
	// Also use Rust engine if available
	if c.rustEngineClient != nil && c.rustEngineClient.IsHealthy() {
		ruleRequest := &ClinicalRuleRequest{
			PatientData:    findings,
			Rules:          []string{}, // Would specify specific rules
			Context:        make(map[string]interface{}),
			EvaluationMode: "comprehensive",
		}
		
		response, err := c.rustEngineClient.EvaluateClinicalRules(ctx, ruleRequest)
		if err != nil {
			c.logger.Warn("Rust engine rule evaluation failed", zap.Error(err))
		} else {
			results = append(results, response.Results...)
		}
	}
	
	return results, nil
}

// performSafetyChecks performs clinical safety checks
func (c *ClinicalIntelligenceService) performSafetyChecks(
	ctx context.Context,
	findings *ClinicalFindings,
) (*SafetyCheckResults, error) {
	// Extract medication names for safety checking
	var medications []string
	for _, med := range findings.ActiveMedications {
		medications = append(medications, med.Name)
	}
	
	safetyRequest := &SafetyCheckRequest{
		PatientData:         findings,
		ProposedMedications: medications,
		CheckTypes:          []string{"drug_interactions", "allergies", "contraindications", "dosage"},
		Context:             make(map[string]interface{}),
	}
	
	response, err := c.rustEngineClient.PerformSafetyChecks(ctx, safetyRequest)
	if err != nil {
		return nil, fmt.Errorf("rust engine safety checks failed: %w", err)
	}
	
	return response.SafetyResults, nil
}

// generateClinicalRecommendations generates clinical recommendations
func (c *ClinicalIntelligenceService) generateClinicalRecommendations(
	ctx context.Context,
	findings *ClinicalFindings,
	riskAssessment *RiskAssessmentResult,
	ruleResults []RuleEngineResult,
) ([]ClinicalRecommendation, error) {
	var recommendations []ClinicalRecommendation
	
	// Generate recommendations based on fired rules
	for _, engineResult := range ruleResults {
		for _, firedRule := range engineResult.FiredRules {
			for _, action := range firedRule.Actions {
				if action.ActionType == "recommendation" {
					recommendation := ClinicalRecommendation{
						RecommendationID: uuid.New(),
						Type:            action.ActionType,
						Category:        "rule_based",
						Title:           firedRule.RuleName,
						Description:     action.Description,
						Rationale:       fmt.Sprintf("Based on rule: %s", firedRule.RuleName),
						Evidence:        []EvidenceSource{},
						Strength:        c.determineRecommendationStrength(firedRule.Confidence),
						Priority:        action.Priority,
						Actions:         []RecommendedAction{},
						CreatedAt:       time.Now(),
					}
					recommendations = append(recommendations, recommendation)
				}
			}
		}
	}
	
	// Generate recommendations based on risk assessment
	if riskAssessment != nil {
		for _, mitigation := range riskAssessment.RiskMitigations {
			recommendation := ClinicalRecommendation{
				RecommendationID: uuid.New(),
				Type:            "risk_mitigation",
				Category:        "risk_based",
				Title:           mitigation.Strategy,
				Description:     mitigation.Description,
				Rationale:       "Risk mitigation recommendation",
				Evidence:        []EvidenceSource{},
				Strength:        c.determineRecommendationStrength(mitigation.EffectivenessScore),
				Priority:        c.convertPriorityString(mitigation.Priority),
				Actions:         []RecommendedAction{},
				CreatedAt:       time.Now(),
			}
			recommendations = append(recommendations, recommendation)
		}
	}
	
	return recommendations, nil
}

// Helper methods for data parsing and processing

func (c *ClinicalIntelligenceService) parseDiagnoses(data interface{}, diagnosisType string) []Diagnosis {
	// Implementation would parse diagnosis data from snapshot
	return []Diagnosis{}
}

func (c *ClinicalIntelligenceService) parseMedications(data interface{}) []MedicationSummary {
	// Implementation would parse medication data from snapshot
	return []MedicationSummary{}
}

func (c *ClinicalIntelligenceService) parseAllergies(data interface{}) []AllergyInfo {
	// Implementation would parse allergy data from snapshot
	return []AllergyInfo{}
}

func (c *ClinicalIntelligenceService) parseVitalSigns(data interface{}) *VitalSignsSummary {
	// Implementation would parse vital signs data from snapshot
	return &VitalSignsSummary{
		LastUpdated: time.Now(),
	}
}

func (c *ClinicalIntelligenceService) parseLabResults(data interface{}) []LabResultSummary {
	// Implementation would parse lab results data from snapshot
	return []LabResultSummary{}
}

func (c *ClinicalIntelligenceService) identifyRiskFactors(findings *ClinicalFindings) []RiskFactor {
	// Implementation would identify risk factors based on clinical findings
	return []RiskFactor{}
}

func (c *ClinicalIntelligenceService) calculateQualityScore(result *ClinicalIntelligenceResult) float64 {
	// Calculate quality score based on completeness, accuracy, and processing success
	baseScore := 0.8
	
	// Adjust based on warnings
	warningPenalty := float64(len(result.Warnings)) * 0.05
	qualityScore := baseScore - warningPenalty
	
	// Ensure score is within valid range
	if qualityScore < 0.0 {
		qualityScore = 0.0
	} else if qualityScore > 1.0 {
		qualityScore = 1.0
	}
	
	return qualityScore
}

func (c *ClinicalIntelligenceService) countDataPoints(findings *ClinicalFindings) int {
	count := len(findings.PrimaryDiagnoses) + len(findings.SecondaryDiagnoses) +
		len(findings.ActiveMedications) + len(findings.Allergies) +
		len(findings.LabResults) + len(findings.RiskFactors)
	
	if findings.VitalSigns != nil {
		count += 8 // Approximate number of vital sign measurements
	}
	
	return count
}

func (c *ClinicalIntelligenceService) countRulesEvaluated(results []RuleEngineResult) int {
	total := 0
	for _, result := range results {
		total += result.RulesEvaluated
	}
	return total
}

func (c *ClinicalIntelligenceService) determineRecommendationStrength(confidence float64) string {
	if confidence >= 0.9 {
		return "strong"
	} else if confidence >= 0.7 {
		return "moderate"
	} else if confidence >= 0.5 {
		return "weak"
	}
	return "insufficient_evidence"
}

func (c *ClinicalIntelligenceService) convertPriorityString(priority string) int {
	switch priority {
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 2
	}
}

func (c *ClinicalIntelligenceService) initializeRuleEngines() {
	// Initialize rule engines based on configuration
	// This would create instances of different rule engines
	c.logger.Info("Initializing rule engines", zap.Strings("engines", c.config.EnableRuleEngines))
}

func (c *ClinicalIntelligenceService) updateProcessingStats(processingTime time.Duration, success bool) {
	c.processingStats.TotalRequests++
	c.processingStats.LastProcessingTime = processingTime
	c.processingStats.LastRequestAt = time.Now()
	
	if success {
		c.processingStats.SuccessfulRequests++
	} else {
		c.processingStats.FailedRequests++
	}
	
	// Update average processing time
	if c.processingStats.TotalRequests > 0 {
		totalTime := c.processingStats.AverageProcessingTime * time.Duration(c.processingStats.TotalRequests-1)
		c.processingStats.AverageProcessingTime = (totalTime + processingTime) / time.Duration(c.processingStats.TotalRequests)
	}
}

func (c *ClinicalIntelligenceService) auditEvent(ctx context.Context, eventType, actor string, data interface{}) {
	if c.auditService == nil {
		return
	}
	
	auditData, _ := json.Marshal(data)
	c.auditService.LogEvent(ctx, &AuditEvent{
		EventType: eventType,
		ActorID:   actor,
		Data:      string(auditData),
		Timestamp: time.Now(),
	})
}

// GetProcessingStatistics returns current processing statistics
func (c *ClinicalIntelligenceService) GetProcessingStatistics() *ProcessingStatistics {
	return c.processingStats
}

// IsHealthy returns the health status of the service
func (c *ClinicalIntelligenceService) IsHealthy(ctx context.Context) bool {
	// Check Rust engine health
	if c.rustEngineClient != nil && !c.rustEngineClient.IsHealthy() {
		return false
	}
	
	// Check knowledge base clients health
	for _, client := range c.knowledgeBaseClients {
		if !client.IsHealthy() {
			return false
		}
	}
	
	// Check rule engines health
	for _, engine := range c.ruleEngines {
		if !engine.IsHealthy() {
			return false
		}
	}
	
	return true
}

// ProcessPhase3IntelligenceWithRustEngine processes Phase 3 clinical intelligence using the high-performance Rust engine
func (c *ClinicalIntelligenceService) ProcessPhase3IntelligenceWithRustEngine(ctx context.Context, request *ClinicalIntelligenceRequest) (*ClinicalIntelligenceResult, error) {
	startTime := time.Now()
	processingID := uuid.New()

	c.logger.Info("Starting Phase 3 clinical intelligence processing with Rust engine",
		zap.String("processing_id", processingID.String()),
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("patient_id", request.PatientID))

	// Audit the processing start
	c.auditEvent(ctx, "phase3_intelligence_start", request.RequestedBy, request)

	c.processingMutex.Lock()
	defer c.processingMutex.Unlock()

	// Convert snapshot data to appropriate format
	snapshotData, err := c.convertToSnapshotData(request.SnapshotData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert snapshot data: %w", err)
	}

	// Define clinical operations for Phase 3 processing
	operations := []ClinicalOperation{
		{
			OperationType:      "drug_interactions",
			OperationID:        "drug_interaction_analysis",
			Parameters:         map[string]interface{}{
				"analysis_depth": "comprehensive",
				"priority":      "high",
			},
			PerformanceTarget:  50 * time.Millisecond,
			RequiredConfidence: 0.85,
		},
		{
			OperationType:      "safety_validation",
			OperationID:        "safety_validation_check",
			Parameters:         map[string]interface{}{
				"validation_level": "comprehensive",
				"proposed_medication": c.extractProposedMedicationFromParams(request.ClinicalParams),
			},
			PerformanceTarget:  75 * time.Millisecond,
			RequiredConfidence: 0.90,
		},
		{
			OperationType:      "rule_evaluation",
			OperationID:        "clinical_rules_evaluation",
			Parameters:         map[string]interface{}{
				"rule_set": "drug_rules",
				"priority": "high",
				"rule_filters": []string{"safety", "efficacy", "interactions"},
			},
			PerformanceTarget:  40 * time.Millisecond,
			RequiredConfidence: 0.80,
		},
	}

	// Add dosage calculations if medication is specified
	if request.ClinicalParams != nil && request.ClinicalParams.MedicationContext != nil {
		operations = append(operations, ClinicalOperation{
			OperationType:      "dosage_calculation",
			OperationID:        "dosage_optimization",
			Parameters:         map[string]interface{}{
				"medication_code":   c.extractMedicationCodeFromParams(request.ClinicalParams),
				"calculation_type":  "standard",
				"patient_weight":    c.extractPatientWeight(snapshotData),
				"patient_age":       c.extractPatientAge(snapshotData),
			},
			PerformanceTarget:  30 * time.Millisecond,
			RequiredConfidence: 0.85,
		})
	}

	// Create Phase 3 request
	phase3Request := &Phase3ClinicalIntelligenceRequest{
		WorkflowID:          request.WorkflowID,
		PatientID:           request.PatientID,
		SnapshotData:        snapshotData,
		RequestedOperations: operations,
		Priority:            "high",
		RequestedBy:         request.RequestedBy,
		RequestedAt:         request.RequestedAt,
	}

	// Process using Clinical Calculation Service
	phase3Response, err := c.clinicalCalculationService.ProcessPhase3Intelligence(ctx, phase3Request)
	if err != nil {
		c.updateProcessingStats(time.Since(startTime), false)
		return nil, fmt.Errorf("Phase 3 processing failed: %w", err)
	}

	// Convert Phase 3 response to Clinical Intelligence Result format
	result := c.convertPhase3ResponseToClinicalResult(phase3Response, processingID, request)

	// Add processing metrics
	result.ProcessingMetrics = &ProcessingMetrics{
		TotalProcessingTime:      phase3Response.ProcessingTime,
		DataExtractionTime:       0, // Extracted from snapshot, no additional extraction needed
		RuleEvaluationTime:       c.extractRuleEvaluationTime(phase3Response),
		RiskAssessmentTime:       0, // Risk assessment is part of overall processing
		SafetyCheckTime:          c.extractSafetyCheckTime(phase3Response),
		RecommendationTime:       0, // Recommendations are generated as part of operations
		DataPointsProcessed:      c.countDataPointsFromSnapshot(snapshotData),
		RulesEvaluated:          c.countRulesEvaluatedFromPhase3(phase3Response),
		RecommendationsGenerated: len(result.ClinicalRecommendations),
		ComponentMetrics: map[string]interface{}{
			"rust_engine_operations":  len(phase3Response.OperationResults),
			"performance_targets_met": phase3Response.PerformanceMetrics.TargetsMet,
			"cache_hit_rate":         phase3Response.PerformanceMetrics.CacheHitRate,
			"overall_quality_score":  phase3Response.QualityScore,
		},
	}

	// Calculate overall quality score
	result.QualityScore = c.calculatePhase3QualityScore(phase3Response, result)

	// Update processing statistics
	processingTime := time.Since(startTime)
	c.updateProcessingStats(processingTime, phase3Response.Success)

	// Audit the processing completion
	c.auditEvent(ctx, "phase3_intelligence_complete", request.RequestedBy, result)

	c.logger.Info("Phase 3 clinical intelligence processing completed",
		zap.String("processing_id", processingID.String()),
		zap.Duration("processing_time", processingTime),
		zap.Bool("success", phase3Response.Success),
		zap.Float64("quality_score", result.QualityScore),
		zap.Int("operations_completed", phase3Response.PerformanceMetrics.SuccessfulOperations))

	return result, nil
}

// Helper methods for Phase 3 processing

func (c *ClinicalIntelligenceService) convertToSnapshotData(data interface{}) (*SnapshotBasedContextData, error) {
	// Convert the interface{} to SnapshotBasedContextData
	// This would depend on the actual structure of the incoming data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot data: %w", err)
	}

	var snapshotData SnapshotBasedContextData
	if err := json.Unmarshal(jsonData, &snapshotData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot data: %w", err)
	}

	return &snapshotData, nil
}

func (c *ClinicalIntelligenceService) extractProposedMedicationFromParams(params *ClinicalIntelligenceParams) map[string]interface{} {
	if params == nil || params.MedicationContext == nil {
		return nil
	}

	// Extract proposed medication from clinical parameters
	return map[string]interface{}{
		"medication_code": params.MedicationContext.MedicationCode,
		"name":           params.MedicationContext.MedicationName,
		"dose":           params.MedicationContext.ProposedDose,
		"route":          params.MedicationContext.Route,
		"frequency":      params.MedicationContext.Frequency,
		"is_active":      true,
	}
}

func (c *ClinicalIntelligenceService) extractMedicationCodeFromParams(params *ClinicalIntelligenceParams) string {
	if params == nil || params.MedicationContext == nil {
		return ""
	}
	return params.MedicationContext.MedicationCode
}

func (c *ClinicalIntelligenceService) extractPatientWeight(snapshotData *SnapshotBasedContextData) float64 {
	// Extract patient weight from snapshot data
	if snapshotData != nil && snapshotData.PatientData != nil {
		if weightData, ok := snapshotData.PatientData["weight"]; ok {
			if weight, ok := weightData.(float64); ok {
				return weight
			}
			if weightInt, ok := weightData.(int); ok {
				return float64(weightInt)
			}
		}
	}
	return 70.0 // Default weight if not available
}

func (c *ClinicalIntelligenceService) extractPatientAge(snapshotData *SnapshotBasedContextData) int {
	// Extract patient age from snapshot data
	if snapshotData != nil && snapshotData.PatientData != nil {
		if ageData, ok := snapshotData.PatientData["age"]; ok {
			if age, ok := ageData.(int); ok {
				return age
			}
			if ageFloat, ok := ageData.(float64); ok {
				return int(ageFloat)
			}
		}
	}
	return 35 // Default age if not available
}

func (c *ClinicalIntelligenceService) convertPhase3ResponseToClinicalResult(phase3Response *Phase3ClinicalIntelligenceResponse, processingID uuid.UUID, request *ClinicalIntelligenceRequest) *ClinicalIntelligenceResult {
	result := &ClinicalIntelligenceResult{
		ProcessingID:            processingID,
		WorkflowID:              phase3Response.WorkflowID,
		PatientID:               phase3Response.PatientID,
		ClinicalFindings:        c.extractClinicalFindingsFromPhase3(phase3Response),
		RiskAssessment:          c.convertOverallRiskToRiskAssessment(phase3Response.OverallRiskAssessment),
		RuleEngineResults:       c.convertPhase3RuleResults(phase3Response.RuleEvaluationResults),
		ClinicalRecommendations: c.convertPhase3Recommendations(phase3Response),
		SafetyChecks:            c.convertPhase3SafetyResults(phase3Response.SafetyValidationResults),
		Warnings:                c.convertPhase3Warnings(phase3Response.Warnings),
		ProcessedAt:             time.Now(),
	}

	return result
}

func (c *ClinicalIntelligenceService) extractClinicalFindingsFromPhase3(phase3Response *Phase3ClinicalIntelligenceResponse) *ClinicalFindings {
	// Extract clinical findings from Phase 3 response
	// This would process the drug interaction and safety validation results
	findings := &ClinicalFindings{
		PrimaryDiagnoses:    []Diagnosis{},
		SecondaryDiagnoses:  []Diagnosis{},
		ActiveMedications:   []MedicationSummary{},
		Allergies:           []AllergyInfo{},
		VitalSigns:          nil,
		LabResults:          []LabResultSummary{},
		RiskFactors:         []RiskFactor{},
		ClinicalIndicators:  make(map[string]interface{}),
	}

	// Process drug interaction analysis if available
	if phase3Response.DrugInteractionAnalysis != nil {
		for _, interaction := range phase3Response.DrugInteractionAnalysis.Interactions {
			// Convert drug interactions to risk factors
			findings.RiskFactors = append(findings.RiskFactors, RiskFactor{
				Factor:       fmt.Sprintf("Drug interaction: %s - %s", interaction.Medication1Name, interaction.Medication2Name),
				Category:     "drug_interaction",
				Severity:     interaction.Severity,
				Value:        interaction.Effect,
				Impact:       interaction.Action,
				IdentifiedAt: time.Now(),
			})
		}
	}

	return findings
}

func (c *ClinicalIntelligenceService) convertOverallRiskToRiskAssessment(overallRisk *OverallRiskAssessment) *RiskAssessmentResult {
	if overallRisk == nil {
		return nil
	}

	assessment := &RiskAssessmentResult{
		OverallRiskScore: overallRisk.OverallRiskScore,
		RiskCategory:     overallRisk.RiskLevel,
		PrimaryRisks:     []IdentifiedRisk{},
		SecondaryRisks:   []IdentifiedRisk{},
		RiskMitigations:  []RiskMitigation{},
		AssessmentMetrics: map[string]float64{
			"overall_risk_score": overallRisk.OverallRiskScore,
		},
		AssessmentDate:   time.Now(),
	}

	// Convert risk factors to identified risks
	for _, riskFactor := range overallRisk.PrimaryRiskFactors {
		assessment.PrimaryRisks = append(assessment.PrimaryRisks, IdentifiedRisk{
			RiskID:        uuid.New(),
			RiskType:      riskFactor.Category,
			Description:   riskFactor.Description,
			Probability:   riskFactor.Score / 10.0, // Normalize to 0-1 range
			Impact:        riskFactor.Impact,
			Severity:      riskFactor.Severity,
			Evidence:      []string{riskFactor.Evidence},
			ClinicalBasis: riskFactor.Mitigation,
		})
	}

	return assessment
}

func (c *ClinicalIntelligenceService) convertPhase3RuleResults(ruleResults []clients.ClinicalRuleEvaluationResponse) []RuleEngineResult {
	var results []RuleEngineResult

	for _, ruleResult := range ruleResults {
		result := RuleEngineResult{
			EngineID:       "rust_clinical_engine",
			EngineName:     "Rust Clinical Engine",
			RulesEvaluated: len(ruleResult.EvaluatedRules),
			RulesFired:     c.countFiredRules(ruleResult.EvaluatedRules),
			FiredRules:     []FiredRule{},
			ProcessingTime: ruleResult.ProcessingTime,
			QualityScore:   ruleResult.OverallScore,
			EngineMetrics:  map[string]interface{}{
				"overall_score": ruleResult.OverallScore,
				"rule_set":     ruleResult.RuleSet,
			},
		}

		// Convert evaluated rules to fired rules
		for _, evaluatedRule := range ruleResult.EvaluatedRules {
			if evaluatedRule.Triggered {
				result.FiredRules = append(result.FiredRules, FiredRule{
					RuleID:     evaluatedRule.RuleID,
					RuleName:   evaluatedRule.RuleName,
					RuleType:   evaluatedRule.RuleType,
					Priority:   1, // Default priority
					Confidence: evaluatedRule.Score,
					Conditions: []RuleCondition{}, // Would need to be extracted from parameters
					Actions:    []RuleAction{},    // Would need to be derived from rule logic
					Evidence:   []string{evaluatedRule.Evidence},
					FiredAt:    time.Now(),
				})
			}
		}

		results = append(results, result)
	}

	return results
}

func (c *ClinicalIntelligenceService) convertPhase3Recommendations(phase3Response *Phase3ClinicalIntelligenceResponse) []ClinicalRecommendation {
	var recommendations []ClinicalRecommendation

	// Convert recommendations from overall risk assessment
	if phase3Response.OverallRiskAssessment != nil {
		for _, rec := range phase3Response.OverallRiskAssessment.Recommendations {
			recommendations = append(recommendations, ClinicalRecommendation{
				RecommendationID: uuid.New(),
				Type:            "clinical_guidance",
				Category:        rec.Category,
				Title:           rec.Title,
				Description:     rec.Description,
				Rationale:       rec.Evidence,
				Evidence:        []EvidenceSource{},
				Strength:        c.determineRecommendationStrength(rec.Confidence),
				Priority:        c.convertPriorityString(rec.Priority),
				Actions:         []RecommendedAction{},
				CreatedAt:       time.Now(),
			})
		}
	}

	// Convert recommendations from rule evaluation results
	for _, ruleResult := range phase3Response.RuleEvaluationResults {
		for _, rec := range ruleResult.Recommendations {
			recommendations = append(recommendations, ClinicalRecommendation{
				RecommendationID: uuid.New(),
				Type:            "rule_based",
				Category:        rec.Category,
				Title:           rec.Title,
				Description:     rec.Description,
				Rationale:       rec.Evidence,
				Evidence:        []EvidenceSource{},
				Strength:        c.determineRecommendationStrength(rec.Confidence),
				Priority:        c.convertPriorityString(rec.Priority),
				Actions:         []RecommendedAction{},
				CreatedAt:       time.Now(),
			})
		}
	}

	return recommendations
}

func (c *ClinicalIntelligenceService) convertPhase3SafetyResults(safetyResults []clients.SafetyValidationResponse) *SafetyCheckResults {
	if len(safetyResults) == 0 {
		return nil
	}

	// Combine all safety results
	allInteractionWarnings := []InteractionWarning{}
	allDosageWarnings := []DosageWarning{}
	allMonitoringAlerts := []MonitoringAlert{}

	overallRiskScore := 0.0
	for _, safetyResult := range safetyResults {
		overallRiskScore += safetyResult.OverallRiskScore

		// Convert safety alerts to warnings
		for _, alert := range safetyResult.SafetyAlerts {
			if alert.Category == "drug_interaction" {
				allInteractionWarnings = append(allInteractionWarnings, InteractionWarning{
					WarningID:     uuid.New(),
					Medication1:   "", // Would need to be extracted from alert
					Medication2:   "", // Would need to be extracted from alert
					InteractionType: alert.Category,
					Severity:      alert.Severity,
					Description:   alert.Description,
					Recommendation: alert.Action,
					Override:      alert.Override,
				})
			}
		}
	}

	return &SafetyCheckResults{
		OverallRiskScore:      overallRiskScore / float64(len(safetyResults)),
		InteractionWarnings:   allInteractionWarnings,
		DosageWarnings:       allDosageWarnings,
		MonitoringAlerts:     allMonitoringAlerts,
		ProcessingTime:       time.Duration(0), // Would be calculated from individual processing times
	}
}

func (c *ClinicalIntelligenceService) convertPhase3Warnings(warnings []ClinicalCalculationWarning) []ClinicalWarning {
	var clinicalWarnings []ClinicalWarning

	for _, warning := range warnings {
		clinicalWarnings = append(clinicalWarnings, ClinicalWarning{
			WarningID: uuid.New(),
			Type:      "calculation_warning",
			Message:   warning.WarningMessage,
			Severity:  warning.Severity,
			Context:   warning.OperationType,
			Actionable: true,
			Timestamp: warning.Timestamp,
		})
	}

	return clinicalWarnings
}

func (c *ClinicalIntelligenceService) extractRuleEvaluationTime(phase3Response *Phase3ClinicalIntelligenceResponse) time.Duration {
	var totalTime time.Duration
	for _, result := range phase3Response.OperationResults {
		if result.OperationType == "rule_evaluation" {
			totalTime += result.ProcessingTime
		}
	}
	return totalTime
}

func (c *ClinicalIntelligenceService) extractSafetyCheckTime(phase3Response *Phase3ClinicalIntelligenceResponse) time.Duration {
	var totalTime time.Duration
	for _, result := range phase3Response.OperationResults {
		if result.OperationType == "safety_validation" {
			totalTime += result.ProcessingTime
		}
	}
	return totalTime
}

func (c *ClinicalIntelligenceService) countDataPointsFromSnapshot(snapshotData *SnapshotBasedContextData) int {
	count := 0
	if snapshotData != nil {
		if snapshotData.PatientData != nil {
			count += len(snapshotData.PatientData)
		}
		if snapshotData.MedicationData != nil {
			count += len(snapshotData.MedicationData)
		}
		if snapshotData.ClinicalData != nil {
			count += len(snapshotData.ClinicalData)
		}
	}
	return count
}

func (c *ClinicalIntelligenceService) countRulesEvaluatedFromPhase3(phase3Response *Phase3ClinicalIntelligenceResponse) int {
	total := 0
	for _, ruleResult := range phase3Response.RuleEvaluationResults {
		total += len(ruleResult.EvaluatedRules)
	}
	return total
}

func (c *ClinicalIntelligenceService) countFiredRules(evaluatedRules []clients.EvaluatedRule) int {
	count := 0
	for _, rule := range evaluatedRules {
		if rule.Triggered {
			count++
		}
	}
	return count
}

func (c *ClinicalIntelligenceService) calculatePhase3QualityScore(phase3Response *Phase3ClinicalIntelligenceResponse, result *ClinicalIntelligenceResult) float64 {
	// Base quality score from Phase 3 response
	baseScore := phase3Response.QualityScore

	// Adjust for completeness of results
	completenessScore := 0.0
	if result.ClinicalFindings != nil {
		completenessScore += 0.25
	}
	if result.RiskAssessment != nil {
		completenessScore += 0.25
	}
	if len(result.RuleEngineResults) > 0 {
		completenessScore += 0.25
	}
	if result.SafetyChecks != nil {
		completenessScore += 0.25
	}

	// Combine scores
	finalScore := (baseScore * 0.7) + (completenessScore * 0.3)

	// Apply warning penalty
	warningPenalty := float64(len(result.Warnings)) * 0.05
	finalScore -= warningPenalty

	// Ensure score is within valid range
	if finalScore < 0.0 {
		finalScore = 0.0
	} else if finalScore > 1.0 {
		finalScore = 1.0
	}

	return finalScore
}