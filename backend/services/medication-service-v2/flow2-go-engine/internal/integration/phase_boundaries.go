package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/models"
)

// PhaseBoundaryManager manages the integration between Phase 1 (Go) and Phase 2+ (Rust)
// It defines clear API contracts and handles the transformation between phase data models
type PhaseBoundaryManager struct {
	rustEngineClient RustEngineClient
	logger          *logrus.Logger
	
	// Performance tracking
	phase1Metrics   *Phase1Metrics
	integrationMetrics *IntegrationMetrics
}

// Phase1ToPhase2Contract defines the contract between Phase 1 and Phase 2+
type Phase1ToPhase2Contract struct {
	// Phase 1 Output (Intent Manifest)
	IntentManifest *models.IntentManifest `json:"intent_manifest"`
	
	// Phase 1 Context
	OriginalRequest *models.MedicationRequest `json:"original_request"`
	
	// Phase 2+ Input Requirements
	ExecutionContext *ExecutionContext `json:"execution_context"`
	
	// Performance Requirements
	RequiredSLA     *SLARequirements `json:"required_sla"`
}

// ExecutionContext provides the context needed for Phase 2+ execution
type ExecutionContext struct {
	// Clinical execution parameters
	PatientSnapshot  *PatientSnapshot  `json:"patient_snapshot"`
	ClinicalProtocol *ClinicalProtocol `json:"clinical_protocol"`
	
	// Execution constraints
	TimeoutMs       int               `json:"timeout_ms"`
	Priority        string            `json:"priority"`
	SafetyMode      string            `json:"safety_mode"`
	
	// Tracing and audit
	TraceID         string            `json:"trace_id"`
	SessionID       string            `json:"session_id"`
	AuditTrail      []AuditEvent      `json:"audit_trail"`
}

// PatientSnapshot represents the patient data snapshot for Phase 2+
type PatientSnapshot struct {
	PatientID       string                 `json:"patient_id"`
	SnapshotID      string                 `json:"snapshot_id"`
	CreatedAt       time.Time              `json:"created_at"`
	ValidUntil      time.Time              `json:"valid_until"`
	
	// Assembled clinical data
	Demographics    *Demographics          `json:"demographics"`
	ClinicalData    map[string]interface{} `json:"clinical_data"`
	CurrentMeds     []CurrentMedication    `json:"current_medications"`
	LabResults      map[string]LabResult   `json:"lab_results"`
	VitalSigns      map[string]VitalSign   `json:"vital_signs"`
	
	// Data quality metrics
	Completeness    float64                `json:"completeness"`
	FreshnessScore  float64                `json:"freshness_score"`
	Sources         []string               `json:"sources"`
}

// ClinicalProtocol represents the resolved clinical protocol for execution
type ClinicalProtocol struct {
	ProtocolID      string                 `json:"protocol_id"`
	Version         string                 `json:"version"`
	
	// Execution instructions
	PrimaryTherapy  *TherapyInstruction    `json:"primary_therapy"`
	AlternativeTherapies []TherapyInstruction `json:"alternative_therapies"`
	
	// Safety and monitoring
	SafetyChecks    []SafetyCheck          `json:"safety_checks"`
	MonitoringPlan  *MonitoringPlan        `json:"monitoring_plan"`
	
	// Decision support
	ClinicalNotes   []ClinicalNote         `json:"clinical_notes"`
	EvidenceRefs    []EvidenceReference    `json:"evidence_references"`
}

// SLARequirements defines performance requirements for each phase
type SLARequirements struct {
	Phase1MaxMs     int  `json:"phase1_max_ms"`     // 25ms for ORB + Recipe Resolution
	Phase2MaxMs     int  `json:"phase2_max_ms"`     // 100ms for clinical execution
	TotalMaxMs      int  `json:"total_max_ms"`      // 150ms total end-to-end
	FailFast        bool `json:"fail_fast"`         // Fail fast on SLA violations
}

