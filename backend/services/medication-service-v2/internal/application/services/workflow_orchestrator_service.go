package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WorkflowPhase represents the current phase of the medication workflow
type WorkflowPhase int

const (
	PhaseRecipeResolution WorkflowPhase = iota + 1
	PhaseContextAssembly
	PhaseClinicalIntelligence
	PhaseProposalGeneration
)

// WorkflowPhaseStatus represents the status of a workflow phase
type WorkflowPhaseStatus int

const (
	StatusPending WorkflowPhaseStatus = iota
	StatusInProgress
	StatusCompleted
	StatusFailed
	StatusSkipped
)

// WorkflowExecutionRequest represents a complete 4-phase workflow request
type WorkflowExecutionRequest struct {
	WorkflowID        uuid.UUID                      `json:"workflow_id"`
	RequestID         string                         `json:"request_id"`
	PatientID         string                         `json:"patient_id"`
	RecipeID          string                         `json:"recipe_id"`
	RequestedBy       string                         `json:"requested_by"`
	PatientContext    map[string]interface{}         `json:"patient_context"`
	ClinicalParams    *ClinicalIntelligenceParams    `json:"clinical_params,omitempty"`
	ProposalParams    *ProposalGenerationParams      `json:"proposal_params,omitempty"`
	Options           *WorkflowExecutionOptions      `json:"options,omitempty"`
	CreatedAt         time.Time                      `json:"created_at"`
}

// ClinicalIntelligenceParams contains parameters for Phase 3
type ClinicalIntelligenceParams struct {
	RuleEngines           []string                   `json:"rule_engines"`
	QualityThresholds     map[string]float64         `json:"quality_thresholds"`
	EnableAdvancedRules   bool                       `json:"enable_advanced_rules"`
	ClinicalContext       map[string]interface{}     `json:"clinical_context"`
	RiskAssessment        *RiskAssessmentParams      `json:"risk_assessment,omitempty"`
}

// RiskAssessmentParams contains risk assessment parameters
type RiskAssessmentParams struct {
	EnableRiskScoring     bool     `json:"enable_risk_scoring"`
	RiskFactors          []string  `json:"risk_factors"`
	MinRiskThreshold     float64   `json:"min_risk_threshold"`
}

// ProposalGenerationParams contains parameters for Phase 4
type ProposalGenerationParams struct {
	ProposalTypes         []string                   `json:"proposal_types"`
	IncludeAlternatives   bool                       `json:"include_alternatives"`
	MaxProposals          int                        `json:"max_proposals"`
	QualityThreshold      float64                    `json:"quality_threshold"`
	FHIRCompliance        *FHIRComplianceParams      `json:"fhir_compliance,omitempty"`
}

// FHIRComplianceParams contains FHIR compliance parameters
type FHIRComplianceParams struct {
	ValidateResources     bool     `json:"validate_resources"`
	IncludeProvenance     bool     `json:"include_provenance"`
	FHIRVersion          string   `json:"fhir_version"`
}

// WorkflowExecutionOptions contains workflow execution options
type WorkflowExecutionOptions struct {
	EnableParallelPhases  bool          `json:"enable_parallel_phases"`
	TimeoutPerPhase      time.Duration  `json:"timeout_per_phase"`
	MaxRetries           int            `json:"max_retries"`
	FailFast             bool           `json:"fail_fast"`
	EnableAuditTrail     bool           `json:"enable_audit_trail"`
	PerformanceTargets   *PerformanceTargets `json:"performance_targets,omitempty"`
}

// PerformanceTargets defines performance expectations
type PerformanceTargets struct {
	TotalLatencyTarget   time.Duration `json:"total_latency_target"`
	Phase1Target         time.Duration `json:"phase1_target"`
	Phase2Target         time.Duration `json:"phase2_target"`
	Phase3Target         time.Duration `json:"phase3_target"`
	Phase4Target         time.Duration `json:"phase4_target"`
}

// WorkflowExecutionResult represents the complete workflow result
type WorkflowExecutionResult struct {
	WorkflowID           uuid.UUID                  `json:"workflow_id"`
	RequestID            string                     `json:"request_id"`
	Status               WorkflowStatus             `json:"status"`
	Phases               map[WorkflowPhase]*PhaseResult `json:"phases"`
	FinalProposal        *MedicationProposal        `json:"final_proposal,omitempty"`
	Alternatives         []*MedicationProposal      `json:"alternatives,omitempty"`
	QualityMetrics       *WorkflowQualityMetrics    `json:"quality_metrics"`
	PerformanceMetrics   *WorkflowPerformanceMetrics `json:"performance_metrics"`
	Errors               []WorkflowError            `json:"errors,omitempty"`
	Warnings             []WorkflowWarning          `json:"warnings,omitempty"`
	AuditTrail           []WorkflowAuditEntry       `json:"audit_trail,omitempty"`
	CompletedAt          time.Time                  `json:"completed_at"`
	TotalDuration        time.Duration              `json:"total_duration"`
}

