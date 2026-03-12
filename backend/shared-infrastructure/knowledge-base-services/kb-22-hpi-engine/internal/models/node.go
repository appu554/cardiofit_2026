package models

// NodeDefinition represents a parsed P1-P26 HPI node YAML.
// This is the contract between KB-22 and the clinical author team.
type NodeDefinition struct {
	NodeID               string   `yaml:"node_id" json:"node_id"`
	Version              string   `yaml:"version" json:"version"`
	TitleEN              string   `yaml:"title_en" json:"title_en"`
	TitleHI              string   `yaml:"title_hi" json:"title_hi"`
	MaxQuestions         int      `yaml:"max_questions" json:"max_questions"`
	ConvergenceThreshold float64  `yaml:"convergence_threshold" json:"convergence_threshold"`
	StrataSupported      []string `yaml:"strata_supported" json:"strata_supported"`

	// R-01: dual-criterion termination
	PosteriorGapThreshold float64 `yaml:"posterior_gap_threshold" json:"posterior_gap_threshold"`
	ConvergenceLogic      string  `yaml:"convergence_logic" json:"convergence_logic"` // BOTH | EITHER | POSTERIOR_ONLY

	// N-01: optional KB-3 guideline prior source
	GuidelinePriorSource string `yaml:"guideline_prior_source,omitempty" json:"guideline_prior_source,omitempty"`

	// G15: implicit 'Other' bucket differential for open-world diagnosis.
	// When enabled, an implicit _OTHER differential is added with the
	// specified prior (default 0.15). Updated via geometric mean of
	// inverse LRs. Triggers DIFFERENTIAL_INCOMPLETE at >0.30.
	OtherBucketEnabled bool    `yaml:"other_bucket_enabled" json:"other_bucket_enabled"`
	OtherBucketPrior   float64 `yaml:"other_bucket_prior,omitempty" json:"other_bucket_prior,omitempty"` // default 0.15

	// G1: safety floor clamping — prevents dangerous differentials from being
	// ruled out by negative evidence alone. Floors are minimum posterior probabilities.
	// Simple format: applies to all strata. Stratum-specific format overrides per-stratum.
	// A03 rule: if StrataSupported is non-empty, SafetyFloorsByStratum SHOULD be used.
	SafetyFloors          map[string]float64            `yaml:"safety_floors,omitempty" json:"safety_floors,omitempty"`                       // differential_id -> floor (e.g. {ACS: 0.05})
	SafetyFloorsByStratum map[string]map[string]float64 `yaml:"safety_floors_by_stratum,omitempty" json:"safety_floors_by_stratum,omitempty"` // stratum -> {differential_id -> floor}

	// G9: BP-status conditional prior overrides. When a patient's bp_status
	// matches a key in this map, the corresponding prior overrides are applied
	// BEFORE converting to log-odds. Values are additive deltas to base priors.
	// E.g., {"SEVERE": {"ACS": 0.05, "AORTIC_DISSECTION": 0.03}} means:
	// when bp_status == SEVERE, ACS prior += 0.05 and AORTIC_DISSECTION += 0.03.
	// The sum of overrides must not cause any prior to exceed 1.0 or go below 0.
	ConditionalPriorOverrides map[string]map[string]float64 `yaml:"conditional_prior_overrides,omitempty" json:"conditional_prior_overrides,omitempty"`

	Differentials    []DifferentialDef    `yaml:"differentials" json:"differentials"`
	SafetyTriggers   []SafetyTriggerDef   `yaml:"safety_triggers" json:"safety_triggers"`
	Questions        []QuestionDef        `yaml:"questions" json:"questions"`
	ContextModifiers []ContextModifierDef `yaml:"context_modifiers,omitempty" json:"context_modifiers,omitempty"`
	SexModifiers     []SexModifierDef     `yaml:"sex_modifiers,omitempty" json:"sex_modifiers,omitempty"`

	// G17: contradiction pair definitions. Each pair is [Q_A, Q_B] where a YES
	// to both is logically contradictory (e.g., "no chest pain at rest" vs
	// "chest pain at rest worsens with breathing"). When detected, the engine
	// triggers a re-ask of the second question using its alt_prompt.
	ContradictionPairs []ContradictionPairDef `yaml:"contradiction_pairs,omitempty" json:"contradiction_pairs,omitempty"`

	// G13: node transition rules. Evaluated after each answer to determine
	// whether the session should spawn a concurrent node, hand off to a
	// different node, or flag for specialist review.
	Transitions []NodeTransitionDef `yaml:"transitions,omitempty" json:"transitions,omitempty"`
}

