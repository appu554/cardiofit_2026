package services

import (
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/models"
)

// MCUGateManager evaluates the MCU_GATE decision table for a given template,
// confidence tier, and patient context. It implements V-06 stress
// hyperglycaemia rules and N-05 dose_adjustment_notes enforcement.
type MCUGateManager struct {
	cfg *config.Config
	log *zap.Logger
}

// NewMCUGateManager creates an MCUGateManager with the given configuration
// and logger.
func NewMCUGateManager(cfg *config.Config, log *zap.Logger) *MCUGateManager {
	return &MCUGateManager{cfg: cfg, log: log}
}

// PatientContext holds clinical context from KB-20 for gate evaluation.
// Phase 8 P8-2: extended with the full field set required by Phase 7
// card-generation code paths. The original 10 fields (P7-era) are
// preserved at the top of the struct so existing consumers continue
// to compile without changes; new Phase 8 fields are grouped below.
type PatientContext struct {
	// ── P7-era core fields (10) ──
	PatientID              string   `json:"patient_id"`
	Stratum                string   `json:"stratum"`
	Medications            []string `json:"medications"`
	EGFRValue              float64  `json:"egfr_value"`
	LatestHbA1c            float64  `json:"latest_hba1c"`
	LatestFBG              float64  `json:"latest_fbg"`
	IsAcuteIll             bool     `json:"is_acute_illness"`
	HasRecentTransfusion   bool     `json:"has_recent_transfusion"`
	HasRecentHypoglycaemia bool     `json:"has_recent_hypoglycaemia"`
	WeightKg               float64  `json:"weight_kg"`

	// ── P8-2: Demographics ──
	// Age + Sex enable the CKM classifier's age-based risk stratification
	// and sex-specific BP / renal thresholds. BMI enables the waist-risk
	// branch of the lifestyle-intervention pathway.
	Age int    `json:"age,omitempty"`
	Sex string `json:"sex,omitempty"`
	BMI float64 `json:"bmi,omitempty"`

	// ── P8-2: CKM stage + substage metadata ──
	// CKMStageV2 carries the full substage string ("0", "1", "2", "3",
	// "4a", "4b", "4c"). The metadata block carries HF subtype + LVEF +
	// ASCVD events + subclinical markers needed by the P7-B 4c pathway
	// routing and future Phase 8 sub-stage-specific card templates.
	CKMStageV2          string             `json:"ckm_stage_v2,omitempty"`
	CKMSubstageMetadata *CKMSubstageMeta   `json:"ckm_substage_metadata,omitempty"`

	// ── P8-2: Extended labs ──
	// LatestPotassium: needed by the MRA / finerenone hyperkalaemia
	// guard and the RAAS creatinine-monitoring window. EGFR stays on
	// the top-level EGFRValue field (already present from P7).
	LatestPotassium float64 `json:"latest_potassium,omitempty"`

	// ── P8-2: Engagement / adherence context ──
	// Used by the P6 adherence-gain factor path and the Phase 8 inertia
	// detector's non-adherence exclusion branch (a patient flagged
	// DISENGAGED should not trigger inertia cards — the target gap is
	// driven by adherence, not clinical drift).
	EngagementComposite *float64 `json:"engagement_composite,omitempty"`
	EngagementStatus    string   `json:"engagement_status,omitempty"`

	// ── P8-2: CGM status ──
	// Fed by a cross-service fetch to KB-26's cgm_period_reports via
	// the P7-E Milestone 2 cgm-latest endpoint. When the patient has
	// no CGM data, HasCGM=false and the downstream inertia / glycaemic
	// detectors fall back to HbA1c. When HasCGM=true, LatestTIR and
	// LatestGRIZone are the freshest 14-day values from the Flink
	// pipeline.
	HasCGM            bool       `json:"has_cgm,omitempty"`
	LatestCGMTIR      *float64   `json:"latest_cgm_tir,omitempty"`
	LatestCGMGRIZone  string     `json:"latest_cgm_gri_zone,omitempty"`
	CGMReportAt       *time.Time `json:"cgm_report_at,omitempty"`
}

// CKMSubstageMeta mirrors kb-20-patient-profile/internal/models.SubstageMetadata
// on the KB-23 side so the JSON wire payload from the summary-context
// endpoint deserializes directly into this struct. Field tags must
// match the KB-20 struct tags exactly — the integration test pins
// this contract so drift is caught at CI time. Phase 8 P8-2.
type CKMSubstageMeta struct {
	// Stage 4c — Heart Failure
	HFClassification string   `json:"hf_type,omitempty"`
	LVEFPercent      *float64 `json:"lvef_pct,omitempty"`
	NYHAClass        string   `json:"nyha_class,omitempty"`
	NTproBNP         *float64 `json:"nt_probnp,omitempty"`
	BNP              *float64 `json:"bnp,omitempty"`
	HFEtiology       string   `json:"hf_etiology,omitempty"`

	// Stage 4a — Subclinical CVD
	CACScore       *float64 `json:"cac_score,omitempty"`
	CIMTPercentile *int     `json:"cimt_percentile,omitempty"`
	HasLVH         bool     `json:"has_lvh,omitempty"`
}