// WorkflowStatus represents the overall workflow status
type WorkflowStatus int

const (
	WorkflowStatusPending WorkflowStatus = iota
	WorkflowStatusInProgress
	WorkflowStatusCompleted
	WorkflowStatusFailed
	WorkflowStatusPartiallyCompleted
)

// PhaseResult represents the result of a single workflow phase
type PhaseResult struct {
	Phase           WorkflowPhase       `json:"phase"`
	Status          WorkflowPhaseStatus `json:"status"`
	StartTime       time.Time           `json:"start_time"`
	EndTime         time.Time           `json:"end_time"`
	Duration        time.Duration       `json:"duration"`
	Result          interface{}         `json:"result,omitempty"`
	QualityScore    float64             `json:"quality_score"`
	Errors          []string            `json:"errors,omitempty"`
	Warnings        []string            `json:"warnings,omitempty"`
	Metrics         map[string]float64  `json:"metrics,omitempty"`
}

// WorkflowQualityMetrics contains quality metrics for the entire workflow
type WorkflowQualityMetrics struct {
	OverallQuality       float64            `json:"overall_quality"`
	PhaseQualities       map[WorkflowPhase]float64 `json:"phase_qualities"`
	DataCompleteness     float64            `json:"data_completeness"`
	ClinicalAccuracy     float64            `json:"clinical_accuracy"`
	FHIRCompliance       float64            `json:"fhir_compliance"`
	SafetyScore          float64            `json:"safety_score"`
}

// WorkflowPerformanceMetrics contains performance metrics
type WorkflowPerformanceMetrics struct {
	TotalLatency         time.Duration      `json:"total_latency"`
	PhaseLatencies       map[WorkflowPhase]time.Duration `json:"phase_latencies"`
	ThroughputRPS        float64            `json:"throughput_rps"`
	ErrorRate            float64            `json:"error_rate"`
	RetryCount           int                `json:"retry_count"`
	CacheHitRate         float64            `json:"cache_hit_rate"`
}

