// Package builders provides builder implementations for constructing
// the KnowledgeSnapshot component of ClinicalExecutionContext.
//
// CRITICAL DESIGN RULE:
// KnowledgeSnapshotBuilder operates on PatientContext ONLY.
// It NEVER inspects raw FHIR - that's KB-2A's job.
//
// The builder queries Knowledge Bases using patient context data
// and assembles pre-computed answers into the frozen contract.
//
// ARCHITECTURE CONSTRAINT (CTO/CMO Directive):
// "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."
//
// KB-7 FHIR API: Used for clinical execution (precomputed, O(1) lookups)
// KB-7 Rules API: Used for admin/build-time only (may use Neo4j)
//
// STRUCTURE (per CTO/CMO spec):
// - Terminology (KB-7): Pre-resolved codes and ValueSet memberships via FHIR
// - Calculators (KB-8): Pre-computed clinical calculations with named fields
// - Safety (KB-4): Allergies, contraindications, pregnancy status
// - Interactions (KB-5): Drug-drug interactions (current and potential)
// - Formulary (KB-6): Formulary status, prior auth, alternatives, NLEM/PBS
// - Dosing (KB-1): Renal/hepatic/weight-based dose adjustments
// - CDI (KB-11): Clinical documentation intelligence facts
package builders

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"vaidshala/clinical-runtime-platform/clients"
	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// KB CLIENT INTERFACES (per CTO/CMO spec)
// ============================================================================

// KB7Client interface for KB-7 Terminology Service (Legacy Rules API).
// DEPRECATED for clinical execution paths - use KB7FHIRClient instead.
// This interface may use Neo4j at runtime which violates the CTO/CMO directive.
type KB7Client interface {
	// ExpandValueSet returns all codes in a ValueSet
	ExpandValueSet(ctx context.Context, valueSetName string) ([]contracts.ClinicalCode, error)

	// CheckMembership checks if a code is in specified ValueSets
	CheckMembership(ctx context.Context, code contracts.ClinicalCode, valueSetNames []string) ([]string, error)

	// GetRelevantValueSets returns ValueSets relevant to patient conditions
	GetRelevantValueSets(ctx context.Context, conditionCodes []contracts.ClinicalCode) ([]string, error)

	// ResolveCode gets display name for a code
	ResolveCode(ctx context.Context, code contracts.ClinicalCode) (string, error)
}

// KB7FHIRClient interface for KB-7 FHIR Terminology Service (Runtime-Safe).
// This is the REQUIRED interface for clinical execution paths.
// Uses precomputed expansions - NO Neo4j at runtime.
type KB7FHIRClient interface {
	// ValidateCode checks if a code is a member of a ValueSet (O(1) lookup)
	ValidateCode(ctx context.Context, valueSetID string, system string, code string) (bool, error)

	// ExpandValueSet returns all codes in a ValueSet's precomputed expansion
	// WARNING: Use ValidateCode for membership checks, not ExpandValueSet!
	ExpandValueSet(ctx context.Context, valueSetID string) ([]contracts.ClinicalCode, error)

	// ListCanonicalValueSets returns all canonical ValueSets from KB-7.
	// DYNAMIC: This replaces hardcoded ValueSet lists!
	ListCanonicalValueSets(ctx context.Context) ([]clients.ValueSetMetadata, error)

	// GetMembershipsForCode returns all canonical ValueSets containing the code
	GetMembershipsForCode(ctx context.Context, code contracts.ClinicalCode) (map[string]bool, error)

	// HealthCheck verifies FHIR endpoints are operational
	HealthCheck(ctx context.Context) error

	// ListValueSetsByContext returns ValueSets filtered by use_context.
	// DYNAMIC: This replaces hardcoded ValueSetsToExpand arrays!
	// Example contexts: "cql", "measure", "lab", "medication"
	ListValueSetsByContext(ctx context.Context, useContext string) ([]clients.ValueSetMetadata, error)

	// GetValueSetNamesForContext returns just the names of ValueSets for a given context.
	// Convenience method for populating ValueSetsToExpand dynamically.
	GetValueSetNamesForContext(ctx context.Context, useContext string) ([]string, error)

	// =============== KB-7 v2 REVERSE LOOKUP METHODS ===============
	// These methods use the new $lookup-memberships endpoint for O(1) lookup
	// instead of iterating over 18,000+ ValueSets.

	// LookupMemberships returns all ValueSets containing a code using REVERSE LOOKUP.
	// This is the KEY method for KB-7 v2 - single indexed query returns ALL memberships.
	// canonicalOnly=true filters to only ~75-100 canonical ValueSets for ICU/Safety workflows.
	LookupMemberships(ctx context.Context, code string, system string, canonicalOnly bool) (*clients.LookupMembershipsResponse, error)

	// GetMembershipsForCodeWithDetails returns detailed membership info including category and semantic names.
	// Uses LookupMemberships internally but returns richer data structure.
	GetMembershipsForCodeWithDetails(ctx context.Context, code contracts.ClinicalCode, canonicalOnly bool) ([]clients.ValueSetMembership, error)

	// GetSemanticNamesForCode returns just the human-readable semantic names for a code.
	// Convenience method for building ValueSetMemberships map with readable flag names.
	GetSemanticNamesForCode(ctx context.Context, code contracts.ClinicalCode, canonicalOnly bool) ([]string, error)
}

