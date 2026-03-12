package cdss

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"kb-7-terminology/internal/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// AlertGenerator Service
// ============================================================================
// AlertGenerator creates clinical decision support alerts from evaluation results.
// It groups matches by clinical domain, determines severity, and generates
// actionable recommendations based on clinical protocols.

// AlertGenerator defines the interface for generating clinical alerts
type AlertGenerator interface {
	// GenerateAlerts creates alerts from evaluation results
	GenerateAlerts(ctx context.Context, request *models.AlertGenerationRequest) (*models.AlertGenerationResponse, error)

	// DetermineSeverity calculates the severity for a clinical indicator
	DetermineSeverity(indicator string, evidence []models.AlertEvidence) models.CDSSAlertSeverity

	// GenerateRecommendations creates recommendations for an alert
	GenerateRecommendations(alert *models.CDSSAlert) []string
}

// alertGeneratorImpl implements the AlertGenerator interface
type alertGeneratorImpl struct {
	logger *logrus.Logger
}

// NewAlertGenerator creates a new AlertGenerator instance
func NewAlertGenerator(logger *logrus.Logger) AlertGenerator {
	return &alertGeneratorImpl{
		logger: logger,
	}
}

// ============================================================================
// Alert Generation
// ============================================================================

// GenerateAlerts creates clinical alerts from evaluation results
func (ag *alertGeneratorImpl) GenerateAlerts(ctx context.Context, request *models.AlertGenerationRequest) (*models.AlertGenerationResponse, error) {
	startTime := time.Now()

	response := &models.AlertGenerationResponse{
		Success:          true,
		AlertsByDomain:   make(map[models.ClinicalDomain]int),
		AlertsBySeverity: make(map[models.CDSSAlertSeverity]int),
	}

	if request == nil {
		response.Success = false
		response.Errors = append(response.Errors, "request is nil")
		return response, nil
	}

	options := request.Options
	if options == nil {
		options = models.DefaultAlertGenerationOptions()
	}

	// Step 1: Group evaluation results by clinical domain
	domainMatches := ag.groupByDomain(request.EvaluationResults)

	ag.logger.WithFields(logrus.Fields{
		"patient_id":     request.PatientID,
		"total_results":  len(request.EvaluationResults),
		"domains_found":  len(domainMatches),
		"group_by_domain": options.GroupByDomain,
	}).Debug("Generating alerts from evaluation results")

	// Step 2: Generate alerts for each domain (or each match if not grouping)
	var alerts []models.CDSSAlert

	if options.GroupByDomain {
		// Generate one alert per domain
		for domain, matches := range domainMatches {
			alert := ag.createDomainAlert(domain, matches, request.FactSet)

			// Apply severity filter
			if alert.Severity.Priority() > options.MinimumSeverity.Priority() {
				continue
			}

			// Add recommendations if requested
			if options.IncludeRecommendations {
				alert.Recommendations = ag.GenerateRecommendations(&alert)
			}

			alerts = append(alerts, alert)

			// Apply max alerts per domain limit
			if options.MaxAlertsPerDomain > 0 {
				if response.AlertsByDomain[domain] >= options.MaxAlertsPerDomain {
					continue
				}
			}

			response.AlertsByDomain[domain]++
			response.AlertsBySeverity[alert.Severity]++
		}
	} else {
		// Generate individual alerts for each match
		for _, result := range request.EvaluationResults {
			if !result.Matched {
				continue
			}

			for _, vsMatch := range result.MatchedValueSets {
				alert := ag.createMatchAlert(&result, vsMatch, request.FactSet)

				// Apply severity filter
				if alert.Severity.Priority() > options.MinimumSeverity.Priority() {
					continue
				}

				// Add recommendations if requested
				if options.IncludeRecommendations {
					alert.Recommendations = ag.GenerateRecommendations(&alert)
				}

				alerts = append(alerts, alert)

				response.AlertsByDomain[alert.ClinicalDomain]++
				response.AlertsBySeverity[alert.Severity]++
			}
		}
	}

	// Merge similar alerts if requested
	if options.MergeSimilarAlerts {
		alerts = ag.mergeSimilarAlerts(alerts)
	}

	// Sort alerts by severity (most severe first)
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Severity.Priority() < alerts[j].Severity.Priority()
	})

	// Set response
	response.Alerts = alerts
	response.TotalAlerts = len(alerts)

	// Count critical and high alerts
	for _, alert := range alerts {
		if alert.Severity == models.SeverityCritical {
			response.CriticalAlerts++
		} else if alert.Severity == models.SeverityHigh {
			response.HighAlerts++
		}
	}

	response.ProcessingTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	ag.logger.WithFields(logrus.Fields{
		"total_alerts":     response.TotalAlerts,
		"critical_alerts":  response.CriticalAlerts,
		"high_alerts":      response.HighAlerts,
		"processing_time_ms": response.ProcessingTimeMs,
	}).Info("Alert generation completed")

	return response, nil
}

