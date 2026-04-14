package services

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

// SeasonalWindow describes a date range during which trajectory cards for
// specific domains should be suppressed or downgraded.
type SeasonalWindow struct {
	Name            string                `yaml:"name"`
	Start           string                `yaml:"start,omitempty"`     // ISO date or empty
	End             string                `yaml:"end,omitempty"`
	StartDOY        int                   `yaml:"start_doy,omitempty"` // 1-366 or 0
	EndDOY          int                   `yaml:"end_doy,omitempty"`
	AffectedDomains []dtModels.MHRIDomain `yaml:"affected_domains"`
	Mode            string                `yaml:"mode"` // DOWNGRADE_URGENCY | SUPPRESS
	Rationale       string                `yaml:"rationale"`
}

type seasonalCalendarFile struct {
	Windows []SeasonalWindow `yaml:"windows"`
}

// SeasonalContext holds the loaded calendar for a single market.
type SeasonalContext struct {
	market  string
	windows []SeasonalWindow
}

// NewSeasonalContext loads a seasonal calendar from a YAML file.
// Returns an empty context (no suppression) if the file is missing.
func NewSeasonalContext(market, calendarPath string) (*SeasonalContext, error) {
	if calendarPath == "" {
		return &SeasonalContext{market: market, windows: nil}, nil
	}

	data, err := os.ReadFile(calendarPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &SeasonalContext{market: market, windows: nil}, nil
		}
		return nil, fmt.Errorf("read seasonal calendar: %w", err)
	}

	var file seasonalCalendarFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse seasonal calendar: %w", err)
	}

	return &SeasonalContext{market: market, windows: file.Windows}, nil
}

// ActiveWindows returns the seasonal windows active at the given timestamp.
func (s *SeasonalContext) ActiveWindows(ts time.Time) []SeasonalWindow {
	var active []SeasonalWindow
	for _, w := range s.windows {
		if windowActiveAt(w, ts) {
			active = append(active, w)
		}
	}
	return active
}

// ShouldSuppress returns (suppress, downgrade, rationale) for a domain at a given time.
// If multiple windows apply, SUPPRESS wins over DOWNGRADE_URGENCY.
func (s *SeasonalContext) ShouldSuppress(domain dtModels.MHRIDomain, ts time.Time) (suppress bool, downgrade bool, rationale string) {
	for _, w := range s.ActiveWindows(ts) {
		if !containsDomain(w.AffectedDomains, domain) {
			continue
		}
		switch w.Mode {
		case "SUPPRESS":
			return true, false, w.Rationale
		case "DOWNGRADE_URGENCY":
			downgrade = true
			rationale = w.Rationale
			// keep scanning for SUPPRESS which would override
		}
	}
	return false, downgrade, rationale
}

func windowActiveAt(w SeasonalWindow, ts time.Time) bool {
	// Year-specific window
	if w.Start != "" && w.End != "" {
		start, err1 := time.Parse("2006-01-02", w.Start)
		end, err2 := time.Parse("2006-01-02", w.End)
		if err1 != nil || err2 != nil {
			return false
		}
		// Inclusive on both ends (entire end day is valid).
		return !ts.Before(start) && ts.Before(end.Add(24*time.Hour))
	}
	// Recurring day-of-year window
	if w.StartDOY > 0 && w.EndDOY > 0 {
		doy := ts.YearDay()
		return doy >= w.StartDOY && doy <= w.EndDOY
	}
	return false
}

func containsDomain(domains []dtModels.MHRIDomain, target dtModels.MHRIDomain) bool {
	for _, d := range domains {
		if d == target {
			return true
		}
	}
	return false
}
