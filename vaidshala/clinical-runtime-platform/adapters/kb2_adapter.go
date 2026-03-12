// Package adapters provides adapter implementations that bridge existing
// knowledge base services to the frozen ClinicalExecutionContext contract.
//
// ARCHITECTURE: This follows the Adapter Pattern with Dependency Inversion.
// - We define INTERFACES for what we need from KB-2
// - KB-2 service IMPLEMENTS these interfaces
// - We do NOT depend on KB-2 internal types
// - We transform KB-2 outputs to frozen contract types
//
// WHY INTERFACE-BASED:
// - Dependency Inversion: Depend on abstractions, not concretions
// - Testable: Can inject test implementations
// - Decoupled: Changes to KB-2 internals don't break this adapter
// - Clean boundaries: Contract types are the shared language
package adapters

import (
	"context"
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// KB-2 SERVICE INTERFACES (Dependency Inversion)
// ============================================================================

// KB2ContextService defines what we need from the existing KB-2 service.
// The actual KB-2 service must implement this interface.
type KB2ContextService interface {
	// BuildPatientContext assembles patient data from FHIR input.
	// Returns raw patient context WITHOUT intelligence fields.
	BuildPatientContext(ctx context.Context, req KB2BuildRequest) (*KB2BuildResponse, error)
}

// KB2IntelligenceService defines KB-2B intelligence operations.
// The actual KB-2 service must implement this interface.
type KB2IntelligenceService interface {
	// DetectPhenotypes identifies clinical phenotypes from patient data.
	DetectPhenotypes(ctx context.Context, patientID string, data map[string]interface{}) ([]DetectedPhenotype, error)

	// AssessRisk calculates risk scores for the patient.
	AssessRisk(ctx context.Context, patientID string, data map[string]interface{}) (*RiskAssessmentResult, error)

	// IdentifyCareGaps finds gaps in patient care.
	IdentifyCareGaps(ctx context.Context, patientID string) ([]IdentifiedCareGap, error)
}

// ============================================================================
// KB-2 INPUT/OUTPUT TYPES (Interface Contracts)
// ============================================================================

// KB2BuildRequest is the input for BuildPatientContext.
type KB2BuildRequest struct {
	PatientID    string
	RawFHIRInput map[string]interface{}
}

// KB2BuildResponse is the output from BuildPatientContext.
type KB2BuildResponse struct {
	Demographics  KB2Demographics
	Conditions    []KB2Condition
	Medications   []KB2Medication
	LabResults    []KB2LabResult
	VitalSigns    []KB2VitalSign
	Allergies     []KB2Allergy
	Encounters    []KB2Encounter
}

// KB2Demographics from KB-2 service.
type KB2Demographics struct {
	PatientID string
	BirthDate *time.Time
	Gender    string
	AgeYears  int
}

// KB2Condition from KB-2 service.
type KB2Condition struct {
	System             string
	Code               string
	Display            string
	OnsetDate          *time.Time
	ClinicalStatus     string
	VerificationStatus string
	Severity           string
	SourceReference    string
}

// KB2Medication from KB-2 service.
type KB2Medication struct {
	System          string
	Code            string
	Display         string
	DoseText        string
	DoseValue       float64
	DoseUnit        string
	Frequency       string
	Route           string
	Status          string
	AuthoredOn      *time.Time
	SourceReference string
}

// KB2LabResult from KB-2 service.
type KB2LabResult struct {
	System            string
	Code              string
	Display           string
	Value             float64
	ValueString       string
	Unit              string
	Interpretation    string
	EffectiveDateTime *time.Time
	SourceReference   string
}

// KB2VitalSign from KB-2 service.
type KB2VitalSign struct {
	System            string
	Code              string
	Display           string
	Value             float64
	Unit              string
	Components        []KB2VitalComponent
	EffectiveDateTime *time.Time
	SourceReference   string
}

// KB2VitalComponent for composite vitals (e.g., BP).
type KB2VitalComponent struct {
	System  string
	Code    string
	Display string
	Value   float64
	Unit    string
}

// KB2Allergy from KB-2 service.
type KB2Allergy struct {
	System          string
	Code            string
	Display         string
	Category        string
	Criticality     string
	ClinicalStatus  string
	Reactions       []string
	SourceReference string
}

// KB2Encounter from KB-2 service.
type KB2Encounter struct {
	EncounterID     string
	TypeSystem      string
	TypeCode        string
	TypeDisplay     string
	Class           string
	Status          string
	PeriodStart     *time.Time
	PeriodEnd       *time.Time
	SourceReference string
}

// DetectedPhenotype from KB-2B phenotype detection.
type DetectedPhenotype struct {
	PhenotypeID string
	Name        string
	Confidence  float64
	Evidence    []string
}

// RiskAssessmentResult from KB-2B risk assessment.
type RiskAssessmentResult struct {
	RiskScores      map[string]float64
	RiskCategories  map[string]string
	ConfidenceScore float64
	ClinicalFlags   map[string]bool
}

// IdentifiedCareGap from KB-2B care gap analysis.
type IdentifiedCareGap struct {
	GapID             string
	MeasureID         string
	Description       string
	Priority          string
	RecommendedAction string
}

// ============================================================================
// KB-2A ADAPTER (Data Assembly - NO Intelligence)
// ============================================================================

// KB2Adapter wraps the existing KB-2 Clinical Context Service
// and produces frozen contract-compliant PatientContext (data-only).
//
// CRITICAL: This adapter does NOT run intelligence logic.
// It extracts raw patient assembly from KB-2 and strips any computed fields.
type KB2Adapter struct {
	// contextService is the KB-2 service implementing KB2ContextService
	contextService KB2ContextService

	// config controls adapter behavior
	config KB2AdapterConfig
}

// KB2AdapterConfig configures the adapter's behavior.
type KB2AdapterConfig struct {
	// LabLookbackDays how far back to fetch lab results (default: 90)
	LabLookbackDays int

	// VitalLookbackDays how far back to fetch vital signs (default: 30)
	VitalLookbackDays int

	// EncounterLookbackDays how far back to fetch encounters (default: 365)
	EncounterLookbackDays int

	// Region for regional adapter selection
	Region string
}

// DefaultKB2AdapterConfig returns sensible defaults.
func DefaultKB2AdapterConfig() KB2AdapterConfig {
	return KB2AdapterConfig{
		LabLookbackDays:       90,
		VitalLookbackDays:     30,
		EncounterLookbackDays: 365,
		Region:                "AU",
	}
}

// NewKB2Adapter creates a new adapter wrapping the KB-2 context service.
func NewKB2Adapter(contextService KB2ContextService, config KB2AdapterConfig) *KB2Adapter {
	return &KB2Adapter{
		contextService: contextService,
		config:         config,
	}
}

// AssemblePatientContext produces a data-only PatientContext.
// This is KB-2A (Assembly) - NO intelligence, NO risk scores, NO phenotypes.
//
// Implementation Strategy:
// 1. Call KB-2 BuildPatientContext() to leverage proven assembly logic
// 2. Convert KB-2 output types to frozen contract types
// 3. STRIP any intelligence fields (risk, phenotypes, care gaps)
// 4. Return pure canonical data
func (a *KB2Adapter) AssemblePatientContext(
	ctx context.Context,
	patientID string,
	rawFHIRInput map[string]interface{},
) (*contracts.PatientContext, error) {

	// Step 1: Build request for KB-2 service
	buildRequest := KB2BuildRequest{
		PatientID:    patientID,
		RawFHIRInput: rawFHIRInput,
	}

	// Step 2: Call KB-2 BuildPatientContext
	kb2Response, err := a.contextService.BuildPatientContext(ctx, buildRequest)
	if err != nil {
		return nil, fmt.Errorf("KB-2 build patient context failed: %w", err)
	}

	// Step 3: Convert KB-2 output to frozen contract
	canonicalContext := a.convertToCanonical(kb2Response, patientID)

	return canonicalContext, nil
}

// convertToCanonical transforms KB-2 response to frozen contract.
// CRITICAL: This function ensures intelligence fields are EMPTY.
func (a *KB2Adapter) convertToCanonical(resp *KB2BuildResponse, patientID string) *contracts.PatientContext {
	canonical := &contracts.PatientContext{
		// KB-2A: Raw patient data
		Demographics:      a.convertDemographics(&resp.Demographics, patientID),
		ActiveConditions:  a.convertConditions(resp.Conditions),
		ActiveMedications: a.convertMedications(resp.Medications),
		RecentLabResults:  a.convertLabs(resp.LabResults),
		RecentVitalSigns:  a.convertVitals(resp.VitalSigns),
		RecentEncounters:  a.convertEncounters(resp.Encounters),
		Allergies:         a.convertAllergies(resp.Allergies),

		// KB-2B: Intelligence fields - INTENTIONALLY EMPTY
		// These will be populated by KB2IntelligenceAdapter, not here
		RiskProfile: contracts.RiskProfile{
			ComputedRisks: make(map[string]contracts.RiskScore),
			ClinicalFlags: make(map[string]bool),
			ComputedAt:    time.Time{}, // Not computed by adapter
		},
		ClinicalSummary: contracts.ClinicalSummary{
			ProblemList:       []string{},
			MedicationSummary: "",
			CareGaps:          []contracts.CareGap{},
			GeneratedAt:       time.Time{}, // Not computed by adapter
		},

		// CQL export bundle not built by adapter
		CQLExportBundle: nil,
	}

	return canonical
}

// convertDemographics maps KB-2 Demographics to frozen contract.
func (a *KB2Adapter) convertDemographics(kb2 *KB2Demographics, patientID string) contracts.PatientDemographics {
	demo := contracts.PatientDemographics{
		PatientID: patientID,
		Gender:    kb2.Gender,
		Region:    a.config.Region,
	}

	// Use birth date if available, otherwise calculate from age
	if kb2.BirthDate != nil {
		demo.BirthDate = kb2.BirthDate
	} else if kb2.AgeYears > 0 {
		birthDate := time.Now().AddDate(-kb2.AgeYears, 0, 0)
		demo.BirthDate = &birthDate
	}

	return demo
}

// convertConditions maps KB-2 Conditions to frozen contract.
func (a *KB2Adapter) convertConditions(kb2Conditions []KB2Condition) []contracts.ClinicalCondition {
	result := make([]contracts.ClinicalCondition, 0, len(kb2Conditions))

	for _, c := range kb2Conditions {
		canonical := contracts.ClinicalCondition{
			Code: contracts.ClinicalCode{
				System:  c.System,
				Code:    c.Code,
				Display: c.Display,
			},
			OnsetDate:          c.OnsetDate,
			ClinicalStatus:     c.ClinicalStatus,
			VerificationStatus: c.VerificationStatus,
			Severity:           c.Severity,
			SourceReference:    c.SourceReference,
		}
		result = append(result, canonical)
	}

	return result
}

// convertMedications maps KB-2 Medications to frozen contract.
func (a *KB2Adapter) convertMedications(kb2Meds []KB2Medication) []contracts.Medication {
	result := make([]contracts.Medication, 0, len(kb2Meds))

	for _, m := range kb2Meds {
		canonical := contracts.Medication{
			Code: contracts.ClinicalCode{
				System:  m.System,
				Code:    m.Code,
				Display: m.Display,
			},
			Status:          m.Status,
			AuthoredOn:      m.AuthoredOn,
			SourceReference: m.SourceReference,
		}

		// Build dosage if available
		if m.DoseText != "" || m.DoseValue > 0 {
			canonical.Dosage = &contracts.Dosage{
				Text:      m.DoseText,
				Frequency: m.Frequency,
				Route:     m.Route,
			}
			if m.DoseValue > 0 {
				canonical.Dosage.DoseQuantity = &contracts.Quantity{
					Value: m.DoseValue,
					Unit:  m.DoseUnit,
				}
			}
		}

		result = append(result, canonical)
	}

	return result
}

// convertLabs maps KB-2 LabResults to frozen contract.
func (a *KB2Adapter) convertLabs(kb2Labs []KB2LabResult) []contracts.LabResult {
	result := make([]contracts.LabResult, 0, len(kb2Labs))

	for _, l := range kb2Labs {
		canonical := contracts.LabResult{
			Code: contracts.ClinicalCode{
				System:  l.System,
				Code:    l.Code,
				Display: l.Display,
			},
			ValueString:       l.ValueString,
			Interpretation:    l.Interpretation,
			EffectiveDateTime: l.EffectiveDateTime,
			SourceReference:   l.SourceReference,
		}

		if l.Value != 0 || l.Unit != "" {
			canonical.Value = &contracts.Quantity{
				Value: l.Value,
				Unit:  l.Unit,
			}
		}

		result = append(result, canonical)
	}

	return result
}

// convertVitals maps KB-2 VitalSigns to frozen contract.
func (a *KB2Adapter) convertVitals(kb2Vitals []KB2VitalSign) []contracts.VitalSign {
	result := make([]contracts.VitalSign, 0, len(kb2Vitals))

	for _, v := range kb2Vitals {
		canonical := contracts.VitalSign{
			Code: contracts.ClinicalCode{
				System:  v.System,
				Code:    v.Code,
				Display: v.Display,
			},
			EffectiveDateTime: v.EffectiveDateTime,
			SourceReference:   v.SourceReference,
		}

		if v.Value != 0 || v.Unit != "" {
			canonical.Value = &contracts.Quantity{
				Value: v.Value,
				Unit:  v.Unit,
			}
		}

		// Convert components (e.g., systolic/diastolic for BP)
		for _, comp := range v.Components {
			canonical.ComponentValues = append(canonical.ComponentValues, contracts.ComponentValue{
				Code: contracts.ClinicalCode{
					System:  comp.System,
					Code:    comp.Code,
					Display: comp.Display,
				},
				Value: &contracts.Quantity{
					Value: comp.Value,
					Unit:  comp.Unit,
				},
			})
		}

		result = append(result, canonical)
	}

	return result
}

// convertEncounters maps KB-2 Encounters to frozen contract.
func (a *KB2Adapter) convertEncounters(kb2Encounters []KB2Encounter) []contracts.Encounter {
	result := make([]contracts.Encounter, 0, len(kb2Encounters))

	for _, e := range kb2Encounters {
		canonical := contracts.Encounter{
			EncounterID: e.EncounterID,
			Type: []contracts.ClinicalCode{{
				System:  e.TypeSystem,
				Code:    e.TypeCode,
				Display: e.TypeDisplay,
			}},
			Class:           e.Class,
			Status:          e.Status,
			SourceReference: e.SourceReference,
		}

		if e.PeriodStart != nil || e.PeriodEnd != nil {
			canonical.Period = &contracts.Period{
				Start: e.PeriodStart,
				End:   e.PeriodEnd,
			}
		}

		result = append(result, canonical)
	}

	return result
}

// convertAllergies maps KB-2 Allergies to frozen contract.
func (a *KB2Adapter) convertAllergies(kb2Allergies []KB2Allergy) []contracts.Allergy {
	result := make([]contracts.Allergy, 0, len(kb2Allergies))

	for _, al := range kb2Allergies {
		canonical := contracts.Allergy{
			Code: contracts.ClinicalCode{
				System:  al.System,
				Code:    al.Code,
				Display: al.Display,
			},
			Category:        al.Category,
			Criticality:     al.Criticality,
			ClinicalStatus:  al.ClinicalStatus,
			SourceReference: al.SourceReference,
		}
		result = append(result, canonical)
	}

	return result
}

// ============================================================================
// KB-2B INTELLIGENCE ADAPTER
// ============================================================================

// KB2Intelligence defines the interface for KB-2B intelligence operations.
// This is what the ExecutionContextFactory calls to enrich PatientContext.
type KB2Intelligence interface {
	// Enrich adds KB-2B computed intelligence to a base PatientContext.
	// This runs: detectPhenotypes, calculateRiskScores, identifyCareGaps
	Enrich(ctx context.Context, base *contracts.PatientContext) (*contracts.PatientContext, error)
}

// KB2IntelligenceAdapter implements KB2Intelligence using KB2IntelligenceService.
type KB2IntelligenceAdapter struct {
	intelligenceService KB2IntelligenceService
}

// NewKB2IntelligenceAdapter creates a new intelligence adapter.
func NewKB2IntelligenceAdapter(service KB2IntelligenceService) *KB2IntelligenceAdapter {
	return &KB2IntelligenceAdapter{
		intelligenceService: service,
	}
}

// Enrich adds KB-2B computed intelligence to a base PatientContext.
// This method preserves all base data and adds:
// - RiskProfile (computed risk scores and clinical flags)
// - ClinicalSummary (problem list, medication summary, care gaps)
// - CQLExportBundle (FHIR Bundle formatted for CQL execution) [PRODUCTION REQUIRED]
//
// PRODUCTION REQUIREMENT:
// CQLExportBundle is built here for CQL engine consumption.
// Engines do NOT build bundles - they receive pre-computed data.
func (a *KB2IntelligenceAdapter) Enrich(
	ctx context.Context,
	base *contracts.PatientContext,
) (*contracts.PatientContext, error) {

	patientID := base.Demographics.PatientID

	// Convert canonical to map for KB-2B service
	patientData := a.canonicalToMap(base)

	// Call KB-2B intelligence methods
	phenotypes, err := a.intelligenceService.DetectPhenotypes(ctx, patientID, patientData)
	if err != nil {
		// Non-fatal: continue without phenotypes
		phenotypes = []DetectedPhenotype{}
	}

	risks, err := a.intelligenceService.AssessRisk(ctx, patientID, patientData)
	if err != nil {
		// Non-fatal: continue without risk scores
		risks = &RiskAssessmentResult{
			RiskScores:     make(map[string]float64),
			RiskCategories: make(map[string]string),
			ClinicalFlags:  make(map[string]bool),
		}
	}

	careGaps, err := a.intelligenceService.IdentifyCareGaps(ctx, patientID)
	if err != nil {
		// Non-fatal: continue without care gaps
		careGaps = []IdentifiedCareGap{}
	}

	// Clone base and add intelligence
	enriched := *base

	// Add risk profile
	enriched.RiskProfile = a.buildRiskProfile(risks, phenotypes)

	// Add clinical summary
	enriched.ClinicalSummary = a.buildClinicalSummary(careGaps, base)

	// PRODUCTION: Build CQLExportBundle for CQL engine consumption
	// This is REQUIRED - CQL engine will fail without it
	enriched.CQLExportBundle = a.buildCQLExportBundle(base)

	return &enriched, nil
}

// buildRiskProfile constructs RiskProfile from KB-2B results.
func (a *KB2IntelligenceAdapter) buildRiskProfile(
	risks *RiskAssessmentResult,
	phenotypes []DetectedPhenotype,
) contracts.RiskProfile {

	profile := contracts.RiskProfile{
		ComputedRisks: make(map[string]contracts.RiskScore),
		ClinicalFlags: make(map[string]bool),
		ComputedAt:    time.Now(),
	}

	// Convert risk scores
	for name, score := range risks.RiskScores {
		category := risks.RiskCategories[name]
		if category == "" {
			category = categorizeRisk(score)
		}
		profile.ComputedRisks[name] = contracts.RiskScore{
			Name:       name,
			Value:      score,
			Category:   category,
			Confidence: risks.ConfidenceScore,
		}
	}

	// Copy clinical flags
	for name, value := range risks.ClinicalFlags {
		profile.ClinicalFlags[name] = value
	}

	// Add phenotype-based flags
	for _, p := range phenotypes {
		if p.Confidence >= 0.7 { // High confidence threshold
			flagName := fmt.Sprintf("phenotype_%s", p.PhenotypeID)
			profile.ClinicalFlags[flagName] = true
		}
	}

	return profile
}

// buildClinicalSummary constructs ClinicalSummary from KB-2B results.
func (a *KB2IntelligenceAdapter) buildClinicalSummary(
	careGaps []IdentifiedCareGap,
	base *contracts.PatientContext,
) contracts.ClinicalSummary {

	summary := contracts.ClinicalSummary{
		GeneratedAt: time.Now(),
	}

	// Build problem list from active conditions
	for _, c := range base.ActiveConditions {
		if c.Code.Display != "" {
			summary.ProblemList = append(summary.ProblemList, c.Code.Display)
		} else {
			summary.ProblemList = append(summary.ProblemList, c.Code.Code)
		}
	}

	// Build medication summary
	if len(base.ActiveMedications) > 0 {
		summary.MedicationSummary = fmt.Sprintf(
			"Patient is on %d active medications",
			len(base.ActiveMedications),
		)
	}

	// Convert care gaps
	for _, gap := range careGaps {
		summary.CareGaps = append(summary.CareGaps, contracts.CareGap{
			MeasureID:         gap.MeasureID,
			Description:       gap.Description,
			Priority:          gap.Priority,
			RecommendedAction: gap.RecommendedAction,
		})
	}

	return summary
}

// ============================================================================
// CQL EXPORT BUNDLE BUILDING (Production Required)
// ============================================================================

// buildCQLExportBundle creates a FHIR R4 Bundle for CQL execution.
// This bundle contains all patient resources in the format expected by
// HAPI CQL evaluator and other CQL execution engines.
//
// FHIR COMPLIANCE:
// - Bundle.type = "collection" (per FHIR R4 spec)
// - All resources include proper coding with system/code/display
// - Dates are formatted as ISO 8601 (FHIR dateTime)
// - References use "ResourceType/id" format
func (a *KB2IntelligenceAdapter) buildCQLExportBundle(base *contracts.PatientContext) *contracts.CQLExportBundle {
	bundle := &contracts.CQLExportBundle{
		ResourceType: "Bundle",
		Type:         "collection",
		Entry:        make([]interface{}, 0),
	}

	patientID := base.Demographics.PatientID

	// 1. Patient resource (required for all CQL measures)
	patientResource := a.buildFHIRPatient(base.Demographics)
	bundle.Entry = append(bundle.Entry, map[string]interface{}{
		"fullUrl":  fmt.Sprintf("Patient/%s", patientID),
		"resource": patientResource,
	})

	// 2. Condition resources
	for i, cond := range base.ActiveConditions {
		condResource := a.buildFHIRCondition(cond, patientID, i)
		bundle.Entry = append(bundle.Entry, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("Condition/%s-cond-%d", patientID, i),
			"resource": condResource,
		})
	}

	// 3. MedicationRequest resources
	for i, med := range base.ActiveMedications {
		medResource := a.buildFHIRMedicationRequest(med, patientID, i)
		bundle.Entry = append(bundle.Entry, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("MedicationRequest/%s-med-%d", patientID, i),
			"resource": medResource,
		})
	}

	// 4. Observation resources (labs)
	for i, lab := range base.RecentLabResults {
		obsResource := a.buildFHIRObservationLab(lab, patientID, i)
		bundle.Entry = append(bundle.Entry, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("Observation/%s-lab-%d", patientID, i),
			"resource": obsResource,
		})
	}

	// 5. Observation resources (vitals)
	for i, vital := range base.RecentVitalSigns {
		obsResource := a.buildFHIRObservationVital(vital, patientID, i)
		bundle.Entry = append(bundle.Entry, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("Observation/%s-vital-%d", patientID, i),
			"resource": obsResource,
		})
	}

	// 6. Encounter resources
	for i, enc := range base.RecentEncounters {
		encResource := a.buildFHIREncounter(enc, patientID, i)
		bundle.Entry = append(bundle.Entry, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("Encounter/%s-enc-%d", patientID, i),
			"resource": encResource,
		})
	}

	// 7. AllergyIntolerance resources
	for i, allergy := range base.Allergies {
		allergyResource := a.buildFHIRAllergyIntolerance(allergy, patientID, i)
		bundle.Entry = append(bundle.Entry, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("AllergyIntolerance/%s-allergy-%d", patientID, i),
			"resource": allergyResource,
		})
	}

	return bundle
}

