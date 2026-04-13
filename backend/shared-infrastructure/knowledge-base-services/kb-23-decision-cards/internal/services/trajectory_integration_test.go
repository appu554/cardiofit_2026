package services

import (
	"testing"
	"time"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

// TestIntegration_ComputeAndEvaluate_RajeshKumar verifies the full pipeline:
// KB-26 ComputeDecomposedTrajectory output flows into KB-23 EvaluateTrajectoryCards
// without type conversion (validates the pkg/trajectory alias facade at runtime).
func TestIntegration_ComputeAndEvaluate_RajeshKumar(t *testing.T) {
	now := time.Now()
	// Rajesh: 3 domains declining, behavioral collapsing fastest, body comp stable.
	points := []dtModels.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 62, GlucoseScore: 55, CardioScore: 58, BodyCompScore: 65, BehavioralScore: 72},
		{Timestamp: now.Add(-10 * 24 * time.Hour), CompositeScore: 58, GlucoseScore: 50, CardioScore: 52, BodyCompScore: 65, BehavioralScore: 65},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 53, GlucoseScore: 45, CardioScore: 48, BodyCompScore: 64, BehavioralScore: 55},
		{Timestamp: now.Add(-4 * 24 * time.Hour), CompositeScore: 48, GlucoseScore: 40, CardioScore: 42, BodyCompScore: 64, BehavioralScore: 42},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 42, GlucoseScore: 35, CardioScore: 38, BodyCompScore: 63, BehavioralScore: 30},
	}

	// Compute via the KB-26 engine through the public facade.
	trajectory := dtModels.Compute("e2e-rajesh-kumar", points)

	// Feed into KB-23 card generator.
	// The trajectory is dtModels.DecomposedTrajectory (alias for internal type),
	// so passing &trajectory directly validates the alias boundary at runtime.
	cards := EvaluateTrajectoryCards(&trajectory)

	if len(cards) == 0 {
		t.Fatal("expected at least one card from full pipeline")
	}

	// Rajesh has 3 domains declining → expect a CONCORDANT_DETERIORATION card with IMMEDIATE urgency.
	foundConcordant := false
	for _, c := range cards {
		if c.CardType == "CONCORDANT_DETERIORATION" {
			foundConcordant = true
			if c.Urgency != "IMMEDIATE" {
				t.Errorf("expected IMMEDIATE urgency for 3-domain Rajesh scenario, got %s", c.Urgency)
			}
		}
	}
	if !foundConcordant {
		t.Error("expected CONCORDANT_DETERIORATION card from full pipeline")
	}

	// Behavioral leading indicator should fire (behavioral declining fastest, then glucose/cardio).
	foundLeading := false
	for _, c := range cards {
		if c.CardType == "BEHAVIORAL_LEADING_INDICATOR" {
			foundLeading = true
		}
	}
	if !foundLeading {
		t.Error("expected BEHAVIORAL_LEADING_INDICATOR card from full pipeline")
	}
}
