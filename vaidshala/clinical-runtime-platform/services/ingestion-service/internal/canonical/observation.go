package canonical

import (
	"time"

	"github.com/google/uuid"
)

// CanonicalObservation is the unified internal representation for all clinical
// observations regardless of their ingestion source (EHR, ABDM, lab, device,
// patient-reported, etc.). Every adapter must convert its raw payload into one
// or more CanonicalObservation values before the pipeline continues.
type CanonicalObservation struct {
	ID              uuid.UUID        `json:"id"`
	PatientID       uuid.UUID        `json:"patient_id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	SourceType      SourceType       `json:"source_type"`
	SourceID        string           `json:"source_id"`
	ObservationType ObservationType  `json:"observation_type"`
	LOINCCode       string           `json:"loinc_code"`
	SNOMEDCode      string           `json:"snomed_code,omitempty"`
	Value           float64          `json:"value"`
	ValueString     string           `json:"value_string,omitempty"`
	Unit            string           `json:"unit"`
	Timestamp       time.Time        `json:"timestamp"`
	QualityScore    float64          `json:"quality_score"`
	Flags           []Flag           `json:"flags,omitempty"`
	DeviceContext   *DeviceContext   `json:"device_context,omitempty"`
	ClinicalContext *ClinicalContext `json:"clinical_context,omitempty"`
	ABDMContext     *ABDMContext     `json:"abdm_context,omitempty"`
	RawPayload      []byte           `json:"raw_payload,omitempty"`

	// DataTier classifies the signal tier for CGM/glucose observations.
	DataTier DataTier `json:"data_tier,omitempty"`

	// V4 Signal Schema Extensions (per Flink Architecture §7.1–7.3)
	SourceProtocol        string  `json:"source_protocol,omitempty"`          // S1 FBG: Tier 3 rotating meal protocol
	LinkedMealID          string  `json:"linked_meal_id,omitempty"`           // S2 PPBG: links to S4 meal log for MealResponseCorrelator
	SodiumEstimatedMg     float64 `json:"sodium_estimated_mg,omitempty"`      // S4 Meal: auto-computed from IFCT/AUSNUT
	PreparationMethod     string  `json:"preparation_method,omitempty"`       // S4 Meal: RAW, BOILED, FRIED, etc.
	FoodNameLocal         string  `json:"food_name_local,omitempty"`          // S4 Meal: regional language food name
	SymptomAwareness      *bool   `json:"symptom_awareness,omitempty"`        // S6 Hypo: CID-03 masking detection
	BPDeviceType          string  `json:"bp_device_type,omitempty"`           // S7 BP: oscillometric_cuff, cuffless_ppg, etc.
	ClinicalGrade         *bool   `json:"clinical_grade,omitempty"`           // S7 BP: validated vs consumer-grade device
	MeasurementMethod     string  `json:"measurement_method,omitempty"`       // S7 BP: auscultatory, oscillometric, cuffless
	LinkedSeatedReadingID string  `json:"linked_seated_reading_id,omitempty"` // S8 BP standing: orthostatic delta
	WakingTime            string  `json:"waking_time,omitempty"`              // S9/S10 BP: HH:MM for surge window
	SleepTime             string  `json:"sleep_time,omitempty"`               // S9/S10 BP: HH:MM for nocturnal window
}

// SourceType identifies the originating system for an observation.
type SourceType string

const (
	SourceEHR             SourceType = "EHR"
	SourceABDM            SourceType = "ABDM"
	SourceLab             SourceType = "LAB"
	SourcePatientReported SourceType = "PATIENT_REPORTED"
	SourceDevice          SourceType = "DEVICE"
	SourceWearable        SourceType = "WEARABLE"
)

// ObservationType classifies the clinical domain of an observation.
type ObservationType string

const (
	ObsVitals          ObservationType = "VITALS"
	ObsLabs            ObservationType = "LABS"
	ObsMedications     ObservationType = "MEDICATIONS"
	ObsPatientReported    ObservationType = "PATIENT_REPORTED"
	ObsDeviceData         ObservationType = "DEVICE_DATA"
	ObsABDMRecords        ObservationType = "ABDM_RECORDS"
	ObsWearableAggregates ObservationType = "WEARABLE_AGGREGATES"
	ObsCGMRaw             ObservationType = "CGM_RAW" // S24
	ObsGeneral            ObservationType = "GENERAL"

	// V4 Signal Types — numbering per NorthStar Architecture §2.1
	ObsSodiumEstimate     ObservationType = "SODIUM_ESTIMATE"     // S23
	ObsInterventionEvent  ObservationType = "INTERVENTION_EVENT"  // S25
	ObsPhysicianFeedback  ObservationType = "PHYSICIAN_FEEDBACK"  // S26
	ObsWaistCircumference ObservationType = "WAIST_CIRCUMFERENCE" // S27
	ObsExerciseSession    ObservationType = "EXERCISE_SESSION"    // S28
	ObsMoodStress         ObservationType = "MOOD_STRESS"         // S29
)

// DataTier classifies glucose data quality tiers for Flink processing.
type DataTier string

const (
	DataTierCGM    DataTier = "TIER_1_CGM"
	DataTierHybrid DataTier = "TIER_2_HYBRID"
	DataTierSMBG   DataTier = "TIER_3_SMBG"
)

// DeviceContext carries metadata about the originating device, applicable
// when SourceType is DEVICE or WEARABLE.
type DeviceContext struct {
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	FirmwareVer  string `json:"firmware_version,omitempty"`
}

// ClinicalContext carries encounter-level metadata when the observation
// originates from a clinical encounter or order.
type ClinicalContext struct {
	EncounterID string `json:"encounter_id,omitempty"`
	OrderID     string `json:"order_id,omitempty"`
	Method      string `json:"method,omitempty"`
	BodySite    string `json:"body_site,omitempty"`
}

// ABDMContext carries Ayushman Bharat Digital Mission consent and request
// metadata for observations received via the ABDM health data exchange.
type ABDMContext struct {
	ConsentID    string `json:"consent_id"`
	HIURequestID string `json:"hiu_request_id"`
	CareContext  string `json:"care_context,omitempty"`
}
