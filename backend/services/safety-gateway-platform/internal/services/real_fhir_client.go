package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// RealFHIRClient implements a real FHIR client that connects to Google Cloud Healthcare FHIR Store
type RealFHIRClient struct {
	logger    *logger.Logger
	baseURL   string
	client    *http.Client
	projectID string
	location  string
	datasetID string
	storeID   string
}

// Ensure RealFHIRClient implements FHIRClient interface
var _ FHIRClient = (*RealFHIRClient)(nil)

// NewRealFHIRClient creates a new real FHIR client
func NewRealFHIRClient(logger *logger.Logger, projectID, location, datasetID, storeID string) *RealFHIRClient {
	baseURL := fmt.Sprintf("https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
		projectID, location, datasetID, storeID)

	return &RealFHIRClient{
		logger:    logger,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
		projectID: projectID,
		location:  location,
		datasetID: datasetID,
		storeID:   storeID,
	}
}

// GetDemographics fetches patient demographics from FHIR Store
func (r *RealFHIRClient) GetDemographics(ctx context.Context, patientID string) (*types.Demographics, error) {
	url := fmt.Sprintf("%s/Patient/%s", r.baseURL, patientID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers (you'll need to implement OAuth2 token)
	// req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/fhir+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: %d - %s", resp.StatusCode, string(body))
	}

	var fhirPatient map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fhirPatient); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert FHIR Patient to Demographics
	demographics := r.convertFHIRPatientToDemographics(fhirPatient)
	
	r.logger.Debug("Real FHIR demographics fetched", 
		zap.String("patient_id", patientID),
		zap.Int("age", demographics.Age))

	return demographics, nil
}

// GetActiveMedications fetches active medications from FHIR Store
func (r *RealFHIRClient) GetActiveMedications(ctx context.Context, patientID string) ([]types.Medication, error) {
	url := fmt.Sprintf("%s/MedicationRequest?patient=%s&status=active", r.baseURL, patientID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: %d - %s", resp.StatusCode, string(body))
	}

	var bundle map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert FHIR Bundle to Medications
	medications := r.convertFHIRBundleToMedications(bundle)
	
	r.logger.Debug("Real FHIR medications fetched", 
		zap.String("patient_id", patientID),
		zap.Int("count", len(medications)))

	return medications, nil
}

// GetAllergies fetches allergies from FHIR Store
func (r *RealFHIRClient) GetAllergies(ctx context.Context, patientID string) ([]types.Allergy, error) {
	url := fmt.Sprintf("%s/AllergyIntolerance?patient=%s", r.baseURL, patientID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: %d - %s", resp.StatusCode, string(body))
	}

	var bundle map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert FHIR Bundle to Allergies
	allergies := r.convertFHIRBundleToAllergies(bundle)
	
	r.logger.Debug("Real FHIR allergies fetched", 
		zap.String("patient_id", patientID),
		zap.Int("count", len(allergies)))

	return allergies, nil
}

// GetConditions fetches conditions from FHIR Store
func (r *RealFHIRClient) GetConditions(ctx context.Context, patientID string) ([]types.Condition, error) {
	url := fmt.Sprintf("%s/Condition?patient=%s", r.baseURL, patientID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: %d - %s", resp.StatusCode, string(body))
	}

	var bundle map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert FHIR Bundle to Conditions
	conditions := r.convertFHIRBundleToConditions(bundle)
	
	r.logger.Debug("Real FHIR conditions fetched", 
		zap.String("patient_id", patientID),
		zap.Int("count", len(conditions)))

	return conditions, nil
}

