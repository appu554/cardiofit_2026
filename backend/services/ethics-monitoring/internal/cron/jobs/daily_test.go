package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
	"github.com/cardiofit/shared/v2_substrate/ethics/pattern_detection"
)

type stubFetcher struct {
	prior, current []pattern_detection.RuleSnapshot
	err            error
}

func (s stubFetcher) LatestRuleSnapshots(_ context.Context) ([]pattern_detection.RuleSnapshot, []pattern_detection.RuleSnapshot, error) {
	return s.prior, s.current, s.err
}

func TestDailyAcceptanceAppropriateness_Divergent_AppendsOneEntry(t *testing.T) {
	store := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(store)
	job := DailyAcceptanceAppropriatenessJob{
		Fetcher: stubFetcher{
			prior:   []pattern_detection.RuleSnapshot{{RuleID: "RULE-A", AcceptanceRate: 0.50, AppropriatenessMean: 4.0}},
			current: []pattern_detection.RuleSnapshot{{RuleID: "RULE-A", AcceptanceRate: 0.65, AppropriatenessMean: 4.0}}, // +15pp accept, flat appropriateness → divergent
		},
		Logger: logger,
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	entries, _ := store.List(context.Background())
	if len(entries) != 1 {
		t.Fatalf("entries=%d, want 1", len(entries))
	}
	e := entries[0]
	if e.EntryType != ethics_log.EntryTypePatternDetected {
		t.Errorf("EntryType=%q, want %q", e.EntryType, ethics_log.EntryTypePatternDetected)
	}
	if e.Severity != 3 {
		t.Errorf("Severity=%d, want 3", e.Severity)
	}
	if e.Status != ethics_log.StatusOpen {
		t.Errorf("Status=%q, want %q", e.Status, ethics_log.StatusOpen)
	}
	if !strings.Contains(e.Description, "RULE-A") {
		t.Errorf("Description missing rule id: %q", e.Description)
	}
}

func TestDailyAcceptanceAppropriateness_NonDivergent_NoEntries(t *testing.T) {
	store := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(store)
	job := DailyAcceptanceAppropriatenessJob{
		Fetcher: stubFetcher{
			prior:   []pattern_detection.RuleSnapshot{{RuleID: "RULE-B", AcceptanceRate: 0.50, AppropriatenessMean: 4.0}},
			current: []pattern_detection.RuleSnapshot{{RuleID: "RULE-B", AcceptanceRate: 0.65, AppropriatenessMean: 4.5}}, // +15pp accept WITH +0.5 appropriateness → NOT divergent
		},
		Logger: logger,
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	entries, _ := store.List(context.Background())
	if len(entries) != 0 {
		t.Fatalf("entries=%d, want 0", len(entries))
	}
}

func TestDailyAcceptanceAppropriateness_FetcherError_Wrapped(t *testing.T) {
	logger := ethics_log.NewLogger(ethics_log.NewInMemoryStore())
	sentinel := errors.New("db down")
	job := DailyAcceptanceAppropriatenessJob{
		Fetcher: stubFetcher{err: sentinel},
		Logger:  logger,
	}
	err := job.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error not wrapped: %v", err)
	}
	if !strings.Contains(err.Error(), "fetch snapshots") {
		t.Errorf("error message missing context: %v", err)
	}
}

// failingStore wraps an in-memory store but returns an error on Append, used
// to test the log-emit error path.
type failingStore struct{ err error }

func (f failingStore) Append(_ context.Context, _ ethics_log.Entry) error { return f.err }
func (f failingStore) List(_ context.Context) ([]ethics_log.Entry, error) { return nil, nil }

func TestDailyAcceptanceAppropriateness_LogEmitError_Wrapped(t *testing.T) {
	sentinel := errors.New("log store boom")
	logger := ethics_log.NewLogger(failingStore{err: sentinel})
	job := DailyAcceptanceAppropriatenessJob{
		Fetcher: stubFetcher{
			prior:   []pattern_detection.RuleSnapshot{{RuleID: "RULE-C", AcceptanceRate: 0.50, AppropriatenessMean: 4.0}},
			current: []pattern_detection.RuleSnapshot{{RuleID: "RULE-C", AcceptanceRate: 0.65, AppropriatenessMean: 4.0}},
		},
		Logger: logger,
	}
	err := job.Run(context.Background())
	if err == nil {
		t.Fatal("expected log emit error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error not wrapped: %v", err)
	}
	if !strings.Contains(err.Error(), "log emit") {
		t.Errorf("error message missing context: %v", err)
	}
}

// indexedFailingStore errors on Append for a specific call index (0-based) and
// records every accepted entry otherwise. Used to verify errors.Join behaviour
// — that a transient log-store failure for one rule does not silently drop
// the surviving N-1 emits in the same batch.
type indexedFailingStore struct {
	failOnCall int
	calls      int
	accepted   []ethics_log.Entry
	err        error
}

func (s *indexedFailingStore) Append(_ context.Context, e ethics_log.Entry) error {
	idx := s.calls
	s.calls++
	if idx == s.failOnCall {
		return s.err
	}
	s.accepted = append(s.accepted, e)
	return nil
}

func (s *indexedFailingStore) List(_ context.Context) ([]ethics_log.Entry, error) {
	return s.accepted, nil
}

func TestDailyAcceptanceAppropriateness_PartialAppendFailure_KeepsSurvivors(t *testing.T) {
	sentinel := errors.New("transient log boom")
	store := &indexedFailingStore{failOnCall: 1, err: sentinel} // fail on the 2nd of 3 emits
	logger := ethics_log.NewLogger(store)

	// Three divergent rules → three Append calls; the middle one will fail.
	mk := func(id string) (pattern_detection.RuleSnapshot, pattern_detection.RuleSnapshot) {
		return pattern_detection.RuleSnapshot{RuleID: id, AcceptanceRate: 0.50, AppropriatenessMean: 4.0},
			pattern_detection.RuleSnapshot{RuleID: id, AcceptanceRate: 0.65, AppropriatenessMean: 4.0}
	}
	pA, cA := mk("RULE-A")
	pB, cB := mk("RULE-B")
	pC, cC := mk("RULE-C")
	job := DailyAcceptanceAppropriatenessJob{
		Fetcher: stubFetcher{
			prior:   []pattern_detection.RuleSnapshot{pA, pB, pC},
			current: []pattern_detection.RuleSnapshot{cA, cB, cC},
		},
		Logger: logger,
	}

	err := job.Run(context.Background())
	if err == nil {
		t.Fatal("expected joined error from middle Append failure, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("joined error should wrap sentinel, got %v", err)
	}
	if store.calls != 3 {
		t.Errorf("Append called %d times, want 3 (no early abort)", store.calls)
	}
	if len(store.accepted) != 2 {
		t.Fatalf("accepted entries=%d, want 2 (N-1 survivors)", len(store.accepted))
	}
	// Survivors should be RULE-A (call 0) and RULE-C (call 2).
	if !strings.Contains(store.accepted[0].Description, "RULE-A") {
		t.Errorf("first survivor description=%q, want RULE-A", store.accepted[0].Description)
	}
	if !strings.Contains(store.accepted[1].Description, "RULE-C") {
		t.Errorf("second survivor description=%q, want RULE-C", store.accepted[1].Description)
	}
}

// stubSuppressionFetcher for the suppression scan job.
type stubSuppressionFetcher struct {
	inputs []pattern_detection.SuppressionInputs
	err    error
}

func (s stubSuppressionFetcher) SuppressionInputs(_ context.Context) ([]pattern_detection.SuppressionInputs, error) {
	return s.inputs, s.err
}

func TestDailySuppressionScan_FlagsHighDeferralUndocumented(t *testing.T) {
	store := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(store)
	job := DailySuppressionScanJob{
		Fetcher: stubSuppressionFetcher{
			inputs: []pattern_detection.SuppressionInputs{
				// 50% deferral, 100% undocumented → triggers (defaults: 0.40 / 0.20)
				{RuleID: "RULE-S", TotalRecommendations: 10, DeferredCount: 5, DeferredWithReasoningCount: 0},
				// 10% deferral → below threshold, no entry
				{RuleID: "RULE-T", TotalRecommendations: 10, DeferredCount: 1, DeferredWithReasoningCount: 0},
			},
		},
		Logger: logger,
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	entries, _ := store.List(context.Background())
	if len(entries) != 1 {
		t.Fatalf("entries=%d, want 1", len(entries))
	}
	if !strings.Contains(entries[0].Description, "RULE-S") {
		t.Errorf("expected RULE-S in description, got %q", entries[0].Description)
	}
}
