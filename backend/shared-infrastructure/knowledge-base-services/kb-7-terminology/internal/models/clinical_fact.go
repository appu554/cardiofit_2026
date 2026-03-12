package models

import (
	"time"
)

// ============================================================================
// Clinical Fact Types for CDSS
// ============================================================================
// ClinicalFact represents a normalized clinical assertion extracted from
// FHIR resources. Facts are the input to the CDSS evaluation pipeline.

// FactType represents the type of clinical fact
type FactType string

const (
	FactTypeCondition   FactType = "condition"
	FactTypeObservation FactType = "observation"
	FactTypeMedication  FactType = "medication"
	FactTypeProcedure   FactType = "procedure"
	FactTypeAllergy     FactType = "allergy"
	FactTypeLab         FactType = "lab"
	FactTypeVitalSign   FactType = "vital_sign"
)

// FactStatus represents the clinical status of a fact
type FactStatus string

const (
	FactStatusActive    FactStatus = "active"
	FactStatusInactive  FactStatus = "inactive"
	FactStatusResolved  FactStatus = "resolved"
	FactStatusPending   FactStatus = "pending"
	FactStatusCompleted FactStatus = "completed"
	FactStatusUnknown   FactStatus = "unknown"
)

// ClinicalFact represents a single clinical fact extracted from a FHIR resource
type ClinicalFact struct {
	// Unique identifier for this fact
	ID string `json:"id"`

	// Type of clinical fact
	FactType FactType `json:"fact_type"`

	// Clinical status
	Status FactStatus `json:"status"`

	// Primary code from terminology
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`

	// Additional codings (if multiple codes present)
	AdditionalCodes []FactCoding `json:"additional_codes,omitempty"`

	// Numeric value (for observations/labs)
	NumericValue *float64 `json:"numeric_value,omitempty"`
	Unit         string   `json:"unit,omitempty"`

	// Value interpretation (for observations)
	Interpretation string `json:"interpretation,omitempty"`

	// Reference ranges (for labs)
	ReferenceRangeLow  *float64 `json:"reference_range_low,omitempty"`
	ReferenceRangeHigh *float64 `json:"reference_range_high,omitempty"`

	// Boolean or coded value (for non-numeric observations)
	BooleanValue *bool  `json:"boolean_value,omitempty"`
	CodedValue   string `json:"coded_value,omitempty"`

	// Severity/criticality
	Severity    string `json:"severity,omitempty"`    // For conditions
	Criticality string `json:"criticality,omitempty"` // For allergies

	// Body site (if applicable)
	BodySite *FactCoding `json:"body_site,omitempty"`

	// Timing information
	EffectiveDateTime *time.Time `json:"effective_datetime,omitempty"`
	OnsetDateTime     *time.Time `json:"onset_datetime,omitempty"`
	RecordedDateTime  *time.Time `json:"recorded_datetime,omitempty"`

	// Source FHIR resource reference
	SourceResourceType string `json:"source_resource_type"`
	SourceResourceID   string `json:"source_resource_id"`

	// Category (e.g., "laboratory", "vital-signs", "encounter-diagnosis")
	Category string `json:"category,omitempty"`

	// Clinical domain (derived from code or category)
	ClinicalDomain string `json:"clinical_domain,omitempty"`

	// Flags for clinical significance
	IsAbnormal bool `json:"is_abnormal,omitempty"`
	IsCritical bool `json:"is_critical,omitempty"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FactCoding represents a code from a terminology system
type FactCoding struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display,omitempty"`
	Version string `json:"version,omitempty"`
}