// buildFHIRPatient creates a FHIR R4 Patient resource.
func (a *KB2IntelligenceAdapter) buildFHIRPatient(demo contracts.PatientDemographics) map[string]interface{} {
	patient := map[string]interface{}{
		"resourceType": "Patient",
		"id":           demo.PatientID,
	}

	if demo.Gender != "" {
		patient["gender"] = demo.Gender
	}

	if demo.BirthDate != nil {
		patient["birthDate"] = demo.BirthDate.Format("2006-01-02")
	}

	// Add extension for region (useful for regional CQL measures)
	if demo.Region != "" {
		patient["extension"] = []map[string]interface{}{
			{
				"url":         "http://cardiofit.health/fhir/StructureDefinition/patient-region",
				"valueString": demo.Region,
			},
		}
	}

	return patient
}

// buildFHIRCondition creates a FHIR R4 Condition resource.
func (a *KB2IntelligenceAdapter) buildFHIRCondition(cond contracts.ClinicalCondition, patientID string, index int) map[string]interface{} {
	condition := map[string]interface{}{
		"resourceType": "Condition",
		"id":           fmt.Sprintf("%s-cond-%d", patientID, index),
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  cond.Code.System,
					"code":    cond.Code.Code,
					"display": cond.Code.Display,
				},
			},
		},
	}

	// Clinical status (required for CQL filtering)
	if cond.ClinicalStatus != "" {
		condition["clinicalStatus"] = map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
					"code":   cond.ClinicalStatus,
				},
			},
		}
	}

	// Verification status
	if cond.VerificationStatus != "" {
		condition["verificationStatus"] = map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
					"code":   cond.VerificationStatus,
				},
			},
		}
	}

	// Onset date (important for CQL measure period calculations)
	if cond.OnsetDate != nil {
		condition["onsetDateTime"] = cond.OnsetDate.Format(time.RFC3339)
	}

	// Severity
	if cond.Severity != "" {
		condition["severity"] = map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system": "http://snomed.info/sct",
					"code":   mapSeverityToSNOMED(cond.Severity),
				},
			},
		}
	}

	return condition
}

