package models

import (
	"time"
)

// Phase 2 Data Structures - Context Assembly with Parallel Fetch & In-Memory Interpretation

// =============================================================================
// Evidence Envelope Management
// =============================================================================

// EvidenceEnvelope represents a consistent set of knowledge base versions
// ensuring clinical safety through version consistency across all KB queries
type EvidenceEnvelope struct {
	VersionSetName   string                    `json:"version_set_name"`
	KBVersions       map[string]string         `json:"kb_versions"`       // KB name -> version mapping
	SnapshotID       string                    `json:"snapshot_id"`
	Environment      string                    `json:"environment"`
	ActivatedAt      time.Time                 `json:"activated_at"`
	
	// Runtime tracking
	UsedVersions     map[string]VersionUsage   `json:"used_versions"`
}

// VersionUsage tracks how KB versions are used during Phase 2 execution
type VersionUsage struct {
	Version         string     `json:"version"`
	AccessedAt      time.Time  `json:"accessed_at"`
	QueryCount      int        `json:"query_count"`
	CacheHits       int        `json:"cache_hits"`
}

// ActiveVersionSet represents the current active knowledge base version set
type ActiveVersionSet struct {
	Name            string                    `json:"name"`
	Environment     string                    `json:"environment"`
	KBVersions      map[string]string         `json:"kb_versions"`
	ActivatedAt     time.Time                 `json:"activated_at"`
	ExpiresAt       time.Time                 `json:"expires_at,omitempty"`
	Description     string                    `json:"description,omitempty"`
}

// =============================================================================
// Phase 2 Context Assembler
// =============================================================================

// Phase2ContextAssembler orchestrates the assembly of complete clinical context
// through parallel data acquisition and in-memory interpretation
type Phase2ContextAssembler struct {
	// Core clients
	ContextGateway    ContextGatewayClient      `json:"-"`
	ApolloClient      ApolloFederationClient    `json:"-"`
	
	// Evidence Envelope management
	EvidenceEnvelope  *EvidenceEnvelope         `json:"evidence_envelope"`
	KBVersionManager  *KBVersionManager         `json:"-"`
	
	// In-memory interpretation engines
	PhenotypeEngine   *PhenotypeEvaluator       `json:"-"`
	RuleInterpreter   *ClinicalRuleInterpreter  `json:"-"`
	
	// Performance optimization
	ParallelExecutor  *ParallelQueryExecutor    `json:"-"`
	Cache            *ContextCache             `json:"-"`
	
	// Monitoring
	Metrics          *Phase2Metrics            `json:"-"`
	Tracer           *Tracer                   `json:"-"`
}

// =============================================================================
// Parallel Query Execution
// =============================================================================

// QueryPayloadSet contains all prepared query payloads for parallel execution
type QueryPayloadSet struct {
	RequestID    string           `json:"request_id"`
	PreparedAt   time.Time        `json:"prepared_at"`
	Queries      []QueryPayload   `json:"queries"`
}

// QueryPayload represents a single query to be executed in parallel
type QueryPayload struct {
	Target          string                    `json:"target"`          // "context_gateway" or "kb_x"
	Operation       string                    `json:"operation"`
	Priority        int                       `json:"priority"`        // For execution ordering
	
	// For Context Gateway
	ContextRequest  *ContextGatewayRequest    `json:"context_request,omitempty"`
	
	// For KB queries via Apollo
	ApolloQuery     *ApolloQueryRequest       `json:"apollo_query,omitempty"`
	
	// Version control
	RequiredVersion string                    `json:"required_version"`
	
	// Performance hints
	CacheStrategy   CacheStrategy             `json:"cache_strategy"`
	Timeout         time.Duration             `json:"timeout"`
}

// ContextGatewayRequest represents a request to the Context Gateway for snapshot creation
type ContextGatewayRequest struct {
	PatientID       string                    `json:"patient_id"`
	RecipeID        string                    `json:"recipe_id"`
	RequiredFields  []FieldRequirement        `json:"required_fields"`
	FreshnessReq    FreshnessRequirements     `json:"freshness_requirements"`
	
	// Snapshot control
	CreateSnapshot  bool                      `json:"create_snapshot"`
	SnapshotTTL     int                       `json:"snapshot_ttl_seconds"`
}

