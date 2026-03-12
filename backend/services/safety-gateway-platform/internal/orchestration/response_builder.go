package orchestration

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// ResponseBuilder aggregates engine results into safety responses
type ResponseBuilder struct {
	logger *logger.Logger
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder(logger *logger.Logger) *ResponseBuilder {
	return &ResponseBuilder{
		logger: logger,
	}
}

// AggregateResults aggregates engine results into a final safety response
func (rb *ResponseBuilder) AggregateResults(
	req *types.SafetyRequest,
	results []types.EngineResult,
	clinicalContext *types.ClinicalContext,
) *types.SafetyResponse {
	response := &types.SafetyResponse{
		RequestID:       req.RequestID,
		EngineResults:   results,
		ContextVersion:  clinicalContext.ContextVersion,
		Timestamp:       time.Now(),
		Metadata:        make(map[string]interface{}),
	}

	// Apply tier-based aggregation rules
	response.Status = rb.determineOverallStatus(results)
	response.RiskScore = rb.calculateRiskScore(results)
	response.CriticalViolations = rb.extractCriticalViolations(results)
	response.Warnings = rb.extractWarnings(results)
	response.EnginesFailed = rb.extractFailedEngines(results)

	// Generate explanation
	response.Explanation = rb.generateExplanation(req, results, response.Status)

	// Generate override token if needed
	if response.Status == types.SafetyStatusUnsafe {
		response.OverrideToken = rb.generateOverrideToken(req, response)
	}

	// Add metadata
	response.Metadata["total_engines"] = len(results)
	response.Metadata["tier1_engines"] = rb.countEnginesByTier(results, types.TierVetoCritical)
	response.Metadata["tier2_engines"] = rb.countEnginesByTier(results, types.TierAdvisory)
	response.Metadata["failed_engines"] = len(response.EnginesFailed)

	rb.logger.Debug("Response aggregated",
		zap.String("request_id", req.RequestID),
		zap.String("final_status", string(response.Status)),
		zap.Float64("risk_score", response.RiskScore),
		zap.Int("critical_violations", len(response.CriticalViolations)),
		zap.Int("warnings", len(response.Warnings)),
	)

	return response
}

// determineOverallStatus determines the overall safety status based on tier-based rules
func (rb *ResponseBuilder) determineOverallStatus(results []types.EngineResult) types.SafetyStatus {
	if len(results) == 0 {
		return types.SafetyStatusManualReview
	}

	// Separate results by tier
	tier1Results := rb.filterResultsByTier(results, types.TierVetoCritical)
	tier2Results := rb.filterResultsByTier(results, types.TierAdvisory)

	// Apply tier-based aggregation rules:
	// ANY Tier 1 engine returns UNSAFE → Final: UNSAFE
	// ANY Tier 1 engine fails/timeouts → Final: UNSAFE (fail closed)
	// ALL Tier 1 engines return SAFE → Proceed to Tier 2 evaluation
	// Tier 2 engine failures → WARNING (degraded check)

	// Check Tier 1 (Veto-Critical) engines first
	if len(tier1Results) > 0 {
		for _, result := range tier1Results {
			if result.Status == types.SafetyStatusUnsafe || result.Error != "" {
				return types.SafetyStatusUnsafe // Fail closed
			}
		}

		// All Tier 1 engines are safe, check if any have warnings
		hasWarnings := false
		for _, result := range tier1Results {
			if result.Status == types.SafetyStatusWarning || len(result.Warnings) > 0 {
				hasWarnings = true
			}
		}

		// If we have Tier 2 engines, evaluate them
		if len(tier2Results) > 0 {
			tier2Status := rb.evaluateTier2Results(tier2Results)
			
			// Combine Tier 1 and Tier 2 results
			if tier2Status == types.SafetyStatusUnsafe {
				return types.SafetyStatusWarning // Tier 2 unsafe becomes warning
			}
			if tier2Status == types.SafetyStatusWarning || hasWarnings {
				return types.SafetyStatusWarning
			}
			return types.SafetyStatusSafe
		}

		// Only Tier 1 engines
		if hasWarnings {
			return types.SafetyStatusWarning
		}
		return types.SafetyStatusSafe
	}

	// Only Tier 2 engines
	if len(tier2Results) > 0 {
		tier2Status := rb.evaluateTier2Results(tier2Results)
		if tier2Status == types.SafetyStatusUnsafe {
			return types.SafetyStatusWarning // Tier 2 unsafe becomes warning
		}
		return tier2Status
	}

	// No results - should not happen, but handle gracefully
	return types.SafetyStatusManualReview
}

// evaluateTier2Results evaluates Tier 2 (Advisory) engine results
func (rb *ResponseBuilder) evaluateTier2Results(results []types.EngineResult) types.SafetyStatus {
	hasUnsafe := false
	hasWarning := false
	hasSafe := false

	for _, result := range results {
		switch result.Status {
		case types.SafetyStatusUnsafe:
			hasUnsafe = true
		case types.SafetyStatusWarning:
			hasWarning = true
		case types.SafetyStatusSafe:
			hasSafe = true
		}
	}

	// For Tier 2, use majority voting with degraded handling
	if hasUnsafe {
		return types.SafetyStatusUnsafe
	}
	if hasWarning {
		return types.SafetyStatusWarning
	}
	if hasSafe {
		return types.SafetyStatusSafe
	}

	return types.SafetyStatusManualReview
}

// calculateRiskScore calculates an overall risk score
func (rb *ResponseBuilder) calculateRiskScore(results []types.EngineResult) float64 {
	if len(results) == 0 {
		return 1.0 // Maximum risk if no engines ran
	}

	var totalScore float64
	var totalWeight float64

	for _, result := range results {
		// Weight Tier 1 engines more heavily
		weight := 1.0
		if result.Tier == types.TierVetoCritical {
			weight = 2.0
		}

		totalScore += result.RiskScore * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 1.0
	}

	return totalScore / totalWeight
}

// extractCriticalViolations extracts critical violations from results
func (rb *ResponseBuilder) extractCriticalViolations(results []types.EngineResult) []string {
	var violations []string
	seen := make(map[string]bool)

	// Prioritize Tier 1 violations
	tier1Results := rb.filterResultsByTier(results, types.TierVetoCritical)
	for _, result := range tier1Results {
		for _, violation := range result.Violations {
			if !seen[violation] {
				violations = append(violations, violation)
				seen[violation] = true
			}
		}
	}

	// Add Tier 2 violations that are marked as critical
	tier2Results := rb.filterResultsByTier(results, types.TierAdvisory)
	for _, result := range tier2Results {
		if result.Status == types.SafetyStatusUnsafe {
			for _, violation := range result.Violations {
				if !seen[violation] {
					violations = append(violations, violation)
					seen[violation] = true
				}
			}
		}
	}

	return violations
}

// extractWarnings extracts warnings from results
func (rb *ResponseBuilder) extractWarnings(results []types.EngineResult) []string {
	var warnings []string
	seen := make(map[string]bool)

	for _, result := range results {
		// Add explicit warnings
		for _, warning := range result.Warnings {
			if !seen[warning] {
				warnings = append(warnings, warning)
				seen[warning] = true
			}
		}

		// Add violations from Tier 2 engines as warnings
		if result.Tier == types.TierAdvisory && result.Status != types.SafetyStatusSafe {
			for _, violation := range result.Violations {
				warningText := fmt.Sprintf("Advisory: %s", violation)
				if !seen[warningText] {
					warnings = append(warnings, warningText)
					seen[warningText] = true
				}
			}
		}
	}

	return warnings
}

// extractFailedEngines extracts engines that failed to execute
func (rb *ResponseBuilder) extractFailedEngines(results []types.EngineResult) []string {
	var failed []string
	
	for _, result := range results {
		if result.Error != "" {
			failed = append(failed, result.EngineID)
		}
	}

	return failed
}

// generateExplanation generates a basic explanation for the decision
func (rb *ResponseBuilder) generateExplanation(
	req *types.SafetyRequest,
	results []types.EngineResult,
	status types.SafetyStatus,
) *types.Explanation {
	explanation := &types.Explanation{
		Level:       types.ExplanationLevelBasic,
		GeneratedAt: time.Now(),
		Details:     []types.ExplanationDetail{},
		Evidence:    []types.Evidence{},
		Actionable:  []types.ActionableGuidance{},
	}

	// Generate summary based on status
	switch status {
	case types.SafetyStatusSafe:
		explanation.Summary = rb.generateSafeSummary(results)
		explanation.Confidence = rb.calculateConfidence(results)
	case types.SafetyStatusUnsafe:
		explanation.Summary = rb.generateUnsafeSummary(results)
		explanation.Confidence = rb.calculateConfidence(results)
		explanation.Actionable = rb.generateUnsafeGuidance(results)
	case types.SafetyStatusWarning:
		explanation.Summary = rb.generateWarningSummary(results)
		explanation.Confidence = rb.calculateConfidence(results)
		explanation.Actionable = rb.generateWarningGuidance(results)
	default:
		explanation.Summary = "Manual review required due to insufficient engine results"
		explanation.Confidence = 0.0
	}

	// Add engine-specific details
	explanation.Details = rb.generateEngineDetails(results)

	return explanation
}

// generateSafeSummary generates a summary for safe decisions
func (rb *ResponseBuilder) generateSafeSummary(results []types.EngineResult) string {
	engineCount := len(results)
	tier1Count := rb.countEnginesByTier(results, types.TierVetoCritical)
	
	return fmt.Sprintf("Safety validation passed. %d engines evaluated (%d critical, %d advisory). No safety concerns identified.",
		engineCount, tier1Count, engineCount-tier1Count)
}

// generateUnsafeSummary generates a summary for unsafe decisions
func (rb *ResponseBuilder) generateUnsafeSummary(results []types.EngineResult) string {
	violations := rb.extractCriticalViolations(results)
	violationCount := len(violations)
	
	if violationCount == 1 {
		return fmt.Sprintf("Safety concern identified: %s. Clinical override required.", violations[0])
	}
	
	return fmt.Sprintf("%d safety concerns identified. Clinical override required for: %s",
		violationCount, strings.Join(violations[:min(3, violationCount)], ", "))
}

// generateWarningSummary generates a summary for warning decisions
func (rb *ResponseBuilder) generateWarningSummary(results []types.EngineResult) string {
	warnings := rb.extractWarnings(results)
	warningCount := len(warnings)
	
	if warningCount == 1 {
		return fmt.Sprintf("Proceed with caution: %s", warnings[0])
	}
	
	return fmt.Sprintf("Proceed with caution. %d advisory warnings identified.", warningCount)
}

// generateEngineDetails generates detailed explanations from engine results
func (rb *ResponseBuilder) generateEngineDetails(results []types.EngineResult) []types.ExplanationDetail {
	var details []types.ExplanationDetail
	
	// Sort results by tier and priority
	sortedResults := make([]types.EngineResult, len(results))
	copy(sortedResults, results)
	sort.Slice(sortedResults, func(i, j int) bool {
		if sortedResults[i].Tier != sortedResults[j].Tier {
			return sortedResults[i].Tier < sortedResults[j].Tier
		}
		return sortedResults[i].RiskScore > sortedResults[j].RiskScore
	})
	
	for _, result := range sortedResults {
		if result.Status != types.SafetyStatusSafe || len(result.Violations) > 0 || len(result.Warnings) > 0 {
			detail := types.ExplanationDetail{
				Category:          rb.getEngineCategory(result.EngineID),
				Severity:          rb.getSeverityFromStatus(result.Status),
				Description:       rb.getEngineDescription(result),
				ClinicalRationale: rb.getClinicalRationale(result),
				Confidence:        result.Confidence,
				EngineSource:      result.EngineName,
			}
			details = append(details, detail)
		}
	}
	
	return details
}

// Helper functions
func (rb *ResponseBuilder) filterResultsByTier(results []types.EngineResult, tier types.CriticalityTier) []types.EngineResult {
	var filtered []types.EngineResult
	for _, result := range results {
		if result.Tier == tier {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func (rb *ResponseBuilder) countEnginesByTier(results []types.EngineResult, tier types.CriticalityTier) int {
	count := 0
	for _, result := range results {
		if result.Tier == tier {
			count++
		}
	}
	return count
}

func (rb *ResponseBuilder) calculateConfidence(results []types.EngineResult) float64 {
	if len(results) == 0 {
		return 0.0
	}
	
	var totalConfidence float64
	for _, result := range results {
		totalConfidence += result.Confidence
	}
	
	return totalConfidence / float64(len(results))
}

func (rb *ResponseBuilder) getEngineCategory(engineID string) string {
	switch {
	case strings.Contains(engineID, "cae"):
		return "Drug Interaction"
	case strings.Contains(engineID, "allergy"):
		return "Allergy Check"
	case strings.Contains(engineID, "protocol"):
		return "Clinical Protocol"
	case strings.Contains(engineID, "constraint"):
		return "Safety Constraints"
	default:
		return "Clinical Safety"
	}
}

func (rb *ResponseBuilder) getSeverityFromStatus(status types.SafetyStatus) string {
	switch status {
	case types.SafetyStatusUnsafe:
		return "high"
	case types.SafetyStatusWarning:
		return "medium"
	case types.SafetyStatusSafe:
		return "low"
	default:
		return "unknown"
	}
}

func (rb *ResponseBuilder) getEngineDescription(result types.EngineResult) string {
	if len(result.Violations) > 0 {
		return result.Violations[0]
	}
	if len(result.Warnings) > 0 {
		return result.Warnings[0]
	}
	return fmt.Sprintf("%s evaluation completed", result.EngineName)
}

func (rb *ResponseBuilder) getClinicalRationale(result types.EngineResult) string {
	// This would be enhanced with actual clinical knowledge
	return fmt.Sprintf("Based on %s analysis with %0.1f%% confidence", 
		result.EngineName, result.Confidence*100)
}

func (rb *ResponseBuilder) generateOverrideToken(req *types.SafetyRequest, response *types.SafetyResponse) *types.OverrideToken {
	// This would be implemented with proper cryptographic signing
	return &types.OverrideToken{
		TokenID:   fmt.Sprintf("override_%s_%d", req.RequestID, time.Now().Unix()),
		RequestID: req.RequestID,
		PatientID: req.PatientID,
		DecisionSummary: &types.DecisionSummary{
			Status:             response.Status,
			CriticalViolations: response.CriticalViolations,
			EnginesFailed:      response.EnginesFailed,
			RiskScore:          response.RiskScore,
			Explanation:        response.Explanation.Summary,
		},
		RequiredLevel: rb.determineRequiredOverrideLevel(response.RiskScore),
		ExpiresAt:     time.Now().Add(5 * time.Minute),
		CreatedAt:     time.Now(),
	}
}

func (rb *ResponseBuilder) determineRequiredOverrideLevel(riskScore float64) types.OverrideLevel {
	switch {
	case riskScore >= 0.9:
		return types.OverrideLevelChief
	case riskScore >= 0.7:
		return types.OverrideLevelPharmacist
	case riskScore >= 0.5:
		return types.OverrideLevelAttending
	default:
		return types.OverrideLevelResident
	}
}

func (rb *ResponseBuilder) generateUnsafeGuidance(results []types.EngineResult) []types.ActionableGuidance {
	return []types.ActionableGuidance{
		{
			Action:   "Review clinical decision with senior clinician",
			Priority: "high",
			Steps:    []string{"Consult attending physician", "Review patient history", "Consider alternative treatments"},
		},
	}
}

func (rb *ResponseBuilder) generateWarningGuidance(results []types.EngineResult) []types.ActionableGuidance {
	return []types.ActionableGuidance{
		{
			Action:   "Monitor patient closely",
			Priority: "medium",
			Steps:    []string{"Increase monitoring frequency", "Document clinical rationale", "Review in 24 hours"},
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