// buildFHIRMedicationRequest creates a FHIR R4 MedicationRequest resource.
func (a *KB2IntelligenceAdapter) buildFHIRMedicationRequest(med contracts.Medication, patientID string, index int) map[string]interface{} {
	medRequest := map[string]interface{}{
		"resourceType": "MedicationRequest",
		"id":           fmt.Sprintf("%s-med-%d", patientID, index),
		"status":       med.Status,
		"intent":       "order",
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"medicationCodeableConcept": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  med.Code.System,
					"code":    med.Code.Code,
					"display": med.Code.Display,
				},
			},
		},
	}

	// Authored date (important for CQL active medication filtering)
	if med.AuthoredOn != nil {
		medRequest["authoredOn"] = med.AuthoredOn.Format(time.RFC3339)
	}

	// Dosage instructions
	if med.Dosage != nil {
		dosageInstruction := map[string]interface{}{}

		if med.Dosage.Text != "" {
			dosageInstruction["text"] = med.Dosage.Text
		}

		if med.Dosage.DoseQuantity != nil {
			dosageInstruction["doseAndRate"] = []map[string]interface{}{
				{
					"doseQuantity": map[string]interface{}{
						"value": med.Dosage.DoseQuantity.Value,
						"unit":  med.Dosage.DoseQuantity.Unit,
					},
				},
			}
		}

		if med.Dosage.Route != "" {
			dosageInstruction["route"] = map[string]interface{}{
				"text": med.Dosage.Route,
			}
		}

		if len(dosageInstruction) > 0 {
			medRequest["dosageInstruction"] = []map[string]interface{}{dosageInstruction}
		}
	}

	return medRequest
}

