// Package extraction provides the FactExtractor interface and supporting types.
// All extractors (LLM, API, ETL) implement this universal interface, making
// intelligence sources completely pluggable while maintaining consistent output.
//
// DESIGN PRINCIPLE: "Freeze meaning. Fluidly replace intelligence."
// The interface is frozen; implementations can evolve freely.
package extraction

import (
	"context"
	"time"

	"github.com/cardiofit/shared/evidence"
	"github.com/cardiofit/shared/factstore"
)

// =============================================================================
// EXTRACTOR INTERFACE
// =============================================================================

// FactExtractor is the universal interface for all fact extraction strategies.
// Whether extracting via LLM, structured API, or ETL pipeline, all extractors
// implement this interface for consistent handling by the Evidence Router.
type FactExtractor interface {
	// ─────────────────────────────────────────────────────────────────────────
	// IDENTITY
	// ─────────────────────────────────────────────────────────────────────────

	// Name returns the unique identifier for this extractor
	// Examples: "gpt4-spl-extractor", "rxnav-api-extractor", "cms-etl-extractor"
	Name() string

	// Version returns the extractor version for provenance tracking
	Version() string

	// ─────────────────────────────────────────────────────────────────────────
	// CAPABILITY DETECTION
	// ─────────────────────────────────────────────────────────────────────────

	// CanExtract returns true if this extractor can process the evidence
	CanExtract(ev *evidence.EvidenceUnit) bool

	// SupportedSourceTypes returns the source types this extractor handles
	SupportedSourceTypes() []evidence.SourceType

	// SupportedFactTypes returns the fact types this extractor can produce
	SupportedFactTypes() []factstore.FactType

	// SupportedDomains returns the clinical domains this extractor covers
	SupportedDomains() []evidence.ClinicalDomain

	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTION
	// ─────────────────────────────────────────────────────────────────────────

	// Extract processes evidence and produces draft facts
	// All facts start as DRAFT status and go through governance
	Extract(ctx context.Context, ev *evidence.EvidenceUnit) (*ExtractionResult, error)

	// ─────────────────────────────────────────────────────────────────────────
	// CONFIDENCE & PROVENANCE
	// ─────────────────────────────────────────────────────────────────────────

	// ConfidenceModel returns the confidence calculation model for this extractor
	ConfidenceModel() *ConfidenceModel

	// Provenance returns metadata about this extractor for audit trails
	Provenance() *ExtractorProvenance
}

// =============================================================================
// EXTRACTION RESULT
// =============================================================================

