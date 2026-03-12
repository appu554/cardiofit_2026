// Package clients provides HTTP clients for KB services.
//
// KB19HTTPClient implements the KB19Client interface for KB-19 Protocol Orchestrator.
// It provides clinical protocol recommendations, guideline reasoning, and pathway orchestration.
//
// ARCHITECTURE CRITICAL (CTO/CMO Directive):
//
//	"CQL explains. KB-19 recommends. ICU decides."
//
// KB-19 is a RUNTIME category KB (Category B). It operates AFTER CQL classification
// and is subject to ICU Dominance veto. KB-19 can ONLY:
//   - Provide recommendations based on CQL classifications
//   - Explain rationale for protocol selection
//   - Suggest pathways and next steps
//   - Track evidence grades (from KB-15)
//
// KB-19 CANNOT:
//   - Make binding clinical decisions
//   - Override safety vetoes
//   - Bypass ICU Dominance authority
//   - Execute actions without ICU approval
//
// Workflow Pattern:
//  1. CQL evaluates → produces classifications (e.g., "Meets Sepsis Criteria")
//  2. KB-19 called → provides protocol recommendations ("Hour-1 Bundle")
//  3. ICU veto check → approve/reject the recommendation
//  4. KB-12/KB-14 → execute approved protocol
//
// All KB-19 recommendations include DeferToICUIfDominant=true by design.
//
// Connects to: http://localhost:8099 (Docker: kb19-protocol-orchestrator)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// KB-19 PROTOCOL TYPES
// ============================================================================

// ProtocolRecommendation represents a KB-19 protocol recommendation.
// Per CTO/CMO directive: All recommendations defer to ICU and can be overridden.
type ProtocolRecommendation struct {
	// RecommendationID unique identifier
	RecommendationID string `json:"recommendation_id"`

	// RecommendedProtocol is the suggested protocol/guideline
	RecommendedProtocol string `json:"recommended_protocol"`

	// ProtocolID for KB-3 lookup
	ProtocolID string `json:"protocol_id,omitempty"`

	// Rationale explains why this protocol is recommended
	Rationale string `json:"rationale"`

	// CQLClassification that triggered this recommendation
	CQLClassification string `json:"cql_classification,omitempty"`

	// EvidenceGrade is the evidence quality (from KB-15)
	// Format: "A", "B-R", "B-NR", "C-LD", "C-EO"
	EvidenceGrade string `json:"evidence_grade"`

	// ClassOfRecommendation (I, IIa, IIb, III)
	ClassOfRecommendation string `json:"class_of_recommendation,omitempty"`

	// GuidelineSource originating guideline
	GuidelineSource string `json:"guideline_source,omitempty"`

	// Priority of this recommendation
	Priority string `json:"priority,omitempty"`

	// TargetTimeframe for implementation
	TargetTimeframe string `json:"target_timeframe,omitempty"`

	// IsTimeCritical if requires immediate action
	IsTimeCritical bool `json:"is_time_critical,omitempty"`

	// DeferToICUIfDominant MUST always be true
	// KB-19 cannot override ICU authority
	DeferToICUIfDominant bool `json:"defer_to_icu_if_dominant"`

	// CanBeOverriddenByICU MUST always be true
	// ICU can always override KB-19 recommendations
	CanBeOverriddenByICU bool `json:"can_be_overridden_by_icu"`

	// NextSteps suggested actions to implement the recommendation
	NextSteps []ProtocolStep `json:"next_steps,omitempty"`

	// RequiresApproval if needs explicit physician approval
	RequiresApproval bool `json:"requires_approval,omitempty"`

	// ApprovalLevel required (resident, attending, specialist)
	ApprovalLevel string `json:"approval_level,omitempty"`

	// Contraindications to this recommendation
	Contraindications []string `json:"contraindications,omitempty"`

	// AlternativeRecommendations if this one is contraindicated
	AlternativeRecommendations []string `json:"alternative_recommendations,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`
}