// buildFHIRObservationLab creates a FHIR R4 Observation (laboratory) resource.
func (a *KB2IntelligenceAdapter) buildFHIRObservationLab(lab contracts.LabResult, patientID string, index int) map[string]interface{} {
	obs := map[string]interface{}{
		"resourceType": "Observation",
		"id":           fmt.Sprintf("%s-lab-%d", patientID, index),
		"status":       "final",
		"category": []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{
						"system": "http://terminology.hl7.org/CodeSystem/observation-category",
						"code":   "laboratory",
					},
				},
			},
		},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  lab.Code.System,
					"code":    lab.Code.Code,
					"display": lab.Code.Display,
				},
			},
		},
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
	}

	// Effective date (CRITICAL for CQL measure period filtering)
	if lab.EffectiveDateTime != nil {
		obs["effectiveDateTime"] = lab.EffectiveDateTime.Format(time.RFC3339)
	}

	// Value (numeric)
	if lab.Value != nil {
		obs["valueQuantity"] = map[string]interface{}{
			"value":  lab.Value.Value,
			"unit":   lab.Value.Unit,
			"system": "http://unitsofmeasure.org",
			"code":   lab.Value.Code,
		}
	} else if lab.ValueString != "" {
		obs["valueString"] = lab.ValueString
	}

	// Interpretation
	if lab.Interpretation != "" {
		obs["interpretation"] = []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{
						"system": "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
						"code":   mapInterpretationCode(lab.Interpretation),
					},
				},
			},
		}
	}

	// Reference range
	if lab.ReferenceRange != nil {
		refRange := map[string]interface{}{}
		if lab.ReferenceRange.Low != nil {
			refRange["low"] = map[string]interface{}{
				"value": lab.ReferenceRange.Low.Value,
				"unit":  lab.ReferenceRange.Low.Unit,
			}
		}
		if lab.ReferenceRange.High != nil {
			refRange["high"] = map[string]interface{}{
				"value": lab.ReferenceRange.High.Value,
				"unit":  lab.ReferenceRange.High.Unit,
			}
		}
		if lab.ReferenceRange.Text != "" {
			refRange["text"] = lab.ReferenceRange.Text
		}
		if len(refRange) > 0 {
			obs["referenceRange"] = []map[string]interface{}{refRange}
		}
	}

	return obs
}

