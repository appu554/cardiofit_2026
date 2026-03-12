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

// PAService handles Prior Authorization business logic
type PAService struct {
	repo         *repository.PARepository
	eventEmitter *EventEmitter
}

// NewPAService creates a new PAService instance
func NewPAService(repo *repository.PARepository) *PAService {
	return &PAService{repo: repo}
}

// SetEventEmitter sets the event emitter for cross-service signaling
func (s *PAService) SetEventEmitter(emitter *EventEmitter) {
	s.eventEmitter = emitter
}

// GetRequirements retrieves PA requirements for a drug
func (s *PAService) GetRequirements(ctx context.Context, req *models.PARequirementsRequest) (*models.PARequirementsResponse, error) {
	paReq, err := s.repo.GetRequirements(ctx, req.DrugRxNorm, req.PayerID, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PA requirements: %w", err)
	}

	// No PA required for this drug
	if paReq == nil {
		return &models.PARequirementsResponse{
			PARequired: false,
			DrugRxNorm: req.DrugRxNorm,
		}, nil
	}

	return &models.PARequirementsResponse{
		PARequired:        true,
		DrugRxNorm:        paReq.DrugRxNorm,
		DrugName:          paReq.DrugName,
		Criteria:          paReq.Criteria,
		RequiredDocuments: paReq.RequiredDocuments,
		ApprovalDuration:  paReq.ApprovalDurationDays,
		UrgencyLevels:     paReq.UrgencyLevels,
		ReviewTimeframes: models.ReviewTimeframes{
			StandardHours:  paReq.StandardReviewHours,
			UrgentHours:    paReq.UrgentReviewHours,
			ExpeditedHours: paReq.ExpeditedReviewHours,
		},
	}, nil
}

// CheckPA evaluates if PA is required and if criteria are met
func (s *PAService) CheckPA(ctx context.Context, req *models.PACheckRequest) (*models.PACheckResponse, error) {
	// Check if there's an active approval for this patient/drug combination
	existingApproval, err := s.repo.GetActiveApproval(ctx, req.PatientID, req.DrugRxNorm, req.PayerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing approval: %w", err)
	}

	if existingApproval != nil {
		return &models.PACheckResponse{
			PARequired:       true,
			PAStatus:         "pre_approved",
			CriteriaMet:      true,
			Message:          "Active prior authorization exists for this patient and drug",
			ExistingApproval: existingApproval,
		}, nil
	}

	// Get PA requirements
	paReq, err := s.repo.GetRequirements(ctx, req.DrugRxNorm, req.PayerID, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PA requirements: %w", err)
	}

	// No PA required
	if paReq == nil {
		return &models.PACheckResponse{
			PARequired:  false,
			PAStatus:    "not_required",
			CriteriaMet: true,
			Message:     "Prior authorization is not required for this drug",
		}, nil
	}

	// Evaluate clinical criteria
	evaluation := s.evaluateCriteria(ctx, paReq.Criteria, req)

	// Determine response based on criteria evaluation
	response := &models.PACheckResponse{
		PARequired:        true,
		CriteriaMet:       evaluation.AllCriteriaMet,
		CriteriaResults:   evaluation.CriteriaResults,
		RequiredDocuments: paReq.RequiredDocuments,
	}

	if evaluation.AllCriteriaMet {
		response.PAStatus = "criteria_met"
		response.Message = "All criteria met. PA submission likely to be approved."
	} else {
		response.PAStatus = "requires_submission"
		response.Message = "Some criteria not met. PA submission required with additional documentation."
		response.MissingCriteria = s.extractMissingCriteria(evaluation)
	}

	return response, nil
}

