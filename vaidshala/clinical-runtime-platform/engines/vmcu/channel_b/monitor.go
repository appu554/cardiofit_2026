package channel_b

import (
	"math"
	"time"
)

// PhysiologySafetyMonitor implements SA-02: raw lab threshold rules.
// 23 rules total: 18 clinical thresholds + 5 data anomaly checks.
//
// Rule evaluation order (most severe first):
//  1. Data anomaly checks (DA-01 through DA-05) → HOLD_DATA
//  2. Critical thresholds (B-01, B-03, B-04, B-05, B-08) → HALT
//  3. Warning thresholds (B-02, B-09, B-10, B-06, B-07) → PAUSE
//  4. No rule fired → CLEAR
//
// B-08, B-09, and B-10 are POLICY EXTENSIONS (eGFR-based, drug-independent).
// See checkB08(), checkB09(), and checkB10() for clinical rationale documentation.
type PhysiologySafetyMonitor struct {
	cfg PhysioConfig
}

// PhysioConfig holds configurable thresholds for Channel B rules.
type PhysioConfig struct {
	// Clinical thresholds
	GlucoseHaltThreshold  float64 // B-01: mmol/L (default 3.9)
	GlucosePauseThreshold float64 // B-02: mmol/L (default 4.5)
	CreatinineDeltaHalt   float64 // B-03: µmol/L in 48h (default 26)
	PotassiumLowHalt      float64 // B-04: mEq/L (default 3.0)
	PotassiumHighHalt     float64 // B-04: mEq/L (default 6.0)
	SBPHaltThreshold      float64 // B-05: mmHg (default 90)
	WeightDeltaPause      float64 // B-06: kg in 72h (default 2.5)
	GlucoseTrendThreshold float64 // B-07: mmol/L (default 5.5)

	// eGFR thresholds (policy extension — see B-08/B-09/B-10 clinical rationale)
	EGFRHaltThreshold     float64 // B-08: mL/min/1.73m² (default 15, CKD Stage 5)
	EGFRPauseThreshold    float64 // B-09: mL/min/1.73m² (default 30, CKD Stage 4)
	EGFRSlopeRapidDecline float64 // B-10: mL/min/1.73m²/year (default -5.0, rapid decline)

	// J-curve eGFR-stratified SBP lower limits (B-12)
	// Below these floors, antihypertensive dose reduction is needed to protect renal perfusion.
	SBPFloorStage3a float64 // B-12: mmHg (default 120, CKD 3a: eGFR 45-59) — PAUSE floor
	SBPFloorStage3b float64 // B-12: mmHg (default 125, CKD 3b: eGFR 30-44) — PAUSE floor
	SBPFloorStage4  float64 // B-12: mmHg (default 130, CKD 4: eGFR 15-29) — PAUSE floor

	// Amendment 8: Stage-specific HALT/PAUSE thresholds (replaces unified SBPHaltFloorStage4)
	// These are the hard lower limits where perfusion concern triggers PAUSE or HALT
	// depending on CKD stage-specific autoregulatory reserve.
	SBPHaltStage3a float64 // B-12: mmHg PAUSE threshold (default 100, CKD 3a — autoregulation partially intact)
	SBPHaltStage3b float64 // B-12: mmHg PAUSE threshold (default 105, CKD 3b — reduced reserve)
	SBPHaltStage4  float64 // B-12: mmHg HALT threshold (default 100, CKD 4 — perfusion danger)
	SBPPauseStage4 float64 // B-12: mmHg PAUSE threshold (default 110, CKD 4 — cautionary zone)

	// Heart rate thresholds (B-13, B-14, B-15)
	HRBradycardiaHalt  float64 // B-13: bpm (default 45) — confirmed resting HR
	HRBradycardiaPause float64 // B-14: bpm (default 55) — beta-blocker + dose change
	HRTachycardiaPause float64 // B-15: bpm (default 120) — confirmed resting HR

	// Hyponatraemia thresholds (B-17, B-18, B-19)
	SodiumHaltThreshold     float64 // B-17: mEq/L (default 132) — thiazide + Na+ < 132
	SodiumPauseThreshold    float64 // B-18: mEq/L (default 135) — thiazide + Na+ 132-135
	SodiumSeasonalThreshold float64 // B-19: mEq/L (default 135) — seasonal amplification

	// Glucose variability threshold (B-20)
	GlucoseCVPauseThreshold float64 // B-20: CV% above this triggers PAUSE (default 36.0)

	// Data anomaly thresholds (SA-05)
	EGFRDeltaHoldData       float64 // DA-01: % in 48h (default 40)
	CreatinineDeltaHoldData float64 // DA-02: % in 48h (default 100)
	GlucoseFloorHoldData    float64 // DA-03: mmol/L (default 1.0)
	HbA1cDeltaHoldData      float64 // DA-04: % in 30d (default 2.0)
	PotassiumCeilingHold    float64 // DA-05: mEq/L (default 8.0)

	// Staleness thresholds (DA-06, DA-07)
	PotassiumStaleDays  float64 // DA-06: days (default 14)
	CreatinineStaleDays float64 // DA-07: days (default 30)
}