// ExtractionResult contains the output of a fact extraction operation
type ExtractionResult struct {
	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTED FACTS
	// ─────────────────────────────────────────────────────────────────────────

	// DraftFacts are the extracted facts (all start as DRAFT status)
	DraftFacts []*factstore.Fact `json:"draftFacts"`

	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTION METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// ExtractorName identifies which extractor produced these facts
	ExtractorName string `json:"extractorName"`

	// ExtractorVersion is the version for reproducibility
	ExtractorVersion string `json:"extractorVersion"`

	// ExtractionID is a unique identifier for this extraction run
	ExtractionID string `json:"extractionId"`

	// ─────────────────────────────────────────────────────────────────────────
	// PERFORMANCE METRICS
	// ─────────────────────────────────────────────────────────────────────────

	// ProcessingTimeMs is how long extraction took
	ProcessingTimeMs int64 `json:"processingTimeMs"`

	// TokensUsed is the LLM token count (for LLM extractors)
	TokensUsed int `json:"tokensUsed,omitempty"`

	// APICallsMade is the number of external API calls
	APICallsMade int `json:"apiCallsMade,omitempty"`

	// CostEstimateUSD is the estimated cost of this extraction
	CostEstimateUSD float64 `json:"costEstimateUsd,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// QUALITY SIGNALS
	// ─────────────────────────────────────────────────────────────────────────

	// AverageConfidence is the mean confidence across all extracted facts
	AverageConfidence float64 `json:"averageConfidence"`

	// HighConfidenceCount is facts with confidence >= 0.85
	HighConfidenceCount int `json:"highConfidenceCount"`

	// LowConfidenceCount is facts with confidence < 0.65
	LowConfidenceCount int `json:"lowConfidenceCount"`

	// ─────────────────────────────────────────────────────────────────────────
	// ISSUES & WARNINGS
	// ─────────────────────────────────────────────────────────────────────────

	// Warnings are non-fatal issues encountered during extraction
	Warnings []string `json:"warnings,omitempty"`

	// SkippedSections lists sections that couldn't be processed
	SkippedSections []SkippedSection `json:"skippedSections,omitempty"`

	// ValidationErrors are schema/format issues found in source
	ValidationErrors []ValidationError `json:"validationErrors,omitempty"`
}

// SkippedSection records a section that was skipped during extraction
type SkippedSection struct {
	// SectionID identifies the section
	SectionID string `json:"sectionId"`

	// Reason explains why it was skipped
	Reason string `json:"reason"`

	// SectionType is the type of section (for SPL: LOINC code)
	SectionType string `json:"sectionType,omitempty"`
}

// ValidationError records a validation issue in the source
type ValidationError struct {
	// Field is the field or path that failed validation
	Field string `json:"field"`

	// Error is the validation error message
	Error string `json:"error"`

	// Value is the invalid value (if safe to include)
	Value string `json:"value,omitempty"`
}

// =============================================================================
// CONFIDENCE MODEL
// =============================================================================

// ConfidenceModel describes how an extractor calculates fact confidence
type ConfidenceModel struct {
	// ─────────────────────────────────────────────────────────────────────────
	// MODEL IDENTITY
	// ─────────────────────────────────────────────────────────────────────────

	// ModelName is the unique name of this confidence model
	ModelName string `json:"modelName"`

	// ModelVersion allows model updates while maintaining comparability
	ModelVersion string `json:"modelVersion"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONFIDENCE FACTORS
	// ─────────────────────────────────────────────────────────────────────────

	// Factors are the components that contribute to confidence
	Factors []ConfidenceFactor `json:"factors"`

	// BaseConfidence is the starting confidence before factors
	BaseConfidence float64 `json:"baseConfidence"`

	// ─────────────────────────────────────────────────────────────────────────
	// THRESHOLDS
	// ─────────────────────────────────────────────────────────────────────────

	// MinAcceptableConfidence is the floor for this extractor
	MinAcceptableConfidence float64 `json:"minAcceptableConfidence"`

	// AutoApproveThreshold is the confidence level for auto-activation
	AutoApproveThreshold float64 `json:"autoApproveThreshold"`

	// ─────────────────────────────────────────────────────────────────────────
	// CALIBRATION
	// ─────────────────────────────────────────────────────────────────────────

	// LastCalibrated is when the model was last validated against ground truth
	LastCalibrated time.Time `json:"lastCalibrated,omitempty"`

	// CalibrationDatasetID references the validation dataset
	CalibrationDatasetID string `json:"calibrationDatasetId,omitempty"`

	// CalibrationAccuracy is the measured accuracy on the calibration set
	CalibrationAccuracy float64 `json:"calibrationAccuracy,omitempty"`
}

// ConfidenceFactor is a component that affects confidence calculation
type ConfidenceFactor struct {
	// Name identifies this factor
	Name string `json:"name"`

	// Description explains what this factor measures
	Description string `json:"description"`

	// Weight is the relative importance (0.0-1.0)
	Weight float64 `json:"weight"`

	// FactorType categorizes the factor
	FactorType ConfidenceFactorType `json:"factorType"`
}

// ConfidenceFactorType categorizes confidence factors
type ConfidenceFactorType string

const (
	// FactorTypeSourceQuality measures source reliability
	FactorTypeSourceQuality ConfidenceFactorType = "SOURCE_QUALITY"

	// FactorTypeExtractionCertainty measures extraction model certainty
	FactorTypeExtractionCertainty ConfidenceFactorType = "EXTRACTION_CERTAINTY"

	// FactorTypeConsensus measures agreement across multiple sources
	FactorTypeConsensus ConfidenceFactorType = "CONSENSUS"

	// FactorTypeRecency measures how current the source is
	FactorTypeRecency ConfidenceFactorType = "RECENCY"

	// FactorTypeSpecificity measures how specific the extraction was
	FactorTypeSpecificity ConfidenceFactorType = "SPECIFICITY"

	// FactorTypeCorroboration measures support from other facts
	FactorTypeCorroboration ConfidenceFactorType = "CORROBORATION"
)

