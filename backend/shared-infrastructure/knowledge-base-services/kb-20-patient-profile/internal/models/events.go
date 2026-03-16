package models

import (
	"time"
)

// Event types published by KB-20 to the event bus.
const (
	EventStratumChange             = "STRATUM_CHANGE"
	EventSafetyAlert               = "SAFETY_ALERT"
	EventMedicationThresholdCrossed = "MEDICATION_THRESHOLD_CROSSED" // F-03 RED
	EventMedicationChange          = "MEDICATION_CHANGE"
	EventLabResult                  = "LAB_RESULT"
	EventRAASMonitoringEscalate     = "RAAS_MONITORING_ESCALATE"     // creatinine rise >20% post-ACEi/ARB not explained
	EventRAASMonitoringResolved     = "RAAS_MONITORING_RESOLVED"     // creatinine stabilised within tolerance
	EventResistantHTNDetected       = "RESISTANT_HTN_DETECTED"       // ≥3 agents at max tolerated + uncontrolled BP
	EventBPTrajectoryConcern        = "BP_TRAJECTORY_CONCERN"        // sustained elevated or J-curve proximity
	EventBPSubclinicalConcern       = "BP_SUBCLINICAL_CONCERN"       // EW-07: damage composite score 3-4
	EventDamageCompositeAlert       = "DAMAGE_COMPOSITE_ALERT"       // EW-08: damage composite score >= 5
	EventACRWorsening               = "ACR_WORSENING"                // ACR trend changed to WORSENING
	EventACRTargetMet               = "ACR_TARGET_MET"               // ACR category improved (e.g., A3 -> A2)
	EventBPVariabilityAlert         = "BP_VARIABILITY_ALERT"         // Wave 3.1: BP variability transitioned to HIGH

	// Glycaemic domain events
	EventFBGWorsening               = "FBG_WORSENING"
	EventFBGTargetMet               = "FBG_TARGET_MET"
	EventGlucoseVariabilityHigh     = "GLUCOSE_VARIABILITY_HIGH"
	EventGlucoseVariabilityResolved = "GLUCOSE_VARIABILITY_RESOLVED"

	// HTN Proposal §3.3 — Core BP events
	EventBPAlert           = "BP_ALERT"              // bp_status transitions to ABOVE_TARGET or DECLINING
	EventBPSevereAlert     = "BP_SEVERE_ALERT"       // bp_status transitions to SEVERE
	EventBPUrgencyAlert    = "BP_URGENCY_ALERT"      // bp_status = URGENCY — immediate notification
	EventBPControlled      = "BP_CONTROLLED"          // bp_status transitions to AT_TARGET (first time or after ABOVE_TARGET)
	EventOrthostaticAlert  = "ORTHOSTATIC_ALERT"      // orthostatic_drop < -20 mmHg confirmed 2 readings
	EventMaskedHTNDetected = "MASKED_HTN_DETECTED"    // bp_pattern = MASKED confirmed over 4+ paired readings
)