// DefaultPhysioConfig returns production-safe threshold defaults.
func DefaultPhysioConfig() PhysioConfig {
	return PhysioConfig{
		GlucoseHaltThreshold:    3.9,
		GlucosePauseThreshold:   4.5,
		CreatinineDeltaHalt:     26.0,
		PotassiumLowHalt:        3.0,
		PotassiumHighHalt:       6.0,
		SBPHaltThreshold:        90.0,
		EGFRHaltThreshold:       15.0,
		EGFRPauseThreshold:      30.0,
		EGFRSlopeRapidDecline:   -5.0,
		SBPFloorStage3a:         120.0,
		SBPFloorStage3b:         125.0,
		SBPFloorStage4:          130.0,
		SBPHaltStage3a:          100.0,
		SBPHaltStage3b:          105.0,
		SBPHaltStage4:           100.0,
		SBPPauseStage4:          110.0,
		HRBradycardiaHalt:       45.0,
		HRBradycardiaPause:      55.0,
		HRTachycardiaPause:      120.0,
		SodiumHaltThreshold:     132.0,
		SodiumPauseThreshold:    135.0,
		SodiumSeasonalThreshold: 135.0,
		GlucoseCVPauseThreshold: 36.0,
		WeightDeltaPause:        2.5,
		GlucoseTrendThreshold:   5.5,
		EGFRDeltaHoldData:       40.0,
		CreatinineDeltaHoldData: 100.0,
		GlucoseFloorHoldData:    1.0,
		HbA1cDeltaHoldData:      2.0,
		PotassiumCeilingHold:    8.0,
		PotassiumStaleDays:      14.0,
		CreatinineStaleDays:     30.0,
	}
}

// NewPhysiologySafetyMonitor creates a monitor with the given config.
func NewPhysiologySafetyMonitor(cfg PhysioConfig) *PhysiologySafetyMonitor {
	return &PhysiologySafetyMonitor{cfg: cfg}
}

// EvaluateOptions controls Channel B behavior for special operating modes.
type EvaluateOptions struct {
	// DeprescribingActive widens glucose thresholds during controlled dose reduction.
	// B-01 HALT: 3.9 → 3.5 mmol/L, B-02 PAUSE: 4.5 → 3.9 mmol/L.
	// This reflects clinical tolerance for slightly lower glucose during
	// deliberate medication tapering, while keeping absolute safety floors.
	DeprescribingActive bool
}

// EvaluateWithOptions runs Channel B rules with mode-specific threshold adjustments.
func (m *PhysiologySafetyMonitor) EvaluateWithOptions(data *RawPatientData, opts EvaluateOptions) PhysioResult {
	if opts.DeprescribingActive {
		// Create a monitor copy with widened glucose thresholds
		widened := *m
		widened.cfg.GlucoseHaltThreshold = 3.5  // 3.9 → 3.5 mmol/L
		widened.cfg.GlucosePauseThreshold = 3.9 // 4.5 → 3.9 mmol/L
		return widened.Evaluate(data)
	}
	return m.Evaluate(data)
}

