// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// These tests prove that recommendations (KB-19) are always filtered by governance (KB-18).
package tests

import (
	"context"
	"time"

	"kb-18-governance-engine/pkg/engine"
	"kb-18-governance-engine/pkg/programs"
	"kb-18-governance-engine/pkg/types"

	"github.com/google/uuid"
)

// =============================================================================
// E2E TEST HELPERS
// These helpers simulate KB-19 recommendations flowing through KB-18 governance.
// Architectural invariant: KB-19 may recommend, KB-18 decides.
// =============================================================================

// E2ETestContext holds shared context for E2E tests
type E2ETestContext struct {
	Engine       *engine.GovernanceEngine
	ProgramStore *programs.ProgramStore
	Ctx          context.Context
}

// NewE2ETestContext creates a new E2E test context
func NewE2ETestContext() *E2ETestContext {
	store := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(store)
	return &E2ETestContext{
		Engine:       eng,
		ProgramStore: store,
		Ctx:          context.Background(),
	}
}

// =============================================================================
// KB-19 RECOMMENDATION SIMULATION
// =============================================================================

// RecommendationType mirrors KB-19 DecisionType
type RecommendationType string

const (
	RecommendDo      RecommendationType = "DO"
	RecommendDelay   RecommendationType = "DELAY"
	RecommendAvoid   RecommendationType = "AVOID"
	RecommendConsider RecommendationType = "CONSIDER"
)

// EvidenceClass represents clinical evidence classification
type EvidenceClass string

const (
	ClassI   EvidenceClass = "CLASS_I"   // Benefit >>> Risk, SHOULD do
	ClassIIa EvidenceClass = "CLASS_IIA" // Benefit >> Risk, reasonable to do
	ClassIIb EvidenceClass = "CLASS_IIB" // Benefit >= Risk, may consider
	ClassIII EvidenceClass = "CLASS_III" // Risk > Benefit, SHOULD NOT do
)

// SimulatedRecommendation represents a KB-19 arbitrated decision
type SimulatedRecommendation struct {
	ID               uuid.UUID
	Target           string
	TargetRxNorm     string
	DrugClass        string
	RecommendedDose  float64
	DoseUnit         string
	RecommendationType RecommendationType
	EvidenceClass    EvidenceClass
	SourceProtocol   string
	Rationale        string
	Urgency          string // STAT, URGENT, ROUTINE, SCHEDULED
}

// E2EScenarioResult captures the complete E2E flow result
type E2EScenarioResult struct {
	// KB-19 input
	Recommendation SimulatedRecommendation

	// KB-18 output
	GovernanceResponse *types.EvaluationResponse

	// Final state
	FinalAllowed       bool
	FinalOutcome       types.Outcome
	EnforcementApplied types.EnforcementLevel
	ViolationCount     int
	RequiresOverride   bool
	RequiresEscalation bool
	EvidenceTrailHash  string
}

// =============================================================================
// PATIENT CONTEXT BUILDERS
// =============================================================================

// SepticShockPatient creates a patient in septic shock
func SepticShockPatient() *types.PatientContext {
	return &types.PatientContext{
		PatientID:  "PT-E2E-SEPSIS",
		Age:        58,
		Sex:        "M",
		IsPregnant: false,
		Weight:     75.0,
		Vitals: &types.Vitals{
			SystolicBP:  85, // Hypotensive (MAP ~55)
			DiastolicBP: 45,
			HeartRate:   115,
			Temperature: 38.9,
			SpO2:        92,
		},
		RecentLabs: []types.LabResult{
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 5.2, Unit: "mmol/L", Timestamp: time.Now().Add(-1 * time.Hour)},
			{Code: "6690-2", CodeSystem: "LOINC", Name: "White Blood Cells", Value: 18.5, Unit: "K/uL", Timestamp: time.Now().Add(-2 * time.Hour)},
			{Code: "2160-0", CodeSystem: "LOINC", Name: "Creatinine", Value: 1.8, Unit: "mg/dL", Timestamp: time.Now().Add(-1 * time.Hour)},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis, unspecified organism"},
			{Code: "J18.9", CodeSystem: "ICD10", Description: "Pneumonia"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "SEPSIS", Status: "ACTIVE"},
			{RegistryCode: "ICU", Status: "ACTIVE"},
		},
	}
}

