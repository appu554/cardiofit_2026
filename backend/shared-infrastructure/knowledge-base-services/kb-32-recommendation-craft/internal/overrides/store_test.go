package overrides_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/cardiofit/kb32/internal/overrides"
)

// ---------------------------------------------------------------------------
// InMemory tests
// ---------------------------------------------------------------------------

func TestInMemoryStore_CreateAndGet_Roundtrip(t *testing.T) {
	s := overrides.NewInMemoryStore()
	ctx := context.Background()

	r := overrides.OverrideReason{
		RecommendationID:    "rec-001",
		ReasonCode:          "alert_fatigue",
		AppropriatenessFlag: "inappropriate_override",
		Reasoning:           "The alert fires too often for this resident.",
		CapturedBy:          "pharmacist-uuid-001",
	}

	created, err := s.Create(ctx, r)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create did not assign an ID")
	}
	if created.CapturedAt.IsZero() {
		t.Fatal("Create did not populate CapturedAt")
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReasonCode != r.ReasonCode {
		t.Errorf("ReasonCode = %q; want %q", got.ReasonCode, r.ReasonCode)
	}
	if got.AppropriatenessFlag != r.AppropriatenessFlag {
		t.Errorf("AppropriatenessFlag = %q; want %q", got.AppropriatenessFlag, r.AppropriatenessFlag)
	}
	if got.Reasoning != r.Reasoning {
		t.Errorf("Reasoning = %q; want %q", got.Reasoning, r.Reasoning)
	}
}

func TestInMemoryStore_Get_NotFound(t *testing.T) {
	s := overrides.NewInMemoryStore()
	_, err := s.Get(context.Background(), "nonexistent-id")
	if err != overrides.ErrNotFound {
		t.Errorf("Get unknown id: want ErrNotFound, got %v", err)
	}
}

func TestInMemoryStore_ListByRule_Filter(t *testing.T) {
	s := overrides.NewInMemoryStore()
	ctx := context.Background()

	// Three overrides for rule-A, two for rule-B.
	for i := 0; i < 3; i++ {
		_, err := s.CreateForRule(ctx, overrides.OverrideReason{
			RecommendationID:    "rec-A",
			ReasonCode:          "clinical_judgment",
			AppropriatenessFlag: "appropriate_override",
			Reasoning:           "clinical judgment applied",
			CapturedBy:          "pharmacist-001",
		}, "rule-A")
		if err != nil {
			t.Fatalf("CreateForRule rule-A: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := s.CreateForRule(ctx, overrides.OverrideReason{
			RecommendationID:    "rec-B",
			ReasonCode:          "patient_preference",
			AppropriatenessFlag: "appropriate_override",
			Reasoning:           "patient declined",
			CapturedBy:          "pharmacist-002",
		}, "rule-B")
		if err != nil {
			t.Fatalf("CreateForRule rule-B: %v", err)
		}
	}

	listA, err := s.ListByRule(ctx, "rule-A")
	if err != nil {
		t.Fatalf("ListByRule rule-A: %v", err)
	}
	if len(listA) != 3 {
		t.Errorf("ListByRule rule-A: got %d; want 3", len(listA))
	}

	listB, err := s.ListByRule(ctx, "rule-B")
	if err != nil {
		t.Fatalf("ListByRule rule-B: %v", err)
	}
	if len(listB) != 2 {
		t.Errorf("ListByRule rule-B: got %d; want 2", len(listB))
	}

	listC, err := s.ListByRule(ctx, "rule-nonexistent")
	if err != nil {
		t.Fatalf("ListByRule nonexistent: %v", err)
	}
	if len(listC) != 0 {
		t.Errorf("ListByRule nonexistent: got %d; want 0", len(listC))
	}
}

