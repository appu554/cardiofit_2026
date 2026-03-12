package arbitration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// PATIENT CONTEXT (for arbitration)
// =============================================================================

// ArbitrationPatientContext contains patient information relevant to arbitration.
// This is a subset of the full PatientContext focused on clinical decision factors.
type ArbitrationPatientContext struct {
	PatientID      string    `json:"patient_id,omitempty"`
	Age            int       `json:"age"`
	Gender         string    `json:"gender"`
	IsPregnant     bool      `json:"is_pregnant"`
	Trimester      *int      `json:"trimester,omitempty"`
	EGFR           *float64  `json:"egfr,omitempty"`
	CrCl           *float64  `json:"crcl,omitempty"`
	CKDStage       *int      `json:"ckd_stage,omitempty"`
	ChildPughClass *string   `json:"child_pugh_class,omitempty"`
	CurrentMeds    []string  `json:"current_meds,omitempty"`
	Allergies      []string  `json:"allergies,omitempty"`
	Genotype       *Genotype `json:"genotype,omitempty"`
}

// Genotype contains pharmacogenomic information.
type Genotype struct {
	CYP2C9   *string `json:"cyp2c9,omitempty"`   // *1/*1, *1/*3, *2/*3, etc. (warfarin metabolism)
	CYP2C19  *string `json:"cyp2c19,omitempty"`  // *1/*1, *1/*2, *2/*2, etc. (clopidogrel, PPIs)
	CYP2D6   *string `json:"cyp2d6,omitempty"`   // *1/*1, *1/*4, etc. (codeine, tamoxifen)
	SLCO1B1  *string `json:"slco1b1,omitempty"`  // 521TT, 521TC, 521CC (statin transport)
	VKORC1   *string `json:"vkorc1,omitempty"`   // -1639G>A (warfarin sensitivity)
	HLA_B    *string `json:"hla_b,omitempty"`    // *57:01, *15:02, etc. (abacavir, carbamazepine)
}

// HasRenalImpairment returns true if patient has eGFR < 60 or CKD stage >= 3.
func (p *ArbitrationPatientContext) HasRenalImpairment() bool {
	if p.EGFR != nil && *p.EGFR < 60 {
		return true
	}
	if p.CKDStage != nil && *p.CKDStage >= 3 {
		return true
	}
	return false
}

// HasSevereRenalImpairment returns true if eGFR < 30.
func (p *ArbitrationPatientContext) HasSevereRenalImpairment() bool {
	if p.EGFR != nil && *p.EGFR < 30 {
		return true
	}
	if p.CrCl != nil && *p.CrCl < 30 {
		return true
	}
	return false
}

// =============================================================================
// ASSERTION STRUCTS
// =============================================================================

// CanonicalRuleAssertion represents a Phase 3b.5 canonical rule.
type CanonicalRuleAssertion struct {
	RuleID          uuid.UUID   `json:"rule_id"`
	Domain          string      `json:"domain"`           // KB-1, KB-4, KB-5
	DrugRxCUI       string      `json:"drug_rxcui"`
	DrugName        string      `json:"drug_name,omitempty"`
	Condition       *Condition  `json:"condition"`
	Action          *Action     `json:"action"`
	Effect          ClinicalEffect `json:"effect"`
	Confidence      float64     `json:"confidence"`
	Fingerprint     string      `json:"fingerprint"`
	ProvenanceCount int         `json:"provenance_count"` // How many sources agree
	SourceLabel     string      `json:"source_label,omitempty"`
}

// Condition represents a rule condition.
type Condition struct {
	Type      string      `json:"type"`      // RENAL, HEPATIC, AGE, PREGNANCY, etc.
	Operator  string      `json:"operator"`  // <, <=, >, >=, ==, IN
	Parameter string      `json:"parameter"` // eGFR, CrCl, age, etc.
	Value     interface{} `json:"value"`     // Threshold value
	Unit      string      `json:"unit,omitempty"`
}

// Action represents a rule action.
type Action struct {
	Type         string  `json:"type"`                    // AVOID, REDUCE_DOSE, MONITOR, etc.
	Description  string  `json:"description"`
	DoseModifier *string `json:"dose_modifier,omitempty"` // "50%", "max 10mg", etc.
}