// ============================================================================
// Alert Creation
// ============================================================================

// createDomainAlert creates an alert for a clinical domain
func (ag *alertGeneratorImpl) createDomainAlert(domain models.ClinicalDomain, matches []domainMatch, factSet *models.PatientFactSet) models.CDSSAlert {
	alert := models.CDSSAlert{
		AlertID:        uuid.New().String(),
		ClinicalDomain: domain,
		GeneratedAt:    time.Now(),
		Status:         "active",
	}

	// Collect all evidence
	for _, match := range matches {
		evidence := models.AlertEvidence{
			FactID:       match.Result.FactID,
			FactType:     match.Result.FactType,
			Code:         match.Result.Code,
			System:       match.Result.System,
			Display:      match.Result.Display,
			ValueSetID:   match.VSMatch.ValueSetID,
			ValueSetName: match.VSMatch.ValueSetName,
			MatchType:    match.VSMatch.MatchType,
			MatchedCode:  match.VSMatch.MatchedCode,
		}

		// Add numeric value context from fact set if available
		if factSet != nil {
			fact := findFactByID(factSet, match.Result.FactID)
			if fact != nil && fact.NumericValue != nil {
				evidence.NumericValue = fact.NumericValue
				evidence.Unit = fact.Unit
				evidence.ReferenceRangeLow = fact.ReferenceRangeLow
				evidence.ReferenceRangeHigh = fact.ReferenceRangeHigh
				evidence.IsAbnormal = fact.IsAbnormal
			}
		}

		alert.Evidence = append(alert.Evidence, evidence)
	}

	// Determine severity from the highest severity value set
	alert.Severity = ag.determineDomainSeverity(domain, matches)

	// Generate title and description
	alert.Title = ag.generateAlertTitle(domain, alert.Severity, len(matches))
	alert.Description = ag.generateAlertDescription(domain, matches)

	return alert
}

// createMatchAlert creates an alert for a specific value set match
func (ag *alertGeneratorImpl) createMatchAlert(result *models.EvaluationResult, vsMatch models.ValueSetMatch, factSet *models.PatientFactSet) models.CDSSAlert {
	alert := models.CDSSAlert{
		AlertID:        uuid.New().String(),
		ClinicalDomain: vsMatch.Domain,
		Severity:       models.GetValueSetSeverity(vsMatch.ValueSetID),
		GeneratedAt:    time.Now(),
		Status:         "active",
	}

	// Create evidence
	evidence := models.AlertEvidence{
		FactID:       result.FactID,
		FactType:     result.FactType,
		Code:         result.Code,
		System:       result.System,
		Display:      result.Display,
		ValueSetID:   vsMatch.ValueSetID,
		ValueSetName: vsMatch.ValueSetName,
		MatchType:    vsMatch.MatchType,
		MatchedCode:  vsMatch.MatchedCode,
	}

	// Add numeric value context
	if factSet != nil {
		fact := findFactByID(factSet, result.FactID)
		if fact != nil && fact.NumericValue != nil {
			evidence.NumericValue = fact.NumericValue
			evidence.Unit = fact.Unit
			evidence.ReferenceRangeLow = fact.ReferenceRangeLow
			evidence.ReferenceRangeHigh = fact.ReferenceRangeHigh
			evidence.IsAbnormal = fact.IsAbnormal
		}
	}

	alert.Evidence = append(alert.Evidence, evidence)

	// Generate title and description
	alert.Title = fmt.Sprintf("%s Indicator Detected", vsMatch.ValueSetName)
	alert.Description = fmt.Sprintf("Patient has %s (%s) matching %s",
		result.Display, result.Code, vsMatch.ValueSetName)

	return alert
}

