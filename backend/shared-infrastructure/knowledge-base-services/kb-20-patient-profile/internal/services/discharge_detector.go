package services

import (
	"fmt"
	"math"
	"time"

	"kb-patient-profile/internal/models"
)

// DischargeInput holds the data needed to register a discharge event.
type DischargeInput struct {
	PatientID        string
	DischargeDate    time.Time
	Source           string // FHIR_ENCOUNTER, MANUAL, PATIENT_REPORTED, ASHA_REPORTED
	FacilityName     string
	FacilityType     string // ACUTE_HOSPITAL, REHAB, AGED_CARE, HITH
	PrimaryDiagnosis string
	LengthOfStayDays int
	Disposition      string // HOME, AGED_CARE_FACILITY, HITH, REHAB
}

// DischargeDetector validates discharge inputs and produces CareTransition records.
// MaxAgeDays is configurable per market — India allows 21 days for ASHA-reported
// discharges that arrive late; Australia uses 14 days (FHIR catches most within hours).
type DischargeDetector struct {
	MaxAgeDays int // default 14, India override 21
}

// NewDischargeDetector creates a detector with the default 14-day rejection window.
func NewDischargeDetector() *DischargeDetector {
	return &DischargeDetector{MaxAgeDays: 14}
}

// NewDischargeDetectorWithConfig creates a detector with a custom age limit.
func NewDischargeDetectorWithConfig(maxAgeDays int) *DischargeDetector {
	if maxAgeDays <= 0 {
		maxAgeDays = 14
	}
	return &DischargeDetector{MaxAgeDays: maxAgeDays}
}

// DetectDischarge validates the input and creates a CareTransition.
func (d *DischargeDetector) DetectDischarge(input DischargeInput) (*models.CareTransition, error) {
	// 1. Validate required fields
	if input.PatientID == "" {
		return nil, fmt.Errorf("patient_id is required")
	}
	if input.DischargeDate.IsZero() {
		return nil, fmt.Errorf("discharge_date is required")
	}

	// 2. Reject if discharge is too old
	daysSinceDischarge := time.Since(input.DischargeDate).Hours() / 24
	if daysSinceDischarge > float64(d.MaxAgeDays) {
		return nil, fmt.Errorf("discharge too old: %.0f days ago exceeds %d-day limit", math.Floor(daysSinceDischarge), d.MaxAgeDays)
	}

	// 3. Determine source confidence
	confidence := sourceConfidence(input.Source)

	// 4. Build CareTransition
	ct := &models.CareTransition{
		PatientID:                    input.PatientID,
		DischargeDate:                input.DischargeDate,
		DetectedAt:                   time.Now().UTC(),
		DischargeSource:              input.Source,
		FacilityName:                 input.FacilityName,
		FacilityType:                 input.FacilityType,
		PrimaryDiagnosis:             input.PrimaryDiagnosis,
		LengthOfStayDays:             input.LengthOfStayDays,
		DischargeDisposition:         input.Disposition,
		TransitionState:              string(models.TransitionActive),
		HeightenedSurveillanceActive: true,
		ReconciliationStatus:         string(models.ReconciliationPending),
		SourceConfidence:             confidence,
		WindowDays:                   30,
	}

	return ct, nil
}

// IsDuplicate checks if a new discharge is a duplicate of an existing one.
// A duplicate is defined as the same patient with discharge dates within
// 24 hours of each other.
func (d *DischargeDetector) IsDuplicate(existing *models.CareTransition, newInput DischargeInput) bool {
	if existing == nil {
		return false
	}
	if existing.PatientID != newInput.PatientID {
		return false
	}
	diff := existing.DischargeDate.Sub(newInput.DischargeDate)
	if diff < 0 {
		diff = -diff
	}
	return diff < 24*time.Hour
}

// sourceConfidence maps a discharge source to its confidence level.
func sourceConfidence(source string) string {
	switch source {
	case models.SourceFHIREncounter, models.SourceManual:
		return "HIGH"
	case models.SourceASHAReported:
		return "MODERATE"
	case models.SourcePatientReported:
		return "LOW"
	default:
		return "LOW"
	}
}
