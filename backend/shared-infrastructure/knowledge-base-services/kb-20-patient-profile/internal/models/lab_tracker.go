package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LabEntry stores a single lab measurement with validation status.
// Finding F-05 (RED): All lab writes go through plausibility validation.
type LabEntry struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string          `gorm:"size:100;not null;index" json:"patient_id"`
	LabType   string          `gorm:"size:30;not null;index" json:"lab_type"`
	Value     decimal.Decimal `gorm:"type:decimal(10,4);not null" json:"value"`
	Unit      string          `gorm:"size:20;not null" json:"unit"`

	// Timing
	MeasuredAt time.Time `gorm:"not null;index" json:"measured_at"`
	Source     string    `gorm:"size:50" json:"source,omitempty"`

	// Derived flag — true for auto-computed values like eGFR
	IsDerived bool `gorm:"default:false" json:"is_derived"`

	// Validation status (F-05)
	ValidationStatus string `gorm:"size:20;not null;default:'ACCEPTED';check:validation_status IN ('ACCEPTED','FLAGGED','REJECTED')" json:"validation_status"`
	FlagReason       string `gorm:"size:200" json:"flag_reason,omitempty"`

	// FHIR integration
	LOINCCode         string `gorm:"size:20;column:loinc_code" json:"loinc_code,omitempty"`
	FHIRObservationID string `gorm:"size:200;index;column:fhir_observation_id" json:"fhir_observation_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// Lab type constants
const (
	LabTypeCreatinine       = "CREATININE"
	LabTypeEGFR             = "EGFR"
	LabTypeFBG              = "FBG"
	LabTypeHbA1c            = "HBA1C"
	LabTypeSBP              = "SBP"
	LabTypeDBP              = "DBP"
	LabTypePotassium        = "POTASSIUM"
	LabTypeTotalCholesterol = "TOTAL_CHOLESTEROL"
	LabTypeSodium           = "SODIUM"
	LabTypeACR              = "ACR" // albumin-to-creatinine ratio (mg/mmol)
)

// Validation status constants
const (
	ValidationAccepted = "ACCEPTED"
	ValidationFlagged  = "FLAGGED"
	ValidationRejected = "REJECTED"
)

// EGFRTrajectoryPoint is a computed eGFR value with its timestamp.
type EGFRTrajectoryPoint struct {
	Value      float64   `json:"value"`
	MeasuredAt time.Time `json:"measured_at"`
	CKDStage   string    `json:"ckd_stage"`
}

// EGFRTrajectoryResponse contains the eGFR history and trend classification.
type EGFRTrajectoryResponse struct {
	PatientID    string                `json:"patient_id"`
	Points       []EGFRTrajectoryPoint `json:"points"`
	Trend        string                `json:"trend"`
	AnnualChange *float64              `json:"annual_change_ml_min,omitempty"`
}

// BPPattern classifies long-term BP behaviour from the last 4 weeks of readings.
type BPPattern string

const (
	BPPatternUnknown       BPPattern = "UNKNOWN"        // insufficient data (<5 readings)
	BPPatternControlled    BPPattern = "CONTROLLED"     // mean SBP <130 AND >80% readings in target
	BPPatternSustainedHigh BPPattern = "SUSTAINED_HIGH" // mean SBP ≥140 over 2+ weeks
	BPPatternWhiteCoat     BPPattern = "WHITE_COAT"     // clinic high + home normal (ABPM/HBPM)
	BPPatternMasked        BPPattern = "MASKED"         // clinic normal + home high (ABPM/HBPM)
	BPPatternDippingAbsent BPPattern = "NON_DIPPER"     // <10% nocturnal SBP dip
	BPPatternMorningHTN    BPPattern = "MORNING_SURGE"  // morning SBP >135 + ≥20 above nocturnal mean
	BPPatternResistant     BPPattern = "RESISTANT"      // ≥3 agents at max tolerated + uncontrolled
)

// BPStatus classifies the current BP control tier for early warning logic (EW-03).
type BPStatus string

const (
	BPStatusAtTarget    BPStatus = "AT_TARGET"
	BPStatusEarlyWatch  BPStatus = "EARLY_WATCH"  // slope positive, sustained, below DECLINING threshold
	BPStatusDeclining   BPStatus = "DECLINING"    // slope exceeds stratum-specific declining threshold
	BPStatusAboveTarget BPStatus = "ABOVE_TARGET" // mean SBP above target but not yet DECLINING
	BPStatusSevere      BPStatus = "SEVERE"       // mean SBP >= 180 mmHg
	BPStatusUrgency     BPStatus = "URGENCY"      // Any single clinic reading >= 180 AND symptoms (headache/visual disturbance)
	BPStatusHypotensive BPStatus = "HYPOTENSIVE"  // sbp_7d_mean < 100 OR orthostatic_drop < -20 mmHg
)

// SBPSevereThreshold is the SBP value at which status escalates to SEVERE (EW-05).
const SBPSevereThreshold = 180.0

// BP variability status constants (Wave 3.1 Amendment 7).
const (
	VariabilityLow      = "LOW"      // SD < 10 mmHg
	VariabilityModerate = "MODERATE" // SD 10-15 mmHg
	VariabilityHigh     = "HIGH"     // SD > 15 mmHg
)

// Pulse pressure trend constants (Wave 3.4 Amendment 13).
const (
	PulsePressureTrendWidening  = "WIDENING"
	PulsePressureTrendStable    = "STABLE"
	PulsePressureTrendNarrowing = "NARROWING"
)

// DamageAlertCooldownHours is the hysteresis cooldown period after a damage
// composite alert is emitted — no re-emission at the same or lower score
// within this window (Wave 2 Track C).
const DamageAlertCooldownHours = 72

// SlopeConfidence classifies data adequacy for the 4-week SBP slope (EW-06).
type SlopeConfidence string

const (
	SlopeConfidenceHigh     SlopeConfidence = "HIGH"     // >= 8 readings in last 4 weeks
	SlopeConfidenceModerate SlopeConfidence = "MODERATE" // 5-7 readings
	SlopeConfidenceLow      SlopeConfidence = "LOW"      // < 5 readings
)

// BPTrajectory holds the computed BP analysis for a patient.
type BPTrajectory struct {
	PatientID              string    `json:"patient_id"`
	Pattern                BPPattern `json:"pattern"`
	MeanSBP28d             *float64  `json:"mean_sbp_28d,omitempty"`
	MeanDBP28d             *float64  `json:"mean_dbp_28d,omitempty"`
	ReadingsInTarget       int       `json:"readings_in_target"`
	TotalReadings28d       int       `json:"total_readings_28d"`
	MeasurementUncertainty float64   `json:"measurement_uncertainty"` // σ from device+operator variation
	ComputedAt             time.Time `json:"computed_at"`

	// --- EW-01/02: Risk-stratified declining thresholds ---
	BPRiskStratum         string  `json:"bp_risk_stratum"`         // DM_ONLY_A1 | DM_CKD3A_A1 | DM_CKD3B_ANY | DM_CKD_A2A3
	SBPDecliningThreshold float64 `json:"sbp_declining_threshold"` // per-patient mmHg/week from risk stratum table

	// --- EW-03: EARLY_WATCH tracking ---
	Status                     BPStatus `json:"bp_status"`
	ConsecutiveEarlyWatchWeeks int      `json:"consecutive_early_watch_weeks"` // counter, resets when slope reverses
	EarlyWatchFloor            float64  `json:"early_watch_floor"`             // stratum-specific minimum positive slope to qualify
	EarlyWatchWeeksThreshold   int      `json:"early_watch_weeks_threshold"`   // stratum-specific weeks before concern event

	// --- EW-05/06: Time-to-severe projection ---
	SBP4wSlope      *float64        `json:"sbp_4w_slope_mmhg_per_week,omitempty"` // linear regression slope over 28d readings
	SBP7dMean       *float64        `json:"sbp_7d_mean,omitempty"`                // rolling 7-day SBP mean
	WeeksToSevere   *float64        `json:"weeks_to_severe,omitempty"`            // (180 - sbp_7d_mean) / slope; nil if slope <= 0
	SlopeConfidence SlopeConfidence `json:"slope_confidence"`                     // HIGH | MODERATE | LOW

	// --- Wave 3.1 Amendment 7: Visit-to-visit BP variability ---
	SBPVisitVariability *float64 `json:"sbp_visit_variability"` // SD of SBP across last 5 visits
	DBPVisitVariability *float64 `json:"dbp_visit_variability"` // SD of DBP across last 5 visits
	VariabilityStatus   string   `json:"variability_status"`    // LOW | MODERATE | HIGH

	// --- Wave 3.4 Amendment 13: Pulse pressure tracking ---
	PulsePressureMean  *float64 `json:"pulse_pressure_mean"`  // mean PP over last 5 readings
	PulsePressureTrend string   `json:"pulse_pressure_trend"` // WIDENING | STABLE | NARROWING

	// --- Wave 2 Track G: SBP slope acceleration ---
	SBPSlopeAcceleration *float64 `json:"sbp_slope_acceleration,omitempty"` // second derivative of SBP trajectory (mmHg/week²)

	// --- Wave 2 Track C: Damage composite hysteresis ---
	LastDamageAlertScore int        `json:"last_damage_alert_score"`
	LastDamageAlertTime  *time.Time `json:"last_damage_alert_time"`

	// --- EW-07/08: Compound damage composite ---
	DamageScore *DamageComposite `json:"damage_composite,omitempty"`

	// --- HTN Integration: Orthostatic and target tracking ---
	OrthostaticDrop      *float64 `json:"orthostatic_drop,omitempty"`        // SBP standing minus SBP seated (negative = drop). < -20 = clinically significant
	LastClinicReadingSBP *float64 `json:"last_clinic_reading_sbp,omitempty"` // Most recent CLINIC-context SBP reading
	BPTargetSBP          *float64 `json:"bp_target_sbp,omitempty"`           // Patient-stratum SBP target (from KDIGO/ADA by eGFR+ACR+age)
	BPTargetDBP          *float64 `json:"bp_target_dbp,omitempty"`           // Patient-stratum DBP target

	// --- v2 reserved fields for non-renal damage markers (EW-09, Wave 3.5) ---
	// These fields are not yet populated; reserved for future data source integration.
	LVHStatus           string `json:"lvh_status,omitempty"`            // reserved: from ECG/ECHO — left ventricular hypertrophy
	RetinopathyGrade    string `json:"retinopathy_grade,omitempty"`     // reserved: from OPHTHALMOLOGY — Keith-Wagener-Barker grade
	CognitiveChangeFlag bool   `json:"cognitive_change_flag,omitempty"` // reserved: from COGNITIVE_SCREENING — cognitive decline flag
}

// DamageComposite is the compound damage concern score (EW-07/08).
// Score range 0-8; thresholds: 3-4 = BP_SUBCLINICAL_CONCERN, >= 5 = DAMAGE_COMPOSITE_ALERT.
type DamageComposite struct {
	Score                int       `json:"score"`                  // 0-8
	VariabilityContrib   int       `json:"variability_contrib"`    // 0-2 (1 for MODERATE, 2 for HIGH)
	ACRTrendContrib      int       `json:"acr_trend_contrib"`      // 0-2 (1 for WORSENING, 2 for A2->A3)
	PulsePressureContrib int       `json:"pulse_pressure_contrib"` // 0-2 (1 for >60+WIDENING, 2 for >80)
	BPStatusContrib      int       `json:"bp_status_contrib"`      // 0-2 (2 for ABOVE_TARGET >=8w + adherence >=0.85)
	ComputedAt           time.Time `json:"computed_at"`
}

// TimestampedReading represents a single timestamped numeric measurement.
// Used by FBGTracking and other longitudinal trackers.
type TimestampedReading struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// FBGTracking maintains rolling FBG readings for trajectory analysis.
type FBGTracking struct {
	PatientID string               `gorm:"primaryKey" json:"patient_id"`
	Readings  []TimestampedReading `json:"readings" gorm:"serializer:json"`
	Trend     string               `json:"trend"`
	SlopePerQ float64              `json:"slope_per_q"`
	CV7d      float64              `json:"cv_7d"`
	CV14d     float64              `json:"cv_14d"`
	CV30d     float64              `json:"cv_30d"`
	OnInsulin bool                 `json:"on_insulin"`
	UpdatedAt time.Time            `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time            `gorm:"autoCreateTime" json:"created_at"`
}

