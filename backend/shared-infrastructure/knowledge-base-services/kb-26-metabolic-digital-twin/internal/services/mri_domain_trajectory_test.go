package services

import (
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ---------------------------------------------------------------------------
// TestDomainTrajectory_GlucoseDeclining_CardioStable
// ---------------------------------------------------------------------------

func TestDomainTrajectory_GlucoseDeclining_CardioStable(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 72, GlucoseScore: 75, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 72},
		{Timestamp: now.Add(-10 * 24 * time.Hour), CompositeScore: 68, GlucoseScore: 65, CardioScore: 71, BodyCompScore: 67, BehavioralScore: 70},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 64, GlucoseScore: 55, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 71},
		{Timestamp: now.Add(-4 * 24 * time.Hour), CompositeScore: 60, GlucoseScore: 48, CardioScore: 69, BodyCompScore: 67, BehavioralScore: 70},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 56, GlucoseScore: 42, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 71},
	}

	result := ComputeDecomposedTrajectory("PAT-001", points)

	if result.CompositeTrend != "DECLINING" && result.CompositeTrend != "RAPID_DECLINING" {
		t.Errorf("expected composite DECLINING or RAPID_DECLINING, got %s", result.CompositeTrend)
	}

	glucoseSlope := result.DomainSlopes[models.DomainGlucose]
	if glucoseSlope.Trend != "RAPID_DECLINING" {
		t.Errorf("expected glucose RAPID_DECLINING, got %s (slope=%.3f)", glucoseSlope.Trend, glucoseSlope.SlopePerDay)
	}
	if glucoseSlope.SlopePerDay >= -1.0 {
		t.Errorf("expected glucose slope < -1.0, got %.3f", glucoseSlope.SlopePerDay)
	}

	cardioSlope := result.DomainSlopes[models.DomainCardio]
	if cardioSlope.Trend != "STABLE" {
		t.Errorf("expected cardio STABLE, got %s (slope=%.3f)", cardioSlope.Trend, cardioSlope.SlopePerDay)
	}

	if result.DominantDriver == nil {
		t.Fatal("expected non-nil DominantDriver")
	}
	if *result.DominantDriver != models.DomainGlucose {
		t.Errorf("expected dominant driver GLUCOSE, got %s", *result.DominantDriver)
	}
	if result.DriverContribution < 40.0 {
		t.Errorf("expected driver contribution >= 40%%, got %.1f%%", result.DriverContribution)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_AllDomainsImproving
// ---------------------------------------------------------------------------

func TestDomainTrajectory_AllDomainsImproving(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 50, GlucoseScore: 45, CardioScore: 50, BodyCompScore: 52, BehavioralScore: 55},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 58, GlucoseScore: 55, CardioScore: 58, BodyCompScore: 58, BehavioralScore: 62},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 66, GlucoseScore: 65, CardioScore: 66, BodyCompScore: 64, BehavioralScore: 70},
	}

	result := ComputeDecomposedTrajectory("PAT-002", points)

	if result.CompositeTrend != "IMPROVING" && result.CompositeTrend != "RAPID_IMPROVING" {
		t.Errorf("expected composite IMPROVING or RAPID_IMPROVING, got %s", result.CompositeTrend)
	}
	for _, domain := range models.AllMHRIDomains {
		slope := result.DomainSlopes[domain]
		if slope.SlopePerDay <= 0 {
			t.Errorf("expected %s to have positive slope, got %.3f", domain, slope.SlopePerDay)
		}
	}
	if result.HasDiscordantTrend {
		t.Error("expected HasDiscordantTrend = false for all-improving")
	}
	if result.DomainsDeteriorating != 0 {
		t.Errorf("expected 0 domains deteriorating, got %d", result.DomainsDeteriorating)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_ConcordantDeterioration
// ---------------------------------------------------------------------------

func TestDomainTrajectory_ConcordantDeterioration(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 70, GlucoseScore: 72, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 68},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 62, GlucoseScore: 60, CardioScore: 58, BodyCompScore: 66, BehavioralScore: 65},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 52, GlucoseScore: 48, CardioScore: 45, BodyCompScore: 64, BehavioralScore: 55},
	}

	result := ComputeDecomposedTrajectory("PAT-003", points)

	if !result.ConcordantDeterioration {
		t.Error("expected ConcordantDeterioration = true")
	}
	if result.DomainsDeteriorating < 2 {
		t.Errorf("expected >= 2 domains deteriorating, got %d", result.DomainsDeteriorating)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_InsufficientData
// ---------------------------------------------------------------------------

func TestDomainTrajectory_InsufficientData(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now, CompositeScore: 70, GlucoseScore: 72, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 68},
	}

	result := ComputeDecomposedTrajectory("PAT-004", points)
	if result.CompositeTrend != "INSUFFICIENT_DATA" {
		t.Errorf("expected INSUFFICIENT_DATA for composite, got %s", result.CompositeTrend)
	}
	for _, domain := range models.AllMHRIDomains {
		if result.DomainSlopes[domain].Trend != "INSUFFICIENT_DATA" {
			t.Errorf("expected INSUFFICIENT_DATA for %s, got %s", domain, result.DomainSlopes[domain].Trend)
		}
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_NoisyData_LowConfidence
// ---------------------------------------------------------------------------

func TestDomainTrajectory_NoisyData_LowConfidence(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), GlucoseScore: 70, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-11 * 24 * time.Hour), GlucoseScore: 45, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 58},
		{Timestamp: now.Add(-9 * 24 * time.Hour), GlucoseScore: 75, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 66},
		{Timestamp: now.Add(-7 * 24 * time.Hour), GlucoseScore: 40, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 57},
		{Timestamp: now.Add(-5 * 24 * time.Hour), GlucoseScore: 72, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-3 * 24 * time.Hour), GlucoseScore: 42, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 58},
		{Timestamp: now.Add(-1 * 24 * time.Hour), GlucoseScore: 68, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 64},
	}

	result := ComputeDecomposedTrajectory("PAT-005", points)
	glucoseSlope := result.DomainSlopes[models.DomainGlucose]
	if glucoseSlope.Confidence != "LOW" {
		t.Errorf("expected LOW confidence for noisy glucose, got %s (R²=%.3f)", glucoseSlope.Confidence, glucoseSlope.R2)
	}
}