// Evaluate runs all Channel B rules against raw inputs.
// Returns the most restrictive gate signal from the first matching rule.
//
// MUST NOT make any network calls. All data comes from RawPatientData.
func (m *PhysiologySafetyMonitor) Evaluate(data *RawPatientData) PhysioResult {
	// Build rawVals map, only including non-nil values.
	rawVals := make(map[string]float64, 7)
	if data.GlucoseCurrent != nil {
		rawVals["glucose_mmol"] = *data.GlucoseCurrent
	}
	if data.CreatinineCurrent != nil {
		rawVals["creatinine_umol"] = *data.CreatinineCurrent
	}
	if data.PotassiumCurrent != nil {
		rawVals["potassium_meq"] = *data.PotassiumCurrent
	}
	if data.SBPCurrent != nil {
		rawVals["sbp_mmhg"] = *data.SBPCurrent
	}
	if data.WeightKgCurrent != nil {
		rawVals["weight_kg"] = *data.WeightKgCurrent
	}
	if data.EGFRCurrent != nil {
		rawVals["egfr"] = *data.EGFRCurrent
	}
	if data.HbA1cCurrent != nil {
		rawVals["hba1c_pct"] = *data.HbA1cCurrent
	}
	if data.SodiumCurrent != nil {
		rawVals["sodium_meq"] = *data.SodiumCurrent
	}
	if data.DBPCurrent != nil {
		rawVals["dbp_mmhg"] = *data.DBPCurrent
	}
	if data.HeartRateCurrent != nil {
		rawVals["hr_bpm"] = *data.HeartRateCurrent
	}

	// ── Phase 1: Data anomaly checks (SA-05) → HOLD_DATA ──
	// These fire BEFORE clinical thresholds because anomalous data
	// should not be used for clinical decisions at all.

	if r := m.checkDA01(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkDA02(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkDA03(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkDA04(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkDA05(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkDA06(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkDA07(data); r != nil {
		r.RawValues = rawVals
		return *r
	}

	// ── Phase 2: Critical thresholds → HALT ──

	// B-11 MUST fire before B-01: beta-blocker masks adrenergic hypoglycaemia
	// warning symptoms, so the raised threshold (4.5) catches danger earlier.
	if r := m.checkB11(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB01(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB03(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB04(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB05(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB08(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-12 can produce HALT (hard floor) or PAUSE (stratified floor)
	if r := m.checkB12(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-13: severe bradycardia (HR < 45 bpm, resting, confirmed)
	if r := m.checkB13(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-17: thiazide + severe hyponatraemia (Na+ < 132)
	if r := m.checkB17(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-21: finerenone + K+ ≥5.5 → HALT (hyperkalemia risk)
	if r := m.checkB21(data); r != nil {
		r.RawValues = rawVals
		return *r
	}

	// ── Phase 3: Warning thresholds → PAUSE ──

	if r := m.checkB02(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB09(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB10(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB06(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	if r := m.checkB07(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-14: beta-blocker bradycardia (HR < 55, beta-blocker active, dose change within 7d)
	if r := m.checkB14(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-15: resting tachycardia (HR > 120, resting, confirmed)
	if r := m.checkB15(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-16: irregular heart rhythm (confirmed) → PAUSE + KB22_TRIGGER
	if r := m.checkB16(data); r != nil {
		rawVals["kb22_trigger"] = 1 // sentinel for orchestrator to publish KB-22 HPI event
		r.RawValues = rawVals
		return *r
	}
	// B-16 was checked inline above; KB22Triggers populated by checkB16 are
	// attached to the result. For non-B-16 CLEAR results, no triggers exist.
	// B-18: thiazide + mild hyponatraemia (Na+ 132-135)
	if r := m.checkB18(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-19: seasonal hyponatraemia amplification (Na+ < 135 + SUMMER + thiazide)
	if r := m.checkB19(data); r != nil {
		r.RawValues = rawVals
		return *r
	}
	// B-20: glucose variability — CV% > 36% → PAUSE
	if r := m.checkB20(data); r != nil {
		r.RawValues = rawVals
		return *r
	}

	// ── No rule fired → CLEAR ──
	return PhysioResult{Gate: PhysioClear, RawValues: rawVals}
}

// ════════════════════════════════════════════════════════════════════════
// DATA ANOMALY RULES (DA-01 through DA-05) → HOLD_DATA
// ════════════════════════════════════════════════════════════════════════

// DA-01: eGFR change > 40% in 48h → likely lab error or AKI
func (m *PhysiologySafetyMonitor) checkDA01(d *RawPatientData) *PhysioResult {
	if d.EGFRCurrent == nil || d.EGFRPrior48h == nil || *d.EGFRPrior48h == 0 {
		return nil
	}
	pctChange := math.Abs(*d.EGFRCurrent-*d.EGFRPrior48h) / *d.EGFRPrior48h * 100
	if pctChange > m.cfg.EGFRDeltaHoldData {
		return &PhysioResult{
			Gate:       PhysioHoldData,
			RuleFired:  "DA-01",
			IsAnomaly:  true,
			AnomalyLab: "EGFR",
		}
	}
	return nil
}

// DA-02: Creatinine change > 100% in 48h (no clinical event) → suspected lab error
func (m *PhysiologySafetyMonitor) checkDA02(d *RawPatientData) *PhysioResult {
	if d.CreatinineCurrent == nil || d.Creatinine48hAgo == nil || *d.Creatinine48hAgo == 0 {
		return nil
	}
	pctChange := math.Abs(*d.CreatinineCurrent-*d.Creatinine48hAgo) / *d.Creatinine48hAgo * 100
	if pctChange > m.cfg.CreatinineDeltaHoldData {
		return &PhysioResult{
			Gate:       PhysioHoldData,
			RuleFired:  "DA-02",
			IsAnomaly:  true,
			AnomalyLab: "CREATININE",
		}
	}
	return nil
}

// DA-03: Glucose < 1.0 mmol/L → instrument calibration error
func (m *PhysiologySafetyMonitor) checkDA03(d *RawPatientData) *PhysioResult {
	if d.GlucoseCurrent == nil {
		return nil // absent data ≠ calibration error
	}
	if *d.GlucoseCurrent < m.cfg.GlucoseFloorHoldData {
		return &PhysioResult{
			Gate:       PhysioHoldData,
			RuleFired:  "DA-03",
			IsAnomaly:  true,
			AnomalyLab: "GLUCOSE",
		}
	}
	return nil
}

// DA-04: HbA1c change > 2.0% in 30d → biologically impossible
func (m *PhysiologySafetyMonitor) checkDA04(d *RawPatientData) *PhysioResult {
	if d.HbA1cCurrent == nil || d.HbA1cPrior30d == nil {
		return nil
	}
	delta := math.Abs(*d.HbA1cCurrent - *d.HbA1cPrior30d)
	if delta > m.cfg.HbA1cDeltaHoldData {
		return &PhysioResult{
			Gate:       PhysioHoldData,
			RuleFired:  "DA-04",
			IsAnomaly:  true,
			AnomalyLab: "HBA1C",
		}
	}
	return nil
}

// DA-05: Potassium > 8.0 mEq/L → extreme value, confirm first
func (m *PhysiologySafetyMonitor) checkDA05(d *RawPatientData) *PhysioResult {
	if d.PotassiumCurrent == nil {
		return nil // absent data ≠ extreme value
	}
	if *d.PotassiumCurrent > m.cfg.PotassiumCeilingHold {
		return &PhysioResult{
			Gate:       PhysioHoldData,
			RuleFired:  "DA-05",
			IsAnomaly:  true,
			AnomalyLab: "POTASSIUM",
		}
	}
	return nil
}

// DA-06: Potassium measurement >14 days old → stale safety data
//
// CLINICAL RATIONALE:
// Potassium is a critical safety parameter for cardiac arrhythmia risk.
// A K+ value older than 14 days cannot be relied upon for safe titration
// decisions — potassium levels can shift significantly with dietary changes,
// medication adjustments, or renal function changes. Channel B must signal
// HOLD_DATA to prevent titration on outdated safety data.
func (m *PhysiologySafetyMonitor) checkDA06(d *RawPatientData) *PhysioResult {
	if d.PotassiumCurrent == nil || d.PotassiumLastMeasuredAt == nil {
		return nil // no value or no timestamp — cannot assess staleness
	}
	if IsStale(d.PotassiumLastMeasuredAt, time.Duration(m.cfg.PotassiumStaleDays*24)*time.Hour) {
		return &PhysioResult{
			Gate:       PhysioHoldData,
			RuleFired:  "DA-06",
			IsAnomaly:  true,
			AnomalyLab: "POTASSIUM",
		}
	}
	return nil
}

// DA-07: Creatinine measurement >30 days old → stale renal data
//
// CLINICAL RATIONALE:
// Creatinine is the primary marker for renal function assessment.
// While slower-moving than potassium, a value older than 30 days is
// unreliable for dose adjustment decisions — AKI or CKD progression
// could have occurred in the interim. HOLD_DATA forces lab refresh
// before any titration proceeds.
func (m *PhysiologySafetyMonitor) checkDA07(d *RawPatientData) *PhysioResult {
	if d.CreatinineCurrent == nil || d.CreatinineLastMeasuredAt == nil {
		return nil // no value or no timestamp — cannot assess staleness
	}
	if IsStale(d.CreatinineLastMeasuredAt, time.Duration(m.cfg.CreatinineStaleDays*24)*time.Hour) {
		return &PhysioResult{
			Gate:       PhysioHoldData,
			RuleFired:  "DA-07",
			IsAnomaly:  true,
			AnomalyLab: "CREATININE",
		}
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════
// CLINICAL THRESHOLD RULES — HALT (B-01, B-03, B-04, B-05)
// ════════════════════════════════════════════════════════════════════════

// B-11: Beta-blocker + glucose < 4.5 mmol/L → HALT (raised threshold)
//
// CLINICAL RATIONALE:
// Beta-blockers block the adrenergic warning cascade (tachycardia, tremor,
// palpitations) that normally alerts patients to falling blood glucose.
// In a beta-blocked patient, the first symptom of hypoglycaemia is
// neuroglycopaenic (confusion, seizure, loss of consciousness) — by which
// point glucose is already dangerously low. The standard B-01 threshold
// of 3.9 mmol/L is therefore too late for beta-blocked patients.
//
// B-11 raises the HALT threshold to 4.5 mmol/L for the ENTIRE range
// below 4.5 (not just the 3.9–4.5 gap). If glucose is 3.2 and a beta-
// blocker is active, B-11 fires (in addition to B-01 which would also
// fire if evaluation continued). Because B-11 is checked before B-01
// in the evaluation chain, it is the dominant rule for beta-blocked
// patients at any glucose below 4.5.
func (m *PhysiologySafetyMonitor) checkB11(d *RawPatientData) *PhysioResult {
	if !d.BetaBlockerActive {
		return nil
	}
	if d.GlucoseCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	if *d.GlucoseCurrent < 4.5 {
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-11"}
	}
	return nil
}

// B-01: Glucose < 3.9 mmol/L → active hypoglycaemia
func (m *PhysiologySafetyMonitor) checkB01(d *RawPatientData) *PhysioResult {
	if d.GlucoseCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	if *d.GlucoseCurrent < m.cfg.GlucoseHaltThreshold {
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-01"}
	}
	return nil
}

// B-03: Creatinine 48h delta > 26 µmol/L → KDIGO AKI Stage 1
//
// RAAS TOLERANCE SUPPRESSION (PG-14 integration):
// After ACEi/ARB initiation or uptitration, a creatinine rise up to 30% is
// expected RAAS pharmacodynamics (reduced efferent arteriolar tone → lower
// filtration pressure → lower GFR). This is NOT AKI if:
//   - CreatinineRiseExplained = true (set by orchestrator from PG-14 evaluation)
//   - OliguriaReported = false (clinician has not reported reduced urine output)
//   - Potassium < 5.5 mEq/L (hyperkalaemia overrides tolerance)
//
// When suppressed: HALT is downgraded to PAUSE (still flags for monitoring,
// but does not freeze the titration cycle).
func (m *PhysiologySafetyMonitor) checkB03(d *RawPatientData) *PhysioResult {
	if d.CreatinineCurrent == nil || d.Creatinine48hAgo == nil {
		return nil
	}
	delta := *d.CreatinineCurrent - *d.Creatinine48hAgo
	if delta > m.cfg.CreatinineDeltaHalt {
		// Check RAAS tolerance suppression conditions.
		// CreatinineRiseExplained is the primary boolean (set by orchestrator from PG-14).
		// Amendment 2 fallback: if the boolean was not set but an ACEi/ARB perturbation
		// window is active, treat the creatinine rise as explained (defense-in-depth).
		raasExplained := d.CreatinineRiseExplained
		if !raasExplained {
			raasExplained = FindActivePerturbation(d.ActivePerturbations, DrugClassACEiARB, time.Now()) != nil
		}
		if raasExplained && !d.OliguriaReported {
			// Verify potassium is not dangerously elevated (overrides RAAS tolerance)
			kSafe := d.PotassiumCurrent == nil || *d.PotassiumCurrent < 5.5
			if kSafe {
				return &PhysioResult{Gate: PhysioPause, RuleFired: "B-03-RAAS-SUPPRESSED"}
			}
		}
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-03"}
	}
	return nil
}

// B-04: Potassium < 3.0 OR > 6.0 mEq/L → cardiac arrhythmia risk
//
// THIAZIDE PERTURBATION DAMPENING (Amendment 2, PG-11 context):
// During the first 3 weeks after thiazide initiation, K+ is expected to
// drop 0.3-0.5 mmol/L due to kaliuresis. A K+ reading in the range
// [2.5, 3.0) that would normally fire HALT is downgraded to PAUSE if:
//   - A thiazide perturbation window is active
//   - K+ >= 2.5 mEq/L (below 2.5 is dangerous regardless of cause)
//
// Hyperkalaemia (K+ > 6.0) is never dampened by perturbation.
func (m *PhysiologySafetyMonitor) checkB04(d *RawPatientData) *PhysioResult {
	if d.PotassiumCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	k := *d.PotassiumCurrent

	// Hyperkalaemia — never dampened
	if k > m.cfg.PotassiumHighHalt {
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-04"}
	}

	// Hypokalaemia
	if k < m.cfg.PotassiumLowHalt {
		// Thiazide perturbation dampening: HALT→PAUSE if K+ in [2.5, 3.0)
		// and thiazide was recently started (expected kaliuresis).
		if k >= 2.5 && FindActivePerturbation(d.ActivePerturbations, DrugClassThiazide, time.Now()) != nil {
			return &PhysioResult{Gate: PhysioPause, RuleFired: "B-04-THIAZIDE-PERTURBATION"}
		}
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-04"}
	}
	return nil
}

// B-05: SBP < 90 mmHg → haemodynamic instability
//
// MEASUREMENT UNCERTAINTY DAMPENING:
// When MeasurementUncertainty is HIGH (noisy reading from irregular HR,
// postural change, or single measurement), downgrade HALT→PAUSE.
// Rationale: a reading of SBP=88 with HIGH uncertainty might represent
// SBP=93 (within σ). Freezing titration on a noisy reading is worse
// than pausing for confirmation.
//
// WHITE-COAT BYPASS:
// When BPPattern is WHITE_COAT, elevated clinic readings are likely
// artefactual. However, B-05 (SBP<90) is a LOW reading alarm and
// white-coat typically produces HIGH readings — so no bypass here.
func (m *PhysiologySafetyMonitor) checkB05(d *RawPatientData) *PhysioResult {
	if d.SBPCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	if *d.SBPCurrent < m.cfg.SBPHaltThreshold {
		// Uncertainty dampening: HIGH uncertainty downgrades HALT→PAUSE
		if d.MeasurementUncertainty > 0 && d.BPPattern == "HIGH_UNCERTAINTY" || d.MeasurementUncertainty >= 15 {
			return &PhysioResult{Gate: PhysioPause, RuleFired: "B-05-DAMPENED"}
		}
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-05"}
	}
	return nil
}

// B-08: eGFR < 15 mL/min/1.73m² → HALT (CKD Stage 5 / kidney failure)
//
// CLINICAL RATIONALE — INTENTIONAL POLICY EXTENSION:
// This rule is NOT in the original specification. The spec places eGFR rules
// in Channel C only (PG-01: eGFR<30+Metformin, PG-02: eGFR<45+SGLT2i),
// making them drug-specific.
//
// This Channel B rule adds a DRUG-INDEPENDENT eGFR floor. Rationale:
//   - At eGFR <15, the kidneys cannot reliably clear insulin or oral agents.
//   - Insulin half-life extends unpredictably in severe CKD, increasing
//     hypoglycaemia risk regardless of which drug is prescribed.
//   - KDIGO CKD G5 classification recommends specialist nephrology referral;
//     automated titration is inappropriate without nephrologist oversight.
//   - This is a defense-in-depth guard: if Channel C's drug-specific rules
//     fail to fire (e.g., medication list incomplete), Channel B still catches
//     the physiological danger.
//
// This threshold was approved as a clinical policy extension on 2026-02-15.
// If the clinical team determines this is too conservative, the threshold
// can be lowered via PhysioConfig.EGFRHaltThreshold without code changes.
func (m *PhysiologySafetyMonitor) checkB08(d *RawPatientData) *PhysioResult {
	if d.EGFRCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	if *d.EGFRCurrent > 0 && *d.EGFRCurrent < m.cfg.EGFRHaltThreshold {
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-08"}
	}
	return nil
}

// B-09: eGFR 15–29 mL/min/1.73m² → PAUSE (CKD Stage 4)
//
// CLINICAL RATIONALE — INTENTIONAL POLICY EXTENSION:
// Same rationale as B-08. CKD Stage 4 patients have significantly impaired
// renal clearance. Automated dose changes should be paused to allow
// clinician review of renal dosing adjustments.
//
// This is distinct from Channel C's PG-01/PG-02 which are drug-specific.
// A patient on a sulfonylurea (not covered by PG-01 or PG-02) with
// eGFR 20 would not be caught by Channel C but IS caught here.
func (m *PhysiologySafetyMonitor) checkB09(d *RawPatientData) *PhysioResult {
	if d.EGFRCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	if *d.EGFRCurrent >= m.cfg.EGFRHaltThreshold && *d.EGFRCurrent < m.cfg.EGFRPauseThreshold {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-09"}
	}
	return nil
}

// B-10: eGFR slope < -5.0 mL/min/1.73m²/year → PAUSE (rapid decline)
//
// CLINICAL RATIONALE — INTENTIONAL POLICY EXTENSION:
// A rapid eGFR decline (>5 mL/min/1.73m²/year) signals accelerated
// nephropathy progression regardless of the absolute eGFR value.
// KDIGO guidelines flag slopes steeper than -5 as requiring urgent
// clinical review: automated titration should pause to allow
// nephrology assessment and medication adjustment.
//
// This is complementary to B-08/B-09 which guard on absolute eGFR
// thresholds. A patient at eGFR 55 (above B-09's threshold) but
// declining at -8 mL/min/year would be missed by absolute checks
// but is caught here.
func (m *PhysiologySafetyMonitor) checkB10(d *RawPatientData) *PhysioResult {
	if d.EGFRSlope == nil {
		return nil // absent trajectory data — no rule to apply
	}
	if *d.EGFRSlope < m.cfg.EGFRSlopeRapidDecline {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-10"}
	}
	return nil
}

// B-12: J-curve eGFR-stratified SBP lower limit → PAUSE or HALT
//
// CLINICAL RATIONALE:
// In CKD patients, aggressive BP lowering below the J-curve nadir risks
// renal hypoperfusion and accelerated GFR decline. The threshold is
// stratified by CKD stage because advanced CKD kidneys are more
// dependent on systemic pressure to maintain filtration:
//   - CKD 3a (eGFR 45-59): SBP floor 120 mmHg
//   - CKD 3b (eGFR 30-44): SBP floor 125 mmHg
//   - CKD 4  (eGFR 15-29): SBP floor 130 mmHg
//
// If the orchestrator pre-computed SBPLowerLimit (from eGFR), that value
// is used for the PAUSE floor. Otherwise, CKDStage string selects the
// appropriate floor.
//
// AMENDMENT 8 — STAGE-SPECIFIC HALT/PAUSE THRESHOLDS:
// The previous unified "SBP < 110 → HALT for all CKD ≥3" was overly
// conservative for stages 3a/3b where autoregulatory reserve is still
// partially intact. Amendment 8 replaces this with:
//   - Stage 3a: SBP < 100 → PAUSE (autoregulation still partially intact)
//   - Stage 3b: SBP < 105 → PAUSE (reduced reserve, but not perfusion-critical)
//   - Stage 4:  SBP < 100 → HALT  (perfusion danger, minimal autoregulatory reserve)
//     SBP 100-110 → PAUSE (cautionary zone)
//
// B-08 (SBP < 90 → HALT via B-05) remains the absolute floor for all stages.
//
// UNCERTAINTY DAMPENING:
// Stage 4 HALT (SBP < 100) is dampened: HIGH uncertainty → PAUSE.
// Stage 3a/3b PAUSE thresholds are NOT dampened (already the softer signal).
//
// WHITE-COAT BYPASS:
// Not applied — B-12 catches hypotension (low readings), not the elevated
// readings that white-coat artefact produces.
func (m *PhysiologySafetyMonitor) checkB12(d *RawPatientData) *PhysioResult {
	if d.SBPCurrent == nil {
		return nil
	}
	sbp := *d.SBPCurrent

	// Determine applicable CKD stage
	stage := d.CKDStage
	if stage == "" {
		return nil // J-curve rule only applies to CKD ≥3
	}

	// ── Amendment 8: Stage-specific hard thresholds ──
	// Evaluated before the PAUSE floors. B-05 (SBP < 90) is the absolute
	// floor checked earlier in the Evaluate chain; we only handle stage-
	// specific thresholds here.
	switch stage {
	case "3a":
		// Stage 3a: autoregulation partially intact → PAUSE only (no HALT)
		if sbp < m.cfg.SBPHaltStage3a {
			return &PhysioResult{Gate: PhysioPause, RuleFired: "B-12-3A"}
		}
	case "3b":
		// Stage 3b: reduced reserve → PAUSE only (no HALT)
		if sbp < m.cfg.SBPHaltStage3b {
			return &PhysioResult{Gate: PhysioPause, RuleFired: "B-12-3B"}
		}
	case "4":
		// Stage 4: minimal autoregulatory reserve → HALT at hard floor
		if sbp < m.cfg.SBPHaltStage4 {
			// Uncertainty dampening: HIGH uncertainty downgrades HALT → PAUSE
			if d.MeasurementUncertainty >= 15 {
				return &PhysioResult{Gate: PhysioPause, RuleFired: "B-12-DAMPENED"}
			}
			return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-12-4-HALT"}
		}
		// Stage 4 cautionary zone: SBP between HALT and PAUSE thresholds
		if sbp < m.cfg.SBPPauseStage4 {
			return &PhysioResult{Gate: PhysioPause, RuleFired: "B-12-4-PAUSE"}
		}
	case "5":
		// CKD 5 is handled by B-08 (eGFR < 15 → HALT); no J-curve rule here
		return nil
	}

	// ── eGFR-stratified PAUSE floors (dose-reduction warning) ──
	var floor float64
	if d.SBPLowerLimit != nil {
		floor = *d.SBPLowerLimit // orchestrator pre-computed
	} else {
		switch stage {
		case "3a":
			floor = m.cfg.SBPFloorStage3a
		case "3b":
			floor = m.cfg.SBPFloorStage3b
		case "4":
			floor = m.cfg.SBPFloorStage4
		default:
			return nil // CKD 1-2 don't trigger J-curve PAUSE
		}
	}

	if sbp < floor {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-12"}
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════
// CLINICAL THRESHOLD RULES — PAUSE (B-02, B-06, B-07)
// ════════════════════════════════════════════════════════════════════════

// B-02: Glucose < 4.5 mmol/L → near-hypoglycaemia
func (m *PhysiologySafetyMonitor) checkB02(d *RawPatientData) *PhysioResult {
	if d.GlucoseCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	if *d.GlucoseCurrent < m.cfg.GlucosePauseThreshold {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-02"}
	}
	return nil
}

// B-06: Weight 72h delta > 2.5 kg → fluid overload signal
func (m *PhysiologySafetyMonitor) checkB06(d *RawPatientData) *PhysioResult {
	if d.WeightKgCurrent == nil || d.Weight72hAgo == nil {
		return nil
	}
	delta := math.Abs(*d.WeightKgCurrent - *d.Weight72hAgo)
	if delta > m.cfg.WeightDeltaPause {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-06"}
	}
	return nil
}

// B-07: 3 consecutive declining glucose readings + current < 5.5 mmol/L
// + recent dose increase → glucose trajectory concern
func (m *PhysiologySafetyMonitor) checkB07(d *RawPatientData) *PhysioResult {
	if d.GlucoseCurrent == nil {
		return nil // absent data ≠ clinical finding
	}
	if !d.RecentDoseIncrease {
		return nil
	}
	if *d.GlucoseCurrent >= m.cfg.GlucoseTrendThreshold {
		return nil
	}
	if len(d.GlucoseReadings) < 3 {
		return nil
	}
	// Check 3 consecutive declines (readings are most-recent-first)
	declining := true
	for i := 0; i < len(d.GlucoseReadings)-1 && i < 2; i++ {
		if d.GlucoseReadings[i].Value >= d.GlucoseReadings[i+1].Value {
			declining = false
			break
		}
	}
	if declining {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-07"}
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════
// HEART RATE RULES (B-13 through B-16) — Wave 2, Amendment 3
// ════════════════════════════════════════════════════════════════════════

// B-13: HR < 45 bpm + RESTING + confirmed → HALT (severe bradycardia)
//
// CLINICAL RATIONALE:
// Resting HR below 45 indicates significant conduction system disease or
// excessive negative chronotropy. Automated titration of any drug that
// might further suppress HR (beta-blockers, non-DHP CCBs, digoxin)
// must halt immediately. "Confirmed" means 2 consecutive readings
// within 10% — a single noisy reading does not fire HALT.
//
// BETA-BLOCKER PERTURBATION ADJUSTMENT:
// During BB perturbation window (recently started/uptitrated), the expected
// HR drops by ~5 bpm. Threshold is lowered from 45 → 40 bpm to accommodate
// the expected pharmacodynamic effect. This prevents false HALTs on patients
// whose HR is dropping as expected after BB initiation.
func (m *PhysiologySafetyMonitor) checkB13(d *RawPatientData) *PhysioResult {
	if d.HeartRateCurrent == nil {
		return nil
	}
	if !d.HeartRateConfirmed || d.HRContext != "RESTING" {
		return nil // only fire on confirmed resting readings
	}
	threshold := m.cfg.HRBradycardiaHalt
	// Beta-blocker perturbation: primary boolean OR Amendment 2 registry fallback
	bbPerturbation := d.BetaBlockerPerturbationActive ||
		FindActivePerturbation(d.ActivePerturbations, DrugClassBetaBlocker, time.Now()) != nil
	if bbPerturbation {
		// Allow 5 bpm lower during BB perturbation window (expected pharmacodynamic effect)
		threshold -= 5.0
	}
	if *d.HeartRateCurrent < threshold {
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-13"}
	}
	return nil
}

// B-14: HR < 55 bpm + beta-blocker active + dose change within 7d → PAUSE
//
// CLINICAL RATIONALE:
// Beta-blocker-induced bradycardia (HR 45-55) during the dose adjustment
// window warrants pausing further titration. The patient may stabilise
// once steady-state is reached. Without the dose_change_7d guard, this
// would fire on every patient with well-tolerated chronic beta-blockade.
//
// BETA-BLOCKER PERTURBATION ADJUSTMENT:
// During BB perturbation window, threshold is lowered from 55 → 50 bpm.
// A 5 bpm drop is expected pharmacodynamic effect after BB start/uptitration.
func (m *PhysiologySafetyMonitor) checkB14(d *RawPatientData) *PhysioResult {
	if d.HeartRateCurrent == nil {
		return nil
	}
	if !d.BetaBlockerActive || !d.BetaBlockerDoseChangeIn7d {
		return nil
	}
	threshold := m.cfg.HRBradycardiaPause
	// Beta-blocker perturbation: primary boolean OR Amendment 2 registry fallback
	bbPerturbation := d.BetaBlockerPerturbationActive ||
		FindActivePerturbation(d.ActivePerturbations, DrugClassBetaBlocker, time.Now()) != nil
	if bbPerturbation {
		// Allow 5 bpm lower during BB perturbation window (expected pharmacodynamic effect)
		threshold -= 5.0
	}
	if *d.HeartRateCurrent < threshold {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-14"}
	}
	return nil
}

// B-15: HR > 120 bpm + RESTING + confirmed → PAUSE (resting tachycardia)
//
// CLINICAL RATIONALE:
// Sustained resting tachycardia may indicate decompensation, infection,
// thyrotoxicosis, or volume depletion. While not immediately dangerous
// (unlike bradycardia), it signals an underlying condition that should
// be investigated before further titration.
func (m *PhysiologySafetyMonitor) checkB15(d *RawPatientData) *PhysioResult {
	if d.HeartRateCurrent == nil {
		return nil
	}
	if !d.HeartRateConfirmed || d.HRContext != "RESTING" {
		return nil
	}
	if *d.HeartRateCurrent > m.cfg.HRTachycardiaPause {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-15"}
	}
	return nil
}

// B-16: Irregular heart rhythm + confirmed → PAUSE + KB22_TRIGGER
//
// CLINICAL RATIONALE:
// Confirmed irregular rhythm requires differential diagnosis (AF, flutter,
// frequent ectopics). This fires PAUSE and annotates the result with
// KB22_TRIGGER so the orchestrator can publish a KB-22 HPI investigation
// event for the AF differential node (p_irregular_hr).
func (m *PhysiologySafetyMonitor) checkB16(d *RawPatientData) *PhysioResult {
	if d.HRRegularity != "IRREGULAR" || !d.HeartRateConfirmed {
		return nil
	}
	// KB22_TRIGGER sentinel is added to rawVals by Evaluate() before returning.
	// Populate KB22TriggerRequest so the orchestrator can route to KB-22 HPI.
	triggerData := make(map[string]interface{})
	if d.HeartRateCurrent != nil {
		triggerData["heart_rate_bpm"] = *d.HeartRateCurrent
	}
	triggerData["hr_regularity"] = d.HRRegularity
	triggerData["hr_context"] = d.HRContext

	return &PhysioResult{
		Gate:      PhysioPause,
		RuleFired: "B-16",
		KB22Triggers: []KB22TriggerRequest{
			{
				SentinelID: "B-16",
				HPINodeID:  "p04_irregular_hr",
				Data:       triggerData,
			},
		},
	}
}

// ════════════════════════════════════════════════════════════════════════
// HYPONATRAEMIA RULES (B-17 through B-19) — Wave 2, Amendment 11
// ════════════════════════════════════════════════════════════════════════

// B-17: Na+ < 132 mEq/L AND thiazide active → HALT
//
// CLINICAL RATIONALE:
// Severe thiazide-induced hyponatraemia (Na+ < 132) is a medical
// emergency. The thiazide must be held immediately, and automated
// titration of any agent must stop. Na+ < 132 without thiazide may
// have other causes (SIADH, heart failure) not within V-MCU's scope.
func (m *PhysiologySafetyMonitor) checkB17(d *RawPatientData) *PhysioResult {
	if d.SodiumCurrent == nil || !d.ThiazideActive {
		return nil
	}
	if *d.SodiumCurrent < m.cfg.SodiumHaltThreshold {
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-17"}
	}
	return nil
}

// B-18: Na+ 132-135 mEq/L AND thiazide active → PAUSE
//
// CLINICAL RATIONALE:
// Mild thiazide-induced hyponatraemia. Dose reduction or switch to
// loop diuretic may be needed. Titration pauses while electrolytes
// are monitored.
func (m *PhysiologySafetyMonitor) checkB18(d *RawPatientData) *PhysioResult {
	if d.SodiumCurrent == nil || !d.ThiazideActive {
		return nil
	}
	if *d.SodiumCurrent >= m.cfg.SodiumHaltThreshold && *d.SodiumCurrent < m.cfg.SodiumPauseThreshold {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-18"}
	}
	return nil
}

// B-19: Na+ < 135 mEq/L AND SUMMER AND thiazide → PAUSE (seasonal amplification)
//
// CLINICAL RATIONALE:
// In hot climates (India SUMMER: May-June), increased perspiration
// combined with thiazide kaliuresis/natriuresis amplifies electrolyte
// losses. A borderline Na+ (133-135) that might be acceptable in
// winter becomes concerning in summer. This rule catches the seasonal
// interaction that B-18 alone would miss (B-18 only fires at <135
// with thiazide; B-19 fires at the same threshold but only in summer,
// providing an independent trigger if B-18 was already checked).
func (m *PhysiologySafetyMonitor) checkB19(d *RawPatientData) *PhysioResult {
	if d.SodiumCurrent == nil || !d.ThiazideActive {
		return nil
	}
	if d.Season != "SUMMER" {
		return nil
	}
	if *d.SodiumCurrent < m.cfg.SodiumSeasonalThreshold {
		return &PhysioResult{Gate: PhysioPause, RuleFired: "B-19"}
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════
// GLUCOSE VARIABILITY RULE (B-20) — Wave 2
// ════════════════════════════════════════════════════════════════════════

// B-21: Finerenone + K+ ≥5.5 → HALT (hyperkalemia risk)
//
// CLINICAL RATIONALE:
// Finerenone (non-steroidal MRA) reduces albuminuria and CKD progression
// but carries significant hyperkalemia risk, especially in CKD patients
// already on RAAS blockade. FIDELIO-DKD and FIGARO-DKD trials mandated
// K+ <5.5 for continuation. At K+ ≥5.5 with finerenone active,
// immediate titration halt is required to prevent life-threatening
// hyperkalemia. This threshold is lower than B-04's general K+ >6.0
// because finerenone potentiates potassium retention.
func (m *PhysiologySafetyMonitor) checkB21(d *RawPatientData) *PhysioResult {
	if !d.FinerenoneActive {
		return nil
	}
	if d.PotassiumCurrent == nil {
		return nil
	}
	if *d.PotassiumCurrent >= 5.5 {
		return &PhysioResult{Gate: PhysioHalt, RuleFired: "B-21"}
	}
	return nil
}

// B-20: Glucose variability — CV% > 36% → PAUSE
// ADA 2024: CV >36% indicates unstable glycaemic control;
// dose adjustments during high variability are unsafe.
func (m *PhysiologySafetyMonitor) checkB20(d *RawPatientData) *PhysioResult {
	if d.GlucoseCV30d == nil {
		return nil
	}
	if *d.GlucoseCV30d > m.cfg.GlucoseCVPauseThreshold {
		return &PhysioResult{
			Gate:      PhysioPause,
			RuleFired: "B-20",
			RawValues: map[string]float64{"glucose_cv_30d": *d.GlucoseCV30d},
		}
	}
	return nil
}