// SubmitPA creates a new PA submission
func (s *PAService) SubmitPA(ctx context.Context, req *models.PASubmitRequest) (*models.PASubmission, error) {
	// Validate required fields
	if req.DrugRxNorm == "" || req.PatientID == "" || req.ProviderID == "" {
		return nil, fmt.Errorf("missing required fields: drug_rxnorm, patient_id, and provider_id are required")
	}

	// Get drug name from PA requirements or use RxNorm code
	drugName := req.DrugRxNorm
	paReq, _ := s.repo.GetRequirements(ctx, req.DrugRxNorm, req.PayerID, req.PlanID)
	if paReq != nil {
		drugName = paReq.DrugName
	}

	// Set default urgency level if not provided
	urgency := req.UrgencyLevel
	if urgency == "" {
		urgency = models.PAUrgencyStandard
	}

	// Create submission
	submission := &models.PASubmission{
		PatientID:             req.PatientID,
		ProviderID:            req.ProviderID,
		ProviderNPI:           req.ProviderNPI,
		DrugRxNorm:            req.DrugRxNorm,
		DrugName:              drugName,
		Quantity:              req.Quantity,
		DaysSupply:            req.DaysSupply,
		ClinicalDocumentation: req.ClinicalDocumentation,
		PayerID:               req.PayerID,
		PlanID:                req.PlanID,
		MemberID:              req.MemberID,
		Status:                models.PAStatusPending,
		UrgencyLevel:          urgency,
		CreatedBy:             &req.ProviderID,
	}

	// Append clinical notes if provided
	if req.ClinicalNotes != "" {
		submission.ClinicalDocumentation.ClinicalNotes = req.ClinicalNotes
	}

	// Save to database
	if err := s.repo.CreateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create PA submission: %w", err)
	}

	// Auto-evaluate criteria for the submission
	if paReq != nil {
		checkReq := &models.PACheckRequest{
			DrugRxNorm:   req.DrugRxNorm,
			PatientID:    req.PatientID,
			PayerID:      req.PayerID,
			PlanID:       req.PlanID,
			Diagnoses:    req.ClinicalDocumentation.Diagnoses,
			LabResults:   req.ClinicalDocumentation.LabResults,
			PriorTherapy: req.ClinicalDocumentation.PriorTherapy,
		}
		evaluation := s.evaluateCriteria(ctx, paReq.Criteria, checkReq)

		// Save criteria evaluations
		for _, result := range evaluation.CriteriaResults {
			if err := s.repo.SaveCriteriaEvaluation(ctx, submission.ID, result); err != nil {
				// Log but don't fail the submission
				continue
			}
		}

		// Auto-approve if all criteria met (for non-controlled substances)
		if evaluation.AllCriteriaMet && !isControlledSubstance(req.DrugRxNorm) {
			expiresAt := time.Now().AddDate(0, 0, paReq.ApprovalDurationDays)
			submission.Status = models.PAStatusApproved
			submission.ExpiresAt = &expiresAt
			submission.ApprovedQuantity = &req.Quantity
			submission.ApprovedDaysSupply = &req.DaysSupply
			reason := "Auto-approved: All clinical criteria met"
			submission.DecisionReason = &reason
			now := time.Now()
			submission.DecisionAt = &now
			s.repo.UpdateSubmissionStatus(ctx, submission.ID, models.PAStatusApproved, reason, "SYSTEM")
		}
	}

	return submission, nil
}

// GetStatus retrieves the status of a PA submission
func (s *PAService) GetStatus(ctx context.Context, paID string) (*models.PAStatusResponse, error) {
	submissionID, err := uuid.Parse(paID)
	if err != nil {
		return nil, fmt.Errorf("invalid PA ID format: %w", err)
	}

	submission, err := s.repo.GetSubmission(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PA submission: %w", err)
	}

	if submission == nil {
		return nil, fmt.Errorf("PA submission not found: %s", paID)
	}

	// Get criteria evaluations if available
	evaluations, _ := s.repo.GetCriteriaEvaluations(ctx, submissionID)

	return &models.PAStatusResponse{
		Submission:      *submission,
		CriteriaResults: evaluations,
		Message:         s.getStatusMessage(submission.Status),
	}, nil
}