// CDIScore is the Composite Deterioration Index: cross-domain (0-20).
// BP domain (0-8) from existing DamageComposite.
// Glycaemic domain (0-6): FBG trend + HbA1c trend + glucose CV%.
// Renal domain (0-6): eGFR slope + ACR trend + creatinine trajectory.
type CDIScore struct {
	PatientID        string    `gorm:"primaryKey" json:"patient_id"`
	TotalScore       int       `json:"total_score"`        // 0-20
	BPComponentScore int       `json:"bp_component_score"` // 0-8
	GlycaemicScore   int       `json:"glycaemic_score"`    // 0-6
	RenalScore       int       `json:"renal_score"`        // 0-6
	RiskLevel        string    `json:"risk_level"`         // LOW (0-6) | MODERATE (7-12) | HIGH (13-16) | CRITICAL (17-20)
	ActiveDomains    string    `json:"active_domains"`     // comma-separated: "BP,GLYCAEMIC,RENAL"
	ComputedAt       time.Time `json:"computed_at"`
	CooldownUntil    time.Time `json:"cooldown_until"` // 72h hysteresis
	LastAlertScore   int       `json:"last_alert_score"`
}

// BPRiskStratumEntry defines a row in the risk stratum table for BP early warning thresholds.
type BPRiskStratumEntry struct {
	Label              string  // human-readable label
	DecliningThreshold float64 // SBP slope mmHg/week above which DECLINING fires
	EarlyWatchFloor    float64 // minimum positive slope to qualify for EARLY_WATCH
	EarlyWatchWeeks    int     // consecutive weeks in EARLY_WATCH before concern event
}

