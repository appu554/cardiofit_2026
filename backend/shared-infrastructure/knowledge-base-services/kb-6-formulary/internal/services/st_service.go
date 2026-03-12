// Package services provides business logic for KB-6 Formulary Service.
package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"kb-formulary/internal/models"
	"kb-formulary/internal/repository"
)

// STService handles Step Therapy business logic
type STService struct {
	repo         *repository.STRepository
	eventEmitter *EventEmitter
}

// SetEventEmitter sets the event emitter for cross-service signaling (Enhancement #2)
func (s *STService) SetEventEmitter(emitter *EventEmitter) {
	s.eventEmitter = emitter
}

// NewSTService creates a new STService instance
func NewSTService(repo *repository.STRepository) *STService {
	return &STService{repo: repo}
}

// GetRequirements retrieves ST requirements for a drug
func (s *STService) GetRequirements(ctx context.Context, req *models.STRequirementsRequest) (*models.STRequirementsResponse, error) {
	rule, err := s.repo.GetRules(ctx, req.DrugRxNorm, req.PayerID, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ST requirements: %w", err)
	}

	// No ST required for this drug
	if rule == nil {
		return &models.STRequirementsResponse{
			STRequired: false,
			DrugRxNorm: req.DrugRxNorm,
		}, nil
	}

	response := &models.STRequirementsResponse{
		STRequired:       true,
		DrugRxNorm:       rule.TargetDrugRxNorm,
		DrugName:         rule.TargetDrugName,
		Steps:            rule.Steps,
		TotalSteps:       len(rule.Steps),
		OverrideCriteria: rule.OverrideCriteria,
		ExceptionCodes:   rule.ExceptionDiagnosisCodes,
	}

	if rule.ProtocolName != nil {
		response.ProtocolName = *rule.ProtocolName
	}
	if rule.EvidenceLevel != nil {
		response.EvidenceLevel = *rule.EvidenceLevel
	}

	return response, nil
}

// CheckStepTherapy evaluates if ST is required and validates drug history
func (s *STService) CheckStepTherapy(ctx context.Context, req *models.STCheckRequest) (*models.STCheckResponse, error) {
	// Check for active override
	override, err := s.repo.GetActiveOverride(ctx, req.PatientID, req.DrugRxNorm)
	if err != nil {
		return nil, fmt.Errorf("failed to check override: %w", err)
	}

	if override != nil {
		return &models.STCheckResponse{
			StepTherapyRequired: true,
			Approved:            true,
			Message:             fmt.Sprintf("Active override exists (reason: %s)", override.OverrideReason),
		}, nil
	}

	// Get ST rules
	rule, err := s.repo.GetRules(ctx, req.DrugRxNorm, req.PayerID, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ST rules: %w", err)
	}

	// No ST required
	if rule == nil {
		return &models.STCheckResponse{
			StepTherapyRequired: false,
			Approved:            true,
			Message:             "Step therapy is not required for this drug",
		}, nil
	}

	// Check for exception diagnosis codes
	if s.checkExceptionApplies(rule.ExceptionDiagnosisCodes, req.Diagnoses) {
		return &models.STCheckResponse{
			StepTherapyRequired: true,
			Approved:            true,
			ExceptionApplies:    true,
			TotalSteps:          len(rule.Steps),
			Message:             "Step therapy waived due to qualifying diagnosis",
		}, nil
	}

	// Evaluate each step
	evaluations := s.evaluateSteps(ctx, rule.Steps, req.DrugHistory)
	stepsSatisfied := []int{}
	currentStep := 1

	for i, eval := range evaluations {
		if eval.Satisfied {
			stepsSatisfied = append(stepsSatisfied, eval.Step.StepNumber)
			currentStep = i + 2 // Next step
		} else {
			// First unsatisfied step is the current step
			currentStep = i + 1
			break
		}
	}

	allSatisfied := len(stepsSatisfied) == len(rule.Steps)

	response := &models.STCheckResponse{
		StepTherapyRequired: true,
		Approved:            allSatisfied,
		TotalSteps:          len(rule.Steps),
		CurrentStep:         currentStep,
		StepsSatisfied:      stepsSatisfied,
		StepEvaluations:     evaluations,
		OverrideAvailable:   len(rule.OverrideCriteria) > 0,
		OverrideCriteria:    rule.OverrideCriteria,
	}

	if allSatisfied {
		response.Message = "All step therapy requirements have been met"
	} else {
		// Find next required step
		for i, eval := range evaluations {
			if !eval.Satisfied {
				response.NextRequiredStep = &rule.Steps[i]
				break
			}
		}
		response.Message = fmt.Sprintf("Step %d of %d not yet satisfied. %s required.",
			currentStep, len(rule.Steps), response.NextRequiredStep.Description)
	}

	// Save the check for audit
	check := &models.StepTherapyCheck{
		PatientID:           req.PatientID,
		TargetDrugRxNorm:    req.DrugRxNorm,
		TargetDrugName:      rule.TargetDrugName,
		PayerID:             req.PayerID,
		PlanID:              req.PlanID,
		DrugHistory:         req.DrugHistory,
		StepTherapyRequired: true,
		TotalSteps:          &response.TotalSteps,
		StepsSatisfied:      stepsSatisfied,
		CurrentStep:         &currentStep,
		Approved:            allSatisfied,
		Message:             response.Message,
		NextRequiredStep:    response.NextRequiredStep,
		RuleID:              &rule.ID,
	}
	s.repo.SaveCheck(ctx, check) // Log but don't fail on error

	return response, nil
}