// =============================================================================
// EXTRACTOR PROVENANCE
// =============================================================================

// ExtractorProvenance contains metadata about an extractor for audit trails
type ExtractorProvenance struct {
	// ─────────────────────────────────────────────────────────────────────────
	// IDENTITY
	// ─────────────────────────────────────────────────────────────────────────

	// ExtractorID is the unique identifier
	ExtractorID string `json:"extractorId"`

	// ExtractorType categorizes the extractor
	ExtractorType ExtractorType `json:"extractorType"`

	// Version is the implementation version
	Version string `json:"version"`

	// ─────────────────────────────────────────────────────────────────────────
	// IMPLEMENTATION DETAILS
	// ─────────────────────────────────────────────────────────────────────────

	// ModelID is the underlying model (for LLM extractors)
	ModelID string `json:"modelId,omitempty"`

	// ModelVersion is the specific model version
	ModelVersion string `json:"modelVersion,omitempty"`

	// PromptTemplateVersion is the prompt version (for LLM extractors)
	PromptTemplateVersion string `json:"promptTemplateVersion,omitempty"`

	// APIVersion is the API version (for API extractors)
	APIVersion string `json:"apiVersion,omitempty"`

	// SchemaVersion is the output schema version
	SchemaVersion string `json:"schemaVersion,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// CAPABILITIES
	// ─────────────────────────────────────────────────────────────────────────

	// SourceTypes lists the source types this extractor handles
	SourceTypes []evidence.SourceType `json:"sourceTypes"`

	// FactTypes lists the fact types this extractor produces
	FactTypes []factstore.FactType `json:"factTypes"`

	// Domains lists the clinical domains covered
	Domains []evidence.ClinicalDomain `json:"domains"`

	// ─────────────────────────────────────────────────────────────────────────
	// QUALITY METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// ValidationStatus indicates if the extractor is validated
	ValidationStatus ValidationStatus `json:"validationStatus"`

	// LastValidated is when validation was last performed
	LastValidated time.Time `json:"lastValidated,omitempty"`

	// ValidationScore is the score from the last validation
	ValidationScore float64 `json:"validationScore,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// OPERATIONAL
	// ─────────────────────────────────────────────────────────────────────────

	// Enabled indicates if the extractor is currently active
	Enabled bool `json:"enabled"`

	// RateLimitPerMinute is the maximum extractions per minute
	RateLimitPerMinute int `json:"rateLimitPerMinute,omitempty"`

	// CostPerExtraction is the estimated cost in USD
	CostPerExtraction float64 `json:"costPerExtraction,omitempty"`

	// AverageLatencyMs is the typical extraction time
	AverageLatencyMs int64 `json:"averageLatencyMs,omitempty"`
}

// ExtractorType categorizes extractor implementations
type ExtractorType string

const (
	// ExtractorTypeLLM uses large language models
	ExtractorTypeLLM ExtractorType = "LLM"

	// ExtractorTypeAPI uses structured APIs
	ExtractorTypeAPI ExtractorType = "API"

	// ExtractorTypeETL uses ETL pipelines for bulk data
	ExtractorTypeETL ExtractorType = "ETL"

	// ExtractorTypeHybrid combines multiple approaches
	ExtractorTypeHybrid ExtractorType = "HYBRID"

	// ExtractorTypeManual is for human-curated facts
	ExtractorTypeManual ExtractorType = "MANUAL"
)

// ValidationStatus indicates extractor validation state
type ValidationStatus string

const (
	// ValidationPending means validation not yet performed
	ValidationPending ValidationStatus = "PENDING"

	// ValidationPassed means extractor passed validation
	ValidationPassed ValidationStatus = "PASSED"

	// ValidationFailed means extractor failed validation
	ValidationFailed ValidationStatus = "FAILED"

	// ValidationExpired means validation needs renewal
	ValidationExpired ValidationStatus = "EXPIRED"
)

// =============================================================================
// EXTRACTOR REGISTRY
// =============================================================================

// ExtractorRegistry manages all registered extractors
type ExtractorRegistry struct {
	extractors map[string]FactExtractor
}