// WorkflowError represents a workflow error
type WorkflowError struct {
	Phase       WorkflowPhase `json:"phase"`
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	Details     interface{}   `json:"details,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Retryable   bool          `json:"retryable"`
}

// WorkflowWarning represents a workflow warning
type WorkflowWarning struct {
	Phase       WorkflowPhase `json:"phase"`
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	Details     interface{}   `json:"details,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// WorkflowAuditEntry represents an audit trail entry
type WorkflowAuditEntry struct {
	Phase       WorkflowPhase `json:"phase"`
	Action      string        `json:"action"`
	Actor       string        `json:"actor"`
	Timestamp   time.Time     `json:"timestamp"`
	Details     interface{}   `json:"details,omitempty"`
}

// MedicationProposal represents a final medication proposal
type MedicationProposal struct {
	ProposalID          uuid.UUID              `json:"proposal_id"`
	MedicationName      string                 `json:"medication_name"`
	Dosage              string                 `json:"dosage"`
	Frequency           string                 `json:"frequency"`
	Duration            string                 `json:"duration"`
	Route               string                 `json:"route"`
	Instructions        string                 `json:"instructions"`
	ClinicalRationale   string                 `json:"clinical_rationale"`
	SafetyAlerts        []SafetyAlert          `json:"safety_alerts,omitempty"`
	DrugInteractions    []DrugInteraction      `json:"drug_interactions,omitempty"`
	AllergyAlerts       []AllergyAlert         `json:"allergy_alerts,omitempty"`
	QualityScore        float64                `json:"quality_score"`
	ConfidenceLevel     float64                `json:"confidence_level"`
	FHIRResources       []FHIRResource         `json:"fhir_resources,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
}

// SafetyAlert represents a medication safety alert
type SafetyAlert struct {
	AlertID     uuid.UUID `json:"alert_id"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Source      string    `json:"source"`
	Timestamp   time.Time `json:"timestamp"`
}

// DrugInteraction represents a potential drug interaction
type DrugInteraction struct {
	InteractionID   uuid.UUID `json:"interaction_id"`
	InteractingDrug string    `json:"interacting_drug"`
	Severity        string    `json:"severity"`
	Description     string    `json:"description"`
	Recommendation  string    `json:"recommendation"`
}

// AllergyAlert represents an allergy alert
type AllergyAlert struct {
	AlertID         uuid.UUID `json:"alert_id"`
	AllergenName    string    `json:"allergen_name"`
	AllergyType     string    `json:"allergy_type"`
	Severity        string    `json:"severity"`
	CrossReactivity []string  `json:"cross_reactivity,omitempty"`
}

// FHIRResource represents a FHIR-compliant resource
type FHIRResource struct {
	ResourceType    string      `json:"resourceType"`
	ID              string      `json:"id"`
	Resource        interface{} `json:"resource"`
	Validated       bool        `json:"validated"`
}

// WorkflowOrchestratorConfig contains configuration for the workflow orchestrator
type WorkflowOrchestratorConfig struct {
	DefaultTimeoutPerPhase   time.Duration `mapstructure:"default_timeout_per_phase" default:"30s"`
	MaxConcurrentWorkflows   int           `mapstructure:"max_concurrent_workflows" default:"50"`
	EnableParallelPhases     bool          `mapstructure:"enable_parallel_phases" default:"true"`
	DefaultMaxRetries        int           `mapstructure:"default_max_retries" default:"3"`
	PerformanceTarget        time.Duration `mapstructure:"performance_target" default:"250ms"`
	QualityThreshold         float64       `mapstructure:"quality_threshold" default:"0.8"`
	EnableStatePersistence   bool          `mapstructure:"enable_state_persistence" default:"true"`
	StateCleanupInterval     time.Duration `mapstructure:"state_cleanup_interval" default:"1h"`
	MaxRetainedStates        int           `mapstructure:"max_retained_states" default:"1000"`
}

// WorkflowOrchestratorService orchestrates the complete 4-phase medication workflow
type WorkflowOrchestratorService struct {
	// Phase services
	recipeResolverIntegration    *RecipeResolverContextIntegration
	clinicalIntelligenceService  *ClinicalIntelligenceService
	proposalGenerationService    *ProposalGenerationService
	workflowStateService         *WorkflowStateService
	
	// Supporting services
	auditService                 *AuditService
	metricsService              *MetricsService
	
	// Configuration and logging
	config                      WorkflowOrchestratorConfig
	logger                      *zap.Logger
	
	// Internal state
	activeWorkflows             map[uuid.UUID]*WorkflowExecutionContext
	performanceMonitor          *PerformanceMonitor
}

// WorkflowExecutionContext maintains context throughout workflow execution
type WorkflowExecutionContext struct {
	Request         *WorkflowExecutionRequest
	State           *WorkflowState
	StartTime       time.Time
	PhaseResults    map[WorkflowPhase]*PhaseResult
	Errors          []WorkflowError
	Warnings        []WorkflowWarning
	AuditTrail      []WorkflowAuditEntry
	CancelFunc      context.CancelFunc
}

// NewWorkflowOrchestratorService creates a new workflow orchestrator service
func NewWorkflowOrchestratorService(
	recipeResolverIntegration *RecipeResolverContextIntegration,
	clinicalIntelligenceService *ClinicalIntelligenceService,
	proposalGenerationService *ProposalGenerationService,
	workflowStateService *WorkflowStateService,
	auditService *AuditService,
	metricsService *MetricsService,
	config WorkflowOrchestratorConfig,
	logger *zap.Logger,
) *WorkflowOrchestratorService {
	return &WorkflowOrchestratorService{
		recipeResolverIntegration:   recipeResolverIntegration,
		clinicalIntelligenceService: clinicalIntelligenceService,
		proposalGenerationService:   proposalGenerationService,
		workflowStateService:        workflowStateService,
		auditService:                auditService,
		metricsService:             metricsService,
		config:                     config,
		logger:                     logger,
		activeWorkflows:            make(map[uuid.UUID]*WorkflowExecutionContext),
		performanceMonitor:         NewPerformanceMonitor(logger),
	}
}

// ExecuteWorkflow executes the complete 4-phase medication workflow
func (w *WorkflowOrchestratorService) ExecuteWorkflow(
	ctx context.Context,
	request *WorkflowExecutionRequest,
) (*WorkflowExecutionResult, error) {
	// Start performance monitoring
	monitor := w.performanceMonitor.StartExecution(request.WorkflowID)
	defer monitor.Complete()
	
	w.logger.Info("Starting 4-phase medication workflow execution",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("patient_id", request.PatientID),
		zap.String("recipe_id", request.RecipeID),
	)
	
	// Initialize workflow context
	workflowCtx, cancel := context.WithTimeout(ctx, w.getWorkflowTimeout(request))
	defer cancel()
	
	execContext := &WorkflowExecutionContext{
		Request:      request,
		State:        w.initializeWorkflowState(request),
		StartTime:    time.Now(),
		PhaseResults: make(map[WorkflowPhase]*PhaseResult),
		Errors:       []WorkflowError{},
		Warnings:     []WorkflowWarning{},
		AuditTrail:   []WorkflowAuditEntry{},
		CancelFunc:   cancel,
	}
	
	// Store active workflow
	w.activeWorkflows[request.WorkflowID] = execContext
	defer delete(w.activeWorkflows, request.WorkflowID)
	
	// Audit workflow start
	w.auditWorkflowEvent(execContext, "workflow_started", request.RequestedBy, request)
	
	// Persist initial state if enabled
	if w.config.EnableStatePersistence {
		if err := w.workflowStateService.PersistState(workflowCtx, execContext.State); err != nil {
			w.logger.Warn("Failed to persist initial workflow state", zap.Error(err))
		}
	}
	
	// Execute phases sequentially or in parallel based on configuration
	var finalResult *WorkflowExecutionResult
	var err error
	
	if w.config.EnableParallelPhases && request.Options != nil && request.Options.EnableParallelPhases {
		finalResult, err = w.executeParallelPhases(workflowCtx, execContext)
	} else {
		finalResult, err = w.executeSequentialPhases(workflowCtx, execContext)
	}
	
	// Complete workflow execution
	finalResult = w.completeWorkflowExecution(execContext, finalResult, err)
	
	// Persist final state
	if w.config.EnableStatePersistence {
		execContext.State.Status = finalResult.Status
		execContext.State.CompletedAt = &finalResult.CompletedAt
		if persistErr := w.workflowStateService.PersistState(workflowCtx, execContext.State); persistErr != nil {
			w.logger.Warn("Failed to persist final workflow state", zap.Error(persistErr))
		}
	}
	
	// Audit workflow completion
	w.auditWorkflowEvent(execContext, "workflow_completed", request.RequestedBy, finalResult)
	
	// Update metrics
	w.updateWorkflowMetrics(finalResult)
	
	w.logger.Info("Completed 4-phase medication workflow execution",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("status", fmt.Sprintf("%d", finalResult.Status)),
		zap.Duration("duration", finalResult.TotalDuration),
		zap.Float64("overall_quality", finalResult.QualityMetrics.OverallQuality),
	)
	
	return finalResult, err
}

// executeSequentialPhases executes all phases sequentially
func (w *WorkflowOrchestratorService) executeSequentialPhases(
	ctx context.Context,
	execContext *WorkflowExecutionContext,
) (*WorkflowExecutionResult, error) {
	phases := []WorkflowPhase{
		PhaseRecipeResolution,
		PhaseContextAssembly,
		PhaseClinicalIntelligence,
		PhaseProposalGeneration,
	}
	
	var lastResult interface{}
	
	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return w.handleWorkflowTimeout(execContext), ctx.Err()
		default:
		}
		
		phaseResult, err := w.executePhase(ctx, execContext, phase, lastResult)
		execContext.PhaseResults[phase] = phaseResult
		
		if err != nil {
			if w.shouldFailFast(execContext.Request, err) {
				return w.handlePhaseFailure(execContext, phase, err), err
			}
			w.addWorkflowError(execContext, phase, err)
		}
		
		if phaseResult.Status == StatusCompleted {
			lastResult = phaseResult.Result
		}
		
		// Update state after each phase
		if w.config.EnableStatePersistence {
			execContext.State.CurrentPhase = phase
			execContext.State.PhaseResults = execContext.PhaseResults
			if persistErr := w.workflowStateService.PersistState(ctx, execContext.State); persistErr != nil {
				w.logger.Warn("Failed to persist phase state", zap.Error(persistErr))
			}
		}
	}
	
	return w.buildWorkflowResult(execContext), nil
}

