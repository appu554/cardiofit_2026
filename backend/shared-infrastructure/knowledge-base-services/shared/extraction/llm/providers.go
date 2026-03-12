// Package llm provides the LLM Provider Interface for clinical fact extraction.
//
// Phase 3c.1: LLM Provider Interface
// Authority Level: GAP-FILLER ONLY (Use only when structured data unavailable)
//
// KEY PRINCIPLE: LLM is a "gap filler of last resort" - NEVER the primary source.
// All LLM extractions require 2-of-3 provider consensus before acceptance.
//
// NAVIGATION RULE 3: "LLMs disagree → HUMAN first"
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// PROVIDER INTERFACE
// =============================================================================

// Provider defines the interface for LLM providers
// All providers must implement this interface for consensus extraction
type Provider interface {
	// Name returns the unique identifier for this provider
	// Examples: "claude-3-opus", "gpt-4-turbo", "gemini-pro"
	Name() string

	// Version returns the provider implementation version
	Version() string

	// Extract processes source text and extracts structured facts
	Extract(ctx context.Context, req *ExtractionRequest) (*ExtractionResult, error)

	// SupportsStructuredOutput returns true if provider natively supports JSON schema
	// Claude and GPT-4 support this; some providers may need post-processing
	SupportsStructuredOutput() bool

	// MaxTokens returns the maximum context window for this provider
	MaxTokens() int

	// CostPerToken returns the cost per token in USD (for budgeting)
	CostPerToken() float64
}

// =============================================================================
// EXTRACTION REQUEST
// =============================================================================

// ExtractionRequest defines what to extract from source text
type ExtractionRequest struct {
	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTION TARGET
	// ─────────────────────────────────────────────────────────────────────────

	// FactType identifies the type of fact to extract
	// Examples: "RENAL_DOSE_ADJUST", "HEPATIC_DOSE_ADJUST", "DRUG_INTERACTION"
	FactType FactType `json:"factType"`

	// Schema defines the expected output structure
	Schema *ExtractionSchema `json:"schema"`

	// ─────────────────────────────────────────────────────────────────────────
	// SOURCE CONTENT
	// ─────────────────────────────────────────────────────────────────────────

	// SourceText is the text to extract from (SPL section, etc.)
	SourceText string `json:"sourceText"`

	// SourceType identifies the source format
	// Examples: "SPL_SECTION", "CLINICAL_NOTE", "GUIDELINE"
	SourceType SourceType `json:"sourceType"`

	// SourceLOINC is the LOINC code for SPL sections
	SourceLOINC string `json:"sourceLoinc,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONTEXT
	// ─────────────────────────────────────────────────────────────────────────

	// DrugContext provides drug information for extraction
	DrugContext *DrugContext `json:"drugContext"`

	// ExistingFacts are related facts already extracted (for consistency)
	ExistingFacts []ExistingFact `json:"existingFacts,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTION OPTIONS
	// ─────────────────────────────────────────────────────────────────────────

	// RequireCitations forces the LLM to cite source text locations
	RequireCitations bool `json:"requireCitations"`

	// StrictSchema enforces exact schema match (no extra fields)
	StrictSchema bool `json:"strictSchema"`

	// Temperature controls LLM creativity (0.0-1.0, lower = more deterministic)
	Temperature float64 `json:"temperature"`

	// MaxRetries for transient failures
	MaxRetries int `json:"maxRetries"`

	// ─────────────────────────────────────────────────────────────────────────
	// AUDIT METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// RequestID for tracing
	RequestID string `json:"requestId"`

	// SourceDocumentID links to source_documents table
	SourceDocumentID string `json:"sourceDocumentId"`

	// RequestedAt is the timestamp of the request
	RequestedAt time.Time `json:"requestedAt"`
}

// FactType categorizes the type of fact being extracted
type FactType string

