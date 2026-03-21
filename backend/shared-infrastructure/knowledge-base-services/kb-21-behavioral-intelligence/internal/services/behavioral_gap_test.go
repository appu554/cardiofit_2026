package services

import (
	"testing"

	"go.uber.org/zap"
)

func TestBehavioralGapDetector_Assess(t *testing.T) {
	detector := NewBehavioralGapDetector(zap.NewNop())

	tests := []struct {
		name      string
		adherence float64
		drugClass string
		expectGap bool
	}{
		{"below threshold", 0.30, "ACE_INHIBITOR", true},
		{"at threshold", 0.40, "STATIN", false},
		{"above threshold", 0.70, "ARB", false},
		{"zero adherence", 0.0, "METFORMIN", true},
		{"full adherence", 1.0, "SGLT2I", false},
		{"just below", 0.39, "CCB", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.Assess(tt.adherence, tt.drugClass)
			if result.GapDetected != tt.expectGap {
				t.Errorf("Assess(%f, %s).GapDetected = %v, want %v",
					tt.adherence, tt.drugClass, result.GapDetected, tt.expectGap)
			}
			if result.Adherence != tt.adherence {
				t.Errorf("Adherence = %f, want %f", result.Adherence, tt.adherence)
			}
			if result.Threshold != BehavioralGapThreshold {
				t.Errorf("Threshold = %f, want %f", result.Threshold, BehavioralGapThreshold)
			}
		})
	}
}
