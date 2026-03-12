package icu

import (
	"context"
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// DominanceEngine implements ICU dominance state classification and veto evaluation.
//
// ARCHITECTURE CRITICAL (CTO/CMO Directive):
//
//	"CQL explains. KB-19 recommends. ICU decides."
//
// This engine is the AUTHORITY for clinical action approval in ICU contexts.
// It is NOT a KB - it is a state-dominance engine that can veto everything except reality.
//
// Usage:
//
//	engine := icu.NewDominanceEngine(config)
//	result, err := engine.Evaluate(ctx, action, facts)
//	if result.Vetoed {
//	    // Action is blocked by ICU dominance
//	}
type DominanceEngine struct {
	config *DominanceConfig
}

// DominanceConfig holds configuration for the dominance engine.
type DominanceConfig struct {
	// StrictMode enables strict threshold checking (no tolerance)
	StrictMode bool

	// AuditEnabled enables override audit logging
	AuditEnabled bool

	// DefaultConfidence is the baseline confidence for classifications
	DefaultConfidence float64
}

// DefaultDominanceConfig returns production-safe default configuration.
func DefaultDominanceConfig() *DominanceConfig {
	return &DominanceConfig{
		StrictMode:        true,
		AuditEnabled:      true,
		DefaultConfidence: 0.95,
	}
}

// NewDominanceEngine creates a new dominance engine with the given configuration.
func NewDominanceEngine(config *DominanceConfig) *DominanceEngine {
	if config == nil {
		config = DefaultDominanceConfig()
	}
	return &DominanceEngine{config: config}
}

// ═══════════════════════════════════════════════════════════════════════════════
// EVALUATE - Main entry point for dominance evaluation
// ═══════════════════════════════════════════════════════════════════════════════

// Evaluate checks if ICU dominance should veto a proposed action.
// This is the MAIN entry point called by all workflow KBs (KB-14, KB-18, KB-19).
//
// CRITICAL: This is called BEFORE any RuntimeClient KB executes an action.
func (e *DominanceEngine) Evaluate(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts) (*DominanceResult, error) {
	if facts == nil {
		return nil, fmt.Errorf("SafetyFacts cannot be nil")
	}

	// Validate input facts
	if err := facts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SafetyFacts: %w", err)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// CONTEXT GATE: Only assert dominance in ICU/Code contexts
	// ═══════════════════════════════════════════════════════════════════════════
	if !facts.IsICUContext() {
		// Not in ICU context - pass unless absolute contraindication
		return e.evaluateAbsoluteContraindicationsOnly(ctx, action, facts)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 1: Classify dominance state (explicit classifier)
	// ═══════════════════════════════════════════════════════════════════════════
	state := e.ClassifyDominanceState(facts)

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 2: Evaluate action against dominance state
	// ═══════════════════════════════════════════════════════════════════════════
	switch state {
	case StateNeurologicCollapse:
		return e.evaluateNeurologicDominance(ctx, action, facts)
	case StateShock:
		return e.evaluateShockDominance(ctx, action, facts)
	case StateHypoxia:
		return e.evaluateHypoxiaDominance(ctx, action, facts)
	case StateActiveBleed:
		return e.evaluateActiveBleedDominance(ctx, action, facts)
	case StateLowOutputFailure:
		return e.evaluateLowOutputDominance(ctx, action, facts)
	default:
		return &DominanceResult{
			CurrentState: StateNone,
			Vetoed:       false,
			Confidence:   e.config.DefaultConfidence,
		}, nil
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// EXPLICIT STATE CLASSIFIER
// ═══════════════════════════════════════════════════════════════════════════════

// ClassifyDominanceState is the EXPLICIT State Classifier.
//
// PRIORITY ORDER (highest to lowest):
//  1. NEUROLOGIC_COLLAPSE - Brain death/herniation trumps everything
//  2. SHOCK              - Hemodynamic instability is next
//  3. HYPOXIA            - Respiratory failure follows
//  4. ACTIVE_BLEED       - Hemorrhage control
//  5. LOW_OUTPUT_FAILURE - Cardiac output failure
//  6. NONE               - Normal state, no dominance
//
// CLINICAL RATIONALE:
//   - Neurologic collapse can cause ALL other states (code blue, herniation)
//   - Shock kills faster than hypoxia (minutes vs hours)
//   - Hypoxia compounds all other states rapidly
//   - Active bleeding must be addressed before optimizing cardiac output
func (e *DominanceEngine) ClassifyDominanceState(facts *SafetyFacts) DominanceState {
	if facts == nil {
		return StateNone
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 1: NEUROLOGIC_COLLAPSE
	// GCS <8, Active seizure, ICP >20, Herniation signs
	// ─────────────────────────────────────────────────────────────────────────
	if facts.GCS < 8 || facts.HasActiveSeizure || facts.ICP > 20 || facts.HasHerniationSigns {
		return StateNeurologicCollapse
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 2: SHOCK
	// MAP <65, Lactate >4, Vasopressor requirement, Septic shock
	// ─────────────────────────────────────────────────────────────────────────
	if facts.MAP < 65 || facts.Lactate > 4.0 || facts.OnVasopressors || facts.HasSepticShock {
		return StateShock
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 3: HYPOXIA
	// SpO2 <88%, P/F ratio <100, FiO2 >0.6
	// ─────────────────────────────────────────────────────────────────────────
	if facts.SpO2 < 88 || facts.PFRatio < 100 || facts.FiO2 > 0.6 {
		return StateHypoxia
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 4: ACTIVE_BLEED
	// Hgb drop >2g/dL/6h, Active transfusion, Surgical bleeding, Critical INR
	// ─────────────────────────────────────────────────────────────────────────
	if facts.HgbDrop6h > 2.0 || facts.HasActiveTransfusion || facts.HasSurgicalBleeding ||
		(facts.INR > 4.0 && facts.HasActiveBleeding) {
		return StateActiveBleed
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 5: LOW_OUTPUT_FAILURE
	// CI <2.0, ScvO2 <60%, Inotrope escalation, Combined AKI + ALF
	// ─────────────────────────────────────────────────────────────────────────
	if facts.CardiacIndex < 2.0 || facts.ScvO2 < 60 || facts.OnInotropeEscalation ||
		(facts.HasAKI && facts.HasALF) {
		return StateLowOutputFailure
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 6: NONE - No dominance state active
	// ─────────────────────────────────────────────────────────────────────────
	return StateNone
}

// ═══════════════════════════════════════════════════════════════════════════════
// STATE-SPECIFIC EVALUATORS
// ═══════════════════════════════════════════════════════════════════════════════

func (e *DominanceEngine) evaluateNeurologicDominance(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts) (*DominanceResult, error) {
	result := &DominanceResult{
		CurrentState: StateNeurologicCollapse,
		Vetoed:       false,
		Confidence:   e.config.DefaultConfidence,
	}

	// In neurologic collapse, veto EVERYTHING except:
	// - Immediate airway management
	// - Seizure control
	// - ICP management
	// - Code Blue response
	switch action.Type {
	case contracts.ActionDischarge, contracts.ActionTransfer:
		result.Vetoed = true
		result.VetoReason = "Patient in neurologic collapse - no transfers permitted"
		result.TriggeringRule = "NEURO_COLLAPSE_NO_TRANSFER"
		result.MustNotify = []string{"Attending", "Neurology", "ICU_Team"}

	case contracts.ActionMedicationOrder:
		// Allow only specific neurologic medications
		if !e.isNeuroEmergencyMedication(action) {
			result.Vetoed = true
			result.VetoReason = "Non-emergent medications held during neurologic collapse"
			result.TriggeringRule = "NEURO_COLLAPSE_MED_HOLD"
		}

	case contracts.ActionProcedureStart:
		// Allow only neuro-emergent procedures
		if !e.isNeuroEmergencyProcedure(action) {
			result.Vetoed = true
			result.VetoReason = "Non-emergent procedures deferred during neurologic collapse"
			result.TriggeringRule = "NEURO_COLLAPSE_PROC_DEFER"
		}
	}

	return result, nil
}

func (e *DominanceEngine) evaluateShockDominance(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts) (*DominanceResult, error) {
	result := &DominanceResult{
		CurrentState: StateShock,
		Vetoed:       false,
		Confidence:   e.config.DefaultConfidence,
	}

	switch action.Type {
	case contracts.ActionDischarge:
		result.Vetoed = true
		result.VetoReason = "Patient in shock - discharge not permitted"
		result.TriggeringRule = "SHOCK_NO_DISCHARGE"
		result.MustNotify = []string{"Attending", "ICU_Team"}

	case contracts.ActionMedicationOrder:
		// Hold nephrotoxic, hepatotoxic medications in shock
		if e.isContraindicatedInShock(action) {
			result.Vetoed = true
			result.VetoReason = "Medication contraindicated in shock state"
			result.TriggeringRule = "SHOCK_MED_CONTRAINDICATION"
		}

	case contracts.ActionProcedureStart:
		// Defer elective procedures
		if action.Urgency < 7 {
			result.Vetoed = true
			result.VetoReason = "Non-urgent procedures deferred during shock"
			result.TriggeringRule = "SHOCK_PROC_DEFER"
		}
	}

	return result, nil
}

func (e *DominanceEngine) evaluateHypoxiaDominance(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts) (*DominanceResult, error) {
	result := &DominanceResult{
		CurrentState: StateHypoxia,
		Vetoed:       false,
		Confidence:   e.config.DefaultConfidence,
	}

	switch action.Type {
	case contracts.ActionDischarge:
		result.Vetoed = true
		result.VetoReason = "Patient hypoxic - discharge not permitted"
		result.TriggeringRule = "HYPOXIA_NO_DISCHARGE"

	case contracts.ActionMedicationOrder:
		// Hold respiratory depressants
		if e.isRespiratoryDepressant(action) {
			result.Vetoed = true
			result.VetoReason = "Respiratory depressant held during hypoxia"
			result.TriggeringRule = "HYPOXIA_RESP_DEPRESSANT_HOLD"
		}

	case contracts.ActionProcedureStart:
		// Defer non-airway procedures
		if !e.isAirwayProcedure(action) && action.Urgency < 8 {
			result.Vetoed = true
			result.VetoReason = "Non-airway procedures deferred during hypoxia"
			result.TriggeringRule = "HYPOXIA_PROC_DEFER"
		}
	}

	return result, nil
}

func (e *DominanceEngine) evaluateActiveBleedDominance(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts) (*DominanceResult, error) {
	result := &DominanceResult{
		CurrentState: StateActiveBleed,
		Vetoed:       false,
		Confidence:   e.config.DefaultConfidence,
	}

	switch action.Type {
	case contracts.ActionDischarge:
		result.Vetoed = true
		result.VetoReason = "Active bleeding - discharge not permitted"
		result.TriggeringRule = "BLEED_NO_DISCHARGE"

	case contracts.ActionMedicationOrder:
		// Hold anticoagulants, antiplatelets
		if e.isAnticoagulant(action) {
			result.Vetoed = true
			result.VetoReason = "Anticoagulant contraindicated during active bleeding"
			result.TriggeringRule = "BLEED_ANTICOAG_HOLD"
			result.MustNotify = []string{"Attending", "Pharmacy", "Blood_Bank"}
		}
	}

	return result, nil
}

func (e *DominanceEngine) evaluateLowOutputDominance(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts) (*DominanceResult, error) {
	result := &DominanceResult{
		CurrentState: StateLowOutputFailure,
		Vetoed:       false,
		Confidence:   e.config.DefaultConfidence,
	}

	switch action.Type {
	case contracts.ActionDischarge:
		result.Vetoed = true
		result.VetoReason = "Low cardiac output - discharge not permitted"
		result.TriggeringRule = "LOW_OUTPUT_NO_DISCHARGE"

	case contracts.ActionMedicationOrder:
		// Hold negative inotropes
		if e.isNegativeInotrope(action) {
			result.Vetoed = true
			result.VetoReason = "Negative inotrope contraindicated in low output state"
			result.TriggeringRule = "LOW_OUTPUT_NEG_INOTROPE_HOLD"
		}
	}

	return result, nil
}

func (e *DominanceEngine) evaluateAbsoluteContraindicationsOnly(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts) (*DominanceResult, error) {
	// Outside ICU context - only block absolute contraindications
	result := &DominanceResult{
		CurrentState: StateNone,
		Vetoed:       false,
		Confidence:   e.config.DefaultConfidence,
	}

	// Example: Block dangerous drug-drug interactions regardless of context
	// This would integrate with KB-5 Drug Interactions in production

	return result, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// MEDICATION CLASSIFICATION HELPERS (Skeletal - to be expanded)
// ═══════════════════════════════════════════════════════════════════════════════

func (e *DominanceEngine) isNeuroEmergencyMedication(action contracts.ProposedAction) bool {
	// TODO: Integrate with KB-1 Drug Rules for medication classification
	// Emergency neuro meds: anticonvulsants, mannitol, hypertonic saline
	return false
}

func (e *DominanceEngine) isNeuroEmergencyProcedure(action contracts.ProposedAction) bool {
	// TODO: Integrate with KB-3 Guidelines for procedure classification
	// Emergency neuro procedures: intubation, EVD, decompressive craniectomy
	return false
}

func (e *DominanceEngine) isContraindicatedInShock(action contracts.ProposedAction) bool {
	// TODO: Integrate with KB-4 Patient Safety
	// Contraindicated in shock: nephrotoxics, high-dose diuretics without monitoring
	return false
}

func (e *DominanceEngine) isRespiratoryDepressant(action contracts.ProposedAction) bool {
	// TODO: Integrate with KB-1 Drug Rules
	// Respiratory depressants: opioids, benzodiazepines, barbiturates
	return false
}

func (e *DominanceEngine) isAirwayProcedure(action contracts.ProposedAction) bool {
	// Airway procedures: intubation, tracheostomy, bronchoscopy
	return false
}

func (e *DominanceEngine) isAnticoagulant(action contracts.ProposedAction) bool {
	// TODO: Integrate with KB-1 Drug Rules
	// Anticoagulants: heparin, enoxaparin, warfarin, DOACs
	return false
}

func (e *DominanceEngine) isNegativeInotrope(action contracts.ProposedAction) bool {
	// TODO: Integrate with KB-1 Drug Rules
	// Negative inotropes: beta-blockers, calcium channel blockers
	return false
}

// ═══════════════════════════════════════════════════════════════════════════════
// VETO CONTRACT IMPLEMENTATION
// ═══════════════════════════════════════════════════════════════════════════════

// Ensure DominanceEngine implements VetoContract
var _ contracts.VetoContract = (*DominanceEngine)(nil)

// CanICUVeto implements contracts.VetoContract
func (e *DominanceEngine) CanICUVeto(actionType contracts.ActionType) bool {
	// Per CTO/CMO directive: ICU can veto everything except reality
	return true
}

// CanKB19Recommend implements contracts.VetoContract
func (e *DominanceEngine) CanKB19Recommend(state contracts.DominanceState) bool {
	// KB-19 can ALWAYS recommend - but recommendations may be ignored
	return true
}

// MustDeferToICU implements contracts.VetoContract
func (e *DominanceEngine) MustDeferToICU(action contracts.ProposedAction, state contracts.DominanceState) bool {
	// All high-risk actions must defer to ICU in active dominance states
	if state != contracts.StateNone {
		return contracts.IsHighRiskAction(action.Type)
	}
	return false
}

// EvaluateVeto implements contracts.VetoContract
func (e *DominanceEngine) EvaluateVeto(ctx context.Context, action contracts.ProposedAction, state contracts.DominanceState) (*contracts.VetoResult, error) {
	// Convert contracts.DominanceState to icu.DominanceState
	icuState := DominanceState(state)

	// Create minimal SafetyFacts from action context
	// In production, this would be populated from patient monitoring
	facts := &SafetyFacts{
		PatientID:   action.PatientID,
		EncounterID: action.EncounterID,
		Timestamp:   time.Now(),
		IsInICU:     true, // Assume ICU context if veto evaluation requested
	}

	result, err := e.evaluateForState(ctx, action, facts, icuState)
	if err != nil {
		return nil, err
	}

	// Convert to contract VetoResult
	return &contracts.VetoResult{
		Vetoed:              result.Vetoed,
		Reason:              result.VetoReason,
		DominanceState:      contracts.DominanceState(result.CurrentState),
		TriggeringRule:      result.TriggeringRule,
		AllowedAlternatives: nil, // TODO: Populate alternatives
		MustNotify:          result.MustNotify,
		Confidence:          result.Confidence,
	}, nil
}

// RecordOverride implements contracts.VetoContract
func (e *DominanceEngine) RecordOverride(ctx context.Context, override contracts.OverrideRecord) error {
	if !e.config.AuditEnabled {
		return nil
	}
	// TODO: Persist override record to audit log
	// This would integrate with observability/audit infrastructure
	return nil
}

// evaluateForState evaluates dominance for a specific state (used by VetoContract)
func (e *DominanceEngine) evaluateForState(ctx context.Context, action contracts.ProposedAction, facts *SafetyFacts, state DominanceState) (*DominanceResult, error) {
	switch state {
	case StateNeurologicCollapse:
		return e.evaluateNeurologicDominance(ctx, action, facts)
	case StateShock:
		return e.evaluateShockDominance(ctx, action, facts)
	case StateHypoxia:
		return e.evaluateHypoxiaDominance(ctx, action, facts)
	case StateActiveBleed:
		return e.evaluateActiveBleedDominance(ctx, action, facts)
	case StateLowOutputFailure:
		return e.evaluateLowOutputDominance(ctx, action, facts)
	default:
		return &DominanceResult{
			CurrentState: StateNone,
			Vetoed:       false,
			Confidence:   e.config.DefaultConfidence,
		}, nil
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// DOMINANCE RESULT
// ═══════════════════════════════════════════════════════════════════════════════

// DominanceResult represents the outcome of an ICU dominance evaluation.
type DominanceResult struct {
	// CurrentState is the active dominance state
	CurrentState DominanceState `json:"current_state"`

	// Vetoed indicates if the proposed action is blocked
	Vetoed bool `json:"vetoed"`

	// VetoReason explains why the action was blocked
	VetoReason string `json:"veto_reason,omitempty"`

	// TriggeringRule identifies which safety rule triggered the veto
	TriggeringRule string `json:"triggering_rule,omitempty"`

	// OverriddenRecommendations lists what KB-19 recommendations are ignored
	OverriddenRecommendations []string `json:"overridden_recommendations,omitempty"`

	// MustNotify lists roles that must be notified of this decision
	MustNotify []string `json:"must_notify,omitempty"`

	// Confidence is the classifier's confidence in this result (0.0-1.0)
	Confidence float64 `json:"confidence"`
}
