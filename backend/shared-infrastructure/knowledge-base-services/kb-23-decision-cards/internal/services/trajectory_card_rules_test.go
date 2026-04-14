package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

// ---------------------------------------------------------------------------
// TestTrajectoryCards_ConcordantDeterioration
// ---------------------------------------------------------------------------

func TestTrajectoryCards_ConcordantDeterioration(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend:          dtModels.TrendDeclining,
		CompositeSlope:          -1.5,
		ConcordantDeterioration: true,
		DomainsDeteriorating:    3,
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Trend: dtModels.TrendDeclining, SlopePerDay: -0.8},
			dtModels.DomainCardio:     {Trend: dtModels.TrendRapidDeclining, SlopePerDay: -1.5},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendDeclining, SlopePerDay: -0.6},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable, SlopePerDay: -0.1},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "CONCORDANT_DETERIORATION" {
			found = true
			if c.Urgency != "IMMEDIATE" {
				t.Errorf("expected IMMEDIATE urgency for 3-domain concordant, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected CONCORDANT_DETERIORATION card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_DivergenceAlert
// ---------------------------------------------------------------------------

func TestTrajectoryCards_DivergenceAlert(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend:     dtModels.TrendStable,
		HasDiscordantTrend: true,
		Divergences: []dtModels.DivergencePattern{
			{
				ImprovingDomain:   dtModels.DomainGlucose,
				DecliningDomain:   dtModels.DomainCardio,
				ImprovingSlope:    0.8,
				DecliningSlope:    -1.2,
				DivergenceRate:    2.0,
				ClinicalConcern:   "GLUCOSE improving while CARDIO declining",
				PossibleMechanism: "Consider SGLT2i for dual benefit",
			},
		},
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Trend: dtModels.TrendImproving},
			dtModels.DomainCardio:     {Trend: dtModels.TrendDeclining},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendStable},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "DOMAIN_DIVERGENCE" {
			found = true
			if c.Urgency != "URGENT" {
				t.Errorf("expected URGENT urgency for divergence, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected DOMAIN_DIVERGENCE card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_BehavioralLeadingIndicator
// ---------------------------------------------------------------------------

func TestTrajectoryCards_BehavioralLeadingIndicator(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: dtModels.TrendDeclining,
		LeadingIndicators: []dtModels.LeadingIndicator{
			{
				LeadingDomain:  dtModels.DomainBehavioral,
				LaggingDomains: []dtModels.MHRIDomain{dtModels.DomainGlucose, dtModels.DomainCardio},
				Interpretation: "Behavioral decline preceded clinical deterioration",
			},
		},
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainBehavioral: {Trend: dtModels.TrendRapidDeclining, SlopePerDay: -2.5},
			dtModels.DomainGlucose:    {Trend: dtModels.TrendDeclining, SlopePerDay: -0.5},
			dtModels.DomainCardio:     {Trend: dtModels.TrendDeclining, SlopePerDay: -0.4},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable, SlopePerDay: 0.0},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "BEHAVIORAL_LEADING_INDICATOR" {
			found = true
			if c.Urgency != "URGENT" {
				t.Errorf("expected URGENT urgency, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected BEHAVIORAL_LEADING_INDICATOR card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_SingleDomainRapidDecline
// ---------------------------------------------------------------------------

func TestTrajectoryCards_SingleDomainRapidDecline(t *testing.T) {
	cardio := dtModels.DomainCardio
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: dtModels.TrendDeclining,
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainCardio: {
				Domain:      dtModels.DomainCardio,
				Trend:       dtModels.TrendRapidDeclining,
				SlopePerDay: -2.0,
				Confidence:  dtModels.ConfidenceHigh,
				R2:          0.85,
				StartScore:  70,
				EndScore:    42,
			},
			dtModels.DomainGlucose:    {Trend: dtModels.TrendStable},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendStable},
		},
		DominantDriver: &cardio,
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "DOMAIN_RAPID_DECLINE" {
			found = true
		}
	}
	if !found {
		t.Error("expected DOMAIN_RAPID_DECLINE card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_AllStable_NoUrgentCards
// ---------------------------------------------------------------------------

func TestTrajectoryCards_AllStable_NoUrgentCards(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: dtModels.TrendStable,
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Trend: dtModels.TrendStable},
			dtModels.DomainCardio:     {Trend: dtModels.TrendStable},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendStable},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	for _, c := range cards {
		if c.Urgency == "IMMEDIATE" || c.Urgency == "URGENT" {
			t.Errorf("expected no urgent/immediate cards for all-stable, got %s (%s)", c.Urgency, c.CardType)
		}
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCardsWithSeasonalContext_GlucoseDowngraded
// ---------------------------------------------------------------------------

func TestTrajectoryCardsWithSeasonalContext_GlucoseDowngraded(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	// Build a Diwali seasonal context inline.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cal.yaml")
	yamlContent := `
windows:
  - name: diwali
    start: "2026-11-04"
    end: "2026-11-14"
    affected_domains: [GLUCOSE]
    mode: DOWNGRADE_URGENCY
    rationale: "festival eating"
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	seasonalCtx, _ := NewSeasonalContext("india", path)

	// A glucose RAPID_DECLINE card should be downgraded URGENT → ROUTINE.
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: dtModels.TrendDeclining,
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Domain: dtModels.DomainGlucose, Trend: dtModels.TrendRapidDeclining, SlopePerDay: -1.5, Confidence: dtModels.ConfidenceHigh, R2: 0.9, StartScore: 70, EndScore: 50},
			dtModels.DomainCardio:     {Trend: dtModels.TrendStable},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendStable},
		},
	}

	cards := EvaluateTrajectoryCardsWithSeasonalContext(traj, seasonalCtx, now)

	found := false
	for _, c := range cards {
		if c.CardType == "DOMAIN_RAPID_DECLINE" && c.Domain == "GLUCOSE" {
			found = true
			if c.Urgency != "ROUTINE" {
				t.Errorf("expected glucose card downgraded to ROUTINE, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected glucose rapid decline card to still be present (downgraded)")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCardsWithSeasonalContext_NilContextPassthrough
// ---------------------------------------------------------------------------

func TestTrajectoryCardsWithSeasonalContext_NilContextPassthrough(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Domain: dtModels.DomainGlucose, Trend: dtModels.TrendRapidDeclining, SlopePerDay: -1.5, Confidence: dtModels.ConfidenceHigh},
			dtModels.DomainCardio:     {Trend: dtModels.TrendStable},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendStable},
		},
	}

	without := EvaluateTrajectoryCards(traj)
	with := EvaluateTrajectoryCardsWithSeasonalContext(traj, nil, time.Now())

	if len(without) != len(with) {
		t.Errorf("expected nil context to produce identical output: %d vs %d cards", len(without), len(with))
	}
}