// ProtocolStep represents a step in a protocol.
type ProtocolStep struct {
	// StepID identifier
	StepID string `json:"step_id"`

	// Order in sequence
	Order int `json:"order"`

	// Description of the step
	Description string `json:"description"`

	// Type of step (medication, lab, procedure, assessment, monitoring)
	Type string `json:"type"`

	// ActionCode (RxNorm, LOINC, SNOMED)
	ActionCode string `json:"action_code,omitempty"`

	// ActionCodeSystem (http://www.nlm.nih.gov/research/umls/rxnorm, etc.)
	ActionCodeSystem string `json:"action_code_system,omitempty"`

	// TimeWindow for completion (e.g., "within 1 hour")
	TimeWindow string `json:"time_window,omitempty"`

	// IsCritical if time-sensitive
	IsCritical bool `json:"is_critical,omitempty"`

	// RequiresVerification after completion
	RequiresVerification bool `json:"requires_verification,omitempty"`

	// Dependencies on other steps (by StepID)
	Dependencies []string `json:"dependencies,omitempty"`
}

// ProtocolBundle represents a collection of related recommendations.
type ProtocolBundle struct {
	// BundleID identifier
	BundleID string `json:"bundle_id"`

	// Name of the bundle (e.g., "Sepsis Hour-1 Bundle")
	Name string `json:"name"`

	// TriggeringCondition that activated this bundle
	TriggeringCondition string `json:"triggering_condition,omitempty"`

	// CQLClassifications that triggered this bundle
	CQLClassifications []string `json:"cql_classifications,omitempty"`

	// Recommendations in this bundle
	Recommendations []ProtocolRecommendation `json:"recommendations"`

	// TimeConstraint for the entire bundle
	TimeConstraint string `json:"time_constraint,omitempty"`

	// CompletionCriteria for bundle completion
	CompletionCriteria []string `json:"completion_criteria,omitempty"`

	// GuidelineSource for the bundle
	GuidelineSource string `json:"guideline_source,omitempty"`

	// EvidenceGrade for the bundle overall
	EvidenceGrade string `json:"evidence_grade,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`
}

// PathwayRecommendation represents a clinical pathway recommendation.
type PathwayRecommendation struct {
	// PathwayID identifier
	PathwayID string `json:"pathway_id"`

	// Name of the pathway
	Name string `json:"name"`

	// Description of the pathway
	Description string `json:"description,omitempty"`

	// CurrentStage in the pathway
	CurrentStage string `json:"current_stage,omitempty"`

	// RecommendedStage to transition to
	RecommendedStage string `json:"recommended_stage,omitempty"`

	// Rationale for stage transition
	Rationale string `json:"rationale,omitempty"`

	// EvidenceGrade for this recommendation
	EvidenceGrade string `json:"evidence_grade,omitempty"`

	// NextActions in the recommended stage
	NextActions []ProtocolStep `json:"next_actions,omitempty"`

	// AlternativePathways if this one is not suitable
	AlternativePathways []string `json:"alternative_pathways,omitempty"`

	// DeferToICUIfDominant MUST always be true
	DeferToICUIfDominant bool `json:"defer_to_icu_if_dominant"`

	// CanBeOverriddenByICU MUST always be true
	CanBeOverriddenByICU bool `json:"can_be_overridden_by_icu"`
}

// RecommendationExplanation provides detailed reasoning for a recommendation.
type RecommendationExplanation struct {
	// RecommendationID being explained
	RecommendationID string `json:"recommendation_id"`

	// Summary brief explanation
	Summary string `json:"summary"`

	// DetailedRationale comprehensive reasoning
	DetailedRationale string `json:"detailed_rationale"`

	// CQLFactsUsed classification facts that led to this recommendation
	CQLFactsUsed []CQLFact `json:"cql_facts_used,omitempty"`

	// GuidelineReferences supporting evidence
	GuidelineReferences []GuidelineReference `json:"guideline_references,omitempty"`

	// RiskBenefitAnalysis for this recommendation
	RiskBenefitAnalysis string `json:"risk_benefit_analysis,omitempty"`

	// Alternatives considered and why rejected
	AlternativesConsidered []AlternativeConsideration `json:"alternatives_considered,omitempty"`

	// PatientSpecificFactors affecting this recommendation
	PatientSpecificFactors []string `json:"patient_specific_factors,omitempty"`
}

// CQLFact represents a fact from CQL classification.
type CQLFact struct {
	// FactName (e.g., "Meets Sepsis Criteria")
	FactName string `json:"fact_name"`

	// Value of the fact
	Value interface{} `json:"value"`

	// Library source CQL library
	Library string `json:"library,omitempty"`

	// Definition name in the library
	Definition string `json:"definition,omitempty"`
}

