// Package fhir provides resource parsing utilities for FHIR bundles.
package fhir

import (
	"encoding/json"
)

// ============================================================================
// Bundle Entry Parsing
// ============================================================================

// parsePatient extracts a Patient from a raw map.
func parsePatient(raw map[string]interface{}) *Patient {
	if raw == nil {
		return nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var patient Patient
	if err := json.Unmarshal(data, &patient); err != nil {
		return nil
	}

	return &patient
}

// parseConditions extracts Conditions from a Bundle.
func parseConditions(bundle *Bundle) []Condition {
	if bundle == nil {
		return nil
	}

	var conditions []Condition
	for _, entry := range bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		data, err := json.Marshal(entry.Resource)
		if err != nil {
			continue
		}

		var cond Condition
		if err := json.Unmarshal(data, &cond); err != nil {
			continue
		}

		if cond.ResourceType == "Condition" {
			conditions = append(conditions, cond)
		}
	}

	return conditions
}

// parseObservations extracts Observations from a Bundle.
func parseObservations(bundle *Bundle) []Observation {
	if bundle == nil {
		return nil
	}

	var observations []Observation
	for _, entry := range bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		data, err := json.Marshal(entry.Resource)
		if err != nil {
			continue
		}

		var obs Observation
		if err := json.Unmarshal(data, &obs); err != nil {
			continue
		}

		if obs.ResourceType == "Observation" {
			observations = append(observations, obs)
		}
	}

	return observations
}

// parseProcedures extracts Procedures from a Bundle.
func parseProcedures(bundle *Bundle) []Procedure {
	if bundle == nil {
		return nil
	}

	var procedures []Procedure
	for _, entry := range bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		data, err := json.Marshal(entry.Resource)
		if err != nil {
			continue
		}

		var proc Procedure
		if err := json.Unmarshal(data, &proc); err != nil {
			continue
		}

		if proc.ResourceType == "Procedure" {
			procedures = append(procedures, proc)
		}
	}

	return procedures
}

// parseMedicationRequests extracts MedicationRequests from a Bundle.
func parseMedicationRequests(bundle *Bundle) []MedicationRequest {
	if bundle == nil {
		return nil
	}

	var medications []MedicationRequest
	for _, entry := range bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		data, err := json.Marshal(entry.Resource)
		if err != nil {
			continue
		}

		var med MedicationRequest
		if err := json.Unmarshal(data, &med); err != nil {
			continue
		}

		if med.ResourceType == "MedicationRequest" {
			medications = append(medications, med)
		}
	}

	return medications
}

// parseImmunizations extracts Immunizations from a Bundle.
func parseImmunizations(bundle *Bundle) []Immunization {
	if bundle == nil {
		return nil
	}

	var immunizations []Immunization
	for _, entry := range bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		data, err := json.Marshal(entry.Resource)
		if err != nil {
			continue
		}

		var imm Immunization
		if err := json.Unmarshal(data, &imm); err != nil {
			continue
		}

		if imm.ResourceType == "Immunization" {
			immunizations = append(immunizations, imm)
		}
	}

	return immunizations
}

// parseEncounters extracts Encounters from a Bundle.
func parseEncounters(bundle *Bundle) []Encounter {
	if bundle == nil {
		return nil
	}

	var encounters []Encounter
	for _, entry := range bundle.Entry {
		if entry.Resource == nil {
			continue
		}

		data, err := json.Marshal(entry.Resource)
		if err != nil {
			continue
		}

		var enc Encounter
		if err := json.Unmarshal(data, &enc); err != nil {
			continue
		}

		if enc.ResourceType == "Encounter" {
			encounters = append(encounters, enc)
		}
	}

	return encounters
}

// ============================================================================
// Code System Constants
// ============================================================================

// Common FHIR code systems.
const (
	// LOINC - Logical Observation Identifiers Names and Codes
	SystemLOINC = "http://loinc.org"

	// SNOMED CT - Clinical terminology
	SystemSNOMED = "http://snomed.info/sct"

	// ICD-10-CM - Diagnosis codes
	SystemICD10CM = "http://hl7.org/fhir/sid/icd-10-cm"

	// CPT - Procedure codes
	SystemCPT = "http://www.ama-assn.org/go/cpt"

	// RxNorm - Medication codes
	SystemRxNorm = "http://www.nlm.nih.gov/research/umls/rxnorm"

	// CVX - Vaccine codes
	SystemCVX = "http://hl7.org/fhir/sid/cvx"

	// UCUM - Units of measure
	SystemUCUM = "http://unitsofmeasure.org"
)

// Common LOINC codes for quality measures.
const (
	// HbA1c
	LOINCHbA1c = "4548-4"

	// Blood Pressure
	LOINCSystolicBP  = "8480-6"
	LOINCDiastolicBP = "8462-4"

	// BMI
	LOINCBMI = "39156-5"

	// PHQ-9 Depression Screening
	LOINCPHQ9 = "44261-6"

	// PHQ-2 Depression Screening
	LOINCPHQ2 = "55758-7"

	// Tobacco Use
	LOINCTobaccoUse = "72166-2"

	// Fecal Occult Blood (FIT/FOBT)
	LOINCFOBT = "29771-3"

	// Lipid Panel
	LOINCTotalCholesterol = "2093-3"
	LOINCLDL              = "2089-1"
	LOINCHDL              = "2085-9"
	LOINCTriglycerides    = "2571-8"

	// eGFR
	LOINCeGFR = "33914-3"

	// Creatinine
	LOINCCreatinine = "2160-0"
)

// Common SNOMED codes for conditions.
const (
	// Diabetes
	SNOMEDDiabetesType2 = "44054006"
	SNOMEDDiabetesType1 = "46635009"

	// Hypertension
	SNOMEDHypertension = "38341003"

	// Depression
	SNOMEDDepression = "35489007"

	// CKD
	SNOMEDCKD = "709044004"
)

// ============================================================================
// Helper Functions
// ============================================================================

// HasCode checks if a CodeableConcept contains a specific code.
func HasCode(cc *CodeableConcept, system, code string) bool {
	if cc == nil {
		return false
	}
	for _, coding := range cc.Coding {
		if coding.System == system && coding.Code == code {
			return true
		}
	}
	return false
}

// GetCodeValue extracts the first code value from a CodeableConcept.
func GetCodeValue(cc *CodeableConcept) string {
	if cc == nil || len(cc.Coding) == 0 {
		return ""
	}
	return cc.Coding[0].Code
}

// GetCodeDisplay extracts the display text from a CodeableConcept.
func GetCodeDisplay(cc *CodeableConcept) string {
	if cc == nil {
		return ""
	}
	if cc.Text != "" {
		return cc.Text
	}
	if len(cc.Coding) > 0 && cc.Coding[0].Display != "" {
		return cc.Coding[0].Display
	}
	return ""
}

// ExtractReferenceID extracts the resource ID from a FHIR reference.
// e.g., "Patient/12345" -> "12345"
func ExtractReferenceID(ref *Reference) string {
	if ref == nil || ref.Reference == "" {
		return ""
	}

	// Simple extraction - find the last '/'
	for i := len(ref.Reference) - 1; i >= 0; i-- {
		if ref.Reference[i] == '/' {
			return ref.Reference[i+1:]
		}
	}
	return ref.Reference
}