// ============================================================================
// Domain Grouping
// ============================================================================

// domainMatch represents a single match within a domain
type domainMatch struct {
	Result  *models.EvaluationResult
	VSMatch models.ValueSetMatch
}

// groupByDomain groups evaluation results by clinical domain
func (ag *alertGeneratorImpl) groupByDomain(results []models.EvaluationResult) map[models.ClinicalDomain][]domainMatch {
	grouped := make(map[models.ClinicalDomain][]domainMatch)

	for i := range results {
		result := &results[i]
		if !result.Matched {
			continue
		}

		for _, vsMatch := range result.MatchedValueSets {
			domain := vsMatch.Domain
			if domain == "" {
				domain = models.DomainGeneral
			}

			grouped[domain] = append(grouped[domain], domainMatch{
				Result:  result,
				VSMatch: vsMatch,
			})
		}
	}

	return grouped
}

// ============================================================================
// Severity Determination
// ============================================================================

// DetermineSeverity calculates the severity for a clinical indicator
func (ag *alertGeneratorImpl) DetermineSeverity(indicator string, evidence []models.AlertEvidence) models.CDSSAlertSeverity {
	// Check if there's a clinical indicator definition
	if ind := models.GetClinicalIndicator(indicator); ind != nil {
		return ind.Severity
	}

	// Fall back to the highest severity from evidence value sets
	highestSeverity := models.SeverityLow
	for _, e := range evidence {
		severity := models.GetValueSetSeverity(e.ValueSetID)
		if severity.Priority() < highestSeverity.Priority() {
			highestSeverity = severity
		}
	}

	return highestSeverity
}

// determineDomainSeverity finds the highest severity for a domain's matches
func (ag *alertGeneratorImpl) determineDomainSeverity(domain models.ClinicalDomain, matches []domainMatch) models.CDSSAlertSeverity {
	highestSeverity := models.SeverityLow

	for _, match := range matches {
		severity := models.GetValueSetSeverity(match.VSMatch.ValueSetID)
		if severity.Priority() < highestSeverity.Priority() {
			highestSeverity = severity
		}

		// If we find critical, no need to check more
		if highestSeverity == models.SeverityCritical {
			break
		}
	}

	return highestSeverity
}

// ============================================================================
// Recommendation Generation
// ============================================================================

// GenerateRecommendations creates recommendations for an alert
func (ag *alertGeneratorImpl) GenerateRecommendations(alert *models.CDSSAlert) []string {
	var recommendations []string

	// Look for matching clinical indicator
	indicatorID := domainToIndicatorID(alert.ClinicalDomain)
	if indicator := models.GetClinicalIndicator(indicatorID); indicator != nil {
		recommendations = append(recommendations, indicator.Recommendations...)
	}

	// Add severity-based recommendations
	switch alert.Severity {
	case models.SeverityCritical:
		recommendations = append([]string{"IMMEDIATE ACTION REQUIRED"}, recommendations...)
	case models.SeverityHigh:
		if len(recommendations) == 0 {
			recommendations = append(recommendations, "Urgent clinical review recommended")
		}
	}

	// Add evidence-based recommendations for specific findings
	for _, evidence := range alert.Evidence {
		if recs := getEvidenceBasedRecommendations(evidence); len(recs) > 0 {
			recommendations = append(recommendations, recs...)
		}
	}

	// Remove duplicates and limit count
	return uniqueStrings(recommendations, 10)
}

