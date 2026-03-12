// Package clients provides HTTP/GraphQL clients for KB services.
//
// KB2GraphQLClient implements:
// - KB2ContextService (KB-2A: data assembly)
// - KB2IntelligenceService (KB-2B: phenotypes, risk, care gaps)
//
// Connects to: http://localhost:8082/graphql
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vaidshala/clinical-runtime-platform/adapters"
)

// KB2GraphQLClient implements KB2ContextService and KB2IntelligenceService
// by calling the KB-2 Clinical Context Service GraphQL API.
type KB2GraphQLClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewKB2GraphQLClient creates a new KB-2 GraphQL client.
func NewKB2GraphQLClient(endpoint string) *KB2GraphQLClient {
	return &KB2GraphQLClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB2GraphQLClientWithHTTP creates a client with custom HTTP client.
func NewKB2GraphQLClientWithHTTP(endpoint string, httpClient *http.Client) *KB2GraphQLClient {
	return &KB2GraphQLClient{
		endpoint:   endpoint,
		httpClient: httpClient,
	}
}

// ============================================================================
// GraphQL Request/Response Types
// ============================================================================

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors,omitempty"`
}

// parseGraphQLErrors handles both string arrays and object arrays for GraphQL errors.
// KB-2 returns errors as ["error message"] while standard GraphQL uses [{"message": "..."}]
func parseGraphQLErrors(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Try string array first (KB-2 format)
	var stringErrors []string
	if err := json.Unmarshal(raw, &stringErrors); err == nil {
		return stringErrors, nil
	}

	// Try object array (standard GraphQL format)
	var objectErrors []struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &objectErrors); err == nil {
		messages := make([]string, len(objectErrors))
		for i, e := range objectErrors {
			messages[i] = e.Message
		}
		return messages, nil
	}

	return []string{string(raw)}, nil
}

// ============================================================================
// KB-2A: KB2ContextService Implementation
// ============================================================================

