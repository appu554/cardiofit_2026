// Package channel_b implements the PhysiologySafetyMonitor (SA-02).
//
// CRITICAL BUILD CONSTRAINT:
// This package must NEVER import:
//   - vmcu (parent) — no GateSignal dependency at import level
//   - channel_a     — Channel B is independent of diagnostic reasoning
//   - titration      — Channel B knows nothing about dose computation
//   - Any KB-22 or KB-23 package
//
// Channel B operates on RAW lab values from KB-20 only.
// It does not see MetabolicState, HyperglycaemiaMechanism, ISF, or any
// derived clinical intelligence.
package channel_b

import "time"

// RawPatientData contains only unprocessed lab values from KB-20.
// This struct must never contain MetabolicState, HyperglycaemiaMechanism,
// ISF, or any KB-22/KB-23 derived field.
type RawPatientData struct {
	// Current lab values (raw from KB-20)
	// SAFETY: All current values use *float64 so that absent data (nil)
	// is distinguishable from a true zero reading. A bare float64 zero-value
	// caused false B-01 HALT when glucose was simply not provided.
	GlucoseCurrent    *float64  // mmol/L, nil if not available
	GlucoseTimestamp  time.Time // when glucose was measured
	CreatinineCurrent *float64  // µmol/L, nil if not available
	PotassiumCurrent  *float64  // mEq/L, nil if not available
	SBPCurrent        *float64  // mmHg, nil if not available
	WeightKgCurrent   *float64  // kg, nil if not available
	EGFRCurrent       *float64  // mL/min/1.73m², nil if not available
	EGFRSlope         *float64  // mL/min/1.73m² per year, nil if unavailable
	HbA1cCurrent      *float64  // %, nil if not available

	// Historical values (for delta computation)
	Creatinine48hAgo *float64 // µmol/L, nil if unavailable
	EGFRPrior48h     *float64 // mL/min/1.73m², nil if unavailable
	HbA1cPrior30d    *float64 // %, nil if unavailable
	Weight72hAgo     *float64 // kg, nil if unavailable

	// ── HTN co-management extensions (Wave 1) ──

	// BP extensions
	SBPLowerLimit *float64 // mmHg, computed by orchestrator from eGFR-stratified J-curve (B-12)
	SodiumCurrent *float64 // mEq/L, nil if not available
	DBPCurrent    *float64 // mmHg, nil if not available

	// RAAS creatinine tolerance context (PG-14 → B-03 suppression)
	// When true, a creatinine delta >26 µmol/L is downgraded from HALT→PAUSE
	// because it is explained by expected RAAS blockade pharmacodynamics.
	CreatinineRiseExplained bool
	OliguriaReported       bool // clinician-reported oliguria overrides RAAS tolerance

	// BP pattern and measurement uncertainty (from KB-20 BPTrajectory)
	BPPattern              string  // matches BPPattern constants in KB-20
	MeasurementUncertainty float64 // σ mmHg — dampens dose change near thresholds

	// ── Heart rate extensions (Wave 2, Amendment 3) ──
	HeartRateCurrent           *float64 // bpm, nil if not available
	HRRegularity               string   // REGULAR | IRREGULAR | UNKNOWN
	HRContext                  string   // RESTING | POST_ACTIVITY | STANDING | SUPINE
	HeartRateConfirmed         bool     // true if 2 consecutive readings within 10%
	BetaBlockerActive              bool     // from KB-20 MedicationFact
	BetaBlockerDoseChangeIn7d      bool     // true if beta-blocker dose changed in last 7 days
	BetaBlockerPerturbationActive  bool     // true if BB perturbation window is active (recently started/uptitrated)
	ThiazideActive             bool     // from KB-20 MedicationFact (for B-17/B-18/B-19)

	// Season (from KB-20 PatientProfile)
	Season string // SUMMER|MONSOON|WINTER|AUTUMN|UNKNOWN

	// CKD stage for J-curve stratification (from KB-20)
	CKDStage string // e.g., "3a", "3b", "4", "5"

	// Glucose trend (last 3 readings, most recent first)
	GlucoseReadings []TimestampedValue

	// Dose context (from V-MCU internal dose_history, not KB-23)
	RecentDoseIncrease bool

	// ── Treatment perturbation windows (Amendment 2) ──
	// Populated by the orchestrator from KB-20 medication events.
	// Each entry represents an active drug class change with an expected
	// pharmacodynamic shift window. Channel B uses these to dampen
	// false safety gates on expected signal movements.
	ActivePerturbations []TreatmentPerturbation
}

// TimestampedValue pairs a measurement with its timestamp.
type TimestampedValue struct {
	Value     float64
	Timestamp time.Time
}

// Float64Ptr returns a pointer to the given float64 value.
// Use this when constructing RawPatientData to set current lab values.
func Float64Ptr(v float64) *float64 { return &v }

// PhysioGate is Channel B's local gate type.
// Mapped to vmcu.GateSignal by the orchestrator.
type PhysioGate string

const (
	PhysioClear    PhysioGate = "CLEAR"
	PhysioModify   PhysioGate = "MODIFY"
	PhysioPause    PhysioGate = "PAUSE"
	PhysioHalt     PhysioGate = "HALT"
	PhysioHoldData PhysioGate = "HOLD_DATA"
)

// KB22TriggerRequest is a request to initiate a KB-22 HPI session,
// populated by Channel B sentinels and consumed by the V-MCU orchestrator.
type KB22TriggerRequest struct {
	SentinelID string
	HPINodeID  string
	Data       map[string]interface{}
}

// PhysioResult is the output of Channel B evaluation.
type PhysioResult struct {
	Gate         PhysioGate           `json:"gate"`
	RuleFired    string               `json:"rule_fired,omitempty"`
	RawValues    map[string]float64   `json:"raw_values,omitempty"`
	IsAnomaly    bool                 `json:"is_anomaly"`
	AnomalyLab   string               `json:"anomaly_lab,omitempty"`
	KB22Triggers []KB22TriggerRequest `json:"kb22_triggers,omitempty"`
}