// Phase2ExecutionRequest represents a request to the Rust engine
type Phase2ExecutionRequest struct {
	// Execution context
	RequestID       string            `json:"request_id"`
	SessionID       string            `json:"session_id"`
	TraceID         string            `json:"trace_id"`
	
	// Patient context
	PatientSnapshot *PatientSnapshot  `json:"patient_snapshot"`
	
	// Clinical protocol
	Protocol        *ClinicalProtocol `json:"protocol"`
	
	// Execution parameters
	ExecutionMode   string            `json:"execution_mode"`    // STANDARD, OPTIMIZATION, SAFETY_FIRST
	TimeoutMs       int               `json:"timeout_ms"`
	Priority        string            `json:"priority"`
	
	// Phase 1 provenance
	IntentManifest  *models.IntentManifest `json:"intent_manifest"`
}

// Phase2ExecutionResponse represents a response from the Rust engine
type Phase2ExecutionResponse struct {
	// Response metadata
	RequestID       string            `json:"request_id"`
	ExecutionID     string            `json:"execution_id"`
	ProcessedAt     time.Time         `json:"processed_at"`
	
	// Clinical results
	Recommendation  *ClinicalRecommendation `json:"recommendation"`
	SafetyResult    *SafetyAssessment       `json:"safety_result"`
	
	// Execution metadata
	ExecutionTime   *ExecutionTiming        `json:"execution_time"`
	ResourceUsage   *ResourceUsage          `json:"resource_usage"`
	
	// Quality assurance
	ConfidenceScore float64                 `json:"confidence_score"`
	QualityMetrics  *QualityMetrics         `json:"quality_metrics"`
	
	// Error handling
	Errors          []ExecutionError        `json:"errors,omitempty"`
	Warnings        []ExecutionWarning      `json:"warnings,omitempty"`
}

// Supporting Types

// Demographics represents patient demographic data
type Demographics struct {
	Age             float64  `json:"age"`
	Sex             string   `json:"sex"`
	Weight          float64  `json:"weight"`
	Height          float64  `json:"height"`
	BMI             float64  `json:"bmi"`
	Ethnicity       string   `json:"ethnicity,omitempty"`
}

// CurrentMedication represents current medications
type CurrentMedication struct {
	MedicationCode  string    `json:"medication_code"`
	MedicationName  string    `json:"medication_name"`
	Dose            string    `json:"dose"`
	Frequency       string    `json:"frequency"`
	Route           string    `json:"route"`
	StartDate       time.Time `json:"start_date"`
	Indication      string    `json:"indication,omitempty"`
	Prescriber      string    `json:"prescriber,omitempty"`
}

// LabResult represents laboratory results
type LabResult struct {
	TestCode        string    `json:"test_code"`
	TestName        string    `json:"test_name"`
	Value           float64   `json:"value"`
	Unit            string    `json:"unit"`
	ReferenceRange  string    `json:"reference_range"`
	Status          string    `json:"status"` // NORMAL, HIGH, LOW, CRITICAL
	CollectedAt     time.Time `json:"collected_at"`
	ReportedAt      time.Time `json:"reported_at"`
}

// VitalSign represents vital sign measurements
type VitalSign struct {
	Parameter       string    `json:"parameter"`
	Value           float64   `json:"value"`
	Unit            string    `json:"unit"`
	MeasuredAt      time.Time `json:"measured_at"`
	Method          string    `json:"method,omitempty"`
	Location        string    `json:"location,omitempty"`
}

// TherapyInstruction represents therapy instructions
type TherapyInstruction struct {
	MedicationCode  string           `json:"medication_code"`
	MedicationName  string           `json:"medication_name"`
	DoseCalculation *DoseCalculation `json:"dose_calculation"`
	Administration  *Administration  `json:"administration"`
	Duration        *Duration        `json:"duration"`
	Rationale       string           `json:"rationale"`
}

// DoseCalculation represents calculated dose information
type DoseCalculation struct {
	CalculatedDose  float64  `json:"calculated_dose"`
	Unit            string   `json:"unit"`
	Method          string   `json:"method"`          // STANDARD, WEIGHT_BASED, BSA_BASED, etc.
	AdjustmentFactors []string `json:"adjustment_factors"`
	ConfidenceLevel float64  `json:"confidence_level"`
}

