// Package patient defines virtual patient archetypes for simulation.
// Each archetype represents a common clinical pattern in the Indian DM+HTN+CKD population.
// These are not random patients — they are clinically grounded edge cases that stress-test
// specific V-MCU safety rules and titration logic.
package patient

import (
	"time"

	"vaidshala/simulation/pkg/types"
)

// VirtualPatient represents a complete synthetic patient state for simulation.
type VirtualPatient struct {
	ID          string
	Archetype   string
	Description string
	Labs        types.RawPatientData
	Context     types.TitrationContext
	MCUGate     types.GateSignal
	Adherence   float64
	LoopTrust   float64
}

// ToTitrationInput converts a VirtualPatient to a TitrationCycleInput.
func (vp *VirtualPatient) ToTitrationInput(cycle int) types.TitrationCycleInput {
	return types.TitrationCycleInput{
		PatientID:        vp.ID,
		CycleNumber:      cycle,
		RawLabs:          &vp.Labs,
		TitrationContext: &vp.Context,
		MCUGate:          vp.MCUGate,
		AdherenceScore:   vp.Adherence,
		LoopTrustScore:   vp.LoopTrust,
	}
}

// now is a helper for generating timestamps relative to current time.
func daysAgo(d int) time.Time { return time.Now().Add(-time.Duration(d) * 24 * time.Hour) }
func hoursAgo(h int) time.Time { return time.Now().Add(-time.Duration(h) * time.Hour) }

// ========================================================================
// SCENARIO 1: Active Hypoglycaemia + Insulin Increase Proposed
// Expected: ALL 3 channels HALT. Zero dose output.
// Tests: B-01, PG-04, arbiter 3-channel confirmation.
// ========================================================================
func ActiveHypoglycaemia() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-HYPO-001",
		Archetype:   "active_hypoglycaemia",
		Description: "Glucose 3.5 mmol/L + K+ 2.8 mmol/L + insulin increase proposed. All channels must HALT.",
		Labs: types.RawPatientData{
			PatientID:        "SIM-HYPO-001",
			Timestamp:        time.Now(),
			GlucoseCurrent:   3.5,  // Below 3.9 → B-01 HALT
			GlucoseTimestamp:  hoursAgo(1),
			PotassiumCurrent: 2.8,  // Below 3.0 → B-03 HALT
			PotassiumTimestamp: daysAgo(0),
			CreatinineCurrent: 88,
			EGFR:              65,
			SBP:               128,
			DBP:               82,
		},
		Context: types.TitrationContext{
			InsulinActive:     true,
			ProposedDoseDelta: 2.0, // Increase proposed → PG-04 HALT
			CurrentDose:       18.0,
			EGFRCurrent:       65,
		},
		MCUGate:   types.HALT, // KB-23 also says HALT
		Adherence: 0.90,
		LoopTrust: 0.85,
	}
}

// ========================================================================
// SCENARIO 2: AKI Mid-Titration (Creatinine Spike)
// Expected: Channel B HALT (B-04), Channel C HALT (PG-03).
// Tests: creatinine 48h delta, cross-channel AKI confirmation.
// ========================================================================
func AKIMidTitration() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-AKI-001",
		Archetype:   "aki_mid_titration",
		Description: "Creatinine spike from 90 to 130 µmol/L in 48h. Genuine AKI — not RAAS response.",
		Labs: types.RawPatientData{
			PatientID:         "SIM-AKI-001",
			Timestamp:         time.Now(),
			GlucoseCurrent:    8.0,
			GlucoseTimestamp:   hoursAgo(6),
			CreatinineCurrent:  130, // delta = 40 µmol/L (>26)
			CreatininePrevious: 90,
			CreatinineTimestamp: daysAgo(0),
			EGFR:               42,
			EGFRTimestamp:       daysAgo(0),
			PotassiumCurrent:   5.1,
			PotassiumTimestamp:  daysAgo(0),
			SBP:                135,
			DBP:                85,
		},
		Context: types.TitrationContext{
			CurrentDose:       20.0,
			EGFRCurrent:       42,
			ACEiActive:        true,
			RAASChangeWithin14Days: false, // Not a RAAS change → no PG-14 suppression
			ActiveMedications: []types.ActiveMedication{
				{DrugClass: "METFORMIN", Dose: 1000},
				{DrugClass: "ACEi", Dose: 10},
			},
		},
		MCUGate:   types.PAUSE,
		Adherence: 0.85,
		LoopTrust: 0.80,
	}
}

