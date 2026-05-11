// Package lifecycle — evidence_trace_test.go covers the Stage 7 emission
// surface end-to-end.
//
// Test groups:
//
//   - DraftedTransitionEntry JSON round-trip (no DB, no EthicsLog needed)
//   - EthicsLogEmitter writes correct EntryType + Severity over an in-memory store
//   - CompositeEmitter fan-out + fail-fast on first error
//   - PostgresEmitter integration test — skips when VAIDSHALA_TEST_DSN unset
package lifecycle

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/kb32/internal/appropriateness"
	"github.com/cardiofit/kb32/internal/citations"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// sampleEntry returns a fully-populated DraftedTransitionEntry suitable as a
// shared fixture across the table-driven tests below.
func sampleEntry(t *testing.T) DraftedTransitionEntry {
	t.Helper()
	return DraftedTransitionEntry{
		RecommendationID: uuid.New(),
		AuthorID:         uuid.New(),
		RuleID:           "TEST-RULE-001",
		ContentHash:      "a1b2c3d4e5f6",
		Assessment: appropriateness.Assessment{
			ClinicalWarrant:        3,
			EvidenceSolidity:       4,
			AlternativesConsidered: 3,
			RestraintConsidered:    3,
			GoalsOfCareAlignment:   5,
		},
		Citations: []citations.RecommendationCitation{
			{
				RecommendationID: "rec-1",
				SourceID:         "ADG-2025-AU",
				Version:          "1",
				PinnedAt:         time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
			},
		},
		Urgency: "red",
		FiredAt: time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
	}
}

// ---------------------------------------------------------------------------
// JSON round-trip
// ---------------------------------------------------------------------------

func TestDraftedTransitionEntry_JSONRoundTrip(t *testing.T) {
	orig := sampleEntry(t)
	raw, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got DraftedTransitionEntry
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.RecommendationID != orig.RecommendationID {
		t.Errorf("RecommendationID: got %s want %s", got.RecommendationID, orig.RecommendationID)
	}
	if got.RuleID != orig.RuleID {
		t.Errorf("RuleID: got %q want %q", got.RuleID, orig.RuleID)
	}
	if got.ContentHash != orig.ContentHash {
		t.Errorf("ContentHash mismatch")
	}
	if got.Assessment != orig.Assessment {
		t.Errorf("Assessment: got %+v want %+v", got.Assessment, orig.Assessment)
	}
	if len(got.Citations) != 1 || got.Citations[0].SourceID != "ADG-2025-AU" {
		t.Errorf("Citations not preserved: %+v", got.Citations)
	}
	if got.Urgency != orig.Urgency {
		t.Errorf("Urgency: got %q want %q", got.Urgency, orig.Urgency)
	}
	if !got.FiredAt.Equal(orig.FiredAt) {
		t.Errorf("FiredAt: got %s want %s", got.FiredAt, orig.FiredAt)
	}
}

// ---------------------------------------------------------------------------
// EthicsLogEmitter
// ---------------------------------------------------------------------------

func TestEthicsLogEmitter_AppendsDecisionEntry(t *testing.T) {
	store := ethics_log.NewInMemoryStore()
	emitter := NewEthicsLogEmitter(ethics_log.NewLogger(store))

	entry := sampleEntry(t)
	if err := emitter.EmitDraftedTransition(context.Background(), entry); err != nil {
		t.Fatalf("EmitDraftedTransition: %v", err)
	}

	entries, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("store.List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry; got %d", len(entries))
	}
	got := entries[0]
	if got.EntryType != ethics_log.EntryTypeDecision {
		t.Errorf("EntryType: got %q want %q", got.EntryType, ethics_log.EntryTypeDecision)
	}
	if got.Severity != 1 {
		t.Errorf("Severity: got %d want 1", got.Severity)
	}
	if got.DecisionID != entry.RecommendationID {
		t.Errorf("DecisionID: got %s want %s", got.DecisionID, entry.RecommendationID)
	}
	// Description must round-trip back to the original DraftedTransitionEntry.
	var decoded DraftedTransitionEntry
	if err := json.Unmarshal([]byte(got.Description), &decoded); err != nil {
		t.Fatalf("unmarshal Description payload: %v", err)
	}
	if decoded.RuleID != entry.RuleID {
		t.Errorf("payload RuleID: got %q want %q", decoded.RuleID, entry.RuleID)
	}
}

func TestEthicsLogEmitter_NilLoggerReturnsError(t *testing.T) {
	emitter := &EthicsLogEmitter{logger: nil}
	err := emitter.EmitDraftedTransition(context.Background(), sampleEntry(t))
	if err == nil {
		t.Fatal("expected error for nil logger; got nil")
	}
}