// buildFHIRObservationVital creates a FHIR R4 Observation (vital-signs) resource.
func (a *KB2IntelligenceAdapter) buildFHIRObservationVital(vital contracts.VitalSign, patientID string, index int) map[string]interface{} {
	obs := map[string]interface{}{
		"resourceType": "Observation",
		"id":           fmt.Sprintf("%s-vital-%d", patientID, index),
		"status":       "final",
		"category": []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{
						"system": "http://terminology.hl7.org/CodeSystem/observation-category",
						"code":   "vital-signs",
					},
				},
			},
		},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  vital.Code.System,
					"code":    vital.Code.Code,
					"display": vital.Code.Display,
				},
			},
		},
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
	}

	// Effective date
	if vital.EffectiveDateTime != nil {
		obs["effectiveDateTime"] = vital.EffectiveDateTime.Format(time.RFC3339)
	}

	// Simple value
	if vital.Value != nil {
		obs["valueQuantity"] = map[string]interface{}{
			"value":  vital.Value.Value,
			"unit":   vital.Value.Unit,
			"system": "http://unitsofmeasure.org",
		}
	}

	// Component values (for composite vitals like BP)
	if len(vital.ComponentValues) > 0 {
		components := make([]map[string]interface{}, 0, len(vital.ComponentValues))
		for _, comp := range vital.ComponentValues {
			component := map[string]interface{}{
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  comp.Code.System,
							"code":    comp.Code.Code,
							"display": comp.Code.Display,
						},
					},
				},
			}
			if comp.Value != nil {
				component["valueQuantity"] = map[string]interface{}{
					"value":  comp.Value.Value,
					"unit":   comp.Value.Unit,
					"system": "http://unitsofmeasure.org",
				}
			}
			components = append(components, component)
		}
		obs["component"] = components
	}

	return obs
}