// ========================================================================
// SCENARIO 3: RAAS Creatinine Tolerance (Expected Rise, NOT AKI)
// Expected: Channel B downgrades to PAUSE (not HALT) via PG-14.
// Tests: PG-14 RAAS tolerance, CreatinineRiseExplained flag.
// This is the MOST CRITICAL false-positive prevention rule.
// ========================================================================
func RAASCreatinineTolerance() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-RAAS-001",
		Archetype:   "raas_creatinine_tolerance",
		Description: "ACEi increased 7 days ago. Creatinine rose 30% (117→90, delta=27 > 26 µmol/L). Expected RAAS response — NOT AKI. B-03 suppressed to PAUSE via RAAS tolerance (PG-14).",
		Labs: types.RawPatientData{
			PatientID:              "SIM-RAAS-001",
			Timestamp:              time.Now(),
			GlucoseCurrent:         7.5,
			GlucoseTimestamp:        hoursAgo(6),
			CreatinineCurrent:      117, // 30% rise from 90 (delta=27 > 26 µmol/L → triggers B-03, suppressed to PAUSE via RAAS tolerance)
			CreatininePrevious:     90,
			CreatinineTimestamp:     daysAgo(0),
			CreatinineRiseExplained: true, // Orchestrator sets this when PG-14 conditions met
			EGFR:                   55,
			EGFRTimestamp:           daysAgo(0),
			PotassiumCurrent:       4.8, // Below 5.5 → RAAS tolerance applies
			PotassiumTimestamp:      daysAgo(0),
			SBP:                    132,
			DBP:                    84,
		},
		Context: types.TitrationContext{
			CurrentDose:            20.0,
			EGFRCurrent:           55,
			ACEiActive:            true,
			RAASChangeWithin14Days: true,
			RAASChangeDate:         daysAgo(7),
			PreRAASCreatinine:     90,
			ActiveMedications: []types.ActiveMedication{
				{DrugClass: "ACEi", Dose: 10, StartDate: daysAgo(7)},
			},
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.90,
		LoopTrust: 0.85,
	}
}

// ========================================================================
// SCENARIO 4: 5-Day Data Drop-Out (Patient Goes Silent)
// Expected: HOLD_DATA on stale K+ (>14 days). No titration during blackout.
// Tests: DA-06 stale lab detection, HOLD_DATA behaviour.
// ========================================================================
func DataDropOut() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-DROP-001",
		Archetype:   "data_dropout",
		Description: "Patient silent for 16 days. K+ is stale (>14 days). System must not titrate on stale safety data.",
		Labs: types.RawPatientData{
			PatientID:          "SIM-DROP-001",
			Timestamp:          time.Now(),
			GlucoseCurrent:     9.0,
			GlucoseTimestamp:    daysAgo(16),
			PotassiumCurrent:   4.5,
			PotassiumTimestamp:  daysAgo(16), // 16 days stale → B-10 HOLD_DATA
			CreatinineCurrent:  95,
			CreatinineTimestamp: daysAgo(16),
			EGFR:               60,
			SBP:                140,
			DBP:                88,
		},
		Context: types.TitrationContext{
			CurrentDose: 16.0,
			EGFRCurrent: 60,
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.70, // PRE_GATEWAY_DEFAULT
		LoopTrust: 0.50,
	}
}