// executeParallelPhases executes compatible phases in parallel
func (w *WorkflowOrchestratorService) executeParallelPhases(
	ctx context.Context,
	execContext *WorkflowExecutionContext,
) (*WorkflowExecutionResult, error) {
	// Phase 1 and 2 must be sequential (Phase 2 depends on Phase 1)
	// Phase 3 and 4 can run in parallel after Phase 2 completes
	
	// Execute Phase 1: Recipe Resolution
	phase1Result, err := w.executePhase(ctx, execContext, PhaseRecipeResolution, nil)
	execContext.PhaseResults[PhaseRecipeResolution] = phase1Result
	
	if err != nil && w.shouldFailFast(execContext.Request, err) {
		return w.handlePhaseFailure(execContext, PhaseRecipeResolution, err), err
	}
	
	// Execute Phase 2: Context Assembly (depends on Phase 1)
	phase2Result, err := w.executePhase(ctx, execContext, PhaseContextAssembly, phase1Result.Result)
	execContext.PhaseResults[PhaseContextAssembly] = phase2Result
	
	if err != nil && w.shouldFailFast(execContext.Request, err) {
		return w.handlePhaseFailure(execContext, PhaseContextAssembly, err), err
	}
	
	// Execute Phase 3 and 4 in parallel
	phase3Ch := make(chan *PhaseResult, 1)
	phase4Ch := make(chan *PhaseResult, 1)
	errCh := make(chan error, 2)
	
	// Start Phase 3: Clinical Intelligence
	go func() {
		result, err := w.executePhase(ctx, execContext, PhaseClinicalIntelligence, phase2Result.Result)
		phase3Ch <- result
		if err != nil {
			errCh <- err
		}
	}()
	
	// Start Phase 4: Proposal Generation (can use Phase 2 result directly)
	go func() {
		result, err := w.executePhase(ctx, execContext, PhaseProposalGeneration, phase2Result.Result)
		phase4Ch <- result
		if err != nil {
			errCh <- err
		}
	}()
	
	// Collect results
	var phase3Result, phase4Result *PhaseResult
	var phase3Err, phase4Err error
	
	for i := 0; i < 2; i++ {
		select {
		case result := <-phase3Ch:
			phase3Result = result
			execContext.PhaseResults[PhaseClinicalIntelligence] = result
		case result := <-phase4Ch:
			phase4Result = result
			execContext.PhaseResults[PhaseProposalGeneration] = result
		case err := <-errCh:
			if phase3Result == nil {
				phase3Err = err
			} else {
				phase4Err = err
			}
		case <-ctx.Done():
			return w.handleWorkflowTimeout(execContext), ctx.Err()
		}
	}
	
	// Handle parallel phase errors
	if phase3Err != nil && w.shouldFailFast(execContext.Request, phase3Err) {
		return w.handlePhaseFailure(execContext, PhaseClinicalIntelligence, phase3Err), phase3Err
	}
	if phase4Err != nil && w.shouldFailFast(execContext.Request, phase4Err) {
		return w.handlePhaseFailure(execContext, PhaseProposalGeneration, phase4Err), phase4Err
	}
	
	return w.buildWorkflowResult(execContext), nil
}