// NewExtractorRegistry creates a new registry
func NewExtractorRegistry() *ExtractorRegistry {
	return &ExtractorRegistry{
		extractors: make(map[string]FactExtractor),
	}
}

// Register adds an extractor to the registry
func (r *ExtractorRegistry) Register(extractor FactExtractor) {
	r.extractors[extractor.Name()] = extractor
}

// Get retrieves an extractor by name
func (r *ExtractorRegistry) Get(name string) (FactExtractor, bool) {
	ext, ok := r.extractors[name]
	return ext, ok
}

// FindForEvidence returns extractors that can process the evidence
func (r *ExtractorRegistry) FindForEvidence(ev *evidence.EvidenceUnit) []FactExtractor {
	var matching []FactExtractor
	for _, ext := range r.extractors {
		if ext.CanExtract(ev) {
			matching = append(matching, ext)
		}
	}
	return matching
}

// FindBySourceType returns extractors for a source type
func (r *ExtractorRegistry) FindBySourceType(st evidence.SourceType) []FactExtractor {
	var matching []FactExtractor
	for _, ext := range r.extractors {
		for _, supported := range ext.SupportedSourceTypes() {
			if supported == st {
				matching = append(matching, ext)
				break
			}
		}
	}
	return matching
}

// FindByFactType returns extractors that produce a fact type
func (r *ExtractorRegistry) FindByFactType(ft factstore.FactType) []FactExtractor {
	var matching []FactExtractor
	for _, ext := range r.extractors {
		for _, supported := range ext.SupportedFactTypes() {
			if supported == ft {
				matching = append(matching, ext)
				break
			}
		}
	}
	return matching
}

// All returns all registered extractors
func (r *ExtractorRegistry) All() []FactExtractor {
	result := make([]FactExtractor, 0, len(r.extractors))
	for _, ext := range r.extractors {
		result = append(result, ext)
	}
	return result
}

// Count returns the number of registered extractors
func (r *ExtractorRegistry) Count() int {
	return len(r.extractors)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewExtractionResult creates a new extraction result with defaults
func NewExtractionResult(extractorName, extractorVersion string) *ExtractionResult {
	return &ExtractionResult{
		ExtractorName:    extractorName,
		ExtractorVersion: extractorVersion,
		DraftFacts:       make([]*factstore.Fact, 0),
		Warnings:         make([]string, 0),
		SkippedSections:  make([]SkippedSection, 0),
		ValidationErrors: make([]ValidationError, 0),
	}
}

// AddFact adds a draft fact to the result
func (r *ExtractionResult) AddFact(fact *factstore.Fact) {
	r.DraftFacts = append(r.DraftFacts, fact)
}

// AddWarning adds a warning message
func (r *ExtractionResult) AddWarning(warning string) {
	r.Warnings = append(r.Warnings, warning)
}

// AddSkippedSection records a skipped section
func (r *ExtractionResult) AddSkippedSection(sectionID, reason, sectionType string) {
	r.SkippedSections = append(r.SkippedSections, SkippedSection{
		SectionID:   sectionID,
		Reason:      reason,
		SectionType: sectionType,
	})
}

// AddValidationError records a validation error
func (r *ExtractionResult) AddValidationError(field, errMsg, value string) {
	r.ValidationErrors = append(r.ValidationErrors, ValidationError{
		Field: field,
		Error: errMsg,
		Value: value,
	})
}

// CalculateConfidenceStats computes confidence statistics
func (r *ExtractionResult) CalculateConfidenceStats() {
	if len(r.DraftFacts) == 0 {
		return
	}

	var sum float64
	r.HighConfidenceCount = 0
	r.LowConfidenceCount = 0

	for _, fact := range r.DraftFacts {
		conf := fact.Confidence.Overall
		sum += conf
		if conf >= 0.85 {
			r.HighConfidenceCount++
		} else if conf < 0.65 {
			r.LowConfidenceCount++
		}
	}

	r.AverageConfidence = sum / float64(len(r.DraftFacts))
}

// FactCount returns the number of extracted facts
func (r *ExtractionResult) FactCount() int {
	return len(r.DraftFacts)
}

// HasWarnings returns true if there are warnings
func (r *ExtractionResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// HasErrors returns true if there are validation errors
func (r *ExtractionResult) HasErrors() bool {
	return len(r.ValidationErrors) > 0
}
