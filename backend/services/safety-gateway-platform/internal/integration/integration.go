package integration

import (
	"context"
	"fmt"
	"time"

	"safety-gateway-platform/internal/types"
	"safety-gateway-platform/pkg/logger"
)

// SafetyGatewayIntegration provides the main integration layer for Phase 2
type SafetyGatewayIntegration struct {
	caeApollo *CAEApolloIntegration
	logger    *logger.Logger
	config    *IntegrationConfig
}

// IntegrationConfig holds configuration for all integrations
type IntegrationConfig struct {
	// CAE Integration settings
	CAE struct {
		ApolloFederationURL string        `yaml:"apollo_federation_url"`
		CAEServiceURL       string        `yaml:"cae_service_url"`
		SnapshotTTL         time.Duration `yaml:"snapshot_ttl"`
		EnableBatchCAE      bool          `yaml:"enable_batch_cae"`
		MaxConcurrentCAE    int           `yaml:"max_concurrent_cae"`
		KBVersionStrategy   string        `yaml:"kb_version_strategy"`
	} `yaml:"cae"`

	// Future integrations can be added here
	// ML struct { ... } `yaml:"ml"`
	// Analytics struct { ... } `yaml:"analytics"`
}

// NewSafetyGatewayIntegration creates the main integration layer
func NewSafetyGatewayIntegration(config *IntegrationConfig, logger *logger.Logger) (*SafetyGatewayIntegration, error) {
	// Initialize CAE-Apollo integration
	caeIntegration, err := NewCAEApolloIntegration(
		config.CAE.ApolloFederationURL,
		config.CAE.CAEServiceURL,
		logger,
		WithSnapshotTTL(config.CAE.SnapshotTTL),
		WithBatchProcessing(config.CAE.EnableBatchCAE, config.CAE.MaxConcurrentCAE),
		WithKBVersionStrategy(config.CAE.KBVersionStrategy),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CAE integration: %w", err)
	}

	return &SafetyGatewayIntegration{
		caeApollo: caeIntegration,
		logger:    logger,
		config:    config,
	}, nil
}

// EvaluateRequest performs a single safety evaluation using the appropriate integration
func (sgi *SafetyGatewayIntegration) EvaluateRequest(
	ctx context.Context,
	request *types.SafetyRequest,
) (*types.SafetyResponse, error) {
	// Route to CAE-Apollo integration
	return sgi.caeApollo.EvaluateWithSnapshot(ctx, request)
}

// BatchEvaluateRequests performs batch safety evaluation
func (sgi *SafetyGatewayIntegration) BatchEvaluateRequests(
	ctx context.Context,
	requests []*types.SafetyRequest,
) ([]*types.SafetyResponse, error) {
	// Route to CAE-Apollo integration
	return sgi.caeApollo.BatchEvaluateWithSnapshots(ctx, requests)
}

// WhatIfAnalysis performs what-if scenario analysis
func (sgi *SafetyGatewayIntegration) WhatIfAnalysis(
	ctx context.Context,
	baselineRequest *types.SafetyRequest,
	scenarios []MedicationScenario,
) (*WhatIfAnalysisResponse, error) {
	// Route to CAE-Apollo integration
	return sgi.caeApollo.WhatIfAnalysisWithSnapshots(ctx, baselineRequest, scenarios)
}

// GetIntegrationHealth returns health status of all integrations
func (sgi *SafetyGatewayIntegration) GetIntegrationHealth(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	// Check CAE integration health
	caeHealth := make(map[string]interface{})
	
	// Check Apollo Federation health
	if err := sgi.caeApollo.apolloClient.Health(ctx); err != nil {
		caeHealth["apollo_federation"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		caeHealth["apollo_federation"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check CAE Engine health
	if err := sgi.caeApollo.caeClient.Health(ctx); err != nil {
		caeHealth["cae_engine"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		caeHealth["cae_engine"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Get snapshot cache stats
	caeHealth["snapshot_cache"] = sgi.caeApollo.GetCacheStats()

	health["cae"] = caeHealth

	return health
}

// GetIntegrationMetrics returns performance metrics for all integrations
func (sgi *SafetyGatewayIntegration) GetIntegrationMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// CAE integration metrics
	caeMetrics := make(map[string]interface{})
	caeMetrics["cache_stats"] = sgi.caeApollo.GetCacheStats()
	caeMetrics["configuration"] = map[string]interface{}{
		"snapshot_ttl":        sgi.config.CAE.SnapshotTTL.String(),
		"batch_enabled":       sgi.config.CAE.EnableBatchCAE,
		"max_concurrent":      sgi.config.CAE.MaxConcurrentCAE,
		"kb_version_strategy": sgi.config.CAE.KBVersionStrategy,
	}

	metrics["cae"] = caeMetrics

	return metrics
}

// Close gracefully shuts down all integrations
func (sgi *SafetyGatewayIntegration) Close() error {
	sgi.logger.Info("Shutting down Safety Gateway integrations")

	// Close CAE integration
	if err := sgi.caeApollo.Close(); err != nil {
		sgi.logger.Error("Failed to close CAE integration", 
			"error", err.Error())
		return err
	}

	sgi.logger.Info("Safety Gateway integrations shut down successfully")
	return nil
}

// DefaultIntegrationConfig returns default configuration for integrations
func DefaultIntegrationConfig() *IntegrationConfig {
	config := &IntegrationConfig{}
	
	// Default CAE configuration
	config.CAE.ApolloFederationURL = "http://localhost:4000"
	config.CAE.CAEServiceURL = "http://localhost:8027"
	config.CAE.SnapshotTTL = 30 * time.Minute
	config.CAE.EnableBatchCAE = true
	config.CAE.MaxConcurrentCAE = 10
	config.CAE.KBVersionStrategy = "latest"

	return config
}

// IntegrationConfigFromEnv creates configuration from environment variables
func IntegrationConfigFromEnv() *IntegrationConfig {
	config := DefaultIntegrationConfig()

	// Override with environment variables if present
	// This would use os.Getenv() calls to override defaults
	// Example:
	// if url := os.Getenv("APOLLO_FEDERATION_URL"); url != "" {
	//     config.CAE.ApolloFederationURL = url
	// }

	return config
}