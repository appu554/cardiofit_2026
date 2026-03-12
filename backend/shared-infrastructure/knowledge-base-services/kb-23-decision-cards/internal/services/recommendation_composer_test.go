package services

import (
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

func TestCompose(t *testing.T) {
	composer := NewRecommendationComposer(testConfig(), zap.NewNop())

	tests := []struct {
		name            string
		tmpl            *models.CardTemplate
		tier            models.ConfidenceTier
		isFirmMedChange bool
		wantCount       int
		wantTypes       []models.RecommendationType
	}{
		{
			name: "V-04: SAFETY_INSTRUCTION with bypass included at UNCERTAIN",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecSafetyInstruction,
						Urgency:                models.UrgencyImmediate,
						ActionTextEn:           "Stop medication immediately",
						BypassesConfidenceGate: true,
						ConfidenceTierRequired: models.TierFirm,
					},
				},
			},
			tier:            models.TierUncertain,
			isFirmMedChange: false,
			wantCount:       1,
			wantTypes:       []models.RecommendationType{models.RecSafetyInstruction},
		},
		{
			name: "V-01: MEDICATION_HOLD at FIRM with isFirmMedChange true included",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecMedicationHold,
						Urgency:                models.UrgencyUrgent,
						ActionTextEn:           "Hold metformin",
						ConfidenceTierRequired: models.TierFirm,
					},
				},
			},
			tier:            models.TierFirm,
			isFirmMedChange: true,
			wantCount:       1,
			wantTypes:       []models.RecommendationType{models.RecMedicationHold},
		},
		{
			name: "V-01: MEDICATION_HOLD at FIRM with isFirmMedChange false excluded",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecMedicationHold,
						Urgency:                models.UrgencyUrgent,
						ActionTextEn:           "Hold metformin",
						ConfidenceTierRequired: models.TierFirm,
					},
				},
			},
			tier:            models.TierFirm,
			isFirmMedChange: false,
			wantCount:       0,
			wantTypes:       nil,
		},
		{
			name: "V-05: MEDICATION_REVIEW at PROBABLE tier included",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecMedicationReview,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Review medication regimen",
						ConfidenceTierRequired: models.TierProbable,
					},
				},
			},
			tier:            models.TierProbable,
			isFirmMedChange: false,
			wantCount:       1,
			wantTypes:       []models.RecommendationType{models.RecMedicationReview},
		},
		{
			name: "V-05: MEDICATION_REVIEW at POSSIBLE tier excluded",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecMedicationReview,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Review medication regimen",
						ConfidenceTierRequired: models.TierProbable,
					},
				},
			},
			tier:            models.TierPossible,
			isFirmMedChange: false,
			wantCount:       0,
			wantTypes:       nil,
		},
		{
			name: "MONITORING at POSSIBLE tier with required POSSIBLE included",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecMonitoring,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Monitor blood glucose daily",
						ConfidenceTierRequired: models.TierPossible,
					},
				},
			},
			tier:            models.TierPossible,
			isFirmMedChange: false,
			wantCount:       1,
			wantTypes:       []models.RecommendationType{models.RecMonitoring},
		},
		{
			name: "INVESTIGATION at UNCERTAIN tier with required FIRM excluded",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecInvestigation,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Order HbA1c test",
						ConfidenceTierRequired: models.TierFirm,
					},
				},
			},
			tier:            models.TierUncertain,
			isFirmMedChange: false,
			wantCount:       0,
			wantTypes:       nil,
		},
		{
			name: "mixed recommendations filtered correctly",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecSafetyInstruction,
						Urgency:                models.UrgencyImmediate,
						ActionTextEn:           "Safety alert",
						BypassesConfidenceGate: true,
						ConfidenceTierRequired: models.TierFirm,
					},
					{
						RecType:                models.RecMedicationHold,
						Urgency:                models.UrgencyUrgent,
						ActionTextEn:           "Hold drug",
						ConfidenceTierRequired: models.TierFirm,
					},
					{
						RecType:                models.RecMonitoring,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Monitor vitals",
						ConfidenceTierRequired: models.TierPossible,
					},
					{
						RecType:                models.RecInvestigation,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Order labs",
						ConfidenceTierRequired: models.TierFirm,
					},
				},
			},
			tier:            models.TierProbable,
			isFirmMedChange: false,
			wantCount:       2, // safety (bypass) + monitoring (POSSIBLE <= PROBABLE)
			wantTypes:       []models.RecommendationType{models.RecSafetyInstruction, models.RecMonitoring},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := composer.Compose(tt.tmpl, tt.tier, tt.isFirmMedChange)
			if len(got) != tt.wantCount {
				t.Fatalf("Compose() returned %d recommendations, want %d", len(got), tt.wantCount)
			}
			for i, wantType := range tt.wantTypes {
				if got[i].RecType != wantType {
					t.Errorf("recommendation[%d].RecType = %q, want %q", i, got[i].RecType, wantType)
				}
			}
		})
	}
}

func TestComposeFromSecondary(t *testing.T) {
	composer := NewRecommendationComposer(testConfig(), zap.NewNop())

	tests := []struct {
		name      string
		tmpl      *models.CardTemplate
		tier      models.ConfidenceTier
		wantCount int
		wantTypes []models.RecommendationType
	}{
		{
			name: "only INVESTIGATION and MONITORING included from secondary",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecInvestigation,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Order renal panel",
						ConfidenceTierRequired: models.TierPossible,
					},
					{
						RecType:                models.RecMonitoring,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Monitor creatinine",
						ConfidenceTierRequired: models.TierPossible,
					},
					{
						RecType:                models.RecReferral,
						Urgency:                models.UrgencyRoutine,
						ActionTextEn:           "Refer to nephrology",
						ConfidenceTierRequired: models.TierPossible,
					},
				},
			},
			tier:      models.TierPossible,
			wantCount: 2,
			wantTypes: []models.RecommendationType{models.RecInvestigation, models.RecMonitoring},
		},
		{
			name: "MEDICATION_HOLD from secondary excluded",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecMedicationHold,
						Urgency:                models.UrgencyUrgent,
						ActionTextEn:           "Hold metformin",
						ConfidenceTierRequired: models.TierFirm,
					},
				},
			},
			tier:      models.TierFirm,
			wantCount: 0,
			wantTypes: nil,
		},
		{
			name: "SAFETY_INSTRUCTION from secondary excluded",
			tmpl: &models.CardTemplate{
				Recommendations: []models.TemplateRecommendation{
					{
						RecType:                models.RecSafetyInstruction,
						Urgency:                models.UrgencyImmediate,
						ActionTextEn:           "Safety alert",
						BypassesConfidenceGate: true,
						ConfidenceTierRequired: models.TierFirm,
					},
				},
			},
			tier:      models.TierFirm,
			wantCount: 0,
			wantTypes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := composer.ComposeFromSecondary(tt.tmpl, tt.tier)
			if len(got) != tt.wantCount {
				t.Fatalf("ComposeFromSecondary() returned %d recommendations, want %d", len(got), tt.wantCount)
			}
			for i, wantType := range tt.wantTypes {
				if got[i].RecType != wantType {
					t.Errorf("recommendation[%d].RecType = %q, want %q", i, got[i].RecType, wantType)
				}
			}
			// Verify all results are marked as from secondary
			for i, rec := range got {
				if !rec.FromSecondaryDifferential {
					t.Errorf("recommendation[%d].FromSecondaryDifferential = false, want true", i)
				}
			}
		})
	}
}