// buildFHIREncounter creates a FHIR R4 Encounter resource.
func (a *KB2IntelligenceAdapter) buildFHIREncounter(enc contracts.Encounter, patientID string, index int) map[string]interface{} {
	encounter := map[string]interface{}{
		"resourceType": "Encounter",
		"id":           enc.EncounterID,
		"status":       enc.Status,
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
	}

	// Class (ambulatory, inpatient, etc.)
	if enc.Class != "" {
		encounter["class"] = map[string]interface{}{
			"system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			"code":   enc.Class,
		}
	}

	// Type
	if len(enc.Type) > 0 {
		typeCoding := make([]map[string]interface{}, 0, len(enc.Type))
		for _, t := range enc.Type {
			typeCoding = append(typeCoding, map[string]interface{}{
				"coding": []map[string]interface{}{
					{
						"system":  t.System,
						"code":    t.Code,
						"display": t.Display,
					},
				},
			})
		}
		encounter["type"] = typeCoding
	}

	// Period (CRITICAL for CQL measure period filtering)
	if enc.Period != nil {
		period := map[string]interface{}{}
		if enc.Period.Start != nil {
			period["start"] = enc.Period.Start.Format(time.RFC3339)
		}
		if enc.Period.End != nil {
			period["end"] = enc.Period.End.Format(time.RFC3339)
		}
		if len(period) > 0 {
			encounter["period"] = period
		}
	}

	return encounter
}

