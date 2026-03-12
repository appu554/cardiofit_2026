package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/semantic"

	// NOTE: valuesets package import REMOVED - value sets are already in PostgreSQL!
	// KB-7 is the SINGLE SOURCE OF TRUTH - no need for hardcoded Go structs.

	"github.com/sirupsen/logrus"
)

// ============================================================================
// RuleManager - Database-driven value set rule engine
// Replaces hardcoded builtin_valuesets.go with dynamic PostgreSQL-backed rules
// ============================================================================

// RuleManager defines the interface for value set rule management
type RuleManager interface {
	// ExpandValueSet returns all codes for a value set (handles all definition types)
	ExpandValueSet(ctx context.Context, identifier string, version string) (*ExpandedValueSet, error)

	// ValidateCodeInValueSet checks if a code is in a value set
	ValidateCodeInValueSet(ctx context.Context, code, system, valueSetID string) (*RuleValidationResult, error)

	// GetValueSetDefinition returns the raw definition (not expanded)
	GetValueSetDefinition(ctx context.Context, identifier string) (*ValueSetDefinition, error)

	// RefreshCache forces re-expansion of a value set (invalidates cache)
	RefreshCache(ctx context.Context, identifier string) error

	// ListValueSets returns all value set definitions with optional filtering
	ListValueSets(ctx context.Context, filter ValueSetFilter) ([]ValueSetDefinition, error)

	// SeedBuiltinValueSets migrates hardcoded value sets to database
	SeedBuiltinValueSets(ctx context.Context) error

	// ClassifyCode finds ALL value sets that a code belongs to (reverse lookup)
	// This is the missing "FindValueSetsForCode" feature documented in specs
	// Input: just {code, system} - KB7 automatically finds matching value sets
	// Uses THREE-CHECK PIPELINE for each value set: Expansion → Exact → Subsumption
	ClassifyCode(ctx context.Context, code, system string) (*ClassificationResult, error)
}

// ============================================================================
// Data Models
// ============================================================================

// DefinitionType specifies how a value set is resolved
type DefinitionType string

const (
	// DefinitionTypeExplicit - Fixed list of codes stored directly in the table
	DefinitionTypeExplicit DefinitionType = "explicit"

	// DefinitionTypeIntensional - Expanded at runtime from graph hierarchy (GraphDB SPARQL)
	DefinitionTypeIntensional DefinitionType = "intensional"

	// DefinitionTypeExtensional - Composed from other value sets
	DefinitionTypeExtensional DefinitionType = "extensional"
)

// ExpansionRule specifies how to traverse the hierarchy for intensional value sets
type ExpansionRule string

const (
	ExpansionRuleDescendants       ExpansionRule = "descendants"
	ExpansionRuleAncestors         ExpansionRule = "ancestors"
	ExpansionRuleDescendantsOrSelf ExpansionRule = "descendants_or_self"
	ExpansionRuleAncestorsOrSelf   ExpansionRule = "ancestors_or_self"
)

// ============================================================================
// Code System Subsumption Support
// ============================================================================
// SNOMED CT has IS-A hierarchies that benefit from subsumption checking.
// LOINC, ICD-10, RxNorm etc. are "flat" systems with no IS-A relationships.
// Subsumption checking for flat systems wastes ~14 seconds per code.

// SubsumptionSupportedSystems - Code systems that have IS-A hierarchies
// Only these systems should attempt Neo4j subsumption queries
var SubsumptionSupportedSystems = map[string]bool{
	"http://snomed.info/sct":                    true, // SNOMED CT International
	"http://snomed.info/sct/32506021000036107":  true, // SNOMED CT-AU (Australian extension)
	"http://snomed.info/sct/900000000000207008": true, // SNOMED CT Core Module
	// AMT (Australian Medicines Terminology) uses SNOMED CT-AU namespace
}

// FlatCodeSystems - Code systems with NO IS-A hierarchies
// These systems should skip subsumption and use exact match only
var FlatCodeSystems = map[string]bool{
	"http://loinc.org":                             true, // LOINC (laboratory codes)
	"http://hl7.org/fhir/sid/icd-10":               true, // ICD-10
	"http://hl7.org/fhir/sid/icd-10-au":            true, // ICD-10-AU
	"http://hl7.org/fhir/sid/icd-10-am":            true, // ICD-10-AM
	"http://hl7.org/fhir/sid/icd-10-cm":            true, // ICD-10-CM (US)
	"http://hl7.org/fhir/sid/icd-9-cm":             true, // ICD-9-CM (legacy)
	"http://www.nlm.nih.gov/research/umls/rxnorm":  true, // RxNorm (medications)
	"http://hl7.org/fhir/sid/ndc":                  true, // NDC (drug codes)
	"urn:oid:2.16.840.1.113883.6.73":               true, // ATC (drug classification)
	"http://unitsofmeasure.org":                    true, // UCUM (units)
}

// ShouldAttemptSubsumption determines if a code system supports IS-A hierarchy queries
// Returns true only for SNOMED-based systems that benefit from subsumption checking
func ShouldAttemptSubsumption(system string) bool {
	// Check if explicitly a subsumption-supported system
	if SubsumptionSupportedSystems[system] {
		return true
	}

	// Check if explicitly a flat system (no subsumption)
	if FlatCodeSystems[system] {
		return false
	}

	// For unknown systems, check if it's a SNOMED namespace
	// (SNOMED extensions use http://snomed.info/sct/XXXXXX format)
	if strings.HasPrefix(system, "http://snomed.info/sct") {
		return true
	}

	// Default: don't attempt subsumption for unknown systems
	// This is safer - better to miss subsumption than waste 14 seconds
	return false
}

// GetPipelineName returns the pipeline type based on system
func GetPipelineName(system string) string {
	if ShouldAttemptSubsumption(system) {
		return "THREE-CHECK"
	}
	return "TWO-CHECK"
}

