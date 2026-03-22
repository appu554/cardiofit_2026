package fhir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationSeverity indicates how serious a FHIR validation finding is.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
)

// ValidationIssue describes a single FHIR IG conformance finding.
type ValidationIssue struct {
	Severity ValidationSeverity `json:"severity"`
	Profile  string             `json:"profile"`
	Path     string             `json:"path"`
	Message  string             `json:"message"`
}

// ValidationResult holds the aggregate result of FHIR validation.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues,omitempty"`
}

// ABDM IG v7.0 profile URLs.
const (
	ProfilePatientIN              = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/Patient"
	ProfileObservationVitalSignsIN = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/ObservationVitalSignsIN"
	ProfileDiagnosticReportLabIN  = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/DiagnosticReportLabIN"
	ProfileMedicationStatementIN  = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/MedicationStatementIN"
	ProfileConditionIN            = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/Condition"
	ProfileEncounterIN            = "https://nrces.in/ndhm/fhir/r4/StructureDefinition/Encounter"
)

// requiredField defines a field that must be present for a given resource type.
type requiredField struct {
	path    string // JSON pointer-like dot path, e.g. "subject.reference"
	message string
}

// profileRules maps resourceType to the required fields for ABDM IG v7.0 conformance.
var profileRules = map[string][]requiredField{
	"Observation": {
		{path: "resourceType", message: "resourceType must be Observation"},
		{path: "status", message: "status is required (registered|preliminary|final|amended)"},
		{path: "code", message: "code is required (LOINC coded)"},
		{path: "code.coding", message: "code.coding is required with at least one LOINC entry"},
		{path: "subject", message: "subject (Patient reference) is required"},
		{path: "subject.reference", message: "subject.reference must be a Patient reference"},
	},
	"DiagnosticReport": {
		{path: "resourceType", message: "resourceType must be DiagnosticReport"},
		{path: "status", message: "status is required"},
		{path: "code", message: "code is required"},
		{path: "subject", message: "subject (Patient reference) is required"},
	},
	"MedicationStatement": {
		{path: "resourceType", message: "resourceType must be MedicationStatement"},
		{path: "status", message: "status is required"},
		{path: "medicationCodeableConcept", message: "medicationCodeableConcept is required"},
		{path: "subject", message: "subject (Patient reference) is required"},
	},
	"Patient": {
		{path: "resourceType", message: "resourceType must be Patient"},
		{path: "name", message: "at least one name is required"},
	},
}

// ValidateFHIRResource checks a FHIR JSON resource against ABDM IG v7.0 profile rules.
// It performs structural validation (required fields, coding system presence).
// This is NOT a full FHIRPath engine — it covers the critical conformance checks that
// would cause Google FHIR Store to reject a resource.
func ValidateFHIRResource(resourceJSON []byte) ValidationResult {
	var resource map[string]interface{}
	if err := json.Unmarshal(resourceJSON, &resource); err != nil {
		return ValidationResult{
			Valid: false,
			Issues: []ValidationIssue{{
				Severity: SeverityError,
				Path:     "$",
				Message:  "invalid JSON: " + err.Error(),
			}},
		}
	}

	resourceType, _ := resource["resourceType"].(string)
	if resourceType == "" {
		return ValidationResult{
			Valid: false,
			Issues: []ValidationIssue{{
				Severity: SeverityError,
				Path:     "resourceType",
				Message:  "resourceType is missing",
			}},
		}
	}

	rules, ok := profileRules[resourceType]
	if !ok {
		// No specific rules for this resource type — pass
		return ValidationResult{Valid: true}
	}

	profile := profileForResource(resourceType)
	var issues []ValidationIssue

	for _, rule := range rules {
		if !hasField(resource, rule.path) {
			issues = append(issues, ValidationIssue{
				Severity: SeverityError,
				Profile:  profile,
				Path:     rule.path,
				Message:  rule.message,
			})
		}
	}

	// LOINC coding system check for Observation
	if resourceType == "Observation" {
		if !hasLOINCCoding(resource) {
			issues = append(issues, ValidationIssue{
				Severity: SeverityWarning,
				Profile:  ProfileObservationVitalSignsIN,
				Path:     "code.coding",
				Message:  "no LOINC coding found (system http://loinc.org expected)",
			})
		}
	}

	// Subject reference format check
	if subj, ok := resource["subject"].(map[string]interface{}); ok {
		if ref, ok := subj["reference"].(string); ok {
			if !strings.HasPrefix(ref, "Patient/") {
				issues = append(issues, ValidationIssue{
					Severity: SeverityError,
					Profile:  profile,
					Path:     "subject.reference",
					Message:  fmt.Sprintf("subject.reference must start with Patient/, got %q", ref),
				})
			}
		}
	}

	return ValidationResult{
		Valid:  len(issues) == 0,
		Issues: issues,
	}
}

// profileForResource returns the ABDM IG profile URL for a given resource type.
func profileForResource(resourceType string) string {
	switch resourceType {
	case "Observation":
		return ProfileObservationVitalSignsIN
	case "DiagnosticReport":
		return ProfileDiagnosticReportLabIN
	case "MedicationStatement":
		return ProfileMedicationStatementIN
	case "Patient":
		return ProfilePatientIN
	case "Condition":
		return ProfileConditionIN
	case "Encounter":
		return ProfileEncounterIN
	default:
		return ""
	}
}

// hasField checks for a nested field using dot notation.
func hasField(obj map[string]interface{}, path string) bool {
	parts := strings.Split(path, ".")
	current := obj
	for i, part := range parts {
		val, ok := current[part]
		if !ok || val == nil {
			return false
		}
		if i < len(parts)-1 {
			switch v := val.(type) {
			case map[string]interface{}:
				current = v
			case []interface{}:
				if len(v) == 0 {
					return false
				}
				// Check first element
				if m, ok := v[0].(map[string]interface{}); ok {
					current = m
				} else {
					return true // array of primitives — field exists
				}
			default:
				return true // primitive value at non-leaf — unexpected but exists
			}
		}
	}
	return true
}

// hasLOINCCoding checks whether an Observation has at least one coding with system "http://loinc.org".
func hasLOINCCoding(resource map[string]interface{}) bool {
	code, ok := resource["code"].(map[string]interface{})
	if !ok {
		return false
	}
	codings, ok := code["coding"].([]interface{})
	if !ok {
		return false
	}
	for _, c := range codings {
		if cm, ok := c.(map[string]interface{}); ok {
			if sys, _ := cm["system"].(string); sys == "http://loinc.org" {
				return true
			}
		}
	}
	return false
}