// CM effect type constants for G5.
const (
	CMEffectIncreasePrior = "INCREASE_PRIOR" // default: logit-based prior shift
	CMEffectDecreasePrior = "DECREASE_PRIOR" // logit-based prior decrease
	CMEffectHardBlock     = "HARD_BLOCK"     // G5: blocks a treatment, no diagnostic shift
	CMEffectOverride      = "OVERRIDE"       // G5: forces posterior minimum for target differentials
	CMEffectSymptomMod    = "SYMPTOM_MODIFICATION" // G8 (deferred): LR suppression
)

// ContextModifierDef represents a node-level context modifier as authored in YAML.
// Uses the compact adjustments map format: {differential_id: magnitude}.
// NodeLoader expands these into flat []ContextModifier structs for CMApplicator.
type ContextModifierDef struct {
	ID          string             `yaml:"id" json:"id"`
	Name        string             `yaml:"name" json:"name"`
	Description string             `yaml:"description,omitempty" json:"description,omitempty"`
	Source      string             `yaml:"source,omitempty" json:"source,omitempty"`
	Adjustments map[string]float64 `yaml:"adjustments" json:"adjustments"` // differential_id -> magnitude (probability delta)

	// G5: Effect type — defaults to INCREASE_PRIOR when empty.
	// HARD_BLOCK: no diagnostic shift; BlockedTreatment specifies the contraindicated therapy.
	// OVERRIDE: no diagnostic shift via CM; OverrideTargets specifies posterior minimums.
	EffectType       string             `yaml:"effect_type,omitempty" json:"effect_type,omitempty"`
	BlockedTreatment string             `yaml:"blocked_treatment,omitempty" json:"blocked_treatment,omitempty"` // e.g. "NITRATE_THERAPY"
	OverrideTargets  map[string]float64 `yaml:"override_targets,omitempty" json:"override_targets,omitempty"`  // differential_id -> min posterior
}

// SexModifierDef defines an OR-based prior adjustment triggered by patient sex/age.
// Unlike CMs (probability deltas via logit), sex modifiers specify direct log-odds
// deltas. OR 1.8 = log(1.8) = +0.59 log-odds. Applied once at session init after
// InitPriors, not per-question.
type SexModifierDef struct {
	ID          string             `yaml:"id" json:"id"`
	Condition   string             `yaml:"condition" json:"condition"`     // e.g. "sex == Female", "sex == Female AND age >= 50"
	Adjustments map[string]float64 `yaml:"adjustments" json:"adjustments"` // differential_id -> log-odds delta
	Source      string             `yaml:"source,omitempty" json:"source,omitempty"`
}

// DifferentialDef defines a diagnosis candidate with per-stratum priors.
type DifferentialDef struct {
	ID      string             `yaml:"id" json:"id"`
	LabelEN string             `yaml:"label" json:"label_en"`
	Priors  map[string]float64 `yaml:"priors" json:"priors"` // stratum -> prior probability

	// G3: medication-conditional differential. When set, this differential is only
	// included in the session if the activation condition is satisfied by the
	// patient's active medication list from KB-20. If excluded, its prior mass
	// redistributes proportionally across remaining active differentials (NOT into
	// the G15 Other bucket).
	// Format: "med_class == SGLT2i" or "med_class == Metformin AND eGFR < 30"
	ActivationCondition string `yaml:"activation_condition,omitempty" json:"activation_condition,omitempty"`

	// R-07: LR source provenance
	PopulationReference string `yaml:"population_reference,omitempty" json:"population_reference,omitempty"`
}

// SafetyTriggerDef defines a safety condition evaluated by SafetyEngine.
type SafetyTriggerDef struct {
	ID        string `yaml:"id" json:"id"`
	Type      string `yaml:"type,omitempty" json:"type,omitempty"` // R-06: BOOLEAN (default) or COMPOSITE_SCORE (stub)
	Condition string `yaml:"condition" json:"condition"`           // Boolean expression: 'Q001=YES AND Q003=YES'
	Severity  string `yaml:"severity" json:"severity"`             // IMMEDIATE | URGENT | WARN
	Action    string `yaml:"recommended_action" json:"action"`

	// R-06: COMPOSITE_SCORE fields (stub — not implemented in Circle 1)
	Weights   map[string]float64 `yaml:"weights,omitempty" json:"weights,omitempty"`
	Threshold float64            `yaml:"threshold,omitempty" json:"threshold,omitempty"`
}