// PatientFactSet represents all clinical facts for a patient
type PatientFactSet struct {
	// Patient identifier
	PatientID string `json:"patient_id"`

	// Encounter context (if applicable)
	EncounterID string `json:"encounter_id,omitempty"`

	// Timestamp when facts were extracted
	ExtractedAt time.Time `json:"extracted_at"`

	// All facts organized by type
	Conditions   []ClinicalFact `json:"conditions,omitempty"`
	Observations []ClinicalFact `json:"observations,omitempty"`
	Medications  []ClinicalFact `json:"medications,omitempty"`
	Procedures   []ClinicalFact `json:"procedures,omitempty"`
	Allergies    []ClinicalFact `json:"allergies,omitempty"`

	// Summary statistics
	TotalFacts       int `json:"total_facts"`
	ActiveConditions int `json:"active_conditions"`
	ActiveMedications int `json:"active_medications"`
	AbnormalLabs     int `json:"abnormal_labs"`
	CriticalFindings int `json:"critical_findings"`

	// Processing metadata
	SourceBundleID   string   `json:"source_bundle_id,omitempty"`
	ProcessingErrors []string `json:"processing_errors,omitempty"`
}

// GetAllFacts returns all facts as a single slice
func (p *PatientFactSet) GetAllFacts() []ClinicalFact {
	totalCap := len(p.Conditions) + len(p.Observations) + len(p.Medications) + len(p.Procedures) + len(p.Allergies)
	allFacts := make([]ClinicalFact, 0, totalCap)
	allFacts = append(allFacts, p.Conditions...)
	allFacts = append(allFacts, p.Observations...)
	allFacts = append(allFacts, p.Medications...)
	allFacts = append(allFacts, p.Procedures...)
	allFacts = append(allFacts, p.Allergies...)
	return allFacts
}

// GetFactsBySystem returns facts filtered by terminology system
func (p *PatientFactSet) GetFactsBySystem(system string) []ClinicalFact {
	var facts []ClinicalFact
	for _, fact := range p.GetAllFacts() {
		if fact.System == system {
			facts = append(facts, fact)
		}
	}
	return facts
}

// GetFactsByDomain returns facts filtered by clinical domain
func (p *PatientFactSet) GetFactsByDomain(domain string) []ClinicalFact {
	var facts []ClinicalFact
	for _, fact := range p.GetAllFacts() {
		if fact.ClinicalDomain == domain {
			facts = append(facts, fact)
		}
	}
	return facts
}

// GetActiveFacts returns only active facts
func (p *PatientFactSet) GetActiveFacts() []ClinicalFact {
	var facts []ClinicalFact
	for _, fact := range p.GetAllFacts() {
		if fact.Status == FactStatusActive {
			facts = append(facts, fact)
		}
	}
	return facts
}

// UpdateStatistics recalculates summary statistics
func (p *PatientFactSet) UpdateStatistics() {
	p.TotalFacts = len(p.Conditions) + len(p.Observations) + len(p.Medications) + len(p.Procedures) + len(p.Allergies)

	p.ActiveConditions = 0
	for _, c := range p.Conditions {
		if c.Status == FactStatusActive {
			p.ActiveConditions++
		}
	}

	p.ActiveMedications = 0
	for _, m := range p.Medications {
		if m.Status == FactStatusActive {
			p.ActiveMedications++
		}
	}

	p.AbnormalLabs = 0
	p.CriticalFindings = 0
	for _, o := range p.Observations {
		if o.IsAbnormal {
			p.AbnormalLabs++
		}
		if o.IsCritical {
			p.CriticalFindings++
		}
	}
}

// ============================================================================
// Fact Builder Options and Request/Response Types
// ============================================================================

// FactBuilderOptions configures fact extraction behavior
type FactBuilderOptions struct {
	// Include inactive/resolved conditions
	IncludeInactive bool `json:"include_inactive"`

	// Include only verified conditions
	OnlyVerified bool `json:"only_verified"`

	// Filter by clinical domains
	ClinicalDomains []string `json:"clinical_domains,omitempty"`

	// Filter by terminology systems
	TerminologySystems []string `json:"terminology_systems,omitempty"`

	// Include all codings or just primary
	IncludeAllCodings bool `json:"include_all_codings"`

	// Time range filter (only include facts within this range)
	EffectiveAfter  *time.Time `json:"effective_after,omitempty"`
	EffectiveBefore *time.Time `json:"effective_before,omitempty"`

	// Maximum number of facts to extract (0 = unlimited)
	MaxFacts int `json:"max_facts,omitempty"`

	// Derive clinical domains from codes
	DeriveClinicalDomains bool `json:"derive_clinical_domains"`

	// Extract nested components (e.g., observation components)
	ExtractComponents bool `json:"extract_components"`
}