const (
	// FactTypeRenalDoseAdjust is for kidney-based dosing adjustments
	FactTypeRenalDoseAdjust FactType = "RENAL_DOSE_ADJUST"

	// FactTypeHepaticDoseAdjust is for liver-based dosing adjustments
	FactTypeHepaticDoseAdjust FactType = "HEPATIC_DOSE_ADJUST"

	// FactTypeDrugInteraction is for drug-drug interactions
	FactTypeDrugInteraction FactType = "DRUG_INTERACTION"

	// FactTypeContraindication is for drug contraindications
	FactTypeContraindication FactType = "CONTRAINDICATION"

	// FactTypeAdverseReaction is for adverse drug reactions
	FactTypeAdverseReaction FactType = "ADVERSE_REACTION"

	// FactTypeBlackBoxWarning is for FDA black box warnings
	FactTypeBlackBoxWarning FactType = "BLACK_BOX_WARNING"

	// FactTypePregnancyRisk is for pregnancy safety information
	FactTypePregnancyRisk FactType = "PREGNANCY_RISK"

	// FactTypeLactationRisk is for breastfeeding safety information
	FactTypeLactationRisk FactType = "LACTATION_RISK"

	// FactTypeGeriatricDosing is for elderly dosing adjustments
	FactTypeGeriatricDosing FactType = "GERIATRIC_DOSING"

	// FactTypePediatricDosing is for pediatric dosing information
	FactTypePediatricDosing FactType = "PEDIATRIC_DOSING"
)

// SourceType identifies the type of source document
type SourceType string

const (
	// SourceTypeSPLSection is an FDA Structured Product Label section
	SourceTypeSPLSection SourceType = "SPL_SECTION"

	// SourceTypeClinicalNote is a clinical note or documentation
	SourceTypeClinicalNote SourceType = "CLINICAL_NOTE"

	// SourceTypeGuideline is a clinical practice guideline
	SourceTypeGuideline SourceType = "GUIDELINE"

	// SourceTypePackageInsert is a drug package insert
	SourceTypePackageInsert SourceType = "PACKAGE_INSERT"
)

// =============================================================================
// DRUG CONTEXT
// =============================================================================

// DrugContext provides drug information for extraction context
type DrugContext struct {
	// RxCUI is the RxNorm Concept Unique Identifier
	RxCUI string `json:"rxcui"`

	// DrugName is the brand or trade name
	DrugName string `json:"drugName"`

	// GenericName is the non-proprietary name
	GenericName string `json:"genericName"`

	// DrugClass is the therapeutic class (e.g., "ACE Inhibitor")
	DrugClass string `json:"drugClass"`

	// ATCCode is the WHO ATC classification
	ATCCode string `json:"atcCode,omitempty"`

	// RouteOfAdministration (e.g., "oral", "IV", "subcutaneous")
	RouteOfAdministration string `json:"routeOfAdministration,omitempty"`

	// DosageForm (e.g., "tablet", "capsule", "injection")
	DosageForm string `json:"dosageForm,omitempty"`

	// Manufacturer name
	Manufacturer string `json:"manufacturer,omitempty"`
}

// ExistingFact represents a previously extracted fact for context
type ExistingFact struct {
	FactType   FactType    `json:"factType"`
	FactData   interface{} `json:"factData"`
	Confidence float64     `json:"confidence"`
}

// =============================================================================
// EXTRACTION SCHEMA
// =============================================================================

// ExtractionSchema defines the expected output structure for extraction
type ExtractionSchema struct {
	// Name is the schema identifier
	Name string `json:"name"`

	// Version allows schema evolution
	Version string `json:"version"`

	// Description explains what this schema captures
	Description string `json:"description"`

	// JSONSchema is the JSON Schema definition for structured output
	JSONSchema json.RawMessage `json:"jsonSchema"`

	// RequiredFields lists fields that must be present
	RequiredFields []string `json:"requiredFields"`

	// ValidationRules are additional validation constraints
	ValidationRules []ValidationRule `json:"validationRules,omitempty"`
}

// ValidationRule defines a validation constraint
type ValidationRule struct {
	// Field is the field path to validate
	Field string `json:"field"`

	// Rule is the validation type
	Rule ValidationRuleType `json:"rule"`

	// Value is the rule parameter (e.g., min value, pattern)
	Value interface{} `json:"value,omitempty"`

	// Message is the error message if validation fails
	Message string `json:"message"`
}

// ValidationRuleType categorizes validation rules
type ValidationRuleType string

const (
	// ValidationRequired means field must be present
	ValidationRequired ValidationRuleType = "REQUIRED"

	// ValidationRange means numeric value must be in range
	ValidationRange ValidationRuleType = "RANGE"

	// ValidationPattern means string must match regex
	ValidationPattern ValidationRuleType = "PATTERN"

	// ValidationEnum means value must be one of allowed values
	ValidationEnum ValidationRuleType = "ENUM"
)

// =============================================================================
// EXTRACTION RESULT
// =============================================================================

