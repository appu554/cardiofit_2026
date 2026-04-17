package services

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"kb-patient-profile/internal/models"

	"gopkg.in/yaml.v3"
)

// ─── YAML config types ────────────────────────────────────────────────────────

type calendarEventConfig struct {
	Name              string   `yaml:"name"`
	Category          string   `yaml:"category"`
	DurationDays      int      `yaml:"duration_days"`
	AffectedOutcomes  []string `yaml:"affected_outcomes"`
	ExpectedDirection string   `yaml:"expected_direction"`
	ExpectedMagnitude string   `yaml:"expected_magnitude"`
	BaseWeight        float64  `yaml:"base_weight"`
	PostEventWashout  int      `yaml:"post_event_washout_days"`
	AppliesTo         string   `yaml:"applies_to"`
	RegionSpecific    bool     `yaml:"region_specific"`
	Regions           []string `yaml:"regions"`
	GregorianMonths   []int    `yaml:"gregorian_approx_month"`
	Dates2026         struct {
		Start string `yaml:"start"`
		End   string `yaml:"end"`
	} `yaml:"dates_2026"`
	Dates2027 struct {
		Start string `yaml:"start"`
		End   string `yaml:"end"`
	} `yaml:"dates_2027"`
}

type seasonalHbA1cConfig struct {
	Pattern      string  `yaml:"pattern"`
	PeakMonths   []int   `yaml:"peak_months"`
	TroughMonths []int   `yaml:"trough_months"`
	MagnitudePct float64 `yaml:"magnitude_pct"`
}

type calendarYAML struct {
	ReligiousEvents  []calendarEventConfig `yaml:"religious_events"`
	FestivalEvents   []calendarEventConfig `yaml:"festival_events"`
	SeasonalEvents   []calendarEventConfig `yaml:"seasonal_events"`
	IndigenousEvents []calendarEventConfig `yaml:"indigenous_events"`
	SeasonalHbA1c    seasonalHbA1cConfig   `yaml:"seasonal_hba1c"`
}

// ─── ConfounderCalendar ───────────────────────────────────────────────────────

// ConfounderCalendar holds market-specific confounder events loaded from YAML.
type ConfounderCalendar struct {
	market           string
	religiousEvents  []calendarEventConfig
	festivalEvents   []calendarEventConfig
	seasonalEvents   []calendarEventConfig
	indigenousEvents []calendarEventConfig
	seasonalHbA1c    seasonalHbA1cConfig
}

// confounderFactorResult is the return type for FindActiveConfounders — wraps
// the model ConfounderFactor fields for convenience within this service layer.
type confounderFactorResult = models.ConfounderFactor

// LoadConfounderCalendar reads {configDir}/{market}/confounder_calendar.yaml
// and returns a ready-to-query ConfounderCalendar.
func LoadConfounderCalendar(configDir, market string) (*ConfounderCalendar, error) {
	path := filepath.Join(configDir, market, "confounder_calendar.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("confounder_calendar: read %s: %w", path, err)
	}

	var raw calendarYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("confounder_calendar: parse %s: %w", path, err)
	}

	return &ConfounderCalendar{
		market:           market,
		religiousEvents:  raw.ReligiousEvents,
		festivalEvents:   raw.FestivalEvents,
		seasonalEvents:   raw.SeasonalEvents,
		indigenousEvents: raw.IndigenousEvents,
		seasonalHbA1c:    raw.SeasonalHbA1c,
	}, nil
}

