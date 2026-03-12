package orb

// ORBRuleSet represents the complete set of ORB rules
type ORBRuleSet struct {
	Metadata struct {
		Version          string `yaml:"version"`
		LastUpdated      string `yaml:"last_updated"`
		Description      string `yaml:"description"`
		Author           string `yaml:"author"`
		ValidationStatus string `yaml:"validation_status"`
	} `yaml:"metadata"`
	
	Rules []ORBRule `yaml:"rules"`
	
	RuleEvaluation struct {
		EvaluationOrder     string `yaml:"evaluation_order"`
		StopOnFirstMatch    bool   `yaml:"stop_on_first_match"`
		EvaluationTimeoutMs int    `yaml:"evaluation_timeout_ms"`
	} `yaml:"rule_evaluation"`
}

// ORBRule represents a single ORB routing rule (Production Format)
type ORBRule struct {
	ID       string `yaml:"id"`
	Priority int    `yaml:"priority"`

	// Production format uses flexible conditions
	Conditions struct {
		AllOf []RuleCondition `yaml:"all_of,omitempty"`
		AnyOf []RuleCondition `yaml:"any_of,omitempty"`
	} `yaml:"conditions"`

	// Production format uses action.generate_manifest or action.type
	Action struct {
		// For recipe selection rules
		GenerateManifest struct {
			RecipeID     string `yaml:"recipe_id"`
			Variant      string `yaml:"variant,omitempty"`
			DataManifest struct {
				Required []string `yaml:"required"`
			} `yaml:"data_manifest"`
			// NEW: Knowledge Manifest for KB optimization
			KnowledgeManifest struct {
				RequiredKBs []string `yaml:"required_kbs"`
			} `yaml:"knowledge_manifest"`
		} `yaml:"generate_manifest"`

		// For safety/validation rules
		Type     string `yaml:"type,omitempty"`
		Severity string `yaml:"severity,omitempty"`
		Message  string `yaml:"message,omitempty"`
	} `yaml:"action"`

	// Legacy fields for backward compatibility
	MedicationCode string `yaml:"medication_code,omitempty"`
	RuleName       string `yaml:"rule_name,omitempty"`

	// Legacy intent manifest (for backward compatibility)
	IntentManifest struct {
		RecipeID                 string   `yaml:"recipe_id"`
		Variant                  string   `yaml:"variant,omitempty"`
		DataRequirements         []string `yaml:"data_requirements"`
		Priority                 string   `yaml:"priority"`
		Rationale                string   `yaml:"rationale"`
		EstimatedExecutionTimeMs int      `yaml:"estimated_execution_time_ms"`
	} `yaml:"intent_manifest,omitempty"`
}

// RuleCondition represents a single condition in the rule engine
type RuleCondition struct {
	Fact     string      `yaml:"fact"`
	Operator string      `yaml:"operator"`
	Value    interface{} `yaml:"value"`
}

// MedicationKnowledgeCore represents TIER 1 core clinical knowledge
type MedicationKnowledgeCore struct {
	DrugEncyclopedia  *DrugEncyclopedia
	DrugInteractions  *DrugInteractions
	Contraindications *Contraindications
}

// DrugEncyclopedia represents the drug encyclopedia JSON structure
type DrugEncyclopedia struct {
	Metadata struct {
		Version     string `json:"version"`
		LastUpdated string `json:"last_updated"`
		Source      string `json:"source"`
		Description string `json:"description"`
	} `json:"metadata"`
	
	Medications map[string]Medication `json:"medications"`
	
	TherapeuticClasses map[string]TherapeuticClass `json:"therapeutic_classes"`
}