// Event is the base envelope for all KB-20 events.
type Event struct {
	EventType string      `json:"event_type"`
	PatientID string      `json:"patient_id"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// StratumChangePayload is published when a patient's stratum label changes.
type StratumChangePayload struct {
	OldStratum    string `json:"old_stratum"`
	NewStratum    string `json:"new_stratum"`
	OldCKDSubstage string `json:"old_ckd_substage,omitempty"`
	NewCKDSubstage string `json:"new_ckd_substage,omitempty"`
	Trigger       string `json:"trigger"`
}

// SafetyAlertPayload is published for clinically significant safety events.
type SafetyAlertPayload struct {
	Severity    string `json:"severity"`
	AlertType   string `json:"alert_type"`
	Description string `json:"description"`
	LabType     string `json:"lab_type,omitempty"`
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
}

// MedicationThresholdCrossedPayload (F-03 RED) fires when eGFR crosses any
// medication-relevant boundary (60, 45, 30, 15) regardless of stratum change.
type MedicationThresholdCrossedPayload struct {
	Lab                string                  `json:"lab"`
	OldValue           float64                 `json:"old_value"`
	NewValue           float64                 `json:"new_value"`
	ThresholdCrossed   float64                 `json:"threshold_crossed"`
	AffectedMedications []AffectedMedication   `json:"affected_medications"`
}

// AffectedMedication describes a drug affected by a threshold crossing.
type AffectedMedication struct {
	DrugClass      string   `json:"drug_class"`
	RequiredAction string   `json:"required_action"`
	MaxDoseMg      *float64 `json:"max_dose_mg,omitempty"`
}

// MedicationChangePayload is published when a medication is added, updated, or removed.
type MedicationChangePayload struct {
	ChangeType string `json:"change_type"`
	DrugName   string `json:"drug_name"`
	DrugClass  string `json:"drug_class"`
	OldDoseMg  string `json:"old_dose_mg,omitempty"`
	NewDoseMg  string `json:"new_dose_mg,omitempty"`
}

// RAASMonitoringEscalatePayload fires when creatinine rise post-ACEi/ARB
// exceeds tolerance and is not explained by the RAAS causal model.
type RAASMonitoringEscalatePayload struct {
	CreatinineRisePct   float64 `json:"creatinine_rise_pct"`
	DaysSinceRAASChange int     `json:"days_since_raas_change"`
	PotassiumCurrent    float64 `json:"potassium_current"`
	RequiredAction      string  `json:"required_action"` // "RECHECK_48H" | "HOLD_AND_REFER"
}

// BPTrajectoryConcernPayload fires when BP trajectory crosses concern thresholds (EW-04).
// This replaces the original pattern-only payload with risk-stratified early warning data.
type BPTrajectoryConcernPayload struct {
	PatientID                  string  `json:"patient_id"`
	SBPSlope                   float64 `json:"sbp_slope_mmhg_per_week"`
	ConsecutiveEarlyWatchWeeks int     `json:"consecutive_early_watch_weeks"`
	BPRiskStratum              string  `json:"bp_risk_stratum"`
	EarlyWatchThreshold        float64 `json:"early_watch_threshold"`
	Pattern                    string  `json:"pattern"`
	MeanSBP28d                 float64 `json:"mean_sbp_28d"`
	ReadingsUsed               int     `json:"readings_used"`
	Suggestion                 string  `json:"suggestion"`
}

// DamageCompositePayload fires for BP_SUBCLINICAL_CONCERN (score 3-4) and
// DAMAGE_COMPOSITE_ALERT (score >= 5) events (EW-07/08).
type DamageCompositePayload struct {
	PatientID            string `json:"patient_id"`
	Score                int    `json:"score"`
	VariabilityContrib   int    `json:"variability_contrib"`
	ACRTrendContrib      int    `json:"acr_trend_contrib"`
	PulsePressureContrib int    `json:"pulse_pressure_contrib"`
	BPStatusContrib      int    `json:"bp_status_contrib"`
}

// ResistantHTNDetectedPayload fires when a patient meets the clinical criteria
// for resistant hypertension: BP above target despite 3+ antihypertensive drug
// classes at optimised doses (including at least one diuretic), with adherence
// >= 0.85 sustained for 12+ weeks.
type ResistantHTNDetectedPayload struct {
	PatientID         string    `json:"patient_id"`
	ActiveDrugClasses []string  `json:"active_drug_classes"`
	DiureticClass     string    `json:"diuretic_class"`
	AdherenceScore    float64   `json:"adherence_score"`
	WeeksAboveTarget  int       `json:"weeks_above_target"`
	BPStatus          string    `json:"bp_status"`
	DetectedAt        time.Time `json:"detected_at"`
}

// ACRWorseningPayload fires when ACR trend changes to WORSENING
// (latest > previous by >20% or category stepped up).
type ACRWorseningPayload struct {
	PatientID        string  `json:"patient_id"`
	CurrentValue     float64 `json:"current_value_mg_mmol"`
	PreviousValue    float64 `json:"previous_value_mg_mmol"`
	CurrentCategory  string  `json:"current_category"`
	PreviousCategory string  `json:"previous_category"`
	OnRAAS           bool    `json:"on_raas"`
}

// ACRTargetMetPayload fires when ACR category improves
// (e.g., A3 to A2 or A2 to A1), indicating RAAS therapy benefit.
type ACRTargetMetPayload struct {
	PatientID        string  `json:"patient_id"`
	CurrentValue     float64 `json:"current_value_mg_mmol"`
	CurrentCategory  string  `json:"current_category"`
	PreviousCategory string  `json:"previous_category"`
}

// BPVariabilityAlertPayload fires when visit-to-visit SBP variability
// transitions to HIGH (SD > 15 mmHg over last 5 readings). (Wave 3.1 Amendment 7)
type BPVariabilityAlertPayload struct {
	PatientID         string  `json:"patient_id"`
	SBPSD             float64 `json:"sbp_sd"`
	DBPSD             float64 `json:"dbp_sd"`
	VariabilityStatus string  `json:"variability_status"`
	ReadingCount      int     `json:"reading_count"`
}

// BPAlertPayload carries data for BP_ALERT and BP_SEVERE_ALERT events.
type BPAlertPayload struct {
	PatientID      string   `json:"patient_id"`
	BPStatus       string   `json:"bp_status"`
	SBP7dMean      float64  `json:"sbp_7d_mean"`
	DBP7dMean      float64  `json:"dbp_7d_mean"`
	SBP4wSlope     float64  `json:"sbp_4w_slope"`
	TargetSBP      float64  `json:"target_sbp"`
	TriggerReading *float64 `json:"trigger_reading,omitempty"` // for URGENCY: the specific reading >= 180
}

// OrthostaticAlertPayload carries data for ORTHOSTATIC_ALERT events.
type OrthostaticAlertPayload struct {
	PatientID       string  `json:"patient_id"`
	OrthostaticDrop float64 `json:"orthostatic_drop"`  // negative value
	SeatedSBP       float64 `json:"seated_sbp"`
	StandingSBP     float64 `json:"standing_sbp"`
}

// MaskedHTNPayload carries data for MASKED_HTN_DETECTED events.
type MaskedHTNPayload struct {
	PatientID         string  `json:"patient_id"`
	ClinicSBPMean     float64 `json:"clinic_sbp_mean"`
	HomeDeviceSBPMean float64 `json:"home_device_sbp_mean"`
	PairedReadings    int     `json:"paired_readings_count"`
}

// FBGWorseningPayload fires when FBG trend transitions to WORSENING (GW-01).
type FBGWorseningPayload struct {
	PatientID     string  `json:"patient_id"`
	CurrentFBG    float64 `json:"current_fbg_mmol"`
	SlopePerQ     float64 `json:"slope_per_quarter_mmol"`
	Trend         string  `json:"trend"`
	PreviousTrend string  `json:"previous_trend"`
	OnInsulin     bool    `json:"on_insulin"`
}

// GlucoseVariabilityPayload fires when glucose CV% exceeds threshold (GW-03).
type GlucoseVariabilityPayload struct {
	PatientID string  `json:"patient_id"`
	CV7d      float64 `json:"cv_7d_pct"`
	CV14d     float64 `json:"cv_14d_pct"`
	CV30d     float64 `json:"cv_30d_pct"`
	Window    string  `json:"trigger_window"`
}

// LabResultPayload is published when a lab result is ingested.
type LabResultPayload struct {
	LabType          string  `json:"lab_type"`
	Value            float64 `json:"value"`
	Unit             string  `json:"unit"`
	MeasuredAt       string  `json:"measured_at"`
	Source           string  `json:"source,omitempty"`
	ValidationStatus string  `json:"validation_status"`
	IsDerived        bool    `json:"is_derived"`
}
