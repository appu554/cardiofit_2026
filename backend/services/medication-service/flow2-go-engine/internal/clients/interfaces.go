package clients

import (
	"context"
	"fmt"

	"flow2-go-engine/internal/config"
	"flow2-go-engine/internal/models"

	"github.com/sirupsen/logrus"
)

// ContextServiceClient interface for communicating with the Context Service
type ContextServiceClient interface {
	// ORB-driven targeted context fetching (NEW PRIMARY METHOD)
	FetchContext(ctx context.Context, request *models.ContextRequest) (*models.ClinicalContext, error)

	// Legacy individual data fetching methods (kept for backward compatibility)
	GetPatientContext(ctx context.Context, patientID string, contextRequirements []string) (*models.ClinicalContext, error)
	GetPatientDemographics(ctx context.Context, patientID string) (*models.PatientDemographics, error)
	GetPatientAllergies(ctx context.Context, patientID string) ([]models.Allergy, error)
	GetPatientConditions(ctx context.Context, patientID string) ([]models.Condition, error)
	GetPatientMedications(ctx context.Context, patientID string) ([]models.Medication, error)
	GetPatientLabResults(ctx context.Context, patientID string) ([]models.LabResult, error)

	Close() error
}

// MedicationAPIClient interface for communicating with the Medication API
type MedicationAPIClient interface {
	GetMedicationInfo(ctx context.Context, medicationCode string) (*models.Medication, error)
	GetFormularyInfo(ctx context.Context, medicationCode string) (*models.FormularyInfo, error)
	GetDrugInteractions(ctx context.Context, medicationCodes []string) ([]models.DrugInteraction, error)
	ValidatePrescription(ctx context.Context, prescription *models.Medication, patientID string) (*models.ValidationResult, error)
	Close() error
}

// JITSafetyClient interface for communicating with the JIT Safety Engine
type JITSafetyClient interface {
	RunJITSafetyCheck(ctx context.Context, request *models.JitSafetyContext) (*models.JitSafetyOutcome, error)
	HealthCheck(ctx context.Context) error
}

// ValidationResult represents the result of prescription validation
type ValidationResult struct {
	Valid       bool     `json:"valid"`
	Warnings    []string `json:"warnings"`
	Errors      []string `json:"errors"`
	Suggestions []string `json:"suggestions"`
}

// NewContextServiceClient creates a new context service client
func NewContextServiceClient(cfg config.ContextServiceConfig) (ContextServiceClient, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("context service URL is required - no fallback available")
	}

	return NewHTTPContextServiceClient(cfg)
}

// NewMedicationAPIClient creates a new medication API client
func NewMedicationAPIClient(cfg config.MedicationAPIConfig) (MedicationAPIClient, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("medication API URL is required - no fallback available")
	}

	// TODO: Implement real HTTP client to Medication API
	return nil, fmt.Errorf("real medication API client not yet implemented")
}

// NewContextGatewayClient creates a new context gateway client  
func NewContextGatewayClient(cfg config.ContextServiceConfig) (ContextGatewayClient, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("context gateway URL is required")
	}

	return NewHTTPContextGatewayClient(cfg)
}

// NewJITSafetyClientFromConfig creates a new JIT Safety client from config
func NewJITSafetyClientFromConfig(cfg config.JITSafetyConfig) (JITSafetyClient, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("JIT Safety Engine URL is required")
	}

	// Use default logger if not provided
	logger := cfg.Logger
	if logger == nil {
		logger = logrus.New()
	}

	clientConfig := JITSafetyConfig{
		BaseURL:        cfg.BaseURL,
		TimeoutSeconds: cfg.TimeoutSeconds,
		RetryAttempts:  cfg.RetryAttempts,
		RetryDelay:     cfg.RetryDelay,
		EnableCircuitBreaker: cfg.EnableCircuitBreaker,
	}

	return NewJITSafetyClient(clientConfig, logger), nil
}