// BPRiskStratumTable maps stratum keys to their threshold parameters (EW-01/02).
// Keyed by composite of CKD stage and ACR category from patient profile.
var BPRiskStratumTable = map[string]BPRiskStratumEntry{
	"DM_ONLY_A1": {
		Label:              "DM only, ACR A1",
		DecliningThreshold: 2.5,
		EarlyWatchFloor:    1.8,
		EarlyWatchWeeks:    6,
	},
	"DM_CKD3A_A1": {
		Label:              "DM + CKD 3a, ACR A1",
		DecliningThreshold: 2.0,
		EarlyWatchFloor:    1.4,
		EarlyWatchWeeks:    4,
	},
	"DM_CKD3B_ANY": {
		Label:              "DM + CKD 3b, any ACR",
		DecliningThreshold: 1.5,
		EarlyWatchFloor:    1.0,
		EarlyWatchWeeks:    3,
	},
	"DM_CKD_A2A3": {
		Label:              "DM + any CKD, ACR A2/A3",
		DecliningThreshold: 1.5,
		EarlyWatchFloor:    0.0, // any positive slope qualifies
		EarlyWatchWeeks:    6,
	},
}

// RAASChangeRecency tracks when the last ACEi/ARB dose change happened,
// so V-MCU can apply the RAAS creatinine tolerance window (PG-14).
type RAASChangeRecency struct {
	LastACEiARBChangeAt   *time.Time `json:"last_acei_arb_change_at,omitempty"`
	DaysSinceChange       int        `json:"days_since_change"`
	InitiationOrTitration string     `json:"initiation_or_titration"` // "INITIATION" | "TITRATION" | "NONE"
}

