// Package period provides measurement period resolution for KB-13 Quality Measures.
//
// 🔴 CRITICAL: All date logic MUST go through this module (CTO/CMO Gate Requirement)
//
// This module is the SINGLE SOURCE OF TRUTH for:
//   - Measurement period start/end dates
//   - Calendar year alignment
//   - Rolling period calculations
//   - Anchor date handling
//
// Rationale: Centralizing date logic ensures:
//   - Consistent period boundaries across all calculations
//   - Audit defensibility ("why was this patient included/excluded?")
//   - Proper handling of edge cases (leap years, year boundaries)
//   - Reproducible calculations
package period

import (
	"fmt"
	"time"
)

// PeriodType defines how measurement periods are calculated.
type PeriodType string

const (
	// PeriodTypeCalendar aligns to calendar boundaries (year, quarter, month).
	PeriodTypeCalendar PeriodType = "calendar"
	// PeriodTypeRolling uses a rolling window from a reference date.
	PeriodTypeRolling PeriodType = "rolling"
)

// AnchorType defines the calendar anchor for period alignment.
type AnchorType string

const (
	AnchorYear    AnchorType = "year"
	AnchorQuarter AnchorType = "quarter"
	AnchorMonth   AnchorType = "month"
)

// MeasurementPeriod represents a resolved measurement period.
type MeasurementPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Type  PeriodType `json:"type"`
	Label string    `json:"label"` // Human-readable label, e.g., "CY2024"
}

// Resolver resolves measurement periods based on configuration.
// 🔴 This is the SINGLE SOURCE OF TRUTH for all period calculations.
type Resolver struct {
	// Reference time for calculations (defaults to now)
	referenceTime time.Time
}

// NewResolver creates a new period resolver.
func NewResolver() *Resolver {
	return &Resolver{
		referenceTime: time.Now(),
	}
}

// NewResolverWithReference creates a resolver with a specific reference time.
// Use this for testing or for calculating historical periods.
func NewResolverWithReference(ref time.Time) *Resolver {
	return &Resolver{
		referenceTime: ref,
	}
}

// SetReferenceTime updates the reference time for calculations.
func (r *Resolver) SetReferenceTime(t time.Time) {
	r.referenceTime = t
}

// GetReferenceTime returns the current reference time.
func (r *Resolver) GetReferenceTime() time.Time {
	return r.referenceTime
}

// Config defines the configuration for period resolution.
type Config struct {
	Type     PeriodType `json:"type" yaml:"type"`         // "calendar" or "rolling"
	Duration string     `json:"duration" yaml:"duration"` // ISO 8601 duration (e.g., "P1Y")
	Anchor   AnchorType `json:"anchor,omitempty" yaml:"anchor"` // For calendar: "year", "quarter", "month"
}

// Resolve calculates the measurement period based on configuration.
//
// Examples:
//   - Calendar year 2024: Config{Type: "calendar", Duration: "P1Y", Anchor: "year"}
//   - Rolling 12 months: Config{Type: "rolling", Duration: "P1Y"}
//   - Calendar Q1 2024: Config{Type: "calendar", Duration: "P3M", Anchor: "quarter"}
func (r *Resolver) Resolve(cfg Config) (*MeasurementPeriod, error) {
	switch cfg.Type {
	case PeriodTypeCalendar:
		return r.resolveCalendar(cfg)
	case PeriodTypeRolling:
		return r.resolveRolling(cfg)
	default:
		return nil, fmt.Errorf("unknown period type: %s", cfg.Type)
	}
}

// ResolveForYear calculates the measurement period for a specific calendar year.
// This is a convenience method for the common case of annual measures.
func (r *Resolver) ResolveForYear(year int) *MeasurementPeriod {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.UTC)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeCalendar,
		Label: fmt.Sprintf("CY%d", year),
	}
}

// ResolveForQuarter calculates the measurement period for a specific quarter.
func (r *Resolver) ResolveForQuarter(year, quarter int) (*MeasurementPeriod, error) {
	if quarter < 1 || quarter > 4 {
		return nil, fmt.Errorf("invalid quarter: %d (must be 1-4)", quarter)
	}

	startMonth := time.Month((quarter-1)*3 + 1)
	endMonth := startMonth + 2

	start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
	end := lastDayOfMonth(year, endMonth)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeCalendar,
		Label: fmt.Sprintf("Q%d %d", quarter, year),
	}, nil
}

// ResolveForMonth calculates the measurement period for a specific month.
func (r *Resolver) ResolveForMonth(year int, month time.Month) *MeasurementPeriod {
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := lastDayOfMonth(year, month)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeCalendar,
		Label: fmt.Sprintf("%s %d", month.String()[:3], year),
	}
}

// ResolveRolling12Months calculates a 12-month rolling period ending at reference time.
func (r *Resolver) ResolveRolling12Months() *MeasurementPeriod {
	end := r.referenceTime
	start := end.AddDate(-1, 0, 0)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeRolling,
		Label: fmt.Sprintf("Rolling 12M (ending %s)", end.Format("2006-01-02")),
	}
}

