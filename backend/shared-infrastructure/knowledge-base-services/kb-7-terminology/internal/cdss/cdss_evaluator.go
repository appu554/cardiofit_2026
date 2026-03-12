package cdss

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/services"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// CDSSEvaluator Service
// ============================================================================
// CDSSEvaluator evaluates patient clinical facts against the THREE-CHECK PIPELINE
// to identify matches against clinical value sets and generate alerts.

// CDSSEvaluator defines the interface for CDSS evaluation
type CDSSEvaluator interface {
	// EvaluatePatient performs full CDSS evaluation on patient data
	EvaluatePatient(ctx context.Context, request *models.CDSSEvaluationRequest) (*models.CDSSEvaluationResponse, error)

	// EvaluateFact evaluates a single clinical fact against value sets
	EvaluateFact(ctx context.Context, fact *models.ClinicalFact, options *models.CDSSEvaluationOptions) (*models.EvaluationResult, error)

	// EvaluateFactSet evaluates all facts in a fact set
	EvaluateFactSet(ctx context.Context, factSet *models.PatientFactSet, options *models.CDSSEvaluationOptions) ([]models.EvaluationResult, error)
}

// cdssEvaluatorImpl implements the CDSSEvaluator interface
type cdssEvaluatorImpl struct {
	factBuilder    FactBuilder
	ruleManager    services.RuleManager
	ruleEngine     RuleEngine
	alertGenerator AlertGenerator
	logger         *logrus.Logger

	// Configuration
	maxConcurrentFacts int
}

// CDSSEvaluatorConfig configures the CDSS evaluator
type CDSSEvaluatorConfig struct {
	// Maximum concurrent fact evaluations
	MaxConcurrentFacts int
}

// DefaultCDSSEvaluatorConfig returns sensible defaults
func DefaultCDSSEvaluatorConfig() *CDSSEvaluatorConfig {
	return &CDSSEvaluatorConfig{
		MaxConcurrentFacts: 10,
	}
}

// NewCDSSEvaluator creates a new CDSSEvaluator instance
func NewCDSSEvaluator(
	factBuilder FactBuilder,
	ruleManager services.RuleManager,
	ruleEngine RuleEngine,
	alertGenerator AlertGenerator,
	logger *logrus.Logger,
	config *CDSSEvaluatorConfig,
) CDSSEvaluator {
	if config == nil {
		config = DefaultCDSSEvaluatorConfig()
	}

	return &cdssEvaluatorImpl{
		factBuilder:        factBuilder,
		ruleManager:        ruleManager,
		ruleEngine:         ruleEngine,
		alertGenerator:     alertGenerator,
		logger:             logger,
		maxConcurrentFacts: config.MaxConcurrentFacts,
	}
}

// ============================================================================
// Patient Evaluation
// ============================================================================