// Medication represents a single medication in the encyclopedia
type Medication struct {
	RxNormCode      string   `json:"rxnorm_code"`
	GenericName     string   `json:"generic_name"`
	BrandNames      []string `json:"brand_names"`
	TherapeuticClass string  `json:"therapeutic_class"`
	Mechanism       string   `json:"mechanism"`
	Indications     []string `json:"indications"`
	
	Pharmacokinetics struct {
		HalfLifeHours            float64 `json:"half_life_hours"`
		ProteinBindingPercent    int     `json:"protein_binding_percent"`
		RenalEliminationPercent  int     `json:"renal_elimination_percent"`
		HepaticMetabolismPercent int     `json:"hepatic_metabolism_percent"`
		VolumeDistributionLPerKg float64 `json:"volume_distribution_l_per_kg"`
	} `json:"pharmacokinetics"`
	
	Dosing struct {
		StandardDoseMgPerKg float64 `json:"standard_dose_mg_per_kg,omitempty"`
		InitialDoseMg       float64 `json:"initial_dose_mg,omitempty"`
		MaxDoseMg           float64 `json:"max_dose_mg"`
		MaxDailyDoseMg      float64 `json:"max_daily_dose_mg,omitempty"`
		FrequencyHours      int     `json:"frequency_hours"`
		Route               string  `json:"route"`
	} `json:"dosing"`
	
	SafetyProfile struct {
		Nephrotoxic        bool     `json:"nephrotoxic,omitempty"`
		Ototoxic           bool     `json:"ototoxic,omitempty"`
		Hepatotoxic        bool     `json:"hepatotoxic,omitempty"`
		BleedingRisk       bool     `json:"bleeding_risk,omitempty"`
		DrugInteractions   string   `json:"drug_interactions,omitempty"`
		RequiresMonitoring bool     `json:"requires_monitoring"`
		BlackBoxWarnings   []string `json:"black_box_warnings"`
		PregnancyCategory  string   `json:"pregnancy_category"`
	} `json:"safety_profile"`
	
	MonitoringRequirements []string `json:"monitoring_requirements"`
	Contraindications      []string `json:"contraindications"`
}

// TherapeuticClass represents a therapeutic classification
type TherapeuticClass struct {
	Description      string   `json:"description"`
	CommonSideEffects []string `json:"common_side_effects"`
	Monitoring       string   `json:"monitoring"`
}

// DrugInteractions represents the drug interactions database
type DrugInteractions struct {
	Metadata struct {
		Version     string `json:"version"`
		LastUpdated string `json:"last_updated"`
		Source      string `json:"source"`
		Description string `json:"description"`
	} `json:"metadata"`
	
	Interactions []DrugInteraction `json:"interactions"`
	
	SeverityDefinitions map[string]SeverityDefinition `json:"severity_definitions"`
	InteractionTypes    map[string]string             `json:"interaction_types"`
}

// DrugInteraction represents a single drug-drug interaction
type DrugInteraction struct {
	InteractionID        string   `json:"interaction_id"`
	Drug1                string   `json:"drug1"`
	Drug2                string   `json:"drug2"`
	Severity             string   `json:"severity"`
	Type                 string   `json:"type"`
	Mechanism            string   `json:"mechanism"`
	Description          string   `json:"description"`
	ClinicalSignificance string   `json:"clinical_significance"`
	Management           []string `json:"management"`
	EvidenceLevel        string   `json:"evidence_level"`
	References           []string `json:"references"`
}

// SeverityDefinition defines interaction severity levels
type SeverityDefinition struct {
	Description string `json:"description"`
	Action      string `json:"action"`
}

// Contraindications represents the contraindications database
type Contraindications struct {
	Metadata struct {
		Version     string `json:"version"`
		LastUpdated string `json:"last_updated"`
		Source      string `json:"source"`
		Description string `json:"description"`
	} `json:"metadata"`
	
	Contraindications map[string]MedicationContraindications `json:"contraindications"`
	
	ConditionDefinitions map[string]ConditionDefinition `json:"condition_definitions"`
	SeverityDefinitions  map[string]string              `json:"severity_definitions"`
}

// MedicationContraindications represents contraindications for a medication
type MedicationContraindications struct {
	Absolute []Contraindication `json:"absolute"`
	Relative []Contraindication `json:"relative"`
}

