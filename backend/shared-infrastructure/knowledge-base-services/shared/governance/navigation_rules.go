// Package governance provides the Navigation Rules Engine for clinical fact extraction.
//
// Phase 3d.1: Navigation Rules Implementation
// Authority Level: GOVERNANCE LAYER (enforces extraction policies)
//
// THE 4 NON-NEGOTIABLE RULES:
//
// Rule 1: "Curated fact exists in authority → Use it, LLM never sees this."
//         If CPIC, CredibleMeds, LiverTox, LactMed, etc. have the answer, use it.
//
// Rule 2: "Table exists → PARSE, don't interpret."
//         Structured tables are extracted deterministically, not by LLM.
//
// Rule 3: "LLMs disagree → HUMAN first."
//         No single LLM's extraction is accepted without 2-of-3 consensus.
//
// Rule 4: "Provenance unclear → DRAFT only, never active."
//         Facts without clear source lineage stay in draft until curated.
//
// PHILOSOPHY: "Freeze meaning. Fluidly replace intelligence."
// LLM is a gap filler of last resort, never the primary source.
package governance

import (
	"context"
	"fmt"
	"time"
)

// =============================================================================
// NAVIGATION RULE ENGINE
// =============================================================================

// NavigationRuleEngine enforces the 4 non-negotiable rules
type NavigationRuleEngine struct {
	authorityRegistry *AuthorityRegistry
	tableDetector     TableDetector
	consensusChecker  ConsensusChecker
	provenanceChecker ProvenanceChecker
}

// NavigationEngineConfig contains navigation rule engine configuration
// Note: Named differently from EngineConfig in engine.go to avoid redeclaration
type NavigationEngineConfig struct {
	// AuthorityRegistry is the registry of authoritative sources
	AuthorityRegistry *AuthorityRegistry

	// TableDetector identifies structured table data
	TableDetector TableDetector

	// ConsensusChecker validates LLM consensus
	ConsensusChecker ConsensusChecker

	// ProvenanceChecker validates source lineage
	ProvenanceChecker ProvenanceChecker
}

// NewNavigationRuleEngine creates a new navigation rule engine
func NewNavigationRuleEngine(config NavigationEngineConfig) *NavigationRuleEngine {
	return &NavigationRuleEngine{
		authorityRegistry: config.AuthorityRegistry,
		tableDetector:     config.TableDetector,
		consensusChecker:  config.ConsensusChecker,
		provenanceChecker: config.ProvenanceChecker,
	}
}

// =============================================================================
// RULE INTERFACE
// =============================================================================

// Rule represents a navigation rule that must be evaluated
type Rule interface {
	// ID returns the rule number (1-4)
	ID() int

	// Name returns the human-readable rule name
	Name() string

	// Description returns a detailed explanation of the rule
	Description() string

	// Check evaluates the rule against an extraction request
	Check(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error)
}

// =============================================================================
// EXTRACTION REQUEST
// =============================================================================

