// Package integration provides integration tests for KB-11 Population Health Engine.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/kb-11-population-health/internal/cohort"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Type Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortTypeValidation verifies cohort type constants and behavior.
func TestCohortTypeValidation(t *testing.T) {
	testCases := []struct {
		name     string
		cohort   models.CohortType
		expected string
	}{
		{"Static Cohort", models.CohortTypeStatic, "STATIC"},
		{"Dynamic Cohort", models.CohortTypeDynamic, "DYNAMIC"},
		{"Snapshot Cohort", models.CohortTypeSnapshot, "SNAPSHOT"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.cohort))
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Criterion Evaluation Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCriterionOperators verifies all criterion operators work correctly.
func TestCriterionOperators(t *testing.T) {
	testCases := []struct {
		name     string
		operator models.CriteriaOperator
		expected string
	}{
		{"Equals", models.OpEquals, "eq"},
		{"Not Equals", models.OpNotEquals, "neq"},
		{"Greater Than", models.OpGreaterThan, "gt"},
		{"Greater Than or Equal", models.OpGreaterEq, "gte"},
		{"Less Than", models.OpLessThan, "lt"},
		{"Less Than or Equal", models.OpLessEq, "lte"},
		{"In List", models.OpIn, "in"},
		{"Not In List", models.OpNotIn, "not_in"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.operator))
		})
	}
}