// AcuityCategory classifies temporal presentation acuity (G7).
type AcuityCategory string

const (
	AcuityAcute    AcuityCategory = "ACUTE"    // onset < 24h
	AcuitySubacute AcuityCategory = "SUBACUTE" // onset 24h–14d
	AcuityChronic  AcuityCategory = "CHRONIC"  // onset > 14d
	AcuityUnknown  AcuityCategory = "UNKNOWN"  // not yet classified
)

// Answer type constants for G10.
const (
	AnswerTypeBinary      = "BINARY"      // default: YES/NO/PATA_NAHI
	AnswerTypeCategorical = "CATEGORICAL" // G10: ordinal/nominal multi-value
)

// QuestionDef defines an HPI question with LR values and metadata.
type QuestionDef struct {
	ID       string `yaml:"id" json:"id"`
	TextEN   string `yaml:"text_en" json:"text_en"`
	TextHI   string `yaml:"text_hi" json:"text_hi"`

	Mandatory bool   `yaml:"mandatory" json:"mandatory"`
	SafetyRole string `yaml:"safety_role,omitempty" json:"safety_role,omitempty"`

	// G10: Answer type — BINARY (default) or CATEGORICAL.
	// BINARY uses LRPositive/LRNegative with YES/NO/PATA_NAHI answers.
	// CATEGORICAL uses LRCategorical with AnswerOptions-defined values + PATA_NAHI.
	AnswerType string `yaml:"answer_type,omitempty" json:"answer_type,omitempty"`

	// Likelihood ratios per differential (BINARY questions)
	LRPositive map[string]float64 `yaml:"lr_positive" json:"lr_positive"`
	LRNegative map[string]float64 `yaml:"lr_negative" json:"lr_negative"`

	// G10: Categorical LR map — answer_value -> differential_id -> LR.
	// Only used when AnswerType == CATEGORICAL.
	// Each valid answer option must have an entry; each entry must cover all
	// differentials declared in the node's Differentials list.
	LRCategorical map[string]map[string]float64 `yaml:"lr_categorical,omitempty" json:"lr_categorical,omitempty"`

	// G10: Valid answer options for CATEGORICAL questions.
	// E.g. ["NONE", "MILD", "MODERATE", "SEVERE"] for a severity scale.
	// PATA_NAHI is always implicitly valid and need not be listed.
	AnswerOptions []string `yaml:"answer_options,omitempty" json:"answer_options,omitempty"`

	// Branch condition: null = always asked
	BranchCondition *string `yaml:"branch_condition" json:"branch_condition,omitempty"`

	// R-02: symptom cluster dampening
	Cluster          string  `yaml:"cluster,omitempty" json:"cluster,omitempty"`
	ClusterDampening float64 `yaml:"cluster_dampening,omitempty" json:"cluster_dampening,omitempty"`

	// G16: alternate prompt for pata-nahi rephrase cascade.
	// When consecutive pata-nahi count reaches 2, the next question is
	// presented using AltPrompt (simpler phrasing) instead of TextEN/TextHI.
	AltPromptEN string `yaml:"alt_prompt_en,omitempty" json:"alt_prompt_en,omitempty"`
	AltPromptHI string `yaml:"alt_prompt_hi,omitempty" json:"alt_prompt_hi,omitempty"`

	// R-05: minimum inclusion guard (auto-injected for safety trigger components)
	MinimumInclusionGuard bool `yaml:"minimum_inclusion_guard,omitempty" json:"minimum_inclusion_guard,omitempty"`

	// G6: stratum-conditional LR overrides. When the patient's stratum matches
	// a key in these maps, the corresponding LR values are used instead of the
	// base LRPositive/LRNegative. This handles cases where a symptom's
	// discriminating power genuinely differs by stratum (e.g., orthopnea LR+
	// drops from 2.2 to 1.2 in CKD+HF patients).
	// Format: stratum -> differential_id -> LR value.
	LRPositiveByStratum map[string]map[string]float64 `yaml:"lr_positive_by_stratum,omitempty" json:"lr_positive_by_stratum,omitempty"`
	LRNegativeByStratum map[string]map[string]float64 `yaml:"lr_negative_by_stratum,omitempty" json:"lr_negative_by_stratum,omitempty"`

	// R-07: LR source provenance
	LRSource        string `yaml:"lr_source,omitempty" json:"lr_source,omitempty"`
	LREvidenceClass string `yaml:"lr_evidence_class,omitempty" json:"lr_evidence_class,omitempty"`

	// G7: Acuity tag — classifies whether this question contributes to
	// temporal acuity scoring. Questions with acuity_tag are evaluated by
	// AcuityScorer in parallel to Bayesian inference.
	AcuityTag string `yaml:"acuity_tag,omitempty" json:"acuity_tag,omitempty"` // ONSET, DURATION, PROGRESSION, PATTERN

	// G19: CM coverage tag — lists CM modifier IDs whose firing makes this
	// question redundant. If all listed CMs have fired, QuestionOrchestrator
	// skips this question (BAY-8 skip-redundancy rule).
	CMCoverage []string `yaml:"cm_coverage,omitempty" json:"cm_coverage,omitempty"`
}