// ========================================================================
// SCENARIO 5: Non-Adherent Patient (Adherence 0.30)
// Expected: gain_factor=0.25. Dose delta ≤25% of calculated.
// Tests: adherence-modulated gain factor, BEHAVIORAL_GAP gating.
// ========================================================================
func NonAdherentPatient() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-NONADH-001",
		Archetype:   "non_adherent",
		Description: "Adherence 0.30. MCU_GATE=MODIFY from KB-23 (BEHAVIORAL_GAP). Dose delta must be minimal.",
		Labs: types.RawPatientData{
			PatientID:        "SIM-NONADH-001",
			Timestamp:        time.Now(),
			GlucoseCurrent:   11.0, // High — would normally trigger +4U
			GlucoseTimestamp:  hoursAgo(2),
			PotassiumCurrent: 4.2,
			PotassiumTimestamp: daysAgo(1),
			CreatinineCurrent: 80,
			CreatinineTimestamp: daysAgo(5),
			EGFR:              72,
			SBP:               138,
			DBP:               86,
		},
		Context: types.TitrationContext{
			CurrentDose: 14.0,
			EGFRCurrent: 72,
		},
		MCUGate:   types.MODIFY, // KB-23 says MODIFY due to BEHAVIORAL_GAP
		Adherence: 0.30,         // Very low → gain_factor = 0.25
		LoopTrust: 0.40,
	}
}

// ========================================================================
// SCENARIO 6: J-Curve CKD Stage 3b Patient
// Expected: Channel B PAUSE when SBP drops below 105 (eGFR-stratified floor).
// Tests: B-12 J-curve rule, eGFR-stratified BP floor.
// ========================================================================
func JCurveCKD3b() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-JCURVE-001",
		Archetype:   "jcurve_ckd3b",
		Description: "CKD Stage 3b (eGFR 35). SBP=102 — below the 105 floor for this CKD stage. Renal perfusion at risk.",
		Labs: types.RawPatientData{
			PatientID:        "SIM-JCURVE-001",
			Timestamp:        time.Now(),
			GlucoseCurrent:   7.8,
			GlucoseTimestamp:  hoursAgo(4),
			PotassiumCurrent: 4.5,
			PotassiumTimestamp: daysAgo(2),
			CreatinineCurrent: 150,
			CreatinineTimestamp: daysAgo(2),
			EGFR:              35, // CKD 3b
			EGFRTimestamp:      daysAgo(2),
			SBP:               102, // Below 105 floor for CKD 3b
			DBP:               68,
		},
		Context: types.TitrationContext{
			CurrentDose: 12.0,
			EGFRCurrent: 35,
			ACEiActive:  true,
			CKDStage:    "3b",
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.88,
		LoopTrust: 0.80,
	}
}

// ========================================================================
// SCENARIO 7: Dual RAAS Contraindication
// Expected: Channel C HALT (PG-08). ACEi + ARB simultaneously.
// Tests: PG-08 HTN composite rule.
// ========================================================================
func DualRAAS() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-DUALRAAS-001",
		Archetype:   "dual_raas",
		Description: "Patient on both ACEi and ARB simultaneously. PG-08 must fire HALT.",
		Labs: types.RawPatientData{
			PatientID:        "SIM-DUALRAAS-001",
			Timestamp:        time.Now(),
			GlucoseCurrent:   8.2,
			GlucoseTimestamp:  hoursAgo(3),
			PotassiumCurrent: 5.2,
			PotassiumTimestamp: daysAgo(1),
			CreatinineCurrent: 110,
			EGFR:              52,
			SBP:               145,
			DBP:               90,
		},
		Context: types.TitrationContext{
			CurrentDose: 16.0,
			EGFRCurrent: 52,
			ACEiActive:  true,
			ARBActive:   true, // DUAL RAAS → PG-08 HALT
			ActiveMedications: []types.ActiveMedication{
				{DrugClass: "ACEi", Dose: 10},
				{DrugClass: "ARB", Dose: 80},
			},
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.85,
		LoopTrust: 0.80,
	}
}