// Administration represents administration instructions
type Administration struct {
	Route           string   `json:"route"`
	Frequency       string   `json:"frequency"`
	Timing          string   `json:"timing,omitempty"`
	SpecialInstructions []string `json:"special_instructions,omitempty"`
}

// Duration represents therapy duration
type Duration struct {
	Value           int      `json:"value"`
	Unit            string   `json:"unit"`     // DAYS, WEEKS, MONTHS, etc.
	Condition       string   `json:"condition,omitempty"` // "until symptom resolution", etc.
}

// SafetyCheck represents safety verification requirements
type SafetyCheck struct {
	CheckType       string                 `json:"check_type"`
	Parameters      map[string]interface{} `json:"parameters"`
	Severity        string                 `json:"severity"`
	Required        bool                   `json:"required"`
}

// MonitoringPlan represents monitoring requirements
type MonitoringPlan struct {
	MonitoringPoints []MonitoringPoint      `json:"monitoring_points"`
	Duration        string                 `json:"duration"`
	EscalationPlan  string                 `json:"escalation_plan,omitempty"`
}

// MonitoringPoint represents a specific monitoring requirement
type MonitoringPoint struct {
	Parameter       string    `json:"parameter"`
	Frequency       string    `json:"frequency"`
	Duration        string    `json:"duration"`
	ThresholdAlert  *Alert    `json:"threshold_alert,omitempty"`
}

// Alert represents monitoring alerts
type Alert struct {
	Threshold       float64  `json:"threshold"`
	Direction       string   `json:"direction"` // ABOVE, BELOW
	Severity        string   `json:"severity"`
	Action          string   `json:"action"`
}

// ClinicalNote represents clinical decision support notes
type ClinicalNote struct {
	Type            string    `json:"type"`     // RATIONALE, WARNING, INFORMATION
	Message         string    `json:"message"`
	Severity        string    `json:"severity"`
	References      []string  `json:"references,omitempty"`
}

// EvidenceReference represents clinical evidence references
type EvidenceReference struct {
	Type            string    `json:"type"`     // GUIDELINE, STUDY, EXPERT_OPINION
	Source          string    `json:"source"`
	Title           string    `json:"title"`
	URL             string    `json:"url,omitempty"`
	EvidenceLevel   string    `json:"evidence_level"`
}

