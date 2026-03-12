// Package services provides business logic for KB-6 Formulary Service.
package services

import (
	"context"
	"fmt"

	"kb-formulary/internal/models"
	"kb-formulary/internal/repository"
)

// QLService handles Quantity Limit business logic
type QLService struct {
	repo         *repository.QLRepository
	eventEmitter *EventEmitter
}

// SetEventEmitter sets the event emitter for cross-service signaling (Enhancement #2)
func (s *QLService) SetEventEmitter(emitter *EventEmitter) {
	s.eventEmitter = emitter
}

// NewQLService creates a new QLService instance
func NewQLService(repo *repository.QLRepository) *QLService {
	return &QLService{repo: repo}
}

// CheckQuantityLimits validates a prescription against quantity limits
func (s *QLService) CheckQuantityLimits(ctx context.Context, req *models.QLCheckRequest) (*models.QLCheckResponse, error) {
	// Get formulary limits for this drug
	limits, drugName, err := s.repo.GetFormularyLimits(ctx, req.DrugRxNorm, req.PayerID, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quantity limits: %w", err)
	}

	response := &models.QLCheckResponse{
		DrugRxNorm:    req.DrugRxNorm,
		DrugName:      drugName,
		RequestedQty:  req.Quantity,
		RequestedDays: req.DaysSupply,
		Limits:        limits,
		WithinLimits:  true,
		Violations:    []models.QLViolation{},
	}

	// No limits found
	if limits == nil {
		response.Message = "No quantity limits defined for this drug"
		return response, nil
	}

	// Check for existing override
	if req.PatientID != nil && *req.PatientID != "" {
		override, err := s.repo.GetActiveOverride(ctx, *req.PatientID, req.DrugRxNorm, req.PayerID)
		if err == nil && override != nil {
			response.ExistingOverride = override
			// Apply override limits
			if override.ApprovedQuantity > 0 {
				limits.MaxQuantity = override.ApprovedQuantity
			}
			if override.ApprovedDaysSupply > 0 {
				maxDays := override.ApprovedDaysSupply
				limits.MaxDaysSupply = &maxDays
			}
			if override.ApprovedFillsYear > 0 {
				limits.MaxFillsPerYear = override.ApprovedFillsYear
			}
		}

		// Get current fill count for the year
		if req.FillsThisYear == 0 {
			fillCount, _ := s.repo.GetPatientFillCount(ctx, *req.PatientID, req.DrugRxNorm, req.PayerID)
			req.FillsThisYear = fillCount
		}
	}

	// Validate against limits using the helper function
	response.Violations = models.CheckQuantityLimits(*req, limits)
	response.WithinLimits = len(response.Violations) == 0

	// Determine override eligibility
	response.OverrideAllowed = s.isOverrideAllowed(response.Violations)

	// Calculate suggested quantities if violations exist
	if !response.WithinLimits {
		suggestedQty := models.CalculateSuggestedQuantity(req.Quantity, limits)
		response.SuggestedQty = &suggestedQty

		if limits.MaxDaysSupply != nil && req.DaysSupply > *limits.MaxDaysSupply {
			suggestedDays := *limits.MaxDaysSupply
			response.SuggestedDays = &suggestedDays
		}

		response.Message = s.buildViolationMessage(response.Violations)
	} else {
		response.Message = "Requested quantity is within limits"
	}

	// Log the check for audit
	s.repo.SaveCheckLog(ctx, req, response)

	return response, nil
}

// RequestOverride creates a quantity limit override request
func (s *QLService) RequestOverride(ctx context.Context, req *models.QLOverrideRequest) (*models.QLOverrideResponse, error) {
	// Validate required fields
	if req.DrugRxNorm == "" || req.PatientID == "" || req.ProviderID == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	if req.OverrideReason == "" {
		return nil, fmt.Errorf("override reason is required")
	}

	// Get current limits to verify the override is needed
	limits, _, err := s.repo.GetFormularyLimits(ctx, req.DrugRxNorm, req.PayerID, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quantity limits: %w", err)
	}

	if limits == nil {
		return &models.QLOverrideResponse{
			Approved: true,
			Message:  "No quantity limits defined for this drug; override not needed",
		}, nil
	}

	// Determine if auto-approval is appropriate
	autoApprove := s.shouldAutoApprove(req.OverrideReason, req.RequestedQuantity, limits)

	override, err := s.repo.CreateOverride(ctx, req, autoApprove)
	if err != nil {
		return nil, fmt.Errorf("failed to create override: %w", err)
	}

	response := &models.QLOverrideResponse{
		Approved: autoApprove,
		Override: override,
	}

	if autoApprove {
		response.Message = "Override approved. New quantity limits applied."
	} else {
		response.Message = "Override request submitted for clinical review."
	}

	return response, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// isOverrideAllowed determines if violations can be overridden
func (s *QLService) isOverrideAllowed(violations []models.QLViolation) bool {
	for _, v := range violations {
		// Fills per year violations typically cannot be overridden easily
		if v.Type == models.QLViolationFillsPerYear {
			return false
		}
	}
	return true
}

// buildViolationMessage creates a human-readable message from violations
func (s *QLService) buildViolationMessage(violations []models.QLViolation) string {
	if len(violations) == 0 {
		return ""
	}

	if len(violations) == 1 {
		return violations[0].Message
	}

	return fmt.Sprintf("Multiple quantity limit violations (%d): %s",
		len(violations), violations[0].Message)
}

// shouldAutoApprove determines if the override should be auto-approved
func (s *QLService) shouldAutoApprove(reason string, requestedQty int, limits *models.ExtendedQuantityLimit) bool {
	// Auto-approve minor quantity increases (up to 10% over limit)
	if limits != nil && limits.MaxQuantity > 0 {
		allowedOverage := float64(limits.MaxQuantity) * 1.10
		if float64(requestedQty) <= allowedOverage {
			return true
		}
	}

	// Auto-approve for certain medical reasons
	autoApproveReasons := map[string]bool{
		"medical_necessity": true,
		"chronic_condition": true,
		"travel_supply":     true,
	}

	return autoApproveReasons[reason]
}

// GetLimitsForDrug retrieves quantity limits without performing a check
func (s *QLService) GetLimitsForDrug(ctx context.Context, rxnormCode string, payerID, planID *string) (*models.ExtendedQuantityLimit, string, error) {
	return s.repo.GetFormularyLimits(ctx, rxnormCode, payerID, planID)
}