// ApolloQueryRequest represents a GraphQL query to be executed via Apollo Federation
type ApolloQueryRequest struct {
	Query           string                    `json:"query"`
	Variables       map[string]interface{}    `json:"variables"`
	KBTarget        string                    `json:"kb_target"`
	
	// Version header to be added
	VersionHeader   string                    `json:"version_header"`
}

// CombinedResults holds the results from parallel query execution
type CombinedResults struct {
	RequestID    string                        `json:"request_id"`
	StartTime    time.Time                     `json:"start_time"`
	Duration     time.Duration                 `json:"duration"`
	Results      map[string]interface{}        `json:"results"`
	Errors       map[string]error              `json:"errors"`
}

// =============================================================================
// Phenotype Evaluation
// =============================================================================

// PhenotypeEvaluationRequest represents a request for phenotype evaluation
type PhenotypeEvaluationRequest struct {
	RequestID          string                    `json:"request_id"`
	ClinicalData       *ClinicalSnapshot         `json:"clinical_data"`
	RuleSet            *PhenotypeRuleSet         `json:"rule_set"`
	EvaluationOptions  *EvaluationOptions        `json:"evaluation_options,omitempty"`
}

// PhenotypeRuleSet contains the rules for phenotype evaluation
type PhenotypeRuleSet struct {
	RuleSetID      string              `json:"rule_set_id"`
	Version        string              `json:"version"`
	Condition      string              `json:"condition"`
	Rules          []*PhenotypeRule    `json:"rules"`
	LoadedAt       time.Time           `json:"loaded_at"`
}

// PhenotypeRule represents a single phenotype evaluation rule
type PhenotypeRule struct {
	RuleID              string                  `json:"rule_id"`
	Name                string                  `json:"name"`
	Condition           string                  `json:"condition"`
	Phenotype           string                  `json:"phenotype"`
	EvaluationLogic     string                  `json:"evaluation_logic"`
	RiskStratification  *PhenotypeRiskStratification     `json:"risk_stratification"`
}

// PhenotypeRiskStratification defines risk levels for a phenotype (renamed to avoid conflict)
type PhenotypeRiskStratification struct {
	LowRisk      *RiskCriteria    `json:"low_risk"`
	ModerateRisk *RiskCriteria    `json:"moderate_risk"`
	HighRisk     *RiskCriteria    `json:"high_risk"`
}

// RiskCriteria defines criteria for a specific risk level
type RiskCriteria struct {
	Threshold   float64  `json:"threshold"`
	Conditions  []string `json:"conditions"`
	Description string   `json:"description"`
}

// PhenotypeResult contains the result of phenotype evaluation
type PhenotypeResult struct {
	PrimaryPhenotype  string              `json:"primary_phenotype"`
	Confidence        float64             `json:"confidence"`
	RiskLevel         string              `json:"risk_level"`
	AllMatches        []PhenotypeMatch    `json:"all_matches"`
	EvaluationTime    time.Duration       `json:"evaluation_time_ms"`
	Evidence          PhenotypeEvidence   `json:"evidence"`
}

// PhenotypeMatch represents a phenotype rule match
type PhenotypeMatch struct {
	RuleID     string      `json:"rule_id"`
	Phenotype  string      `json:"phenotype"`
	Confidence float64     `json:"confidence"`
	Evidence   []string    `json:"evidence"`
}

// PhenotypeEvidence contains evidence used in phenotype evaluation
type PhenotypeEvidence struct {
	RulesEvaluated   int       `json:"rules_evaluated"`
	DataPointsUsed   int       `json:"data_points_used"`
	RuleVersion      string    `json:"rule_version"`
}