// executePhase executes a single workflow phase
func (w *WorkflowOrchestratorService) executePhase(
	ctx context.Context,
	execContext *WorkflowExecutionContext,
	phase WorkflowPhase,
	previousResult interface{},
) (*PhaseResult, error) {
	phaseStartTime := time.Now()
	
	w.logger.Debug("Starting workflow phase",
		zap.String("workflow_id", execContext.Request.WorkflowID.String()),
		zap.Int("phase", int(phase)),
	)
	
	// Create phase context with timeout
	phaseTimeout := w.getPhaseTimeout(execContext.Request, phase)
	phaseCtx, cancel := context.WithTimeout(ctx, phaseTimeout)
	defer cancel()
	
	// Initialize phase result
	phaseResult := &PhaseResult{
		Phase:     phase,
		Status:    StatusInProgress,
		StartTime: phaseStartTime,
		Metrics:   make(map[string]float64),
	}
	
	// Audit phase start
	w.auditWorkflowEvent(execContext, fmt.Sprintf("phase_%d_started", phase), execContext.Request.RequestedBy, nil)
	
	var result interface{}
	var err error
	
	// Execute phase-specific logic
	switch phase {
	case PhaseRecipeResolution:
		result, err = w.executeRecipeResolutionPhase(phaseCtx, execContext, previousResult)
	case PhaseContextAssembly:
		result, err = w.executeContextAssemblyPhase(phaseCtx, execContext, previousResult)
	case PhaseClinicalIntelligence:
		result, err = w.executeClinicalIntelligencePhase(phaseCtx, execContext, previousResult)
	case PhaseProposalGeneration:
		result, err = w.executeProposalGenerationPhase(phaseCtx, execContext, previousResult)
	default:
		err = fmt.Errorf("unknown workflow phase: %d", phase)
	}
	
	// Complete phase result
	phaseResult.EndTime = time.Now()
	phaseResult.Duration = phaseResult.EndTime.Sub(phaseResult.StartTime)
	phaseResult.Result = result
	
	if err != nil {
		phaseResult.Status = StatusFailed
		phaseResult.Errors = []string{err.Error()}
		w.logger.Error("Workflow phase failed",
			zap.String("workflow_id", execContext.Request.WorkflowID.String()),
			zap.Int("phase", int(phase)),
			zap.Error(err),
			zap.Duration("duration", phaseResult.Duration),
		)
	} else {
		phaseResult.Status = StatusCompleted
		w.logger.Debug("Workflow phase completed",
			zap.String("workflow_id", execContext.Request.WorkflowID.String()),
			zap.Int("phase", int(phase)),
			zap.Duration("duration", phaseResult.Duration),
		)
	}
	
	// Calculate phase quality score
	phaseResult.QualityScore = w.calculatePhaseQualityScore(phase, result, err)
	
	// Audit phase completion
	w.auditWorkflowEvent(execContext, fmt.Sprintf("phase_%d_completed", phase), execContext.Request.RequestedBy, phaseResult)
	
	return phaseResult, err
}

