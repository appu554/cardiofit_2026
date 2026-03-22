package wearables

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// HealthConnectRecord represents a single Health Connect data record
// received from Android Health Connect API.
type HealthConnectRecord struct {
	RecordType  string             `json:"record_type"`
	PackageName string             `json:"package_name"`
	DeviceModel string             `json:"device_model"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	Values      map[string]float64 `json:"values"`
	Unit        string             `json:"unit"`
}

// HealthConnectPayload is the top-level envelope for a batch of Health
// Connect records arriving from the patient app.
type HealthConnectPayload struct {
	PatientID uuid.UUID            `json:"patient_id"`
	TenantID  uuid.UUID            `json:"tenant_id"`
	DeviceID  string               `json:"device_id"`
	Records   []HealthConnectRecord `json:"records"`
}

// healthConnectLOINCMap maps Health Connect record types to LOINC codes.
var healthConnectLOINCMap = map[string]string{
	"BloodPressureSystolic":  "8480-6",
	"BloodPressureDiastolic": "8462-4",
	"HeartRate":              "8867-4",
	"Steps":                  "55423-8",
	"Weight":                 "29463-7",
	"BloodGlucose":           "2339-0",
	"OxygenSaturation":       "2708-6",
	"BodyTemperature":        "8310-5",
	"SleepSession":           "93832-4",
	"ActiveCaloriesBurned":   "41981-2",
}

// HealthConnectAdapter converts Health Connect payloads into canonical
// observations.
type HealthConnectAdapter struct{}

// Convert transforms a HealthConnectPayload into a slice of
// CanonicalObservation values. Blood pressure records produce two
// observations (systolic and diastolic).
func (a *HealthConnectAdapter) Convert(payload HealthConnectPayload) ([]canonical.CanonicalObservation, error) {
	if len(payload.Records) == 0 {
		return nil, fmt.Errorf("health connect payload contains no records")
	}

	var observations []canonical.CanonicalObservation

	for _, rec := range payload.Records {
		// Special handling: BloodPressure produces two observations.
		if rec.RecordType == "BloodPressure" {
			obs, err := a.convertBloodPressure(payload, rec)
			if err != nil {
				return nil, err
			}
			observations = append(observations, obs...)
			continue
		}

		loinc, ok := healthConnectLOINCMap[rec.RecordType]
		if !ok {
			return nil, fmt.Errorf("unsupported Health Connect record type: %s", rec.RecordType)
		}

		value, ok := rec.Values["value"]
		if !ok {
			return nil, fmt.Errorf("missing 'value' in record type %s", rec.RecordType)
		}

		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			PatientID:       payload.PatientID,
			TenantID:        payload.TenantID,
			SourceType:      canonical.SourceWearable,
			SourceID:        fmt.Sprintf("healthconnect:%s:%s", payload.DeviceID, rec.RecordType),
			ObservationType: canonical.ObsVitals,
			LOINCCode:       loinc,
			Value:           value,
			Unit:            rec.Unit,
			Timestamp:       rec.StartTime,
			QualityScore:    0.85,
			DeviceContext: &canonical.DeviceContext{
				DeviceID:     payload.DeviceID,
				DeviceType:   "health_connect",
				Manufacturer: rec.PackageName,
				Model:        rec.DeviceModel,
			},
		}

		observations = append(observations, obs)
	}

	return observations, nil
}

// convertBloodPressure splits a BloodPressure record into systolic and
// diastolic canonical observations.
func (a *HealthConnectAdapter) convertBloodPressure(payload HealthConnectPayload, rec HealthConnectRecord) ([]canonical.CanonicalObservation, error) {
	systolic, hasSys := rec.Values["systolic"]
	diastolic, hasDia := rec.Values["diastolic"]

	if !hasSys || !hasDia {
		return nil, fmt.Errorf("blood pressure record missing systolic or diastolic value")
	}

	base := canonical.CanonicalObservation{
		PatientID:       payload.PatientID,
		TenantID:        payload.TenantID,
		SourceType:      canonical.SourceWearable,
		ObservationType: canonical.ObsVitals,
		Timestamp:       rec.StartTime,
		QualityScore:    0.85,
		DeviceContext: &canonical.DeviceContext{
			DeviceID:     payload.DeviceID,
			DeviceType:   "health_connect",
			Manufacturer: rec.PackageName,
			Model:        rec.DeviceModel,
		},
	}

	sysObs := base
	sysObs.ID = uuid.New()
	sysObs.SourceID = fmt.Sprintf("healthconnect:%s:BloodPressureSystolic", payload.DeviceID)
	sysObs.LOINCCode = healthConnectLOINCMap["BloodPressureSystolic"]
	sysObs.Value = systolic
	sysObs.Unit = "mmHg"

	diaObs := base
	diaObs.ID = uuid.New()
	diaObs.SourceID = fmt.Sprintf("healthconnect:%s:BloodPressureDiastolic", payload.DeviceID)
	diaObs.LOINCCode = healthConnectLOINCMap["BloodPressureDiastolic"]
	diaObs.Value = diastolic
	diaObs.Unit = "mmHg"

	return []canonical.CanonicalObservation{sysObs, diaObs}, nil
}