// AuthorityFactAssertion represents a curated authority guideline.
type AuthorityFactAssertion struct {
	ID             uuid.UUID      `json:"id"`
	Authority      string         `json:"authority"`       // CPIC, CREDIBLEMEDS, LACTMED
	AuthorityLevel AuthorityLevel `json:"authority_level"` // DEFINITIVE, PRIMARY, etc.
	DrugRxCUI      string         `json:"drug_rxcui,omitempty"`
	DrugName       string         `json:"drug_name,omitempty"`
	GeneSymbol     *string        `json:"gene_symbol,omitempty"`    // CYP2C19, CYP2D6
	Phenotype      *string        `json:"phenotype,omitempty"`      // Poor Metabolizer, etc.
	ConditionCode  *string        `json:"condition_code,omitempty"` // ICD-10
	ConditionName  *string        `json:"condition_name,omitempty"`
	Assertion      string         `json:"assertion"`        // What the authority says
	Effect         ClinicalEffect `json:"effect"`           // CONTRAINDICATED, AVOID, etc.
	EvidenceLevel  string         `json:"evidence_level"`   // 1A, 1B, 2A, 2B
	Recommendation string         `json:"recommendation,omitempty"`
	DosingGuidance string         `json:"dosing_guidance,omitempty"`
	LastUpdated    time.Time      `json:"last_updated"`
}

// LabInterpretationAssertion represents a KB-16 lab interpretation result.
type LabInterpretationAssertion struct {
	LabTest         string  `json:"lab_test"`         // eGFR, Creatinine, Potassium
	LOINCCode       string  `json:"loinc_code"`
	Value           float64 `json:"value"`
	Unit            string  `json:"unit"`
	Interpretation  string  `json:"interpretation"`   // NORMAL, ABNORMAL, CRITICAL
	ReferenceRange  string  `json:"reference_range"`  // Context-aware range used
	ClinicalContext string  `json:"clinical_context"` // Pregnancy T3, CKD Stage 4
	Specificity     int     `json:"specificity"`      // Specificity score of range used
	Effect          ClinicalEffect `json:"effect,omitempty"` // Derived effect
}

// IsCritical returns true if the lab interpretation is critical.
func (l *LabInterpretationAssertion) IsCritical() bool {
	return l.Interpretation == "CRITICAL" || l.Interpretation == "PANIC_LOW" || l.Interpretation == "PANIC_HIGH"
}

// RegulatoryBlockAssertion represents an FDA Black Box or REMS requirement.
type RegulatoryBlockAssertion struct {
	ID                   uuid.UUID      `json:"id"`
	DrugRxCUI            string         `json:"drug_rxcui"`
	DrugName             string         `json:"drug_name"`
	BlockType            string         `json:"block_type"`             // BLACK_BOX, REMS, CONTRAINDICATION
	ConditionDescription string         `json:"condition_description"`
	AffectedPopulation   string         `json:"affected_population,omitempty"`
	Effect               ClinicalEffect `json:"effect"` // Always CONTRAINDICATED for blocks
	Severity             string         `json:"severity"`
	FDALabelDate         *time.Time     `json:"fda_label_date,omitempty"`
}

// LocalPolicyAssertion represents a hospital/institution-specific override.
type LocalPolicyAssertion struct {
	ID                   uuid.UUID      `json:"id"`
	InstitutionID        string         `json:"institution_id"`
	InstitutionName      string         `json:"institution_name"`
	PolicyCode           string         `json:"policy_code"`
	PolicyName           string         `json:"policy_name"`
	DrugRxCUI            *string        `json:"drug_rxcui,omitempty"`
	DrugClass            *string        `json:"drug_class,omitempty"`
	ConditionDescription string         `json:"condition_description,omitempty"`
	OverrideTarget       SourceType     `json:"override_target"` // What this policy overrides
	Effect               ClinicalEffect `json:"effect"`
	Justification        string         `json:"justification"`
	Restrictions         string         `json:"restrictions,omitempty"`
	ApprovalRequired     bool           `json:"approval_required"`
}

// =============================================================================
// ARBITRATION INPUT
// =============================================================================

