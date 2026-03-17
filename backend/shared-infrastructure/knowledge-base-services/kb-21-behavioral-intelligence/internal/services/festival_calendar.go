package services

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// FestivalType classifies the dietary pattern of a festival period.
type FestivalType string

const (
	FestivalSweetHeavy FestivalType = "SWEET_HEAVY" // High glycemic load (sweets, fried foods)
	FestivalFasting    FestivalType = "FASTING"      // Extended fasting periods
	FestivalMixed      FestivalType = "MIXED"        // Combination of fasting and feasting
	FestivalMeatFeast  FestivalType = "MEAT_FEAST"   // High protein/fat meals
)

// FestivalRegion represents an Indian geographic region.
type FestivalRegion string

const (
	RegionAll   FestivalRegion = "ALL"
	RegionNorth FestivalRegion = "NORTH"
	RegionSouth FestivalRegion = "SOUTH"
	RegionEast  FestivalRegion = "EAST"
	RegionWest  FestivalRegion = "WEST"
)

// FestivalEntry represents a single festival period from the YAML calendar.
type FestivalEntry struct {
	Name        string   `yaml:"name"`
	DateStart   string   `yaml:"date_start"`
	DateEnd     string   `yaml:"date_end"`
	Regions     []string `yaml:"regions"`
	FastingType string   `yaml:"fasting_type"`
	Notes       string   `yaml:"notes"`
}

// FestivalCalendarFile is the top-level YAML structure.
type FestivalCalendarFile struct {
	Festivals []FestivalEntry `yaml:"festivals"`
}

// FestivalWindow represents a resolved festival period with parsed dates.
type FestivalWindow struct {
	Name        string       `json:"name"`
	Start       time.Time    `json:"start"`        // core start - suppressionDays
	End         time.Time    `json:"end"`           // core end + suppressionDays
	CoreStart   time.Time    `json:"core_start"`    // actual festival start
	CoreEnd     time.Time    `json:"core_end"`      // actual festival end
	FastingType FestivalType `json:"fasting_type"`
	Regions     []string     `json:"regions"`
}

// FestivalCalendar provides festival-aware adherence suppression for KB-21.
// Integration point: AdherenceService.RecomputeAdherence() checks this calendar
// before penalizing missed doses during festival periods.
type FestivalCalendar struct {
	festivals      []FestivalEntry
	suppressionDays int // ±days around festival core dates (default 2)
}

// NewFestivalCalendar loads a festival calendar from a YAML file.
func NewFestivalCalendar(yamlPath string) (*FestivalCalendar, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read festival calendar: %w", err)
	}

	var cal FestivalCalendarFile
	if err := yaml.Unmarshal(data, &cal); err != nil {
		return nil, fmt.Errorf("failed to parse festival calendar: %w", err)
	}

	return &FestivalCalendar{
		festivals:       cal.Festivals,
		suppressionDays: 2,
	}, nil
}

// NewFestivalCalendarFromEntries creates a calendar from pre-loaded entries (for testing).
func NewFestivalCalendarFromEntries(entries []FestivalEntry) *FestivalCalendar {
	return &FestivalCalendar{
		festivals:       entries,
		suppressionDays: 2,
	}
}

// IsFestivalPeriod returns true if the given date falls within a festival window
// (core date ± suppressionDays) for the specified region.
func (fc *FestivalCalendar) IsFestivalPeriod(date time.Time, region string) bool {
	window := fc.GetActiveFestival(date, region)
	return window != nil
}

// GetActiveFestival returns the festival window active on the given date for the
// given region, or nil if no festival is active.
func (fc *FestivalCalendar) GetActiveFestival(date time.Time, region string) *FestivalWindow {
	dateOnly := truncateToDate(date)

	for _, f := range fc.festivals {
		if !fc.regionMatches(f.Regions, region) {
			continue
		}

		coreStart, err := time.Parse("2006-01-02", f.DateStart)
		if err != nil {
			continue
		}
		coreEnd, err := time.Parse("2006-01-02", f.DateEnd)
		if err != nil {
			continue
		}

		windowStart := coreStart.AddDate(0, 0, -fc.suppressionDays)
		windowEnd := coreEnd.AddDate(0, 0, fc.suppressionDays)

		if !dateOnly.Before(windowStart) && !dateOnly.After(windowEnd) {
			return &FestivalWindow{
				Name:        f.Name,
				Start:       windowStart,
				End:         windowEnd,
				CoreStart:   coreStart,
				CoreEnd:     coreEnd,
				FastingType: FestivalType(f.FastingType),
				Regions:     f.Regions,
			}
		}
	}
	return nil
}

