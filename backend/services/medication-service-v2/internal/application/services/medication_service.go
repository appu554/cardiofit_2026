package services

import (
	"context"
	"fmt"
	"time"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/domain/repositories"
	"medication-service-v2/internal/infrastructure/monitoring"
	
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.opentelemetry.io/otel/trace"
)

// MedicationService handles medication proposal business logic
type MedicationService struct {
	medicationRepo        repositories.MedicationRepository
	recipeService         *RecipeService
	snapshotService       *SnapshotService
	clinicalEngineService *ClinicalEngineService
	auditService          *AuditService
	notificationService   *NotificationService
	logger                *zap.Logger
	metrics              *monitoring.Metrics
	tracer               trace.Tracer
}

// NewMedicationService creates a new medication service
func NewMedicationService(
	medicationRepo repositories.MedicationRepository,
	recipeService *RecipeService,
	snapshotService *SnapshotService,
	clinicalEngineService *ClinicalEngineService,
	auditService *AuditService,
	notificationService *NotificationService,
	logger *zap.Logger,
	metrics *monitoring.Metrics,
) *MedicationService {
	return &MedicationService{
		medicationRepo:        medicationRepo,
		recipeService:         recipeService,
		snapshotService:       snapshotService,
		clinicalEngineService: clinicalEngineService,
		auditService:          auditService,
		notificationService:   notificationService,
		logger:                logger,
		metrics:              metrics,
	}
}

// ProposeMedication creates a new medication proposal using the 4-phase workflow
func (s *MedicationService) ProposeMedication(ctx context.Context, request *ProposeMedicationRequest) (*ProposeMedicationResponse, error) {
	startTime := time.Now()
	ctx, span := s.tracer.Start(ctx, "medication_service.propose_medication")
	defer span.End()

	// Phase 1: Recipe Resolution & Ingestion
	recipe, err := s.recipeService.ResolveRecipe(ctx, &ResolveRecipeRequest{
		ProtocolID:      request.ProtocolID,
		Indication:      request.Indication,
		PatientContext:  request.ClinicalContext,
	})
	if err != nil {
		s.logger.Error("Failed to resolve recipe", zap.Error(err))
		s.metrics.RecordError("recipe_resolution", err)
		return nil, fmt.Errorf("recipe resolution failed: %w", err)
	}

	// Phase 2: Context Assembly & Snapshot Creation
	snapshot, err := s.snapshotService.CreateSnapshot(ctx, &CreateSnapshotRequest{
		PatientID:             request.PatientID,
		RecipeID:              recipe.ID,
		SnapshotType:          entities.SnapshotTypeCalculation,
		ClinicalContext:       request.ClinicalContext,
		FreshnessRequirements: recipe.ContextRequirements.FreshnessRequirements,
	})
	if err != nil {
		s.logger.Error("Failed to create clinical snapshot", zap.Error(err))
		s.metrics.RecordError("snapshot_creation", err)
		return nil, fmt.Errorf("snapshot creation failed: %w", err)
	}

	// Phase 3: Clinical Intelligence & Calculations
	calculations, err := s.clinicalEngineService.CalculateDosages(ctx, &CalculateDosagesRequest{
		Recipe:     recipe,
		Snapshot:   snapshot,
		PatientID:  request.PatientID,
		Parameters: request.CalculationParameters,
	})
	if err != nil {
		s.logger.Error("Failed to calculate dosages", zap.Error(err))
		s.metrics.RecordError("dosage_calculation", err)
		return nil, fmt.Errorf("dosage calculation failed: %w", err)
	}

	// Phase 4: Proposal Generation
	proposal := &entities.MedicationProposal{
		ID:                    uuid.New(),
		PatientID:             request.PatientID,
		ProtocolID:            request.ProtocolID,
		Indication:            request.Indication,
		Status:                entities.ProposalStatusProposed,
		ClinicalContext:       request.ClinicalContext,
		MedicationDetails:     recipe.MedicationDetails,
		DosageRecommendations: calculations.DosageRecommendations,
		SafetyConstraints:     calculations.SafetyConstraints,
		SnapshotID:            snapshot.ID,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		CreatedBy:             request.CreatedBy,
	}

	// Validate proposal
	if err := proposal.Validate(); err != nil {
		s.logger.Error("Proposal validation failed", zap.Error(err))
		return nil, fmt.Errorf("proposal validation failed: %w", err)
	}

	// Store proposal
	if err := s.medicationRepo.CreateProposal(ctx, proposal); err != nil {
		s.logger.Error("Failed to create proposal", zap.Error(err))
		s.metrics.RecordError("proposal_creation", err)
		return nil, fmt.Errorf("proposal creation failed: %w", err)
	}

	// Record audit event
	s.auditService.RecordEvent(ctx, &AuditEvent{
		EntityType:   "medication_proposal",
		EntityID:     proposal.ID.String(),
		Action:       "created",
		UserID:       request.CreatedBy,
		Details:      "Medication proposal created through 4-phase workflow",
		Timestamp:    time.Now(),
	})

	// Send notifications if needed
	if calculations.HasCriticalAlerts() {
		s.notificationService.SendAlert(ctx, &AlertNotification{
			Type:     "critical_safety_alert",
			Severity: "high",
			Message:  "Critical safety alerts detected in medication proposal",
			Recipients: []string{request.CreatedBy},
			Metadata: map[string]interface{}{
				"proposal_id": proposal.ID,
				"patient_id":  request.PatientID,
				"alerts":      calculations.SafetyAlerts,
			},
		})
	}

	// Record metrics
	processingTime := time.Since(startTime)
	s.metrics.RecordDuration("medication_proposal_processing_time", processingTime)
	s.metrics.RecordCounter("medication_proposals_created", 1, map[string]string{
		"indication": request.Indication,
		"protocol":   request.ProtocolID,
	})

	return &ProposeMedicationResponse{
		Proposal:       proposal,
		ProcessingTime: processingTime,
		Warnings:       calculations.Warnings,
		Recommendations: calculations.ClinicalRecommendations,
	}, nil
}

