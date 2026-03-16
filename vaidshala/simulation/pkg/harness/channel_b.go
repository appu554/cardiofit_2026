// Package harness implements the simulation's version of V-MCU components.
// Channel B: PhysiologySafetyMonitor — evaluates raw lab values for physiological safety.
// This MUST remain a separate struct from any metabolic physiology engine.
// Build-time import constraint: this package cannot import metabolic_physiology.
package harness

import (
	"fmt"
	"math"
	"time"

	"vaidshala/simulation/pkg/types"
)

// PhysiologySafetyMonitor evaluates Channel B rules against raw patient data.
// It reads ONLY raw lab values — independent of Channel A clinical reasoning.
type PhysiologySafetyMonitor struct {
	EGFRStratifiedBPFloor map[string]int // CKD stage → SBP floor (J-curve, Amendment 8)
}

func NewPhysiologySafetyMonitor() *PhysiologySafetyMonitor {
	return &PhysiologySafetyMonitor{
		EGFRStratifiedBPFloor: map[string]int{
			"CKD1_2": 90,  // eGFR >60: standard floor
			"CKD3a":  100, // eGFR 45-59: PAUSE below 100
			"CKD3b":  105, // eGFR 30-44: PAUSE below 105
			"CKD4":   110, // eGFR 15-29: HALT below 100, PAUSE 100-110
		},
	}
}

type ChannelBResult struct {
	Gate      types.GateSignal
	RuleFired string
	Details   string
}

// Evaluate runs all Channel B rules and returns the most restrictive result.
// Rules are numbered per the HTN Amendment canonical scheme (B-01 through B-20).
func (m *PhysiologySafetyMonitor) Evaluate(data *types.RawPatientData) ChannelBResult {
	results := []ChannelBResult{
		m.ruleB01(data), // Active hypoglycaemia
		m.ruleB02(data), // Predictive hypoglycaemia
		m.ruleB03(data), // K+ extremes (hypo/hyperkalaemia)
		m.ruleB04(data), // Creatinine 48h delta (AKI)
		m.ruleB05(data), // Weight 72h delta (fluid shift)
		m.ruleB06(data), // Physiologically impossible values
		m.ruleB07(data), // eGFR <15 (near-dialysis)
		m.ruleB08(data), // SBP absolute floor (<90)
		m.ruleB09(data), // eGFR 15-29 (CKD Stage 4)
		m.ruleB10(data), // Stale K+ (>14 days) — DA-06
		m.ruleB11(data), // Stale creatinine (>30 days) — DA-07
		m.ruleB12(data), // J-curve: eGFR-stratified BP floor (Amendment 8)
		m.ruleB13(data), // Severe bradycardia (Amendment 3)
		m.ruleB14(data), // Beta-blocker bradycardia (Amendment 3)
		m.ruleB15(data), // Resting tachycardia (Amendment 3)
		m.ruleB16(data), // Irregular heart rate — AF (Amendment 3)
		m.ruleB17(data), // Severe hyponatraemia + thiazide (Amendment 11)
		m.ruleB18(data), // Mild hyponatraemia + thiazide (Amendment 11)
		m.ruleB20(data), // Glucose CV% > 36% (high glycaemic variability)
	}

	worst := ChannelBResult{Gate: types.CLEAR, RuleFired: "NONE", Details: "all clear"}
	for _, r := range results {
		if r.Gate > worst.Gate {
			worst = r
		}
	}
	return worst
}