// KB8Client interface for KB-8 Calculator Service.
type KB8Client interface {
	// CalculateEGFR calculates Estimated Glomerular Filtration Rate
	CalculateEGFR(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateASCVD calculates 10-year ASCVD risk
	CalculateASCVD(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateCHA2DS2VASc calculates stroke risk in AFib
	CalculateCHA2DS2VASc(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateHASBLED calculates bleeding risk
	CalculateHASBLED(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateBMI calculates Body Mass Index
	CalculateBMI(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateChildPugh calculates liver function score
	CalculateChildPugh(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateMELD calculates Model for End-Stage Liver Disease
	CalculateMELD(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateSOFA calculates Sequential Organ Failure Assessment
	CalculateSOFA(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)

	// CalculateQSOFA calculates Quick SOFA
	CalculateQSOFA(ctx context.Context, patient *contracts.PatientContext) (*contracts.CalculationResult, error)
}

// KB4Client interface for KB-4 Patient Safety Service.
// Provides comprehensive medication safety checking including:
// - Black box warnings (FDA/TGA/EMA)
// - Contraindications (drug-condition)
// - Dose limits and validation
// - Pregnancy safety (FDA PLLR, TGA categories)
// - Lactation safety (LactMed)
// - High-alert medications (ISMP)
// - Beers criteria (AGS geriatric)
// - Anticholinergic burden (ACB scale)
// - Lab monitoring requirements
type KB4Client interface {
	// ============================================================================
	// BASIC SAFETY METHODS (Original)
	// ============================================================================

	// GetActiveAllergies returns allergy information
	GetActiveAllergies(ctx context.Context, patient *contracts.PatientContext) ([]contracts.AllergyInfo, error)

	// CheckContraindications checks drug-condition contraindications
	CheckContraindications(ctx context.Context, meds []contracts.ClinicalCode, conditions []contracts.ClinicalCode) ([]contracts.ContraindicationInfo, error)

	// GetPregnancyStatus returns pregnancy information if applicable
	GetPregnancyStatus(ctx context.Context, patient *contracts.PatientContext) (*contracts.PregnancyInfo, error)

	// NeedsRenalDoseAdjustment checks if renal adjustment needed based on eGFR
	NeedsRenalDoseAdjustment(ctx context.Context, eGFR float64) (bool, error)

	// NeedsHepaticDoseAdjustment checks if hepatic adjustment needed
	NeedsHepaticDoseAdjustment(ctx context.Context, patient *contracts.PatientContext) (bool, error)

	// ============================================================================
	// ENHANCED SAFETY METHODS (KB-4 Full API)
	// ============================================================================

	// CheckMedicationSafety performs comprehensive safety evaluation for a medication.
	// Returns all applicable safety alerts (black box, contraindications, dose limits, etc.)
	CheckMedicationSafety(ctx context.Context, drug *contracts.ClinicalCode, proposedDose float64, doseUnit string, patient *contracts.PatientContext) (*clients.KB4SafetyCheckResponse, error)

	// GetBlackBoxWarnings retrieves FDA/TGA black box warnings for medications.
	GetBlackBoxWarnings(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4BlackBoxWarning, error)

	// GetHighAlertStatus retrieves ISMP high-alert medication status.
	GetHighAlertStatus(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4HighAlertMedication, error)

	// GetPregnancySafetyInfo retrieves comprehensive pregnancy safety information.
	// Returns FDA PLLR data, TGA categories, teratogenicity info, alternatives.
	GetPregnancySafetyInfo(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4PregnancySafety, error)

	// GetLactationSafetyInfo retrieves LactMed lactation safety information.
	GetLactationSafetyInfo(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4LactationSafety, error)

	// GetBeersCriteria retrieves AGS Beers Criteria entries for geriatric patients.
	GetBeersCriteria(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4BeersEntry, error)

	// GetAnticholinergicBurden retrieves ACB scores for medications.
	GetAnticholinergicBurden(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4AnticholinergicBurden, error)

	// CalculateTotalAnticholinergicBurden calculates cumulative ACB score.
	// Returns total score and risk level (Low: 1-2, Moderate: 3-4, High: 5+).
	CalculateTotalAnticholinergicBurden(ctx context.Context, rxnormCodes []string) (*clients.KB4ACBCalculation, error)

	// GetLabRequirements retrieves required lab monitoring for medications.
	GetLabRequirements(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4LabRequirement, error)

	// GetDoseLimits retrieves dose limit information for medications.
	GetDoseLimits(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4DoseLimit, error)

	// ValidateDose validates a proposed dose against KB-4 dose limits.
	ValidateDose(ctx context.Context, drug *contracts.ClinicalCode, proposedDose float64, doseUnit string, patient *contracts.PatientContext) (*clients.KB4DoseValidation, error)

	// GetAgeLimits retrieves age restriction information for medications.
	GetAgeLimits(ctx context.Context, rxnormCodes []string) (map[string]*clients.KB4AgeLimit, error)

	// HealthCheck verifies KB-4 service is healthy.
	HealthCheck(ctx context.Context) error
}

// KB5Client interface for KB-5 Drug Interactions Service.
type KB5Client interface {
	// GetCurrentInteractions checks interactions between patient's active medications
	GetCurrentInteractions(ctx context.Context, medications []contracts.ClinicalCode) ([]contracts.DrugInteraction, error)

	// GetPotentialInteractions checks interactions if adding common drugs
	GetPotentialInteractions(ctx context.Context, medications []contracts.ClinicalCode) ([]contracts.DrugInteraction, error)
}

// KB6Client interface for KB-6 Formulary Service.
type KB6Client interface {
	// GetFormularyStatus returns formulary status for medications
	GetFormularyStatus(ctx context.Context, medications []contracts.ClinicalCode, region string) (map[string]contracts.FormularyStatus, error)

	// GetPriorAuthRequired returns medications needing prior authorization
	GetPriorAuthRequired(ctx context.Context, medications []contracts.ClinicalCode) ([]contracts.ClinicalCode, error)

	// GetGenericAlternatives returns generic alternatives for brand drugs
	GetGenericAlternatives(ctx context.Context, medication contracts.ClinicalCode) ([]contracts.ClinicalCode, error)

	// GetNLEMAvailability checks if drugs are on India NLEM
	GetNLEMAvailability(ctx context.Context, medications []contracts.ClinicalCode) (map[string]bool, error)

	// GetPBSAvailability checks if drugs are on Australia PBS
	GetPBSAvailability(ctx context.Context, medications []contracts.ClinicalCode) (map[string]bool, error)
}

// KB1Client interface for KB-1 Drug Rules Service.
type KB1Client interface {
	// GetRenalAdjustments returns renal dose adjustments for medications
	GetRenalAdjustments(ctx context.Context, medications []contracts.ClinicalCode, eGFR float64) (map[string]contracts.DoseAdjustment, error)

	// GetHepaticAdjustments returns hepatic dose adjustments
	GetHepaticAdjustments(ctx context.Context, medications []contracts.ClinicalCode, childPugh string) (map[string]contracts.DoseAdjustment, error)

	// GetWeightBasedDoses returns weight-based dose calculations
	GetWeightBasedDoses(ctx context.Context, medications []contracts.ClinicalCode, weightKg float64, bsa float64) (map[string]contracts.DoseCalculation, error)

	// GetAgeBasedAdjustments returns age-based dose adjustments
	GetAgeBasedAdjustments(ctx context.Context, medications []contracts.ClinicalCode, ageYears int) (map[string]contracts.DoseAdjustment, error)
}

// KB11Client interface for KB-11 Clinical Documentation Intelligence.
type KB11Client interface {
	// ExtractFacts extracts clinical facts from notes
	ExtractFacts(ctx context.Context, patient *contracts.PatientContext) ([]contracts.CDIFact, error)

	// GetCodingOpportunities identifies coding improvements
	GetCodingOpportunities(ctx context.Context, patient *contracts.PatientContext) ([]contracts.CodingOpportunity, error)

	// GetQueryOpportunities identifies documentation queries
	GetQueryOpportunities(ctx context.Context, patient *contracts.PatientContext) ([]contracts.QueryOpportunity, error)
}

// KB16Client interface for KB-16 Lab Interpretation Service (Category A - SNAPSHOT KB).
// Lab reference ranges are pre-computed at snapshot build time - CQL evaluates against
// frozen lab interpretation data. Engines NEVER call KB-16 directly at execution time.
type KB16Client interface {
	// GetReferenceRangesForPatient returns all applicable reference ranges
	// Primary method for snapshot building - fetches all ranges in one call
	GetReferenceRangesForPatient(ctx context.Context, demographics KB16PatientDemographics, loincCodes []string) (map[string]*KB16ReferenceRange, error)

	// GetCriticalValues returns critical/panic thresholds for a LOINC code
	GetCriticalValues(ctx context.Context, loincCode string) (*KB16CriticalValues, error)

	// BatchInterpretLabs interprets multiple lab results in a single call
	BatchInterpretLabs(ctx context.Context, labs []KB16LabResult, demographics KB16PatientDemographics) ([]KB16LabInterpretation, error)

	// HealthCheck verifies KB-16 service is operational
	HealthCheck(ctx context.Context) error
}

// KB16PatientDemographics contains patient info for reference range selection.
type KB16PatientDemographics struct {
	AgeYears   int    `json:"age_years"`
	Sex        string `json:"sex"`
	IsPregnant bool   `json:"is_pregnant,omitempty"`
	Trimester  int    `json:"trimester,omitempty"`
	Ethnicity  string `json:"ethnicity,omitempty"`
	Region     string `json:"region,omitempty"`
	IsFasting  bool   `json:"is_fasting,omitempty"`
}

// KB16ReferenceRange contains the normal range for a lab value.
type KB16ReferenceRange struct {
	LOINCCode        string   `json:"loinc_code"`
	DisplayName      string   `json:"display_name,omitempty"`
	LowNormal        float64  `json:"low_normal"`
	HighNormal       float64  `json:"high_normal"`
	Unit             string   `json:"unit"`
	CriticalLow      *float64 `json:"critical_low,omitempty"`
	CriticalHigh     *float64 `json:"critical_high,omitempty"`
	AgeMin           int      `json:"age_min,omitempty"`
	AgeMax           int      `json:"age_max,omitempty"`
	Sex              string   `json:"sex,omitempty"`
	IsPregnancyRange bool     `json:"is_pregnancy_range,omitempty"`
	Source           string   `json:"source,omitempty"`
}

// KB16CriticalValues contains critical/panic value thresholds.
type KB16CriticalValues struct {
	LOINCCode           string   `json:"loinc_code"`
	DisplayName         string   `json:"display_name,omitempty"`
	CriticalLow         *float64 `json:"critical_low,omitempty"`
	CriticalHigh        *float64 `json:"critical_high,omitempty"`
	Unit                string   `json:"unit"`
	RequiredAction      string   `json:"required_action,omitempty"`
	NotifyWithinMinutes int      `json:"notify_within_minutes,omitempty"`
	Source              string   `json:"source,omitempty"`
}

// KB16LabResult represents a single laboratory measurement for KB-16.
type KB16LabResult struct {
	LOINCCode   string    `json:"loinc_code"`
	DisplayName string    `json:"display_name,omitempty"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	CollectedAt time.Time `json:"collected_at"`
	LabFlag     string    `json:"lab_flag,omitempty"`
}

// KB16LabInterpretation provides clinical interpretation of a lab result.
type KB16LabInterpretation struct {
	LOINCCode            string   `json:"loinc_code"`
	DisplayName          string   `json:"display_name,omitempty"`
	Value                float64  `json:"value"`
	Unit                 string   `json:"unit"`
	AbnormalityLevel     string   `json:"abnormality_level"`
	Flag                 string   `json:"flag"`
	IsCritical           bool     `json:"is_critical"`
	ClinicalSignificance string   `json:"clinical_significance"`
	PossibleCauses       []string `json:"possible_causes,omitempty"`
	SuggestedActions     []string `json:"suggested_actions,omitempty"`
	Trend                string   `json:"trend,omitempty"`
	Narrative            string   `json:"narrative,omitempty"`
}

// ============================================================================
// KNOWLEDGE SNAPSHOT BUILDER
// ============================================================================

// KnowledgeSnapshotBuilder assembles KnowledgeSnapshot from all KBs.
// It operates ONLY on PatientContext - never raw FHIR.
//
// IMPORTANT: For clinical execution, use NewKnowledgeSnapshotBuilderFHIR
// which enforces the use of KB7FHIRClient for terminology operations.
type KnowledgeSnapshotBuilder struct {
	// kb7Client is the LEGACY Rules API client (DEPRECATED for clinical execution)
	kb7Client KB7Client

	// kb7FHIRClient is the REQUIRED FHIR client for clinical execution
	// Uses precomputed expansions - NO Neo4j at runtime
	kb7FHIRClient KB7FHIRClient

	kb8Client  KB8Client
	kb4Client  KB4Client
	kb5Client  KB5Client
	kb6Client  KB6Client
	kb1Client  KB1Client
	kb11Client KB11Client
	kb16Client KB16Client // KB-16 Lab Interpretation (Category A - SNAPSHOT KB)

	// config for builder behavior
	config KnowledgeSnapshotConfig

	// kbVersions tracks which KB versions were used
	kbVersions map[string]string

	// useFHIR indicates whether to use FHIR client for terminology (REQUIRED for clinical execution)
	useFHIR bool
}

// KnowledgeSnapshotConfig configures the builder.
type KnowledgeSnapshotConfig struct {
	// Region for formulary lookups (IN, AU)
	Region string

	// ParallelQueries enables parallel KB queries
	ParallelQueries bool

	// QueryTimeout for each KB query
	QueryTimeout time.Duration

	// ValueSetsToExpand specific ValueSets to always expand
	ValueSetsToExpand []string

	// ValueSetMembershipQueries pre-defined ValueSet membership questions
	// e.g., "Is patient diabetic?" checks "Diabetes" ValueSet
	ValueSetMembershipQueries map[string]string
}

// DefaultKnowledgeSnapshotConfig returns sensible defaults.
//
// DYNAMIC VALUESET DISCOVERY (KB-7 as SINGLE SOURCE OF TRUTH):
// ValueSetsToExpand is now fetched dynamically from KB-7 using use_context="cql".
// The hardcoded list below is only used as a FALLBACK if KB-7 is unavailable.
// To add new ValueSets for CQL expansion, update KB-7's use_context field, NOT this list.
func DefaultKnowledgeSnapshotConfig() KnowledgeSnapshotConfig {
	return KnowledgeSnapshotConfig{
		Region:          "AU",
		ParallelQueries: true,
		QueryTimeout:    5 * time.Second,
		// DYNAMIC: ValueSets are now fetched from KB-7 using GetValueSetNamesForContext("cql").
		// This fallback list is only used if KB-7 is unavailable.
		// To add ValueSets: Update use_context in KB-7's value_sets table, NOT here!
		ValueSetsToExpand: []string{
			// FALLBACK ONLY - these are overridden by KB-7 dynamic discovery
		},
		// ValueSet membership questions for CTO/CMO spec
		// DYNAMIC: ValueSetMemberships are populated from KB-7's canonical ValueSets
		// using ListCanonicalValueSets() and DeriveSemanticFlagNameWithCategory().
		// This map provides fallback/override capability only.
		ValueSetMembershipQueries: map[string]string{
			// FALLBACK ONLY - membership is now computed dynamically from KB-7
		},
	}
}

// NewKnowledgeSnapshotBuilder creates a new builder with all KB clients.
// DEPRECATED: Use NewKnowledgeSnapshotBuilderFHIR for clinical execution paths.
// This constructor uses the legacy KB7Client which may use Neo4j at runtime.
func NewKnowledgeSnapshotBuilder(
	kb7 KB7Client,
	kb8 KB8Client,
	kb4 KB4Client,
	kb5 KB5Client,
	kb6 KB6Client,
	kb1 KB1Client,
	kb11 KB11Client,
	kb16 KB16Client,
	config KnowledgeSnapshotConfig,
) *KnowledgeSnapshotBuilder {
	return &KnowledgeSnapshotBuilder{
		kb7Client:  kb7,
		kb8Client:  kb8,
		kb4Client:  kb4,
		kb5Client:  kb5,
		kb6Client:  kb6,
		kb1Client:  kb1,
		kb11Client: kb11,
		kb16Client: kb16,
		config:     config,
		kbVersions: make(map[string]string),
		useFHIR:    false, // Legacy mode
	}
}

// NewKnowledgeSnapshotBuilderFHIR creates a builder using the FHIR client for terminology.
// This is the REQUIRED constructor for clinical execution paths.
//
// ARCHITECTURE CONSTRAINT (CTO/CMO Directive):
// "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."
//
// This constructor enforces:
// - Precomputed expansions from PostgreSQL
// - O(1) indexed lookups via $validate-code
// - NO Neo4j traversal at runtime
// - Deterministic, auditable results
func NewKnowledgeSnapshotBuilderFHIR(
	kb7FHIR KB7FHIRClient,
	kb8 KB8Client,
	kb4 KB4Client,
	kb5 KB5Client,
	kb6 KB6Client,
	kb1 KB1Client,
	kb11 KB11Client,
	kb16 KB16Client,
	config KnowledgeSnapshotConfig,
) *KnowledgeSnapshotBuilder {
	return &KnowledgeSnapshotBuilder{
		kb7FHIRClient: kb7FHIR,
		kb8Client:     kb8,
		kb4Client:     kb4,
		kb5Client:     kb5,
		kb6Client:     kb6,
		kb1Client:     kb1,
		kb11Client:    kb11,
		kb16Client:    kb16,
		config:        config,
		kbVersions:    make(map[string]string),
		useFHIR:       true, // FHIR mode (REQUIRED for clinical execution)
	}
}

// Build constructs KnowledgeSnapshot from PatientContext.
// CRITICAL: Input is PatientContext, NOT raw FHIR.
//
// ORDER (per CTO/CMO spec):
// 1. KB-7: Terminology (needed for ValueSet membership questions)
// 2. KB-8: Calculators (needed for dose adjustment decisions)
// 3. KB-4: Safety (uses eGFR for renal check)
// 4. KB-5: Interactions
// 5. KB-6: Formulary
// 6. KB-1: Dosing (uses eGFR, Child-Pugh)
// 7. KB-11: CDI
func (b *KnowledgeSnapshotBuilder) Build(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.KnowledgeSnapshot, error) {

	snapshot := &contracts.KnowledgeSnapshot{
		SnapshotTimestamp: time.Now(),
		SnapshotVersion:   "1.0.0",
		KBVersions:        make(map[string]string),
	}

	if b.config.ParallelQueries {
		return b.buildParallel(ctx, patient, snapshot)
	}
	return b.buildSequential(ctx, patient, snapshot)
}

// buildParallel queries all KBs in parallel for performance.
func (b *KnowledgeSnapshotBuilder) buildParallel(
	ctx context.Context,
	patient *contracts.PatientContext,
	snapshot *contracts.KnowledgeSnapshot,
) (*contracts.KnowledgeSnapshot, error) {

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Extract codes for queries
	conditionCodes := extractConditionCodes(patient)
	medicationCodes := extractMedicationCodes(patient)
	labCodes := extractLabCodes(patient)

	// Query KB-7 Terminology
	wg.Add(1)
	go func() {
		defer wg.Done()
		terminology, err := b.buildTerminologySnapshot(ctx, patient, conditionCodes, medicationCodes, labCodes)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-7: %w", err))
		} else {
			snapshot.Terminology = *terminology
		}
		mu.Unlock()
	}()

	// Query KB-8 Calculators
	wg.Add(1)
	go func() {
		defer wg.Done()
		calcs, err := b.buildCalculatorSnapshot(ctx, patient)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-8: %w", err))
		} else {
			snapshot.Calculators = *calcs
		}
		mu.Unlock()
	}()

	// Wait for KB-8 before KB-4 (needs eGFR)
	wg.Wait()

	// Now query dependent KBs
	var wg2 sync.WaitGroup

	// Query KB-4 Safety (needs eGFR from KB-8)
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		eGFR := getEGFRValue(&snapshot.Calculators)
		safety, err := b.buildSafetySnapshot(ctx, patient, medicationCodes, conditionCodes, eGFR)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-4: %w", err))
		} else {
			snapshot.Safety = *safety
		}
		mu.Unlock()
	}()

	// Query KB-5 Interactions
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		interactions, err := b.buildInteractionSnapshot(ctx, medicationCodes)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-5: %w", err))
		} else {
			snapshot.Interactions = *interactions
		}
		mu.Unlock()
	}()

	// Query KB-6 Formulary
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		formulary, err := b.buildFormularySnapshot(ctx, medicationCodes)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-6: %w", err))
		} else {
			snapshot.Formulary = *formulary
		}
		mu.Unlock()
	}()

	// Query KB-1 Dosing (needs eGFR, Child-Pugh)
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		eGFR := getEGFRValue(&snapshot.Calculators)
		childPugh := getChildPughClass(&snapshot.Calculators)
		dosing, err := b.buildDosingSnapshot(ctx, patient, medicationCodes, eGFR, childPugh)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-1: %w", err))
		} else {
			snapshot.Dosing = *dosing
		}
		mu.Unlock()
	}()

	// Query KB-11 CDI
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		cdi, err := b.buildCDIFacts(ctx, patient)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-11: %w", err))
		} else {
			snapshot.CDI = *cdi
		}
		mu.Unlock()
	}()

	// Query KB-16 Lab Interpretation
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		labInterp, err := b.buildLabInterpretationSnapshot(ctx, patient)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("KB-16: %w", err))
		} else {
			snapshot.LabInterpretation = *labInterp
		}
		mu.Unlock()
	}()

	wg2.Wait()

	// Copy KB versions
	mu.Lock()
	for k, v := range b.kbVersions {
		snapshot.KBVersions[k] = v
	}
	mu.Unlock()

	// Non-fatal errors: log but continue
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("KnowledgeSnapshot warning: %v\n", err)
		}
	}

	return snapshot, nil
}

// buildSequential queries KBs one by one (fallback mode).
func (b *KnowledgeSnapshotBuilder) buildSequential(
	ctx context.Context,
	patient *contracts.PatientContext,
	snapshot *contracts.KnowledgeSnapshot,
) (*contracts.KnowledgeSnapshot, error) {

	conditionCodes := extractConditionCodes(patient)
	medicationCodes := extractMedicationCodes(patient)
	labCodes := extractLabCodes(patient)

	// KB-7 Terminology
	terminology, err := b.buildTerminologySnapshot(ctx, patient, conditionCodes, medicationCodes, labCodes)
	if err == nil {
		snapshot.Terminology = *terminology
	}

	// KB-8 Calculators (needed by subsequent KBs)
	calcs, err := b.buildCalculatorSnapshot(ctx, patient)
	if err == nil {
		snapshot.Calculators = *calcs
	}

	// Get values for dependent KBs
	eGFR := getEGFRValue(&snapshot.Calculators)
	childPugh := getChildPughClass(&snapshot.Calculators)

	// KB-4 Safety
	safety, err := b.buildSafetySnapshot(ctx, patient, medicationCodes, conditionCodes, eGFR)
	if err == nil {
		snapshot.Safety = *safety
	}

	// KB-5 Interactions
	interactions, err := b.buildInteractionSnapshot(ctx, medicationCodes)
	if err == nil {
		snapshot.Interactions = *interactions
	}

	// KB-6 Formulary
	formulary, err := b.buildFormularySnapshot(ctx, medicationCodes)
	if err == nil {
		snapshot.Formulary = *formulary
	}

	// KB-1 Dosing
	dosing, err := b.buildDosingSnapshot(ctx, patient, medicationCodes, eGFR, childPugh)
	if err == nil {
		snapshot.Dosing = *dosing
	}

	// KB-11 CDI
	cdi, err := b.buildCDIFacts(ctx, patient)
	if err == nil {
		snapshot.CDI = *cdi
	}

	// KB-16 Lab Interpretation
	labInterp, err := b.buildLabInterpretationSnapshot(ctx, patient)
	if err == nil {
		snapshot.LabInterpretation = *labInterp
	}

	// Copy versions
	for k, v := range b.kbVersions {
		snapshot.KBVersions[k] = v
	}

	return snapshot, nil
}

// ============================================================================
// KB-7: TERMINOLOGY SNAPSHOT (per CTO/CMO spec)
// ============================================================================

func (b *KnowledgeSnapshotBuilder) buildTerminologySnapshot(
	ctx context.Context,
	patient *contracts.PatientContext,
	conditionCodes []contracts.ClinicalCode,
	medicationCodes []contracts.ClinicalCode,
	labCodes []contracts.ClinicalCode,
) (*contracts.TerminologySnapshot, error) {

	snapshot := &contracts.TerminologySnapshot{
		PatientConditionCodes:  make([]contracts.ResolvedCode, 0),
		PatientMedicationCodes: make([]contracts.ResolvedCode, 0),
		ValueSetMemberships:    make(map[string]bool),
		ExpandedValueSets:      make(map[string][]contracts.ClinicalCode),
		CodeMemberships:        make(map[string][]string),
	}

	// ARCHITECTURE DECISION: Use FHIR client if available (REQUIRED for clinical execution)
	if b.useFHIR && b.kb7FHIRClient != nil {
		return b.buildTerminologySnapshotFHIR(ctx, patient, conditionCodes, medicationCodes, labCodes, snapshot)
	}

	// LEGACY PATH: Uses Rules API (may use Neo4j at runtime)
	// This path is DEPRECATED for clinical execution
	if b.kb7Client == nil {
		return snapshot, nil
	}

	// Resolve condition codes with display names
	for _, code := range conditionCodes {
		display, _ := b.kb7Client.ResolveCode(ctx, code)
		if display == "" {
			display = code.Display
		}
		snapshot.PatientConditionCodes = append(snapshot.PatientConditionCodes, contracts.ResolvedCode{
			System:  code.System,
			Code:    code.Code,
			Display: display,
		})
	}

	// Resolve medication codes with display names
	for _, code := range medicationCodes {
		display, _ := b.kb7Client.ResolveCode(ctx, code)
		if display == "" {
			display = code.Display
		}
		snapshot.PatientMedicationCodes = append(snapshot.PatientMedicationCodes, contracts.ResolvedCode{
			System:  code.System,
			Code:    code.Code,
			Display: display,
		})
	}

	// Get relevant ValueSets for patient conditions
	relevantVS, err := b.kb7Client.GetRelevantValueSets(ctx, conditionCodes)
	if err != nil {
		return snapshot, fmt.Errorf("get relevant valuesets: %w", err)
	}

	// Combine with configured ValueSets
	allValueSets := append(b.config.ValueSetsToExpand, relevantVS...)
	uniqueVS := uniqueStrings(allValueSets)

	// Expand each ValueSet
	for _, vsName := range uniqueVS {
		codes, err := b.kb7Client.ExpandValueSet(ctx, vsName)
		if err != nil {
			continue // Non-fatal
		}
		snapshot.ExpandedValueSets[vsName] = codes
	}

	// Build ValueSetMemberships map (per CTO/CMO spec)
	// This answers questions like "Is patient diabetic?" → true
	for queryName, valueSetName := range b.config.ValueSetMembershipQueries {
		hasMembership := false

		// Check condition memberships
		for _, cond := range conditionCodes {
			memberships, err := b.kb7Client.CheckMembership(ctx, cond, []string{valueSetName})
			if err == nil && len(memberships) > 0 {
				hasMembership = true
				// Also populate CodeMemberships map
				codeKey := fmt.Sprintf("%s|%s", cond.System, cond.Code)
				snapshot.CodeMemberships[codeKey] = append(snapshot.CodeMemberships[codeKey], memberships...)
			}
		}

		// Check medication memberships
		for _, med := range medicationCodes {
			memberships, err := b.kb7Client.CheckMembership(ctx, med, []string{valueSetName})
			if err == nil && len(memberships) > 0 {
				hasMembership = true
				codeKey := fmt.Sprintf("%s|%s", med.System, med.Code)
				snapshot.CodeMemberships[codeKey] = append(snapshot.CodeMemberships[codeKey], memberships...)
			}
		}

		snapshot.ValueSetMemberships[queryName] = hasMembership
	}

	b.kbVersions["KB-7"] = "1.0.0"
	return snapshot, nil
}

// buildTerminologySnapshotFHIR builds terminology snapshot using FHIR client.
// This is the REQUIRED path for clinical execution.
//
// ARCHITECTURE CONSTRAINT (CTO/CMO Directive):
// "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."
//
// KB-7 v2 ARCHITECTURE (PERFORMANCE OPTIMIZED):
// - Uses REVERSE LOOKUP: "Which ValueSets contain code X?" via $lookup-memberships
// - Single indexed query returns ALL ValueSets for a code (O(1) per code)
// - NEVER iterates over 18,000+ ValueSets (the OLD approach)
// - Returns SEMANTIC NAMES like "Diabetes", "ACE Inhibitors" (not OIDs!)
//
// This method:
// - Uses LookupMemberships for O(1) reverse lookup (KB-7 v2)
// - Uses precomputed expansions from PostgreSQL
// - NEVER calls Neo4j at runtime
// - Produces deterministic, auditable results
func (b *KnowledgeSnapshotBuilder) buildTerminologySnapshotFHIR(
	ctx context.Context,
	patient *contracts.PatientContext,
	conditionCodes []contracts.ClinicalCode,
	medicationCodes []contracts.ClinicalCode,
	labCodes []contracts.ClinicalCode,
	snapshot *contracts.TerminologySnapshot,
) (*contracts.TerminologySnapshot, error) {

	// Step 1: Resolve condition codes (use display from input if available)
	for _, code := range conditionCodes {
		display := code.Display
		snapshot.PatientConditionCodes = append(snapshot.PatientConditionCodes, contracts.ResolvedCode{
			System:  code.System,
			Code:    code.Code,
			Display: display,
		})
	}

	// Step 2: Resolve medication codes
	for _, code := range medicationCodes {
		display := code.Display
		snapshot.PatientMedicationCodes = append(snapshot.PatientMedicationCodes, contracts.ResolvedCode{
			System:  code.System,
			Code:    code.Code,
			Display: display,
		})
	}

	// =========================================================================
	// Step 3: Build ValueSetMemberships using KB-7 v2 REVERSE LOOKUP
	// =========================================================================
	// KB-7 v2 ARCHITECTURE:
	// OLD: For each of 18,000 ValueSets → check if code is member → O(18,000 * n)
	// NEW: For each patient code → single reverse lookup → O(n)
	//
	// The reverse lookup query returns ALL ValueSets containing the code
	// with SEMANTIC NAMES like "Diabetes", "ACE Inhibitors", "Chronic Kidney Disease"
	// instead of cryptic OIDs like "2.16.840.1.113883.3.464.1003.103.12.1001"
	// =========================================================================

	// Track all semantic names that matched patient codes
	// Key: semantic name (e.g., "Diabetes"), Value: true if patient has matching code
	semanticMemberships := make(map[string]bool)

	// Process all condition codes using reverse lookup
	for _, code := range conditionCodes {
		codeKey := fmt.Sprintf("%s|%s", code.System, code.Code)

		// KB-7 v2: Single reverse lookup returns ALL ValueSets for this code!
		// This replaces the old approach of iterating over 18,000+ ValueSets
		memberships, err := b.kb7FHIRClient.GetMembershipsForCodeWithDetails(ctx, code, false)
		if err != nil {
			// Log warning but continue with other codes
			fmt.Printf("KnowledgeSnapshot warning: reverse lookup failed for %s|%s: %v\n", code.System, code.Code, err)
			continue
		}

		// Process each ValueSet membership from the reverse lookup
		for _, membership := range memberships {
			// Use semantic name as the flag key (KB-7 v2 provides human-readable names!)
			flagName := membership.SemanticName
			if flagName == "" {
				// Fallback to deriving from URL if semantic name not available
				flagName = clients.DeriveSemanticFlagNameWithCategory(membership.ValueSetURL, membership.Category)
			}

			// Mark this semantic flag as true (patient has a matching code)
			semanticMemberships[flagName] = true

			// Also populate CodeMemberships map (tracks which ValueSets each code belongs to)
			snapshot.CodeMemberships[codeKey] = append(snapshot.CodeMemberships[codeKey], membership.SemanticName)
		}
	}

	// Process all medication codes using reverse lookup
	for _, code := range medicationCodes {
		codeKey := fmt.Sprintf("%s|%s", code.System, code.Code)

		// KB-7 v2: Single reverse lookup for medication code
		memberships, err := b.kb7FHIRClient.GetMembershipsForCodeWithDetails(ctx, code, false)
		if err != nil {
			fmt.Printf("KnowledgeSnapshot warning: reverse lookup failed for %s|%s: %v\n", code.System, code.Code, err)
			continue
		}

		for _, membership := range memberships {
			flagName := membership.SemanticName
			if flagName == "" {
				flagName = clients.DeriveSemanticFlagNameWithCategory(membership.ValueSetURL, membership.Category)
			}

			semanticMemberships[flagName] = true
			snapshot.CodeMemberships[codeKey] = append(snapshot.CodeMemberships[codeKey], membership.SemanticName)
		}
	}

	// Process all lab codes using reverse lookup
	// This enables CQL engines to identify HbA1c, eGFR, uACR, etc. through KB-7
	for _, code := range labCodes {
		codeKey := fmt.Sprintf("%s|%s", code.System, code.Code)

		// KB-7 v2: Single reverse lookup for lab LOINC code
		memberships, err := b.kb7FHIRClient.GetMembershipsForCodeWithDetails(ctx, code, false)
		if err != nil {
			fmt.Printf("KnowledgeSnapshot warning: reverse lookup failed for lab %s|%s: %v\n", code.System, code.Code, err)
			continue
		}

		for _, membership := range memberships {
			flagName := membership.SemanticName
			if flagName == "" {
				flagName = clients.DeriveSemanticFlagNameWithCategory(membership.ValueSetURL, membership.Category)
			}

			semanticMemberships[flagName] = true
			// CRITICAL: Populate CodeMemberships so IsHbA1cCode() can find lab codes!
			snapshot.CodeMemberships[codeKey] = append(snapshot.CodeMemberships[codeKey], membership.SemanticName)
		}
	}

	// Copy semantic memberships to snapshot
	// Now flags use semantic names like "HasDiabetes", "OnACEInhibitors" instead of OIDs
	for flagName, hasMembership := range semanticMemberships {
		// Convert semantic name to CQL-friendly flag format
		cqlFlagName := toSemanticFlagName(flagName)
		snapshot.ValueSetMemberships[cqlFlagName] = hasMembership
	}

	// =========================================================================
	// Step 4: Expand ValueSets ONLY for CQL execution (NOT for membership checks!)
	// =========================================================================
	// This is stored for downstream CQL engines that need full expansions.
	// DYNAMIC EXPANSION: Fetch ValueSets to expand from KB-7 based on use_context.
	// KB-7 is the SINGLE SOURCE OF TRUTH - NO hardcoded ValueSetsToExpand!
	valueSetsToExpand := b.config.ValueSetsToExpand // Fallback to config if KB-7 fails

	// Try to get dynamic ValueSet list from KB-7
	dynamicValueSets, err := b.kb7FHIRClient.GetValueSetNamesForContext(ctx, "cql")
	if err == nil && len(dynamicValueSets) > 0 {
		// DYNAMIC: Use KB-7's use_context="cql" ValueSets instead of hardcoded list
		valueSetsToExpand = dynamicValueSets
	}

	for _, vsName := range valueSetsToExpand {
		codes, err := b.kb7FHIRClient.ExpandValueSet(ctx, vsName)
		if err != nil {
			continue // Non-fatal
		}
		snapshot.ExpandedValueSets[vsName] = codes
	}

	b.kbVersions["KB-7"] = "2.0.0-FHIR-ReverseLookup"
	return snapshot, nil
}

// toSemanticFlagName converts a semantic name to a CQL-friendly flag name.
// Examples:
//
//	"Diabetes" → "HasDiabetes"
//	"ACE Inhibitors" → "OnACEInhibitors"
//	"Chronic Kidney Disease" → "HasChronicKidneyDisease"
//	"Essential Hypertension" → "HasEssentialHypertension"
func toSemanticFlagName(semanticName string) string {
	if semanticName == "" {
		return ""
	}

	// Medication-related ValueSets get "On" prefix, conditions get "Has" prefix
	medicationKeywords := []string{
		"inhibitor", "blocker", "anticoagulant", "diuretic", "statin",
		"insulin", "metformin", "medication", "drug", "therapy",
		"agonist", "antagonist", "nsaid", "aspirin", "warfarin",
		"heparin", "doac", "sglt2", "glp1", "arb", "ace",
	}

	prefix := "Has" // Default for conditions
	nameLower := strings.ToLower(semanticName)
	for _, keyword := range medicationKeywords {
		if strings.Contains(nameLower, keyword) {
			prefix = "On"
			break
		}
	}

	// Remove spaces and make CamelCase
	words := strings.Fields(semanticName)
	result := prefix
	for _, word := range words {
		if len(word) > 0 {
			// Capitalize first letter of each word
			result += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return result
}

// ============================================================================
// KB-8: CALCULATOR SNAPSHOT (per CTO/CMO spec - named fields)
// ============================================================================

func (b *KnowledgeSnapshotBuilder) buildCalculatorSnapshot(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculatorSnapshot, error) {

	snapshot := &contracts.CalculatorSnapshot{
		AdditionalCalculations: make(map[string]contracts.CalculationResult),
	}

	if b.kb8Client == nil {
		return snapshot, nil
	}

	// Named calculator fields per CTO/CMO spec
	var wg sync.WaitGroup
	var mu sync.Mutex

	// eGFR - always calculated
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := b.kb8Client.CalculateEGFR(ctx, patient)
		if err == nil {
			mu.Lock()
			snapshot.EGFR = result
			mu.Unlock()
		}
	}()

	// ASCVD 10-year risk
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := b.kb8Client.CalculateASCVD(ctx, patient)
		if err == nil {
			mu.Lock()
			snapshot.ASCVD10Year = result
			mu.Unlock()
		}
	}()

	// CHA2DS2-VASc (for AFib patients)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if hasCondition(patient, "49436004") { // SNOMED: AFib
			result, err := b.kb8Client.CalculateCHA2DS2VASc(ctx, patient)
			if err == nil {
				mu.Lock()
				snapshot.CHA2DS2VASc = result
				mu.Unlock()
			}
		}
	}()

	// HAS-BLED (for patients on anticoagulants)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if hasCondition(patient, "49436004") { // SNOMED: AFib
			result, err := b.kb8Client.CalculateHASBLED(ctx, patient)
			if err == nil {
				mu.Lock()
				snapshot.HASBLED = result
				mu.Unlock()
			}
		}
	}()

	// BMI - always calculated
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := b.kb8Client.CalculateBMI(ctx, patient)
		if err == nil {
			mu.Lock()
			snapshot.BMI = result
			mu.Unlock()
		}
	}()

	// Child-Pugh (for liver disease patients)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if hasLiverDisease(patient) {
			result, err := b.kb8Client.CalculateChildPugh(ctx, patient)
			if err == nil {
				mu.Lock()
				snapshot.ChildPugh = result
				mu.Unlock()
			}
		}
	}()

	// MELD (for liver disease patients)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if hasLiverDisease(patient) {
			result, err := b.kb8Client.CalculateMELD(ctx, patient)
			if err == nil {
				mu.Lock()
				snapshot.MELD = result
				mu.Unlock()
			}
		}
	}()

	// SOFA (for sepsis/ICU)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := b.kb8Client.CalculateSOFA(ctx, patient)
		if err == nil {
			mu.Lock()
			snapshot.SOFA = result
			mu.Unlock()
		}
	}()

	// qSOFA (quick screening)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := b.kb8Client.CalculateQSOFA(ctx, patient)
		if err == nil {
			mu.Lock()
			snapshot.QSOFA = result
			mu.Unlock()
		}
	}()

	wg.Wait()

	b.kbVersions["KB-8"] = "1.0.0"
	return snapshot, nil
}

// ============================================================================
// KB-4: SAFETY SNAPSHOT (per CTO/CMO spec)
// ============================================================================

func (b *KnowledgeSnapshotBuilder) buildSafetySnapshot(
	ctx context.Context,
	patient *contracts.PatientContext,
	medicationCodes []contracts.ClinicalCode,
	conditionCodes []contracts.ClinicalCode,
	eGFR float64,
) (*contracts.SafetySnapshot, error) {

	snapshot := &contracts.SafetySnapshot{
		// Original fields
		ActiveAllergies:             make([]contracts.AllergyInfo, 0),
		Contraindications:           make([]contracts.ContraindicationInfo, 0),
		SafetyAlerts:                make([]contracts.SafetyAlert, 0),
		RenalDoseAdjustmentNeeded:   false,
		HepaticDoseAdjustmentNeeded: false,
		// Enhanced fields
		BlackBoxWarnings:      make(map[string]*contracts.BlackBoxWarningInfo),
		HighAlertMedications:  make(map[string]*contracts.HighAlertInfo),
		PregnancySafetyInfo:   make(map[string]*contracts.PregnancySafetyInfo),
		LactationSafetyInfo:   make(map[string]*contracts.LactationSafetyInfo),
		BeersCriteria:         make(map[string]*contracts.BeersEntryInfo),
		AnticholinergicBurden: make(map[string]*contracts.AnticholinergicInfo),
		LabRequirements:       make(map[string]*contracts.LabRequirementInfo),
		DoseLimits:            make(map[string]*contracts.DoseLimitInfo),
		AgeLimits:             make(map[string]*contracts.AgeLimitInfo),
	}

	if b.kb4Client == nil {
		return snapshot, nil
	}

	// Extract RxNorm codes for enhanced safety checks
	rxnormCodes := extractRxNormCodes(medicationCodes)

	// ============================================================================
	// ORIGINAL KB-4 CALLS (Backward Compatible)
	// ============================================================================

	// Get active allergies
	allergies, err := b.kb4Client.GetActiveAllergies(ctx, patient)
	if err == nil {
		snapshot.ActiveAllergies = allergies
	}

	// Check contraindications
	contraindications, err := b.kb4Client.CheckContraindications(ctx, medicationCodes, conditionCodes)
	if err == nil {
		snapshot.Contraindications = contraindications
	}

	// Get pregnancy status (per CTO/CMO spec)
	pregnancy, err := b.kb4Client.GetPregnancyStatus(ctx, patient)
	if err == nil {
		snapshot.PregnancyStatus = pregnancy
	}

	// Check if renal dose adjustment needed
	if eGFR > 0 {
		needsRenal, err := b.kb4Client.NeedsRenalDoseAdjustment(ctx, eGFR)
		if err == nil {
			snapshot.RenalDoseAdjustmentNeeded = needsRenal
		}
	}

	// Check if hepatic dose adjustment needed
	needsHepatic, err := b.kb4Client.NeedsHepaticDoseAdjustment(ctx, patient)
	if err == nil {
		snapshot.HepaticDoseAdjustmentNeeded = needsHepatic
	}

	// ============================================================================
	// ENHANCED KB-4 CALLS (Full Safety API)
	// ============================================================================

	if len(rxnormCodes) > 0 {
		// Get black box warnings
		blackBoxWarnings, err := b.kb4Client.GetBlackBoxWarnings(ctx, rxnormCodes)
		if err == nil && blackBoxWarnings != nil {
			for code, warning := range blackBoxWarnings {
				if warning != nil {
					snapshot.BlackBoxWarnings[code] = convertBlackBoxWarning(warning)
					snapshot.HasBlackBoxWarnings = true
				}
			}
		}

		// Get high-alert medication status
		highAlerts, err := b.kb4Client.GetHighAlertStatus(ctx, rxnormCodes)
		if err == nil && highAlerts != nil {
			for code, alert := range highAlerts {
				if alert != nil {
					snapshot.HighAlertMedications[code] = convertHighAlertInfo(alert)
					snapshot.HasHighAlertDrugs = true
				}
			}
		}

		// Get pregnancy safety info (if pregnant)
		if snapshot.PregnancyStatus != nil && snapshot.PregnancyStatus.IsPregnant {
			pregnancyInfo, err := b.kb4Client.GetPregnancySafetyInfo(ctx, rxnormCodes)
			if err == nil && pregnancyInfo != nil {
				for code, info := range pregnancyInfo {
					if info != nil {
						snapshot.PregnancySafetyInfo[code] = convertPregnancySafetyInfo(info)
					}
				}
			}
		}

		// Get lactation safety info (if lactating)
		if snapshot.PregnancyStatus != nil && snapshot.PregnancyStatus.LactationStatus {
			lactationInfo, err := b.kb4Client.GetLactationSafetyInfo(ctx, rxnormCodes)
			if err == nil && lactationInfo != nil {
				for code, info := range lactationInfo {
					if info != nil {
						snapshot.LactationSafetyInfo[code] = convertLactationSafetyInfo(info)
					}
				}
			}
		}

		// Get Beers criteria (if patient >= 65 years old)
		patientAge := calculatePatientAge(patient)
		if patientAge >= 65 {
			beersEntries, err := b.kb4Client.GetBeersCriteria(ctx, rxnormCodes)
			if err == nil && beersEntries != nil {
				for code, entry := range beersEntries {
					if entry != nil {
						snapshot.BeersCriteria[code] = convertBeersEntry(entry)
						snapshot.HasBeersWarnings = true
					}
				}
			}
		}

		// Get anticholinergic burden
		acbScores, err := b.kb4Client.GetAnticholinergicBurden(ctx, rxnormCodes)
		if err == nil && acbScores != nil {
			for code, acb := range acbScores {
				if acb != nil {
					snapshot.AnticholinergicBurden[code] = convertAnticholinergicInfo(acb)
				}
			}
		}

		// Calculate total ACB score
		acbCalc, err := b.kb4Client.CalculateTotalAnticholinergicBurden(ctx, rxnormCodes)
		if err == nil && acbCalc != nil {
			snapshot.TotalACBScore = acbCalc.TotalScore
			snapshot.ACBRiskLevel = acbCalc.RiskLevel
		}

		// Get lab requirements
		labReqs, err := b.kb4Client.GetLabRequirements(ctx, rxnormCodes)
		if err == nil && labReqs != nil {
			for code, req := range labReqs {
				if req != nil {
					snapshot.LabRequirements[code] = convertLabRequirement(req)
				}
			}
		}

		// Get dose limits
		doseLimits, err := b.kb4Client.GetDoseLimits(ctx, rxnormCodes)
		if err == nil && doseLimits != nil {
			for code, limit := range doseLimits {
				if limit != nil {
					snapshot.DoseLimits[code] = convertDoseLimit(limit)
				}
			}
		}

		// Get age limits
		ageLimits, err := b.kb4Client.GetAgeLimits(ctx, rxnormCodes)
		if err == nil && ageLimits != nil {
			for code, limit := range ageLimits {
				if limit != nil {
					snapshot.AgeLimits[code] = convertAgeLimit(limit)
				}
			}
		}
	}

	// Generate safety alerts (enhanced with all safety data)
	snapshot.SafetyAlerts = generateSafetyAlerts(snapshot)

	b.kbVersions["KB-4"] = "2.0.0" // Enhanced version
	return snapshot, nil
}

// extractRxNormCodes extracts RxNorm codes from clinical codes.
func extractRxNormCodes(codes []contracts.ClinicalCode) []string {
	rxnormCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		// Check if it's an RxNorm code
		if code.System == "http://www.nlm.nih.gov/research/umls/rxnorm" ||
			strings.Contains(strings.ToLower(code.System), "rxnorm") {
			if code.Code != "" {
				rxnormCodes = append(rxnormCodes, code.Code)
			}
		}
	}
	return rxnormCodes
}

// calculatePatientAge calculates patient age in years.
func calculatePatientAge(patient *contracts.PatientContext) int {
	if patient == nil || patient.Demographics.BirthDate == nil {
		return 0
	}
	birthDate := *patient.Demographics.BirthDate
	now := time.Now()
	years := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}

// ============================================================================
// KB-4 TYPE CONVERTERS
// ============================================================================

func convertBlackBoxWarning(w *clients.KB4BlackBoxWarning) *contracts.BlackBoxWarningInfo {
	return &contracts.BlackBoxWarningInfo{
		RxNormCode:       w.RxNormCode,
		DrugName:         w.DrugName,
		RiskCategories:   w.RiskCategories,
		WarningText:      w.WarningText,
		Severity:         w.Severity,
		HasREMS:          w.HasREMS,
		REMSProgram:      w.REMSProgram,
		REMSRequirements: w.REMSRequirements,
		SourceAuthority:  w.Governance.SourceAuthority,
		SourceDocument:   w.Governance.SourceDocument,
	}
}

func convertHighAlertInfo(a *clients.KB4HighAlertMedication) *contracts.HighAlertInfo {
	return &contracts.HighAlertInfo{
		RxNormCode:             a.RxNormCode,
		DrugName:               a.DrugName,
		Category:               a.Category,
		TallManName:            a.TallManName,
		Requirements:           a.Requirements,
		Safeguards:             a.Safeguards,
		DoubleCheckRequired:    a.DoubleCheck,
		SmartPumpRequired:      a.SmartPump,
		IndependentDoubleCheck: a.IndependentDoubleCheck,
	}
}

func convertPregnancySafetyInfo(p *clients.KB4PregnancySafety) *contracts.PregnancySafetyInfo {
	return &contracts.PregnancySafetyInfo{
		RxNormCode:         p.RxNormCode,
		DrugName:           p.DrugName,
		Category:           p.Category,
		PLLRRiskSummary:    p.PLLRRiskSummary,
		Teratogenic:        p.Teratogenic,
		TeratogenicEffects: p.TeratogenicEffects,
		TrimesterRisks:     p.TrimesterRisks,
		Recommendation:     p.Recommendation,
		AlternativeDrugs:   p.AlternativeDrugs,
		MonitoringRequired: p.MonitoringRequired,
		SourceAuthority:    p.Governance.SourceAuthority,
	}
}

func convertLactationSafetyInfo(l *clients.KB4LactationSafety) *contracts.LactationSafetyInfo {
	return &contracts.LactationSafetyInfo{
		RxNormCode:        l.RxNormCode,
		DrugName:          l.DrugName,
		Risk:              l.Risk,
		RiskSummary:       l.RiskSummary,
		ExcretedInMilk:    l.ExcretedInMilk,
		MilkPlasmaRatio:   l.MilkPlasmaRatio,
		InfantDosePercent: l.InfantDosePercent,
		HalfLifeHours:     l.HalfLifeHours,
		InfantEffects:     l.InfantEffects,
		InfantMonitoring:  l.InfantMonitoring,
		Recommendation:    l.Recommendation,
		AlternativeDrugs:  l.AlternativeDrugs,
		TimingAdvice:      l.TimingAdvice,
	}
}

func convertBeersEntry(b *clients.KB4BeersEntry) *contracts.BeersEntryInfo {
	return &contracts.BeersEntryInfo{
		RxNormCode:               b.RxNormCode,
		DrugName:                 b.DrugName,
		DrugClass:                b.DrugClass,
		Recommendation:           b.Recommendation,
		Rationale:                b.Rationale,
		QualityOfEvidence:        b.QualityOfEvidence,
		StrengthOfRecommendation: b.StrengthOfRecommendation,
		Conditions:               b.Conditions,
		ACBScore:                 b.ACBScore,
		AlternativeDrugs:         b.AlternativeDrugs,
		NonPharmacologic:         b.NonPharmacologic,
		AgeThreshold:             b.AgeThreshold,
	}
}

func convertAnticholinergicInfo(a *clients.KB4AnticholinergicBurden) *contracts.AnticholinergicInfo {
	return &contracts.AnticholinergicInfo{
		RxNormCode:        a.RxNormCode,
		DrugName:          a.DrugName,
		ACBScore:          a.ACBScore,
		RiskLevel:         a.RiskLevel,
		Effects:           a.Effects,
		CognitiveRisk:     a.CognitiveRisk,
		PeripheralEffects: a.PeripheralEffects,
	}
}

func convertLabRequirement(r *clients.KB4LabRequirement) *contracts.LabRequirementInfo {
	return &contracts.LabRequirementInfo{
		RxNormCode:         r.RxNormCode,
		DrugName:           r.DrugName,
		MonitoringRequired: r.MonitoringRequired,
		CriticalMonitoring: r.CriticalMonitoring,
		REMSProgram:        r.REMSProgram,
		RequiredLabs:       r.RequiredLabs,
		LabCodes:           r.LabCodes,
		Frequency:          r.Frequency,
		BaselineRequired:   r.BaselineRequired,
		InitialMonitoring:  r.InitialMonitoring,
		OngoingMonitoring:  r.OngoingMonitoring,
		CriticalValues:     r.CriticalValues,
		ActionRequired:     r.ActionRequired,
	}
}

func convertDoseLimit(d *clients.KB4DoseLimit) *contracts.DoseLimitInfo {
	return &contracts.DoseLimitInfo{
		RxNormCode:         d.RxNormCode,
		DrugName:           d.DrugName,
		MaxSingleDose:      d.MaxSingleDose,
		MaxSingleDoseUnit:  d.MaxSingleDoseUnit,
		MaxDailyDose:       d.MaxDailyDose,
		MaxDailyDoseUnit:   d.MaxDailyDoseUnit,
		MaxCumulativeDose:  d.MaxCumulativeDose,
		GeriatricMaxDose:   d.GeriatricMaxDose,
		PediatricMaxDose:   d.PediatricMaxDose,
		RenalAdjustment:    d.RenalAdjustment,
		HepaticAdjustment:  d.HepaticAdjustment,
		RenalDoseByEGFR:    d.RenalDoseByEGFR,
		HepaticDoseByClass: d.HepaticDoseByClass,
	}
}

func convertAgeLimit(a *clients.KB4AgeLimit) *contracts.AgeLimitInfo {
	return &contracts.AgeLimitInfo{
		RxNormCode:  a.RxNormCode,
		DrugName:    a.DrugName,
		MinAgeYears: int(a.MinAgeYears),
		MaxAgeYears: int(a.MaxAgeYears),
		Rationale:   a.Rationale,
		Severity:    a.Severity,
	}
}

// ============================================================================
// KB-5: INTERACTION SNAPSHOT (per CTO/CMO spec)
// ============================================================================

func (b *KnowledgeSnapshotBuilder) buildInteractionSnapshot(
	ctx context.Context,
	medicationCodes []contracts.ClinicalCode,
) (*contracts.InteractionSnapshot, error) {

	snapshot := &contracts.InteractionSnapshot{
		CurrentDDIs:            make([]contracts.DrugInteraction, 0),
		PotentialDDIs:          make([]contracts.DrugInteraction, 0),
		SeverityMax:            "none",
		HasCriticalInteraction: false,
	}

	if b.kb5Client == nil {
		return snapshot, nil
	}

	// Get current interactions (between patient's active meds)
	current, err := b.kb5Client.GetCurrentInteractions(ctx, medicationCodes)
	if err == nil {
		snapshot.CurrentDDIs = current
	}

	// Get potential interactions (if adding common drugs)
	potential, err := b.kb5Client.GetPotentialInteractions(ctx, medicationCodes)
	if err == nil {
		snapshot.PotentialDDIs = potential
	}

	// Calculate SeverityMax and HasCriticalInteraction
	snapshot.SeverityMax, snapshot.HasCriticalInteraction = calculateInteractionSeverity(
		append(snapshot.CurrentDDIs, snapshot.PotentialDDIs...),
	)

	b.kbVersions["KB-5"] = "1.0.0"
	return snapshot, nil
}

// ============================================================================
// KB-6: FORMULARY SNAPSHOT (per CTO/CMO spec)
// ============================================================================

func (b *KnowledgeSnapshotBuilder) buildFormularySnapshot(
	ctx context.Context,
	medicationCodes []contracts.ClinicalCode,
) (*contracts.FormularySnapshot, error) {

	snapshot := &contracts.FormularySnapshot{
		MedicationStatus:    make(map[string]contracts.FormularyStatus),
		PriorAuthRequired:   make([]contracts.ClinicalCode, 0),
		GenericAlternatives: make(map[string][]contracts.ClinicalCode),
		NLEMAvailability:    make(map[string]bool),
		PBSAvailability:     make(map[string]bool),
	}

	if b.kb6Client == nil {
		return snapshot, nil
	}

	// Get formulary status
	status, err := b.kb6Client.GetFormularyStatus(ctx, medicationCodes, b.config.Region)
	if err == nil {
		snapshot.MedicationStatus = status
	}

	// Get prior auth requirements
	priorAuth, err := b.kb6Client.GetPriorAuthRequired(ctx, medicationCodes)
	if err == nil {
		snapshot.PriorAuthRequired = priorAuth
	}

	// Get generic alternatives for each medication
	for _, med := range medicationCodes {
		alts, err := b.kb6Client.GetGenericAlternatives(ctx, med)
		if err == nil && len(alts) > 0 {
			key := fmt.Sprintf("%s|%s", med.System, med.Code)
			snapshot.GenericAlternatives[key] = alts
		}
	}

	// Get NLEM availability (India)
	if b.config.Region == "IN" {
		nlem, err := b.kb6Client.GetNLEMAvailability(ctx, medicationCodes)
		if err == nil {
			snapshot.NLEMAvailability = nlem
		}
	}

	// Get PBS availability (Australia)
	if b.config.Region == "AU" {
		pbs, err := b.kb6Client.GetPBSAvailability(ctx, medicationCodes)
		if err == nil {
			snapshot.PBSAvailability = pbs
		}
	}

	b.kbVersions["KB-6"] = "1.0.0"
	return snapshot, nil
}

// ============================================================================
// KB-1: DOSING SNAPSHOT (per CTO/CMO spec)
// ============================================================================

func (b *KnowledgeSnapshotBuilder) buildDosingSnapshot(
	ctx context.Context,
	patient *contracts.PatientContext,
	medicationCodes []contracts.ClinicalCode,
	eGFR float64,
	childPugh string,
) (*contracts.DosingSnapshot, error) {

	snapshot := &contracts.DosingSnapshot{
		RenalAdjustments:    make(map[string]contracts.DoseAdjustment),
		HepaticAdjustments:  make(map[string]contracts.DoseAdjustment),
		WeightBasedDoses:    make(map[string]contracts.DoseCalculation),
		AgeBasedAdjustments: make(map[string]contracts.DoseAdjustment),
	}

	if b.kb1Client == nil {
		return snapshot, nil
	}

	// Get renal dose adjustments
	if eGFR < 90 { // Only if kidney function impaired
		renal, err := b.kb1Client.GetRenalAdjustments(ctx, medicationCodes, eGFR)
		if err == nil {
			snapshot.RenalAdjustments = renal
		}
	}

	// Get hepatic dose adjustments
	if childPugh != "" && childPugh != "A" { // Only if liver function impaired
		hepatic, err := b.kb1Client.GetHepaticAdjustments(ctx, medicationCodes, childPugh)
		if err == nil {
			snapshot.HepaticAdjustments = hepatic
		}
	}

	// Get weight-based doses
	weight := extractWeight(patient)
	bsa := extractBSA(patient)
	if weight > 0 {
		weightDoses, err := b.kb1Client.GetWeightBasedDoses(ctx, medicationCodes, weight, bsa)
		if err == nil {
			snapshot.WeightBasedDoses = weightDoses
		}
	}

	// Get age-based adjustments (pediatric/geriatric)
	age := extractAge(patient)
	if age < 18 || age > 65 {
		ageDoses, err := b.kb1Client.GetAgeBasedAdjustments(ctx, medicationCodes, age)
		if err == nil {
			snapshot.AgeBasedAdjustments = ageDoses
		}
	}

	b.kbVersions["KB-1"] = "1.0.0"
	return snapshot, nil
}

// ============================================================================
// KB-11: CDI FACTS
// ============================================================================

func (b *KnowledgeSnapshotBuilder) buildCDIFacts(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CDIFacts, error) {

	facts := &contracts.CDIFacts{
		ExtractedFacts:      make([]contracts.CDIFact, 0),
		CodingOpportunities: make([]contracts.CodingOpportunity, 0),
		QueryOpportunities:  make([]contracts.QueryOpportunity, 0),
	}

	if b.kb11Client == nil {
		return facts, nil
	}

	// Extract clinical facts
	extracted, err := b.kb11Client.ExtractFacts(ctx, patient)
	if err == nil {
		facts.ExtractedFacts = extracted
	}

	// Get coding opportunities
	coding, err := b.kb11Client.GetCodingOpportunities(ctx, patient)
	if err == nil {
		facts.CodingOpportunities = coding
	}

	// Get query opportunities
	queries, err := b.kb11Client.GetQueryOpportunities(ctx, patient)
	if err == nil {
		facts.QueryOpportunities = queries
	}

	b.kbVersions["KB-11"] = "1.0.0"
	return facts, nil
}

// ============================================================================
// KB-16: LAB INTERPRETATION SNAPSHOT (Category A - SNAPSHOT KB)
// ============================================================================

// buildLabInterpretationSnapshot builds pre-computed lab interpretations for snapshot.
//
// ARCHITECTURE (CTO/CMO Directive):
// KB-16 is a Category A SNAPSHOT KB - lab reference ranges and interpretations
// are pre-computed at build time. CQL evaluates against frozen lab data.
// Engines NEVER call KB-16 directly at execution time.
//
// This method:
// - Extracts patient demographics for age/sex-appropriate reference ranges
// - Gets reference ranges for all recent lab results
// - Pre-computes lab interpretations with abnormality flags
// - Identifies critical/panic values for immediate attention
func (b *KnowledgeSnapshotBuilder) buildLabInterpretationSnapshot(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.LabInterpretationSnapshot, error) {

	snapshot := &contracts.LabInterpretationSnapshot{
		ReferenceRanges:      make(map[string]contracts.LabReferenceRange),
		CriticalValues:       make(map[string]contracts.LabCriticalValue),
		LabInterpretations:   make(map[string]contracts.LabInterpretationResult),
		PanelInterpretations: make(map[string]contracts.LabPanelResult),
		HasCriticalValue:     false,
		CriticalLabCodes:     make([]string, 0),
	}

	if b.kb16Client == nil {
		return snapshot, nil
	}

	// Build patient demographics for reference range selection
	demographics := b.buildKB16Demographics(patient)

	// Collect LOINC codes from patient's recent lab results
	loincCodes := b.extractLabLOINCCodes(patient)
	if len(loincCodes) == 0 {
		// No labs to interpret, but still get common reference ranges
		loincCodes = commonLabLOINCCodes
	}

	// Get reference ranges for all relevant LOINC codes
	refRanges, err := b.kb16Client.GetReferenceRangesForPatient(ctx, demographics, loincCodes)
	if err == nil && refRanges != nil {
		for loincCode, refRange := range refRanges {
			if refRange != nil {
				snapshot.ReferenceRanges[loincCode] = contracts.LabReferenceRange{
					LOINCCode:        refRange.LOINCCode,
					DisplayName:      refRange.DisplayName,
					LowNormal:        refRange.LowNormal,
					HighNormal:       refRange.HighNormal,
					Unit:             refRange.Unit,
					CriticalLow:      refRange.CriticalLow,
					CriticalHigh:     refRange.CriticalHigh,
					AgeMin:           refRange.AgeMin,
					AgeMax:           refRange.AgeMax,
					Sex:              refRange.Sex,
					IsPregnancyRange: refRange.IsPregnancyRange,
					Source:           refRange.Source,
				}
			}
		}
	}

	// Get critical values for key labs
	criticalLabCodes := []string{
		"2345-7",  // Glucose
		"2823-3",  // Potassium
		"2951-2",  // Sodium
		"718-7",   // Hemoglobin
		"777-3",   // Platelets
		"6301-6",  // INR
		"10839-9", // Troponin I
		"49498-9", // Lactate
	}
	for _, loincCode := range criticalLabCodes {
		critVals, err := b.kb16Client.GetCriticalValues(ctx, loincCode)
		if err == nil && critVals != nil {
			snapshot.CriticalValues[loincCode] = contracts.LabCriticalValue{
				LOINCCode:           critVals.LOINCCode,
				DisplayName:         critVals.DisplayName,
				CriticalLow:         critVals.CriticalLow,
				CriticalHigh:        critVals.CriticalHigh,
				Unit:                critVals.Unit,
				RequiredAction:      critVals.RequiredAction,
				NotifyWithinMinutes: critVals.NotifyWithinMinutes,
				Source:              critVals.Source,
			}
		}
	}

	// Batch interpret patient's recent lab results
	kb16Labs := b.convertLabsToKB16Format(patient)
	if len(kb16Labs) > 0 {
		interpretations, err := b.kb16Client.BatchInterpretLabs(ctx, kb16Labs, demographics)
		if err == nil && interpretations != nil {
			for _, interp := range interpretations {
				key := interp.LOINCCode
				snapshot.LabInterpretations[key] = contracts.LabInterpretationResult{
					LOINCCode:            interp.LOINCCode,
					DisplayName:          interp.DisplayName,
					Value:                interp.Value,
					Unit:                 interp.Unit,
					AbnormalityLevel:     interp.AbnormalityLevel,
					Flag:                 interp.Flag,
					IsCritical:           interp.IsCritical,
					ClinicalSignificance: interp.ClinicalSignificance,
					PossibleCauses:       interp.PossibleCauses,
					SuggestedActions:     interp.SuggestedActions,
					Trend:                interp.Trend,
					Narrative:            interp.Narrative,
				}

				// Track critical values
				if interp.IsCritical {
					snapshot.HasCriticalValue = true
					snapshot.CriticalLabCodes = append(snapshot.CriticalLabCodes, interp.LOINCCode)
				}
			}
		}
	}

	b.kbVersions["KB-16"] = "1.0.0"
	return snapshot, nil
}

// buildKB16Demographics converts PatientContext demographics to KB16 format.
func (b *KnowledgeSnapshotBuilder) buildKB16Demographics(patient *contracts.PatientContext) KB16PatientDemographics {
	demo := KB16PatientDemographics{
		Region: b.config.Region,
	}

	// Calculate age
	if patient.Demographics.BirthDate != nil {
		demo.AgeYears = int(time.Since(*patient.Demographics.BirthDate).Hours() / 24 / 365)
	}

	// Map gender
	demo.Sex = patient.Demographics.Gender

	// Check pregnancy status from RiskProfile flags if available
	if patient.RiskProfile.ClinicalFlags != nil {
		if isPregnant, ok := patient.RiskProfile.ClinicalFlags["is_pregnant"]; ok {
			demo.IsPregnant = isPregnant
		}
	}

	return demo
}

// extractLabLOINCCodes extracts unique LOINC codes from patient's recent labs.
func (b *KnowledgeSnapshotBuilder) extractLabLOINCCodes(patient *contracts.PatientContext) []string {
	seen := make(map[string]bool)
	codes := make([]string, 0)

	for _, lab := range patient.RecentLabResults {
		if lab.Code.Code != "" && !seen[lab.Code.Code] {
			seen[lab.Code.Code] = true
			codes = append(codes, lab.Code.Code)
		}
	}

	return codes
}

// convertLabsToKB16Format converts PatientContext labs to KB16 format.
func (b *KnowledgeSnapshotBuilder) convertLabsToKB16Format(patient *contracts.PatientContext) []KB16LabResult {
	labs := make([]KB16LabResult, 0, len(patient.RecentLabResults))

	for _, lab := range patient.RecentLabResults {
		if lab.Value == nil {
			continue // Skip labs without numeric values
		}

		kb16Lab := KB16LabResult{
			LOINCCode:   lab.Code.Code,
			DisplayName: lab.Code.Display,
			Value:       lab.Value.Value,
			Unit:        lab.Value.Unit,
			LabFlag:     lab.Interpretation,
		}

		if lab.EffectiveDateTime != nil {
			kb16Lab.CollectedAt = *lab.EffectiveDateTime
		}

		labs = append(labs, kb16Lab)
	}

	return labs
}

// commonLabLOINCCodes contains frequently used lab LOINC codes for snapshot building.
// Used when patient has no recent labs but we still want reference ranges available.
var commonLabLOINCCodes = []string{
	// Basic Metabolic Panel
	"2345-7",  // Glucose
	"2160-0",  // Creatinine
	"3094-0",  // BUN
	"2951-2",  // Sodium
	"2823-3",  // Potassium
	"2075-0",  // Chloride
	"1963-8",  // Bicarbonate
	"17861-6", // Calcium

	// Complete Blood Count
	"718-7",  // Hemoglobin
	"4544-3", // Hematocrit
	"6690-2", // WBC
	"777-3",  // Platelet Count

	// Lipid Panel
	"2093-3",  // Total Cholesterol
	"2085-9",  // HDL
	"13457-7", // LDL
	"2571-8",  // Triglycerides

	// Liver Function
	"1742-6", // ALT
	"1920-8", // AST
	"1975-2", // Bilirubin Total

	// Coagulation
	"6301-6", // INR
	"5902-2", // PT

	// Cardiac
	"10839-9", // Troponin I
	"33762-6", // NT-proBNP

	// Renal
	"33914-3", // eGFR

	// Diabetes
	"4548-4", // HbA1c
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func extractConditionCodes(patient *contracts.PatientContext) []contracts.ClinicalCode {
	codes := make([]contracts.ClinicalCode, 0, len(patient.ActiveConditions))
	for _, c := range patient.ActiveConditions {
		codes = append(codes, c.Code)
	}
	return codes
}

func extractMedicationCodes(patient *contracts.PatientContext) []contracts.ClinicalCode {
	codes := make([]contracts.ClinicalCode, 0, len(patient.ActiveMedications))
	for _, m := range patient.ActiveMedications {
		codes = append(codes, m.Code)
	}
	return codes
}

// extractLabCodes extracts LOINC codes from patient lab results.
// These codes are sent to KB-7 for reverse lookup to determine ValueSet memberships
// (e.g., HbA1c "4548-4" → "LabHbA1c" ValueSet).
func extractLabCodes(patient *contracts.PatientContext) []contracts.ClinicalCode {
	codes := make([]contracts.ClinicalCode, 0, len(patient.RecentLabResults))
	for _, lab := range patient.RecentLabResults {
		codes = append(codes, lab.Code)
	}
	return codes
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func getEGFRValue(calcs *contracts.CalculatorSnapshot) float64 {
	if calcs.EGFR != nil {
		return calcs.EGFR.Value
	}
	return 90.0 // Default normal
}

func getChildPughClass(calcs *contracts.CalculatorSnapshot) string {
	if calcs.ChildPugh != nil {
		return calcs.ChildPugh.Category
	}
	return ""
}

func hasCondition(patient *contracts.PatientContext, snomedCode string) bool {
	for _, cond := range patient.ActiveConditions {
		if cond.Code.Code == snomedCode {
			return true
		}
	}
	return false
}

func hasLiverDisease(patient *contracts.PatientContext) bool {
	liverCodes := []string{
		"235856003", // Hepatic impairment
		"19943007",  // Cirrhosis
		"197321007", // Chronic hepatitis
	}
	for _, cond := range patient.ActiveConditions {
		for _, code := range liverCodes {
			if cond.Code.Code == code {
				return true
			}
		}
	}
	return false
}

func extractWeight(patient *contracts.PatientContext) float64 {
	for _, vs := range patient.RecentVitalSigns {
		if vs.Code.Code == "29463-7" { // LOINC: Body weight
			if vs.Value != nil {
				return vs.Value.Value
			}
		}
	}
	return 0
}

func extractBSA(patient *contracts.PatientContext) float64 {
	for _, vs := range patient.RecentVitalSigns {
		if vs.Code.Code == "8277-6" { // LOINC: Body surface area
			if vs.Value != nil {
				return vs.Value.Value
			}
		}
	}
	return 0
}

func extractAge(patient *contracts.PatientContext) int {
	if patient.Demographics.BirthDate == nil {
		return 0
	}
	return int(time.Since(*patient.Demographics.BirthDate).Hours() / 24 / 365)
}

func calculateInteractionSeverity(interactions []contracts.DrugInteraction) (string, bool) {
	severityOrder := map[string]int{
		"none":     0,
		"low":      1,
		"mild":     1,
		"medium":   2,
		"moderate": 2,
		"high":     3,
		"severe":   3,
		"critical": 4,
	}

	maxSeverity := "none"
	hasCritical := false

	for _, ddi := range interactions {
		if severityOrder[ddi.Severity] > severityOrder[maxSeverity] {
			maxSeverity = ddi.Severity
		}
		if ddi.Severity == "critical" {
			hasCritical = true
		}
	}

	return maxSeverity, hasCritical
}

func generateSafetyAlerts(safety *contracts.SafetySnapshot) []contracts.SafetyAlert {
	alerts := make([]contracts.SafetyAlert, 0)
	now := time.Now()

	// Generate alerts for high-criticality allergies
	for _, allergy := range safety.ActiveAllergies {
		if allergy.Criticality == "high" {
			alerts = append(alerts, contracts.SafetyAlert{
				AlertID:     fmt.Sprintf("ALLERGY-%s", allergy.Allergen.Code),
				Type:        "allergy",
				Severity:    "high",
				Title:       fmt.Sprintf("High-Risk Allergy: %s", allergy.Allergen.Display),
				Description: fmt.Sprintf("Patient has high-criticality allergy to %s", allergy.Allergen.Display),
				CreatedAt:   now,
			})
		}
	}

	// Generate alerts for contraindications
	for _, ci := range safety.Contraindications {
		alerts = append(alerts, contracts.SafetyAlert{
			AlertID:     fmt.Sprintf("CI-%s-%s", ci.Medication.Code, ci.Condition.Code),
			Type:        "contraindication",
			Severity:    ci.Severity,
			Title:       fmt.Sprintf("Contraindication: %s with %s", ci.Medication.Display, ci.Condition.Display),
			Description: ci.Description,
			CreatedAt:   now,
		})
	}

	// Generate alert for pregnancy (if applicable)
	if safety.PregnancyStatus != nil && safety.PregnancyStatus.IsPregnant {
		alerts = append(alerts, contracts.SafetyAlert{
			AlertID:     "PREGNANCY-ACTIVE",
			Type:        "pregnancy",
			Severity:    "high",
			Title:       "Pregnancy Alert",
			Description: fmt.Sprintf("Patient is pregnant (Trimester %d). Review all medications for pregnancy safety.", safety.PregnancyStatus.Trimester),
			CreatedAt:   now,
		})
	}

	return alerts
}