// RequestOverride creates a step therapy override request
func (s *STService) RequestOverride(ctx context.Context, req *models.STOverrideRequest) (*models.STOverrideResponse, error) {
	// Validate required fields
	if req.PatientID == "" || req.ProviderID == "" || req.DrugRxNorm == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	// Verify the override reason is valid
	if !s.isValidOverrideReason(req.OverrideReason) {
		return nil, fmt.Errorf("invalid override reason: %s", req.OverrideReason)
	}

	// Check if ST is actually required for this drug
	rule, err := s.repo.GetRules(ctx, req.DrugRxNorm, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get ST rules: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("step therapy not required for drug %s", req.DrugRxNorm)
	}

	// Check if the override reason is allowed for this rule
	if !s.isOverrideAllowed(rule.OverrideCriteria, req.OverrideReason) {
		return nil, fmt.Errorf("override reason '%s' not allowed for this drug", req.OverrideReason)
	}

	// Create override request
	override := &models.StepTherapyOverride{
		CheckID:               req.CheckID,
		PatientID:             req.PatientID,
		ProviderID:            req.ProviderID,
		TargetDrugRxNorm:      req.DrugRxNorm,
		OverrideReason:        req.OverrideReason,
		ClinicalJustification: req.ClinicalJustification,
		SupportingDocumentation: req.SupportingDocuments,
		SubmittedBy:           &req.ProviderID,
	}

	if err := s.repo.CreateOverride(ctx, override); err != nil {
		return nil, fmt.Errorf("failed to create override: %w", err)
	}

	// Auto-approve certain override types (e.g., documented contraindication)
	if s.shouldAutoApprove(req.OverrideReason) {
		expiresAt := time.Now().AddDate(1, 0, 0) // 1 year
		override.Status = models.STOverrideApproved
		override.ExpiresAt = &expiresAt
		s.repo.UpdateOverrideStatus(ctx, override.ID, models.STOverrideApproved,
			"Auto-approved based on override type", "SYSTEM")
	}

	return &models.STOverrideResponse{
		Override: *override,
		Message:  s.getOverrideStatusMessage(override.Status),
	}, nil
}

// GetOverrideStatus retrieves the status of an override request
func (s *STService) GetOverrideStatus(ctx context.Context, overrideID string) (*models.STOverrideResponse, error) {
	id, err := uuid.Parse(overrideID)
	if err != nil {
		return nil, fmt.Errorf("invalid override ID format: %w", err)
	}

	override, err := s.repo.GetOverride(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get override: %w", err)
	}

	if override == nil {
		return nil, fmt.Errorf("override not found: %s", overrideID)
	}

	return &models.STOverrideResponse{
		Override: *override,
		Message:  s.getOverrideStatusMessage(override.Status),
	}, nil
}

// =============================================================================
// STEP EVALUATION ENGINE
// =============================================================================

// evaluateSteps evaluates all steps against drug history
func (s *STService) evaluateSteps(ctx context.Context, steps []models.Step, history []models.DrugHistory) []models.StepEvaluation {
	evaluations := make([]models.StepEvaluation, len(steps))

	for i, step := range steps {
		evaluations[i] = s.evaluateSingleStep(ctx, step, history)
	}

	return evaluations
}

// evaluateSingleStep evaluates a single step against drug history
func (s *STService) evaluateSingleStep(ctx context.Context, step models.Step, history []models.DrugHistory) models.StepEvaluation {
	eval := models.StepEvaluation{
		Step:          step,
		Satisfied:     false,
		DurationMet:   false,
		MatchingDrugs: []models.DrugHistory{},
	}

	if len(history) == 0 {
		eval.Notes = "No medication history provided"
		return eval
	}

	// Find all drugs that satisfy this step
	for _, drug := range history {
		if s.drugMatchesStep(drug, step) {
			eval.MatchingDrugs = append(eval.MatchingDrugs, drug)
		}
	}

	if len(eval.MatchingDrugs) == 0 {
		eval.Notes = fmt.Sprintf("No history of %s found", step.Description)
		return eval
	}

	// Calculate total duration across all matching drugs
	totalDuration := 0
	for _, drug := range eval.MatchingDrugs {
		duration := s.calculateDrugDuration(drug)
		totalDuration += duration
	}

	eval.ActualDuration = totalDuration

	// Check if minimum duration is met
	if step.MinDurationDays > 0 {
		if totalDuration >= step.MinDurationDays {
			eval.DurationMet = true
			eval.Satisfied = true
			eval.Notes = fmt.Sprintf("%s therapy documented for %d days (required: %d days)",
				step.Description, totalDuration, step.MinDurationDays)
		} else {
			eval.Notes = fmt.Sprintf("%s therapy only %d days (required: %d days)",
				step.Description, totalDuration, step.MinDurationDays)
		}
	} else {
		// No duration requirement, just need evidence of use
		eval.DurationMet = true
		eval.Satisfied = true
		eval.Notes = fmt.Sprintf("%s therapy documented", step.Description)
	}

	return eval
}

