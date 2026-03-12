package tests

import (
	"testing"
	"time"

	"kb-13-quality-measures/internal/period"
)

func TestResolver_ResolveForYear(t *testing.T) {
	resolver := period.NewResolver()
	mp := resolver.ResolveForYear(2024)

	expectedStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	if !mp.Start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, mp.Start)
	}

	// End date should be December 31
	if mp.End.Month() != time.December || mp.End.Day() != 31 {
		t.Errorf("Expected end December 31, got %v", mp.End)
	}

	// Check label
	if mp.Label != "CY2024" {
		t.Errorf("Expected label CY2024, got %s", mp.Label)
	}
}

func TestMeasurementPeriod_IsInPeriod(t *testing.T) {
	resolver := period.NewResolver()
	mp := resolver.ResolveForYear(2024)

	// Date within period
	withinDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	if !mp.IsInPeriod(withinDate) {
		t.Error("Expected date 2024-06-15 to be within 2024 calendar year")
	}

	// Date before period
	beforeDate := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	if mp.IsInPeriod(beforeDate) {
		t.Error("Expected date 2023-12-31 to be outside 2024 calendar year")
	}

	// Date after period
	afterDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if mp.IsInPeriod(afterDate) {
		t.Error("Expected date 2025-01-01 to be outside 2024 calendar year")
	}
}

func TestMeasurementPeriod_Duration(t *testing.T) {
	resolver := period.NewResolver()
	mp := resolver.ResolveForYear(2024)

	// 2024 is a leap year, so 366 days
	actualDays := mp.Days()

	// Allow for some rounding due to nanosecond precision
	if actualDays < 365 || actualDays > 367 {
		t.Errorf("Expected around 366 days (leap year), got %d", actualDays)
	}
}

func TestResolver_ResolveForQuarter(t *testing.T) {
	resolver := period.NewResolver()

	testCases := []struct {
		quarter       int
		expectedStart time.Month
		expectedEnd   time.Month
	}{
		{1, time.January, time.March},
		{2, time.April, time.June},
		{3, time.July, time.September},
		{4, time.October, time.December},
	}

	for _, tc := range testCases {
		mp, err := resolver.ResolveForQuarter(2024, tc.quarter)
		if err != nil {
			t.Errorf("Q%d: unexpected error: %v", tc.quarter, err)
			continue
		}

		if mp.Start.Month() != tc.expectedStart {
			t.Errorf("Q%d: expected start month %v, got %v", tc.quarter, tc.expectedStart, mp.Start.Month())
		}

		if mp.End.Month() != tc.expectedEnd {
			t.Errorf("Q%d: expected end month %v, got %v", tc.quarter, tc.expectedEnd, mp.End.Month())
		}
	}
}

func TestResolver_ResolveForQuarter_Invalid(t *testing.T) {
	resolver := period.NewResolver()

	_, err := resolver.ResolveForQuarter(2024, 0)
	if err == nil {
		t.Error("Expected error for invalid quarter 0")
	}

	_, err = resolver.ResolveForQuarter(2024, 5)
	if err == nil {
		t.Error("Expected error for invalid quarter 5")
	}
}

func TestResolver_ResolveForMonth(t *testing.T) {
	resolver := period.NewResolver()
	mp := resolver.ResolveForMonth(2024, time.February)

	// February 2024 is a leap year
	if mp.Start.Day() != 1 {
		t.Errorf("Expected start day 1, got %d", mp.Start.Day())
	}

	if mp.End.Day() != 29 {
		t.Errorf("Expected end day 29 (leap year), got %d", mp.End.Day())
	}
}

func TestResolver_ResolveRolling12Months(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	resolver := period.NewResolverWithReference(refTime)

	mp := resolver.ResolveRolling12Months()

	// Start should be 12 months before reference
	expectedStart := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	if !mp.Start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, mp.Start)
	}

	if !mp.End.Equal(refTime) {
		t.Errorf("Expected end %v, got %v", refTime, mp.End)
	}

	if mp.Type != period.PeriodTypeRolling {
		t.Errorf("Expected type rolling, got %s", mp.Type)
	}
}

func TestResolver_CurrentCalendarYear(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	resolver := period.NewResolverWithReference(refTime)

	mp := resolver.CurrentCalendarYear()

	if mp.Start.Year() != 2024 || mp.Start.Month() != time.January || mp.Start.Day() != 1 {
		t.Errorf("Expected start Jan 1 2024, got %v", mp.Start)
	}

	if mp.End.Year() != 2024 || mp.End.Month() != time.December || mp.End.Day() != 31 {
		t.Errorf("Expected end Dec 31 2024, got %v", mp.End)
	}
}

func TestResolver_PreviousCalendarYear(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	resolver := period.NewResolverWithReference(refTime)

	mp := resolver.PreviousCalendarYear()

	if mp.Start.Year() != 2023 {
		t.Errorf("Expected year 2023, got %d", mp.Start.Year())
	}

	if mp.Label != "CY2023" {
		t.Errorf("Expected label CY2023, got %s", mp.Label)
	}
}

func TestResolver_Resolve_Calendar(t *testing.T) {
	resolver := period.NewResolverWithReference(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))

	cfg := period.Config{
		Type:     period.PeriodTypeCalendar,
		Duration: "P1Y",
		Anchor:   period.AnchorYear,
	}

	mp, err := resolver.Resolve(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mp.Type != period.PeriodTypeCalendar {
		t.Errorf("Expected calendar type, got %s", mp.Type)
	}
}

func TestResolver_Resolve_Rolling(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	resolver := period.NewResolverWithReference(refTime)

	cfg := period.Config{
		Type:     period.PeriodTypeRolling,
		Duration: "P6M",
	}

	mp, err := resolver.Resolve(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mp.Type != period.PeriodTypeRolling {
		t.Errorf("Expected rolling type, got %s", mp.Type)
	}

	// End should be reference time
	if !mp.End.Equal(refTime) {
		t.Errorf("Expected end %v, got %v", refTime, mp.End)
	}
}

func TestResolver_Resolve_InvalidType(t *testing.T) {
	resolver := period.NewResolver()

	cfg := period.Config{
		Type:     "invalid",
		Duration: "P1Y",
	}

	_, err := resolver.Resolve(cfg)
	if err == nil {
		t.Error("Expected error for invalid period type")
	}
}

func TestResolver_Resolve_InvalidDuration(t *testing.T) {
	resolver := period.NewResolver()

	cfg := period.Config{
		Type:     period.PeriodTypeCalendar,
		Duration: "invalid",
	}

	_, err := resolver.Resolve(cfg)
	if err == nil {
		t.Error("Expected error for invalid duration")
	}
}

func TestMeasurementPeriod_Fields(t *testing.T) {
	resolver := period.NewResolver()
	mp := resolver.ResolveForYear(2024)

	// Verify all required fields are set
	if mp.Start.IsZero() {
		t.Error("Expected non-zero start time")
	}
	if mp.End.IsZero() {
		t.Error("Expected non-zero end time")
	}
	if mp.Type == "" {
		t.Error("Expected non-empty type")
	}
	if mp.Label == "" {
		t.Error("Expected non-empty label")
	}
}