// ListPending returns pending PA submissions for review
func (s *PAService) ListPending(ctx context.Context, limit, offset int) ([]models.PASubmission, error) {
	return s.repo.ListPendingSubmissions(ctx, limit, offset)
}

// =============================================================================
// CRITERIA EVALUATION ENGINE
// =============================================================================

// evaluateCriteria evaluates all PA criteria against patient data
func (s *PAService) evaluateCriteria(ctx context.Context, criteria []models.PACriterion, req *models.PACheckRequest) *models.PAEvaluation {
	evaluation := &models.PAEvaluation{
		AllCriteriaMet:  true,
		CriteriaResults: make([]models.CriterionEvaluation, 0, len(criteria)),
		EvaluatedAt:     time.Now(),
	}

	for _, criterion := range criteria {
		result := s.evaluateSingleCriterion(ctx, criterion, req)
		evaluation.CriteriaResults = append(evaluation.CriteriaResults, result)

		if !result.Met && criterion.Required {
			evaluation.AllCriteriaMet = false
		}
	}

	// Determine recommended status
	if evaluation.AllCriteriaMet {
		evaluation.RecommendedStatus = models.PAStatusApproved
	} else {
		evaluation.RecommendedStatus = models.PAStatusNeedInfo
	}

	return evaluation
}

// evaluateSingleCriterion evaluates a single PA criterion
func (s *PAService) evaluateSingleCriterion(ctx context.Context, criterion models.PACriterion, req *models.PACheckRequest) models.CriterionEvaluation {
	result := models.CriterionEvaluation{
		Criterion:   criterion,
		Met:         false,
		EvaluatedAt: time.Now(),
	}

	switch criterion.Type {
	case models.PACriterionDiagnosis:
		result.Met, result.Evidence, result.Notes = s.evaluateDiagnosis(criterion, req.Diagnoses)

	case models.PACriterionLab:
		result.Met, result.Evidence, result.Notes = s.evaluateLab(criterion, req.LabResults)

	case models.PACriterionPriorTherapy:
		result.Met, result.Evidence, result.Notes = s.evaluatePriorTherapy(criterion, req.PriorTherapy)

	case models.PACriterionAge:
		result.Met, result.Evidence, result.Notes = s.evaluateAge(criterion, req.PatientAge)

	case models.PACriterionContraindication:
		result.Met, result.Evidence, result.Notes = s.evaluateContraindication(criterion, req)

	default:
		result.Notes = fmt.Sprintf("Unknown criterion type: %s", criterion.Type)
	}

	return result
}

// evaluateDiagnosis checks if patient has required diagnoses
func (s *PAService) evaluateDiagnosis(criterion models.PACriterion, diagnoses []models.DiagnosisCode) (bool, interface{}, string) {
	if len(criterion.Codes) == 0 {
		return true, nil, "No specific diagnosis codes required"
	}

	if len(diagnoses) == 0 {
		return false, nil, "No diagnoses provided"
	}

	matchedCodes := []string{}
	for _, requiredCode := range criterion.Codes {
		for _, patientDiag := range diagnoses {
			// Check exact match or prefix match (for ICD-10 hierarchy)
			if patientDiag.Code == requiredCode ||
				strings.HasPrefix(patientDiag.Code, requiredCode+".") ||
				strings.HasPrefix(requiredCode, patientDiag.Code+".") {
				matchedCodes = append(matchedCodes, patientDiag.Code)
			}
		}
	}

	if len(matchedCodes) > 0 {
		return true, matchedCodes, fmt.Sprintf("Diagnosis criteria met: %v", matchedCodes)
	}

	return false, nil, fmt.Sprintf("Required diagnosis not found. Expected one of: %v", criterion.Codes)
}