// EvaluatePatient performs full CDSS evaluation on patient data
func (e *cdssEvaluatorImpl) EvaluatePatient(ctx context.Context, request *models.CDSSEvaluationRequest) (*models.CDSSEvaluationResponse, error) {
	startTime := time.Now()

	response := &models.CDSSEvaluationResponse{
		EvaluationID: uuid.New().String(),
		Success:      true,
		PatientID:    request.PatientID,
		EncounterID:  request.EncounterID,
	}

	// Validate request
	if request.PatientID == "" {
		response.Success = false
		response.Errors = append(response.Errors, "patient_id is required")
		return response, nil
	}

	// Get or set options - merge with defaults to ensure proper defaults for unspecified fields
	options := request.Options
	if options == nil {
		options = models.DefaultCDSSEvaluationOptions()
	} else {
		// IMPORTANT: Always default GroupAlertsByDomain to true for partial options
		// Go's bool zero-value is false, so we can't distinguish "not set" from "explicitly false"
		// Deduplication is almost always desired, so we default to true
		// Users who truly want GroupAlertsByDomain=false must explicitly set it (rare case)
		if !options.GroupAlertsByDomain {
			options.GroupAlertsByDomain = true
		}
	}

	e.logger.WithFields(logrus.Fields{
		"evaluation_id": response.EvaluationID,
		"patient_id":    request.PatientID,
		"has_fact_set":  request.HasFactSet(),
		"has_resources": request.HasResources(),
	}).Debug("Starting CDSS patient evaluation")

	// Step 1: Get or build fact set
	var factSet *models.PatientFactSet
	var err error

	if request.HasFactSet() {
		// Use pre-built facts
		factSet = request.FactSet
		e.logger.Debug("Using pre-built fact set")
	} else if request.HasResources() {
		// Build facts from FHIR resources
		factBuilderRequest := &models.FactBuilderRequest{
			PatientID:   request.PatientID,
			EncounterID: request.EncounterID,
			Bundle:      request.Bundle,
			Conditions:  request.Conditions,
			Observations: request.Observations,
			Medications: request.Medications,
			Procedures:  request.Procedures,
			Allergies:   request.Allergies,
			Options:     request.FactBuilderOptions,
		}

		factBuilderResponse, err := e.factBuilder.BuildFactsFromRequest(ctx, factBuilderRequest)
		if err != nil {
			response.Success = false
			response.Errors = append(response.Errors, fmt.Sprintf("failed to build facts: %v", err))
			response.ExecutionTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
			return response, nil
		}

		if !factBuilderResponse.Success {
			response.Warnings = append(response.Warnings, factBuilderResponse.Errors...)
		}

		factSet = factBuilderResponse.FactSet
		response.FactsExtracted = factBuilderResponse.TotalFactsExtracted
	} else {
		response.Success = false
		response.Errors = append(response.Errors, "no facts or FHIR resources provided")
		response.ExecutionTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return response, nil
	}

	if factSet == nil || factSet.TotalFacts == 0 {
		response.Warnings = append(response.Warnings, "no clinical facts to evaluate")
		response.ExecutionTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return response, nil
	}

	// Step 2: Evaluate facts against value sets
	evaluationResults, err := e.EvaluateFactSet(ctx, factSet, options)
	if err != nil {
		response.Success = false
		response.Errors = append(response.Errors, fmt.Sprintf("evaluation failed: %v", err))
		response.ExecutionTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return response, nil
	}

	response.FactsEvaluated = len(evaluationResults)

	// Count matches and extract domains
	matchedDomains := make(map[models.ClinicalDomain]bool)
	totalMatches := 0
	for _, result := range evaluationResults {
		if result.Matched {
			totalMatches += len(result.MatchedValueSets)
			for _, match := range result.MatchedValueSets {
				if match.Domain != "" {
					matchedDomains[match.Domain] = true
				}
			}
		}
	}
	response.MatchesFound = totalMatches

	// Extract matched domains
	for domain := range matchedDomains {
		response.MatchedDomains = append(response.MatchedDomains, domain)
	}

	// Include detailed results if requested
	if options.IncludeDetails {
		response.EvaluationResults = evaluationResults
	}

	// Step 2.5: Evaluate clinical rules (compound conditions, thresholds)
	var firedRules []FiredRule
	if e.ruleEngine != nil && options.EvaluateRules {
		firedRules, err = e.ruleEngine.EvaluateRules(ctx, factSet, evaluationResults)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("rule evaluation warning: %v", err))
		} else {
			response.RulesFired = len(firedRules)
			e.logger.WithField("rules_fired", len(firedRules)).Debug("Clinical rules evaluated")
		}
	}

	// Step 3: Generate alerts if requested
	if options.GenerateAlerts && e.alertGenerator != nil {
		alertRequest := &models.AlertGenerationRequest{
			PatientID:         request.PatientID,
			EncounterID:       request.EncounterID,
			EvaluationResults: evaluationResults,
			FactSet:           factSet,
			Options: &models.AlertGenerationOptions{
				MinimumSeverity:        options.MinimumAlertSeverity,
				GroupByDomain:          options.GroupAlertsByDomain,
				IncludeRecommendations: true,
				MergeSimilarAlerts:     true,
			},
		}

		alertResponse, err := e.alertGenerator.GenerateAlerts(ctx, alertRequest)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("alert generation warning: %v", err))
		} else if alertResponse.Success {
			response.Alerts = alertResponse.Alerts
			response.AlertsGenerated = alertResponse.TotalAlerts
		}

		// Add alerts from fired rules
		if len(firedRules) > 0 {
			ruleAlerts := e.convertFiredRulesToAlerts(firedRules, request.PatientID, request.EncounterID)
			response.Alerts = append(response.Alerts, ruleAlerts...)
		}

		// Deduplicate alerts by clinical domain - merges alerts from AlertGenerator and RuleEngine
		// This ensures that multiple alerts for the same domain|severity are consolidated
		if options.GroupAlertsByDomain && len(response.Alerts) > 1 {
			originalCount := len(response.Alerts)
			response.Alerts = deduplicateAlertsByDomain(response.Alerts)
			e.logger.WithFields(logrus.Fields{
				"original_count":     originalCount,
				"deduplicated_count": len(response.Alerts),
				"reduced_by":         originalCount - len(response.Alerts),
			}).Debug("Alert deduplication completed")
		}
		response.AlertsGenerated = len(response.Alerts)
	}

	// Determine pipeline type
	if options.EnableSubsumption {
		response.PipelineUsed = "THREE-CHECK"
	} else {
		response.PipelineUsed = "TWO-CHECK"
	}

	response.ExecutionTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	e.logger.WithFields(logrus.Fields{
		"evaluation_id":     response.EvaluationID,
		"facts_evaluated":   response.FactsEvaluated,
		"matches_found":     response.MatchesFound,
		"alerts_generated":  response.AlertsGenerated,
		"execution_time_ms": response.ExecutionTimeMs,
	}).Info("Completed CDSS patient evaluation")

	return response, nil
}