func TestInMemoryStore_PatternSummary_Aggregation(t *testing.T) {
	s := overrides.NewInMemoryStore()
	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	// 5 appropriate, 3 inappropriate, 2 mixed for rule-X
	flags := []struct {
		flag string
		n    int
	}{
		{"appropriate_override", 5},
		{"inappropriate_override", 3},
		{"mixed", 2},
	}
	for _, f := range flags {
		for i := 0; i < f.n; i++ {
			_, err := s.CreateForRule(ctx, overrides.OverrideReason{
				RecommendationID:    "rec-X",
				ReasonCode:          "alert_fatigue",
				AppropriatenessFlag: f.flag,
				Reasoning:           "test",
				CapturedBy:          "user",
			}, "rule-X")
			if err != nil {
				t.Fatalf("CreateForRule: %v", err)
			}
		}
	}

	summary, err := s.PatternSummary(ctx, "rule-X", since)
	if err != nil {
		t.Fatalf("PatternSummary: %v", err)
	}
	if summary["appropriate_override"] != 5 {
		t.Errorf("appropriate_override = %d; want 5", summary["appropriate_override"])
	}
	if summary["inappropriate_override"] != 3 {
		t.Errorf("inappropriate_override = %d; want 3", summary["inappropriate_override"])
	}
	if summary["mixed"] != 2 {
		t.Errorf("mixed = %d; want 2", summary["mixed"])
	}
}

func TestInMemoryStore_PatternSummary_SinceFilter(t *testing.T) {
	s := overrides.NewInMemoryStore()
	ctx := context.Background()

	// Create one override in the past (before the since cutoff) and one recent.
	past := time.Now().Add(-48 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)
	since := time.Now().Add(-24 * time.Hour) // window covers only recent

	pastRec := overrides.OverrideReason{
		RecommendationID:    "rec-old",
		ReasonCode:          "alert_fatigue",
		AppropriatenessFlag: "inappropriate_override",
		Reasoning:           "old override",
		CapturedBy:          "user",
		CapturedAt:          past,
	}
	recentRec := overrides.OverrideReason{
		RecommendationID:    "rec-new",
		ReasonCode:          "alert_fatigue",
		AppropriatenessFlag: "inappropriate_override",
		Reasoning:           "new override",
		CapturedBy:          "user",
		CapturedAt:          recent,
	}

	if _, err := s.CreateForRule(ctx, pastRec, "rule-time"); err != nil {
		t.Fatalf("CreateForRule past: %v", err)
	}
	if _, err := s.CreateForRule(ctx, recentRec, "rule-time"); err != nil {
		t.Fatalf("CreateForRule recent: %v", err)
	}

	summary, err := s.PatternSummary(ctx, "rule-time", since)
	if err != nil {
		t.Fatalf("PatternSummary: %v", err)
	}
	if summary["inappropriate_override"] != 1 {
		t.Errorf("inappropriate_override = %d; want 1 (past override excluded)", summary["inappropriate_override"])
	}
}

// ---------------------------------------------------------------------------
// Postgres integration tests (skipped unless VAIDSHALA_TEST_DSN is set)
// ---------------------------------------------------------------------------

func TestPostgresStore_CreateAndGet_Roundtrip(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set — skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}

	s := overrides.NewPostgresStore(db)
	ctx := context.Background()

	r := overrides.OverrideReason{
		RecommendationID:    "00000000-0000-0000-0000-000000000001",
		ReasonCode:          "alert_fatigue",
		AppropriatenessFlag: "inappropriate_override",
		Reasoning:           "Integration test override.",
		CapturedBy:          "test-pharmacist",
	}

	created, err := s.Create(ctx, r)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create did not assign an ID")
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReasonCode != r.ReasonCode {
		t.Errorf("ReasonCode = %q; want %q", got.ReasonCode, r.ReasonCode)
	}
}

func TestPostgresStore_Get_NotFound(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set — skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}

	s := overrides.NewPostgresStore(db)
	_, err = s.Get(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != overrides.ErrNotFound {
		t.Errorf("Get unknown UUID: want ErrNotFound, got %v", err)
	}
}
