package measures

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS2 GOLDEN TESTS
// ============================================================================

func TestCMS2_GoldenTests(t *testing.T) {
	evaluator := NewCMS2Evaluator()

	for _, tc := range CMS2GoldenTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			// Build test context
			ctx := buildCMS2TestContext(tc)

			// Evaluate
			result := evaluator.Evaluate(ctx)

			// Verify Initial Population
			if result.InInitialPopulation != tc.ExpectedInPopulation {
				t.Errorf("InInitialPopulation: got %v, want %v (rationale: %s)",
					result.InInitialPopulation, tc.ExpectedInPopulation, result.Rationale)
			}

			// Verify Exclusion (only if in population)
			if tc.ExpectedInPopulation && tc.ExpectedExcluded {
				if !result.InDenominatorExclusion {
					t.Errorf("InDenominatorExclusion: got %v, want %v (rationale: %s)",
						result.InDenominatorExclusion, tc.ExpectedExcluded, result.Rationale)
				}
			}

			// Verify Numerator (only if in population and not excluded)
			if tc.ExpectedInPopulation && !tc.ExpectedExcluded {
				if result.InNumerator != tc.ExpectedInNumerator {
					t.Errorf("InNumerator: got %v, want %v (rationale: %s)",
						result.InNumerator, tc.ExpectedInNumerator, result.Rationale)
				}
			}

			// Verify Care Gap
			if result.CareGapIdentified != tc.ExpectedCareGap {
				t.Errorf("CareGapIdentified: got %v, want %v (rationale: %s)",
					result.CareGapIdentified, tc.ExpectedCareGap, result.Rationale)
			}

			// Verify audit metadata
			if result.MeasureVersion == "" {
				t.Error("MeasureVersion is empty - audit metadata required")
			}
		})
	}
}

// TestCMS2_SpecificScenarios tests the primary golden cases explicitly
func TestCMS2_SpecificScenarios(t *testing.T) {
	evaluator := NewCMS2Evaluator()

	t.Run("GOLDEN_1_ScreeningNegative_InNumerator", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Patient screened negative for depression → IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		now := time.Now()
		birthDate := now.AddDate(-45, 0, 0)

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-depression-negative",
					BirthDate: &birthDate,
					Gender:    "female",
				},
				RecentEncounters: []contracts.Encounter{
					{EncounterID: "enc-1", Status: "finished"},
				},
			},
			Knowledge: contracts.KnowledgeSnapshot{
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: map[string]bool{
						"depression_screening_negative": true,
					},
				},
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-cms2-1",
			},
		}

		result := evaluator.Evaluate(ctx)

		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if !result.InNumerator {
			t.Fatalf("Expected patient to be IN numerator (screening negative), got rationale: %s",
				result.Rationale)
		}
		if result.CareGapIdentified {
			t.Fatal("Expected NO care gap (screening negative)")
		}

		t.Logf("✅ GOLDEN TEST 1 PASSED: Screening negative → Numerator=%v, CareGap=%v",
			result.InNumerator, result.CareGapIdentified)
	})

	t.Run("GOLDEN_2_ScreeningPositiveWithFollowUp_InNumerator", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Patient screened positive with follow-up → IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		now := time.Now()
		birthDate := now.AddDate(-35, 0, 0)

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-depression-positive-followup",
					BirthDate: &birthDate,
					Gender:    "male",
				},
				RecentEncounters: []contracts.Encounter{
					{EncounterID: "enc-2", Status: "finished"},
				},
			},
			Knowledge: contracts.KnowledgeSnapshot{
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: map[string]bool{
						"depression_screening_positive": true,
						"has_depression_followup":       true,
					},
				},
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-cms2-2",
			},
		}

		result := evaluator.Evaluate(ctx)

		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if !result.InNumerator {
			t.Fatalf("Expected patient to be IN numerator (positive with follow-up), got rationale: %s",
				result.Rationale)
		}
		if result.CareGapIdentified {
			t.Fatal("Expected NO care gap (positive with follow-up)")
		}

		t.Logf("✅ GOLDEN TEST 2 PASSED: Positive with follow-up → Numerator=%v, CareGap=%v",
			result.InNumerator, result.CareGapIdentified)
	})

	t.Run("CareGap_ScreeningPositiveNoFollowUp", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Patient screened positive without follow-up → CARE GAP
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		now := time.Now()
		birthDate := now.AddDate(-40, 0, 0)

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-depression-positive-no-followup",
					BirthDate: &birthDate,
					Gender:    "female",
				},
				RecentEncounters: []contracts.Encounter{
					{EncounterID: "enc-3", Status: "finished"},
				},
			},
			Knowledge: contracts.KnowledgeSnapshot{
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: map[string]bool{
						"depression_screening_positive": true,
						// No has_depression_followup
					},
				},
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-cms2-3",
			},
		}

		result := evaluator.Evaluate(ctx)

		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if result.InNumerator {
			t.Fatalf("Expected patient to NOT be in numerator (positive without follow-up), got rationale: %s",
				result.Rationale)
		}
		if !result.CareGapIdentified {
			t.Fatal("Expected care gap to be identified (positive without follow-up)")
		}

		t.Logf("✅ CARE GAP TEST PASSED: Positive no follow-up → Numerator=%v, CareGap=%v",
			result.InNumerator, result.CareGapIdentified)
	})
}

