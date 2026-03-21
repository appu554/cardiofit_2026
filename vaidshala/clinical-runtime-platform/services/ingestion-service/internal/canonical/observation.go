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
}

// SourceType identifies the originating system for an observation.
type SourceType string

const (
	SourceEHR             SourceType = "EHR"
	SourceABDM            SourceType = "ABDM"
	SourceLab             SourceType = "LAB"
	SourcePatientReported SourceType = "PATIENT_REPORTED"
	SourceHPI             SourceType = "HPI"
	SourceDevice          SourceType = "DEVICE"
	SourceWearable        SourceType = "WEARABLE"
)

// ObservationType classifies the clinical domain of an observation.
type ObservationType string

const (
	ObsVitals          ObservationType = "VITALS"
	ObsLabs            ObservationType = "LABS"
	ObsMedications     ObservationType = "MEDICATIONS"
	ObsPatientReported ObservationType = "PATIENT_REPORTED"
	ObsHPI             ObservationType = "HPI"
	ObsDeviceData      ObservationType = "DEVICE_DATA"
	ObsABDMRecords     ObservationType = "ABDM_RECORDS"
	ObsGeneral         ObservationType = "GENERAL"
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