// ExtractionRequest represents a request to extract clinical facts
type ExtractionRequest struct {
	// ─────────────────────────────────────────────────────────────────────────
	// TARGET IDENTIFICATION
	// ─────────────────────────────────────────────────────────────────────────

	// RxCUI is the RxNorm Concept Unique Identifier
	RxCUI string `json:"rxcui"`

	// DrugName is the drug name
	DrugName string `json:"drugName"`

	// FactType is the type of fact being extracted
	FactType FactType `json:"factType"`

	// ─────────────────────────────────────────────────────────────────────────
	// SOURCE CONTENT
	// ─────────────────────────────────────────────────────────────────────────

	// SourceText is the text to extract from
	SourceText string `json:"sourceText,omitempty"`

	// SourceDocumentID links to source_documents table
	SourceDocumentID string `json:"sourceDocumentId,omitempty"`

	// SourceSectionID links to source_sections table
	SourceSectionID string `json:"sourceSectionId,omitempty"`

	// LOINCCode is the section LOINC code (for SPL sections)
	LOINCCode string `json:"loincCode,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTED DATA (if already extracted)
	// ─────────────────────────────────────────────────────────────────────────

	// HasTableData indicates if structured table data was found
	HasTableData bool `json:"hasTableData"`

	// TableCount is the number of relevant tables found
	TableCount int `json:"tableCount"`

	// ExtractionMethod is how the data was extracted (if known)
	ExtractionMethod ExtractionMethod `json:"extractionMethod,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONSENSUS DATA (if LLM was used)
	// ─────────────────────────────────────────────────────────────────────────

	// LLMConsensusAchieved indicates if LLM consensus was achieved
	LLMConsensusAchieved bool `json:"llmConsensusAchieved"`

	// LLMProviderCount is the number of LLM providers queried
	LLMProviderCount int `json:"llmProviderCount"`

	// LLMAgreementCount is the number of providers that agreed
	LLMAgreementCount int `json:"llmAgreementCount"`

	// LLMDisagreements lists fields where LLMs disagreed
	LLMDisagreements []string `json:"llmDisagreements,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// PROVENANCE
	// ─────────────────────────────────────────────────────────────────────────

	// HasClearProvenance indicates if source lineage is complete
	HasClearProvenance bool `json:"hasClearProvenance"`

	// ProvenanceIssues lists any provenance problems
	ProvenanceIssues []string `json:"provenanceIssues,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// RequestID is a unique identifier for this request
	RequestID string `json:"requestId"`

	// RequestedAt is when the request was made
	RequestedAt time.Time `json:"requestedAt"`

	// RequestedBy identifies who/what made the request
	RequestedBy string `json:"requestedBy,omitempty"`
}

// FactType categorizes the type of clinical fact
type FactType string

const (
	FactTypeRenalDosing    FactType = "RENAL_DOSING"
	FactTypeHepaticDosing  FactType = "HEPATIC_DOSING"
	FactTypeDrugInteraction FactType = "DRUG_INTERACTION"
	FactTypeQTProlongation FactType = "QT_PROLONGATION"
	FactTypeHepatotoxicity FactType = "HEPATOTOXICITY"
	FactTypeLactationRisk  FactType = "LACTATION_RISK"
	FactTypePharmacogenomics FactType = "PHARMACOGENOMICS"
	FactTypeContraindication FactType = "CONTRAINDICATION"
	FactTypePregnancyRisk  FactType = "PREGNANCY_RISK"
	FactTypeAdverseReaction FactType = "ADVERSE_REACTION"
)

// ExtractionMethod indicates how data was extracted
type ExtractionMethod string

const (
	MethodAuthority ExtractionMethod = "AUTHORITY"     // From curated source
	MethodTable     ExtractionMethod = "TABLE_PARSE"   // From structured table
	MethodRegex     ExtractionMethod = "REGEX_PARSE"   // From regex patterns
	MethodLLM       ExtractionMethod = "LLM_CONSENSUS" // From LLM with consensus
	MethodHuman     ExtractionMethod = "HUMAN_CURATED" // Manually curated
)

// =============================================================================
// RULE DECISION
// =============================================================================