// evaluateLab checks if patient has required lab values
func (s *PAService) evaluateLab(criterion models.PACriterion, labs []models.LabResult) (bool, interface{}, string) {
	if criterion.Test == "" {
		return true, nil, "No specific lab test required"
	}

	if len(labs) == 0 {
		return false, nil, fmt.Sprintf("Lab result for %s not provided", criterion.Test)
	}

	// Find matching lab result
	var matchingLab *models.LabResult
	for i := range labs {
		lab := &labs[i]
		// Match by test name or LOINC code
		if strings.EqualFold(lab.Test, criterion.Test) ||
			(criterion.LOINC != "" && lab.LOINC == criterion.LOINC) {

			// Check if lab is within max age
			if criterion.MaxAgeDays > 0 {
				maxAge := time.Now().AddDate(0, 0, -criterion.MaxAgeDays)
				if lab.Date.Before(maxAge) {
					continue // Lab too old
				}
			}

			// Use most recent matching lab
			if matchingLab == nil || lab.Date.After(matchingLab.Date) {
				matchingLab = lab
			}
		}
	}

	if matchingLab == nil {
		notes := fmt.Sprintf("No recent %s result found", criterion.Test)
		if criterion.MaxAgeDays > 0 {
			notes += fmt.Sprintf(" (must be within %d days)", criterion.MaxAgeDays)
		}
		return false, nil, notes
	}

	// Compare value against threshold
	met := s.compareValue(matchingLab.Value, criterion.Operator, criterion.Value)

	evidence := map[string]interface{}{
		"test":     matchingLab.Test,
		"value":    matchingLab.Value,
		"unit":     matchingLab.Unit,
		"date":     matchingLab.Date,
		"required": fmt.Sprintf("%s %s", criterion.Operator, formatFloat(criterion.Value)),
	}

	if met {
		return true, evidence, fmt.Sprintf("%s = %.2f %s criteria (%s %.2f)",
			criterion.Test, matchingLab.Value, "meets", criterion.Operator, criterion.Value)
	}

	return false, evidence, fmt.Sprintf("%s = %.2f does not meet criteria (%s %.2f)",
		criterion.Test, matchingLab.Value, criterion.Operator, criterion.Value)
}