// SepticShockWithHFPatient creates a septic patient with heart failure
func SepticShockWithHFPatient() *types.PatientContext {
	patient := SepticShockPatient()
	patient.PatientID = "PT-E2E-SEPSIS-HF"
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "I50.9",
		CodeSystem:  "ICD10",
		Description: "Heart failure, unspecified",
	})
	patient.RegistryMemberships = append(patient.RegistryMemberships, types.RegistryMembership{
		RegistryCode: "CHF",
		Status:       "ACTIVE",
	})
	// Add EF to context
	return patient
}

// AFibWithThrombocytopeniaPatient creates AFib patient with low platelets
func AFibWithThrombocytopeniaPatient() *types.PatientContext {
	return &types.PatientContext{
		PatientID:  "PT-E2E-AFIB-TCP",
		Age:        72,
		Sex:        "F",
		IsPregnant: false,
		Weight:     68.0,
		RecentLabs: []types.LabResult{
			{Code: "PLT", Name: "Platelets", Value: 38.0, Unit: "K/uL", Timestamp: time.Now().Add(-2 * time.Hour)},
			{Code: "INR", Name: "INR", Value: 1.1, Unit: "", Timestamp: time.Now().Add(-2 * time.Hour)},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I48.91", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
			{Code: "I10", CodeSystem: "ICD10", Description: "Hypertension"},
			{Code: "E11.9", CodeSystem: "ICD10", Description: "Type 2 diabetes"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
		},
	}
}

// PregnantPatient creates a pregnant patient
func PregnantPatient(gestationalWeeks int) *types.PatientContext {
	return &types.PatientContext{
		PatientID:      "PT-E2E-PREGNANT",
		Age:            28,
		Sex:            "F",
		IsPregnant:     true,
		GestationalAge: gestationalWeeks,
		Weight:         72.0,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I10", CodeSystem: "ICD10", Description: "Hypertension"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "PREGNANCY", Status: "ACTIVE"},
			{RegistryCode: "MATERNAL_MEDICATION", Status: "ACTIVE"},
		},
	}
}

// OpioidNaivePatient creates a patient with no prior opioid use
func OpioidNaivePatient() *types.PatientContext {
	return &types.PatientContext{
		PatientID:  "PT-E2E-OPIOID-NAIVE",
		Age:        45,
		Sex:        "M",
		IsPregnant: false,
		Weight:     82.0,
		CurrentMedications: []types.Medication{
			{Code: "APAP", Name: "Acetaminophen", DrugClass: "ANALGESIC", Dose: 650, DoseUnit: "mg"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "OPIOID_NAIVE", Status: "ACTIVE"},
		},
	}
}

// AKIPatient creates a patient with acute kidney injury
func AKIPatient(stage int) *types.PatientContext {
	var creatinine float64
	var ckdStage string
	switch stage {
	case 1:
		creatinine = 1.5
		ckdStage = "AKI_1"
	case 2:
		creatinine = 2.5
		ckdStage = "AKI_2"
	case 3:
		creatinine = 4.0
		ckdStage = "AKI_3"
	default:
		creatinine = 5.5
		ckdStage = "AKI_3"
	}

	return &types.PatientContext{
		PatientID:  "PT-E2E-AKI",
		Age:        65,
		Sex:        "M",
		IsPregnant: false,
		Weight:     78.0,
		RenalFunction: &types.RenalFunction{
			EGFR:       15.0,
			Creatinine: creatinine,
			CKDStage:   ckdStage,
			OnDialysis: false,
		},
		RecentLabs: []types.LabResult{
			{Code: "CREAT", Name: "Creatinine", Value: creatinine, Unit: "mg/dL", Timestamp: time.Now().Add(-1 * time.Hour)},
			{Code: "BUN", Name: "BUN", Value: 45.0, Unit: "mg/dL", Timestamp: time.Now().Add(-1 * time.Hour)},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "N17.9", CodeSystem: "ICD10", Description: "Acute kidney injury"},
		},
	}
}