// DefaultFactBuilderOptions returns sensible default options
func DefaultFactBuilderOptions() *FactBuilderOptions {
	return &FactBuilderOptions{
		IncludeInactive:       false,
		OnlyVerified:          false,
		IncludeAllCodings:     true,
		DeriveClinicalDomains: true,
		ExtractComponents:     true,
	}
}

// FactBuilderRequest represents a request to build facts from FHIR resources
type FactBuilderRequest struct {
	// Patient identifier
	PatientID string `json:"patient_id" binding:"required"`

	// Optional encounter context
	EncounterID string `json:"encounter_id,omitempty"`

	// FHIR Bundle (if providing a bundle)
	Bundle *FHIRBundle `json:"bundle,omitempty"`

	// Individual FHIR resources
	Conditions   []FHIRCondition         `json:"conditions,omitempty"`
	Observations []FHIRObservation       `json:"observations,omitempty"`
	Medications  []FHIRMedicationRequest `json:"medications,omitempty"`
	Procedures   []FHIRProcedure         `json:"procedures,omitempty"`
	Allergies    []FHIRAllergyIntolerance `json:"allergies,omitempty"`

	// Extraction options
	Options *FactBuilderOptions `json:"options,omitempty"`
}

// HasResources returns true if the request contains any FHIR resources
func (r *FactBuilderRequest) HasResources() bool {
	return r.Bundle != nil ||
		len(r.Conditions) > 0 ||
		len(r.Observations) > 0 ||
		len(r.Medications) > 0 ||
		len(r.Procedures) > 0 ||
		len(r.Allergies) > 0
}

