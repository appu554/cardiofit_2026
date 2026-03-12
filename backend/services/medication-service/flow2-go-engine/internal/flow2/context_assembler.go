package flow2

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/clients"
	"flow2-go-engine/internal/models"
	"flow2-go-engine/internal/services"
)

// ContextAssembler assembles clinical context from multiple sources
type ContextAssembler struct {
	contextServiceClient clients.ContextServiceClient
	medicationAPIClient  clients.MedicationAPIClient
	cacheService         services.CacheService
	logger               *logrus.Logger
}

// NewContextAssembler creates a new context assembler
func NewContextAssembler(
	contextServiceClient clients.ContextServiceClient,
	medicationAPIClient clients.MedicationAPIClient,
	cacheService services.CacheService,
	logger *logrus.Logger,
) *ContextAssembler {
	return &ContextAssembler{
		contextServiceClient: contextServiceClient,
		medicationAPIClient:  medicationAPIClient,
		cacheService:         cacheService,
		logger:               logger,
	}
}

// AssembleContext assembles comprehensive clinical context for a Flow 2 request
func (ca *ContextAssembler) AssembleContext(ctx context.Context, request *models.Flow2Request) (*models.ClinicalContext, error) {
	start := time.Now()
	
	ca.logger.WithFields(logrus.Fields{
		"request_id": request.RequestID,
		"patient_id": request.PatientID,
		"action_type": request.ActionType,
	}).Info("Starting context assembly")

	// Initialize clinical context
	clinicalContext := &models.ClinicalContext{}

	// Use WaitGroup for parallel context gathering
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Gather patient demographics
	wg.Add(1)
	go func() {
		defer wg.Done()
		demographics, err := ca.contextServiceClient.GetPatientDemographics(ctx, request.PatientID)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
			return
		}
		mu.Lock()
		clinicalContext.PatientDemographics = demographics
		mu.Unlock()
	}()

	// Gather current medications
	wg.Add(1)
	go func() {
		defer wg.Done()
		medications, err := ca.contextServiceClient.GetPatientMedications(ctx, request.PatientID)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
			return
		}
		mu.Lock()
		clinicalContext.CurrentMedications = medications
		mu.Unlock()
	}()

	// Gather allergies
	wg.Add(1)
	go func() {
		defer wg.Done()
		allergies, err := ca.contextServiceClient.GetPatientAllergies(ctx, request.PatientID)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
			return
		}
		mu.Lock()
		clinicalContext.Allergies = allergies
		mu.Unlock()
	}()

	// Gather conditions
	wg.Add(1)
	go func() {
		defer wg.Done()
		conditions, err := ca.contextServiceClient.GetPatientConditions(ctx, request.PatientID)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
			return
		}
		mu.Lock()
		clinicalContext.Conditions = conditions
		mu.Unlock()
	}()

	// Gather lab results
	wg.Add(1)
	go func() {
		defer wg.Done()
		labResults, err := ca.contextServiceClient.GetPatientLabResults(ctx, request.PatientID)
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
			return
		}
		mu.Lock()
		clinicalContext.LabResults = labResults
		mu.Unlock()
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		ca.logger.WithFields(logrus.Fields{
			"request_id": request.RequestID,
			"errors":     len(errors),
		}).Warn("Some context gathering operations failed")
		// Continue with partial context rather than failing completely
	}

	// Enrich with medication-specific context if needed
	if request.MedicationData != nil {
		ca.enrichWithMedicationContext(ctx, clinicalContext, request.MedicationData)
	}

	// Add request-specific context
	ca.addRequestContext(clinicalContext, request)

	executionTime := time.Since(start)
	ca.logger.WithFields(logrus.Fields{
		"request_id":        request.RequestID,
		"execution_time_ms": executionTime.Milliseconds(),
		"context_sources":   ca.getContextSources(clinicalContext),
	}).Info("Context assembly completed")

	return clinicalContext, nil
}

// AssembleEnhancedContext assembles enhanced context for medication intelligence
func (ca *ContextAssembler) AssembleEnhancedContext(ctx context.Context, request *models.MedicationIntelligenceRequest) (*models.ClinicalContext, error) {
	// Convert to Flow2Request for compatibility
	flow2Request := &models.Flow2Request{
		RequestID: request.RequestID,
		PatientID: request.PatientID,
		ActionType: "MEDICATION_INTELLIGENCE",
		MedicationData: map[string]interface{}{
			"medications": request.Medications,
		},
	}

	return ca.AssembleContext(ctx, flow2Request)
}

