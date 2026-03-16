// Package channel_c implements the ProtocolGuard (SA-03).
//
// Rules are pre-compiled from protocol_rules.yaml at startup.
// Zero network calls during evaluation — all data comes from TitrationContext.
package channel_c

// TitrationContext provides the data needed for protocol rule evaluation.
// Assembled by the V-MCU orchestrator from the local safety cache.
type TitrationContext struct {
	// From local safety cache (KB-20 raw data, refreshed hourly)
	EGFR              float64  // mL/min/1.73m²
	ActiveMedications []string // drug class list (e.g., "METFORMIN", "SGLT2I")

	// Derived from Channel B evaluation (passed by orchestrator)
	AKIDetected         bool // creatinine delta triggered B-03
	ActiveHypoglycaemia bool // glucose triggered B-01

	// From dose history (V-MCU internal)
	HypoglycaemiaWithin7d bool // any hypo event in last 7 days

	// From proposed titration action
	ProposedAction   string  // "dose_increase" | "dose_decrease" | "dose_hold"
	DoseDeltaPercent float64 // absolute % change proposed

	// ── HTN co-management fields (Wave 1 P0) ──
	// Pre-computed boolean composites (set by orchestrator from cache data).
	// Each maps to one or more PG-08..PG-14 rule conditions.

	// PG-08: ACEi/ARB + K+ ≥5.5 + declining eGFR → HALT uptitration
	ACEiARBHyperKDecliningEGFR bool

	// PG-08-DUAL-RAAS: ACEi AND ARB simultaneously (contraindicated per ONTARGET 2008)
	DualRAASActive bool

	// PG-09: Beta-blocker + active insulin therapy → MODIFY (mask hypo)
	BetaBlockerInsulinActive bool

	// PG-10: ≥3 antihypertensives at max + uncontrolled → PAUSE (resistant HTN)
	ResistantHTNDetected bool

	// PG-11: Thiazide + Na+ <130 mEq/L → HALT (hyponatraemia risk)
	ThiazideHyponatraemia bool

	// PG-12: MRA + K+ >5.0 + eGFR <45 → MODIFY (MRA dose cap)
	MRAHyperKLowEGFR bool

	// PG-13: CCB + SBP <110 + recent dose increase → MODIFY (excessive response)
	CCBExcessiveResponse bool

	// PG-14: RAAS creatinine tolerance (ACEi/ARB within 14d + Cr rise <30% + K+ <5.5 + no oliguria)
	RAASCreatinineTolerant bool

	// Numeric values for threshold comparisons in PG-08..PG-14
	PotassiumCurrent  float64 // mEq/L (from cache)
	SBPCurrent        float64 // mmHg (from cache)
	SodiumCurrent     float64 // mEq/L (from cache)
	CreatinineRisePct float64 // % rise from baseline (for PG-14 monitoring)

	// ── HTN co-management fields (Wave 2 P1) ──

	// PG-15: ACEi-induced cough probability (KB-22 posterior from p_acei_cough node)
	// When > 0.70, Channel C issues MODIFY to recommend ARB switch.
	ACEiInducedCoughProbability float64

	// PG-16: AF confirmed (from B-16 sentinel) but no anticoagulation documented
	// When true, Channel C issues PAUSE until anticoagulation is reviewed.
	AFConfirmedNoAnticoagulation bool

	// CKD Stage 4 deprescribing hard block (AD-09)
	CKDStage4DeprescribingBlocked bool // true = eGFR <30 + attempting to deprescribe renoprotective agent

	// ACR-based RAAS escalation (PG-17)
	ACRA3NoRAAS bool   // PG-17-A3: ACR category A3 AND NOT on ACEi/ARB
	ACRA2NoRAAS bool   // PG-17-A2: ACR category A2 AND NOT on ACEi/ARB
	ACRCategory string // A1 | A2 | A3 (for audit trail)
	ACRTrend    string // IMPROVING | STABLE | WORSENING (for audit trail)
}

// ProtocolGate is Channel C's local gate type.
type ProtocolGate string

const (
	ProtoClear  ProtocolGate = "CLEAR"
	ProtoModify ProtocolGate = "MODIFY"
	ProtoPause  ProtocolGate = "PAUSE"
	ProtoHalt   ProtocolGate = "HALT"
)

// ProtocolResult is the output of Channel C evaluation.
type ProtocolResult struct {
	Gate              ProtocolGate `json:"gate"`
	RuleID            string       `json:"rule_id,omitempty"`
	RuleVersion       string       `json:"rule_version"`
	GuidelineRef      string       `json:"guideline_ref,omitempty"`
	SafetyInstruction string       `json:"safety_instruction,omitempty"` // human-readable constraint for MODIFY gate
}