// buildFHIRAllergyIntolerance creates a FHIR R4 AllergyIntolerance resource.
func (a *KB2IntelligenceAdapter) buildFHIRAllergyIntolerance(allergy contracts.Allergy, patientID string, index int) map[string]interface{} {
	allergyResource := map[string]interface{}{
		"resourceType": "AllergyIntolerance",
		"id":           fmt.Sprintf("%s-allergy-%d", patientID, index),
		"patient": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  allergy.Code.System,
					"code":    allergy.Code.Code,
					"display": allergy.Code.Display,
				},
			},
		},
	}

	// Clinical status
	if allergy.ClinicalStatus != "" {
		allergyResource["clinicalStatus"] = map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system": "http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical",
					"code":   allergy.ClinicalStatus,
				},
			},
		}
	}

	// Category
	if allergy.Category != "" {
		allergyResource["category"] = []string{allergy.Category}
	}

	// Criticality
	if allergy.Criticality != "" {
		allergyResource["criticality"] = allergy.Criticality
	}

	return allergyResource
}

// mapSeverityToSNOMED maps severity string to SNOMED code.
func mapSeverityToSNOMED(severity string) string {
	switch severity {
	case "mild":
		return "255604002"
	case "moderate":
		return "6736007"
	case "severe":
		return "24484000"
	default:
		return "6736007" // Default to moderate
	}
}