// ---------------------------------------------------------------------------
// TestDomainCategoryCrossing_GlucoseOptimalToMild
// ---------------------------------------------------------------------------

func TestDomainCategoryCrossing_GlucoseOptimalToMild(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-7 * 24 * time.Hour), GlucoseScore: 72, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 70, CompositeScore: 70},
		{Timestamp: now.Add(-1 * 24 * time.Hour), GlucoseScore: 66, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 70, CompositeScore: 68},
	}

	result := ComputeDecomposedTrajectory("PAT-006", points)
	if len(result.DomainCrossings) == 0 {
		t.Fatal("expected at least one domain crossing")
	}

	found := false
	for _, c := range result.DomainCrossings {
		if c.Domain == models.DomainGlucose {
			found = true
			if c.PrevCategory != "OPTIMAL" {
				t.Errorf("expected prev category OPTIMAL, got %s", c.PrevCategory)
			}
			if c.CurrCategory != "MILD" {
				t.Errorf("expected curr category MILD, got %s", c.CurrCategory)
			}
			if c.Direction != "WORSENED" {
				t.Errorf("expected direction WORSENED, got %s", c.Direction)
			}
		}
	}
	if !found {
		t.Error("expected glucose domain crossing not found")
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_ZeroPoints
// ---------------------------------------------------------------------------

func TestDomainTrajectory_ZeroPoints(t *testing.T) {
	result := ComputeDecomposedTrajectory("PAT-007", nil)
	if result.CompositeTrend != "INSUFFICIENT_DATA" {
		t.Errorf("expected INSUFFICIENT_DATA for nil points, got %s", result.CompositeTrend)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_RajeshKumar
// ---------------------------------------------------------------------------

func TestDomainTrajectory_RajeshKumar(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 62, GlucoseScore: 55, CardioScore: 58, BodyCompScore: 65, BehavioralScore: 72},
		{Timestamp: now.Add(-10 * 24 * time.Hour), CompositeScore: 58, GlucoseScore: 50, CardioScore: 52, BodyCompScore: 65, BehavioralScore: 65},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 53, GlucoseScore: 45, CardioScore: 48, BodyCompScore: 64, BehavioralScore: 55},
		{Timestamp: now.Add(-4 * 24 * time.Hour), CompositeScore: 48, GlucoseScore: 40, CardioScore: 42, BodyCompScore: 64, BehavioralScore: 42},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 42, GlucoseScore: 35, CardioScore: 38, BodyCompScore: 63, BehavioralScore: 30},
	}

	result := ComputeDecomposedTrajectory("e2e-rajesh-kumar-002", points)

	if result.CompositeTrend != "DECLINING" && result.CompositeTrend != "RAPID_DECLINING" {
		t.Errorf("expected DECLINING or RAPID_DECLINING, got %s", result.CompositeTrend)
	}

	if result.DomainsDeteriorating < 3 {
		t.Errorf("expected >= 3 domains deteriorating, got %d", result.DomainsDeteriorating)
	}
	if !result.ConcordantDeterioration {
		t.Error("expected ConcordantDeterioration = true")
	}

	bcSlope := result.DomainSlopes[models.DomainBodyComp]
	if bcSlope.Trend != "STABLE" {
		t.Errorf("expected body comp STABLE, got %s (slope=%.3f)", bcSlope.Trend, bcSlope.SlopePerDay)
	}

	behSlope := result.DomainSlopes[models.DomainBehavioral]
	if behSlope.Trend != "RAPID_DECLINING" {
		t.Errorf("expected behavioral RAPID_DECLINING, got %s (slope=%.3f)", behSlope.Trend, behSlope.SlopePerDay)
	}

	if len(result.DomainCrossings) < 2 {
		t.Errorf("expected >= 2 category crossings, got %d", len(result.DomainCrossings))
	}
}