// Phase execution methods (to be continued in next part due to length)
// These methods delegate to the respective service implementations

func (w *WorkflowOrchestratorService) executeRecipeResolutionPhase(
	ctx context.Context,
	execContext *WorkflowExecutionContext,
	previousResult interface{},
) (interface{}, error) {
	// Create integrated workflow request for phases 1 & 2
	integrationRequest := &IntegratedWorkflowRequest{
		RecipeResolutionRequest: &RecipeResolutionRequest{
			RecipeID:       execContext.Request.RecipeID,
			PatientID:      execContext.Request.PatientID,
			PatientContext: execContext.Request.PatientContext,
			RequestedBy:    execContext.Request.RequestedBy,
			WorkflowID:     execContext.Request.WorkflowID,
		},
		CreateSnapshot:     true,
		SnapshotType:       "calculation",
		RequireValidation:  false,
		WorkflowID:        execContext.Request.WorkflowID,
		RequestedBy:       execContext.Request.RequestedBy,
	}
	
	return w.recipeResolverIntegration.ExecuteIntegratedWorkflow(ctx, integrationRequest)
}

func (w *WorkflowOrchestratorService) executeContextAssemblyPhase(
	ctx context.Context,
	execContext *WorkflowExecutionContext,
	previousResult interface{},
) (interface{}, error) {
	// Context assembly is already handled in Phase 1 via RecipeResolverContextIntegration
	// This phase validates and enhances the snapshot if needed
	if integrationResult, ok := previousResult.(*IntegratedWorkflowResponse); ok {
		return integrationResult.SnapshotResult, nil
	}
	return previousResult, nil
}

func (w *WorkflowOrchestratorService) executeClinicalIntelligencePhase(
	ctx context.Context,
	execContext *WorkflowExecutionContext,
	previousResult interface{},
) (interface{}, error) {
	clinicalRequest := &ClinicalIntelligenceRequest{
		WorkflowID:      execContext.Request.WorkflowID,
		PatientID:       execContext.Request.PatientID,
		SnapshotData:    previousResult,
		ClinicalParams:  execContext.Request.ClinicalParams,
		RequestedBy:     execContext.Request.RequestedBy,
	}
	
	return w.clinicalIntelligenceService.ProcessClinicalIntelligence(ctx, clinicalRequest)
}

func (w *WorkflowOrchestratorService) executeProposalGenerationPhase(
	ctx context.Context,
	execContext *WorkflowExecutionContext,
	previousResult interface{},
) (interface{}, error) {
	// Get clinical intelligence result if available
	var clinicalResult interface{}
	if phase3Result, exists := execContext.PhaseResults[PhaseClinicalIntelligence]; exists {
		clinicalResult = phase3Result.Result
	}
	
	proposalRequest := &ProposalGenerationRequest{
		WorkflowID:        execContext.Request.WorkflowID,
		PatientID:         execContext.Request.PatientID,
		SnapshotData:      previousResult,
		ClinicalData:      clinicalResult,
		ProposalParams:    execContext.Request.ProposalParams,
		RequestedBy:       execContext.Request.RequestedBy,
	}
	
	return w.proposalGenerationService.GenerateProposals(ctx, proposalRequest)
}

// Helper methods for workflow orchestration

func (w *WorkflowOrchestratorService) getWorkflowTimeout(request *WorkflowExecutionRequest) time.Duration {
	if request.Options != nil && request.Options.TimeoutPerPhase > 0 {
		return request.Options.TimeoutPerPhase * 4 // 4 phases
	}
	return w.config.DefaultTimeoutPerPhase * 4
}

func (w *WorkflowOrchestratorService) getPhaseTimeout(request *WorkflowExecutionRequest, phase WorkflowPhase) time.Duration {
	if request.Options != nil && request.Options.TimeoutPerPhase > 0 {
		return request.Options.TimeoutPerPhase
	}
	return w.config.DefaultTimeoutPerPhase
}

func (w *WorkflowOrchestratorService) shouldFailFast(request *WorkflowExecutionRequest, err error) bool {
	if request.Options != nil {
		return request.Options.FailFast
	}
	return false // Default to continue on errors
}

func (w *WorkflowOrchestratorService) initializeWorkflowState(request *WorkflowExecutionRequest) *WorkflowState {
	return &WorkflowState{
		WorkflowID:    request.WorkflowID,
		Status:        WorkflowStatusInProgress,
		CurrentPhase:  PhaseRecipeResolution,
		CreatedAt:     time.Now(),
		PhaseResults:  make(map[WorkflowPhase]*PhaseResult),
	}
}