// mapInterpretationCode maps interpretation string to HL7 code.
func mapInterpretationCode(interpretation string) string {
	switch interpretation {
	case "normal", "N":
		return "N"
	case "high", "H":
		return "H"
	case "low", "L":
		return "L"
	case "critical", "critical-high", "HH":
		return "HH"
	case "critical-low", "LL":
		return "LL"
	case "abnormal", "A":
		return "A"
	default:
		return interpretation
	}
}

// canonicalToMap converts canonical PatientContext to map for KB-2B service.
func (a *KB2IntelligenceAdapter) canonicalToMap(canonical *contracts.PatientContext) map[string]interface{} {
	data := make(map[string]interface{})

	// Demographics
	demographics := map[string]interface{}{
		"patient_id": canonical.Demographics.PatientID,
		"gender":     canonical.Demographics.Gender,
		"region":     canonical.Demographics.Region,
	}
	if canonical.Demographics.BirthDate != nil {
		demographics["birth_date"] = canonical.Demographics.BirthDate.Format(time.RFC3339)
		age := time.Now().Year() - canonical.Demographics.BirthDate.Year()
		demographics["age_years"] = age
	}
	data["demographics"] = demographics

	// Conditions
	conditions := make([]map[string]interface{}, 0, len(canonical.ActiveConditions))
	for _, c := range canonical.ActiveConditions {
		conditions = append(conditions, map[string]interface{}{
			"system":   c.Code.System,
			"code":     c.Code.Code,
			"display":  c.Code.Display,
			"severity": c.Severity,
		})
	}
	data["conditions"] = conditions

	// Medications
	medications := make([]map[string]interface{}, 0, len(canonical.ActiveMedications))
	for _, m := range canonical.ActiveMedications {
		med := map[string]interface{}{
			"system":  m.Code.System,
			"code":    m.Code.Code,
			"display": m.Code.Display,
		}
		if m.Dosage != nil {
			med["dose_text"] = m.Dosage.Text
			med["frequency"] = m.Dosage.Frequency
		}
		medications = append(medications, med)
	}
	data["medications"] = medications

	// Labs
	labs := make([]map[string]interface{}, 0, len(canonical.RecentLabResults))
	for _, l := range canonical.RecentLabResults {
		lab := map[string]interface{}{
			"system":  l.Code.System,
			"code":    l.Code.Code,
			"display": l.Code.Display,
		}
		if l.Value != nil {
			lab["value"] = l.Value.Value
			lab["unit"] = l.Value.Unit
		}
		if l.EffectiveDateTime != nil {
			lab["effective_date"] = l.EffectiveDateTime.Format(time.RFC3339)
		}
		labs = append(labs, lab)
	}
	data["lab_results"] = labs

	return data
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// categorizeRisk converts numeric risk score to category.
func categorizeRisk(score float64) string {
	switch {
	case score >= 0.75:
		return "critical"
	case score >= 0.5:
		return "high"
	case score >= 0.25:
		return "moderate"
	default:
		return "low"
	}
}