// ValidateProposal validates a medication proposal against safety rules
func (s *MedicationService) ValidateProposal(ctx context.Context, request *ValidateProposalRequest) (*ValidateProposalResponse, error) {
	ctx, span := s.tracer.Start(ctx, "medication_service.validate_proposal")
	defer span.End()

	// Get existing proposal
	proposal, err := s.medicationRepo.GetProposalByID(ctx, request.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposal: %w", err)
	}

	// Check if proposal can be validated
	if !proposal.CanTransitionTo(entities.ProposalStatusValidated) {
		return nil, fmt.Errorf("proposal cannot be validated in current status: %s", proposal.Status)
	}

	// Get associated recipe and snapshot for validation
	recipe, err := s.recipeService.GetRecipeByID(ctx, proposal.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}

	snapshot, err := s.snapshotService.GetSnapshotByID(ctx, proposal.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Validate against clinical rules
	validationResults, err := s.clinicalEngineService.ValidateProposal(ctx, &ValidateProposalEngineRequest{
		Proposal: proposal,
		Recipe:   recipe,
		Snapshot: snapshot,
		ValidationLevel: request.ValidationLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("clinical validation failed: %w", err)
	}

	// Update proposal based on validation results
	if validationResults.IsValid {
		proposal.Status = entities.ProposalStatusValidated
		proposal.ValidatedBy = &request.ValidatedBy
		now := time.Now()
		proposal.ValidationTimestamp = &now
	} else {
		proposal.Status = entities.ProposalStatusRejected
	}

	proposal.UpdatedAt = time.Now()

	// Update in repository
	if err := s.medicationRepo.UpdateProposal(ctx, proposal); err != nil {
		return nil, fmt.Errorf("failed to update proposal: %w", err)
	}

	// Record audit event
	s.auditService.RecordEvent(ctx, &AuditEvent{
		EntityType: "medication_proposal",
		EntityID:   proposal.ID.String(),
		Action:     "validated",
		UserID:     request.ValidatedBy,
		Details:    fmt.Sprintf("Proposal validation completed. Valid: %t", validationResults.IsValid),
		Timestamp:  time.Now(),
	})

	return &ValidateProposalResponse{
		Proposal:          proposal,
		ValidationResults: validationResults,
		IsValid:          validationResults.IsValid,
		Violations:       validationResults.SafetyViolations,
		Warnings:         validationResults.Warnings,
	}, nil
}

// CommitProposal commits a validated medication proposal
func (s *MedicationService) CommitProposal(ctx context.Context, request *CommitProposalRequest) (*CommitProposalResponse, error) {
	ctx, span := s.tracer.Start(ctx, "medication_service.commit_proposal")
	defer span.End()

	// Get and validate proposal
	proposal, err := s.medicationRepo.GetProposalByID(ctx, request.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposal: %w", err)
	}

	if !proposal.CanTransitionTo(entities.ProposalStatusCommitted) {
		return nil, fmt.Errorf("proposal cannot be committed in current status: %s", proposal.Status)
	}

	// Create commit snapshot
	commitSnapshot, err := s.snapshotService.CreateSnapshot(ctx, &CreateSnapshotRequest{
		PatientID:      proposal.PatientID,
		RecipeID:       uuid.New(), // Will be set by snapshot service
		SnapshotType:   entities.SnapshotTypeCommit,
		ClinicalContext: proposal.ClinicalContext,
		FreshnessRequirements: map[string]time.Duration{
			"final_validation": 5 * time.Minute,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create commit snapshot: %w", err)
	}

	// Final safety validation before commit
	finalValidation, err := s.clinicalEngineService.FinalSafetyCheck(ctx, &FinalSafetyCheckRequest{
		Proposal: proposal,
		CommitSnapshot: commitSnapshot,
	})
	if err != nil {
		return nil, fmt.Errorf("final safety check failed: %w", err)
	}

	if !finalValidation.IsSafe {
		return nil, fmt.Errorf("final safety check failed: %s", finalValidation.Reason)
	}

	// Update proposal status
	proposal.Status = entities.ProposalStatusCommitted
	proposal.UpdatedAt = time.Now()

	if err := s.medicationRepo.UpdateProposal(ctx, proposal); err != nil {
		return nil, fmt.Errorf("failed to commit proposal: %w", err)
	}

	// Record audit event
	s.auditService.RecordEvent(ctx, &AuditEvent{
		EntityType: "medication_proposal",
		EntityID:   proposal.ID.String(),
		Action:     "committed",
		UserID:     request.CommittedBy,
		Details:    "Proposal successfully committed to clinical record",
		Timestamp:  time.Now(),
	})

	// Send success notification
	s.notificationService.SendNotification(ctx, &Notification{
		Type:    "proposal_committed",
		Message: "Medication proposal has been successfully committed",
		Recipients: []string{proposal.CreatedBy, request.CommittedBy},
		Metadata: map[string]interface{}{
			"proposal_id": proposal.ID,
			"patient_id":  proposal.PatientID,
		},
	})

	return &CommitProposalResponse{
		Proposal:       proposal,
		CommitSnapshot: commitSnapshot,
		CommittedAt:    time.Now(),
		Success:        true,
	}, nil
}

// GetProposal retrieves a medication proposal by ID
func (s *MedicationService) GetProposal(ctx context.Context, proposalID uuid.UUID) (*entities.MedicationProposal, error) {
	ctx, span := s.tracer.Start(ctx, "medication_service.get_proposal")
	defer span.End()

	proposal, err := s.medicationRepo.GetProposalByID(ctx, proposalID)
	if err != nil {
		s.logger.Error("Failed to get proposal", zap.String("proposal_id", proposalID.String()), zap.Error(err))
		return nil, err
	}

	// Record audit view event
	s.auditService.RecordEvent(ctx, &AuditEvent{
		EntityType: "medication_proposal",
		EntityID:   proposalID.String(),
		Action:     "viewed",
		Timestamp:  time.Now(),
	})

	return proposal, nil
}

// ListPatientProposals retrieves all proposals for a patient
func (s *MedicationService) ListPatientProposals(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicationProposal, error) {
	ctx, span := s.tracer.Start(ctx, "medication_service.list_patient_proposals")
	defer span.End()

	proposals, err := s.medicationRepo.GetProposalsByPatientID(ctx, patientID)
	if err != nil {
		s.logger.Error("Failed to get patient proposals", zap.String("patient_id", patientID.String()), zap.Error(err))
		return nil, err
	}

	return proposals, nil
}

// SearchProposals searches for proposals based on criteria
func (s *MedicationService) SearchProposals(ctx context.Context, criteria *repositories.SearchCriteria) ([]*entities.MedicationProposal, error) {
	ctx, span := s.tracer.Start(ctx, "medication_service.search_proposals")
	defer span.End()

	proposals, err := s.medicationRepo.SearchProposals(ctx, *criteria)
	if err != nil {
		s.logger.Error("Failed to search proposals", zap.Error(err))
		return nil, err
	}

	return proposals, nil
}

// GetProposalStatistics retrieves proposal statistics for analytics
func (s *MedicationService) GetProposalStatistics(ctx context.Context, timeRange repositories.TimeRange) (*repositories.ProposalStatistics, error) {
	ctx, span := s.tracer.Start(ctx, "medication_service.get_proposal_statistics")
	defer span.End()

	stats, err := s.medicationRepo.GetProposalStatistics(ctx, timeRange)
	if err != nil {
		s.logger.Error("Failed to get proposal statistics", zap.Error(err))
		return nil, err
	}

	return stats, nil
}