// FactBuilderResponse represents the result of fact building
type FactBuilderResponse struct {
	// Success indicator
	Success bool `json:"success"`

	// The extracted facts
	FactSet *PatientFactSet `json:"fact_set,omitempty"`

	// Processing summary
	ConditionsProcessed   int `json:"conditions_processed"`
	ObservationsProcessed int `json:"observations_processed"`
	MedicationsProcessed  int `json:"medications_processed"`
	ProceduresProcessed   int `json:"procedures_processed"`
	AllergiesProcessed    int `json:"allergies_processed"`

	// Extraction statistics
	TotalResourcesProcessed int `json:"total_resources_processed"`
	TotalFactsExtracted     int `json:"total_facts_extracted"`
	FactsFiltered           int `json:"facts_filtered"`

	// Processing time
	ProcessingTimeMs float64 `json:"processing_time_ms"`

	// Errors and warnings
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ============================================================================
// Clinical Domain Definitions
// ============================================================================

// ClinicalDomain represents a clinical specialty or area
type ClinicalDomain string

const (
	DomainSepsis       ClinicalDomain = "sepsis"
	DomainRenal        ClinicalDomain = "renal"
	DomainCardiac      ClinicalDomain = "cardiac"
	DomainRespiratory  ClinicalDomain = "respiratory"
	DomainMetabolic    ClinicalDomain = "metabolic"
	DomainNeurological ClinicalDomain = "neurological"
	DomainHematologic  ClinicalDomain = "hematologic"
	DomainInfectious   ClinicalDomain = "infectious"
	DomainEndocrine    ClinicalDomain = "endocrine"
	DomainGI           ClinicalDomain = "gastrointestinal"
	DomainMSK          ClinicalDomain = "musculoskeletal"
	DomainDermatologic ClinicalDomain = "dermatologic"
	DomainOncologic    ClinicalDomain = "oncologic"
	DomainPsychiatric  ClinicalDomain = "psychiatric"
	DomainGeneral      ClinicalDomain = "general"
)

// AllClinicalDomains returns all known clinical domains
func AllClinicalDomains() []ClinicalDomain {
	return []ClinicalDomain{
		DomainSepsis,
		DomainRenal,
		DomainCardiac,
		DomainRespiratory,
		DomainMetabolic,
		DomainNeurological,
		DomainHematologic,
		DomainInfectious,
		DomainEndocrine,
		DomainGI,
		DomainMSK,
		DomainDermatologic,
		DomainOncologic,
		DomainPsychiatric,
		DomainGeneral,
	}
}

// String returns the string representation of a clinical domain
func (d ClinicalDomain) String() string {
	return string(d)
}

// ============================================================================
// Value Set to Domain Mapping
// ============================================================================

// ValueSetDomainMapping maps value set identifiers to clinical domains
var ValueSetDomainMapping = map[string]ClinicalDomain{
	// Sepsis domain
	"SepsisDiagnosis":    DomainSepsis,
	"AUSepsisConditions": DomainSepsis,
	"SepsisIndicators":   DomainSepsis,

	// Renal domain
	"AcuteRenalFailure":      DomainRenal,
	"AUAKIConditions":        DomainRenal,
	"ChronicKidneyDisease":   DomainRenal,
	"RenalLabTests":          DomainRenal,
	"NephrotoxicMedications": DomainRenal,

	// Cardiac domain
	"HeartFailure":              DomainCardiac,
	"AcuteCoronarySyndrome":     DomainCardiac,
	"CardiacArrhythmias":        DomainCardiac,
	"Hypertension":              DomainCardiac,
	"CardiacBiomarkers":         DomainCardiac,
	"AnticoagulantMedications":  DomainCardiac,
	"AntiarrhythmicMedications": DomainCardiac,
	"AntihypertensiveMedications": DomainCardiac,

	// Respiratory domain
	"RespiratoryFailure":       DomainRespiratory,
	"COPD":                     DomainRespiratory,
	"Asthma":                   DomainRespiratory,
	"Pneumonia":                DomainRespiratory,
	"RespiratoryLabTests":      DomainRespiratory,
	"BronchodilatorMedications": DomainRespiratory,

	// Metabolic domain
	"DiabetesMellitus":           DomainMetabolic,
	"MetabolicSyndrome":          DomainMetabolic,
	"Hypoglycemia":               DomainMetabolic,
	"ElectrolyteDisorders":       DomainMetabolic,
	"GlucoseMonitoring":          DomainMetabolic,
	"AntidiabeticMedications":    DomainMetabolic,
	"InsulinMedications":         DomainMetabolic,

	// Hematologic domain
	"Anemia":                    DomainHematologic,
	"Coagulopathy":              DomainHematologic,
	"Thrombocytopenia":          DomainHematologic,
	"BloodTransfusionProducts":  DomainHematologic,
	"HematologyLabTests":        DomainHematologic,

	// Infectious domain
	"BacterialInfection":       DomainInfectious,
	"ViralInfection":           DomainInfectious,
	"FungalInfection":          DomainInfectious,
	"AntibioticMedications":    DomainInfectious,
	"AntiviralMedications":     DomainInfectious,
	"AntifungalMedications":    DomainInfectious,
	"InfectionMarkers":         DomainInfectious,

	// Neurological domain
	"Stroke":                  DomainNeurological,
	"Seizure":                 DomainNeurological,
	"AlteredMentalStatus":     DomainNeurological,
	"NeurologicalConditions":  DomainNeurological,
	"AnticonvulsantMedications": DomainNeurological,

	// Endocrine domain
	"ThyroidDisorders":        DomainEndocrine,
	"AdrenalDisorders":        DomainEndocrine,
	"EndocrineLabTests":       DomainEndocrine,
	"ThyroidMedications":      DomainEndocrine,

	// GI domain
	"GastrointestinalBleeding": DomainGI,
	"LiverDisease":             DomainGI,
	"PancreatitisConditions":   DomainGI,
	"GILabTests":               DomainGI,

	// Oncologic domain
	"MalignantNeoplasm":        DomainOncologic,
	"ChemotherapyMedications":  DomainOncologic,
	"OncologyBiomarkers":       DomainOncologic,
}

// GetDomainForValueSet returns the clinical domain for a value set identifier
func GetDomainForValueSet(valueSetID string) ClinicalDomain {
	if domain, ok := ValueSetDomainMapping[valueSetID]; ok {
		return domain
	}
	return DomainGeneral
}