// ValueSetDefinition represents a value set rule stored in the database
type ValueSetDefinition struct {
	ID              string         `json:"id" db:"id"`
	Name            string         `json:"name" db:"name"`
	URL             string         `json:"url" db:"url"`
	Title           string         `json:"title" db:"title"`
	Description     string         `json:"description" db:"description"`
	Publisher       string         `json:"publisher" db:"publisher"`
	DefinitionType  DefinitionType `json:"definition_type" db:"definition_type"`
	Version         string         `json:"version" db:"version"`
	Status          string         `json:"status" db:"status"`
	ClinicalDomain  string         `json:"clinical_domain" db:"clinical_domain"`

	// For EXPLICIT definitions (fixed code list)
	ExplicitCodes []ExplicitCode `json:"explicit_codes,omitempty"`

	// For INTENSIONAL definitions (graph-based expansion)
	RootConceptCode   string        `json:"root_concept_code,omitempty" db:"root_concept_code"`
	RootConceptSystem string        `json:"root_concept_system,omitempty" db:"root_concept_system"`
	ExpansionRule     ExpansionRule `json:"expansion_rule,omitempty" db:"expansion_rule"`

	// For EXTENSIONAL definitions (composed from other value sets)
	ComposedOf []ComposedValueSet `json:"composed_of,omitempty"`

	// Metadata
	UseContext models.JSONB `json:"use_context,omitempty" db:"use_context"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at"`
	CreatedBy  string       `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy  string       `json:"updated_by,omitempty" db:"updated_by"`
}

// ExplicitCode represents a single code in an explicit value set
type ExplicitCode struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
	Version string `json:"version,omitempty"`
}

// ComposedValueSet represents a reference to another value set in an extensional definition
type ComposedValueSet struct {
	ValueSetID string `json:"valueset_id"`
	Operation  string `json:"operation"` // "include" or "exclude"
}

// ExpandedValueSet represents the result of expanding a value set
type ExpandedValueSet struct {
	Identifier    string         `json:"identifier"`
	URL           string         `json:"url"`
	Version       string         `json:"version"`
	Total         int            `json:"total"`
	Codes         []ExpandedCode `json:"codes"`
	ExpansionTime time.Time      `json:"expansion_time"`
	CachedResult  bool           `json:"cached_result"`

	// CodeIndex provides O(1) lookup for exact match checking
	// Structure: map[system]map[code]*ExpandedCode
	// This replaces O(n) linear scan with O(1) hash lookup
	CodeIndex map[string]map[string]*ExpandedCode `json:"-"`
}

// BuildIndex constructs the O(1) hash index from the Codes slice
// Must be called after populating Codes to enable fast lookup
func (e *ExpandedValueSet) BuildIndex() {
	e.CodeIndex = make(map[string]map[string]*ExpandedCode)
	for i := range e.Codes {
		code := &e.Codes[i]
		system := code.System
		if system == "" {
			system = "_default_" // Handle empty system case
		}
		if e.CodeIndex[system] == nil {
			e.CodeIndex[system] = make(map[string]*ExpandedCode)
		}
		e.CodeIndex[system][code.Code] = code
	}
}

// Contains performs O(1) lookup to check if a code exists in the value set
// Returns the matching ExpandedCode and true if found, nil and false otherwise
// If system is empty, searches across all systems in the index
func (e *ExpandedValueSet) Contains(system, code string) (*ExpandedCode, bool) {
	if e.CodeIndex == nil {
		// Fallback: index not built, build it now
		e.BuildIndex()
	}

	// If system specified, do direct lookup - O(1)
	if system != "" {
		if systemMap, ok := e.CodeIndex[system]; ok {
			if ec, found := systemMap[code]; found {
				return ec, true
			}
		}
		return nil, false
	}

	// If system is empty, search across all systems - O(systems) but typically small
	for _, systemMap := range e.CodeIndex {
		if ec, found := systemMap[code]; found {
			return ec, true
		}
	}
	return nil, false
}

// ExpandedCode represents a single code in an expanded value set
type ExpandedCode struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
	Version string `json:"version,omitempty"`
}

// MatchType describes how a code matched a value set
type MatchType string

const (
	// MatchTypeExact - Code is directly listed in the value set
	MatchTypeExact MatchType = "exact"
	// MatchTypeSubsumption - Code is a descendant (IS-A) of a code in the value set
	MatchTypeSubsumption MatchType = "subsumption"
	// MatchTypeNone - Code did not match via any method
	MatchTypeNone MatchType = "none"
)

// RuleValidationResult represents the result of validating a code against a rule-based value set
type RuleValidationResult struct {
	Valid       bool      `json:"valid"`
	ValueSetID  string    `json:"value_set_id"`
	Code        string    `json:"code"`
	System      string    `json:"system"`
	Display     string    `json:"display,omitempty"`
	Message     string    `json:"message,omitempty"`
	MatchType   MatchType `json:"match_type,omitempty"`   // How the code matched (exact, subsumption, none)
	MatchedCode string    `json:"matched_code,omitempty"` // For subsumption: the ancestor code that matched

	// PipelineDetails shows ALL THREE steps of the validation pipeline
	// This is the detailed audit trail for clinical transparency
	PipelineDetails *PipelineDetails `json:"pipeline,omitempty"`
}

// PipelineDetails provides transparency into the THREE-CHECK validation pipeline
// Shows each step: Expansion → Exact Match → Subsumption
type PipelineDetails struct {
	// Step 1: Expansion
	Step1Expansion *ExpansionStep `json:"step1_expansion"`

	// Step 2: Exact Membership Check
	Step2ExactMatch *ExactMatchStep `json:"step2_exact_match"`

	// Step 3: Subsumption (Hierarchical) Check
	Step3Subsumption *SubsumptionStep `json:"step3_subsumption"`
}

// ExpansionStep details for Step 1 of the pipeline
type ExpansionStep struct {
	Status     string `json:"status"`      // "completed", "failed", "cached"
	CodesCount int    `json:"codes_count"` // Number of codes in the expanded value set
	Cached     bool   `json:"cached"`      // Was this from cache?
	Duration   string `json:"duration"`    // How long expansion took
}

// ExactMatchStep details for Step 2 of the pipeline
type ExactMatchStep struct {
	Status      string `json:"status"`       // "match", "no_match", "skipped"
	Checked     bool   `json:"checked"`      // Was this step executed?
	MatchFound  bool   `json:"match_found"`  // Did we find an exact match?
	CheckedCode string `json:"checked_code"` // The code we were looking for
}

// SubsumptionStep details for Step 3 of the pipeline
type SubsumptionStep struct {
	Status          string `json:"status"`                     // "match", "no_match", "skipped", "disabled"
	Checked         bool   `json:"checked"`                    // Was this step executed?
	MatchFound      bool   `json:"match_found"`                // Did we find a subsumption match?
	CheckedCode     string `json:"checked_code"`               // The code we were checking
	MatchedAncestor string `json:"matched_ancestor,omitempty"` // The ancestor code that matched
	AncestorDisplay string `json:"ancestor_display,omitempty"` // Display name of ancestor
	PathLength      int    `json:"path_length,omitempty"`      // Distance in hierarchy
	Source          string `json:"source,omitempty"`           // "neo4j" or "graphdb"
	CodesChecked    int    `json:"codes_checked,omitempty"`    // How many value set codes were checked
}

// ClassificationResult represents the result of classifying a code across ALL value sets
// This is the "FindValueSetsForCode" feature - reverse lookup from code to value sets
type ClassificationResult struct {
	Code              string                   `json:"code"`
	System            string                   `json:"system"`
	MatchingValueSets []ValueSetMatch          `json:"matching_value_sets"`
	TotalValueSets    int                      `json:"total_value_sets_checked"`
	MatchCount        int                      `json:"match_count"`
	ProcessingTime    string                   `json:"processing_time"`
	Message           string                   `json:"message,omitempty"`
}

// ValueSetMatch represents a single value set that matched a code
type ValueSetMatch struct {
	ValueSetID      string    `json:"value_set_id"`
	ValueSetName    string    `json:"value_set_name"`
	ValueSetURL     string    `json:"value_set_url"`
	ClinicalDomain  string    `json:"clinical_domain,omitempty"`
	MatchType       MatchType `json:"match_type"`        // "exact" or "subsumption"
	MatchedCode     string    `json:"matched_code,omitempty"` // For subsumption: the ancestor code
	MatchedDisplay  string    `json:"matched_display,omitempty"`
}

// ValueSetFilter provides filtering options for listing value sets
type ValueSetFilter struct {
	Status         string `json:"status,omitempty"`
	ClinicalDomain string `json:"clinical_domain,omitempty"`
	Publisher      string `json:"publisher,omitempty"`
	NameContains   string `json:"name_contains,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	Offset         int    `json:"offset,omitempty"`
}

