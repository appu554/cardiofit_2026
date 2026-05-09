package capacity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/ethics/consent_extension"
	"github.com/cardiofit/shared/v2_substrate/ethics/vulnerability"
)

// fakeSource is a table-driven test double for CapacitySource.
// All branches are controlled by struct fields so the table tests stay
// declarative.
type fakeSource struct {
	assessment    vulnerability.Assessment
	assessmentErr error
	consent       *consent_extension.RestrictivePracticeConsent
	consentErr    error
}

func (f *fakeSource) AssessmentFor(_ context.Context, _ uuid.UUID) (vulnerability.Assessment, error) {
	return f.assessment, f.assessmentErr
}

func (f *fakeSource) RestrictivePracticeConsentFor(_ context.Context, _ uuid.UUID,
	_ consent_extension.PracticeType) (*consent_extension.RestrictivePracticeConsent, error) {
	return f.consent, f.consentErr
}

// activeConsent returns a RestrictivePracticeConsent that Allows(asOf) for an
// assessment performed at time t.
func activeConsent(t time.Time) *consent_extension.RestrictivePracticeConsent {
	return &consent_extension.RestrictivePracticeConsent{
		ID:                                    uuid.New(),
		ConsentID:                             uuid.New(),
		PracticeType:                          consent_extension.PracticeChemicalRestraint,
		Status:                                "active",
		LessRestrictiveAlternativesDocumented: true,
		GrantedAt:                             t.Add(-1 * time.Hour),
		MaxDuration:                           7 * 24 * time.Hour, // 1 week
		DesignatedPractitionerID:              uuid.New(),
		MandatoryReviewDueAt:                  t.Add(7 * 24 * time.Hour),
	}
}

// expiredConsent returns a record whose Allows() returns false (status not
// active, simulating a revoked / pending record).
func expiredConsent() *consent_extension.RestrictivePracticeConsent {
	return &consent_extension.RestrictivePracticeConsent{
		ID:                                    uuid.New(),
		Status:                                "expired",
		LessRestrictiveAlternativesDocumented: true,
		GrantedAt:                             time.Now().Add(-30 * 24 * time.Hour),
		MaxDuration:                           1 * time.Hour,
	}
}

func TestGate_Evaluate(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)

	srcAssessmentErr := errors.New("source: assessment fetch failed")
	srcConsentErr := errors.New("source: consent fetch failed")

	tests := []struct {
		name            string
		assessment      vulnerability.Assessment
		assessmentErr   error
		consent         *consent_extension.RestrictivePracticeConsent
		consentErr      error
		restrictiveType consent_extension.PracticeType
		wantErr         error
		wantErrContains string // for source-error propagation
	}{
		{
			name: "capacity_intact_non_restrictive_proceeds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityIntact,
				AssessedAt:        now,
			},
			restrictiveType: "",
			wantErr:         nil,
		},
		{
			name: "capacity_intact_restrictive_active_consent_proceeds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityIntact,
				AssessedAt:        now,
			},
			consent:         activeConsent(now),
			restrictiveType: consent_extension.PracticeChemicalRestraint,
			wantErr:         nil,
		},
		{
			name: "capacity_intact_restrictive_nil_consent_holds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityIntact,
				AssessedAt:        now,
			},
			consent:         nil,
			restrictiveType: consent_extension.PracticeChemicalRestraint,
			wantErr:         ErrRestrictivePracticeNoConsent,
		},
		{
			name: "capacity_intact_restrictive_consent_not_allowed_holds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityIntact,
				AssessedAt:        now,
			},
			consent:         expiredConsent(),
			restrictiveType: consent_extension.PracticeChemicalRestraint,
			wantErr:         ErrRestrictivePracticeNoConsent,
		},
		{
			name: "capacity_uncertain_no_sdm_holds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityUncertain,
				SDMRequired:       false,
				AssessedAt:        now,
			},
			wantErr: ErrSDMRequired,
		},
		{
			name: "capacity_uncertain_sdm_in_place_proceeds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityUncertain,
				SDMRequired:       true,
				AssessedAt:        now,
			},
			wantErr: nil,
		},
		{
			name: "capacity_moderate_no_sdm_holds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityModerateImpairment,
				SDMRequired:       false,
				AssessedAt:        now,
			},
			wantErr: ErrSDMRequired,
		},
		{
			name: "capacity_severe_no_sdm_holds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacitySevereImpairment,
				SDMRequired:       false,
				AssessedAt:        now,
			},
			wantErr: ErrSDMRequired,
		},
		{
			name: "capacity_mild_no_sdm_proceeds",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityMildImpairment,
				SDMRequired:       false,
				AssessedAt:        now,
			},
			wantErr: nil,
		},
		{
			name:            "source_assessment_error_propagates",
			assessmentErr:   srcAssessmentErr,
			wantErrContains: "assessment fetch failed",
		},
		{
			name: "source_consent_error_propagates",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityIntact,
				AssessedAt:        now,
			},
			consentErr:      srcConsentErr,
			restrictiveType: consent_extension.PracticeChemicalRestraint,
			wantErrContains: "consent fetch failed",
		},
		{
			name: "uncertain_sdm_in_place_restrictive_no_consent_holds_on_consent",
			assessment: vulnerability.Assessment{
				CognitiveCapacity: vulnerability.CapacityUncertain,
				SDMRequired:       true,
				AssessedAt:        now,
			},
			consent:         nil,
			restrictiveType: consent_extension.PracticeChemicalRestraint,
			wantErr:         ErrRestrictivePracticeNoConsent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := &fakeSource{
				assessment:    tc.assessment,
				assessmentErr: tc.assessmentErr,
				consent:       tc.consent,
				consentErr:    tc.consentErr,
			}
			gate := NewGate(src)

			err := gate.Evaluate(context.Background(), uuid.New(), tc.restrictiveType)

			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
				}
				if got := err.Error(); !strings.Contains(got, tc.wantErrContains) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErrContains, got)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err=%v, got %v", tc.wantErr, err)
			}
		})
	}
}
