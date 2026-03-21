package fhir

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// CompositeMapper implements the pipeline.Mapper interface.
// It selects the correct FHIR resource mapper based on observation type.
type CompositeMapper struct {
	logger *zap.Logger
}

// NewCompositeMapper creates a new CompositeMapper.
func NewCompositeMapper(logger *zap.Logger) *CompositeMapper {
	return &CompositeMapper{logger: logger}
}

// MapToFHIR converts a CanonicalObservation to a FHIR R4 resource JSON.
// For lab results, it produces both an Observation and a DiagnosticReport.
// For medications, it produces a MedicationStatement.
// For vitals and other types, it produces an Observation.
func (m *CompositeMapper) MapToFHIR(ctx context.Context, obs *canonical.CanonicalObservation) ([]byte, error) {
	switch obs.ObservationType {
	case canonical.ObsMedications:
		m.logger.Debug("mapping to MedicationStatement",
			zap.String("patient_id", obs.PatientID.String()),
		)
		return MapMedicationStatement(obs)

	case canonical.ObsLabs:
		// Lab results map to Observation (the DiagnosticReport wrapper
		// is created separately after the Observation ID is known)
		m.logger.Debug("mapping lab to Observation",
			zap.String("loinc", obs.LOINCCode),
			zap.String("patient_id", obs.PatientID.String()),
		)
		return MapObservation(obs)

	case canonical.ObsVitals, canonical.ObsDeviceData, canonical.ObsPatientReported,
		canonical.ObsHPI, canonical.ObsABDMRecords, canonical.ObsGeneral:
		m.logger.Debug("mapping to Observation",
			zap.String("type", string(obs.ObservationType)),
			zap.String("patient_id", obs.PatientID.String()),
		)
		return MapObservation(obs)

	default:
		return nil, fmt.Errorf("unsupported observation type for FHIR mapping: %s", obs.ObservationType)
	}
}