// ArbitrationInput contains all inputs for a single arbitration decision.
type ArbitrationInput struct {
	// The clinical question being arbitrated
	DrugRxCUI       string                     `json:"drug_rxcui"`
	DrugName        string                     `json:"drug_name,omitempty"`
	PatientContext  *ArbitrationPatientContext `json:"patient_context"`
	ClinicalIntent  string                     `json:"clinical_intent"` // PRESCRIBE, CONTINUE, MODIFY, DISCONTINUE

	// All available assertions about this decision
	CanonicalRules     []CanonicalRuleAssertion     `json:"canonical_rules,omitempty"`
	AuthorityFacts     []AuthorityFactAssertion     `json:"authority_facts,omitempty"`
	LabInterpretations []LabInterpretationAssertion `json:"lab_interpretations,omitempty"`
	RegulatoryBlocks   []RegulatoryBlockAssertion   `json:"regulatory_blocks,omitempty"`
	LocalPolicies      []LocalPolicyAssertion       `json:"local_policies,omitempty"`

	// Request metadata
	RequestID   string    `json:"request_id,omitempty"`
	RequestedAt time.Time `json:"requested_at"`
	RequestedBy string    `json:"requested_by,omitempty"`
}

// Hash returns a SHA256 hash of the input for audit purposes.
func (i *ArbitrationInput) Hash() string {
	data, _ := json.Marshal(i)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// HasRegulatoryBlocks returns true if any regulatory blocks exist.
func (i *ArbitrationInput) HasRegulatoryBlocks() bool {
	return len(i.RegulatoryBlocks) > 0
}

// HasAuthorityFacts returns true if any authority facts exist.
func (i *ArbitrationInput) HasAuthorityFacts() bool {
	return len(i.AuthorityFacts) > 0
}

// TotalAssertionCount returns the total number of assertions.
func (i *ArbitrationInput) TotalAssertionCount() int {
	return len(i.CanonicalRules) + len(i.AuthorityFacts) +
		len(i.LabInterpretations) + len(i.RegulatoryBlocks) +
		len(i.LocalPolicies)
}

// =============================================================================
// ARBITRATION DECISION
// =============================================================================

// ArbitrationDecision contains the complete result of truth arbitration.
type ArbitrationDecision struct {
	// Primary decision
	Decision   DecisionType `json:"decision"`   // ACCEPT, BLOCK, OVERRIDE, DEFER, ESCALATE
	Confidence float64      `json:"confidence"` // 0.0-1.0

	// What drove this decision
	WinningSource    *SourceType `json:"winning_source,omitempty"`
	WinningAssertion interface{} `json:"winning_assertion,omitempty"`
	PrecedenceRule   string      `json:"precedence_rule,omitempty"` // P1, P2, P3...

	// Conflicts detected
	ConflictsFound []Conflict `json:"conflicts_found"`
	ConflictCount  int        `json:"conflict_count"`

	// What action should be taken
	RecommendedAction  string   `json:"recommended_action"`  // Human-readable
	ClinicalRationale  string   `json:"clinical_rationale"`
	AlternativeActions []string `json:"alternative_actions,omitempty"`

	// Patient context snapshot
	PatientAge       *int    `json:"patient_age,omitempty"`
	PatientGender    *string `json:"patient_gender,omitempty"`
	PatientPregnant  *bool   `json:"patient_pregnant,omitempty"`
	PatientTrimester *int    `json:"patient_trimester,omitempty"`
	PatientEGFR      *float64 `json:"patient_egfr,omitempty"`
	PatientCKDStage  *int    `json:"patient_ckd_stage,omitempty"`

	// Governance trail
	ArbitrationID uuid.UUID    `json:"arbitration_id"`
	DrugRxCUI     string       `json:"drug_rxcui"`
	DrugName      string       `json:"drug_name,omitempty"`
	ClinicalIntent string      `json:"clinical_intent"`
	ArbitratedAt  time.Time    `json:"arbitrated_at"`
	ArbitratedBy  string       `json:"arbitrated_by"`
	InputHash     string       `json:"input_hash"` // SHA256 of inputs
	AuditTrail    []AuditEntry `json:"audit_trail"`

	// Override tracking
	WasOverridden   bool       `json:"was_overridden"`
	OverriddenBy    *string    `json:"overridden_by,omitempty"`
	OverrideReason  *string    `json:"override_reason,omitempty"`
	OverrideAt      *time.Time `json:"override_at,omitempty"`
}

// TableName returns the database table name.
func (ArbitrationDecision) TableName() string {
	return "arbitration_decisions"
}

// NewArbitrationDecision creates a new decision with initialized fields.
func NewArbitrationDecision(input *ArbitrationInput) *ArbitrationDecision {
	decision := &ArbitrationDecision{
		ArbitrationID:  uuid.New(),
		ArbitratedAt:   time.Now(),
		ArbitratedBy:   "SYSTEM",
		InputHash:      input.Hash(),
		DrugRxCUI:      input.DrugRxCUI,
		DrugName:       input.DrugName,
		ClinicalIntent: input.ClinicalIntent,
		ConflictsFound: make([]Conflict, 0),
		AuditTrail:     make([]AuditEntry, 0),
	}

	// Capture patient context snapshot
	if input.PatientContext != nil {
		ctx := input.PatientContext
		decision.PatientAge = &ctx.Age
		decision.PatientGender = &ctx.Gender
		decision.PatientPregnant = &ctx.IsPregnant
		decision.PatientTrimester = ctx.Trimester
		decision.PatientEGFR = ctx.EGFR
		decision.PatientCKDStage = ctx.CKDStage
	}

	return decision
}

// AddAuditEntry adds a step to the audit trail.
func (d *ArbitrationDecision) AddAuditEntry(stepName, description string, inputs, outputs map[string]interface{}) {
	entry := AuditEntry{
		ID:              uuid.New(),
		ArbitrationID:   d.ArbitrationID,
		StepNumber:      len(d.AuditTrail) + 1,
		StepName:        stepName,
		StepDescription: description,
		Inputs:          inputs,
		Outputs:         outputs,
		CreatedAt:       time.Now(),
	}
	d.AuditTrail = append(d.AuditTrail, entry)
}

// AddConflict adds a conflict to the decision.
func (d *ArbitrationDecision) AddConflict(conflict Conflict) {
	conflict.ArbitrationID = d.ArbitrationID
	if conflict.ID == uuid.Nil {
		conflict.ID = uuid.New()
	}
	if conflict.DetectedAt.IsZero() {
		conflict.DetectedAt = time.Now()
	}
	d.ConflictsFound = append(d.ConflictsFound, conflict)
	d.ConflictCount = len(d.ConflictsFound)
}

// IsBlocking returns true if this decision prevents the clinical action.
func (d *ArbitrationDecision) IsBlocking() bool {
	return d.Decision == DecisionBlock
}

// RequiresEscalation returns true if human review is needed.
func (d *ArbitrationDecision) RequiresEscalation() bool {
	return d.Decision == DecisionEscalate
}

// CanProceed returns true if the clinical action can proceed (possibly with override).
func (d *ArbitrationDecision) CanProceed() bool {
	return d.Decision == DecisionAccept || d.Decision == DecisionOverride
}

// =============================================================================
// EVALUATED ASSERTIONS
// =============================================================================

// EvaluatedAssertions contains all assertions after evaluation against patient context.
type EvaluatedAssertions struct {
	TriggeredRules       []CanonicalRuleAssertion     `json:"triggered_rules"`
	ApplicableAuthorities []AuthorityFactAssertion    `json:"applicable_authorities"`
	RelevantLabs         []LabInterpretationAssertion `json:"relevant_labs"`
	ActiveBlocks         []RegulatoryBlockAssertion   `json:"active_blocks"`
	ApplicablePolicies   []LocalPolicyAssertion       `json:"applicable_policies"`
}

// HasTriggers returns true if any assertions are triggered/applicable.
func (e *EvaluatedAssertions) HasTriggers() bool {
	return len(e.TriggeredRules) > 0 ||
		len(e.ApplicableAuthorities) > 0 ||
		len(e.RelevantLabs) > 0 ||
		len(e.ActiveBlocks) > 0 ||
		len(e.ApplicablePolicies) > 0
}

// MostRestrictiveEffect returns the most restrictive clinical effect across all assertions.
func (e *EvaluatedAssertions) MostRestrictiveEffect() ClinicalEffect {
	mostRestrictive := EffectNoEffect

	for _, r := range e.TriggeredRules {
		if r.Effect.MoreRestrictiveThan(mostRestrictive) {
			mostRestrictive = r.Effect
		}
	}
	for _, a := range e.ApplicableAuthorities {
		if a.Effect.MoreRestrictiveThan(mostRestrictive) {
			mostRestrictive = a.Effect
		}
	}
	for _, b := range e.ActiveBlocks {
		if b.Effect.MoreRestrictiveThan(mostRestrictive) {
			mostRestrictive = b.Effect
		}
	}

	return mostRestrictive
}