// EvaluationOptions contains options for phenotype evaluation
type EvaluationOptions struct {
	IncludeAllMatches  bool     `json:"include_all_matches"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	MaxMatches         int      `json:"max_matches"`
}

// =============================================================================
// Enriched Context Output
// =============================================================================

// EnrichedContext represents the final output of Phase 2 context assembly
type EnrichedContext struct {
	// Identity
	RequestID          string                 `json:"request_id"`
	PatientID          string                 `json:"patient_id"`
	SnapshotID         string                 `json:"snapshot_id"`
	SnapshotCreatedAt  time.Time              `json:"snapshot_created_at"`
	
	// Patient clinical data (using existing types from other models)
	Demographics       Phase2Demographics     `json:"demographics"`
	CurrentMedications []CurrentMedication    `json:"current_medications"`
	Allergies         []Allergy              `json:"allergies"`
	Conditions        []Condition            `json:"conditions"`
	LabResults        []Phase2LabResult      `json:"lab_results"`
	Vitals            []VitalSign            `json:"vitals"`
	
	// Computed phenotype (from in-memory evaluation)
	Phenotype         string                 `json:"phenotype"`
	RiskLevel         string                 `json:"risk_level"`
	PhenotypeEvidence PhenotypeEvidence      `json:"phenotype_evidence"`
	
	// Additional KB data
	Guidelines        []ClinicalGuideline    `json:"guidelines"`
	FormularyStatus   FormularyInfo          `json:"formulary_status"`
	ResistanceProfile ResistanceData         `json:"resistance_profile"`
	
	// Evidence & Audit
	EvidenceEnvelope  *EvidenceEnvelope      `json:"evidence_envelope"`
	
	// Performance tracking
	Phase2Duration    time.Duration          `json:"phase2_duration_ms"`
	
	// Quality flags
	Warnings          []Phase2Warning        `json:"warnings,omitempty"`
	SafetyFlags       []Phase2SafetyFlag     `json:"safety_flags,omitempty"`
}

// =============================================================================
// Supporting Data Structures (using existing structures where possible)
// =============================================================================

// Phase2Demographics represents patient demographic information for Phase 2
type Phase2Demographics struct {
	Age    int    `json:"age"`
	Sex    string `json:"sex"`
	Weight int    `json:"weight"`
	Height int    `json:"height"`
	BMI    float64 `json:"bmi"`
}

// Phase2LabResult represents a laboratory result for Phase 2
type Phase2LabResult struct {
	TestCode      string    `json:"test_code"`
	TestName      string    `json:"test_name"`
	Value         float64   `json:"value"`
	Unit          string    `json:"unit"`
	ReferenceRange string  `json:"reference_range"`
	Date          time.Time `json:"date"`
}

// ClinicalGuideline represents a clinical guideline
type ClinicalGuideline struct {
	GuidelineID   string `json:"guideline_id"`
	Title         string `json:"title"`
	Recommendation string `json:"recommendation"`
	EvidenceLevel string `json:"evidence_level"`
}

// FormularyInfo represents formulary status information
type FormularyInfo struct {
	Covered      bool   `json:"covered"`
	Tier         string `json:"tier"`
	Restrictions []string `json:"restrictions"`
}

// ResistanceData represents resistance profile data
type ResistanceData struct {
	ResistanceMarkers []string `json:"resistance_markers"`
	SensitivityProfile map[string]string `json:"sensitivity_profile"`
}

// Phase2Warning represents a non-critical warning for Phase 2
type Phase2Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Phase2SafetyFlag represents a clinical safety flag for Phase 2
type Phase2SafetyFlag struct {
	FlagType   string    `json:"flag_type"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	DataPoint  string    `json:"data_point"`
	Timestamp  time.Time `json:"timestamp"`
}

// =============================================================================
// Enums and Constants
// =============================================================================

// CacheStrategy defines different caching strategies for queries
type CacheStrategy string

const (
	CacheStrategyNone      CacheStrategy = "none"
	CacheStrategyTTL       CacheStrategy = "ttl"
	CacheStrategyVersioned CacheStrategy = "versioned"
	CacheStrategyAggressive CacheStrategy = "aggressive"
)

// Note: FreshnessRequirements is already defined in phase1_models.go