// ============================================================================
// Alert Merging
// ============================================================================

// mergeSimilarAlerts combines alerts with the same domain and severity
func (ag *alertGeneratorImpl) mergeSimilarAlerts(alerts []models.CDSSAlert) []models.CDSSAlert {
	// Group by domain + severity key
	grouped := make(map[string][]models.CDSSAlert)

	for _, alert := range alerts {
		key := fmt.Sprintf("%s|%s", alert.ClinicalDomain, alert.Severity)
		grouped[key] = append(grouped[key], alert)
	}

	// Merge each group
	var merged []models.CDSSAlert
	for _, group := range grouped {
		if len(group) == 1 {
			merged = append(merged, group[0])
			continue
		}

		// Merge multiple alerts in the group
		primary := group[0]
		for i := 1; i < len(group); i++ {
			// Combine evidence
			primary.Evidence = append(primary.Evidence, group[i].Evidence...)
			// Combine recommendations
			primary.Recommendations = append(primary.Recommendations, group[i].Recommendations...)
		}

		// Update title to reflect merged count
		primary.Title = fmt.Sprintf("%s (%d findings)",
			ag.domainDisplayName(primary.ClinicalDomain), len(group))

		// Update description
		primary.Description = ag.generateMergedDescription(primary.ClinicalDomain, primary.Evidence)

		// Deduplicate recommendations
		primary.Recommendations = uniqueStrings(primary.Recommendations, 10)

		merged = append(merged, primary)
	}

	return merged
}

// ============================================================================
// Title and Description Generation
// ============================================================================

// generateAlertTitle creates a title for a domain-based alert
func (ag *alertGeneratorImpl) generateAlertTitle(domain models.ClinicalDomain, severity models.CDSSAlertSeverity, matchCount int) string {
	domainName := ag.domainDisplayName(domain)

	switch severity {
	case models.SeverityCritical:
		return fmt.Sprintf("⚠️ CRITICAL: %s Alert", domainName)
	case models.SeverityHigh:
		return fmt.Sprintf("🔴 %s Alert - Urgent", domainName)
	case models.SeverityModerate:
		return fmt.Sprintf("🟡 %s Indicator", domainName)
	default:
		return fmt.Sprintf("ℹ️ %s Finding", domainName)
	}
}

// generateAlertDescription creates a description for a domain-based alert
func (ag *alertGeneratorImpl) generateAlertDescription(domain models.ClinicalDomain, matches []domainMatch) string {
	if len(matches) == 0 {
		return ""
	}

	// Collect unique codes
	var codes []string
	seenCodes := make(map[string]bool)
	for _, match := range matches {
		if !seenCodes[match.Result.Code] {
			seenCodes[match.Result.Code] = true
			display := match.Result.Display
			if display == "" {
				display = match.Result.Code
			}
			codes = append(codes, display)
		}
	}

	domainName := ag.domainDisplayName(domain)

	if len(codes) == 1 {
		return fmt.Sprintf("Patient has %s indicators: %s", domainName, codes[0])
	}

	if len(codes) <= 3 {
		return fmt.Sprintf("Patient has multiple %s indicators: %s", domainName, strings.Join(codes, ", "))
	}

	return fmt.Sprintf("Patient has %d %s indicators including: %s, and %d more",
		len(codes), domainName, strings.Join(codes[:3], ", "), len(codes)-3)
}

