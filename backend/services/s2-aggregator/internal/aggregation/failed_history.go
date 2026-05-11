package aggregation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// DefaultFailedInterventionTimeHorizonDays is the v1.0 Part 8.4 default
// (12 months / 365 days). Matches the failed_interventions package's
// DefaultRetryWindow so a record currently within its veto window is
// always within the panel's default horizon.
const DefaultFailedInterventionTimeHorizonDays = 365

// FIRGapBadgeText is the exact gap-badge message surfaced when FIR
// retrieval by resident-id is not wired. Step 4 Task B's documented
// uuid.Nil limitation makes this the operational reality in Phase 1
// production until kb-32 ships the RecommendationID→ResidentID
// JOIN-resolver.
const FIRGapBadgeText = "FIR retrieval incomplete — kb-32 RecommendationID→ResidentID resolver pending"

// FailedInterventionCard is the per-record rendering unit per v1.0
// Part 8.2. IsActiveVeto mirrors failed_interventions.IsVetoActive
// (record.RetryEligibleDate.After(now)). LinkedRecommendationIDs lists
// kb-32 packets currently surfaced for this resident whose classified
// InterventionType matches this record — Purpose 1 of v1.0 Part 8.3
// (prevent unproductive re-recommendation).
type FailedInterventionCard struct {
	Record                  substrate_types.FailedInterventionRecord
	IsActiveVeto            bool
	LinkedRecommendationIDs []uuid.UUID
	SubstrateRefs           []SubstrateRef
}

// FailedInterventionPattern is the v1.0 Part 8.5 cluster output (e.g.,
// "3 PPI deprescribing recs declined over 18 months"). Pattern
// detection rules are deferred to senior consultant pharmacist
// authoring per S2 Addendum Part 6.1 — Phase 1 ships the structural
// slot only.
type FailedInterventionPattern struct {
	PatternID             string
	Description           string
	ContributingRecordIDs []uuid.UUID
	DetectedAt            time.Time
}

// FailedInterventionPanel is the full Panel F rendering unit. GapBadge
// is non-empty when RetrievalAvailable is false — the renderer SHOULD
// display GapBadge prominently so the pharmacist knows the panel is
// degraded.
type FailedInterventionPanel struct {
	Cards              []FailedInterventionCard
	RetrievalAvailable bool
	GapBadge           string
	TimeHorizonDays    int
	Patterns           []FailedInterventionPattern
}

// PatternDetector is the v1.0 Part 8.5 extension point. Phase 1 ships
// NoOpPatternDetector; senior consultant pharmacist authoring fills in
// rules later per S2 Addendum Part 6.1.
//
// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1)
type PatternDetector interface {
	Detect(cards []FailedInterventionCard, now time.Time) []FailedInterventionPattern
}

// NoOpPatternDetector is the default Phase 1 detector. Returns a
// non-nil empty slice so the panel's Patterns field is never nil.
//
// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1)
type NoOpPatternDetector struct{}

// Detect implements PatternDetector by returning no patterns.
func (NoOpPatternDetector) Detect(_ []FailedInterventionCard, _ time.Time) []FailedInterventionPattern {
	return []FailedInterventionPattern{}
}

// ClassifierAdapter maps a kb-32 RuleID into the canonical
// InterventionType vocabulary used by FailedInterventionRecord. Phase
// 1 production wiring (Task 8) adapts the shared
// failed_interventions.ClassifyInterventionType function; tests
// supply an inline prefix mapper.
type ClassifierAdapter interface {
	ClassifyInterventionType(ruleID string) (interventionType string, classified bool)
}

// PanelOption configures BuildFailedInterventionPanel. Options compose
// via the functional-options pattern.
type PanelOption func(*panelOptions)

type panelOptions struct {
	timeHorizonDays int
	detector        PatternDetector
	classifier      ClassifierAdapter
}

// WithTimeHorizon overrides the default 12-month horizon. Values ≤ 0
// fall back to DefaultFailedInterventionTimeHorizonDays.
func WithTimeHorizon(days int) PanelOption {
	return func(o *panelOptions) {
		if days > 0 {
			o.timeHorizonDays = days
		}
	}
}

// WithPatternDetector overrides the default NoOpPatternDetector.
func WithPatternDetector(d PatternDetector) PanelOption {
	return func(o *panelOptions) {
		if d != nil {
			o.detector = d
		}
	}
}

// WithClassifier overrides the default RuleID→InterventionType
// classifier (which is a no-op that links no recommendations).
// Production wiring (Task 8) supplies an adapter for the shared
// failed_interventions.ClassifyInterventionType function.
func WithClassifier(c ClassifierAdapter) PanelOption {
	return func(o *panelOptions) {
		if c != nil {
			o.classifier = c
		}
	}
}