// BPMeasurementContext captures contextual metadata for BP readings.
// This metadata enables Channel B to apply context-aware safety rules:
//   - MeasurementContext: CLINIC readings may differ from HOME (white-coat effect)
//   - Posture: STANDING BP is systematically lower than SEATED
//   - ConsecutiveReadingIndex: NICE HTN 2023 recommends averaging 2-3 readings
//   - MeasurementUncertainty: HIGH when irregular HR, postural change, or single reading
type BPMeasurementContext struct {
	MeasurementContext       string `json:"measurement_context,omitempty"`         // CLINIC | HOME | AMBULATORY | PHARMACY
	Posture                  string `json:"posture,omitempty"`                     // SEATED | STANDING | SUPINE | UNKNOWN
	Arm                      string `json:"arm,omitempty"`                         // LEFT | RIGHT | BOTH | UNKNOWN
	TimeOfDay                string `json:"time_of_day,omitempty"`                 // MORNING_FASTING | MORNING_POST_MED | AFTERNOON | EVENING | NOCTURNAL
	ConsecutiveReadingIndex  int    `json:"consecutive_reading_index,omitempty"`   // 0-based; 0 = first reading, 1 = second
	MinutesSinceLastActivity *int   `json:"minutes_since_last_activity,omitempty"` // nil = unknown
	WhiteCoatFlag            bool   `json:"white_coat_flag,omitempty"`             // historical white-coat pattern
	MeasurementUncertainty   string `json:"measurement_uncertainty,omitempty"`     // LOW | MODERATE | HIGH
}