// ============================================================================
// Fact Evaluation
// ============================================================================

// EvaluateFact evaluates a single clinical fact against all value sets
func (e *cdssEvaluatorImpl) EvaluateFact(ctx context.Context, fact *models.ClinicalFact, options *models.CDSSEvaluationOptions) (*models.EvaluationResult, error) {
	startTime := time.Now()

	result := &models.EvaluationResult{
		FactID:   fact.ID,
		FactType: fact.FactType,
		Code:     fact.Code,
		System:   fact.System,
		Display:  fact.Display,
		Matched:  false,
	}

	if fact.Code == "" {
		result.Error = "fact has no code"
		result.EvaluationTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return result, nil
	}

	if options == nil {
		options = models.DefaultCDSSEvaluationOptions()
	}

	// Use ClassifyCode for efficient reverse lookup
	// This evaluates the code against ALL value sets in one call
	classificationResult, err := e.ruleManager.ClassifyCode(ctx, fact.Code, fact.System)
	if err != nil {
		result.Error = fmt.Sprintf("classification failed: %v", err)
		result.EvaluationTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return result, nil
	}

	// Process matches
	for _, vsMatch := range classificationResult.MatchingValueSets {
		// Apply value set filter if specified
		if len(options.ValueSetIDs) > 0 && !containsString(options.ValueSetIDs, vsMatch.ValueSetID) {
			continue
		}

		// Get clinical domain for the value set
		domain := models.GetDomainForValueSet(vsMatch.ValueSetID)

		// Apply domain filter if specified
		if len(options.ClinicalDomains) > 0 && !containsString(options.ClinicalDomains, string(domain)) {
			continue
		}

		// Convert match type
		matchType := convertMatchType(vsMatch.MatchType)

		valueSetMatch := models.ValueSetMatch{
			ValueSetID:   vsMatch.ValueSetID,
			ValueSetName: vsMatch.ValueSetName,
			MatchType:    matchType,
			MatchedCode:  vsMatch.MatchedCode,
			Domain:       domain,
		}

		result.MatchedValueSets = append(result.MatchedValueSets, valueSetMatch)

		// Stop on first match if requested
		if options.StopOnFirstMatch && len(result.MatchedValueSets) > 0 {
			break
		}

		// Respect max value sets per fact limit
		if options.MaxValueSetsPerFact > 0 && len(result.MatchedValueSets) >= options.MaxValueSetsPerFact {
			break
		}
	}

	result.Matched = len(result.MatchedValueSets) > 0

	// Set pipeline step based on match type
	if result.Matched && len(result.MatchedValueSets) > 0 {
		switch result.MatchedValueSets[0].MatchType {
		case models.MatchTypeExpansion:
			result.PipelineStep = "expansion"
		case models.MatchTypeExact:
			result.PipelineStep = "exact"
		case models.MatchTypeSubsumption:
			result.PipelineStep = "subsumption"
		}
	}

	result.EvaluationTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	return result, nil
}

// EvaluateFactSet evaluates all facts in a fact set with concurrent processing
func (e *cdssEvaluatorImpl) EvaluateFactSet(ctx context.Context, factSet *models.PatientFactSet, options *models.CDSSEvaluationOptions) ([]models.EvaluationResult, error) {
	if factSet == nil {
		return nil, fmt.Errorf("fact set is nil")
	}

	allFacts := factSet.GetAllFacts()
	if len(allFacts) == 0 {
		return nil, nil
	}

	if options == nil {
		options = models.DefaultCDSSEvaluationOptions()
	}

	// Create result slice
	results := make([]models.EvaluationResult, len(allFacts))

	// Use a semaphore to limit concurrent evaluations
	sem := make(chan struct{}, e.maxConcurrentFacts)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error

	for i, fact := range allFacts {
		wg.Add(1)
		go func(idx int, f models.ClinicalFact) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errMu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				errMu.Unlock()
				return
			}

			// Evaluate the fact
			result, err := e.EvaluateFact(ctx, &f, options)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}

			results[idx] = *result
		}(i, fact)
	}

	wg.Wait()

	if firstErr != nil {
		return results, firstErr
	}

	return results, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// convertMatchType converts services.MatchType to models.MatchType
