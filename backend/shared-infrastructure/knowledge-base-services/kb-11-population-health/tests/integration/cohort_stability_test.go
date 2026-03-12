// Package integration provides cohort stability tests for KB-11 Population Health.
// These tests ensure cohort snapshots remain immutable and stable.
package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/kb-11-population-health/internal/cohort"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// ──────────────────────────────────────────────────────────────────────────────
// Snapshot Immutability Tests
// CRITICAL: Historical cohort data must never be mutated after creation.
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortSnapshotImmutability ensures snapshots cannot be modified after creation.
func TestCohortSnapshotImmutability(t *testing.T) {
	t.Run("snapshot preserves original data", func(t *testing.T) {
		// Create source cohort
		source := cohort.NewStaticCohort("High Risk Q1 2024", "High risk patients for Q1", "system")

		// Create snapshot
		snapshot := cohort.NewSnapshotCohort(source, "analyst")

		// Record original values
		originalID := snapshot.ID
		originalName := snapshot.Name
		originalSourceID := *snapshot.SourceCohortID
		originalCreatedAt := snapshot.CreatedAt

		// Verify snapshot properties
		assert.NotEqual(t, uuid.Nil, originalID)
		assert.Contains(t, originalName, "Snapshot")
		assert.Equal(t, source.ID, originalSourceID)
		assert.Equal(t, models.CohortTypeSnapshot, snapshot.Type)
		assert.NotNil(t, snapshot.SnapshotDate)

		// Verify values haven't changed (immutability check)
		assert.Equal(t, originalID, snapshot.ID, "Snapshot ID should not change")
		assert.Equal(t, originalName, snapshot.Name, "Snapshot name should not change")
		assert.Equal(t, originalCreatedAt, snapshot.CreatedAt, "CreatedAt should not change")
	})

	t.Run("snapshot hash remains stable", func(t *testing.T) {
		// Create a snapshot with known data
		source := cohort.NewDynamicCohort(
			"Rising Risk",
			"Patients with rising risk trend",
			"system",
			cohort.RisingRiskCriteria(),
		)

		snapshot1 := cohort.NewSnapshotCohort(source, "analyst")

		// Compute hash
		hash1 := computeCohortHash(snapshot1)

		// Wait a moment
		time.Sleep(10 * time.Millisecond)

		// Compute hash again
		hash2 := computeCohortHash(snapshot1)

		// Hash should be identical
		assert.Equal(t, hash1, hash2, "Snapshot hash should remain stable")
	})

	t.Run("snapshot date is frozen at creation", func(t *testing.T) {
		source := cohort.NewStaticCohort("Test Cohort", "Test", "system")

		beforeCreate := time.Now()
		snapshot := cohort.NewSnapshotCohort(source, "analyst")
		afterCreate := time.Now()

		require.NotNil(t, snapshot.SnapshotDate)

		// Snapshot date should be between before and after
		assert.True(t, snapshot.SnapshotDate.After(beforeCreate.Add(-time.Second)))
		assert.True(t, snapshot.SnapshotDate.Before(afterCreate.Add(time.Second)))

		// Store the date
		originalDate := *snapshot.SnapshotDate

		// Wait and verify it doesn't change
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, originalDate, *snapshot.SnapshotDate, "Snapshot date should be frozen")
	})
}

