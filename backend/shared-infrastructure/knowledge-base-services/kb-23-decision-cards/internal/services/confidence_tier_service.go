package services

import (
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/models"
)

// ConfidenceTierService implements V-01: per-template threshold computation
// with a separate firm_medication_change threshold for medication-altering
// recommendations.
type ConfidenceTierService struct {
	cfg *config.Config
	log *zap.Logger
}

// NewConfidenceTierService creates a ConfidenceTierService with the given
// configuration defaults and logger.
func NewConfidenceTierService(cfg *config.Config, log *zap.Logger) *ConfidenceTierService {
	return &ConfidenceTierService{cfg: cfg, log: log}
}

// ComputeTier determines the confidence tier from posterior probability using
// template-specific thresholds (V-01). Falls back to global defaults when a
// template does not define its own thresholds.
func (s *ConfidenceTierService) ComputeTier(posterior float64, tmpl *models.CardTemplate) models.ConfidenceTier {
	thresholds := s.getThresholds(tmpl)

	if posterior >= thresholds.FirmPosterior {
		return models.TierFirm
	}
	if posterior >= thresholds.ProbablePosterior {
		return models.TierProbable
	}
	if posterior >= thresholds.PossiblePosterior {
		return models.TierPossible
	}
	return models.TierUncertain
}

// IsFirmForMedicationChange checks the separate firm_medication_change
// threshold (V-01). This gate is used to prevent MEDICATION_HOLD and
// MEDICATION_MODIFY recommendations from being emitted unless the posterior
// exceeds a stricter cutoff than the standard firm threshold.
func (s *ConfidenceTierService) IsFirmForMedicationChange(posterior float64, tmpl *models.CardTemplate) bool {
	thresholds := s.getThresholds(tmpl)
	return posterior >= thresholds.FirmMedicationChange
}

// getThresholds returns template-specific thresholds when available, falling
// back to global config defaults for any zero-valued fields.
func (s *ConfidenceTierService) getThresholds(tmpl *models.CardTemplate) models.TemplateThresholds {
	// Use template-specific thresholds if available
	if tmpl != nil && tmpl.Thresholds != (models.TemplateThresholds{}) {
		t := tmpl.Thresholds
		// Fill in any zero values with defaults
		if t.FirmPosterior == 0 {
			t.FirmPosterior = s.cfg.DefaultFirmPosterior
		}
		if t.FirmMedicationChange == 0 {
			t.FirmMedicationChange = s.cfg.DefaultFirmMedicationChange
		}
		if t.ProbablePosterior == 0 {
			t.ProbablePosterior = s.cfg.DefaultProbablePosterior
		}
		if t.PossiblePosterior == 0 {
			t.PossiblePosterior = s.cfg.DefaultPossiblePosterior
		}
		return t
	}

	return models.TemplateThresholds{
		FirmPosterior:        s.cfg.DefaultFirmPosterior,
		FirmMedicationChange: s.cfg.DefaultFirmMedicationChange,
		ProbablePosterior:    s.cfg.DefaultProbablePosterior,
		PossiblePosterior:    s.cfg.DefaultPossiblePosterior,
	}
}