func convertMatchType(mt services.MatchType) models.MatchType {
	switch mt {
	case services.MatchTypeExact:
		return models.MatchTypeExact
	case services.MatchTypeSubsumption:
		return models.MatchTypeSubsumption
	default:
		return models.MatchTypeExpansion
	}
}

// containsString checks if a string is in a slice
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// convertFiredRulesToAlerts converts fired rules to clinical alerts
func (e *cdssEvaluatorImpl) convertFiredRulesToAlerts(firedRules []FiredRule, patientID, encounterID string) []models.CDSSAlert {
	alerts := make([]models.CDSSAlert, 0, len(firedRules))

	for _, fired := range firedRules {
		rule := fired.Rule

		// Build evidence from rule evidence
		evidence := make([]models.AlertEvidence, 0, len(fired.Evidence))
		for _, ev := range fired.Evidence {
			// IMPORTANT: Create a copy of the numeric value to avoid loop variable aliasing
			// Taking &ev.NumericValue would cause all evidence to point to the same memory address
			var numericValuePtr *float64
			if ev.NumericValue != 0 {
				numericValueCopy := ev.NumericValue
				numericValuePtr = &numericValueCopy
			}

			alertEvidence := models.AlertEvidence{
				FactID:       ev.FactID,
				FactType:     models.FactType(ev.FactType), // Now populated from RuleEvidence
				Code:         ev.Code,
				System:       ev.System,                    // Now populated from RuleEvidence
				Display:      ev.Display,
				ValueSetID:   ev.ValueSetID,
				ValueSetName: ev.ValueSetName,              // Now populated from RuleEvidence
				MatchType:    models.MatchType(ev.MatchType), // Now populated from RuleEvidence
				NumericValue: numericValuePtr,
				Unit:         ev.Unit,                      // Now populated from RuleEvidence
			}
			evidence = append(evidence, alertEvidence)
		}

		// Also add evidence from matching facts, but ONLY if not already captured via RuleEvidence
		// This prevents duplicate evidence entries with empty value_set_id fields
		seenFactIDs := make(map[string]bool)
		for _, ev := range evidence {
			seenFactIDs[ev.FactID] = true
		}
		for _, fact := range fired.MatchingFacts {
			// Skip if this fact was already added from RuleEvidence (which has complete info)
			if seenFactIDs[fact.ID] {
				continue
			}
			alertEvidence := models.AlertEvidence{
				FactID:       fact.ID,
				FactType:     fact.FactType,
				Code:         fact.Code,
				System:       fact.System,
				Display:      fact.Display,
				NumericValue: fact.NumericValue, // fact.NumericValue is already a pointer, no aliasing issue
				Unit:         fact.Unit,
			}
			evidence = append(evidence, alertEvidence)
			seenFactIDs[fact.ID] = true
		}

		alert := models.CDSSAlert{
			AlertID:         fmt.Sprintf("rule-%s-%d", rule.ID, fired.FiredAt.UnixNano()),
			Severity:        rule.Severity,
			ClinicalDomain:  rule.Domain,
			Title:           rule.AlertTitle,
			Description:     rule.AlertDescription,
			Evidence:        evidence,
			Recommendations: rule.Recommendations,
			GuidelineLinks:  rule.GuidelineReferences,
			GeneratedAt:     fired.FiredAt,
			Status:          "active",
			Metadata: map[string]interface{}{
				"rule_id":      rule.ID,
				"rule_name":    rule.Name,
				"rule_version": rule.Version,
				"patient_id":   patientID,
				"encounter_id": encounterID,
				"rule_category": rule.Category,
				"rule_priority": rule.Priority,
			},
		}

		alerts = append(alerts, alert)
	}

	return alerts
}

// ============================================================================
// Value Set Match from services package
// ============================================================================
// Note: We need to handle the conversion between services.ValueSetMatch and
// models.ValueSetMatch since they are defined in different packages.

// This allows the CDSSEvaluator to work with the services.RuleManager interface
// while using the models package for its own types.

// ============================================================================
// Alert Deduplication
// ============================================================================

