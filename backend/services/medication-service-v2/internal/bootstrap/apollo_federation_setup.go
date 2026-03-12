package bootstrap

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/infrastructure"
	httpHandlers "medication-service-v2/internal/interfaces/http"
)

// ApolloFederationSetup handles the setup and initialization of Apollo Federation components
type ApolloFederationSetup struct {
	logger                    *zap.Logger
	config                    *infrastructure.ApolloFederationConfig
	factory                   *infrastructure.ApolloFederationFactory
	apolloFederationService   *services.ApolloFederationService
	knowledgeBaseService      *services.KnowledgeBaseIntegrationService
	httpHandler               *httpHandlers.ApolloFederationHandler
}

// SetupDependencies holds dependencies needed for Apollo Federation setup
type SetupDependencies struct {
	Logger                    *zap.Logger
	Config                    map[string]interface{} // Configuration map from Viper
	CacheService             services.CacheServiceInterface
	PerformanceMonitor       services.PerformanceMonitorInterface
	CircuitBreaker           services.CircuitBreakerInterface
}

// NewApolloFederationSetup creates a new Apollo Federation setup instance
func NewApolloFederationSetup(deps *SetupDependencies) (*ApolloFederationSetup, error) {
	// Create Apollo Federation configuration
	federationConfigMap, exists := deps.Config["external_services"].(map[string]interface{})
	if !exists {
		return nil, fmt.Errorf("external_services configuration not found")
	}

	apolloConfigMap, exists := federationConfigMap["apollo_federation"].(map[string]interface{})
	if !exists {
		return nil, fmt.Errorf("apollo_federation configuration not found")
	}

	apolloConfig, err := infrastructure.NewApolloFederationConfigFromMap(apolloConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create Apollo Federation config: %w", err)
	}

	// Create factory
	factory := infrastructure.NewApolloFederationFactory(apolloConfig, deps.Logger)

	setup := &ApolloFederationSetup{
		logger:  deps.Logger,
		config:  apolloConfig,
		factory: factory,
	}

	// Initialize services
	if err := setup.initializeServices(deps); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Create HTTP handler
	setup.httpHandler = httpHandlers.NewApolloFederationHandler(
		setup.knowledgeBaseService,
		deps.Logger,
	)

	deps.Logger.Info("Apollo Federation setup completed",
		zap.String("gateway_url", apolloConfig.URL),
		zap.Duration("timeout", apolloConfig.Timeout),
		zap.Bool("health_check_enabled", apolloConfig.HealthCheckEnabled),
	)

	return setup, nil
}

// initializeServices initializes Apollo Federation services
func (s *ApolloFederationSetup) initializeServices(deps *SetupDependencies) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get Apollo Federation client from factory
	client, err := s.factory.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Apollo Federation client: %w", err)
	}

	// Create Apollo Federation service
	s.apolloFederationService = services.NewApolloFederationService(
		client,
		s.logger,
		deps.CacheService,
		deps.PerformanceMonitor,
		deps.CacheService.(services.HealthCheckerInterface), // Assuming cache service implements health checking
	)

	// Create Knowledge Base Integration service
	s.knowledgeBaseService = services.NewKnowledgeBaseIntegrationService(
		s.apolloFederationService,
		s.logger,
		deps.CacheService,
		deps.PerformanceMonitor,
		deps.CircuitBreaker,
	)

	// Test connectivity
	if err := s.testConnectivity(ctx); err != nil {
		s.logger.Warn("Apollo Federation connectivity test failed", zap.Error(err))
		// Don't fail setup - service may come online later
	} else {
		s.logger.Info("Apollo Federation connectivity test passed")
	}

	return nil
}

// testConnectivity tests connectivity to Apollo Federation gateway
func (s *ApolloFederationSetup) testConnectivity(ctx context.Context) error {
	return s.apolloFederationService.HealthCheck(ctx)
}

// GetApolloFederationService returns the Apollo Federation service
func (s *ApolloFederationSetup) GetApolloFederationService() *services.ApolloFederationService {
	return s.apolloFederationService
}

// GetKnowledgeBaseService returns the Knowledge Base Integration service
func (s *ApolloFederationSetup) GetKnowledgeBaseService() *services.KnowledgeBaseIntegrationService {
	return s.knowledgeBaseService
}

// GetHTTPHandler returns the HTTP handler for Apollo Federation endpoints
func (s *ApolloFederationSetup) GetHTTPHandler() *httpHandlers.ApolloFederationHandler {
	return s.httpHandler
}

// GetFactory returns the Apollo Federation factory
func (s *ApolloFederationSetup) GetFactory() *infrastructure.ApolloFederationFactory {
	return s.factory
}

// GetConfig returns the Apollo Federation configuration
func (s *ApolloFederationSetup) GetConfig() *infrastructure.ApolloFederationConfig {
	return s.config
}