// GetRecentVitals fetches recent vital signs from FHIR Store
func (r *RealFHIRClient) GetRecentVitals(ctx context.Context, patientID string, hours int) ([]types.VitalSign, error) {
	// Calculate the date filter for recent vitals
	since := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02")
	url := fmt.Sprintf("%s/Observation?patient=%s&category=vital-signs&date=ge%s", r.baseURL, patientID, since)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: %d - %s", resp.StatusCode, string(body))
	}

	var bundle map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert FHIR Bundle to VitalSigns
	vitals := r.convertFHIRBundleToVitals(bundle)
	
	r.logger.Debug("Real FHIR vitals fetched", 
		zap.String("patient_id", patientID),
		zap.Int("count", len(vitals)))

	return vitals, nil
}

// GetRecentLabResults fetches recent lab results from FHIR Store
func (r *RealFHIRClient) GetRecentLabResults(ctx context.Context, patientID string, hours int) ([]types.LabResult, error) {
	// Calculate the date filter for recent labs
	since := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02")
	url := fmt.Sprintf("%s/Observation?patient=%s&category=laboratory&date=ge%s", r.baseURL, patientID, since)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: %d - %s", resp.StatusCode, string(body))
	}

	var bundle map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert FHIR Bundle to LabResults
	labs := r.convertFHIRBundleToLabResults(bundle)
	
	r.logger.Debug("Real FHIR lab results fetched", 
		zap.String("patient_id", patientID),
		zap.Int("count", len(labs)))

	return labs, nil
}

// GetRecentEncounters fetches recent encounters from FHIR Store
func (r *RealFHIRClient) GetRecentEncounters(ctx context.Context, patientID string, days int) ([]types.Encounter, error) {
	// Calculate the date filter for recent encounters
	since := time.Now().Add(-time.Duration(days*24) * time.Hour).Format("2006-01-02")
	url := fmt.Sprintf("%s/Encounter?patient=%s&date=ge%s", r.baseURL, patientID, since)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: %d - %s", resp.StatusCode, string(body))
	}

	var bundle map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert FHIR Bundle to Encounters
	encounters := r.convertFHIRBundleToEncounters(bundle)
	
	r.logger.Debug("Real FHIR encounters fetched", 
		zap.String("patient_id", patientID),
		zap.Int("count", len(encounters)))

	return encounters, nil
}

// Conversion methods to transform FHIR resources to internal types

// convertFHIRPatientToDemographics converts FHIR Patient resource to Demographics
func (r *RealFHIRClient) convertFHIRPatientToDemographics(fhirPatient map[string]interface{}) *types.Demographics {
	demographics := &types.Demographics{
		PatientID: getStringFromPath(fhirPatient, "id"),
		Gender:    getStringFromPath(fhirPatient, "gender"),
	}

	// Extract birth date and calculate age
	if birthDate := getStringFromPath(fhirPatient, "birthDate"); birthDate != "" {
		if birth, err := time.Parse("2006-01-02", birthDate); err == nil {
			demographics.Age = int(time.Since(birth).Hours() / 24 / 365.25)
			demographics.DateOfBirth = birth
		}
	}

	// Extract name
	if names, ok := fhirPatient["name"].([]interface{}); ok && len(names) > 0 {
		if name, ok := names[0].(map[string]interface{}); ok {
			if given, ok := name["given"].([]interface{}); ok && len(given) > 0 {
				demographics.FirstName = given[0].(string)
			}
			if family, ok := name["family"].(string); ok {
				demographics.LastName = family
			}
		}
	}

	return demographics
}

// convertFHIRBundleToMedications converts FHIR Bundle to Medications
func (r *RealFHIRClient) convertFHIRBundleToMedications(bundle map[string]interface{}) []types.Medication {
	var medications []types.Medication

	if entries, ok := bundle["entry"].([]interface{}); ok {
		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					if resourceType := getStringFromPath(resource, "resourceType"); resourceType == "MedicationRequest" {
						medication := types.Medication{
							ID:        getStringFromPath(resource, "id"),
							Name:      extractMedicationName(resource),
							Dosage:    extractDosage(resource),
							Frequency: extractFrequency(resource),
							StartDate: extractDate(resource, "authoredOn"),
							Status:    getStringFromPath(resource, "status"),
							Prescriber: extractPrescriber(resource),
						}
						medications = append(medications, medication)
					}
				}
			}
		}
	}

	return medications
}

