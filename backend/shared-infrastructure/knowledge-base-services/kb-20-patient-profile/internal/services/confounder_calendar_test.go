package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadTestCalendar loads a ConfounderCalendar from the market-configs directory.
func loadTestCalendar(t *testing.T, market string) *ConfounderCalendar {
	t.Helper()
	cal, err := LoadConfounderCalendar("../../../../market-configs", market)
	require.NoError(t, err)
	return cal
}

func TestCalendar_RamadanOverlap(t *testing.T) {
	cal := loadTestCalendar(t, "india")

	windowStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)

	factors := cal.FindActiveConfounders(windowStart, windowEnd, "MUSLIM", "MUMBAI")

	// Should find RAMADAN
	var ramadan *confounderFactorResult
	for i := range factors {
		if factors[i].Name == "RAMADAN" {
			ramadan = &factors[i]
			break
		}
	}
	require.NotNil(t, ramadan, "expected RAMADAN factor")
	assert.Equal(t, "RELIGIOUS_FASTING", string(ramadan.Category))
	assert.GreaterOrEqual(t, ramadan.OverlapDays, 28)
	assert.Greater(t, ramadan.Weight, 0.15)
	assert.Contains(t, ramadan.AffectedOutcomes, "DELTA_HBA1C")
}

func TestCalendar_RamadanWashout(t *testing.T) {
	cal := loadTestCalendar(t, "india")

	// Ramadan ends Mar 19 2026, washout 28d → extends to Apr 16.
	// Window Mar 20–Apr 20 should still detect RAMADAN via washout overlap.
	windowStart := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)

	factors := cal.FindActiveConfounders(windowStart, windowEnd, "MUSLIM", "MUMBAI")

	found := false
	for _, f := range factors {
		if f.Name == "RAMADAN" {
			found = true
			assert.Greater(t, f.OverlapDays, 0)
			break
		}
	}
	assert.True(t, found, "expected RAMADAN factor via washout overlap")
}

func TestCalendar_DiwaliSeason(t *testing.T) {
	cal := loadTestCalendar(t, "india")

	windowStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	factors := cal.FindActiveConfounders(windowStart, windowEnd, "HINDU", "MUMBAI")

	names := make(map[string]bool)
	for _, f := range factors {
		names[f.Name] = true
	}
	assert.True(t, names["DIWALI_SEASON"], "expected DIWALI_SEASON factor")
	assert.True(t, names["NAVRATRI_SHARAD"], "expected NAVRATRI_SHARAD factor")
}

func TestCalendar_MonsoonMumbai(t *testing.T) {
	cal := loadTestCalendar(t, "india")

	windowStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	// MUMBAI should find MONSOON_SEASON
	factors := cal.FindActiveConfounders(windowStart, windowEnd, "HINDU", "MUMBAI")
	found := false
	for _, f := range factors {
		if f.Name == "MONSOON_SEASON" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected MONSOON_SEASON for MUMBAI")

	// DELHI should NOT find MONSOON_SEASON
	factorsDelhi := cal.FindActiveConfounders(windowStart, windowEnd, "HINDU", "DELHI")
	foundDelhi := false
	for _, f := range factorsDelhi {
		if f.Name == "MONSOON_SEASON" {
			foundDelhi = true
			break
		}
	}
	assert.False(t, foundDelhi, "DELHI should NOT have MONSOON_SEASON")
}

func TestCalendar_NonMuslimSkipsRamadan(t *testing.T) {
	cal := loadTestCalendar(t, "india")

	windowStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)

	factors := cal.FindActiveConfounders(windowStart, windowEnd, "HINDU", "MUMBAI")

	for _, f := range factors {
		assert.NotEqual(t, "RAMADAN", f.Name, "HINDU should not get RAMADAN factor")
	}
}

func TestCalendar_AustraliaWinter(t *testing.T) {
	cal := loadTestCalendar(t, "australia")

	windowStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	factors := cal.FindActiveConfounders(windowStart, windowEnd, "ALL", "VIC")

	var winter *confounderFactorResult
	for i := range factors {
		if factors[i].Name == "WINTER_TEMPERATE" {
			winter = &factors[i]
			break
		}
	}
	require.NotNil(t, winter, "expected WINTER_TEMPERATE factor for VIC")
	assert.GreaterOrEqual(t, winter.Weight, 0.10)
}

func TestCalendar_NoOverlap_EmptyFactors(t *testing.T) {
	cal := loadTestCalendar(t, "india")

	// Mar 1-15 2026, HINDU, DELHI — should have no major confounders (weight >= 0.15)
	windowStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	factors := cal.FindActiveConfounders(windowStart, windowEnd, "HINDU", "DELHI")

	for _, f := range factors {
		assert.Less(t, f.Weight, 0.15, "expected no major confounders (weight >= 0.15), found %s with weight %.2f", f.Name, f.Weight)
	}
}

func TestCalendar_SeasonalHbA1cAdjustment(t *testing.T) {
	cal := loadTestCalendar(t, "australia")

	// Winter peak (July) → positive adjustment
	adj7 := cal.GetSeasonalHbA1cAdjustment(7)
	assert.Greater(t, adj7, 0.0, "July (winter peak) should have positive HbA1c adjustment")

	// Summer trough (January) → negative adjustment
	adj1 := cal.GetSeasonalHbA1cAdjustment(1)
	assert.Less(t, adj1, 0.0, "January (summer trough) should have negative HbA1c adjustment")
}