// evaluatePriorTherapy checks if patient has required prior medication history
func (s *PAService) evaluatePriorTherapy(criterion models.PACriterion, history []models.DrugHistory) (bool, interface{}, string) {
	if len(criterion.RxNormCodes) == 0 && criterion.DrugClass == "" {
		return true, nil, "No prior therapy required"
	}

	if len(history) == 0 {
		// Check if contraindication is acceptable alternative
		if criterion.OrContraindication {
			return false, nil, "No prior therapy history provided. Contraindication documentation may be acceptable."
		}
		return false, nil, "No prior medication history provided"
	}

	// Find matching prior therapy
	for _, drug := range history {
		matched := false

		// Check RxNorm codes
		for _, reqCode := range criterion.RxNormCodes {
			if drug.RxNormCode == reqCode {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		// Calculate duration
		duration := drug.DurationDays
		if duration == 0 && drug.EndDate != nil {
			duration = int(drug.EndDate.Sub(drug.StartDate).Hours() / 24)
		} else if duration == 0 {
			duration = int(time.Since(drug.StartDate).Hours() / 24)
		}

		// Check minimum duration
		if criterion.MinDurationDays > 0 && duration < criterion.MinDurationDays {
			evidence := map[string]interface{}{
				"drug":             drug.DrugName,
				"rxnorm":           drug.RxNormCode,
				"duration_days":    duration,
				"required_days":    criterion.MinDurationDays,
			}
			return false, evidence, fmt.Sprintf("%s therapy duration (%d days) less than required (%d days)",
				drug.DrugName, duration, criterion.MinDurationDays)
		}

		// Duration met
		evidence := map[string]interface{}{
			"drug":          drug.DrugName,
			"rxnorm":        drug.RxNormCode,
			"duration_days": duration,
			"start_date":    drug.StartDate,
		}
		return true, evidence, fmt.Sprintf("Prior %s therapy documented (%d days)", drug.DrugName, duration)
	}

	// No matching therapy found
	if criterion.OrContraindication {
		return false, nil, fmt.Sprintf("Prior therapy with %s not found. Contraindication documentation may be acceptable.", criterion.DrugClass)
	}
	return false, nil, fmt.Sprintf("Prior therapy with %s not documented", criterion.DrugClass)
}

// evaluateAge checks patient age against criteria
func (s *PAService) evaluateAge(criterion models.PACriterion, patientAge *int) (bool, interface{}, string) {
	if patientAge == nil {
		return false, nil, "Patient age not provided"
	}

	met := s.compareValue(float64(*patientAge), criterion.Operator, criterion.Value)

	evidence := map[string]interface{}{
		"patient_age": *patientAge,
		"required":    fmt.Sprintf("%s %v", criterion.Operator, int(criterion.Value)),
	}

	if met {
		return true, evidence, fmt.Sprintf("Patient age %d meets criteria (%s %d)",
			*patientAge, criterion.Operator, int(criterion.Value))
	}

	return false, evidence, fmt.Sprintf("Patient age %d does not meet criteria (%s %d)",
		*patientAge, criterion.Operator, int(criterion.Value))
}

// evaluateContraindication checks for contraindication documentation
func (s *PAService) evaluateContraindication(criterion models.PACriterion, req *models.PACheckRequest) (bool, interface{}, string) {
	// Check if any contraindication conditions are documented
	for _, condition := range criterion.Conditions {
		for _, diag := range req.Diagnoses {
			if strings.Contains(strings.ToLower(diag.Description), strings.ToLower(condition)) ||
				diag.Code == condition {
				evidence := map[string]interface{}{
					"condition": condition,
					"diagnosis": diag,
				}
				return true, evidence, fmt.Sprintf("Contraindication documented: %s", condition)
			}
		}
	}

	return false, nil, "No contraindication documentation found"
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// compareValue compares a value against a threshold using the specified operator
func (s *PAService) compareValue(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "=", "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

// extractMissingCriteria extracts criteria that were not met
func (s *PAService) extractMissingCriteria(evaluation *models.PAEvaluation) []models.PACriterion {
	missing := make([]models.PACriterion, 0)
	for _, result := range evaluation.CriteriaResults {
		if !result.Met && result.Criterion.Required {
			missing = append(missing, result.Criterion)
		}
	}
	return missing
}

// getStatusMessage returns a human-readable message for PA status
func (s *PAService) getStatusMessage(status models.PAStatus) string {
	messages := map[models.PAStatus]string{
		models.PAStatusPending:     "PA request is pending initial review",
		models.PAStatusUnderReview: "PA request is under clinical review",
		models.PAStatusApproved:    "PA request has been approved",
		models.PAStatusDenied:      "PA request has been denied",
		models.PAStatusNeedInfo:    "Additional information required for PA decision",
		models.PAStatusExpired:     "PA approval has expired",
		models.PAStatusCancelled:   "PA request has been cancelled",
	}

	if msg, ok := messages[status]; ok {
		return msg
	}
	return "Unknown PA status"
}

// isControlledSubstance checks if a drug is a controlled substance (simplified check)
func isControlledSubstance(rxnormCode string) bool {
	// RxNorm codes for common controlled substances
	// In production, this would query a database table
	controlledCodes := map[string]bool{
		"7804":  true, // Oxycodone
		"5489":  true, // Hydrocodone
		"3423":  true, // Fentanyl
		"6813":  true, // Morphine
		"6470":  true, // Methadone
		"2670":  true, // Codeine
		"37801": true, // Amphetamine salts
		"40114": true, // Methylphenidate
	}
	return controlledCodes[rxnormCode]
}

// formatFloat formats a float for display
func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return fmt.Sprintf("%d", int(f))
	}
	return fmt.Sprintf("%.2f", f)
}
