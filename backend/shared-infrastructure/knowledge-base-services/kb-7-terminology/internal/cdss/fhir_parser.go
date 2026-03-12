package cdss

import (
	"encoding/json"
	"fmt"
	"strings"

	"kb-7-terminology/internal/models"
)

// ============================================================================
// FHIR Parser Utilities
// ============================================================================
// Utilities for parsing FHIR resources from bundles and JSON payloads.
// Handles the polymorphic nature of FHIR resource types.

// ParsedBundle represents the result of parsing a FHIR Bundle
type ParsedBundle struct {
	BundleID    string
	BundleType  string
	Conditions  []models.FHIRCondition
	Observations []models.FHIRObservation
	Medications []models.FHIRMedicationRequest
	Procedures  []models.FHIRProcedure
	Allergies   []models.FHIRAllergyIntolerance

	// Parsing statistics
	TotalEntries     int
	ParsedEntries    int
	SkippedEntries   int
	UnsupportedTypes map[string]int

	// Errors encountered during parsing
	Errors []ParseError
}

// ParseError represents an error during FHIR parsing
type ParseError struct {
	EntryIndex   int    `json:"entry_index"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id,omitempty"`
	Error        string `json:"error"`
}

// resourceTypeExtractor is a helper struct to extract resourceType from JSON
type resourceTypeExtractor struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id"`
}

// ParseBundle parses a FHIR Bundle and extracts all supported resources
func ParseBundle(bundle *models.FHIRBundle) (*ParsedBundle, error) {
	if bundle == nil {
		return nil, fmt.Errorf("bundle is nil")
	}

	result := &ParsedBundle{
		BundleID:         bundle.ID,
		BundleType:       bundle.Type,
		TotalEntries:     len(bundle.Entry),
		UnsupportedTypes: make(map[string]int),
	}

	for i, entry := range bundle.Entry {
		if len(entry.Resource) == 0 {
			result.SkippedEntries++
			continue
		}

		// First, extract the resource type
		var extractor resourceTypeExtractor
		if err := json.Unmarshal(entry.Resource, &extractor); err != nil {
			result.Errors = append(result.Errors, ParseError{
				EntryIndex: i,
				Error:      fmt.Sprintf("failed to extract resource type: %v", err),
			})
			result.SkippedEntries++
			continue
		}

		// Parse based on resource type
		switch extractor.ResourceType {
		case models.ResourceTypeCondition:
			var condition models.FHIRCondition
			if err := json.Unmarshal(entry.Resource, &condition); err != nil {
				result.Errors = append(result.Errors, ParseError{
					EntryIndex:   i,
					ResourceType: extractor.ResourceType,
					ResourceID:   extractor.ID,
					Error:        fmt.Sprintf("failed to parse Condition: %v", err),
				})
			} else {
				result.Conditions = append(result.Conditions, condition)
				result.ParsedEntries++
			}

		case models.ResourceTypeObservation:
			var observation models.FHIRObservation
			if err := json.Unmarshal(entry.Resource, &observation); err != nil {
				result.Errors = append(result.Errors, ParseError{
					EntryIndex:   i,
					ResourceType: extractor.ResourceType,
					ResourceID:   extractor.ID,
					Error:        fmt.Sprintf("failed to parse Observation: %v", err),
				})
			} else {
				result.Observations = append(result.Observations, observation)
				result.ParsedEntries++
			}

		case models.ResourceTypeMedicationRequest:
			var medication models.FHIRMedicationRequest
			if err := json.Unmarshal(entry.Resource, &medication); err != nil {
				result.Errors = append(result.Errors, ParseError{
					EntryIndex:   i,
					ResourceType: extractor.ResourceType,
					ResourceID:   extractor.ID,
					Error:        fmt.Sprintf("failed to parse MedicationRequest: %v", err),
				})
			} else {
				result.Medications = append(result.Medications, medication)
				result.ParsedEntries++
			}

		case models.ResourceTypeProcedure:
			var procedure models.FHIRProcedure
			if err := json.Unmarshal(entry.Resource, &procedure); err != nil {
				result.Errors = append(result.Errors, ParseError{
					EntryIndex:   i,
					ResourceType: extractor.ResourceType,
					ResourceID:   extractor.ID,
					Error:        fmt.Sprintf("failed to parse Procedure: %v", err),
				})
			} else {
				result.Procedures = append(result.Procedures, procedure)
				result.ParsedEntries++
			}

		case models.ResourceTypeAllergyIntolerance:
			var allergy models.FHIRAllergyIntolerance
			if err := json.Unmarshal(entry.Resource, &allergy); err != nil {
				result.Errors = append(result.Errors, ParseError{
					EntryIndex:   i,
					ResourceType: extractor.ResourceType,
					ResourceID:   extractor.ID,
					Error:        fmt.Sprintf("failed to parse AllergyIntolerance: %v", err),
				})
			} else {
				result.Allergies = append(result.Allergies, allergy)
				result.ParsedEntries++
			}

		default:
			// Track unsupported resource types
			result.UnsupportedTypes[extractor.ResourceType]++
			result.SkippedEntries++
		}
	}

	return result, nil
}