// GuidelineReference references supporting guideline evidence.
type GuidelineReference struct {
	// GuidelineID identifier
	GuidelineID string `json:"guideline_id,omitempty"`

	// Title of the guideline
	Title string `json:"title"`

	// Source organization
	Source string `json:"source"`

	// Year of publication
	Year int `json:"year,omitempty"`

	// Recommendation text from the guideline
	RecommendationText string `json:"recommendation_text,omitempty"`

	// ClassOfRecommendation (I, IIa, IIb, III)
	ClassOfRecommendation string `json:"class_of_recommendation,omitempty"`

	// LevelOfEvidence (A, B-R, B-NR, C-LD, C-EO)
	LevelOfEvidence string `json:"level_of_evidence,omitempty"`

	// URL to the guideline
	URL string `json:"url,omitempty"`
}

// AlternativeConsideration explains why an alternative was not chosen.
type AlternativeConsideration struct {
	// Alternative recommendation considered
	Alternative string `json:"alternative"`

	// Reason for not choosing this alternative
	Reason string `json:"reason"`
}

// ICUDominanceContext provides context about ICU dominance state.
type ICUDominanceContext struct {
	// IsICUDominant if ICU has current authority
	IsICUDominant bool `json:"is_icu_dominant"`

	// DominanceReason why ICU dominance is active
	DominanceReason string `json:"dominance_reason,omitempty"`

	// TriggeringConditions that activated dominance
	TriggeringConditions []string `json:"triggering_conditions,omitempty"`

	// ActivatedAt when dominance was activated
	ActivatedAt *time.Time `json:"activated_at,omitempty"`

	// RecommendationsDeferred recommendations waiting for ICU approval
	RecommendationsDeferred []string `json:"recommendations_deferred,omitempty"`
}

// ProtocolExecutionPlan represents a plan for executing recommendations.
type ProtocolExecutionPlan struct {
	// PlanID identifier
	PlanID string `json:"plan_id"`

	// PatientID the plan is for
	PatientID string `json:"patient_id"`

	// RecommendationIDs being executed
	RecommendationIDs []string `json:"recommendation_ids"`

	// Steps in the execution plan
	Steps []PlannedStep `json:"steps"`

	// ICUApprovalRequired if needs ICU approval
	ICUApprovalRequired bool `json:"icu_approval_required"`

	// ICUApprovalStatus (pending, approved, rejected)
	ICUApprovalStatus string `json:"icu_approval_status,omitempty"`

	// TargetCompletionTime for the plan
	TargetCompletionTime *time.Time `json:"target_completion_time,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`
}

// PlannedStep represents a step in an execution plan.
type PlannedStep struct {
	// StepID identifier
	StepID string `json:"step_id"`

	// Order in sequence
	Order int `json:"order"`

	// ActionType (medication, lab, procedure, assessment)
	ActionType string `json:"action_type"`

	// Description of the action
	Description string `json:"description"`

	// TargetTime for this step
	TargetTime *time.Time `json:"target_time,omitempty"`

	// AssignedTo role or person
	AssignedTo string `json:"assigned_to,omitempty"`

	// Status (pending, in_progress, completed, skipped)
	Status string `json:"status"`

	// CompletedAt if done
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// RequiresICUApproval for this specific step
	RequiresICUApproval bool `json:"requires_icu_approval,omitempty"`
}

// ============================================================================
// KB-19 HTTP CLIENT
// ============================================================================