// BuildPatientContext calls KB-2's buildContext mutation and parses FHIR input.
// This implements the KB2ContextService interface.
//
// KB-2 stores patient data and returns processing metadata (phenotypes, cacheHit).
// The client parses the input FHIR bundle to populate the response structure.
func (c *KB2GraphQLClient) BuildPatientContext(
	ctx context.Context,
	req adapters.KB2BuildRequest,
) (*adapters.KB2BuildResponse, error) {

	// Serialize patient data to JSON string (KB-2 expects string)
	patientJSON, err := json.Marshal(req.RawFHIRInput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patient data: %w", err)
	}

	// GraphQL mutation - KB-2 returns processing metadata, not full context
	query := `
		mutation BuildContext($input: BuildContextInput!) {
			buildContext(input: $input) {
				phenotypes
				cacheHit
				processedAt
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"patientId": req.PatientID,
			"patient":   string(patientJSON),
		},
	}

	// Execute GraphQL request
	var result struct {
		BuildContext struct {
			Phenotypes  []string `json:"phenotypes"`
			CacheHit    bool     `json:"cacheHit"`
			ProcessedAt string   `json:"processedAt"`
		} `json:"buildContext"`
	}

	if err := c.executeGraphQL(ctx, query, variables, &result); err != nil {
		return nil, fmt.Errorf("KB-2 buildContext failed: %w", err)
	}

	// Parse input FHIR data to populate response
	response := c.parseFHIRInput(req.PatientID, req.RawFHIRInput)

	return response, nil
}

// parseFHIRInput extracts patient data from FHIR Bundle input.
func (c *KB2GraphQLClient) parseFHIRInput(patientID string, input interface{}) *adapters.KB2BuildResponse {
	response := &adapters.KB2BuildResponse{
		Demographics: adapters.KB2Demographics{
			PatientID: patientID,
		},
		Conditions:  []adapters.KB2Condition{},
		Medications: []adapters.KB2Medication{},
		LabResults:  []adapters.KB2LabResult{},
		VitalSigns:  []adapters.KB2VitalSign{},
		Allergies:   []adapters.KB2Allergy{},
		Encounters:  []adapters.KB2Encounter{},
	}

	// Parse input as map
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return response
	}

	// Check for FHIR Bundle
	if inputMap["resourceType"] == "Bundle" {
		entries, ok := inputMap["entry"].([]map[string]interface{})
		if !ok {
			// Try []interface{} and convert
			if entriesArr, ok := inputMap["entry"].([]interface{}); ok {
				for _, e := range entriesArr {
					if entryMap, ok := e.(map[string]interface{}); ok {
						c.parseResource(entryMap, response)
					}
				}
			}
		} else {
			for _, entry := range entries {
				c.parseResource(entry, response)
			}
		}
	}

	return response
}

// parseResource extracts data from a FHIR Bundle entry.
func (c *KB2GraphQLClient) parseResource(entry map[string]interface{}, response *adapters.KB2BuildResponse) {
	resource, ok := entry["resource"].(map[string]interface{})
	if !ok {
		return
	}

	resourceType, _ := resource["resourceType"].(string)

	switch resourceType {
	case "Patient":
		c.parsePatient(resource, response)
	case "Condition":
		c.parseCondition(resource, response)
	case "MedicationRequest":
		c.parseMedication(resource, response)
	case "Observation":
		c.parseObservation(resource, response)
	}
}

// parsePatient extracts demographics from FHIR Patient resource.
func (c *KB2GraphQLClient) parsePatient(resource map[string]interface{}, response *adapters.KB2BuildResponse) {
	if gender, ok := resource["gender"].(string); ok {
		response.Demographics.Gender = gender
	}
	if birthDate, ok := resource["birthDate"].(string); ok {
		if bd, err := time.Parse("2006-01-02", birthDate); err == nil {
			response.Demographics.AgeYears = int(time.Since(bd).Hours() / 24 / 365)
		}
	}
}

// parseCondition extracts condition from FHIR Condition resource.
func (c *KB2GraphQLClient) parseCondition(resource map[string]interface{}, response *adapters.KB2BuildResponse) {
	cond := adapters.KB2Condition{
		ClinicalStatus: "active",
	}

	if code, ok := resource["code"].(map[string]interface{}); ok {
		if coding, ok := code["coding"].([]interface{}); ok && len(coding) > 0 {
			if c0, ok := coding[0].(map[string]interface{}); ok {
				cond.Code, _ = c0["code"].(string)
				cond.System, _ = c0["system"].(string)
				cond.Display, _ = c0["display"].(string)
			}
		}
	}

	response.Conditions = append(response.Conditions, cond)
}

// parseMedication extracts medication from FHIR MedicationRequest resource.
func (c *KB2GraphQLClient) parseMedication(resource map[string]interface{}, response *adapters.KB2BuildResponse) {
	med := adapters.KB2Medication{
		Status: "active",
	}

	if medCode, ok := resource["medicationCodeableConcept"].(map[string]interface{}); ok {
		if coding, ok := medCode["coding"].([]interface{}); ok && len(coding) > 0 {
			if c0, ok := coding[0].(map[string]interface{}); ok {
				med.Code, _ = c0["code"].(string)
				med.System, _ = c0["system"].(string)
				med.Display, _ = c0["display"].(string)
			}
		}
	}

	response.Medications = append(response.Medications, med)
}

// parseObservation extracts observation from FHIR Observation resource.
func (c *KB2GraphQLClient) parseObservation(resource map[string]interface{}, response *adapters.KB2BuildResponse) {
	lab := adapters.KB2LabResult{}

	if code, ok := resource["code"].(map[string]interface{}); ok {
		if coding, ok := code["coding"].([]interface{}); ok && len(coding) > 0 {
			if c0, ok := coding[0].(map[string]interface{}); ok {
				lab.Code, _ = c0["code"].(string)
				lab.System, _ = c0["system"].(string)
				lab.Display, _ = c0["display"].(string)
			}
		}
	}

	if valueQuantity, ok := resource["valueQuantity"].(map[string]interface{}); ok {
		if v, ok := valueQuantity["value"].(float64); ok {
			lab.Value = v
		}
		lab.Unit, _ = valueQuantity["unit"].(string)
	}

	response.LabResults = append(response.LabResults, lab)
}

// ============================================================================
// KB-2B: KB2IntelligenceService Implementation
// ============================================================================

// DetectPhenotypes calls KB-2's detectPhenotypes mutation.
func (c *KB2GraphQLClient) DetectPhenotypes(
	ctx context.Context,
	patientID string,
	data map[string]interface{},
) ([]adapters.DetectedPhenotype, error) {

	patientDataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patient data: %w", err)
	}

	query := `
		mutation DetectPhenotypes($input: PhenotypeDetectionInput!) {
			detectPhenotypes(input: $input) {
				patientId
				detectedPhenotypes {
					phenotypeId
					confidence
					detectedAt
					supportingEvidence
				}
				totalPhenotypes
				processingTimeMs
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"patientId":   patientID,
			"patientData": string(patientDataJSON),
		},
	}

	var result struct {
		DetectPhenotypes struct {
			PatientID          string `json:"patientId"`
			DetectedPhenotypes []struct {
				PhenotypeID        string   `json:"phenotypeId"`
				Confidence         float64  `json:"confidence"`
				DetectedAt         string   `json:"detectedAt"`
				SupportingEvidence []string `json:"supportingEvidence"`
			} `json:"detectedPhenotypes"`
			TotalPhenotypes  int   `json:"totalPhenotypes"`
			ProcessingTimeMs int64 `json:"processingTimeMs"`
		} `json:"detectPhenotypes"`
	}

	if err := c.executeGraphQL(ctx, query, variables, &result); err != nil {
		return nil, fmt.Errorf("KB-2 detectPhenotypes failed: %w", err)
	}

	// Convert to adapters.DetectedPhenotype
	phenotypes := make([]adapters.DetectedPhenotype, 0, len(result.DetectPhenotypes.DetectedPhenotypes))
	for _, p := range result.DetectPhenotypes.DetectedPhenotypes {
		phenotypes = append(phenotypes, adapters.DetectedPhenotype{
			PhenotypeID: p.PhenotypeID,
			Name:        p.PhenotypeID, // Use ID as name if not available
			Confidence:  p.Confidence,
			Evidence:    p.SupportingEvidence,
		})
	}

	return phenotypes, nil
}