// ============================================================================
// RuleManager Implementation
// ============================================================================

// ruleManagerImpl implements the RuleManager interface
type ruleManagerImpl struct {
	db             *sql.DB
	cache          *cache.RedisClient
	graphDBClient  *semantic.GraphDBClient
	subsumptionSvc *SubsumptionService // Fallback for three-check pipeline (GraphDB)
	neo4jBridge    *Neo4jBridge        // Primary for three-check pipeline (Neo4j - faster)
	logger         *logrus.Logger
	metrics        *metrics.Collector

	// Cache settings
	explicitCacheTTL    time.Duration // 7 days for explicit (rarely change)
	intensionalCacheTTL time.Duration // 24 hours for intensional (graph may update)
	extensionalCacheTTL time.Duration // 24 hours for extensional (composed sets)

	// Feature flags
	enableSubsumptionCheck bool // Enable subsumption in validation pipeline
}

// NewRuleManager creates a new RuleManager instance
// The subsumptionSvc/neo4jBridge parameters enable the three-check validation pipeline:
// 1. Expansion - Get all codes in value set
// 2. Exact Match - Check if code is directly in the expanded list
// 3. Subsumption - Check if code IS-A (descendant of) any code in the value set
// Priority: Neo4jBridge (fast ELK hierarchy) > SubsumptionService (GraphDB OWL reasoning)
func NewRuleManager(
	db *sql.DB,
	cache *cache.RedisClient,
	graphDBClient *semantic.GraphDBClient,
	subsumptionSvc *SubsumptionService, // Fallback: GraphDB OWL reasoning
	neo4jBridge *Neo4jBridge,           // Primary: Neo4j ELK materialized hierarchy
	logger *logrus.Logger,
	metrics *metrics.Collector,
) RuleManager {
	// Enable subsumption if either Neo4j or GraphDB is available
	enableSubsumption := neo4jBridge != nil || subsumptionSvc != nil
	if neo4jBridge != nil {
		logger.Info("RuleManager: Subsumption checking ENABLED via Neo4jBridge (fast ELK hierarchy)")
	} else if subsumptionSvc != nil {
		logger.Info("RuleManager: Subsumption checking ENABLED via SubsumptionService (GraphDB fallback)")
	} else {
		logger.Warn("RuleManager: Subsumption checking DISABLED (no Neo4j or GraphDB available)")
	}

	return &ruleManagerImpl{
		db:                     db,
		cache:                  cache,
		graphDBClient:          graphDBClient,
		subsumptionSvc:         subsumptionSvc,
		neo4jBridge:            neo4jBridge,
		logger:                 logger,
		metrics:                metrics,
		explicitCacheTTL:       7 * 24 * time.Hour,  // 7 days
		intensionalCacheTTL:    24 * time.Hour,      // 24 hours
		extensionalCacheTTL:    24 * time.Hour,      // 24 hours
		enableSubsumptionCheck: enableSubsumption,
	}
}