// TestCriterionValidation verifies criterion validation logic.
func TestCriterionValidation(t *testing.T) {
	t.Run("valid criterion with numeric value", func(t *testing.T) {
		criterion := cohort.Criterion{
			ID:       uuid.New(),
			Field:    "age",
			Operator: models.OpGreaterThan,
			Value:    65,
		}

		assert.NotEmpty(t, criterion.Field)
		assert.NotEmpty(t, criterion.Operator)
		assert.NotNil(t, criterion.Value)
	})

	t.Run("valid criterion with string value", func(t *testing.T) {
		criterion := cohort.Criterion{
			ID:       uuid.New(),
			Field:    "gender",
			Operator: models.OpEquals,
			Value:    "male",
		}

		assert.Equal(t, "gender", criterion.Field)
		assert.Equal(t, "male", criterion.Value)
	})

	t.Run("valid criterion with list value", func(t *testing.T) {
		criterion := cohort.Criterion{
			ID:       uuid.New(),
			Field:    "current_risk_tier",
			Operator: models.OpIn,
			Value:    []string{"HIGH", "VERY_HIGH"},
		}

		assert.Equal(t, "current_risk_tier", criterion.Field)
		assert.Equal(t, models.OpIn, criterion.Operator)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Creation Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortCreation verifies cohort object creation.
func TestCohortCreation(t *testing.T) {
	t.Run("create static cohort", func(t *testing.T) {
		cohortObj := cohort.NewStaticCohort("Test Static Cohort", "A test cohort", "test-user")

		assert.NotEqual(t, uuid.Nil, cohortObj.ID)
		assert.Equal(t, "Test Static Cohort", cohortObj.Name)
		assert.Equal(t, models.CohortTypeStatic, cohortObj.Type)
		assert.True(t, cohortObj.IsActive)
		assert.Empty(t, cohortObj.Criteria)
	})

	t.Run("create dynamic cohort with criteria", func(t *testing.T) {
		criteria := []cohort.Criterion{
			{
				ID:       uuid.New(),
				Field:    "current_risk_tier",
				Operator: models.OpIn,
				Value:    []string{"HIGH", "VERY_HIGH"},
			},
			{
				ID:       uuid.New(),
				Field:    "age",
				Operator: models.OpGreaterEq,
				Value:    65,
			},
		}
		cohortObj := cohort.NewDynamicCohort("High Risk Elderly", "Dynamic cohort", "test-user", criteria)

		assert.NotEqual(t, uuid.Nil, cohortObj.ID)
		assert.Equal(t, "High Risk Elderly", cohortObj.Name)
		assert.Equal(t, models.CohortTypeDynamic, cohortObj.Type)
		assert.True(t, cohortObj.IsActive)
		assert.NotEmpty(t, cohortObj.Criteria)
		assert.Len(t, cohortObj.Criteria, 2)
	})

	t.Run("create snapshot cohort", func(t *testing.T) {
		sourceCohort := cohort.NewStaticCohort("Source Cohort", "Source", "test-user")
		snapshotCohort := cohort.NewSnapshotCohort(sourceCohort, "test-user")

		assert.NotEqual(t, uuid.Nil, snapshotCohort.ID)
		assert.Equal(t, models.CohortTypeSnapshot, snapshotCohort.Type)
		assert.Equal(t, &sourceCohort.ID, snapshotCohort.SourceCohortID)
		assert.NotNil(t, snapshotCohort.SnapshotDate)
	})
}

// TestCohortTypeChecks verifies cohort type checking methods.
func TestCohortTypeChecks(t *testing.T) {
	t.Run("IsDynamic returns true for dynamic cohorts", func(t *testing.T) {
		cohortObj := cohort.NewDynamicCohort("Test", "Desc", "user", nil)
		assert.True(t, cohortObj.IsDynamic())
		assert.False(t, cohortObj.IsStatic())
		assert.False(t, cohortObj.IsSnapshot())
	})

	t.Run("IsStatic returns true for static cohorts", func(t *testing.T) {
		cohortObj := cohort.NewStaticCohort("Test", "Desc", "user")
		assert.True(t, cohortObj.IsStatic())
		assert.False(t, cohortObj.IsDynamic())
		assert.False(t, cohortObj.IsSnapshot())
	})

	t.Run("IsSnapshot returns true for snapshot cohorts", func(t *testing.T) {
		source := cohort.NewStaticCohort("Source", "Desc", "user")
		cohortObj := cohort.NewSnapshotCohort(source, "user")
		assert.True(t, cohortObj.IsSnapshot())
		assert.False(t, cohortObj.IsDynamic())
		assert.False(t, cohortObj.IsStatic())
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Member Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortMemberCreation verifies cohort member object creation.
func TestCohortMemberCreation(t *testing.T) {
	t.Run("create member with all fields", func(t *testing.T) {
		cohortID := uuid.New()
		patientID := uuid.New()
		member := &cohort.CohortMember{
			ID:            uuid.New(),
			CohortID:      cohortID,
			PatientID:     patientID,
			FHIRPatientID: "patient-123",
			JoinedAt:      time.Now().UTC(),
			IsActive:      true,
		}

		assert.NotEqual(t, uuid.Nil, member.ID)
		assert.Equal(t, cohortID, member.CohortID)
		assert.Equal(t, "patient-123", member.FHIRPatientID)
		assert.True(t, member.IsActive)
	})

	t.Run("member can be removed", func(t *testing.T) {
		now := time.Now().UTC()
		member := &cohort.CohortMember{
			ID:            uuid.New(),
			CohortID:      uuid.New(),
			PatientID:     uuid.New(),
			FHIRPatientID: "patient-456",
			JoinedAt:      now,
			RemovedAt:     &now,
			IsActive:      false,
		}

		assert.NotNil(t, member.RemovedAt)
		assert.False(t, member.IsActive)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Predefined Cohort Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPredefinedHighRiskCohortCriteria verifies high-risk cohort configuration.
func TestPredefinedHighRiskCohortCriteria(t *testing.T) {
	criteria := cohort.HighRiskCriteria()

	assert.Len(t, criteria, 1)
	assert.Equal(t, "current_risk_tier", criteria[0].Field)
	assert.Equal(t, models.OpIn, criteria[0].Operator)

	values := criteria[0].Value.([]string)
	assert.Contains(t, values, "HIGH")
	assert.Contains(t, values, "VERY_HIGH")
}

// TestPredefinedRisingRiskCohortCriteria verifies rising-risk cohort configuration.
func TestPredefinedRisingRiskCohortCriteria(t *testing.T) {
	criteria := cohort.RisingRiskCriteria()

	assert.Len(t, criteria, 1)
	assert.Equal(t, "current_risk_tier", criteria[0].Field)
	assert.Equal(t, models.OpEquals, criteria[0].Operator)
	assert.Equal(t, "RISING", criteria[0].Value)
}

// TestPredefinedCareGapCohortCriteria verifies care gap cohort configuration.
func TestPredefinedCareGapCohortCriteria(t *testing.T) {
	criteria := cohort.CareGapCriteria(3)

	assert.Len(t, criteria, 1)
	assert.Equal(t, "care_gap_count", criteria[0].Field)
	assert.Equal(t, models.OpGreaterEq, criteria[0].Operator)
	assert.Equal(t, 3, criteria[0].Value)
}

// TestPredefinedPCPCohortCriteria verifies PCP cohort configuration.
func TestPredefinedPCPCohortCriteria(t *testing.T) {
	criteria := cohort.PCPCriteria("dr-smith-123")

	assert.Len(t, criteria, 1)
	assert.Equal(t, "attributed_pcp", criteria[0].Field)
	assert.Equal(t, models.OpEquals, criteria[0].Operator)
	assert.Equal(t, "dr-smith-123", criteria[0].Value)
}

// TestPredefinedPracticeCohortCriteria verifies practice cohort configuration.
func TestPredefinedPracticeCohortCriteria(t *testing.T) {
	criteria := cohort.PracticeCriteria("cardiology-associates")

	assert.Len(t, criteria, 1)
	assert.Equal(t, "attributed_practice", criteria[0].Field)
	assert.Equal(t, models.OpEquals, criteria[0].Operator)
	assert.Equal(t, "cardiology-associates", criteria[0].Value)
}

// ──────────────────────────────────────────────────────────────────────────────
// Criterion Builder Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCriterionBuilder verifies the fluent criterion builder.
func TestCriterionBuilder(t *testing.T) {
	t.Run("build single criterion", func(t *testing.T) {
		criteria := cohort.NewCriterionBuilder().
			Where("age", models.OpGreaterThan, 65).
			Build()

		require.Len(t, criteria, 1)
		assert.Equal(t, "age", criteria[0].Field)
		assert.Equal(t, models.OpGreaterThan, criteria[0].Operator)
		assert.Equal(t, 65, criteria[0].Value)
	})

	t.Run("build multiple AND criteria", func(t *testing.T) {
		criteria := cohort.NewCriterionBuilder().
			Where("age", models.OpGreaterEq, 65).
			And("current_risk_tier", models.OpIn, []string{"HIGH", "VERY_HIGH"}).
			Build()

		require.Len(t, criteria, 2)
		assert.Equal(t, "AND", criteria[0].Logic)
	})

	t.Run("build criteria with OR", func(t *testing.T) {
		criteria := cohort.NewCriterionBuilder().
			Where("current_risk_tier", models.OpEquals, "HIGH").
			Or("current_risk_tier", models.OpEquals, "VERY_HIGH").
			Build()

		require.Len(t, criteria, 2)
		assert.Equal(t, "OR", criteria[0].Logic)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Refresh Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortRefreshResult verifies refresh result structure.
func TestCohortRefreshResult(t *testing.T) {
	t.Run("refresh result with changes", func(t *testing.T) {
		result := &cohort.CohortRefreshResult{
			CohortID:      uuid.New(),
			CohortName:    "High Risk Patients",
			PreviousCount: 100,
			NewCount:      115,
			Added:         20,
			Removed:       5,
			RefreshedAt:   time.Now().UTC(),
			Duration:      250 * time.Millisecond,
		}

		assert.Equal(t, 100, result.PreviousCount)
		assert.Equal(t, 115, result.NewCount)
		assert.Equal(t, 20, result.Added)
		assert.Equal(t, 5, result.Removed)
		assert.Equal(t, 15, result.NewCount-result.PreviousCount)
	})

	t.Run("refresh result with no changes", func(t *testing.T) {
		result := &cohort.CohortRefreshResult{
			CohortID:      uuid.New(),
			CohortName:    "Stable Cohort",
			PreviousCount: 100,
			NewCount:      100,
			Added:         0,
			Removed:       0,
			RefreshedAt:   time.Now().UTC(),
			Duration:      150 * time.Millisecond,
		}

		assert.Equal(t, result.PreviousCount, result.NewCount)
		assert.Zero(t, result.Added)
		assert.Zero(t, result.Removed)
	})
}

// TestCohortNeedsRefresh verifies refresh check logic.
func TestCohortNeedsRefresh(t *testing.T) {
	t.Run("dynamic cohort without refresh needs refresh", func(t *testing.T) {
		cohortObj := cohort.NewDynamicCohort("Test", "Desc", "user", nil)
		assert.True(t, cohortObj.NeedsRefresh(24*time.Hour))
	})

	t.Run("static cohort never needs refresh", func(t *testing.T) {
		cohortObj := cohort.NewStaticCohort("Test", "Desc", "user")
		assert.False(t, cohortObj.NeedsRefresh(24*time.Hour))
	})

	t.Run("recently refreshed cohort does not need refresh", func(t *testing.T) {
		cohortObj := cohort.NewDynamicCohort("Test", "Desc", "user", nil)
		now := time.Now()
		cohortObj.LastRefreshed = &now
		assert.False(t, cohortObj.NeedsRefresh(24*time.Hour))
	})

	t.Run("stale cohort needs refresh", func(t *testing.T) {
		cohortObj := cohort.NewDynamicCohort("Test", "Desc", "user", nil)
		oldTime := time.Now().Add(-48 * time.Hour)
		cohortObj.LastRefreshed = &oldTime
		assert.True(t, cohortObj.NeedsRefresh(24*time.Hour))
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Statistics Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortStats verifies cohort statistics structure.
func TestCohortStats(t *testing.T) {
	t.Run("stats with risk distribution", func(t *testing.T) {
		stats := &cohort.CohortStats{
			CohortID:     uuid.New(),
			CohortName:   "High Risk Patients",
			MemberCount:  500,
			RiskDistribution: map[models.RiskTier]int{
				models.RiskTierLow:      50,
				models.RiskTierModerate: 100,
				models.RiskTierHigh:     250,
				models.RiskTierVeryHigh: 100,
			},
			AverageRiskScore: 0.72,
			HighRiskCount:    350,
			CalculatedAt:     time.Now().UTC(),
		}

		assert.Equal(t, 500, stats.MemberCount)
		assert.Equal(t, 350, stats.HighRiskCount)
		assert.Len(t, stats.RiskDistribution, 4)
		assert.InDelta(t, 0.72, stats.AverageRiskScore, 0.01)
	})

	t.Run("stats by practice", func(t *testing.T) {
		stats := &cohort.CohortStats{
			CohortID:    uuid.New(),
			CohortName:  "All Patients",
			MemberCount: 1000,
			ByPractice: map[string]int{
				"cardiology":     400,
				"primary_care":   350,
				"endocrinology":  250,
			},
			CalculatedAt: time.Now().UTC(),
		}

		assert.Len(t, stats.ByPractice, 3)
		assert.Equal(t, 400, stats.ByPractice["cardiology"])
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Criteria Serialization Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCriteriaSerialization verifies criteria JSON serialization.
func TestCriteriaSerialization(t *testing.T) {
	_ = context.Background() // suppress unused import warning

	t.Run("save and load criteria", func(t *testing.T) {
		criteria := []cohort.Criterion{
			{
				ID:       uuid.New(),
				Field:    "age",
				Operator: models.OpGreaterThan,
				Value:    65,
				Logic:    "AND",
			},
			{
				ID:       uuid.New(),
				Field:    "current_risk_tier",
				Operator: models.OpIn,
				Value:    []interface{}{"HIGH", "VERY_HIGH"},
				Logic:    "AND",
			},
		}

		cohortObj := cohort.NewDynamicCohort("Test", "Desc", "user", criteria)

		// Criteria should be serialized
		assert.NotEmpty(t, cohortObj.CriteriaJSON)

		// Clear and reload
		cohortObj.Criteria = nil
		err := cohortObj.LoadCriteria()
		require.NoError(t, err)

		assert.Len(t, cohortObj.Criteria, 2)
		assert.Equal(t, "age", cohortObj.Criteria[0].Field)
	})

	t.Run("empty criteria serialization", func(t *testing.T) {
		cohortObj := cohort.NewStaticCohort("Test", "Desc", "user")

		err := cohortObj.SaveCriteria()
		require.NoError(t, err)
		assert.Nil(t, cohortObj.CriteriaJSON)

		err = cohortObj.LoadCriteria()
		require.NoError(t, err)
		assert.Empty(t, cohortObj.Criteria)
	})
}