// ACRReading represents a single ACR measurement.
type ACRReading struct {
	ValueMgMmol     float64   `json:"value_mg_mmol" gorm:"not null"`
	CollectedAt     time.Time `json:"collected_at" gorm:"not null"`
	UrineCollection string    `json:"urine_collection"` // SPOT | 24H
}

// ACRTracking holds longitudinal ACR trajectory for a patient.
// Readings are stored as JSONB via GORM's serializer:json tag.
type ACRTracking struct {
	PatientID string       `json:"patient_id" gorm:"primaryKey"`
	Readings  []ACRReading `json:"readings" gorm:"serializer:json"`
	Trend     string       `json:"trend"`    // IMPROVING | STABLE | WORSENING
	Category  string       `json:"category"` // A1 | A2 | A3
	OnRAAS    bool         `json:"on_raas"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ACR category constants (KDIGO 2024)
const (
	ACRCategoryA1 = "A1" // < 3 mg/mmol (normal to mildly increased)
	ACRCategoryA2 = "A2" // 3-30 mg/mmol (moderately increased)
	ACRCategoryA3 = "A3" // > 30 mg/mmol (severely increased)
)

// ACR trend constants
const (
	ACRTrendImproving = "IMPROVING"
	ACRTrendStable    = "STABLE"
	ACRTrendWorsening = "WORSENING"
)

// CategorizeACR returns the KDIGO category for a given ACR value in mg/mmol.
//
//	A1: < 3 mg/mmol (normal to mildly increased)
//	A2: 3-30 mg/mmol (moderately increased)
//	A3: > 30 mg/mmol (severely increased)
func CategorizeACR(valueMgMmol float64) string {
	switch {
	case valueMgMmol < 3:
		return ACRCategoryA1
	case valueMgMmol <= 30:
		return ACRCategoryA2
	default:
		return ACRCategoryA3
	}
}

// ACRCategoryOrdinal returns a numeric ordinal for category comparison.
// A1=1, A2=2, A3=3. Unknown returns 0.
func ACRCategoryOrdinal(category string) int {
	switch category {
	case ACRCategoryA1:
		return 1
	case ACRCategoryA2:
		return 2
	case ACRCategoryA3:
		return 3
	default:
		return 0
	}
}

// AddLabRequest is the JSON body for POST /patient/:id/labs.
type AddLabRequest struct {
	LabType    string  `json:"lab_type" binding:"required"`
	Value      float64 `json:"value" binding:"required"`
	Unit       string  `json:"unit" binding:"required"`
	MeasuredAt string  `json:"measured_at" binding:"required"`
	Source     string  `json:"source,omitempty"`

	// BP-specific context (only populated for SBP/DBP lab types)
	BPContext *BPMeasurementContext `json:"bp_context,omitempty"`
}