// ExpandValueSet expands a value set and returns all codes
func (r *ruleManagerImpl) ExpandValueSet(ctx context.Context, identifier string, version string) (*ExpandedValueSet, error) {
	start := time.Now()

	// Step 1: Check Redis cache first
	cacheKey := r.buildCacheKey(identifier, version)
	var cachedResult ExpandedValueSet
	if err := r.cache.Get(cacheKey, &cachedResult); err == nil {
		cachedResult.CachedResult = true
		cachedResult.BuildIndex() // Build O(1) hash index after cache retrieval
		r.metrics.RecordCacheHit("rule_manager", "valueset_expansion")
		r.logger.WithFields(logrus.Fields{
			"identifier": identifier,
			"version":    version,
			"total":      cachedResult.Total,
			"cache_hit":  true,
			"duration":   time.Since(start),
		}).Debug("Value set expansion returned from cache")
		return &cachedResult, nil
	}
	r.metrics.RecordCacheMiss("rule_manager", "valueset_expansion")

	// Step 2: Get the value set definition from PostgreSQL
	definition, err := r.GetValueSetDefinition(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get value set definition: %w", err)
	}

	// Step 3: Expand based on definition type
	var codes []ExpandedCode
	switch definition.DefinitionType {
	case DefinitionTypeExplicit:
		codes = r.expandExplicit(definition)
	case DefinitionTypeIntensional:
		codes, err = r.expandIntensional(ctx, definition)
		if err != nil {
			return nil, fmt.Errorf("failed to expand intensional value set: %w", err)
		}
	case DefinitionTypeExtensional:
		codes, err = r.expandExtensional(ctx, definition)
		if err != nil {
			return nil, fmt.Errorf("failed to expand extensional value set: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown definition type: %s", definition.DefinitionType)
	}

	// Step 4: Build result
	result := &ExpandedValueSet{
		Identifier:    definition.Name,
		URL:           definition.URL,
		Version:       definition.Version,
		Total:         len(codes),
		Codes:         codes,
		ExpansionTime: time.Now(),
		CachedResult:  false,
	}

	// Step 4b: Build O(1) hash index for fast exact match lookup
	result.BuildIndex()

	// Step 5: Cache the result
	cacheTTL := r.getCacheTTL(definition.DefinitionType)
	if err := r.cache.Set(cacheKey, result, cacheTTL); err != nil {
		r.logger.WithError(err).Warn("Failed to cache value set expansion")
	}

	r.logger.WithFields(logrus.Fields{
		"identifier":      identifier,
		"version":         version,
		"definition_type": definition.DefinitionType,
		"total":           result.Total,
		"cache_hit":       false,
		"duration":        time.Since(start),
	}).Info("Value set expanded")

	return result, nil
}

// ValidateCodeInValueSet implements the three-check validation pipeline:
//
// ┌─────────────────────────────────────────────────────────────────────────┐
// │                    RULE ENGINE BRIDGE - THREE-CHECK PIPELINE            │
// ├─────────────────────────────────────────────────────────────────────────┤
// │  STEP 1: EXPANSION                                                      │
// │  → Get all codes in the value set (explicit, intensional, extensional)  │
// │                                                                         │
// │  STEP 2: EXACT MEMBERSHIP CHECK (fast path)                             │
// │  → Is the input code directly listed in the expanded set?               │
// │  → Result: MatchType="exact" if found                                   │
// │                                                                         │
// │  STEP 3: SUBSUMPTION CHECK (hierarchical matching)                      │
// │  → For each code in the value set, check if input IS-A that code        │
// │  → Uses SNOMED CT hierarchy via GraphDB/Neo4j                           │
// │  → Result: MatchType="subsumption" if input is descendant               │
// └─────────────────────────────────────────────────────────────────────────┘
//
// Clinical Safety: ALL THREE CHECKS MUST RUN for proper SNOMED validation.
// A code like "Bacterial Sepsis" (10001005) is valid for a "Sepsis" value set
// even if not explicitly listed, because it IS-A descendant of Sepsis.
func (r *ruleManagerImpl) ValidateCodeInValueSet(ctx context.Context, code, system, valueSetID string) (*RuleValidationResult, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordValidation(system, "valueset_validation", time.Since(start))
	}()

	r.logger.WithFields(logrus.Fields{
		"code":       code,
		"system":     system,
		"valueSetID": valueSetID,
	}).Debug("Starting three-check validation pipeline")

	// Initialize pipeline details for transparency
	pipeline := &PipelineDetails{
		Step1Expansion: &ExpansionStep{
			Status: "pending",
		},
		Step2ExactMatch: &ExactMatchStep{
			Status:      "pending",
			CheckedCode: code,
		},
		Step3Subsumption: &SubsumptionStep{
			Status:      "pending",
			CheckedCode: code,
		},
	}

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 1: EXPANSION - Get all codes in the value set
	// ═══════════════════════════════════════════════════════════════════════
	expansionStart := time.Now()
	expanded, err := r.ExpandValueSet(ctx, valueSetID, "")
	if err != nil {
		pipeline.Step1Expansion.Status = "failed"
		pipeline.Step1Expansion.Duration = time.Since(expansionStart).String()
		pipeline.Step2ExactMatch.Status = "skipped"
		pipeline.Step2ExactMatch.Checked = false
		pipeline.Step3Subsumption.Status = "skipped"
		pipeline.Step3Subsumption.Checked = false

		return &RuleValidationResult{
			Valid:           false,
			ValueSetID:      valueSetID,
			Code:            code,
			System:          system,
			MatchType:       MatchTypeNone,
			Message:         fmt.Sprintf("Failed to expand value set: %v", err),
			PipelineDetails: pipeline,
		}, nil
	}

	// Update Step 1 details
	pipeline.Step1Expansion.Status = "completed"
	pipeline.Step1Expansion.CodesCount = expanded.Total
	pipeline.Step1Expansion.Cached = expanded.CachedResult
	pipeline.Step1Expansion.Duration = time.Since(expansionStart).String()

	r.logger.WithFields(logrus.Fields{
		"valueSetID": valueSetID,
		"codeCount":  expanded.Total,
	}).Debug("Step 1 complete: Value set expanded")

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 2: EXACT MEMBERSHIP CHECK (O(1) hash lookup - optimized from O(n) loop)
	// ═══════════════════════════════════════════════════════════════════════
	pipeline.Step2ExactMatch.Checked = true

	// Use O(1) hash-based lookup instead of O(n) linear scan
	if ec, found := expanded.Contains(system, code); found {
		r.logger.WithFields(logrus.Fields{
			"code":      code,
			"matchType": "exact",
			"lookup":    "O(1)_hash",
		}).Debug("Step 2 complete: Exact match found via O(1) hash lookup")

		// Update pipeline details for exact match
		pipeline.Step2ExactMatch.Status = "match"
		pipeline.Step2ExactMatch.MatchFound = true
		pipeline.Step3Subsumption.Status = "skipped"
		pipeline.Step3Subsumption.Checked = false

		return &RuleValidationResult{
			Valid:           true,
			ValueSetID:      valueSetID,
			Code:            code,
			System:          ec.System,
			Display:         ec.Display,
			MatchType:       MatchTypeExact,
			MatchedCode:     code,
			Message:         "Code found in value set via exact membership match (O(1) hash lookup)",
			PipelineDetails: pipeline,
		}, nil
	}

	// No exact match found
	pipeline.Step2ExactMatch.Status = "no_match"
	pipeline.Step2ExactMatch.MatchFound = false

	r.logger.WithField("code", code).Debug("Step 2: No exact match, proceeding to subsumption check")

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 3: SUBSUMPTION CHECK (hierarchical matching)
	// For each code in the value set, check if input code IS-A that code
	// Priority: Neo4jBridge (fast ELK hierarchy) > SubsumptionService (GraphDB)
	// ═══════════════════════════════════════════════════════════════════════
	if r.enableSubsumptionCheck {
		pipeline.Step3Subsumption.Checked = true

		// Determine the system to use for subsumption
		subsumptionSystem := system
		if subsumptionSystem == "" {
			subsumptionSystem = "http://snomed.info/sct" // Default to SNOMED CT
		}

		// PERFORMANCE OPTIMIZATION: Skip subsumption for flat code systems
		// Use the ShouldAttemptSubsumption helper to check if this system has IS-A hierarchies.
		// LOINC, ICD-10, RxNorm etc. are flat - no IS-A relationships.
		// Only SNOMED-based systems benefit from subsumption checking.
		// Skipping this saves ~14 seconds per non-SNOMED code validation.
		if !ShouldAttemptSubsumption(subsumptionSystem) {
			r.logger.WithFields(logrus.Fields{
				"code":     code,
				"system":   subsumptionSystem,
				"pipeline": GetPipelineName(subsumptionSystem),
			}).Debug("Step 3: Skipping subsumption for flat code system (no IS-A hierarchy)")
			pipeline.Step3Subsumption.Status = "skipped"
			pipeline.Step3Subsumption.Checked = false
			return &RuleValidationResult{
				Valid:           false,
				ValueSetID:      valueSetID,
				Code:            code,
				System:          system,
				MatchType:       MatchTypeNone,
				Message:         fmt.Sprintf("Code '%s' not found in value set '%s' (%s pipeline - exact match only)", code, valueSetID, GetPipelineName(subsumptionSystem)),
				PipelineDetails: pipeline,
			}, nil
		}

		codesChecked := 0
		for _, ec := range expanded.Codes {
			// Skip if systems don't match (only check SNOMED against SNOMED, etc.)
			if ec.System != "" && subsumptionSystem != "" && ec.System != subsumptionSystem {
				continue
			}

			codesChecked++

			var subsumes bool
			var pathLength int
			var subsumptionErr error
			var source string

			// Priority 1: Use Neo4jBridge if available (fast ELK materialized hierarchy)
			if r.neo4jBridge != nil && r.neo4jBridge.IsNeo4jAvailable() {
				source = "neo4j"
				result, err := r.neo4jBridge.TestSubsumption(ctx, code, ec.Code, subsumptionSystem)
				if err != nil {
					r.logger.WithError(err).WithFields(logrus.Fields{
						"code":         code,
						"valueSetCode": ec.Code,
						"system":       subsumptionSystem,
						"source":       "neo4j",
					}).Debug("Neo4j subsumption check failed, trying GraphDB fallback")
					subsumptionErr = err
				} else {
					subsumes = result.Subsumes
					pathLength = result.PathLength
				}
			}

			// Priority 2: Fallback to SubsumptionService (GraphDB OWL reasoning)
			if !subsumes && subsumptionErr != nil && r.subsumptionSvc != nil {
				source = "graphdb"
				subsumptionReq := &models.SubsumptionRequest{
					CodeA:  code,
					CodeB:  ec.Code,
					System: subsumptionSystem,
				}
				result, err := r.subsumptionSvc.TestSubsumption(ctx, subsumptionReq)
				if err != nil {
					r.logger.WithError(err).WithFields(logrus.Fields{
						"code":         code,
						"valueSetCode": ec.Code,
						"system":       subsumptionSystem,
						"source":       "graphdb",
					}).Warn("Subsumption check failed on both Neo4j and GraphDB")
					continue
				}
				subsumes = result.Subsumes
				pathLength = result.PathLength
			}

			if subsumes {
				r.logger.WithFields(logrus.Fields{
					"code":         code,
					"ancestorCode": ec.Code,
					"matchType":    "subsumption",
					"pathLength":   pathLength,
				}).Debug("Step 3 complete: Subsumption match found")

				// Update pipeline details for subsumption match
				pipeline.Step3Subsumption.Status = "match"
				pipeline.Step3Subsumption.MatchFound = true
				pipeline.Step3Subsumption.MatchedAncestor = ec.Code
				pipeline.Step3Subsumption.AncestorDisplay = ec.Display
				pipeline.Step3Subsumption.PathLength = pathLength
				pipeline.Step3Subsumption.Source = source
				pipeline.Step3Subsumption.CodesChecked = codesChecked

				return &RuleValidationResult{
					Valid:       true,
					ValueSetID:  valueSetID,
					Code:        code,
					System:      subsumptionSystem,
					MatchType:   MatchTypeSubsumption,
					MatchedCode: ec.Code, // The ancestor code in the value set
					Message: fmt.Sprintf(
						"Code '%s' is valid via subsumption: IS-A '%s' (%s) with path length %d",
						code, ec.Code, ec.Display, pathLength,
					),
					PipelineDetails: pipeline,
				}, nil
			}
		}

		// No subsumption match found
		pipeline.Step3Subsumption.Status = "no_match"
		pipeline.Step3Subsumption.MatchFound = false
		pipeline.Step3Subsumption.CodesChecked = codesChecked
		if r.neo4jBridge != nil && r.neo4jBridge.IsNeo4jAvailable() {
			pipeline.Step3Subsumption.Source = "neo4j"
		} else if r.subsumptionSvc != nil {
			pipeline.Step3Subsumption.Source = "graphdb"
		}

		r.logger.WithField("code", code).Debug("Step 3: No subsumption match found")
	} else {
		// Subsumption disabled
		pipeline.Step3Subsumption.Status = "disabled"
		pipeline.Step3Subsumption.Checked = false
		r.logger.WithField("code", code).Debug("Step 3: Subsumption check skipped (disabled or service unavailable)")
	}

	// ═══════════════════════════════════════════════════════════════════════
	// ALL THREE CHECKS FAILED - Code is not valid for this value set
	// ═══════════════════════════════════════════════════════════════════════
	return &RuleValidationResult{
		Valid:      false,
		ValueSetID: valueSetID,
		Code:       code,
		System:     system,
		MatchType:  MatchTypeNone,
		Message: fmt.Sprintf(
			"Code '%s' not found in value set '%s' via membership or subsumption (checked %d codes)",
			code, valueSetID, expanded.Total,
		),
		PipelineDetails: pipeline,
	}, nil
}