// drugMatchesStep checks if a drug from history matches a step requirement
func (s *STService) drugMatchesStep(drug models.DrugHistory, step models.Step) bool {
	// Check RxNorm codes
	for _, reqCode := range step.RxNormCodes {
		if drug.RxNormCode == reqCode {
			return true
		}
	}

	// If allow_any_in_class is true and drug class matches
	if step.AllowAnyInClass && step.DrugClass != "" {
		// In production, this would query a drug classification database
		// For now, we do a simple name-based match
		if strings.Contains(strings.ToLower(drug.DrugName), strings.ToLower(step.DrugClass)) {
			return true
		}
	}

	return false
}

// calculateDrugDuration calculates the duration of drug use in days
func (s *STService) calculateDrugDuration(drug models.DrugHistory) int {
	// Use explicit duration if provided
	if drug.DurationDays > 0 {
		return drug.DurationDays
	}

	// Calculate from dates
	endDate := drug.EndDate
	if endDate == nil {
		// If no end date, assume still taking (use current date)
		now := time.Now()
		endDate = &now
	}

	duration := int(endDate.Sub(drug.StartDate).Hours() / 24)
	if duration < 0 {
		return 0
	}
	return duration
}

// checkExceptionApplies checks if any exception diagnosis applies
func (s *STService) checkExceptionApplies(exceptionCodes []string, diagnoses []models.DiagnosisCode) bool {
	if len(exceptionCodes) == 0 || len(diagnoses) == 0 {
		return false
	}

	for _, exCode := range exceptionCodes {
		for _, diag := range diagnoses {
			// Check exact match or ICD-10 hierarchy match
			if diag.Code == exCode ||
				strings.HasPrefix(diag.Code, exCode+".") ||
				strings.HasPrefix(exCode, diag.Code+".") {
				return true
			}
		}
	}

	return false
}

// =============================================================================
// OVERRIDE HELPERS
// =============================================================================

// isValidOverrideReason checks if the override reason is valid
func (s *STService) isValidOverrideReason(reason models.STOverrideReason) bool {
	validReasons := map[models.STOverrideReason]bool{
		models.STOverrideContraindication:  true,
		models.STOverrideAdverseReaction:   true,
		models.STOverrideTreatmentFailure:  true,
		models.STOverrideMedicalNecessity:  true,
		models.STOverrideDrugInteraction:   true,
		models.STOverrideRenalImpairment:   true,
		models.STOverrideHepaticImpairment: true,
		models.STOverridePregnancy:         true,
		models.STOverrideAgeRestriction:    true,
		models.STOverrideOther:             true,
	}
	return validReasons[reason]
}

// isOverrideAllowed checks if the override reason is allowed for this rule
func (s *STService) isOverrideAllowed(allowedCriteria []string, reason models.STOverrideReason) bool {
	if len(allowedCriteria) == 0 {
		return true // No restrictions, all reasons allowed
	}

	for _, criteria := range allowedCriteria {
		if strings.EqualFold(criteria, string(reason)) {
			return true
		}
	}
	return false
}

// shouldAutoApprove checks if the override should be auto-approved
func (s *STService) shouldAutoApprove(reason models.STOverrideReason) bool {
	// Auto-approve contraindications and adverse reactions
	autoApproveReasons := map[models.STOverrideReason]bool{
		models.STOverrideContraindication: true,
		models.STOverrideAdverseReaction:  true,
		models.STOverridePregnancy:        true,
	}
	return autoApproveReasons[reason]
}

// getOverrideStatusMessage returns a human-readable message for override status
func (s *STService) getOverrideStatusMessage(status models.STOverrideStatus) string {
	messages := map[models.STOverrideStatus]string{
		models.STOverridePending:   "Override request is pending review",
		models.STOverrideApproved:  "Override request has been approved",
		models.STOverrideDenied:    "Override request has been denied",
		models.STOverrideExpired:   "Override approval has expired",
		models.STOverrideCancelled: "Override request has been cancelled",
	}

	if msg, ok := messages[status]; ok {
		return msg
	}
	return "Unknown override status"
}