// G13: Node transition mode constants.
const (
	TransitionConcurrent = "CONCURRENT" // run target node in parallel with current
	TransitionHandoff    = "HANDOFF"    // complete current, pass posteriors as priors to target
	TransitionFlag       = "FLAG"       // flag for specialist review, no automatic node start
)

// NodeTransitionDef defines a G13 node transition rule.
// When the trigger condition evaluates true, the session transitions
// to the target node using the specified mode.
type NodeTransitionDef struct {
	ID          string `yaml:"id" json:"id"`
	TargetNode  string `yaml:"target_node" json:"target_node"`   // e.g. "P2_DYSPNEA"
	Mode        string `yaml:"mode" json:"mode"`                 // CONCURRENT, HANDOFF, FLAG
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Trigger condition — evaluated against current session state.
	// Supported condition types:
	//   "posterior:ACS >= 0.40"          — differential posterior threshold
	//   "questions_asked >= 8"           — question count threshold
	//   "converged"                      — node has reached convergence
	//   "safety_flag:ST_ELEVATION"       — specific safety flag has fired
	TriggerCondition string `yaml:"trigger_condition" json:"trigger_condition"`

	// Priority for ordering when multiple transitions fire simultaneously.
	// Lower number = higher priority. Default 0.
	Priority int `yaml:"priority,omitempty" json:"priority,omitempty"`
}

// TransitionEvent records that a node transition was evaluated and triggered.
type TransitionEvent struct {
	TransitionID string `json:"transition_id"`
	SourceNode   string `json:"source_node"`
	TargetNode   string `json:"target_node"`
	Mode         string `json:"mode"`
	Reason       string `json:"reason"` // human-readable trigger description
}

// ContradictionPairDef defines a G17 contradiction pair between two questions.
// When both questions have been answered YES, the pair is flagged as contradictory
// and the second question is re-asked using its alt_prompt.
type ContradictionPairDef struct {
	ID          string `yaml:"id" json:"id"`
	QuestionA   string `yaml:"question_a" json:"question_a"`     // first question ID
	QuestionB   string `yaml:"question_b" json:"question_b"`     // second question ID (re-asked on contradiction)
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// ContradictionEvent records that a contradiction was detected between two answers.
type ContradictionEvent struct {
	PairID       string `json:"pair_id"`
	QuestionA    string `json:"question_a"`
	QuestionB    string `json:"question_b"`
	ReaskQuestion string `json:"reask_question"` // the question to re-ask (always QuestionB)
	UseAltPrompt bool   `json:"use_alt_prompt"`
}

// CrossNodeTrigger is a global safety trigger evaluated regardless of active node.
// F-07: loaded from cross_node_triggers.yaml.
type CrossNodeTrigger struct {
	TriggerID       string `gorm:"type:varchar(64);primaryKey" yaml:"trigger_id" json:"trigger_id"`
	Condition       string `gorm:"type:text;not null" yaml:"condition" json:"condition"`
	Severity        string `gorm:"type:varchar(16);not null" yaml:"severity" json:"severity"`
	RecommendedAction string `gorm:"type:text;not null" yaml:"recommended_action" json:"recommended_action"`
	Active          bool   `gorm:"type:bool;default:true" yaml:"active" json:"active"`
}

func (CrossNodeTrigger) TableName() string { return "cross_node_triggers" }

// CrossNodeTriggersFile is the YAML structure for cross_node_triggers.yaml.
type CrossNodeTriggersFile struct {
	Triggers []CrossNodeTrigger `yaml:"triggers"`
}