// FindActiveConfounders returns all confounder factors that overlap the given
// outcome window, filtered by the patient's religious affiliation and region.
func (c *ConfounderCalendar) FindActiveConfounders(
	windowStart, windowEnd time.Time,
	religiousAffiliation, region string,
) []models.ConfounderFactor {
	var results []models.ConfounderFactor

	allEvents := make([]calendarEventConfig, 0,
		len(c.religiousEvents)+len(c.festivalEvents)+len(c.seasonalEvents)+len(c.indigenousEvents))
	allEvents = append(allEvents, c.religiousEvents...)
	allEvents = append(allEvents, c.festivalEvents...)
	allEvents = append(allEvents, c.seasonalEvents...)
	allEvents = append(allEvents, c.indigenousEvents...)

	for _, event := range allEvents {
		// Filter by religious affiliation
		if event.AppliesTo != "ALL" && event.AppliesTo != religiousAffiliation {
			continue
		}

		// Filter by region
		if event.RegionSpecific && !containsString(event.Regions, region) {
			continue
		}

		// Try resolving event dates for each year that could overlap the window
		for year := windowStart.Year(); year <= windowEnd.Year(); year++ {
			eventStart, eventEnd := resolveEventDates(event, year)
			if eventStart.IsZero() || eventEnd.IsZero() {
				continue
			}

			// Extend end by washout period
			effectiveEnd := eventEnd.AddDate(0, 0, event.PostEventWashout)

			// Check overlap between [eventStart, effectiveEnd] and [windowStart, windowEnd]
			overlapStart := maxTime(eventStart, windowStart)
			overlapEnd := minTime(effectiveEnd, windowEnd)

			if !overlapStart.Before(overlapEnd) {
				continue // no overlap
			}

			overlapDays := int(math.Round(overlapEnd.Sub(overlapStart).Hours() / 24))
			if overlapDays <= 0 {
				continue
			}

			windowDays := int(math.Round(windowEnd.Sub(windowStart).Hours() / 24))
			if windowDays <= 0 {
				windowDays = 1
			}
			overlapPct := float64(overlapDays) / float64(windowDays) * 100.0
			weight := event.BaseWeight * math.Min(overlapPct/100.0, 1.0)

			results = append(results, models.ConfounderFactor{
				Category:          models.ConfounderCategory(event.Category),
				Name:              event.Name,
				Weight:            weight,
				AffectedOutcomes:  event.AffectedOutcomes,
				ExpectedDirection: event.ExpectedDirection,
				ExpectedMagnitude: event.ExpectedMagnitude,
				WindowStart:       eventStart,
				WindowEnd:         effectiveEnd,
				OverlapDays:       overlapDays,
				OverlapPct:        math.Round(overlapPct*100) / 100,
				Source:            fmt.Sprintf("calendar:%s", c.market),
				Confidence:        "HIGH",
			})
		}
	}

	return results
}

// GetSeasonalHbA1cAdjustment returns the seasonal HbA1c adjustment for a given
// month: +MagnitudePct for peak months, -MagnitudePct for trough months, 0 otherwise.
func (c *ConfounderCalendar) GetSeasonalHbA1cAdjustment(month int) float64 {
	for _, m := range c.seasonalHbA1c.PeakMonths {
		if m == month {
			return c.seasonalHbA1c.MagnitudePct
		}
	}
	for _, m := range c.seasonalHbA1c.TroughMonths {
		if m == month {
			return -c.seasonalHbA1c.MagnitudePct
		}
	}
	return 0.0
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// resolveEventDates returns the start and end dates for an event in the given year.
// Priority: exact dates_2026/2027 → GregorianMonths fallback.
func resolveEventDates(event calendarEventConfig, year int) (time.Time, time.Time) {
	const layout = "2006-01-02"

	// Try exact dates for 2026
	if year == 2026 && event.Dates2026.Start != "" && event.Dates2026.End != "" {
		start, err1 := time.Parse(layout, event.Dates2026.Start)
		end, err2 := time.Parse(layout, event.Dates2026.End)
		if err1 == nil && err2 == nil {
			return start, end
		}
	}

	// Try exact dates for 2027
	if year == 2027 && event.Dates2027.Start != "" && event.Dates2027.End != "" {
		start, err1 := time.Parse(layout, event.Dates2027.Start)
		end, err2 := time.Parse(layout, event.Dates2027.End)
		if err1 == nil && err2 == nil {
			return start, end
		}
	}

	// Fallback to GregorianMonths
	if len(event.GregorianMonths) > 0 {
		firstMonth := event.GregorianMonths[0]
		start := time.Date(year, time.Month(firstMonth), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 0, event.DurationDays)
		return start, end
	}

	return time.Time{}, time.Time{}
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