// KB19HTTPClient implements KB19Client by calling the KB-19 Protocol Orchestrator REST API.
type KB19HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB19HTTPClient creates a new KB-19 HTTP client.
func NewKB19HTTPClient(baseURL string) *KB19HTTPClient {
	return &KB19HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB19HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB19HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB19HTTPClient {
	return &KB19HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// RECOMMENDATION METHODS
// ============================================================================

// GetRecommendations returns protocol recommendations based on CQL classifications.
// Calls KB-19 POST /api/v1/recommendations endpoint.
//
// ARCHITECTURE NOTE: All recommendations returned include:
//   - DeferToICUIfDominant = true
//   - CanBeOverriddenByICU = true
//
// Caller MUST check ICU dominance before executing any recommendations.
func (c *KB19HTTPClient) GetRecommendations(
	ctx context.Context,
	patient *contracts.PatientContext,
	cqlClassifications map[string]interface{},
) ([]ProtocolRecommendation, error) {

	// Extract condition codes for protocol matching
	conditionCodes := make([]string, 0, len(patient.ActiveConditions))
	for _, cond := range patient.ActiveConditions {
		conditionCodes = append(conditionCodes, cond.Code.Code)
	}

	req := kb19RecommendationRequest{
		PatientID:          patient.Demographics.PatientID,
		Region:             patient.Demographics.Region,
		ConditionCodes:     conditionCodes,
		CQLClassifications: cqlClassifications,
	}

	resp, err := c.callKB19(ctx, "/api/v1/recommendations", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendations: %w", err)
	}

	var result kb19RecommendationsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse recommendations: %w", err)
	}

	// Ensure all recommendations have proper ICU deferral flags
	for i := range result.Recommendations {
		result.Recommendations[i].DeferToICUIfDominant = true
		result.Recommendations[i].CanBeOverriddenByICU = true
	}

	return result.Recommendations, nil
}

// GetRecommendationDetail returns detailed information about a recommendation.
// Calls KB-19 GET /api/v1/recommendations/{id} endpoint.
func (c *KB19HTTPClient) GetRecommendationDetail(
	ctx context.Context,
	recommendationID string,
) (*ProtocolRecommendation, error) {

	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/recommendations/%s", url.PathEscape(recommendationID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendation detail: %w", err)
	}

	var recommendation ProtocolRecommendation
	if err := json.Unmarshal(resp, &recommendation); err != nil {
		return nil, fmt.Errorf("failed to parse recommendation: %w", err)
	}

	// Ensure ICU deferral flags are set
	recommendation.DeferToICUIfDominant = true
	recommendation.CanBeOverriddenByICU = true

	return &recommendation, nil
}

// ============================================================================
// BUNDLE METHODS
// ============================================================================

// GetProtocolBundle returns a bundle of recommendations for a clinical trigger.
// Calls KB-19 POST /api/v1/bundles endpoint.
//
// Example: Sepsis Hour-1 Bundle, Stroke Code Bundle, ACS STEMI Bundle
func (c *KB19HTTPClient) GetProtocolBundle(
	ctx context.Context,
	triggeringCondition string,
	cqlClassifications map[string]interface{},
) (*ProtocolBundle, error) {

	req := kb19BundleRequest{
		TriggeringCondition: triggeringCondition,
		CQLClassifications:  cqlClassifications,
	}

	resp, err := c.callKB19(ctx, "/api/v1/bundles", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol bundle: %w", err)
	}

	var bundle ProtocolBundle
	if err := json.Unmarshal(resp, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse protocol bundle: %w", err)
	}

	// Ensure all recommendations in bundle have ICU deferral flags
	for i := range bundle.Recommendations {
		bundle.Recommendations[i].DeferToICUIfDominant = true
		bundle.Recommendations[i].CanBeOverriddenByICU = true
	}

	return &bundle, nil
}

// GetAvailableBundles returns available protocol bundles.
// Calls KB-19 GET /api/v1/bundles endpoint.
func (c *KB19HTTPClient) GetAvailableBundles(ctx context.Context) ([]ProtocolBundle, error) {
	resp, err := c.doGet(ctx, "/api/v1/bundles")
	if err != nil {
		return nil, fmt.Errorf("failed to get available bundles: %w", err)
	}

	var result kb19BundlesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse bundles: %w", err)
	}

	return result.Bundles, nil
}

// ============================================================================
// PATHWAY METHODS
// ============================================================================

// GetPathwayRecommendation returns a pathway stage recommendation.
// Calls KB-19 POST /api/v1/pathways/recommend endpoint.
func (c *KB19HTTPClient) GetPathwayRecommendation(
	ctx context.Context,
	patientID string,
	pathwayID string,
	currentStage string,
	cqlClassifications map[string]interface{},
) (*PathwayRecommendation, error) {

	req := kb19PathwayRequest{
		PatientID:          patientID,
		PathwayID:          pathwayID,
		CurrentStage:       currentStage,
		CQLClassifications: cqlClassifications,
	}

	resp, err := c.callKB19(ctx, "/api/v1/pathways/recommend", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get pathway recommendation: %w", err)
	}

	var recommendation PathwayRecommendation
	if err := json.Unmarshal(resp, &recommendation); err != nil {
		return nil, fmt.Errorf("failed to parse pathway recommendation: %w", err)
	}

	// Ensure ICU deferral flags are set
	recommendation.DeferToICUIfDominant = true
	recommendation.CanBeOverriddenByICU = true

	return &recommendation, nil
}

