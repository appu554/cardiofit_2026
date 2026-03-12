package services

import (
	"fmt"

	"go.uber.org/zap"

	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// PipelineService handles batch write operations from the guideline extraction
// pipeline and SPLGuard ETL path.
type PipelineService struct {
	db         *database.Database
	logger     *zap.Logger
	adrService *ADRService
}

// NewPipelineService creates the pipeline batch write service.
func NewPipelineService(db *database.Database, logger *zap.Logger, adrService *ADRService) *PipelineService {
	return &PipelineService{db: db, logger: logger, adrService: adrService}
}

// BatchWriteModifiers stores multiple context modifiers from pipeline extraction.
// Auto-verifies completeness_grade on write (server recomputation wins on mismatch).
func (s *PipelineService) BatchWriteModifiers(modifiers []models.ContextModifier) (*BatchWriteResult, error) {
	result := &BatchWriteResult{}

	for i := range modifiers {
		cm := &modifiers[i]

		// Auto-verify completeness grade
		cm.CompletenessGrade = computeModifierGrade(cm)

		if err := s.db.DB.Create(cm).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("modifier %d: %s", i, err.Error()))
			result.Failed++
		} else {
			result.Succeeded++
		}
	}

	s.logger.Info("Batch write modifiers completed",
		zap.Int("succeeded", result.Succeeded),
		zap.Int("failed", result.Failed))

	return result, nil
}

// BatchWriteADRProfiles stores multiple ADR profiles from pipeline extraction.
// Uses upsert with merge strategy for dual-path records.
func (s *PipelineService) BatchWriteADRProfiles(profiles []models.AdverseReactionProfile) (*BatchWriteResult, error) {
	result := &BatchWriteResult{}

	for i := range profiles {
		profile := &profiles[i]

		// Server recomputes completeness grade
		profile.ComputeCompletenessGrade()

		if err := s.adrService.Upsert(profile); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("profile %d: %s", i, err.Error()))
			result.Failed++
		} else {
			result.Succeeded++
		}
	}

	s.logger.Info("Batch write ADR profiles completed",
		zap.Int("succeeded", result.Succeeded),
		zap.Int("failed", result.Failed))

	return result, nil
}

// BatchWriteResult summarizes the outcome of a batch write.
type BatchWriteResult struct {
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

// computeModifierGrade determines completeness grade for a context modifier.
func computeModifierGrade(cm *models.ContextModifier) string {
	hasCore := cm.ModifierValue != "" && cm.Effect != ""
	hasStructured := false

	switch cm.ModifierType {
	case "LAB_VALUE":
		hasStructured = cm.LabParameter != "" && cm.LabOperator != "" && cm.LabThreshold != 0
	case "CONCOMITANT_DRUG":
		hasStructured = cm.DrugClassTrigger != ""
	default:
		mag, _ := cm.Magnitude.Float64()
		hasStructured = mag > 0
	}

	if hasCore && hasStructured {
		return "FULL"
	}
	if hasCore {
		return "PARTIAL"
	}
	return "STUB"
}