func (w *WorkflowOrchestratorService) calculatePhaseQualityScore(phase WorkflowPhase, result interface{}, err error) float64 {
	if err != nil {
		return 0.0
	}
	
	// Phase-specific quality calculation logic would go here
	// For now, return a baseline score
	return 0.8
}

func (w *WorkflowOrchestratorService) auditWorkflowEvent(execContext *WorkflowExecutionContext, action, actor string, details interface{}) {
	entry := WorkflowAuditEntry{
		Action:    action,
		Actor:     actor,
		Timestamp: time.Now(),
		Details:   details,
	}
	execContext.AuditTrail = append(execContext.AuditTrail, entry)
	
	// Persist to audit service if available
	if w.auditService != nil {
		auditData, _ := json.Marshal(entry)
		w.auditService.LogEvent(context.Background(), &AuditEvent{
			EventType: "workflow_event",
			ActorID:   actor,
			Data:      string(auditData),
			Timestamp: entry.Timestamp,
		})
	}
}

func (w *WorkflowOrchestratorService) addWorkflowError(execContext *WorkflowExecutionContext, phase WorkflowPhase, err error) {
	workflowError := WorkflowError{
		Phase:     phase,
		Code:      "PHASE_ERROR",
		Message:   err.Error(),
		Timestamp: time.Now(),
		Retryable: w.isRetryableError(err),
	}
	execContext.Errors = append(execContext.Errors, workflowError)
}

func (w *WorkflowOrchestratorService) isRetryableError(err error) bool {
	// Logic to determine if error is retryable
	// This would check error types, network issues, etc.
	return false // Placeholder
}

func (w *WorkflowOrchestratorService) buildWorkflowResult(execContext *WorkflowExecutionContext) *WorkflowExecutionResult {
	endTime := time.Now()
	
	// Determine overall workflow status
	status := w.determineWorkflowStatus(execContext)
	
	// Extract final proposal and alternatives
	var finalProposal *MedicationProposal
	var alternatives []*MedicationProposal
	
	if proposalPhase, exists := execContext.PhaseResults[PhaseProposalGeneration]; exists && proposalPhase.Result != nil {
		if proposalResult, ok := proposalPhase.Result.(*ProposalGenerationResult); ok {
			if len(proposalResult.Proposals) > 0 {
				finalProposal = proposalResult.Proposals[0]
				if len(proposalResult.Proposals) > 1 {
					alternatives = proposalResult.Proposals[1:]
				}
			}
		}
	}
	
	return &WorkflowExecutionResult{
		WorkflowID:         execContext.Request.WorkflowID,
		RequestID:          execContext.Request.RequestID,
		Status:             status,
		Phases:             execContext.PhaseResults,
		FinalProposal:      finalProposal,
		Alternatives:       alternatives,
		QualityMetrics:     w.calculateWorkflowQualityMetrics(execContext),
		PerformanceMetrics: w.calculateWorkflowPerformanceMetrics(execContext),
		Errors:             execContext.Errors,
		Warnings:           execContext.Warnings,
		AuditTrail:         execContext.AuditTrail,
		CompletedAt:        endTime,
		TotalDuration:      endTime.Sub(execContext.StartTime),
	}
}

func (w *WorkflowOrchestratorService) determineWorkflowStatus(execContext *WorkflowExecutionContext) WorkflowStatus {
	completedPhases := 0
	failedPhases := 0
	
	for _, result := range execContext.PhaseResults {
		switch result.Status {
		case StatusCompleted:
			completedPhases++
		case StatusFailed:
			failedPhases++
		}
	}
	
	totalPhases := len(execContext.PhaseResults)
	
	if failedPhases > 0 && completedPhases == 0 {
		return WorkflowStatusFailed
	} else if completedPhases == totalPhases {
		return WorkflowStatusCompleted
	} else if completedPhases > 0 {
		return WorkflowStatusPartiallyCompleted
	}
	
	return WorkflowStatusInProgress
}

func (w *WorkflowOrchestratorService) calculateWorkflowQualityMetrics(execContext *WorkflowExecutionContext) *WorkflowQualityMetrics {
	phaseQualities := make(map[WorkflowPhase]float64)
	totalQuality := 0.0
	phaseCount := 0
	
	for phase, result := range execContext.PhaseResults {
		phaseQualities[phase] = result.QualityScore
		totalQuality += result.QualityScore
		phaseCount++
	}
	
	overallQuality := 0.0
	if phaseCount > 0 {
		overallQuality = totalQuality / float64(phaseCount)
	}
	
	return &WorkflowQualityMetrics{
		OverallQuality:   overallQuality,
		PhaseQualities:   phaseQualities,
		DataCompleteness: 0.8, // Placeholder - would calculate from actual data
		ClinicalAccuracy: 0.8, // Placeholder - would calculate from clinical validation
		FHIRCompliance:   0.9, // Placeholder - would calculate from FHIR validation
		SafetyScore:      0.9, // Placeholder - would calculate from safety checks
	}
}