// GetAvailablePathways returns available clinical pathways.
// Calls KB-19 GET /api/v1/pathways endpoint.
func (c *KB19HTTPClient) GetAvailablePathways(ctx context.Context, conditionCode string) ([]PathwayRecommendation, error) {
	reqURL := "/api/v1/pathways"
	if conditionCode != "" {
		reqURL += "?condition=" + url.QueryEscape(conditionCode)
	}

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get available pathways: %w", err)
	}

	var result kb19PathwaysResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pathways: %w", err)
	}

	return result.Pathways, nil
}

// ============================================================================
// EXPLANATION METHODS
// ============================================================================

// GetRecommendationExplanation returns detailed explanation for a recommendation.
// Calls KB-19 GET /api/v1/recommendations/{id}/explain endpoint.
//
// This implements the "CQL explains" part of the directive - providing
// transparent reasoning for why a recommendation was made.
func (c *KB19HTTPClient) GetRecommendationExplanation(
	ctx context.Context,
	recommendationID string,
) (*RecommendationExplanation, error) {

	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/recommendations/%s/explain", url.PathEscape(recommendationID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get explanation: %w", err)
	}

	var explanation RecommendationExplanation
	if err := json.Unmarshal(resp, &explanation); err != nil {
		return nil, fmt.Errorf("failed to parse explanation: %w", err)
	}

	return &explanation, nil
}

// ExplainCQLClassification explains how CQL classification led to recommendations.
// Calls KB-19 POST /api/v1/explain/classification endpoint.
func (c *KB19HTTPClient) ExplainCQLClassification(
	ctx context.Context,
	cqlLibrary string,
	cqlDefinition string,
	classificationResult interface{},
) (string, error) {

	req := kb19ExplainRequest{
		CQLLibrary:           cqlLibrary,
		CQLDefinition:        cqlDefinition,
		ClassificationResult: classificationResult,
	}

	resp, err := c.callKB19(ctx, "/api/v1/explain/classification", req)
	if err != nil {
		return "", fmt.Errorf("failed to explain classification: %w", err)
	}

	var result kb19ExplainResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse explanation: %w", err)
	}

	return result.Explanation, nil
}

// ============================================================================
// ICU DOMINANCE INTEGRATION METHODS
// ============================================================================

// CheckICUDominance checks if ICU dominance affects recommendations.
// Calls KB-19 POST /api/v1/icu/check endpoint.
//
// ARCHITECTURE NOTE: KB-19 checks ICU status but CANNOT override it.
// If ICU is dominant, KB-19 recommendations are subject to ICU approval.
func (c *KB19HTTPClient) CheckICUDominance(
	ctx context.Context,
	patientID string,
) (*ICUDominanceContext, error) {

	req := kb19ICUCheckRequest{
		PatientID: patientID,
	}

	resp, err := c.callKB19(ctx, "/api/v1/icu/check", req)
	if err != nil {
		return nil, fmt.Errorf("failed to check ICU dominance: %w", err)
	}

	var context ICUDominanceContext
	if err := json.Unmarshal(resp, &context); err != nil {
		return nil, fmt.Errorf("failed to parse ICU context: %w", err)
	}

	return &context, nil
}

// SubmitForICUApproval submits recommendations for ICU approval when dominant.
// Calls KB-19 POST /api/v1/icu/submit endpoint.
//
// This implements the "ICU decides" part of the directive.
func (c *KB19HTTPClient) SubmitForICUApproval(
	ctx context.Context,
	patientID string,
	recommendationIDs []string,
	urgencyLevel string,
) (string, error) {

	req := kb19ICUSubmitRequest{
		PatientID:         patientID,
		RecommendationIDs: recommendationIDs,
		UrgencyLevel:      urgencyLevel,
	}

	resp, err := c.callKB19(ctx, "/api/v1/icu/submit", req)
	if err != nil {
		return "", fmt.Errorf("failed to submit for ICU approval: %w", err)
	}

	var result kb19ICUSubmitResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse ICU submission result: %w", err)
	}

	return result.SubmissionID, nil
}

