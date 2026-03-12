package context

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// ContextBuilder builds clinical context from raw data
type ContextBuilder struct {
	logger *logger.Logger
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(logger *logger.Logger) *ContextBuilder {
	return &ContextBuilder{
		logger: logger,
	}
}

// Build builds a clinical context from raw data
func (cb *ContextBuilder) Build(data *ContextData) *types.ClinicalContext {
	startTime := time.Now()

	context := &types.ClinicalContext{
		PatientID:         data.PatientID,
		Demographics:      data.Demographics,
		ActiveMedications: cb.processActiveMedications(data.Medications),
		Allergies:         cb.processAllergies(data.Allergies),
		Conditions:        cb.processConditions(data.Conditions),
		RecentVitals:      cb.processRecentVitals(data.Vitals),
		LabResults:        cb.processLabResults(data.LabResults),
		RecentEncounters:  cb.processRecentEncounters(data.Encounters),
		AssemblyTime:      startTime,
		DataSources:       cb.determineDataSources(data),
		Metadata:          make(map[string]interface{}),
	}

	// Generate context version
	context.ContextVersion = cb.generateContextVersion(context)

	// Add metadata
	context.Metadata["build_duration_ms"] = time.Since(startTime).Milliseconds()
	context.Metadata["medication_count"] = len(context.ActiveMedications)
	context.Metadata["allergy_count"] = len(context.Allergies)
	context.Metadata["condition_count"] = len(context.Conditions)
	context.Metadata["vital_count"] = len(context.RecentVitals)
	context.Metadata["lab_count"] = len(context.LabResults)
	context.Metadata["encounter_count"] = len(context.RecentEncounters)

	// Add clinical insights
	cb.addClinicalInsights(context, data)

	cb.logger.Debug("Clinical context built",
		zap.String("patient_id", data.PatientID),
		zap.String("context_version", context.ContextVersion),
		zap.Int64("build_duration_ms", time.Since(startTime).Milliseconds()),
		zap.Float64("data_completeness", cb.calculateDataCompleteness(context)),
	)

	return context
}

// processActiveMedications processes and filters active medications
func (cb *ContextBuilder) processActiveMedications(medications []types.Medication) []types.Medication {
	if medications == nil {
		return []types.Medication{}
	}

	var active []types.Medication
	now := time.Now()

	for _, med := range medications {
		// Filter for active medications
		if med.Status == "active" || med.Status == "on-hold" {
			// Check if medication is still within active period
			if med.EndDate == nil || med.EndDate.After(now) {
				// Ensure medication has required fields
				if med.ID != "" && med.Name != "" {
					active = append(active, med)
				}
			}
		}
	}

	// Sort by start date (most recent first)
	sort.Slice(active, func(i, j int) bool {
		return active[i].StartDate.After(active[j].StartDate)
	})

	return active
}

// processAllergies processes and validates allergies
func (cb *ContextBuilder) processAllergies(allergies []types.Allergy) []types.Allergy {
	if allergies == nil {
		return []types.Allergy{}
	}

	var processed []types.Allergy

	for _, allergy := range allergies {
		// Filter for active/confirmed allergies
		if allergy.Status == "active" || allergy.Status == "confirmed" {
			// Ensure allergy has required fields
			if allergy.ID != "" && allergy.Allergen != "" {
				processed = append(processed, allergy)
			}
		}
	}

	// Sort by severity (high severity first)
	sort.Slice(processed, func(i, j int) bool {
		return cb.getSeverityWeight(processed[i].Severity) > cb.getSeverityWeight(processed[j].Severity)
	})

	return processed
}

// processConditions processes and validates conditions
func (cb *ContextBuilder) processConditions(conditions []types.Condition) []types.Condition {
	if conditions == nil {
		return []types.Condition{}
	}

	var processed []types.Condition

	for _, condition := range conditions {
		// Filter for active conditions
		if condition.Status == "active" || condition.Status == "confirmed" {
			// Ensure condition has required fields
			if condition.ID != "" && (condition.Code != "" || condition.Display != "") {
				processed = append(processed, condition)
			}
		}
	}

	// Sort by onset date (most recent first)
	sort.Slice(processed, func(i, j int) bool {
		return processed[i].OnsetDate.After(processed[j].OnsetDate)
	})

	return processed
}

// processRecentVitals processes and validates recent vital signs
func (cb *ContextBuilder) processRecentVitals(vitals []types.VitalSign) []types.VitalSign {
	if vitals == nil {
		return []types.VitalSign{}
	}

	var processed []types.VitalSign
	cutoff := time.Now().Add(-24 * time.Hour) // Last 24 hours

	for _, vital := range vitals {
		// Filter for recent vitals
		if vital.Timestamp.After(cutoff) && vital.ID != "" && vital.Type != "" {
			processed = append(processed, vital)
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(processed, func(i, j int) bool {
		return processed[i].Timestamp.After(processed[j].Timestamp)
	})

	// Keep only the most recent vital of each type
	seen := make(map[string]bool)
	var unique []types.VitalSign

	for _, vital := range processed {
		if !seen[vital.Type] {
			unique = append(unique, vital)
			seen[vital.Type] = true
		}
	}

	return unique
}

// processLabResults processes and validates recent lab results
func (cb *ContextBuilder) processLabResults(labResults []types.LabResult) []types.LabResult {
	if labResults == nil {
		return []types.LabResult{}
	}

	var processed []types.LabResult
	cutoff := time.Now().Add(-72 * time.Hour) // Last 72 hours

	for _, lab := range labResults {
		// Filter for recent lab results
		if lab.Timestamp.After(cutoff) && lab.ID != "" && lab.TestName != "" {
			processed = append(processed, lab)
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(processed, func(i, j int) bool {
		return processed[i].Timestamp.After(processed[j].Timestamp)
	})

	return processed
}

// processRecentEncounters processes and validates recent encounters
func (cb *ContextBuilder) processRecentEncounters(encounters []types.Encounter) []types.Encounter {
	if encounters == nil {
		return []types.Encounter{}
	}

	var processed []types.Encounter
	cutoff := time.Now().Add(-30 * 24 * time.Hour) // Last 30 days

	for _, encounter := range encounters {
		// Filter for recent encounters
		if encounter.StartTime.After(cutoff) && encounter.ID != "" {
			processed = append(processed, encounter)
		}
	}

	// Sort by start time (most recent first)
	sort.Slice(processed, func(i, j int) bool {
		return processed[i].StartTime.After(processed[j].StartTime)
	})

	// Limit to most recent 10 encounters
	if len(processed) > 10 {
		processed = processed[:10]
	}

	return processed
}

// determineDataSources determines which data sources were used
func (cb *ContextBuilder) determineDataSources(data *ContextData) []string {
	sources := []string{}

	// Always include FHIR if we have any FHIR data
	if data.Demographics != nil || len(data.Medications) > 0 || 
		len(data.Allergies) > 0 || len(data.Conditions) > 0 ||
		len(data.Vitals) > 0 || len(data.LabResults) > 0 || 
		len(data.Encounters) > 0 {
		sources = append(sources, "fhir")
	}

	// Include GraphDB if we have graph context
	if data.GraphContext != nil && len(data.GraphContext) > 0 {
		sources = append(sources, "graphdb")
	}

	return sources
}

// generateContextVersion generates a version string for the context
func (cb *ContextBuilder) generateContextVersion(context *types.ClinicalContext) string {
	// Create a hash based on key data elements
	hasher := sha256.New()
	
	// Include patient ID
	hasher.Write([]byte(context.PatientID))
	
	// Include assembly time
	hasher.Write([]byte(context.AssemblyTime.Format(time.RFC3339)))
	
	// Include counts of key data elements
	hasher.Write([]byte(fmt.Sprintf("%d-%d-%d-%d-%d-%d",
		len(context.ActiveMedications),
		len(context.Allergies),
		len(context.Conditions),
		len(context.RecentVitals),
		len(context.LabResults),
		len(context.RecentEncounters),
	)))

	hash := hasher.Sum(nil)
	return fmt.Sprintf("v1_%x", hash[:8])
}

// addClinicalInsights adds clinical insights to the context
func (cb *ContextBuilder) addClinicalInsights(context *types.ClinicalContext, data *ContextData) {
	insights := make(map[string]interface{})

	// Calculate risk factors
	insights["risk_factors"] = cb.calculateRiskFactors(context)
	
	// Identify potential drug interactions
	insights["potential_interactions"] = cb.identifyPotentialInteractions(context.ActiveMedications)
	
	// Calculate allergy risk
	insights["allergy_risk_level"] = cb.calculateAllergyRisk(context.Allergies)
	
	// Add clinical complexity score
	insights["complexity_score"] = cb.calculateComplexityScore(context)

	context.Metadata["clinical_insights"] = insights
}

// calculateRiskFactors calculates patient risk factors
func (cb *ContextBuilder) calculateRiskFactors(context *types.ClinicalContext) []string {
	var riskFactors []string

	// Age-based risk
	if context.Demographics != nil {
		if context.Demographics.Age >= 65 {
			riskFactors = append(riskFactors, "elderly")
		}
		if context.Demographics.Age < 18 {
			riskFactors = append(riskFactors, "pediatric")
		}
	}

	// Medication count risk
	if len(context.ActiveMedications) >= 5 {
		riskFactors = append(riskFactors, "polypharmacy")
	}

	// High-severity allergies
	for _, allergy := range context.Allergies {
		if allergy.Severity == "severe" || allergy.Severity == "high" {
			riskFactors = append(riskFactors, "severe_allergies")
			break
		}
	}

	// Chronic conditions
	chronicConditions := []string{"diabetes", "hypertension", "heart_failure", "copd", "ckd"}
	for _, condition := range context.Conditions {
		for _, chronic := range chronicConditions {
			if condition.Code == chronic || condition.Display == chronic {
				riskFactors = append(riskFactors, "chronic_conditions")
				break
			}
		}
	}

	return riskFactors
}

// identifyPotentialInteractions identifies potential drug interactions
func (cb *ContextBuilder) identifyPotentialInteractions(medications []types.Medication) []string {
	// This is a simplified implementation - in production, this would use
	// comprehensive drug interaction databases
	var interactions []string

	// Simple interaction checks (this would be much more comprehensive in production)
	medicationNames := make([]string, len(medications))
	for i, med := range medications {
		medicationNames[i] = med.Name
	}

	// Check for common interaction pairs
	commonInteractions := map[string][]string{
		"warfarin": {"aspirin", "ibuprofen", "naproxen"},
		"digoxin":  {"furosemide", "spironolactone"},
		"lithium":  {"furosemide", "lisinopril"},
	}

	for _, med := range medicationNames {
		if interactsWith, exists := commonInteractions[med]; exists {
			for _, other := range medicationNames {
				for _, interacting := range interactsWith {
					if other == interacting {
						interactions = append(interactions, fmt.Sprintf("%s + %s", med, other))
					}
				}
			}
		}
	}

	return interactions
}

// calculateAllergyRisk calculates overall allergy risk level
func (cb *ContextBuilder) calculateAllergyRisk(allergies []types.Allergy) string {
	if len(allergies) == 0 {
		return "low"
	}

	highSeverityCount := 0
	for _, allergy := range allergies {
		if allergy.Severity == "severe" || allergy.Severity == "high" {
			highSeverityCount++
		}
	}

	if highSeverityCount > 0 {
		return "high"
	}
	if len(allergies) > 3 {
		return "medium"
	}
	return "low"
}

// calculateComplexityScore calculates clinical complexity score
func (cb *ContextBuilder) calculateComplexityScore(context *types.ClinicalContext) float64 {
	score := 0.0

	// Medication complexity
	score += float64(len(context.ActiveMedications)) * 0.5

	// Condition complexity
	score += float64(len(context.Conditions)) * 0.3

	// Allergy complexity
	score += float64(len(context.Allergies)) * 0.2

	// Age factor
	if context.Demographics != nil {
		if context.Demographics.Age >= 65 {
			score += 2.0
		}
		if context.Demographics.Age < 18 {
			score += 1.5
		}
	}

	// Recent encounters (indicates active care)
	score += float64(len(context.RecentEncounters)) * 0.1

	return score
}

// calculateDataCompleteness calculates data completeness percentage
func (cb *ContextBuilder) calculateDataCompleteness(context *types.ClinicalContext) float64 {
	totalFields := 7.0 // Demographics, Medications, Allergies, Conditions, Vitals, Labs, Encounters
	completedFields := 0.0

	if context.Demographics != nil {
		completedFields++
	}
	if len(context.ActiveMedications) > 0 {
		completedFields++
	}
	if len(context.Allergies) > 0 {
		completedFields++
	}
	if len(context.Conditions) > 0 {
		completedFields++
	}
	if len(context.RecentVitals) > 0 {
		completedFields++
	}
	if len(context.LabResults) > 0 {
		completedFields++
	}
	if len(context.RecentEncounters) > 0 {
		completedFields++
	}

	return (completedFields / totalFields) * 100.0
}

// getSeverityWeight returns a numeric weight for severity levels
func (cb *ContextBuilder) getSeverityWeight(severity string) int {
	switch severity {
	case "severe", "high":
		return 3
	case "moderate", "medium":
		return 2
	case "mild", "low":
		return 1
	default:
		return 0
	}
}