// RuleDecision indicates what action to take based on rule evaluation
type RuleDecision struct {
	// ─────────────────────────────────────────────────────────────────────────
	// RULE IDENTITY
	// ─────────────────────────────────────────────────────────────────────────

	// RuleID is the rule number (1-4)
	RuleID int `json:"ruleId"`

	// RuleName is the rule name
	RuleName string `json:"ruleName"`

	// ─────────────────────────────────────────────────────────────────────────
	// DECISION
	// ─────────────────────────────────────────────────────────────────────────

	// Action is the required action based on this rule
	Action Action `json:"action"`

	// AllowLLM indicates if LLM extraction is permitted
	AllowLLM bool `json:"allowLlm"`

	// Reason explains why this decision was made
	Reason string `json:"reason"`

	// ─────────────────────────────────────────────────────────────────────────
	// AUTHORITY HIT (for Rule 1)
	// ─────────────────────────────────────────────────────────────────────────

	// AuthorityHit contains the authority source if found
	AuthorityHit *AuthoritySource `json:"authorityHit,omitempty"`

	// AuthorityFactID is the ID of the fact in the authority
	AuthorityFactID string `json:"authorityFactId,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// GOVERNANCE STATUS
	// ─────────────────────────────────────────────────────────────────────────

	// SuggestedStatus is the recommended fact status
	SuggestedStatus FactStatus `json:"suggestedStatus"`

	// RequiresHumanReview indicates if human review is needed
	RequiresHumanReview bool `json:"requiresHumanReview"`

	// EscalationReason explains why escalation is needed
	EscalationReason string `json:"escalationReason,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// EvaluatedAt is when the rule was evaluated
	EvaluatedAt time.Time `json:"evaluatedAt"`

	// EvaluationDuration is how long the evaluation took
	EvaluationDuration time.Duration `json:"evaluationDuration"`
}

// Action defines the action to take based on rule evaluation
type Action string

const (
	// ActionBlockLLM means LLM is not permitted (authority exists)
	ActionBlockLLM Action = "BLOCK_LLM"

	// ActionParseTable means use structured table parsing
	ActionParseTable Action = "PARSE_TABLE"

	// ActionHumanReview means escalate to human review
	ActionHumanReview Action = "HUMAN_REVIEW"

	// ActionDraftOnly means fact can only be draft status
	ActionDraftOnly Action = "DRAFT_ONLY"

	// ActionAllowLLM means LLM extraction is permitted
	ActionAllowLLM Action = "ALLOW_LLM"

	// ActionUseAuthority means use the authoritative fact
	ActionUseAuthority Action = "USE_AUTHORITY"
)

// FactStatus represents the governance status of a fact
type FactStatus string

const (
	StatusDraft    FactStatus = "DRAFT"     // Not yet reviewed
	StatusPending  FactStatus = "PENDING"   // Awaiting review
	StatusApproved FactStatus = "APPROVED"  // Reviewed and approved
	StatusRejected FactStatus = "REJECTED"  // Reviewed and rejected
	StatusActive   FactStatus = "ACTIVE"    // In production use
)

// =============================================================================
// AUTHORITY TYPES
// =============================================================================

// AuthoritySource represents a curated data source
type AuthoritySource struct {
	// Name is the authority name (e.g., "CPIC", "CredibleMeds")
	Name string `json:"name"`

	// FactType is the fact type this authority provides
	FactType FactType `json:"factType"`

	// ConfidenceLevel is the trust level (DEFINITIVE, HIGH, MEDIUM)
	ConfidenceLevel AuthorityLevel `json:"confidenceLevel"`

	// LastUpdated is when the authority was last refreshed
	LastUpdated time.Time `json:"lastUpdated"`
}

// AuthorityLevel indicates the trust level of an authority
type AuthorityLevel string

const (
	// AuthorityDefinitive means LLM is NEVER allowed (CPIC, CredibleMeds, etc.)
	AuthorityDefinitive AuthorityLevel = "DEFINITIVE"

	// AuthorityHigh means LLM is discouraged but allowed for gaps
	AuthorityHigh AuthorityLevel = "HIGH"

	// AuthorityMedium means LLM can supplement but not replace
	AuthorityMedium AuthorityLevel = "MEDIUM"
)

// =============================================================================
// RULE IMPLEMENTATIONS
// =============================================================================

// Rule1AuthorityCheck - "Curated fact exists in authority → Use it, LLM never sees this"
type Rule1AuthorityCheck struct {
	registry *AuthorityRegistry
}

func (r *Rule1AuthorityCheck) ID() int { return 1 }

func (r *Rule1AuthorityCheck) Name() string { return "Authority Existence Check" }