// GetICUApprovalStatus gets the status of an ICU approval request.
// Calls KB-19 GET /api/v1/icu/status/{submission_id} endpoint.
func (c *KB19HTTPClient) GetICUApprovalStatus(
	ctx context.Context,
	submissionID string,
) (string, error) {

	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/icu/status/%s", url.PathEscape(submissionID)))
	if err != nil {
		return "", fmt.Errorf("failed to get ICU approval status: %w", err)
	}

	var result kb19ICUStatusResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse ICU status: %w", err)
	}

	return result.Status, nil
}

// ============================================================================
// EXECUTION PLAN METHODS
// ============================================================================

// CreateExecutionPlan creates a plan for executing approved recommendations.
// Calls KB-19 POST /api/v1/plans endpoint.
//
// NOTE: Recommendations must be ICU-approved (if ICU dominant) before planning.
func (c *KB19HTTPClient) CreateExecutionPlan(
	ctx context.Context,
	patientID string,
	recommendationIDs []string,
) (*ProtocolExecutionPlan, error) {

	req := kb19PlanRequest{
		PatientID:         patientID,
		RecommendationIDs: recommendationIDs,
	}

	resp, err := c.callKB19(ctx, "/api/v1/plans", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}

	var plan ProtocolExecutionPlan
	if err := json.Unmarshal(resp, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse execution plan: %w", err)
	}

	return &plan, nil
}

// GetExecutionPlan returns an execution plan by ID.
// Calls KB-19 GET /api/v1/plans/{id} endpoint.
func (c *KB19HTTPClient) GetExecutionPlan(
	ctx context.Context,
	planID string,
) (*ProtocolExecutionPlan, error) {

	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/plans/%s", url.PathEscape(planID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}

	var plan ProtocolExecutionPlan
	if err := json.Unmarshal(resp, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse execution plan: %w", err)
	}

	return &plan, nil
}

// UpdatePlanStepStatus updates the status of a step in an execution plan.
// Calls KB-19 PUT /api/v1/plans/{id}/steps/{step_id} endpoint.
func (c *KB19HTTPClient) UpdatePlanStepStatus(
	ctx context.Context,
	planID string,
	stepID string,
	status string,
	completedBy string,
) error {

	req := kb19StepUpdateRequest{
		Status:      status,
		CompletedBy: completedBy,
	}

	_, err := c.callKB19(ctx, fmt.Sprintf("/api/v1/plans/%s/steps/%s", planID, stepID), req)
	if err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	return nil
}

// ============================================================================
// TRACKING METHODS
// ============================================================================

// GetPatientRecommendations returns all recommendations for a patient.
// Calls KB-19 GET /api/v1/patients/{id}/recommendations endpoint.
func (c *KB19HTTPClient) GetPatientRecommendations(
	ctx context.Context,
	patientID string,
	status string,
) ([]ProtocolRecommendation, error) {

	reqURL := fmt.Sprintf("/api/v1/patients/%s/recommendations", url.PathEscape(patientID))
	if status != "" {
		reqURL += "?status=" + url.QueryEscape(status)
	}

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient recommendations: %w", err)
	}

	var result kb19RecommendationsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse patient recommendations: %w", err)
	}

	return result.Recommendations, nil
}

// GetPatientPlans returns all execution plans for a patient.
// Calls KB-19 GET /api/v1/patients/{id}/plans endpoint.
func (c *KB19HTTPClient) GetPatientPlans(
	ctx context.Context,
	patientID string,
) ([]ProtocolExecutionPlan, error) {

	resp, err := c.doGet(ctx, fmt.Sprintf("/api/v1/patients/%s/plans", url.PathEscape(patientID)))
	if err != nil {
		return nil, fmt.Errorf("failed to get patient plans: %w", err)
	}

	var result kb19PlansResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse patient plans: %w", err)
	}

	return result.Plans, nil
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// HealthCheck verifies KB-19 service availability.
func (c *KB19HTTPClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("KB-19 health check failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-19 unhealthy: status %d", httpResp.StatusCode)
	}

	return nil
}

// ============================================================================
// PRIVATE HELPER METHODS
// ============================================================================

// doGet makes a GET request to KB-19 service.
func (c *KB19HTTPClient) doGet(ctx context.Context, endpoint string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-19 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	return body, nil
}

// callKB19 makes a POST request to KB-19 service.
func (c *KB19HTTPClient) callKB19(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-19 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	return body, nil
}

// ============================================================================
// REQUEST/RESPONSE TYPES (PRIVATE)
// ============================================================================

