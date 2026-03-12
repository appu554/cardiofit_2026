package services

import (
	"context"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// FHIRClient interface defines the contract for FHIR clients
type FHIRClient interface {
	GetDemographics(ctx context.Context, patientID string) (*types.Demographics, error)
	GetActiveMedications(ctx context.Context, patientID string) ([]types.Medication, error)
	GetAllergies(ctx context.Context, patientID string) ([]types.Allergy, error)
	GetConditions(ctx context.Context, patientID string) ([]types.Condition, error)
	GetRecentVitals(ctx context.Context, patientID string, hours int) ([]types.VitalSign, error)
	GetRecentLabResults(ctx context.Context, patientID string, hours int) ([]types.LabResult, error)
	GetRecentEncounters(ctx context.Context, patientID string, days int) ([]types.Encounter, error)
}

// NewFHIRClient creates a new FHIR client based on configuration
func NewFHIRClient(cfg *config.Config, logger *logger.Logger) FHIRClient {
	// Always use real Google Cloud Healthcare FHIR client
	return NewRealFHIRClient(
		logger,
		cfg.ExternalServices.GoogleHealthcareAPI.ProjectID,
		cfg.ExternalServices.GoogleHealthcareAPI.Location,
		cfg.ExternalServices.GoogleHealthcareAPI.DatasetID,
		cfg.ExternalServices.GoogleHealthcareAPI.FHIRStoreID,
	)
}