func (r *Rule1AuthorityCheck) Description() string {
	return "If a curated fact exists in an authoritative source (CPIC, CredibleMeds, LiverTox, LactMed), use it. LLM never sees this fact type."
}

func (r *Rule1AuthorityCheck) Check(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error) {
	startTime := time.Now()

	decision := &RuleDecision{
		RuleID:      1,
		RuleName:    r.Name(),
		EvaluatedAt: time.Now(),
	}

	// Check if this fact type has an authoritative source
	authority, found := r.registry.GetAuthorityForFactType(req.FactType)
	if !found {
		decision.Action = ActionAllowLLM
		decision.AllowLLM = true
		decision.Reason = "No authority defined for this fact type"
		decision.SuggestedStatus = StatusDraft
		decision.EvaluationDuration = time.Since(startTime)
		return decision, nil
	}

	// Check if authority has data for this drug
	hasFact, factID, err := r.registry.HasFact(ctx, authority.Name, req.RxCUI, req.FactType)
	if err != nil {
		return nil, fmt.Errorf("checking authority %s: %w", authority.Name, err)
	}

	if hasFact {
		// Rule 1 triggered: Authority has the fact, block LLM
		decision.Action = ActionUseAuthority
		decision.AllowLLM = false
		decision.Reason = fmt.Sprintf("Fact exists in DEFINITIVE authority: %s (LLM NEVER allowed)", authority.Name)
		decision.AuthorityHit = authority
		decision.AuthorityFactID = factID
		decision.SuggestedStatus = StatusActive // Authority facts go straight to active

		// DEFINITIVE authorities = LLM is absolutely forbidden
		if authority.ConfidenceLevel == AuthorityDefinitive {
			decision.Action = ActionBlockLLM
		}

		decision.EvaluationDuration = time.Since(startTime)
		return decision, nil
	}

	// Authority exists but doesn't have this specific fact
	decision.Action = ActionAllowLLM
	decision.AllowLLM = true
	decision.Reason = fmt.Sprintf("Fact not found in authority %s, LLM gap-filling permitted", authority.Name)
	decision.SuggestedStatus = StatusDraft
	decision.EvaluationDuration = time.Since(startTime)
	return decision, nil
}

// Rule2TableCheck - "Table exists → PARSE, don't interpret"
type Rule2TableCheck struct{}

func (r *Rule2TableCheck) ID() int { return 2 }

func (r *Rule2TableCheck) Name() string { return "Structured Table Check" }

func (r *Rule2TableCheck) Description() string {
	return "If structured table data exists in the source, parse it deterministically. Do not use LLM to interpret tables."
}

func (r *Rule2TableCheck) Check(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error) {
	startTime := time.Now()

	decision := &RuleDecision{
		RuleID:      2,
		RuleName:    r.Name(),
		EvaluatedAt: time.Now(),
	}

	if req.HasTableData && req.TableCount > 0 {
		// Rule 2 triggered: Table exists, use deterministic parsing
		decision.Action = ActionParseTable
		decision.AllowLLM = false
		decision.Reason = fmt.Sprintf("Found %d structured table(s) in source - use deterministic parsing, not LLM", req.TableCount)
		decision.SuggestedStatus = StatusPending // Tables need verification but not LLM
		decision.EvaluationDuration = time.Since(startTime)
		return decision, nil
	}

	// No table data, LLM may be needed
	decision.Action = ActionAllowLLM
	decision.AllowLLM = true
	decision.Reason = "No structured table data found, LLM may be used for prose extraction"
	decision.SuggestedStatus = StatusDraft
	decision.EvaluationDuration = time.Since(startTime)
	return decision, nil
}

// Rule3ConsensusCheck - "LLMs disagree → HUMAN first"
type Rule3ConsensusCheck struct{}

func (r *Rule3ConsensusCheck) ID() int { return 3 }

