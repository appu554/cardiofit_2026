package models

import (
	"testing"
	"time"
)

func TestDeriveSeason(t *testing.T) {
	tests := []struct {
		name   string
		time   time.Time
		expect string
	}{
		// Summer: March–May
		{"March is SUMMER", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), SeasonSummer},
		{"April is SUMMER", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), SeasonSummer},
		{"May is SUMMER", time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), SeasonSummer},

		// Monsoon: June–September
		{"June is MONSOON", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), SeasonMonsoon},
		{"July is MONSOON", time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), SeasonMonsoon},
		{"September is MONSOON", time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), SeasonMonsoon},

		// Autumn: October–November
		{"October is AUTUMN", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), SeasonAutumn},
		{"November is AUTUMN", time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC), SeasonAutumn},

		// Winter: December–February
		{"December is WINTER", time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC), SeasonWinter},
		{"January is WINTER", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), SeasonWinter},
		{"February is WINTER", time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC), SeasonWinter},

		// Edge: zero time
		{"zero time is UNKNOWN", time.Time{}, SeasonUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveSeason(tt.time)
			if got != tt.expect {
				t.Errorf("DeriveSeason(%v) = %q, want %q", tt.time, got, tt.expect)
			}
		})
	}
}
