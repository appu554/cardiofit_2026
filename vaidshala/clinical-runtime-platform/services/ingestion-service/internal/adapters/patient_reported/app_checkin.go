package patient_reported

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// AppCheckinPayload represents the JSON body from the Flutter app checkin.
type AppCheckinPayload struct {
	PatientID uuid.UUID    `json:"patient_id"`
	TenantID  uuid.UUID    `json:"tenant_id"`
	Timestamp time.Time    `json:"timestamp"`
	Readings  []AppReading `json:"readings"`
}

// AppReading is a single observation reading from the app.
type AppReading struct {
	Analyte string  `json:"analyte"` // e.g., "fasting_glucose", "systolic_bp", "weight"
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
}

// vitalsAnalytes lists analyte names that should be categorized as VITALS.
var vitalsAnalytes = map[string]bool{
	"systolic_bp":  true,
	"diastolic_bp": true,
	"heart_rate":   true,
	"spo2":         true,
	"temperature":  true,
	"weight":       true,
	"height":       true,
	"bmi":          true,
}

// AppCheckinAdapter converts Flutter app structured JSON into CanonicalObservations.
type AppCheckinAdapter struct {
	logger *zap.Logger
}

// NewAppCheckinAdapter creates a new AppCheckinAdapter.
func NewAppCheckinAdapter(logger *zap.Logger) *AppCheckinAdapter {
	return &AppCheckinAdapter{logger: logger}
}

// Parse converts an AppCheckinPayload into one or more CanonicalObservations.
func (a *AppCheckinAdapter) Parse(payload AppCheckinPayload) ([]canonical.CanonicalObservation, error) {
	if payload.PatientID == uuid.Nil {
		return nil, fmt.Errorf("app checkin missing patient_id")
	}
	if len(payload.Readings) == 0 {
		return nil, fmt.Errorf("app checkin has no readings")
	}

	timestamp := payload.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	observations := make([]canonical.CanonicalObservation, 0, len(payload.Readings))

	for _, reading := range payload.Readings {
		obs := canonical.CanonicalObservation{
			ID:          uuid.New(),
			PatientID:   payload.PatientID,
			TenantID:    payload.TenantID,
			SourceType:  canonical.SourcePatientReported,
			SourceID:    "app_checkin",
			Value:       reading.Value,
			Unit:        reading.Unit,
			ValueString: reading.Analyte,
			Timestamp:   timestamp,
			Flags:       []canonical.Flag{canonical.FlagManualEntry},
		}

		// Categorize as VITALS or PATIENT_REPORTED based on analyte
		if vitalsAnalytes[reading.Analyte] {
			obs.ObservationType = canonical.ObsVitals
		} else {
			obs.ObservationType = canonical.ObsPatientReported
		}

		// Try to resolve LOINC code from analyte name
		if loincCode, ok := coding.LookupLOINCByAnalyte(reading.Analyte); ok {
			obs.LOINCCode = loincCode
		}

		observations = append(observations, obs)
	}

	a.logger.Info("parsed app checkin",
		zap.String("patient_id", payload.PatientID.String()),
		zap.Int("reading_count", len(observations)),
	)

	return observations, nil
}