// EvaluateGate determines the MCU_GATE from the template's gate rules and
// patient context. It returns the gate value, a human-readable rationale,
// and any dose adjustment notes (N-05).
//
// Evaluation order:
//  1. Template gate rules are evaluated in declaration order (first match wins).
//  2. V-06: Stress hyperglycaemia override escalates gate to at least PAUSE
//     when acute illness is detected.
//  3. N-05: MODIFY gate requires non-empty dose_adjustment_notes.
func (m *MCUGateManager) EvaluateGate(tmpl *models.CardTemplate, tier models.ConfidenceTier, patientCtx *PatientContext) (models.MCUGate, string, string) {
	// Start with template default
	gate := tmpl.MCUGateDefault
	rationale := "template default gate"
	adjustmentNotes := ""

	// Evaluate template gate rules in order (first matching rule wins)
	for _, rule := range tmpl.GateRules {
		if m.evaluateCondition(rule.Condition, tier, patientCtx, tmpl) {
			gate = rule.Gate
			rationale = rule.Rationale
			adjustmentNotes = rule.AdjustmentNotes
			m.log.Debug("gate rule matched",
				zap.String("condition", rule.Condition),
				zap.String("gate", string(gate)),
			)
			break
		}
	}

	// V-06: Stress hyperglycaemia override -- if acute illness detected,
	// gate must be at least PAUSE to prevent medication intensification
	if patientCtx != nil && patientCtx.IsAcuteIll {
		if gate.Level() < models.GatePause.Level() {
			gate = models.GatePause
			rationale = "V-06: stress hyperglycaemia -- acute illness, medication intensification paused"
			adjustmentNotes = "STRESS_HYPERGLYCAEMIA: do not intensify. Re-evaluate after illness resolution."
			m.log.Info("V-06 stress hyperglycaemia override applied",
				zap.String("patient_id", patientCtx.PatientID),
			)
		}
	}

	// N-05: MODIFY gate requires non-null dose_adjustment_notes
	if gate == models.GateModify && adjustmentNotes == "" {
		adjustmentNotes = "MODIFY_GATE: titration adjustment required -- see recommendations"
	}

	return gate, rationale, adjustmentNotes
}

// evaluateCondition checks a gate rule condition against the current state.
// Conditions are simple string tokens authored within template YAML files.
// The template is provided so that differential-specific conditions (e.g.
// ACS_STEMI, ACS_NSTEMI) can match on the template's DifferentialID.
func (m *MCUGateManager) evaluateCondition(condition string, tier models.ConfidenceTier, ctx *PatientContext, tmpl *models.CardTemplate) bool {
	switch condition {
	// --- Tier-based conditions ---
	case "TIER_FIRM":
		return tier == models.TierFirm
	case "TIER_PROBABLE":
		return tier == models.TierProbable
	case "TIER_POSSIBLE":
		return tier == models.TierPossible
	case "TIER_UNCERTAIN":
		return tier == models.TierUncertain

	// --- Patient-context conditions ---
	case "ACUTE_ILLNESS":
		return ctx != nil && ctx.IsAcuteIll
	case "RECENT_TRANSFUSION":
		return ctx != nil && ctx.HasRecentTransfusion
	case "EGFR_LOW":
		return ctx != nil && ctx.EGFRValue > 0 && ctx.EGFRValue < 30

	// --- ACS / cardiology gate conditions ---
	case "stemi_confirmed":
		// STEMI pathway: firm confidence with confirmed STEMI differential
		return tier == models.TierFirm && tmpl != nil && tmpl.DifferentialID == "ACS_STEMI"
	case "nstemi_high_risk":
		// NSTEMI high-risk: firm or probable confidence
		return (tier == models.TierFirm || tier == models.TierProbable) &&
			tmpl != nil && tmpl.DifferentialID == "ACS_NSTEMI"
	case "nstemi_haemodynamic_instability":
		// NSTEMI with haemodynamic compromise
		return ctx != nil && ctx.IsAcuteIll &&
			tmpl != nil && tmpl.DifferentialID == "ACS_NSTEMI"
	case "nstemi_resolved_stable":
		// NSTEMI stabilised: firm confidence, no acute illness
		return tier == models.TierFirm &&
			ctx != nil && !ctx.IsAcuteIll &&
			tmpl != nil && tmpl.DifferentialID == "ACS_NSTEMI"
	case "post_reperfusion_stable_48h":
		// General post-intervention stability: firm confidence, no acute illness
		return tier == models.TierFirm && ctx != nil && !ctx.IsAcuteIll
	case "cardiogenic_shock":
		// Acute illness proxy for cardiogenic shock (vital-sign integration pending)
		return ctx != nil && ctx.IsAcuteIll && ctx.LatestFBG > 0

	// --- Glycaemic gate conditions ---
	case "hba1c_high":
		return ctx != nil && ctx.LatestHbA1c > 9.0
	case "hypoglycaemia_recent":
		return ctx != nil && ctx.HasRecentHypoglycaemia

	// --- Catch-all ---
	case "ALWAYS":
		return true
	default:
		m.log.Debug("unknown gate condition", zap.String("condition", condition))
		return false
	}
}