// convertFHIRBundleToAllergies converts FHIR Bundle to Allergies
func (r *RealFHIRClient) convertFHIRBundleToAllergies(bundle map[string]interface{}) []types.Allergy {
	var allergies []types.Allergy

	if entries, ok := bundle["entry"].([]interface{}); ok {
		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					if resourceType := getStringFromPath(resource, "resourceType"); resourceType == "AllergyIntolerance" {
						allergy := types.Allergy{
							ID:       getStringFromPath(resource, "id"),
							Allergen: extractAllergen(resource),
							Severity: getStringFromPath(resource, "criticality"),
							Reaction: extractReaction(resource),
							OnsetDate: extractDate(resource, "onsetDateTime"),
						}
						allergies = append(allergies, allergy)
					}
				}
			}
		}
	}

	return allergies
}

// convertFHIRBundleToConditions converts FHIR Bundle to Conditions
func (r *RealFHIRClient) convertFHIRBundleToConditions(bundle map[string]interface{}) []types.Condition {
	var conditions []types.Condition

	if entries, ok := bundle["entry"].([]interface{}); ok {
		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					if resourceType := getStringFromPath(resource, "resourceType"); resourceType == "Condition" {
						condition := types.Condition{
							ID:          getStringFromPath(resource, "id"),
							Code:        extractConditionCode(resource),
							Display:     extractConditionDescription(resource),
							Status:      getStringFromPath(resource, "clinicalStatus", "coding", "0", "code"),
							Severity:    extractSeverity(resource),
							OnsetDate:   extractDate(resource, "onsetDateTime"),
							DiagnosedBy: extractDiagnoser(resource),
						}
						conditions = append(conditions, condition)
					}
				}
			}
		}
	}

	return conditions
}

// convertFHIRBundleToVitals converts FHIR Bundle to VitalSigns
func (r *RealFHIRClient) convertFHIRBundleToVitals(bundle map[string]interface{}) []types.VitalSign {
	var vitals []types.VitalSign

	if entries, ok := bundle["entry"].([]interface{}); ok {
		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					if resourceType := getStringFromPath(resource, "resourceType"); resourceType == "Observation" {
						vital := types.VitalSign{
							ID:        getStringFromPath(resource, "id"),
							Type:      extractVitalType(resource),
							Value:     extractVitalValue(resource),
							Unit:      extractVitalUnit(resource),
							Timestamp: extractDate(resource, "effectiveDateTime"),
							Status:    getStringFromPath(resource, "status"),
						}
						vitals = append(vitals, vital)
					}
				}
			}
		}
	}

	return vitals
}

// convertFHIRBundleToLabResults converts FHIR Bundle to LabResults
func (r *RealFHIRClient) convertFHIRBundleToLabResults(bundle map[string]interface{}) []types.LabResult {
	var labs []types.LabResult

	if entries, ok := bundle["entry"].([]interface{}); ok {
		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					if resourceType := getStringFromPath(resource, "resourceType"); resourceType == "Observation" {
						lab := types.LabResult{
							ID:           getStringFromPath(resource, "id"),
							TestName:     extractLabTestName(resource),
							Value:        extractLabValue(resource),
							Unit:         extractLabUnit(resource),
							ReferenceRange: extractReferenceRange(resource),
							Status:       getStringFromPath(resource, "status"),
							Timestamp:    extractDate(resource, "effectiveDateTime"),
						}
						labs = append(labs, lab)
					}
				}
			}
		}
	}

	return labs
}