type kb19RecommendationRequest struct {
	PatientID          string                 `json:"patient_id"`
	Region             string                 `json:"region,omitempty"`
	ConditionCodes     []string               `json:"condition_codes"`
	CQLClassifications map[string]interface{} `json:"cql_classifications"`
}

type kb19RecommendationsResult struct {
	Recommendations []ProtocolRecommendation `json:"recommendations"`
}

type kb19BundleRequest struct {
	TriggeringCondition string                 `json:"triggering_condition"`
	CQLClassifications  map[string]interface{} `json:"cql_classifications"`
}

type kb19BundlesResult struct {
	Bundles []ProtocolBundle `json:"bundles"`
}

type kb19PathwayRequest struct {
	PatientID          string                 `json:"patient_id"`
	PathwayID          string                 `json:"pathway_id"`
	CurrentStage       string                 `json:"current_stage"`
	CQLClassifications map[string]interface{} `json:"cql_classifications"`
}

type kb19PathwaysResult struct {
	Pathways []PathwayRecommendation `json:"pathways"`
}

type kb19ExplainRequest struct {
	CQLLibrary           string      `json:"cql_library"`
	CQLDefinition        string      `json:"cql_definition"`
	ClassificationResult interface{} `json:"classification_result"`
}

type kb19ExplainResult struct {
	Explanation string `json:"explanation"`
}

type kb19ICUCheckRequest struct {
	PatientID string `json:"patient_id"`
}

type kb19ICUSubmitRequest struct {
	PatientID         string   `json:"patient_id"`
	RecommendationIDs []string `json:"recommendation_ids"`
	UrgencyLevel      string   `json:"urgency_level"`
}

type kb19ICUSubmitResult struct {
	SubmissionID string `json:"submission_id"`
}

type kb19ICUStatusResult struct {
	Status string `json:"status"`
}

type kb19PlanRequest struct {
	PatientID         string   `json:"patient_id"`
	RecommendationIDs []string `json:"recommendation_ids"`
}

type kb19PlansResult struct {
	Plans []ProtocolExecutionPlan `json:"plans"`
}

type kb19StepUpdateRequest struct {
	Status      string `json:"status"`
	CompletedBy string `json:"completed_by,omitempty"`
}

// ============================================================================
// INTERFACE COMPLIANCE DOCUMENTATION
// ============================================================================
//
// KB19HTTPClient implements the KB19Client interface for Protocol Orchestration.
//
// ARCHITECTURE DIRECTIVE: "CQL explains. KB-19 recommends. ICU decides."
//
// All methods that return recommendations MUST enforce:
//   - DeferToICUIfDominant = true
//   - CanBeOverriddenByICU = true
//
// Recommendation Methods:
//   - GetRecommendations(ctx, patient, cqlClassifications) → []ProtocolRecommendation
//   - GetRecommendationDetail(ctx, recommendationID) → *ProtocolRecommendation
//
// Bundle Methods:
//   - GetProtocolBundle(ctx, triggeringCondition, cqlClassifications) → *ProtocolBundle
//   - GetAvailableBundles(ctx) → []ProtocolBundle
//
// Pathway Methods:
//   - GetPathwayRecommendation(ctx, patientID, pathwayID, currentStage, cqlClassifications) → *PathwayRecommendation
//   - GetAvailablePathways(ctx, conditionCode) → []PathwayRecommendation
//
// Explanation Methods ("CQL explains"):
//   - GetRecommendationExplanation(ctx, recommendationID) → *RecommendationExplanation
//   - ExplainCQLClassification(ctx, library, definition, result) → string
//
// ICU Dominance Integration ("ICU decides"):
//   - CheckICUDominance(ctx, patientID) → *ICUDominanceContext
//   - SubmitForICUApproval(ctx, patientID, recommendationIDs, urgency) → submissionID
//   - GetICUApprovalStatus(ctx, submissionID) → status
//
// Execution Plan Methods:
//   - CreateExecutionPlan(ctx, patientID, recommendationIDs) → *ProtocolExecutionPlan
//   - GetExecutionPlan(ctx, planID) → *ProtocolExecutionPlan
//   - UpdatePlanStepStatus(ctx, planID, stepID, status, completedBy) → error
//
// Tracking Methods:
//   - GetPatientRecommendations(ctx, patientID, status) → []ProtocolRecommendation
//   - GetPatientPlans(ctx, patientID) → []ProtocolExecutionPlan
//
// Health:
//   - HealthCheck(ctx) → error
// ============================================================================
