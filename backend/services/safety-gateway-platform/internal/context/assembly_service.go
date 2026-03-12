package context

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/services"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// AssemblyService assembles clinical context from multiple data sources
type AssemblyService struct {
	fhirClient    services.FHIRClient
	graphClient   GraphDBClient
	cacheManager  *CacheManager
	contextBuilder *ContextBuilder
	config        *config.Config
	logger        *logger.Logger
}

// GraphDBClient interface for GraphDB data access
type GraphDBClient interface {
	GetPatientContext(ctx context.Context, patientID string) (map[string]interface{}, error)
	GetClinicalRelationships(ctx context.Context, patientID string) (map[string]interface{}, error)
}

// NewAssemblyService creates a new context assembly service
func NewAssemblyService(
	graphClient GraphDBClient,
	cfg *config.Config,
	logger *logger.Logger,
) *AssemblyService {
	// Create FHIR client based on configuration
	fhirClient := services.NewFHIRClient(cfg, logger)
	cacheManager := NewCacheManager(cfg.Caching, logger)
	contextBuilder := NewContextBuilder(logger)

	return &AssemblyService{
		fhirClient:     fhirClient,
		graphClient:    graphClient,
		cacheManager:   cacheManager,
		contextBuilder: contextBuilder,
		config:         cfg,
		logger:         logger,
	}
}

// AssembleContext assembles clinical context for a patient
func (as *AssemblyService) AssembleContext(ctx context.Context, patientID string) (*types.ClinicalContext, error) {
	startTime := time.Now()
	contextLogger := as.logger.WithPatientID(patientID)

	contextLogger.Debug("Starting context assembly", zap.String("patient_id", patientID))

	// Check cache first
	if cached := as.cacheManager.Get(patientID); cached != nil {
		contextLogger.Debug("Context cache hit",
			zap.String("patient_id", patientID),
			zap.Int64("duration_ms", time.Since(startTime).Milliseconds()),
		)
		
		as.logger.LogContextAssembly(patientID, time.Since(startTime).Milliseconds(), []string{"cache"}, true)
		return cached, nil
	}

	// Parallel data fetching
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	// Data containers
	var demographics *types.PatientDemographics
	var medications []types.Medication
	var allergies []types.Allergy
	var conditions []types.Condition
	var vitals []types.VitalSign
	var labResults []types.LabResult
	var encounters []types.Encounter
	var graphContext map[string]interface{}
	
	// Error collection
	var errors []error
	
	// Fetch demographics
	wg.Add(1)
	go func() {
		defer wg.Done()
		demo, err := as.fhirClient.GetDemographics(ctx, patientID)
		mu.Lock()
		if demo != nil {
			// Convert Demographics to PatientDemographics
			demographics = &types.PatientDemographics{
				Age:             demo.Age,
				Gender:          demo.Gender,
				Weight:          demo.Weight,
				Height:          demo.Height,
				BMI:             demo.BMI,
				PregnancyStatus: demo.PregnancyStatus,
			}
		}
		if err != nil {
			errors = append(errors, fmt.Errorf("demographics: %w", err))
		}
		mu.Unlock()
	}()

	// Fetch medications
	wg.Add(1)
	go func() {
		defer wg.Done()
		meds, err := as.fhirClient.GetActiveMedications(ctx, patientID)
		mu.Lock()
		medications = meds
		if err != nil {
			errors = append(errors, fmt.Errorf("medications: %w", err))
		}
		mu.Unlock()
	}()

	// Fetch allergies
	wg.Add(1)
	go func() {
		defer wg.Done()
		allerg, err := as.fhirClient.GetAllergies(ctx, patientID)
		mu.Lock()
		allergies = allerg
		if err != nil {
			errors = append(errors, fmt.Errorf("allergies: %w", err))
		}
		mu.Unlock()
	}()

	// Fetch conditions
	wg.Add(1)
	go func() {
		defer wg.Done()
		conds, err := as.fhirClient.GetConditions(ctx, patientID)
		mu.Lock()
		conditions = conds
		if err != nil {
			errors = append(errors, fmt.Errorf("conditions: %w", err))
		}
		mu.Unlock()
	}()

	// Fetch recent vitals (last 24 hours)
	wg.Add(1)
	go func() {
		defer wg.Done()
		vits, err := as.fhirClient.GetRecentVitals(ctx, patientID, 24)
		mu.Lock()
		vitals = vits
		if err != nil {
			errors = append(errors, fmt.Errorf("vitals: %w", err))
		}
		mu.Unlock()
	}()

	// Fetch recent lab results (last 72 hours)
	wg.Add(1)
	go func() {
		defer wg.Done()
		labs, err := as.fhirClient.GetRecentLabResults(ctx, patientID, 72)
		mu.Lock()
		labResults = labs
		if err != nil {
			errors = append(errors, fmt.Errorf("lab_results: %w", err))
		}
		mu.Unlock()
	}()

	// Fetch recent encounters (last 30 days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		encs, err := as.fhirClient.GetRecentEncounters(ctx, patientID, 30)
		mu.Lock()
		encounters = encs
		if err != nil {
			errors = append(errors, fmt.Errorf("encounters: %w", err))
		}
		mu.Unlock()
	}()

	// Fetch graph context (optional)
	if as.graphClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			graphCtx, err := as.graphClient.GetPatientContext(ctx, patientID)
			mu.Lock()
			graphContext = graphCtx
			if err != nil {
				// Graph context is optional, log but don't fail
				contextLogger.Warn("Failed to fetch graph context", zap.Error(err))
			}
			mu.Unlock()
		}()
	}

	// Wait for all data fetching to complete
	wg.Wait()

	// Check for critical errors
	if len(errors) > 0 {
		// Log all errors but continue if we have some data
		for _, err := range errors {
			contextLogger.Warn("Data fetching error", zap.Error(err))
		}

		// Fail if we don't have essential data
		if demographics == nil {
			return nil, fmt.Errorf("failed to fetch essential patient demographics")
		}
	}

	// Build context
	contextData := &ContextData{
		PatientID:     patientID,
		Demographics:  demographics,
		Medications:   medications,
		Allergies:     allergies,
		Conditions:    conditions,
		Vitals:        vitals,
		LabResults:    labResults,
		Encounters:    encounters,
		GraphContext:  graphContext,
	}

	clinicalContext := as.contextBuilder.Build(contextData)

	// Cache the context
	ttl := time.Duration(as.config.Caching.ContextTTLMinutes) * time.Minute
	as.cacheManager.Set(patientID, clinicalContext, ttl)

	duration := time.Since(startTime)
	dataSources := as.getDataSources(errors)

	contextLogger.Debug("Context assembly completed",
		zap.String("patient_id", patientID),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.Strings("data_sources", dataSources),
		zap.Int("errors", len(errors)),
	)

	as.logger.LogContextAssembly(patientID, duration.Milliseconds(), dataSources, false)

	return clinicalContext, nil
}