// GetValueSetDefinition retrieves a value set definition from the database
func (r *ruleManagerImpl) GetValueSetDefinition(ctx context.Context, identifier string) (*ValueSetDefinition, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordDBQuery("get_valueset_definition", "success", time.Since(start))
	}()

	// Query the value_sets table (existing table from migrations)
	query := `
		SELECT id, url, version, name, title, description, status, publisher,
		       compose, expansion, clinical_domain, created_at, updated_at
		FROM value_sets
		WHERE name = $1 OR url = $1 OR id::text = $1
		ORDER BY created_at DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, identifier)

	var def ValueSetDefinition
	var composeJSON, expansionJSON []byte
	err := row.Scan(
		&def.ID, &def.URL, &def.Version, &def.Name, &def.Title, &def.Description,
		&def.Status, &def.Publisher, &composeJSON, &expansionJSON,
		&def.ClinicalDomain, &def.CreatedAt, &def.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("value set not found: %s", identifier)
		}
		r.logger.WithError(err).Error("Failed to get value set definition")
		return nil, err
	}

	// Parse the compose/expansion JSONB to determine definition type and extract codes
	if err := r.parseValueSetDefinition(&def, composeJSON, expansionJSON); err != nil {
		return nil, fmt.Errorf("failed to parse value set definition: %w", err)
	}

	return &def, nil
}

// RefreshCache invalidates and rebuilds the cache for a value set
func (r *ruleManagerImpl) RefreshCache(ctx context.Context, identifier string) error {
	// Delete the cached entry
	cacheKey := r.buildCacheKey(identifier, "")
	if err := r.cache.Delete(cacheKey); err != nil {
		r.logger.WithError(err).Warn("Failed to delete cached value set")
	}

	// Also delete versioned keys
	pattern := fmt.Sprintf("kb7:valueset:expanded:%s:*", identifier)
	if err := r.cache.DeletePattern(pattern); err != nil {
		r.logger.WithError(err).Warn("Failed to delete cached value set versions")
	}

	// Re-expand to populate cache
	_, err := r.ExpandValueSet(ctx, identifier, "")
	return err
}

// ListValueSets returns value set definitions with optional filtering
func (r *ruleManagerImpl) ListValueSets(ctx context.Context, filter ValueSetFilter) ([]ValueSetDefinition, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordDBQuery("list_valuesets", "success", time.Since(start))
	}()

	// Build query with filters
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `SELECT id, url, version, name, title, description, status, publisher,
	              clinical_domain, created_at, updated_at FROM value_sets WHERE 1=1`

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf(" AND status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.ClinicalDomain != "" {
		conditions = append(conditions, fmt.Sprintf(" AND clinical_domain = $%d", argIndex))
		args = append(args, filter.ClinicalDomain)
		argIndex++
	}

	if filter.Publisher != "" {
		conditions = append(conditions, fmt.Sprintf(" AND publisher = $%d", argIndex))
		args = append(args, filter.Publisher)
		argIndex++
	}

	if filter.NameContains != "" {
		conditions = append(conditions, fmt.Sprintf(" AND (name ILIKE $%d OR title ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filter.NameContains+"%")
		argIndex++
	}

	// Build final query
	query := baseQuery + strings.Join(conditions, "") + " ORDER BY name"

	// Add pagination
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to list value sets")
		return nil, err
	}
	defer rows.Close()

	var definitions []ValueSetDefinition
	for rows.Next() {
		var def ValueSetDefinition
		err := rows.Scan(
			&def.ID, &def.URL, &def.Version, &def.Name, &def.Title, &def.Description,
			&def.Status, &def.Publisher, &def.ClinicalDomain, &def.CreatedAt, &def.UpdatedAt,
		)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan value set row")
			continue
		}
		def.DefinitionType = DefinitionTypeExplicit // Default, will be determined on expansion
		definitions = append(definitions, def)
	}

	return definitions, nil
}

// SeedBuiltinValueSets is a NO-OP - value sets are already in PostgreSQL!
// KB-7 is the SINGLE SOURCE OF TRUTH. Value sets are managed via:
//   - Database migrations (SQL files)
//   - Admin API endpoints
//   - Direct PostgreSQL inserts
//
// This function is kept for interface compatibility but does nothing.
// The old implementation that used the valuesets Go package has been removed.
func (r *ruleManagerImpl) SeedBuiltinValueSets(ctx context.Context) error {
	// Count existing value sets in the database
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM value_sets WHERE status = 'active'").Scan(&count)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to count value sets")
		return nil // Non-fatal - just log and continue
	}

	r.logger.WithField("count", count).Info("SeedBuiltinValueSets: Value sets already in PostgreSQL - no seeding needed")
	return nil
}

// ClassifyCode finds ALL value sets that contain a given code (reverse lookup)
// This implements the missing "FindValueSetsForCode" feature from the specs.
//
// ┌─────────────────────────────────────────────────────────────────────────┐
// │                    CLASSIFY CODE - REVERSE LOOKUP                        │
// ├─────────────────────────────────────────────────────────────────────────┤
// │  INPUT:  Just {code, system} - user doesn't need to know value sets     │
// │  OUTPUT: ALL value sets that match (via exact OR subsumption)           │
// │                                                                          │
// │  OPTIMIZED APPROACH (v2.0):                                              │
// │    - Direct SQL lookup in precomputed_valueset_codes table               │
// │    - Searches ALL 17,000+ valueset_urls (not just value_sets table)      │
// │    - O(1) indexed lookup instead of O(n) value set expansion             │
// │                                                                          │
// │  Clinical Use Case:                                                      │
// │    - Clinician enters code "448417001" (Streptococcal sepsis)            │
// │    - KB7 returns ALL matching ValueSets from precomputed table           │
// │    - No need for clinician to know which value sets exist!               │
// └─────────────────────────────────────────────────────────────────────────┘
func (r *ruleManagerImpl) ClassifyCode(ctx context.Context, code, system string) (*ClassificationResult, error) {
	start := time.Now()
	defer func() {
		r.metrics.RecordValidation(system, "classify_code", time.Since(start))
	}()

	r.logger.WithFields(logrus.Fields{
		"code":   code,
		"system": system,
	}).Info("Starting code classification (direct precomputed lookup)")

	result := &ClassificationResult{
		Code:              code,
		System:            system,
		MatchingValueSets: []ValueSetMatch{},
	}

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 1: Direct lookup in precomputed_valueset_codes table
	// This searches ALL 17,000+ valueset_urls, not just 29 in value_sets table
	// ═══════════════════════════════════════════════════════════════════════
	var query string
	var args []interface{}

	if system != "" {
		// Search with specific code system
		query = `
			SELECT DISTINCT valueset_url, code, display, code_system
			FROM precomputed_valueset_codes
			WHERE code = $1 AND code_system = $2
			ORDER BY valueset_url
			LIMIT 1000`
		args = []interface{}{code, system}
	} else {
		// Search across all code systems
		query = `
			SELECT DISTINCT valueset_url, code, display, code_system
			FROM precomputed_valueset_codes
			WHERE code = $1
			ORDER BY valueset_url
			LIMIT 1000`
		args = []interface{}{code}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to query precomputed_valueset_codes")
		return nil, fmt.Errorf("failed to classify code: %w", err)
	}
	defer rows.Close()

	// Collect all matching ValueSets
	for rows.Next() {
		var valuesetURL, matchedCode, display, codeSystem string
		if err := rows.Scan(&valuesetURL, &matchedCode, &display, &codeSystem); err != nil {
			r.logger.WithError(err).Warn("Failed to scan classification row")
			continue
		}

		match := ValueSetMatch{
			ValueSetID:     valuesetURL, // Use URL as ID for precomputed entries
			ValueSetName:   valuesetURL, // Use URL as name since we don't have title
			ValueSetURL:    valuesetURL,
			MatchType:      MatchTypeExact, // Direct match in precomputed table
			MatchedCode:    matchedCode,
			MatchedDisplay: display,
		}
		result.MatchingValueSets = append(result.MatchingValueSets, match)
	}

	if err := rows.Err(); err != nil {
		r.logger.WithError(err).Error("Error iterating classification results")
	}

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 2: Get total count of unique valueset_urls for reporting
	// ═══════════════════════════════════════════════════════════════════════
	var totalValueSets int
	countQuery := `SELECT COUNT(DISTINCT valueset_url) FROM precomputed_valueset_codes`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalValueSets); err != nil {
		r.logger.WithError(err).Warn("Failed to count total valueset_urls")
		totalValueSets = 17293 // Fallback to known count
	}
	result.TotalValueSets = totalValueSets

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 3: Finalize result
	// ═══════════════════════════════════════════════════════════════════════
	result.MatchCount = len(result.MatchingValueSets)
	result.ProcessingTime = time.Since(start).String()

	if result.MatchCount == 0 {
		result.Message = fmt.Sprintf(
			"Code '%s' does not match any of the %d value sets in precomputed_valueset_codes",
			code, result.TotalValueSets,
		)
	} else {
		result.Message = fmt.Sprintf(
			"Code '%s' matches %d value sets out of %d (direct precomputed lookup)",
			code, result.MatchCount, result.TotalValueSets,
		)
	}

	r.logger.WithFields(logrus.Fields{
		"code":           code,
		"system":         system,
		"matchCount":     result.MatchCount,
		"totalValueSets": result.TotalValueSets,
		"processingTime": result.ProcessingTime,
	}).Info("Code classification complete (precomputed lookup)")

	return result, nil
}

// ============================================================================
// Private Helper Methods
// ============================================================================

// buildCacheKey creates a Redis cache key for a value set expansion
func (r *ruleManagerImpl) buildCacheKey(identifier, version string) string {
	if version == "" {
		version = "latest"
	}
	return fmt.Sprintf("kb7:valueset:expanded:%s:%s", identifier, version)
}

// getCacheTTL returns the appropriate cache TTL based on definition type
func (r *ruleManagerImpl) getCacheTTL(defType DefinitionType) time.Duration {
	switch defType {
	case DefinitionTypeExplicit:
		return r.explicitCacheTTL
	case DefinitionTypeIntensional:
		return r.intensionalCacheTTL
	case DefinitionTypeExtensional:
		return r.extensionalCacheTTL
	default:
		return r.explicitCacheTTL
	}
}

// expandExplicit returns codes from explicit definitions (direct from ExplicitCodes)
func (r *ruleManagerImpl) expandExplicit(def *ValueSetDefinition) []ExpandedCode {
	codes := make([]ExpandedCode, len(def.ExplicitCodes))
	for i, ec := range def.ExplicitCodes {
		codes[i] = ExpandedCode{
			System:  ec.System,
			Code:    ec.Code,
			Display: ec.Display,
			Version: ec.Version,
		}
	}
	return codes
}

// expandIntensional expands a value set using GraphDB SPARQL queries
func (r *ruleManagerImpl) expandIntensional(ctx context.Context, def *ValueSetDefinition) ([]ExpandedCode, error) {
	if r.graphDBClient == nil {
		return nil, fmt.Errorf("GraphDB client not configured for intensional expansion")
	}

	// Build SPARQL query based on expansion rule
	var sparqlQuery string
	switch def.ExpansionRule {
	case ExpansionRuleDescendants:
		sparqlQuery = fmt.Sprintf(`
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
			PREFIX snomed: <http://snomed.info/id/>

			SELECT ?code ?display WHERE {
				?concept rdfs:subClassOf+ snomed:%s .
				?concept skos:notation ?code .
				?concept skos:prefLabel ?display .
			}
			LIMIT 10000
		`, def.RootConceptCode)

	case ExpansionRuleDescendantsOrSelf:
		sparqlQuery = fmt.Sprintf(`
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
			PREFIX snomed: <http://snomed.info/id/>

			SELECT ?code ?display WHERE {
				{
					BIND(snomed:%s AS ?concept)
				} UNION {
					?concept rdfs:subClassOf+ snomed:%s .
				}
				?concept skos:notation ?code .
				?concept skos:prefLabel ?display .
			}
			LIMIT 10000
		`, def.RootConceptCode, def.RootConceptCode)

	case ExpansionRuleAncestors:
		sparqlQuery = fmt.Sprintf(`
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
			PREFIX snomed: <http://snomed.info/id/>

			SELECT ?code ?display WHERE {
				snomed:%s rdfs:subClassOf+ ?concept .
				?concept skos:notation ?code .
				?concept skos:prefLabel ?display .
			}
			LIMIT 1000
		`, def.RootConceptCode)

	case ExpansionRuleAncestorsOrSelf:
		sparqlQuery = fmt.Sprintf(`
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
			PREFIX snomed: <http://snomed.info/id/>

			SELECT ?code ?display WHERE {
				{
					BIND(snomed:%s AS ?concept)
				} UNION {
					snomed:%s rdfs:subClassOf+ ?concept .
				}
				?concept skos:notation ?code .
				?concept skos:prefLabel ?display .
			}
			LIMIT 1000
		`, def.RootConceptCode, def.RootConceptCode)

	default:
		return nil, fmt.Errorf("unknown expansion rule: %s", def.ExpansionRule)
	}

	// Execute SPARQL query
	sparqlRequest := &semantic.SPARQLQuery{
		Query: sparqlQuery,
	}
	results, err := r.graphDBClient.ExecuteSPARQL(ctx, sparqlRequest)
	if err != nil {
		return nil, fmt.Errorf("SPARQL query failed: %w", err)
	}

	// Convert results to ExpandedCode
	var codes []ExpandedCode
	for _, binding := range results.Results.Bindings {
		codes = append(codes, ExpandedCode{
			System:  def.RootConceptSystem,
			Code:    binding["code"].Value,
			Display: binding["display"].Value,
		})
	}

	return codes, nil
}

// expandExtensional expands a value set by composing other value sets
func (r *ruleManagerImpl) expandExtensional(ctx context.Context, def *ValueSetDefinition) ([]ExpandedCode, error) {
	codeMap := make(map[string]ExpandedCode) // Use map to handle duplicates

	for _, composed := range def.ComposedOf {
		// Recursively expand the referenced value set
		expanded, err := r.ExpandValueSet(ctx, composed.ValueSetID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to expand composed value set %s: %w", composed.ValueSetID, err)
		}

		switch composed.Operation {
		case "include":
			for _, code := range expanded.Codes {
				key := fmt.Sprintf("%s|%s", code.System, code.Code)
				codeMap[key] = code
			}
		case "exclude":
			for _, code := range expanded.Codes {
				key := fmt.Sprintf("%s|%s", code.System, code.Code)
				delete(codeMap, key)
			}
		}
	}

	// Convert map back to slice
	codes := make([]ExpandedCode, 0, len(codeMap))
	for _, code := range codeMap {
		codes = append(codes, code)
	}

	return codes, nil
}

// parseValueSetDefinition parses JSONB columns to populate the definition struct
func (r *ruleManagerImpl) parseValueSetDefinition(def *ValueSetDefinition, composeJSON, expansionJSON []byte) error {
	// Try to parse expansion first (pre-computed codes)
	if len(expansionJSON) > 0 {
		var expansion struct {
			Contains []ExplicitCode `json:"contains"`
		}
		if err := json.Unmarshal(expansionJSON, &expansion); err == nil && len(expansion.Contains) > 0 {
			def.DefinitionType = DefinitionTypeExplicit
			def.ExplicitCodes = expansion.Contains
			return nil
		}
	}

	// Try to parse compose to determine definition type
	if len(composeJSON) > 0 {
		var compose struct {
			Include []struct {
				System  string `json:"system"`
				Concept []struct {
					Code    string `json:"code"`
					Display string `json:"display"`
				} `json:"concept"`
				Filter []struct {
					Property string `json:"property"`
					Op       string `json:"op"`
					Value    string `json:"value"`
				} `json:"filter"`
				ValueSet []string `json:"valueSet"`
			} `json:"include"`
		}

		if err := json.Unmarshal(composeJSON, &compose); err == nil {
			for _, inc := range compose.Include {
				// Check if it's intensional (has filters)
				if len(inc.Filter) > 0 {
					def.DefinitionType = DefinitionTypeIntensional
					def.RootConceptSystem = inc.System
					for _, f := range inc.Filter {
						if f.Property == "concept" && (f.Op == "is-a" || f.Op == "descendent-of") {
							def.RootConceptCode = f.Value
							def.ExpansionRule = ExpansionRuleDescendantsOrSelf
							break
						}
					}
					return nil
				}

				// Check if it's extensional (references other value sets)
				if len(inc.ValueSet) > 0 {
					def.DefinitionType = DefinitionTypeExtensional
					for _, vsRef := range inc.ValueSet {
						def.ComposedOf = append(def.ComposedOf, ComposedValueSet{
							ValueSetID: vsRef,
							Operation:  "include",
						})
					}
					return nil
				}

				// Otherwise it's explicit (has concept list)
				if len(inc.Concept) > 0 {
					def.DefinitionType = DefinitionTypeExplicit
					for _, c := range inc.Concept {
						def.ExplicitCodes = append(def.ExplicitCodes, ExplicitCode{
							System:  inc.System,
							Code:    c.Code,
							Display: c.Display,
						})
					}
				}
			}
		}
	}

	// Default to explicit if no codes found
	if def.DefinitionType == "" {
		def.DefinitionType = DefinitionTypeExplicit
	}

	return nil
}

// NOTE: getBuiltinValueSetDefinitions() has been REMOVED
// Value sets are now managed through the single source of truth: valuesets.GetBuiltinValueSets()
// This eliminates duplicate hardcoded value sets and ensures consistency across the codebase.
// See SeedBuiltinValueSets() above which now uses the valuesets package directly.

// REMOVED ~285 lines of hardcoded value set definitions (18 FHIR R4 value sets)
// The following value sets are now sourced from internal/valuesets/builtin_valuesets.go:
// - administrative-gender, marital-status, contact-point-system, contact-point-use
// - address-use, address-type, identifier-use, name-use
// - observation-status, vital-signs, condition-clinical-status, condition-verification-status
// - medication-request-status, allergy-intolerance-clinical-status, allergy-intolerance-criticality
// - encounter-status, procedure-status, diagnostic-report-status
// Plus 6 AU Clinical value sets (sepsis/renal protocols)

// IMPORTANT: To add new value sets, update internal/valuesets/builtin_valuesets.go only.
// The SeedBuiltinValueSets() function will automatically pick them up.