// ICUPatientWithMultiOrganFailure creates an ICU patient with MODS
func ICUPatientWithMultiOrganFailure() *types.PatientContext {
	return &types.PatientContext{
		PatientID:  "PT-E2E-ICU-MODS",
		Age:        62,
		Sex:        "M",
		IsPregnant: false,
		Weight:     80.0,
		Vitals: &types.Vitals{
			SystolicBP:  72, // Hypotensive (MAP ~48)
			DiastolicBP: 36,
			HeartRate:   125,
			Temperature: 39.2,
			SpO2:        88,
		},
		RenalFunction: &types.RenalFunction{
			EGFR:       12.0,
			Creatinine: 3.8,
			CKDStage:   "AKI_3",
			OnDialysis: false,
		},
		HepaticFunction: &types.HepaticFunction{
			ChildPughScore: 9,
			ChildPughClass: "B",
		},
		RecentLabs: []types.LabResult{
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 6.1, Unit: "mmol/L", Timestamp: time.Now().Add(-30 * time.Minute)},
			{Code: "777-3", CodeSystem: "LOINC", Name: "Platelets", Value: 28.0, Unit: "K/uL", Timestamp: time.Now().Add(-1 * time.Hour)},
			{Code: "5902-2", CodeSystem: "LOINC", Name: "INR", Value: 3.1, Unit: "", Timestamp: time.Now().Add(-1 * time.Hour)},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
			{Code: "N17.9", CodeSystem: "ICD10", Description: "Acute kidney injury"},
			{Code: "D65", CodeSystem: "ICD10", Description: "DIC"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU", Status: "ACTIVE"},
			{RegistryCode: "SEPSIS", Status: "ACTIVE"},
		},
	}
}

// =============================================================================
// RECOMMENDATION BUILDERS
// =============================================================================

// FluidRecommendation creates a fluid resuscitation recommendation
func FluidRecommendation(volume float64) SimulatedRecommendation {
	return SimulatedRecommendation{
		ID:                 uuid.New(),
		Target:             "Normal Saline",
		TargetRxNorm:       "313002",
		DrugClass:          "IV_FLUID",
		RecommendedDose:    volume,
		DoseUnit:           "mL",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPSIS_HOUR_1",
		Rationale:          "SSC 2021: 30 mL/kg crystalloid for sepsis-induced hypoperfusion",
		Urgency:            "STAT",
	}
}

// AnticoagulationRecommendation creates an anticoagulation recommendation
func AnticoagulationRecommendation(drug, rxnorm string, dose float64) SimulatedRecommendation {
	return SimulatedRecommendation{
		ID:                 uuid.New(),
		Target:             drug,
		TargetRxNorm:       rxnorm,
		DrugClass:          "ANTICOAGULANT",
		RecommendedDose:    dose,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "AFIB_STROKE_PREVENTION",
		Rationale:          "CHEST Guidelines: Anticoagulation for CHA2DS2-VASc >= 2",
		Urgency:            "ROUTINE",
	}
}

// ACEInhibitorRecommendation creates an ACE inhibitor recommendation
func ACEInhibitorRecommendation() SimulatedRecommendation {
	return SimulatedRecommendation{
		ID:                 uuid.New(),
		Target:             "Lisinopril",
		TargetRxNorm:       "29046",
		DrugClass:          "ACE_INHIBITOR",
		RecommendedDose:    10.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "HYPERTENSION_MANAGEMENT",
		Rationale:          "JNC 8: ACE inhibitor first-line for hypertension",
		Urgency:            "ROUTINE",
	}
}

// OpioidRecommendation creates an opioid recommendation
func OpioidRecommendation(drug, rxnorm string, dose float64) SimulatedRecommendation {
	return SimulatedRecommendation{
		ID:                 uuid.New(),
		Target:             drug,
		TargetRxNorm:       rxnorm,
		DrugClass:          "OPIOID",
		RecommendedDose:    dose,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "PAIN_MANAGEMENT",
		Rationale:          "Pain control for acute postoperative pain",
		Urgency:            "URGENT",
	}
}

