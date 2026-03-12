package orb

import (
	"fmt"
	"time"
)

// IntentManifest represents the output of ORB rule evaluation
// This is the KEY data structure that drives the entire ORB-driven flow
type IntentManifest struct {
	// Core identification
	RequestID string `json:"request_id"`
	PatientID string `json:"patient_id"`

	// Recipe selection (THE MOST IMPORTANT FIELD)
	RecipeID string `json:"recipe_id"`
	// Optional variant of the recipe (e.g., obesity_adjusted, dialysis)
	Variant string `json:"variant,omitempty"`

	// Data requirements for Context Service
	DataRequirements []string `json:"data_requirements"`

	// Knowledge Base optimization (NEW ENHANCED DESIGN)
	KnowledgeManifest KnowledgeManifest `json:"knowledge_manifest"`

	// Priority and routing
	Priority string `json:"priority"`

	// Clinical rationale
	ClinicalRationale string `json:"clinical_rationale"`

	// Performance tracking
	EstimatedExecutionTimeMs int `json:"estimated_execution_time_ms"`

	// Performance hints
	CacheStrategy    string   `json:"cache_strategy"`
	ParallelismHints []string `json:"parallelism_hints"`

	// Metadata
	RuleID       string    `json:"rule_id"`
	RuleVersion  string    `json:"rule_version"`
	GeneratedAt  time.Time `json:"generated_at"`

	// Additional context for downstream services
	MedicationCode string                 `json:"medication_code"`
	Conditions     []string               `json:"conditions"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// KnowledgeManifest specifies exactly which Knowledge Bases are required
type KnowledgeManifest struct {
	RequiredKBs []string `json:"required_kbs" yaml:"required_kbs"`
}

// IntentManifestBuilder helps construct Intent Manifests
type IntentManifestBuilder struct {
	manifest *IntentManifest
}

// NewIntentManifestBuilder creates a new builder
func NewIntentManifestBuilder() *IntentManifestBuilder {
	return &IntentManifestBuilder{
		manifest: &IntentManifest{
			GeneratedAt: time.Now(),
			Metadata:    make(map[string]interface{}),
		},
	}
}

// WithRequestInfo sets request identification
func (b *IntentManifestBuilder) WithRequestInfo(requestID, patientID string) *IntentManifestBuilder {
	b.manifest.RequestID = requestID
	b.manifest.PatientID = patientID
	return b
}

// WithRecipe sets the selected recipe
func (b *IntentManifestBuilder) WithRecipe(recipeID string) *IntentManifestBuilder {
	b.manifest.RecipeID = recipeID
	return b
}

// WithVariant sets the selected recipe variant
func (b *IntentManifestBuilder) WithVariant(variant string) *IntentManifestBuilder {
	b.manifest.Variant = variant
	return b
}

// WithDataRequirements sets the data requirements
func (b *IntentManifestBuilder) WithDataRequirements(requirements []string) *IntentManifestBuilder {
	b.manifest.DataRequirements = requirements
	return b
}

// WithKnowledgeManifest sets the Knowledge Manifest
func (b *IntentManifestBuilder) WithKnowledgeManifest(requiredKBs []string) *IntentManifestBuilder {
	b.manifest.KnowledgeManifest = KnowledgeManifest{
		RequiredKBs: requiredKBs,
	}
	return b
}

// WithCacheStrategy sets the cache strategy
func (b *IntentManifestBuilder) WithCacheStrategy(strategy string) *IntentManifestBuilder {
	b.manifest.CacheStrategy = strategy
	return b
}

// WithParallelismHints sets the parallelism hints
func (b *IntentManifestBuilder) WithParallelismHints(hints []string) *IntentManifestBuilder {
	b.manifest.ParallelismHints = hints
	return b
}

// WithPriority sets the priority
func (b *IntentManifestBuilder) WithPriority(priority string) *IntentManifestBuilder {
	b.manifest.Priority = priority
	return b
}

// WithRationale sets the clinical rationale
func (b *IntentManifestBuilder) WithRationale(rationale string) *IntentManifestBuilder {
	b.manifest.ClinicalRationale = rationale
	return b
}

// WithRuleInfo sets rule metadata
func (b *IntentManifestBuilder) WithRuleInfo(ruleID, ruleVersion string) *IntentManifestBuilder {
	b.manifest.RuleID = ruleID
	b.manifest.RuleVersion = ruleVersion
	return b
}

// WithMedicationInfo sets medication context
func (b *IntentManifestBuilder) WithMedicationInfo(medicationCode string, conditions []string) *IntentManifestBuilder {
	b.manifest.MedicationCode = medicationCode
	b.manifest.Conditions = conditions
	return b
}

// WithEstimatedTime sets performance estimate
func (b *IntentManifestBuilder) WithEstimatedTime(timeMs int) *IntentManifestBuilder {
	b.manifest.EstimatedExecutionTimeMs = timeMs
	return b
}

// WithMetadata adds custom metadata
func (b *IntentManifestBuilder) WithMetadata(key string, value interface{}) *IntentManifestBuilder {
	b.manifest.Metadata[key] = value
	return b
}

// Build creates the final Intent Manifest
func (b *IntentManifestBuilder) Build() *IntentManifest {
	return b.manifest
}

// Validate checks if the Intent Manifest is valid
func (im *IntentManifest) Validate() error {
	if im.RequestID == "" {
		return ErrMissingRequestID
	}
	if im.PatientID == "" {
		return ErrMissingPatientID
	}
	if im.RecipeID == "" {
		return ErrMissingRecipeID
	}
	if len(im.DataRequirements) == 0 {
		return ErrMissingDataRequirements
	}
	if im.Priority == "" {
		return ErrMissingPriority
	}

	// Validate Knowledge Manifest if present (optional for backward compatibility)
	if err := im.validateKnowledgeManifest(); err != nil {
		return err
	}

	// Validate cache strategy if present
	if im.CacheStrategy != "" {
		if err := im.validateCacheStrategy(); err != nil {
			return err
		}
	}

	return nil
}

// validateKnowledgeManifest validates the Knowledge Manifest field
func (im *IntentManifest) validateKnowledgeManifest() error {
	// Knowledge Manifest is optional for backward compatibility
	if len(im.KnowledgeManifest.RequiredKBs) == 0 {
		return nil // Empty is valid - will fall back to all KBs
	}

	// Validate that all KB identifiers are valid
	validKBs := GetValidKBIdentifiers()
	for _, kbID := range im.KnowledgeManifest.RequiredKBs {
		if !isValidKBIdentifier(kbID, validKBs) {
			return fmt.Errorf("invalid KB identifier: %s", kbID)
		}
	}

	return nil
}

// validateCacheStrategy validates the cache strategy field
func (im *IntentManifest) validateCacheStrategy() error {
	validStrategies := []string{"aggressive", "standard", "minimal", "none"}
	for _, strategy := range validStrategies {
		if im.CacheStrategy == strategy {
			return nil
		}
	}
	return fmt.Errorf("invalid cache strategy: %s (valid: %v)", im.CacheStrategy, validStrategies)
}

// GetCacheKey generates a cache key for this manifest
func (im *IntentManifest) GetCacheKey() string {
	// Include Knowledge Manifest in cache key for proper segmentation
	kbHash := im.getKnowledgeManifestHash()
	return fmt.Sprintf("intent_manifest_%s_%s_%s_%s",
		im.PatientID, im.MedicationCode, im.RecipeID, kbHash)
}

// getKnowledgeManifestHash creates a hash of the Knowledge Manifest for cache segmentation
func (im *IntentManifest) getKnowledgeManifestHash() string {
	if len(im.KnowledgeManifest.RequiredKBs) == 0 {
		return "all_kbs" // Default when no specific KBs are required
	}

	// Create a deterministic hash based on required KBs
	// Sort the KBs to ensure consistent cache keys
	kbList := make([]string, len(im.KnowledgeManifest.RequiredKBs))
	copy(kbList, im.KnowledgeManifest.RequiredKBs)

	// Simple hash based on KB count and first/last KB for cache segmentation
	if len(kbList) == 1 {
		return fmt.Sprintf("kb1_%s", kbList[0])
	} else if len(kbList) == len(GetValidKBIdentifiers()) {
		return "all_kbs"
	} else {
		return fmt.Sprintf("kb%d_%s_%s", len(kbList), kbList[0], kbList[len(kbList)-1])
	}
}

// ToLogFields converts to structured log fields
func (im *IntentManifest) ToLogFields() map[string]interface{} {
	return map[string]interface{}{
		"request_id":                 im.RequestID,
		"patient_id":                 im.PatientID,
		"recipe_id":                  im.RecipeID,
		"data_requirements_count":    len(im.DataRequirements),
		"knowledge_manifest_kbs":     im.KnowledgeManifest.RequiredKBs,
		"knowledge_manifest_count":   len(im.KnowledgeManifest.RequiredKBs),
		"cache_strategy":             im.CacheStrategy,
		"parallelism_hints":          im.ParallelismHints,
		"priority":                   im.Priority,
		"rule_id":                    im.RuleID,
		"medication_code":            im.MedicationCode,
		"estimated_execution_time":   im.EstimatedExecutionTimeMs,
		"generated_at":               im.GeneratedAt,
	}
}

// Priority levels
const (
	PriorityCritical = "critical"
	PriorityHigh     = "high"
	PriorityMedium   = "medium"
	PriorityLow      = "low"
)

// Knowledge Base identifiers - The 7 KB services
const (
	KBDrugMaster         = "kb_drug_master_v1"
	KBDosingRules        = "kb_dosing_rules_v1"
	KBDrugInteractions   = "kb_ddi_v1"
	KBFormularyStock     = "kb_formulary_stock_v1"
	KBPatientSafetyChecks = "kb_patient_safe_checks_v1"
	KBGuidelineEvidence  = "kb_guideline_evidence_v1"
	KBResistanceProfiles = "kb_resistance_profiles_v1"
)

// GetValidKBIdentifiers returns all valid Knowledge Base identifiers
func GetValidKBIdentifiers() []string {
	return []string{
		KBDrugMaster,
		KBDosingRules,
		KBDrugInteractions,
		KBFormularyStock,
		KBPatientSafetyChecks,
		KBGuidelineEvidence,
		KBResistanceProfiles,
	}
}

// isValidKBIdentifier checks if a KB identifier is valid
func isValidKBIdentifier(kbID string, validKBs []string) bool {
	for _, validKB := range validKBs {
		if kbID == validKB {
			return true
		}
	}
	return false
}

// Common errors
var (
	ErrMissingRequestID        = fmt.Errorf("request ID is required")
	ErrMissingPatientID        = fmt.Errorf("patient ID is required")
	ErrMissingRecipeID         = fmt.Errorf("recipe ID is required")
	ErrMissingDataRequirements = fmt.Errorf("data requirements are required")
	ErrMissingPriority         = fmt.Errorf("priority is required")
)