func (r *Rule3ConsensusCheck) Name() string { return "LLM Consensus Check" }

func (r *Rule3ConsensusCheck) Description() string {
	return "If LLM extraction was used and consensus was not achieved (2-of-3 agreement), escalate to human review. No single LLM is trusted alone."
}

func (r *Rule3ConsensusCheck) Check(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error) {
	startTime := time.Now()

	decision := &RuleDecision{
		RuleID:      3,
		RuleName:    r.Name(),
		EvaluatedAt: time.Now(),
	}

	// Only applies if LLM was used
	if req.ExtractionMethod != MethodLLM || req.LLMProviderCount == 0 {
		decision.Action = ActionAllowLLM
		decision.AllowLLM = true
		decision.Reason = "LLM extraction not used, rule not applicable"
		decision.SuggestedStatus = StatusDraft
		decision.EvaluationDuration = time.Since(startTime)
		return decision, nil
	}

	// Check if consensus was achieved
	minRequired := 2 // Require 2-of-3 agreement
	if req.LLMAgreementCount < minRequired {
		// Rule 3 triggered: LLMs disagree, require human review
		decision.Action = ActionHumanReview
		decision.AllowLLM = false // Block further LLM use
		decision.Reason = fmt.Sprintf("LLM consensus not achieved: %d/%d providers agreed (minimum %d required). Disagreements: %v",
			req.LLMAgreementCount, req.LLMProviderCount, minRequired, req.LLMDisagreements)
		decision.SuggestedStatus = StatusPending
		decision.RequiresHumanReview = true
		decision.EscalationReason = "LLM disagreement on clinical fact - human verification required"
		decision.EvaluationDuration = time.Since(startTime)
		return decision, nil
	}

	// Consensus achieved
	decision.Action = ActionAllowLLM
	decision.AllowLLM = true
	decision.Reason = fmt.Sprintf("LLM consensus achieved: %d/%d providers agreed", req.LLMAgreementCount, req.LLMProviderCount)
	decision.SuggestedStatus = StatusDraft // Still draft, needs governance review
	decision.EvaluationDuration = time.Since(startTime)
	return decision, nil
}

// Rule4ProvenanceCheck - "Provenance unclear → DRAFT only, never active"
type Rule4ProvenanceCheck struct{}

func (r *Rule4ProvenanceCheck) ID() int { return 4 }

func (r *Rule4ProvenanceCheck) Name() string { return "Provenance Clarity Check" }

func (r *Rule4ProvenanceCheck) Description() string {
	return "Facts without clear source lineage (source document, section, extraction method) remain in DRAFT status and cannot be activated until curated."
}

func (r *Rule4ProvenanceCheck) Check(ctx context.Context, req *ExtractionRequest) (*RuleDecision, error) {
	startTime := time.Now()

	decision := &RuleDecision{
		RuleID:      4,
		RuleName:    r.Name(),
		EvaluatedAt: time.Now(),
	}

	var issues []string

	// Check required provenance fields
	if req.SourceDocumentID == "" && req.SourceText == "" {
		issues = append(issues, "missing source document ID and source text")
	}
	if req.ExtractionMethod == "" {
		issues = append(issues, "missing extraction method")
	}
	if req.RxCUI == "" && req.DrugName == "" {
		issues = append(issues, "missing drug identification (RxCUI or drug name)")
	}

	// Combine with any issues already identified
	issues = append(issues, req.ProvenanceIssues...)

	if len(issues) > 0 || !req.HasClearProvenance {
		// Rule 4 triggered: Provenance unclear, DRAFT only
		decision.Action = ActionDraftOnly
		decision.AllowLLM = false // Can't use LLM without clear source
		decision.Reason = fmt.Sprintf("Provenance unclear: %v. Fact must remain DRAFT until source lineage is established.", issues)
		decision.SuggestedStatus = StatusDraft
		decision.RequiresHumanReview = true
		decision.EscalationReason = "Unclear provenance - manual source verification required"
		decision.EvaluationDuration = time.Since(startTime)
		return decision, nil
	}

	// Provenance is clear
	decision.Action = ActionAllowLLM
	decision.AllowLLM = true
	decision.Reason = "Provenance is clear and complete"
	decision.SuggestedStatus = StatusDraft // Will be promoted through governance
	decision.EvaluationDuration = time.Since(startTime)
	return decision, nil
}