// Contraindication represents a single contraindication
type Contraindication struct {
	Condition    string   `json:"condition"`
	Severity     string   `json:"severity"`
	Description  string   `json:"description"`
	Rationale    string   `json:"rationale"`
	Management   string   `json:"management,omitempty"`
	Alternatives []string `json:"alternatives,omitempty"`
}

// ConditionDefinition defines a medical condition
type ConditionDefinition struct {
	ICD10              string   `json:"icd10"`
	Description        string   `json:"description"`
	ScreeningQuestions []string `json:"screening_questions"`
}

// ContextServiceRecipeBook represents TIER 2 context recipes
type ContextServiceRecipeBook struct {
	Recipes map[string]*ContextRecipe
}

// ContextRecipe represents a context data recipe
type ContextRecipe struct {
	RecipeID        string `yaml:"recipe_id"`
	Version         string `yaml:"version"`
	LastUpdated     string `yaml:"last_updated"`
	MedicationCode  string `yaml:"medication_code,omitempty"`
	Description     string `yaml:"description"`
	
	BaseRequirements []ContextRequirement `yaml:"base_requirements"`
	
	RecipeSpecificRequirements map[string]struct {
		AdditionalRequirements []ContextRequirement `yaml:"additional_requirements"`
	} `yaml:"recipe_specific_requirements,omitempty"`
	
	MedicationSpecificRequirements map[string]struct {
		AdditionalRequirements []ContextRequirement `yaml:"additional_requirements"`
	} `yaml:"medication_specific_requirements,omitempty"`
}

// ContextRequirement represents a single data requirement
type ContextRequirement struct {
	Field       string `yaml:"field"`
	Source      string `yaml:"source"`
	Endpoint    string `yaml:"endpoint"`
	Required    bool   `yaml:"required"`
	MaxAgeHours int    `yaml:"max_age_hours,omitempty"`
	MaxAgeDays  int    `yaml:"max_age_days,omitempty"`
	Units       string `yaml:"units,omitempty"`
	Values      []string `yaml:"values,omitempty"`
	Includes    []string `yaml:"includes,omitempty"`
	Fallback    string `yaml:"fallback,omitempty"`
}

// Clinical Recipe Book structures (TIER 1)
type ClinicalRecipeBook struct {
	Metadata ClinicalRecipeMetadata       `yaml:"metadata"`
	Recipes  map[string]*ClinicalRecipe   `yaml:"recipes"`
}

type ClinicalRecipeMetadata struct {
	Version     string `yaml:"version"`
	LastUpdated string `yaml:"last_updated"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
}

type ClinicalRecipe struct {
	RecipeID        string                    `yaml:"recipe_id"`
	Version         string                    `yaml:"version"`
	Name            string                    `yaml:"name"`
	Description     string                    `yaml:"description"`
	MedicationCode  string                    `yaml:"medication_code"`
	Indication      string                    `yaml:"indication"`
	Algorithm       ClinicalAlgorithm         `yaml:"algorithm"`
	SafetyChecks    []SafetyCheck             `yaml:"safety_checks"`
	Monitoring      ClinicalMonitoring        `yaml:"monitoring"`
	Documentation   ClinicalDocumentation     `yaml:"documentation"`
}

type ClinicalAlgorithm struct {
	Type        string                 `yaml:"type"`
	Steps       []AlgorithmStep        `yaml:"steps"`
	Parameters  map[string]interface{} `yaml:"parameters"`
	Validation  []ValidationRule       `yaml:"validation"`
}

type AlgorithmStep struct {
	StepID      string                 `yaml:"step_id"`
	Description string                 `yaml:"description"`
	Action      string                 `yaml:"action"`
	Parameters  map[string]interface{} `yaml:"parameters"`
	Conditions  []StepCondition        `yaml:"conditions"`
}

type StepCondition struct {
	Field    string      `yaml:"field"`
	Operator string      `yaml:"operator"`
	Value    interface{} `yaml:"value"`
}

type ValidationRule struct {
	RuleID      string      `yaml:"rule_id"`
	Description string      `yaml:"description"`
	Field       string      `yaml:"field"`
	Operator    string      `yaml:"operator"`
	Value       interface{} `yaml:"value"`
	ErrorMsg    string      `yaml:"error_message"`
}

type SafetyCheck struct {
	CheckID     string `yaml:"check_id"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Severity    string `yaml:"severity"`
	Action      string `yaml:"action"`
}

