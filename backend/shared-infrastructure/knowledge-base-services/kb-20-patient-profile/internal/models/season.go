package models

import "time"

// Indian Meteorological Department (IMD) season constants.
// Used for chronotherapy (bedtime dosing varies by daylight hours)
// and BP seasonality (winter typically shows higher BP readings).
const (
	SeasonSummer  = "SUMMER"  // March–May
	SeasonMonsoon = "MONSOON" // June–September
	SeasonAutumn  = "AUTUMN"  // October–November
	SeasonWinter  = "WINTER"  // December–February
	SeasonUnknown = "UNKNOWN"
)

// DeriveSeason returns the Indian meteorological season for a given date.
// India uses a 4-season model from IMD (India Meteorological Department).
// Returns SeasonUnknown if the provided time is the zero value.
func DeriveSeason(t time.Time) string {
	if t.IsZero() {
		return SeasonUnknown
	}

	switch t.Month() {
	case time.March, time.April, time.May:
		return SeasonSummer
	case time.June, time.July, time.August, time.September:
		return SeasonMonsoon
	case time.October, time.November:
		return SeasonAutumn
	case time.December, time.January, time.February:
		return SeasonWinter
	default:
		return SeasonUnknown
	}
}
