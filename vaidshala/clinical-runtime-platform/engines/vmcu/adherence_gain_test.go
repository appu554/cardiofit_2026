package vmcu

import "testing"

func TestAdherenceToGainFactor(t *testing.T) {
	tests := []struct {
		name       string
		adherence  float64
		wantFactor float64
	}{
		{name: "high adherence", adherence: 0.85, wantFactor: 1.0},
		{name: "boundary 0.70 (inclusive)", adherence: 0.70, wantFactor: 1.0},
		{name: "mid adherence", adherence: 0.55, wantFactor: 0.5},
		{name: "boundary 0.40 (inclusive)", adherence: 0.40, wantFactor: 0.5},
		{name: "low adherence", adherence: 0.30, wantFactor: 0.0},
		{name: "zero adherence", adherence: 0.0, wantFactor: 0.0},
		{name: "perfect adherence", adherence: 1.0, wantFactor: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AdherenceToGainFactor(tt.adherence)
			if got != tt.wantFactor {
				t.Errorf("AdherenceToGainFactor(%v) = %v, want %v", tt.adherence, got, tt.wantFactor)
			}
		})
	}
}