// ExtractionResult contains the LLM's extraction output
type ExtractionResult struct {
	// ─────────────────────────────────────────────────────────────────────────
	// PROVIDER IDENTITY
	// ─────────────────────────────────────────────────────────────────────────

	// Provider is the name of the provider that produced this result
	Provider string `json:"provider"`

	// ProviderVersion is the specific model version used
	ProviderVersion string `json:"providerVersion"`

	// RequestID links to the original request
	RequestID string `json:"requestId"`

	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTED DATA
	// ─────────────────────────────────────────────────────────────────────────

	// FactType is the type of fact extracted
	FactType FactType `json:"factType"`

	// ExtractedData contains the structured extraction (matches schema)
	ExtractedData interface{} `json:"extractedData"`

	// ExtractedDataJSON is the JSON representation for comparison
	ExtractedDataJSON json.RawMessage `json:"extractedDataJson"`

	// ─────────────────────────────────────────────────────────────────────────
	// CONFIDENCE & CITATIONS
	// ─────────────────────────────────────────────────────────────────────────

	// Confidence is the provider's self-assessed confidence (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// ConfidenceExplanation describes why this confidence level
	ConfidenceExplanation string `json:"confidenceExplanation,omitempty"`

	// Citations are quotes from source text supporting the extraction
	Citations []Citation `json:"citations,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTION METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// Latency is how long the extraction took
	Latency time.Duration `json:"latency"`

	// TokensUsed is the total tokens consumed
	TokensUsed TokenUsage `json:"tokensUsed"`

	// Cost is the estimated cost in USD
	Cost float64 `json:"cost"`

	// RetryCount is how many retries were needed
	RetryCount int `json:"retryCount"`

	// ─────────────────────────────────────────────────────────────────────────
	// RAW RESPONSE
	// ─────────────────────────────────────────────────────────────────────────

	// RawResponse is the complete LLM response (for debugging/audit)
	RawResponse string `json:"rawResponse,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// STATUS
	// ─────────────────────────────────────────────────────────────────────────

	// Success indicates if extraction completed successfully
	Success bool `json:"success"`

	// Error contains any error message if extraction failed
	Error string `json:"error,omitempty"`

	// Warnings are non-fatal issues encountered
	Warnings []string `json:"warnings,omitempty"`

	// ExtractedAt is when extraction completed
	ExtractedAt time.Time `json:"extractedAt"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	// PromptTokens is the input token count
	PromptTokens int `json:"promptTokens"`

	// CompletionTokens is the output token count
	CompletionTokens int `json:"completionTokens"`

	// TotalTokens is the sum
	TotalTokens int `json:"totalTokens"`
}

// Citation represents a quote from source text
type Citation struct {
	// StartOffset is the character offset in source text
	StartOffset int `json:"startOffset"`

	// EndOffset is the end character offset
	EndOffset int `json:"endOffset"`

	// QuotedText is the exact text quoted
	QuotedText string `json:"quotedText"`

	// SupportsFact describes what this citation supports
	SupportsFact string `json:"supportsFact,omitempty"`

	// Confidence for this specific citation
	Confidence float64 `json:"confidence"`
}

// =============================================================================
// PREDEFINED SCHEMAS
// =============================================================================

// RenalDoseSchema is the schema for renal dose adjustments
var RenalDoseSchema = &ExtractionSchema{
	Name:        "renal_dose_adjustment",
	Version:     "1.0.0",
	Description: "Kidney function-based dosing adjustments from SPL labels",
	JSONSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"hasRenalDosing": {"type": "boolean"},
			"gfrBands": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"minGFR": {"type": "number"},
						"maxGFR": {"type": "number"},
						"action": {"type": "string", "enum": ["NO_CHANGE", "REDUCE", "AVOID", "CONTRAINDICATED"]},
						"recommendedDose": {"type": "string"},
						"maxDose": {"type": "string"},
						"frequency": {"type": "string"}
					},
					"required": ["minGFR", "action"]
				}
			},
			"dialysisGuidance": {
				"type": "object",
				"properties": {
					"hemodialysis": {"type": "string"},
					"peritonealDialysis": {"type": "string"},
					"crrt": {"type": "string"}
				}
			},
			"specialPopulations": {
				"type": "array",
				"items": {"type": "string"}
			}
		},
		"required": ["hasRenalDosing"]
	}`),
	RequiredFields: []string{"hasRenalDosing"},
}