// ============================================================================
// Code Extraction Utilities
// ============================================================================

// ExtractedCode represents a normalized code extracted from a FHIR resource
type ExtractedCode struct {
	Code    string
	System  string
	Display string
	Version string
}

// ExtractCodesFromCodeableConcept extracts all codes from a CodeableConcept
func ExtractCodesFromCodeableConcept(cc *models.CodeableConcept) []ExtractedCode {
	if cc == nil {
		return nil
	}

	var codes []ExtractedCode
	for _, coding := range cc.Coding {
		codes = append(codes, ExtractedCode{
			Code:    coding.Code,
			System:  NormalizeSystemURI(coding.System),
			Display: coding.Display,
			Version: coding.Version,
		})
	}
	return codes
}

// ExtractPrimaryCode extracts the primary (first) code from a CodeableConcept
func ExtractPrimaryCode(cc *models.CodeableConcept) *ExtractedCode {
	if cc == nil || len(cc.Coding) == 0 {
		return nil
	}

	coding := cc.Coding[0]
	return &ExtractedCode{
		Code:    coding.Code,
		System:  NormalizeSystemURI(coding.System),
		Display: coding.Display,
		Version: coding.Version,
	}
}

// ExtractCodeBySystem extracts the first code matching a specific system
func ExtractCodeBySystem(cc *models.CodeableConcept, system string) *ExtractedCode {
	if cc == nil {
		return nil
	}

	normalizedSystem := NormalizeSystemURI(system)
	for _, coding := range cc.Coding {
		if NormalizeSystemURI(coding.System) == normalizedSystem {
			return &ExtractedCode{
				Code:    coding.Code,
				System:  normalizedSystem,
				Display: coding.Display,
				Version: coding.Version,
			}
		}
	}
	return nil
}

// ============================================================================
// System URI Normalization
// ============================================================================

// NormalizeSystemURI normalizes terminology system URIs to canonical forms
func NormalizeSystemURI(uri string) string {
	if uri == "" {
		return ""
	}

	// Trim whitespace and normalize case for known systems
	uri = strings.TrimSpace(uri)

	// Map common variations to canonical URIs
	switch strings.ToLower(uri) {
	case "http://snomed.info/sct", "snomed", "snomed-ct", "sct":
		return models.SystemSNOMED
	case "http://loinc.org", "loinc":
		return models.SystemLOINC
	case "http://www.nlm.nih.gov/research/umls/rxnorm", "rxnorm":
		return models.SystemRxNorm
	case "http://hl7.org/fhir/sid/icd-10", "http://hl7.org/fhir/sid/icd-10-cm", "icd-10", "icd10":
		return models.SystemICD10
	case "http://www.ama-assn.org/go/cpt", "cpt":
		return models.SystemCPT
	case "http://hl7.org/fhir/sid/ndc", "ndc":
		return models.SystemNDC
	case "http://www.whocc.no/atc", "atc":
		return models.SystemATC
	case "http://unitsofmeasure.org", "ucum":
		return models.SystemUCUM
	default:
		return uri
	}
}