type ClinicalMonitoring struct {
	Required    bool                   `yaml:"required"`
	Frequency   string                 `yaml:"frequency"`
	Parameters  []MonitoringParameter  `yaml:"parameters"`
	Alerts      []MonitoringAlert      `yaml:"alerts"`
}

type MonitoringParameter struct {
	Parameter   string      `yaml:"parameter"`
	Type        string      `yaml:"type"`
	NormalRange interface{} `yaml:"normal_range"`
	CriticalRange interface{} `yaml:"critical_range"`
}

type MonitoringAlert struct {
	AlertID     string `yaml:"alert_id"`
	Condition   string `yaml:"condition"`
	Severity    string `yaml:"severity"`
	Action      string `yaml:"action"`
	Message     string `yaml:"message"`
}

type ClinicalDocumentation struct {
	Guidelines  []string `yaml:"guidelines"`
	References  []string `yaml:"references"`
	Evidence    string   `yaml:"evidence"`
	LastReview  string   `yaml:"last_review"`
}

// Formulary & Cost Database structures (TIER 3)
type FormularyDatabase struct {
	Metadata    FormularyMetadata           `json:"metadata"`
	Formularies map[string]*FormularyEntry  `json:"formularies"`
}

type FormularyMetadata struct {
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type FormularyEntry struct {
	MedicationCode   string                `json:"medication_code"`
	GenericName      string                `json:"generic_name"`
	BrandName        string                `json:"brand_name"`
	FormularyStatus  string                `json:"formulary_status"`
	Tier             int                   `json:"tier"`
	PriorAuth        bool                  `json:"prior_authorization"`
	StepTherapy      bool                  `json:"step_therapy"`
	QuantityLimits   *QuantityLimits       `json:"quantity_limits"`
	CostInfo         CostInformation       `json:"cost_info"`
	Alternatives     []Alternative         `json:"alternatives"`
	Coverage         []CoverageInfo        `json:"coverage"`
}

type QuantityLimits struct {
	MaxQuantity     int    `json:"max_quantity"`
	Period          string `json:"period"`
	Override        bool   `json:"override_available"`
	OverrideReason  string `json:"override_reason"`
}

type CostInformation struct {
	AWP             float64 `json:"awp"`
	WAC             float64 `json:"wac"`
	Copay           float64 `json:"copay"`
	Coinsurance     float64 `json:"coinsurance"`
	Deductible      float64 `json:"deductible"`
	ThreeFourtyB    float64 `json:"340b_price"`
}

type Alternative struct {
	MedicationCode  string  `json:"medication_code"`
	GenericName     string  `json:"generic_name"`
	Reason          string  `json:"reason"`
	CostSavings     float64 `json:"cost_savings"`
	TherapeuticEq   bool    `json:"therapeutic_equivalent"`
}

type CoverageInfo struct {
	InsurancePlan   string  `json:"insurance_plan"`
	Covered         bool    `json:"covered"`
	CopayAmount     float64 `json:"copay_amount"`
	CoinsuranceRate float64 `json:"coinsurance_rate"`
	Restrictions    []string `json:"restrictions"`
}

// Monitoring Requirements Database structures (TIER 3)
type MonitoringDatabase struct {
	Metadata           MonitoringMetadata              `json:"metadata"`
	MonitoringProfiles map[string]*MonitoringProfile   `json:"monitoring_profiles"`
}

type MonitoringMetadata struct {
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type MonitoringProfile struct {
	MedicationCode    string                  `json:"medication_code"`
	ProfileName       string                  `json:"profile_name"`
	BaselineRequired  bool                    `json:"baseline_required"`
	BaselineTests     []BaselineTest          `json:"baseline_tests"`
	OngoingMonitoring []OngoingMonitoringTest `json:"ongoing_monitoring"`
	SafetyAlerts      []SafetyAlert           `json:"safety_alerts"`
	SpecialPopulations []SpecialPopulationMonitoring `json:"special_populations"`
}

type BaselineTest struct {
	TestName        string   `json:"test_name"`
	Required        bool     `json:"required"`
	Timing          string   `json:"timing"`
	NormalRange     string   `json:"normal_range"`
	ContraindicationThreshold string `json:"contraindication_threshold"`
}

type OngoingMonitoringTest struct {
	TestName        string   `json:"test_name"`
	Frequency       string   `json:"frequency"`
	Duration        string   `json:"duration"`
	NormalRange     string   `json:"normal_range"`
	ActionThreshold string   `json:"action_threshold"`
	Action          string   `json:"action"`
}

type SafetyAlert struct {
	AlertID         string   `json:"alert_id"`
	Parameter       string   `json:"parameter"`
	Condition       string   `json:"condition"`
	Severity        string   `json:"severity"`
	Action          string   `json:"action"`
	Notification    []string `json:"notification"`
}

type SpecialPopulationMonitoring struct {
	Population      string                  `json:"population"`
	Modifications   []MonitoringModification `json:"modifications"`
	AdditionalTests []OngoingMonitoringTest  `json:"additional_tests"`
}

type MonitoringModification struct {
	TestName        string `json:"test_name"`
	NewFrequency    string `json:"new_frequency"`
	NewThreshold    string `json:"new_threshold"`
	Rationale       string `json:"rationale"`
}

// Evidence Repository structures (TIER 4)
type EvidenceRepository struct {
	Metadata  EvidenceMetadata           `json:"metadata"`
	Evidence  map[string]*EvidenceEntry  `json:"evidence"`
}

type EvidenceMetadata struct {
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type EvidenceEntry struct {
	EvidenceID      string              `json:"evidence_id"`
	MedicationCode  string              `json:"medication_code"`
	Indication      string              `json:"indication"`
	EvidenceType    string              `json:"evidence_type"`
	EvidenceLevel   string              `json:"evidence_level"`
	Recommendation  string              `json:"recommendation"`
	Guidelines      []GuidelineRef      `json:"guidelines"`
	Studies         []StudyRef          `json:"studies"`
	QualityMetrics  QualityMetrics      `json:"quality_metrics"`
	LastReviewed    string              `json:"last_reviewed"`
}

type GuidelineRef struct {
	Organization    string `json:"organization"`
	GuidelineName   string `json:"guideline_name"`
	Version         string `json:"version"`
	Recommendation  string `json:"recommendation"`
	StrengthOfRec   string `json:"strength_of_recommendation"`
	URL             string `json:"url"`
}

type StudyRef struct {
	StudyID         string   `json:"study_id"`
	Title           string   `json:"title"`
	Authors         []string `json:"authors"`
	Journal         string   `json:"journal"`
	Year            int      `json:"year"`
	StudyType       string   `json:"study_type"`
	SampleSize      int      `json:"sample_size"`
	PrimaryOutcome  string   `json:"primary_outcome"`
	Results         string   `json:"results"`
	PMID            string   `json:"pmid"`
}

type QualityMetrics struct {
	OverallQuality  string  `json:"overall_quality"`
	BiasRisk        string  `json:"bias_risk"`
	Consistency     string  `json:"consistency"`
	Directness      string  `json:"directness"`
	Precision       string  `json:"precision"`
	ConfidenceLevel float64 `json:"confidence_level"`
}