// ---------------------------------------------------------------------------
// CompositeEmitter
// ---------------------------------------------------------------------------

// recordingEmitter is a controllable EvidenceTraceEmitter test double.
type recordingEmitter struct {
	calls int
	err   error
}

func (r *recordingEmitter) EmitDraftedTransition(_ context.Context, _ DraftedTransitionEntry) error {
	r.calls++
	return r.err
}

func TestCompositeEmitter_FansOutToAll(t *testing.T) {
	a := &recordingEmitter{}
	b := &recordingEmitter{}
	c := NewCompositeEmitter(a, b)

	if err := c.EmitDraftedTransition(context.Background(), sampleEntry(t)); err != nil {
		t.Fatalf("EmitDraftedTransition: %v", err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("expected both emitters called once; got a=%d b=%d", a.calls, b.calls)
	}
}

func TestCompositeEmitter_FailFastOnFirstError(t *testing.T) {
	boom := errors.New("synthetic emitter failure")
	a := &recordingEmitter{err: boom}
	b := &recordingEmitter{}
	c := NewCompositeEmitter(a, b)

	err := c.EmitDraftedTransition(context.Background(), sampleEntry(t))
	if err == nil {
		t.Fatal("expected error; got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("expected wrapped boom error; got %v", err)
	}
	if a.calls != 1 {
		t.Errorf("expected first emitter called once; got %d", a.calls)
	}
	if b.calls != 0 {
		t.Errorf("expected second emitter NOT called after first error; got %d", b.calls)
	}
}

func TestCompositeEmitter_EmptyIsNoop(t *testing.T) {
	c := NewCompositeEmitter()
	if err := c.EmitDraftedTransition(context.Background(), sampleEntry(t)); err != nil {
		t.Errorf("expected empty composite to succeed; got %v", err)
	}
}

func TestCompositeEmitter_NilMemberReturnsError(t *testing.T) {
	c := NewCompositeEmitter(nil)
	if err := c.EmitDraftedTransition(context.Background(), sampleEntry(t)); err == nil {
		t.Error("expected error for nil member; got nil")
	}
}

// ---------------------------------------------------------------------------
// PostgresEmitter (skips without VAIDSHALA_TEST_DSN)
// ---------------------------------------------------------------------------

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}
	return db
}

func TestPostgresEmitter_InsertAndReadBack(t *testing.T) {
	db := openTestDB(t)
	emitter := NewPostgresEmitter(db)

	entry := sampleEntry(t)
	// Per-test isolation: delete on cleanup.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `DELETE FROM evidence_trace_entries WHERE recommendation_id = $1`,
			entry.RecommendationID)
	})

	if err := emitter.EmitDraftedTransition(context.Background(), entry); err != nil {
		t.Fatalf("EmitDraftedTransition: %v", err)
	}

	// Read back and verify the columns.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var (
		ruleID      string
		contentHash string
		urgency     string
		firedAt     time.Time
	)
	err := db.QueryRowContext(ctx,
		`SELECT rule_id, content_hash, urgency, fired_at FROM evidence_trace_entries WHERE recommendation_id = $1`,
		entry.RecommendationID,
	).Scan(&ruleID, &contentHash, &urgency, &firedAt)
	if err != nil {
		t.Fatalf("read-back: %v", err)
	}
	if ruleID != entry.RuleID {
		t.Errorf("rule_id: got %q want %q", ruleID, entry.RuleID)
	}
	if contentHash != entry.ContentHash {
		t.Errorf("content_hash: got %q want %q", contentHash, entry.ContentHash)
	}
	if urgency != entry.Urgency {
		t.Errorf("urgency: got %q want %q", urgency, entry.Urgency)
	}
}

func TestPostgresEmitter_DuplicateRejected(t *testing.T) {
	db := openTestDB(t)
	emitter := NewPostgresEmitter(db)

	entry := sampleEntry(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `DELETE FROM evidence_trace_entries WHERE recommendation_id = $1`,
			entry.RecommendationID)
	})

	if err := emitter.EmitDraftedTransition(context.Background(), entry); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	if err := emitter.EmitDraftedTransition(context.Background(), entry); err == nil {
		t.Fatal("expected PK violation on duplicate emit; got nil")
	}
}

func TestPostgresEmitter_NilDBReturnsError(t *testing.T) {
	emitter := &PostgresEmitter{db: nil}
	if err := emitter.EmitDraftedTransition(context.Background(), sampleEntry(t)); err == nil {
		t.Error("expected error for nil db; got nil")
	}
}
