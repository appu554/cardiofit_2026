package wearables

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// AppleHealthSample represents a single HealthKit sample received from
// an iOS device.
type AppleHealthSample struct {
	SampleType  string    `json:"sample_type"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	SourceName  string    `json:"source_name"`
	DeviceModel string    `json:"device_model"`
}

// AppleHealthPayload is the top-level envelope for a batch of HealthKit
// samples arriving from the patient app.
type AppleHealthPayload struct {
	PatientID uuid.UUID           `json:"patient_id"`
	TenantID  uuid.UUID           `json:"tenant_id"`
	DeviceID  string              `json:"device_id"`
	Samples   []AppleHealthSample `json:"samples"`
}

// appleHealthLOINCMap maps HealthKit sample type identifiers to LOINC codes.
var appleHealthLOINCMap = map[string]string{
	"HKQuantityTypeIdentifierHeartRate":              "8867-4",
	"HKQuantityTypeIdentifierBloodPressureSystolic":  "8480-6",
	"HKQuantityTypeIdentifierBloodPressureDiastolic": "8462-4",
	"HKQuantityTypeIdentifierStepCount":              "55423-8",
	"HKQuantityTypeIdentifierBodyMass":               "29463-7",
	"HKQuantityTypeIdentifierBloodGlucose":           "2339-0",
	"HKQuantityTypeIdentifierOxygenSaturation":       "2708-6",
	"HKQuantityTypeIdentifierBodyTemperature":        "8310-5",
	"HKQuantityTypeIdentifierRespiratoryRate":        "9279-1",
	"HKQuantityTypeIdentifierActiveEnergyBurned":     "41981-2",
	"HKQuantityTypeIdentifierRestingHeartRate":       "40443-4",
	"HKCategoryTypeIdentifierSleepAnalysis":          "93832-4",
}

// AppleHealthAdapter converts Apple HealthKit payloads into canonical
// observations.
type AppleHealthAdapter struct{}

// Convert transforms an AppleHealthPayload into a slice of
// CanonicalObservation values. Unit conversions are applied
// automatically for lbs, degF, mmol/L, and kPa.
func (a *AppleHealthAdapter) Convert(payload AppleHealthPayload) ([]canonical.CanonicalObservation, error) {
	if len(payload.Samples) == 0 {
		return nil, fmt.Errorf("apple health payload contains no samples")
	}

	var observations []canonical.CanonicalObservation

	for _, sample := range payload.Samples {
		loinc, ok := appleHealthLOINCMap[sample.SampleType]
		if !ok {
			return nil, fmt.Errorf("unsupported Apple Health sample type: %s", sample.SampleType)
		}

		value, unit := convertAppleHealthUnit(sample.Value, sample.Unit)

		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			PatientID:       payload.PatientID,
			TenantID:        payload.TenantID,
			SourceType:      canonical.SourceWearable,
			SourceID:        fmt.Sprintf("applehealth:%s:%s", payload.DeviceID, sample.SampleType),
			ObservationType: canonical.ObsVitals,
			LOINCCode:       loinc,
			Value:           value,
			Unit:            unit,
			Timestamp:       sample.StartDate,
			QualityScore:    0.90,
			DeviceContext: &canonical.DeviceContext{
				DeviceID:     payload.DeviceID,
				DeviceType:   "apple_health",
				Manufacturer: "Apple",
				Model:        sample.DeviceModel,
			},
		}

		observations = append(observations, obs)
	}

	return observations, nil
}

// convertAppleHealthUnit normalises common HealthKit units to their
// canonical clinical equivalents:
//   - lbs  → kg   (divide by 2.20462)
//   - degF → degC ((F - 32) * 5/9)
//   - mmol/L → mg/dL (multiply by 18.0182)
//   - kPa  → mmHg (multiply by 7.50062)
func convertAppleHealthUnit(value float64, unit string) (float64, string) {
	switch unit {
	case "lb", "lbs":
		return value / 2.20462, "kg"
	case "degF", "°F":
		return (value - 32) * 5.0 / 9.0, "°C"
	case "mmol/L":
		return value * 18.0182, "mg/dL"
	case "kPa":
		return value * 7.50062, "mmHg"
	default:
		return value, unit
	}
}
