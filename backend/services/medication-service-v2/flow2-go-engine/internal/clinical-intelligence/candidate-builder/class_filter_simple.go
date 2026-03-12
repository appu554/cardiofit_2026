package candidatebuilder

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// ClassFilter implements therapeutic class filtering - Step 1 of the filtering funnel
type ClassFilter struct {
	logger *log.Logger
}

// NewClassFilter creates a new class filter
func NewClassFilter(logger *log.Logger) *ClassFilter {
	return &ClassFilter{
		logger: logger,
	}
}

// FilterByRecommendedClass filters drugs by therapeutic class
// This is the first filter in the funnel - narrows from all drugs to clinically indicated classes
func (cf *ClassFilter) FilterByRecommendedClass(
	initialPool []Drug,
	recommendedClasses []string,
) ([]Drug, error) {
	
	startTime := time.Now()
	
	cf.logger.Printf("Starting therapeutic class filtering: initial_pool=%d, recommended_classes=%v", 
		len(initialPool), recommendedClasses)

	var candidatePool []Drug

	// If no specific classes recommended, keep all drugs (broad search)
	if len(recommendedClasses) == 0 {
		cf.logger.Printf("No specific drug classes recommended - keeping all drugs for broad search")
		return initialPool, nil
	}

	// Filter by therapeutic class
	includedCount := 0
	excludedCount := 0
	
	for _, drug := range initialPool {
		if cf.drugClassIsRecommended(drug, recommendedClasses) {
			candidatePool = append(candidatePool, drug)
			includedCount++
			
			therapeuticClass := cf.getTherapeuticClass(drug)
			cf.logger.Printf("INCLUDED: %s (class: %s) - matches recommended class",
				drug.Name, therapeuticClass)

		} else {
			excludedCount++
			therapeuticClass := cf.getTherapeuticClass(drug)
			cf.logger.Printf("EXCLUDED: %s (class: %s) - not in recommended classes %v",
				drug.Name, therapeuticClass, recommendedClasses)
		}
	}

	processingTime := time.Since(startTime)
	reductionPercent := cf.calculateReductionPercent(len(initialPool), len(candidatePool))

	cf.logger.Printf("Class filtering completed: initial=%d, filtered=%d, included=%d, excluded=%d, reduction=%.1f%%, time=%dms", 
		len(initialPool), len(candidatePool), includedCount, excludedCount, reductionPercent, processingTime.Milliseconds())

	// Validate results
	if len(candidatePool) == 0 {
		cf.logger.Printf("WARNING: Class filtering resulted in zero candidates - no drugs match recommended classes: %v", 
			recommendedClasses)
		
		return candidatePool, &FilterError{
			Stage:   "class_filtering",
			Message: fmt.Sprintf("no drugs found matching recommended classes: %s", strings.Join(recommendedClasses, ", ")),
		}
	}

	return candidatePool, nil
}

// getTherapeuticClass gets the primary therapeutic class for a drug
func (cf *ClassFilter) getTherapeuticClass(drug Drug) string {
	if len(drug.TherapeuticClasses) > 0 {
		return drug.TherapeuticClasses[0]
	}
	return ""
}

// drugClassIsRecommended checks if a drug's therapeutic class is in the recommended list
func (cf *ClassFilter) drugClassIsRecommended(drug Drug, recommendedClasses []string) bool {
	// Check primary therapeutic class
	primaryClass := cf.getTherapeuticClass(drug)
	for _, recommendedClass := range recommendedClasses {
		if primaryClass == recommendedClass {
			cf.logger.Printf("Drug %s matched by primary class: %s", drug.Name, recommendedClass)
			return true
		}
	}

	// Check all therapeutic classes
	for _, drugClass := range drug.TherapeuticClasses {
		for _, recommendedClass := range recommendedClasses {
			if drugClass == recommendedClass {
				cf.logger.Printf("Drug %s matched by therapeutic class: %s", drug.Name, recommendedClass)
				return true
			}
		}
	}

	// Check sub-classes for more specific matching
	for _, subClass := range drug.SubClasses {
		for _, recommendedClass := range recommendedClasses {
			if subClass == recommendedClass {
				cf.logger.Printf("Drug %s matched by sub-class: %s", drug.Name, recommendedClass)
				return true
			}
		}
	}

	// Check for partial matches (e.g., "ACE" matches "ACE_INHIBITOR")
	for _, recommendedClass := range recommendedClasses {
		if cf.isPartialClassMatch(primaryClass, recommendedClass) {
			cf.logger.Printf("Drug %s matched by partial match: %s ~ %s",
				drug.Name, primaryClass, recommendedClass)
			return true
		}
	}

	return false
}

// isPartialClassMatch checks for partial therapeutic class matches
func (cf *ClassFilter) isPartialClassMatch(drugClass, recommendedClass string) bool {
	// Convert to uppercase for case-insensitive comparison
	drugClassUpper := strings.ToUpper(drugClass)
	recommendedUpper := strings.ToUpper(recommendedClass)

	// Check if either contains the other
	if strings.Contains(drugClassUpper, recommendedUpper) || strings.Contains(recommendedUpper, drugClassUpper) {
		return true
	}

	// Check for common abbreviations and synonyms
	classAliases := map[string][]string{
		"ACE_INHIBITOR": {"ACE", "ACEI", "ACE_I"},
		"ARB":          {"ANGIOTENSIN_RECEPTOR_BLOCKER", "AT1_BLOCKER"},
		"BETA_BLOCKER": {"BB", "BETA", "B_BLOCKER"},
		"CCB":          {"CALCIUM_CHANNEL_BLOCKER", "CA_CHANNEL_BLOCKER"},
	}

	// Check aliases
	for canonical, aliases := range classAliases {
		if drugClassUpper == canonical || recommendedUpper == canonical {
			for _, alias := range aliases {
				if drugClassUpper == alias || recommendedUpper == alias {
					return true
				}
			}
		}
	}

	return false
}

// calculateReductionPercent calculates percentage reduction
func (cf *ClassFilter) calculateReductionPercent(before, after int) float64 {
	if before == 0 {
		return 0
	}
	return float64(before-after) / float64(before) * 100
}

// GetSupportedClasses returns list of supported therapeutic classes
func (cf *ClassFilter) GetSupportedClasses() []string {
	return []string{
		"ACE_INHIBITOR",
		"ARB", 
		"THIAZIDE_DIURETIC",
		"BETA_BLOCKER",
		"CALCIUM_CHANNEL_BLOCKER",
		"ANTIBIOTIC",
		"ANTIDIABETIC",
		"ANTICOAGULANT",
		"ANTIPLATELET",
		"STATIN",
		"NSAID",
		"PROTON_PUMP_INHIBITOR",
		"H2_RECEPTOR_ANTAGONIST",
		"BRONCHODILATOR",
		"CORTICOSTEROID",
		"ANTIDEPRESSANT",
		"ANTIPSYCHOTIC",
		"ANTICONVULSANT",
		"OPIOID_ANALGESIC",
		"MUSCLE_RELAXANT",
	}
}

// ValidateClass checks if a therapeutic class is supported
func (cf *ClassFilter) ValidateClass(class string) bool {
	supportedClasses := cf.GetSupportedClasses()
	for _, supported := range supportedClasses {
		if class == supported {
			return true
		}
	}
	return false
}