// nilClassifier is the default ClassifierAdapter — classifies nothing
// and links no pending recommendations. Used when the caller doesn't
// supply WithClassifier; the panel still renders correctly.
type nilClassifier struct{}

func (nilClassifier) ClassifyInterventionType(_ string) (string, bool) { return "", false }

// BuildFailedInterventionPanel composes the per-resident Panel F per
// S2 v1.0 Part 8. Surfaces the FIR retrieval gap per Step 4 Task B
// documented uuid.Nil limitation when
// client.FailedInterventionRetrievalAvailable() reports false.
//
// Empty-state: Cards is always non-nil (empty slice when no records).
// Patterns is always non-nil (empty when the detector finds none).
func BuildFailedInterventionPanel(
	ctx context.Context,
	client SubstrateClient,
	residentID uuid.UUID,
	now time.Time,
	opts ...PanelOption,
) (FailedInterventionPanel, error) {
	if client == nil {
		return FailedInterventionPanel{}, fmt.Errorf("BuildFailedInterventionPanel: nil SubstrateClient")
	}

	o := panelOptions{
		timeHorizonDays: DefaultFailedInterventionTimeHorizonDays,
		detector:        NoOpPatternDetector{},
		classifier:      nilClassifier{},
	}
	for _, opt := range opts {
		opt(&o)
	}

	available := client.FailedInterventionRetrievalAvailable()
	panel := FailedInterventionPanel{
		Cards:              []FailedInterventionCard{},
		RetrievalAvailable: available,
		TimeHorizonDays:    o.timeHorizonDays,
		Patterns:           []FailedInterventionPattern{},
	}
	if !available {
		panel.GapBadge = FIRGapBadgeText
		// Degraded mode: surface the gap, do not fetch records (the
		// production adapter would return [] anyway, but we honour the
		// contract explicitly so the panel is empty-by-design when the
		// resolver is missing).
		return panel, nil
	}

	since := now.AddDate(0, 0, -o.timeHorizonDays)
	records, err := client.FailedInterventionHistory(ctx, residentID, since)
	if err != nil {
		return panel, fmt.Errorf("FailedInterventionHistory: %w", err)
	}

	// Fetch pending recommendations once for the linkage scan.
	pending, err := client.PendingRecommendations(ctx, residentID)
	if err != nil {
		return panel, fmt.Errorf("PendingRecommendations: %w", err)
	}
	linkIndex := buildLinkIndex(pending, o.classifier)

	cards := make([]FailedInterventionCard, 0, len(records))
	for _, r := range records {
		card := FailedInterventionCard{
			Record:                  r,
			IsActiveVeto:            r.RetryEligibleDate.After(now),
			LinkedRecommendationIDs: linkIndex[strings.ToLower(r.InterventionType)],
			SubstrateRefs:           buildFIRSubstrateRefs(r),
		}
		cards = append(cards, card)
	}
	// Most-recent-first (the substrate fake already sorts but production
	// adapters may not — re-sort here for invariant).
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Record.AttemptDate.After(cards[j].Record.AttemptDate)
	})
	panel.Cards = cards
	panel.Patterns = o.detector.Detect(cards, now)
	if panel.Patterns == nil {
		panel.Patterns = []FailedInterventionPattern{}
	}
	return panel, nil
}

// buildLinkIndex maps each canonical interventionType to the list of
// pending recommendation IDs whose classified type matches.
// Case-insensitive on the key.
func buildLinkIndex(pending []substrate_types.RecommendationPacket, classifier ClassifierAdapter) map[string][]uuid.UUID {
	out := map[string][]uuid.UUID{}
	for _, pkt := range pending {
		it, ok := classifier.ClassifyInterventionType(pkt.AppliedRule.RuleID)
		if !ok || it == "" {
			continue
		}
		key := strings.ToLower(it)
		out[key] = append(out[key], pkt.RecommendationID)
	}
	return out
}

// buildFIRSubstrateRefs anchors the verification-not-belief invariant:
// every card carries ≥1 SubstrateRef. The DocumentedBy pharmacist UUID
// is the substrate ID — the FIR row itself has no separate UUID in the
// canonical store (it's keyed on resident_id + intervention_type +
// attempt_date), so we name the row by the documenter + attempt date.
func buildFIRSubstrateRefs(r substrate_types.FailedInterventionRecord) []SubstrateRef {
	return []SubstrateRef{{
		Source: "kb-fir",
		ID:     r.DocumentedBy,
		Description: fmt.Sprintf(
			"failed intervention %s attempted %s (outcome: %s)",
			r.InterventionType,
			r.AttemptDate.Format("2006-01-02"),
			r.Outcome,
		),
	}}
}