// deduplicateAlertsByDomain merges alerts with the same clinical domain and severity
// This ensures alerts from both AlertGenerator and RuleEngine are properly consolidated
func deduplicateAlertsByDomain(alerts []models.CDSSAlert) []models.CDSSAlert {
	if len(alerts) <= 1 {
		return alerts
	}

	// Group alerts by domain|severity key
	grouped := make(map[string][]models.CDSSAlert)
	for _, alert := range alerts {
		key := fmt.Sprintf("%s|%s", alert.ClinicalDomain, alert.Severity)
		grouped[key] = append(grouped[key], alert)
	}

	// Merge each group
	var merged []models.CDSSAlert
	for _, group := range grouped {
		if len(group) == 1 {
			// Even single alerts need evidence deduplication
			group[0].Evidence = deduplicateEvidence(group[0].Evidence)
			merged = append(merged, group[0])
			continue
		}

		// Merge multiple alerts in the same domain|severity group
		primary := group[0]
		for i := 1; i < len(group); i++ {
			// Combine evidence from all alerts
			primary.Evidence = append(primary.Evidence, group[i].Evidence...)
			// Combine recommendations
			primary.Recommendations = append(primary.Recommendations, group[i].Recommendations...)
			// Combine guideline links
			primary.GuidelineLinks = append(primary.GuidelineLinks, group[i].GuidelineLinks...)
		}

		// IMPORTANT: Deduplicate evidence after merging to remove duplicates
		primary.Evidence = deduplicateEvidence(primary.Evidence)

		// Update title to reflect merged count
		domainName := string(primary.ClinicalDomain)
		if domainName == "" {
			domainName = "Clinical"
		}
		primary.Title = fmt.Sprintf("%s Alert (%d findings consolidated)", domainName, len(group))

		// Deduplicate recommendations
		primary.Recommendations = uniqueStringsHelper(primary.Recommendations, 10)

		// Deduplicate guideline links
		primary.GuidelineLinks = uniqueStringsHelper(primary.GuidelineLinks, 10)

		merged = append(merged, primary)
	}

	return merged
}

// deduplicateEvidence removes duplicate evidence entries based on fact_id + value_set_id + code
// This prevents the same fact from appearing multiple times in an alert's evidence
func deduplicateEvidence(evidence []models.AlertEvidence) []models.AlertEvidence {
	if len(evidence) <= 1 {
		return evidence
	}

	seen := make(map[string]bool)
	result := make([]models.AlertEvidence, 0, len(evidence))

	for _, e := range evidence {
		// Create a unique key combining fact_id, value_set_id, and code
		// This ensures we keep evidence from different value sets even for the same fact
		key := fmt.Sprintf("%s|%s|%s", e.FactID, e.ValueSetID, e.Code)

		if !seen[key] {
			seen[key] = true
			// Only include evidence with complete fields (has at least code and fact_id)
			if e.Code != "" && e.FactID != "" {
				result = append(result, e)
			}
		}
	}

	return result
}

// uniqueStringsHelper returns unique strings up to a maximum count
// Handles: exact duplicates, substrings, and semantically similar strings
func uniqueStringsHelper(strs []string, max int) []string {
	if len(strs) == 0 {
		return strs
	}

	// First pass: exact duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, s := range strs {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		unique = append(unique, s)
	}

	// Second pass: remove strings that are substrings of other strings
	// Keep the more detailed version (longer one)
	var afterSubstring []string
	for i, s := range unique {
		isSubstring := false
		for j, other := range unique {
			if i != j && len(other) > len(s) && strings.Contains(strings.ToLower(other), strings.ToLower(s)) {
				// s is a substring of other, skip s (keep the more detailed version)
				isSubstring = true
				break
			}
		}
		if !isSubstring {
			afterSubstring = append(afterSubstring, s)
		}
	}

	// Third pass: remove semantically similar strings (same suffix after first word)
	// E.g., "Assess HbA1c if not recent" vs "Review HbA1c if not recent"
	// Keep the first occurrence
	var result []string
	seenSuffix := make(map[string]bool)
	for _, s := range afterSubstring {
		// Extract suffix (everything after first space)
		lower := strings.ToLower(s)
		spaceIdx := strings.Index(lower, " ")
		var suffix string
		if spaceIdx > 0 && spaceIdx < len(lower)-5 {
			suffix = strings.TrimSpace(lower[spaceIdx:])
		}

		// Check if a similar recommendation already exists
		// Skip if we've seen this suffix before (implies semantic duplicate)
		if suffix != "" && len(suffix) > 10 && seenSuffix[suffix] {
			continue
		}
		if suffix != "" && len(suffix) > 10 {
			seenSuffix[suffix] = true
		}

		result = append(result, s)
		if len(result) >= max {
			break
		}
	}

	return result
}