// CurrentCalendarYear returns the measurement period for the current calendar year.
func (r *Resolver) CurrentCalendarYear() *MeasurementPeriod {
	return r.ResolveForYear(r.referenceTime.Year())
}

// PreviousCalendarYear returns the measurement period for the previous calendar year.
func (r *Resolver) PreviousCalendarYear() *MeasurementPeriod {
	return r.ResolveForYear(r.referenceTime.Year() - 1)
}

// resolveCalendar handles calendar-aligned period resolution.
func (r *Resolver) resolveCalendar(cfg Config) (*MeasurementPeriod, error) {
	duration, err := parseISO8601Duration(cfg.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	switch cfg.Anchor {
	case AnchorYear:
		return r.resolveCalendarYear(duration)
	case AnchorQuarter:
		return r.resolveCalendarQuarter(duration)
	case AnchorMonth:
		return r.resolveCalendarMonth(duration)
	default:
		// Default to year if no anchor specified
		return r.resolveCalendarYear(duration)
	}
}

// resolveRolling handles rolling period resolution.
func (r *Resolver) resolveRolling(cfg Config) (*MeasurementPeriod, error) {
	duration, err := parseISO8601Duration(cfg.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	end := r.referenceTime
	start := end.AddDate(-duration.years, -duration.months, -duration.days)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeRolling,
		Label: fmt.Sprintf("Rolling %s", cfg.Duration),
	}, nil
}

func (r *Resolver) resolveCalendarYear(d duration) (*MeasurementPeriod, error) {
	year := r.referenceTime.Year()
	// For multi-year periods, go back the appropriate number of years
	if d.years > 1 {
		year -= d.years - 1
	}

	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+d.years-1, 12, 31, 23, 59, 59, 999999999, time.UTC)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeCalendar,
		Label: fmt.Sprintf("CY%d", year),
	}, nil
}

func (r *Resolver) resolveCalendarQuarter(d duration) (*MeasurementPeriod, error) {
	// Determine current quarter
	month := r.referenceTime.Month()
	quarter := (int(month) - 1) / 3

	year := r.referenceTime.Year()
	startMonth := time.Month(quarter*3 + 1)

	start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
	endMonth := startMonth + time.Month(d.months) - 1
	end := lastDayOfMonth(year, endMonth)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeCalendar,
		Label: fmt.Sprintf("Q%d %d", quarter+1, year),
	}, nil
}

func (r *Resolver) resolveCalendarMonth(d duration) (*MeasurementPeriod, error) {
	year := r.referenceTime.Year()
	month := r.referenceTime.Month()

	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := lastDayOfMonth(year, month+time.Month(d.months)-1)

	return &MeasurementPeriod{
		Start: start,
		End:   end,
		Type:  PeriodTypeCalendar,
		Label: fmt.Sprintf("%s %d", month.String()[:3], year),
	}, nil
}

// IsInPeriod checks if a given time falls within the measurement period.
func (p *MeasurementPeriod) IsInPeriod(t time.Time) bool {
	return !t.Before(p.Start) && !t.After(p.End)
}

// Duration returns the duration of the measurement period.
func (p *MeasurementPeriod) Duration() time.Duration {
	return p.End.Sub(p.Start)
}

// Days returns the number of days in the measurement period.
func (p *MeasurementPeriod) Days() int {
	return int(p.Duration().Hours() / 24)
}

// --- ISO 8601 Duration Parsing ---

type duration struct {
	years  int
	months int
	days   int
}

// parseISO8601Duration parses a subset of ISO 8601 durations.
// Supports: P1Y (1 year), P6M (6 months), P30D (30 days), P1Y6M (1 year 6 months)
func parseISO8601Duration(s string) (duration, error) {
	if len(s) < 2 || s[0] != 'P' {
		return duration{}, fmt.Errorf("invalid ISO 8601 duration: %s", s)
	}

	d := duration{}
	num := 0
	hasNum := false

	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			num = num*10 + int(c-'0')
			hasNum = true
		case c == 'Y':
			if !hasNum {
				return duration{}, fmt.Errorf("missing number before Y in: %s", s)
			}
			d.years = num
			num = 0
			hasNum = false
		case c == 'M':
			if !hasNum {
				return duration{}, fmt.Errorf("missing number before M in: %s", s)
			}
			d.months = num
			num = 0
			hasNum = false
		case c == 'D':
			if !hasNum {
				return duration{}, fmt.Errorf("missing number before D in: %s", s)
			}
			d.days = num
			num = 0
			hasNum = false
		default:
			return duration{}, fmt.Errorf("unexpected character '%c' in: %s", c, s)
		}
	}

	return d, nil
}

// lastDayOfMonth returns the last moment of the given month.
func lastDayOfMonth(year int, month time.Month) time.Time {
	// Handle month overflow
	for month > 12 {
		month -= 12
		year++
	}

	// First day of next month, minus 1 nanosecond
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	return time.Date(nextYear, nextMonth, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
}