// convertFHIRBundleToEncounters converts FHIR Bundle to Encounters
func (r *RealFHIRClient) convertFHIRBundleToEncounters(bundle map[string]interface{}) []types.Encounter {
	var encounters []types.Encounter

	if entries, ok := bundle["entry"].([]interface{}); ok {
		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					if resourceType := getStringFromPath(resource, "resourceType"); resourceType == "Encounter" {
						encounter := types.Encounter{
							ID:        getStringFromPath(resource, "id"),
							Type:      extractEncounterType(resource),
							Status:    getStringFromPath(resource, "status"),
							StartTime: extractDate(resource, "period", "start"),
							Provider:  extractProvider(resource),
							Location:  extractLocation(resource),
						}

						// Handle EndTime as pointer
						if endTime := extractDate(resource, "period", "end"); !endTime.IsZero() {
							encounter.EndTime = &endTime
						}
						encounters = append(encounters, encounter)
					}
				}
			}
		}
	}

	return encounters
}

// Helper functions for extracting data from FHIR resources

// getStringFromPath safely extracts a string value from nested map using path
func getStringFromPath(data map[string]interface{}, path ...string) string {
	current := data
	for i, key := range path {
		if i == len(path)-1 {
			if val, ok := current[key].(string); ok {
				return val
			}
			return ""
		}
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else if next, ok := current[key].([]interface{}); ok && len(next) > 0 {
			if nextMap, ok := next[0].(map[string]interface{}); ok {
				current = nextMap
			} else {
				return ""
			}
		} else {
			return ""
		}
	}
	return ""
}

// extractMedicationName extracts medication name from MedicationRequest
func extractMedicationName(resource map[string]interface{}) string {
	// Try medicationCodeableConcept first
	if medConcept, ok := resource["medicationCodeableConcept"].(map[string]interface{}); ok {
		if coding, ok := medConcept["coding"].([]interface{}); ok && len(coding) > 0 {
			if code, ok := coding[0].(map[string]interface{}); ok {
				if display, ok := code["display"].(string); ok {
					return display
				}
			}
		}
		if text, ok := medConcept["text"].(string); ok {
			return text
		}
	}
	return "Unknown Medication"
}

// extractDosage extracts dosage information
func extractDosage(resource map[string]interface{}) string {
	if dosageInstr, ok := resource["dosageInstruction"].([]interface{}); ok && len(dosageInstr) > 0 {
		if dosage, ok := dosageInstr[0].(map[string]interface{}); ok {
			if text, ok := dosage["text"].(string); ok {
				return text
			}
			if doseAndRate, ok := dosage["doseAndRate"].([]interface{}); ok && len(doseAndRate) > 0 {
				if dose, ok := doseAndRate[0].(map[string]interface{}); ok {
					if doseQuantity, ok := dose["doseQuantity"].(map[string]interface{}); ok {
						value := ""
						unit := ""
						if v, ok := doseQuantity["value"].(float64); ok {
							value = fmt.Sprintf("%.0f", v)
						}
						if u, ok := doseQuantity["unit"].(string); ok {
							unit = u
						}
						return fmt.Sprintf("%s %s", value, unit)
					}
				}
			}
		}
	}
	return ""
}