// GetSystemShortName returns a short display name for a system URI
func GetSystemShortName(uri string) string {
	normalizedURI := NormalizeSystemURI(uri)
	switch normalizedURI {
	case models.SystemSNOMED:
		return "SNOMED"
	case models.SystemLOINC:
		return "LOINC"
	case models.SystemRxNorm:
		return "RxNorm"
	case models.SystemICD10, models.SystemICD10CM:
		return "ICD-10"
	case models.SystemCPT:
		return "CPT"
	case models.SystemNDC:
		return "NDC"
	case models.SystemATC:
		return "ATC"
	case models.SystemUCUM:
		return "UCUM"
	default:
		// Return the last segment of the URI
		parts := strings.Split(uri, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return uri
	}
}

// ============================================================================
// Clinical Status Extraction
// ============================================================================

// ExtractClinicalStatus extracts the clinical status from a CodeableConcept
func ExtractClinicalStatus(status *models.CodeableConcept) models.FactStatus {
	if status == nil {
		return models.FactStatusUnknown
	}

	for _, coding := range status.Coding {
		switch strings.ToLower(coding.Code) {
		case "active", "recurrence", "relapse":
			return models.FactStatusActive
		case "inactive", "remission":
			return models.FactStatusInactive
		case "resolved":
			return models.FactStatusResolved
		case "pending":
			return models.FactStatusPending
		case "completed":
			return models.FactStatusCompleted
		}
	}

	// Check text if no matching coding found
	if status.Text != "" {
		switch strings.ToLower(status.Text) {
		case "active":
			return models.FactStatusActive
		case "inactive":
			return models.FactStatusInactive
		case "resolved":
			return models.FactStatusResolved
		}
	}

	return models.FactStatusUnknown
}

// ExtractMedicationStatus converts FHIR medication status to FactStatus
func ExtractMedicationStatus(status string) models.FactStatus {
	switch strings.ToLower(status) {
	case "active":
		return models.FactStatusActive
	case "completed":
		return models.FactStatusCompleted
	case "on-hold", "stopped", "cancelled", "entered-in-error":
		return models.FactStatusInactive
	case "draft":
		return models.FactStatusPending
	default:
		return models.FactStatusUnknown
	}
}

// ExtractObservationStatus converts FHIR observation status to FactStatus
func ExtractObservationStatus(status string) models.FactStatus {
	switch strings.ToLower(status) {
	case "final", "amended", "corrected":
		return models.FactStatusCompleted
	case "preliminary", "registered":
		return models.FactStatusPending
	case "cancelled", "entered-in-error":
		return models.FactStatusInactive
	default:
		return models.FactStatusUnknown
	}
}

// ExtractProcedureStatus converts FHIR procedure status to FactStatus
func ExtractProcedureStatus(status string) models.FactStatus {
	switch strings.ToLower(status) {
	case "completed":
		return models.FactStatusCompleted
	case "in-progress", "preparation":
		return models.FactStatusActive
	case "not-done", "on-hold", "stopped", "entered-in-error":
		return models.FactStatusInactive
	default:
		return models.FactStatusUnknown
	}
}

// ============================================================================
// Observation Category Utilities
// ============================================================================

// ObservationCategory represents the category of an observation
type ObservationCategory string

const (
	CategoryVitalSigns    ObservationCategory = "vital-signs"
	CategoryLaboratory    ObservationCategory = "laboratory"
	CategorySurvey        ObservationCategory = "survey"
	CategoryExam          ObservationCategory = "exam"
	CategoryImaging       ObservationCategory = "imaging"
	CategoryProcedure     ObservationCategory = "procedure"
	CategorySocialHistory ObservationCategory = "social-history"
	CategoryActivity      ObservationCategory = "activity"
	CategoryUnknown       ObservationCategory = "unknown"
)

// ExtractObservationCategory extracts the primary category from observation categories
func ExtractObservationCategory(categories []models.CodeableConcept) ObservationCategory {
	for _, category := range categories {
		for _, coding := range category.Coding {
			switch strings.ToLower(coding.Code) {
			case "vital-signs":
				return CategoryVitalSigns
			case "laboratory":
				return CategoryLaboratory
			case "survey":
				return CategorySurvey
			case "exam":
				return CategoryExam
			case "imaging":
				return CategoryImaging
			case "procedure":
				return CategoryProcedure
			case "social-history":
				return CategorySocialHistory
			case "activity":
				return CategoryActivity
			}
		}
	}
	return CategoryUnknown
}

// CategoryToFactType converts observation category to fact type
func CategoryToFactType(category ObservationCategory) models.FactType {
	switch category {
	case CategoryVitalSigns:
		return models.FactTypeVitalSign
	case CategoryLaboratory:
		return models.FactTypeLab
	default:
		return models.FactTypeObservation
	}
}

// ============================================================================
// Interpretation Utilities
// ============================================================================

// InterpretationCode represents an observation interpretation
type InterpretationCode string

const (
	InterpNormal      InterpretationCode = "N"
	InterpAbnormal    InterpretationCode = "A"
	InterpHigh        InterpretationCode = "H"
	InterpLow         InterpretationCode = "L"
	InterpCriticalHigh InterpretationCode = "HH"
	InterpCriticalLow  InterpretationCode = "LL"
	InterpPositive    InterpretationCode = "POS"
	InterpNegative    InterpretationCode = "NEG"
)

// ExtractInterpretation extracts the primary interpretation from an observation
func ExtractInterpretation(interpretations []models.CodeableConcept) (InterpretationCode, bool, bool) {
	var primaryInterp InterpretationCode
	isAbnormal := false
	isCritical := false

	for _, interp := range interpretations {
		for _, coding := range interp.Coding {
			code := strings.ToUpper(coding.Code)
			switch code {
			case "H", "HH", "HU":
				isAbnormal = true
				if code == "HH" {
					isCritical = true
					primaryInterp = InterpCriticalHigh
				} else if primaryInterp == "" {
					primaryInterp = InterpHigh
				}
			case "L", "LL", "LU":
				isAbnormal = true
				if code == "LL" {
					isCritical = true
					primaryInterp = InterpCriticalLow
				} else if primaryInterp == "" {
					primaryInterp = InterpLow
				}
			case "A", "AA":
				isAbnormal = true
				if code == "AA" {
					isCritical = true
				}
				if primaryInterp == "" {
					primaryInterp = InterpAbnormal
				}
			case "N":
				if primaryInterp == "" {
					primaryInterp = InterpNormal
				}
			case "POS":
				isAbnormal = true
				primaryInterp = InterpPositive
			case "NEG":
				primaryInterp = InterpNegative
			}
		}
	}

	return primaryInterp, isAbnormal, isCritical
}

// ============================================================================
// Severity Utilities
// ============================================================================

// ExtractSeverity extracts severity from a CodeableConcept
func ExtractSeverity(severity *models.CodeableConcept) string {
	if severity == nil {
		return ""
	}

	for _, coding := range severity.Coding {
		code := strings.ToLower(coding.Code)
		switch code {
		case "severe", "24484000": // SNOMED: Severe
			return "severe"
		case "moderate", "6736007": // SNOMED: Moderate
			return "moderate"
		case "mild", "255604002": // SNOMED: Mild
			return "mild"
		}
	}

	// Check display text
	if severity.Text != "" {
		text := strings.ToLower(severity.Text)
		if strings.Contains(text, "severe") {
			return "severe"
		} else if strings.Contains(text, "moderate") {
			return "moderate"
		} else if strings.Contains(text, "mild") {
			return "mild"
		}
	}

	return ""
}

// ============================================================================
// Reference Utilities
// ============================================================================

// ExtractPatientIDFromReference extracts patient ID from a FHIR reference
func ExtractPatientIDFromReference(ref *models.Reference) string {
	if ref == nil || ref.Reference == "" {
		return ""
	}

	// Handle format: "Patient/123"
	if strings.HasPrefix(ref.Reference, "Patient/") {
		return strings.TrimPrefix(ref.Reference, "Patient/")
	}

	// Handle format: "urn:uuid:123"
	if strings.HasPrefix(ref.Reference, "urn:uuid:") {
		return strings.TrimPrefix(ref.Reference, "urn:uuid:")
	}

	// Handle absolute URL format
	if strings.Contains(ref.Reference, "/Patient/") {
		parts := strings.Split(ref.Reference, "/Patient/")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}

	return ref.Reference
}

// ExtractEncounterIDFromReference extracts encounter ID from a FHIR reference
func ExtractEncounterIDFromReference(ref *models.Reference) string {
	if ref == nil || ref.Reference == "" {
		return ""
	}

	// Handle format: "Encounter/123"
	if strings.HasPrefix(ref.Reference, "Encounter/") {
		return strings.TrimPrefix(ref.Reference, "Encounter/")
	}

	// Handle format: "urn:uuid:123"
	if strings.HasPrefix(ref.Reference, "urn:uuid:") {
		return strings.TrimPrefix(ref.Reference, "urn:uuid:")
	}

	// Handle absolute URL format
	if strings.Contains(ref.Reference, "/Encounter/") {
		parts := strings.Split(ref.Reference, "/Encounter/")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}

	return ""
}