// ========================================================================
// SCENARIO 8: Severe Hyponatraemia + Thiazide (Indian Summer)
// Expected: Channel B HALT (B-17). Na+ <132 with thiazide.
// Tests: B-17 hyponatraemia rule (HTN Amendment 11).
// ========================================================================
func HyponatraemiaThiazide() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-HYPONA-001",
		Archetype:   "hyponatraemia_thiazide",
		Description: "Na+=128 on thiazide in Indian summer. Cerebral oedema risk. Thiazide must be stopped.",
		Labs: types.RawPatientData{
			PatientID:        "SIM-HYPONA-001",
			Timestamp:        time.Now(),
			GlucoseCurrent:   7.5,
			GlucoseTimestamp:  hoursAgo(4),
			SodiumCurrent:    128, // Severe hyponatraemia <132
			SodiumTimestamp:   daysAgo(0),
			PotassiumCurrent: 3.8,
			PotassiumTimestamp: daysAgo(0),
			CreatinineCurrent: 95,
			EGFR:              58,
			SBP:               130,
			DBP:               80,
		},
		Context: types.TitrationContext{
			CurrentDose:    14.0,
			EGFRCurrent:   58,
			ThiazideActive: true,
			Season:         "SUMMER",
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.80,
		LoopTrust: 0.75,
	}
}

// ========================================================================
// SCENARIO 9: GREEN Trajectory — Stable, Ready for Normal Titration
// Expected: All channels CLEAR. Dose applied normally.
// Tests: Happy path — the system correctly titrates when safe.
// ========================================================================
func GreenTrajectory() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-GREEN-001",
		Archetype:   "green_trajectory",
		Description: "Stable patient. All labs normal. Glucose slightly above target. System should titrate normally.",
		Labs: types.RawPatientData{
			PatientID:         "SIM-GREEN-001",
			Timestamp:         time.Now(),
			GlucoseCurrent:    8.5, // Above target — should trigger +2U
			GlucosePrevious:   8.8,
			GlucoseTimestamp:   hoursAgo(2),
			PotassiumCurrent:  4.3,
			PotassiumTimestamp: daysAgo(1),
			CreatinineCurrent: 85,
			CreatininePrevious: 83,
			CreatinineTimestamp: daysAgo(1),
			EGFR:               72,
			EGFRTimestamp:       daysAgo(1),
			SBP:                128,
			DBP:                80,
			HeartRate:          72,
			HeartRateRegularity: "REGULAR",
			Weight:             76.0,
			WeightPrevious:    76.2,
			SodiumCurrent:     140,
		},
		Context: types.TitrationContext{
			CurrentDose: 16.0,
			EGFRCurrent: 72,
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.92,
		LoopTrust: 0.90,
	}
}

// ========================================================================
// SCENARIO 10: Metformin + CKD Stage 4
// Expected: PG-01 HALT (metformin contraindicated at eGFR <30).
// Tests: KDIGO absolute contraindication.
// ========================================================================
func MetforminCKD4() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-METCKD-001",
		Archetype:   "metformin_ckd4",
		Description: "eGFR=25, on metformin. PG-01 must fire HALT — KDIGO absolute contraindication.",
		Labs: types.RawPatientData{
			PatientID:        "SIM-METCKD-001",
			Timestamp:        time.Now(),
			GlucoseCurrent:   9.0,
			GlucoseTimestamp:  hoursAgo(3),
			PotassiumCurrent: 5.0,
			PotassiumTimestamp: daysAgo(1),
			CreatinineCurrent: 220,
			EGFR:              25,
			SBP:               142,
			DBP:               88,
		},
		Context: types.TitrationContext{
			CurrentDose: 12.0,
			EGFRCurrent: 25,
			CKDStage:    "4",
			ActiveMedications: []types.ActiveMedication{
				{DrugClass: "METFORMIN", Dose: 500},
			},
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.85,
		LoopTrust: 0.80,
	}
}

// AllScenarios returns all virtual patients for the standard test suite.
func AllScenarios() []VirtualPatient {
	return []VirtualPatient{
		ActiveHypoglycaemia(),
		AKIMidTitration(),
		RAASCreatinineTolerance(),
		DataDropOut(),
		NonAdherentPatient(),
		JCurveCKD3b(),
		DualRAAS(),
		HyponatraemiaThiazide(),
		GreenTrajectory(),
		MetforminCKD4(),
	}
}