// AssembleContextForDoseOptimization assembles context specifically for dose optimization
func (ca *ContextAssembler) AssembleContextForDoseOptimization(ctx context.Context, request *models.DoseOptimizationRequest) (*models.ClinicalContext, error) {
	// Convert to Flow2Request for compatibility
	flow2Request := &models.Flow2Request{
		RequestID: request.RequestID,
		PatientID: request.PatientID,
		ActionType: "DOSE_OPTIMIZATION",
		MedicationData: map[string]interface{}{
			"medication_code": request.MedicationCode,
			"clinical_parameters": request.ClinicalParameters,
		},
	}

	return ca.AssembleContext(ctx, flow2Request)
}

// enrichWithMedicationContext enriches context with medication-specific information
func (ca *ContextAssembler) enrichWithMedicationContext(ctx context.Context, clinicalContext *models.ClinicalContext, medicationData map[string]interface{}) {
	// Extract medication codes from medication data
	var medicationCodes []string
	
	if medications, ok := medicationData["medications"].([]interface{}); ok {
		for _, med := range medications {
			if medMap, ok := med.(map[string]interface{}); ok {
				if code, ok := medMap["code"].(string); ok {
					medicationCodes = append(medicationCodes, code)
				}
			}
		}
	}

	if medicationCode, ok := medicationData["medication_code"].(string); ok {
		medicationCodes = append(medicationCodes, medicationCode)
	}

	// Get drug interactions for all medications
	if len(medicationCodes) > 0 {
		interactions, err := ca.medicationAPIClient.GetDrugInteractions(ctx, medicationCodes)
		if err != nil {
			ca.logger.WithError(err).Warn("Failed to get drug interactions")
		} else {
			// Store interactions in clinical context (we'll need to add this field to the model)
			// For now, we'll store it in a generic observations field
			if clinicalContext.Observations == nil {
				clinicalContext.Observations = []models.Observation{}
			}
			
			for _, interaction := range interactions {
				clinicalContext.Observations = append(clinicalContext.Observations, models.Observation{
					Code:      "drug_interaction",
					Name:      "Drug Interaction",
					Value:     interaction,
					Timestamp: time.Now(),
				})
			}
		}
	}
}

// addRequestContext adds request-specific context
func (ca *ContextAssembler) addRequestContext(clinicalContext *models.ClinicalContext, request *models.Flow2Request) {
	// Add processing hints as observations
	if request.ProcessingHints != nil {
		if clinicalContext.Observations == nil {
			clinicalContext.Observations = []models.Observation{}
		}

		for key, value := range request.ProcessingHints {
			clinicalContext.Observations = append(clinicalContext.Observations, models.Observation{
				Code:      "processing_hint",
				Name:      key,
				Value:     value,
				Timestamp: time.Now(),
			})
		}
	}

	// Add patient data as observations
	if request.PatientData != nil {
		if clinicalContext.Observations == nil {
			clinicalContext.Observations = []models.Observation{}
		}

		for key, value := range request.PatientData {
			clinicalContext.Observations = append(clinicalContext.Observations, models.Observation{
				Code:      "patient_data",
				Name:      key,
				Value:     value,
				Timestamp: time.Now(),
			})
		}
	}
}

// getContextSources returns a list of context sources that were used
func (ca *ContextAssembler) getContextSources(clinicalContext *models.ClinicalContext) []string {
	sources := []string{}

	if clinicalContext.PatientDemographics != nil {
		sources = append(sources, "demographics")
	}
	if len(clinicalContext.CurrentMedications) > 0 {
		sources = append(sources, "medications")
	}
	if len(clinicalContext.Allergies) > 0 {
		sources = append(sources, "allergies")
	}
	if len(clinicalContext.Conditions) > 0 {
		sources = append(sources, "conditions")
	}
	if len(clinicalContext.LabResults) > 0 {
		sources = append(sources, "lab_results")
	}
	if len(clinicalContext.Observations) > 0 {
		sources = append(sources, "observations")
	}

	return sources
}
