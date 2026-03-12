package services

import (
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/models"
)

func testConfig() *config.Config {
	return &config.Config{
		DefaultFirmPosterior:        0.75,
		DefaultFirmMedicationChange: 0.82,
		DefaultProbablePosterior:    0.60,
		DefaultPossiblePosterior:    0.40,
	}
}

func TestComputeTier(t *testing.T) {
	svc := NewConfidenceTierService(testConfig(), zap.NewNop())

	tests := []struct {
		name      string
		posterior float64
		tmpl      *models.CardTemplate
		want      models.ConfidenceTier
	}{
		{
			name:      "high posterior returns FIRM",
			posterior: 0.90,
			tmpl:      nil,
			want:      models.TierFirm,
		},
		{
			name:      "exact firm boundary returns FIRM",
			posterior: 0.75,
			tmpl:      nil,
			want:      models.TierFirm,
		},
		{
			name:      "just below firm boundary returns PROBABLE",
			posterior: 0.74,
			tmpl:      nil,
			want:      models.TierProbable,
		},
		{
			name:      "exact probable boundary returns PROBABLE",
			posterior: 0.60,
			tmpl:      nil,
			want:      models.TierProbable,
		},
		{
			name:      "just below probable boundary returns POSSIBLE",
			posterior: 0.59,
			tmpl:      nil,
			want:      models.TierPossible,
		},
		{
			name:      "exact possible boundary returns POSSIBLE",
			posterior: 0.40,
			tmpl:      nil,
			want:      models.TierPossible,
		},
		{
			name:      "just below possible boundary returns UNCERTAIN",
			posterior: 0.39,
			tmpl:      nil,
			want:      models.TierUncertain,
		},
		{
			name:      "zero posterior returns UNCERTAIN",
			posterior: 0.0,
			tmpl:      nil,
			want:      models.TierUncertain,
		},
		{
			name:      "template-specific thresholds override defaults",
			posterior: 0.56,
			tmpl: &models.CardTemplate{
				Thresholds: models.TemplateThresholds{
					FirmPosterior:        0.80,
					FirmMedicationChange: 0.90,
					ProbablePosterior:    0.55,
					PossiblePosterior:    0.30,
				},
			},
			want: models.TierProbable,
		},
		{
			name:      "template with partial thresholds uses defaults for missing",
			posterior: 0.76,
			tmpl: &models.CardTemplate{
				Thresholds: models.TemplateThresholds{
					FirmPosterior: 0.80,
					// ProbablePosterior, PossiblePosterior, FirmMedicationChange: 0 -> defaults
				},
			},
			want: models.TierProbable, // below 0.80 (template firm), above 0.60 (default probable)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ComputeTier(tt.posterior, tt.tmpl)
			if got != tt.want {
				t.Errorf("ComputeTier(%v) = %q, want %q", tt.posterior, got, tt.want)
			}
		})
	}
}

func TestIsFirmForMedicationChange(t *testing.T) {
	svc := NewConfidenceTierService(testConfig(), zap.NewNop())

	tests := []struct {
		name      string
		posterior float64
		tmpl      *models.CardTemplate
		want      bool
	}{
		{
			name:      "above firm medication change threshold",
			posterior: 0.85,
			tmpl:      nil,
			want:      true,
		},
		{
			name:      "exact firm medication change boundary",
			posterior: 0.82,
			tmpl:      nil,
			want:      true,
		},
		{
			name:      "just below firm medication change threshold",
			posterior: 0.81,
			tmpl:      nil,
			want:      false,
		},
		{
			name:      "FIRM for general but not medication change",
			posterior: 0.75,
			tmpl:      nil,
			want:      false,
		},
		{
			name:      "template with custom firm_medication_change",
			posterior: 0.89,
			tmpl: &models.CardTemplate{
				Thresholds: models.TemplateThresholds{
					FirmMedicationChange: 0.90,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.IsFirmForMedicationChange(tt.posterior, tt.tmpl)
			if got != tt.want {
				t.Errorf("IsFirmForMedicationChange(%v) = %v, want %v", tt.posterior, got, tt.want)
			}
		})
	}
}