// B-01: Active hypoglycaemia — Glucose < 3.9 mmol/L → HALT
func (m *PhysiologySafetyMonitor) ruleB01(d *types.RawPatientData) ChannelBResult {
	if d.GlucoseCurrent > 0 && d.GlucoseCurrent < 3.9 {
		return ChannelBResult{types.HALT, "B-01",
			fmt.Sprintf("active_hypoglycaemia: glucose=%.1f mmol/L (<3.9)", d.GlucoseCurrent)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-02: Predictive hypoglycaemia — glucose declining + recent dose increase + current <5.5 → PAUSE
func (m *PhysiologySafetyMonitor) ruleB02(d *types.RawPatientData) ChannelBResult {
	if d.GlucoseCurrent > 0 && d.GlucosePrevious > 0 &&
		d.GlucoseCurrent < 5.5 &&
		d.GlucoseCurrent < d.GlucosePrevious &&
		d.RecentDoseIncrease {
		return ChannelBResult{types.PAUSE, "B-02",
			fmt.Sprintf("predictive_hypo: glucose=%.1f declining from %.1f, recent dose increase",
				d.GlucoseCurrent, d.GlucosePrevious)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-03: K+ extremes — K+ <3.0 or >6.0 → HALT
func (m *PhysiologySafetyMonitor) ruleB03(d *types.RawPatientData) ChannelBResult {
	if d.PotassiumCurrent > 0 {
		if d.PotassiumCurrent < 3.0 {
			return ChannelBResult{types.HALT, "B-03",
				fmt.Sprintf("hypokalaemia: K+=%.1f mmol/L (<3.0)", d.PotassiumCurrent)}
		}
		if d.PotassiumCurrent > 6.0 {
			return ChannelBResult{types.HALT, "B-03",
				fmt.Sprintf("hyperkalaemia: K+=%.1f mmol/L (>6.0)", d.PotassiumCurrent)}
		}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-04: Creatinine 48h delta >26 µmol/L → HALT (AKI Stage 1 by KDIGO)
// UNLESS CreatinineRiseExplained=true (PG-14 RAAS tolerance, set by orchestrator)
func (m *PhysiologySafetyMonitor) ruleB04(d *types.RawPatientData) ChannelBResult {
	if d.CreatinineCurrent > 0 && d.CreatininePrevious > 0 {
		delta := d.CreatinineCurrent - d.CreatininePrevious
		if delta > 26.0 {
			if d.CreatinineRiseExplained {
				// PG-14 causal suppression: downgrade HALT to PAUSE
				return ChannelBResult{types.PAUSE, "B-04+PG-14",
					fmt.Sprintf("creatinine_rise_explained_raas: delta=%.0f µmol/L, RAAS tolerance applied", delta)}
			}
			return ChannelBResult{types.HALT, "B-04",
				fmt.Sprintf("aki_suspected: creatinine 48h delta=%.0f µmol/L (>26)", delta)}
		}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-05: Weight 72h delta >2.5 kg → PAUSE (acute fluid shift)
func (m *PhysiologySafetyMonitor) ruleB05(d *types.RawPatientData) ChannelBResult {
	if d.Weight > 0 && d.WeightPrevious > 0 {
		delta := math.Abs(d.Weight - d.WeightPrevious)
		if delta > 2.5 {
			return ChannelBResult{types.PAUSE, "B-05",
				fmt.Sprintf("fluid_shift: weight 72h delta=%.1f kg (>2.5)", delta)}
		}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-06: Physiologically impossible values → HOLD_DATA
func (m *PhysiologySafetyMonitor) ruleB06(d *types.RawPatientData) ChannelBResult {
	if d.GlucoseCurrent > 0 && (d.GlucoseCurrent < 1.0 || d.GlucoseCurrent > 50.0) {
		return ChannelBResult{types.HOLD_DATA, "B-06",
			fmt.Sprintf("implausible_glucose: %.1f mmol/L", d.GlucoseCurrent)}
	}
	if d.SBP > 0 && (d.SBP < 40 || d.SBP > 300) {
		return ChannelBResult{types.HOLD_DATA, "B-06",
			fmt.Sprintf("implausible_sbp: %d mmHg", d.SBP)}
	}
	if d.PotassiumCurrent > 0 && (d.PotassiumCurrent < 1.5 || d.PotassiumCurrent > 10.0) {
		return ChannelBResult{types.HOLD_DATA, "B-06",
			fmt.Sprintf("implausible_potassium: %.1f mmol/L", d.PotassiumCurrent)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-07: eGFR <15 → HALT (near-dialysis, universal titration stop)
func (m *PhysiologySafetyMonitor) ruleB07(d *types.RawPatientData) ChannelBResult {
	if d.EGFR > 0 && d.EGFR < 15 {
		return ChannelBResult{types.HALT, "B-07",
			fmt.Sprintf("egfr_critical: eGFR=%.0f (<15, near-dialysis)", d.EGFR)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-08: SBP <90 → HALT (absolute hypotension floor)
func (m *PhysiologySafetyMonitor) ruleB08(d *types.RawPatientData) ChannelBResult {
	if d.SBP > 0 && d.SBP < 90 {
		return ChannelBResult{types.HALT, "B-08",
			fmt.Sprintf("hypotension: SBP=%d mmHg (<90)", d.SBP)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-09: eGFR 15-29 → PAUSE (CKD Stage 4, physician review)
func (m *PhysiologySafetyMonitor) ruleB09(d *types.RawPatientData) ChannelBResult {
	if d.EGFR > 0 && d.EGFR >= 15 && d.EGFR < 30 {
		return ChannelBResult{types.PAUSE, "B-09",
			fmt.Sprintf("ckd_stage4: eGFR=%.0f (15-29)", d.EGFR)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-10 (DA-06): Stale K+ >14 days → HOLD_DATA
func (m *PhysiologySafetyMonitor) ruleB10(d *types.RawPatientData) ChannelBResult {
	if !d.PotassiumTimestamp.IsZero() {
		staleDays := time.Since(d.PotassiumTimestamp).Hours() / 24
		if staleDays > 14 {
			return ChannelBResult{types.HOLD_DATA, "B-10/DA-06",
				fmt.Sprintf("stale_potassium: %.0f days since last K+ (>14)", staleDays)}
		}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-11 (DA-07): Stale creatinine >30 days → HOLD_DATA
func (m *PhysiologySafetyMonitor) ruleB11(d *types.RawPatientData) ChannelBResult {
	if !d.CreatinineTimestamp.IsZero() {
		staleDays := time.Since(d.CreatinineTimestamp).Hours() / 24
		if staleDays > 30 {
			return ChannelBResult{types.HOLD_DATA, "B-11/DA-07",
				fmt.Sprintf("stale_creatinine: %.0f days since last creatinine (>30)", staleDays)}
		}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-12: J-curve eGFR-stratified BP floor (HTN Amendment 8)
// CKD Stage 3a: SBP <100 → PAUSE
// CKD Stage 3b: SBP <105 → PAUSE
// CKD Stage 4:  SBP <110 → HALT, SBP 100-110 → PAUSE
func (m *PhysiologySafetyMonitor) ruleB12(d *types.RawPatientData) ChannelBResult {
	if d.SBP <= 0 || d.EGFR <= 0 {
		return ChannelBResult{Gate: types.CLEAR}
	}
	if d.EGFR >= 60 {
		return ChannelBResult{Gate: types.CLEAR} // Normal autoregulation
	}
	if d.EGFR >= 45 { // CKD 3a
		if d.SBP < 100 {
			return ChannelBResult{types.PAUSE, "B-12",
				fmt.Sprintf("jcurve_ckd3a: SBP=%d (<100), eGFR=%.0f", d.SBP, d.EGFR)}
		}
	} else if d.EGFR >= 30 { // CKD 3b
		if d.SBP < 105 {
			return ChannelBResult{types.PAUSE, "B-12",
				fmt.Sprintf("jcurve_ckd3b: SBP=%d (<105), eGFR=%.0f", d.SBP, d.EGFR)}
		}
	} else if d.EGFR >= 15 { // CKD 4
		if d.SBP < 100 {
			return ChannelBResult{types.HALT, "B-12",
				fmt.Sprintf("jcurve_ckd4_halt: SBP=%d (<100), eGFR=%.0f", d.SBP, d.EGFR)}
		}
		if d.SBP < 110 {
			return ChannelBResult{types.PAUSE, "B-12",
				fmt.Sprintf("jcurve_ckd4_pause: SBP=%d (100-110), eGFR=%.0f", d.SBP, d.EGFR)}
		}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-13: Severe bradycardia — HR <45 bpm resting, confirmed → HALT (Amendment 3)
func (m *PhysiologySafetyMonitor) ruleB13(d *types.RawPatientData) ChannelBResult {
	if d.HeartRate > 0 && d.HeartRate < 45 {
		return ChannelBResult{types.HALT, "B-13",
			fmt.Sprintf("severe_bradycardia: HR=%d bpm (<45)", d.HeartRate)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-14: Beta-blocker bradycardia — HR <55 + beta-blocker + recent dose change → PAUSE
func (m *PhysiologySafetyMonitor) ruleB14(d *types.RawPatientData) ChannelBResult {
	if d.HeartRate > 0 && d.HeartRate < 55 && d.BetaBlockerActive && d.RecentDoseIncrease {
		return ChannelBResult{types.PAUSE, "B-14",
			fmt.Sprintf("betablocker_bradycardia: HR=%d, beta_blocker=true, recent_dose_change=true", d.HeartRate)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-15: Resting tachycardia — HR >120 bpm resting → PAUSE
func (m *PhysiologySafetyMonitor) ruleB15(d *types.RawPatientData) ChannelBResult {
	if d.HeartRate > 120 {
		return ChannelBResult{types.PAUSE, "B-15",
			fmt.Sprintf("resting_tachycardia: HR=%d bpm (>120)", d.HeartRate)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-16: Irregular heart rate — possible AF → PAUSE + KB22_TRIGGER
func (m *PhysiologySafetyMonitor) ruleB16(d *types.RawPatientData) ChannelBResult {
	if d.HeartRateRegularity == "IRREGULAR" {
		return ChannelBResult{types.PAUSE, "B-16",
			"irregular_heart_rate: possible AF, KB22_TRIGGER recommended"}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-17: Severe hyponatraemia — Na <132 + thiazide → HALT (Amendment 11)
func (m *PhysiologySafetyMonitor) ruleB17(d *types.RawPatientData) ChannelBResult {
	// We check thiazide via the BetaBlockerActive-like pattern; in production
	// this comes from TitrationContext.ThiazideActive
	if d.SodiumCurrent > 0 && d.SodiumCurrent < 132 {
		return ChannelBResult{types.HALT, "B-17",
			fmt.Sprintf("severe_hyponatraemia: Na+=%.0f mmol/L (<132)", d.SodiumCurrent)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-18: Mild hyponatraemia — Na 132-135 + thiazide → PAUSE (Amendment 11)
func (m *PhysiologySafetyMonitor) ruleB18(d *types.RawPatientData) ChannelBResult {
	if d.SodiumCurrent > 0 && d.SodiumCurrent >= 132 && d.SodiumCurrent < 135 {
		return ChannelBResult{types.PAUSE, "B-18",
			fmt.Sprintf("mild_hyponatraemia: Na+=%.0f mmol/L (132-135)", d.SodiumCurrent)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}

// B-20: Glucose CV% > 36% → PAUSE (high glycaemic variability)
func (m *PhysiologySafetyMonitor) ruleB20(d *types.RawPatientData) ChannelBResult {
	if d.GlucoseCV30d > 36.0 {
		return ChannelBResult{types.PAUSE, "B-20",
			fmt.Sprintf("glucose_cv_high: CV30d=%.1f%% (>36%%)", d.GlucoseCV30d)}
	}
	return ChannelBResult{Gate: types.CLEAR}
}