// =============================================================================
// ENGINE EVALUATION
// =============================================================================

// EvaluationResult contains the complete evaluation outcome
type EvaluationResult struct {
	// FinalDecision is the overall decision
	FinalDecision *RuleDecision `json:"finalDecision"`

	// RuleDecisions contains decisions from each rule
	RuleDecisions []*RuleDecision `json:"ruleDecisions"`

	// AllRulesPassed indicates if all rules allowed LLM
	AllRulesPassed bool `json:"allRulesPassed"`

	// BlockingRule is the first rule that blocked LLM (if any)
	BlockingRule int `json:"blockingRule,omitempty"`

	// TotalDuration is the total evaluation time
	TotalDuration time.Duration `json:"totalDuration"`

	// EvaluatedAt is when evaluation completed
	EvaluatedAt time.Time `json:"evaluatedAt"`
}

// Evaluate runs all 4 rules in order and returns the final decision
func (e *NavigationRuleEngine) Evaluate(ctx context.Context, req *ExtractionRequest) (*EvaluationResult, error) {
	startTime := time.Now()

	result := &EvaluationResult{
		RuleDecisions: make([]*RuleDecision, 0, 4),
		AllRulesPassed: true,
		EvaluatedAt:   time.Now(),
	}

	// Define rules in order
	rules := []Rule{
		&Rule1AuthorityCheck{registry: e.authorityRegistry},
		&Rule2TableCheck{},
		&Rule3ConsensusCheck{},
		&Rule4ProvenanceCheck{},
	}

	// Evaluate each rule
	for _, rule := range rules {
		decision, err := rule.Check(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("rule %d (%s) failed: %w", rule.ID(), rule.Name(), err)
		}

		result.RuleDecisions = append(result.RuleDecisions, decision)

		// If rule blocks LLM, stop and return
		if !decision.AllowLLM {
			result.FinalDecision = decision
			result.AllRulesPassed = false
			result.BlockingRule = rule.ID()
			result.TotalDuration = time.Since(startTime)
			return result, nil
		}
	}

	// All rules passed, LLM is permitted
	result.FinalDecision = &RuleDecision{
		RuleID:          0, // Combined decision
		RuleName:        "All Rules",
		Action:          ActionAllowLLM,
		AllowLLM:        true,
		Reason:          "All 4 navigation rules passed - LLM extraction is permitted",
		SuggestedStatus: StatusDraft,
		EvaluatedAt:     time.Now(),
	}
	result.TotalDuration = time.Since(startTime)

	return result, nil
}

// =============================================================================
// AUTHORITY REGISTRY
// =============================================================================

// AuthorityRegistry manages authoritative data sources
type AuthorityRegistry struct {
	authorities map[FactType]*AuthoritySource
	factCheckers map[string]AuthorityFactChecker
}

// AuthorityFactChecker interface for checking if an authority has a fact
type AuthorityFactChecker interface {
	HasFact(ctx context.Context, rxcui string, factType FactType) (bool, string, error)
}

// NewAuthorityRegistry creates a new authority registry
func NewAuthorityRegistry() *AuthorityRegistry {
	return &AuthorityRegistry{
		authorities: make(map[FactType]*AuthoritySource),
		factCheckers: make(map[string]AuthorityFactChecker),
	}
}

// RegisterAuthority registers an authoritative source for a fact type
func (r *AuthorityRegistry) RegisterAuthority(factType FactType, source *AuthoritySource) {
	r.authorities[factType] = source
}