// AssessRisk calls KB-2's assessRisk mutation.
func (c *KB2GraphQLClient) AssessRisk(
	ctx context.Context,
	patientID string,
	data map[string]interface{},
) (*adapters.RiskAssessmentResult, error) {

	patientDataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patient data: %w", err)
	}

	query := `
		mutation AssessRisk($input: RiskAssessmentInput!) {
			assessRisk(input: $input) {
				patientId
				riskScores
				riskFactors
				recommendations
				confidenceScore
				assessmentTimestamp
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"patientId":   patientID,
			"patientData": string(patientDataJSON),
		},
	}

	var result struct {
		AssessRisk struct {
			PatientID           string                 `json:"patientId"`
			RiskScores          map[string]interface{} `json:"riskScores"`
			RiskFactors         map[string]interface{} `json:"riskFactors"`
			Recommendations     []string               `json:"recommendations"`
			ConfidenceScore     float64                `json:"confidenceScore"`
			AssessmentTimestamp string                 `json:"assessmentTimestamp"`
		} `json:"assessRisk"`
	}

	if err := c.executeGraphQL(ctx, query, variables, &result); err != nil {
		return nil, fmt.Errorf("KB-2 assessRisk failed: %w", err)
	}

	// Convert risk scores to float64
	riskScores := make(map[string]float64)
	for k, v := range result.AssessRisk.RiskScores {
		if f, ok := v.(float64); ok {
			riskScores[k] = f
		}
	}

	// Convert risk factors to categories
	riskCategories := make(map[string]string)
	for k, v := range result.AssessRisk.RiskFactors {
		if s, ok := v.(string); ok {
			riskCategories[k] = s
		}
	}

	return &adapters.RiskAssessmentResult{
		RiskScores:      riskScores,
		RiskCategories:  riskCategories,
		ConfidenceScore: result.AssessRisk.ConfidenceScore,
		ClinicalFlags:   make(map[string]bool),
	}, nil
}

// IdentifyCareGaps calls KB-2's patientCareGaps query.
func (c *KB2GraphQLClient) IdentifyCareGaps(
	ctx context.Context,
	patientID string,
) ([]adapters.IdentifiedCareGap, error) {

	query := `
		query PatientCareGaps($patientId: ID!) {
			patientCareGaps(patientId: $patientId) {
				id
				type
				description
				priority
				dueDays
				actions
			}
		}
	`

	variables := map[string]interface{}{
		"patientId": patientID,
	}

	var result struct {
		PatientCareGaps []struct {
			ID          string   `json:"id"`
			Type        string   `json:"type"`
			Description string   `json:"description"`
			Priority    string   `json:"priority"`
			DueDays     int      `json:"dueDays"`
			Actions     []string `json:"actions"`
		} `json:"patientCareGaps"`
	}

	if err := c.executeGraphQL(ctx, query, variables, &result); err != nil {
		return nil, fmt.Errorf("KB-2 patientCareGaps failed: %w", err)
	}

	// Convert to adapters.IdentifiedCareGap
	careGaps := make([]adapters.IdentifiedCareGap, 0, len(result.PatientCareGaps))
	for _, g := range result.PatientCareGaps {
		action := ""
		if len(g.Actions) > 0 {
			action = g.Actions[0]
		}
		careGaps = append(careGaps, adapters.IdentifiedCareGap{
			GapID:             g.ID,
			MeasureID:         g.Type,
			Description:       g.Description,
			Priority:          g.Priority,
			RecommendedAction: action,
		})
	}

	return careGaps, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// executeGraphQL sends a GraphQL request and unmarshals the response.
func (c *KB2GraphQLClient) executeGraphQL(
	ctx context.Context,
	query string,
	variables map[string]interface{},
	result interface{},
) error {

	// Build request
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse GraphQL response
	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		errors, _ := parseGraphQLErrors(gqlResp.Errors)
		if len(errors) > 0 {
			return fmt.Errorf("GraphQL error: %s", errors[0])
		}
	}

	// Unmarshal data into result
	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// HealthCheck verifies KB-2 service is healthy.
func (c *KB2GraphQLClient) HealthCheck(ctx context.Context) error {
	query := `{ systemHealth { status } }`

	var result struct {
		SystemHealth struct {
			Status string `json:"status"`
		} `json:"systemHealth"`
	}

	if err := c.executeGraphQL(ctx, query, nil, &result); err != nil {
		return err
	}

	if result.SystemHealth.Status != "healthy" {
		return fmt.Errorf("KB-2 unhealthy: %s", result.SystemHealth.Status)
	}

	return nil
}