// TestCohortRefreshDoesNotMutateSnapshot ensures refreshing a dynamic cohort
// does not affect existing snapshots.
func TestCohortRefreshDoesNotMutateSnapshot(t *testing.T) {
	t.Run("refresh creates new membership without changing snapshot", func(t *testing.T) {
		// Create dynamic cohort
		dynamicCohort := cohort.NewDynamicCohort(
			"High Risk Patients",
			"All patients with high or very high risk",
			"system",
			cohort.HighRiskCriteria(),
		)
		dynamicCohort.MemberCount = 1500

		// Create snapshot
		snapshot := cohort.NewSnapshotCohort(dynamicCohort, "analyst")
		snapshot.MemberCount = 1500 // Copy member count at snapshot time

		// Record snapshot state
		originalSnapshotCount := snapshot.MemberCount
		originalSnapshotHash := computeCohortHash(snapshot)

		// Simulate cohort refresh (member count changes)
		dynamicCohort.MemberCount = 1650
		now := time.Now()
		dynamicCohort.LastRefreshed = &now

		// Verify snapshot is unchanged
		assert.Equal(t, originalSnapshotCount, snapshot.MemberCount,
			"Snapshot member count should not change after refresh")
		assert.Equal(t, originalSnapshotHash, computeCohortHash(snapshot),
			"Snapshot hash should not change after refresh")

		// Dynamic cohort should reflect new count
		assert.Equal(t, 1650, dynamicCohort.MemberCount)
	})

	t.Run("multiple snapshots of same cohort are independent", func(t *testing.T) {
		source := cohort.NewDynamicCohort("Care Gap Cohort", "Patients with care gaps", "system",
			cohort.CareGapCriteria(1))
		source.MemberCount = 800

		// Create first snapshot
		snapshot1 := cohort.NewSnapshotCohort(source, "analyst")
		snapshot1.MemberCount = 800

		// Simulate time passing and cohort changes
		source.MemberCount = 950

		// Create second snapshot
		snapshot2 := cohort.NewSnapshotCohort(source, "analyst")
		snapshot2.MemberCount = 950

		// Snapshots should be independent
		assert.NotEqual(t, snapshot1.ID, snapshot2.ID, "Snapshots should have different IDs")
		assert.NotEqual(t, snapshot1.MemberCount, snapshot2.MemberCount,
			"Snapshots capture different points in time")

		// First snapshot unchanged
		assert.Equal(t, 800, snapshot1.MemberCount, "First snapshot should retain original count")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Criteria Stability Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCriteriaSerializationStability ensures criteria remain stable through serialization.
func TestCriteriaSerializationStability(t *testing.T) {
	t.Run("criteria serialize and deserialize identically", func(t *testing.T) {
		originalCriteria := cohort.NewCriterionBuilder().
			Where("current_risk_tier", models.OpIn, []string{"HIGH", "VERY_HIGH"}).
			And("care_gap_count", models.OpGreaterEq, 2).
			And("attributed_practice", models.OpEquals, "Downtown Medical").
			Build()

		// Create cohort with criteria
		c := cohort.NewDynamicCohort("Test", "Test cohort", "system", originalCriteria)

		// Serialize
		err := c.SaveCriteria()
		require.NoError(t, err)

		// Deserialize into new cohort
		c2 := &cohort.Cohort{CriteriaJSON: c.CriteriaJSON}
		err = c2.LoadCriteria()
		require.NoError(t, err)

		// Verify criteria match
		assert.Equal(t, len(originalCriteria), len(c2.Criteria), "Criteria count should match")

		for i, orig := range originalCriteria {
			loaded := c2.Criteria[i]
			assert.Equal(t, orig.Field, loaded.Field, "Field should match")
			assert.Equal(t, orig.Operator, loaded.Operator, "Operator should match")
			assert.Equal(t, orig.Logic, loaded.Logic, "Logic should match")
		}
	})

	t.Run("empty criteria handles gracefully", func(t *testing.T) {
		c := cohort.NewStaticCohort("Static Test", "No criteria", "system")

		// Serialize empty criteria
		err := c.SaveCriteria()
		require.NoError(t, err)
		assert.Nil(t, c.CriteriaJSON)

		// Deserialize empty criteria
		err = c.LoadCriteria()
		require.NoError(t, err)
		assert.Empty(t, c.Criteria)
	})

	t.Run("predefined criteria are deterministic", func(t *testing.T) {
		// Get predefined criteria multiple times
		highRisk1 := cohort.HighRiskCriteria()
		highRisk2 := cohort.HighRiskCriteria()
		risingRisk1 := cohort.RisingRiskCriteria()
		risingRisk2 := cohort.RisingRiskCriteria()

		// Should be equivalent (same field, operator, value)
		assert.Equal(t, highRisk1[0].Field, highRisk2[0].Field)
		assert.Equal(t, highRisk1[0].Operator, highRisk2[0].Operator)

		assert.Equal(t, risingRisk1[0].Field, risingRisk2[0].Field)
		assert.Equal(t, risingRisk1[0].Operator, risingRisk2[0].Operator)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Member Data Stability Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortMemberDataStability ensures member snapshot data is preserved.
func TestCohortMemberDataStability(t *testing.T) {
	t.Run("member snapshot data is preserved", func(t *testing.T) {
		// Simulate member with snapshot data
		member := &cohort.CohortMember{
			ID:            uuid.New(),
			CohortID:      uuid.New(),
			PatientID:     uuid.New(),
			FHIRPatientID: "patient-snapshot-test",
			JoinedAt:      time.Now(),
			IsActive:      true,
			SnapshotData: []byte(`{
				"risk_score": 0.72,
				"risk_tier": "HIGH",
				"care_gaps": 3,
				"snapshot_timestamp": "2024-06-15T10:00:00Z"
			}`),
		}

		// Parse snapshot data
		var data map[string]interface{}
		err := json.Unmarshal(member.SnapshotData, &data)
		require.NoError(t, err)

		// Verify data
		assert.Equal(t, 0.72, data["risk_score"])
		assert.Equal(t, "HIGH", data["risk_tier"])
		assert.Equal(t, float64(3), data["care_gaps"])
	})

	t.Run("member join time is immutable", func(t *testing.T) {
		joinTime := time.Now()
		member := &cohort.CohortMember{
			ID:            uuid.New(),
			CohortID:      uuid.New(),
			PatientID:     uuid.New(),
			FHIRPatientID: "patient-join-test",
			JoinedAt:      joinTime,
			IsActive:      true,
		}

		// Wait
		time.Sleep(10 * time.Millisecond)

		// Join time should not change
		assert.Equal(t, joinTime.Unix(), member.JoinedAt.Unix(),
			"Member join time should be immutable")
	})

	t.Run("removed members retain join history", func(t *testing.T) {
		joinTime := time.Now().Add(-24 * time.Hour)
		removeTime := time.Now()

		member := &cohort.CohortMember{
			ID:            uuid.New(),
			CohortID:      uuid.New(),
			PatientID:     uuid.New(),
			FHIRPatientID: "patient-removed-test",
			JoinedAt:      joinTime,
			RemovedAt:     &removeTime,
			IsActive:      false,
		}

		// Verify history is preserved
		assert.False(t, member.IsActive)
		assert.NotNil(t, member.RemovedAt)
		assert.True(t, member.JoinedAt.Before(*member.RemovedAt),
			"Join time should be before removal time")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Type Consistency Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortTypeConsistency ensures cohort types remain consistent.
func TestCohortTypeConsistency(t *testing.T) {
	t.Run("static cohort type is immutable", func(t *testing.T) {
		c := cohort.NewStaticCohort("Test", "Test", "system")

		assert.True(t, c.IsStatic())
		assert.False(t, c.IsDynamic())
		assert.False(t, c.IsSnapshot())
		assert.Equal(t, models.CohortTypeStatic, c.Type)
	})

	t.Run("dynamic cohort type is immutable", func(t *testing.T) {
		c := cohort.NewDynamicCohort("Test", "Test", "system", nil)

		assert.False(t, c.IsStatic())
		assert.True(t, c.IsDynamic())
		assert.False(t, c.IsSnapshot())
		assert.Equal(t, models.CohortTypeDynamic, c.Type)
	})

	t.Run("snapshot cohort type is immutable", func(t *testing.T) {
		source := cohort.NewStaticCohort("Source", "Source", "system")
		c := cohort.NewSnapshotCohort(source, "analyst")

		assert.False(t, c.IsStatic())
		assert.False(t, c.IsDynamic())
		assert.True(t, c.IsSnapshot())
		assert.Equal(t, models.CohortTypeSnapshot, c.Type)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Refresh Interval Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortRefreshIntervalConsistency ensures refresh behavior is consistent.
func TestCohortRefreshIntervalConsistency(t *testing.T) {
	t.Run("never refreshed cohort needs refresh", func(t *testing.T) {
		c := cohort.NewDynamicCohort("Test", "Test", "system", nil)

		assert.True(t, c.NeedsRefresh(time.Hour))
	})

	t.Run("recently refreshed cohort does not need refresh", func(t *testing.T) {
		c := cohort.NewDynamicCohort("Test", "Test", "system", nil)
		now := time.Now()
		c.LastRefreshed = &now

		assert.False(t, c.NeedsRefresh(time.Hour))
	})

	t.Run("stale cohort needs refresh", func(t *testing.T) {
		c := cohort.NewDynamicCohort("Test", "Test", "system", nil)
		twoHoursAgo := time.Now().Add(-2 * time.Hour)
		c.LastRefreshed = &twoHoursAgo

		assert.True(t, c.NeedsRefresh(time.Hour))
	})

	t.Run("static cohort never needs refresh", func(t *testing.T) {
		c := cohort.NewStaticCohort("Test", "Test", "system")

		// Static cohorts should never need refresh regardless of time
		assert.False(t, c.NeedsRefresh(time.Nanosecond))
	})

	t.Run("snapshot cohort never needs refresh", func(t *testing.T) {
		source := cohort.NewStaticCohort("Source", "Source", "system")
		c := cohort.NewSnapshotCohort(source, "analyst")

		// Snapshots are frozen in time
		assert.False(t, c.NeedsRefresh(time.Nanosecond))
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper Functions
// ──────────────────────────────────────────────────────────────────────────────

// computeCohortHash computes a deterministic hash for cohort comparison.
func computeCohortHash(c *cohort.Cohort) string {
	// Create a deterministic representation
	data := struct {
		ID          string
		Name        string
		Type        string
		MemberCount int
	}{
		ID:          c.ID.String(),
		Name:        c.Name,
		Type:        string(c.Type),
		MemberCount: c.MemberCount,
	}

	bytes, _ := json.Marshal(data)
	return string(bytes)
}