// Shutdown gracefully shuts down Apollo Federation components
func (s *ApolloFederationSetup) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down Apollo Federation components")

	// Shutdown factory (which handles clients and health checkers)
	if err := s.factory.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to shutdown Apollo Federation factory", zap.Error(err))
		return err
	}

	s.logger.Info("Apollo Federation components shutdown completed")
	return nil
}

// GetHealthStatus returns health status for all Apollo Federation components
func (s *ApolloFederationSetup) GetHealthStatus(ctx context.Context) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Factory health status
	status["factory"] = s.factory.GetHealthStatus()

	// Service health status  
	serviceHealth, err := s.knowledgeBaseService.GetServiceHealth(ctx)
	if err != nil {
		status["service"] = map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	} else {
		status["service"] = serviceHealth
	}

	// Configuration status
	status["configuration"] = map[string]interface{}{
		"gateway_url":              s.config.URL,
		"timeout_seconds":          s.config.Timeout.Seconds(),
		"max_retries":              s.config.MaxRetries,
		"health_check_enabled":     s.config.HealthCheckEnabled,
		"metrics_enabled":          s.config.MetricsEnabled,
		"cache_enabled":            s.config.CacheEnabled,
		"circuit_breaker_enabled":  s.config.CircuitBreaker.Enabled,
	}

	return status, nil
}

// GetMetrics returns performance metrics for Apollo Federation
func (s *ApolloFederationSetup) GetMetrics() map[string]interface{} {
	metrics := s.factory.GetMetrics()
	return metrics.GetSnapshot()
}

// ValidateSetup validates the Apollo Federation setup
func (s *ApolloFederationSetup) ValidateSetup(ctx context.Context) error {
	s.logger.Info("Validating Apollo Federation setup")

	// Validate configuration
	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Test basic connectivity
	if err := s.testConnectivity(ctx); err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}

	// Test basic query functionality
	if err := s.testBasicQuery(ctx); err != nil {
		return fmt.Errorf("basic query test failed: %w", err)
	}

	s.logger.Info("Apollo Federation setup validation completed successfully")
	return nil
}