// NSAIDRecommendation creates an NSAID recommendation
func NSAIDRecommendation() SimulatedRecommendation {
	return SimulatedRecommendation{
		ID:                 uuid.New(),
		Target:             "Ibuprofen",
		TargetRxNorm:       "5640",
		DrugClass:          "NSAID",
		RecommendedDose:    400.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "PAIN_MANAGEMENT",
		Rationale:          "NSAID for musculoskeletal pain",
		Urgency:            "ROUTINE",
	}
}

// MethotrexateRecommendation creates a methotrexate recommendation
func MethotrexateRecommendation() SimulatedRecommendation {
	return SimulatedRecommendation{
		ID:                 uuid.New(),
		Target:             "Methotrexate",
		TargetRxNorm:       "6851",
		DrugClass:          "METHOTREXATE",
		RecommendedDose:    15.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "RHEUMATOID_ARTHRITIS",
		Rationale:          "DMARD therapy for RA",
		Urgency:            "SCHEDULED",
	}
}

// =============================================================================
// E2E FLOW EXECUTION
// =============================================================================

// ExecuteE2EFlow runs a complete KB-19 → KB-18 flow
func (ctx *E2ETestContext) ExecuteE2EFlow(patient *types.PatientContext, rec SimulatedRecommendation) (*E2EScenarioResult, error) {
	// Convert KB-19 recommendation to KB-18 evaluation request
	evalReq := &types.EvaluationRequest{
		RequestID:      uuid.New().String(),
		PatientID:      patient.PatientID,
		PatientContext: patient,
		Order: &types.MedicationOrder{
			MedicationCode: rec.TargetRxNorm,
			MedicationName: rec.Target,
			DrugClass:      rec.DrugClass,
			Dose:           rec.RecommendedDose,
			DoseUnit:       rec.DoseUnit,
			Frequency:      "once",
			Route:          "PO",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "KB19-ORCHESTRATOR",
		RequestorRole:  "SYSTEM",
		Timestamp:      time.Now(),
	}

	// Execute governance evaluation
	resp, err := ctx.Engine.Evaluate(ctx.Ctx, evalReq)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &E2EScenarioResult{
		Recommendation:     rec,
		GovernanceResponse: resp,
		FinalAllowed:       resp.IsApproved,
		FinalOutcome:       resp.Outcome,
		ViolationCount:     len(resp.Violations),
		RequiresOverride:   resp.Outcome == types.OutcomePendingOverride,
		RequiresEscalation: resp.Outcome == types.OutcomeEscalated,
	}

	// Extract enforcement level from violations
	if len(resp.Violations) > 0 {
		result.EnforcementApplied = resp.Violations[0].EnforcementLevel
	}

	// Capture evidence trail
	if resp.EvidenceTrail != nil {
		result.EvidenceTrailHash = resp.EvidenceTrail.Hash
	}

	return result, nil
}

// =============================================================================
// ASSERTION HELPERS
// =============================================================================

// AssertBlocked verifies the recommendation was blocked
func (r *E2EScenarioResult) IsBlocked() bool {
	return r.FinalOutcome == types.OutcomeBlocked
}

// AssertApproved verifies the recommendation was approved
func (r *E2EScenarioResult) IsApproved() bool {
	return r.FinalOutcome == types.OutcomeApproved || r.FinalOutcome == types.OutcomeApprovedWithWarns
}

// HasEnforcement checks if a specific enforcement was applied
func (r *E2EScenarioResult) HasEnforcement(level types.EnforcementLevel) bool {
	return r.EnforcementApplied == level
}

// HasViolationCategory checks if a violation category was triggered
func (r *E2EScenarioResult) HasViolationCategory(category types.ViolationCategory) bool {
	for _, v := range r.GovernanceResponse.Violations {
		if v.Category == category {
			return true
		}
	}
	return false
}

// HasEvidenceTrail verifies evidence trail was generated
func (r *E2EScenarioResult) HasEvidenceTrail() bool {
	return r.EvidenceTrailHash != "" && len(r.EvidenceTrailHash) > 10
}