// extractFrequency extracts frequency information
func extractFrequency(resource map[string]interface{}) string {
	if dosageInstr, ok := resource["dosageInstruction"].([]interface{}); ok && len(dosageInstr) > 0 {
		if dosage, ok := dosageInstr[0].(map[string]interface{}); ok {
			if timing, ok := dosage["timing"].(map[string]interface{}); ok {
				if repeat, ok := timing["repeat"].(map[string]interface{}); ok {
					if frequency, ok := repeat["frequency"].(float64); ok {
						if period, ok := repeat["period"].(float64); ok {
							if periodUnit, ok := repeat["periodUnit"].(string); ok {
								return fmt.Sprintf("%.0f times per %.0f %s", frequency, period, periodUnit)
							}
						}
					}
				}
				if code, ok := timing["code"].(map[string]interface{}); ok {
					if text, ok := code["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

// extractDate extracts date from FHIR resource
func extractDate(resource map[string]interface{}, path ...string) time.Time {
	dateStr := getStringFromPath(resource, path...)
	if dateStr == "" {
		return time.Time{}
	}

	// Try different date formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date
		}
	}

	return time.Time{}
}

// extractPrescriber extracts prescriber information
func extractPrescriber(resource map[string]interface{}) string {
	if requester, ok := resource["requester"].(map[string]interface{}); ok {
		if display, ok := requester["display"].(string); ok {
			return display
		}
		if reference, ok := requester["reference"].(string); ok {
			return reference
		}
	}
	return ""
}

// extractAllergen extracts allergen from AllergyIntolerance
func extractAllergen(resource map[string]interface{}) string {
	if code, ok := resource["code"].(map[string]interface{}); ok {
		if coding, ok := code["coding"].([]interface{}); ok && len(coding) > 0 {
			if allergen, ok := coding[0].(map[string]interface{}); ok {
				if display, ok := allergen["display"].(string); ok {
					return display
				}
			}
		}
		if text, ok := code["text"].(string); ok {
			return text
		}
	}
	return ""
}

// extractReaction extracts reaction from AllergyIntolerance
func extractReaction(resource map[string]interface{}) string {
	if reactions, ok := resource["reaction"].([]interface{}); ok && len(reactions) > 0 {
		if reaction, ok := reactions[0].(map[string]interface{}); ok {
			if manifestations, ok := reaction["manifestation"].([]interface{}); ok && len(manifestations) > 0 {
				if manifestation, ok := manifestations[0].(map[string]interface{}); ok {
					if coding, ok := manifestation["coding"].([]interface{}); ok && len(coding) > 0 {
						if code, ok := coding[0].(map[string]interface{}); ok {
							if display, ok := code["display"].(string); ok {
								return display
							}
						}
					}
					if text, ok := manifestation["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

// extractConditionCode extracts condition code
func extractConditionCode(resource map[string]interface{}) string {
	if code, ok := resource["code"].(map[string]interface{}); ok {
		if coding, ok := code["coding"].([]interface{}); ok && len(coding) > 0 {
			if conditionCode, ok := coding[0].(map[string]interface{}); ok {
				if codeVal, ok := conditionCode["code"].(string); ok {
					return codeVal
				}
			}
		}
	}
	return ""
}

// extractConditionDescription extracts condition description
func extractConditionDescription(resource map[string]interface{}) string {
	if code, ok := resource["code"].(map[string]interface{}); ok {
		if coding, ok := code["coding"].([]interface{}); ok && len(coding) > 0 {
			if conditionCode, ok := coding[0].(map[string]interface{}); ok {
				if display, ok := conditionCode["display"].(string); ok {
					return display
				}
			}
		}
		if text, ok := code["text"].(string); ok {
			return text
		}
	}
	return ""
}

// extractSeverity extracts severity information
func extractSeverity(resource map[string]interface{}) string {
	if severity, ok := resource["severity"].(map[string]interface{}); ok {
		if coding, ok := severity["coding"].([]interface{}); ok && len(coding) > 0 {
			if sev, ok := coding[0].(map[string]interface{}); ok {
				if display, ok := sev["display"].(string); ok {
					return display
				}
			}
		}
		if text, ok := severity["text"].(string); ok {
			return text
		}
	}
	return ""
}

// extractDiagnoser extracts diagnoser information
func extractDiagnoser(resource map[string]interface{}) string {
	if asserter, ok := resource["asserter"].(map[string]interface{}); ok {
		if display, ok := asserter["display"].(string); ok {
			return display
		}
	}
	return ""
}

// extractVitalType extracts vital sign type
func extractVitalType(resource map[string]interface{}) string {
	if code, ok := resource["code"].(map[string]interface{}); ok {
		if coding, ok := code["coding"].([]interface{}); ok && len(coding) > 0 {
			if vitalCode, ok := coding[0].(map[string]interface{}); ok {
				if display, ok := vitalCode["display"].(string); ok {
					return display
				}
			}
		}
		if text, ok := code["text"].(string); ok {
			return text
		}
	}
	return ""
}

// extractVitalValue extracts vital sign value
func extractVitalValue(resource map[string]interface{}) float64 {
	if valueQuantity, ok := resource["valueQuantity"].(map[string]interface{}); ok {
		if value, ok := valueQuantity["value"].(float64); ok {
			return value
		}
	}
	return 0
}

// extractVitalUnit extracts vital sign unit
func extractVitalUnit(resource map[string]interface{}) string {
	if valueQuantity, ok := resource["valueQuantity"].(map[string]interface{}); ok {
		if unit, ok := valueQuantity["unit"].(string); ok {
			return unit
		}
	}
	return ""
}

// extractLabTestName extracts lab test name
func extractLabTestName(resource map[string]interface{}) string {
	return extractVitalType(resource) // Same logic as vital type
}

// extractLabValue extracts lab value
func extractLabValue(resource map[string]interface{}) string {
	if valueQuantity, ok := resource["valueQuantity"].(map[string]interface{}); ok {
		if value, ok := valueQuantity["value"].(float64); ok {
			return fmt.Sprintf("%.2f", value)
		}
	}
	if valueString, ok := resource["valueString"].(string); ok {
		return valueString
	}
	return ""
}

// extractLabUnit extracts lab unit
func extractLabUnit(resource map[string]interface{}) string {
	return extractVitalUnit(resource) // Same logic as vital unit
}

// extractReferenceRange extracts reference range
func extractReferenceRange(resource map[string]interface{}) string {
	if refRanges, ok := resource["referenceRange"].([]interface{}); ok && len(refRanges) > 0 {
		if refRange, ok := refRanges[0].(map[string]interface{}); ok {
			low := ""
			high := ""
			if lowVal, ok := refRange["low"].(map[string]interface{}); ok {
				if val, ok := lowVal["value"].(float64); ok {
					low = fmt.Sprintf("%.2f", val)
				}
			}
			if highVal, ok := refRange["high"].(map[string]interface{}); ok {
				if val, ok := highVal["value"].(float64); ok {
					high = fmt.Sprintf("%.2f", val)
				}
			}
			if low != "" && high != "" {
				return fmt.Sprintf("%s - %s", low, high)
			}
		}
	}
	return ""
}

// extractEncounterType extracts encounter type
func extractEncounterType(resource map[string]interface{}) string {
	if class, ok := resource["class"].(map[string]interface{}); ok {
		if display, ok := class["display"].(string); ok {
			return display
		}
	}
	if types, ok := resource["type"].([]interface{}); ok && len(types) > 0 {
		if encType, ok := types[0].(map[string]interface{}); ok {
			if coding, ok := encType["coding"].([]interface{}); ok && len(coding) > 0 {
				if code, ok := coding[0].(map[string]interface{}); ok {
					if display, ok := code["display"].(string); ok {
						return display
					}
				}
			}
		}
	}
	return ""
}

// extractProvider extracts provider information
func extractProvider(resource map[string]interface{}) string {
	if participants, ok := resource["participant"].([]interface{}); ok {
		for _, participant := range participants {
			if part, ok := participant.(map[string]interface{}); ok {
				if individual, ok := part["individual"].(map[string]interface{}); ok {
					if display, ok := individual["display"].(string); ok {
						return display
					}
				}
			}
		}
	}
	return ""
}

// extractLocation extracts location information
func extractLocation(resource map[string]interface{}) string {
	if locations, ok := resource["location"].([]interface{}); ok && len(locations) > 0 {
		if location, ok := locations[0].(map[string]interface{}); ok {
			if loc, ok := location["location"].(map[string]interface{}); ok {
				if display, ok := loc["display"].(string); ok {
					return display
				}
			}
		}
	}
	return ""
}