// GetFestivalsInRange returns all festival windows that overlap with the given date range
// for the specified region.
func (fc *FestivalCalendar) GetFestivalsInRange(start, end time.Time, region string) []FestivalWindow {
	var results []FestivalWindow

	for _, f := range fc.festivals {
		if !fc.regionMatches(f.Regions, region) {
			continue
		}

		coreStart, err := time.Parse("2006-01-02", f.DateStart)
		if err != nil {
			continue
		}
		coreEnd, err := time.Parse("2006-01-02", f.DateEnd)
		if err != nil {
			continue
		}

		windowStart := coreStart.AddDate(0, 0, -fc.suppressionDays)
		windowEnd := coreEnd.AddDate(0, 0, fc.suppressionDays)

		// Check overlap: window overlaps range if windowStart <= end AND windowEnd >= start
		if !windowStart.After(end) && !windowEnd.Before(start) {
			results = append(results, FestivalWindow{
				Name:        f.Name,
				Start:       windowStart,
				End:         windowEnd,
				CoreStart:   coreStart,
				CoreEnd:     coreEnd,
				FastingType: FestivalType(f.FastingType),
				Regions:     f.Regions,
			})
		}
	}
	return results
}

// CountFestivalDays returns the number of days within the date range that fall
// inside a festival suppression window for the given region.
func (fc *FestivalCalendar) CountFestivalDays(start, end time.Time, region string) int {
	festivals := fc.GetFestivalsInRange(start, end, region)
	if len(festivals) == 0 {
		return 0
	}

	// Mark each day in range as festival or not
	festivalDays := make(map[string]bool)
	for d := truncateToDate(start); !d.After(truncateToDate(end)); d = d.AddDate(0, 0, 1) {
		for _, fw := range festivals {
			if !d.Before(fw.Start) && !d.After(fw.End) {
				festivalDays[d.Format("2006-01-02")] = true
				break
			}
		}
	}
	return len(festivalDays)
}

// AdjustAdherenceScore applies festival-mode suppression to an adherence score.
// During festival periods, missed doses are weighted less heavily.
//
// Adjustment formula:
//   - festivalDays / totalDays = festival fraction
//   - Effective score = rawScore + (1 - rawScore) * festivalFraction * suppressionFactor
//
// suppressionFactor varies by fasting type:
//   - SWEET_HEAVY: 0.7 (expect significant dietary deviation)
//   - FASTING:     0.8 (fasting is more controlled, but still disrupted)
//   - MIXED:       0.6 (unpredictable pattern)
//   - MEAT_FEAST:  0.5 (moderate glycemic impact)
func (fc *FestivalCalendar) AdjustAdherenceScore(rawScore float64, windowStart, windowEnd time.Time, region string) float64 {
	totalDays := int(windowEnd.Sub(windowStart).Hours()/24) + 1
	if totalDays <= 0 {
		return rawScore
	}

	festivals := fc.GetFestivalsInRange(windowStart, windowEnd, region)
	if len(festivals) == 0 {
		return rawScore
	}

	festivalDays := fc.CountFestivalDays(windowStart, windowEnd, region)
	if festivalDays == 0 {
		return rawScore
	}

	// Find the dominant fasting type (most festival days)
	typeDays := make(map[FestivalType]int)
	for d := truncateToDate(windowStart); !d.After(truncateToDate(windowEnd)); d = d.AddDate(0, 0, 1) {
		for _, fw := range festivals {
			if !d.Before(fw.Start) && !d.After(fw.End) {
				typeDays[fw.FastingType]++
				break
			}
		}
	}

	dominantType := FestivalSweetHeavy
	maxDays := 0
	for ft, days := range typeDays {
		if days > maxDays {
			dominantType = ft
			maxDays = days
		}
	}

	suppressionFactor := suppressionFactorForType(dominantType)
	festivalFraction := float64(festivalDays) / float64(totalDays)

	// Boost: partially forgive the gap between raw score and perfect adherence
	adjusted := rawScore + (1-rawScore)*festivalFraction*suppressionFactor
	if adjusted > 1.0 {
		adjusted = 1.0
	}
	return adjusted
}

func suppressionFactorForType(ft FestivalType) float64 {
	switch ft {
	case FestivalSweetHeavy:
		return 0.7
	case FestivalFasting:
		return 0.8
	case FestivalMixed:
		return 0.6
	case FestivalMeatFeast:
		return 0.5
	default:
		return 0.5
	}
}

func (fc *FestivalCalendar) regionMatches(festivalRegions []string, patientRegion string) bool {
	for _, r := range festivalRegions {
		if r == "ALL" || r == patientRegion {
			return true
		}
	}
	return false
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