// generateMergedDescription creates a description for merged alerts
func (ag *alertGeneratorImpl) generateMergedDescription(domain models.ClinicalDomain, evidence []models.AlertEvidence) string {
	var displays []string
	seen := make(map[string]bool)

	for _, e := range evidence {
		if !seen[e.Code] {
			seen[e.Code] = true
			display := e.Display
			if display == "" {
				display = e.Code
			}
			displays = append(displays, display)
		}
	}

	domainName := ag.domainDisplayName(domain)
	return fmt.Sprintf("Multiple %s findings detected: %s", domainName, strings.Join(displays, ", "))
}

// domainDisplayName returns a human-readable name for a clinical domain
func (ag *alertGeneratorImpl) domainDisplayName(domain models.ClinicalDomain) string {
	switch domain {
	case models.DomainSepsis:
		return "Sepsis"
	case models.DomainRenal:
		return "Renal"
	case models.DomainCardiac:
		return "Cardiac"
	case models.DomainRespiratory:
		return "Respiratory"
	case models.DomainMetabolic:
		return "Metabolic"
	case models.DomainNeurological:
		return "Neurological"
	case models.DomainHematologic:
		return "Hematologic"
	case models.DomainInfectious:
		return "Infectious Disease"
	case models.DomainEndocrine:
		return "Endocrine"
	case models.DomainGI:
		return "Gastrointestinal"
	case models.DomainMSK:
		return "Musculoskeletal"
	case models.DomainDermatologic:
		return "Dermatologic"
	case models.DomainOncologic:
		return "Oncologic"
	case models.DomainPsychiatric:
		return "Psychiatric"
	default:
		return "Clinical"
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// findFactByID finds a fact in a fact set by ID
func findFactByID(factSet *models.PatientFactSet, factID string) *models.ClinicalFact {
	if factSet == nil {
		return nil
	}

	for i := range factSet.Conditions {
		if factSet.Conditions[i].ID == factID {
			return &factSet.Conditions[i]
		}
	}
	for i := range factSet.Observations {
		if factSet.Observations[i].ID == factID {
			return &factSet.Observations[i]
		}
	}
	for i := range factSet.Medications {
		if factSet.Medications[i].ID == factID {
			return &factSet.Medications[i]
		}
	}
	for i := range factSet.Procedures {
		if factSet.Procedures[i].ID == factID {
			return &factSet.Procedures[i]
		}
	}
	for i := range factSet.Allergies {
		if factSet.Allergies[i].ID == factID {
			return &factSet.Allergies[i]
		}
	}

	return nil
}

// domainToIndicatorID maps a clinical domain to its indicator ID
func domainToIndicatorID(domain models.ClinicalDomain) string {
	switch domain {
	case models.DomainSepsis:
		return "sepsis"
	case models.DomainRenal:
		return "aki"
	case models.DomainCardiac:
		return "heart_failure"
	case models.DomainMetabolic:
		return "diabetes"
	case models.DomainRespiratory:
		return "respiratory_failure"
	default:
		return string(domain)
	}
}

// getEvidenceBasedRecommendations returns recommendations based on specific evidence
func getEvidenceBasedRecommendations(evidence models.AlertEvidence) []string {
	var recs []string

	// Add recommendations based on abnormal lab values
	if evidence.IsAbnormal && evidence.NumericValue != nil {
		// This would be expanded with more specific clinical logic
		if evidence.ReferenceRangeHigh != nil && *evidence.NumericValue > *evidence.ReferenceRangeHigh {
			recs = append(recs, fmt.Sprintf("Elevated %s - review clinical significance", evidence.Display))
		} else if evidence.ReferenceRangeLow != nil && *evidence.NumericValue < *evidence.ReferenceRangeLow {
			recs = append(recs, fmt.Sprintf("Low %s - assess clinical impact", evidence.Display))
		}
	}

	return recs
}

// uniqueStrings removes duplicates from a string slice and limits to maxCount
func uniqueStrings(input []string, maxCount int) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
			if maxCount > 0 && len(result) >= maxCount {
				break
			}
		}
	}

	return result
}