func (w *WorkflowOrchestratorService) calculateWorkflowPerformanceMetrics(execContext *WorkflowExecutionContext) *WorkflowPerformanceMetrics {
	phaseLatencies := make(map[WorkflowPhase]time.Duration)
	totalLatency := time.Since(execContext.StartTime)
	
	for phase, result := range execContext.PhaseResults {
		phaseLatencies[phase] = result.Duration
	}
	
	errorRate := float64(len(execContext.Errors)) / float64(len(execContext.PhaseResults)+1)
	
	return &WorkflowPerformanceMetrics{
		TotalLatency:   totalLatency,
		PhaseLatencies: phaseLatencies,
		ThroughputRPS:  1.0 / totalLatency.Seconds(), // Simplified calculation
		ErrorRate:      errorRate,
		RetryCount:     0, // Would track actual retries
		CacheHitRate:   0.0, // Would calculate from cache statistics
	}
}

func (w *WorkflowOrchestratorService) handleWorkflowTimeout(execContext *WorkflowExecutionContext) *WorkflowExecutionResult {
	w.logger.Warn("Workflow execution timed out",
		zap.String("workflow_id", execContext.Request.WorkflowID.String()),
	)
	
	// Mark current phase as failed due to timeout
	timeoutError := WorkflowError{
		Code:      "WORKFLOW_TIMEOUT",
		Message:   "Workflow execution exceeded timeout",
		Timestamp: time.Now(),
		Retryable: true,
	}
	execContext.Errors = append(execContext.Errors, timeoutError)
	
	return w.buildWorkflowResult(execContext)
}

func (w *WorkflowOrchestratorService) handlePhaseFailure(execContext *WorkflowExecutionContext, phase WorkflowPhase, err error) *WorkflowExecutionResult {
	w.logger.Error("Phase failure caused workflow termination",
		zap.String("workflow_id", execContext.Request.WorkflowID.String()),
		zap.Int("failed_phase", int(phase)),
		zap.Error(err),
	)
	
	return w.buildWorkflowResult(execContext)
}

func (w *WorkflowOrchestratorService) completeWorkflowExecution(execContext *WorkflowExecutionContext, result *WorkflowExecutionResult, err error) *WorkflowExecutionResult {
	if result == nil {
		result = w.buildWorkflowResult(execContext)
	}
	
	if err != nil && result.Status == WorkflowStatusInProgress {
		result.Status = WorkflowStatusFailed
	}
	
	return result
}

func (w *WorkflowOrchestratorService) updateWorkflowMetrics(result *WorkflowExecutionResult) {
	if w.metricsService != nil {
		// Update workflow execution metrics
		w.metricsService.RecordWorkflowExecution(
			result.Status,
			result.TotalDuration,
			result.QualityMetrics.OverallQuality,
		)
		
		// Update phase-specific metrics
		for phase, phaseResult := range result.Phases {
			w.metricsService.RecordPhaseExecution(
				phase,
				phaseResult.Status,
				phaseResult.Duration,
				phaseResult.QualityScore,
			)
		}
	}
}

// GetWorkflowStatus returns the current status of a workflow
func (w *WorkflowOrchestratorService) GetWorkflowStatus(ctx context.Context, workflowID uuid.UUID) (*WorkflowState, error) {
	// Check active workflows first
	if execContext, exists := w.activeWorkflows[workflowID]; exists {
		return execContext.State, nil
	}
	
	// Check persisted state
	if w.config.EnableStatePersistence {
		return w.workflowStateService.GetState(ctx, workflowID)
	}
	
	return nil, fmt.Errorf("workflow not found: %s", workflowID.String())
}

// CancelWorkflow cancels an active workflow
func (w *WorkflowOrchestratorService) CancelWorkflow(ctx context.Context, workflowID uuid.UUID, reason string) error {
	if execContext, exists := w.activeWorkflows[workflowID]; exists {
		w.logger.Info("Cancelling workflow",
			zap.String("workflow_id", workflowID.String()),
			zap.String("reason", reason),
		)
		
		execContext.CancelFunc()
		w.auditWorkflowEvent(execContext, "workflow_cancelled", "system", map[string]string{"reason": reason})
		
		return nil
	}
	
	return fmt.Errorf("active workflow not found: %s", workflowID.String())
}

// ListActiveWorkflows returns all currently active workflows
func (w *WorkflowOrchestratorService) ListActiveWorkflows(ctx context.Context) []uuid.UUID {
	workflowIDs := make([]uuid.UUID, 0, len(w.activeWorkflows))
	for id := range w.activeWorkflows {
		workflowIDs = append(workflowIDs, id)
	}
	return workflowIDs
}