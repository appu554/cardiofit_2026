// Package programs defines governance programs and their associated rules.
// Programs are domain-specific collections of governance rules that are
// automatically activated based on patient context and registry memberships.
package programs

import (
	"sync"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// PROGRAM DEFINITION
// =============================================================================

// Program represents a governance program (e.g., Maternal Safety, Opioid Stewardship)
type Program struct {
	Code             string              `json:"code"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	Category         string              `json:"category"` // MATERNAL, OPIOID, ANTICOAGULATION, etc.
	Version          string              `json:"version"`
	IsActive         bool                `json:"isActive"`
	ActivationCriteria ActivationCriteria `json:"activationCriteria"`
	Rules            []Rule              `json:"rules"`
	AccountabilityChain []string         `json:"accountabilityChain"`
	EvidenceLevel    string              `json:"evidenceLevel"` // A, B, C, D, Expert
	References       []string            `json:"references,omitempty"`
}

// ActivationCriteria defines when a program should be activated
type ActivationCriteria struct {
	RequiresRegistry   []string             `json:"requiresRegistry,omitempty"`   // Patient must be in one of these registries
	RequiresDiagnosis  []string             `json:"requiresDiagnosis,omitempty"`  // Patient must have one of these diagnoses
	RequiresMedication []string             `json:"requiresMedication,omitempty"` // Order must be for one of these medications
	RequiresDrugClass  []string             `json:"requiresDrugClass,omitempty"`  // Order must be for one of these drug classes
	Demographics       *DemographicCriteria `json:"demographics,omitempty"`
	RequiresPregnancy  bool                 `json:"requiresPregnancy,omitempty"`  // Convenience flag for pregnancy requirement
	// Aliases for compatibility
	RegistryCodes    []string `json:"-"` // Alias for RequiresRegistry
	DiagnosisCodes   []string `json:"-"` // Alias for RequiresDiagnosis
	MedicationCodes  []string `json:"-"` // Alias for RequiresMedication
}

// DemographicCriteria defines demographic-based activation
type DemographicCriteria struct {
	MinAge      *int  `json:"minAge,omitempty"`
	MaxAge      *int  `json:"maxAge,omitempty"`
	Sex         string `json:"sex,omitempty"` // M, F, or empty for any
	IsPregnant  *bool  `json:"isPregnant,omitempty"`
	IsLactating *bool  `json:"isLactating,omitempty"`
}

// =============================================================================
// RULE DEFINITION
// =============================================================================

// Rule represents a governance rule within a program
type Rule struct {
	ID               string                 `json:"id"`
	Code             string                 `json:"code,omitempty"` // Alias for ID
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         types.ViolationCategory `json:"category"`
	Priority         int                    `json:"priority"` // Higher = evaluated first
	Conditions       []Condition            `json:"conditions"`
	ConditionLogic   string                 `json:"conditionLogic"` // AND, OR
	Severity         types.Severity         `json:"severity"`
	EnforcementLevel types.EnforcementLevel `json:"enforcementLevel"`
	ClinicalRisk     string                 `json:"clinicalRisk"`
	EvidenceLevel    string                 `json:"evidenceLevel"`
	References       []string               `json:"references,omitempty"`
	Recommendations  []types.Recommendation `json:"recommendations,omitempty"`
	IsActive         bool                   `json:"isActive"`
}

// GetCode returns the rule code (prefers ID)
func (r *Rule) GetCode() string {
	if r.ID != "" {
		return r.ID
	}
	return r.Code
}

// Condition represents a single rule condition
type Condition struct {
	Type       ConditionType `json:"type"`
	Field      string        `json:"field"`      // What to check (e.g., "drugClass", "age", "isPregnant")
	Operator   string        `json:"operator"`   // IN, NOT_IN, EQUALS, NOT_EQUALS, GT, LT, GTE, LTE, BETWEEN
	Value      interface{}   `json:"value"`      // Expected value(s)
	LabCode    string        `json:"labCode,omitempty"`    // For lab conditions
	LabWindow  int           `json:"labWindow,omitempty"`  // Hours to look back for labs
}

// ConditionType defines the type of condition
type ConditionType string

const (
	ConditionTypeDrugClass   ConditionType = "DRUG_CLASS"
	ConditionTypeMedication  ConditionType = "MEDICATION"
	ConditionTypeDiagnosis   ConditionType = "DIAGNOSIS"
	ConditionTypeDemographic ConditionType = "DEMOGRAPHIC"
	ConditionTypeLabValue    ConditionType = "LAB_VALUE"
	ConditionTypeVitalSign   ConditionType = "VITAL_SIGN"
	ConditionTypeAllergy     ConditionType = "ALLERGY"
	ConditionTypeRenal       ConditionType = "RENAL"
	ConditionTypeHepatic     ConditionType = "HEPATIC"
	ConditionTypePregnancy   ConditionType = "PREGNANCY"
	ConditionTypeRegistry    ConditionType = "REGISTRY"
	ConditionTypeDose        ConditionType = "DOSE"
	ConditionTypeFrequency   ConditionType = "FREQUENCY"
)

// =============================================================================
// PROGRAM STORE
// =============================================================================

// ProgramStore manages governance programs
type ProgramStore struct {
	programs map[string]*Program
	mu       sync.RWMutex
}

// NewProgramStore creates a new program store with pre-configured programs
func NewProgramStore() *ProgramStore {
	store := &ProgramStore{
		programs: make(map[string]*Program),
	}

	// Register pre-configured programs
	store.registerMaternalSafetyPrograms()
	store.registerOpioidStewardshipPrograms()
	store.registerAnticoagulationPrograms()
	store.registerRenalSafetyPrograms()

	return store
}

// GetProgram retrieves a program by code
func (s *ProgramStore) GetProgram(code string) (*Program, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.programs[code]
	return p, ok
}

// GetAllPrograms returns all programs
func (s *ProgramStore) GetAllPrograms() []*Program {
	s.mu.RLock()
	defer s.mu.RUnlock()

	programs := make([]*Program, 0, len(s.programs))
	for _, p := range s.programs {
		programs = append(programs, p)
	}
	return programs
}

// GetActivePrograms returns only active programs
func (s *ProgramStore) GetActivePrograms() []*Program {
	s.mu.RLock()
	defer s.mu.RUnlock()

	programs := make([]*Program, 0)
	for _, p := range s.programs {
		if p.IsActive {
			programs = append(programs, p)
		}
	}
	return programs
}

// RegisterProgram adds or updates a program
func (s *ProgramStore) RegisterProgram(program *Program) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.programs[program.Code] = program
}

// Count returns the number of programs
func (s *ProgramStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.programs)
}

// Get retrieves a program by code (alias for GetProgram for compatibility)
func (s *ProgramStore) Get(code string) *Program {
	p, _ := s.GetProgram(code)
	return p
}

// GetAll returns all programs as a map (for compatibility)
func (s *ProgramStore) GetAll() map[string]*Program {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*Program)
	for code, p := range s.programs {
		result[code] = p
	}
	return result
}

// =============================================================================
// MATERNAL SAFETY PROGRAMS
// =============================================================================

func (s *ProgramStore) registerMaternalSafetyPrograms() {
	// MATERNAL_MEDICATION - Core pregnancy medication safety
	s.programs["MATERNAL_MEDICATION"] = &Program{
		Code:        "MATERNAL_MEDICATION",
		Name:        "Maternal Medication Safety",
		Description: "Medication safety enforcement for pregnant patients",
		Category:    "MATERNAL",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresPregnancy: true,
			Demographics: &DemographicCriteria{
				Sex:        "F",
				IsPregnant: boolPtr(true),
			},
		},
		Rules: []Rule{
			{
				ID:          "MAT-001",
				Name:        "Teratogenic Medication Block",
				Description: "Block Category X medications in pregnancy",
				Category:    types.ViolationPregnancySafety,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{
						"METHOTREXATE", "ISOTRETINOIN", "THALIDOMIDE", "WARFARIN",
						"LEFLUNOMIDE", "MYCOPHENOLATE", "RIBAVIRIN", "MISOPROSTOL",
						"STATIN", "RETINOID", "IMMUNOMODULATOR", // Additional teratogens
						"ACE_INHIBITOR", "ARB", // ACE/ARB are absolute contraindications in 2nd/3rd trimester
					}},
					{Type: ConditionTypePregnancy, Field: "isPregnant", Operator: "EQUALS", Value: true},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Teratogenic effects with proven fetal harm - Category X",
				EvidenceLevel:    "A",
				References:       []string{"FDA Pregnancy Categories", "ACOG Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "consult", Title: "Consult Specialist", Description: "Consult with maternal-fetal medicine or rheumatology for pregnancy-safe alternatives"},
				},
				IsActive: true,
			},
			{
				ID:          "MAT-003",
				Name:        "Warfarin Absolute Block in Pregnancy",
				Description: "Block Warfarin specifically in pregnancy - causes embryopathy",
				Category:    types.ViolationPregnancySafety,
				Priority:    105, // Higher than general teratogen rule
				Conditions: []Condition{
					{Type: ConditionTypeMedication, Field: "medication", Operator: "IN", Value: []string{
						"Warfarin", "WARFARIN", "warfarin", "Coumadin", "COUMADIN",
					}},
					{Type: ConditionTypePregnancy, Field: "isPregnant", Operator: "EQUALS", Value: true},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Warfarin causes embryopathy, CNS abnormalities, and fetal bleeding - ABSOLUTE contraindication",
				EvidenceLevel:    "A",
				References:       []string{"ACOG Guidelines", "CHEST Antithrombotic Guidelines in Pregnancy"},
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Use LMWH", Description: "Low molecular weight heparin is preferred anticoagulant in pregnancy"},
					{Type: "consult", Title: "MFM Consult", Description: "Consult maternal-fetal medicine for anticoagulation management"},
				},
				IsActive: true,
			},
			{
				ID:          "MAT-002",
				Name:        "Category D Medication Warning",
				Description: "Warn for Category D medications requiring risk/benefit analysis",
				Category:    types.ViolationPregnancySafety,
				Priority:    90,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{
						"PHENYTOIN", "VALPROIC_ACID", "LITHIUM", "TETRACYCLINE", "NSAID",
						// Note: ACE_INHIBITOR and ARB moved to MAT-001 as absolute contraindications in pregnancy
					}},
					{Type: ConditionTypePregnancy, Field: "isPregnant", Operator: "EQUALS", Value: true},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "Potential fetal harm - Category D - requires documented risk/benefit analysis",
				EvidenceLevel:    "A",
				References:       []string{"FDA Pregnancy Categories", "ACOG Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Consider Alternative", Description: "Consider pregnancy-safe alternatives before proceeding"},
					{Type: "monitoring", Title: "Enhanced Monitoring", Description: "If used, implement enhanced fetal monitoring"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "OB_ATTENDING", "MFM_SPECIALIST", "DEPARTMENT_CHIEF"},
		EvidenceLevel:       "A",
		References:          []string{"ACOG Practice Bulletins", "FDA Pregnancy Labeling"},
	}

	// PREECLAMPSIA_PROTOCOL
	s.programs["PREECLAMPSIA_PROTOCOL"] = &Program{
		Code:        "PREECLAMPSIA_PROTOCOL",
		Name:        "Preeclampsia Management Protocol",
		Description: "Governance for severe preeclampsia management including magnesium sulfate",
		Category:    "MATERNAL",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDiagnosis: []string{"O14.1", "O14.2"}, // Severe preeclampsia, HELLP
			Demographics: &DemographicCriteria{
				IsPregnant: boolPtr(true),
			},
		},
		Rules: []Rule{
			{
				ID:          "PRE-001",
				Name:        "Magnesium Sulfate Requirement",
				Description: "Magnesium sulfate is required for seizure prophylaxis in severe preeclampsia",
				Category:    types.ViolationProtocolDeviation,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDiagnosis, Field: "diagnosis", Operator: "IN", Value: []string{"O14.1", "O14.2"}},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementMandatoryEscalation,
				ClinicalRisk:     "Eclamptic seizures without magnesium prophylaxis",
				EvidenceLevel:    "A",
				References:       []string{"ACOG Practice Bulletin 222", "Magpie Trial"},
				IsActive:         true,
			},
			{
				ID:          "PRE-002",
				Name:        "Blood Pressure Target",
				Description: "Severe hypertension requires urgent treatment",
				Category:    types.ViolationProtocolDeviation,
				Priority:    95,
				Conditions: []Condition{
					{Type: ConditionTypeVitalSign, Field: "systolicBp", Operator: "GTE", Value: 160},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementMandatoryEscalation,
				ClinicalRisk:     "Stroke, placental abruption risk with uncontrolled severe hypertension",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "treatment", Title: "Urgent Antihypertensive", Description: "Administer IV labetalol or hydralazine per protocol"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "OB_ATTENDING", "MFM_SPECIALIST", "DEPARTMENT_CHIEF", "CMO"},
		EvidenceLevel:       "A",
	}

	// MAGNESIUM_PROTOCOL
	s.programs["MAGNESIUM_PROTOCOL"] = &Program{
		Code:        "MAGNESIUM_PROTOCOL",
		Name:        "Magnesium Sulfate Safety Protocol",
		Description: "Safety monitoring for magnesium sulfate administration",
		Category:    "MATERNAL",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDrugClass: []string{"MAGNESIUM_SULFATE"},
		},
		Rules: []Rule{
			{
				ID:          "MAG-001",
				Name:        "Magnesium Toxicity Monitoring",
				Description: "Monitor for magnesium toxicity symptoms",
				Category:    types.ViolationMonitoringRequired,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "MAGNESIUM_SULFATE"},
					{Type: ConditionTypeLabValue, Field: "magnesium", Operator: "GTE", Value: 9.0, LabCode: "19123-9", LabWindow: 4},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Respiratory depression, cardiac arrest at toxic levels",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "treatment", Title: "Stop Infusion", Description: "Immediately stop magnesium infusion"},
					{Type: "treatment", Title: "Calcium Gluconate", Description: "Administer calcium gluconate 1g IV as antidote"},
				},
				IsActive: true,
			},
			{
				ID:          "MAG-002",
				Name:        "Renal Function Check",
				Description: "Check renal function before magnesium administration",
				Category:    types.ViolationLabRequired,
				Priority:    90,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "MAGNESIUM_SULFATE"},
					{Type: ConditionTypeRenal, Field: "egfr", Operator: "LT", Value: 30},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "Magnesium accumulation in renal impairment",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "dosing", Title: "Dose Reduction", Description: "Reduce loading and maintenance dose by 50%"},
					{Type: "monitoring", Title: "Frequent Levels", Description: "Check magnesium levels every 2 hours"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "NURSE", "PHARMACIST", "OB_ATTENDING"},
		EvidenceLevel:       "A",
	}

	// GESTATIONAL_DM
	s.programs["GESTATIONAL_DM"] = &Program{
		Code:        "GESTATIONAL_DM",
		Name:        "Gestational Diabetes Management",
		Description: "Glucose management and insulin dosing in gestational diabetes",
		Category:    "MATERNAL",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDiagnosis: []string{"O24.4"}, // Gestational diabetes
		},
		Rules: []Rule{
			{
				ID:          "GDM-001",
				Name:        "Oral Hypoglycemic Block in Pregnancy",
				Description: "Most oral hypoglycemics are contraindicated in pregnancy",
				Category:    types.ViolationPregnancySafety,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{
						"SULFONYLUREA", "THIAZOLIDINEDIONE", "SGLT2_INHIBITOR", "DPP4_INHIBITOR",
					}},
					{Type: ConditionTypePregnancy, Field: "isPregnant", Operator: "EQUALS", Value: true},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "Limited safety data in pregnancy, potential fetal harm",
				EvidenceLevel:    "B",
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Use Insulin", Description: "Insulin is first-line for gestational diabetes requiring medication"},
					{Type: "alternative", Title: "Consider Metformin", Description: "Metformin may be considered with appropriate counseling"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "ENDOCRINOLOGIST", "OB_ATTENDING", "MFM_SPECIALIST"},
		EvidenceLevel:       "B",
	}
}

// =============================================================================
// OPIOID STEWARDSHIP PROGRAMS
// =============================================================================

func (s *ProgramStore) registerOpioidStewardshipPrograms() {
	// OPIOID_STEWARDSHIP - Overall opioid safety
	s.programs["OPIOID_STEWARDSHIP"] = &Program{
		Code:        "OPIOID_STEWARDSHIP",
		Name:        "Opioid Stewardship Program",
		Description: "Comprehensive opioid safety including MME limits, PDMP, and interaction checks",
		Category:    "OPIOID",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDrugClass: []string{"OPIOID", "OPIOID_AGONIST"},
		},
		Rules: []Rule{
			{
				ID:          "OPI-001",
				Name:        "High MME Warning",
				Description: "Morphine Milligram Equivalent exceeds 50 MME/day",
				Category:    types.ViolationDoseExceeded,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "OPIOID"},
					{Type: ConditionTypeDose, Field: "mme", Operator: "GTE", Value: 50},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityHigh,
				EnforcementLevel: types.EnforcementWarnAcknowledge,
				ClinicalRisk:     "Increased risk of overdose and death at ≥50 MME/day",
				EvidenceLevel:    "A",
				References:       []string{"CDC Opioid Prescribing Guidelines 2022"},
				IsActive:         true,
			},
			{
				ID:          "OPI-002",
				Name:        "Very High MME Block",
				Description: "Morphine Milligram Equivalent exceeds 90 MME/day - hard block",
				Category:    types.ViolationDoseExceeded,
				Priority:    110,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "OPIOID"},
					{Type: ConditionTypeDose, Field: "mme", Operator: "GTE", Value: 90},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "High risk of fatal overdose at ≥90 MME/day without tolerance",
				EvidenceLevel:    "A",
				References:       []string{"CDC Opioid Prescribing Guidelines 2022"},
				Recommendations: []types.Recommendation{
					{Type: "consult", Title: "Pain Specialist Consult", Description: "Consult pain medicine specialist before exceeding 90 MME/day"},
					{Type: "treatment", Title: "Consider Rotation", Description: "Consider opioid rotation or adjuvant therapy"},
				},
				IsActive: true,
			},
			{
				ID:          "OPI-003",
				Name:        "Opioid-Benzodiazepine Interaction",
				Description: "Concurrent opioid and benzodiazepine prescribing",
				Category:    types.ViolationDrugInteraction,
				Priority:    95,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "OPIOID"},
					{Type: ConditionTypeMedication, Field: "currentMedications", Operator: "HAS_CLASS", Value: "BENZODIAZEPINE"},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "FDA Black Box Warning: Concurrent use increases risk of respiratory depression and death",
				EvidenceLevel:    "A",
				References:       []string{"FDA Drug Safety Communication 2016"},
				IsActive:         true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "PAIN_SPECIALIST", "DEPARTMENT_CHIEF", "CMO"},
		EvidenceLevel:       "A",
	}

	// OPIOID_NAIVE - Opioid-naive patient safety
	s.programs["OPIOID_NAIVE"] = &Program{
		Code:        "OPIOID_NAIVE",
		Name:        "Opioid-Naive Patient Safety",
		Description: "Safety measures for patients new to opioid therapy",
		Category:    "OPIOID",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDrugClass: []string{"OPIOID", "OPIOID_AGONIST", "OPIOID_ER", "OPIOID_LA", "OPIOID_EXTENDED_RELEASE", "OPIOID_HIGH_POTENCY"},
			RequiresRegistry:  []string{"OPIOID_NAIVE"},
		},
		Rules: []Rule{
			{
				ID:          "ONV-001",
				Name:        "Extended-Release Opioid Block in Naive",
				Description: "ER/LA opioids blocked in opioid-naive patients",
				Category:    types.ViolationContraindication,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeMedication, Field: "formulation", Operator: "IN", Value: []string{"ER", "LA", "EXTENDED_RELEASE"}},
					{Type: ConditionTypeRegistry, Field: "registry", Operator: "IN", Value: []string{"OPIOID_NAIVE"}},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "FDA Black Box Warning: ER/LA opioids not for opioid-naive patients - fatal overdose risk",
				EvidenceLevel:    "A",
				References:       []string{"FDA Opioid REMS"},
				IsActive:         true,
			},
			{
				ID:          "ONV-003",
				Name:        "Extended-Release Drug Class Block in Naive",
				Description: "OPIOID_ER drug class blocked in opioid-naive patients",
				Category:    types.ViolationContraindication,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"OPIOID_ER", "OPIOID_LA", "OPIOID_EXTENDED_RELEASE"}},
					{Type: ConditionTypeRegistry, Field: "registry", Operator: "IN", Value: []string{"OPIOID_NAIVE"}},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "FDA Black Box Warning: Extended-release opioids not for opioid-naive patients - fatal overdose risk",
				EvidenceLevel:    "A",
				References:       []string{"FDA Opioid REMS", "OxyContin REMS"},
				IsActive:         true,
			},
			{
				ID:          "ONV-004",
				Name:        "High Dose Opioid in Naive Patient",
				Description: "High-dose opioids require specialist approval in naive patients",
				Category:    types.ViolationDoseExceeded,
				Priority:    95,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"OPIOID", "OPIOID_AGONIST", "OPIOID_HIGH_POTENCY"}},
					{Type: ConditionTypeRegistry, Field: "registry", Operator: "IN", Value: []string{"OPIOID_NAIVE"}},
					{Type: ConditionTypeDose, Field: "dose", Operator: "GTE", Value: 10.0},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "High-dose opioids in naive patients require specialist approval - overdose risk",
				EvidenceLevel:    "A",
				References:       []string{"CDC Opioid Prescribing Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "consult", Title: "Pain Specialist", Description: "Consult pain specialist before high-dose opioid in naive patient"},
				},
				IsActive: true,
			},
			{
				ID:          "ONV-002",
				Name:        "Starting Dose Limit",
				Description: "Starting dose should not exceed 30 MME/day in opioid-naive patients",
				Category:    types.ViolationDoseExceeded,
				Priority:    90,
				Conditions: []Condition{
					{Type: ConditionTypeDose, Field: "mme", Operator: "GT", Value: 30},
					{Type: ConditionTypeRegistry, Field: "registry", Operator: "IN", Value: []string{"OPIOID_NAIVE"}},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityHigh,
				EnforcementLevel: types.EnforcementWarnAcknowledge,
				ClinicalRisk:     "Increased overdose risk with high starting doses in naive patients",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "dosing", Title: "Start Low", Description: "Start at lowest effective dose and titrate slowly"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "PAIN_SPECIALIST"},
		EvidenceLevel:       "A",
	}

	// OPIOID_MAT - Medication-Assisted Treatment
	s.programs["OPIOID_MAT"] = &Program{
		Code:        "OPIOID_MAT",
		Name:        "Medication-Assisted Treatment Protocol",
		Description: "Buprenorphine and methadone treatment protocols",
		Category:    "OPIOID",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDrugClass: []string{"BUPRENORPHINE", "METHADONE"},
			RequiresRegistry:  []string{"MAT_PROGRAM", "OUD_TREATMENT"},
		},
		Rules: []Rule{
			{
				ID:          "MAT-001",
				Name:        "Methadone QTc Monitoring",
				Description: "ECG required for methadone doses >100mg/day",
				Category:    types.ViolationMonitoringRequired,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeMedication, Field: "medication", Operator: "EQUALS", Value: "METHADONE"},
					{Type: ConditionTypeDose, Field: "dose", Operator: "GT", Value: 100},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "QTc prolongation and Torsades de Pointes risk at high doses",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "monitoring", Title: "ECG Required", Description: "Obtain baseline ECG and repeat with dose changes"},
					{Type: "monitoring", Title: "Check QTc", Description: "Hold if QTc >500ms or increases >60ms from baseline"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "ADDICTION_SPECIALIST", "DEPARTMENT_CHIEF"},
		EvidenceLevel:       "A",
	}
}

// =============================================================================
// ANTICOAGULATION PROGRAMS
// =============================================================================

func (s *ProgramStore) registerAnticoagulationPrograms() {
	// ANTICOAGULATION - General anticoagulant safety
	s.programs["ANTICOAGULATION"] = &Program{
		Code:        "ANTICOAGULATION",
		Name:        "Anticoagulation Safety Program",
		Description: "General anticoagulant safety including duplication and renal dosing",
		Category:    "ANTICOAGULATION",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDrugClass: []string{"ANTICOAGULANT", "DOAC", "WARFARIN", "HEPARIN", "LMWH"},
		},
		Rules: []Rule{
			{
				ID:          "ACG-001",
				Name:        "Duplicate Anticoagulation Block",
				Description: "Block duplicate anticoagulant therapy",
				Category:    types.ViolationDuplicateTherapy,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"ANTICOAGULANT", "DOAC", "WARFARIN"}},
					{Type: ConditionTypeMedication, Field: "currentMedications", Operator: "HAS_CLASS", Value: "ANTICOAGULANT"},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "Major bleeding risk with duplicate anticoagulation",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Discontinue Current", Description: "Discontinue current anticoagulant before starting new one"},
					{Type: "monitoring", Title: "Bridging Protocol", Description: "If bridging required, follow institutional protocol"},
				},
				IsActive: true,
			},
			{
				ID:          "ACG-002",
				Name:        "Antiplatelet Combination Warning",
				Description: "Warn for anticoagulant + antiplatelet combination",
				Category:    types.ViolationDrugInteraction,
				Priority:    90,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"ANTICOAGULANT", "DOAC", "WARFARIN"}},
					{Type: ConditionTypeMedication, Field: "currentMedications", Operator: "HAS_CLASS", Value: "ANTIPLATELET"},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityHigh,
				EnforcementLevel: types.EnforcementWarnAcknowledge,
				ClinicalRisk:     "Increased bleeding risk with combination therapy",
				EvidenceLevel:    "A",
				IsActive:         true,
			},
			// =============================================================================
			// BLEEDING CONTRAINDICATION RULES - Critical Safety
			// =============================================================================
			{
				ID:          "ACG-003",
				Name:        "Severe Thrombocytopenia Contraindication",
				Description: "Block anticoagulation with platelets < 50k",
				Category:    types.ViolationContraindication,
				Priority:    110, // Highest priority - safety critical
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"ANTICOAGULANT", "DOAC", "WARFARIN", "HEPARIN", "LMWH"}},
					{Type: ConditionTypeLabValue, Field: "platelets", Operator: "LT", Value: 50.0, LabCode: "PLT", LabWindow: 72},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Critical bleeding risk with severe thrombocytopenia - spontaneous hemorrhage likely",
				EvidenceLevel:    "A",
				References:       []string{"CHEST Antithrombotic Guidelines", "ISTH Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Hold Anticoagulation", Description: "Hold anticoagulation until platelets > 50k"},
					{Type: "consult", Title: "Hematology Consult", Description: "Consult hematology for management"},
				},
				IsActive: true,
			},
			{
				ID:          "ACG-004",
				Name:        "Active GI Hemorrhage Contraindication",
				Description: "Block anticoagulation with active GI bleeding",
				Category:    types.ViolationContraindication,
				Priority:    110, // Highest priority - safety critical
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"ANTICOAGULANT", "DOAC", "WARFARIN", "HEPARIN", "LMWH"}},
					{Type: ConditionTypeDiagnosis, Field: "diagnosis", Operator: "IN", Value: []string{"K92.2", "K92.0", "K92.1"}}, // GI hemorrhage, hematemesis, melena
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Active GI bleeding is absolute contraindication - will worsen hemorrhage",
				EvidenceLevel:    "A",
				References:       []string{"ACG GI Bleeding Guidelines", "CHEST Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "treatment", Title: "Stop All Anticoagulation", Description: "Discontinue anticoagulation immediately"},
					{Type: "treatment", Title: "Reversal Agent", Description: "Consider reversal agent if life-threatening bleed"},
				},
				IsActive: true,
			},
			{
				ID:          "ACG-005",
				Name:        "Intracranial Hemorrhage Contraindication",
				Description: "Block anticoagulation with ICH - absolute contraindication",
				Category:    types.ViolationContraindication,
				Priority:    120, // Highest priority - absolute contraindication
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"ANTICOAGULANT", "DOAC", "WARFARIN", "HEPARIN", "LMWH"}},
					{Type: ConditionTypeDiagnosis, Field: "diagnosis", Operator: "IN", Value: []string{"I61.9", "I61.0", "I61.1", "I61.2", "I62.9", "S06.6"}}, // ICH codes
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Intracranial hemorrhage is ABSOLUTE contraindication - fatal hematoma expansion risk",
				EvidenceLevel:    "A",
				References:       []string{"AHA/ASA ICH Guidelines", "Neurocritical Care Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "treatment", Title: "Immediate Reversal", Description: "Reverse any existing anticoagulation immediately"},
					{Type: "consult", Title: "Neurosurgery Consult", Description: "Emergent neurosurgery consultation"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "ANTICOAGULATION_CLINIC", "HEMATOLOGIST"},
		EvidenceLevel:       "A",
	}

	// WARFARIN_MANAGEMENT
	s.programs["WARFARIN_MANAGEMENT"] = &Program{
		Code:        "WARFARIN_MANAGEMENT",
		Name:        "Warfarin Management Protocol",
		Description: "INR monitoring and drug interaction management for warfarin",
		Category:    "ANTICOAGULATION",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDrugClass: []string{"WARFARIN"},
		},
		Rules: []Rule{
			{
				ID:          "WAR-001",
				Name:        "INR Monitoring Required",
				Description: "INR must be checked within past 7 days for warfarin therapy",
				Category:    types.ViolationLabRequired,
				Priority:    90,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "WARFARIN"},
					{Type: ConditionTypeLabValue, Field: "inr", Operator: "MISSING", Value: nil, LabCode: "5902-2", LabWindow: 168}, // 7 days in hours
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityHigh,
				EnforcementLevel: types.EnforcementWarnAcknowledge,
				ClinicalRisk:     "Supratherapeutic INR increases bleeding risk; subtherapeutic increases thrombosis risk",
				EvidenceLevel:    "A",
				IsActive:         true,
			},
			{
				ID:          "WAR-002",
				Name:        "High INR Alert",
				Description: "INR >4.0 requires immediate attention",
				Category:    types.ViolationMonitoringRequired,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "WARFARIN"},
					{Type: ConditionTypeLabValue, Field: "inr", Operator: "GT", Value: 4.0, LabCode: "5902-2", LabWindow: 24},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementMandatoryEscalation,
				ClinicalRisk:     "High bleeding risk with INR >4.0",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "treatment", Title: "Hold Warfarin", Description: "Consider holding warfarin until INR <3.0"},
					{Type: "treatment", Title: "Vitamin K", Description: "Consider vitamin K if INR >10 or bleeding"},
				},
				IsActive: true,
			},
			{
				ID:          "WAR-003",
				Name:        "Major Drug Interaction",
				Description: "Major warfarin drug interactions requiring dose adjustment",
				Category:    types.ViolationDrugInteraction,
				Priority:    95,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "WARFARIN"},
					{Type: ConditionTypeMedication, Field: "interactingDrugs", Operator: "HAS_ANY", Value: []string{
						"AMIODARONE", "FLUCONAZOLE", "METRONIDAZOLE", "COTRIMOXAZOLE",
						"CIPROFLOXACIN", "RIFAMPIN", "CARBAMAZEPINE", "PHENYTOIN",
					}},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityHigh,
				EnforcementLevel: types.EnforcementWarnAcknowledge,
				ClinicalRisk:     "Drug interaction may significantly alter warfarin effect",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "monitoring", Title: "More Frequent INR", Description: "Check INR within 3-5 days of starting/stopping interacting drug"},
					{Type: "dosing", Title: "Dose Adjustment", Description: "Consider empiric warfarin dose adjustment"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "ANTICOAGULATION_CLINIC"},
		EvidenceLevel:       "A",
	}

	// DOAC_MANAGEMENT
	s.programs["DOAC_MANAGEMENT"] = &Program{
		Code:        "DOAC_MANAGEMENT",
		Name:        "DOAC Management Protocol",
		Description: "Renal-based dosing and reversal protocols for DOACs",
		Category:    "ANTICOAGULATION",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			RequiresDrugClass: []string{"DOAC", "APIXABAN", "RIVAROXABAN", "DABIGATRAN", "EDOXABAN"},
		},
		Rules: []Rule{
			{
				ID:          "DOAC-001",
				Name:        "Severe Renal Impairment - Dabigatran Contraindicated",
				Description: "Dabigatran contraindicated with CrCl <30 mL/min",
				Category:    types.ViolationRenalDosing,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeMedication, Field: "medication", Operator: "EQUALS", Value: "DABIGATRAN"},
					{Type: ConditionTypeRenal, Field: "egfr", Operator: "LT", Value: 30},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Dabigatran accumulation leads to major bleeding - contraindicated in severe renal impairment",
				EvidenceLevel:    "A",
				References:       []string{"Pradaxa Prescribing Information"},
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Use Alternative DOAC", Description: "Consider apixaban with appropriate renal dosing"},
					{Type: "alternative", Title: "Consider Warfarin", Description: "Warfarin may be preferred in severe renal impairment"},
				},
				IsActive: true,
			},
			{
				ID:          "DOAC-002",
				Name:        "Moderate Renal Impairment - Dose Adjustment Required",
				Description: "DOAC dose adjustment required for CrCl 30-50 mL/min",
				Category:    types.ViolationRenalDosing,
				Priority:    90,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "DOAC"},
					{Type: ConditionTypeRenal, Field: "egfr", Operator: "BETWEEN", Value: []float64{30, 50}},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityHigh,
				EnforcementLevel: types.EnforcementWarnAcknowledge,
				ClinicalRisk:     "Dose adjustment required to prevent drug accumulation and bleeding",
				EvidenceLevel:    "A",
				Recommendations: []types.Recommendation{
					{Type: "dosing", Title: "Review Renal Dosing", Description: "Verify dose is appropriate for renal function"},
				},
				IsActive: true,
			},
			{
				ID:          "DOAC-003",
				Name:        "ESRD on Dialysis",
				Description: "Limited DOAC data in dialysis patients",
				Category:    types.ViolationRenalDosing,
				Priority:    95,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "EQUALS", Value: "DOAC"},
					{Type: ConditionTypeRenal, Field: "onDialysis", Operator: "EQUALS", Value: true},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "Limited safety data for DOACs in dialysis - consider warfarin",
				EvidenceLevel:    "C",
				Recommendations: []types.Recommendation{
					{Type: "consult", Title: "Nephrology Consult", Description: "Consult nephrology for anticoagulation in dialysis"},
					{Type: "alternative", Title: "Consider Warfarin", Description: "Warfarin traditionally preferred in dialysis patients"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "NEPHROLOGIST", "HEMATOLOGIST"},
		EvidenceLevel:       "A",
	}
}

// =============================================================================
// RENAL SAFETY PROGRAMS
// =============================================================================

func (s *ProgramStore) registerRenalSafetyPrograms() {
	// RENAL_SAFETY - Nephrotoxin avoidance in AKI/CKD
	s.programs["RENAL_SAFETY"] = &Program{
		Code:        "RENAL_SAFETY",
		Name:        "Renal Safety Program",
		Description: "Nephrotoxin avoidance and renal dosing safety in AKI and CKD",
		Category:    "RENAL",
		Version:     "1.0.0",
		IsActive:    true,
		ActivationCriteria: ActivationCriteria{
			// Activated for any patient with renal impairment
			// Program rules will check specific eGFR thresholds
		},
		Rules: []Rule{
			{
				ID:          "REN-001",
				Name:        "NSAID Contraindication in Severe Renal Impairment",
				Description: "Block NSAIDs when eGFR < 30 mL/min",
				Category:    types.ViolationContraindication,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"NSAID"}},
					{Type: ConditionTypeRenal, Field: "egfr", Operator: "LT", Value: 30.0},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "NSAIDs cause afferent arteriolar constriction - will worsen AKI and may cause permanent renal damage",
				EvidenceLevel:    "A",
				References:       []string{"KDIGO AKI Guidelines", "FDA NSAID Renal Warnings"},
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Use Acetaminophen", Description: "Acetaminophen is safe alternative for pain in renal impairment"},
					{Type: "monitoring", Title: "Monitor Renal Function", Description: "If NSAID absolutely necessary, monitor creatinine daily"},
				},
				IsActive: true,
			},
			{
				ID:          "REN-002",
				Name:        "Metformin Contraindication in Severe Renal Impairment",
				Description: "Block Metformin when eGFR < 30 mL/min",
				Category:    types.ViolationContraindication,
				Priority:    100,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"BIGUANIDE"}},
					{Type: ConditionTypeRenal, Field: "egfr", Operator: "LT", Value: 30.0},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityFatal,
				EnforcementLevel: types.EnforcementHardBlock,
				ClinicalRisk:     "Metformin accumulation in renal failure causes life-threatening lactic acidosis",
				EvidenceLevel:    "A",
				References:       []string{"FDA Metformin Label", "ADA Diabetes Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Use Insulin", Description: "Insulin is safe for glycemic control in severe renal impairment"},
					{Type: "alternative", Title: "Consider SGLT2i", Description: "Some SGLT2 inhibitors may be used with caution if eGFR > 20"},
				},
				IsActive: true,
			},
			{
				ID:          "REN-003",
				Name:        "IV Contrast Contraindication in Severe AKI",
				Description: "Block IV contrast in severe renal impairment",
				Category:    types.ViolationContraindication,
				Priority:    90,
				Conditions: []Condition{
					{Type: ConditionTypeDrugClass, Field: "drugClass", Operator: "IN", Value: []string{"IV_CONTRAST"}},
					{Type: ConditionTypeRenal, Field: "egfr", Operator: "LT", Value: 30.0},
				},
				ConditionLogic:   "AND",
				Severity:         types.SeverityCritical,
				EnforcementLevel: types.EnforcementHardBlockWithOverride,
				ClinicalRisk:     "Contrast-induced nephropathy risk significantly elevated in severe renal impairment",
				EvidenceLevel:    "A",
				References:       []string{"ACR Contrast Guidelines", "KDIGO Guidelines"},
				Recommendations: []types.Recommendation{
					{Type: "alternative", Title: "Non-contrast Imaging", Description: "Consider non-contrast alternatives (ultrasound, MRI without gadolinium)"},
					{Type: "treatment", Title: "Pre-hydration", Description: "If contrast essential, ensure aggressive IV hydration pre/post"},
				},
				IsActive: true,
			},
		},
		AccountabilityChain: []string{"PRESCRIBER", "PHARMACIST", "NEPHROLOGIST"},
		EvidenceLevel:       "A",
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