// RegisterFactChecker registers a fact checker for an authority
func (r *AuthorityRegistry) RegisterFactChecker(authorityName string, checker AuthorityFactChecker) {
	r.factCheckers[authorityName] = checker
}

// GetAuthorityForFactType returns the authority for a fact type
func (r *AuthorityRegistry) GetAuthorityForFactType(factType FactType) (*AuthoritySource, bool) {
	auth, ok := r.authorities[factType]
	return auth, ok
}

// HasFact checks if an authority has a fact for a drug
func (r *AuthorityRegistry) HasFact(ctx context.Context, authorityName, rxcui string, factType FactType) (bool, string, error) {
	checker, ok := r.factCheckers[authorityName]
	if !ok {
		return false, "", fmt.Errorf("no fact checker registered for authority: %s", authorityName)
	}
	return checker.HasFact(ctx, rxcui, factType)
}

// =============================================================================
// INTERFACES FOR DEPENDENCIES
// =============================================================================

// TableDetector interface for detecting structured table data
type TableDetector interface {
	HasTables(sourceText string) (bool, int)
}

// ConsensusChecker interface for checking LLM consensus
type ConsensusChecker interface {
	CheckConsensus(results []interface{}) (bool, int, []string)
}

// ProvenanceChecker interface for validating provenance
type ProvenanceChecker interface {
	ValidateProvenance(req *ExtractionRequest) (bool, []string)
}

// =============================================================================
// DEFAULT IMPLEMENTATIONS
// =============================================================================

// DefaultAuthorityRegistry creates a registry with CardioFit authorities
func DefaultAuthorityRegistry() *AuthorityRegistry {
	registry := NewAuthorityRegistry()

	// Register DEFINITIVE authorities (LLM = NEVER)
	registry.RegisterAuthority(FactTypePharmacogenomics, &AuthoritySource{
		Name:            "CPIC",
		FactType:        FactTypePharmacogenomics,
		ConfidenceLevel: AuthorityDefinitive,
	})

	registry.RegisterAuthority(FactTypeQTProlongation, &AuthoritySource{
		Name:            "CredibleMeds",
		FactType:        FactTypeQTProlongation,
		ConfidenceLevel: AuthorityDefinitive,
	})

	registry.RegisterAuthority(FactTypeHepatotoxicity, &AuthoritySource{
		Name:            "LiverTox",
		FactType:        FactTypeHepatotoxicity,
		ConfidenceLevel: AuthorityDefinitive,
	})

	registry.RegisterAuthority(FactTypeLactationRisk, &AuthoritySource{
		Name:            "LactMed",
		FactType:        FactTypeLactationRisk,
		ConfidenceLevel: AuthorityDefinitive,
	})

	return registry
}

// GetRuleSummary returns a human-readable summary of all rules
func GetRuleSummary() string {
	return `
Navigation Rules for Clinical Fact Extraction
==============================================

Rule 1: Authority Existence Check
---------------------------------
"Curated fact exists in authority → Use it, LLM never sees this."
Authorities: CPIC (pharmacogenomics), CredibleMeds (QT risk),
             LiverTox (hepatotoxicity), LactMed (lactation)
These are DEFINITIVE - LLM is NEVER allowed.

Rule 2: Structured Table Check
------------------------------
"Table exists → PARSE, don't interpret."
If structured table data exists in the source, parse it
deterministically. Do not use LLM to interpret tables.

Rule 3: LLM Consensus Check
---------------------------
"LLMs disagree → HUMAN first."
No single LLM's extraction is trusted. Requires 2-of-3
provider consensus. Disagreements escalate to human review.

Rule 4: Provenance Clarity Check
--------------------------------
"Provenance unclear → DRAFT only, never active."
Facts without clear source lineage remain in DRAFT status
until source is established through human curation.

PHILOSOPHY: "Freeze meaning. Fluidly replace intelligence."
LLM is a gap filler of last resort, never the primary source.
`
}
