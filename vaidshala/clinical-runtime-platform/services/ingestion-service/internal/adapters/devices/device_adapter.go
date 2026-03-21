package devices

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// DevicePayload represents BLE device data relayed through the Flutter app.
type DevicePayload struct {
	PatientID uuid.UUID       `json:"patient_id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Timestamp time.Time       `json:"timestamp"`
	Device    DeviceInfo      `json:"device"`
	Readings  []DeviceReading `json:"readings"`
}

// DeviceInfo holds BLE device metadata.
type DeviceInfo struct {
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	FirmwareVer  string `json:"firmware_version,omitempty"`
}

// DeviceReading is a single measurement from a BLE device.
type DeviceReading struct {
	Analyte string  `json:"analyte"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
}

// DeviceAdapter converts BLE device readings (relayed via app) into CanonicalObservations.
type DeviceAdapter struct {
	logger *zap.Logger
}

// NewDeviceAdapter creates a new DeviceAdapter.
func NewDeviceAdapter(logger *zap.Logger) *DeviceAdapter {
	return &DeviceAdapter{logger: logger}
}

// Parse converts a DevicePayload into one or more CanonicalObservations.
func (a *DeviceAdapter) Parse(payload DevicePayload) ([]canonical.CanonicalObservation, error) {
	if payload.PatientID == uuid.Nil {
		return nil, fmt.Errorf("device reading missing patient_id")
	}
	if len(payload.Readings) == 0 {
		return nil, fmt.Errorf("device payload has no readings")
	}

	timestamp := payload.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	deviceCtx := &canonical.DeviceContext{
		DeviceID:     payload.Device.DeviceID,
		DeviceType:   payload.Device.DeviceType,
		Manufacturer: payload.Device.Manufacturer,
		Model:        payload.Device.Model,
		FirmwareVer:  payload.Device.FirmwareVer,
	}

	observations := make([]canonical.CanonicalObservation, 0, len(payload.Readings))

	for _, reading := range payload.Readings {
		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			PatientID:       payload.PatientID,
			TenantID:        payload.TenantID,
			SourceType:      canonical.SourceDevice,
			SourceID:        payload.Device.DeviceID,
			ObservationType: canonical.ObsDeviceData,
			Value:           reading.Value,
			Unit:            reading.Unit,
			ValueString:     reading.Analyte,
			Timestamp:       timestamp,
			DeviceContext:   deviceCtx,
			ClinicalContext: &canonical.ClinicalContext{
				Method: "automated",
			},
		}

		// Resolve LOINC code from analyte name
		if loincCode, ok := coding.LookupLOINCByAnalyte(reading.Analyte); ok {
			obs.LOINCCode = loincCode
		}

		observations = append(observations, obs)
	}

	a.logger.Info("parsed device reading",
		zap.String("patient_id", payload.PatientID.String()),
		zap.String("device_id", payload.Device.DeviceID),
		zap.String("device_type", payload.Device.DeviceType),
		zap.Int("reading_count", len(observations)),
	)

	return observations, nil
}
