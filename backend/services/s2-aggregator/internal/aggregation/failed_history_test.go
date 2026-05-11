package aggregation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

func mkFIR(residentID uuid.UUID, interventionType string, attempt time.Time, retryEligible time.Time, outcome string) substrate_types.FailedInterventionRecord {
	return substrate_types.FailedInterventionRecord{
		ResidentID:        residentID,
		InterventionType:  interventionType,
		AttemptDate:       attempt,
		Outcome:           outcome,
		DocumentedReason:  "test reason",
		RetryEligibleDate: retryEligible,
		DocumentedBy:      uuid.New(),
	}
}

func TestBuildFailedInterventionPanel_GapBadge_WhenRetrievalUnavailable(t *testing.T) {
	rid := uuid.New()
	client := NewInMemorySubstrateClient().WithFIRRetrievalAvailable(false)

	panel, err := BuildFailedInterventionPanel(context.Background(), client, rid, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panel.RetrievalAvailable {
		t.Error("expected RetrievalAvailable=false")
	}
	if panel.GapBadge != FIRGapBadgeText {
		t.Errorf("GapBadge = %q, want %q", panel.GapBadge, FIRGapBadgeText)
	}
	if panel.Cards == nil {
		t.Error("Cards must be non-nil even in degraded mode")
	}
	if len(panel.Cards) != 0 {
		t.Errorf("Cards must be empty in degraded mode; got %d", len(panel.Cards))
	}
	if panel.Patterns == nil {
		t.Error("Patterns must be non-nil")
	}
}

func TestBuildFailedInterventionPanel_PopulatedCards_AndActiveVeto(t *testing.T) {
	rid := uuid.New()
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	// Record 1: within veto window (retry-eligible in future) → IsActiveVeto=true
	rec1 := mkFIR(rid, "antipsychotic_deprescribing",
		now.AddDate(0, -2, 0),
		now.AddDate(0, 10, 0),
		substrate_types.OutcomeReversedDueToBPSDRecurrence,
	)
	// Record 2: within default horizon (10 months ago) but veto window
	// elapsed (retry-eligible 2 months ago) → IsActiveVeto=false.
	rec2 := mkFIR(rid, "benzodiazepine_deprescribing",
		now.AddDate(0, -10, 0),
		now.AddDate(0, -2, 0),
		substrate_types.OutcomeReversedDueToFamilyRequest,
	)

	client := NewInMemorySubstrateClient().WithFailedInterventions(rid, rec1, rec2)
	panel, err := BuildFailedInterventionPanel(context.Background(), client, rid, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !panel.RetrievalAvailable {
		t.Error("expected RetrievalAvailable=true")
	}
	if panel.GapBadge != "" {
		t.Errorf("GapBadge = %q, want empty string", panel.GapBadge)
	}
	if got := len(panel.Cards); got != 2 {
		t.Fatalf("Cards count = %d, want 2", got)
	}
	// Sort: most-recent-first
	if !panel.Cards[0].Record.AttemptDate.After(panel.Cards[1].Record.AttemptDate) {
		t.Error("Cards must be sorted most-recent-first")
	}
	// rec1 is the recent one → active veto
	if !panel.Cards[0].IsActiveVeto {
		t.Error("expected rec1 (recent) to be active veto")
	}
	if panel.Cards[1].IsActiveVeto {
		t.Error("expected rec2 (elapsed) to not be active veto")
	}
	if panel.TimeHorizonDays != DefaultFailedInterventionTimeHorizonDays {
		t.Errorf("TimeHorizonDays = %d, want %d", panel.TimeHorizonDays, DefaultFailedInterventionTimeHorizonDays)
	}
}

func TestBuildFailedInterventionPanel_TimeHorizonFilters(t *testing.T) {
	rid := uuid.New()
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	// Outside default 12-month horizon
	old := mkFIR(rid, "antipsychotic_deprescribing",
		now.AddDate(-2, 0, 0),
		now.AddDate(-1, 0, 0),
		substrate_types.OutcomeReversedDueToBPSDRecurrence,
	)
	// Inside default horizon
	recent := mkFIR(rid, "benzodiazepine_deprescribing",
		now.AddDate(0, -6, 0),
		now.AddDate(0, 6, 0),
		substrate_types.OutcomeReversedDueToFamilyRequest,
	)
	client := NewInMemorySubstrateClient().WithFailedInterventions(rid, old, recent)
	panel, err := BuildFailedInterventionPanel(context.Background(), client, rid, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(panel.Cards); got != 1 {
		t.Fatalf("with default horizon expected 1 card, got %d", got)
	}
	// Override horizon to 36 months → both records surface
	panel2, err := BuildFailedInterventionPanel(context.Background(), client, rid, now, WithTimeHorizon(36*30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(panel2.Cards); got != 2 {
		t.Fatalf("with 36-month horizon expected 2 cards, got %d", got)
	}
}

// prefixClassifier is a test ClassifierAdapter that maps simple RuleID
// prefixes to InterventionType, mirroring the canonical shared
// classifier without taking a dependency on it.
type prefixClassifier struct{}

func (prefixClassifier) ClassifyInterventionType(ruleID string) (string, bool) {
	up := strings.ToUpper(ruleID)
	switch {
	case strings.HasPrefix(up, "STOP_PSYCH_"):
		return "antipsychotic_deprescribing", true
	case strings.HasPrefix(up, "STOP_BENZO_"):
		return "benzodiazepine_deprescribing", true
	}
	return "", false
}

func TestBuildFailedInterventionPanel_LinksPendingRecommendations(t *testing.T) {
	rid := uuid.New()
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	rec := mkFIR(rid, "antipsychotic_deprescribing",
		now.AddDate(0, -2, 0),
		now.AddDate(0, 10, 0),
		substrate_types.OutcomeReversedDueToBPSDRecurrence,
	)
	// Pending recommendation whose RuleID classifies to the same intervention type
	pendingMatching := substrate_types.RecommendationPacket{
		RecommendationID: uuid.New(),
		AuthorID:         uuid.New(),
		Type:             "STOP",
		Sections:         map[string]string{"layer_1": "body"},
		AppliedRule:      substrate_types.AppliedRule{RuleID: "STOP_PSYCH_HALDOL", Type: "STOP", Urgency: "amber"},
		SnapshotRef:      rid,
	}
	pendingUnrelated := substrate_types.RecommendationPacket{
		RecommendationID: uuid.New(),
		AuthorID:         uuid.New(),
		Type:             "ADD",
		Sections:         map[string]string{"layer_1": "body"},
		AppliedRule:      substrate_types.AppliedRule{RuleID: "ADD_VITD", Type: "ADD", Urgency: "green"},
		SnapshotRef:      rid,
	}
	client := NewInMemorySubstrateClient().
		WithFailedInterventions(rid, rec).
		WithPackets(pendingMatching, pendingUnrelated)

	panel, err := BuildFailedInterventionPanel(
		context.Background(), client, rid, now,
		WithClassifier(prefixClassifier{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(panel.Cards); got != 1 {
		t.Fatalf("expected 1 card, got %d", got)
	}
	if got := panel.Cards[0].LinkedRecommendationIDs; len(got) != 1 || got[0] != pendingMatching.RecommendationID {
		t.Errorf("LinkedRecommendationIDs = %v, want [%s]", got, pendingMatching.RecommendationID)
	}
}

// stubDetector returns a fixed pattern set so we can assert
// WithPatternDetector wiring without authoring real rules (deferred to
// senior consultant pharmacist per Addendum Part 6.1).
type stubDetector struct{ patterns []FailedInterventionPattern }

func (s stubDetector) Detect(_ []FailedInterventionCard, _ time.Time) []FailedInterventionPattern {
	return s.patterns
}

func TestBuildFailedInterventionPanel_PatternDetectorWired(t *testing.T) {
	rid := uuid.New()
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	client := NewInMemorySubstrateClient().WithFailedInterventions(rid,
		mkFIR(rid, "antipsychotic_deprescribing",
			now.AddDate(0, -2, 0),
			now.AddDate(0, 10, 0),
			substrate_types.OutcomeReversedDueToBPSDRecurrence,
		),
	)
	want := []FailedInterventionPattern{{
		PatternID:   "TEST-PATTERN",
		Description: "test",
		DetectedAt:  now,
	}}
	panel, err := BuildFailedInterventionPanel(
		context.Background(), client, rid, now,
		WithPatternDetector(stubDetector{patterns: want}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(panel.Patterns); got != 1 || panel.Patterns[0].PatternID != "TEST-PATTERN" {
		t.Errorf("Patterns = %v, want stub-detector output", panel.Patterns)
	}
}

func TestNoOpPatternDetector_ReturnsNonNilEmpty(t *testing.T) {
	got := NoOpPatternDetector{}.Detect(nil, time.Now())
	if got == nil {
		t.Fatal("NoOpPatternDetector.Detect returned nil; must return non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("NoOpPatternDetector.Detect returned %d patterns, want 0", len(got))
	}
}

func TestBuildFailedInterventionPanel_NilClient(t *testing.T) {
	_, err := BuildFailedInterventionPanel(context.Background(), nil, uuid.New(), time.Now())
	if err == nil {
		t.Fatal("expected error on nil SubstrateClient")
	}
}