// AuditEvent represents audit trail events
type AuditEvent struct {
	EventID         string                 `json:"event_id"`
	Timestamp       time.Time              `json:"timestamp"`
	Phase           string                 `json:"phase"`
	EventType       string                 `json:"event_type"`
	Description     string                 `json:"description"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Phase 2+ Response Types

// ClinicalRecommendation represents the clinical recommendation
type ClinicalRecommendation struct {
	Action          string               `json:"action"`  // PRESCRIBE, ADJUST, HOLD, CONTRAINDICATED
	PrimaryTherapy  *TherapyRecommendation `json:"primary_therapy"`
	Alternatives    []TherapyRecommendation `json:"alternatives,omitempty"`
	ClinicalRationale string             `json:"clinical_rationale"`
	ConfidenceScore float64              `json:"confidence_score"`
}

// TherapyRecommendation represents a specific therapy recommendation
type TherapyRecommendation struct {
	MedicationCode  string           `json:"medication_code"`
	MedicationName  string           `json:"medication_name"`
	FinalDose       *FinalDose       `json:"final_dose"`
	Administration  *Administration  `json:"administration"`
	Duration        *Duration        `json:"duration"`
	MonitoringReq   []string         `json:"monitoring_required"`
	Rationale       string           `json:"rationale"`
}

// FinalDose represents the final calculated dose
type FinalDose struct {
	Amount          float64  `json:"amount"`
	Unit            string   `json:"unit"`
	Strength        string   `json:"strength"`
	FormFactor      string   `json:"form_factor"`  // TABLET, CAPSULE, INJECTION, etc.
}

// SafetyAssessment represents safety verification results
type SafetyAssessment struct {
	OverallStatus   string             `json:"overall_status"` // SAFE, CAUTION, CONTRAINDICATED
	SafetyFindings  []SafetyFinding    `json:"safety_findings"`
	RiskScore       float64            `json:"risk_score"`
	MitigationPlan  *MitigationPlan    `json:"mitigation_plan,omitempty"`
}

// SafetyFinding represents individual safety findings
type SafetyFinding struct {
	Type            string   `json:"type"`     // INTERACTION, ALLERGY, CONTRAINDICATION
	Severity        string   `json:"severity"`
	Description     string   `json:"description"`
	Recommendation  string   `json:"recommendation"`
	EvidenceLevel   string   `json:"evidence_level"`
}

// MitigationPlan represents risk mitigation strategies
type MitigationPlan struct {
	Strategies      []MitigationStrategy `json:"strategies"`
	MonitoringReq   []string            `json:"monitoring_required"`
	EscalationPlan  string              `json:"escalation_plan"`
}

// MitigationStrategy represents individual mitigation strategies
type MitigationStrategy struct {
	Type            string   `json:"type"`
	Description     string   `json:"description"`
	Implementation  string   `json:"implementation"`
	Effectiveness   string   `json:"effectiveness"`
}

// Performance and Quality Types

// ExecutionTiming represents execution performance metrics
type ExecutionTiming struct {
	TotalMs         int64    `json:"total_ms"`
	DoseCalcMs      int64    `json:"dose_calculation_ms"`
	SafetyCheckMs   int64    `json:"safety_check_ms"`
	ValidationMs    int64    `json:"validation_ms"`
	NetworkMs       int64    `json:"network_ms"`
}

// ResourceUsage represents resource utilization
type ResourceUsage struct {
	CPUUsagePercent float64  `json:"cpu_usage_percent"`
	MemoryUsageMB   float64  `json:"memory_usage_mb"`
	NetworkCallCount int     `json:"network_call_count"`
	CacheHitRate    float64  `json:"cache_hit_rate"`
}

// QualityMetrics represents quality assurance metrics
type QualityMetrics struct {
	DataCompleteness float64            `json:"data_completeness"`
	RulesCovered     int                `json:"rules_covered"`
	ValidationsRun   int                `json:"validations_run"`
	QualityScore     float64            `json:"quality_score"`
	QualityFlags     []string           `json:"quality_flags,omitempty"`
}

// ExecutionError represents execution errors
type ExecutionError struct {
	ErrorCode       string             `json:"error_code"`
	ErrorType       string             `json:"error_type"`
	Message         string             `json:"message"`
	Context         map[string]interface{} `json:"context,omitempty"`
	Recoverable     bool               `json:"recoverable"`
}

// ExecutionWarning represents execution warnings
type ExecutionWarning struct {
	WarningCode     string             `json:"warning_code"`
	WarningType     string             `json:"warning_type"`
	Message         string             `json:"message"`
	Severity        string             `json:"severity"`
	Context         map[string]interface{} `json:"context,omitempty"`
}

// RustEngineClient interface for communicating with the Rust engine
type RustEngineClient interface {
	ExecutePhase2(ctx context.Context, request *Phase2ExecutionRequest) (*Phase2ExecutionResponse, error)
	HealthCheck(ctx context.Context) error
	GetCapabilities(ctx context.Context) (*EngineCapabilities, error)
}

// EngineCapabilities represents Rust engine capabilities
type EngineCapabilities struct {
	SupportedProtocols []string          `json:"supported_protocols"`
	MaxConcurrentReqs  int               `json:"max_concurrent_requests"`
	AverageBenchmarkMs float64           `json:"average_benchmark_ms"`
	Features           []string          `json:"features"`
}

// Metrics tracking types

// Phase1Metrics tracks Phase 1 performance
type Phase1Metrics struct {
	TotalRequests       int64   `json:"total_requests"`
	AverageLatencyMs    float64 `json:"average_latency_ms"`
	SLAViolations      int64   `json:"sla_violations"`
	SLAComplianceRate  float64 `json:"sla_compliance_rate"`
}

// IntegrationMetrics tracks cross-phase integration performance
type IntegrationMetrics struct {
	TotalIntegrations    int64   `json:"total_integrations"`
	SuccessfulHandoffs   int64   `json:"successful_handoffs"`
	FailedHandoffs       int64   `json:"failed_handoffs"`
	AverageHandoffMs     float64 `json:"average_handoff_ms"`
	DataTransformErrors  int64   `json:"data_transform_errors"`
}

// NewPhaseBoundaryManager creates a new phase boundary manager
func NewPhaseBoundaryManager(rustClient RustEngineClient, logger *logrus.Logger) *PhaseBoundaryManager {
	return &PhaseBoundaryManager{
		rustEngineClient:   rustClient,
		logger:            logger,
		phase1Metrics:     &Phase1Metrics{},
		integrationMetrics: &IntegrationMetrics{},
	}
}

// TransformPhase1ToPhase2 transforms Phase 1 output to Phase 2+ input
func (pbm *PhaseBoundaryManager) TransformPhase1ToPhase2(
	ctx context.Context,
	intentManifest *models.IntentManifest,
	originalRequest *models.MedicationRequest,
) (*Phase2ExecutionRequest, error) {
	
	startTime := time.Now()
	
	// Create patient snapshot from original request
	patientSnapshot, err := pbm.createPatientSnapshot(originalRequest, intentManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to create patient snapshot: %w", err)
	}
	
	// Create clinical protocol from intent manifest
	clinicalProtocol, err := pbm.createClinicalProtocol(intentManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to create clinical protocol: %w", err)
	}
	
	// Build Phase 2 execution request
	phase2Request := &Phase2ExecutionRequest{
		RequestID:       originalRequest.RequestID,
		SessionID:       fmt.Sprintf("session_%s", originalRequest.RequestID),
		TraceID:         fmt.Sprintf("trace_%s", intentManifest.ManifestID),
		PatientSnapshot: patientSnapshot,
		Protocol:        clinicalProtocol,
		ExecutionMode:   "STANDARD",
		TimeoutMs:       100, // 100ms for Phase 2
		Priority:        string(originalRequest.Urgency),
		IntentManifest:  intentManifest,
	}
	
	// Track integration metrics
	pbm.integrationMetrics.TotalIntegrations++
	transformTime := time.Since(startTime)
	pbm.integrationMetrics.AverageHandoffMs = 
		(pbm.integrationMetrics.AverageHandoffMs * float64(pbm.integrationMetrics.TotalIntegrations-1) +
		 float64(transformTime.Milliseconds())) / float64(pbm.integrationMetrics.TotalIntegrations)
	
	pbm.logger.WithFields(logrus.Fields{
		"request_id":     originalRequest.RequestID,
		"manifest_id":    intentManifest.ManifestID,
		"transform_ms":   transformTime.Milliseconds(),
	}).Debug("Phase 1 to Phase 2 transformation completed")
	
	return phase2Request, nil
}

// ExecuteFullWorkflow executes the complete Phase 1 → Phase 2+ workflow
func (pbm *PhaseBoundaryManager) ExecuteFullWorkflow(
	ctx context.Context,
	intentManifest *models.IntentManifest,
	originalRequest *models.MedicationRequest,
) (*Phase2ExecutionResponse, error) {
	
	// Transform Phase 1 to Phase 2
	phase2Request, err := pbm.TransformPhase1ToPhase2(ctx, intentManifest, originalRequest)
	if err != nil {
		pbm.integrationMetrics.FailedHandoffs++
		return nil, fmt.Errorf("phase transformation failed: %w", err)
	}
	
	// Execute Phase 2
	phase2Response, err := pbm.rustEngineClient.ExecutePhase2(ctx, phase2Request)
	if err != nil {
		pbm.integrationMetrics.FailedHandoffs++
		return nil, fmt.Errorf("phase 2 execution failed: %w", err)
	}
	
	pbm.integrationMetrics.SuccessfulHandoffs++
	
	return phase2Response, nil
}

// Helper methods

func (pbm *PhaseBoundaryManager) createPatientSnapshot(
	request *models.MedicationRequest,
	manifest *models.IntentManifest,
) (*PatientSnapshot, error) {
	
	snapshot := &PatientSnapshot{
		PatientID:  request.PatientID,
		SnapshotID: fmt.Sprintf("snapshot_%s", manifest.ManifestID),
		CreatedAt:  time.Now(),
		ValidUntil: time.Now().Add(time.Duration(manifest.SnapshotTTL) * time.Second),
		Demographics: &Demographics{
			Age:    request.ClinicalContext.Age,
			Sex:    request.ClinicalContext.Sex,
			Weight: request.ClinicalContext.Weight,
		},
		ClinicalData:   make(map[string]interface{}),
		CurrentMeds:    pbm.transformCurrentMedications(request.ClinicalContext.CurrentMeds),
		LabResults:     pbm.transformLabResults(request.ClinicalContext.RecentLabs),
		VitalSigns:     pbm.transformVitalSigns(request.ClinicalContext.VitalSigns),
		Completeness:   0.85, // Calculated based on available data
		FreshnessScore: 0.90, // Calculated based on data timestamps
		Sources:        []string{"EHR", "LAB", "DEVICE"},
	}
	
	return snapshot, nil
}

func (pbm *PhaseBoundaryManager) createClinicalProtocol(manifest *models.IntentManifest) (*ClinicalProtocol, error) {
	protocol := &ClinicalProtocol{
		ProtocolID: manifest.ProtocolID,
		Version:    manifest.ProtocolVersion,
		PrimaryTherapy: &TherapyInstruction{
			// This would be populated based on the therapy options in the manifest
		},
		SafetyChecks: []SafetyCheck{
			{
				CheckType: "DRUG_INTERACTIONS",
				Severity:  "HIGH",
				Required:  true,
			},
		},
		MonitoringPlan: &MonitoringPlan{
			MonitoringPoints: []MonitoringPoint{},
			Duration:        "30_DAYS",
		},
	}
	
	return protocol, nil
}

// Transform methods for data conversion

func (pbm *PhaseBoundaryManager) transformCurrentMedications(meds []models.CurrentMedication) []CurrentMedication {
	result := make([]CurrentMedication, len(meds))
	for i, med := range meds {
		result[i] = CurrentMedication{
			MedicationCode: med.MedicationCode,
			MedicationName: med.MedicationName,
			Dose:          med.Dose,
			Frequency:     med.Frequency,
			Route:         med.Route,
			StartDate:     med.StartDate,
			Indication:    med.Indication,
		}
	}
	return result
}

func (pbm *PhaseBoundaryManager) transformLabResults(labs map[string]models.LabValue) map[string]LabResult {
	result := make(map[string]LabResult)
	for key, lab := range labs {
		result[key] = LabResult{
			TestCode:       key,
			TestName:       key,
			Value:         lab.Value,
			Unit:          lab.Unit,
			ReferenceRange: lab.ReferenceRange,
			Status:        lab.Status,
			CollectedAt:   lab.Timestamp,
			ReportedAt:    lab.Timestamp,
		}
	}
	return result
}

func (pbm *PhaseBoundaryManager) transformVitalSigns(vitals map[string]models.VitalSign) map[string]VitalSign {
	result := make(map[string]VitalSign)
	for key, vital := range vitals {
		result[key] = VitalSign{
			Parameter:  key,
			Value:     vital.Value,
			Unit:      vital.Unit,
			MeasuredAt: vital.Timestamp,
			Method:    vital.Method,
		}
	}
	return result
}

// GetMetrics returns integration metrics
func (pbm *PhaseBoundaryManager) GetMetrics() (*IntegrationMetrics, *Phase1Metrics) {
	return pbm.integrationMetrics, pbm.phase1Metrics
}