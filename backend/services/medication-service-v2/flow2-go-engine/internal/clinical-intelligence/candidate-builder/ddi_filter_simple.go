package candidatebuilder

import (
	"fmt"
	"log"
	"time"
)

// DDIFilter implements drug-drug interaction filtering - Step 3 of the filtering funnel
// Removes candidates with "Contraindicated" severity interactions with active medications
type DDIFilter struct {
	logger *log.Logger
}

// NewDDIFilter creates a new DDI filter
func NewDDIFilter(logger *log.Logger) *DDIFilter {
	return &DDIFilter{
		logger: logger,
	}
}

// FilterByContraindicatedDDIs removes drugs with absolute DDI contraindications
// Only "Contraindicated" severity interactions are filtered here
// Lower severities (Major, Moderate) are passed to ranking phase for scoring
func (df *DDIFilter) FilterByContraindicatedDDIs(
	candidatePool []Drug,
	activeMedications []ActiveMedication,
	ddiRules []DrugInteraction,
) ([]Drug, error) {
	
	startTime := time.Now()
	
	df.logger.Printf("Starting contraindicated DDI filtering: candidates=%d, active_meds=%d, ddi_rules=%d", 
		len(candidatePool), len(activeMedications), len(ddiRules))

	var finalCandidatePool []Drug
	var exclusionLog []ExclusionRecord
	excludedCount := 0

	// If no active medications, skip DDI filtering
	if len(activeMedications) == 0 {
		df.logger.Printf("No active medications - skipping DDI filtering")
		return candidatePool, nil
	}

	// Get list of active medication names for logging
	activeMedNames := df.getActiveMedicationNames(activeMedications)
	df.logger.Printf("Active medications for DDI checking: %v", activeMedNames)

	for _, candidate := range candidatePool {
		isContraindicatedByDDI := false

		// Check candidate against each active medication
		for _, activeMed := range activeMedications {
			// Only check active medications
			if !activeMed.IsActive {
				continue
			}

			interaction := df.findDDIInteraction(candidate, activeMed, ddiRules)
			
			if interaction != nil && interaction.Severity == "Contraindicated" {
				isContraindicatedByDDI = true
				
				// Create detailed exclusion record
				exclusionRecord := ExclusionRecord{
					DrugName:        candidate.Name,
					DrugCode:        candidate.Code,
					ExclusionReason: "contraindicated_ddi",
					FilterStage:     "contraindicated_ddi",
					InteractingDrug: activeMed.Name,
					Severity:        string(interaction.Severity),
					Timestamp:       time.Now(),
					ClinicalReason:  df.generateDDIClinicalReason(candidate, activeMed, interaction),
				}
				exclusionLog = append(exclusionLog, exclusionRecord)
				
				df.logger.Printf("DDI FILTER EXCLUDED: %s (code: %s) due to contraindicated interaction with %s - %s", 
					candidate.Name, candidate.Code, activeMed.Name, interaction.Description)
				break // No need to check other active medications for this candidate
			} else if interaction != nil {
				// Log non-contraindicated interactions for awareness (will be handled in ranking)
				df.logger.Printf("DDI NOTED: %s has %s interaction with %s - will be considered in ranking", 
					candidate.Name, interaction.Severity, activeMed.Name)
			}
		}

		if !isContraindicatedByDDI {
			finalCandidatePool = append(finalCandidatePool, candidate)
			df.logger.Printf("DDI FILTER INCLUDED: %s (code: %s) - no contraindicated interactions found", 
				candidate.Name, candidate.Code)
		} else {
			excludedCount++
		}
	}

	processingTime := time.Since(startTime)
	ddiPassRate := df.calculatePassRate(len(candidatePool), len(finalCandidatePool))

	df.logger.Printf("DDI filtering completed: initial=%d, final=%d, excluded=%d, pass_rate=%.1f%%, contraindicated_interactions=%d, time=%dms", 
		len(candidatePool), len(finalCandidatePool), excludedCount, ddiPassRate, len(exclusionLog), processingTime.Milliseconds())

	return finalCandidatePool, nil
}

// findDDIInteraction finds interaction between candidate drug and active medication
func (df *DDIFilter) findDDIInteraction(candidate Drug, activeMed ActiveMedication, ddiRules []DrugInteraction) *DrugInteraction {
	for _, rule := range ddiRules {
		// Check both directions of interaction
		if (rule.Drug1 == candidate.Code && rule.Drug2 == activeMed.MedicationCode) ||
		   (rule.Drug1 == activeMed.MedicationCode && rule.Drug2 == candidate.Code) {
			
			df.logger.Printf("DDI found by code: %s <-> %s (severity: %s)", 
				candidate.Name, activeMed.Name, rule.Severity)
			return &rule
		}
		
		// Also check by drug names (fallback if codes don't match)
		if (rule.Drug1 == candidate.Name && rule.Drug2 == activeMed.Name) ||
		   (rule.Drug1 == activeMed.Name && rule.Drug2 == candidate.Name) {
			
			df.logger.Printf("DDI found by name: %s <-> %s (severity: %s)", 
				candidate.Name, activeMed.Name, rule.Severity)
			return &rule
		}
	}
	return nil
}

// generateDDIClinicalReason generates clinical reasoning for DDI exclusions
func (df *DDIFilter) generateDDIClinicalReason(candidate Drug, activeMed ActiveMedication, interaction *DrugInteraction) string {
	return fmt.Sprintf(
		"%s contraindicated with %s: %s. Mechanism: %s. Clinical risk: %s severity interaction requiring alternative therapy selection.",
		candidate.Name,
		activeMed.Name,
		interaction.Description,
		interaction.Mechanism,
		interaction.Severity,
	)
}

// getActiveMedicationNames extracts medication names for logging
func (df *DDIFilter) getActiveMedicationNames(medications []ActiveMedication) []string {
	names := make([]string, 0, len(medications))
	for _, med := range medications {
		if med.IsActive {
			names = append(names, med.Name)
		}
	}
	return names
}

// calculatePassRate calculates the percentage of candidates that passed filtering
func (df *DDIFilter) calculatePassRate(initial, passed int) float64 {
	if initial == 0 {
		return 0
	}
	return float64(passed) / float64(initial) * 100
}

// CheckInteraction checks for interaction between two specific drugs
func (df *DDIFilter) CheckInteraction(drug1, drug2 string, ddiRules []DrugInteraction) (*DrugInteraction, error) {
	for _, rule := range ddiRules {
		if (rule.Drug1 == drug1 && rule.Drug2 == drug2) ||
		   (rule.Drug1 == drug2 && rule.Drug2 == drug1) {
			return &rule, nil
		}
	}
	return nil, nil
}

// GetInteractionSeverities returns supported interaction severity levels
func (df *DDIFilter) GetInteractionSeverities() []string {
	return []string{"Contraindicated", "Major", "Moderate", "Minor"}
}