// TestCMS2_AuditMetadata verifies all audit fields are populated
func TestCMS2_AuditMetadata(t *testing.T) {
	evaluator := NewCMS2Evaluator()

	now := time.Now()
	birthDate := now.AddDate(-30, 0, 0)

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "audit-test-depression",
				BirthDate: &birthDate,
				Gender:    "female",
			},
			RecentEncounters: []contracts.Encounter{
				{EncounterID: "enc-audit", Status: "finished"},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"depression_screening_negative": true,
				},
			},
		},
	}

	result := evaluator.Evaluate(ctx)

	t.Run("MeasureID", func(t *testing.T) {
		if result.MeasureID != "CMS2" {
			t.Errorf("MeasureID: got %q, want %q", result.MeasureID, "CMS2")
		}
	})

	t.Run("MeasureVersion", func(t *testing.T) {
		if result.MeasureVersion != "2024.0.0" {
			t.Errorf("MeasureVersion: got %q, want %q", result.MeasureVersion, "2024.0.0")
		}
	})

	t.Run("ELMCorrespondence", func(t *testing.T) {
		expected := "PCSDepressionScreenAndFollowUpFHIR:0.2.000"
		if result.ELMCorrespondence != expected {
			t.Errorf("ELMCorrespondence: got %q, want %q", result.ELMCorrespondence, expected)
		}
	})

	t.Run("EvaluatedAt", func(t *testing.T) {
		if result.EvaluatedAt.IsZero() {
			t.Error("EvaluatedAt is zero")
		}
	})

	t.Run("Rationale", func(t *testing.T) {
		if result.Rationale == "" {
			t.Error("Rationale is empty")
		}
	})
}

// buildCMS2TestContext creates a test context from a test case
func buildCMS2TestContext(tc CMS2TestCase) *contracts.ClinicalExecutionContext {
	now := time.Now()
	birthDate := now.AddDate(-tc.Age, 0, 0)

	// Build ValueSet memberships based on test case
	memberships := map[string]bool{
		"has_bipolar_disorder": tc.HasBipolarDisorder,
		"has_depression_followup": tc.HasFollowUp,
	}

	// Set screening result flag
	switch tc.ScreeningResult {
	case ScreeningNegative:
		memberships["depression_screening_negative"] = true
	case ScreeningPositive:
		memberships["depression_screening_positive"] = true
	}

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "test-" + tc.Name,
				BirthDate: &birthDate,
				Gender:    "female",
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: memberships,
			},
			SnapshotTimestamp: now,
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID: "test-request-" + tc.Name,
		},
	}

	// Add qualifying encounter if needed
	if tc.HasQualifyingEncounter {
		ctx.Patient.RecentEncounters = []contracts.Encounter{
			{EncounterID: "encounter-test", Status: "finished"},
		}
	}

	return ctx
}
