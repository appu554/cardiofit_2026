// Package services provides data source registry for clinical data federation
package services

import (
	"context"
	"fmt"
	"log"
	"time"
)

// DataSourceConfig represents configuration for a clinical data source
type DataSourceConfig struct {
	ServiceType string            `json:"service_type"`
	Endpoint    string            `json:"endpoint"`
	Query       string            `json:"query,omitempty"`
	Method      string            `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	TimeoutMs   int32             `json:"timeout_ms"`
}

// FieldData represents fetched clinical field data
type FieldData struct {
	Data              map[string]interface{} `json:"data"`
	CompletenessScore float64                `json:"completeness_score"`
	Timestamp         time.Time              `json:"timestamp"`
	Source            string                 `json:"source"`
	ResponseTimeMs    int64                  `json:"response_time_ms"`
}

// DataSourceRegistry manages routing to different clinical data sources
type DataSourceRegistry struct {
	sources map[string]*DataSourceConfig
}

// NewDataSourceRegistry creates a new data source registry
func NewDataSourceRegistry() *DataSourceRegistry {
	registry := &DataSourceRegistry{
		sources: make(map[string]*DataSourceConfig),
	}
	
	// Initialize default data source configurations
	registry.loadDefaultSources()
	
	return registry
}

// loadDefaultSources loads default clinical data source configurations
func (dr *DataSourceRegistry) loadDefaultSources() {
	// Patient Service
	dr.sources["patient_demographics"] = &DataSourceConfig{
		ServiceType: "grpc",
		Endpoint:    "localhost:8003",
		Method:      "GetPatientDemographics",
		TimeoutMs:   2000,
	}
	
	// Medication Service (Flow2 Go Engine)
	dr.sources["patient_medications"] = &DataSourceConfig{
		ServiceType: "grpc", 
		Endpoint:    "localhost:8080",
		Method:      "GetPatientMedications",
		TimeoutMs:   3000,
	}
	
	// Observation Service
	dr.sources["patient_observations"] = &DataSourceConfig{
		ServiceType: "rest",
		Endpoint:    "http://localhost:8010",
		Query:       "/api/observations/patient/{patient_id}",
		TimeoutMs:   2500,
	}
	
	// FHIR Store Direct
	dr.sources["fhir_resources"] = &DataSourceConfig{
		ServiceType: "fhir",
		Endpoint:    "https://healthcare.googleapis.com/v1/projects/{project}/locations/{location}/datasets/{dataset}/fhirStores/{store}/fhir",
		TimeoutMs:   5000,
	}
	
	// Safety Gateway
	dr.sources["safety_alerts"] = &DataSourceConfig{
		ServiceType: "grpc",
		Endpoint:    "localhost:8020", // Safety Gateway port
		Method:      "GetSafetyAlerts",
		TimeoutMs:   1500,
	}
	
	// Elasticsearch Clinical Data
	dr.sources["clinical_search"] = &DataSourceConfig{
		ServiceType: "elasticsearch",
		Endpoint:    "http://localhost:9200",
		Query:       "/clinical-data/_search",
		TimeoutMs:   3000,
	}
	
	// Apollo Federation
	dr.sources["federated_query"] = &DataSourceConfig{
		ServiceType: "graphql",
		Endpoint:    "http://localhost:4000/graphql",
		TimeoutMs:   4000,
	}
}

// FetchFieldWithFreshness fetches a specific clinical data field with freshness validation
func (dr *DataSourceRegistry) FetchFieldWithFreshness(
	ctx context.Context,
	field string,
	patientID string,
	freshnessRequirement int32,
) (*FieldData, error) {
	
	startTime := time.Now()
	
	config, exists := dr.sources[field]
	if !exists {
		return nil, fmt.Errorf("unknown field: %s", field)
	}
	
	log.Printf("Fetching field %s for patient %s from %s", field, patientID, config.ServiceType)
	
	// Route to appropriate service based on type
	data, err := dr.routeToService(ctx, config, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", config.ServiceType, err)
	}
	
	responseTime := time.Since(startTime).Milliseconds()
	
	// Create field data response
	fieldData := &FieldData{
		Data:              data,
		CompletenessScore: dr.calculateCompleteness(data),
		Timestamp:         time.Now().UTC(),
		Source:            config.ServiceType,
		ResponseTimeMs:    responseTime,
	}
	
	// Validate freshness requirement
	if !dr.validateFreshness(fieldData, freshnessRequirement) {
		return nil, fmt.Errorf("data too stale for field %s, age: %v, requirement: %d minutes", 
			field, time.Since(fieldData.Timestamp), freshnessRequirement)
	}
	
	log.Printf("Successfully fetched field %s in %dms", field, responseTime)
	return fieldData, nil
}

// FetchFields fetches multiple clinical data fields
func (dr *DataSourceRegistry) FetchFields(ctx context.Context, patientID string, fields []string) (*FieldData, error) {
	startTime := time.Now()
	
	combinedData := make(map[string]interface{})
	var totalResponseTime int64
	sources := make(map[string]bool)
	
	// Fetch each field
	for _, field := range fields {
		fieldData, err := dr.FetchFieldWithFreshness(ctx, field, patientID, 60) // 60 minutes default freshness
		if err != nil {
			log.Printf("Warning: Failed to fetch field %s: %v", field, err)
			// Continue with other fields for partial results
			combinedData[field] = map[string]interface{}{
				"error": err.Error(),
				"status": "unavailable",
			}
			continue
		}
		
		// Merge field data
		for key, value := range fieldData.Data {
			combinedData[key] = value
		}
		
		totalResponseTime += fieldData.ResponseTimeMs
		sources[fieldData.Source] = true
	}
	
	// Calculate combined completeness
	completeness := dr.calculateCompleteness(combinedData)
	
	result := &FieldData{
		Data:              combinedData,
		CompletenessScore: completeness,
		Timestamp:         time.Now().UTC(),
		Source:            fmt.Sprintf("multi_source_%d", len(sources)),
		ResponseTimeMs:    totalResponseTime,
	}
	
	log.Printf("Fetched %d fields in %dms with %.2f%% completeness", 
		len(fields), time.Since(startTime).Milliseconds(), completeness*100)
	
	return result, nil
}

// routeToService routes requests to appropriate service based on configuration
func (dr *DataSourceRegistry) routeToService(ctx context.Context, config *DataSourceConfig, patientID string) (map[string]interface{}, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutMs)*time.Millisecond)
	defer cancel()
	
	switch config.ServiceType {
	case "grpc":
		return dr.fetchFromGRPC(timeoutCtx, config, patientID)
	case "rest":
		return dr.fetchFromREST(timeoutCtx, config, patientID)
	case "graphql":
		return dr.fetchFromGraphQL(timeoutCtx, config, patientID)
	case "fhir":
		return dr.fetchFromFHIR(timeoutCtx, config, patientID)
	case "elasticsearch":
		return dr.fetchFromElasticsearch(timeoutCtx, config, patientID)
	default:
		return nil, fmt.Errorf("unsupported service type: %s", config.ServiceType)
	}
}

// fetchFromGRPC fetches data from gRPC services (mock implementation)
func (dr *DataSourceRegistry) fetchFromGRPC(ctx context.Context, config *DataSourceConfig, patientID string) (map[string]interface{}, error) {
	// Mock gRPC response - in production, you'd use actual gRPC clients
	log.Printf("Mock gRPC call to %s.%s for patient %s", config.Endpoint, config.Method, patientID)
	
	// Simulate network delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
	}
	
	// Mock response based on method
	switch config.Method {
	case "GetPatientDemographics":
		return map[string]interface{}{
			"patient_id": patientID,
			"age":        35,
			"gender":     "M",
			"weight_kg":  75.0,
			"height_cm":  175,
		}, nil
	case "GetPatientMedications":
		return map[string]interface{}{
			"medications": []interface{}{
				map[string]interface{}{
					"name":   "Lisinopril",
					"dose":   "10mg",
					"frequency": "daily",
				},
			},
		}, nil
	case "GetSafetyAlerts":
		return map[string]interface{}{
			"alerts": []interface{}{},
		}, nil
	default:
		return map[string]interface{}{
			"status": "success",
			"data":   "mock_grpc_response",
		}, nil
	}
}

// fetchFromREST fetches data from REST APIs (mock implementation)
func (dr *DataSourceRegistry) fetchFromREST(ctx context.Context, config *DataSourceConfig, patientID string) (map[string]interface{}, error) {
	// Mock REST API response
	log.Printf("Mock REST call to %s for patient %s", config.Endpoint, patientID)
	
	// Simulate network delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(15 * time.Millisecond):
	}
	
	return map[string]interface{}{
		"observations": []interface{}{
			map[string]interface{}{
				"type":  "blood_pressure",
				"value": "120/80",
				"unit":  "mmHg",
				"date":  time.Now().Format(time.RFC3339),
			},
		},
	}, nil
}

// fetchFromGraphQL fetches data from GraphQL services (mock implementation)
func (dr *DataSourceRegistry) fetchFromGraphQL(ctx context.Context, config *DataSourceConfig, patientID string) (map[string]interface{}, error) {
	// Mock GraphQL response
	log.Printf("Mock GraphQL call to %s for patient %s", config.Endpoint, patientID)
	
	// Simulate network delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(20 * time.Millisecond):
	}
	
	return map[string]interface{}{
		"federated_data": map[string]interface{}{
			"patient_summary": map[string]interface{}{
				"id":   patientID,
				"name": "John Doe",
			},
		},
	}, nil
}

// fetchFromFHIR fetches data from FHIR stores (mock implementation)
func (dr *DataSourceRegistry) fetchFromFHIR(ctx context.Context, config *DataSourceConfig, patientID string) (map[string]interface{}, error) {
	// Mock FHIR response
	log.Printf("Mock FHIR call to %s for patient %s", config.Endpoint, patientID)
	
	// Simulate network delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(25 * time.Millisecond):
	}
	
	return map[string]interface{}{
		"resourceType": "Patient",
		"id":           patientID,
		"active":       true,
		"name": []interface{}{
			map[string]interface{}{
				"family": "Doe",
				"given":  []string{"John"},
			},
		},
	}, nil
}

// fetchFromElasticsearch fetches data from Elasticsearch (mock implementation)
func (dr *DataSourceRegistry) fetchFromElasticsearch(ctx context.Context, config *DataSourceConfig, patientID string) (map[string]interface{}, error) {
	// Mock Elasticsearch response
	log.Printf("Mock Elasticsearch call to %s for patient %s", config.Endpoint, patientID)
	
	// Simulate network delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Millisecond):
	}
	
	return map[string]interface{}{
		"hits": map[string]interface{}{
			"total": map[string]interface{}{
				"value": 1,
			},
			"hits": []interface{}{
				map[string]interface{}{
					"_source": map[string]interface{}{
						"patient_id":    patientID,
						"clinical_note": "Patient doing well",
						"timestamp":     time.Now().Format(time.RFC3339),
					},
				},
			},
		},
	}, nil
}

// validateFreshness checks if data meets freshness requirements
func (dr *DataSourceRegistry) validateFreshness(fieldData *FieldData, freshnessMinutes int32) bool {
	if freshnessMinutes <= 0 {
		return true // No freshness requirement
	}
	
	age := time.Since(fieldData.Timestamp)
	maxAge := time.Duration(freshnessMinutes) * time.Minute
	
	return age <= maxAge
}

// calculateCompleteness calculates data completeness score
func (dr *DataSourceRegistry) calculateCompleteness(data map[string]interface{}) float64 {
	if len(data) == 0 {
		return 0.0
	}
	
	totalFields := len(data)
	validFields := 0
	
	for _, value := range data {
		if dr.isValidValue(value) {
			validFields++
		}
	}
	
	return float64(validFields) / float64(totalFields)
}

// isValidValue checks if a field value is valid (not nil, empty, or error)
func (dr *DataSourceRegistry) isValidValue(value interface{}) bool {
	if value == nil {
		return false
	}
	
	switch v := value.(type) {
	case string:
		return v != "" && v != "null" && v != "undefined"
	case map[string]interface{}:
		// Check if it's an error object
		if _, hasError := v["error"]; hasError {
			return false
		}
		return len(v) > 0
	case []interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// RegisterSource adds a new data source configuration
func (dr *DataSourceRegistry) RegisterSource(fieldName string, config *DataSourceConfig) {
	dr.sources[fieldName] = config
	log.Printf("Registered data source for field %s: %s", fieldName, config.ServiceType)
}

// GetSourceConfig returns the configuration for a specific field
func (dr *DataSourceRegistry) GetSourceConfig(field string) (*DataSourceConfig, bool) {
	config, exists := dr.sources[field]
	return config, exists
}

// ListSources returns all registered data source configurations
func (dr *DataSourceRegistry) ListSources() map[string]*DataSourceConfig {
	// Return a copy to prevent external modification
	sources := make(map[string]*DataSourceConfig)
	for k, v := range dr.sources {
		sources[k] = v
	}
	return sources
}