// HepaticDoseSchema is the schema for hepatic dose adjustments
var HepaticDoseSchema = &ExtractionSchema{
	Name:        "hepatic_dose_adjustment",
	Version:     "1.0.0",
	Description: "Liver function-based dosing adjustments from SPL labels",
	JSONSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"hasHepaticDosing": {"type": "boolean"},
			"childPughAdjustments": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"class": {"type": "string", "enum": ["A", "B", "C"]},
						"action": {"type": "string", "enum": ["NO_CHANGE", "REDUCE", "AVOID", "CONTRAINDICATED"]},
						"recommendedDose": {"type": "string"},
						"maxDose": {"type": "string"}
					},
					"required": ["class", "action"]
				}
			},
			"cirrhoticPatients": {"type": "string"},
			"monitoringRequired": {"type": "boolean"},
			"lftsRequired": {"type": "boolean"}
		},
		"required": ["hasHepaticDosing"]
	}`),
	RequiredFields: []string{"hasHepaticDosing"},
}

// DrugInteractionSchema is the schema for drug-drug interactions
var DrugInteractionSchema = &ExtractionSchema{
	Name:        "drug_interaction",
	Version:     "1.0.0",
	Description: "Drug-drug interaction information from SPL labels",
	JSONSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"interactions": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"interactingDrug": {"type": "string"},
						"interactingDrugClass": {"type": "string"},
						"severity": {"type": "string", "enum": ["CONTRAINDICATED", "MAJOR", "MODERATE", "MINOR"]},
						"mechanism": {"type": "string"},
						"clinicalEffect": {"type": "string"},
						"management": {"type": "string"},
						"monitoringRequired": {"type": "boolean"}
					},
					"required": ["interactingDrug", "severity", "clinicalEffect"]
				}
			}
		},
		"required": ["interactions"]
	}`),
	RequiredFields: []string{"interactions"},
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewExtractionRequest creates a new extraction request with defaults
func NewExtractionRequest(factType FactType, sourceText string, schema *ExtractionSchema) *ExtractionRequest {
	return &ExtractionRequest{
		FactType:         factType,
		SourceText:       sourceText,
		Schema:           schema,
		Temperature:      0.0, // Deterministic by default for clinical data
		RequireCitations: true,
		StrictSchema:     true,
		MaxRetries:       3,
		RequestedAt:      time.Now(),
	}
}

// NewExtractionResult creates a new extraction result
func NewExtractionResult(provider string, factType FactType) *ExtractionResult {
	return &ExtractionResult{
		Provider:    provider,
		FactType:    factType,
		ExtractedAt: time.Now(),
		Warnings:    make([]string, 0),
		Citations:   make([]Citation, 0),
	}
}

// SetExtractedData sets the extracted data and its JSON representation
func (r *ExtractionResult) SetExtractedData(data interface{}) error {
	r.ExtractedData = data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling extracted data: %w", err)
	}
	r.ExtractedDataJSON = jsonData
	return nil
}

// AddCitation adds a citation to the result
func (r *ExtractionResult) AddCitation(start, end int, text, supports string, confidence float64) {
	r.Citations = append(r.Citations, Citation{
		StartOffset:  start,
		EndOffset:    end,
		QuotedText:   text,
		SupportsFact: supports,
		Confidence:   confidence,
	})
}

// AddWarning adds a warning to the result
func (r *ExtractionResult) AddWarning(warning string) {
	r.Warnings = append(r.Warnings, warning)
}

// SetError marks the result as failed with an error
func (r *ExtractionResult) SetError(err error) {
	r.Success = false
	r.Error = err.Error()
}

// MarkSuccess marks the result as successful
func (r *ExtractionResult) MarkSuccess() {
	r.Success = true
	r.Error = ""
}

// CalculateCost estimates the cost based on token usage
func (r *ExtractionResult) CalculateCost(inputCostPerMillion, outputCostPerMillion float64) {
	inputCost := float64(r.TokensUsed.PromptTokens) * inputCostPerMillion / 1000000
	outputCost := float64(r.TokensUsed.CompletionTokens) * outputCostPerMillion / 1000000
	r.Cost = inputCost + outputCost
}

// =============================================================================
// PROVIDER REGISTRY
// =============================================================================

// ProviderRegistry manages registered LLM providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// All returns all registered providers
func (r *ProviderRegistry) All() []Provider {
	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// Count returns the number of registered providers
func (r *ProviderRegistry) Count() int {
	return len(r.providers)
}