// getDataSources returns the list of data sources used
func (as *AssemblyService) getDataSources(errors []error) []string {
	sources := []string{"fhir"}
	
	if as.graphClient != nil {
		// Check if graph context was successfully fetched
		graphError := false
		for _, err := range errors {
			if err != nil && (err.Error() == "graph context failed" || 
				err.Error() == "graph client unavailable") {
				graphError = true
				break
			}
		}
		if !graphError {
			sources = append(sources, "graphdb")
		}
	}
	
	return sources
}

// GetContextVersion returns a version string for the context
func (as *AssemblyService) GetContextVersion(patientID string) string {
	// This would typically include timestamps of last updates from each source
	return fmt.Sprintf("v1_%s_%d", patientID[:8], time.Now().Unix())
}

// InvalidateCache invalidates the cache for a patient
func (as *AssemblyService) InvalidateCache(patientID string) {
	as.cacheManager.Delete(patientID)
	as.logger.Debug("Context cache invalidated", zap.String("patient_id", patientID))
}

// GetCacheStats returns cache statistics
func (as *AssemblyService) GetCacheStats() map[string]interface{} {
	return as.cacheManager.GetStats()
}

// Shutdown shuts down the assembly service
func (as *AssemblyService) Shutdown() error {
	if as.cacheManager != nil {
		as.cacheManager.Shutdown()
	}
	as.logger.Info("Context assembly service shut down")
	return nil
}

// ContextData holds raw data for context building
type ContextData struct {
	PatientID     string
	Demographics  *types.PatientDemographics
	Medications   []types.Medication
	Allergies     []types.Allergy
	Conditions    []types.Condition
	Vitals        []types.VitalSign
	LabResults    []types.LabResult
	Encounters    []types.Encounter
	GraphContext  map[string]interface{}
}