// testBasicQuery tests basic query functionality
func (s *ApolloFederationSetup) testBasicQuery(ctx context.Context) error {
	// Test availability check (simplest query)
	testDrugCode := "aspirin"
	region := "US"

	request := &services.KnowledgeBaseQueryRequest{
		DrugCode:     testDrugCode,
		Region:       &region,
		QueryTypes:   []string{"availability"},
		CacheEnabled: false, // Disable cache for test
		Priority:     "low",
	}

	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.knowledgeBaseService.QueryKnowledgeBases(testCtx, request)
	if err != nil {
		s.logger.Warn("Basic query test failed - this may be expected if gateway is not running",
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Basic query test passed")
	return nil
}

// ReconfigureFromMap updates configuration from a new configuration map
func (s *ApolloFederationSetup) ReconfigureFromMap(configMap map[string]interface{}) error {
	federationConfigMap, exists := configMap["external_services"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("external_services configuration not found")
	}

	apolloConfigMap, exists := federationConfigMap["apollo_federation"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("apollo_federation configuration not found")
	}

	newConfig, err := infrastructure.NewApolloFederationConfigFromMap(apolloConfigMap)
	if err != nil {
		return fmt.Errorf("failed to create new Apollo Federation config: %w", err)
	}

	// Validate new configuration
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("new configuration is invalid: %w", err)
	}

	// Update configuration
	oldConfig := s.config
	s.config = newConfig

	s.logger.Info("Apollo Federation configuration updated",
		zap.String("old_url", oldConfig.URL),
		zap.String("new_url", newConfig.URL),
		zap.Duration("old_timeout", oldConfig.Timeout),
		zap.Duration("new_timeout", newConfig.Timeout),
	)

	return nil
}

// CreateExample creates an example usage of the Apollo Federation client
func (s *ApolloFederationSetup) CreateExample(ctx context.Context) error {
	s.logger.Info("Running Apollo Federation example")

	// Example 1: Basic dosing rule query
	if err := s.exampleDosingRuleQuery(ctx); err != nil {
		s.logger.Error("Dosing rule example failed", zap.Error(err))
	}

	// Example 2: Patient-specific dosing calculation
	if err := s.examplePatientDosingCalculation(ctx); err != nil {
		s.logger.Error("Patient dosing calculation example failed", zap.Error(err))
	}

	// Example 3: Batch query
	if err := s.exampleBatchQuery(ctx); err != nil {
		s.logger.Error("Batch query example failed", zap.Error(err))
	}

	// Example 4: Comprehensive clinical intelligence
	if err := s.exampleClinicalIntelligence(ctx); err != nil {
		s.logger.Error("Clinical intelligence example failed", zap.Error(err))
	}

	return nil
}

// exampleDosingRuleQuery demonstrates basic dosing rule query
func (s *ApolloFederationSetup) exampleDosingRuleQuery(ctx context.Context) error {
	s.logger.Info("Example: Basic dosing rule query")

	request := &services.KnowledgeBaseQueryRequest{
		DrugCode:     "vancomycin",
		Region:       stringPtr("US"),
		QueryTypes:   []string{"dosing"},
		CacheEnabled: true,
		CacheTTL:     30 * time.Minute,
		Priority:     "normal",
	}

	response, err := s.knowledgeBaseService.QueryKnowledgeBases(ctx, request)
	if err != nil {
		return err
	}

	s.logger.Info("Dosing rule query completed",
		zap.String("drug_code", response.DrugCode),
		zap.Int("dosing_rules_found", len(response.DosingRules)),
		zap.Duration("response_time", response.QueryMetrics.TotalDuration),
		zap.Bool("cache_hit", response.CacheStatus.HitsByType["dosing"] > 0),
	)

	return nil
}

// examplePatientDosingCalculation demonstrates patient-specific dosing calculation
func (s *ApolloFederationSetup) examplePatientDosingCalculation(ctx context.Context) error {
	s.logger.Info("Example: Patient-specific dosing calculation")

	patientContext := &infrastructure.PatientContextInput{
		WeightKg:            70.0,
		EGFR:               85.0,
		AgeYears:            45,
		Sex:                "male",
		Pregnant:           boolPtr(false),
		CreatinineClearance: float64Ptr(90.0),
	}

	request := &services.KnowledgeBaseQueryRequest{
		DrugCode:       "vancomycin",
		PatientContext: patientContext,
		Region:         stringPtr("US"),
		QueryTypes:     []string{"dosing"},
		CacheEnabled:   true,
		CacheTTL:       15 * time.Minute,
		Priority:       "high",
	}

	response, err := s.knowledgeBaseService.QueryKnowledgeBases(ctx, request)
	if err != nil {
		return err
	}

	s.logger.Info("Patient dosing calculation completed",
		zap.String("drug_code", response.DrugCode),
		zap.Int("recommendations_found", len(response.DosingRecommendations)),
		zap.Duration("response_time", response.QueryMetrics.TotalDuration),
	)

	return nil
}

// exampleBatchQuery demonstrates batch querying
func (s *ApolloFederationSetup) exampleBatchQuery(ctx context.Context) error {
	s.logger.Info("Example: Batch dosing query")

	drugCodes := []string{"vancomycin", "gentamicin", "cefazolin", "warfarin"}
	requests := make([]*services.KnowledgeBaseQueryRequest, len(drugCodes))

	for i, drugCode := range drugCodes {
		requests[i] = &services.KnowledgeBaseQueryRequest{
			DrugCode:     drugCode,
			Region:       stringPtr("US"),
			QueryTypes:   []string{"dosing", "availability"},
			CacheEnabled: true,
			CacheTTL:     30 * time.Minute,
		}
	}

	responses, err := s.knowledgeBaseService.BatchQueryKnowledgeBases(ctx, requests)
	if err != nil {
		return err
	}

	s.logger.Info("Batch query completed",
		zap.Int("requested_drugs", len(drugCodes)),
		zap.Int("successful_responses", len(responses)),
	)

	return nil
}

// exampleClinicalIntelligence demonstrates comprehensive clinical intelligence query
func (s *ApolloFederationSetup) exampleClinicalIntelligence(ctx context.Context) error {
	s.logger.Info("Example: Comprehensive clinical intelligence")

	request := &services.KnowledgeBaseQueryRequest{
		DrugCode:     "warfarin",
		Region:       stringPtr("US"),
		QueryTypes:   []string{"dosing", "guidelines", "interactions", "safety", "availability"},
		CacheEnabled: true,
		CacheTTL:     20 * time.Minute,
		Priority:     "normal",
	}

	response, err := s.knowledgeBaseService.QueryKnowledgeBases(ctx, request)
	if err != nil {
		return err
	}

	s.logger.Info("Clinical intelligence query completed",
		zap.String("drug_code", response.DrugCode),
		zap.Int("dosing_rules", len(response.DosingRules)),
		zap.Int("guidelines", len(response.ClinicalGuidelines)),
		zap.Int("interactions", len(response.DrugInteractions)),
		zap.Int("safety_alerts", len(response.SafetyAlerts)),
		zap.Duration("total_response_time", response.QueryMetrics.TotalDuration),
		zap.Int("successful_queries", response.QueryMetrics.SuccessfulQueries),
		zap.Int("failed_queries", response.QueryMetrics.FailedQueries),
	)

	return nil
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}